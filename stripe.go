// Package payment 支付相关功能
package payment

import (
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v74"
	stripeCheckout "github.com/stripe/stripe-go/v74/checkout/session"
	stripeIntent "github.com/stripe/stripe-go/v74/paymentintent"
	stripePrice "github.com/stripe/stripe-go/v74/price"
	stripeProduct "github.com/stripe/stripe-go/v74/product"
)

// StripePaymentProvider Stripe支付提供商
// 实现Stripe支付功能
type StripePaymentProvider struct {
	PublishableKey string // 可发布密钥
	SecretKey      string // 秘密密钥
	isProd         bool   // 是否为生产环境
}

// NewStripePaymentProvider 创建新的Stripe支付提供商实例
// 参数:
//   - PublishableKey: Stripe可发布密钥
//   - SecretKey: Stripe秘密密钥
// 返回:
//   - *StripePaymentProvider: Stripe支付提供商实例
//   - error: 错误信息
func NewStripePaymentProvider(PublishableKey, SecretKey string) (*StripePaymentProvider, error) {
	isProd := true // 默认为生产环境
	
	// 创建支付提供商实例
	pp := &StripePaymentProvider{
		PublishableKey: PublishableKey,
		SecretKey:      SecretKey,
		isProd:         isProd,
	}
	
	// 设置Stripe API密钥
	stripe.Key = pp.SecretKey
	return pp, nil
}

// Pay 执行Stripe支付操作
// 创建产品、价格和结账会话
// 参数:
//   - r: 支付请求信息
// 返回:
//   - *PayResp: 支付响应信息
//   - error: 错误信息
func (pp *StripePaymentProvider) Pay(r *PayReq) (*PayResp, error) {
	// 创建临时产品
	description := joinAttachString([]string{r.ProductName, r.ProductDisplayName, r.ProviderName})
	productParams := &stripe.ProductParams{
		Name:        stripe.String(r.ProductDisplayName),
		Description: stripe.String(description),
		DefaultPriceData: &stripe.ProductDefaultPriceDataParams{
			UnitAmount: stripe.Int64(priceFloat64ToInt64(r.Price)),
			Currency:   stripe.String(r.Currency),
		},
	}
	sProduct, err := stripeProduct.New(productParams)
	if err != nil {
		return nil, err
	}
	
	// 为现有产品创建价格
	priceParams := &stripe.PriceParams{
		Currency:   stripe.String(r.Currency),
		UnitAmount: stripe.Int64(priceFloat64ToInt64(r.Price)),
		Product:    stripe.String(sProduct.ID),
	}
	sPrice, err := stripePrice.New(priceParams)
	if err != nil {
		return nil, err
	}
	
	// 创建结账会话
	checkoutParams := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(sPrice.ID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:        stripe.String(r.ReturnUrl),
		CancelURL:         stripe.String(r.ReturnUrl),
		ClientReferenceID: stripe.String(r.PaymentName),
		ExpiresAt:         stripe.Int64(time.Now().Add(30 * time.Minute).Unix()), // 30分钟后过期
	}
	
	// 添加产品描述元数据
	checkoutParams.AddMetadata("product_description", description)
	
	// 创建结账会话
	sCheckout, err := stripeCheckout.New(checkoutParams)
	if err != nil {
		return nil, err
	}
	
	// 构造支付响应
	payResp := &PayResp{
		PayUrl:  sCheckout.URL,
		OrderId: sCheckout.ID,
	}
	return payResp, nil
}

// Notify 处理Stripe支付通知
// 查询结账会话和支付意图状态并返回通知结果
// 参数:
//   - body: 通知内容
//   - orderId: 订单ID
// 返回:
//   - *NotifyResult: 通知结果
//   - error: 错误信息
func (pp *StripePaymentProvider) Notify(body []byte, orderId string) (*NotifyResult, error) {
	notifyResult := &NotifyResult{}
	
	// 获取结账会话信息
	sCheckout, err := stripeCheckout.Get(orderId, nil)
	if err != nil {
		return nil, err
	}
	
	// 根据结账会话状态设置支付状态
	switch sCheckout.Status {
	case "open": // 结账会话仍在进行中，支付处理尚未开始
		notifyResult.PaymentStatus = PaymentStateCreated
		return notifyResult, nil
	case "complete": // 结账会话已完成，支付处理可能仍在进行中
		// 继续处理
	case "expired": // 结账会话已过期，不会再进行处理
		notifyResult.PaymentStatus = PaymentStateTimeout
		return notifyResult, nil
	default: // 未知状态
		notifyResult.PaymentStatus = PaymentStateError
		notifyResult.NotifyMessage = fmt.Sprintf("unexpected stripe checkout status: %v", sCheckout.Status)
		return notifyResult, nil
	}
	
	// 根据支付状态进一步判断
	switch sCheckout.PaymentStatus {
	case "paid": // 已支付
		// 继续处理
	case "unpaid": // 未支付
		notifyResult.PaymentStatus = PaymentStateCreated
		return notifyResult, nil
	default: // 未知支付状态
		notifyResult.PaymentStatus = PaymentStateError
		notifyResult.NotifyMessage = fmt.Sprintf("unexpected stripe checkout payment status: %v", sCheckout.PaymentStatus)
		return notifyResult, nil
	}
	
	// 支付成功后，结账会话将包含对成功的PaymentIntent的引用
	sIntent, err := stripeIntent.Get(sCheckout.PaymentIntent.ID, nil)
	if err != nil {
		return nil, err
	}
	
	// 解析产品信息
	var (
		productName        string
		productDisplayName string
		providerName       string
	)
	if description, ok := sCheckout.Metadata["product_description"]; ok {
		productName, productDisplayName, providerName, _ = parseAttachString(description)
	}
	
	// 构造通知结果
	notifyResult = &NotifyResult{
		PaymentName:   sCheckout.ClientReferenceID,
		PaymentStatus: PaymentStatePaid,

		ProductName:        productName,
		ProductDisplayName: productDisplayName,
		ProviderName:       providerName,

		Price:    priceInt64ToFloat64(sIntent.Amount),
		Currency: string(sIntent.Currency),

		OrderId: orderId,
	}
	return notifyResult, nil
}

// GetInvoice 获取Stripe发票
// 当前不支持发票功能，返回空字符串
// 参数:
//   - paymentName: 支付名称
//   - personName: 个人姓名
//   - personIdCard: 身份证号
//   - personEmail: 邮箱
//   - personPhone: 电话
//   - invoiceType: 发票类型
//   - invoiceTitle: 发票抬头
//   - invoiceTaxId: 税号
// 返回:
//   - string: 发票信息（空）
//   - error: 错误信息
func (pp *StripePaymentProvider) GetInvoice(paymentName string, personName string, personIdCard string, personEmail string, personPhone string, invoiceType string, invoiceTitle string, invoiceTaxId string) (string, error) {
	return "", nil
}

// GetResponseError 获取Stripe响应错误信息
// 根据错误状态返回相应的字符串
// 参数:
//   - err: 错误对象
// 返回:
//   - string: 错误响应字符串
func (pp *StripePaymentProvider) GetResponseError(err error) string {
	if err == nil {
		return "success" // 成功
	} else {
		return "fail" // 失败
	}
}

// payment 包提供了多种支付方式的统一接口实现
package payment

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/paypal"
)

// PaypalPaymentProvider PayPal支付提供者结构体
type PaypalPaymentProvider struct {
	Client *paypal.Client // PayPal客户端实例
}

// NewPaypalPaymentProvider 创建新的PayPal支付提供者实例
// clientID: PayPal应用的客户端ID
// secret: PayPal应用的密钥
// 返回PayPal支付提供者实例和可能的错误
func NewPaypalPaymentProvider(clientID string, secret string) (*PaypalPaymentProvider, error) {
	pp := &PaypalPaymentProvider{}
	// 创建PayPal客户端，第三个参数true表示使用沙箱环境
	client, err := paypal.NewClient(clientID, secret, true)
	if err != nil {
		return nil, err
	}

	pp.Client = client
	return pp, nil
}

// Pay 处理PayPal支付请求
// r: 支付请求参数
// 返回支付响应和可能的错误
func (pp *PaypalPaymentProvider) Pay(r *PayReq) (*PayResp, error) {
	// 参考文档: https://github.com/go-pay/gopay/blob/main/doc/paypal.md
	// 创建购买单元数组
	units := make([]*paypal.PurchaseUnit, 0, 1)
	// 构建购买单元
	unit := &paypal.PurchaseUnit{
		ReferenceId: GetRandomString(16), // 生成随机引用ID
		Amount: &paypal.Amount{
			CurrencyCode: r.Currency,                    // 货币代码，例如"USD"
			Value:        priceFloat64ToString(r.Price), // 价格字符串，例如"100.00"
		},
		// 将产品信息组合为描述
		Description: joinAttachString([]string{r.ProductDisplayName, r.ProductName, r.ProviderName}),
	}
	units = append(units, unit)

	// 构建请求体参数
	bm := make(gopay.BodyMap)
	bm.Set("intent", "CAPTURE")     // 设置支付意图为捕获
	bm.Set("purchase_units", units) // 设置购买单元
	// 设置应用上下文
	bm.SetBodyMap("application_context", func(b gopay.BodyMap) {
		b.Set("brand_name", "Casdoor")   // 品牌名称
		b.Set("locale", "en-PT")         // 语言环境
		b.Set("return_url", r.ReturnUrl) // 支付成功返回URL
		b.Set("cancel_url", r.ReturnUrl) // 支付取消返回URL
	})

	// 创建PayPal订单
	ppRsp, err := pp.Client.CreateOrder(context.Background(), bm)
	if err != nil {
		return nil, err
	}
	// 检查响应状态
	if ppRsp.Code != paypal.Success {
		return nil, errors.New(ppRsp.Error)
	}
	// PayPal响应示例:
	// {"id":"9BR68863NE220374S","status":"CREATED",
	// "links":[{"href":"https://api.sandbox.paypal.com/v2/checkout/orders/9BR68863NE220374S","rel":"self","method":"GET"},
	// 			{"href":"https://www.sandbox.paypal.com/checkoutnow?token=9BR68863NE220374S","rel":"approve","method":"GET"},
	// 			{"href":"https://api.sandbox.paypal.com/v2/checkout/orders/9BR68863NE220374S","rel":"update","method":"PATCH"},
	// 			{"href":"https://api.sandbox.paypal.com/v2/checkout/orders/9BR68863NE220374S/capture","rel":"capture","method":"POST"}]}
	// 构建支付响应
	payResp := &PayResp{
		PayUrl:  ppRsp.Response.Links[1].Href, // 获取支付链接（approve链接）
		OrderId: ppRsp.Response.Id,            // 订单ID
	}
	return payResp, nil
}

// Notify 处理PayPal支付回调通知
// body: 回调请求体
// orderId: 订单ID
// 返回通知结果和可能的错误
func (pp *PaypalPaymentProvider) Notify(body []byte, orderId string) (*NotifyResult, error) {
	notifyResult := &NotifyResult{}
	// 尝试捕获订单支付
	captureRsp, err := pp.Client.OrderCapture(context.Background(), orderId, nil)
	if err != nil {
		return nil, err
	}
	// 检查捕获响应状态
	if captureRsp.Code != paypal.Success {
		errDetail := captureRsp.ErrorResponse.Details[0]
		switch errDetail.Issue {
		// 如果订单已经被捕获，跳过此类错误并检查订单详情
		case "ORDER_ALREADY_CAPTURED":
			// 跳过处理
		case "ORDER_NOT_APPROVED":
			// 订单未被批准，设置为取消状态
			notifyResult.PaymentStatus = PaymentStateCanceled
			notifyResult.NotifyMessage = errDetail.Description
			return notifyResult, nil
		default:
			// 其他错误
			err = fmt.Errorf(errDetail.Description)
			return nil, err
		}
	}
	// 检查订单详情
	detailRsp, err := pp.Client.OrderDetail(context.Background(), orderId, nil)
	if err != nil {
		return nil, err
	}
	// 检查订单详情响应状态
	if detailRsp.Code != paypal.Success {
		errDetail := detailRsp.ErrorResponse.Details[0]
		switch errDetail.Issue {
		case "ORDER_NOT_APPROVED":
			// 订单未被批准，设置为取消状态
			notifyResult.PaymentStatus = PaymentStateCanceled
			notifyResult.NotifyMessage = errDetail.Description
			return notifyResult, nil
		default:
			// 其他错误
			err = fmt.Errorf(errDetail.Description)
			return nil, err
		}
	}

	// 解析订单详情
	paymentName := detailRsp.Response.Id
	// 解析价格
	price, err := strconv.ParseFloat(detailRsp.Response.PurchaseUnits[0].Amount.Value, 64)
	if err != nil {
		return nil, err
	}
	// 获取货币代码
	currency := detailRsp.Response.PurchaseUnits[0].Amount.CurrencyCode
	// 解析产品信息
	productDisplayName, productName, providerName, err := parseAttachString(detailRsp.Response.PurchaseUnits[0].Description)
	if err != nil {
		return nil, err
	}
	// TODO: 更好的状态处理，例如处理挂起状态
	// 根据订单状态设置支付状态
	var paymentStatus PaymentState
	switch detailRsp.Response.Status { // 可能的状态：CREATED、SAVED、APPROVED、VOIDED、COMPLETED、PAYER_ACTION_REQUIRED
	case "COMPLETED":
		// 订单已完成
		paymentStatus = PaymentStatePaid
	default:
		// 其他状态视为错误
		paymentStatus = PaymentStateError
	}
	// 构建通知结果
	notifyResult = &NotifyResult{
		PaymentStatus:      paymentStatus,      // 支付状态
		PaymentName:        paymentName,        // 支付名称
		ProductName:        productName,        // 产品名称
		ProductDisplayName: productDisplayName, // 产品显示名称
		ProviderName:       providerName,       // 提供者名称
		Price:              price,              // 价格
		Currency:           currency,           // 货币

		OrderId: orderId, // 订单ID
	}
	return notifyResult, nil
}

// GetInvoice 获取发票信息
// paymentName: 支付名称
// personName: 个人姓名
// personIdCard: 身份证号
// personEmail: 邮箱
// personPhone: 电话
// invoiceType: 发票类型
// invoiceTitle: 发票抬头
// invoiceTaxId: 税号
// 返回发票URL和可能的错误
func (pp *PaypalPaymentProvider) GetInvoice(paymentName string, personName string, personIdCard string, personEmail string, personPhone string, invoiceType string, invoiceTitle string, invoiceTaxId string) (string, error) {
	// PayPal暂不支持发票功能
	return "", nil
}

// GetResponseError 获取响应错误信息
// err: 错误对象
// 返回错误描述字符串
func (pp *PaypalPaymentProvider) GetResponseError(err error) string {
	if err == nil {
		return "success" // 成功
	} else {
		return "fail" // 失败
	}
}

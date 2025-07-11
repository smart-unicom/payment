// Package payment 支付相关功能
package payment

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
)

// AlipayPaymentProvider 支付宝支付提供商
// 实现支付宝支付功能
type AlipayPaymentProvider struct {
	Client *alipay.Client // 支付宝客户端
}

// NewAlipayPaymentProvider 创建新的支付宝支付提供商实例
// 参数:
//   - appId: 应用ID
//   - appCertificate: 应用证书
//   - appPrivateKey: 应用私钥
//   - authorityPublicKey: 支付宝公钥
//   - authorityRootPublicKey: 支付宝根证书
// 返回:
//   - *AlipayPaymentProvider: 支付宝支付提供商实例
//   - error: 错误信息
func NewAlipayPaymentProvider(appId string, appCertificate string, appPrivateKey string, authorityPublicKey string, authorityRootPublicKey string) (*AlipayPaymentProvider, error) {
	// 参数映射说明:
	// clientId => appId
	// cert.Certificate => appCertificate
	// cert.PrivateKey => appPrivateKey
	// rootCert.Certificate => authorityPublicKey
	// rootCert.PrivateKey => authorityRootPublicKey
	pp := &AlipayPaymentProvider{}

	// 创建支付宝客户端
	client, err := alipay.NewClient(appId, appPrivateKey, true)
	if err != nil {
		return nil, err
	}

	// 设置证书
	err = client.SetCertSnByContent([]byte(appCertificate), []byte(authorityRootPublicKey), []byte(authorityPublicKey))
	if err != nil {
		return nil, err
	}

	pp.Client = client
	return pp, nil
}

// Pay 执行支付宝支付操作
// 参数:
//   - r: 支付请求信息
// 返回:
//   - *PayResp: 支付响应信息
//   - error: 错误信息
func (pp *AlipayPaymentProvider) Pay(r *PayReq) (*PayResp, error) {
	// 可选：开启调试模式
	// pp.Client.DebugSwitch = gopay.DebugOn
	bm := gopay.BodyMap{}
	
	// 设置回调URL
	pp.Client.SetReturnUrl(r.ReturnUrl)
	pp.Client.SetNotifyUrl(r.NotifyUrl)
	
	// 设置支付参数
	bm.Set("subject", joinAttachString([]string{r.ProductName, r.ProductDisplayName, r.ProviderName}))
	bm.Set("out_trade_no", r.PaymentName)
	bm.Set("total_amount", priceFloat64ToString(r.Price))

	// 创建支付页面
	payUrl, err := pp.Client.TradePagePay(context.Background(), bm)
	if err != nil {
		return nil, err
	}
	
	// 构造支付响应
	payResp := &PayResp{
		PayUrl:  payUrl,
		OrderId: r.PaymentName,
	}
	return payResp, nil
}

// Notify 处理支付宝支付通知
// 查询订单状态并返回通知结果
// 参数:
//   - body: 通知内容
//   - orderId: 订单ID
// 返回:
//   - *NotifyResult: 通知结果
//   - error: 错误信息
func (pp *AlipayPaymentProvider) Notify(body []byte, orderId string) (*NotifyResult, error) {
	bm := gopay.BodyMap{}
	bm.Set("out_trade_no", orderId)
	
	// 查询交易状态
	aliRsp, err := pp.Client.TradeQuery(context.Background(), bm)
	notifyResult := &NotifyResult{}
	if err != nil {
		// 解析错误响应
		errRsp := &alipay.ErrorResponse{}
		unmarshalErr := json.Unmarshal([]byte(err.Error()), errRsp)
		if unmarshalErr != nil {
			return nil, err
		}
		// 如果交易不存在，标记为已取消
		if errRsp.SubCode == "ACQ.TRADE_NOT_EXIST" {
			notifyResult.PaymentStatus = PaymentStateCanceled
			return notifyResult, nil
		}
		return nil, err
	}
	
	// 根据交易状态设置支付状态
	switch aliRsp.Response.TradeStatus {
	case "WAIT_BUYER_PAY": // 等待买家付款
		notifyResult.PaymentStatus = PaymentStateCreated
		return notifyResult, nil
	case "TRADE_CLOSED": // 交易关闭
		notifyResult.PaymentStatus = PaymentStateTimeout
		return notifyResult, nil
	case "TRADE_SUCCESS": // 交易成功
		// 继续处理
	default: // 未知状态
		notifyResult.PaymentStatus = PaymentStateError
		notifyResult.NotifyMessage = fmt.Sprintf("unexpected alipay trade state: %v", aliRsp.Response.TradeStatus)
		return notifyResult, nil
	}
	
	// 解析产品信息
	productDisplayName, productName, providerName, _ := parseAttachString(aliRsp.Response.Subject)
	
	// 构造通知结果
	notifyResult = &NotifyResult{
		ProductName:        productName,
		ProductDisplayName: productDisplayName,
		ProviderName:       providerName,
		OrderId:            orderId,
		PaymentStatus:      PaymentStatePaid,
		Price:              priceStringToFloat64(aliRsp.Response.TotalAmount),
		PaymentName:        orderId,
	}
	return notifyResult, nil
}

// GetInvoice 获取支付宝发票
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
func (pp *AlipayPaymentProvider) GetInvoice(paymentName string, personName string, personIdCard string, personEmail string, personPhone string, invoiceType string, invoiceTitle string, invoiceTaxId string) (string, error) {
	return "", nil
}

// GetResponseError 获取支付宝响应错误信息
// 根据错误状态返回相应的字符串
// 参数:
//   - err: 错误对象
// 返回:
//   - string: 错误响应字符串
func (pp *AlipayPaymentProvider) GetResponseError(err error) string {
	if err == nil {
		return "success" // 成功
	} else {
		return "fail" // 失败
	}
}

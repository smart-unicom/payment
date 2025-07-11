// payment 包提供了多种支付方式的统一接口实现
package payment

import (
	"fmt"
)

// BalancePaymentProvider 余额支付提供者结构体
type BalancePaymentProvider struct{}

// NewBalancePaymentProvider 创建新的余额支付提供者实例
// 返回余额支付提供者实例和可能的错误
func NewBalancePaymentProvider() (*BalancePaymentProvider, error) {
	pp := &BalancePaymentProvider{}
	return pp, nil
}

// Pay 处理余额支付请求
// r: 支付请求参数
// 返回支付响应和可能的错误
func (pp *BalancePaymentProvider) Pay(r *PayReq) (*PayResp, error) {
	// 从支付者ID中获取所有者信息
	owner, _ := GetOwnerAndNameFromId(r.PayerId)
	return &PayResp{
		PayUrl:  r.ReturnUrl,                                // 直接返回到返回URL
		OrderId: fmt.Sprintf("%s/%s", owner, r.PaymentName), // 构建订单ID
	}, nil
}

// Notify 处理余额支付回调通知
// body: 回调请求体
// orderId: 订单ID
// 返回通知结果和可能的错误
func (pp *BalancePaymentProvider) Notify(body []byte, orderId string) (*NotifyResult, error) {
	// 余额支付直接返回支付成功状态
	return &NotifyResult{
		PaymentStatus: PaymentStatePaid, // 支付状态为已支付
	}, nil
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
func (pp *BalancePaymentProvider) GetInvoice(paymentName string, personName string, personIdCard string, personEmail string, personPhone string, invoiceType string, invoiceTitle string, invoiceTaxId string) (string, error) {
	// 余额支付暂不支持发票功能
	return "", nil
}

// GetResponseError 获取响应错误信息
// err: 错误对象
// 返回错误描述字符串
func (pp *BalancePaymentProvider) GetResponseError(err error) string {
	// 余额支付始终返回空字符串
	return ""
}

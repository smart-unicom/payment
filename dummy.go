// Package payment 支付相关功能
package payment

// DummyPaymentProvider 虚拟支付提供商
// 用于测试和开发环境的模拟支付
type DummyPaymentProvider struct{}

// NewDummyPaymentProvider 创建新的虚拟支付提供商实例
// 返回:
//   - *DummyPaymentProvider: 虚拟支付提供商实例
//   - error: 错误信息
func NewDummyPaymentProvider() (*DummyPaymentProvider, error) {
	pp := &DummyPaymentProvider{}
	return pp, nil
}

// Pay 执行虚拟支付操作
// 直接返回成功响应，用于测试
// 参数:
//   - r: 支付请求信息
// 返回:
//   - *PayResp: 支付响应信息
//   - error: 错误信息
func (pp *DummyPaymentProvider) Pay(r *PayReq) (*PayResp, error) {
	return &PayResp{
		PayUrl: r.ReturnUrl,
	}, nil
}

// Notify 处理虚拟支付通知
// 直接返回支付成功状态
// 参数:
//   - body: 通知内容
//   - orderId: 订单ID
// 返回:
//   - *NotifyResult: 通知结果
//   - error: 错误信息
func (pp *DummyPaymentProvider) Notify(body []byte, orderId string) (*NotifyResult, error) {
	return &NotifyResult{
		PaymentStatus: PaymentStatePaid,
	}, nil
}

// GetInvoice 获取虚拟发票
// 返回空字符串，不支持发票功能
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
func (pp *DummyPaymentProvider) GetInvoice(paymentName string, personName string, personIdCard string, personEmail string, personPhone string, invoiceType string, invoiceTitle string, invoiceTaxId string) (string, error) {
	return "", nil
}

// GetResponseError 获取虚拟响应错误信息
// 返回空字符串
// 参数:
//   - err: 错误对象
// 返回:
//   - string: 错误响应字符串（空）
func (pp *DummyPaymentProvider) GetResponseError(err error) string {
	return ""
}

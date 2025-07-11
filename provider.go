// Package payment 支付相关功能
// 提供多种支付方式的统一接口，包括支付宝、微信支付、Stripe等
package payment

// PaymentState 支付状态类型
type PaymentState string

// 支付状态常量定义
const (
	PaymentStatePaid     PaymentState = "Paid"     // 已支付
	PaymentStateCreated  PaymentState = "Created"  // 已创建
	PaymentStateCanceled PaymentState = "Canceled" // 已取消
	PaymentStateTimeout  PaymentState = "Timeout"  // 超时
	PaymentStateError    PaymentState = "Error"    // 错误
)

// 支付环境常量定义
const (
	PaymentEnvWechatBrowser = "WechatBrowser" // 微信浏览器环境
)

// PayReq 支付请求结构体
// 包含支付所需的所有参数信息
type PayReq struct {
	ProviderName       string  // 支付提供商名称
	ProductName        string  // 产品名称
	PayerName          string  // 付款人姓名
	PayerId            string  // 付款人ID
	PayerEmail         string  // 付款人邮箱
	PaymentName        string  // 支付名称
	ProductDisplayName string  // 产品显示名称
	ProductDescription string  // 产品描述
	ProductImage       string  // 产品图片
	Price              float64 // 价格
	Currency           string  // 货币类型

	ReturnUrl string // 返回URL
	NotifyUrl string // 通知URL

	PaymentEnv string // 支付环境
}

// PayResp 支付响应结构体
// 包含支付后返回的信息
type PayResp struct {
	PayUrl     string                 // 支付URL
	OrderId    string                 // 订单ID
	AttachInfo map[string]interface{} // 附加信息
}

// NotifyResult 支付通知结果结构体
// 包含支付通知回调的结果信息
type NotifyResult struct {
	PaymentName   string       // 支付名称
	PaymentStatus PaymentState // 支付状态
	NotifyMessage string       // 通知消息

	ProductName        string  // 产品名称
	ProductDisplayName string  // 产品显示名称
	ProviderName       string  // 支付提供商名称
	Price              float64 // 价格
	Currency           string  // 货币类型

	OrderId string // 订单ID
}

// PaymentProvider 支付提供商接口
// 定义所有支付提供商必须实现的方法
type PaymentProvider interface {
	// Pay 执行支付操作
	// 参数:
	//   - req: 支付请求信息
	// 返回:
	//   - *PayResp: 支付响应信息
	//   - error: 错误信息
	Pay(req *PayReq) (*PayResp, error)
	
	// Notify 处理支付通知
	// 参数:
	//   - body: 通知内容
	//   - orderId: 订单ID
	// 返回:
	//   - *NotifyResult: 通知结果
	//   - error: 错误信息
	Notify(body []byte, orderId string) (*NotifyResult, error)
	
	// GetInvoice 获取发票
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
	//   - string: 发票信息
	//   - error: 错误信息
	GetInvoice(paymentName string, personName string, personIdCard string, personEmail string, personPhone string, invoiceType string, invoiceTitle string, invoiceTaxId string) (string, error)
	
	// GetResponseError 获取响应错误信息
	// 参数:
	//   - err: 错误对象
	// 返回:
	//   - string: 错误响应字符串
	GetResponseError(err error) string
}

// payment 包提供了多种支付方式的统一接口实现
package payment

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/casdoor/casdoor/util"
)

// GcPaymentProvider GC支付提供者结构体
type GcPaymentProvider struct {
	Xmpch     string // 商户号
	SecretKey string // 密钥
	Host      string // 主机地址
}

// GcPayReqInfo GC支付请求信息结构体
type GcPayReqInfo struct {
	OrderDate string `json:"orderdate"` // 订单日期
	OrderNo   string `json:"orderno"`   // 订单号
	Amount    string `json:"amount"`    // 金额
	Xmpch     string `json:"xmpch"`     // 商户号
	Body      string `json:"body"`      // 商品描述
	ReturnUrl string `json:"return_url"` // 返回URL
	NotifyUrl string `json:"notify_url"` // 通知URL
	PayerId   string `json:"payerid"`   // 支付者ID
	PayerName string `json:"payername"` // 支付者姓名
	Remark1   string `json:"remark1"`   // 备注1
	Remark2   string `json:"remark2"`   // 备注2
}

// GcPayRespInfo GC支付响应信息结构体
type GcPayRespInfo struct {
	Jylsh     string `json:"jylsh"`     // 交易流水号
	Amount    string `json:"amount"`    // 金额
	PayerId   string `json:"payerid"`   // 支付者ID
	PayerName string `json:"payername"` // 支付者姓名
	PayUrl    string `json:"payurl"`    // 支付URL
}

// GcNotifyRespInfo GC通知响应信息结构体
type GcNotifyRespInfo struct {
	Xmpch      string  `json:"xmpch"`      // 商户号
	OrderDate  string  `json:"orderdate"`  // 订单日期
	OrderNo    string  `json:"orderno"`    // 订单号
	Amount     float64 `json:"amount"`     // 金额
	Jylsh      string  `json:"jylsh"`      // 交易流水号
	TradeNo    string  `json:"tradeno"`    // 交易号
	PayMethod  string  `json:"paymethod"`  // 支付方式
	OrderState string  `json:"orderstate"` // 订单状态
	ReturnType string  `json:"return_type"` // 返回类型
	PayerId    string  `json:"payerid"`    // 支付者ID
	PayerName  string  `json:"payername"`  // 支付者姓名
}

// GcRequestBody GC请求体结构体
type GcRequestBody struct {
	Op          string `json:"op"`          // 操作类型
	Xmpch       string `json:"xmpch"`       // 商户号
	Version     string `json:"version"`     // 版本号
	Data        string `json:"data"`        // 数据（Base64编码）
	RequestTime string `json:"requesttime"` // 请求时间
	Sign        string `json:"sign"`        // 签名
}

// GcResponseBody GC响应体结构体
type GcResponseBody struct {
	Op         string `json:"op"`          // 操作类型
	Xmpch      string `json:"xmpch"`       // 商户号
	Version    string `json:"version"`     // 版本号
	ReturnCode string `json:"return_code"` // 返回码
	ReturnMsg  string `json:"return_msg"`  // 返回消息
	Data       string `json:"data"`        // 数据（Base64编码）
	NotifyTime string `json:"notifytime"`  // 通知时间
	Sign       string `json:"sign"`        // 签名
}

// GcInvoiceReqInfo GC发票请求信息结构体
type GcInvoiceReqInfo struct {
	BusNo        string `json:"busno"`        // 业务号
	PayerName    string `json:"payername"`    // 支付者姓名
	IdNum        string `json:"idnum"`        // 身份证号
	PayerType    string `json:"payertype"`    // 支付者类型
	InvoiceTitle string `json:"invoicetitle"` // 发票抬头
	Tin          string `json:"tin"`          // 税号
	Phone        string `json:"phone"`        // 电话
	Email        string `json:"email"`        // 邮箱
}

// GcInvoiceRespInfo GC发票响应信息结构体
type GcInvoiceRespInfo struct {
	BusNo     string `json:"busno"`     // 业务号
	State     string `json:"state"`     // 状态
	EbillCode string `json:"ebillcode"` // 电子票据代码
	EbillNo   string `json:"ebillno"`   // 电子票据号码
	CheckCode string `json:"checkcode"` // 校验码
	Url       string `json:"url"`       // 发票URL
	Content   string `json:"content"`   // 内容
}

// NewGcPaymentProvider 创建新的GC支付提供者实例
// clientId: 客户端ID（商户号）
// clientSecret: 客户端密钥
// host: 主机地址
// 返回GC支付提供者实例
func NewGcPaymentProvider(clientId string, clientSecret string, host string) *GcPaymentProvider {
	pp := &GcPaymentProvider{}

	pp.Xmpch = clientId      // 设置商户号
	pp.SecretKey = clientSecret // 设置密钥
	pp.Host = host           // 设置主机地址
	return pp
}

// doPost 执行POST请求
// postBytes: 请求体字节数组
// 返回响应字节数组和可能的错误
func (pp *GcPaymentProvider) doPost(postBytes []byte) ([]byte, error) {
	client := &http.Client{}

	var resp *http.Response
	var err error

	// 设置请求内容类型
	contentType := "text/plain;charset=UTF-8"
	body := bytes.NewReader(postBytes)

	// 创建POST请求
	req, err := http.NewRequest("POST", pp.Host, body)
	if err != nil {
		return nil, err
	}

	// 设置请求头
	req.Header.Set("Content-Type", contentType)

	// 执行请求
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	// 确保响应体被关闭
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	// 读取响应体
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return respBytes, nil
}

// Pay 处理GC支付请求
// r: 支付请求参数
// 返回支付响应和可能的错误
func (pp *GcPaymentProvider) Pay(r *PayReq) (*PayResp, error) {
	// 构建支付请求信息
	payReqInfo := GcPayReqInfo{
		OrderDate: util.GenerateSimpleTimeId(), // 生成订单日期
		OrderNo:   r.PaymentName,               // 订单号
		Amount:    getPriceString(r.Price),     // 金额
		Xmpch:     pp.Xmpch,                    // 商户号
		Body:      r.ProductDisplayName,        // 商品描述
		ReturnUrl: r.ReturnUrl,                 // 返回URL
		NotifyUrl: r.NotifyUrl,                 // 通知URL
		Remark1:   r.PayerName,                 // 备注1：支付者姓名
		Remark2:   r.ProductName,               // 备注2：产品名称
	}

	// 序列化支付请求信息
	b, err := json.Marshal(payReqInfo)
	if err != nil {
		return nil, err
	}

	// 构建请求体
	body := GcRequestBody{
		Op:          "OrderCreate",                           // 操作类型：创建订单
		Xmpch:       pp.Xmpch,                               // 商户号
		Version:     "1.4",                                  // 版本号
		Data:        base64.StdEncoding.EncodeToString(b),   // Base64编码的数据
		RequestTime: util.GenerateSimpleTimeId(),            // 请求时间
	}

	// 生成签名参数字符串
	params := fmt.Sprintf("data=%s&op=%s&requesttime=%s&version=%s&xmpch=%s%s", body.Data, body.Op, body.RequestTime, body.Version, body.Xmpch, pp.SecretKey)
	// 计算MD5签名并转为大写
	body.Sign = strings.ToUpper(util.GetMd5Hash(params))

	// 序列化请求体
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	// 发送POST请求
	respBytes, err := pp.doPost(bodyBytes)
	if err != nil {
		return nil, err
	}

	// 解析响应体
	var respBody GcResponseBody
	err = json.Unmarshal(respBytes, &respBody)
	if err != nil {
		return nil, err
	}

	// 检查返回码
	if respBody.ReturnCode != "SUCCESS" {
		return nil, fmt.Errorf("%s: %s", respBody.ReturnCode, respBody.ReturnMsg)
	}

	// 解码响应数据
	payRespInfoBytes, err := base64.StdEncoding.DecodeString(respBody.Data)
	if err != nil {
		return nil, err
	}

	// 解析支付响应信息
	var payRespInfo GcPayRespInfo
	err = json.Unmarshal(payRespInfoBytes, &payRespInfo)
	if err != nil {
		return nil, err
	}
	// 构建支付响应
	payResp := &PayResp{
		PayUrl: payRespInfo.PayUrl, // 支付URL
	}
	return payResp, nil
}

// Notify 处理GC支付的回调通知
// body: 通知请求体字节数组
// orderId: 订单ID
// 返回通知结果和可能的错误
func (pp *GcPaymentProvider) Notify(body []byte, orderId string) (*NotifyResult, error) {
	reqBody := GcRequestBody{}
	// 解析URL编码的请求体
	m, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, err
	}

	// 提取请求参数
	reqBody.Op = m["op"][0]              // 操作类型
	reqBody.Xmpch = m["xmpch"][0]        // 商户号
	reqBody.Version = m["version"][0]    // 版本号
	reqBody.Data = m["data"][0]          // 数据
	reqBody.RequestTime = m["requesttime"][0] // 请求时间
	reqBody.Sign = m["sign"][0]          // 签名

	// 解码Base64数据
	notifyReqInfoBytes, err := base64.StdEncoding.DecodeString(reqBody.Data)
	if err != nil {
		return nil, err
	}

	// 解析通知响应信息
	var notifyRespInfo GcNotifyRespInfo
	err = json.Unmarshal(notifyReqInfoBytes, &notifyRespInfo)
	if err != nil {
		return nil, err
	}

	// 初始化响应字段
	providerName := ""       // 提供者名称
	productName := ""        // 产品名称

	productDisplayName := "" // 产品显示名称
	paymentName := notifyRespInfo.OrderNo // 支付名称（订单号）
	price := notifyRespInfo.Amount        // 价格

	// 检查订单状态，"1"表示支付成功
	if notifyRespInfo.OrderState != "1" {
		return nil, fmt.Errorf("error order state: %s", notifyRespInfo.OrderDate)
	}
	// 构建通知结果
	notifyResult := &NotifyResult{
		ProductName:        productName,        // 产品名称
		ProductDisplayName: productDisplayName, // 产品显示名称
		ProviderName:       providerName,       // 提供者名称
		OrderId:            orderId,            // 订单ID
		Price:              price,              // 价格
		PaymentStatus:      PaymentStatePaid,   // 支付状态：已支付
		PaymentName:        paymentName,        // 支付名称
	}
	return notifyResult, nil
}

// GetInvoice 获取GC支付的发票
// 参数:
//   - paymentName: 支付名称（订单号）
//   - personName: 个人姓名
//   - personIdCard: 个人身份证号
//   - personEmail: 个人邮箱
//   - personPhone: 个人电话
//   - invoiceType: 发票类型
//   - invoiceTitle: 发票抬头
//   - invoiceTaxId: 发票税号
// 返回值:
//   - string: 发票URL
//   - error: 错误信息
func (pp *GcPaymentProvider) GetInvoice(paymentName string, personName string, personIdCard string, personEmail string, personPhone string, invoiceType string, invoiceTitle string, invoiceTaxId string) (string, error) {
	// 设置支付者类型，默认为个人(0)，组织为1
	payerType := "0"
	if invoiceType == "Organization" {
		payerType = "1"
	}

	// 构建发票请求信息
	invoiceReqInfo := GcInvoiceReqInfo{
		BusNo:        paymentName,   // 业务号（支付名称）
		PayerName:    personName,    // 支付者姓名
		IdNum:        personIdCard,  // 身份证号
		PayerType:    payerType,     // 支付者类型
		InvoiceTitle: invoiceTitle,  // 发票抬头
		Tin:          invoiceTaxId,  // 税号
		Phone:        personPhone,   // 电话
		Email:        personEmail,   // 邮箱
	}

	// 序列化发票请求信息
	b, err := json.Marshal(invoiceReqInfo)
	if err != nil {
		return "", err
	}

	// 构建请求体
	body := GcRequestBody{
		Op:          "InvoiceEBillByOrder",                 // 操作类型：按订单开具电子票据
		Xmpch:       pp.Xmpch,                            // 商户号
		Version:     "1.4",                              // 版本号
		Data:        base64.StdEncoding.EncodeToString(b), // Base64编码的数据
		RequestTime: util.GenerateSimpleTimeId(),         // 请求时间
	}

	// 生成签名参数字符串
	params := fmt.Sprintf("data=%s&op=%s&requesttime=%s&version=%s&xmpch=%s%s", body.Data, body.Op, body.RequestTime, body.Version, body.Xmpch, pp.SecretKey)
	// 计算MD5签名并转为大写
	body.Sign = strings.ToUpper(util.GetMd5Hash(params))

	// 序列化请求体
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	// 发送POST请求
	respBytes, err := pp.doPost(bodyBytes)
	if err != nil {
		return "", err
	}

	// 解析响应体
	var respBody GcResponseBody
	err = json.Unmarshal(respBytes, &respBody)
	if err != nil {
		return "", err
	}

	// 检查返回码
	if respBody.ReturnCode != "SUCCESS" {
		return "", fmt.Errorf("%s: %s", respBody.ReturnCode, respBody.ReturnMsg)
	}

	// 解码响应数据
	invoiceRespInfoBytes, err := base64.StdEncoding.DecodeString(respBody.Data)
	if err != nil {
		return "", err
	}

	// 解析发票响应信息
	var invoiceRespInfo GcInvoiceRespInfo
	err = json.Unmarshal(invoiceRespInfoBytes, &invoiceRespInfo)
	if err != nil {
		return "", err
	}

	// 检查发票状态，"0"表示申请成功但正在开票中
	if invoiceRespInfo.State == "0" {
		return "", fmt.Errorf("申请成功，开票中")
	}

	// 检查发票URL是否为空
	if invoiceRespInfo.Url == "" {
		return "", fmt.Errorf("invoice URL is empty")
	}

	// 返回发票URL
	return invoiceRespInfo.Url, nil
}

// GetResponseError 根据错误信息返回响应状态
// 参数:
//   - err: 错误信息
// 返回值:
//   - string: 响应状态（"success"或"fail"）
func (pp *GcPaymentProvider) GetResponseError(err error) string {
	if err == nil {
		return "success" // 无错误时返回成功
	} else {
		return "fail" // 有错误时返回失败
	}
}

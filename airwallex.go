// payment 包提供了多种支付方式的统一接口实现
package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// AirwallexPaymentProvider Airwallex支付提供者结构体
type AirwallexPaymentProvider struct {
	Client *AirwallexClient // Airwallex客户端实例
}

// NewAirwallexPaymentProvider 创建新的Airwallex支付提供者实例
// clientId: Airwallex客户端ID
// apiKey: Airwallex API密钥
// 返回Airwallex支付提供者实例和可能的错误
func NewAirwallexPaymentProvider(clientId string, apiKey string) (*AirwallexPaymentProvider, error) {
	// 设置API端点和结账页面URL
	apiEndpoint := "https://api.airwallex.com/api/v1"
	apiCheckout := "https://checkout.airwallex.com/#/standalone/checkout?"
	// 创建Airwallex客户端
	client := &AirwallexClient{
		ClientId:    clientId,
		APIKey:      apiKey,
		APIEndpoint: apiEndpoint,
		APICheckout: apiCheckout,
		client:      &http.Client{Timeout: 15 * time.Second}, // 设置15秒超时
	}
	pp := &AirwallexPaymentProvider{
		Client: client,
	}
	return pp, nil
}

// Pay 处理Airwallex支付请求
// r: 支付请求参数
// 返回支付响应和可能的错误
func (pp *AirwallexPaymentProvider) Pay(r *PayReq) (*PayResp, error) {
	// 创建支付意图
	intent, err := pp.Client.CreateIntent(r)
	if err != nil {
		return nil, err
	}
	// 获取结账页面URL
	payUrl, err := pp.Client.GetCheckoutUrl(intent, r)
	if err != nil {
		return nil, err
	}
	return &PayResp{
		PayUrl:  payUrl,                  // 支付URL
		OrderId: intent.MerchantOrderId, // 商户订单ID
	}, nil
}

// Notify 处理Airwallex支付回调通知
// body: 回调请求体
// orderId: 订单ID
// 返回通知结果和可能的错误
func (pp *AirwallexPaymentProvider) Notify(body []byte, orderId string) (*NotifyResult, error) {
	notifyResult := &NotifyResult{}
	// 根据订单ID获取支付意图
	intent, err := pp.Client.GetIntentByOrderId(orderId)
	if err != nil {
		return nil, err
	}
	// 检查支付意图状态
	switch intent.Status {
	case "PENDING", "REQUIRES_PAYMENT_METHOD", "REQUIRES_CUSTOMER_ACTION", "REQUIRES_CAPTURE":
		// 支付进行中的各种状态
		notifyResult.PaymentStatus = PaymentStateCreated
		return notifyResult, nil
	case "CANCELLED":
		// 支付已取消
		notifyResult.PaymentStatus = PaymentStateCanceled
		return notifyResult, nil
	case "EXPIRED":
		// 支付已过期
		notifyResult.PaymentStatus = PaymentStateTimeout
		return notifyResult, nil
	case "SUCCEEDED":
		// 支付成功，继续处理
	default:
		// 未知状态，视为错误
		notifyResult.PaymentStatus = PaymentStateError
		notifyResult.NotifyMessage = fmt.Sprintf("unexpected airwallex checkout status: %v", intent.Status)
		return notifyResult, nil
	}
	// 检查支付尝试状态
	if intent.PaymentStatus != "" {
		switch intent.PaymentStatus {
		case "CANCELLED", "EXPIRED", "RECEIVED", "AUTHENTICATION_REDIRECTED", "AUTHORIZED", "CAPTURE_REQUESTED":
			// 支付进行中的各种状态
			notifyResult.PaymentStatus = PaymentStateCreated
			return notifyResult, nil
		case "PAID", "SETTLED":
			// 支付已完成，继续处理
		default:
			// 未知支付状态，视为错误
			notifyResult.PaymentStatus = PaymentStateError
			notifyResult.NotifyMessage = fmt.Sprintf("unexpected airwallex checkout payment status: %v", intent.PaymentStatus)
			return notifyResult, nil
		}
	}
	// 支付已成功完成
	var productDisplayName, productName, providerName string
	// 从元数据中解析产品信息
	if description, ok := intent.Metadata["description"]; ok {
		productName, productDisplayName, providerName, _ = parseAttachString(description.(string))
	}
	orderId = intent.MerchantOrderId
	// 构建通知结果
	return &NotifyResult{
		PaymentName:        orderId,                                     // 支付名称
		PaymentStatus:      PaymentStatePaid,                           // 支付状态为已支付
		ProductName:        productName,                                 // 产品名称
		ProductDisplayName: productDisplayName,                         // 产品显示名称
		ProviderName:       providerName,                               // 提供者名称
		Price:              priceStringToFloat64(intent.Amount.String()), // 价格
		Currency:           intent.Currency,                            // 货币
		OrderId:            orderId,                                     // 订单ID
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
func (pp *AirwallexPaymentProvider) GetInvoice(paymentName, personName, personIdCard, personEmail, personPhone, invoiceType, invoiceTitle, invoiceTaxId string) (string, error) {
	// Airwallex暂不支持发票功能
	return "", nil
}

// GetResponseError 获取响应错误信息
// err: 错误对象
// 返回错误描述字符串
func (pp *AirwallexPaymentProvider) GetResponseError(err error) string {
	if err == nil {
		return "success" // 成功
	}
	return "fail" // 失败
}

/*
 * Airwallex客户端实现（官方SDK发布后将被移除）
 */

// AirwallexClient Airwallex客户端结构体
type AirwallexClient struct {
	ClientId    string                // 客户端ID
	APIKey      string                // API密钥
	APIEndpoint string                // API端点
	APICheckout string                // 结账页面URL
	client      *http.Client          // HTTP客户端
	tokenCache  *AirWallexTokenInfo   // 令牌缓存
	tokenMutex  sync.RWMutex          // 令牌读写锁
}

// AirWallexTokenInfo Airwallex令牌信息结构体
type AirWallexTokenInfo struct {
	Token           string    `json:"token"`      // 访问令牌
	ExpiresAt       string    `json:"expires_at"` // 过期时间字符串
	parsedExpiresAt time.Time                     // 解析后的过期时间
}

// AirWallexIntentResp Airwallex支付意图响应结构体
type AirWallexIntentResp struct {
	Id              string `json:"id"`                // 支付意图ID
	ClientSecret    string `json:"client_secret"`    // 客户端密钥
	MerchantOrderId string `json:"merchant_order_id"` // 商户订单ID
}

func (c *AirwallexClient) GetToken() (string, error) {
	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()
	if c.tokenCache != nil && time.Now().Before(c.tokenCache.parsedExpiresAt) {
		return c.tokenCache.Token, nil
	}
	req, _ := http.NewRequest("POST", c.APIEndpoint+"/authentication/login", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("x-client-id", c.ClientId)
	req.Header.Set("x-api-key", c.APIKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result AirWallexTokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Token == "" {
		return "", fmt.Errorf("invalid token response")
	}
	expiresAt := strings.Replace(result.ExpiresAt, "+0000", "+00:00", 1)
	result.parsedExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	c.tokenCache = &result
	return result.Token, nil
}

func (c *AirwallexClient) authRequest(method, url string, body interface{}) (map[string]interface{}, error) {
	token, err := c.GetToken()
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(body)
	var reqBody io.Reader
	if method != "GET" {
		reqBody = bytes.NewBuffer(b)
	}
	req, _ := http.NewRequest(method, url, reqBody)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *AirwallexClient) CreateIntent(r *PayReq) (*AirWallexIntentResp, error) {
	description := joinAttachString([]string{r.ProductName, r.ProductDisplayName, r.ProviderName})
	orderId := r.PaymentName
	intentReq := map[string]interface{}{
		"currency":          r.Currency,
		"amount":            r.Price,
		"merchant_order_id": orderId,
		"request_id":        orderId,
		"descriptor":        strings.ReplaceAll(string([]rune(description)[:32]), "\x00", ""),
		"metadata":          map[string]interface{}{"description": description},
		"order":             map[string]interface{}{"products": []map[string]interface{}{{"name": r.ProductDisplayName, "quantity": 1, "desc": r.ProductDescription, "image_url": r.ProductImage}}},
		"customer":          map[string]interface{}{"merchant_customer_id": r.PayerId, "email": r.PayerEmail, "first_name": r.PayerName, "last_name": r.PayerName},
	}
	intentUrl := fmt.Sprintf("%s/pa/payment_intents/create", c.APIEndpoint)
	intentRes, err := c.authRequest("POST", intentUrl, intentReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment intent: %v", err)
	}
	return &AirWallexIntentResp{
		Id:              intentRes["id"].(string),
		ClientSecret:    intentRes["client_secret"].(string),
		MerchantOrderId: intentRes["merchant_order_id"].(string),
	}, nil
}

type AirwallexIntent struct {
	Amount               json.Number `json:"amount"`
	Currency             string      `json:"currency"`
	Id                   string      `json:"id"`
	Status               string      `json:"status"`
	Descriptor           string      `json:"descriptor"`
	MerchantOrderId      string      `json:"merchant_order_id"`
	LatestPaymentAttempt struct {
		Status string `json:"status"`
	} `json:"latest_payment_attempt"`
	Metadata map[string]interface{} `json:"metadata"`
}

type AirwallexIntents struct {
	Items []AirwallexIntent `json:"items"`
}

type AirWallexIntentInfo struct {
	Amount          json.Number
	Currency        string
	Id              string
	Status          string
	Descriptor      string
	MerchantOrderId string
	PaymentStatus   string
	Metadata        map[string]interface{}
}

func (c *AirwallexClient) GetIntentByOrderId(orderId string) (*AirWallexIntentInfo, error) {
	intentUrl := fmt.Sprintf("%s/pa/payment_intents/?merchant_order_id=%s", c.APIEndpoint, orderId)
	intentRes, err := c.authRequest("GET", intentUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment intent: %v", err)
	}
	items := intentRes["items"].([]interface{})
	if len(items) == 0 {
		return nil, fmt.Errorf("no payment intent found for order id: %s", orderId)
	}
	var intent AirwallexIntent
	if b, err := json.Marshal(items[0]); err == nil {
		json.Unmarshal(b, &intent)
	}
	return &AirWallexIntentInfo{
		Id:              intent.Id,
		Amount:          intent.Amount,
		Currency:        intent.Currency,
		Status:          intent.Status,
		Descriptor:      intent.Descriptor,
		MerchantOrderId: intent.MerchantOrderId,
		PaymentStatus:   intent.LatestPaymentAttempt.Status,
		Metadata:        intent.Metadata,
	}, nil
}

func (c *AirwallexClient) GetCheckoutUrl(intent *AirWallexIntentResp, r *PayReq) (string, error) {
	return fmt.Sprintf("%sintent_id=%s&client_secret=%s&mode=payment&currency=%s&amount=%v&requiredBillingContactFields=%s&successUrl=%s&failUrl=%s&logoUrl=%s",
		c.APICheckout,
		intent.Id,
		intent.ClientSecret,
		r.Currency,
		r.Price,
		url.QueryEscape(`["address"]`),
		r.ReturnUrl,
		r.ReturnUrl,
		"data:image/gif;base64,R0lGODlhAQABAAD/ACwAAAAAAQABAAACADs=", // replace default logo
	), nil
}

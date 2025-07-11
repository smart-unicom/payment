// Package payment 支付相关功能
package payment

import (
	"context"
	"errors"
	"fmt"

	"github.com/casdoor/casdoor/util"
	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/wechat/v3"
)

// WechatPayNotifyResponse 微信支付通知响应结构体
type WechatPayNotifyResponse struct {
	Code    string `json:"Code"`    // 响应代码
	Message string `json:"Message"` // 响应消息
}

// WechatPaymentProvider 微信支付提供商
// 实现微信支付功能
type WechatPaymentProvider struct {
	Client *wechat.ClientV3 // 微信支付客户端
	AppId  string           // 应用ID
}

// NewWechatPaymentProvider 创建新的微信支付提供商实例
// 参考文档: https://pay.weixin.qq.com/docs/merchant/products/native-payment/preparation.html
// 参数:
//   - mchId: 商户号
//   - apiV3Key: API v3密钥
//   - appId: 应用ID
//   - serialNo: 证书序列号
//   - privateKey: 私钥
//
// 返回:
//   - *WechatPaymentProvider: 微信支付提供商实例
//   - error: 错误信息
func NewWechatPaymentProvider(mchId string, apiV3Key string, appId string, serialNo string, privateKey string) (*WechatPaymentProvider, error) {
	// 参数映射说明:
	// clientId => mchId
	// clientSecret => apiV3Key
	// clientId2 => appId
	// appCertificate => serialNo
	// appPrivateKey => privateKey

	// 检查必要参数
	if appId == "" || mchId == "" || serialNo == "" || apiV3Key == "" || privateKey == "" {
		return &WechatPaymentProvider{}, nil
	}

	// 创建微信支付客户端
	clientV3, err := wechat.NewClientV3(mchId, serialNo, apiV3Key, privateKey)
	if err != nil {
		return nil, err
	}

	// 获取平台证书
	platformCert, serialNo, err := clientV3.GetAndSelectNewestCert()
	if err != nil {
		return nil, err
	}

	// 创建支付提供商实例
	pp := &WechatPaymentProvider{
		Client: clientV3.SetPlatformCert([]byte(platformCert), serialNo),
		AppId:  appId,
	}

	return pp, nil
}

// Pay 执行微信支付操作
// 根据支付环境选择JSAPI或Native支付方式
// 参数:
//   - r: 支付请求信息
//
// 返回:
//   - *PayResp: 支付响应信息
//   - error: 错误信息
func (pp *WechatPaymentProvider) Pay(r *PayReq) (*PayResp, error) {
	bm := gopay.BodyMap{}

	// 构造商品描述信息
	desc := joinAttachString([]string{r.ProductDisplayName, r.ProductName, r.ProviderName})

	// 设置基本支付参数
	bm.Set("attach", desc)
	bm.Set("appid", pp.AppId)
	bm.Set("description", r.ProductDisplayName)
	bm.Set("notify_url", r.NotifyUrl)
	bm.Set("out_trade_no", r.PaymentName)

	// 设置金额信息
	bm.SetBodyMap("amount", func(bm gopay.BodyMap) {
		bm.Set("total", priceFloat64ToInt64(r.Price))
		bm.Set("currency", r.Currency)
	})

	// 在微信浏览器环境中使用JSAPI支付
	if r.PaymentEnv == PaymentEnvWechatBrowser {
		// 检查是否有付款人OpenID
		if r.PayerId == "" {
			return nil, errors.New("failed to get the payer's openid, please retry login")
		}

		// 设置付款人信息
		bm.SetBodyMap("payer", func(bm gopay.BodyMap) {
			// 如果账户是通过微信注册的，PayerId就是微信OpenId，例如：oxW9O1ZDvgreSHuBSQDiQ2F055PI
			bm.Set("openid", r.PayerId)
		})

		// 调用JSAPI支付接口
		jsapiRsp, err := pp.Client.V3TransactionJsapi(context.Background(), bm)
		if err != nil {
			return nil, err
		}
		if jsapiRsp.Code != wechat.Success {
			return nil, errors.New(jsapiRsp.Error)
		}

		// 使用RSA256签名支付请求
		params, err := pp.Client.PaySignOfJSAPI(pp.AppId, jsapiRsp.Response.PrepayId)
		if err != nil {
			return nil, err
		}

		// 构造JSAPI支付响应
		payResp := &PayResp{
			PayUrl:  "",
			OrderId: r.PaymentName, // 微信可以使用paymentName作为OutTradeNo来查询订单状态
			AttachInfo: map[string]interface{}{
				"appId":     params.AppId,
				"timeStamp": params.TimeStamp,
				"nonceStr":  params.NonceStr,
				"package":   params.Package,
				"signType":  "RSA",
				"paySign":   params.PaySign,
			},
		}
		return payResp, nil
	} else {
		// 在其他情况下使用Native支付
		nativeRsp, err := pp.Client.V3TransactionNative(context.Background(), bm)
		if err != nil {
			return nil, err
		}
		if nativeRsp.Code != wechat.Success {
			return nil, errors.New(nativeRsp.Error)
		}

		// 构造Native支付响应
		payResp := &PayResp{
			PayUrl:  nativeRsp.Response.CodeUrl,
			OrderId: r.PaymentName, // 微信可以使用paymentName作为OutTradeNo来查询订单状态
		}
		return payResp, nil
	}
}

// Notify 处理微信支付通知
// 查询订单状态并返回通知结果
// 参数:
//   - body: 通知内容
//   - orderId: 订单ID
//
// 返回:
//   - *NotifyResult: 通知结果
//   - error: 错误信息
func (pp *WechatPaymentProvider) Notify(body []byte, orderId string) (*NotifyResult, error) {
	notifyResult := &NotifyResult{}

	// 查询订单状态
	queryRsp, err := pp.Client.V3TransactionQueryOrder(context.Background(), wechat.OutTradeNo, orderId)
	if err != nil {
		return nil, err
	}
	if queryRsp.Code != wechat.Success {
		return nil, errors.New(queryRsp.Error)
	}

	// 根据交易状态设置支付状态
	switch queryRsp.Response.TradeState {
	case "SUCCESS": // 支付成功
		// 继续处理
	case "CLOSED": // 已关闭
		notifyResult.PaymentStatus = PaymentStateCanceled
		return notifyResult, nil
	case "NOTPAY", "USERPAYING": // 未支付：等待用户支付；用户支付中：用户正在支付
		notifyResult.PaymentStatus = PaymentStateCreated
		return notifyResult, nil
	default: // 未知状态
		notifyResult.PaymentStatus = PaymentStateError
		notifyResult.NotifyMessage = fmt.Sprintf("unexpected wechat trade state: %v", queryRsp.Response.TradeState)
		return notifyResult, nil
	}

	// 解析产品信息
	productDisplayName, productName, providerName, _ := parseAttachString(queryRsp.Response.Attach)

	// 构造通知结果
	notifyResult = &NotifyResult{
		ProductName:        productName,
		ProductDisplayName: productDisplayName,
		ProviderName:       providerName,
		OrderId:            orderId,
		Price:              priceInt64ToFloat64(int64(queryRsp.Response.Amount.Total)),
		PaymentStatus:      PaymentStatePaid,
		PaymentName:        queryRsp.Response.OutTradeNo,
	}
	return notifyResult, nil
}

// GetInvoice 获取微信支付发票
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
//
// 返回:
//   - string: 发票信息（空）
//   - error: 错误信息
func (pp *WechatPaymentProvider) GetInvoice(paymentName string, personName string, personIdCard string, personEmail string, personPhone string, invoiceType string, invoiceTitle string, invoiceTaxId string) (string, error) {
	return "", nil
}

// GetResponseError 获取微信支付响应错误信息
// 根据错误状态构造微信支付通知响应
// 参数:
//   - err: 错误对象
//
// 返回:
//   - string: 错误响应JSON字符串
func (pp *WechatPaymentProvider) GetResponseError(err error) string {
	// 构造响应结构
	response := &WechatPayNotifyResponse{
		Code:    "SUCCESS", // 默认成功
		Message: "",
	}

	// 如果有错误，设置失败状态
	if err != nil {
		response.Code = "FAIL"
		response.Message = err.Error()
	}

	// 转换为JSON字符串
	return util.StructToJson(response)
}

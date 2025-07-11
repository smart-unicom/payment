# Payment 支付组件库

Golang付组件库，提供多种支付方式的统一接口实现，支持国内外主流支付平台,支持支付宝、微信支付、Stripe、PayPal、Airwallex、GC支付等。

## 🚀 特性

- **多支付平台支持**: 支持支付宝、微信支付、Stripe、PayPal、Airwallex、GC支付等
- **统一接口设计**: 所有支付提供商实现相同的接口，便于切换和扩展
- **完整的支付流程**: 支持支付创建、回调通知、订单查询、发票获取等完整流程
- **灵活的配置**: 支持多种支付环境和参数配置
- **详细的中文注释**: 所有代码都有详细的中文注释，便于理解和维护
- **类型安全**: 使用Go语言的类型系统确保代码安全性

## 📦 支持的支付平台

| 支付平台 | 状态 | 支持功能 |
|---------|------|----------|
| 支付宝 (Alipay) | ✅ | 支付、通知、查询 |
| 微信支付 (WeChat Pay) | ✅ | 支付、通知、查询、JSAPI/Native |
| Stripe | ✅ | 支付、通知、查询、发票 |
| PayPal | ✅ | 支付、通知、查询 |
| Airwallex | ✅ | 支付、通知、查询 |
| GC支付 | ✅ | 支付、通知、查询、发票 |
| 余额支付 (Balance) | ✅ | 内部余额扣减 |
| 虚拟支付 (Dummy) | ✅ | 测试和开发环境 |

## 🛠 安装

```bash
go get github.com/smart-unicom/payment
```

## 📖 快速开始

### 基本使用

```go
package main

import (
    "fmt"
    "github.com/smart-unicom/payment"
)

func main() {
    // 创建支付宝支付提供商
    provider, err := payment.NewAlipayPaymentProvider(
        "your_app_id",
        "your_app_certificate", 
        "your_private_key",
        "alipay_public_key",
        "alipay_root_certificate",
    )
    if err != nil {
        panic(err)
    }

    // 创建支付请求
    payReq := &payment.PayReq{
        ProviderName:       "alipay",
        ProductName:        "测试商品",
        ProductDisplayName: "测试商品显示名称",
        PaymentName:        "order_123456",
        Price:              99.99,
        Currency:           "CNY",
        ReturnUrl:          "https://your-domain.com/return",
        NotifyUrl:          "https://your-domain.com/notify",
    }

    // 执行支付
    payResp, err := provider.Pay(payReq)
    if err != nil {
        panic(err)
    }

    fmt.Printf("支付URL: %s\n", payResp.PayUrl)
    fmt.Printf("订单ID: %s\n", payResp.OrderId)
}
```

### 处理支付通知

```go
func handleNotify(provider payment.PaymentProvider, body []byte, orderId string) {
    // 处理支付通知
    result, err := provider.Notify(body, orderId)
    if err != nil {
        fmt.Printf("处理通知失败: %v\n", err)
        return
    }

    // 检查支付状态
    switch result.PaymentStatus {
    case payment.PaymentStatePaid:
        fmt.Println("支付成功")
        // 处理支付成功逻辑
    case payment.PaymentStateFailed:
        fmt.Println("支付失败")
        // 处理支付失败逻辑
    case payment.PaymentStateCanceled:
        fmt.Println("支付取消")
        // 处理支付取消逻辑
    }
}
```

## 🔧 配置说明

### 支付宝配置

```go
provider, err := payment.NewAlipayPaymentProvider(
    "your_app_id",           // 应用ID
    "your_app_certificate",   // 应用证书
    "your_private_key",       // 应用私钥
    "alipay_public_key",      // 支付宝公钥
    "alipay_root_certificate", // 支付宝根证书
)
```

### 微信支付配置

```go
provider, err := payment.NewWechatPaymentProvider(
    "your_mch_id",     // 商户号
    "your_api_v3_key", // API v3密钥
    "your_app_id",     // 应用ID
    "your_serial_no",  // 证书序列号
    "your_private_key", // 私钥
)
```

### Stripe配置

```go
provider, err := payment.NewStripePaymentProvider(
    "your_secret_key",      // 密钥
    "your_endpoint_secret", // 端点密钥
)
```

## 📚 API文档

### PaymentProvider 接口

所有支付提供商都实现以下接口：

```go
type PaymentProvider interface {
    // 执行支付操作
    Pay(req *PayReq) (*PayResp, error)
    
    // 处理支付通知
    Notify(body []byte, orderId string) (*NotifyResult, error)
    
    // 获取发票
    GetInvoice(paymentName, personName, personIdCard, personEmail, 
               personPhone, invoiceType, invoiceTitle, invoiceTaxId string) (string, error)
    
    // 获取响应错误信息
    GetResponseError(err error) string
}
```

### 数据结构

#### PayReq - 支付请求

```go
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
    ReturnUrl          string  // 返回URL
    NotifyUrl          string  // 通知URL
    PaymentEnv         string  // 支付环境
}
```

#### PayResp - 支付响应

```go
type PayResp struct {
    PayUrl     string                 // 支付URL
    OrderId    string                 // 订单ID
    AttachInfo map[string]interface{} // 附加信息
}
```

#### NotifyResult - 通知结果

```go
type NotifyResult struct {
    PaymentName        string       // 支付名称
    PaymentStatus      PaymentState // 支付状态
    NotifyMessage      string       // 通知消息
    ProductName        string       // 产品名称
    ProductDisplayName string       // 产品显示名称
    ProviderName       string       // 支付提供商名称
    Price              float64      // 价格
    Currency           string       // 货币类型
    OrderId            string       // 订单ID
}
```

### 支付状态

```go
const (
    PaymentStatePaid     PaymentState = "Paid"     // 已支付
    PaymentStateCreated  PaymentState = "Created"  // 已创建
    PaymentStateCanceled PaymentState = "Canceled" // 已取消
    PaymentStateTimeout  PaymentState = "Timeout"  // 超时
    PaymentStateError    PaymentState = "Error"    // 错误
)
```

## 🔍 高级用法

### 微信支付环境检测

```go
// 在微信浏览器中使用JSAPI支付
payReq.PaymentEnv = payment.PaymentEnvWechatBrowser
payReq.PayerId = "user_openid" // 用户的OpenID

payResp, err := provider.Pay(payReq)
// payResp.AttachInfo 包含JSAPI支付所需的参数
```

### 发票功能

```go
// 获取发票（支持Stripe和GC支付）
invoiceUrl, err := provider.GetInvoice(
    "payment_name",
    "张三",
    "身份证号",
    "email@example.com",
    "13800138000",
    "Individual", // 或 "Organization"
    "发票抬头",
    "税号",
)
```

## 🧪 测试

使用虚拟支付提供商进行测试：

```go
// 创建虚拟支付提供商（用于测试）
provider, _ := payment.NewDummyPaymentProvider()

// 虚拟支付总是返回成功
payResp, _ := provider.Pay(payReq)
notifyResult, _ := provider.Notify(nil, "test_order")
// notifyResult.PaymentStatus == PaymentStatePaid
```

## 🛡️ 安全注意事项

1. **密钥安全**: 所有API密钥和证书都应该安全存储，不要硬编码在代码中
2. **HTTPS**: 生产环境必须使用HTTPS
3. **签名验证**: 处理支付通知时要验证签名
4. **幂等性**: 支付通知可能重复发送，要确保处理的幂等性
5. **超时处理**: 设置合理的网络请求超时时间

## 🤝 贡献

欢迎提交Issue和Pull Request来改进这个项目。

### 开发指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 📞 支持

如果您在使用过程中遇到问题，请：

1. 查看文档和示例代码
2. 搜索已有的Issues
3. 创建新的Issue描述问题

## 🔗 相关链接

- [支付宝开放平台](https://open.alipay.com/)
- [微信支付开发文档](https://pay.weixin.qq.com/docs/)
- [Stripe API文档](https://stripe.com/docs/api)
- [PayPal开发者文档](https://developer.paypal.com/)
- [Airwallex API文档](https://www.airwallex.com/docs/)

---

**注意**: 在生产环境使用前，请仔细阅读各支付平台的官方文档，确保正确配置和使用。

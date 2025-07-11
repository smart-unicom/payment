// Package payment 支付相关功能
package payment

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

// getPriceString 将价格浮点数转换为字符串
// 去除末尾的零和小数点
// 参数:
//   - price: 价格浮点数
//
// 返回:
//   - string: 格式化后的价格字符串
func getPriceString(price float64) string {
	priceString := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", price), "0"), ".")
	return priceString
}

// joinAttachString 将字符串数组用分隔符连接
// 参数:
//   - tokens: 字符串数组
//
// 返回:
//   - string: 用"|"分隔的字符串
func joinAttachString(tokens []string) string {
	return strings.Join(tokens, "|")
}

// parseAttachString 解析附加字符串
// 将用"|"分隔的字符串解析为三个部分
// 参数:
//   - s: 待解析的字符串
//
// 返回:
//   - string: 第一部分
//   - string: 第二部分
//   - string: 第三部分
//   - error: 错误信息
func parseAttachString(s string) (string, string, string, error) {
	tokens := strings.Split(s, "|")
	if len(tokens) != 3 {
		return "", "", "", fmt.Errorf("parseAttachString() error: len(tokens) expected 3, got: %d", len(tokens))
	}
	return tokens[0], tokens[1], tokens[2], nil
}

// priceInt64ToFloat64 将整数价格转换为浮点数价格
// 除以100将分转换为元
// 参数:
//   - price: 整数价格（分）
//
// 返回:
//   - float64: 浮点数价格（元）
func priceInt64ToFloat64(price int64) float64 {
	return float64(price) / 100
}

// priceFloat64ToInt64 将浮点数价格转换为整数价格
// 乘以100将元转换为分
// 参数:
//   - price: 浮点数价格（元）
//
// 返回:
//   - int64: 整数价格（分）
func priceFloat64ToInt64(price float64) int64 {
	return int64(math.Round(price * 100))
}

// priceFloat64ToString 将浮点数价格转换为字符串
// 保留两位小数
// 参数:
//   - price: 浮点数价格
//
// 返回:
//   - string: 价格字符串
func priceFloat64ToString(price float64) string {
	return strconv.FormatFloat(price, 'f', 2, 64)
}

// priceStringToFloat64 将价格字符串转换为浮点数
// 参数:
//   - price: 价格字符串
//
// 返回:
//   - float64: 浮点数价格
func priceStringToFloat64(price string) float64 {
	f, err := strconv.ParseFloat(price, 64)
	if err != nil {
		panic(err)
	}
	return f
}

func GetOwnerAndNameFromId(id string) (string, string) {
	tokens := strings.Split(id, "/")
	if len(tokens) != 2 {
		panic(errors.New("GetOwnerAndNameFromId() error, wrong token count for ID: " + id))
	}

	return tokens[0], tokens[1]
}

// 随机生成字符串
func GetRandomString(l int) string {
	str := "0123456789AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZz"
	bytes := []byte(str)
	var result []byte = make([]byte, 0, l)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < l; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return BytesToString(result)
}

// BytesToString 0 拷贝转换 slice byte 为 string
func BytesToString(b []byte) (s string) {
	_bptr := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	_sptr := (*reflect.StringHeader)(unsafe.Pointer(&s))
	_sptr.Data = _bptr.Data
	_sptr.Len = _bptr.Len
	return s
}



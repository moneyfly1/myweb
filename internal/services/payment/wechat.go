package payment

import (
	"crypto/md5"
	"fmt"
	"sort"
	"strings"
	"time"

	"cboard-go/internal/models"
)

// WechatService 微信支付服务
type WechatService struct {
	AppID     string
	MchID     string
	APIKey    string
	NotifyURL string
}

// NewWechatService 创建微信支付服务
func NewWechatService(paymentConfig *models.PaymentConfig) (*WechatService, error) {
	return &WechatService{
		AppID:     paymentConfig.WechatAppID.String,
		MchID:     paymentConfig.WechatMchID.String,
		APIKey:    paymentConfig.WechatAPIKey.String,
		NotifyURL: paymentConfig.NotifyURL.String,
	}, nil
}

// CreatePayment 创建支付
func (s *WechatService) CreatePayment(order *models.Order, amount float64) (string, error) {
	if s.APIKey == "" {
		return "", fmt.Errorf("API密钥未配置")
	}

	// 构建请求参数
	params := make(map[string]string)
	params["appid"] = s.AppID
	params["mch_id"] = s.MchID
	params["nonce_str"] = generateNonceStr(32)
	params["body"] = "订单支付"
	params["out_trade_no"] = order.OrderNo
	params["total_fee"] = fmt.Sprintf("%.0f", amount*100) // 转换为分
	params["spbill_create_ip"] = "127.0.0.1"
	params["notify_url"] = s.NotifyURL
	params["trade_type"] = "NATIVE" // 扫码支付

	// 签名
	sign := s.Sign(params)
	params["sign"] = sign

	// 构建 XML 请求（用于后续 HTTP 请求）
	_ = mapToXML(params) // 暂时不使用，但保留函数调用

	// 这里应该发送 HTTP 请求到微信支付 API
	// 由于需要实际的 API 调用，这里返回一个占位符
	// 实际使用时需要调用微信支付统一下单接口
	return fmt.Sprintf("weixin://wxpay/bizpayurl?pr=%s", params["nonce_str"]), nil
}

// generateNonceStr 生成随机字符串
func generateNonceStr(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// mapToXML 将 map 转换为 XML
func mapToXML(params map[string]string) string {
	var builder strings.Builder
	builder.WriteString("<xml>")
	for k, v := range params {
		builder.WriteString(fmt.Sprintf("<%s><![CDATA[%s]]></%s>", k, v, k))
	}
	builder.WriteString("</xml>")
	return builder.String()
}

// VerifyNotify 验证回调
func (s *WechatService) VerifyNotify(params map[string]string) bool {
	if s.APIKey == "" {
		return false
	}

	// 提取签名
	sign, ok := params["sign"]
	if !ok || sign == "" {
		return false
	}

	// 移除签名参数
	delete(params, "sign")

	// 计算签名
	calculatedSign := s.Sign(params)

	// 验证签名
	return strings.EqualFold(sign, calculatedSign)
}

// Sign 签名（MD5）
func (s *WechatService) Sign(params map[string]string) string {
	// 排序参数
	var keys []string
	for k := range params {
		if k != "sign" && params[k] != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	// 构建待签名字符串
	var signStr strings.Builder
	for i, k := range keys {
		if i > 0 {
			signStr.WriteString("&")
		}
		signStr.WriteString(k)
		signStr.WriteString("=")
		signStr.WriteString(params[k])
	}
	signStr.WriteString("&key=")
	signStr.WriteString(s.APIKey)

	// MD5 签名
	hash := md5.Sum([]byte(signStr.String()))
	return strings.ToUpper(fmt.Sprintf("%x", hash))
}

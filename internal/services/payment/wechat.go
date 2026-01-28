package payment

import (
	"crypto/md5"
	"fmt"
	"sort"
	"strings"
	"time"

	"cboard-go/internal/models"
)

type WechatService struct {
	AppID     string
	MchID     string
	APIKey    string
	NotifyURL string
}

func NewWechatService(paymentConfig *models.PaymentConfig) (*WechatService, error) {
	return &WechatService{
		AppID:     paymentConfig.WechatAppID.String,
		MchID:     paymentConfig.WechatMchID.String,
		APIKey:    paymentConfig.WechatAPIKey.String,
		NotifyURL: paymentConfig.NotifyURL.String,
	}, nil
}

func (s *WechatService) CreatePayment(order *models.Order, amount float64) (string, error) {
	if s.APIKey == "" {
		return "", fmt.Errorf("API密钥未配置")
	}

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

	sign := s.Sign(params)
	params["sign"] = sign

	_ = mapToXML(params) // 暂时不使用，但保留函数调用

	return fmt.Sprintf("weixin://wxpay/bizpayurl?pr=%s", params["nonce_str"]), nil
}

func generateNonceStr(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

func mapToXML(params map[string]string) string {
	var builder strings.Builder
	builder.WriteString("<xml>")
	for k, v := range params {
		builder.WriteString(fmt.Sprintf("<%s><![CDATA[%s]]></%s>", k, v, k))
	}
	builder.WriteString("</xml>")
	return builder.String()
}

func (s *WechatService) VerifyNotify(params map[string]string) bool {
	if s.APIKey == "" {
		return false
	}

	sign, ok := params["sign"]
	if !ok || sign == "" {
		return false
	}

	delete(params, "sign")

	calculatedSign := s.Sign(params)

	return strings.EqualFold(sign, calculatedSign)
}

func (s *WechatService) Sign(params map[string]string) string {
	var keys []string
	for k := range params {
		if k != "sign" && params[k] != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

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

	hash := md5.Sum([]byte(signStr.String()))
	return strings.ToUpper(fmt.Sprintf("%x", hash))
}

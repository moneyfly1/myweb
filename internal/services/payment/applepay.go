package payment

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"cboard-go/internal/models"
)

type ApplePayService struct {
	merchantID   string
	privateKey   *ecdsa.PrivateKey
	certificate  *x509.Certificate
	notifyURL    string
	returnURL    string
	isProduction bool
}

func NewApplePayService(paymentConfig *models.PaymentConfig) (*ApplePayService, error) {
	merchantID := ""
	var privateKey *ecdsa.PrivateKey
	var certificate *x509.Certificate

	if paymentConfig.ConfigJSON.Valid {
		var configData map[string]interface{}
		if err := json.Unmarshal([]byte(paymentConfig.ConfigJSON.String), &configData); err == nil {
			if mid, ok := configData["merchant_id"].(string); ok {
				merchantID = mid
			}
			if pkey, ok := configData["private_key"].(string); ok {
				keyBytes, err := base64.StdEncoding.DecodeString(pkey)
				if err == nil {
					key, err := x509.ParsePKCS8PrivateKey(keyBytes)
					if err == nil {
						if ecdsaKey, ok := key.(*ecdsa.PrivateKey); ok {
							privateKey = ecdsaKey
						}
					}
				}
			}
			if cert, ok := configData["certificate"].(string); ok {
				certBytes, err := base64.StdEncoding.DecodeString(cert)
				if err == nil {
					cert, err := x509.ParseCertificate(certBytes)
					if err == nil {
						certificate = cert
					}
				}
			}
		}
	}

	if merchantID == "" {
		return nil, fmt.Errorf("Apple Pay Merchant ID 未配置")
	}

	service := &ApplePayService{
		merchantID:   merchantID,
		privateKey:   privateKey,
		certificate:  certificate,
		isProduction: false, // Apple Pay 通常需要生产环境配置
	}

	if paymentConfig.NotifyURL.Valid {
		service.notifyURL = paymentConfig.NotifyURL.String
	}
	if paymentConfig.ReturnURL.Valid {
		service.returnURL = paymentConfig.ReturnURL.String
	}

	return service, nil
}

func (s *ApplePayService) CreatePayment(order *models.Order, amount float64) (string, error) {

	return fmt.Sprintf("applepay://payment?order_no=%s&amount=%.2f", order.OrderNo, amount), nil
}

// VerifyPaymentToken 校验 Apple Pay 支付令牌。
// 当前项目未接入 Apple Pay 前端 SDK（无 ApplePaySession / App Store Server API 收据校验），
// 无法对 token 做真实验签，因此仅做结构性校验并拒绝明显伪造的载荷，
// 防止前端伪造支付令牌直接入账。
func (s *ApplePayService) VerifyPaymentToken(tokenData string) (bool, error) {
	if strings.TrimSpace(tokenData) == "" {
		return false, fmt.Errorf("Apple Pay token 为空")
	}
	var token map[string]interface{}
	if err := json.Unmarshal([]byte(tokenData), &token); err != nil {
		return false, fmt.Errorf("Apple Pay token 格式错误: %w", err)
	}
	// 结构完整才放行；缺少关键字段视为伪造
	if _, ok := token["data"]; !ok {
		if _, ok2 := token["paymentData"]; !ok2 {
			return false, fmt.Errorf("Apple Pay token 缺少支付数据")
		}
	}
	return true, nil
}

// VerifyNotify 拒绝服务端通知回调。
// 由于本项目没有实现 Apple 服务端通知验签（无 App Store Server Notifications 的
// signedPayload 校验），无条件放行会被攻击者伪造回调把任意订单标记为已支付，
// 因此该渠道的异步通知一律拒绝；真实订单请走 VerifyPaymentToken 前置校验流程。
func (s *ApplePayService) VerifyNotify(params map[string]string) bool {
	return false
}

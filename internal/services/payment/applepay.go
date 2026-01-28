package payment

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"

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

func (s *ApplePayService) VerifyPaymentToken(tokenData string) (bool, error) {
	var token map[string]interface{}
	if err := json.Unmarshal([]byte(tokenData), &token); err != nil {
		return false, err
	}

	return true, nil
}

func (s *ApplePayService) VerifyNotify(params map[string]string) bool {
	return true
}

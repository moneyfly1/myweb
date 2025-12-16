package payment

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"cboard-go/internal/core/config"
	"cboard-go/internal/models"
)

// AlipayService 支付宝支付服务
type AlipayService struct {
	AppID      string
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	NotifyURL  string
	ReturnURL  string
	GatewayURL string
}

// NewAlipayService 创建支付宝服务
func NewAlipayService(paymentConfig *models.PaymentConfig) (*AlipayService, error) {
	service := &AlipayService{
		AppID:      "",
		NotifyURL:  "",
		ReturnURL:  "",
		GatewayURL: "https://openapi.alipay.com/gateway.do", // 默认网关
	}

	if paymentConfig.AppID.Valid {
		service.AppID = paymentConfig.AppID.String
	}
	if paymentConfig.NotifyURL.Valid {
		service.NotifyURL = paymentConfig.NotifyURL.String
	}
	if paymentConfig.ReturnURL.Valid {
		service.ReturnURL = paymentConfig.ReturnURL.String
	}

	// 解析私钥
	if paymentConfig.MerchantPrivateKey.Valid && paymentConfig.MerchantPrivateKey.String != "" {
		privateKey, err := parseRSAPrivateKey(paymentConfig.MerchantPrivateKey.String)
		if err != nil {
			return nil, fmt.Errorf("解析私钥失败: %v", err)
		}
		service.PrivateKey = privateKey
	}

	// 解析公钥
	if paymentConfig.AlipayPublicKey.Valid && paymentConfig.AlipayPublicKey.String != "" {
		publicKey, err := parseRSAPublicKey(paymentConfig.AlipayPublicKey.String)
		if err != nil {
			return nil, fmt.Errorf("解析公钥失败: %v", err)
		}
		service.PublicKey = publicKey
	}

	// 从配置中获取网关地址（如果有）
	cfg := config.AppConfig
	if cfg != nil && cfg.AlipayNotifyURL != "" {
		// 可以从配置中获取自定义网关
	}

	return service, nil
}

// parseRSAPrivateKey 解析 RSA 私钥
func parseRSAPrivateKey(keyStr string) (*rsa.PrivateKey, error) {
	// 清理密钥字符串
	keyStr = strings.ReplaceAll(keyStr, "-----BEGIN RSA PRIVATE KEY-----", "")
	keyStr = strings.ReplaceAll(keyStr, "-----END RSA PRIVATE KEY-----", "")
	keyStr = strings.ReplaceAll(keyStr, "-----BEGIN PRIVATE KEY-----", "")
	keyStr = strings.ReplaceAll(keyStr, "-----END PRIVATE KEY-----", "")
	keyStr = strings.ReplaceAll(keyStr, "\n", "")
	keyStr = strings.ReplaceAll(keyStr, " ", "")

	// Base64 解码
	keyBytes, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, err
	}

	// 尝试 PKCS1 格式
	key, err := x509.ParsePKCS1PrivateKey(keyBytes)
	if err == nil {
		return key, nil
	}

	// 尝试 PKCS8 格式
	keyInterface, err := x509.ParsePKCS8PrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("无法解析私钥: %v", err)
	}

	rsaKey, ok := keyInterface.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("不是 RSA 私钥")
	}

	return rsaKey, nil
}

// parseRSAPublicKey 解析 RSA 公钥
func parseRSAPublicKey(keyStr string) (*rsa.PublicKey, error) {
	// 清理密钥字符串
	keyStr = strings.ReplaceAll(keyStr, "-----BEGIN PUBLIC KEY-----", "")
	keyStr = strings.ReplaceAll(keyStr, "-----END PUBLIC KEY-----", "")
	keyStr = strings.ReplaceAll(keyStr, "\n", "")
	keyStr = strings.ReplaceAll(keyStr, " ", "")

	// Base64 解码
	keyBytes, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, err
	}

	// 解析 PEM
	block, _ := pem.Decode(keyBytes)
	if block != nil {
		keyBytes = block.Bytes
	}

	// 解析公钥
	pubInterface, err := x509.ParsePKIXPublicKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("无法解析公钥: %v", err)
	}

	rsaKey, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("不是 RSA 公钥")
	}

	return rsaKey, nil
}

// CreatePayment 创建支付
func (s *AlipayService) CreatePayment(order *models.Order, amount float64) (string, error) {
	if s.PrivateKey == nil {
		return "", fmt.Errorf("私钥未配置")
	}

	// 构建请求参数
	params := make(map[string]string)
	params["app_id"] = s.AppID
	params["method"] = "alipay.trade.page.pay"
	params["charset"] = "utf-8"
	params["sign_type"] = "RSA2"
	params["timestamp"] = time.Now().Format("2006-01-02 15:04:05")
	params["version"] = "1.0"
	params["notify_url"] = s.NotifyURL
	params["return_url"] = s.ReturnURL
	params["biz_content"] = fmt.Sprintf(`{"out_trade_no":"%s","total_amount":"%.2f","subject":"订单支付","product_code":"FAST_INSTANT_TRADE_PAY"}`, order.OrderNo, amount)

	// 签名
	sign, err := s.Sign(params)
	if err != nil {
		return "", fmt.Errorf("签名失败: %v", err)
	}
	params["sign"] = sign

	// 构建支付链接
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}

	paymentURL := s.GatewayURL + "?" + values.Encode()
	return paymentURL, nil
}

// VerifyNotify 验证回调
func (s *AlipayService) VerifyNotify(params map[string]string) bool {
	if s.PublicKey == nil {
		return false
	}

	// 提取签名
	sign, ok := params["sign"]
	if !ok || sign == "" {
		return false
	}

	// 移除签名参数
	signType := params["sign_type"]
	delete(params, "sign")
	delete(params, "sign_type")

	// 构建待验证字符串
	var keys []string
	for k := range params {
		if params[k] != "" {
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

	// 验证签名
	signBytes, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return false
	}

	hash := sha256.Sum256([]byte(signStr.String()))
	if signType == "RSA2" {
		err = rsa.VerifyPKCS1v15(s.PublicKey, crypto.SHA256, hash[:], signBytes)
		return err == nil
	}

	return false
}

// Sign 签名
func (s *AlipayService) Sign(params map[string]string) (string, error) {
	// 排序参数
	var keys []string
	for k := range params {
		if k != "sign" && k != "sign_type" && params[k] != "" {
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

	// RSA2 签名
	hash := sha256.Sum256([]byte(signStr.String()))
	signature, err := rsa.SignPKCS1v15(nil, s.PrivateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

package payment

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cboard-go/internal/models"
)

// ============================================================
// PayPal 支付（国际）
// 通过 PayPal REST API 创建订单，获取 approval URL 跳转支付，
// 回调通过 Webhook 验证（PayPal 需在商户后台配置 Webhook URL）。
// ============================================================

const (
	paypalAPIBase     = "https://api-m.paypal.com"
	paypalSandboxBase = "https://api-m.sandbox.paypal.com"
)

type PayPalService struct {
	clientID     string
	secret       string
	webhookID    string
	apiBase      string
	returnURL    string
	accessToken  string
	tokenExpires time.Time
}

func NewPayPalService(paymentConfig *models.PaymentConfig) (*PayPalService, error) {
	clientID := ""
	if paymentConfig.PaypalClientID.Valid {
		clientID = strings.TrimSpace(paymentConfig.PaypalClientID.String)
	}
	if clientID == "" {
		return nil, fmt.Errorf("PayPal Client ID 未配置")
	}
	secret := ""
	if paymentConfig.PaypalSecret.Valid {
		secret = strings.TrimSpace(paymentConfig.PaypalSecret.String)
	}
	if secret == "" {
		return nil, fmt.Errorf("PayPal Secret 未配置")
	}
	svc := &PayPalService{
		clientID: clientID,
		secret:   secret,
		apiBase:  paypalAPIBase,
	}
	if paymentConfig.ReturnURL.Valid {
		svc.returnURL = strings.TrimSpace(paymentConfig.ReturnURL.String)
	}
	if paymentConfig.ConfigJSON.Valid {
		var cfg map[string]interface{}
		if json.Unmarshal([]byte(paymentConfig.ConfigJSON.String), &cfg) == nil {
			if sandbox, ok := cfg["sandbox"].(bool); ok && sandbox {
				svc.apiBase = paypalSandboxBase
			}
			if wid, ok := cfg["webhook_id"].(string); ok && wid != "" {
				svc.webhookID = wid
			}
		}
	}
	return svc, nil
}

// getAccessToken 获取 OAuth2 访问令牌（带缓存）
func (s *PayPalService) getAccessToken() (string, error) {
	if s.accessToken != "" && time.Now().Before(s.tokenExpires) {
		return s.accessToken, nil
	}

	req, err := http.NewRequest("POST", s.apiBase+"/v1/oauth2/token", bytes.NewBufferString("grant_type=client_credentials"))
	if err != nil {
		return "", err
	}
	auth := base64.StdEncoding.EncodeToString([]byte(s.clientID + ":" + s.secret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("PayPal 令牌响应解析失败: %v", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("PayPal 获取令牌失败: %s %s", result.Error, result.ErrorDesc)
	}
	s.accessToken = result.AccessToken
	s.tokenExpires = time.Now().Add(time.Duration(result.ExpiresIn-60) * time.Second)
	return s.accessToken, nil
}

type paypalPurchaseUnit struct {
	ReferenceID string          `json:"reference_id"`
	Description string          `json:"description,omitempty"`
	Amount      paypalAmount    `json:"amount"`
}

type paypalAmount struct {
	CurrencyCode string `json:"currency_code"`
	Value        string `json:"value"`
}

type paypalCreateOrderRequest struct {
	Intent        string                `json:"intent"`
	PurchaseUnits []paypalPurchaseUnit  `json:"purchase_units"`
	ApplicationContext paypalAppContext `json:"application_context"`
}

type paypalAppContext struct {
	ReturnURL string `json:"return_url,omitempty"`
	CancelURL string `json:"cancel_url,omitempty"`
	BrandName string `json:"brand_name,omitempty"`
}

// CreatePayment 创建 PayPal 订单，返回 approval 链接。
func (s *PayPalService) CreatePayment(order *models.Order, amount float64, email string) (string, error) {
	token, err := s.getAccessToken()
	if err != nil {
		return "", err
	}
	if amount <= 0 {
		return "", fmt.Errorf("订单金额无效")
	}

	reqBody := paypalCreateOrderRequest{
		Intent: "CAPTURE",
		PurchaseUnits: []paypalPurchaseUnit{
			{
				ReferenceID: order.OrderNo,
				Description: "CBoard 订单 " + order.OrderNo,
				Amount: paypalAmount{
					CurrencyCode: "USD",
					Value:        fmt.Sprintf("%.2f", amount),
				},
			},
		},
		ApplicationContext: paypalAppContext{
			ReturnURL: s.buildPayPalReturnURL(order, "success"),
			CancelURL: s.buildPayPalReturnURL(order, "cancel"),
			BrandName: "CBoard",
		},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", s.apiBase+"/v2/checkout/orders", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ID    string `json:"id"`
		Links []struct {
			Href   string `json:"href"`
			Rel    string `json:"rel"`
			Method string `json:"method"`
		} `json:"links"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("PayPal 订单响应解析失败: %v", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("PayPal 创建订单失败: %s", result.Error.Message)
	}
	for _, link := range result.Links {
		if link.Rel == "approve" && link.Href != "" {
			return link.Href, nil
		}
	}
	return "", fmt.Errorf("PayPal 未返回支付跳转链接")
}

// paypalWebhookPayload PayPal Webhook 事件
type PayPalWebhookPayload struct {
	ID       string `json:"id"`
	EventType string `json:"event_type"`
	Resource struct {
		ID             string `json:"id"`
		Status         string `json:"status"`
		CustomID       string `json:"custom_id"`
		SupplementaryData *struct {
			RelatedIDs struct {
				OrderID string `json:"order_id"`
			} `json:"related_ids"`
		} `json:"supplementary_data"`
		PurchaseUnits []struct {
			ReferenceID string `json:"reference_id"`
		} `json:"purchase_units"`
	} `json:"resource"`
}

// ExtractOrderNo 从 Webhook 事件提取订单号
func (e *PayPalWebhookPayload) ExtractOrderNo() string {
	if e.Resource.CustomID != "" {
		return e.Resource.CustomID
	}
	if len(e.Resource.PurchaseUnits) > 0 && e.Resource.PurchaseUnits[0].ReferenceID != "" {
		return e.Resource.PurchaseUnits[0].ReferenceID
	}
	return e.Resource.SupplementaryData.RelatedIDs.OrderID
}

// IsPaid 判断事件是否为支付成功
func (e *PayPalWebhookPayload) IsPaid() bool {
	return e.EventType == "PAYMENT.CAPTURE.COMPLETED" || e.EventType == "CHECKOUT.ORDER.APPROVED"
}

// VerifyWebhook 验证 PayPal Webhook 签名。
// PayPal 使用 webhook_id + 请求头传输的签名（需调用验证接口）。
func (s *PayPalService) VerifyWebhook(payload []byte, headers map[string]string) (bool, error) {
	if s.webhookID == "" {
		return false, fmt.Errorf("PayPal webhook_id 未配置")
	}
	token, err := s.getAccessToken()
	if err != nil {
		return false, err
	}

	verifyBody := map[string]string{
		"auth_algo":         headers["PAYPAL-AUTH-ALGO"],
		"cert_url":          headers["PAYPAL-CERT-URL"],
		"transmission_id":   headers["PAYPAL-TRANSMISSION-ID"],
		"transmission_sig":  headers["PAYPAL-TRANSMISSION-SIG"],
		"transmission_time": headers["PAYPAL-TRANSMISSION-TIME"],
		"webhook_id":        s.webhookID,
		"webhook_event":     string(payload),
	}
	body, _ := json.Marshal(verifyBody)
	req, err := http.NewRequest("POST", s.apiBase+"/v1/notifications/verify-webhook-signature", bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result struct {
		VerificationStatus string `json:"verification_status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("PayPal 验签响应解析失败: %v", err)
	}
	return result.VerificationStatus == "SUCCESS", nil
}

// buildPayPalReturnURL 构造支付完成回跳地址（优先用配置的 return_url）
func (s *PayPalService) buildPayPalReturnURL(order *models.Order, status string) string {
	if s.returnURL != "" {
		return s.returnURL + "?order_no=" + order.OrderNo + "&status=" + status
	}
	return "https://dy.moneyfly.top/payment/return?order_no=" + order.OrderNo + "&status=" + status
}

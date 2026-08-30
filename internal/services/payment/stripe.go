package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cboard-go/internal/models"
)

// ============================================================
// Stripe 支付（国际信用卡）
// 通过 Stripe REST API 创建 PaymentIntent（Checkout Session），
// Webhook 验签（Stripe-Signature 头 + 时间戳 + HMAC SHA256）。
// ============================================================

const stripeAPIBase = "https://api.stripe.com"

type StripeService struct {
	secretKey     string
	webhookSecret string
	returnURL     string
	notifyURL     string
}

func NewStripeService(paymentConfig *models.PaymentConfig) (*StripeService, error) {
	secretKey := ""
	if paymentConfig.StripeSecretKey.Valid {
		secretKey = strings.TrimSpace(paymentConfig.StripeSecretKey.String)
	}
	if secretKey == "" {
		return nil, fmt.Errorf("Stripe Secret Key 未配置")
	}
	svc := &StripeService{secretKey: secretKey}
	if paymentConfig.ReturnURL.Valid {
		svc.returnURL = strings.TrimSpace(paymentConfig.ReturnURL.String)
	}
	if paymentConfig.NotifyURL.Valid {
		svc.notifyURL = strings.TrimSpace(paymentConfig.NotifyURL.String)
	}
	if paymentConfig.ConfigJSON.Valid {
		var cfg map[string]interface{}
		if json.Unmarshal([]byte(paymentConfig.ConfigJSON.String), &cfg) == nil {
			if ws, ok := cfg["webhook_secret"].(string); ok && ws != "" {
				svc.webhookSecret = ws
			}
		}
	}
	return svc, nil
}

// stripeCheckoutRequest 创建 Checkout Session 的请求体
type stripeCheckoutRequest struct {
	Mode                string                    `json:"mode"`
	SuccessURL          string                    `json:"success_url"`
	CancelURL           string                    `json:"cancel_url"`
	ClientReferenceID   string                    `json:"client_reference_id"`
	CustomerEmail       string                    `json:"customer_email,omitempty"`
	LineItems           []stripeCheckoutLineItem  `json:"line_items"`
	PaymentMethodTypes  []string                  `json:"payment_method_types"`
	Metadata            map[string]string         `json:"metadata,omitempty"`
}

type stripeCheckoutLineItem struct {
	PriceData stripePriceData `json:"price_data"`
	Quantity  int64           `json:"quantity"`
}

type stripePriceData struct {
	Currency    string         `json:"currency"`
	ProductData stripeProduct  `json:"product_data"`
	UnitAmount  int64          `json:"unit_amount"`
}

type stripeProduct struct {
	Name string `json:"name"`
}

// CreatePayment 创建 Stripe Checkout Session，返回支付跳转 URL。
func (s *StripeService) CreatePayment(order *models.Order, amount float64, email string) (string, error) {
	if amount <= 0 {
		return "", fmt.Errorf("订单金额无效")
	}
	// Stripe 金额单位为分（最小货币单位）
	unitAmount := int64(amount * 100)
	if unitAmount <= 0 {
		return "", fmt.Errorf("订单金额过小，Stripe 不支持")
	}

	productName := "CBoard 订单 " + order.OrderNo
	if order.PaymentMethodName.Valid && order.PaymentMethodName.String != "" {
		productName = "CBoard-" + order.PaymentMethodName.String
	}

	reqBody := stripeCheckoutRequest{
		Mode:              "payment",
		SuccessURL:        s.buildReturnURL(order, "success"),
		CancelURL:         s.buildReturnURL(order, "cancel"),
		ClientReferenceID: order.OrderNo,
		PaymentMethodTypes: []string{"card"},
		Metadata: map[string]string{
			"order_no": order.OrderNo,
			"order_id": fmt.Sprintf("%d", order.ID),
		},
		LineItems: []stripeCheckoutLineItem{
			{
				PriceData: stripePriceData{
					Currency: "usd",
					ProductData: stripeProduct{Name: productName},
					UnitAmount:  unitAmount,
				},
				Quantity: 1,
			},
		},
	}
	if email != "" {
		reqBody.CustomerEmail = email
	}

	body, _ := json.Marshal(reqBody)
	resp, err := s.postJSON("/v1/checkout/sessions", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ID     string `json:"id"`
		URL    string `json:"url"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("Stripe 响应解析失败: %v", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("Stripe 创建支付失败: %s", result.Error.Message)
	}
	if result.URL == "" {
		return "", fmt.Errorf("Stripe 未返回支付跳转地址")
	}
	return result.URL, nil
}

// VerifyWebhook 验证 Stripe Webhook 签名并返回事件。
// payload: 原始请求体；signatureHeader: Stripe-Signature 头
func (s *StripeService) VerifyWebhook(payload []byte, signatureHeader string) (*StripeEvent, error) {
	if s.webhookSecret == "" {
		return nil, fmt.Errorf("Stripe webhook_secret 未配置")
	}
	if !verifyStripeSignature(payload, signatureHeader, s.webhookSecret) {
		return nil, fmt.Errorf("Stripe Webhook 签名验证失败")
	}

	var event StripeEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("Stripe 事件解析失败: %v", err)
	}
	return &event, nil
}

// StripeEvent Stripe Webhook 事件结构
type StripeEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object struct {
			ID                 string `json:"id"`
			ClientReferenceID  string `json:"client_reference_id"`
			PaymentStatus      string `json:"payment_status"`
			Status             string `json:"status"`
			AmountTotal        int64  `json:"amount_total"`
			Metadata           map[string]string `json:"metadata"`
		} `json:"object"`
	} `json:"data"`
}

// ExtractOrderNo 从事件中提取订单号（优先 metadata，其次 client_reference_id）
func (e *StripeEvent) ExtractOrderNo() string {
	if e.Data.Object.Metadata != nil {
		if no := e.Data.Object.Metadata["order_no"]; no != "" {
			return no
		}
	}
	return e.Data.Object.ClientReferenceID
}

// IsPaid 判断事件是否为支付成功
func (e *StripeEvent) IsPaid() bool {
	return e.Type == "checkout.session.completed" &&
		(e.Data.Object.PaymentStatus == "paid" || e.Data.Object.Status == "complete")
}

func (s *StripeService) postJSON(path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", stripeAPIBase+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+s.secretKey)
	req.Header.Set("Stripe-Version", "2024-06-20")

	client := &http.Client{Timeout: 15 * time.Second}
	return client.Do(req)
}

// buildReturnURL 构造支付完成回跳地址（优先用配置的 return_url）
func (s *StripeService) buildReturnURL(order *models.Order, status string) string {
	if s.returnURL != "" {
		return s.returnURL + "?order_no=" + order.OrderNo + "&status=" + status
	}
	return fmt.Sprintf("%s/payment/return?order_no=%s&status=%s", "https://dy.moneyfly.top", order.OrderNo, status)
}

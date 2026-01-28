package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"cboard-go/internal/models"
)

type PayPalService struct {
	clientID    string
	secret      string
	baseURL     string
	notifyURL   string
	returnURL   string
	accessToken string
	tokenExpiry time.Time
}

func NewPayPalService(paymentConfig *models.PaymentConfig) (*PayPalService, error) {
	clientID := ""
	if paymentConfig.PaypalClientID.Valid {
		clientID = paymentConfig.PaypalClientID.String
	}
	if clientID == "" {
		return nil, fmt.Errorf("PayPal ClientID 未配置")
	}

	secret := ""
	if paymentConfig.PaypalSecret.Valid {
		secret = paymentConfig.PaypalSecret.String
	}
	if secret == "" {
		return nil, fmt.Errorf("PayPal Secret 未配置")
	}

	isProduction := false
	if paymentConfig.ConfigJSON.Valid {
		var configData map[string]interface{}
		if err := json.Unmarshal([]byte(paymentConfig.ConfigJSON.String), &configData); err == nil {
			if prod, ok := configData["is_production"].(bool); ok {
				isProduction = prod
			}
		}
	}

	baseURL := "https://api.sandbox.paypal.com"
	if isProduction {
		baseURL = "https://api.paypal.com"
	}

	service := &PayPalService{
		clientID: clientID,
		secret:   secret,
		baseURL:  baseURL,
	}

	if paymentConfig.NotifyURL.Valid {
		service.notifyURL = paymentConfig.NotifyURL.String
	}
	if paymentConfig.ReturnURL.Valid {
		service.returnURL = paymentConfig.ReturnURL.String
	}

	return service, nil
}

func (s *PayPalService) getAccessToken() (string, error) {
	if s.accessToken != "" && time.Now().Before(s.tokenExpiry) {
		return s.accessToken, nil
	}

	url := s.baseURL + "/v1/oauth2/token"
	req, err := http.NewRequest("POST", url, bytes.NewBufferString("grant_type=client_credentials"))
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(s.clientID, s.secret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}

	s.accessToken = tokenResp.AccessToken
	s.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second) // 提前60秒过期

	return s.accessToken, nil
}

func (s *PayPalService) CreatePayment(order *models.Order, amount float64) (string, error) {
	accessToken, err := s.getAccessToken()
	if err != nil {
		return "", fmt.Errorf("获取访问令牌失败: %v", err)
	}

	paymentReq := map[string]interface{}{
		"intent": "sale",
		"payer": map[string]string{
			"payment_method": "paypal",
		},
		"transactions": []map[string]interface{}{
			{
				"amount": map[string]interface{}{
					"total":    fmt.Sprintf("%.2f", amount),
					"currency": "USD",
				},
				"description":    fmt.Sprintf("订单支付-%s", order.OrderNo),
				"invoice_number": order.OrderNo,
			},
		},
		"redirect_urls": map[string]string{
			"return_url": s.returnURL + "?order_no=" + order.OrderNo,
			"cancel_url": s.returnURL + "?order_no=" + order.OrderNo + "&cancel=1",
		},
	}

	jsonData, err := json.Marshal(paymentReq)
	if err != nil {
		return "", err
	}

	url := s.baseURL + "/v1/payments/payment"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var paymentResp struct {
		ID    string `json:"id"`
		State string `json:"state"`
		Links []struct {
			Href   string `json:"href"`
			Rel    string `json:"rel"`
			Method string `json:"method"`
		} `json:"links"`
	}

	if err := json.Unmarshal(body, &paymentResp); err != nil {
		return "", err
	}

	for _, link := range paymentResp.Links {
		if link.Rel == "approval_url" {
			return link.Href, nil
		}
	}

	return "", fmt.Errorf("未找到支付URL")
}

func (s *PayPalService) VerifyNotify(params map[string]string) bool {
	paymentID := params["paymentId"]
	if paymentID == "" {
		return false
	}

	accessToken, err := s.getAccessToken()
	if err != nil {
		return false
	}

	url := s.baseURL + "/v1/payments/payment/" + paymentID
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	return true
}

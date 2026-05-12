package payment

import (
	"crypto/md5" // #nosec G501 - MD5 required by codepay API specification
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"cboard-go/internal/models"
	"cboard-go/internal/utils"
)

type CodepayService struct {
	PID       string
	Key       string
	APIURL    string // mapi.php 地址
	SubmitURL string // submit.php 地址
	QueryURL  string
	NotifyURL string
	ReturnURL string
}

type CodepayResponse struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	TradeNo   string `json:"trade_no"`
	PayURL    string `json:"payurl"`
	QRCode    string `json:"qrcode"`
	URLScheme string `json:"urlscheme"`
	Money     string `json:"money"`
}

func NewCodepayService(paymentConfig *models.PaymentConfig) (*CodepayService, error) {
	pid := ""
	if paymentConfig.AppID.Valid {
		pid = strings.TrimSpace(paymentConfig.AppID.String)
	}
	if pid == "" {
		return nil, fmt.Errorf("码支付商户ID未配置")
	}

	key := ""
	if paymentConfig.MerchantPrivateKey.Valid {
		key = strings.TrimSpace(paymentConfig.MerchantPrivateKey.String)
	}
	if key == "" {
		return nil, fmt.Errorf("码支付商户密钥未配置")
	}

	configData := parseConfigData(paymentConfig.ConfigJSON)
	apiURL := getConfigString(configData, "api_url")
	if apiURL == "" {
		gatewayURL := getConfigString(configData, "gateway_url")
		if gatewayURL != "" {
			gatewayURL = strings.TrimSuffix(gatewayURL, "/")
			apiURL = gatewayURL + "/xpay/epay/mapi.php"
		}
	}
	// 修正路径重复问题（用户可能填了包含 /xpay/epay 的网关地址）
	if strings.Contains(apiURL, "/xpay/epay/xpay/epay/") {
		apiURL = strings.Replace(apiURL, "/xpay/epay/xpay/epay/", "/xpay/epay/", 1)
	}
	if apiURL == "" {
		return nil, fmt.Errorf("码支付API地址未配置")
	}
	queryURL := getConfigString(configData, "query_url")
	if queryURL == "" {
		queryURL = buildEpayOrderQueryURL(apiURL)
	}

	// 从 mapi.php 地址推导 submit.php 地址
	submitURL := strings.Replace(apiURL, "/mapi.php", "/submit.php", 1)

	return &CodepayService{
		PID:       pid,
		Key:       key,
		APIURL:    apiURL,
		SubmitURL: submitURL,
		QueryURL:  queryURL,
		NotifyURL: resolveCallbackURL(paymentConfig.NotifyURL, getConfigString(configData, "notify_url"), "/api/v1/payment/notify/codepay", true),
		ReturnURL: resolveCallbackURL(paymentConfig.ReturnURL, "", "/payment/return", false),
	}, nil
}

// codepaySign 按照码支付签名规则生成签名
// 1. 过滤空值和 sign/sign_type 参数
// 2. 按参数名 ASCII 码排序
// 3. 拼接成 key=value& 格式
// 4. 末尾拼接商户密钥后 MD5
func (s *CodepayService) codepaySign(params map[string]string) string {
	var keys []string
	for k, v := range params {
		if v == "" || k == "sign" || k == "sign_type" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString("&")
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(params[k])
	}
	sb.WriteString(s.Key)

	signStr := sb.String()
	utils.LogInfo("码支付签名字符串(隐藏密钥): %s", strings.Replace(signStr, s.Key, "***KEY***", 1))

	hash := md5.Sum([]byte(signStr)) // #nosec G401
	return fmt.Sprintf("%x", hash)
}

func (s *CodepayService) CreatePayment(order *models.Order, amount float64, paymentType string) (string, error) {
	if order == nil || order.OrderNo == "" {
		return "", fmt.Errorf("订单信息无效")
	}
	if amount <= 0 {
		return "", fmt.Errorf("支付金额无效: %.2f", amount)
	}
	if paymentType == "" {
		paymentType = "alipay"
	}

	name := fmt.Sprintf("订单支付-%s", order.OrderNo)
	if len(name) > 127 {
		name = name[:127]
	}

	params := map[string]string{
		"pid":          s.PID,
		"type":         paymentType,
		"out_trade_no": order.OrderNo,
		"money":        fmt.Sprintf("%.2f", amount),
		"name":         name,
	}

	if s.NotifyURL == "" {
		return "", fmt.Errorf("码支付回调地址未配置")
	}
	params["notify_url"] = s.NotifyURL

	if s.ReturnURL != "" {
		if parsedURL, err := url.Parse(s.ReturnURL); err == nil {
			parsedURL.RawQuery = ""
			params["return_url"] = parsedURL.String()
		}
	}

	params["sign"] = s.codepaySign(params)
	params["sign_type"] = "MD5"

	// 先尝试 mapi.php 获取二维码链接
	utils.LogInfo("码支付发起mapi请求: URL=%s, Order=%s, Amount=%s, Type=%s", s.APIURL, order.OrderNo, params["money"], paymentType)

	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.PostForm(s.APIURL, formData)
	if err == nil {
		defer resp.Body.Close()
		body, readErr := io.ReadAll(resp.Body)
		if readErr == nil && resp.StatusCode == http.StatusOK {
			respStr := strings.TrimSpace(string(body))
			utils.LogInfo("码支付mapi响应: %s", truncateString(respStr, 500))

			// 如果直接返回 URL
			if strings.HasPrefix(respStr, "http://") || strings.HasPrefix(respStr, "https://") {
				return respStr, nil
			}

			var codepayResp CodepayResponse
			if json.Unmarshal(body, &codepayResp) == nil && codepayResp.Code == 1 {
				utils.LogInfo("码支付mapi返回: code=%d, trade_no=%s, payurl=%s, qrcode=%s, urlscheme=%s",
					codepayResp.Code, codepayResp.TradeNo, codepayResp.PayURL, codepayResp.QRCode, codepayResp.URLScheme)

				if codepayResp.QRCode != "" {
					return codepayResp.QRCode, nil
				}
				if codepayResp.PayURL != "" {
					return codepayResp.PayURL, nil
				}
				if codepayResp.URLScheme != "" {
					return codepayResp.URLScheme, nil
				}
			}
		}
	}

	// mapi 未返回支付链接，回退到 submit.php 页面跳转方式
	utils.LogInfo("码支付mapi未返回支付链接，使用submit.php页面方式: Order=%s", order.OrderNo)
	submitParams := url.Values{}
	for k, v := range params {
		submitParams.Set(k, v)
	}
	submitURL := fmt.Sprintf("%s?%s", s.SubmitURL, submitParams.Encode())
	utils.LogInfo("码支付submit URL: %s", submitURL)
	return submitURL, nil
}

func (s *CodepayService) QueryOrder(orderNo string) (*EpayQueryResult, error) {
	return queryEpayOrder("码支付", s.QueryURL, s.PID, s.Key, orderNo)
}

func GetCodepaySupportedTypes(paymentConfig *models.PaymentConfig) []string {
	defaultTypes := []string{"alipay", "wxpay"}
	data := parseConfigData(paymentConfig.ConfigJSON)
	if data == nil {
		return defaultTypes
	}
	if list, ok := data["supported_types"].([]interface{}); ok {
		var result []string
		for _, v := range list {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultTypes
}

func (s *CodepayService) VerifyNotify(params map[string]string) bool {
	sign, ok := params["sign"]
	if !ok || sign == "" {
		utils.LogError("码支付回调缺少签名", nil, map[string]interface{}{
			"params": params,
		})
		return false
	}

	// 记录回调参数（隐藏敏感信息）
	safeParams := make(map[string]string)
	for k, v := range params {
		if k != "sign" {
			safeParams[k] = v
		}
	}
	utils.LogInfo("码支付回调参数验证: %+v", safeParams)

	calcSign := s.codepaySign(params)
	match := strings.EqualFold(sign, calcSign)
	if !match {
		utils.LogError("码支付MD5校验失败", nil, map[string]interface{}{
			"received":   sign,
			"calculated": calcSign,
		})
	} else {
		utils.LogInfo("码支付签名验证成功")
	}
	return match
}

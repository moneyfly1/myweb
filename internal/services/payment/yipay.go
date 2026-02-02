package payment

import (
	"crypto"
	"crypto/md5"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"
)

type YipayService struct {
	PID                string
	Key                string
	SignType           string
	PlatformPublicKey  string
	MerchantPrivateKey string
	APIURL             string
	NotifyURL          string
	ReturnURL          string
	Adapter            YipayPlatformAdapter
	PlatformName       string
}

type YipayResponse struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	TradeNo   string `json:"trade_no"`
	PayURL    string `json:"payurl"`
	QRCode    string `json:"qrcode"`
	URLScheme string `json:"urlscheme"`
}

func parseConfigData(configJSON sql.NullString) map[string]interface{} {
	if !configJSON.Valid {
		return nil
	}
	var data map[string]interface{}
	_ = json.Unmarshal([]byte(configJSON.String), &data)
	return data
}

func getConfigString(data map[string]interface{}, key string) string {
	if data == nil {
		return ""
	}
	if val, ok := data[key].(string); ok {
		return strings.TrimSpace(val)
	}
	return ""
}

func isLocalDomain(domain string) bool {
	domain = strings.ToLower(domain)
	locals := []string{"localhost", "127.0.0.1", "192.168.", "10.", "172.", "local"}
	for _, l := range locals {
		if strings.Contains(domain, l) {
			return true
		}
	}
	return false
}

func buildBaseURL(domain, path string) string {
	domain = strings.TrimSuffix(domain, "/")
	if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
		return fmt.Sprintf("%s%s", domain, path)
	}
	if isLocalDomain(domain) {
		return fmt.Sprintf("http://%s%s", domain, path)
	}
	return fmt.Sprintf("%s%s", utils.FormatDomainURL(domain), path)
}

func resolveCallbackURL(explicit sql.NullString, jsonVal string, path string, isNotify bool) string {
	if explicit.Valid && explicit.String != "" {
		urlStr := strings.TrimSpace(explicit.String)
		utils.LogInfo("易支付使用配置的回调地址: %s", urlStr)
		return urlStr
	}

	if jsonVal != "" {
		utils.LogInfo("易支付从配置JSON中获取回调地址: %s", jsonVal)
		if isNotify && !isLocalDomain(jsonVal) {
			if domain := utils.GetDomainFromDB(database.GetDB()); isLocalDomain(domain) {
				utils.LogWarn("易支付回调地址配置为生产域名 (%s)，但当前环境是本地 (%s)，回调可能无法到达", jsonVal, domain)
			}
		}
		return jsonVal
	}

	db := database.GetDB()
	if db == nil {
		if isNotify {
			utils.LogWarn("易支付无法生成回调地址：数据库连接为空")
		}
		return ""
	}

	domain := utils.GetDomainFromDB(db)
	if domain == "" {
		if isNotify {
			utils.LogWarn("易支付无法生成回调地址：域名配置为空")
		}
		return ""
	}

	genURL := buildBaseURL(domain, path)
	if isNotify {
		if isLocalDomain(domain) {
			utils.LogWarn("易支付本地环境自动生成回调地址: %s (需内网穿透)", genURL)
		} else {
			utils.LogInfo("易支付生产环境自动生成回调地址: %s", genURL)
		}
	} else {
		utils.LogInfo("易支付自动生成同步回调地址: %s", genURL)
	}
	return genURL
}

func detectDeviceType(userAgent string, paymentType string) string {
	if userAgent == "" {
		return "pc"
	}

	ua := strings.ToLower(userAgent)

	if strings.Contains(ua, "micromessenger") {
		if paymentType == "wxpay" {
			return "wechat"
		}
		return "mobile"
	}

	if strings.Contains(ua, "alipay") {
		return "alipay"
	}

	if strings.Contains(ua, "qq/") {
		return "qq"
	}

	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") ||
		strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") ||
		strings.Contains(ua, "ios") {
		return "mobile"
	}

	return "pc"
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func buildSignString(params map[string]string, excludeKeys ...string) string {
	var keys []string
	excludeMap := make(map[string]bool)
	for _, k := range excludeKeys {
		excludeMap[k] = true
	}

	for k, v := range params {
		if v == "" || excludeMap[k] {
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
	return sb.String()
}

func NewYipayService(paymentConfig *models.PaymentConfig) (*YipayService, error) {
	pid := ""
	if paymentConfig.AppID.Valid {
		pid = strings.TrimSpace(paymentConfig.AppID.String)
	}
	if pid == "" {
		return nil, fmt.Errorf("易支付商户ID未配置")
	}

	configData := parseConfigData(paymentConfig.ConfigJSON)
	signType := getConfigString(configData, "sign_type")
	if signType == "" {
		signType = "MD5"
	}

	adapter, platformName := detectYipayPlatform(paymentConfig)
	if !adapter.SupportsSignatureType(signType) {
		utils.LogWarn("平台 %s 可能不支持签名类型 %s，将尝试使用", platformName, signType)
	}

	key := ""
	if paymentConfig.MerchantPrivateKey.Valid {
		key = strings.TrimSpace(paymentConfig.MerchantPrivateKey.String)
	}
	if key == "" && (signType == "MD5" || signType == "MD5+RSA") {
		return nil, fmt.Errorf("易支付MD5密钥未配置")
	}

	platformPublicKey := ""
	merchantPrivateKey := ""
	if strings.Contains(signType, "RSA") {
		if paymentConfig.AlipayPublicKey.Valid {
			platformPublicKey = strings.TrimSpace(paymentConfig.AlipayPublicKey.String)
		}
		if platformPublicKey == "" {
			platformPublicKey = getConfigString(configData, "platform_public_key")
		}
		merchantPrivateKey = getConfigString(configData, "merchant_private_key")

		if signType == "RSA" && (platformPublicKey == "" || merchantPrivateKey == "") {
			return nil, fmt.Errorf("易支付RSA签名需配置平台公钥和商户私钥")
		}
		if signType == "MD5+RSA" && platformPublicKey == "" {
			return nil, fmt.Errorf("易支付MD5+RSA签名需配置平台公钥")
		}
	}

	gatewayURL := getConfigString(configData, "gateway_url")
	apiURL := getConfigString(configData, "api_url")
	if apiURL == "" {
		if gatewayURL != "" {
			apiURL = adapter.GetAPIURL(gatewayURL)
			utils.LogInfo("易支付从gateway_url生成api_url: gateway_url=%s, api_url=%s, platform=%s", gatewayURL, apiURL, platformName)
		}
	}
	if apiURL == "" {
		return nil, fmt.Errorf("易支付API地址未配置")
	}

	utils.LogInfo("易支付初始化: platform=%s, api_url=%s, pid=%s, sign_type=%s", platformName, apiURL, pid, signType)

	return &YipayService{
		PID:                pid,
		Key:                key,
		SignType:           signType,
		PlatformPublicKey:  platformPublicKey,
		MerchantPrivateKey: merchantPrivateKey,
		APIURL:             apiURL,
		NotifyURL:          resolveCallbackURL(paymentConfig.NotifyURL, getConfigString(configData, "notify_url"), "/api/v1/payment/notify/yipay", true),
		ReturnURL:          resolveCallbackURL(paymentConfig.ReturnURL, "", "/payment/return", false),
		Adapter:            adapter,
		PlatformName:       platformName,
	}, nil
}

func (s *YipayService) CreatePayment(order *models.Order, amount float64, paymentType string) (string, error) {
	return s.CreatePaymentWithDevice(order, amount, paymentType, "")
}

func (s *YipayService) CreatePaymentWithDevice(order *models.Order, amount float64, paymentType string, userAgent string) (string, error) {
	if order == nil || order.OrderNo == "" {
		return "", fmt.Errorf("订单信息无效")
	}
	if amount <= 0 {
		return "", fmt.Errorf("支付金额无效: %.2f", amount)
	}
	if paymentType == "" {
		paymentType = "alipay"
		utils.LogWarn("易支付类型默认: alipay")
	}

	deviceType := detectDeviceType(userAgent, paymentType)
	utils.LogInfo("易支付设备类型检测: order_no=%s, userAgent长度=%d, userAgent前50字符=%s, paymentType=%s, device=%s",
		order.OrderNo, len(userAgent), truncateString(userAgent, 50), paymentType, deviceType)

	params := map[string]string{
		"pid":          s.PID,
		"type":         paymentType,
		"out_trade_no": order.OrderNo,
		"money":        fmt.Sprintf("%.2f", amount),
		"name":         fmt.Sprintf("订单支付-%s", order.OrderNo),
		"clientip":     "127.0.0.1",
		"device":       deviceType,
	}

	utils.LogInfo("易支付device参数: order_no=%s, device=%s, type=%s", order.OrderNo, deviceType, paymentType)

	if s.NotifyURL == "" {
		return "", fmt.Errorf("回调地址未配置")
	}
	params["notify_url"] = s.NotifyURL

	if s.ReturnURL != "" {
		// 确保 return_url 不包含查询参数，易支付平台会自动添加订单号等参数
		returnURL := s.ReturnURL
		if parsedURL, err := url.Parse(returnURL); err == nil {
			// 移除所有查询参数，只保留基础路径
			parsedURL.RawQuery = ""
			returnURL = parsedURL.String()
		}
		params["return_url"] = returnURL
		utils.LogInfo("易支付设置return_url: %s (已移除查询参数，易支付平台会自动添加)", returnURL)
	}

	signStr := buildSignString(params, "sign", "sign_type", "rsa_sign")

	if s.SignType == "MD5" || s.SignType == "" {
		params["sign"] = s.calcMD5FromStr(signStr)
		params["sign_type"] = "MD5"
	} else if s.SignType == "MD5+RSA" {
		md5Sign := s.calcMD5FromStr(signStr)
		params["sign"] = md5Sign
		params["sign_type"] = "MD5+RSA"
		if rsaSign, err := s.signRSASign(signStr); err == nil {
			params["rsa_sign"] = rsaSign
		} else {
			utils.LogError("易支付RSA签名生成失败", err, nil)
		}
	} else if s.SignType == "RSA" {
		if rsaSign, err := s.signRSASign(signStr); err == nil {
			params["sign"] = rsaSign
			params["sign_type"] = "RSA"
		} else {
			utils.LogError("易支付RSA签名生成失败", err, nil)
			return "", fmt.Errorf("RSA签名生成失败: %v", err)
		}
	}

	utils.LogInfo("易支付发起请求: URL=%s, Order=%s, Amount=%s, SignType=%s", s.APIURL, order.OrderNo, params["money"], s.SignType)
	rsaSignPreview := "(无)"
	if rsa, ok := params["rsa_sign"]; ok && len(rsa) > 20 {
		rsaSignPreview = rsa[:20] + "..."
	} else if rsa, ok := params["rsa_sign"]; ok {
		rsaSignPreview = rsa
	}
	utils.LogInfo("易支付请求参数: pid=%s, type=%s, out_trade_no=%s, money=%s, sign=%s, rsa_sign=%s",
		params["pid"], params["type"], params["out_trade_no"], params["money"],
		params["sign"], rsaSignPreview)

	respBytes, err := s.postForm(s.APIURL, params)
	if err != nil {
		return "", err
	}

	respStr := string(respBytes)
	if strings.HasPrefix(respStr, "<!DOCTYPE") || strings.HasPrefix(respStr, "<html") {
		if strings.Contains(respStr, "404") || strings.Contains(respStr, "Not Found") {
			return "", fmt.Errorf("易支付API 404错误，请检查网关地址")
		}
		return "", fmt.Errorf("易支付返回HTML页面而非JSON，可能配置错误或被拦截")
	}
	if strings.HasPrefix(respStr, "http://") || strings.HasPrefix(respStr, "https://") {
		return respStr, nil
	}

	var rawResp map[string]interface{}
	if err := json.Unmarshal(respBytes, &rawResp); err != nil {
		utils.LogError("易支付解析响应失败", err, map[string]interface{}{"resp": respStr})
		return "", fmt.Errorf("易支付解析失败: %v", err)
	}

	var yipayResp *YipayResponse
	if s.Adapter != nil {
		yipayResp = s.Adapter.NormalizeResponse(rawResp)
	} else {
		var standardResp YipayResponse
		if err := json.Unmarshal(respBytes, &standardResp); err == nil {
			yipayResp = &standardResp
		} else {
			return "", fmt.Errorf("易支付解析失败: %v", err)
		}
	}

	utils.LogInfo("易支付返回结果 [平台=%s]: code=%d, msg=%s, trade_no=%s, device=%s, payurl=%s, qrcode=%s, urlscheme=%s",
		s.PlatformName, yipayResp.Code, yipayResp.Msg, yipayResp.TradeNo, deviceType, yipayResp.PayURL, yipayResp.QRCode, yipayResp.URLScheme)

	if yipayResp.Code != 1 {
		return "", fmt.Errorf("易支付API错误: %s (code: %d)", yipayResp.Msg, yipayResp.Code)
	}

	if yipayResp.URLScheme != "" {
		utils.LogInfo("易支付返回URLScheme: %s", yipayResp.URLScheme)
		return yipayResp.URLScheme, nil
	}

	if yipayResp.PayURL != "" {
		utils.LogInfo("易支付返回PayURL: %s", yipayResp.PayURL)
		return yipayResp.PayURL, nil
	}

	if yipayResp.QRCode != "" {
		utils.LogInfo("易支付返回QRCode: %s", yipayResp.QRCode)
		return yipayResp.QRCode, nil
	}

	return "", fmt.Errorf("易支付未返回有效支付链接")
}

func (s *YipayService) postForm(apiURL string, params map[string]string) ([]byte, error) {
	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.PostForm(apiURL, formData)
	if err != nil {
		utils.LogError("易支付请求网络错误", err, map[string]interface{}{"url": apiURL})
		return nil, fmt.Errorf("网络请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyStr := string(body)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500] + "..."
		}
		utils.LogError("易支付HTTP状态异常", nil, map[string]interface{}{
			"status": resp.StatusCode,
			"url":    apiURL,
			"body":   bodyStr,
		})
		utils.LogError("易支付请求详情", nil, map[string]interface{}{
			"request_url":   apiURL,
			"status_code":   resp.StatusCode,
			"response_body": bodyStr,
		})
		return nil, fmt.Errorf("API状态码异常: %d, 响应: %s", resp.StatusCode, bodyStr)
	}

	return body, nil
}

func (s *YipayService) Sign(params map[string]string) string {
	return s.calculateMD5Sign(params)
}

func (s *YipayService) VerifyNotify(params map[string]string) bool {
	sign, ok := params["sign"]
	if !ok || sign == "" {
		utils.LogError("易支付回调缺少签名", nil, nil)
		return false
	}

	signType := s.SignType
	if t, ok := params["sign_type"]; ok && t != "" {
		signType = t
	}

	signStr := buildSignString(params, "sign", "sign_type", "rsa_sign")
	utils.LogInfo("易支付验签字符串: %s", signStr)

	switch signType {
	case "RSA":
		return s.verifyRSASign(signStr, sign)
	case "MD5+RSA":
		md5Sign := s.calcMD5FromStr(signStr)
		if !strings.EqualFold(sign, md5Sign) {
			utils.LogError("MD5+RSA模式: MD5校验失败", nil, nil)
			return false
		}
		if rsaSign, ok := params["rsa_sign"]; ok && rsaSign != "" {
			return s.verifyRSASign(signStr, rsaSign)
		}
		utils.LogWarn("MD5+RSA模式: 缺少rsa_sign，仅通过MD5校验")
		return true
	default: // MD5
		calcSign := s.calcMD5FromStr(signStr)
		match := strings.EqualFold(sign, calcSign)
		if !match {
			utils.LogError("MD5校验失败", nil, map[string]interface{}{
				"received": sign, "calculated": calcSign, "signStr": signStr + "&key=***",
			})
		}
		return match
	}
}

func (s *YipayService) calculateMD5Sign(params map[string]string) string {
	signStr := buildSignString(params, "sign", "sign_type", "rsa_sign")
	return s.calcMD5FromStr(signStr)
}

func (s *YipayService) calcMD5FromStr(signStr string) string {
	fullStr := signStr + s.Key
	hash := md5.Sum([]byte(fullStr))
	return strings.ToLower(fmt.Sprintf("%x", hash))
}

func (s *YipayService) verifyRSASign(content, sign string) bool {
	if s.PlatformPublicKey == "" {
		utils.LogError("RSA验签失败: 未配置平台公钥", nil, nil)
		return false
	}

	var pubKeyBytes []byte
	var err error

	block, _ := pem.Decode([]byte(s.PlatformPublicKey))
	if block != nil {
		pubKeyBytes = block.Bytes
	} else {
		pubKeyBytes, err = base64.StdEncoding.DecodeString(s.PlatformPublicKey)
		if err != nil {
			utils.LogError("RSA公钥格式错误", err, nil)
			return false
		}
	}

	pubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		utils.LogError("RSA公钥解析失败", err, nil)
		return false
	}

	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return false
	}

	signBytes, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return false
	}

	hashed := sha256.Sum256([]byte(content))
	err = rsa.VerifyPKCS1v15(rsaPubKey, crypto.SHA256, hashed[:], signBytes)
	if err != nil {
		utils.LogError("RSA签名校验不通过", err, nil)
		return false
	}
	return true
}

func (s *YipayService) signRSASign(content string) (string, error) {
	if s.MerchantPrivateKey == "" {
		utils.LogError("易支付RSA签名: 商户私钥为空", nil, nil)
		return "", fmt.Errorf("商户私钥未配置")
	}
	utils.LogInfo("易支付RSA签名: 私钥长度=%d, 内容前50字符=%s", len(s.MerchantPrivateKey), s.MerchantPrivateKey[:min(50, len(s.MerchantPrivateKey))])

	var privKeyBytes []byte
	var err error

	block, _ := pem.Decode([]byte(s.MerchantPrivateKey))
	if block != nil {
		privKeyBytes = block.Bytes
	} else {
		privKeyBytes, err = base64.StdEncoding.DecodeString(s.MerchantPrivateKey)
		if err != nil {
			return "", fmt.Errorf("RSA私钥格式错误: %v", err)
		}
	}

	privKey, err := x509.ParsePKCS8PrivateKey(privKeyBytes)
	if err != nil {
		privKey, err = x509.ParsePKCS1PrivateKey(privKeyBytes)
		if err != nil {
			return "", fmt.Errorf("RSA私钥解析失败: %v", err)
		}
	}

	rsaPrivKey, ok := privKey.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("不是有效的RSA私钥")
	}

	hashed := sha256.Sum256([]byte(content))
	signBytes, err := rsa.SignPKCS1v15(nil, rsaPrivKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", fmt.Errorf("RSA签名失败: %v", err)
	}

	return base64.StdEncoding.EncodeToString(signBytes), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *YipayService) extractQRCodeFromPaymentPage(pageURL string, paymentType string) (string, error) {
	utils.LogInfo("开始从页面提取二维码: %s", pageURL)

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 15 {
				return fmt.Errorf("重定向过多")
			}
			return nil
		},
	}

	req, _ := http.NewRequest("GET", pageURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求页面失败: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	htmlContent := string(bodyBytes)

	if strings.Contains(htmlContent, "submit.php") {
		return s.handleFormRedirect(htmlContent, paymentType)
	}

	if redirect, ok := matchJSRedirect(htmlContent); ok {
		target := s.resolveRelativeURL(redirect, pageURL)
		return s.extractQRCodeFromPaymentPage(target, paymentType)
	}

	return s.extractQRCodeFromHTML(htmlContent, pageURL, paymentType)
}

func (s *YipayService) handleFormRedirect(htmlContent, paymentType string) (string, error) {
	formRe := regexp.MustCompile(`<form[^>]*action=["']([^"']+)["'][^>]*>([\s\S]*?)</form>`)
	matches := formRe.FindStringSubmatch(htmlContent)
	if len(matches) < 3 {
		return "", fmt.Errorf("未找到重定向表单")
	}
	actionURL := matches[1]
	formBody := matches[2]

	data := url.Values{}
	inputRe := regexp.MustCompile(`<input[^>]*name=["']([^"']+)["'][^>]*value=["']([^"']*)["']`)
	for _, m := range inputRe.FindAllStringSubmatch(formBody, -1) {
		data.Set(m[1], m[2])
	}

	resp, err := http.PostForm(actionURL, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	postHTML := string(body)

	if redirect, ok := matchJSRedirect(postHTML); ok {
		target := s.resolveRelativeURL(redirect, actionURL)
		return s.extractQRCodeFromPaymentPage(target, paymentType)
	}
	return s.extractQRCodeFromHTML(postHTML, actionURL, paymentType)
}

func matchJSRedirect(html string) (string, bool) {
	re := regexp.MustCompile(`window\.location\.(replace|href)\s*=\s*["']([^"']+)["']`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 2 {
		return matches[2], true
	}
	return "", false
}

func (s *YipayService) resolveRelativeURL(rel, base string) string {
	if strings.HasPrefix(rel, "http") {
		return rel
	}
	u, _ := url.Parse(base)
	if u != nil {
		if strings.HasPrefix(rel, "/") {
			return u.Scheme + "://" + u.Host + rel
		}
		return u.Scheme + "://" + u.Host + "/" + rel
	}
	return rel
}

func (s *YipayService) extractQRCodeFromHTML(html, baseURL, paymentType string) (string, error) {
	jsPattern := regexp.MustCompile(`(code_url|url)\s*[:=]\s*["']([^"']+)["']`)
	if m := jsPattern.FindStringSubmatch(html); len(m) > 2 {
		val := m[2]
		if strings.HasPrefix(val, "weixin") || strings.HasPrefix(val, "alipays") || strings.HasPrefix(val, "http") {
			return val, nil
		}
	}

	patterns := s.getQRCodePatterns(paymentType)
	for _, p := range patterns {
		re := regexp.MustCompile(p)
		if m := re.FindStringSubmatch(html); len(m) > 1 {
			return s.resolveRelativeURL(m[1], baseURL), nil
		}
	}

	if m := regexp.MustCompile(`data:image/[^;]+;base64,([A-Za-z0-9+/=]{100,})`).FindStringSubmatch(html); len(m) > 0 {
		return m[0], nil
	}

	return "", fmt.Errorf("未找到二维码")
}

func (s *YipayService) getQRCodePatterns(paymentType string) []string {
	common := []string{
		`<img[^>]*src=["']([^"']*qrcode[^"']*)["']`,
		`<img[^>]*class=["'][^"']*qrcode[^"']*["'][^>]*src=["']([^"']+)["']`,
		`<div[^>]*class=["'][^"']*qrcode[^"']*["'][^>]*>.*?<img[^>]*src=["']([^"']+)["']`,
	}
	if paymentType == "wxpay" {
		return append(common, `weixin://wxpay[^"'\s]+`, `wxp://[^"'\s]+`)
	}
	if paymentType == "alipay" {
		return append(common, `alipays://[^"'\s]+`)
	}
	return common
}

func GetYipaySupportedTypes(paymentConfig *models.PaymentConfig) []string {
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

type YipayRefundResponse struct {
	Code        int    `json:"code"`
	Msg         string `json:"msg"`
	RefundNo    string `json:"refund_no,omitempty"`     // 平台退款单号（新格式）
	OutRefundNo string `json:"out_refund_no,omitempty"` // 商户退款单号（新格式）
	TradeNo     string `json:"trade_no,omitempty"`      // 平台订单号（新格式）
	Money       string `json:"money,omitempty"`         // 退款金额（新格式）
	ReduceMoney string `json:"reducemoney,omitempty"`   // 扣减商户余额（新格式）
	Timestamp   string `json:"timestamp,omitempty"`     // 时间戳（新格式）
	Sign        string `json:"sign,omitempty"`          // 签名（新格式）
	SignType    string `json:"sign_type,omitempty"`     // 签名类型（新格式）
}

// RefundOrder 提交订单退款（兼容两种API格式）
// orderNo: 商户订单号
// tradeNo: 易支付订单号（可选，如果提供则优先使用）
// refundAmount: 退款金额
func (s *YipayService) RefundOrder(orderNo, tradeNo string, refundAmount float64) error {
	if orderNo == "" && tradeNo == "" {
		return fmt.Errorf("订单号不能为空")
	}
	if refundAmount <= 0 {
		return fmt.Errorf("退款金额必须大于0")
	}

	// 检测退款API格式
	refundURL, isNewFormat := s.buildRefundURL()

	var params map[string]string
	if isNewFormat {
		// 新格式：需要timestamp和签名
		params = s.buildNewFormatRefundParams(orderNo, tradeNo, refundAmount)
	} else {
		// 旧格式：标准格式
		params = s.buildOldFormatRefundParams(orderNo, tradeNo, refundAmount)
	}

	utils.LogInfo("易支付退款请求: URL=%s, format=%s, order_no=%s, trade_no=%s, refund_amount=%.2f",
		refundURL, map[bool]string{true: "新格式", false: "旧格式"}[isNewFormat], orderNo, tradeNo, refundAmount)

	respBytes, err := s.postForm(refundURL, params)
	if err != nil {
		return fmt.Errorf("退款请求失败: %v", err)
	}

	respStr := string(respBytes)
	utils.LogInfo("易支付退款响应: %s", respStr)

	var refundResp YipayRefundResponse
	if err := json.Unmarshal(respBytes, &refundResp); err != nil {
		utils.LogError("易支付退款响应解析失败", err, map[string]interface{}{"resp": respStr})
		return fmt.Errorf("退款响应解析失败: %v", err)
	}

	if refundResp.Code != 0 {
		utils.LogError("易支付退款失败", nil, map[string]interface{}{
			"code": refundResp.Code,
			"msg":  refundResp.Msg,
		})
		return fmt.Errorf("退款失败: %s (code: %d)", refundResp.Msg, refundResp.Code)
	}

	// 记录退款成功信息
	refundInfo := fmt.Sprintf("order_no=%s, trade_no=%s, refund_amount=%.2f, msg=%s", orderNo, tradeNo, refundAmount, refundResp.Msg)
	if refundResp.RefundNo != "" {
		refundInfo += fmt.Sprintf(", refund_no=%s", refundResp.RefundNo)
	}
	if refundResp.OutRefundNo != "" {
		refundInfo += fmt.Sprintf(", out_refund_no=%s", refundResp.OutRefundNo)
	}
	utils.LogInfo("易支付退款成功: %s", refundInfo)
	return nil
}

// buildRefundURL 构建退款API地址，返回URL和是否为新格式
func (s *YipayService) buildRefundURL() (string, bool) {
	apiURL := s.APIURL
	apiURLLower := strings.ToLower(apiURL)

	// 检测是否是新格式（包含 /api/pay/ 路径或特定域名）
	// 新格式特征：包含 /api/pay/ 或域名包含 myzfw.com
	if strings.Contains(apiURL, "/api/pay/") || strings.Contains(apiURLLower, "myzfw.com") {
		// 新格式：/api/pay/refund
		baseURL := apiURL
		// 移除可能的查询参数
		if idx := strings.Index(baseURL, "?"); idx > 0 {
			baseURL = baseURL[:idx]
		}
		// 移除可能的路径参数（如 /mapi.php, /api.php）
		if strings.Contains(baseURL, "/mapi.php") {
			baseURL = strings.Replace(baseURL, "/mapi.php", "", 1)
		}
		if strings.Contains(baseURL, "/api.php") {
			baseURL = strings.Split(baseURL, "/api.php")[0]
		}
		// 移除末尾的斜杠
		baseURL = strings.TrimSuffix(baseURL, "/")
		// 构建新格式退款URL
		return baseURL + "/api/pay/refund", true
	}

	// 旧格式：api.php?act=refund
	refundURL := apiURL
	// 移除可能的查询参数
	if idx := strings.Index(refundURL, "?"); idx > 0 {
		refundURL = refundURL[:idx]
	}

	if strings.Contains(refundURL, "api.php") {
		// 如果已经包含api.php，添加或替换act参数
		if strings.Contains(apiURL, "act=") {
			// 替换现有的act参数
			refundURL = strings.Split(apiURL, "act=")[0] + "act=refund"
			if idx := strings.Index(apiURL, "&"); idx > 0 && strings.Contains(apiURL[idx:], "&") {
				// 保留其他参数
				parts := strings.Split(apiURL, "&")
				for _, part := range parts {
					if !strings.HasPrefix(part, "act=") {
						refundURL += "&" + part
					}
				}
			}
		} else {
			refundURL = refundURL + "?act=refund"
		}
	} else {
		// 如果不包含api.php，添加路径
		if strings.HasSuffix(refundURL, "/") {
			refundURL = strings.TrimSuffix(refundURL, "/") + "/api.php?act=refund"
		} else {
			refundURL = refundURL + "/api.php?act=refund"
		}
	}
	return refundURL, false
}

// buildOldFormatRefundParams 构建旧格式退款参数
func (s *YipayService) buildOldFormatRefundParams(orderNo, tradeNo string, refundAmount float64) map[string]string {
	params := map[string]string{
		"pid":   s.PID,
		"key":   s.Key,
		"money": fmt.Sprintf("%.2f", refundAmount),
	}

	// trade_no 和 out_trade_no 不能同时为空，如果都传了以trade_no为准
	if tradeNo != "" {
		params["trade_no"] = tradeNo
	} else {
		params["out_trade_no"] = orderNo
	}

	return params
}

// buildNewFormatRefundParams 构建新格式退款参数（需要签名）
func (s *YipayService) buildNewFormatRefundParams(orderNo, tradeNo string, refundAmount float64) map[string]string {
	// 生成商户退款单号（避免重复请求）
	outRefundNo := fmt.Sprintf("REF%s%d", orderNo, time.Now().Unix())

	// 生成时间戳（10位整数，单位秒）
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	params := map[string]string{
		"pid":           s.PID,
		"money":         fmt.Sprintf("%.2f", refundAmount),
		"out_refund_no": outRefundNo,
		"timestamp":     timestamp,
	}

	// trade_no 和 out_trade_no 不能同时为空，如果都传了以trade_no为准
	if tradeNo != "" {
		params["trade_no"] = tradeNo
	} else {
		params["out_trade_no"] = orderNo
	}

	// 设置签名类型
	signType := s.SignType
	if signType == "" {
		signType = "RSA" // 新格式默认使用RSA
	}
	params["sign_type"] = signType

	// 生成签名
	signStr := buildSignString(params, "sign", "sign_type", "rsa_sign")
	var sign string
	var err error

	switch signType {
	case "RSA":
		sign, err = s.signRSASign(signStr)
		if err != nil {
			utils.LogError("易支付退款RSA签名生成失败", err, nil)
			// 如果RSA签名失败，尝试使用MD5（某些平台可能支持）
			sign = s.calcMD5FromStr(signStr)
			params["sign_type"] = "MD5"
		}
	case "MD5+RSA":
		md5Sign := s.calcMD5FromStr(signStr)
		params["sign"] = md5Sign
		if rsaSign, err := s.signRSASign(signStr); err == nil {
			params["rsa_sign"] = rsaSign
		} else {
			utils.LogWarn("易支付退款MD5+RSA模式: RSA签名生成失败，仅使用MD5", nil)
		}
		sign = md5Sign
	default: // MD5
		sign = s.calcMD5FromStr(signStr)
	}

	params["sign"] = sign

	return params
}

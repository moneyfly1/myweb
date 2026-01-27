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

// --- Constants & Structs ---

type YipayService struct {
	PID                string
	Key                string
	SignType           string
	PlatformPublicKey  string
	MerchantPrivateKey string
	APIURL             string
	NotifyURL          string
	ReturnURL          string
}

type YipayResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		TradeNo    string `json:"trade_no"`
		OutTradeNo string `json:"out_trade_no"`
		PayURL     string `json:"pay_url"`
		QRCode     string `json:"qrcode"`
		Img        string `json:"img"`
	} `json:"data"`
}

// --- Helper Functions ---

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

// resolveCallbackURL 统一处理回调地址生成的优先级：Config字段 -> JSON配置 -> 数据库自动生成
func resolveCallbackURL(explicit sql.NullString, jsonVal string, path string, isNotify bool) string {
	// 1. 优先使用数据库字段显式配置
	if explicit.Valid && explicit.String != "" {
		urlStr := strings.TrimSpace(explicit.String)
		utils.LogInfo("易支付使用配置的回调地址: %s", urlStr)
		return urlStr
	}

	// 2. 使用JSON中的配置
	if jsonVal != "" {
		utils.LogInfo("易支付从配置JSON中获取回调地址: %s", jsonVal)
		// 检查生产环境配置了本地地址的警告
		if isNotify && !isLocalDomain(jsonVal) {
			if domain := utils.GetDomainFromDB(database.GetDB()); isLocalDomain(domain) {
				utils.LogWarn("易支付回调地址配置为生产域名 (%s)，但当前环境是本地 (%s)，回调可能无法到达", jsonVal, domain)
			}
		}
		return jsonVal
	}

	// 3. 自动生成
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

// buildSignString 构建待签名字符串 (按key排序，排除空值和指定key)
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

// --- Service Implementation ---

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

	key := ""
	if paymentConfig.MerchantPrivateKey.Valid {
		key = strings.TrimSpace(paymentConfig.MerchantPrivateKey.String)
	}
	if key == "" && (signType == "MD5" || signType == "MD5+RSA") {
		return nil, fmt.Errorf("易支付MD5密钥未配置")
	}

	// RSA Key Logic
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

	// API URL Logic
	apiURL := getConfigString(configData, "gateway_url")
	if apiURL != "" {
		apiURL = strings.TrimSuffix(apiURL, "/") + "/openapi/pay/create"
	} else {
		apiURL = getConfigString(configData, "api_url")
		if strings.Contains(apiURL, "/api/pay/create") {
			apiURL = strings.Replace(apiURL, "/api/pay/create", "/openapi/pay/create", 1)
		}
	}
	if apiURL == "" {
		return nil, fmt.Errorf("易支付API地址未配置")
	}

	utils.LogInfo("易支付初始化: api_url=%s, pid=%s, sign_type=%s", apiURL, pid, signType)

	return &YipayService{
		PID:                pid,
		Key:                key,
		SignType:           signType,
		PlatformPublicKey:  platformPublicKey,
		MerchantPrivateKey: merchantPrivateKey,
		APIURL:             apiURL,
		NotifyURL:          resolveCallbackURL(paymentConfig.NotifyURL, getConfigString(configData, "notify_url"), "/api/v1/payment/notify/yipay", true),
		ReturnURL:          resolveCallbackURL(paymentConfig.ReturnURL, "", "/payment/return", false),
	}, nil
}

func (s *YipayService) CreatePayment(order *models.Order, amount float64, paymentType string) (string, error) {
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

	// 1. 构建参数
	params := map[string]string{
		"pid":          s.PID,
		"paytype_code": paymentType,
		"out_trade_no": order.OrderNo,
		"total_amount": fmt.Sprintf("%.2f", amount),
		"subject":      fmt.Sprintf("订单支付-%s", order.OrderNo),
		"timestamp":    fmt.Sprintf("%d", time.Now().Unix()),
		"sign_type":    s.SignType,
	}

	if s.NotifyURL == "" {
		return "", fmt.Errorf("回调地址未配置")
	}
	params["notify_url"] = s.NotifyURL

	if s.ReturnURL != "" {
		returnURL := s.ReturnURL
		sep := "?"
		if strings.Contains(returnURL, "?") {
			sep = "&"
		}
		if !strings.Contains(returnURL, "out_trade_no") {
			returnURL = fmt.Sprintf("%s%sout_trade_no=%s", returnURL, sep, order.OrderNo)
		}
		params["return_url"] = returnURL
	}

	// 2. 签名
	params["sign"] = s.Sign(params)

	utils.LogInfo("易支付发起请求: URL=%s, Order=%s, Amount=%s", s.APIURL, order.OrderNo, params["total_amount"])

	// 3. 发送请求
	respBytes, err := s.postForm(s.APIURL, params)
	if err != nil {
		return "", err
	}

	respStr := string(respBytes)
	// 4. 处理特殊响应 (HTML 或 URL)
	if strings.HasPrefix(respStr, "<!DOCTYPE") || strings.HasPrefix(respStr, "<html") {
		// 检查是否有错误提示
		if strings.Contains(respStr, "404") || strings.Contains(respStr, "Not Found") {
			return "", fmt.Errorf("易支付API 404错误，请检查网关地址")
		}
		return "", fmt.Errorf("易支付返回HTML页面而非JSON，可能配置错误或被拦截")
	}
	if strings.HasPrefix(respStr, "http://") || strings.HasPrefix(respStr, "https://") {
		return respStr, nil
	}

	// 5. 解析 JSON
	var yipayResp YipayResponse
	if err := json.Unmarshal(respBytes, &yipayResp); err != nil {
		utils.LogError("易支付解析响应失败", err, map[string]interface{}{"resp": respStr})
		return "", fmt.Errorf("易支付解析失败: %v", err)
	}

	if yipayResp.Code != 1 {
		return "", fmt.Errorf("易支付API错误: %s (code: %d)", yipayResp.Msg, yipayResp.Code)
	}

	// 6. 返回结果优先级: Img > QRCode > PayURL
	if yipayResp.Data.Img != "" {
		return yipayResp.Data.Img, nil
	}
	if yipayResp.Data.QRCode != "" {
		return yipayResp.Data.QRCode, nil
	}
	if yipayResp.Data.PayURL != "" {
		return yipayResp.Data.PayURL, nil
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
		utils.LogError("易支付HTTP状态异常", nil, map[string]interface{}{"status": resp.StatusCode, "body": string(body)})
		return nil, fmt.Errorf("API状态码异常: %d", resp.StatusCode)
	}

	return body, nil
}

// --- Signature Logic ---

func (s *YipayService) Sign(params map[string]string) string {
	// 注意：根据原始逻辑，即便是RSA模式，请求签名也使用MD5逻辑。
	// 这通常取决于具体的易支付实现，保持原逻辑不变。
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

	// 过滤不需要签名的参数
	signStr := buildSignString(params, "sign", "sign_type", "rsa_sign")
	utils.LogInfo("易支付验签字符串: %s", signStr)

	switch signType {
	case "RSA":
		return s.verifyRSASign(signStr, sign)
	case "MD5+RSA":
		// 先验MD5
		md5Sign := s.calcMD5FromStr(signStr)
		if !strings.EqualFold(sign, md5Sign) {
			utils.LogError("MD5+RSA模式: MD5校验失败", nil, nil)
			return false
		}
		// 再验RSA
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
	fullStr := signStr + "&key=" + s.Key
	hash := md5.Sum([]byte(fullStr))
	return strings.ToUpper(fmt.Sprintf("%x", hash))
}

func (s *YipayService) verifyRSASign(content, sign string) bool {
	if s.PlatformPublicKey == "" {
		utils.LogError("RSA验签失败: 未配置平台公钥", nil, nil)
		return false
	}

	var pubKeyBytes []byte
	var err error

	// 尝试作为 PEM 解析
	block, _ := pem.Decode([]byte(s.PlatformPublicKey))
	if block != nil {
		pubKeyBytes = block.Bytes
	} else {
		// 尝试作为 Base64 解析
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

// --- Scraping Utilities (Unchanged functionality, improved structure) ---

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

	// 1. 处理特定的表单自动提交 (idzew.com / submit.php)
	if strings.Contains(htmlContent, "<form") && strings.Contains(htmlContent, "action=") && strings.Contains(htmlContent, "submit") {
		return s.handleFormRedirect(htmlContent, paymentType)
	}

	// 2. 处理 JS 跳转
	if redirect, ok := matchJSRedirect(htmlContent); ok {
		target := s.resolveRelativeURL(redirect, pageURL)
		return s.extractQRCodeFromPaymentPage(target, paymentType)
	}

	// 3. 提取二维码
	return s.extractQRCodeFromHTML(htmlContent, pageURL, paymentType)
}

func (s *YipayService) handleFormRedirect(htmlContent, paymentType string) (string, error) {
	// 简化的表单提取逻辑
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
	// 尝试 JS 变量 code_url
	jsPattern := regexp.MustCompile(`(code_url|url)\s*[:=]\s*["']([^"']+)["']`)
	if m := jsPattern.FindStringSubmatch(html); len(m) > 2 {
		val := m[2]
		if strings.HasPrefix(val, "weixin") || strings.HasPrefix(val, "alipays") || strings.HasPrefix(val, "http") {
			return val, nil
		}
	}

	// 尝试 img src
	patterns := s.getQRCodePatterns(paymentType)
	for _, p := range patterns {
		re := regexp.MustCompile(p)
		if m := re.FindStringSubmatch(html); len(m) > 1 {
			return s.resolveRelativeURL(m[1], baseURL), nil
		}
	}

	// 尝试 Base64
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

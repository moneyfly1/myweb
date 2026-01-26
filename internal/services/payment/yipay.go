package payment

import (
	"crypto"
	"crypto/md5"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
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

func NewYipayService(paymentConfig *models.PaymentConfig) (*YipayService, error) {
	pid := ""
	if paymentConfig.AppID.Valid {
		pid = strings.TrimSpace(paymentConfig.AppID.String)
	}
	if pid == "" {
		return nil, fmt.Errorf("易支付商户ID未配置，请在支付配置中设置AppID")
	}

	signType := "MD5"
	if paymentConfig.ConfigJSON.Valid {
		var configData map[string]interface{}
		if err := json.Unmarshal([]byte(paymentConfig.ConfigJSON.String), &configData); err == nil {
			if signTypeFromConfig, ok := configData["sign_type"].(string); ok && signTypeFromConfig != "" {
				signType = signTypeFromConfig
			}
		}
	}

	key := ""
	if paymentConfig.MerchantPrivateKey.Valid {
		key = strings.TrimSpace(paymentConfig.MerchantPrivateKey.String)
	}
	if key == "" && (signType == "MD5" || signType == "MD5+RSA") {
		return nil, fmt.Errorf("易支付MD5密钥未配置，请在支付配置中设置MerchantPrivateKey")
	}

	platformPublicKey := ""
	merchantPrivateKey := ""
	if signType == "RSA" || signType == "MD5+RSA" {
		if paymentConfig.AlipayPublicKey.Valid && paymentConfig.AlipayPublicKey.String != "" {
			platformPublicKey = strings.TrimSpace(paymentConfig.AlipayPublicKey.String)
			utils.LogInfo("易支付服务初始化: 从AlipayPublicKey字段获取平台公钥，长度=%d", len(platformPublicKey))
		}
		if platformPublicKey == "" && paymentConfig.ConfigJSON.Valid {
			var configData map[string]interface{}
			if err := json.Unmarshal([]byte(paymentConfig.ConfigJSON.String), &configData); err == nil {
				if pubKey, ok := configData["platform_public_key"].(string); ok && pubKey != "" {
					platformPublicKey = strings.TrimSpace(pubKey)
					utils.LogInfo("易支付服务初始化: 从ConfigJSON获取平台公钥，长度=%d", len(platformPublicKey))
				}
			}
		}

		if paymentConfig.ConfigJSON.Valid {
			var configData map[string]interface{}
			if err := json.Unmarshal([]byte(paymentConfig.ConfigJSON.String), &configData); err == nil {
				if privKey, ok := configData["merchant_private_key"].(string); ok && privKey != "" {
					merchantPrivateKey = strings.TrimSpace(privKey)
					utils.LogInfo("易支付服务初始化: 从ConfigJSON获取商户私钥，长度=%d", len(merchantPrivateKey))
				}
			}
		}

		if signType == "RSA" && (platformPublicKey == "" || merchantPrivateKey == "") {
			return nil, fmt.Errorf("易支付RSA签名需要配置平台公钥和商户私钥")
		}
		if signType == "MD5+RSA" && platformPublicKey == "" {
			return nil, fmt.Errorf("易支付MD5+RSA签名需要配置平台公钥")
		}
		if signType == "MD5+RSA" {
			utils.LogInfo("易支付服务初始化: MD5+RSA模式，平台公钥已配置，长度=%d", len(platformPublicKey))
		}
	}

	apiURL := ""
	if paymentConfig.ConfigJSON.Valid {
		var configData map[string]interface{}
		if err := json.Unmarshal([]byte(paymentConfig.ConfigJSON.String), &configData); err == nil {
			if gatewayURL, ok := configData["gateway_url"].(string); ok && gatewayURL != "" {
				gatewayURL = strings.TrimSuffix(strings.TrimSpace(gatewayURL), "/")
				apiURL = fmt.Sprintf("%s/openapi/pay/create", gatewayURL)
			} else if apiURLFromConfig, ok := configData["api_url"].(string); ok && apiURLFromConfig != "" {
				apiURL = strings.TrimSpace(apiURLFromConfig)
				if strings.Contains(apiURL, "/api/pay/create") {
					apiURL = strings.Replace(apiURL, "/api/pay/create", "/openapi/pay/create", 1)
				}
			}
		}
	}
	if apiURL == "" {
		return nil, fmt.Errorf("易支付API地址未配置，请在支付配置中设置网关地址或API地址")
	}
	utils.LogInfo("易支付服务初始化: api_url=%s, pid=%s, sign_type=%s", apiURL, pid, signType)

	notifyURL := ""
	if paymentConfig.NotifyURL.Valid && paymentConfig.NotifyURL.String != "" {
		notifyURL = strings.TrimSpace(paymentConfig.NotifyURL.String)
		utils.LogInfo("易支付使用配置的回调地址: %s", notifyURL)
	} else {
		// 优先从支付配置的 ConfigJSON 中获取 notify_url
		if paymentConfig.ConfigJSON.Valid {
			var configData map[string]interface{}
			if err := json.Unmarshal([]byte(paymentConfig.ConfigJSON.String), &configData); err == nil {
				if notifyURLFromConfig, ok := configData["notify_url"].(string); ok && notifyURLFromConfig != "" {
					notifyURL = strings.TrimSpace(notifyURLFromConfig)
					utils.LogInfo("易支付从配置JSON中获取回调地址: %s", notifyURL)
					// 检查是否是本地环境，如果是本地环境但配置了生产域名，给出警告
					notifyURLLower := strings.ToLower(notifyURL)
					isLocalEnv := strings.Contains(notifyURLLower, "localhost") ||
						strings.Contains(notifyURLLower, "127.0.0.1") ||
						strings.Contains(notifyURLLower, "192.168.") ||
						strings.Contains(notifyURLLower, "10.") ||
						strings.Contains(notifyURLLower, "172.")
					if !isLocalEnv {
						domain := utils.GetDomainFromDB(database.GetDB())
						domainLower := strings.ToLower(domain)
						if strings.Contains(domainLower, "localhost") || strings.Contains(domainLower, "127.0.0.1") {
							utils.LogWarn("易支付回调地址配置为生产域名 (%s)，但当前环境是本地 (%s)，回调可能无法到达", notifyURL, domain)
						}
					}
				}
			}
		}

		// 如果仍然为空，从数据库域名配置生成
		if notifyURL == "" {
			db := database.GetDB()
			if db != nil {
				domain := utils.GetDomainFromDB(db)
				utils.LogInfo("易支付从数据库读取域名配置: %s", domain)
				if domain != "" {
					// 检查是否是本地开发环境
					domainLower := strings.ToLower(domain)
					isLocal := strings.Contains(domainLower, "localhost") ||
						strings.Contains(domainLower, "127.0.0.1") ||
						strings.Contains(domainLower, "192.168.") ||
						strings.Contains(domainLower, "10.") ||
						strings.Contains(domainLower, "172.") ||
						strings.Contains(domainLower, "local")

					if isLocal {
						// 本地环境，使用 http
						if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
							notifyURL = fmt.Sprintf("%s/api/v1/payment/notify/yipay", strings.TrimSuffix(domain, "/"))
						} else {
							notifyURL = fmt.Sprintf("http://%s/api/v1/payment/notify/yipay", strings.TrimSuffix(domain, "/"))
						}
						utils.LogWarn("易支付本地环境自动生成回调地址: %s (域名: %s) - 注意：本地环境无法接收易支付回调，需要使用内网穿透工具", notifyURL, domain)
					} else {
						// 生产环境，使用 https
						baseURL := utils.FormatDomainURL(domain)
						notifyURL = fmt.Sprintf("%s/api/v1/payment/notify/yipay", baseURL)
						utils.LogInfo("易支付生产环境自动生成回调地址: %s (域名: %s)", notifyURL, domain)
					}
				} else {
					utils.LogWarn("易支付无法生成回调地址：域名配置为空，请在支付配置中设置 notify_url")
				}
			}
		}
	}

	returnURL := ""
	if paymentConfig.ReturnURL.Valid && paymentConfig.ReturnURL.String != "" {
		returnURL = strings.TrimSpace(paymentConfig.ReturnURL.String)
		utils.LogInfo("易支付使用配置的同步回调地址: %s", returnURL)
	} else {
		db := database.GetDB()
		if db != nil {
			domain := utils.GetDomainFromDB(db)
			if domain != "" {
				domainLower := strings.ToLower(domain)
				isLocal := strings.Contains(domainLower, "localhost") ||
					strings.Contains(domainLower, "127.0.0.1") ||
					strings.Contains(domainLower, "192.168.") ||
					strings.Contains(domainLower, "10.") ||
					strings.Contains(domainLower, "172.") ||
					strings.Contains(domainLower, "local")

				if isLocal {
					if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
						returnURL = fmt.Sprintf("%s/payment/return", strings.TrimSuffix(domain, "/"))
					} else {
						returnURL = fmt.Sprintf("http://%s/payment/return", strings.TrimSuffix(domain, "/"))
					}
				} else {
					baseURL := utils.FormatDomainURL(domain)
					returnURL = fmt.Sprintf("%s/payment/return", baseURL)
				}
				utils.LogInfo("易支付自动生成同步回调地址: %s", returnURL)
			}
		}
	}

	return &YipayService{
		PID:                pid,
		Key:                key,
		SignType:           signType,
		PlatformPublicKey:  platformPublicKey,
		MerchantPrivateKey: merchantPrivateKey,
		APIURL:             apiURL,
		NotifyURL:          notifyURL,
		ReturnURL:          returnURL,
	}, nil
}

func (s *YipayService) CreatePayment(order *models.Order, amount float64, paymentType string) (string, error) {
	if order == nil {
		return "", fmt.Errorf("订单信息不能为空")
	}
	if order.OrderNo == "" {
		return "", fmt.Errorf("订单号不能为空")
	}
	if amount <= 0 {
		return "", fmt.Errorf("支付金额必须大于0，当前金额: %.2f", amount)
	}
	if paymentType == "" {
		paymentType = "alipay"
		utils.LogWarn("易支付支付类型为空，使用默认值 alipay")
	}

	utils.LogInfo("易支付CreatePayment: paymentType=%s, order_no=%s, amount=%.2f", paymentType, order.OrderNo, amount)

	params := make(map[string]string)
	params["pid"] = s.PID
	params["paytype_code"] = paymentType
	params["out_trade_no"] = order.OrderNo
	params["total_amount"] = fmt.Sprintf("%.2f", amount)
	params["subject"] = fmt.Sprintf("订单支付-%s", order.OrderNo)
	params["timestamp"] = fmt.Sprintf("%d", time.Now().Unix())

	if s.NotifyURL != "" {
		params["notify_url"] = s.NotifyURL
	} else {
		utils.LogError("易支付回调地址未配置", nil, map[string]interface{}{
			"error": "notify_url is required for yipay",
		})
		return "", fmt.Errorf("易支付回调地址未配置，notify_url是必填参数")
	}

	returnURL := s.ReturnURL
	if returnURL == "" {
		db := database.GetDB()
		if db != nil {
			domain := utils.GetDomainFromDB(db)
			if domain != "" {
				domainLower := strings.ToLower(domain)
				isLocal := strings.Contains(domainLower, "localhost") ||
					strings.Contains(domainLower, "127.0.0.1") ||
					strings.Contains(domainLower, "192.168.") ||
					strings.Contains(domainLower, "10.") ||
					strings.Contains(domainLower, "172.") ||
					strings.Contains(domainLower, "local")

				if isLocal {
					if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
						returnURL = fmt.Sprintf("%s/payment/return", strings.TrimSuffix(domain, "/"))
					} else {
						returnURL = fmt.Sprintf("http://%s/payment/return", strings.TrimSuffix(domain, "/"))
					}
				} else {
					baseURL := utils.FormatDomainURL(domain)
					returnURL = fmt.Sprintf("%s/payment/return", baseURL)
				}
				utils.LogInfo("易支付自动生成同步回调地址: %s", returnURL)
			}
		}
	}

	if returnURL != "" {
		if !strings.Contains(returnURL, "out_trade_no") && !strings.Contains(returnURL, "order_no") {
			if strings.Contains(returnURL, "?") {
				returnURL = fmt.Sprintf("%s&out_trade_no=%s", returnURL, order.OrderNo)
			} else {
				returnURL = fmt.Sprintf("%s?out_trade_no=%s", returnURL, order.OrderNo)
			}
		}
		params["return_url"] = returnURL
		utils.LogInfo("易支付使用同步回调地址: %s", returnURL)
	} else {
		utils.LogWarn("易支付同步回调地址为空，支付完成后可能无法正确跳转")
	}

	if s.SignType == "" {
		s.SignType = "MD5"
	}
	params["sign_type"] = s.SignType

	sign := s.Sign(params)
	params["sign"] = sign

	paramsCopy := make(map[string]string)
	for k, v := range params {
		if k != "sign" && k != "rsa_sign" {
			paramsCopy[k] = v
		}
	}
	utils.LogInfo("易支付创建支付请求: api_url=%s, pid=%s, paytype_code=%s, out_trade_no=%s, total_amount=%s, sign_type=%s, notify_url=%s",
		s.APIURL, s.PID, paymentType, order.OrderNo, params["total_amount"], s.SignType, params["notify_url"])
	utils.LogInfo("易支付创建支付请求完整参数（不含签名）: %v", paramsCopy)

	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.PostForm(s.APIURL, formData)
	if err != nil {
		utils.LogError("易支付API请求失败", err, map[string]interface{}{
			"api_url":  s.APIURL,
			"order_no": order.OrderNo,
			"params":   params,
		})
		return "", fmt.Errorf("易支付API请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		responsePreview := string(body)
		if len(responsePreview) > 500 {
			responsePreview = responsePreview[:500] + "..."
		}
		utils.LogError("易支付API返回非200状态码", nil, map[string]interface{}{
			"status_code":      resp.StatusCode,
			"api_url":          s.APIURL,
			"request_method":   "POST",
			"request_params":   params,
			"response_preview": responsePreview,
		})
		if resp.StatusCode == 404 {
			return "", fmt.Errorf("易支付API地址不存在(404)，请检查API地址配置。当前地址: %s，请确认易支付网关地址和API路径是否正确", s.APIURL)
		}
		return "", fmt.Errorf("易支付API返回错误状态码: %d，请检查API地址和配置。当前地址: %s", resp.StatusCode, s.APIURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.LogError("易支付API响应读取失败", err, nil)
		return "", fmt.Errorf("易支付API响应读取失败: %v", err)
	}

	responseStr := string(body)

	if strings.HasPrefix(responseStr, "<!DOCTYPE") || strings.HasPrefix(responseStr, "<html") {
		responsePreview := responseStr
		if len(responsePreview) > 1000 {
			responsePreview = responsePreview[:1000] + "..."
		}
		utils.LogError("易支付API返回HTML页面", nil, map[string]interface{}{
			"api_url":          s.APIURL,
			"response_preview": responsePreview,
			"params":           params,
			"pid":              s.PID,
			"sign_type":        s.SignType,
		})

		if strings.Contains(responseStr, "404") || strings.Contains(responseStr, "Not Found") {
			return "", fmt.Errorf("易支付API地址不存在(404)，请检查API地址配置。当前地址: %s", s.APIURL)
		}
		if strings.Contains(responseStr, "403") || strings.Contains(responseStr, "Forbidden") {
			return "", fmt.Errorf("易支付API访问被拒绝(403)，请检查商户ID和密钥配置")
		}
		return "", fmt.Errorf("易支付API返回错误页面，请检查API地址、商户ID、密钥和签名方式配置。API地址: %s", s.APIURL)
	}

	var yipayResp YipayResponse
	if err := json.Unmarshal(body, &yipayResp); err != nil {
		utils.LogError("易支付API响应解析失败", err, map[string]interface{}{
			"response":     responseStr,
			"api_url":      s.APIURL,
			"paytype_code": paymentType,
		})

		if strings.HasPrefix(responseStr, "http://") || strings.HasPrefix(responseStr, "https://") {
			utils.LogInfo("易支付返回直接URL格式: %s (支付类型: %s)", responseStr, paymentType)
			return responseStr, nil
		}

		if len(responseStr) > 500 {
			responseStr = responseStr[:500] + "..."
		}
		return "", fmt.Errorf("易支付API响应解析失败: %v, 响应内容: %s", err, responseStr)
	}

	// 记录原始响应以便调试
	utils.LogInfo("易支付API原始响应: %s", responseStr)

	utils.LogInfo("易支付API响应解析成功: code=%d, msg=%s, pay_url=%s, qrcode=%s, img=%s (支付类型: %s)",
		yipayResp.Code, yipayResp.Msg, yipayResp.Data.PayURL, yipayResp.Data.QRCode, yipayResp.Data.Img, paymentType)

	// 记录完整的响应数据以便调试
	responseJSON, _ := json.Marshal(yipayResp.Data)
	utils.LogInfo("易支付API完整响应数据: %s", string(responseJSON))

	if yipayResp.Code != 1 {
		utils.LogError("易支付API返回错误", nil, map[string]interface{}{
			"code": yipayResp.Code,
			"msg":  yipayResp.Msg,
		})
		return "", fmt.Errorf("易支付API返回错误: %s (code: %d)", yipayResp.Msg, yipayResp.Code)
	}

	// 优先使用直接返回的二维码图片地址
	if yipayResp.Data.Img != "" {
		utils.LogInfo("易支付返回二维码图片地址: %s (订单号: %s, 支付类型: %s)", yipayResp.Data.Img, order.OrderNo, paymentType)
		return yipayResp.Data.Img, nil
	}

	// 其次使用二维码链接
	if yipayResp.Data.QRCode != "" {
		utils.LogInfo("易支付返回二维码链接: %s (订单号: %s, 支付类型: %s)", yipayResp.Data.QRCode, order.OrderNo, paymentType)
		return yipayResp.Data.QRCode, nil
	}

	// 易支付直接返回支付页面URL，让前端跳转到新页面
	if yipayResp.Data.PayURL != "" {
		utils.LogInfo("易支付返回支付页面URL: %s (订单号: %s, 支付类型: %s)", yipayResp.Data.PayURL, order.OrderNo, paymentType)
		if !strings.HasPrefix(yipayResp.Data.PayURL, "http://") && !strings.HasPrefix(yipayResp.Data.PayURL, "https://") {
			utils.LogWarn("易支付返回的支付链接格式异常: %s", yipayResp.Data.PayURL)
		}
		// 易支付统一返回支付页面URL，前端跳转到新页面
		return yipayResp.Data.PayURL, nil
	}

	return "", fmt.Errorf("易支付API未返回支付链接或二维码")
}

// extractQRCodeFromPaymentPage 从易支付支付页面URL中提取支付二维码（支持微信和支付宝）
func (s *YipayService) extractQRCodeFromPaymentPage(paymentPageURL string, paymentType string) (string, error) {
	utils.LogInfo("尝试从支付页面提取%s支付二维码: %s", paymentType, paymentPageURL)

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 允许最多15次重定向，跟随所有重定向直到获取到真正的支付页面
			if len(via) >= 15 {
				return fmt.Errorf("重定向次数过多（超过15次），可能陷入重定向循环")
			}
			redirectURL := req.URL.String()
			utils.LogInfo("支付页面重定向 %d: %s -> %s", len(via), via[len(via)-1].URL.String(), redirectURL)

			// 如果重定向到 idzew.com 或其他支付网关，继续跟随
			if strings.Contains(redirectURL, "idzew.com") || strings.Contains(redirectURL, "submit.php") {
				utils.LogInfo("跟随重定向到支付网关: %s", redirectURL)
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", paymentPageURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置User-Agent，模拟浏览器访问
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("访问支付页面失败: %v", err)
	}
	defer resp.Body.Close()

	utils.LogInfo("支付页面响应: status=%d, content_type=%s, location=%s",
		resp.StatusCode, resp.Header.Get("Content-Type"), resp.Header.Get("Location"))

	if resp.StatusCode != http.StatusOK {
		// 如果是重定向，HTTP客户端会自动跟随，这里不应该到达
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			location := resp.Header.Get("Location")
			if location != "" {
				utils.LogInfo("支付页面HTTP重定向到: %s，客户端将自动跟随", location)
				// HTTP客户端会自动跟随重定向，这里不应该返回错误
			}
		}
		// 如果状态码不是200，记录警告但继续处理（可能是302等重定向）
		utils.LogWarn("支付页面返回状态码: %d，继续尝试提取二维码", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取支付页面内容失败: %v", err)
	}

	htmlContent := string(body)
	utils.LogInfo("支付页面内容长度: %d 字符", len(htmlContent))

	// 检查是否是重定向页面（包含表单提交）
	if strings.Contains(htmlContent, "idzew.com") || strings.Contains(htmlContent, "submit.php") {
		// 提取表单提交的目标URL和所有表单字段
		formPattern := regexp.MustCompile(`<form[^>]*action=["']([^"']+)["'][^>]*>([\s\S]*?)</form>`)
		formMatches := formPattern.FindStringSubmatch(htmlContent)
		if len(formMatches) > 2 {
			submitURL := formMatches[1]
			formContent := formMatches[2]
			utils.LogInfo("发现表单提交URL，尝试POST提交: %s", submitURL)

			// 提取所有隐藏字段
			inputPattern := regexp.MustCompile(`<input[^>]*type=["']hidden["'][^>]*name=["']([^"']+)["'][^>]*value=["']([^"']*)["']`)
			formData := url.Values{}
			allInputs := inputPattern.FindAllStringSubmatch(formContent, -1)
			for _, input := range allInputs {
				if len(input) >= 3 {
					formData.Set(input[1], input[2])
					utils.LogInfo("表单字段: %s = %s", input[1], input[2])
				}
			}

			// 创建新的HTTP客户端用于POST请求
			postClient := &http.Client{
				Timeout: 30 * time.Second,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					// 允许最多15次重定向
					if len(via) >= 15 {
						return fmt.Errorf("重定向次数过多（超过15次）")
					}
					redirectURL := req.URL.String()
					utils.LogInfo("表单提交后重定向 %d: %s", len(via), redirectURL)
					return nil
				},
			}

			// POST提交表单
			postResp, err := postClient.PostForm(submitURL, formData)
			if err != nil {
				return "", fmt.Errorf("提交表单失败: %v", err)
			}
			defer postResp.Body.Close()

			postBody, err := io.ReadAll(postResp.Body)
			if err != nil {
				return "", fmt.Errorf("读取表单提交响应失败: %v", err)
			}

			postHTML := string(postBody)
			utils.LogInfo("表单提交响应: status=%d, content_length=%d", postResp.StatusCode, len(postHTML))

			// 检查是否有JavaScript跳转
			jsRedirectPattern := regexp.MustCompile(`window\.location\.(replace|href)\s*=\s*["']([^"']+)["']`)
			jsMatches := jsRedirectPattern.FindStringSubmatch(postHTML)
			if len(jsMatches) > 2 {
				redirectPath := jsMatches[2]
				// 如果是相对路径，需要构建完整URL
				if strings.HasPrefix(redirectPath, "/") {
					u, _ := url.Parse(submitURL)
					if u != nil {
						redirectPath = u.Scheme + "://" + u.Host + redirectPath
					}
				} else if !strings.HasPrefix(redirectPath, "http") {
					u, _ := url.Parse(submitURL)
					if u != nil {
						redirectPath = u.Scheme + "://" + u.Host + "/" + redirectPath
					}
				}
				utils.LogInfo("发现JavaScript跳转，访问: %s", redirectPath)
				// 递归调用，访问跳转后的页面
				return s.extractQRCodeFromPaymentPage(redirectPath, paymentType)
			}

			// 如果没有JavaScript跳转，直接在响应中查找二维码
			return s.extractQRCodeFromHTML(postHTML, submitURL, paymentType)
		}
	}

	// 检查是否有JavaScript跳转（在支付页面响应中）
	jsRedirectPattern := regexp.MustCompile(`window\.location\.(replace|href)\s*=\s*["']([^"']+)["']`)
	jsMatches := jsRedirectPattern.FindStringSubmatch(htmlContent)
	if len(jsMatches) > 2 {
		redirectPath := jsMatches[2]
		// 如果是相对路径，需要构建完整URL
		if strings.HasPrefix(redirectPath, "/") {
			u, _ := url.Parse(paymentPageURL)
			if u != nil {
				redirectPath = u.Scheme + "://" + u.Host + redirectPath
			}
		} else if !strings.HasPrefix(redirectPath, "http") {
			u, _ := url.Parse(paymentPageURL)
			if u != nil {
				redirectPath = u.Scheme + "://" + u.Host + "/" + redirectPath
			}
		}
		utils.LogInfo("发现JavaScript跳转，访问: %s", redirectPath)
		// 递归调用，访问跳转后的页面
		return s.extractQRCodeFromPaymentPage(redirectPath, paymentType)
	}

	// 从HTML中提取二维码
	return s.extractQRCodeFromHTML(htmlContent, paymentPageURL, paymentType)
}

// extractQRCodeFromHTML 从HTML内容中提取支付二维码
func (s *YipayService) extractQRCodeFromHTML(htmlContent string, baseURL string, paymentType string) (string, error) {
	// 优先提取JavaScript中的code_url变量（这是真正的支付二维码URL）
	// 支持多种格式：var code_url = '...' 或 code_url = "..." 或 code_url='...'
	codeURLPatterns := []string{
		`var\s+code_url\s*=\s*["']([^"']+)["']`,
		`code_url\s*=\s*["']([^"']+)["']`,
		`code_url\s*:\s*["']([^"']+)["']`,
		`"code_url"\s*:\s*["']([^"']+)["']`,
		`'code_url'\s*:\s*["']([^"']+)["']`,
	}

	for _, pattern := range codeURLPatterns {
		codeURLPattern := regexp.MustCompile(pattern)
		codeURLMatches := codeURLPattern.FindStringSubmatch(htmlContent)
		if len(codeURLMatches) > 1 && codeURLMatches[1] != "" {
			codeURL := codeURLMatches[1]
			utils.LogInfo("从JavaScript提取到code_url: %s (使用模式: %s)", codeURL, pattern)
			// 如果是支付链接（如微信支付链接、支付宝链接、HTTP/HTTPS链接），直接返回
			if strings.HasPrefix(codeURL, "weixin://") || strings.HasPrefix(codeURL, "wxp://") ||
				strings.HasPrefix(codeURL, "alipays://") || strings.HasPrefix(codeURL, "https://") ||
				strings.HasPrefix(codeURL, "http://") {
				return codeURL, nil
			}
		}
	}

	// 根据支付类型选择不同的提取模式
	var patterns []string
	if paymentType == "wxpay" {
		patterns = []string{
			`<img[^>]*src=["']([^"']*qrcode[^"']*)["']`,
			`<img[^>]*src=["']([^"']*wxpay[^"']*)["']`,
			`<img[^>]*src=["']([^"']*wechat[^"']*)["']`,
			`<img[^>]*src=["']([^"']*pay[^"']*qrcode[^"']*)["']`,
			`weixin://wxpay[^"'\s]+`,
			`wxp://[^"'\s]+`,
			`<img[^>]*id=["']qrcode["'][^>]*src=["']([^"']+)["']`,
			`<div[^>]*class=["'][^"']*qrcode[^"']*["'][^>]*>.*?<img[^>]*src=["']([^"']+)["']`,
			`<img[^>]*class=["'][^"']*qrcode[^"']*["'][^>]*src=["']([^"']+)["']`,
		}
	} else if paymentType == "alipay" {
		patterns = []string{
			`<img[^>]*src=["']([^"']*qrcode[^"']*)["']`,
			`<img[^>]*src=["']([^"']*alipay[^"']*)["']`,
			`<img[^>]*src=["']([^"']*pay[^"']*qrcode[^"']*)["']`,
			`alipays://[^"'\s]+`,
			`<img[^>]*id=["']qrcode["'][^>]*src=["']([^"']+)["']`,
			`<div[^>]*class=["'][^"']*qrcode[^"']*["'][^>]*>.*?<img[^>]*src=["']([^"']+)["']`,
			`<img[^>]*class=["'][^"']*qrcode[^"']*["'][^>]*src=["']([^"']+)["']`,
		}
	} else {
		patterns = []string{
			`<img[^>]*src=["']([^"']*qrcode[^"']*)["']`,
			`<img[^>]*src=["']([^"']*pay[^"']*qrcode[^"']*)["']`,
		}
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(htmlContent)
		if len(matches) > 1 && matches[1] != "" {
			qrURL := matches[1]
			// 如果是相对路径，转换为绝对路径
			if strings.HasPrefix(qrURL, "//") {
				qrURL = "https:" + qrURL
			} else if strings.HasPrefix(qrURL, "/") {
				u, _ := url.Parse(baseURL)
				if u != nil {
					qrURL = u.Scheme + "://" + u.Host + qrURL
				}
			}
			utils.LogInfo("从HTML提取到二维码URL: %s", qrURL)
			return qrURL, nil
		}
	}

	// 尝试查找base64编码的二维码
	base64Pattern := regexp.MustCompile(`data:image/[^;]+;base64,([A-Za-z0-9+/=]{100,})`)
	base64Matches := base64Pattern.FindStringSubmatch(htmlContent)
	if len(base64Matches) > 1 {
		utils.LogInfo("从HTML提取到base64二维码")
		return base64Matches[0], nil
	}

	// 尝试查找canvas中的二维码数据
	canvasPattern := regexp.MustCompile(`canvas[^>]*>.*?toDataURL\(['"]([^'"]+)['"]\)`)
	canvasMatches := canvasPattern.FindStringSubmatch(htmlContent)
	if len(canvasMatches) > 1 {
		utils.LogInfo("从HTML提取到canvas二维码")
		return canvasMatches[1], nil
	}

	preview := htmlContent
	if len(preview) > 1000 {
		preview = preview[:1000] + "..."
	}
	utils.LogWarn("未在HTML中找到%s支付二维码，页面内容预览: %s", paymentType, preview)

	return "", fmt.Errorf("未在HTML中找到%s支付二维码", paymentType)
}

func (s *YipayService) VerifyNotify(params map[string]string) bool {
	utils.LogInfo("易支付回调验证: 收到参数数量=%d, 参数列表=%v", len(params), params)

	sign, ok := params["sign"]
	if !ok || sign == "" {
		utils.LogError("易支付回调验证失败: 缺少签名参数", nil, map[string]interface{}{
			"params": params,
		})
		return false
	}

	signType := s.SignType
	if signTypeFromParams, ok := params["sign_type"]; ok && signTypeFromParams != "" {
		signType = signTypeFromParams
	}

	utils.LogInfo("易支付回调验证: sign_type=%s, received_sign=%s, key_length=%d, key_prefix=%s", signType, sign, len(s.Key), func() string {
		if len(s.Key) > 8 {
			return s.Key[:8] + "..."
		}
		return s.Key
	}())

	paramsCopy := make(map[string]string)
	for k, v := range params {
		if k != "sign" && k != "sign_type" && v != "" {
			paramsCopy[k] = v
		}
	}

	utils.LogInfo("易支付回调验证: 参与签名的参数数量=%d", len(paramsCopy))
	for k, v := range paramsCopy {
		utils.LogInfo("易支付回调验证: 参数[%s]=%s", k, v)
	}

	switch signType {
	case "RSA":
		return s.verifyRSASign(paramsCopy, sign)
	case "MD5+RSA":
		if s.Key == "" {
			utils.LogError("易支付MD5+RSA验证失败: MD5密钥为空", nil, nil)
			return false
		}
		if s.PlatformPublicKey == "" {
			utils.LogError("易支付MD5+RSA验证失败: 平台公钥为空", nil, nil)
			return false
		}
		
		md5Sign := s.calculateMD5Sign(paramsCopy)
		if !strings.EqualFold(sign, md5Sign) {
			utils.LogError("易支付MD5+RSA验证失败: MD5签名不匹配", nil, map[string]interface{}{
				"received_md5_sign":   sign,
				"calculated_md5_sign": md5Sign,
			})
			return false
		}
		utils.LogInfo("易支付MD5+RSA验证: MD5签名验证成功")
		
		if rsaSign, ok := params["rsa_sign"]; ok && rsaSign != "" {
			rsaValid := s.verifyRSASign(paramsCopy, rsaSign)
			if !rsaValid {
				utils.LogError("易支付MD5+RSA验证失败: RSA签名验证失败", nil, map[string]interface{}{
					"rsa_sign": rsaSign,
				})
				return false
			}
			utils.LogInfo("易支付MD5+RSA验证: RSA签名验证成功")
			return true
		}
		
		utils.LogWarn("易支付MD5+RSA验证: 未找到rsa_sign参数，仅验证MD5签名")
		return true
	case "MD5", "":
		fallthrough
	default:
		if s.Key == "" {
			utils.LogError("易支付回调验证失败: 密钥为空", nil, map[string]interface{}{
				"sign_type": signType,
			})
			return false
		}
		calculatedSign := s.calculateMD5Sign(paramsCopy)
		isValid := strings.EqualFold(sign, calculatedSign)
		if !isValid {
			utils.LogError("易支付回调验证失败: 签名不匹配", nil, map[string]interface{}{
				"received_sign":   sign,
				"calculated_sign": calculatedSign,
				"sign_type":       signType,
				"params_count":    len(paramsCopy),
				"key_length":      len(s.Key),
			})
			utils.LogError("易支付回调验证失败: 详细参数信息", nil, map[string]interface{}{
				"params": paramsCopy,
			})
			
			var keys []string
			for k := range paramsCopy {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			var debugStr strings.Builder
			for i, k := range keys {
				if i > 0 {
					debugStr.WriteString("&")
				}
				debugStr.WriteString(k)
				debugStr.WriteString("=")
				debugStr.WriteString(paramsCopy[k])
			}
			debugStr.WriteString("&key=")
			debugStr.WriteString(s.Key)
			utils.LogError("易支付回调验证失败: 调试签名字符串", nil, map[string]interface{}{
				"debug_sign_string": debugStr.String(),
			})
		} else {
			utils.LogInfo("易支付回调验证成功: 签名匹配 - received=%s, calculated=%s", sign, calculatedSign)
		}
		return isValid
	}
}

func (s *YipayService) Sign(params map[string]string) string {
	switch s.SignType {
	case "RSA":
		return s.generateRSASign(params)
	case "MD5+RSA":
		return s.calculateMD5Sign(params)
	case "MD5", "":
		fallthrough
	default:
		return s.calculateMD5Sign(params)
	}
}

func (s *YipayService) calculateMD5Sign(params map[string]string) string {
	if s.Key == "" {
		return ""
	}

	var keys []string
	for k, v := range params {
		if k != "sign" && k != "sign_type" && v != "" {
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
		value := params[k]
		signStr.WriteString(value)
		utils.LogInfo("易支付签名计算: 添加参数 %s=%s", k, value)
	}
	signStr.WriteString("&key=")
	signStr.WriteString(s.Key)

	signString := signStr.String()
	utils.LogInfo("易支付签名计算(官方文档方式): sign_string=%s (长度=%d, key_length=%d)", signString, len(signString), len(s.Key))
	utils.LogInfo("易支付签名计算: 参数排序后的key列表=%v", keys)
	utils.LogInfo("易支付签名计算: 完整签名字符串(包含密钥)=%s", signString)

	hash := md5.Sum([]byte(signString))
	signResult := strings.ToUpper(fmt.Sprintf("%x", hash))

	utils.LogInfo("易支付签名结果: sign=%s", signResult)

	return signResult
}

func (s *YipayService) generateRSASign(params map[string]string) string {
	if s.MerchantPrivateKey == "" {
		return ""
	}
	return s.calculateMD5Sign(params)
}

func (s *YipayService) verifyRSASign(params map[string]string, sign string) bool {
	if s.PlatformPublicKey == "" {
		utils.LogError("易支付RSA验证失败: 平台公钥为空", nil, nil)
		return false
	}

	var keys []string
	for k, v := range params {
		if k != "sign" && k != "sign_type" && k != "rsa_sign" && v != "" {
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

	signString := signStr.String()
	utils.LogInfo("易支付RSA验证: 待签名字符串=%s", signString)

	var pubKeyBytes []byte
	var err error

	block, _ := pem.Decode([]byte(s.PlatformPublicKey))
	if block != nil {
		pubKeyBytes = block.Bytes
		utils.LogInfo("易支付RSA验证: 使用PEM格式公钥")
	} else {
		pubKeyBytes, err = base64.StdEncoding.DecodeString(s.PlatformPublicKey)
		if err != nil {
			utils.LogError("易支付RSA验证失败: 平台公钥格式错误，既不是PEM也不是Base64", err, nil)
			return false
		}
		utils.LogInfo("易支付RSA验证: 使用Base64格式公钥")
	}

	pubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		utils.LogError("易支付RSA验证失败: 平台公钥解析失败", err, nil)
		return false
	}

	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		utils.LogError("易支付RSA验证失败: 平台公钥不是RSA公钥", nil, nil)
		return false
	}

	signBytes, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		utils.LogError("易支付RSA验证失败: 签名Base64解码失败", err, nil)
		return false
	}

	hashed := sha256.Sum256([]byte(signString))
	err = rsa.VerifyPKCS1v15(rsaPubKey, crypto.SHA256, hashed[:], signBytes)
	if err != nil {
		utils.LogError("易支付RSA验证失败: 签名验证失败", err, map[string]interface{}{
			"sign_string": signString,
			"sign":        sign,
		})
		return false
	}

	utils.LogInfo("易支付RSA验证成功")
	return true
}

func GetYipaySupportedTypes(paymentConfig *models.PaymentConfig) []string {
	defaultTypes := []string{"alipay", "wxpay"}

	if !paymentConfig.ConfigJSON.Valid {
		return defaultTypes
	}

	var configData map[string]interface{}
	if err := json.Unmarshal([]byte(paymentConfig.ConfigJSON.String), &configData); err != nil {
		return defaultTypes
	}

	if types, ok := configData["supported_types"].([]interface{}); ok {
		result := make([]string, 0, len(types))
		for _, t := range types {
			if typeStr, ok := t.(string); ok {
				result = append(result, typeStr)
			}
		}
		if len(result) > 0 {
			return result
		}
	}

	return defaultTypes
}

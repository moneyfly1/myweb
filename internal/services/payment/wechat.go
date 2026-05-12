package payment

import (
	"crypto/md5" // #nosec G501 - MD5 required by WeChat Pay API specification
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"cboard-go/internal/models"
)

type WechatService struct {
	AppID     string
	MchID     string
	APIKey    string
	NotifyURL string
	QueryURL  string
}

func NewWechatService(paymentConfig *models.PaymentConfig) (*WechatService, error) {
	configData := parseConfigData(paymentConfig.ConfigJSON)
	queryURL := getConfigString(configData, "query_url")
	if queryURL == "" {
		queryURL = "https://api.mch.weixin.qq.com/pay/orderquery"
	}
	return &WechatService{
		AppID:     paymentConfig.WechatAppID.String,
		MchID:     paymentConfig.WechatMchID.String,
		APIKey:    paymentConfig.WechatAPIKey.String,
		NotifyURL: paymentConfig.NotifyURL.String,
		QueryURL:  queryURL,
	}, nil
}

func (s *WechatService) CreatePayment(order *models.Order, amount float64) (string, error) {
	if s.APIKey == "" {
		return "", fmt.Errorf("API密钥未配置")
	}
	if s.AppID == "" || s.MchID == "" {
		return "", fmt.Errorf("微信支付未完整配置（缺少 AppID 或 MchID）")
	}

	params := make(map[string]string)
	params["appid"] = s.AppID
	params["mch_id"] = s.MchID
	params["nonce_str"] = generateNonceStr(32)
	params["body"] = "订单支付"
	params["out_trade_no"] = order.OrderNo
	params["total_fee"] = fmt.Sprintf("%.0f", amount*100)
	params["spbill_create_ip"] = "127.0.0.1"
	params["notify_url"] = s.NotifyURL
	params["trade_type"] = "NATIVE"

	sign := s.Sign(params)
	params["sign"] = sign

	xmlBody := mapToXML(params)

	resp, err := http.Post("https://api.mch.weixin.qq.com/pay/unifiedorder", "application/xml", strings.NewReader(xmlBody))
	if err != nil {
		return "", fmt.Errorf("请求微信统一下单接口失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取微信响应失败: %v", err)
	}

	respMap, parseErr := parseWechatXMLParams(body)
	if parseErr != nil {
		return "", fmt.Errorf("解析微信响应失败: %v", parseErr)
	}
	if respMap["return_code"] != "SUCCESS" {
		return "", fmt.Errorf("微信下单失败: %s", respMap["return_msg"])
	}
	if respMap["result_code"] != "SUCCESS" {
		return "", fmt.Errorf("微信下单失败: %s (%s)", respMap["err_code_des"], respMap["err_code"])
	}

	codeURL := respMap["code_url"]
	if codeURL == "" {
		return "", fmt.Errorf("微信返回的 code_url 为空")
	}
	return codeURL, nil
}

func generateNonceStr(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

func mapToXML(params map[string]string) string {
	var builder strings.Builder
	builder.WriteString("<xml>")
	for k, v := range params {
		builder.WriteString(fmt.Sprintf("<%s><![CDATA[%s]]></%s>", k, v, k))
	}
	builder.WriteString("</xml>")
	return builder.String()
}

type WechatQueryResult struct {
	ReturnCode     string
	ReturnMsg      string
	ResultCode     string
	ErrCode        string
	TradeState     string
	TradeStateDesc string
	TransactionID  string
	OutTradeNo     string
	TotalFee       string
	Raw            map[string]string
}

func (r *WechatQueryResult) IsPaid() bool {
	return r != nil &&
		r.ReturnCode == "SUCCESS" &&
		r.ResultCode == "SUCCESS" &&
		r.TradeState == "SUCCESS"
}

func (s *WechatService) QueryOrder(orderNo string) (*WechatQueryResult, error) {
	if orderNo == "" {
		return nil, fmt.Errorf("订单号不能为空")
	}
	if s.AppID == "" || s.MchID == "" || s.APIKey == "" {
		return nil, fmt.Errorf("微信支付AppID、商户号或API密钥未配置")
	}
	if s.QueryURL == "" {
		return nil, fmt.Errorf("微信支付查单地址未配置")
	}

	params := map[string]string{
		"appid":        s.AppID,
		"mch_id":       s.MchID,
		"out_trade_no": orderNo,
		"nonce_str":    generateNonceStr(32),
	}
	params["sign"] = s.Sign(params)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(s.QueryURL, "text/xml; charset=utf-8", strings.NewReader(mapToXML(params)))
	if err != nil {
		return nil, fmt.Errorf("微信支付查单请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("微信支付查单响应读取失败: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		bodyStr := strings.TrimSpace(string(body))
		if len(bodyStr) > 300 {
			bodyStr = bodyStr[:300] + "..."
		}
		return nil, fmt.Errorf("微信支付查单HTTP状态异常: %d, 响应: %s", resp.StatusCode, bodyStr)
	}

	xmlParams, err := parseWechatXMLParams(body)
	if err != nil {
		return nil, fmt.Errorf("微信支付查单响应解析失败: %v", err)
	}
	if sign := xmlParams["sign"]; sign != "" {
		checkParams := make(map[string]string, len(xmlParams))
		for k, v := range xmlParams {
			if k != "sign" {
				checkParams[k] = v
			}
		}
		if !strings.EqualFold(sign, s.Sign(checkParams)) {
			return nil, fmt.Errorf("微信支付查单响应签名验证失败")
		}
	}

	return &WechatQueryResult{
		ReturnCode:     xmlParams["return_code"],
		ReturnMsg:      xmlParams["return_msg"],
		ResultCode:     xmlParams["result_code"],
		ErrCode:        xmlParams["err_code"],
		TradeState:     xmlParams["trade_state"],
		TradeStateDesc: xmlParams["trade_state_desc"],
		TransactionID:  xmlParams["transaction_id"],
		OutTradeNo:     xmlParams["out_trade_no"],
		TotalFee:       xmlParams["total_fee"],
		Raw:            xmlParams,
	}, nil
}

func parseWechatXMLParams(body []byte) (map[string]string, error) {
	params := make(map[string]string)
	decoder := xml.NewDecoder(strings.NewReader(string(body)))
	var currentKey string
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := token.(type) {
		case xml.StartElement:
			currentKey = t.Name.Local
		case xml.CharData:
			if currentKey != "" && currentKey != "xml" {
				value := strings.TrimSpace(string(t))
				if value != "" {
					params[currentKey] = value
				}
			}
		case xml.EndElement:
			if currentKey == t.Name.Local {
				currentKey = ""
			}
		}
	}
	return params, nil
}

func (s *WechatService) VerifyNotify(params map[string]string) bool {
	if s.APIKey == "" {
		return false
	}

	sign, ok := params["sign"]
	if !ok || sign == "" {
		return false
	}

	delete(params, "sign")

	calculatedSign := s.Sign(params)

	return strings.EqualFold(sign, calculatedSign)
}

func (s *WechatService) Sign(params map[string]string) string {
	var keys []string
	for k := range params {
		if k != "sign" && params[k] != "" {
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
	signStr.WriteString("&key=")
	signStr.WriteString(s.APIKey)

	// #nosec G401 - MD5 is required by WeChat Pay API specification
	hash := md5.Sum([]byte(signStr.String())) // #nosec G401
	return strings.ToUpper(fmt.Sprintf("%x", hash))
}

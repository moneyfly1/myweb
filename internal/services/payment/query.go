package payment

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type EpayQueryResult struct {
	Provider   string
	Code       string
	Msg        string
	TradeNo    string
	OutTradeNo string
	Status     string
	Amount     string
	Raw        map[string]string
}

func (r *EpayQueryResult) IsPaid() bool {
	if r == nil {
		return false
	}
	status := strings.ToUpper(strings.TrimSpace(r.Status))
	switch status {
	case "1", "2", "SUCCESS", "TRADE_SUCCESS", "TRADE_FINISHED", "PAID", "COMPLETE", "COMPLETED", "OK", "200":
		return true
	default:
		return false
	}
}

func buildEpayOrderQueryURL(apiURL string) string {
	endpoint := strings.TrimSpace(apiURL)
	if endpoint == "" {
		return ""
	}

	lower := strings.ToLower(endpoint)
	for _, filename := range []string{"mapi.php", "submit.php"} {
		if idx := strings.Index(lower, filename); idx >= 0 {
			endpoint = endpoint[:idx] + "api.php" + endpoint[idx+len(filename):]
			break
		}
	}

	if !strings.Contains(strings.ToLower(endpoint), "api.php") {
		endpoint = strings.TrimRight(endpoint, "/") + "/api.php"
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	query := parsed.Query()
	query.Set("act", "order")
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func queryEpayOrder(provider, queryURL, pid, key, orderNo string) (*EpayQueryResult, error) {
	if orderNo == "" {
		return nil, fmt.Errorf("订单号不能为空")
	}
	if queryURL == "" {
		return nil, fmt.Errorf("%s查单地址未配置", provider)
	}
	if pid == "" || key == "" {
		return nil, fmt.Errorf("%s商户ID或密钥未配置", provider)
	}

	parsed, err := url.Parse(queryURL)
	if err != nil {
		return nil, fmt.Errorf("%s查单地址无效: %v", provider, err)
	}
	query := parsed.Query()
	if query.Get("act") == "" {
		query.Set("act", "order")
	}
	query.Set("pid", pid)
	query.Set("key", key)
	query.Set("out_trade_no", orderNo)
	parsed.RawQuery = query.Encode()

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(parsed.String())
	if err != nil {
		return nil, fmt.Errorf("%s查单请求失败: %v", provider, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s查单响应读取失败: %v", provider, err)
	}
	bodyStr := strings.TrimSpace(string(body))
	if resp.StatusCode != http.StatusOK {
		if len(bodyStr) > 300 {
			bodyStr = bodyStr[:300] + "..."
		}
		return nil, fmt.Errorf("%s查单HTTP状态异常: %d, 响应: %s", provider, resp.StatusCode, bodyStr)
	}
	if bodyStr == "" {
		return nil, fmt.Errorf("%s查单响应为空", provider)
	}
	if strings.HasPrefix(strings.ToLower(bodyStr), "<!doctype") || strings.HasPrefix(strings.ToLower(bodyStr), "<html") {
		return nil, fmt.Errorf("%s查单返回HTML页面，请检查query_url/api_url配置", provider)
	}

	raw, err := parseEpayQueryResponse(bodyStr)
	if err != nil {
		return nil, fmt.Errorf("%s查单响应解析失败: %v", provider, err)
	}
	return normalizeEpayQueryResult(provider, raw), nil
}

func parseEpayQueryResponse(body string) (map[string]string, error) {
	var raw map[string]interface{}
	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err == nil {
		result := make(map[string]string, len(raw))
		for k, v := range raw {
			result[k] = stringifyQueryValue(v)
		}
		return result, nil
	}

	values, err := url.ParseQuery(body)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(values))
	for k, v := range values {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("无法识别响应格式")
	}
	return result, nil
}

func stringifyQueryValue(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(val)
	case json.Number:
		return val.String()
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", val))
	}
}

func normalizeEpayQueryResult(provider string, raw map[string]string) *EpayQueryResult {
	return &EpayQueryResult{
		Provider:   provider,
		Code:       firstQueryValue(raw, "code", "ret", "result_code"),
		Msg:        firstQueryValue(raw, "msg", "message", "return_msg"),
		TradeNo:    firstQueryValue(raw, "trade_no", "pay_no", "transaction_id", "platform_trade_no"),
		OutTradeNo: firstQueryValue(raw, "out_trade_no", "order_no", "order_id", "out_trade_id"),
		Status:     firstQueryValue(raw, "trade_status", "status", "state", "pay_status"),
		Amount:     firstQueryValue(raw, "money", "amount", "total_amount", "price"),
		Raw:        raw,
	}
}

func firstQueryValue(raw map[string]string, keys ...string) string {
	for _, key := range keys {
		if val := strings.TrimSpace(raw[key]); val != "" {
			return val
		}
	}
	return ""
}

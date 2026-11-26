package smservice

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"cboard-go/internal/models"
)

const (
	configCategory = "sms_service"
	keyEnabled     = "sms_service_enabled"
	keyBaseURL     = "sms_service_base_url"
	keyAPIKey      = "sms_service_api_key"
	keyMultiplier  = "sms_service_multiplier"
	keyExchange    = "sms_service_exchange_rate"
	keyMinPrice    = "sms_service_min_price"
)

// Config 验证码服务的后台配置（存 system_configs，category=sms_service）。
type Config struct {
	Enabled      bool
	BaseURL      string
	APIKey       string
	Multiplier   float64
	ExchangeRate float64
	MinPrice     float64
}

func configValue(db *gorm.DB, key, fallback string) string {
	var cfg models.SystemConfig
	if err := db.Where("key = ? AND category = ?", key, configCategory).First(&cfg).Error; err != nil {
		return fallback
	}
	if strings.TrimSpace(cfg.Value) == "" {
		return fallback
	}
	return strings.TrimSpace(cfg.Value)
}

// LoadConfig 从数据库读取验证码服务配置。
func LoadConfig(db *gorm.DB) *Config {
	c := &Config{
		Enabled:      configValue(db, keyEnabled, "false") == "true",
		BaseURL:      configValue(db, keyBaseURL, "https://hero-sms.com"),
		APIKey:       configValue(db, keyAPIKey, ""),
		Multiplier:   2.0,
		ExchangeRate: 7.2,
		MinPrice:     1.0,
	}
	if v, err := strconv.ParseFloat(configValue(db, keyMultiplier, "2.0"), 64); err == nil && v > 0 {
		c.Multiplier = v
	}
	if v, err := strconv.ParseFloat(configValue(db, keyExchange, "7.2"), 64); err == nil && v > 0 {
		c.ExchangeRate = v
	}
	if v, err := strconv.ParseFloat(configValue(db, keyMinPrice, "1.0"), 64); err == nil && v >= 0 {
		c.MinPrice = v
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	return c
}

// PriceCNY 官方美元价 → 人民币售价：usd × 倍率 × 汇率，最低按 MinPrice，保留两位小数。
func (c *Config) PriceCNY(usd float64) float64 {
	v := usd * c.Multiplier * c.ExchangeRate
	if v < c.MinPrice {
		v = c.MinPrice
	}
	return math.Round(v*100) / 100
}

// Client HeroSMS（SMS-Activate 兼容）客户端。
type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

func NewClient(cfg *Config) *Client {
	return &Client{
		BaseURL: strings.TrimRight(cfg.BaseURL, "/"),
		APIKey:  cfg.APIKey,
		HTTP:    &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *Client) call(action string, params map[string]string) (string, error) {
	q := url.Values{}
	q.Set("api_key", c.APIKey)
	q.Set("action", action)
	for k, v := range params {
		q.Set(k, v)
	}
	reqURL := c.BaseURL + "/stubs/handler_api.php?" + q.Encode()
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(string(body))
	if text == "" {
		return "", fmt.Errorf("接口返回为空（HTTP %d）", resp.StatusCode)
	}
	if strings.Contains(text, "BAD_KEY") || strings.Contains(text, "BAD_API_KEY") || strings.Contains(text, "Invalid API key") {
		return "", fmt.Errorf("API 密钥无效（BAD_KEY）")
	}
	if strings.Contains(text, "UNAUTHORIZED") || strings.Contains(text, "Unauthorized") {
		return "", fmt.Errorf("API 密钥无效（Unauthorized）")
	}
	return text, nil
}

// GetBalance 账户余额（美元）。
func (c *Client) GetBalance() (float64, error) {
	text, err := c.call("getBalance", nil)
	if err != nil {
		return 0, err
	}
	if !strings.HasPrefix(text, "ACCESS_BALANCE:") {
		return 0, fmt.Errorf("余额查询失败：%s", text)
	}
	return strconv.ParseFloat(strings.TrimPrefix(text, "ACCESS_BALANCE:"), 64)
}

// GetCountries 国家列表：countryID → 名称（返回原始 map）。
func (c *Client) GetCountries() (map[string]map[string]interface{}, error) {
	text, err := c.call("getCountries", nil)
	if err != nil {
		return nil, err
	}
	var out map[string]map[string]interface{}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("国家列表解析失败：%w", err)
	}
	return out, nil
}

// GetServicesList 服务列表。
func (c *Client) GetServicesList() ([]map[string]interface{}, error) {
	text, err := c.call("getServicesList", nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		Status   string                   `json:"status"`
		Services []map[string]interface{} `json:"services"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("服务列表解析失败：%w", err)
	}
	return out.Services, nil
}

// GetPrices 价格表：countryID → service → {cost, count}。
func (c *Client) GetPrices() (map[string]map[string]struct {
	Cost  float64 `json:"cost"`
	Count int     `json:"count"`
}, error) {
	text, err := c.call("getPrices", nil)
	if err != nil {
		return nil, err
	}
	out := map[string]map[string]struct {
		Cost  float64 `json:"cost"`
		Count int     `json:"count"`
	}{}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("价格表解析失败：%w", err)
	}
	return out, nil
}

// GetNumber 购买号码：返回 activationID、手机号。
func (c *Client) GetNumber(service, country string) (string, string, error) {
	text, err := c.call("getNumber", map[string]string{"service": service, "country": country})
	if err != nil {
		return "", "", err
	}
	if !strings.HasPrefix(text, "ACCESS_NUMBER:") {
		return "", "", fmt.Errorf("购买失败：%s", text)
	}
	parts := strings.Split(strings.TrimPrefix(text, "ACCESS_NUMBER:"), ":")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("购买返回格式异常：%s", text)
	}
	return parts[0], parts[1], nil
}

// GetStatus 查询验证码状态：返回 (状态, 验证码)。
func (c *Client) GetStatus(activationID string) (string, string, error) {
	text, err := c.call("getStatus", map[string]string{"id": activationID})
	if err != nil {
		return "", "", err
	}
	if text == "STATUS_WAIT_CODE" {
		return "waiting", "", nil
	}
	if strings.HasPrefix(text, "STATUS_OK:") {
		return "ok", strings.TrimPrefix(text, "STATUS_OK:"), nil
	}
	if text == "STATUS_WAIT_RETRY" {
		return "waiting", "", nil
	}
	return "unknown", "", nil
}

// Finish 完成激活。
func (c *Client) Finish(activationID string) error {
	_, err := c.call("finishActivation", map[string]string{"id": activationID})
	return err
}

// Cancel 取消激活（平台会退款给本站账户）。
func (c *Client) Cancel(activationID string) error {
	_, err := c.call("cancelActivation", map[string]string{"id": activationID})
	return err
}

// ProlongOption 续期选项（时长单位：小时）。
type ProlongOption struct {
	DurationHours int     `json:"duration_hours"`
	PriceUSD      float64 `json:"price_usd"`
}

// GetProlongOptions 获取某激活可用的续租选项（平台返回格式不稳定，做多形态解析）。
func (c *Client) GetProlongOptions(activationID string) ([]ProlongOption, error) {
	text, err := c.call("prolongOptions", map[string]string{"id": activationID})
	if err != nil {
		return nil, err
	}
	// 一次性验证码号码不支持 prolong：平台返回 ACTION_NOT_AVAILABLE / BAD_STATUS
	// （JSON 错误体，不会变成 Go error），此时回退 reactivateOptions（恢复费用）。
	if strings.Contains(text, "ACTION_NOT_AVAILABLE") || strings.Contains(text, "not available") || strings.Contains(text, "BAD_STATUS") {
		if txt2, e2 := c.call("reactivateOptions", map[string]string{"id": activationID}); e2 == nil {
			if !strings.Contains(txt2, "BAD_STATUS") && !strings.Contains(txt2, "NOT_AVAILABLE") && !strings.Contains(txt2, "NOT_FOUND") {
				return parseProlongOptions(txt2), nil
			}
		}
		return nil, fmt.Errorf("当前号码状态暂不支持续期（需号码完成验证或过期后才能恢复）")
	}
	return parseProlongOptions(text), nil
}

// parseProlongOptions 解析平台返回的续期选项（多形态兼容）。
func parseProlongOptions(text string) []ProlongOption {
	// 平台可能返回:
	//   {"<duration>": <price>, ...}
	//   [{"duration":24,"price":0.1}, ...]
	//   {"options":[...]}
	var asMap map[string]float64
	if err := json.Unmarshal([]byte(text), &asMap); err == nil && len(asMap) > 0 {
		out := make([]ProlongOption, 0, len(asMap))
		for k, v := range asMap {
			d, e := strconv.Atoi(k)
			if e != nil {
				continue
			}
			out = append(out, ProlongOption{DurationHours: d, PriceUSD: v})
		}
		if len(out) > 0 {
			return out
		}
	}
	var asObjs []map[string]interface{}
	if err := json.Unmarshal([]byte(text), &asObjs); err == nil {
		out := make([]ProlongOption, 0, len(asObjs))
		for _, o := range asObjs {
			d := 0
			if v, ok := o["duration"].(float64); ok {
				d = int(v)
			} else if v, ok := o["hours"].(float64); ok {
				d = int(v)
			}
			price := 0.0
			if v, ok := o["price"].(float64); ok {
				price = v
			} else if v, ok := o["cost"].(float64); ok {
				price = v
			}
			if d > 0 {
				out = append(out, ProlongOption{DurationHours: d, PriceUSD: price})
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	// 兜底：固定时长（价格由平台 prolong 时扣，页面显示"以实际扣费为准"）
	defaults := []int{24, 72, 168, 336, 720, 1440, 2160, 4320}
	out := make([]ProlongOption, 0, len(defaults))
	for _, d := range defaults {
		out = append(out, ProlongOption{DurationHours: d, PriceUSD: 0})
	}
	return out
}

// Prolong 延长号码租期（durationHours 必须是平台支持的时长）。
// 一次性验证码不支持 prolong 时，回退到 reactivate（恢复使用）。
func (c *Client) Prolong(activationID string, durationHours int) (string, error) {
	text, err := c.call("prolong", map[string]string{"id": activationID, "option": strconv.Itoa(durationHours)})
	if err == nil {
		return text, nil
	}
	if strings.Contains(err.Error(), "ACTION_NOT_AVAILABLE") || strings.Contains(err.Error(), "not available") {
		return c.call("reactivate", map[string]string{"id": activationID})
	}
	return "", err
}

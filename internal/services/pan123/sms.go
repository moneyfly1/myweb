package pan123

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// 短信验证码登录（海外/风控场景）：
// 账号密码在境外 IP 登录会返回 7012（当前账号存在境外登录风险，请使用短信验证码或微信登录），
// 需要先发送短信验证码，再凭验证码完成登录。这些接口在 user.123pan.cn（登录门户），无需 API 签名。
const (
	smsPortalBase   = "https://user.123pan.cn/api"
	loginAPIBase    = "https://login.123pan.com/api"
)

// RiskError 风控登录错误（code 7012）
type RiskError struct {
	HashCode string
	Message  string
}

func (e *RiskError) Error() string {
	return e.Message
}

// IsRiskBlocked 判断登录响应是否为境外登录风控（7012）
func IsRiskBlocked(code int) bool {
	return code == 7012
}

// SendSmsCode 发送登录短信验证码。
// 需要先从账号密码登录的 7012 响应中拿到 hashCode。
// 返回 needsCaptcha=true 表示被阿里云无痕验证（滑块）拦截、短信未发出，需人工在浏览器完成验证。
func (c *Client) SendSmsCode(passport, hashCode string) (needsCaptcha bool, err error) {
	body := map[string]interface{}{
		"passport": strings.TrimSpace(passport),
		"hashCode": hashCode,
	}
	payload, err := postJSON(smsPortalBase+"/user/get_vcode", body, nil)
	if err != nil {
		return false, err
	}
	var out struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			SerialNo string `json:"serial_no"`
			Tips     string `json:"Tips"`
			Traceless struct {
				Code int `json:"code"`
			} `json:"traceless"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return false, fmt.Errorf("发送验证码响应解析失败: %w", err)
	}
	if out.Code != 0 {
		return false, fmt.Errorf("发送验证码失败: %s", firstNonEmpty(out.Message, fmt.Sprintf("code=%d", out.Code)))
	}
	// serial_no 为空 或 traceless.code != 0 → 未真正发送（被滑块验证拦截）
	if strings.TrimSpace(out.Data.SerialNo) == "" || out.Data.Traceless.Code != 0 {
		return true, fmt.Errorf("需要滑块验证后发送（traceless code=%d）", out.Data.Traceless.Code)
	}
	if strings.TrimSpace(out.Data.Tips) != "" {
		return false, fmt.Errorf("发送验证码提示: %s", out.Data.Tips)
	}
	return false, nil
}

// LoginWithSmsCode 使用短信验证码完成登录，返回登录 token（绑定当前请求 IP）。
func (c *Client) LoginWithSmsCode(passport, vcode, hashCode string) (string, error) {
	body := map[string]interface{}{
		"passport": strings.TrimSpace(passport),
		"type":     3, // 3 = 短信验证码登录
		"vcode":    strings.TrimSpace(vcode),
		"hashCode": hashCode,
	}
	payload, err := postJSON(smsPortalBase+"/user/sign_in", body, nil)
	if err != nil {
		return "", err
	}
	var out struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return "", fmt.Errorf("验证码登录响应解析失败: %w", err)
	}
	if out.Code != 0 || out.Data.Token == "" {
		return "", fmt.Errorf("验证码登录失败: %s", firstNonEmpty(out.Message, fmt.Sprintf("code=%d", out.Code)))
	}
	c.Token = out.Data.Token
	c.Cookies = "jwt=" + out.Data.Token
	return out.Data.Token, nil
}

// GetLoginHashCode 发起账号密码登录，若触发境外风控（7012）则返回 hashCode 供短信验证。
func (c *Client) GetLoginHashCode(passport, password string) (string, error) {
	body := map[string]interface{}{}
	if strings.Contains(passport, "@") {
		body = map[string]interface{}{"mail": passport, "password": password, "type": 2}
	} else {
		body = map[string]interface{}{"passport": passport, "password": password, "remember": true}
	}
	payload, err := postJSON(loginAPIBase+"/user/sign_in", body, map[string]string{
		"origin":      "https://yun.123pan.com",
		"platform":    "web",
		"app-version": "3",
		"user-agent":  defaultUserAgent,
	})
	if err != nil {
		return "", err
	}
	var out struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			HashCode string `json:"hashCode"`
			Token    string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return "", fmt.Errorf("登录响应解析失败: %w", err)
	}
	if out.Code == 200 && out.Data.Token != "" {
		return "", nil // 正常登录成功（无风控）
	}
	if IsRiskBlocked(out.Code) {
		return out.Data.HashCode, &RiskError{HashCode: out.Data.HashCode, Message: out.Message}
	}
	return "", fmt.Errorf("登录失败: %s", firstNonEmpty(out.Message, fmt.Sprintf("code=%d", out.Code)))
}

// postJSON 简单 POST JSON（无签名），用于登录门户接口
func postJSON(rawURL string, body interface{}, headers map[string]string) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, rawURL, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("user-agent", defaultUserAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	return payload, nil
}

var _ = errors.New

// ParseTokenExpiry 解析 JWT token 的 exp 声明（有效期），解析失败返回零值
func ParseTokenExpiry(token string) time.Time {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) < 2 {
		return time.Time{}
	}
	payload := parts[1]
	if b, err := base64.RawURLEncoding.DecodeString(payload); err == nil {
		var claims struct {
			Exp int64 `json:"exp"`
		}
		if json.Unmarshal(b, &claims) == nil && claims.Exp > 0 {
			return time.Unix(claims.Exp, 0)
		}
	}
	return time.Time{}
}

// TokenDaysLeft token 剩余有效天数（已过期返回负数）
func TokenDaysLeft(token string) int {
	exp := ParseTokenExpiry(token)
	if exp.IsZero() {
		return 0
	}
	return int(time.Until(exp).Hours() / 24)
}

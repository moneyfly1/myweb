package pan123

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
// traceless 为阿里云无痕验证（滑块）结果字符串（面板内滑块完成后由 SDK 给出）；
// 为空表示首次尝试（可能被拦，返回 needsCaptcha=true 需先完成滑块）。
// 返回 needsCaptcha=true 表示被滑块验证拦截、短信未发出。
// SendSmsCode 发送登录短信验证码。
// 需要先从账号密码登录的 7012 响应中拿到 hashCode。
// traceless 为阿里云无痕验证（滑块）结果字符串（面板内滑块完成后由 SDK 给出）；
// 为空表示首次尝试（可能被拦，返回 needsCaptcha=true 需先完成滑块）。
// 返回 timeStamp（get_vcode 的 Timestamp，登录时需回传）与 needsCaptcha。
func (c *Client) SendSmsCode(passport, hashCode, traceless string) (timeStamp string, needsCaptcha bool, err error) {
	body := map[string]interface{}{
		"passport":   strings.TrimSpace(passport),
		"hashCode":   hashCode,
		"aliVersion": 2,
	}
	if strings.TrimSpace(traceless) != "" {
		body["traceless"] = strings.TrimSpace(traceless)
	}
	payload, err := postJSON(smsPortalBase+"/user/get_vcode", body, nil)
	if err != nil {
		return "", false, err
	}
	var out struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			SerialNo  string `json:"serial_no"`
			Timestamp int64  `json:"Timestamp"`
			Tips      string `json:"Tips"`
			Traceless struct {
				Code int `json:"code"`
			} `json:"traceless"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return "", false, fmt.Errorf("发送验证码响应解析失败: %w", err)
	}
	if out.Code != 0 {
		// 无痕验证失败 / 需要滑块 → 让前端展示滑块
		if out.Code == 5019 || strings.Contains(out.Message, "无痕") || strings.Contains(out.Message, "验证") {
			return "", true, fmt.Errorf("需要滑块验证: %s", firstNonEmpty(out.Message, fmt.Sprintf("code=%d", out.Code)))
		}
		return "", false, fmt.Errorf("发送验证码失败: %s", firstNonEmpty(out.Message, fmt.Sprintf("code=%d", out.Code)))
	}
	// serial_no 为空 或 traceless.code != 0 → 未真正发送（被滑块验证拦截）
	if strings.TrimSpace(out.Data.SerialNo) == "" || out.Data.Traceless.Code != 0 {
		return "", true, fmt.Errorf("需要滑块验证后发送（traceless code=%d）", out.Data.Traceless.Code)
	}
	if strings.TrimSpace(out.Data.Tips) != "" {
		return "", false, fmt.Errorf("发送验证码提示: %s", out.Data.Tips)
	}
	return fmt.Sprintf("%d", out.Data.Timestamp), false, nil
}

// LoginWithSmsCode 使用短信验证码完成登录，返回登录 token（绑定当前请求 IP）。
// 参数与登录中心网页端一致：v_code（验证码）、time_stamp（get_vcode 返回的 Timestamp）、hashCode。
func (c *Client) LoginWithSmsCode(passport, vcode, timeStamp, hashCode string) (string, error) {
	body := map[string]interface{}{
		"passport":  strings.TrimSpace(passport),
		"type":      3, // 3 = 短信验证码登录
		"v_code":    strings.TrimSpace(vcode),
		"remember":  true,
		"hashCode":  hashCode,
	}
	if strings.TrimSpace(timeStamp) != "" {
		body["time_stamp"] = strings.TrimSpace(timeStamp)
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

// GenerateQrCode 生成登录二维码（123pan 手机 App / 微信扫码），返回 QR 内容与 uniID
func (c *Client) GenerateQrCode() (qrURL, uniID string, err error) {
	payload, err := getJSON(smsPortalBase+"/user/qr-code/generate", nil)
	if err != nil {
		return "", "", err
	}
	var out struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			URL   string `json:"url"`
			UniID string `json:"uniID"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return "", "", fmt.Errorf("二维码响应解析失败: %w", err)
	}
	if out.Code != 0 || out.Data.UniID == "" {
		return "", "", fmt.Errorf("生成二维码失败: %s", firstNonEmpty(out.Message, fmt.Sprintf("code=%d", out.Code)))
	}
	return out.Data.URL, out.Data.UniID, nil
}

// BuildQrValue 构造二维码编码内容（与登录中心一致：env/uniID/source/type 参数缺一不可，
// 否则手机 App / 微信无法识别登录意图）
func BuildQrValue(qrURL, uniID string) string {
	return fmt.Sprintf("%s?env=production&uniID=%s&source=123pan&type=login",
		strings.TrimRight(qrURL, "/"), url.QueryEscape(uniID))
}

// WechatLoginByQr 微信扫码确认后：wx_code 换登录凭证
func (c *Client) WechatLoginByQr(uniID string) (string, error) {
	payload, err := postJSON(smsPortalBase+"/user/qr-code/wx_code", map[string]interface{}{"uniID": uniID}, nil)
	if err != nil {
		return "", err
	}
	var wx struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			WxCode string `json:"wxCode"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &wx); err != nil {
		return "", fmt.Errorf("wx_code 响应解析失败: %w", err)
	}
	if wx.Code != 0 || wx.Data.WxCode == "" {
		return "", fmt.Errorf("获取微信凭证失败: %s", firstNonEmpty(wx.Message, fmt.Sprintf("code=%d", wx.Code)))
	}
	// 微信扫码登录（type=4），与登录中心一致
	loginPayload, err := postJSON(smsPortalBase+"/user/sign_in", map[string]interface{}{
		"from":         "web",
		"wechat_code":  wx.Data.WxCode,
		"type":         4,
		"remember":     true,
	}, nil)
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
	if err := json.Unmarshal(loginPayload, &out); err != nil {
		return "", fmt.Errorf("微信登录响应解析失败: %w", err)
	}
	if out.Code != 200 || out.Data.Token == "" {
		return "", fmt.Errorf("微信登录失败: %s", firstNonEmpty(out.Message, fmt.Sprintf("code=%d", out.Code)))
	}
	c.Token = out.Data.Token
	c.Cookies = "jwt=" + out.Data.Token
	return out.Data.Token, nil
}

// GetQrCodeStatus 轮询扫码登录状态；loginStatus: 0未扫 1已扫 2拒绝 3已确认(含 token) 4过期
func (c *Client) GetQrCodeStatus(uniID string) (loginStatus int, token string, err error) {
	rawURL := smsPortalBase + "/user/qr-code/result?uniID=" + url.QueryEscape(uniID) + "&remember=true"
	payload, err := getJSON(rawURL, nil)
	if err != nil {
		return 0, "", err
	}
	var out struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			LoginStatus  int    `json:"loginStatus"`
			ScanPlatform int    `json:"scanPlatform"`
			Token        string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return 0, "", fmt.Errorf("扫码状态解析失败: %w", err)
	}
	if out.Code != 0 {
		return 0, "", fmt.Errorf("查询扫码状态失败: %s", firstNonEmpty(out.Message, fmt.Sprintf("code=%d", out.Code)))
	}
	return out.Data.LoginStatus, out.Data.Token, nil
}

// GetQrCodeStatusV2 轮询扫码登录状态（含扫码平台）；loginStatus: 0未扫 1已扫 3已确认 4过期；scanPlatform: 4微信 7App
func (c *Client) GetQrCodeStatusV2(uniID string) (loginStatus, scanPlatform int, token string, err error) {
	rawURL := smsPortalBase + "/user/qr-code/result?uniID=" + url.QueryEscape(uniID) + "&remember=true"
	payload, err := getJSON(rawURL, nil)
	if err != nil {
		return 0, 0, "", err
	}
	var out struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			LoginStatus  int    `json:"loginStatus"`
			ScanPlatform int    `json:"scanPlatform"`
			Token        string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return 0, 0, "", fmt.Errorf("扫码状态解析失败: %w", err)
	}
	if out.Code != 0 {
		return 0, 0, "", fmt.Errorf("查询扫码状态失败: %s", firstNonEmpty(out.Message, fmt.Sprintf("code=%d", out.Code)))
	}
	return out.Data.LoginStatus, out.Data.ScanPlatform, out.Data.Token, nil
}

// getJSON 简单 GET JSON（无签名），用于登录门户接口
func getJSON(rawURL string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
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

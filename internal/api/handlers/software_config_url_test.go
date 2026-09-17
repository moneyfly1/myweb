package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestIsAllowedDownloadURL 覆盖下载地址协议白名单：
// 只允许 http/https/pan，其余（javascript:、data:、file: 等）必须被拒绝。
func TestIsAllowedDownloadURL(t *testing.T) {
	allowed := []string{
		"https://example.com/moneyfly.apk",
		"http://example.com/moneyfly.exe",
		"pan://moneyfly_windows",
		"HTTPS://EXAMPLE.COM/a.dmg",
		"Pan://key",
	}
	for _, u := range allowed {
		if !isAllowedDownloadURL(u) {
			t.Errorf("应放行: %s", u)
		}
	}

	rejected := []string{
		"javascript:alert(1)",
		"JavaScript:alert(1)",
		"data:text/html;base64,PHNjcmlwdD4=",
		"file:///etc/passwd",
		"vbscript:msgbox(1)",
		"ftp://example.com/a.apk",
		"example.com/a.apk",
		"//example.com/a.apk",
	}
	for _, u := range rejected {
		if isAllowedDownloadURL(u) {
			t.Errorf("应拒绝危险/非法地址: %s", u)
		}
	}
}

// newTestContext 构造带指定 JSON body 的 gin 测试上下文
func newTestContext(t *testing.T, body string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/software-config", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c, w
}

// TestValidateSoftwareConfigURLs 校验通过时应还原并规范化请求体（去空格）
func TestValidateSoftwareConfigURLs(t *testing.T) {
	body := `{"moneyfly_windows_url":"  https://example.com/mf.exe  ","moneyfly_enabled":true,"moneyfly_version":"2.1.2"}`
	c, w := newTestContext(t, body)

	if !validateSoftwareConfigURLs(c) {
		t.Fatalf("合法配置应通过校验，状态码=%d body=%s", w.Code, w.Body.String())
	}

	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		t.Fatalf("读取还原后的请求体失败: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("还原后的请求体不是合法 JSON: %v (%s)", err, string(raw))
	}
	if got := payload["moneyfly_windows_url"]; got != "https://example.com/mf.exe" {
		t.Errorf("URL 应被去除首尾空格，实际 %v", got)
	}
	// 非 _url 字段必须原样保留，否则会丢失配置
	if _, ok := payload["moneyfly_enabled"]; !ok {
		t.Error("非 URL 字段不应被丢弃: moneyfly_enabled")
	}
	if payload["moneyfly_version"] != "2.1.2" {
		t.Errorf("非 URL 字段应保留原值，实际 %v", payload["moneyfly_version"])
	}
}

// TestValidateSoftwareConfigURLs_RejectsDangerousScheme 危险协议必须被拒绝且不进入保存流程
func TestValidateSoftwareConfigURLs_RejectsDangerousScheme(t *testing.T) {
	cases := []string{
		`{"moneyfly_android_url":"javascript:alert(1)"}`,
		`{"moneyfly_windows_url":"data:text/html,<script>alert(1)</script>"}`,
		`{"some_client_url":"file:///etc/passwd"}`,
	}
	for _, body := range cases {
		c, w := newTestContext(t, body)
		if validateSoftwareConfigURLs(c) {
			t.Errorf("危险地址应被拒绝: %s", body)
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("应返回 400，实际 %d", w.Code)
		}
	}
}

// TestValidateSoftwareConfigURLs_AllowsEmptyAndNonURLKeys 允许清空 URL（表示暂不提供），
// 且不干扰非 URL 字段与空 body 之外的正常保存
func TestValidateSoftwareConfigURLs_AllowsEmptyAndNonURLKeys(t *testing.T) {
	body := `{"moneyfly_android_url":"","moneyfly_note":"官方自研客户端","pan_sync_enabled":true}`
	c, w := newTestContext(t, body)
	if !validateSoftwareConfigURLs(c) {
		t.Fatalf("清空 URL 应被允许，状态码=%d body=%s", w.Code, w.Body.String())
	}
}

// TestValidateSoftwareConfigURLs_KeepsExistingPanValues 兼容线上已有的 pan:// 配置值
func TestValidateSoftwareConfigURLs_KeepsExistingPanValues(t *testing.T) {
	body := `{"v2rayng_url":"pan://v2rayng_url","v2rayn_url":"pan://v2rayn_url","sparkle_windows_url":"https://github.com/x/y/releases/download/v1/z.exe"}`
	c, w := newTestContext(t, body)
	if !validateSoftwareConfigURLs(c) {
		t.Fatalf("既有 pan:// 与 https 配置应通过校验，状态码=%d body=%s", w.Code, w.Body.String())
	}
}

package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestRenewDomainPoolUnavailable 环境不具备自动化条件（无 nginx/certbot 或非 root）时，
// 续期接口必须明确报「未开启一键配置」，而不是静默成功或直接 500 崩掉。
func TestRenewDomainPoolUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/domains/pool/renew", strings.NewReader(`{"force":false}`))
	c.Request.Header.Set("Content-Type", "application/json")

	RenewDomainPool(c)

	m := domainPoolManager()
	if ok, _ := m.Available(); ok {
		t.Skip("本机具备 nginx+certbot 且为 root，跳过「不可用」分支")
	}
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("状态码 = %d，期望 503（body=%s）", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "一键配置") {
		t.Errorf("错误信息应说明未开启一键配置：%s", w.Body.String())
	}
}

// TestRenewDomainPoolAllowsEmptyBody 空 body 必须按「不强制」处理（前端按钮不带参数也要能用）
func TestRenewDomainPoolAllowsEmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/domains/pool/renew", strings.NewReader(""))
	c.Request.Header.Set("Content-Type", "application/json")

	RenewDomainPool(c)

	if w.Code == http.StatusBadRequest {
		t.Fatalf("空 body 不应被判为参数错误：%s", w.Body.String())
	}
}

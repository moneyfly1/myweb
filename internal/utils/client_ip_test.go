package utils

import (
	"net"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// 真实客户端 IP 解析的信任链测试。
//
// 背景（线上缺陷）：
//  1. directClientIP 曾用 c.ClientIP()（Gin 按其自身信任链解析 XFF 后的结果）去做
//     "是否来自可信代理"的判断 —— 循环论证，使 CF-Connecting-IP / X-Real-IP 分支
//     在多数部署下不可达；
//  2. Gin 的 SetTrustedProxies 曾硬编码 127.0.0.1/::1，而 GetRealClientIP 读
//     TRUSTED_PROXIES(.env)，两条信任链不一致 → 同一请求在 server.log 与
//     audit_logs 里 IP 不同；
//  3. TRUSTED_PROXIES 曾只用 os.Getenv 读取，start.sh 启动方式不 export .env
//     → 配置静默失效。
//
// 这里锁定修复后的行为：只有真实直连对端在可信列表内，才采信转发头。

func newIPContext(remoteAddr string, headers map[string]string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = remoteAddr
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	c.Request = req
	return c
}

// TestGetRealClientIPIgnoresForwardHeadersWhenPeerNotTrusted 对端不可信时，
// 客户端伪造的转发头必须被忽略（否则可绕过登录/注册限流）
func TestGetRealClientIPIgnoresForwardHeadersWhenPeerNotTrusted(t *testing.T) {
	InitTrustedProxies("127.0.0.1,::1")
	defer InitTrustedProxies("")

	c := newIPContext("203.0.113.9:51234", map[string]string{
		"CF-Connecting-IP": "1.2.3.4",
		"X-Forwarded-For":  "5.6.7.8",
		"X-Real-IP":        "9.9.9.9",
	})
	if got := GetRealClientIP(c); got != "203.0.113.9" {
		t.Errorf("直连对端不可信时应返回真实对端 203.0.113.9，实际 %q（伪造头被采信）", got)
	}
}

// TestGetRealClientIPTrustsCloudflareHeader 直连对端是可信 Cloudflare 边缘节点时，
// 必须采信 CF-Connecting-IP（修复前该分支因用派生值判断而常不可达）
func TestGetRealClientIPTrustsCloudflareHeader(t *testing.T) {
	InitTrustedProxies("127.0.0.1,::1,172.64.0.0/13")
	defer InitTrustedProxies("")

	c := newIPContext("172.64.1.1:443", map[string]string{
		"CF-Connecting-IP": "45.124.25.136",
		"X-Forwarded-For":  "45.124.25.136, 172.64.1.1",
	})
	if got := GetRealClientIP(c); got != "45.124.25.136" {
		t.Errorf("可信 CF 边缘应采信 CF-Connecting-IP，实际 %q", got)
	}
}

// TestGetRealClientIPTrustsNginxHeaders 本机 nginx 反代场景：采信 X-Forwarded-For 最右侧公网 IP
func TestGetRealClientIPTrustsNginxHeaders(t *testing.T) {
	InitTrustedProxies("127.0.0.1,::1")
	defer InitTrustedProxies("")

	c := newIPContext("127.0.0.1:40000", map[string]string{
		"X-Forwarded-For": "45.124.25.136, 127.0.0.1",
	})
	if got := GetRealClientIP(c); got != "45.124.25.136" {
		t.Errorf("本机 nginx 场景应返回 45.124.25.136，实际 %q", got)
	}

	// X-Real-IP 兜底（无 XFF 时）
	c = newIPContext("127.0.0.1:40001", map[string]string{"X-Real-IP": "45.124.25.137"})
	if got := GetRealClientIP(c); got != "45.124.25.137" {
		t.Errorf("无 XFF 时应采信 X-Real-IP，实际 %q", got)
	}
}

// TestGetRealClientIPNoTrustedProxies 未配置可信代理时一律返回直连地址
func TestGetRealClientIPNoTrustedProxies(t *testing.T) {
	InitTrustedProxies("")
	defer InitTrustedProxies("")

	c := newIPContext("127.0.0.1:40002", map[string]string{
		"CF-Connecting-IP": "1.2.3.4",
		"X-Forwarded-For":  "5.6.7.8",
	})
	if got := GetRealClientIP(c); got != "127.0.0.1" {
		t.Errorf("未配置可信代理时应返回直连地址 127.0.0.1，实际 %q", got)
	}
}

// TestGetRealClientIPSkipsPrivateForwardedIP 转发头里全是内网地址时，
// 应跳过内网值取公网值；全为内网时退回最后一个可解析值
func TestGetRealClientIPSkipsPrivateForwardedIP(t *testing.T) {
	InitTrustedProxies("127.0.0.1,::1")
	defer InitTrustedProxies("")

	c := newIPContext("127.0.0.1:40003", map[string]string{
		"X-Forwarded-For": "192.168.1.5, 45.124.25.136, 10.0.0.1",
	})
	if got := GetRealClientIP(c); got != "45.124.25.136" {
		t.Errorf("应跳过内网地址取公网值，实际 %q", got)
	}

	c = newIPContext("127.0.0.1:40004", map[string]string{
		"X-Forwarded-For": "192.168.1.5, 10.0.0.1",
	})
	if got := GetRealClientIP(c); got != "10.0.0.1" {
		t.Errorf("全为内网时应退回最后一个可解析值，实际 %q", got)
	}
}

// TestInitTrustedProxiesParsesSingleIPAndCIDR 单 IP 与 CIDR 都要支持
func TestInitTrustedProxiesParsesSingleIPAndCIDR(t *testing.T) {
	InitTrustedProxies("127.0.0.1, 172.64.0.0/13 ,::1")
	defer InitTrustedProxies("")

	for _, ip := range []string{"127.0.0.1", "172.64.5.5", "::1"} {
		if !isTrustedProxy(parseIPForTest(ip)) {
			t.Errorf("%s 应被识别为可信代理", ip)
		}
	}
	for _, ip := range []string{"45.124.25.136", "8.8.8.8"} {
		if isTrustedProxy(parseIPForTest(ip)) {
			t.Errorf("%s 不应被识别为可信代理", ip)
		}
	}
	// 空配置不得 panic，且不信任任何地址
	InitTrustedProxies("")
	if isTrustedProxy(parseIPForTest("127.0.0.1")) {
		t.Error("空配置时不应信任任何地址")
	}
}

// parseIPForTest 内部测试辅助：解析 IP 字符串
func parseIPForTest(s string) net.IP { return net.ParseIP(s) }

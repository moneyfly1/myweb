package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setBlockBrowserAccess 建内存库并写入「浏览器访问返回空」开关
func setBlockBrowserAccess(t *testing.T, enabled bool) {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.SystemConfig{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	value := "false"
	if enabled {
		value = "true"
	}
	db.Create(&models.SystemConfig{Key: blockBrowserSubscriptionAccessKey, Value: value, Category: "subscription_access", Type: "boolean"})
	prev := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = prev
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
}

func newAccessContext(ua string, headers map[string]string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/clash/token", nil)
	if ua != "" {
		c.Request.Header.Set("User-Agent", ua)
	}
	for k, v := range headers {
		c.Request.Header.Set(k, v)
	}
	return c
}

const browserUA = "Mozilla/5.0 (Windows NT 10.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.0.25 Safari/537.36 GooBrowser/100.0.0.25"

// TestSelfClientNeverBlockedEvenWithBrowserUA 回归：线上客户 dos2009 把 MoneyFly 的
// 「订阅自定义 UA」设成了浏览器 UA（GooBrowser/Chrome），面板按 UA 判定成浏览器后
// 返回 200 + 0 字节 → App 报「拉取订阅失败」。自研客户端带的 X-MF-* 设备头必须优先放行。
func TestSelfClientNeverBlockedEvenWithBrowserUA(t *testing.T) {
	setBlockBrowserAccess(t, true)

	c := newAccessContext(browserUA, map[string]string{"X-MF-Device-Id": "abc123", "X-MF-OS": "Windows 10"})
	if shouldBlockBrowserSubscriptionAccess(c) {
		t.Fatal("带 X-MF-* 设备头的自研客户端不应被拦截（即使 UA 像浏览器）")
	}
}

// TestPlainBrowserStillBlocked 普通浏览器（无 X-MF 头）仍要被拦，功能不能被削弱
func TestPlainBrowserStillBlocked(t *testing.T) {
	setBlockBrowserAccess(t, true)

	if !shouldBlockBrowserSubscriptionAccess(newAccessContext(browserUA, nil)) {
		t.Fatal("普通浏览器应被拦截")
	}
	chrome := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0 Safari/537.36"
	if !shouldBlockBrowserSubscriptionAccess(newAccessContext(chrome, nil)) {
		t.Fatal("Chrome 应被拦截")
	}
}

// TestClientUAsAllowed 常见客户端 UA 一律放行
func TestClientUAsAllowed(t *testing.T) {
	setBlockBrowserAccess(t, true)

	for _, ua := range []string{
		"MoneyFly/2.2.20 (Windows NT 10.0; Win64; x64)",
		"ClashforWindows/0.20.39",
		"clash-verge/v2.0.3",
		"Shadowrocket/2515 CFNetwork/3860.700.1 Darwin/25.6.0 iPhone16,1",
		"v2rayNG/1.8.5",
		"Mclash/1.0 (windows)",
		"Dart/3.9 (dart:io)", // 无浏览器特征，视为客户端
	} {
		if shouldBlockBrowserSubscriptionAccess(newAccessContext(ua, nil)) {
			t.Errorf("客户端 UA 不应被拦截: %s", ua)
		}
	}
}

// TestBlockDisabled 开关关闭时谁都不拦
func TestBlockDisabled(t *testing.T) {
	setBlockBrowserAccess(t, false)
	if shouldBlockBrowserSubscriptionAccess(newAccessContext(browserUA, nil)) {
		t.Fatal("开关关闭时不应拦截")
	}
}

// 硬件指纹头（X-MF-Device-Fp）必须被 extractMFHeaders 透传：
// 设备身份去重依赖它（同一台电脑重装/升级后认回老行，不再多占设备名额）。
// 漏掉这个 key 的话后端收不到指纹，重装就会重新登记成新设备（幽灵设备）。
func TestExtractMFHeadersIncludesHardwareFingerprint(t *testing.T) {
	c := newAccessContext("MoneyFly/2.2.29 (Windows NT 10.0; Win64; x64)", map[string]string{
		"X-MF-Device-Id":    "install-id-1",
		"X-MF-Device-Fp":    "hw-fp-abcdef0123456789",
		"X-MF-Device-Model": "Windows 10 Pro",
	})

	hdrs := extractMFHeaders(c)
	if got := hdrs["X-MF-Device-Fp"]; got != "hw-fp-abcdef0123456789" {
		t.Fatalf("X-MF-Device-Fp 必须透传，实际 %q（全部头：%v）", got, hdrs)
	}
	if got := hdrs["X-MF-Device-Id"]; got != "install-id-1" {
		t.Fatalf("X-MF-Device-Id 必须透传，实际 %q", got)
	}
}

// 订阅访问控制必须拦住「非客户端」：脚本、转换站、链接预览爬虫、UA 就是链接本身。
// 线上事故：开关只拦浏览器 UA，Go-http-client / subconverter 能拿到完整配置（462KB），
// 导致订阅被转换站与脚本批量抓取。
func TestNonClientUserAgentsBlocked(t *testing.T) {
	blocked := []string{
		"Go-http-client/1.1",
		"curl/8.5.0",
		"python-requests/2.31.0",
		"subconverter/0.9.0",
		"WhatsApp/2.23.20.0",
		"DingTalkBot-LinkService/1.0 (+https://open-doc.dingtalk.com)",
		"TelegramBot (like TwitterBot)",
		"Mobile/15E148 wxwork/5.0.6 MicroMessenger/7.0.1",
		"https://dy.moneyfly.top/api/v1/subscriptions/clash/abc", // UA 就是订阅链接（含 clash 字样也要拦）
		"vmess://YXV0bzpmMmU2",
		"",
	}
	for _, ua := range blocked {
		if !isNonClientUserAgent(ua) {
			t.Errorf("应拦截但放行了: %q", ua)
		}
	}

	allowed := []string{
		"clash.meta/1.19.0",
		"ClashMetaForAndroid/2.11.0",
		"Shadowrocket/3445 CFNetwork/3896.100.1.2.1 Darwin/27.0.0 iPhone12,1",
		"MoneyFly/2.2.33 (Windows NT 10.0)",
		"MBox/1.0 (Windows NT 10.0)",
		"v2rayN/6.31",
		"sing-box 1.12.2",
		"mihomo.party/v1.9.5 (clash.meta)",
	}
	for _, ua := range allowed {
		if isNonClientUserAgent(ua) {
			t.Errorf("误拦了真实客户端: %q", ua)
		}
	}
}

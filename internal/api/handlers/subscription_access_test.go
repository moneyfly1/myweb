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

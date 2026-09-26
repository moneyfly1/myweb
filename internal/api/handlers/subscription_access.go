package handlers

import (
	"net/http"
	"strings"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"

	"github.com/gin-gonic/gin"
)

const blockBrowserSubscriptionAccessKey = "block_browser_subscription_access"

func shouldBlockBrowserSubscriptionAccess(c *gin.Context) bool {
	// 自研客户端（MoneyFly）一律放行：App 里允许用户自定义「订阅 UA」
	// （有些机场按 UA 返回不同格式，用户会粘贴浏览器 UA 自救），
	// 但同一个 Dio 一定会带上 X-MF-* 设备头 —— 有这些头就说明是本站在客户端，
	// 此时若还按 UA 判定成浏览器并返回空内容，用户就会看到「拉取订阅失败」。
	// 线上实例：客户 dos2009 自定义 UA 为 GooBrowser/Chrome，订阅返回 200 + 0 字节。
	if extractMFHeaders(c) != nil {
		return false
	}
	if !isBrowserUserAgent(c.GetHeader("User-Agent")) {
		return false
	}

	var config models.SystemConfig
	if err := database.GetDB().
		Where("key = ? AND category = ?", blockBrowserSubscriptionAccessKey, "subscription_access").
		First(&config).Error; err != nil {
		return false
	}

	return strings.EqualFold(strings.TrimSpace(config.Value), "true")
}

func respondEmptySubscriptionForBrowser(c *gin.Context) {
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, "")
}

// extractMFHeaders 提取 MoneyFly 客户端发送的 X-MF-* 自定义设备信息头
func extractMFHeaders(c *gin.Context) map[string]string {
	keys := []string{"X-MF-Device-Model", "X-MF-Device-Brand", "X-MF-OS", "X-MF-Device-Type", "X-MF-Device-Id"}
	headers := make(map[string]string, len(keys))
	for _, k := range keys {
		if v := c.GetHeader(k); v != "" {
			headers[k] = v
		}
	}
	if len(headers) == 0 {
		return nil
	}
	return headers
}

func isBrowserUserAgent(userAgent string) bool {
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" {
		return false
	}

	clientMarkers := []string{
		"clash", "stash", "surge", "quantumult", "loon", "sing-box", "singbox",
		"shadowrocket", "v2ray", "v2rayn", "v2rayng", "hiddify", "sing-box",
		"nekobox", "surfboard", "karing", "openclash", "passwall", "moneyfly",
	}
	for _, marker := range clientMarkers {
		if strings.Contains(ua, marker) {
			return false
		}
	}

	browserMarkers := []string{
		"chrome/", "crios/", "firefox/", "fxios/", "safari/", "edg/", "edge/",
		"opr/", "opera/", "msie", "trident/", "duckduckgo/", "vivaldi/",
	}
	for _, marker := range browserMarkers {
		if strings.Contains(ua, marker) {
			return true
		}
	}

	return strings.Contains(ua, "mozilla/") && strings.Contains(ua, "applewebkit/")
}

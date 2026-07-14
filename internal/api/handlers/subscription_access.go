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

func isBrowserUserAgent(userAgent string) bool {
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" {
		return false
	}

	clientMarkers := []string{
		"clash", "stash", "surge", "quantumult", "loon", "sing-box", "singbox",
		"shadowrocket", "v2ray", "v2rayn", "v2rayng", "hiddify", "sing-box",
		"nekobox", "surfboard", "karing", "openclash", "passwall",
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

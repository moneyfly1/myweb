package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// 维护模式缓存
var (
	maintenanceCache     *maintenanceCacheData
	maintenanceCacheMu   sync.RWMutex
	maintenanceCacheTTL  = 30 * time.Second
)

type maintenanceCacheData struct {
	enabled    bool
	message    string
	siteName   string
	logoURL    string
	expireAt   time.Time
}

func getMaintenanceConfig() *maintenanceCacheData {
	maintenanceCacheMu.RLock()
	if maintenanceCache != nil && time.Now().Before(maintenanceCache.expireAt) {
		data := maintenanceCache
		maintenanceCacheMu.RUnlock()
		return data
	}
	maintenanceCacheMu.RUnlock()

	maintenanceCacheMu.Lock()
	defer maintenanceCacheMu.Unlock()

	// double check
	if maintenanceCache != nil && time.Now().Before(maintenanceCache.expireAt) {
		return maintenanceCache
	}

	db := database.GetDB()
	data := &maintenanceCacheData{
		message:  "系统维护中，请稍后再试",
		siteName: "CBoard Modern",
		expireAt: time.Now().Add(maintenanceCacheTTL),
	}

	// 一次查出所有需要的配置
	var configs []models.SystemConfig
	db.Where("(key = ? AND category = ?) OR (key = ? AND category = ?) OR (key = ? AND category IN (?,?)) OR (key = ? AND category IN (?,?))",
		"maintenance_mode", "system",
		"maintenance_message", "system",
		"site_name", "general", "system",
		"logo_url", "general", "system",
	).Find(&configs)

	for _, cfg := range configs {
		switch {
		case cfg.Key == "maintenance_mode" && cfg.Category == "system":
			data.enabled = cfg.Value == "true"
		case cfg.Key == "maintenance_message" && cfg.Category == "system":
			data.message = cfg.Value
		case cfg.Key == "site_name":
			if data.siteName == "CBoard Modern" || cfg.Category == "general" {
				data.siteName = cfg.Value
			}
		case cfg.Key == "logo_url":
			if data.logoURL == "" || cfg.Category == "general" {
				data.logoURL = cfg.Value
			}
		}
	}

	maintenanceCache = data
	return data
}

// InvalidateMaintenanceCache 供外部在修改维护配置时清除缓存
func InvalidateMaintenanceCache() {
	maintenanceCacheMu.Lock()
	maintenanceCache = nil
	maintenanceCacheMu.Unlock()
}

func MaintenanceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		allowedPaths := []string{
			"/api/v1/admin",
			"/api/v1/settings/public-settings",
			"/api/v1/auth/login",
			"/api/v1/auth/login-json",
			"/api/v1/payment/notify/",
			"/health",
			"/static",
			"/uploads",
		}

		isAllowed := false
		for _, allowed := range allowedPaths {
			if strings.HasPrefix(path, allowed) {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			data := getMaintenanceConfig()
			if data.enabled {
				if strings.HasPrefix(path, "/api/") {
					utils.ErrorResponse(c, http.StatusServiceUnavailable, data.message, nil)
					c.Abort()
					return
				}

				htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - 系统维护中</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'Helvetica Neue', Helvetica, Arial, sans-serif; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 20px; }
        .maintenance-container { background: #ffffff; border-radius: 16px; box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3); padding: 60px 40px; max-width: 600px; width: 100%%; text-align: center; animation: fadeIn 0.5s ease-in; }
        @keyframes fadeIn { from { opacity: 0; transform: translateY(-20px); } to { opacity: 1; transform: translateY(0); } }
        .logo { width: 120px; height: 120px; margin: 0 auto 30px; border-radius: 50%%; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); display: flex; align-items: center; justify-content: center; font-size: 48px; color: #ffffff; box-shadow: 0 10px 30px rgba(102, 126, 234, 0.3); }
        .logo img { width: 100%%; height: 100%%; object-fit: cover; border-radius: 50%%; }
        h1 { font-size: 32px; color: #303133; margin-bottom: 20px; font-weight: 600; }
        .message { font-size: 18px; color: #606266; line-height: 1.8; margin-bottom: 40px; white-space: pre-wrap; }
        .icon { font-size: 80px; color: #e6a23c; margin-bottom: 30px; animation: pulse 2s ease-in-out infinite; }
        @keyframes pulse { 0%%, 100%% { transform: scale(1); } 50%% { transform: scale(1.1); } }
        .footer { margin-top: 40px; padding-top: 30px; border-top: 1px solid #e4e7ed; color: #909399; font-size: 14px; }
        @media (max-width: 768px) { .maintenance-container { padding: 40px 20px; } h1 { font-size: 24px; } .message { font-size: 16px; } .icon { font-size: 60px; } }
    </style>
</head>
<body>
    <div class="maintenance-container">
        <div class="logo">%s</div>
        <div class="icon">⚠️</div>
        <h1>系统维护中</h1>
        <div class="message">%s</div>
        <div class="footer">
            <p>%s</p>
            <p style="margin-top: 10px;">我们正在努力为您提供更好的服务</p>
        </div>
    </div>
</body>
</html>`, data.siteName, getLogoHTML(data.logoURL), data.message, data.siteName)

				c.Data(http.StatusServiceUnavailable, "text/html; charset=utf-8", []byte(htmlContent))
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

func getLogoHTML(logoURL string) string {
	if logoURL != "" {
		return fmt.Sprintf(`<img src="%s" alt="Logo" />`, logoURL)
	}
	return "🔧"
}

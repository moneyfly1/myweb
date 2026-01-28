package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

func MaintenanceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		allowedPaths := []string{
			"/api/v1/admin",                    // 所有管理员接口
			"/api/v1/settings/public-settings", // 公开设置（包含维护状态）
			"/api/v1/auth/login",               // 登录接口（需要在登录处理中检查维护模式）
			"/api/v1/auth/login-json",          // 登录接口（需要在登录处理中检查维护模式）
			"/health",                          // 健康检查
			"/static",                          // 静态文件
			"/uploads",                         // 上传文件
		}

		isAllowed := false
		for _, allowed := range allowedPaths {
			if strings.HasPrefix(path, allowed) {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			db := database.GetDB()
			var maintenanceConfig models.SystemConfig
			if err := db.Where("key = ? AND category = ?", "maintenance_mode", "system").First(&maintenanceConfig).Error; err == nil {
				if maintenanceConfig.Value == "true" {
					var messageConfig models.SystemConfig
					maintenanceMessage := "系统维护中，请稍后再试"
					if err := db.Where("key = ? AND category = ?", "maintenance_message", "system").First(&messageConfig).Error; err == nil {
						maintenanceMessage = messageConfig.Value
					}

					var siteNameConfig models.SystemConfig
					siteName := "CBoard Modern"
					if err := db.Where("key = ? AND category = ?", "site_name", "general").First(&siteNameConfig).Error; err == nil {
						siteName = siteNameConfig.Value
					} else if err := db.Where("key = ? AND category = ?", "site_name", "system").First(&siteNameConfig).Error; err == nil {
						siteName = siteNameConfig.Value
					}

					var logoConfig models.SystemConfig
					logoURL := ""
					if err := db.Where("key = ? AND category = ?", "logo_url", "general").First(&logoConfig).Error; err == nil {
						logoURL = logoConfig.Value
					} else if err := db.Where("key = ? AND category = ?", "logo_url", "system").First(&logoConfig).Error; err == nil {
						logoURL = logoConfig.Value
					}

					if strings.HasPrefix(path, "/api/") {
						utils.ErrorResponse(c, http.StatusServiceUnavailable, maintenanceMessage, nil)
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
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'Helvetica Neue', Helvetica, Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .maintenance-container {
            background: #ffffff;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            padding: 60px 40px;
            max-width: 600px;
            width: 100%%;
            text-align: center;
            animation: fadeIn 0.5s ease-in;
        }
        @keyframes fadeIn {
            from {
                opacity: 0;
                transform: translateY(-20px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }
        .logo {
            width: 120px;
            height: 120px;
            margin: 0 auto 30px;
            border-radius: 50%%;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 48px;
            color: #ffffff;
            box-shadow: 0 10px 30px rgba(102, 126, 234, 0.3);
        }
        .logo img {
            width: 100%%;
            height: 100%%;
            object-fit: cover;
            border-radius: 50%%;
        }
        h1 {
            font-size: 32px;
            color: #303133;
            margin-bottom: 20px;
            font-weight: 600;
        }
        .message {
            font-size: 18px;
            color: #606266;
            line-height: 1.8;
            margin-bottom: 40px;
            white-space: pre-wrap;
        }
        .icon {
            font-size: 80px;
            color: #e6a23c;
            margin-bottom: 30px;
            animation: pulse 2s ease-in-out infinite;
        }
        @keyframes pulse {
            0%%, 100%% {
                transform: scale(1);
            }
            50%% {
                transform: scale(1.1);
            }
        }
        .footer {
            margin-top: 40px;
            padding-top: 30px;
            border-top: 1px solid #e4e7ed;
            color: #909399;
            font-size: 14px;
        }
        @media (max-width: 768px) {
            .maintenance-container {
                padding: 40px 20px;
            }
            h1 {
                font-size: 24px;
            }
            .message {
                font-size: 16px;
            }
            .icon {
                font-size: 60px;
            }
        }
    </style>
</head>
<body>
    <div class="maintenance-container">
        <div class="logo">
            %s
        </div>
        <div class="icon">⚠️</div>
        <h1>系统维护中</h1>
        <div class="message">%s</div>
        <div class="footer">
            <p>%s</p>
            <p style="margin-top: 10px;">我们正在努力为您提供更好的服务</p>
        </div>
    </div>
</body>
</html>`, siteName, getLogoHTML(logoURL), maintenanceMessage, siteName)

					c.Data(http.StatusServiceUnavailable, "text/html; charset=utf-8", []byte(htmlContent))
					c.Abort()
					return
				}
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

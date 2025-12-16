package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware 添加安全响应头
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// X-Content-Type-Options: 防止MIME类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")
		
		// X-Frame-Options: 防止点击劫持
		c.Header("X-Frame-Options", "DENY")
		
		// X-XSS-Protection: 启用XSS过滤器（虽然已过时，但为了兼容性保留）
		c.Header("X-XSS-Protection", "1; mode=block")
		
		// Referrer-Policy: 控制referrer信息
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Permissions-Policy: 控制浏览器功能
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		
		// Content-Security-Policy: 内容安全策略（可以根据需要调整）
		// 注意：如果前端使用了内联脚本，需要调整这个策略
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'")
		
		// Strict-Transport-Security: 强制HTTPS（仅在HTTPS环境下启用）
		// 注意：如果使用HTTPS，取消下面的注释
		// c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		
		c.Next()
	}
}


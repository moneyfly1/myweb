package middleware

import (
	"github.com/gin-gonic/gin"
)

func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")

		c.Header("X-Frame-Options", "DENY")

		c.Header("X-XSS-Protection", "1; mode=block")

		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'")

		c.Next()
	}
}

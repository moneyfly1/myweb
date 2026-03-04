package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"cboard-go/internal/core/config"
	"cboard-go/internal/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware 安全响应头中间件
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

// CORSMiddleware CORS跨域中间件
func CORSMiddleware() gin.HandlerFunc {
	cfg := config.AppConfig
	origins := []string{"*"}
	if cfg != nil && len(cfg.CorsOrigins) > 0 {
		origins = cfg.CorsOrigins
	}

	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "accept", "origin", "Cache-Control", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}

// ==========================================
// 日志和请求处理中间件
// ==========================================

// LoggerMiddleware 请求日志中间件
func LoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	})
}

// ErrorRecoveryMiddleware 错误恢复中间件
func ErrorRecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered interface{}) {
		var err error
		var errMsg string
		if e, ok := recovered.(error); ok {
			err = e
			errMsg = e.Error()
		} else {
			errMsg = fmt.Sprintf("%v", recovered)
			err = fmt.Errorf("%v", recovered)
		}

		stack := string(debug.Stack())

		// 清理错误信息中的敏感路径
		safeErrMsg := utils.SanitizeErrorPath(errMsg)

		utils.CreateSystemErrorLog(c, http.StatusInternalServerError,
			fmt.Sprintf("系统异常: %s", safeErrMsg), err)
		c.Set("system_error_logged", true)

		if utils.AppLogger != nil {
			// 生产环境不记录完整堆栈信息
			if utils.IsProduction() {
				utils.AppLogger.Error("[PANIC] %s", safeErrMsg)
			} else {
				utils.AppLogger.Error("[PANIC] %s\n堆栈信息:\n%s", safeErrMsg, stack)
			}
		}

		// ErrorResponse 会自动处理生产环境的错误信息隐藏
		utils.ErrorResponse(c, http.StatusInternalServerError, "服务器内部错误，请稍后重试", err)
		c.Abort()
	})
}

// RequestIDMiddleware 请求ID中间件
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

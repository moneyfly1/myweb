package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// LoggerMiddleware 日志中间件
func LoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// 注意：系统错误日志通过ErrorResponse函数和ErrorRecoveryMiddleware记录
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

// ErrorRecoveryMiddleware 错误恢复中间件（增强版，记录错误到系统日志）
func ErrorRecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered interface{}) {
		// 获取错误信息
		var err error
		var errMsg string
		if e, ok := recovered.(error); ok {
			err = e
			errMsg = e.Error()
		} else {
			errMsg = fmt.Sprintf("%v", recovered)
			err = fmt.Errorf("%v", recovered)
		}

		// 获取堆栈信息
		stack := string(debug.Stack())

		// 记录到系统错误日志
		utils.CreateSystemErrorLog(c, http.StatusInternalServerError,
			fmt.Sprintf("系统异常: %s", errMsg), err)

		// 记录堆栈信息到文件日志
		if utils.AppLogger != nil {
			utils.AppLogger.Error("[PANIC] %s\n堆栈信息:\n%s", errMsg, stack)
		}

		// 返回错误响应
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

// generateRequestID 生成请求ID
func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

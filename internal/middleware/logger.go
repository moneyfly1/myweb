package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

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

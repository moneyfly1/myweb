package utils

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

type ResponseBase struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// IsProduction 检查是否为生产环境
func IsProduction() bool {
	env := os.Getenv("ENV")
	return env == "production" || env == "prod"
}

func SuccessResponse(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(code, ResponseBase{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func ErrorResponse(c *gin.Context, code int, message string, err error) {
	// 记录详细错误到日志
	if err != nil {
		LogError(message, err, map[string]interface{}{
			"path":   c.Request.URL.Path,
			"method": c.Request.Method,
		})
	}

	// 系统错误记录到审计日志
	if code >= http.StatusInternalServerError {
		CreateSystemErrorLog(c, code, message, err)
	}

	// 生产环境：隐藏详细错误信息，使用通用错误消息
	userMessage := message
	if IsProduction() && err != nil {
		// 生产环境不返回详细错误信息
		if code >= http.StatusInternalServerError {
			userMessage = "服务器内部错误，请稍后重试"
		} else if strings.Contains(strings.ToLower(message), "error") || 
		          strings.Contains(strings.ToLower(err.Error()), "path") ||
		          strings.Contains(strings.ToLower(err.Error()), "file") {
			// 如果错误信息包含路径、文件等敏感信息，使用通用消息
			userMessage = "操作失败，请稍后重试"
		}
	}

	c.JSON(code, ResponseBase{
		Success: false,
		Message: userMessage,
	})
}

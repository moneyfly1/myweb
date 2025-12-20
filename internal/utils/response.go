package utils

import (
	"github.com/gin-gonic/gin"
)

// ResponseBase 统一响应格式
type ResponseBase struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// SuccessResponse 成功响应
func SuccessResponse(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(code, ResponseBase{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// ErrorResponse 错误响应（带错误日志记录）
func ErrorResponse(c *gin.Context, code int, message string, err error) {
	if err != nil {
		LogError(message, err, map[string]interface{}{
			"path":   c.Request.URL.Path,
			"method": c.Request.Method,
		})
	}
	c.JSON(code, ResponseBase{
		Success: false,
		Message: message,
	})
}

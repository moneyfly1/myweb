package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ResponseBase struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func SuccessResponse(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(code, ResponseBase{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func ErrorResponse(c *gin.Context, code int, message string, err error) {
	if err != nil {
		LogError(message, err, map[string]interface{}{
			"path":   c.Request.URL.Path,
			"method": c.Request.Method,
		})
	}

	if code >= http.StatusInternalServerError {
		CreateSystemErrorLog(c, code, message, err)
	}

	c.JSON(code, ResponseBase{
		Success: false,
		Message: message,
	})
}

package utils

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// APIResponse 统一API响应格式
type APIResponse struct {
	Success    bool        `json:"success"`
	Code       int         `json:"code"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data,omitempty"`
	Timestamp  int64       `json:"timestamp"`
	RequestID  string      `json:"request_id,omitempty"`
}

// Standard error codes
const (
	ErrCodeSuccess = 0
	ErrCodeBadRequest = 400
	ErrCodeUnauthorized = 401
	ErrCodeForbidden = 403
	ErrCodeNotFound = 404
	ErrCodeInternal = 500
	ErrCodeValidation = 1000
	ErrCodeDatabase = 2000
)

// GetRequestID 从上下文中获取请求ID
func GetRequestID(c *gin.Context) string {
	if rid, exists := c.Get("request_id"); exists {
		if s, ok := rid.(string); ok {
			return s
		}
	}
	return ""
}

// IsProduction 检查是否为生产环境
func IsProduction() bool {
	env := os.Getenv("ENV")
	return env == "production" || env == "prod"
}

func SuccessResponse(c *gin.Context, code int, message string, data interface{}) {
	resp := APIResponse{
		Success:   true,
		Code:      ErrCodeSuccess,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Unix(),
		RequestID: GetRequestID(c),
	}
	c.JSON(code, resp)
}

func ErrorResponse(c *gin.Context, code int, message string, err error) {
	// 确定错误码
	errCode := getErrorCode(code)

	// 记录详细错误到日志
	if err != nil {
		LogError(message, err, map[string]interface{}{
			"path":   c.Request.URL.Path,
			"method": c.Request.Method,
			"code":   code,
			"err_code": errCode,
		})
	}

	// 系统错误记录到审计日志
	if code >= http.StatusInternalServerError {
		alreadyLogged := false
		if logged, exists := c.Get("system_error_logged"); exists {
			if v, ok := logged.(bool); ok && v {
				alreadyLogged = true
			}
		}
		if !alreadyLogged {
			CreateSystemErrorLog(c, code, message, err)
			c.Set("system_error_logged", true)
		}
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

	resp := APIResponse{
		Success:   false,
		Code:      errCode,
		Message:   userMessage,
		Timestamp: time.Now().Unix(),
		RequestID: GetRequestID(c),
	}
	c.JSON(code, resp)
}

// getErrorCode 根据HTTP状态码返回对应的业务错误码
func getErrorCode(statusCode int) int {
	switch statusCode {
	case http.StatusBadRequest:
		return ErrCodeBadRequest
	case http.StatusUnauthorized:
		return ErrCodeUnauthorized
	case http.StatusForbidden:
		return ErrCodeForbidden
	case http.StatusNotFound:
		return ErrCodeNotFound
	case http.StatusInternalServerError:
		return ErrCodeInternal
	default:
		return statusCode
	}
}

// ========== 分页相关 ==========

type PaginationParams struct {
	Page int
	Size int
}

func ParsePagination(c *gin.Context) PaginationParams {
	page := 1
	size := 20

	if pageStr := c.Query("page"); pageStr != "" {
		if _, err := fmt.Sscanf(pageStr, "%d", &page); err != nil {
			page = 1
		}
	}
	if sizeStr := c.Query("size"); sizeStr != "" {
		if _, err := fmt.Sscanf(sizeStr, "%d", &size); err != nil {
			size = 20
		}
	}

	if skipStr := c.Query("skip"); skipStr != "" {
		var skip int
		if _, err := fmt.Sscanf(skipStr, "%d", &skip); err != nil {
			skip = 0
		}
		if skip < 0 {
			skip = 0
		}
		if skip > 100000 {
			skip = 100000
		}
		if page == 1 && size == 20 {
			page = (skip / size) + 1
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		var limit int
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil {
			limit = 20
		}
		if size == 20 {
			size = limit
		}
	}

	if page < 1 {
		page = 1
	}
	if page > 10000 {
		page = 10000
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}

	return PaginationParams{Page: page, Size: size}
}

func (p PaginationParams) GetOffset() int {
	return (p.Page - 1) * p.Size
}

// ========== 错误处理相关 ==========

type SafeError struct {
	UserMessage string // 返回给用户的消息
	InternalErr error  // 内部错误（仅记录到日志）
}

func (e *SafeError) Error() string {
	return e.UserMessage
}

func HandleError(err error, userMessage string) error {
	if err == nil {
		return nil
	}

	log.Printf("Error: %v", err)

	return &SafeError{
		UserMessage: userMessage,
		InternalErr: err,
	}
}

func GetSafeErrorMessage(err error, defaultMessage string) string {
	if err == nil {
		return defaultMessage
	}

	var safeErr *SafeError
	if errors.As(err, &safeErr) {
		return safeErr.UserMessage
	}

	return defaultMessage
}

// 敏感字段列表，这些字段在日志中会被隐藏
var sensitiveFields = map[string]bool{
	"password":         true,
	"token":            true,
	"secret":           true,
	"api_key":          true,
	"api_key_id":       true,
	"access_token":     true,
	"refresh_token":    true,
	"csrf_token":       true,
	"session_id":       true,
	"private_key":      true,
	"public_key":       true,
	"secret_key":       true,
	"encryption_key":   true,
	"decryption_key":   true,
	"auth_token":       true,
	"bearer_token":     true,
	"jwt_token":        true,
	"session_token":    true,
	"api_secret":       true,
	"client_secret":    true,
	"app_secret":       true,
	"webhook_secret":   true,
	"signature":        true,
	"hmac":             true,
	"credential":       true,
	"credentials":      true,
	"subscription_url": true, // 订阅URL也是敏感信息
	"subscription_key": true,
}

func LogError(operation string, err error, context map[string]interface{}) {
	if err == nil {
		return
	}

	// 过滤错误信息中的敏感路径
	errMsg := err.Error()
	if errMsg != "" {
		// 移除文件路径中的敏感信息
		errMsg = sanitizeErrorPath(errMsg)
	}

	msg := fmt.Sprintf("Operation: %s, Error: %v", operation, errMsg)
	if context != nil {
		safeContext := make(map[string]interface{})
		for k, v := range context {
			// 检查字段名（不区分大小写）
			keyLower := strings.ToLower(k)
			if sensitiveFields[keyLower] {
				safeContext[k] = "***REDACTED***"
			} else {
				// 检查值中是否包含敏感信息
				if strVal, ok := v.(string); ok {
					safeContext[k] = sanitizeSensitiveValue(strVal)
				} else {
					safeContext[k] = v
				}
			}
		}
		msg += fmt.Sprintf(", Context: %+v", safeContext)
	}

	if AppLogger != nil {
		AppLogger.Error("%s", msg)
	} else {
		log.Printf("[ERROR] %s", msg)
	}
}

// sanitizeErrorPath 清理错误信息中的文件路径
func sanitizeErrorPath(errMsg string) string {
	// 移除绝对路径，只保留文件名
	// 例如: /Users/apple/Downloads/goweb/file.go -> file.go
	parts := strings.Split(errMsg, "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		// 如果包含文件名，尝试提取
		if strings.Contains(lastPart, ".") {
			// 保留文件名部分
			return strings.Join(parts[len(parts)-2:], "/")
		}
	}
	return errMsg
}

// sanitizeSensitiveValue 清理值中的敏感信息
func sanitizeSensitiveValue(value string) string {
	valueLower := strings.ToLower(value)
	// 检查是否包含敏感关键词
	for field := range sensitiveFields {
		if strings.Contains(valueLower, field) {
			return "***REDACTED***"
		}
	}
	// 检查是否看起来像token或密钥（长字符串）
	if len(value) > 20 && (strings.Contains(value, "-") || strings.Contains(value, "_")) {
		// 可能是token或密钥，部分隐藏
		if len(value) > 40 {
			return value[:10] + "..." + value[len(value)-10:]
		}
	}
	return value
}

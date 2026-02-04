package utils

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

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
	"password":        true,
	"token":           true,
	"secret":          true,
	"api_key":         true,
	"api_key_id":      true,
	"access_token":    true,
	"refresh_token":   true,
	"csrf_token":      true,
	"session_id":      true,
	"private_key":     true,
	"public_key":      true,
	"secret_key":      true,
	"encryption_key":  true,
	"decryption_key":  true,
	"auth_token":      true,
	"bearer_token":    true,
	"jwt_token":       true,
	"session_token":   true,
	"api_secret":      true,
	"client_secret":   true,
	"app_secret":      true,
	"webhook_secret":  true,
	"signature":       true,
	"hmac":            true,
	"credential":      true,
	"credentials":     true,
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
		AppLogger.Error(msg)
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

package utils

import (
	"errors"
	"fmt"
	"log"
)

// SafeError 安全错误处理，不向客户端泄露敏感信息
type SafeError struct {
	UserMessage string // 返回给用户的消息
	InternalErr error  // 内部错误（仅记录到日志）
}

func (e *SafeError) Error() string {
	return e.UserMessage
}

// HandleError 统一处理错误，返回安全的错误信息
func HandleError(err error, userMessage string) error {
	if err == nil {
		return nil
	}

	// 记录详细错误到日志（不包含敏感信息）
	log.Printf("Error: %v", err)

	// 返回用户友好的错误信息
	return &SafeError{
		UserMessage: userMessage,
		InternalErr: err,
	}
}

// GetSafeErrorMessage 获取安全的错误消息
func GetSafeErrorMessage(err error, defaultMessage string) string {
	if err == nil {
		return defaultMessage
	}

	var safeErr *SafeError
	if errors.As(err, &safeErr) {
		return safeErr.UserMessage
	}

	// 生产环境不返回详细错误
	// 开发环境可以返回更多信息
	return defaultMessage
}

// LogError 记录错误到日志（脱敏处理）
func LogError(operation string, err error, context map[string]interface{}) {
	if err == nil {
		return
	}

	// 构建日志消息
	msg := fmt.Sprintf("Operation: %s, Error: %v", operation, err)
	if context != nil {
		// 过滤敏感信息
		safeContext := make(map[string]interface{})
		for k, v := range context {
			// 不记录密码、token等敏感信息
			if k == "password" || k == "token" || k == "secret" || k == "api_key" {
				safeContext[k] = "***REDACTED***"
			} else {
				safeContext[k] = v
			}
		}
		msg += fmt.Sprintf(", Context: %+v", safeContext)
	}

	log.Printf("[ERROR] %s", msg)
}

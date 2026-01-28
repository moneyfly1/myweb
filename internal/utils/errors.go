package utils

import (
	"errors"
	"fmt"
	"log"
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

func LogError(operation string, err error, context map[string]interface{}) {
	if err == nil {
		return
	}

	msg := fmt.Sprintf("Operation: %s, Error: %v", operation, err)
	if context != nil {
		safeContext := make(map[string]interface{})
		for k, v := range context {
			if k == "password" || k == "token" || k == "secret" || k == "api_key" {
				safeContext[k] = "***REDACTED***"
			} else {
				safeContext[k] = v
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

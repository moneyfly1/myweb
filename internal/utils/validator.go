package utils

import (
	"regexp"
	"strings"
)

// ValidateEmail 验证邮箱格式
func ValidateEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// ValidatePhone 验证手机号格式（中国大陆）
func ValidatePhone(phone string) bool {
	pattern := `^1[3-9]\d{9}$`
	matched, _ := regexp.MatchString(pattern, phone)
	return matched
}

// SanitizeInput 清理输入（防止 XSS）
func SanitizeInput(input string) string {
	// 移除危险字符
	dangerous := []string{"<", ">", "\"", "'", "&", ";", "(", ")", "|", "`", "$"}
	result := input
	for _, char := range dangerous {
		result = strings.ReplaceAll(result, char, "")
	}
	return strings.TrimSpace(result)
}

// ValidateUsername 验证用户名格式
func ValidateUsername(username string) bool {
	// 用户名：3-20个字符，只能包含字母、数字、下划线
	pattern := `^[a-zA-Z0-9_]{3,20}$`
	matched, _ := regexp.MatchString(pattern, username)
	return matched
}


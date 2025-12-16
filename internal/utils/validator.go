package utils

import (
	"regexp"
	"strings"
	"unicode"
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

// SanitizeInput 清理输入（防止 XSS 和 SQL 注入）
func SanitizeInput(input string) string {
	if input == "" {
		return ""
	}
	
	// 移除危险字符
	dangerous := []string{"<", ">", "\"", "'", "&", ";", "(", ")", "|", "`", "$", "\\", "/", "*", "%"}
	result := input
	for _, char := range dangerous {
		result = strings.ReplaceAll(result, char, "")
	}
	return strings.TrimSpace(result)
}

// SanitizeSearchKeyword 清理搜索关键词（防止 SQL 注入）
func SanitizeSearchKeyword(keyword string) string {
	if keyword == "" {
		return ""
	}
	
	// 限制长度
	if len(keyword) > 100 {
		keyword = keyword[:100]
	}
	
	// 移除 SQL 注入危险字符
	dangerous := []string{"'", "\"", ";", "--", "/*", "*/", "xp_", "sp_", "exec", "union", "select", "insert", "update", "delete", "drop", "create", "alter"}
	result := strings.ToLower(keyword)
	for _, char := range dangerous {
		result = strings.ReplaceAll(result, char, "")
	}
	
	// 只允许字母、数字、中文、空格、下划线、连字符、@、点号
	var builder strings.Builder
	for _, r := range result {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Han, r) || 
			r == ' ' || r == '_' || r == '-' || r == '@' || r == '.' {
			builder.WriteRune(r)
		}
	}
	
	return strings.TrimSpace(builder.String())
}

// ValidateUsername 验证用户名格式
func ValidateUsername(username string) bool {
	// 用户名：3-20个字符，只能包含字母、数字、下划线
	pattern := `^[a-zA-Z0-9_]{3,20}$`
	matched, _ := regexp.MatchString(pattern, username)
	return matched
}

// ValidatePath 验证文件路径（防止路径遍历）
func ValidatePath(path string, baseDir string) bool {
	// 使用 filepath.Clean 清理路径
	cleaned := strings.TrimSpace(path)
	if cleaned == "" {
		return false
	}
	
	// 检查是否包含路径遍历字符
	if strings.Contains(cleaned, "..") || strings.Contains(cleaned, "~") {
		return false
	}
	
	// 检查是否在基础目录内
	return strings.HasPrefix(cleaned, baseDir)
}


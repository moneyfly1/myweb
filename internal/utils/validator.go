package utils

import (
	"regexp"
	"strings"
	"unicode"
)

func ValidateEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

func ValidatePhone(phone string) bool {
	pattern := `^1[3-9]\d{9}$`
	matched, _ := regexp.MatchString(pattern, phone)
	return matched
}

func SanitizeInput(input string) string {
	if input == "" {
		return ""
	}

	dangerous := []string{"<", ">", "\"", "'", "&", ";", "(", ")", "|", "`", "$", "\\", "/", "*", "%"}
	result := input
	for _, char := range dangerous {
		result = strings.ReplaceAll(result, char, "")
	}
	return strings.TrimSpace(result)
}

func SanitizeSearchKeyword(keyword string) string {
	if keyword == "" {
		return ""
	}

	if len(keyword) > 100 {
		keyword = keyword[:100]
	}

	dangerous := []string{"'", "\"", ";", "--", "/*", "*/", "xp_", "sp_", "exec", "union", "select", "insert", "update", "delete", "drop", "create", "alter"}
	result := strings.ToLower(keyword)
	for _, char := range dangerous {
		result = strings.ReplaceAll(result, char, "")
	}

	var builder strings.Builder
	for _, r := range result {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Han, r) ||
			r == ' ' || r == '_' || r == '-' || r == '@' || r == '.' {
			builder.WriteRune(r)
		}
	}

	return strings.TrimSpace(builder.String())
}

func ValidateUsername(username string) bool {
	pattern := `^[a-zA-Z0-9_]{3,20}$`
	matched, _ := regexp.MatchString(pattern, username)
	return matched
}

func ValidatePath(path string, baseDir string) bool {
	cleaned := strings.TrimSpace(path)
	if cleaned == "" {
		return false
	}

	if strings.Contains(cleaned, "..") || strings.Contains(cleaned, "~") {
		return false
	}

	return strings.HasPrefix(cleaned, baseDir)
}

package utils

import (
	"path/filepath"
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
	pattern := `^[a-zA-Z0-9_\x{4e00}-\x{9fa5}]{2,20}$`
	matched, _ := regexp.MatchString(pattern, username)
	return matched
}

func ValidatePath(path string, baseDir string) bool {
	target := strings.TrimSpace(path)
	base := strings.TrimSpace(baseDir)
	if target == "" || base == "" {
		return false
	}
	return IsWithinBaseDir(base, target)
}

func IsWithinBaseDir(baseDir, targetPath string) bool {
	baseAbs, err := filepath.Abs(filepath.Clean(baseDir))
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(filepath.Clean(targetPath))
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func JoinWithinBaseDir(baseDir string, elems ...string) (string, bool) {
	joined := filepath.Join(append([]string{baseDir}, elems...)...)
	cleaned := filepath.Clean(joined)
	return cleaned, IsWithinBaseDir(baseDir, cleaned)
}

// EscapeLikePattern 转义LIKE查询中的特殊字符，防止注入
// 注意: 只转义 % 和 \，不转义 _ 因为 _ 匹配单字符对搜索无害，且 SQL 未使用 ESCAPE 子句
func EscapeLikePattern(pattern string) string {
	pattern = strings.ReplaceAll(pattern, "\\", "\\\\")
	pattern = strings.ReplaceAll(pattern, "%", "\\%")
	return pattern
}

// SanitizeErrorPath 清理错误信息中的文件路径，防止泄露系统结构
func SanitizeErrorPath(errMsg string) string {
	if errMsg == "" {
		return errMsg
	}

	// 移除绝对路径，只保留文件名
	// 例如: /Users/apple/Downloads/goweb/file.go -> file.go
	parts := strings.Split(errMsg, "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		// 如果包含文件名，尝试提取
		if strings.Contains(lastPart, ".") {
			// 保留最后两个部分（目录名和文件名）
			if len(parts) >= 2 {
				return strings.Join(parts[len(parts)-2:], "/")
			}
			return lastPart
		}
	}

	// 移除常见的系统路径前缀
	pathPrefixes := []string{
		"/Users/",
		"/home/",
		"/var/www/",
		"/usr/local/",
		"/opt/",
		"C:\\",
		"D:\\",
	}

	result := errMsg
	for _, prefix := range pathPrefixes {
		if strings.Contains(result, prefix) {
			// 移除路径前缀，只保留相对路径
			idx := strings.Index(result, prefix)
			if idx >= 0 {
				// 找到最后一个斜杠后的部分
				remaining := result[idx+len(prefix):]
				if slashIdx := strings.LastIndex(remaining, "/"); slashIdx >= 0 {
					result = "..." + remaining[slashIdx:]
				} else {
					result = "..." + remaining
				}
			}
		}
	}

	return result
}

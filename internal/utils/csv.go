package utils

// SanitizeCSVField 防止 CSV 公式注入（CSV injection）：
// 以 = + - @ 或制表符/回车开头的字段加单引号前缀，
// 避免字段被 Excel / WPS / LibreOffice 当作公式执行。
// 适用于所有写入 CSV 的用户可控字段（用户名、邮箱、备注、日志内容等）。
func SanitizeCSVField(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
}

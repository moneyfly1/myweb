package utils

import "strings"

// 字符串截断的唯一实现。
//
// 背景：全仓曾散落 13 份 truncate，语义被分成 4 种（有的按字节切、有的按字符切、
// 有的带 "..." 有的带 "…"）。按**字节**切在中文/表情上会切出半个 UTF-8 序列，
// 在日志、审计详情、邮件里显示成乱码；同一个名称在不同页面被截成不同长度也让人困惑。
// 现在统一按**字符（rune）**截断，省略号由调用方显式指定。

// TruncateRunes 截断到最多 maxRunes 个字符（不追加省略号）。
// maxRunes <= 0 时返回空串。
func TruncateRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes])
}

// TruncateRunesEllipsis 截断到最多 maxRunes 个字符，超长时追加 ellipsis。
// 注意省略号使总长度变成 maxRunes + len(ellipsis)，与既有各处行为保持一致。
func TruncateRunesEllipsis(s string, maxRunes int, ellipsis string) string {
	if maxRunes <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes]) + ellipsis
}

// TruncateRunesTrimEllipsis 先去掉首尾空白再截断（UA/客户端标识常用），超长追 ellipsis。
func TruncateRunesTrimEllipsis(s string, maxRunes int, ellipsis string) string {
	s = strings.TrimSpace(s)
	if maxRunes <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes]) + ellipsis
}

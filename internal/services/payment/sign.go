package payment

import (
	"crypto/md5" // #nosec G501 - MD5 required by epay/wechat/codepay API specifications
	"fmt"
	"sort"
	"strings"
)

// buildSignString 按各支付平台通用签名规则拼串：
// 过滤空值与 excludeKeys 中的键，其余参数按 key 排序后拼接为 k=v&k=v...。
// 返回的字符串不含商户密钥，由调用方决定追加方式（直接拼接或 &key= 形式）。
func buildSignString(params map[string]string, excludeKeys ...string) string {
	var keys []string
	excludeMap := make(map[string]bool)
	for _, k := range excludeKeys {
		excludeMap[k] = true
	}

	for k, v := range params {
		if v == "" || excludeMap[k] {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString("&")
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(params[k])
	}
	return sb.String()
}

// md5Sign 计算字符串的小写 MD5 十六进制摘要。
// #nosec G401 - MD5 is required by payment API specifications, not used for security-critical operations
func md5Sign(str string) string {
	hash := md5.Sum([]byte(str)) // #nosec G401
	return fmt.Sprintf("%x", hash)
}

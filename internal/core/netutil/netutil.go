// Package netutil 是全站唯一的「IP 判定与规范化」公共实现（叶子包，不 import 任何 internal 包）。
//
// 为什么单独建包：internal/utils 与 internal/services/geoip 互相不能复用对方的实现
// （utils 已 import services/geoip，反向 import 会形成循环依赖）。历史上因此出现了
// 9 份互不等价的"内网判断"与 6 份 IP 规范化实现，并直接导致：
//   - 192.168.x.x 被当成公网 IP 外发到第三方查询接口（内网拓扑泄漏）
//   - "内网"/"本地" 这类占位字符串被当成国家名进入地区统计
//   - 同一 IP 在不同模块判定结果不同
package netutil

import (
	"net"
	"strings"
)

// IsPrivateOrReserved 判断 IP 是否为私有/保留地址（内网、回环、链路本地、CGNAT、文档用途等）。
//
// 覆盖范围（合并此前 8 份实现的并集）：
//
//	IPv4: 10/8、172.16/12、192.168/16、127/8、169.254/16、0.0.0.0、
//	      100.64/10（CGNAT）、192.0.0/24、192.0.2/24、198.18/15、198.51.100/24、
//	      203.0.113/24、224/4（组播）、240/4（保留）
//	IPv6: ::1、fe80::/10（链路本地）、fc00::/7（唯一本地）、::（未指定）、
//	      2001:db8::/32（文档用途）
func IsPrivateOrReserved(ip net.IP) bool {
	if ip == nil {
		return true
	}
	// 统一按 4 字节形式处理 ::ffff:127.0.0.1 这类 IPv4-mapped 地址
	if v4 := ip.To4(); v4 != nil {
		switch {
		case v4[0] == 10:
			return true
		case v4[0] == 172 && v4[1] >= 16 && v4[1] <= 31:
			return true
		case v4[0] == 192 && v4[1] == 168:
			return true
		case v4[0] == 127:
			return true
		case v4[0] == 169 && v4[1] == 254:
			return true
		case v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127: // CGNAT 100.64/10
			return true
		case v4[0] == 0:
			return true
		case v4[0] == 192 && v4[1] == 0 && v4[2] == 0: // 192.0.0/24
			return true
		case v4[0] == 192 && v4[1] == 0 && v4[2] == 2: // TEST-NET-1
			return true
		case v4[0] == 198 && (v4[1] == 18 || v4[1] == 19): // 198.18/15
			return true
		case v4[0] == 198 && v4[1] == 51 && v4[2] == 100: // TEST-NET-2
			return true
		case v4[0] == 203 && v4[1] == 0 && v4[2] == 113: // TEST-NET-3
			return true
		case v4[0] >= 224: // 组播 + 保留
			return true
		}
		return false
	}

	switch {
	case ip.IsUnspecified():
		return true
	case ip.IsLoopback(), ip.IsLinkLocalUnicast(), ip.IsLinkLocalMulticast(), ip.IsInterfaceLocalMulticast():
		return true
	case ip.IsMulticast():
		return true
	}
	if len(ip) == net.IPv6len {
		// fc00::/7 唯一本地地址
		if ip[0]&0xfe == 0xfc {
			return true
		}
		// 2001:db8::/32 文档用途
		if ip[0] == 0x20 && ip[1] == 0x01 && ip[2] == 0x0d && ip[3] == 0xb8 {
			return true
		}
	}
	return false
}

// IsPrivateOrReservedString 字符串版判定；空串/非法 IP 视为不可用（返回 true）
func IsPrivateOrReservedString(raw string) bool {
	ip := net.ParseIP(strings.TrimSpace(raw))
	if ip == nil {
		return true
	}
	return IsPrivateOrReserved(ip)
}

// StripHostPort 去掉 "ip:port" 里的端口；IPv6 形如 "[::1]:8080" 也能正确处理
func StripHostPort(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(s); err == nil {
		return host
	}
	return s
}

// Normalize 规范化 IP 字符串，统一全站写法：
//   - 去掉 host:port
//   - "::1" / "localhost" → "127.0.0.1"
//   - ::ffff:x.x.x.x → x.x.x.x
//   - 非法输入：strict=true 返回 ""（落库场景）；strict=false 原样返回（展示场景）
func Normalize(raw string, strict bool) string {
	s := StripHostPort(raw)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	if lower == "localhost" {
		return "127.0.0.1"
	}
	ip := net.ParseIP(s)
	if ip == nil {
		if strict {
			return ""
		}
		return s
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}
	if ip.Equal(net.IPv6loopback) {
		return "127.0.0.1"
	}
	return ip.String()
}

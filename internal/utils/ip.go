package utils

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetRealClientIP 获取客户端真实IP地址
// 优先级：CF-Connecting-IP > True-Client-IP > X-Forwarded-For > X-Real-IP > RemoteAddr
// 正确处理代理和负载均衡器的情况
func GetRealClientIP(c *gin.Context) string {
		// 1. Cloudflare 真实IP（如果使用 Cloudflare CDN）
		if ip := c.GetHeader("CF-Connecting-IP"); ip != "" {
			if realIP := ParseIP(ip); realIP != "" {
				return realIP
			}
		}

		// 2. True-Client-IP（某些代理使用）
		if ip := c.GetHeader("True-Client-IP"); ip != "" {
			if realIP := ParseIP(ip); realIP != "" {
				return realIP
			}
		}

		// 3. X-Forwarded-For（可能包含多个IP，取第一个）
		if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
			// X-Forwarded-For 可能包含多个IP，用逗号分隔，取第一个
			ips := strings.Split(xff, ",")
			for _, ip := range ips {
				ip = strings.TrimSpace(ip)
				if realIP := ParseIP(ip); realIP != "" {
					return realIP
				}
			}
		}

		// 4. X-Real-IP
		if ip := c.GetHeader("X-Real-IP"); ip != "" {
			if realIP := ParseIP(ip); realIP != "" {
				return realIP
			}
		}

		// 5. 使用 Gin 的 ClientIP（它会尝试从上述头获取，但可能不够完善）
		if ip := c.ClientIP(); ip != "" {
			if realIP := ParseIP(ip); realIP != "" {
				return realIP
			}
		}

		// 6. 最后备选：从 RemoteAddr 获取
		if ip, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
			if realIP := ParseIP(ip); realIP != "" {
				return realIP
			}
		}

	// 如果都获取不到，返回空字符串
	return ""
}

// ParseIP 解析并验证IP地址，返回有效的IP字符串
// 这是一个导出的函数，可以被其他包使用
func ParseIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}

	// 移除端口号（如果有）
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}

	// 验证是否为有效的IP地址
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return ""
	}

	// 处理IPv6 localhost，转换为IPv4格式
	if ip == "::1" {
		return "127.0.0.1"
	}

	// 处理IPv6映射的IPv4地址（如 ::ffff:127.0.0.1 或 ::ffff:192.168.1.1）
	if strings.HasPrefix(ip, "::ffff:") {
		ipv4 := strings.TrimPrefix(ip, "::ffff:")
		if parsedIPv4 := net.ParseIP(ipv4); parsedIPv4 != nil && parsedIPv4.To4() != nil {
			return ipv4
		}
	}

	// 如果是IPv4地址，直接返回
	if parsedIP.To4() != nil {
		return ip
	}

	// 对于其他IPv6地址，返回原值
	return ip
}

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
		if realIP := parseIP(ip); realIP != "" {
			return realIP
		}
	}

	// 2. True-Client-IP（某些代理使用）
	if ip := c.GetHeader("True-Client-IP"); ip != "" {
		if realIP := parseIP(ip); realIP != "" {
			return realIP
		}
	}

	// 3. X-Forwarded-For（可能包含多个IP，取第一个）
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For 可能包含多个IP，用逗号分隔，取第一个
		ips := strings.Split(xff, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if realIP := parseIP(ip); realIP != "" {
				return realIP
			}
		}
	}

	// 4. X-Real-IP
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		if realIP := parseIP(ip); realIP != "" {
			return realIP
		}
	}

	// 5. 使用 Gin 的 ClientIP（它会尝试从上述头获取，但可能不够完善）
	if ip := c.ClientIP(); ip != "" {
		if realIP := parseIP(ip); realIP != "" {
			return realIP
		}
	}

	// 6. 最后备选：从 RemoteAddr 获取
	if ip, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
		if realIP := parseIP(ip); realIP != "" {
			return realIP
		}
	}

	// 如果都获取不到，返回空字符串
	return ""
}

// parseIP 解析并验证IP地址，返回有效的IP字符串
func parseIP(ip string) string {
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

	// 排除私有IP和本地IP（如果这些是代理IP，可能需要保留）
	// 但通常我们想要的是客户端真实IP，所以保留所有IP
	return ip
}

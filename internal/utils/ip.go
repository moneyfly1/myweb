package utils

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetRealClientIP(c *gin.Context) string {
	if ip := c.GetHeader("CF-Connecting-IP"); ip != "" {
		if realIP := ParseIP(ip); realIP != "" {
			return realIP
		}
	}

	if ip := c.GetHeader("True-Client-IP"); ip != "" {
		if realIP := ParseIP(ip); realIP != "" {
			return realIP
		}
	}

	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if realIP := ParseIP(ip); realIP != "" {
				return realIP
			}
		}
	}

	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		if realIP := ParseIP(ip); realIP != "" {
			return realIP
		}
	}

	if ip := c.ClientIP(); ip != "" {
		if realIP := ParseIP(ip); realIP != "" {
			return realIP
		}
	}

	if ip, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
		if realIP := ParseIP(ip); realIP != "" {
			return realIP
		}
	}

	return ""
}

func ParseIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}

	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return ""
	}

	if ip == "::1" {
		return "127.0.0.1"
	}

	if strings.HasPrefix(ip, "::ffff:") {
		ipv4 := strings.TrimPrefix(ip, "::ffff:")
		if parsedIPv4 := net.ParseIP(ipv4); parsedIPv4 != nil && parsedIPv4.To4() != nil {
			return ipv4
		}
	}

	if parsedIP.To4() != nil {
		return ip
	}

	return ip
}

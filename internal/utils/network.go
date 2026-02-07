package utils

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"cboard-go/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ========== URL相关 ==========

func BuildBaseURL(r *http.Request, domainName string) string {
	if domainName != "" {
		domain := strings.TrimSpace(domainName)
		if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
			return strings.TrimSuffix(domain, "/")
		}

		scheme := "https"
		if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else if r.TLS == nil {
			scheme = "http"
		}
		return fmt.Sprintf("%s://%s", scheme, domain)
	}

	scheme := "http"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

func GetBuildBaseURL(c *http.Request, db *gorm.DB) string {
	var cfg models.SystemConfig
	var domain string
	if db != nil {
		if err := db.Where("key = ? AND category = ?", "domain_name", "general").First(&cfg).Error; err == nil {
			domain = cfg.Value
		} else if err := db.Where("key = ? AND category = ?", "domain_name", "system").First(&cfg).Error; err == nil {
			domain = cfg.Value
		}
	}
	return BuildBaseURL(c, domain)
}

func GetDomainFromDB(db *gorm.DB) string {
	if db == nil {
		return ""
	}
	var cfg models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "domain_name", "general").First(&cfg).Error; err == nil {
		return strings.TrimSpace(cfg.Value)
	} else if err := db.Where("key = ? AND category = ?", "domain_name", "system").First(&cfg).Error; err == nil {
		return strings.TrimSpace(cfg.Value)
	}
	return ""
}

func FormatDomainURL(domain string) string {
	if domain == "" {
		return ""
	}
	domain = strings.TrimSpace(domain)
	if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
		return strings.TrimSuffix(domain, "/")
	}
	return "https://" + strings.TrimRight(domain, "/")
}

// ========== IP相关 ==========

// IsPrivateIP 检查IP是否为私有IP（内网IP或本地IP）
func IsPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// 检查是否为本地回环地址
	if ip.IsLoopback() {
		return true
	}

	// 检查IPv4私有地址范围
	if ip.To4() != nil {
		// 127.0.0.0/8 - 本地回环
		if ip[0] == 127 {
			return true
		}
		// 10.0.0.0/8 - 私有网络
		if ip[0] == 10 {
			return true
		}
		// 172.16.0.0/12 - 私有网络
		if ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31 {
			return true
		}
		// 192.168.0.0/16 - 私有网络
		if ip[0] == 192 && ip[1] == 168 {
			return true
		}
		// 169.254.0.0/16 - 链路本地地址
		if ip[0] == 169 && ip[1] == 254 {
			return true
		}
		return false
	}

	// 检查IPv6私有地址
	if ip.To16() != nil {
		// ::1 - 本地回环
		if ip.Equal(net.IPv6loopback) {
			return true
		}
		// fe80::/10 - 链路本地地址
		if ip[0] == 0xfe && (ip[1]&0xc0) == 0x80 {
			return true
		}
		// fc00::/7 - 唯一本地地址
		if (ip[0] & 0xfe) == 0xfc {
			return true
		}
	}

	return false
}

func GetRealClientIP(c *gin.Context) string {
	// 优先级1: CF-Connecting-IP (Cloudflare)
	if ip := c.GetHeader("CF-Connecting-IP"); ip != "" {
		if realIP := ParseIP(ip); realIP != "" && !IsPrivateIP(net.ParseIP(realIP)) {
			return realIP
		}
	}

	// 优先级2: True-Client-IP (Cloudflare Enterprise)
	if ip := c.GetHeader("True-Client-IP"); ip != "" {
		if realIP := ParseIP(ip); realIP != "" && !IsPrivateIP(net.ParseIP(realIP)) {
			return realIP
		}
	}

	// 优先级3: X-Forwarded-For (从右到左检查，最后一个通常是客户端IP)
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		// 从最后一个IP开始检查（客户端IP通常在最后）
		for i := len(ips) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(ips[i])
			if realIP := ParseIP(ip); realIP != "" {
				parsedIP := net.ParseIP(realIP)
				if parsedIP != nil && !IsPrivateIP(parsedIP) {
					return realIP
				}
			}
		}
		// 如果所有IP都是内网IP，返回最后一个（可能是内网代理后的真实客户端）
		for i := len(ips) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(ips[i])
			if realIP := ParseIP(ip); realIP != "" {
				return realIP
			}
		}
	}

	// 优先级4: X-Real-IP
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		if realIP := ParseIP(ip); realIP != "" && !IsPrivateIP(net.ParseIP(realIP)) {
			return realIP
		}
	}

	// 优先级5: Gin的ClientIP()方法
	if ip := c.ClientIP(); ip != "" {
		if realIP := ParseIP(ip); realIP != "" {
			parsedIP := net.ParseIP(realIP)
			if parsedIP != nil && !IsPrivateIP(parsedIP) {
				return realIP
			}
		}
	}

	// 优先级6: RemoteAddr (最后备选)
	if ip, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
		if realIP := ParseIP(ip); realIP != "" {
			parsedIP := net.ParseIP(realIP)
			if parsedIP != nil && !IsPrivateIP(parsedIP) {
				return realIP
			}
		}
	}

	// 如果所有IP都是内网IP，返回最后一个获取到的IP（至少记录一个IP）
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

	// 将IPv6映射的IPv4地址转换为IPv4
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

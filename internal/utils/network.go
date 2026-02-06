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

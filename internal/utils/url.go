package utils

import (
	"fmt"
	"net/http"
	"strings"

	"cboard-go/internal/models"

	"gorm.io/gorm"
)

// BuildBaseURL 根据配置的域名或请求头构造基础 URL
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

// GetBuildBaseURL 根据 gin.Context 获取基础 URL
func GetBuildBaseURL(c *http.Request, db *gorm.DB) string {
	var cfg models.SystemConfig
	var domain string
	if db != nil {
		// 优先从 category = "general" 获取域名配置，如果没有则从 "system" 获取
		// 因为保存时 domain_name 会被特殊处理保存到 "system" 分类
		if err := db.Where("key = ? AND category = ?", "domain_name", "general").First(&cfg).Error; err == nil {
			domain = cfg.Value
		} else if err := db.Where("key = ? AND category = ?", "domain_name", "system").First(&cfg).Error; err == nil {
			domain = cfg.Value
		}
	}
	return BuildBaseURL(c, domain)
}

// GetDomainFromDB 从数据库获取域名配置（优先从 category = "general" 获取，如果没有则从 "system" 获取）
func GetDomainFromDB(db *gorm.DB) string {
	if db == nil {
		return ""
	}
	var cfg models.SystemConfig
	// 优先从 category = "general" 获取，如果没有则从 "system" 获取
	// 因为保存时 domain_name 会被特殊处理保存到 "system" 分类
	if err := db.Where("key = ? AND category = ?", "domain_name", "general").First(&cfg).Error; err == nil {
		return strings.TrimSpace(cfg.Value)
	} else if err := db.Where("key = ? AND category = ?", "domain_name", "system").First(&cfg).Error; err == nil {
		return strings.TrimSpace(cfg.Value)
	}
	return ""
}

// FormatDomainURL 格式化域名URL（如果包含协议则直接使用，否则默认使用 https）
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

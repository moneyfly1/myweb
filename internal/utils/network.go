package utils

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"cboard-go/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ========== URL相关 ==========

// ValidateHTTPURL 验证HTTP URL以防止SSRF攻击
// 检查URL格式、协议和主机地址，确保不访问内网资源
func ValidateHTTPURL(rawURL string) error {
	// 解析URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("无效的URL格式: %w", err)
	}

	// 验证协议只允许 http 或 https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("不支持的协议: %s，仅允许 http 或 https", parsedURL.Scheme)
	}

	// 获取主机名
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL缺少主机名")
	}

	// 检查是否为localhost
	if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
		return fmt.Errorf("禁止访问本地地址")
	}

	// 解析主机名为IP地址
	ips, err := net.LookupIP(hostname)
	if err != nil {
		// 如果无法解析，可能是无效的域名，但我们允许继续（可能是DNS问题）
		// 在生产环境中，可以选择更严格的策略
		return nil
	}

	// 检查所有解析出的IP地址
	for _, ip := range ips {
		if IsPrivateIP(ip) {
			return fmt.Errorf("禁止访问内网地址: %s", ip.String())
		}
	}

	return nil
}

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

// trustedProxyCIDRs 存放可信代理 CIDR 列表（来自 TRUSTED_PROXIES 环境变量）。
// 仅在直连地址命中该列表时才信任 X-Forwarded-For / CF-Connecting-IP 等转发头，
// 否则这些头可被客户端伪造，导致限流/审计/IP 管控全部失效。
var trustedProxyCIDRs []*net.IPNet

// InitTrustedProxies 从环境变量 TRUSTED_PROXIES（逗号分隔的 IP/CIDR）初始化可信代理列表。
// 为空表示不信任任何代理：GetRealClientIP 只返回直连地址。
func InitTrustedProxies(value string) {
	trustedProxyCIDRs = nil
	if strings.TrimSpace(value) == "" {
		return
	}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ipNet, err := net.ParseCIDR(part); err == nil {
			trustedProxyCIDRs = append(trustedProxyCIDRs, ipNet)
			continue
		}
		if ip := net.ParseIP(part); ip != nil {
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			trustedProxyCIDRs = append(trustedProxyCIDRs, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
		}
	}
}

func isTrustedProxy(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, ipNet := range trustedProxyCIDRs {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

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

// GetRealClientIP 获取真实客户端 IP。
//
// 安全模型：默认（未配置 TRUSTED_PROXIES）只信任直连地址（Gin ClientIP / RemoteAddr），
// 客户端传入的 CF-Connecting-IP / X-Forwarded-For / X-Real-IP 等转发头一律忽略，
// 防止攻击者伪造 IP 绕过登录/注册/验证码限流。
// 当部署在可信反代（nginx/Cloudflare）之后时，设置 TRUSTED_PROXIES=<代理IP或CIDR,逗号分隔>
// 才会读取转发头（且只接受首个非内网值）。
func GetRealClientIP(c *gin.Context) string {
	directIP := directClientIP(c)
	directParsed := net.ParseIP(directIP)

	// 只有直连地址来自可信代理时才读取转发头
	if isTrustedProxy(directParsed) {
		// 优先级1: CF-Connecting-IP (Cloudflare)
		if ip := c.GetHeader("CF-Connecting-IP"); ip != "" {
			if realIP := ParseIP(ip); realIP != "" && !IsPrivateIP(net.ParseIP(realIP)) {
				return realIP
			}
		}
		// 优先级2: X-Forwarded-For（从右向左取第一个公网 IP）
		if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
			ips := strings.Split(xff, ",")
			for i := len(ips) - 1; i >= 0; i-- {
				ip := strings.TrimSpace(ips[i])
				if realIP := ParseIP(ip); realIP != "" {
					parsedIP := net.ParseIP(realIP)
					if parsedIP != nil && !IsPrivateIP(parsedIP) {
						return realIP
					}
				}
			}
			// 全部为内网（如多层内网代理），取最后一个
			for i := len(ips) - 1; i >= 0; i-- {
				if realIP := ParseIP(strings.TrimSpace(ips[i])); realIP != "" {
					return realIP
				}
			}
		}
		// 优先级3: X-Real-IP
		if ip := c.GetHeader("X-Real-IP"); ip != "" {
			if realIP := ParseIP(ip); realIP != "" && !IsPrivateIP(net.ParseIP(realIP)) {
				return realIP
			}
		}
	}

	if directIP != "" {
		return directIP
	}
	if ip, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
		if realIP := ParseIP(ip); realIP != "" {
			return realIP
		}
	}
	return ""
}

// directClientIP 返回与请求直连的地址（Gin ClientIP，优先）。
func directClientIP(c *gin.Context) string {
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

// NormalizeIP 规范化 IP 字符串：去除 host:port、::1→127.0.0.1、去 ::ffff: 前缀。
// 与 ParseIP 的区别：对无法解析的字符串原样返回（而非丢弃），适合展示场景。
func NormalizeIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}

	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
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

	return ip
}

// FormatIP 规范化 IP 并处理空值（空值返回 "-"），用于前端展示。
func FormatIP(ip string) string {
	normalized := NormalizeIP(ip)
	if normalized == "" {
		return "-"
	}
	return normalized
}

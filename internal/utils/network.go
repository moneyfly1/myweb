package utils

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"cboard-go/internal/core/netutil"
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

// 订阅域名相关配置键（category=general）。
// 为什么订阅域名要和网站域名分开（2026-09-23 线上需求）：
// 官网域名在部分地区会被屏蔽，客户连官网都打不开 —— 把订阅链接放在独立域名上，
// 客户端只要有一个域名能通就能继续更新订阅。老域名一直可用（订阅 token 与域名无关，
// 服务端不校验 Host），所以换域名不会让已发出的订阅地址失效。
const (
	SubscriptionDomainKey        = "subscription_domain"
	SubscriptionBackupDomainsKey = "subscription_backup_domains"
)

// normalizeBaseURLValue 把配置值规整成 "scheme://host"（缺 scheme 时默认 https）。
func normalizeBaseURLValue(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if !strings.Contains(v, "://") {
		v = "https://" + v
	}
	return strings.TrimSuffix(v, "/")
}

// configValueOf 读取 system_configs 中某个 key 的值（先 general 后 system 分类）。
func configValueOf(db *gorm.DB, key string) string {
	if db == nil {
		return ""
	}
	var cfg models.SystemConfig
	if err := db.Where("key = ? AND category = ?", key, "general").First(&cfg).Error; err == nil {
		return cfg.Value
	} else if err := db.Where("key = ? AND category = ?", key, "system").First(&cfg).Error; err == nil {
		return cfg.Value
	}
	return ""
}

// SubscriptionBaseURL 返回订阅链接使用的主域名（不含结尾斜杠）。
// 未配置 subscription_domain 时回退到网站域名（GetBuildBaseURL），保持旧行为。
// c 允许为 nil（异步任务/邮件场景）：此时只用配置值，取不到则返回空串，由调用方兜底。
func SubscriptionBaseURL(c *http.Request, db *gorm.DB) string {
	if v := normalizeBaseURLValue(configValueOf(db, SubscriptionDomainKey)); v != "" {
		return v
	}
	if c == nil {
		return normalizeBaseURLValue(configValueOf(db, "domain_name"))
	}
	return GetBuildBaseURL(c, db)
}

// SubscriptionBaseURLs 返回订阅地址可用的全部域名：主域名在前，其后是备用域名（去重、封顶 5 个）。
// 备用域名由 subscription_backup_domains 配置（逗号/换行/分号分隔），客户端可逐个尝试。
func SubscriptionBaseURLs(c *http.Request, db *gorm.DB) []string {
	primary := SubscriptionBaseURL(c, db)
	out := make([]string, 0, 5)
	seen := make(map[string]bool, 5)
	add := func(v string) {
		v = normalizeBaseURLValue(v)
		if v == "" || seen[v] || len(out) >= 5 {
			return
		}
		seen[v] = true
		out = append(out, v)
	}

	add(primary)
	raw := configValueOf(db, SubscriptionBackupDomainsKey)
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == ';' || r == ' ' || r == '\t' || r == '\r'
	}) {
		add(part)
	}
	// 主域名永远可用的兜底：网站域名也在列表里（同一个后端，token 通用）
	if c != nil {
		add(GetBuildBaseURL(c, db))
	} else {
		add(configValueOf(db, "domain_name"))
	}
	return out
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
// IsPrivateIP 判断是否为私有/保留地址。
// 实现统一委托 internal/core/netutil（叶子包），避免 geoip / statistics 等处
// 各写一份互不等价的判断（历史上有 8 份，导致同一 IP 在不同模块结论不同）。
func IsPrivateIP(ip net.IP) bool {
	return netutil.IsPrivateOrReserved(ip)
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

// directClientIP 返回与请求**真实直连**的对端地址。
//
// 必须是 TCP 对端（RemoteAddr），不能用 c.ClientIP()：
// 后者是 Gin 按其自身信任链解析 XFF 之后的结果，拿它再去做"是否来自可信代理"的判断
// 属于循环论证 —— 会让 CF-Connecting-IP / X-Real-IP 分支在多数部署下不可达。
// 只有在 RemoteAddr 不可解析时才退回 c.ClientIP() 兜底。
func directClientIP(c *gin.Context) string {
	if c.Request != nil {
		if host, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
			if realIP := ParseIP(host); realIP != "" {
				return realIP
			}
		}
		if realIP := ParseIP(c.Request.RemoteAddr); realIP != "" {
			return realIP
		}
	}
	if ip := c.ClientIP(); ip != "" {
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
	// 统一委托 netutil.Normalize（全站唯一的 IP 规范化实现）。
	// 此前这里是一份手写实现，与 netutil.Normalize 行为不一致：
	//   - "localhost" 这里原样保留，netutil 归一为 127.0.0.1；
	//   - IPv6 这里原样返回（大小写/压缩形式保持库里的原始写法），
	//     netutil 会规范成标准压缩小写形式。
	// 结果是同一个 IP 在"用户详情抽屉"（走本函数）与其它页面（走 netutil）
	// 显示不一致，故统一到 netutil。
	return netutil.Normalize(ip, false)
}

// FormatIP 规范化 IP 并处理空值（空值返回 "-"），用于前端展示。
func FormatIP(ip string) string {
	normalized := NormalizeIP(ip)
	if normalized == "" {
		return "-"
	}
	return normalized
}

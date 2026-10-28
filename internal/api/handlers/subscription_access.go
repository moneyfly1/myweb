package handlers

import (
	"net/http"
	"strings"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"

	"github.com/gin-gonic/gin"
)

const blockBrowserSubscriptionAccessKey = "block_browser_subscription_access"

func shouldBlockBrowserSubscriptionAccess(c *gin.Context) bool {
	// 自研客户端（MoneyFly）一律放行：App 里允许用户自定义「订阅 UA」
	// （有些机场按 UA 返回不同格式，用户会粘贴浏览器 UA 自救），
	// 但同一个 Dio 一定会带上 X-MF-* 设备头 —— 有这些头就说明是本站在客户端，
	// 此时若还按 UA 判定成浏览器并返回空内容，用户就会看到「拉取订阅失败」。
	// 线上实例：客户 dos2009 自定义 UA 为 GooBrowser/Chrome，订阅返回 200 + 0 字节。
	if extractMFHeaders(c) != nil {
		return false
	}
	if !isNonClientUserAgent(c.GetHeader("User-Agent")) {
		return false
	}

	var config models.SystemConfig
	if err := database.GetDB().
		Where("key = ? AND category = ?", blockBrowserSubscriptionAccessKey, "subscription_access").
		First(&config).Error; err != nil {
		return false
	}

	return strings.EqualFold(strings.TrimSpace(config.Value), "true")
}

func respondEmptySubscriptionForBrowser(c *gin.Context) {
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, "")
}

// extractMFHeaders 提取 MoneyFly 客户端发送的 X-MF-* 自定义设备信息头
func extractMFHeaders(c *gin.Context) map[string]string {
	keys := []string{"X-MF-Device-Model", "X-MF-Device-Brand", "X-MF-OS", "X-MF-Device-Type",
		"X-MF-Device-Id", "X-MF-Device-Fp"}
	headers := make(map[string]string, len(keys))
	for _, k := range keys {
		if v := c.GetHeader(k); v != "" {
			headers[k] = v
		}
	}
	if len(headers) == 0 {
		return nil
	}
	return headers
}

// isNonClientUserAgent 判断这个 UA 是否属于「非客户端」——
// 浏览器、脚本/抓取工具、订阅转换站、或干脆没有 UA。
//
// 业务约定：订阅配置只给**软件客户端**。此前只拦浏览器 UA，
// 于是 Go-http-client / curl / python-requests / subconverter 这类程序
// 都能直接拿到完整配置（实测 190KB 全量返回），用户把订阅地址贴到在线转换站
// 或分享给脚本批量拉取时完全不受限。
//
// 判定顺序：
//  1. 命中已知客户端标记 → 是客户端（放行）；
//  2. 空 UA → 非客户端（正常客户端都会带 UA）；
//  3. 命中浏览器标记 → 非客户端；
//  4. 命中自动化/脚本标记 → 非客户端；
//  5. 其它未知 UA → 放行（宁可放行也不误伤没收录的小众客户端）。
func isNonClientUserAgent(userAgent string) bool {
	ua := strings.ToLower(strings.TrimSpace(userAgent))

	// 注意顺序：先判「UA 本身就是链接」。订阅链接里通常含 "clash" 字样
	//（/api/v1/subscriptions/clash/<token>），若先走客户端标记就会把这种请求放行。
	if strings.HasPrefix(ua, "http://") || strings.HasPrefix(ua, "https://") ||
		strings.HasPrefix(ua, "vmess://") || strings.HasPrefix(ua, "vless://") ||
		strings.HasPrefix(ua, "ss://") || strings.HasPrefix(ua, "ssr://") ||
		strings.HasPrefix(ua, "trojan://") {
		return true
	}

	clientMarkers := []string{
		"clash", "stash", "surge", "quantumult", "loon", "sing-box", "singbox",
		"shadowrocket", "v2ray", "v2rayn", "v2rayng", "hiddify", "nekobox",
		"surfboard", "karing", "openclash", "passwall", "moneyfly", "mbox",
		"flclash", "mihomo", "verge", "nekoray", "sfa/", "sfi/", "sfm/",
	}
	for _, marker := range clientMarkers {
		if strings.Contains(ua, marker) {
			return false
		}
	}

	if ua == "" {
		return true
	}

	browserMarkers := []string{
		"chrome/", "crios/", "firefox/", "fxios/", "safari/", "edg/", "edge/",
		"opr/", "opera/", "msie", "trident/", "duckduckgo/", "vivaldi/",
	}
	for _, marker := range browserMarkers {
		if strings.Contains(ua, marker) {
			return true
		}
	}
	if strings.Contains(ua, "mozilla/") && strings.Contains(ua, "applewebkit/") {
		return true
	}

	// 社交/IM 的链接预览爬虫：用户把订阅链接发进聊天窗口，平台自动抓一次。
	// 它们既不该拿到配置，也不该被登记成设备（返回空内容即可）。
	previewMarkers := []string{
		"whatsapp/", "facebookexternalhit", "facebot", "twitterbot", "telegrambot",
		"googlemessages", "dingtalkbot", "wxwork", "micromessenger", "slackbot",
		"discordbot", "linkedinbot", "pinterest", "redditbot", "bingbot",
		"googlebot", "applebot", "vkshare", "skypeuripreview", "zalo", "kakaotalk",
		"viber", "lark", "feishu", "mattermost", "rocket.chat",
	}
	for _, marker := range previewMarkers {
		if strings.Contains(ua, marker) {
			return true
		}
	}

	// 自动化/脚本/抓取/转换站：不是软件客户端，不应该拿到配置
	automationMarkers := []string{
		"go-http-client", "curl/", "wget/", "python-requests", "python-urllib",
		"okhttp", "postmanruntime", "axios/", "node-fetch", "apache-httpclient",
		"libwww-perl", "got (", "got/", "undici", "java/", "java-http-client",
		"subconverter", "clash-subconverter", "sub-web", "node ", "go-resty",
		"insomnia", "httpie", "powershell", "winhttp", "libcurl",
	}
	for _, marker := range automationMarkers {
		if strings.Contains(ua, marker) {
			return true
		}
	}

	return false
}

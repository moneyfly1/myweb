package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// 国内 GitHub 加速镜像（2026-08 实测可用；用于下载解析与同步下载）
var defaultDownloadProxyPrefixes = []string{
	"https://ghfast.top/{url}",
	"https://gh-proxy.com/{url}",
	"https://gh.llkk.cc/{url}",
	"https://gh.ddlc.top/{url}",
	"{url}",
}

func ResolveDownload(c *gin.Context) {
	target := strings.TrimSpace(c.Query("target"))
	if target == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "缺少 target 参数"})
		return
	}

	// 仅允许 http(s) 链接，避免被滥用于任意协议跳转
	if !strings.HasPrefix(target, "https://") && !strings.HasPrefix(target, "http://") {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的下载链接"})
		return
	}

	// 防 SSRF：目标与代理前缀均需通过 URL 合法性校验（协议白名单 + 禁止内网地址）
	if err := validateDownloadURL(target); err != nil {
		utils.LogWarn("ResolveDownload: target 校验失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的下载链接"})
		return
	}

	candidates := buildDownloadCandidates(target, loadDownloadProxyPrefixes())

	// 并行检测所有候选 URL，第一个可达的立即返回
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	resultCh := make(chan string, len(candidates))
	for _, candidate := range candidates {
		go func(url string) {
			if isDownloadURLReachable(url) {
				resultCh <- url
			}
		}(candidate)
	}

	select {
	case url := <-resultCh:
		c.Redirect(http.StatusFound, url)
		return
	case <-ctx.Done():
	}

	// 所有代理不可用时回退原始地址
	c.Redirect(http.StatusFound, target)
}

// validateDownloadURL 校验下载目标 URL：协议白名单 + 解析后主机非内网/回环地址。
// 防止通过 /download/resolve 探测内网服务（SSRF）。
func validateDownloadURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("不支持的协议: %s", parsed.Scheme)
	}
	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL缺少主机名")
	}
	if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
		return fmt.Errorf("禁止访问本地地址")
	}
	ips, err := net.LookupIP(hostname)
	if err == nil {
		for _, ip := range ips {
			if utils.IsPrivateIP(ip) || ip.IsLoopback() {
				return fmt.Errorf("禁止访问内网地址: %s", ip.String())
			}
		}
	}
	return nil
}

func loadDownloadProxyPrefixes() []string {
	db := database.GetDB()
	if db == nil {
		return defaultDownloadProxyPrefixes
	}

	var conf models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "download_proxy_prefixes", "software").First(&conf).Error; err != nil {
		return defaultDownloadProxyPrefixes
	}

	custom := parseProxyPrefixes(conf.Value)
	if len(custom) == 0 {
		return defaultDownloadProxyPrefixes
	}
	return custom
}

func parseProxyPrefixes(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	var parsed []string
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
			return normalizeProxyPrefixes(parsed)
		}
	}

	lines := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == ',' || r == ';'
	})
	return normalizeProxyPrefixes(lines)
}

func normalizeProxyPrefixes(items []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(items)+1)
	for _, item := range items {
		p := strings.TrimSpace(item)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	hasDirect := false
	for _, p := range out {
		if p == "{url}" || p == "direct" || p == "DIRECT" {
			hasDirect = true
			break
		}
	}
	if !hasDirect {
		out = append(out, "{url}")
	}
	return out
}

func buildDownloadCandidates(target string, prefixes []string) []string {
	out := make([]string, 0, len(prefixes))
	seen := make(map[string]struct{})

	for _, p := range prefixes {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		var candidate string
		switch {
		case p == "{url}" || strings.EqualFold(p, "direct"):
			candidate = target
		case strings.Contains(p, "{url}"):
			candidate = strings.ReplaceAll(p, "{url}", target)
		default:
			base := strings.TrimRight(p, "/")
			candidate = base + "/" + target
		}

		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
	}

	if _, ok := seen[target]; !ok {
		out = append(out, target)
	}

	return out
}

func isDownloadURLReachable(url string) bool {
	// 不跟随重定向：重定向目标可能是内网地址（SSRF 跳板），探测只认原始 URL 的直接响应
	client := &http.Client{
		Timeout: 2500 * time.Millisecond,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return true
		}
	}

	req, err = http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Range", "bytes=0-0")
	resp, err = client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

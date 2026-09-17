// Package ghrelease 封装 GitHub Releases API 的版本查询与文件下载，
// 支持项目配置的下载加速前缀（download_proxy_prefixes）。
package ghrelease

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	githubAPIBase = "https://api.github.com"
	userAgent     = "cboard-software-sync"
)

// Asset GitHub Release 附件
type Asset struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"browser_download_url"`
	ContentType string `json:"content_type"`
	UpdatedAt   string `json:"updated_at"`
}

// Release GitHub Release 信息
type Release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Assets  []Asset `json:"assets"`
}

// Version 去掉 tag 前缀 v 的版本号，如 v1.8.1 → 1.8.1
func (r Release) Version() string {
	return strings.TrimPrefix(r.TagName, "v")
}

// Latest 获取仓库最新 Release（自动尝试配置的加速前缀；token 非空时用于提高 API 限额）
func Latest(repo string, proxyPrefixes []string, token string) (*Release, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/releases/latest", githubAPIBase, repo)
	candidates := buildCandidates(apiURL, proxyPrefixes)
	var lastErr error
	for _, candidate := range candidates {
		rel, err := fetchJSON(candidate, token)
		if err != nil {
			lastErr = err
			continue
		}
		return rel, nil
	}
	if lastErr == nil {
		lastErr = errors.New("所有加速前缀均不可用")
	}
	return nil, lastErr
}

// FindAsset 按文件名正则列表匹配附件，返回第一个命中的
func (r Release) FindAsset(patterns []*regexp.Regexp) (*Asset, error) {
	for i := range r.Assets {
		for _, p := range patterns {
			if p.MatchString(r.Assets[i].Name) {
				return &r.Assets[i], nil
			}
		}
	}
	return nil, fmt.Errorf("未找到匹配的安装包（版本 %s）", r.Version())
}

// DefaultProxyPrefixes 与后端 download.go 一致的默认前缀
func DefaultProxyPrefixes() []string {
	return []string{
		"https://ghproxy.com/{url}",
		"https://ghproxy.net/{url}",
		"{url}",
	}
}

func buildCandidates(rawURL string, prefixes []string) []string {
	if len(prefixes) == 0 {
		prefixes = DefaultProxyPrefixes()
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(prefixes))
	for _, p := range prefixes {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var candidate string
		switch {
		case p == "{url}" || strings.EqualFold(p, "direct"):
			candidate = rawURL
		case strings.Contains(p, "{url}"):
			candidate = strings.ReplaceAll(p, "{url}", rawURL)
		default:
			candidate = strings.TrimRight(p, "/") + "/" + rawURL
		}
		if !seen[candidate] {
			seen[candidate] = true
			out = append(out, candidate)
		}
	}
	if !seen[rawURL] {
		out = append(out, rawURL)
	}
	return out
}

func fetchJSON(rawURL, token string) (*Release, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("user-agent", userAgent)
	req.Header.Set("accept", "application/vnd.github.v3+json")
	if token != "" {
		req.Header.Set("authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("GitHub API HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

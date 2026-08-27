// Package ghrelease 封装 GitHub Releases API 的版本查询与文件下载，
// 支持项目配置的下载加速前缀（download_proxy_prefixes）。
package ghrelease

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
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
	Name          string `json:"name"`
	Size          int64  `json:"size"`
	DownloadURL   string `json:"browser_download_url"`
	ContentType   string `json:"content_type"`
	UpdatedAt     string `json:"updated_at"`
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

// Download 下载附件到本地文件（自动尝试加速前缀；token 非空时带上鉴权头）
func Download(asset *Asset, destPath string, proxyPrefixes []string, token string) error {
	candidates := buildCandidates(asset.DownloadURL, proxyPrefixes)
	var lastErr error
	for _, candidate := range candidates {
		if err := downloadFileValidated(candidate, destPath, asset.Size, token); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr == nil {
		lastErr = errors.New("所有下载地址均不可用")
	}
	return fmt.Errorf("下载 %s 失败: %w", asset.Name, lastErr)
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

func downloadFile(rawURL, destPath, token string) error {
	return downloadFileValidated(rawURL, destPath, 0, token)
}

// downloadFileValidated 下载并校验内容：
// - 拒绝代理返回的 HTML 页面（ghproxy 等代理故障时会把 HTML 当 200 返回）
// - expectedSize > 0 时校验实际字节数必须一致（防止截断/错误页被当作安装包上传）
func downloadFileValidated(rawURL, destPath string, expectedSize int64, token string) error {
	client := &http.Client{Timeout: 0} // 大文件不限总超时
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("user-agent", userAgent)
	if token != "" {
		req.Header.Set("authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载 HTTP %d", resp.StatusCode)
	}
	// 内容类型为 HTML 的几乎必然是代理的错误/等待页，直接拒绝换下一个源
	ct := strings.ToLower(resp.Header.Get("content-type"))
	if strings.Contains(ct, "text/html") {
		return fmt.Errorf("下载源返回 HTML 页面而非安装包（%s）", ct)
	}
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if n == 0 {
		return fmt.Errorf("下载内容为空")
	}
	// 再校验首字节不是 HTML（部分代理不设置 Content-Type）
	if b, rerr := os.ReadFile(destPath); rerr == nil {
		head := strings.TrimSpace(string(b[:min(len(b), 64)]))
		low := strings.ToLower(head)
		if strings.HasPrefix(low, "<!doctype") || strings.HasPrefix(low, "<html") {
			return fmt.Errorf("下载内容为 HTML 页面而非安装包")
		}
	}
	if expectedSize > 0 && n != expectedSize {
		return fmt.Errorf("下载大小不匹配: got %d, want %d", n, expectedSize)
	}
	return nil
}

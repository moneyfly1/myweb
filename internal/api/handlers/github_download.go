package handlers

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"cboard-go/internal/services/ghrelease"
	"cboard-go/internal/services/software_sync"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// GitHub Releases 国内镜像直链（方案A）
//
// 用户点击下载 → /download/gh?key=<配置键> → VPS 查 GitHub 最新 Release（30分钟缓存）
// → 匹配安装包资产 → 302 跳转到国内加速镜像（ghfast.top 等，5分钟探测缓存轮换）。
// 用户直接连国内镜像下载，速度快，且不占用 VPS 带宽、无云盘审核。
// ---------------------------------------------------------------------------

const (
	ghReleaseCacheTTL = 30 * time.Minute
	mirrorProbeTTL    = 5 * time.Minute
)

// 公共解析接口全局限流：防止滥用（每次请求会消耗 GitHub API 额度）
var resolveGate = struct {
	mu   sync.Mutex
	last time.Time
}{}

func resolveThrottled() bool {
	resolveGate.mu.Lock()
	defer resolveGate.mu.Unlock()
	if time.Since(resolveGate.last) < 200*time.Millisecond {
		return false
	}
	resolveGate.last = time.Now()
	return true
}

// GitHubResolve 根据软件配置 key 解析 GitHub 最新 Release 资产并 302 到国内镜像直链
func GitHubResolve(c *gin.Context) {
	if !resolveThrottled() {
		utils.ErrorResponse(c, http.StatusTooManyRequests, "请求过于频繁，请稍后再试", nil)
		return
	}
	key := strings.TrimSpace(c.Query("key"))
	if key == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "缺少 key 参数", nil)
		return
	}
	sw := software_sync.FindSoftwareByConfigKey(key)
	if sw == nil {
		utils.ErrorResponse(c, http.StatusNotFound, "未知的软件配置键", nil)
		return
	}
	t := software_sync.FindTarget(key)
	if t == nil {
		utils.ErrorResponse(c, http.StatusNotFound, "未知的软件配置键", nil)
		return
	}

	prefixes := loadDownloadProxyPrefixes()
	release, err := cachedLatestRelease(sw.Repo, prefixes, software_sync.LoadGitHubToken())
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadGateway, "获取 GitHub 版本失败: "+err.Error(), nil)
		return
	}
	asset, aerr := software_sync.FindAssetFor(release, t)
	if aerr != nil {
		utils.ErrorResponse(c, http.StatusNotFound, aerr.Error(), nil)
		return
	}

	dlURL := pickMirrorURL(prefixes, asset.DownloadURL)
	c.Redirect(http.StatusFound, dlURL)
}

// ---------------------------------------------------------------------------
// Release 缓存（30 分钟，避免每个用户点击都打 GitHub API）
// ---------------------------------------------------------------------------

type ghReleaseCacheEntry struct {
	Release *ghrelease.Release
	Expire  time.Time
}

var ghReleaseCache sync.Map // repo → ghReleaseCacheEntry

func cachedLatestRelease(repo string, prefixes []string, token string) (*ghrelease.Release, error) {
	if v, ok := ghReleaseCache.Load(repo); ok {
		e := v.(ghReleaseCacheEntry)
		if time.Now().Before(e.Expire) {
			return e.Release, nil
		}
		ghReleaseCache.Delete(repo)
	}
	rel, err := ghrelease.Latest(repo, prefixes, token)
	if err != nil {
		return nil, err
	}
	ghReleaseCache.Store(repo, ghReleaseCacheEntry{Release: rel, Expire: time.Now().Add(ghReleaseCacheTTL)})
	return rel, nil
}

// ---------------------------------------------------------------------------
// 镜像选择：按配置前缀构造候选地址，逐个探测可用性（5 分钟缓存），全挂则直连 GitHub
// ---------------------------------------------------------------------------

type mirrorProbeEntry struct {
	OK     bool
	Expire time.Time
}

var mirrorProbeCache sync.Map // host → mirrorProbeEntry

func pickMirrorURL(prefixes []string, rawURL string) string {
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
		if probeMirrorOK(candidate) {
			return candidate
		}
	}
	return rawURL // 兜底：直连 GitHub
}

func probeMirrorOK(rawURL string) bool {
	host := urlHost(rawURL)
	if host != "" {
		if v, ok := mirrorProbeCache.Load(host); ok {
			e := v.(mirrorProbeEntry)
			if time.Now().Before(e.Expire) {
				return e.OK
			}
		}
	}
	ok := probeHead(rawURL)
	if host != "" {
		mirrorProbeCache.Store(host, mirrorProbeEntry{OK: ok, Expire: time.Now().Add(mirrorProbeTTL)})
	}
	return ok
}

// probeHead 用 Range 请求探测镜像是否可下载（206/200 且非 HTML 视为可用）
func probeHead(rawURL string) bool {
	client := &http.Client{Timeout: 6 * time.Second}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Range", "bytes=0-0")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 512))
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return false
	}
	ct := strings.ToLower(resp.Header.Get("content-type"))
	if strings.Contains(ct, "text/html") {
		return false
	}
	return true
}

func urlHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

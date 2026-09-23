package handlers

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"cboard-go/internal/services/ghrelease"
	"cboard-go/internal/services/software_sync"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// 自研客户端「最新版本」查询（供 App 的更新检查使用）
//
// 为什么必须有这个接口（2026-09-23 实测结论）：
// App 的更新检查原先直连 api.github.com，国内常被阻断，用户看到「检查更新失败」
// 就永远停在旧版本。而 App 端自己换加速镜像这条路走不通 —— 实测这些镜像
// **只代理 Release 资产，不代理 API**：
//   ghfast.top   api.github.com → 403（资产 → 206 正常）
//   gh.ddlc.top  api.github.com → 404（资产 → 206 正常）
//   gh-proxy.com / gh.llkk.cc   连不上
// 服务器本身能直连 GitHub（实测 0.27s），于是改为「服务器代查、App 只问服务器」：
// App 走自己的域名池（多域名 + 自动轮换）拿到版本号、资产名、体积与 sha256，
// 再去镜像下载 —— 整条链路在国内可用，且 sha256 由可信来源给出，校验不降级。
//
// 与 /download/gh 的分工：那个给浏览器 302 到镜像；这个给 App 结构化元数据。
// ---------------------------------------------------------------------------

// softwareLatestSHA256TTL 校验和缓存时长（与 Release 缓存一致，避免每次点击都拉一遍）
const softwareLatestSHA256TTL = 30 * time.Minute

// 可替换的取数函数（单测注入，避免联网）
var (
	latestReleaseFor = cachedLatestRelease
	releaseTextFetch = fetchReleaseText
)

// SoftwareLatest 返回某个下载入口对应平台的最新安装包元数据
func SoftwareLatest(c *gin.Context) {
	key := strings.TrimSpace(c.Query("key"))
	if key == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "缺少 key 参数", nil)
		return
	}
	sw := software_sync.FindSoftwareByConfigKey(key)
	target := software_sync.FindTarget(key)
	if sw == nil || target == nil {
		utils.ErrorResponse(c, http.StatusNotFound, "未知的软件配置键", nil)
		return
	}

	release, err := latestReleaseFor(sw.Repo, loadDownloadProxyPrefixes(), software_sync.LoadGitHubToken())
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadGateway, "获取 GitHub 版本失败: "+err.Error(), nil)
		return
	}
	asset, aerr := software_sync.FindAssetFor(release, target)
	if aerr != nil {
		utils.ErrorResponse(c, http.StatusNotFound, aerr.Error(), nil)
		return
	}

	// sha256 取不到时返回空串：客户端按「无校验值则跳过校验」处理，不阻塞更新
	sha := cachedAssetSHA256(release, target, asset.Name)

	c.Header("Cache-Control", "public, max-age=300")
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"config_key":   key,
		"version":      release.Version(),
		"tag":          release.TagName,
		"asset_name":   asset.Name,
		"size_bytes":   asset.Size,
		"sha256":       sha,
		"download_url": asset.DownloadURL,
	})
}

// ---------------------------------------------------------------------------
// 校验和：从 Release 里的 SHA256SUMS-*.txt 取（30 分钟内存缓存）
// ---------------------------------------------------------------------------

type sumsCacheEntry struct {
	SHA256 string
	Expire time.Time
}

var sumsCache sync.Map // sums 资产 URL → sumsCacheEntry

// cachedAssetSHA256 返回 assetName 的 sha256（拿不到返回空串）
func cachedAssetSHA256(release *ghrelease.Release, target *software_sync.Target, assetName string) string {
	sums := findSumsAsset(release, target)
	if sums == nil {
		return ""
	}
	cacheKey := sums.DownloadURL + "|" + assetName
	if v, ok := sumsCache.Load(cacheKey); ok {
		if e, ok2 := v.(sumsCacheEntry); ok2 && time.Now().Before(e.Expire) {
			return e.SHA256
		}
		sumsCache.Delete(cacheKey)
	}
	text := releaseTextFetch(sums.DownloadURL)
	if text == "" {
		return ""
	}
	sha := parseSHA256FromSums(text, assetName)
	sumsCache.Store(cacheKey, sumsCacheEntry{SHA256: sha, Expire: time.Now().Add(softwareLatestSHA256TTL)})
	return sha
}

// findSumsAsset 按目标的 SumsAsset 名（如 SHA256SUMS-windows.txt）找校验和资产；
// 找不到再退回发布流程合并生成的 SHA256SUMS.txt（任一环节改名都不该让校验整条失效）
func findSumsAsset(release *ghrelease.Release, target *software_sync.Target) *ghrelease.Asset {
	if release == nil {
		return nil
	}
	names := make([]string, 0, 2)
	if target != nil && target.SumsAsset != "" {
		names = append(names, target.SumsAsset)
	}
	names = append(names, "SHA256SUMS.txt")
	for _, want := range names {
		for i := range release.Assets {
			if strings.EqualFold(release.Assets[i].Name, want) {
				return &release.Assets[i]
			}
		}
	}
	return nil
}

// parseSHA256FromSums 从 sha256sum 输出里取指定文件名的摘要。
// 兼容两种行格式：`<hash>  <name>`（文本模式）与 `<hash> *<name>`（二进制模式，Windows）
func parseSHA256FromSums(text, assetName string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		if !strings.Contains(line, assetName) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		hash := strings.ToLower(fields[0])
		if len(hash) == 64 && isHexString(hash) {
			return hash
		}
	}
	return ""
}

func isHexString(s string) bool {
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// fetchReleaseText 拉取一个小文本资产（校验和文件）。
// 服务器能直连 GitHub；同时按配置的加速前缀依次兜底，任一环节慢/挂都不至于让校验失效。
func fetchReleaseText(rawURL string) string {
	if strings.TrimSpace(rawURL) == "" {
		return ""
	}
	client := &http.Client{Timeout: 8 * time.Second}
	for _, candidate := range buildDownloadCandidates(rawURL, loadDownloadProxyPrefixes()) {
		req, err := http.NewRequest(http.MethodGet, candidate, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "cboard-software-sync")
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		body, rerr := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		resp.Body.Close()
		if rerr != nil || resp.StatusCode != http.StatusOK {
			continue
		}
		if text := strings.TrimSpace(string(body)); text != "" {
			return text
		}
	}
	return ""
}

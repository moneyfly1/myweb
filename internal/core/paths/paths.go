// Package paths 统一解析「运行环境相关的目录与可执行文件」。
//
// 为什么需要它：面板要在不同机器上跑（宝塔 nginx / 系统包 nginx / 自建 nginx、
// 不同的站点根目录、不同的 ACME 客户端），过去这些路径被写死在代码里，导致
// 换机器/换域名后出现「功能静默失效」——例如订阅域名池的一键配置在非宝塔环境下
// 因为找不到 /www/server/panel/vhost/nginx 直接不可用，或建出来的站点没有前端界面。
//
// 解析优先级（从高到低）：
//  1. 环境变量显式指定（写入 .env 即可，重启生效）；
//  2. 已知的常见位置自动探测（必须真实存在）；
//  3. 兜底默认值。
//
// 所有结果都会记录来源（env / detected / default），可通过 Describe() 输出，
// 便于在日志和后台直接看到「这台机器上到底用了哪个路径」。
package paths

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Source 取值来源，用于诊断输出
type Source string

const (
	SourceEnv      Source = "env"      // 环境变量显式指定
	SourceDetected Source = "detected" // 自动探测到存在的路径
	SourceDefault  Source = "default"  // 兜底默认
)

// Item 一条解析结果（供日志/后台展示）
type Item struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Source Source `json:"source"`
	Note   string `json:"note,omitempty"`
}

// 环境变量名（写进 .env 即可覆盖）
const (
	EnvSiteRoot       = "SITE_ROOT"            // 面板站点根目录（含 cboard.db / frontend/dist）
	EnvNginxBin       = "NGINX_BIN"            // nginx 可执行文件
	EnvNginxVhostDir  = "NGINX_VHOST_DIR"      // nginx 站点配置目录（会被 nginx include）
	EnvAcmeWebroot    = "ACME_WEBROOT"         // ACME HTTP-01 校验用的 webroot
	EnvLetsencryptDir = "LETSENCRYPT_LIVE_DIR" // 证书目录
	EnvFrontendDist   = "FRONTEND_DIST"        // 前端构建产物目录
	EnvGeoIPDir       = "GEOIP_DIR"            // GeoIP 数据库目录
)

// 常见位置候选（宝塔 / 系统包 / 源码编译 / Homebrew）
var (
	nginxBinCandidates = []string{
		"/www/server/nginx/sbin/nginx",
		"/usr/sbin/nginx",
		"/usr/local/nginx/sbin/nginx",
		"/usr/local/sbin/nginx",
		"/opt/homebrew/bin/nginx",
		"/opt/local/bin/nginx",
	}
	vhostDirCandidates = []string{
		"/www/server/panel/vhost/nginx", // 宝塔面板
		"/etc/nginx/conf.d",             // 系统包（被 nginx.conf include）
		"/etc/nginx/sites-enabled",      // Debian/Ubuntu 常见
		"/etc/nginx/vhost",              // 自建常见
		"/usr/local/nginx/conf/vhost",   // 源码编译常见
		"/opt/homebrew/etc/nginx/servers",
	}
	geoipDirCandidates = []string{
		"/usr/share/GeoIP",
		"/var/lib/GeoIP",
		"/usr/local/share/GeoIP",
	}
)

var (
	mu     sync.Mutex
	cached map[string]Item
)

// ResetCache 清空缓存（测试用；正常情况下解析结果在进程生命周期内不变）
func ResetCache() {
	mu.Lock()
	defer mu.Unlock()
	cached = nil
}

func envValue(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

// dirExists 目录是否存在
func dirExists(p string) bool {
	if p == "" {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// fileExecutable 文件是否存在且可执行
func fileExecutable(p string) bool {
	if p == "" {
		return false
	}
	st, err := os.Stat(p)
	if err != nil || st.IsDir() {
		return false
	}
	return st.Mode()&0o111 != 0
}

func remember(name string, item Item) Item {
	mu.Lock()
	defer mu.Unlock()
	if cached == nil {
		cached = map[string]Item{}
	}
	cached[name] = item
	return item
}

// SiteRoot 面板站点根目录。
// 默认取进程工作目录（systemd 的 WorkingDirectory，即包含 cboard.db 的目录），
// 这样只要服务的工作目录对，换机器/换目录都不需要改代码。
func SiteRoot() string {
	mu.Lock()
	if it, ok := cached["site_root"]; ok {
		mu.Unlock()
		return it.Value
	}
	mu.Unlock()

	if v := envValue(EnvSiteRoot); v != "" {
		if dirExists(v) {
			return remember("site_root", Item{"site_root", v, SourceEnv, ""}).Value
		}
		// 环境变量写了但目录不存在：不要静默忽略，记录原因后继续探测
		defer func() {
			mu.Lock()
			if cached != nil {
				it := cached["site_root"]
				it.Note = fmt.Sprintf("%s=%s 不存在，已回退自动探测", EnvSiteRoot, v)
				cached["site_root"] = it
			}
			mu.Unlock()
		}()
	}

	if wd, err := os.Getwd(); err == nil && wd != "" {
		return remember("site_root", Item{"site_root", wd, SourceDetected, "进程工作目录"}).Value
	}
	return remember("site_root", Item{"site_root", ".", SourceDefault, "无法获取工作目录"}).Value
}

// NginxBin 定位 nginx 可执行文件
func NginxBin() (string, error) {
	if it, ok := lookup("nginx_bin"); ok {
		if it.Value == "" {
			return "", fmt.Errorf("%s", it.Note)
		}
		return it.Value, nil
	}

	if v := envValue(EnvNginxBin); v != "" {
		if fileExecutable(v) {
			return remember("nginx_bin", Item{"nginx_bin", v, SourceEnv, ""}).Value, nil
		}
	} else {
		for _, c := range nginxBinCandidates {
			if fileExecutable(c) {
				return remember("nginx_bin", Item{"nginx_bin", c, SourceDetected, "常见位置"}).Value, nil
			}
		}
		if p, err := exec.LookPath("nginx"); err == nil {
			return remember("nginx_bin", Item{"nginx_bin", p, SourceDetected, "PATH"}).Value, nil
		}
	}

	tried := strings.Join(nginxBinCandidates, "、")
	note := fmt.Sprintf("未找到 nginx 可执行文件（已尝试 %s）；可在 .env 设置 %s=/path/to/nginx", tried, EnvNginxBin)
	remember("nginx_bin", Item{"nginx_bin", "", SourceDefault, note})
	return "", fmt.Errorf("%s", note)
}

// NginxVhostDir nginx 站点配置目录（必须是被 nginx.conf include 的目录，
// 否则写进去的配置不会生效——这正是「一键配置成功但域名打不开」的根因之一）。
func NginxVhostDir() (string, error) {
	if it, ok := lookup("nginx_vhost_dir"); ok {
		if it.Value == "" {
			return "", fmt.Errorf("%s", it.Note)
		}
		return it.Value, nil
	}

	if v := envValue(EnvNginxVhostDir); v != "" {
		if dirExists(v) {
			return remember("nginx_vhost_dir", Item{"nginx_vhost_dir", v, SourceEnv, ""}).Value, nil
		}
	} else {
		for _, c := range vhostDirCandidates {
			if dirExists(c) {
				return remember("nginx_vhost_dir", Item{"nginx_vhost_dir", c, SourceDetected, "常见位置"}).Value, nil
			}
		}
	}

	note := fmt.Sprintf("未找到 nginx 站点配置目录（已尝试 %s）；可在 .env 设置 %s=/path/to/vhost",
		strings.Join(vhostDirCandidates, "、"), EnvNginxVhostDir)
	remember("nginx_vhost_dir", Item{"nginx_vhost_dir", "", SourceDefault, note})
	return "", fmt.Errorf("%s", note)
}

// AcmeWebroot ACME HTTP-01 校验用的目录：证书签发时把校验文件写在这里，
// nginx 的 /.well-known/acme-challenge/ 指向同一目录即可通过校验。
func AcmeWebroot() string {
	if it, ok := lookup("acme_webroot"); ok {
		return it.Value
	}
	if v := envValue(EnvAcmeWebroot); v != "" && dirExists(v) {
		return remember("acme_webroot", Item{"acme_webroot", v, SourceEnv, ""}).Value
	}
	// 常见历史默认（老部署把 webroot 固定在这里）
	for _, c := range []string{"/www/wwwroot/cboard", "/var/www/html"} {
		if dirExists(c) {
			return remember("acme_webroot", Item{"acme_webroot", c, SourceDetected, "常见位置"}).Value
		}
	}
	root := SiteRoot()
	return remember("acme_webroot", Item{"acme_webroot", root, SourceDefault, "回退到站点根目录"}).Value
}

// LetsencryptLiveDir 证书目录（certbot 默认位置）
func LetsencryptLiveDir() string {
	if it, ok := lookup("letsencrypt_live"); ok {
		return it.Value
	}
	if v := envValue(EnvLetsencryptDir); v != "" {
		return remember("letsencrypt_live", Item{"letsencrypt_live", v, SourceEnv, ""}).Value
	}
	return remember("letsencrypt_live", Item{"letsencrypt_live", "/etc/letsencrypt/live", SourceDefault, ""}).Value
}

// FrontendDist 前端构建产物目录；找不到返回空字符串，调用方据此降级为纯 API 反代。
// siteDomain 传当前站点域名，用于支持 /www/wwwroot/<域名>/frontend/dist 这种布局。
func FrontendDist(siteDomain string) string {
	if it, ok := lookup("frontend_dist"); ok {
		if it.Value == "" {
			return ""
		}
		if siteDomain == "" || it.Note != siteDomain {
			return it.Value
		}
	}
	if v := envValue(EnvFrontendDist); v != "" {
		if hasIndexHTML(v) {
			return remember("frontend_dist", Item{"frontend_dist", v, SourceEnv, siteDomain}).Value
		}
	} else {
		root := SiteRoot()
		candidates := []string{filepath.Join(root, "frontend", "dist")}
		if siteDomain != "" {
			candidates = append(candidates, filepath.Join("/www/wwwroot", siteDomain, "frontend", "dist"))
		}
		candidates = append(candidates,
			"/www/wwwroot/cboard/frontend/dist",
			filepath.Join(root, "dist"),
		)
		for _, c := range candidates {
			if hasIndexHTML(c) {
				return remember("frontend_dist", Item{"frontend_dist", c, SourceDetected, siteDomain}).Value
			}
		}
	}
	note := "未找到前端构建产物（index.html）；可在 .env 设置 " + EnvFrontendDist
	remember("frontend_dist", Item{"frontend_dist", "", SourceDefault, note})
	return ""
}

func hasIndexHTML(dir string) bool {
	if dir == "" {
		return false
	}
	st, err := os.Stat(filepath.Join(dir, "index.html"))
	return err == nil && !st.IsDir()
}

// GeoIPDir GeoIP 数据库目录
func GeoIPDir() (string, error) {
	if it, ok := lookup("geoip_dir"); ok {
		if it.Value == "" {
			return "", fmt.Errorf("%s", it.Note)
		}
		return it.Value, nil
	}
	if v := envValue(EnvGeoIPDir); v != "" {
		if dirExists(v) {
			return remember("geoip_dir", Item{"geoip_dir", v, SourceEnv, ""}).Value, nil
		}
	} else {
		candidates := append([]string{filepath.Join(SiteRoot(), "uploads", "geoip")}, geoipDirCandidates...)
		for _, c := range candidates {
			if dirExists(c) {
				return remember("geoip_dir", Item{"geoip_dir", c, SourceDetected, "常见位置"}).Value, nil
			}
		}
	}
	note := fmt.Sprintf("未找到 GeoIP 目录（已尝试 %s）；可在 .env 设置 %s=/path/to/geoip",
		strings.Join(geoipDirCandidates, "、"), EnvGeoIPDir)
	remember("geoip_dir", Item{"geoip_dir", "", SourceDefault, note})
	return "", fmt.Errorf("%s", note)
}

func lookup(key string) (Item, bool) {
	mu.Lock()
	defer mu.Unlock()
	if cached == nil {
		return Item{}, false
	}
	it, ok := cached[key]
	return it, ok
}

// Describe 输出全部解析结果，用于启动日志与后台展示（不返回错误，探测失败也照实列出）
func Describe(siteDomain string) []Item {
	items := []Item{}
	items = append(items, lookupOrResolve("site_root", func() Item {
		v := SiteRoot()
		it, _ := lookup("site_root")
		it.Value = v
		return it
	}))
	items = append(items, resolveWithErr("nginx_bin", func() (string, error) { return NginxBin() }))
	items = append(items, resolveWithErr("nginx_vhost_dir", func() (string, error) { return NginxVhostDir() }))
	items = append(items, resolveWithErr("geoip_dir", func() (string, error) { return GeoIPDir() }))
	items = append(items, resolveSimple("acme_webroot", AcmeWebroot()))
	items = append(items, resolveSimple("letsencrypt_live", LetsencryptLiveDir()))
	items = append(items, resolveSimple("frontend_dist", FrontendDist(siteDomain)))
	return items
}

// DescribeMap 便于写进日志/接口的简表（name -> value）
func DescribeMap(siteDomain string) map[string]string {
	out := map[string]string{}
	for _, it := range Describe(siteDomain) {
		out[it.Name] = it.Value
	}
	return out
}

// SummaryLine 一行式摘要，启动日志用
func SummaryLine() string {
	var parts []string
	for _, it := range Describe("") {
		v := it.Value
		if v == "" {
			v = "(未找到)"
		}
		parts = append(parts, fmt.Sprintf("%s=%s[%s]", it.Name, v, it.Source))
	}
	return strings.Join(parts, " ")
}

func lookupOrResolve(key string, fn func() Item) Item {
	if it, ok := lookup(key); ok {
		return it
	}
	_ = fn()
	if it, ok := lookup(key); ok {
		return it
	}
	return Item{Name: key}
}

func resolveWithErr(key string, fn func() (string, error)) Item {
	if it, ok := lookup(key); ok {
		return it
	}
	_, _ = fn()
	if it, ok := lookup(key); ok {
		return it
	}
	return Item{Name: key}
}

func resolveSimple(key, value string) Item {
	if it, ok := lookup(key); ok {
		return it
	}
	return Item{Name: key, Value: value}
}

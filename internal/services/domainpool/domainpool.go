// Package domainpool 提供「订阅域名池」的服务器侧自动化：
// 一键给新域名建 nginx 反向代理站点、签发（并自动续期）证书、重载 nginx，
// 并把域名写进面板的订阅域名配置。
//
// 为什么需要（2026-09-23 线上需求）：官网域名在部分地区被屏蔽，需要经常更换
// 订阅域名。手工做一遍要：点 DNS → 写 vhost → 签证书 → 重载 nginx → 改面板配置，
// 任何一步漏了都会导致「换了域名客户还是连不上」。这里把整条链路收敛成一次调用，
// 并逐步返回执行结果，管理员在后台「系统设置 → 订阅域名池」里点一下即可完成。
//
// 安全边界：
//   - 仅当进程以 root 运行、且能找到 nginx 与 certbot 时才启用自动化（否则明确报错，
//     提示手工步骤，不静默失败）；
//   - 域名先做严格格式校验（只允许字母/数字/连字符/点），所有外部命令都用参数数组
//     执行（不拼 shell 字符串），避免命令注入；
//   - 写 vhost 前若已存在同名文件，先备份成 <file>.bak-<时间戳>；
//   - DNS 解析不匹配本机时只警告，不阻断（域名可能刚解析、缓存未生效）。
package domainpool

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

// 面板里存订阅域名的两个配置键（category=general）
const (
	ConfigKeySubscriptionDomain        = "subscription_domain"
	ConfigKeySubscriptionBackupDomains = "subscription_backup_domains"
	ConfigCategory                     = "general"
)

const (
	// BT 面板的 vhost 目录（被 nginx.conf include）
	VhostDir = "/www/server/panel/vhost/nginx"
	// certbot 证书目录
	LetsencryptLiveDir = "/etc/letsencrypt/live"
	// ACME 校验用的 webroot（与手工配置时保持一致）
	DefaultWebroot = "/www/wwwroot/cboard"
	// 面板后端地址（所有订阅域名都反代到这里）
	PanelUpstream = "http://127.0.0.1:8000"

	nginxBinCandidates = "/www/server/nginx/sbin/nginx,/usr/sbin/nginx,/usr/local/nginx/sbin/nginx"
)

// Status 单个域名的检测结果（既用于展示，也用于「添加后自检」）
type Status struct {
	Domain       string `json:"domain"`
	URL          string `json:"url"`
	InPool       bool   `json:"in_pool"`
	IsPrimary    bool   `json:"is_primary"`
	IsSiteDomain bool   `json:"is_site_domain"`
	VhostExists  bool   `json:"vhost_exists"`
	DNSResolved  bool   `json:"dns_resolved"`
	DNSIPs       string `json:"dns_ips,omitempty"`
	ResolvesHere bool   `json:"resolves_here"`
	HTTPSOK      bool   `json:"https_ok"`
	HTTPStatus   int    `json:"http_status,omitempty"`
	CertExists   bool   `json:"cert_exists"`
	CertDaysLeft int    `json:"cert_days_left,omitempty"`
	CertNotAfter string `json:"cert_not_after,omitempty"`
	AutoRenew    bool   `json:"auto_renew"`
	Error        string `json:"error,omitempty"`
}

// Step 一键配置的执行步骤（前端按顺序展示，便于定位卡在哪一步）
type Step struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// Manager 自动化执行器
type Manager struct {
	// 面板站点根目录（含 frontend/dist 的目录），用于新域名的静态资源
	PanelRoot string
	// 已配置的网站域名（不参与删除，且会作为备用订阅地址兜底）
	SiteDomain string
}

func New(panelRoot, siteDomain string) *Manager {
	return &Manager{PanelRoot: panelRoot, SiteDomain: strings.TrimSpace(siteDomain)}
}

// Available 是否具备自动化条件（root + nginx + certbot）
func (m *Manager) Available() (bool, string) {
	if os.Geteuid() != 0 {
		return false, "面板进程不是 root，无法自动写 nginx 配置/签证书（可手工配置，见文档 docs/domain_pool.md）"
	}
	if _, err := m.nginxBin(); err != nil {
		return false, err.Error()
	}
	if _, err := exec.LookPath("certbot"); err != nil {
		return false, "未找到 certbot，无法自动签发证书"
	}
	if st, err := os.Stat(VhostDir); err != nil || !st.IsDir() {
		return false, fmt.Sprintf("vhost 目录不存在：%s", VhostDir)
	}
	return true, ""
}

func (m *Manager) nginxBin() (string, error) {
	for _, p := range strings.Split(nginxBinCandidates, ",") {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	if p, err := exec.LookPath("nginx"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("未找到 nginx 可执行文件（已尝试 %s）", nginxBinCandidates)
}

var domainRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$`)

// ValidateDomain 严格校验域名（只允许小写字母/数字/连字符/点，且必须有至少一个点）
func ValidateDomain(raw string) (string, error) {
	d := strings.ToLower(strings.TrimSpace(raw))
	d = strings.TrimPrefix(d, "https://")
	d = strings.TrimPrefix(d, "http://")
	if i := strings.IndexAny(d, "/?#"); i >= 0 {
		d = d[:i]
	}
	// 去掉端口（证书按域名签发，端口由 nginx 监听决定）
	if i := strings.Index(d, ":"); i >= 0 {
		d = d[:i]
	}
	if d == "" {
		return "", fmt.Errorf("域名不能为空")
	}
	if len(d) > 253 {
		return "", fmt.Errorf("域名过长")
	}
	if net.ParseIP(d) != nil {
		return "", fmt.Errorf("请填写域名（证书不支持直接给 IP 签发）")
	}
	if !domainRe.MatchString(d) {
		return "", fmt.Errorf("域名格式不正确：%s", d)
	}
	return d, nil
}

// CertNameFor 由域名生成 certbot 证书名（域名里的点换成连字符）
func CertNameFor(domain string) string {
	return strings.NewReplacer(".", "-", "/", "-").Replace(strings.ToLower(strings.TrimSpace(domain)))
}

// CertPaths 证书文件路径
func CertPaths(domain string) (fullchain, privkey string) {
	base := filepath.Join(LetsencryptLiveDir, CertNameFor(domain))
	return filepath.Join(base, "fullchain.pem"), filepath.Join(base, "privkey.pem")
}

// VhostPath 该域名的 vhost 文件路径
func VhostPath(domain string) string {
	return filepath.Join(VhostDir, CertNameFor(domain)+".conf")
}

// acmeOnlyVhost 只监听 80、仅用于 ACME 校验的临时站点
// （证书还没签出来时，带 443 ssl 的配置会让 nginx -t 失败，所以先落这个）
func acmeOnlyVhost(domain, webroot string) string {
	return fmt.Sprintf(`# 由面板「订阅域名池」自动生成（ACM+HTTP-01 校验用，配置完成后会被覆盖为完整反代配置）
server {
    listen 80;
    server_name %s;
    location ^~ /.well-known/acme-challenge/ {
        root %s;
        default_type text/plain;
        try_files $uri =404;
    }
    location / { return 301 https://$host$request_uri; }
}
`, domain, webroot)
}

// fullVhost 完整反代配置：80 保留 ACME 校验 + 跳转，443 反代面板并服务前端静态资源
func fullVhost(domain, webroot, staticRoot, certFullchain, certPrivkey string) string {
	staticBlock := ""
	if staticRoot != "" {
		staticBlock = fmt.Sprintf(`    root %s;
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
`, staticRoot)
	}
	return fmt.Sprintf(`# 由面板「订阅域名池」自动生成 —— 与主站同一后端，订阅 token 通用。
# 手工改动会在下次「一键配置」时被覆盖；如需自定义请在面板里调整。
server {
    listen 80;
    server_name %s;
    location ^~ /.well-known/acme-challenge/ {
        root %s;
        default_type text/plain;
        try_files $uri =404;
    }
    location / { return 301 https://$host$request_uri; }
}
server {
    listen 443 ssl;
    server_name %s;
    http2 on;
    ssl_certificate %s;
    ssl_certificate_key %s;
%s    client_max_body_size 64m;

    location /api/ {
        proxy_pass %s;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 300s;
    }
    location /uploads/ {
        proxy_pass %s;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        add_header Cache-Control "no-cache, no-store, must-revalidate";
    }
    location /repo-sync/ {
        proxy_pass %s;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    location /assets/ { expires 1y; add_header Cache-Control "public, immutable"; }
    location / { %s }
}
`, domain, webroot, domain, certFullchain, certPrivkey, staticBlock,
		PanelUpstream, PanelUpstream, PanelUpstream,
		func() string {
			if staticRoot != "" {
				return "try_files $uri $uri/ /index.html;"
			}
			return fmt.Sprintf("proxy_pass %s;\n        proxy_set_header Host $host;\n        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n        proxy_set_header X-Forwarded-Proto $scheme;", PanelUpstream)
		}())
}

// staticRootOf 面板前端静态资源目录（找不到则返回空 = 该域名只做 API 反代）
func (m *Manager) staticRootOf() string {
	candidates := []string{}
	if m.PanelRoot != "" {
		candidates = append(candidates, filepath.Join(m.PanelRoot, "frontend", "dist"))
	}
	candidates = append(candidates,
		"/www/wwwroot/dy.moneyfly.top/frontend/dist",
		"/www/wwwroot/cboard/frontend/dist",
	)
	for _, c := range candidates {
		if st, err := os.Stat(filepath.Join(c, "index.html")); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

func (m *Manager) webroot() string {
	if st, err := os.Stat(DefaultWebroot); err == nil && st.IsDir() {
		return DefaultWebroot
	}
	if m.PanelRoot != "" {
		return m.PanelRoot
	}
	return "/tmp"
}

func runCmd(ctx context.Context, timeout time.Duration, name string, args ...string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, name, args...)
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			text = err.Error()
		}
		return text, fmt.Errorf("%s %s 失败: %s", filepath.Base(name), strings.Join(args, " "), truncate(text, 400))
	}
	return text, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// nginxTestAndReload 配置校验 + 重载；校验失败会把错误原文返回（便于定位哪一行写错）
func (m *Manager) nginxTestAndReload(ctx context.Context) error {
	bin, err := m.nginxBin()
	if err != nil {
		return err
	}
	if out, err := runCmd(ctx, 15*time.Second, bin, "-t"); err != nil {
		return fmt.Errorf("nginx 配置校验失败：%s", out)
	}
	if _, err := runCmd(ctx, 20*time.Second, bin, "-s", "reload"); err != nil {
		return err
	}
	return nil
}

// writeFileWithBackup 写文件（同名文件先备份）
func writeFileWithBackup(path, content string) (string, error) {
	if _, err := os.Stat(path); err == nil {
		bak := fmt.Sprintf("%s.bak-%s", path, time.Now().Format("20060102-150405"))
		if err := os.Rename(path, bak); err != nil {
			return "", fmt.Errorf("备份原配置失败: %v", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// Configure 一键配置一个订阅域名：
//  1. 校验域名 → 2. 检查 DNS 是否指向本机（仅告警）→ 3. 写 ACME 临时站点并重载
//  4. 用 webroot 模式签证书（带 deploy-hook，续期后自动重载 nginx）
//  5. 写完整反代站点并重载 → 6. 访问 https://域名/api/v1/packages 自检
func (m *Manager) Configure(ctx context.Context, rawDomain string) (Status, []Step, error) {
	domain, err := ValidateDomain(rawDomain)
	if err != nil {
		return Status{Domain: rawDomain}, nil, err
	}
	if ok, why := m.Available(); !ok {
		return Status{Domain: domain}, nil, fmt.Errorf("%s", why)
	}

	steps := make([]Step, 0, 8)
	webroot := m.webroot()
	addStep := func(name string, ok bool, detail string) {
		steps = append(steps, Step{Name: name, OK: ok, Detail: detail})
	}

	// DNS 检查（不阻断：解析可能还没生效/有缓存）
	ips, dnsOK := resolveDomain(ctx, domain)
	addStep("检查 DNS 解析", dnsOK, func() string {
		if dnsOK {
			return "解析到 " + strings.Join(ips, ", ")
		}
		return "暂未解析出地址 —— 请先在 DNS 服务商添加 A 记录指向本服务器，再重试"
	}())

	// 1) 临时 ACME 站点（只有 80 端口，证书未签发时也能通过 nginx -t）
	if _, err := writeFileWithBackup(VhostPath(domain), acmeOnlyVhost(domain, webroot)); err != nil {
		addStep("写入站点配置（ACM 校验用）", false, err.Error())
		return Status{Domain: domain}, steps, err
	}
	if err := m.nginxTestAndReload(ctx); err != nil {
		addStep("重载 nginx", false, err.Error())
		return Status{Domain: domain}, steps, err
	}
	addStep("写入站点配置（ACM 校验用）", true, VhostPath(domain))

	// 2) 签证书（webroot 模式：不占用 80 端口，可与 nginx 共存；
	//    --deploy-hook 保证以后自动续期后重载 nginx，新证书真正生效）
	nginxBin, _ := m.nginxBin()
	certbotOut, certErr := runCmd(ctx, 180*time.Second, "certbot", "certonly",
		"--webroot", "-w", webroot,
		"--cert-name", CertNameFor(domain),
		"-d", domain,
		"--non-interactive", "--agree-tos", "--no-eff-email",
		"--deploy-hook", nginxBin+" -s reload",
	)
	if certErr != nil {
		addStep("签发证书", false, certbotOut)
		return Status{Domain: domain}, steps, certErr
	}
	if _, err := os.Stat(filepath.Join(LetsencryptLiveDir, CertNameFor(domain), "fullchain.pem")); err != nil {
		addStep("签发证书", false, "certbot 未报错但未找到证书文件")
		return Status{Domain: domain}, steps, fmt.Errorf("证书文件不存在")
	}
	addStep("签发证书（含自动续期钩子）", true, fmt.Sprintf("%s（证书名 %s）", domain, CertNameFor(domain)))

	// 3) 完整反代站点
	full, key := CertPaths(domain)
	staticRoot := m.staticRootOf()
	if _, err := writeFileWithBackup(VhostPath(domain), fullVhost(domain, webroot, staticRoot, full, key)); err != nil {
		addStep("写入站点配置（反代）", false, err.Error())
		return Status{Domain: domain}, steps, err
	}
	if err := m.nginxTestAndReload(ctx); err != nil {
		addStep("重载 nginx（反代）", false, err.Error())
		return Status{Domain: domain}, steps, err
	}
	addStep("写入站点配置（反代）", true, func() string {
		if staticRoot == "" {
			return "已反代接口（未找到前端静态目录，该域名仅提供 API/订阅）"
		}
		return "已反代接口 + 前端静态资源（" + staticRoot + "）"
	}())

	// 4) 自检
	st := m.Inspect(ctx, domain, false, false)
	if st.HTTPSOK {
		addStep("自检 https 访问", true, fmt.Sprintf("HTTP %d", st.HTTPStatus))
	} else {
		addStep("自检 https 访问", false, st.Error)
	}
	return st, steps, nil
}

// RemoveVhost 删除该域名的 vhost（不动证书，便于以后换回来）
func (m *Manager) RemoveVhost(ctx context.Context, rawDomain string) ([]Step, error) {
	domain, err := ValidateDomain(rawDomain)
	if err != nil {
		return nil, err
	}
	if m.SiteDomain != "" && domain == strings.ToLower(m.SiteDomain) {
		return nil, fmt.Errorf("%s 是网站域名，不能从这里移除", domain)
	}
	var steps []Step
	path := VhostPath(domain)
	if _, err := os.Stat(path); err == nil {
		bak := fmt.Sprintf("%s.bak-%s", path, time.Now().Format("20060102-150405"))
		if err := os.Rename(path, bak); err != nil {
			return nil, fmt.Errorf("备份配置失败: %v", err)
		}
		steps = append(steps, Step{Name: "移除站点配置", OK: true, Detail: "已备份到 " + bak})
	} else {
		steps = append(steps, Step{Name: "移除站点配置", OK: true, Detail: "该域名本来就没有站点配置"})
	}
	if err := m.nginxTestAndReload(ctx); err != nil {
		steps = append(steps, Step{Name: "重载 nginx", OK: false, Detail: err.Error()})
		return steps, err
	}
	steps = append(steps, Step{Name: "重载 nginx", OK: true, Detail: "已生效"})
	return steps, nil
}

// Inspect 汇总单个域名的检测结果
func (m *Manager) Inspect(ctx context.Context, domain string, inPool, isPrimary bool) Status {
	d, err := ValidateDomain(domain)
	if err != nil {
		return Status{Domain: domain, Error: err.Error()}
	}
	st := Status{
		Domain:       d,
		URL:          "https://" + d,
		InPool:       inPool,
		IsPrimary:    isPrimary,
		IsSiteDomain: m.SiteDomain != "" && d == strings.ToLower(m.SiteDomain),
	}
	if _, err := os.Stat(VhostPath(d)); err == nil {
		st.VhostExists = true
	}
	if ips, ok := resolveDomain(ctx, d); ok {
		st.DNSResolved = true
		st.DNSIPs = strings.Join(ips, ", ")
	}
	full, _ := CertPaths(d)
	if data, err := os.ReadFile(full); err == nil {
		st.CertExists = true
		if notAfter, derr := certNotAfter(data); derr == nil {
			st.CertNotAfter = notAfter.Format("2006-01-02 15:04")
			st.CertDaysLeft = int(time.Until(notAfter).Hours() / 24)
		}
	}
	if _, err := os.Stat(filepath.Join("/etc/letsencrypt/renewal", CertNameFor(d)+".conf")); err == nil {
		st.AutoRenew = true
	}

	// HTTPS 自检：直接打 /api/v1/packages（面板公开接口，返回 JSON 即说明反代正确）
	client := &http.Client{
		Timeout: 12 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+d+"/api/v1/packages", nil)
	req.Header.Set("User-Agent", "MoneyFly-panel-domainpool/1.0")
	resp, rerr := client.Do(req)
	if rerr != nil {
		st.Error = rerr.Error()
		return st
	}
	defer resp.Body.Close()
	st.HTTPStatus = resp.StatusCode
	st.HTTPSOK = resp.StatusCode >= 200 && resp.StatusCode < 400
	if !st.HTTPSOK {
		st.Error = fmt.Sprintf("HTTPS 返回 %d", resp.StatusCode)
	}
	return st
}

// InspectMany 批量检测（结果按传入顺序返回）
func (m *Manager) InspectMany(ctx context.Context, domains []string, primary, siteDomain string) []Status {
	out := make([]Status, 0, len(domains))
	for _, d := range domains {
		out = append(out, m.Inspect(ctx, d, true, strings.EqualFold(d, primary)))
	}
	return out
}

// certNotAfter 读取证书到期时间（解析 PEM 里的第一张证书）
func certNotAfter(pemData []byte) (time.Time, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return time.Time{}, fmt.Errorf("证书内容无法解析")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, err
	}
	return cert.NotAfter, nil
}

// ResolveIPs 解析域名（供 DNS 提示用）
func resolveDomain(ctx context.Context, domain string) ([]string, bool) {
	cctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupHost(cctx, domain)
	if err != nil || len(addrs) == 0 {
		return nil, false
	}
	uniq := map[string]bool{}
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		if !uniq[a] {
			uniq[a] = true
			out = append(out, a)
		}
	}
	sort.Strings(out)
	return out, true
}

// HostOf 从「域名 / https://域名 / https://域名/api/v1」里取出主机名
func HostOf(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return ""
	}
	if i := strings.Index(v, "://"); i >= 0 {
		v = v[i+3:]
	}
	if i := strings.IndexAny(v, "/?#"); i >= 0 {
		v = v[:i]
	}
	if i := strings.Index(v, ":"); i >= 0 {
		v = v[:i]
	}
	return strings.ToLower(strings.TrimSpace(v))
}

// PoolFromConfig 读取面板配置里的订阅域名池：主域名（已取主机名）+ 备用域名（去重、不含主域名）
func PoolFromConfig(db *gorm.DB) (primary string, backups []string) {
	primary = HostOf(utils.SubscriptionBaseURL(nil, db))
	raw := ""
	if db != nil {
		var cfg models.SystemConfig
		if err := db.Where("key = ? AND category = ?", ConfigKeySubscriptionBackupDomains, ConfigCategory).
			First(&cfg).Error; err == nil {
			raw = cfg.Value
		}
	}
	seen := map[string]bool{}
	if primary != "" {
		seen[primary] = true
	}
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == ';' || r == ' ' || r == '\t' || r == '\r'
	}) {
		h := HostOf(part)
		if h == "" || seen[h] {
			continue
		}
		seen[h] = true
		backups = append(backups, h)
	}
	return primary, backups
}

// WritePool 写回订阅域名池（主域名 + 备用域名列表）
func WritePool(db *gorm.DB, primary string, backups []string) error {
	if db == nil {
		return fmt.Errorf("数据库不可用")
	}
	if p := HostOf(primary); p != "" {
		if err := upsertConfigValue(db, ConfigKeySubscriptionDomain, "https://"+p); err != nil {
			return err
		}
	}
	clean := make([]string, 0, len(backups))
	seen := map[string]bool{}
	for _, b := range backups {
		h := HostOf(b)
		if h == "" || seen[h] || h == HostOf(primary) {
			continue
		}
		seen[h] = true
		clean = append(clean, h)
	}
	return upsertConfigValue(db, ConfigKeySubscriptionBackupDomains, strings.Join(clean, "\n"))
}

// upsertConfigValue 写/更新一条系统配置（存在则更新 value，不存在则创建）
func upsertConfigValue(db *gorm.DB, key, value string) error {
	var cfg models.SystemConfig
	err := db.Where("key = ? AND category = ?", key, ConfigCategory).First(&cfg).Error
	if err == gorm.ErrRecordNotFound {
		return db.Create(&models.SystemConfig{
			Key: key, Value: value, Type: "string",
			Category: ConfigCategory, DisplayName: key,
		}).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&models.SystemConfig{}).Where("id = ?", cfg.ID).
		Updates(map[string]interface{}{"value": value, "updated_at": time.Now()}).Error
}

// DomainsForInspect 需要体检的域名：主域名 → 备用 → 网站域名（同一后端，可作兜底）
func DomainsForInspect(primary string, backups []string, siteDomain string) []string {
	out := make([]string, 0, len(backups)+2)
	seen := map[string]bool{}
	add := func(d string) {
		h := HostOf(d)
		if h == "" || seen[h] {
			return
		}
		seen[h] = true
		out = append(out, h)
	}
	add(primary)
	for _, b := range backups {
		add(b)
	}
	add(siteDomain)
	return out
}

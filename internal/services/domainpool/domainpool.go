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
	CertName     string `json:"cert_name,omitempty"`
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

// VhostPath 新建 vhost 时的默认文件路径（域名原样 + .conf，与面板里手工建的保持一致）
func VhostPath(domain string) string {
	return filepath.Join(VhostDir, domain+".conf")
}

// FindVhostFile 找出该域名**当前实际生效**的 vhost 文件。
//
// 为什么不能直接用 VhostPath：历史配置文件命名并不统一（例如 sub.moneyfly.dpdns.org
// 用的是 cboard_sub.conf），按固定文件名找会误判「站点不存在」，一键配置还会再写一个
// 同 server_name 的文件，造成 nginx "conflicting server name" 警告、实际生效的可能还是旧文件。
// 这里按 server_name 扫描，命中就原地改那个文件（并保留备份）。
func FindVhostFile(domain string) string {
	entries, err := os.ReadDir(VhostDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".conf") {
				continue
			}
			full := filepath.Join(VhostDir, e.Name())
			data, rerr := os.ReadFile(full)
			if rerr != nil {
				continue
			}
			if serverNameMatches(string(data), domain) {
				return full
			}
		}
	}
	return VhostPath(domain)
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

// RenewResult 手动续期结果（后台「立即续期」按钮用）
type RenewResult struct {
	Renewed  []string `json:"renewed"`  // 本次真正续期的证书名
	NotDue   []string `json:"not_due"`  // 未到期、被 certbot 跳过的证书名
	Failed   []string `json:"failed"`   // 续期失败的证书名
	Reloaded bool     `json:"reloaded"` // 是否已重载 nginx
}

// Renew 手动续期池内域名涉及的证书。
//
// 为什么按 --cert-name 逐个续而不是直接 `certbot renew`：本机可能有与域名池无关的
// 站点证书（例如别人的站），逐个点名只动我们管的那几张，避免误伤。
//
// force=true 用 --force-renewal（未到期也重签）。注意 Let's Encrypt 对「同一组域名
// 重复签发」有每周次数限制，仅在证书确实异常时才用。
//
// 续期成功后重载**真正在跑的** nginx（宝塔与 apt 版路径不同，见 nginxBin）：
// certbot 的 --deploy-hook 只在真发生续期时触发，这里再兜一次，保证新证书被加载。
func (m *Manager) Renew(ctx context.Context, domains []string, force bool) (RenewResult, []Step, error) {
	var res RenewResult
	steps := make([]Step, 0, len(domains)+3)
	addStep := func(name string, ok bool, detail string) {
		steps = append(steps, Step{Name: name, OK: ok, Detail: detail})
	}

	bin, err := m.nginxBin()
	if err != nil {
		addStep("定位 nginx", false, err.Error())
		return res, steps, err
	}
	if _, err := exec.LookPath("certbot"); err != nil {
		err := fmt.Errorf("未找到 certbot，无法续期（apt install certbot 或面板一键安装）")
		addStep("检查 certbot", false, err.Error())
		return res, steps, err
	}

	// 收集涉及的证书名（同一张证书可能覆盖多个域名 → 去重，只续一次）
	nameSet := map[string]bool{}
	for _, d := range domains {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		info, ok := FindCertCovering(d)
		if !ok {
			addStep("查找证书 "+d, false, "未找到覆盖该域名的证书，请先点「一键修复」")
			continue
		}
		nameSet[info.Name] = true
	}
	if len(nameSet) == 0 {
		err := fmt.Errorf("没有找到可续期的证书")
		return res, steps, err
	}
	names := make([]string, 0, len(nameSet))
	for n := range nameSet {
		names = append(names, n)
	}
	sort.Strings(names)
	addStep("查找证书", true, strings.Join(names, "、"))

	for _, name := range names {
		args := []string{"renew", "--cert-name", name, "--deploy-hook", bin + " -s reload"}
		if force {
			args = append(args, "--force-renewal")
		}
		out, rerr := runCmd(ctx, 240*time.Second, "certbot", args...)
		if rerr != nil {
			res.Failed = append(res.Failed, name)
			addStep("续期 "+name, false, truncate(out, 300))
			continue
		}
		if !force && (strings.Contains(out, "not yet due") || strings.Contains(out, "not due for renewal")) {
			res.NotDue = append(res.NotDue, name)
			addStep("续期 "+name, true, "未到期，跳过（到期前 30 天自动续期）")
			continue
		}
		res.Renewed = append(res.Renewed, name)
		addStep("续期 "+name, true, truncate(strings.TrimSpace(out), 200))
	}

	// 再重载一次：一是 certbot 未真正续期时也确认 nginx 状态正常，二是确保新证书生效
	if err := m.nginxTestAndReload(ctx); err != nil {
		addStep("重载 nginx", false, err.Error())
		return res, steps, err
	}
	res.Reloaded = true
	addStep("重载 nginx", true, bin+" -s reload")

	if len(res.Failed) > 0 {
		return res, steps, fmt.Errorf("%d 张证书续期失败：%s", len(res.Failed), strings.Join(res.Failed, "、"))
	}
	return res, steps, nil
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
		bak := fmt.Sprintf("%s.bak-%s", path, time.Now().Format("20060102-150405.000"))
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

	// 1) 证书：已有覆盖该域名且剩余 >30 天的证书就直接复用（常见于共享证书/SAN 覆盖），
	//    避免重复签发 —— Let's Encrypt 有「同域名每周 5 张」的限流，撞上就得等一周。
	nginxBin, _ := m.nginxBin()
	full, key := "", ""
	if info, ok := FindCertCovering(domain); ok && info.DaysLeft > 30 {
		full, key = info.Fullchain, info.Privkey
		addStep("复用已有证书", true, fmt.Sprintf("证书 %s 覆盖该域名，剩余 %d 天，无需重新签发", info.Name, info.DaysLeft))
	}
	if full == "" {
		// 需要签发新证书：先落一个只监听 80 的临时站点用于 HTTP-01 校验
		// （证书还不存在时，带 443 ssl 的配置会让 nginx -t 失败）
		if _, err := writeFileWithBackup(FindVhostFile(domain), acmeOnlyVhost(domain, webroot)); err != nil {
			addStep("写入站点配置（ACM 校验用）", false, err.Error())
			return Status{Domain: domain}, steps, err
		}
		if err := m.nginxTestAndReload(ctx); err != nil {
			addStep("重载 nginx", false, err.Error())
			return Status{Domain: domain}, steps, err
		}
		addStep("写入站点配置（ACM 校验用）", true, FindVhostFile(domain))
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
		certFull, certKey := CertPaths(domain)
		if _, err := os.Stat(certFull); err != nil {
			addStep("签发证书", false, "certbot 未报错但未找到证书文件")
			return Status{Domain: domain}, steps, fmt.Errorf("证书文件不存在")
		}
		full, key = certFull, certKey
		addStep("签发证书（含自动续期钩子）", true, fmt.Sprintf("%s（证书名 %s）", domain, CertNameFor(domain)))
	} // end if full == ""

	// 3) 完整反代站点
	staticRoot := m.staticRootOf()
	if _, err := writeFileWithBackup(FindVhostFile(domain), fullVhost(domain, webroot, staticRoot, full, key)); err != nil {
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
	path := FindVhostFile(domain)
	if _, err := os.Stat(path); err == nil {
		bak := fmt.Sprintf("%s.bak-%s", path, time.Now().Format("20060102-150405.000"))
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
	if _, err := os.Stat(FindVhostFile(d)); err == nil {
		st.VhostExists = true
	}
	if ips, ok := resolveDomain(ctx, d); ok {
		st.DNSResolved = true
		st.DNSIPs = strings.Join(ips, ", ")
	}
	if info, ok := FindCertCovering(d); ok {
		st.CertExists = true
		st.CertName = info.Name
		st.CertDaysLeft = info.DaysLeft
		st.CertNotAfter = time.Now().Add(time.Duration(info.DaysLeft) * 24 * time.Hour).Format("2006-01-02 15:04")
		st.AutoRenew = info.AutoRenew
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

// serverNameMatches 判断配置内容里的 server_name 是否**精确**包含该域名。
//
// 必须按 token 精确比较，不能用子串包含：`server_name sub.moneyfly.dpdns.org;`
// 用子串判断会命中 `moneyfly.dpdns.org` —— 线上真实事故：给 moneyfly.dpdns.org
// 做一键配置时，误把 sub.moneyfly.dpdns.org 的站点配置覆盖掉了（订阅域名直接 404）。
func serverNameMatches(content, domain string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return false
	}
	for _, line := range strings.Split(strings.ToLower(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "server_name ") {
			continue
		}
		value := strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(trimmed, "server_name ")), ";")
		for _, token := range strings.Fields(value) {
			if token == domain {
				return true
			}
		}
	}
	return false
}

// CertInfo 覆盖某域名的证书信息
type CertInfo struct {
	Name        string
	Fullchain   string
	Privkey     string
	DaysLeft    int
	AutoRenew   bool
	IsExactName bool // 证书名就是按该域名生成的（而非共享证书/SAN 覆盖）
}

// FindCertCovering 找出覆盖该域名的证书：优先同名证书，否则扫描所有证书看 SAN/CN 是否包含它。
//
// 为什么不能只用 CertNameFor(domain) 直接拼路径：线上一张证书常常覆盖多个域名
// （例如 sub-moneyfly-dpdns-org 同时覆盖 sub.* / moneyfly.dpdns.org / new.*），
// 而 dy.moneyfly.top 的证书名又带点（dy.moneyfly.top）—— 直接拼路径会误报「无证书」，
// 一键配置还会重复签发同一域名（浪费 ACME 次数、可能撞 Let's Encrypt 限流）。
func FindCertCovering(domain string) (CertInfo, bool) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	entries, err := os.ReadDir(LetsencryptLiveDir)
	if err != nil {
		return CertInfo{}, false
	}
	build := func(name string) (CertInfo, bool) {
		full := filepath.Join(LetsencryptLiveDir, name, "fullchain.pem")
		data, rerr := os.ReadFile(full)
		if rerr != nil {
			return CertInfo{}, false
		}
		notAfter, perr := certNotAfter(data)
		if perr != nil {
			return CertInfo{}, false
		}
		info := CertInfo{
			Name:        name,
			Fullchain:   full,
			Privkey:     filepath.Join(LetsencryptLiveDir, name, "privkey.pem"),
			DaysLeft:    int(time.Until(notAfter).Hours() / 24),
			IsExactName: name == CertNameFor(domain),
		}
		if _, serr := os.Stat(filepath.Join("/etc/letsencrypt/renewal", name+".conf")); serr == nil {
			info.AutoRenew = true
		}
		return info, true
	}

	// ① 同名证书
	if info, ok := build(CertNameFor(domain)); ok {
		return info, true
	}
	// ② 扫描：证书名与目录名都可能是带点的域名形式
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() || name == "README" {
			continue
		}
		full := filepath.Join(LetsencryptLiveDir, name, "fullchain.pem")
		data, rerr := os.ReadFile(full)
		if rerr != nil {
			continue
		}
		if !certCoversDomain(data, domain) {
			continue
		}
		if info, ok := build(name); ok {
			return info, true
		}
	}
	return CertInfo{}, false
}

// certCoversDomain 证书 SAN（或 CN）是否包含该域名
func certCoversDomain(pemData []byte, domain string) bool {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return false
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false
	}
	if strings.EqualFold(cert.Subject.CommonName, domain) {
		return true
	}
	for _, n := range cert.DNSNames {
		if strings.EqualFold(n, domain) {
			return true
		}
	}
	return false
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

package domainpool

import (
	"strings"
	"testing"
)

func TestValidateDomain(t *testing.T) {
	ok := []struct {
		in   string
		want string
	}{
		{"sub.moneyfly.dpdns.org", "sub.moneyfly.dpdns.org"},
		{"  Dy.Moneyfly.Top  ", "dy.moneyfly.top"},
		{"https://new.moneyfly.dpdns.org", "new.moneyfly.dpdns.org"},
		{"https://moneyfly.dpdns.org/api/v1", "moneyfly.dpdns.org"},
		{"http://a-b.example.com:8443/x?y=1", "a-b.example.com"},
	}
	for _, tc := range ok {
		got, err := ValidateDomain(tc.in)
		if err != nil {
			t.Errorf("ValidateDomain(%q) 报错: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ValidateDomain(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	bad := []string{
		"",
		"localhost",      // 没有点
		"1.2.3.4",        // IP：证书不能直接给 IP 签
		"bad_domain.com", // 下划线
		"-leading.example.com",
		"trailing-.com",
		"a..b.com",
		"example.com;rm -rf /", // 命令注入尝试
		"example.com && whoami",
	}
	for _, in := range bad {
		if got, err := ValidateDomain(in); err == nil {
			t.Errorf("ValidateDomain(%q) 应该报错，却返回 %q", in, got)
		}
	}
}

func TestCertNameAndPaths(t *testing.T) {
	if got := CertNameFor("sub.moneyfly.dpdns.org"); got != "sub-moneyfly-dpdns-org" {
		t.Errorf("CertNameFor = %q", got)
	}
	full, key := CertPaths("sub.moneyfly.dpdns.org")
	if !strings.HasSuffix(full, "/sub-moneyfly-dpdns-org/fullchain.pem") {
		t.Errorf("fullchain 路径异常: %s", full)
	}
	if !strings.HasSuffix(key, "/sub-moneyfly-dpdns-org/privkey.pem") {
		t.Errorf("privkey 路径异常: %s", key)
	}
	// 新建 vhost 用「域名.conf」（与面板里手工建的命名一致）；已有配置由 FindVhostFile 定位
	if !strings.HasSuffix(VhostPath("sub.moneyfly.dpdns.org"), "/sub.moneyfly.dpdns.org.conf") {
		t.Errorf("vhost 路径异常: %s", VhostPath("sub.moneyfly.dpdns.org"))
	}
}

// ACME 临时配置：只有 80 端口（证书没签出来时带 443 ssl 会让 nginx -t 失败）
func TestAcmeOnlyVhost(t *testing.T) {
	cfg := acmeOnlyVhost("sub.example.com", "/www/wwwroot/cboard")
	if !strings.Contains(cfg, "listen 80;") {
		t.Error("缺少 80 监听")
	}
	if strings.Contains(cfg, "listen 443") {
		t.Error("ACME 临时配置不应包含 443（证书未签发时会导致 nginx 校验失败）")
	}
	if !strings.Contains(cfg, "server_name sub.example.com;") {
		t.Error("缺少 server_name")
	}
	if !strings.Contains(cfg, "/.well-known/acme-challenge/") {
		t.Error("缺少 ACME 校验路径")
	}
	if !strings.Contains(cfg, "root /www/wwwroot/cboard;") {
		t.Error("ACME 校验目录应指向 webroot")
	}
}

// 完整配置：80 保留 ACME + 跳转；443 反代面板并可选服务前端静态资源
func TestFullVhost(t *testing.T) {
	cfg := fullVhost("sub.example.com", "/www/wwwroot/cboard",
		"/www/wwwroot/panel/frontend/dist",
		"/etc/letsencrypt/live/sub-example-com/fullchain.pem",
		"/etc/letsencrypt/live/sub-example-com/privkey.pem")

	for _, want := range []string{
		"listen 80;",
		"listen 443 ssl;",
		"server_name sub.example.com;",
		"/.well-known/acme-challenge/",
		"ssl_certificate /etc/letsencrypt/live/sub-example-com/fullchain.pem;",
		"ssl_certificate_key /etc/letsencrypt/live/sub-example-com/privkey.pem;",
		"proxy_pass http://127.0.0.1:8000;",
		"root /www/wwwroot/panel/frontend/dist;",
		"try_files $uri $uri/ /index.html;",
		"location /api/",
	} {
		if !strings.Contains(cfg, want) {
			t.Errorf("完整配置缺少: %s\n%s", want, cfg)
		}
	}
}

// 找不到前端静态目录时：入口也走反代（该域名只提供 API/订阅，也不会 404）
func TestFullVhostWithoutStaticRoot(t *testing.T) {
	cfg := fullVhost("sub.example.com", "/tmp", "",
		"/etc/letsencrypt/live/x/fullchain.pem", "/etc/letsencrypt/live/x/privkey.pem")
	if strings.Contains(cfg, "try_files $uri $uri/ /index.html;") {
		t.Error("没有静态目录时不应出现 SPA fallback（会返回 404）")
	}
	if !strings.Contains(cfg, "location / {") {
		t.Error("缺少默认 location")
	}
	if strings.Count(cfg, "proxy_pass http://127.0.0.1:8000;") < 4 {
		t.Error("入口也应反代到面板后端")
	}
}

// 按 server_name 定位 vhost：历史文件命名不统一（cboard_sub.conf 之类）时也要能找到，
// 否则一键配置会另写一个同 server_name 的文件，nginx 会告警且实际生效的可能还是旧文件。
func TestFindVhostFileFallsBackToConvention(t *testing.T) {
	// 测试环境里没有 /www/server/panel/vhost/nginx，应回退到「域名.conf」
	got := FindVhostFile("sub.example.com")
	if !strings.HasSuffix(got, "/sub.example.com.conf") {
		t.Errorf("回退路径异常: %s", got)
	}
	if !strings.HasSuffix(VhostPath("sub.example.com"), "/sub.example.com.conf") {
		t.Errorf("默认路径应为 域名.conf: %s", VhostPath("sub.example.com"))
	}
}

// 关键回归（线上事故）：server_name 必须**精确**匹配。
// `server_name sub.moneyfly.dpdns.org;` 用子串包含会命中 moneyfly.dpdns.org，
// 导致给 moneyfly.dpdns.org 做一键配置时改写了 sub.* 的站点配置（订阅域名直接 404）。
func TestServerNameMatchesIsExact(t *testing.T) {
	subConf := "server {\n    listen 443 ssl;\n    server_name sub.moneyfly.dpdns.org;\n}\n"
	if serverNameMatches(subConf, "moneyfly.dpdns.org") {
		t.Error("子域名站点不应匹配到父域名（会误改配置）")
	}
	if serverNameMatches(subConf, "sub.moneyfly.dpdns.org") {
		// 应该匹配（正向用例放下面断言）
	} else {
		t.Error("精确同名应匹配")
	}

	multi := "    server_name moneyfly.dpdns.org dy.moneyfly.club moneyfly.eu.org;\n"
	if !serverNameMatches(multi, "dy.moneyfly.club") {
		t.Error("多域名 server_name 里的成员应匹配")
	}
	if serverNameMatches(multi, "moneyfly.dpdns.org.evil.com") {
		t.Error("后缀不同的域名不应匹配")
	}
	if serverNameMatches("server_name other.com;", "moneyfly.dpdns.org") {
		t.Error("无关域名不应匹配")
	}
	if serverNameMatches("", "a.com") {
		t.Error("空内容不应匹配")
	}
	// 注释里的 server_name 不应参与匹配
	if serverNameMatches("# server_name a.com;\nserver_name b.com;", "a.com") {
		t.Error("注释里的 server_name 不应匹配")
	}
}

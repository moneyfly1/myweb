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
	if !strings.HasSuffix(VhostPath("sub.moneyfly.dpdns.org"), "/sub-moneyfly-dpdns-org.conf") {
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

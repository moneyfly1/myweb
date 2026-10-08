package domainpool

import (
	"strings"
	"testing"
)

// 回归：全新服务器上 certbot 没有 ACME 账户，首次签发必须带 --email，
// 否则 certbot 报 MissingCommandlineFlag，一键配置会卡在「签发证书」这一步
// （DNS 校验和写站点配置都成功，管理员看到的现象就是「点了没作用」）。
func TestCertbotIssueArgsIncludesEmail(t *testing.T) {
	args := certbotIssueArgs("speedora.top", "/www/wwwroot/cboard", "speedora-top", "admin@example.com", "/usr/sbin/nginx -s reload")
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "--email admin@example.com") {
		t.Fatalf("缺少 --email，新服务器会注册失败：%v", args)
	}
	if !strings.Contains(joined, "--agree-tos") || !strings.Contains(joined, "--non-interactive") {
		t.Fatalf("缺少非交互同意条款参数：%v", args)
	}
	if !strings.Contains(joined, "certonly") || !strings.Contains(joined, "-d speedora.top") {
		t.Fatalf("签发参数不完整：%v", args)
	}
	if !strings.Contains(joined, "--deploy-hook") {
		t.Fatalf("缺少续期后重载 nginx 的钩子：%v", args)
	}
}

// 邮箱为空时不应塞一个空的 --email（会让 certbot 直接报错），保持旧行为，
// 依赖服务器上已注册的账户（老服务器就是这种情况）。
func TestCertbotIssueArgsOmitsEmptyEmail(t *testing.T) {
	for _, empty := range []string{"", "   "} {
		args := certbotIssueArgs("a.example.com", "/www/wwwroot/cboard", "a-example-com", empty, "")
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "--email") {
			t.Fatalf("邮箱为空时不该出现 --email：%v", args)
		}
		if strings.Contains(joined, "--deploy-hook") {
			t.Fatalf("deploy-hook 为空时不该出现该参数：%v", args)
		}
	}
}

// 邮箱两端空白要清掉（DB 里的配置值经常带空格/换行）。
func TestCertbotIssueArgsTrimsEmail(t *testing.T) {
	args := certbotIssueArgs("a.example.com", "/w", "n", "  admin@example.com\n", "")
	found := false
	for i, a := range args {
		if a == "--email" && i+1 < len(args) && args[i+1] == "admin@example.com" {
			found = true
		}
	}
	if !found {
		t.Fatalf("邮箱未 trim：%v", args)
	}
}

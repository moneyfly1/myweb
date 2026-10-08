package paths

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resetAll() { ResetCache() }

// 环境变量指定且目录存在时，必须优先于自动探测。
func TestEnvOverrideWins(t *testing.T) {
	root := t.TempDir()
	dist := filepath.Join(root, "frontend", "dist")
	if err := os.MkdirAll(dist, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dist, "index.html"), "<html></html>")

	resetAll()
	t.Setenv(EnvSiteRoot, root)
	t.Setenv(EnvFrontendDist, dist)

	if got := SiteRoot(); got != root {
		t.Fatalf("SITE_ROOT 未生效：期望 %s，实际 %s", root, got)
	}
	if got := FrontendDist("example.com"); got != dist {
		t.Fatalf("FRONTEND_DIST 未生效：期望 %s，实际 %s", dist, got)
	}
	for _, it := range Describe("example.com") {
		if it.Name == "frontend_dist" && it.Source != SourceEnv {
			t.Fatalf("frontend_dist 来源应为 env，实际 %s", it.Source)
		}
	}
}

// 环境变量指向不存在的目录时，不能静默采用（否则会写到一个没人加载的目录），
// 必须回退到自动探测并在诊断里说明原因。
func TestEnvOverrideIgnoredWhenMissing(t *testing.T) {
	vhost := t.TempDir()
	resetAll()
	t.Setenv(EnvNginxVhostDir, "/nonexistent/vhost-dir-should-be-ignored")
	// 造一个「常见位置」不可控，这里直接验证：拿不到该 env 时不会返回它
	got, err := NginxVhostDir()
	if err == nil && got == "/nonexistent/vhost-dir-should-be-ignored" {
		t.Fatalf("不存在的 %s 被采用了：%s", EnvNginxVhostDir, got)
	}
	_ = vhost
}

// 站点根目录默认取进程工作目录（含 cboard.db 的目录），这是换机器后仍然成立的关键。
func TestSiteRootFallsBackToWorkingDir(t *testing.T) {
	resetAll()
	t.Setenv(EnvSiteRoot, "")
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if got := SiteRoot(); got != wd {
		t.Fatalf("期望回退到工作目录 %s，实际 %s", wd, got)
	}
}

// 前端产物目录：按 PanelRoot/站点域名 探测；找不到时返回空字符串以便调用方降级为纯 API 反代。
func TestFrontendDistDetectionAndDegrade(t *testing.T) {
	root := t.TempDir()
	resetAll()
	t.Setenv(EnvSiteRoot, root)
	t.Setenv(EnvFrontendDist, "")

	if got := FrontendDist("no-such-domain.invalid"); got != "" {
		t.Fatalf("没有产物时应返回空字符串，实际 %q", got)
	}

	dist := filepath.Join(root, "frontend", "dist")
	if err := os.MkdirAll(dist, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dist, "index.html"), "x")
	resetAll()
	if got := FrontendDist("example.com"); got != dist {
		t.Fatalf("期望探测到 %s，实际 %s", dist, got)
	}
}

// 找不到 nginx 时错误信息必须包含「怎么用环境变量覆盖」，否则运维只能靠猜。
func TestNginxErrorsCarryOverrideHint(t *testing.T) {
	resetAll()
	t.Setenv(EnvNginxBin, filepath.Join(t.TempDir(), "not-nginx"))
	// 若机器上恰好装了 nginx（PATH 命中），这条断言不成立，跳过
	if _, err := NginxBin(); err == nil {
		t.Skip("本机存在 nginx，无法验证未找到分支")
	} else if !strings.Contains(err.Error(), EnvNginxBin) {
		t.Fatalf("错误信息应提示可用 %s 覆盖，实际：%v", EnvNginxBin, err)
	}

	resetAll()
	t.Setenv(EnvNginxVhostDir, filepath.Join(t.TempDir(), "absent"))
	if _, err := NginxVhostDir(); err == nil {
		t.Skip("本机存在常见 vhost 目录，无法验证未找到分支")
	} else if !strings.Contains(err.Error(), EnvNginxVhostDir) {
		t.Fatalf("错误信息应提示可用 %s 覆盖，实际：%v", EnvNginxVhostDir, err)
	}
}

// Describe 必须如实体现在用的路径与来源，供启动日志和后台展示。
func TestDescribeReportsSources(t *testing.T) {
	resetAll()
	t.Setenv(EnvLetsencryptDir, "/tmp/le-live-test")
	items := Describe("example.com")
	if len(items) == 0 {
		t.Fatal("Describe 不应为空")
	}
	var found bool
	for _, it := range items {
		if it.Name == "letsencrypt_live" {
			found = true
			if it.Value != "/tmp/le-live-test" || it.Source != SourceEnv {
				t.Fatalf("letsencrypt_live 解析不符：%+v", it)
			}
		}
		if it.Value != "" && it.Source == "" {
			t.Fatalf("%s 有值但没标来源：%+v", it.Name, it)
		}
	}
	if !found {
		t.Fatal("缺少 letsencrypt_live 项")
	}
	if line := SummaryLine(); !strings.Contains(line, "letsencrypt_live=/tmp/le-live-test[env]") {
		t.Fatalf("SummaryLine 输出不符：%s", line)
	}
	if m := DescribeMap("example.com"); m["letsencrypt_live"] != "/tmp/le-live-test" {
		t.Fatalf("DescribeMap 输出不符：%v", m)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

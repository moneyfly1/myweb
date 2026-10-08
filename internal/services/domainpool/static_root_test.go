package domainpool

import (
	"os"
	"path/filepath"
	"testing"
)

// 回归：换域名/迁移后，新站点目录（/www/wwwroot/<站点域名>/frontend/dist）必须能被识别。
// 之前候选里写死了旧站点路径且 PanelRoot 传入空字符串，导致一键配置建出来的域名
// 「只有 API、没有前端界面」。
func TestStaticRootOfFindsPanelRoot(t *testing.T) {
	root := t.TempDir()
	dist := filepath.Join(root, "frontend", "dist")
	if err := os.MkdirAll(dist, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := &Manager{PanelRoot: root}
	if got := m.staticRootOf(); got != dist {
		t.Fatalf("期望命中 %s，实际 %q", dist, got)
	}
}

// 找不到任何前端目录时要返回空（调用方据此降级为纯 API 反代），不能瞎返回一个路径
// 否则 nginx 会因为 root 指向不存在的目录而 404。
func TestStaticRootOfReturnsEmptyWhenMissing(t *testing.T) {
	root := t.TempDir() // 没有 frontend/dist
	m := &Manager{PanelRoot: root, SiteDomain: "not-exist-domain.invalid"}
	if got := m.staticRootOf(); got != "" {
		t.Fatalf("期望空字符串（降级为 API 反代），实际 %q", got)
	}
}

package config_update

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ============================================================================
// anytls 的 sing-box 支持（版本边界）+ 真实链接端到端
//
// 背景（全部为真内核实测，二进制见 /root/sb-kernels，项目内稳定路径 bin/sing-box）：
//
//	sing-box 1.11.15 check → FATAL decode config: outbounds[1]: unknown outbound type: anytls（exit=1）
//	sing-box 1.12.0  check → exit=0
//	sing-box 1.14.2  check → exit=0
//
// anytls 是 **1.12.0 才加入**的 outbound 类型。未知 outbound type 会让**整份配置解码失败**
// （不是"该节点不可用"），所以：
//   - 生成器/格式表恢复 anytls（本文件所在分支的改动）；
//   - client_capability.go 按 UA 里的内核版本放行：<1.12 整批剔除 anytls，≥1.12 放行。
// ============================================================================

// anytlsCapabilityNodes 构造一组"只差 anytls 一个协议"的节点，用于 UA 矩阵对比。
func anytlsCapabilityNodes() []*ProxyNode {
	mk := func(t string) *ProxyNode {
		return &ProxyNode{Name: "n-" + t, Type: t, Server: "1.2.3.4", Port: 443, Password: "pw"}
	}
	return []*ProxyNode{mk("ss"), mk("vmess"), mk("trojan"), mk("vless"), mk("hysteria2"), mk("tuic"), mk("anytls")}
}

func countType(nodes []*ProxyNode, typ string) int {
	n := 0
	for _, p := range nodes {
		if p.Type == typ {
			n++
		}
	}
	return n
}

// UA 矩阵：sing-box <1.12 剔除 anytls；≥1.12 保留；clash-meta / stash 完全不受影响。
func TestSingBoxAnyTLSVersionGateUAMatrix(t *testing.T) {
	svc := &ConfigUpdateService{} // db==nil → 能力过滤开关默认开启
	all := anytlsCapabilityNodes()

	cases := []struct {
		ua          string
		wantAnyTLS  int
		wantTotal   int
		description string
	}{
		{"sing-box/1.11.15", 0, len(all) - 1, "sing-box 1.11 < 1.12：必须剔除 anytls"},
		{"sing-box/1.11", 0, len(all) - 1, "sing-box 1.11（两段版本号）"},
		{"sing-box/1.11.0", 0, len(all) - 1, "sing-box 1.11.0"},
		{"sing-box/1.10.0", 0, len(all) - 1, "现网真实 UA：sing-box/1.10.0（96 次请求）"},
		{"sing-box/1.8.0", 0, len(all) - 1, "现网真实 UA：sing-box/1.8.0"},
		{"sing-box/1.0", 0, len(all) - 1, "现网真实 UA：sing-box/1.0"},
		{"singbox/1.0", 0, len(all) - 1, "现网真实 UA：singbox/1.0（无连字符写法）"},
		{"sing-box/1.12.0", 1, len(all), "闸门边界：1.12.0 应放行 anytls"},
		{"sing-box/1.12", 1, len(all), "闸门边界：1.12 应放行 anytls"},
		{"sing-box/1.12.2", 1, len(all), "现网真实 UA：sing-box/1.12.2"},
		{"sing-box/1.13.0", 1, len(all), "1.13 放行"},
		{"sing-box/1.14.2", 1, len(all), "1.14.2（本机自检内核版本）放行"},
		// clash-meta / stash 用 mihomo 内核，原生支持 anytls，**不得**被 1.12 阈值误砍
		{"ClashMetaForAndroid/2.11.24.Meta", 1, len(all), "clash-meta 不受 sing-box 版本边界影响"},
		{"Linux/mihomo/1.19.28", 1, len(all), "mihomo 不受影响"},
		{"ClashMetaforWindows/1.0.0", 1, len(all), "clash-meta（驼峰写法）不受影响"},
		{"Stash/2.5.0", 1, len(all), "stash 不受影响"},
	}
	for _, c := range cases {
		got := svc.filterProxiesByClientCapability(all, c.ua)
		if n := countType(got, "anytls"); n != c.wantAnyTLS {
			t.Errorf("UA %q (%s): anytls 下发 %d 个, want %d", c.ua, c.description, n, c.wantAnyTLS)
		}
		if len(got) != c.wantTotal {
			t.Errorf("UA %q: 节点总数 %d, want %d（除 anytls 外不应减少任何节点）", c.ua, len(got), c.wantTotal)
		}
		// 除 anytls 外的协议一个都不能少——闸门只允许剔除 anytls
		for _, typ := range []string{"ss", "vmess", "trojan", "vless", "hysteria2", "tuic"} {
			if countType(got, typ) != 1 {
				t.Errorf("UA %q: 闸门误删了 %s 节点", c.ua, typ)
			}
		}
	}
}

// 闸门必须只对 sing-box 生效：clash-meta/stash 的 clientCapabilities 里不能出现
// anytls 版本阈值（防止有人把三条 case 又合并回去）。
func TestAnyTLSVersionGateOnlyOnSingBox(t *testing.T) {
	for _, ct := range []string{"clash-meta", "stash", "loon", "quantumult", "surge"} {
		caps := getClientCapabilities(ct)
		if caps == nil {
			continue
		}
		if v, ok := caps.unsupportedBefore["anytls"]; ok {
			t.Errorf("客户端 %s 被加上了 anytls 最低版本 %v —— 只有 sing-box 有 1.12 边界", ct, v)
		}
		if caps.unsupportedProtocols["anytls"] && ct != "clash-legacy" {
			t.Errorf("客户端 %s 被整协议禁用 anytls", ct)
		}
	}
	sb := getClientCapabilities("sing-box")
	if sb == nil || sb.unsupportedBefore["anytls"] != (clientVersion{major: 1, minor: 12}) {
		t.Fatalf("sing-box 必须声明 unsupportedBefore[anytls]=1.12，实际 %+v", sb)
	}
	// 老版 Clash 仍应整协议禁用 anytls（Clash Premium 不支持）
	if legacy := getClientCapabilities("clash-legacy"); !legacy.unsupportedProtocols["anytls"] {
		t.Error("clash-legacy 应整协议禁用 anytls")
	}
}

// findSingBoxKernel 定位真实 sing-box 内核二进制用于集成校验。
// 优先级：$MF_SINGBOX_BIN > 项目 bin/sing-box。找不到就跳过（CI/无内核环境不失败）。
func findSingBoxKernel(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("MF_SINGBOX_BIN"); p != "" {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
		t.Fatalf("MF_SINGBOX_BIN=%s 不存在", p)
	}
	p := filepath.Join("..", "..", "..", "bin", "sing-box")
	if st, err := os.Stat(p); err == nil && !st.IsDir() {
		abs, _ := filepath.Abs(p)
		return abs
	}
	return ""
}

// singBoxCheck 用真内核校验一份完整 sing-box 配置，返回 (exit code, 输出)。
func singBoxCheck(t *testing.T, kernel, cfg string) (int, string) {
	t.Helper()
	dir := t.TempDir()
	f := filepath.Join(dir, "config.json")
	if err := os.WriteFile(f, []byte(cfg), 0o600); err != nil {
		t.Fatalf("写临时配置失败: %v", err)
	}
	cmd := exec.Command(kernel, "check", "-c", f)
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
		} else {
			t.Fatalf("执行内核失败: %v", err)
		}
	}
	return code, string(out)
}

// 真实内核验收：生成的 sing-box 产物（含 anytls）必须能通过内核 `check`（exit=0）。
func TestSingBoxPayloadWithAnyTLSPassesRealKernel(t *testing.T) {
	kernel := findSingBoxKernel(t)
	if kernel == "" {
		t.Skip("未找到 sing-box 内核（bin/sing-box 或 $MF_SINGBOX_BIN），跳过真内核验收")
	}

	svc := &ConfigUpdateService{}
	nodes := []*ProxyNode{
		mkSS("ss-1"),
		{Name: "vless-1", Type: "vless", Server: "1.1.1.1", Port: 443, UUID: "11111111-2222-3333-4444-555555555555", TLS: true,
			Options: map[string]any{"sni": "a.example.com"}},
		{Name: "anytls-1", Type: "anytls", Server: "2.2.2.2", Port: 443, Password: "pw1", TLS: true,
			Options: map[string]any{"sni": "b.example.com", "skip-cert-verify": true}},
		{Name: "anytls-2", Type: "anytls", Server: "3.3.3.3", Port: 8443, Password: "pw2", TLS: true,
			Options: map[string]any{"sni": "c.example.com", "alpn": []string{"http/1.1"}, "client-fingerprint": "chrome"}},
	}
	payload := svc.generateSingBoxConfig(nodes)
	if !strings.Contains(payload, `"type": "anytls"`) {
		t.Fatalf("产物里没有 anytls outbound:\n%s", payload)
	}

	code, out := singBoxCheck(t, kernel, payload)
	t.Logf("内核 %s 对含 anytls 的产物 check => exit=%d %s", kernel, code, strings.TrimSpace(out))
	if code != 0 {
		t.Fatalf("真内核拒绝含 anytls 的产物（exit=%d）: %s\n%s", code, out, payload)
	}
	if len(out) != 0 && strings.Contains(out, "FATAL") {
		t.Fatalf("内核 FATAL: %s", out)
	}
}

// 对照组：同一份产物在 sing-box 1.11.x 上必须 **FATAL**——
// 这是"为什么要加客户端版本闸门"的实证。仅在提供旧内核时执行（$MF_SINGBOX_OLD_BIN）。
func TestSingBoxPayloadWithAnyTLSFailsOnPre112Kernel(t *testing.T) {
	old := os.Getenv("MF_SINGBOX_OLD_BIN")
	if old == "" {
		t.Skip("未设置 $MF_SINGBOX_OLD_BIN（<1.12 的对照内核），跳过对照组")
	}
	svc := &ConfigUpdateService{}
	payload := svc.generateSingBoxConfig([]*ProxyNode{
		{Name: "anytls-1", Type: "anytls", Server: "2.2.2.2", Port: 443, Password: "pw1", TLS: true,
			Options: map[string]any{"sni": "b.example.com"}},
	})
	code, out := singBoxCheck(t, old, payload)
	t.Logf("旧内核 %s check => exit=%d %s", old, code, strings.TrimSpace(out))
	if code == 0 {
		t.Fatalf("预期旧内核拒绝 anytls，实际 exit=0——版本边界的结论需重新核实")
	}
	if !strings.Contains(out, "unknown outbound type: anytls") {
		t.Errorf("旧内核报错文案与预期不符（期望含 unknown outbound type: anytls）: %s", out)
	}
}

// repoSyncDir 定位 repo-sync 落盘目录（同步下来但可能没被采集的订阅文件都在这里）。
func repoSyncDir() string {
	if d := os.Getenv("MF_REPO_SYNC_DIR"); d != "" {
		return d
	}
	return filepath.Join("..", "..", "..", "uploads", "repo_sync")
}

// 端到端：用**真实 anytls 链接**（取自实盘 repo-sync 文件）跑
// 「解析 → 第一层静态校验 → sing-box 渲染 → 格式校验」全链路。
//
// 用真实文件而不是硬编码夹具：夹具里含真实凭据，本仓库有"脱敏测试夹具"的既定约束
// （commit a4ea63c），所以链接一律从磁盘读，不进源码。
func TestAnyTLSRealLinksEndToEnd(t *testing.T) {
	dir := repoSyncDir()
	file := filepath.Join(dir, "shanlian_local_plain_nodes.txt")
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Skipf("未找到实盘 anytls 源文件 %s（跳过真实链接端到端）: %v", file, err)
	}

	var links []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "anytls://") {
			links = append(links, line)
		}
	}
	if len(links) == 0 {
		t.Fatalf("%s 里没有 anytls:// 链接", file)
	}
	t.Logf("从 %s 读到 %d 条真实 anytls 链接", filepath.Base(file), len(links))

	const take = 20
	nodes := make([]*ProxyNode, 0, take)
	for i, link := range links {
		if i >= take {
			break
		}
		n, err := ParseNodeLink(link)
		if err != nil {
			t.Errorf("真实链接解析失败（第 %d 条）: %v", i+1, err)
			continue
		}
		if n.Type != "anytls" {
			t.Errorf("解析类型 = %q, want anytls", n.Type)
		}
		// 第一层静态校验：这些节点必须能通过，否则就是"用户要的节点被我们自己丢了"
		if verr := ValidateProxyNode(n); verr != nil {
			t.Errorf("真实 anytls 节点未通过第一层校验: name=%q err=%v", n.Name, verr)
			continue
		}
		if strings.TrimSpace(n.Password) == "" {
			t.Errorf("真实 anytls 节点解析后 password 为空: name=%q", n.Name)
		}
		if !n.TLS {
			t.Errorf("真实 anytls 节点解析后 TLS=false（anytls 必须走 TLS）: name=%q", n.Name)
		}
		nodes = append(nodes, n)
	}
	if len(nodes) == 0 {
		t.Fatal("没有任何真实 anytls 节点走通解析+校验")
	}

	// sing-box JSON 必须渲染出 anytls outbound，且通过结构校验
	svc := &ConfigUpdateService{}
	payload := svc.generateSingBoxConfig(nodes)
	var cfg struct {
		Outbounds []map[string]any `json:"outbounds"`
	}
	if err := json.Unmarshal([]byte(payload), &cfg); err != nil {
		t.Fatalf("sing-box 产物不是合法 JSON: %v", err)
	}
	got := 0
	for _, ob := range cfg.Outbounds {
		if ob["type"] == "anytls" {
			got++
		}
	}
	if got != len(nodes) {
		t.Errorf("产物里 anytls outbound = %d, want %d", got, len(nodes))
	}
	if err := validateFormatPayload(FmtSingBox, payload); err != nil {
		t.Errorf("真实 anytls 产物未通过格式校验: %v", err)
	}
	// 基64 链接格式也必须带上这些节点（v2rayN/Shadowrocket/通用格式的覆盖不能丢）
	for _, n := range nodes {
		if link := svc.nodeToLink(n); !strings.HasPrefix(link, "anytls://") {
			t.Errorf("nodeToLink 未还原出 anytls 链接: %q", link)
		}
	}

	// 有真内核时再做一次端到端内核验收
	if kernel := findSingBoxKernel(t); kernel != "" {
		code, out := singBoxCheck(t, kernel, payload)
		if code != 0 {
			t.Fatalf("真内核拒绝真实 anytls 产物（exit=%d）: %s", code, out)
		}
		t.Logf("真内核 %s 对 %d 条真实 anytls 节点产物 check => exit=0", kernel, len(nodes))
	}
}

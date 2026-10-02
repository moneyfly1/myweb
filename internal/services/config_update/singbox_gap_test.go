package config_update

import (
	"encoding/json"
	"strings"
	"testing"
)

// sing-box 不渲染 wireguard：内核 sing-box 1.11.15 实测（outbound 形态）——
//
//	缺 private_key → FATAL "missing private key"
//	且 legacy wireguard outbound 自 1.11 废弃、1.13 将移除
//
// 现网 wg:// 节点为 0；宁可丢弃并记录，也不渲染即将被移除的写法。
func TestSingBoxDoesNotRenderWireGuard(t *testing.T) {
	wg := &ProxyNode{
		Type: "wireguard", Name: "wg-1", Server: "1.2.3.4", Port: 51820,
		Options: map[string]any{
			"private-key": "priv", "public-key": "pub", "ip": "10.0.0.2/32",
		},
	}
	svc := &ConfigUpdateService{}
	out := svc.generateSingBoxConfig([]*ProxyNode{wg})
	var cfg map[string]any
	if err := json.Unmarshal([]byte(out), &cfg); err != nil {
		t.Fatalf("sing-box 产物不是合法 JSON: %v", err)
	}
	obs, _ := cfg["outbounds"].([]any)
	for _, o := range obs {
		m, _ := o.(map[string]any)
		if m["type"] == "wireguard" {
			t.Fatal("sing-box 产物里出现了 wireguard outbound（内核已废弃该形态，不应渲染）")
		}
	}
	// 格式层必须把它记为"该格式不支持"而丢弃，而不是静默消失
	if formatRenderTypes[FmtSingBox][NormalizeNodeType("wireguard")] {
		t.Error("formatRenderTypes[FmtSingBox] 不应包含 wireguard")
	}
}

// ssh 没有解析器（protocolParsers 无 ssh://），因此不可能有 ssh 节点；
// 这里同时锁定"能力表不声明 ssh"，避免以后有人误加死代码。
func TestNoSSHParserAndNotDeclaredForSingBox(t *testing.T) {
	if _, ok := protocolParsers["ssh://"]; ok {
		t.Error("出现了 ssh:// 解析器：需同步评估 sing-box/Loon 等格式的 ssh 渲染")
	}
	// Clash 侧声明 ssh 是**正确**的（mihomo 本身有 ssh outbound），只是项目没有 ssh:// 解析器，
	// 所以永远不会有 ssh 节点走到这里 —— 声明无害，故不在此断言。
	for _, f := range []OutputFormat{FmtSingBox, FmtSurge, FmtLoon, FmtQuantumultX} {
		if formatRenderTypes[f]["ssh"] {
			t.Errorf("格式 %s 声明支持 ssh，但项目没有 ssh:// 解析器（死代码）", f)
		}
	}
}

// 回归保护：sing-box 产物必须始终是合法 JSON 且 tag 唯一（subscription 客户端按 tag 引用）。
func TestSingBoxPayloadJSONShape(t *testing.T) {
	svc := &ConfigUpdateService{}
	nodes := []*ProxyNode{
		{Type: "vless", Name: "v", Server: "1.1.1.1", Port: 443, UUID: "u", TLS: true},
		{Type: "hysteria2", Name: "h", Server: "2.2.2.2", Port: 443, Password: "p"},
		{Type: "ss", Name: "s", Server: "3.3.3.3", Port: 8388, Cipher: "aes-256-gcm", Password: "p"},
	}
	out := svc.generateSingBoxConfig(nodes)
	var cfg struct {
		Outbounds []struct {
			Type string `json:"type"`
			Tag  string `json:"tag"`
		} `json:"outbounds"`
	}
	if err := json.Unmarshal([]byte(out), &cfg); err != nil {
		t.Fatalf("产物 JSON 解析失败: %v", err)
	}
	if len(cfg.Outbounds) != len(nodes)+1 { // + DIRECT
		t.Errorf("outbounds = %d, want %d", len(cfg.Outbounds), len(nodes)+1)
	}
	seen := map[string]bool{}
	for _, o := range cfg.Outbounds {
		if o.Tag == "" {
			t.Errorf("存在空 tag 的 outbound: %+v", o)
		}
		if seen[o.Tag] {
			t.Errorf("tag 重复: %s", o.Tag)
		}
		seen[o.Tag] = true
	}
	if !strings.Contains(out, `"outbounds"`) {
		t.Error("产物缺少 outbounds 键")
	}
}

// anytls 是 sing-box **1.12.0** 才加入的 outbound 类型；真内核三版本实测（/root/sb-kernels）：
//
//	1.11.15 → FATAL decode config: outbounds[1]: unknown outbound type: anytls （exit=1）
//	1.12.0  → exit=0
//	1.14.2  → exit=0
//
// 这是**版本边界**，不是"sing-box 不支持 anytls"。因此正确修法是：
//   - 生成器/格式表恢复 anytls（= 生成器能渲染）；
//   - 由 client_capability.go 按 UA 里的内核版本放行（<1.12 整批剔除）。
//
// 本用例锁定这两件事同时成立，防止有人再"一刀切"把 anytls 从生成器里删掉。
func TestSingBoxRendersAnyTLS(t *testing.T) {
	if !formatRenderTypes[FmtSingBox]["anytls"] {
		t.Fatal("formatRenderTypes[FmtSingBox] 应包含 anytls（1.12+ 内核支持；<1.12 由能力闸门剔除）")
	}
	node := &ProxyNode{
		Type: "anytls", Name: "anytls-1", Server: "1.2.3.4", Port: 443, Password: "pw", TLS: true,
		Options: map[string]any{"sni": "s.example.com"},
	}
	svc := &ConfigUpdateService{}
	out := svc.generateSingBoxConfig([]*ProxyNode{node})
	var cfg struct {
		Outbounds []map[string]any `json:"outbounds"`
	}
	if err := json.Unmarshal([]byte(out), &cfg); err != nil {
		t.Fatalf("sing-box 产物不是合法 JSON: %v", err)
	}
	if len(cfg.Outbounds) != 2 { // anytls + DIRECT
		t.Fatalf("anytls 节点应被渲染进 sing-box，实际 outbounds=%d:\n%s", len(cfg.Outbounds), out)
	}
	ob := cfg.Outbounds[1]
	if ob["type"] != "anytls" {
		t.Errorf("outbound type = %v, want anytls", ob["type"])
	}
	if ob["password"] != "pw" {
		t.Errorf("anytls 出站缺 password: %+v", ob)
	}
	if ob["server_port"] != float64(443) {
		t.Errorf("anytls 出站 server_port 错误: %+v", ob)
	}
	// TLS 块必须存在且 enabled=true（anytls 是 TLS-only 协议，缺了内核报 TLS is required）
	tls, ok := ob["tls"].(map[string]any)
	if !ok || tls["enabled"] != true {
		t.Fatalf("anytls 出站缺 TLS 块或未启用: %+v", ob["tls"])
	}
	if tls["server_name"] != "s.example.com" {
		t.Errorf("anytls TLS server_name = %v, want s.example.com", tls["server_name"])
	}
	// 结构校验层必须放行
	if err := validateFormatPayload(FmtSingBox, out); err != nil {
		t.Errorf("validateFormatPayload(FmtSingBox) 拒绝了含 anytls 的产物: %v", err)
	}
}

// anytls 只是"对 sing-box <1.12 不安全"，对其他格式仍应正常可用——避免这次修复误伤。
func TestAnyTLSStillAvailableInOtherFormats(t *testing.T) {
	for _, f := range []OutputFormat{FmtClash, FmtLinksBase64, FmtLinksPlain, FmtLoon, FmtQuantumultX, FmtSingBox} {
		if !formatRenderTypes[f]["anytls"] {
			t.Errorf("格式 %s 丢失了 anytls 支持", f)
		}
	}
}

// 白名单锁定：本表内容 = 真内核 `check` 实测可通过的类型集合。
// 任何人想把新类型加进来，必须先拿真实内核跑出 exit=0（见 format_verify.go 里的注释）。
func TestSingBoxTypeAllowlistMatchesKernelEvidence(t *testing.T) {
	// 内核实测 exit=0：shadowsocks / vmess / vless / trojan / hysteria / hysteria2 /
	// socks / http / direct / ssh / anytls（anytls 仅 1.12.0+，见 format_verify.go 的版本边界说明）。
	//   · "socks5" 是**节点类型别名**（socks5:// 链接解析出的 Type），生成器把它渲染成
	//     内核里的 "socks"，因此两者都要在表里（否则 socks5 节点会被 format 层误剔）。
	//   · ssh 不在表内：项目没有 ssh:// 解析器（死代码），不是 sing-box 不支持。
	//   · direct 由生成器固定追加，不来自节点，故不在表内。
	want := map[string]bool{
		"ss": true, "vmess": true, "vless": true, "trojan": true,
		"hysteria": true, "hysteria2": true, "tuic": true, "anytls": true,
		"socks": true, "socks5": true, "http": true,
	}
	got := formatRenderTypes[FmtSingBox]
	for k := range want {
		if !got[k] {
			t.Errorf("表缺少内核实测可用的类型 %q", k)
		}
	}
	for k := range got {
		if !want[k] {
			t.Errorf("表含有未经内核实证的类型 %q —— 未知 outbound type 会让整份配置 FATAL", k)
		}
	}
}

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

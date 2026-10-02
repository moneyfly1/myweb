package config_update

import (
	"encoding/base64"
	"strings"
	"testing"
)

// ============================================================================
// 内核拒收类的第一层加固（全部来自实盘采集源的**真实**坏数据，非构造臆想）
//
// 三类缺陷都遵循同一个致命语义：**不是"该节点不可用"，而是"整份订阅起不来"**。
//
//	① reality public-key 长度不对
//	     mihomo   → proxy N: invalid REALITY public key        （-t 整份失败）
//	     sing-box → FATAL initialize outbound[N]: invalid public_key
//	     实盘来源：flushleopard_nodes.txt 25 条 vless，pbk 恒为 46 字符（正确 43 + "3ac"）
//	② uuid 非 UUID 形态
//	     sing-box → FATAL initialize outbound[N]: invalid uuid: uuid: invalid UUID format
//	     实盘来源：某上游 4 条 tuic，段长 8-3-3-3-11（如 65F7C474-BE1-BA2-983-D40071464C1）
//	③ transport 类型 sing-box 不认
//	     sing-box → FATAL decode config: outbounds[N].transport: unknown transport type: xhttp
//	     实盘来源：新增源里 17 条 vless(xhttp)；mihomo 支持 xhttp，故**只能按格式剔除**
// ============================================================================

// 32 字节 X25519 公钥的 base64url（43 字符，无 padding）
var validRealityPK = base64.RawURLEncoding.EncodeToString(make([]byte, 32))

func vlessWithReality(pk string) *ProxyNode {
	return &ProxyNode{
		Name: "r", Type: "vless", Server: "1.2.3.4", Port: 443,
		UUID: "11111111-2222-3333-4444-555555555555", TLS: true,
		Options: map[string]any{"reality-opts": map[string]any{"public-key": pk, "short-id": "0123456789abcdef"}},
	}
}

func TestRealityPublicKeyLengthValidated(t *testing.T) {
	if err := ValidateProxyNode(vlessWithReality(validRealityPK)); err != nil {
		t.Fatalf("合法 32 字节 reality 公钥被误拒: %v", err)
	}
	// 43 字符但带 padding（44）也应接受
	if err := ValidateProxyNode(vlessWithReality(validRealityPK + "=")); err != nil {
		t.Errorf("带 padding 的等长公钥被误拒: %v", err)
	}
	cases := []struct{ name, pk string }{
		{"实盘 46 字符（43+3ac）", "c6F6xZqeMH9qAwg4Wx0WgAwRis26apZzP3fbuUCeVCc3ac"},
		{"31 字节", base64.RawURLEncoding.EncodeToString(make([]byte, 31))},
		{"33 字节", base64.RawURLEncoding.EncodeToString(make([]byte, 33))},
		{"非 base64", "not!a!base64!!"},
	}
	for _, c := range cases {
		err := ValidateProxyNode(vlessWithReality(c.pk))
		if err == nil {
			t.Errorf("%s：非法 reality 公钥未被拒（会让整份配置 FATAL）", c.name)
			continue
		}
		if !strings.Contains(err.Error(), "public-key") {
			t.Errorf("%s：拒绝原因文案不含 public-key: %v", c.name, err)
		}
	}
}

func TestUUIDFormatValidated(t *testing.T) {
	const good = "65F7C474-BE14-BA29-9830-D40071464C11"
	// 实盘 4 条 tuic 的真实坏值
	bad := []string{
		"65F7C474-BE1-BA2-983-D40071464C1",
		"8D73CC8C-36C-254-5FE-9D49B559C55",
		"92A95292-6FA-E73-F78-C629D0BADF8",
		"E91D6371-1F1-0F5-1DE-FF1D177843F",
	}
	for _, u := range bad {
		if isValidUUID(u) {
			t.Errorf("非标准 UUID %q 被判为合法", u)
		}
		if err := ValidateProxyNode(&ProxyNode{Name: "t", Type: "tuic", Server: "1.2.3.4", Port: 443, UUID: u, Password: "pw"}); err == nil {
			t.Errorf("tuic 非标准 uuid %q 未被拒（sing-box 会整份 FATAL）", u)
		}
		if err := ValidateProxyNode(&ProxyNode{Name: "v", Type: "vless", Server: "1.2.3.4", Port: 443, UUID: u}); err == nil {
			t.Errorf("vless 非标准 uuid %q 未被拒（sing-box 会整份 FATAL）", u)
		}
		if err := ValidateProxyNode(&ProxyNode{Name: "m", Type: "vmess", Server: "1.2.3.4", Port: 443, UUID: u}); err == nil {
			t.Errorf("vmess 非标准 uuid %q 未被拒（sing-box 会整份 FATAL）", u)
		}
	}
	// 合法 UUID 必须放行（含大写、小写、无连字符 32 位十六进制三种等价写法）
	for _, u := range []string{
		good,
		"11111111-2222-3333-4444-555555555555",
		strings.ToLower(good),
		"11111111222233334444555555555555",
	} {
		if err := ValidateProxyNode(&ProxyNode{Name: "t", Type: "tuic", Server: "1.2.3.4", Port: 443, UUID: u, Password: "pw"}); err != nil {
			t.Errorf("合法 tuic uuid %q 被误拒: %v", u, err)
		}
	}
}

// xhttp：mihomo 支持、sing-box 不支持 → 只能在**格式层**剔除，绝不能在第一层整协议拒收
// （否则会连累 clash 用户白丢节点）。
func TestXHTTPKeptForClashDroppedForSingBox(t *testing.T) {
	x := &ProxyNode{
		Name: "xhttp-1", Type: "vless", Server: "1.2.3.4", Port: 443,
		UUID: "11111111-2222-3333-4444-555555555555", Network: "xhttp", TLS: true,
		Options: map[string]any{"sni": "a.example.com"},
	}
	// 第一层：必须放行（mihomo 能用）
	if err := ValidateProxyNode(x); err != nil {
		t.Fatalf("xhttp 节点被第一层误拒（会连累 clash 用户）: %v", err)
	}
	// 格式层：sing-box 剔、clash 留
	if _, bad := formatUnsupportedNode(FmtSingBox, x); !bad {
		t.Error("sing-box 应剔除 xhttp 节点（否则 unknown transport type 整份 FATAL）")
	}
	for _, f := range []OutputFormat{FmtClash, FmtLinksBase64, FmtLoon, FmtQuantumultX} {
		if _, bad := formatUnsupportedNode(f, x); bad {
			t.Errorf("格式 %s 不应剔除 xhttp 节点", f)
		}
	}
	// ws / grpc / http 是 sing-box 认的传输，不能误剔
	for _, nw := range []string{"ws", "grpc", "http", "quic", "httpupgrade", "tcp", ""} {
		n := &ProxyNode{Name: "n", Type: "vless", Server: "1.2.3.4", Port: 443, Network: nw}
		if _, bad := formatUnsupportedNode(FmtSingBox, n); bad {
			t.Errorf("sing-box 误剔了合法传输 %q", nw)
		}
	}
}

// 端到端：buildVerifiedPayload 必须把 xhttp 从 sing-box 产物里剔掉并记账，
// 且产出的 JSON 通过结构校验（第二道 transport 白名单也会兜底）。
func TestSingBoxPayloadExcludesXHTTPAndStaysValid(t *testing.T) {
	svc, _ := setupSelfCheckTest(t, nil)
	x := &ProxyNode{
		Name: "xhttp-1", Type: "vless", Server: "1.2.3.4", Port: 443,
		UUID: "11111111-2222-3333-4444-555555555555", Network: "xhttp", TLS: true,
	}
	keep := mkSS("ok-ss")
	r := svc.buildVerifiedPayload(FmtSingBox, []*ProxyNode{x, keep},
		func(ns []*ProxyNode) string { return svc.generateSingBoxConfig(ns) }, "xhttp-guard")
	if strings.Contains(r.Payload, `"xhttp"`) {
		t.Errorf("sing-box 产物里仍含 xhttp transport（内核会整份 FATAL）:\n%s", r.Payload)
	}
	found := false
	for _, d := range r.Dropped {
		if d.Name == "xhttp-1" && d.Reason == "format-unsupported-transport" {
			found = true
		}
	}
	if !found {
		t.Errorf("xhttp 节点未被记为 format-unsupported-transport: %+v", r.Dropped)
	}
	if err := validateFormatPayload(FmtSingBox, r.Payload); err != nil {
		t.Errorf("剔除后产物仍未通过格式校验: %v", err)
	}
	if !strings.Contains(r.Payload, "ok-ss") {
		t.Error("同批的正常节点被误剔")
	}
}

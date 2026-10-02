package config_update

import "testing"

// applyCapability 直接复用生产判定链（识别客户端 → 取能力表 → 过滤），便于表驱动测试。
func applyCapability(ua string, proxies []*ProxyNode) []*ProxyNode {
	clientType, version, ok := detectClientVersion(ua)
	if !ok {
		return proxies
	}
	kept, _ := filterByCapabilities(proxies, version, getClientCapabilities(clientType))
	return kept
}

// Meta 内核客户端必须被识别为 clash-meta：否则 ClashX Meta（UA 含 "clashx" 子串）
// 会被误判为老版 Clash，vless/hysteria2 整批被剔除（现网 279+93=372 个节点）。
// 现网实测 UA 均在此列。
func TestDetectClientVersionMetaKernelClients(t *testing.T) {
	meta := []string{
		"mihomo.party/v2.0.0 (clash.meta)",
		"clash-verge/v2.5.2",
		"ClashX Meta/1.4.3",
		"FlClash/0.8.75",
		"Clash Nyanpasu/1.5.2",
		"ClashMetaForAndroid/2.11.31.Meta",
	}
	for _, ua := range meta {
		gotType, _, ok := detectClientVersion(ua)
		if !ok || gotType != "clash-meta" {
			t.Errorf("detectClientVersion(%q) = (%q, ok=%v), want clash-meta", ua, gotType, ok)
		}
	}
	// 真正的老版 Clash（内置 Clash Premium/开源内核，不含 vless/hysteria2）仍须识别为 clash-legacy
	legacy := []string{
		"ClashforWindows/0.19.23",
		"ClashforWindows/0.20.39",
		"ClashX/1.118.0",
		"ClashforAndroid/2.5.12",
	}
	for _, ua := range legacy {
		gotType, _, ok := detectClientVersion(ua)
		if !ok || gotType != "clash-legacy" {
			t.Errorf("detectClientVersion(%q) = (%q, ok=%v), want clash-legacy", ua, gotType, ok)
		}
	}
}

func realityVLESS() *ProxyNode {
	return &ProxyNode{
		Type: "vless", Name: "reality节点", Server: "1.2.3.4", Port: 443, UUID: "uuid-reality",
		Options: map[string]any{"reality-opts": map[string]any{"public-key": "PBK", "short-id": "ab"}},
	}
}

// "reality" 能力键必须真正生效。Reality 是 vless 的安全层选项，ProxyNode.Type 永远
// 不等于 "reality"；历史实现只比较 p.Type，于是 unsupportedProtocols["reality"] /
// unsupportedBefore["reality"] 是死键，老客户端照样收到 Reality 节点（本层原本要防的事）。
func TestRealityCapabilityKeyActuallyFilters(t *testing.T) {
	r := realityVLESS()
	if !nodeHasReality(r) {
		t.Fatal("nodeHasReality 未识别 reality-opts.public-key")
	}
	hasRealityKey := false
	for _, k := range nodeCapabilityKeys(r) {
		if k == "reality" {
			hasRealityKey = true
		}
	}
	if !hasRealityKey {
		t.Fatalf("nodeCapabilityKeys(%v) 缺少 reality 键", nodeCapabilityKeys(r))
	}

	// 不支持 Reality 的客户端/版本：必须剔除
	unsupported := []string{
		"Shadowrocket/1500 CFNetwork/3826.400.120", // <1744
		"Surge/4.9.0", // <5.0
		"Loon/2.9.0",  // <3.0
	}
	for _, ua := range unsupported {
		if got := applyCapability(ua, []*ProxyNode{r}); len(got) != 0 {
			t.Errorf("%s 应剔除 Reality 节点，实际保留 %d 个", ua, len(got))
		}
	}

	// 支持 Reality 的客户端/版本：必须保留
	supported := []string{
		"Shadowrocket/3378 CFNetwork/3860.700.1",
		"Surge/5.5.0",
		"Loon/3.2.1",
		"mihomo.party/v2.0.0 (clash.meta)",
	}
	for _, ua := range supported {
		if got := applyCapability(ua, []*ProxyNode{r}); len(got) != 1 {
			t.Errorf("%s 应保留 Reality 节点，实际保留 %d 个", ua, len(got))
		}
	}
}

// 非 Reality 的普通 vless 不受 Reality 版本门槛影响，不能被一起砍掉。
func TestNonRealityVLESSNotGatedByRealityVersion(t *testing.T) {
	plain := &ProxyNode{Type: "vless", Name: "普通vless", Server: "1.2.3.4", Port: 443, UUID: "uuid-plain"}
	if nodeHasReality(plain) {
		t.Fatal("普通 vless 被误判为 Reality")
	}
	if got := applyCapability("Shadowrocket/1500 CFNetwork/3826.400.120", []*ProxyNode{plain}); len(got) != 1 {
		t.Errorf("普通 vless 应保留（仅 Reality 受版本门槛约束），实际保留 %d 个", len(got))
	}
}

// 老版 Clash：内置 Clash Premium 内核，仅支持 ss/vmess/trojan——
// 这条过滤是**正确行为**（该客户端确实解析不了 vless/hysteria2），本用例锁定它不被误改。
func TestClashLegacyKeepsOnlyClassicProtocols(t *testing.T) {
	mk := func(typ string) *ProxyNode { return &ProxyNode{Type: typ, Name: typ} }
	all := []*ProxyNode{mk("ss"), mk("vmess"), mk("trojan"), mk("vless"), mk("hysteria2"), mk("tuic"), mk("anytls")}
	kept := applyCapability("ClashforWindows/0.20.39", all)
	var types []string
	for _, p := range kept {
		types = append(types, p.Type)
	}
	if len(kept) != 3 {
		t.Fatalf("ClashforWindows/0.20.39 保留 %v，want 仅 ss/vmess/trojan", types)
	}
	for _, p := range kept {
		if p.Type == "vless" || p.Type == "hysteria2" || p.Type == "tuic" || p.Type == "anytls" {
			t.Errorf("legacy Clash 不应保留 %s", p.Type)
		}
	}
}

func TestNodeHasRealityVariants(t *testing.T) {
	cases := []struct {
		name string
		opts map[string]any
		want bool
	}{
		{"map[string]any 带 public-key", map[string]any{"reality-opts": map[string]any{"public-key": "PBK"}}, true},
		{"map[string]string 带 public-key", map[string]any{"reality-opts": map[string]string{"public-key": "PBK"}}, true},
		{"reality-opts 缺 public-key", map[string]any{"reality-opts": map[string]any{"short-id": "ab"}}, false},
		{"public-key 为空白", map[string]any{"reality-opts": map[string]any{"public-key": "   "}}, false},
		{"reality-opts 为 nil", map[string]any{"reality-opts": nil}, false},
		{"reality-opts 类型异常", map[string]any{"reality-opts": "not-a-map"}, false},
		{"无 Options", nil, false},
	}
	for _, c := range cases {
		n := &ProxyNode{Type: "vless", Options: c.opts}
		if got := nodeHasReality(n); got != c.want {
			t.Errorf("%s: nodeHasReality = %v, want %v", c.name, got, c.want)
		}
	}
}

// caps 为 nil（未识别客户端）时必须原样返回，不得丢节点。
func TestFilterByCapabilitiesNilCapsReturnsInput(t *testing.T) {
	in := []*ProxyNode{{Type: "vless", Name: "x"}}
	got, dropped := filterByCapabilities(in, clientVersion{major: 1}, nil)
	if dropped != 0 || len(got) != 1 {
		t.Fatalf("got %d 个 dropped=%d, want 1/0", len(got), dropped)
	}
}

// QuantumultX 使用独立版本号体系（1.x），历史上与 Loon 共用 3.0 阈值是错的：
// 现网实测 UA "Quantumult%20X/1.4.0" 解析为 1.4，而 1.4 < 3.0 恒成立，
// 等于把所有 QX 用户"支持的" reality/hysteria2/tuic 永久整批剔除。
// QX 不支持的类型由格式层拦截（生成器 + formatRenderTypes + go/ast 机械校验）。
func TestQuantumultXNotVersionGated(t *testing.T) {
	r := realityVLESS()
	for _, ua := range []string{"Quantumult%20X/1.4.0", "Quantumult%20X/1.0.20", "Quantumultx"} {
		ct, _, _ := detectClientVersion(ua)
		caps := getClientCapabilities(ct)
		if caps == nil {
			t.Fatalf("%s 未被识别为 quantumult", ua)
		}
		if caps.nodeUnsupported(r, clientVersion{major: 1, minor: 4}) {
			t.Errorf("%s：QX 的 reality 节点被版本阈值误剔除（QX 不应做版本门控）", ua)
		}
		if got := applyCapability(ua, []*ProxyNode{r}); len(got) != 1 {
			t.Errorf("%s：reality 节点应保留，实际保留 %d 个", ua, len(got))
		}
	}
}

// 反向保护：修 QX 时不能顺手把 Loon 的 3.0 阈值也删掉。
func TestLoonStillVersionGated(t *testing.T) {
	r := realityVLESS()
	if got := applyCapability("Loon/2.9.0", []*ProxyNode{r}); len(got) != 0 {
		t.Errorf("Loon 2.9 应剔除 reality 节点，实际保留 %d 个", len(got))
	}
	if got := applyCapability("Loon/3.5.0", []*ProxyNode{r}); len(got) != 1 {
		t.Errorf("Loon 3.5 应保留 reality 节点，实际保留 %d 个", len(got))
	}
}

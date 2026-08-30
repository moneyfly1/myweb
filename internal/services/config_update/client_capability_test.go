package config_update

import "testing"

func TestDetectClientVersion(t *testing.T) {
	cases := []struct {
		ua         string
		wantType   string
		wantVer    float64
		wantOK     bool
	}{
		{"ClashforWindows/0.19.23", "clash-legacy", 0.19, true},
		{"ClashforWindows/0.20.39", "clash-legacy", 0.2, true},
		{"ClashMetaForAndroid/2.11.24.Meta", "clash-meta", 2.11, true},
		{"clash.meta/alpha-de19f92", "clash-meta", 19, true}, // 回退匹配到 de19f92 中的 19（meta 无版本过滤，无影响）
		{"Shadowrocket/1744 CFNetwork/3860.700.1", "shadowrocket", 1744, true},
		{"Shadowrocket/1500 CFNetwork/3826.400.120", "shadowrocket", 1500, true},
		{"sing-box/1.12.2", "sing-box", 1.12, true},
		{"Stash/2.5.0", "stash", 2.5, true},
		{"Loon/3.2.1", "loon", 3.2, true},
		{"Surge/5.2.0", "surge", 5.2, true},
		{"v2rayN/7.2.0", "v2ray", 7.2, true},
		{"Mozilla/5.0 (iPhone) AppleWebKit/605.1.15", "", 0, false}, // 浏览器
		{"curl/8.13.0", "", 0, false},                                // curl
	}
	for _, c := range cases {
		gotType, gotVer, gotOK := detectClientVersion(c.ua)
		if gotType != c.wantType || gotVer != c.wantVer || gotOK != c.wantOK {
			t.Errorf("detectClientVersion(%q) = (%q, %v, %v), want (%q, %v, %v)",
				c.ua, gotType, gotVer, gotOK, c.wantType, c.wantVer, c.wantOK)
		}
	}
}

func TestFilterProxiesByClientCapability(t *testing.T) {
	s := &ConfigUpdateService{}
	mk := func(typ string) *ProxyNode { return &ProxyNode{Type: typ} }
	all := []*ProxyNode{
		mk("ss"), mk("vmess"), mk("trojan"),
		mk("vless"), mk("reality"), mk("hysteria2"), mk("tuic"), mk("anytls"),
	}

	// 老版 Clash：只保留 ss/vmess/trojan
	legacy := s.filterProxiesByClientCapability(all, "ClashforWindows/0.19.23")
	if len(legacy) != 3 {
		t.Errorf("legacy clash: got %d proxies, want 3", len(legacy))
	}
	for _, p := range legacy {
		if p.Type == "vless" || p.Type == "reality" {
			t.Errorf("legacy clash should not contain %s", p.Type)
		}
	}

	// Clash Meta：全部保留
	meta := s.filterProxiesByClientCapability(all, "ClashMetaForAndroid/2.11.24.Meta")
	if len(meta) != len(all) {
		t.Errorf("clash meta: got %d proxies, want %d", len(meta), len(all))
	}

	// Shadowrocket 老构建号（1500）：过滤 reality/hysteria2/tuic/anytls
	srOld := s.filterProxiesByClientCapability(all, "Shadowrocket/1500 CFNetwork/3860.700.1")
	for _, p := range srOld {
		if p.Type == "reality" || p.Type == "anytls" {
			t.Errorf("shadowrocket 1500 should not contain %s", p.Type)
		}
	}

	// Shadowrocket 新构建号（1900）：全部保留
	srNew := s.filterProxiesByClientCapability(all, "Shadowrocket/1900 CFNetwork/3860.700.1")
	if len(srNew) != len(all) {
		t.Errorf("shadowrocket 1900: got %d proxies, want %d", len(srNew), len(all))
	}

	// 浏览器 UA：原样返回
	browser := s.filterProxiesByClientCapability(all, "Mozilla/5.0 (iPhone) AppleWebKit/605.1.15")
	if len(browser) != len(all) {
		t.Errorf("browser: got %d proxies, want %d", len(browser), len(all))
	}
}

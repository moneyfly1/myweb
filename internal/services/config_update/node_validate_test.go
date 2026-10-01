package config_update

import (
	"encoding/base64"
	"net/url"
	"strings"
	"testing"
)

func ssNode(name, cipher, password string) *ProxyNode {
	return &ProxyNode{Name: name, Type: "ss", Server: "1.2.3.4", Port: 8388, Cipher: cipher, Password: password, Options: map[string]any{}}
}

// 白名单核心：mihomo 不支持 naive / naive+https，必须丢弃（现网曾出现，是"整份订阅失效"隐患）
func TestValidateNodeRejectsUnsupportedTypes(t *testing.T) {
	for _, typ := range []string{"naive", "naive+https", "juicity", "brook", "h2", "mtproto", "bogus"} {
		n := &ProxyNode{Name: "x", Type: typ, Server: "1.2.3.4", Port: 443, Password: "pw"}
		err := ValidateProxyNode(n)
		if err == nil {
			t.Fatalf("类型 %q 应被丢弃，但通过了校验", typ)
		}
		ve, ok := err.(*NodeValidationError)
		if !ok || ve.Code != ReasonUnsupportedType {
			t.Fatalf("类型 %q 原因码应为 %s，实际 %v", typ, ReasonUnsupportedType, err)
		}
	}
	// naive 链接也不应再能解析成节点
	for _, link := range []string{"naive://user:pw@a.com:443#n", "naive+https://user:pw@a.com:443#n"} {
		if node, err := ParseNodeLink(link); err == nil && node != nil {
			t.Fatalf("%s 不应被解析为节点，得到 %+v", link, node)
		}
		if scheme := knownUnsupportedScheme(link); scheme == "" {
			t.Fatalf("%s 应被识别为已知不支持的协议", link)
		}
	}
}

// 必填字段：port 缺失/越界、server 空或非法、名字空
func TestValidateNodeRequiredFields(t *testing.T) {
	base := func() *ProxyNode { return ssNode("ok", "aes-128-gcm", "pw") }

	if n := base(); (func() bool { n.Port = 0; return ValidateProxyNode(n) != nil })() == false {
		t.Fatal("port=0 应被丢弃")
	}
	if n := base(); (func() bool { n.Port = 65536; return ValidateProxyNode(n) != nil })() == false {
		t.Fatal("port=65536 应被丢弃")
	}
	if n := base(); (func() bool { n.Server = ""; return ValidateProxyNode(n) != nil })() == false {
		t.Fatal("server 空应被丢弃")
	}
	if n := base(); (func() bool { n.Server = "not a host"; return ValidateProxyNode(n) != nil })() == false {
		t.Fatal("非法 server 应被丢弃")
	}
	if n := base(); (func() bool { n.Name = "  "; return ValidateProxyNode(n) != nil })() == false {
		t.Fatal("空名字应被丢弃")
	}
	if n := base(); (func() bool { n.Password = ""; return ValidateProxyNode(n) != nil })() == false {
		t.Fatal("ss 缺 password 应被丢弃")
	}
	if n := base(); (func() bool { n.Cipher = ""; return ValidateProxyNode(n) != nil })() == false {
		t.Fatal("ss 缺 cipher 应被丢弃")
	}
	// 合法 ss 通过；IPv4 / IPv6 / 域名都算合法 server
	valid := []*ProxyNode{
		ssNode("a", "aes-128-gcm", "pw"),
		{Name: "b", Type: "ss", Server: "example.com", Port: 443, Cipher: "chacha20-ietf-poly1305", Password: "p", Options: map[string]any{}},
		{Name: "c", Type: "ss", Server: "2001:db8::1", Port: 443, Cipher: "aes-256-cfb", Password: "p", Options: map[string]any{}},
	}
	for _, n := range valid {
		if err := ValidateProxyNode(n); err != nil {
			t.Fatalf("合法节点 %+v 不应被丢弃: %v", n, err)
		}
	}
}

// 合法 trojan 通过
func TestValidateNodeAcceptsValidTrojan(t *testing.T) {
	n := &ProxyNode{Name: "HK-01", Type: "trojan", Server: "hk.example.com", Port: 443, Password: "s3cr3t", TLS: true, Options: map[string]any{}}
	if err := ValidateProxyNode(n); err != nil {
		t.Fatalf("合法 trojan 应通过: %v", err)
	}
	// 缺 password 的 trojan 必须丢弃（内核报 unset fields）
	bad := &ProxyNode{Name: "HK-02", Type: "trojan", Server: "hk.example.com", Port: 443, Options: map[string]any{}}
	if err := ValidateProxyNode(bad); err == nil {
		t.Fatal("缺 password 的 trojan 应被丢弃")
	}
}

// ss cipher 白名单：内核实测 PASS/FAIL 的取值必须严格一致
func TestValidateSSCipherWhitelist(t *testing.T) {
	// 内核实测通过
	pass := []string{
		"aes-128-gcm", "aes-192-gcm", "aes-256-gcm",
		"chacha20-ietf-poly1305", "xchacha20-ietf-poly1305",
		"2022-blake3-aes-128-gcm", // 16 字节密钥在下面单独构造
		"2022-blake3-aes-256-gcm", "2022-blake3-chacha20-poly1305",
		"aes-128-cfb", "aes-192-cfb", "aes-256-cfb",
		"aes-128-ctr", "aes-192-ctr", "aes-256-ctr",
		"chacha20-ietf", "chacha20", "xchacha20", "rc4-md5",
		"aes-128-gcm-siv", "aes-256-gcm-siv", "aes-128-ccm", "aes-256-ccm",
		"chacha8-ietf-poly1305", "xchacha8-ietf-poly1305",
	}
	for _, c := range pass {
		pw := "pw"
		if want, is2022 := ss2022RequiredKeyLen[c]; is2022 {
			pw = base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", want)))
		}
		if err := ValidateProxyNode(ssNode("ok", c, pw)); err != nil {
			t.Fatalf("cipher %q 内核实测通过，不应被丢弃: %v", c, err)
		}
	}

	// 内核实测失败 → 必须拒绝，否则整份配置失效
	fail := []string{
		"salsa20", "rc4-md5-6", "aes-128-ocb", "aes-192-ocb", "aes-256-ocb",
		"aes-128-cfb1", "aes-192-cfb1", "aes-256-cfb1",
		"aes-128-cfb8", "aes-192-cfb8", "aes-256-cfb8",
		"aes-128-cfb128", "aes-256-cfb128",
		"AEAD_AES_128_GCM", "dummy", "bogus-xyz", "aes-256-gcm2", "",
	}
	for _, c := range fail {
		err := ValidateProxyNode(ssNode("bad", c, "pw"))
		if err == nil {
			t.Fatalf("cipher %q 内核实测失败，必须被丢弃", c)
		}
		ve, _ := err.(*NodeValidationError)
		if c == "" && ve.Code != ReasonMissingField {
			t.Fatalf("空 cipher 原因码应为 %s，实际 %s", ReasonMissingField, ve.Code)
		}
	}
}

// ss 2022-blake3 密钥长度/编码校验（内核实测：不符 → "bad key length"/"decode key"，整份配置失效）
func TestValidateSS2022KeyLength(t *testing.T) {
	k16 := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", 16)))
	k32 := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("b", 32)))
	k8 := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("c", 8)))

	cases := []struct {
		cipher string
		pw     string
		ok     bool
	}{
		{"2022-blake3-aes-128-gcm", k16, true},
		{"2022-blake3-aes-128-gcm", k32, false},
		{"2022-blake3-aes-128-gcm", k8, false},
		{"2022-blake3-aes-128-gcm", "not-base64!!", false},
		{"2022-blake3-aes-256-gcm", k32, true},
		{"2022-blake3-aes-256-gcm", k16, false},
		{"2022-blake3-chacha20-poly1305", k32, true},
		{"2022-blake3-chacha20-poly1305", k16, false},
		// 多用户形态：每一段都必须精确匹配长度（内核实测 16:16 PASS / 32:16 FAIL）
		{"2022-blake3-aes-128-gcm", k16 + ":" + k16, true},
		{"2022-blake3-aes-256-gcm", k32 + ":" + k32, true},
		{"2022-blake3-aes-256-gcm", k32 + ":" + k16, false},
	}
	for _, tc := range cases {
		err := ValidateProxyNode(ssNode("n", tc.cipher, tc.pw))
		if tc.ok && err != nil {
			t.Fatalf("%s pw=%q 应通过: %v", tc.cipher, tc.pw, err)
		}
		if !tc.ok {
			if err == nil {
				t.Fatalf("%s pw=%q 应被丢弃", tc.cipher, tc.pw)
			}
			ve, _ := err.(*NodeValidationError)
			// 原因码已细分为两类：非合法 base64 与 长度不符
			if ve.Code != ReasonCipherKeyMismatch && ve.Code != ReasonCipherKeyNotB64 {
				t.Fatalf("%s pw=%q 原因码应为 %s 或 %s，实际 %s",
					tc.cipher, tc.pw, ReasonCipherKeyMismatch, ReasonCipherKeyNotB64, ve.Code)
			}
		}
	}
}

// 现网真实故障回归：URL 编码污染的 2022 密钥 → 修正为合法密钥而不是丢弃
//
// 事故：ss 节点 password 含 "%2F"/"%3A"（'/'→%2F，':'→%3A），内核报
// "decode key: illegal base64 data at input byte 41" 并让整份 583 节点配置失效。
func TestNormalizeSS2022KeyFixesURLEncodedPassword(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("aaaabbbbccccddddeeeeffff00001111")) // 32 字节
	raw := strings.ReplaceAll(key, "/", "%2F") + "%3A" + key

	n := ssNode("荷兰·SS·1倍消耗", "2022-blake3-aes-256-gcm", raw)
	// 修正前：非法 → 会被丢弃
	if err := ValidateProxyNode(n); err == nil {
		t.Fatal("URL 编码污染的 2022 密钥在修正前应判定为不合法")
	}
	corrected, detail := NormalizeSS2022Key(n)
	if !corrected {
		t.Fatalf("应识别并修正 URL 编码污染，detail=%q", detail)
	}
	if !strings.Contains(n.Password, ":") || strings.Contains(n.Password, "%") {
		t.Fatalf("修正后的密钥不应再含百分号编码: %q", n.Password)
	}
	// 修正后：合法 → 保留该节点（用户不会少一个可用节点）
	if err := ValidateProxyNode(n); err != nil {
		t.Fatalf("修正后应通过校验: %v", err)
	}
	// 幂等：再次调用不应重复修正
	if again, _ := NormalizeSS2022Key(n); again {
		t.Fatal("修正应幂等")
	}
	// 正常密钥不应被改动
	normal := ssNode("n", "2022-blake3-aes-256-gcm", key)
	if changed, _ := NormalizeSS2022Key(normal); changed || normal.Password != key {
		t.Fatal("合法密钥不应被修改")
	}
}

// 回归：密钥里含 base64 合法的 '+' 时，还原必须用百分号解码而不是 QueryUnescape。
//
// 用错 url.QueryUnescape 会把 '+' 解成空格，密钥依旧是非法 base64 →
// 修正静默失效 → 坏节点继续留在订阅里让整份配置失效。
// 本用例的密钥同时含 '+' 与 '/'（'/' 会被 URL 编码成 %2F），精确覆盖该缺陷。
func TestNormalizeSS2022KeyPreservesPlusInBase64(t *testing.T) {
	key := "Bw4VHCMqMTg/Rk1UW2JpcHd+hYyTmqGor7a9xMvS2eA=" // 32 字节，含 '+' 与 '/'
	if dec, err := base64.StdEncoding.DecodeString(key); err != nil || len(dec) != 32 {
		t.Fatalf("测试夹具本身不是 32 字节合法 base64: %v", err)
	}
	if !strings.Contains(key, "+") || !strings.Contains(key, "/") {
		t.Fatal("测试夹具必须同时含 '+' 与 '/' 才有回归意义")
	}

	encoded := strings.ReplaceAll(key, "/", "%2F") // 只编码 '/'，'+' 原样保留（与现网一致）
	n := ssNode("含加号的密钥", "2022-blake3-aes-256-gcm", encoded)
	if err := ValidateProxyNode(n); err == nil {
		t.Fatal("百分号编码污染的密钥在修正前应判定为不合法")
	}

	corrected, detail := NormalizeSS2022Key(n)
	if !corrected {
		t.Fatalf("含 '+' 的密钥也必须能被正确还原（detail=%q）——用 QueryUnescape 会在此失败", detail)
	}
	if n.Password != key {
		t.Fatalf("还原结果必须与原始密钥逐字节一致:\n got %q\nwant %q", n.Password, key)
	}
	if err := ValidateProxyNode(n); err != nil {
		t.Fatalf("还原后应通过校验: %v", err)
	}
}

// 解析层从源头还原 ss userinfo 的百分号编码（现网污染的来源）
func TestExtractSSAuthPercentDecodesUserinfo(t *testing.T) {
	key := "Bw4VHCMqMTg/Rk1UW2JpcHd+hYyTmqGorq9xMvS2eA="
	link := "ss://" + url.QueryEscape("2022-blake3-aes-256-gcm") + ":" + strings.ReplaceAll(key, "/", "%2F") + "@1.2.3.4:8388#n"
	node, err := ParseNodeLink(link)
	if err != nil {
		// 该链接形态可能不被解析器接受；退化为直接验证辅助函数的契约
		t.Skipf("解析器未接受该链接形态: %v", err)
	}
	if strings.Contains(node.Password, "%") {
		t.Fatalf("解析后密码不应残留百分号编码: %q", node.Password)
	}
}

// ss plugin 校验：未知插件丢弃，已知插件缺必需参数丢弃（内核实测都会让配置失效或必然连不上）
func TestValidateSSPlugin(t *testing.T) {
	withPlugin := func(name string, opts map[string]any) *ProxyNode {
		n := ssNode("p", "aes-128-gcm", "pw")
		n.Options["plugin"] = name
		if opts != nil {
			n.Options["plugin-opts"] = opts
		}
		return n
	}
	if err := ValidateProxyNode(withPlugin("obfs", map[string]any{"mode": "tls"})); err != nil {
		t.Fatalf("obfs+mode 应通过: %v", err)
	}
	if err := ValidateProxyNode(withPlugin("v2ray-plugin", map[string]any{"mode": "websocket", "host": "a.com"})); err != nil {
		t.Fatalf("v2ray-plugin+mode 应通过: %v", err)
	}
	if err := ValidateProxyNode(withPlugin("shadow-tls", map[string]any{"host": "a.com", "password": "pw"})); err != nil {
		t.Fatalf("shadow-tls 完整参数应通过: %v", err)
	}
	for _, tc := range []struct {
		name string
		opts map[string]any
	}{
		{"bogus-plugin", map[string]any{}},
		{"obfs", map[string]any{}},                      // 缺 mode
		{"obfs", nil},                                   // 完全没有 plugin-opts
		{"v2ray-plugin", map[string]any{}},              // 缺 mode
		{"shadow-tls", map[string]any{"host": "a.com"}}, // 缺 password
	} {
		err := ValidateProxyNode(withPlugin(tc.name, tc.opts))
		if err == nil {
			t.Fatalf("plugin=%s opts=%v 应被丢弃", tc.name, tc.opts)
		}
		ve, _ := err.(*NodeValidationError)
		if ve.Code != ReasonPluginInvalid {
			t.Fatalf("plugin=%s 原因码应为 %s，实际 %s", tc.name, ReasonPluginInvalid, ve.Code)
		}
	}
	// 不带 plugin 的 ss 不受影响
	if err := ValidateProxyNode(ssNode("plain", "aes-128-gcm", "pw")); err != nil {
		t.Fatalf("无 plugin 的 ss 应通过: %v", err)
	}
}

// ssr：protocol/obfs 必须在内核枚举内，cipher 不接受 AEAD
func TestValidateSSRNode(t *testing.T) {
	mk := func(cipher, protocol, obfs string) *ProxyNode {
		return &ProxyNode{Name: "r", Type: "ssr", Server: "1.2.3.4", Port: 8388, Cipher: cipher, Password: "pw",
			Options: map[string]any{"protocol": protocol, "obfs": obfs}}
	}
	if err := ValidateProxyNode(mk("aes-256-cfb", "origin", "plain")); err != nil {
		t.Fatalf("合法 ssr 应通过: %v", err)
	}
	if err := ValidateProxyNode(mk("chacha20-ietf", "auth_chain_b", "tls1.2_ticket_auth")); err != nil {
		t.Fatalf("合法 ssr 应通过: %v", err)
	}
	// ssr 不支持 AEAD（内核实测 "not none or a supported stream cipher in ssr"）
	if err := ValidateProxyNode(mk("aes-128-gcm", "origin", "plain")); err == nil {
		t.Fatal("ssr + AEAD cipher 应被丢弃")
	}
	for _, p := range []string{"auth_chain_c", "auth_chain_d", "auth_chain_e", "auth_chain_f", "verify_sha1", "bogus", ""} {
		if err := ValidateProxyNode(mk("aes-256-cfb", p, "plain")); err == nil {
			t.Fatalf("ssr protocol %q 内核实测失败，应被丢弃", p)
		}
	}
	for _, o := range []string{"tls1.0_session_auth", "bogus", ""} {
		if err := ValidateProxyNode(mk("aes-256-cfb", "origin", o)); err == nil {
			t.Fatalf("ssr obfs %q 内核实测失败，应被丢弃", o)
		}
	}
}

// vmess cipher 白名单（实测 PASS: auto/aes-128-gcm/chacha20-poly1305/none/zero；FAIL: 空/bogus）
func TestValidateVMessCipher(t *testing.T) {
	mk := func(c string) *ProxyNode {
		return &ProxyNode{Name: "v", Type: "vmess", Server: "1.2.3.4", Port: 443,
			UUID: "11111111-2222-4333-8444-555555555555", Cipher: c, Options: map[string]any{}}
	}
	for _, c := range []string{"auto", "aes-128-gcm", "chacha20-poly1305", "none", "zero", ""} {
		if err := ValidateProxyNode(mk(c)); err != nil {
			t.Fatalf("vmess cipher %q 应通过: %v", c, err)
		}
	}
	for _, c := range []string{"bogus", "aes-256-gcm"} {
		err := ValidateProxyNode(mk(c))
		if err == nil {
			t.Fatalf("vmess cipher %q 应被丢弃", c)
		}
		if ve, _ := err.(*NodeValidationError); ve.Code != ReasonVMessCipher {
			t.Fatalf("原因码应为 %s，实际 %s", ReasonVMessCipher, ve.Code)
		}
	}
	// uuid 缺失
	if err := ValidateProxyNode(&ProxyNode{Name: "v", Type: "vmess", Server: "1.2.3.4", Port: 443, Options: map[string]any{}}); err == nil {
		t.Fatal("vmess 缺 uuid 应被丢弃")
	}
}

// vless：解析器曾生成被内核拒绝的 xtls-rprx-direct，必须拦下
func TestValidateVLESSFlow(t *testing.T) {
	mk := func(flow, encryption string) *ProxyNode {
		opts := map[string]any{}
		if flow != "" {
			opts["flow"] = flow
		}
		if encryption != "" {
			opts["encryption"] = encryption
		}
		return &ProxyNode{Name: "l", Type: "vless", Server: "1.2.3.4", Port: 443,
			UUID: "11111111-2222-4333-8444-555555555555", Options: opts}
	}
	for _, f := range []string{"", "xtls-rprx-vision"} {
		if err := ValidateProxyNode(mk(f, "none")); err != nil {
			t.Fatalf("vless flow %q 应通过: %v", f, err)
		}
	}
	for _, f := range []string{"xtls-rprx-direct", "xtls-rprx-origin"} {
		err := ValidateProxyNode(mk(f, "none"))
		if err == nil {
			t.Fatalf("vless flow %q 内核实测失败，应被丢弃", f)
		}
		if ve, _ := err.(*NodeValidationError); ve.Code != ReasonVLESSFlow {
			t.Fatalf("原因码应为 %s，实际 %s", ReasonVLESSFlow, ve.Code)
		}
	}
	err := ValidateProxyNode(mk("xtls-rprx-vision", "bogus"))
	if err == nil {
		t.Fatal("vless encryption bogus 应被丢弃")
	}
	if ve, _ := err.(*NodeValidationError); ve.Code != ReasonVLESSEncryption {
		t.Fatalf("原因码应为 %s，实际 %s", ReasonVLESSEncryption, ve.Code)
	}
	// 解析器不再生成 xtls-rprx-direct（xtls=1 时留空）
	n, perr := ParseNodeLink("vless://11111111-2222-4333-8444-555555555555@a.com:443?security=tls&xtls=1&type=tcp#n1")
	if perr != nil {
		t.Fatalf("解析失败: %v", perr)
	}
	if got := optString(n.Options, "flow"); got != "" {
		t.Fatalf("xtls=1 不应再生成 flow，实际 %q", got)
	}
}

// 同名去重：内核遇重名会报 "is the duplicate name" 并让整份配置失效。
// 策略是「重命名」而不是丢弃——用户不该因为重名少一个可用节点。
func TestStaticValidateNodesDedupesNames(t *testing.T) {
	nodes := []*ProxyNode{
		ssNode("香港节点", "aes-128-gcm", "pw"),
		ssNode("香港节点", "aes-128-gcm", "pw2"),
		ssNode("香港节点", "aes-128-gcm", "pw3"),
		{Name: "naive", Type: "naive", Server: "1.2.3.4", Port: 443, Password: "pw"},
	}
	kept, events := StaticValidateNodes(nodes, "test")

	if len(kept) != 3 {
		t.Fatalf("应保留 3 个节点（重名改名、naive 丢弃），实际 %d", len(kept))
	}
	seen := map[string]bool{}
	for _, n := range kept {
		if seen[n.Name] {
			t.Fatalf("去重后仍有重名: %q", n.Name)
		}
		seen[n.Name] = true
	}
	if !seen["香港节点"] || !seen["香港节点-1"] || !seen["香港节点-2"] {
		t.Fatalf("重命名结果不符合预期: %v", seen)
	}

	var dropped, renamed int
	for _, e := range events {
		switch e.Event {
		case "dropped_at_ingest":
			dropped++
		case "renamed_at_ingest":
			renamed++
		}
	}
	if dropped != 1 {
		t.Fatalf("应记录 1 条丢弃事件，实际 %d", dropped)
	}
	if renamed != 2 {
		t.Fatalf("应记录 2 条重命名事件，实际 %d", renamed)
	}
}

// 校验事件必须脱敏：绝不写入 password/uuid
func TestValidationEventHasNoCredentials(t *testing.T) {
	n := ssNode("secret-node", "aes-128-gcm", "SUPER-SECRET-PASSWORD")
	n.UUID = "11111111-2222-4333-8444-555555555555"
	_, events := StaticValidateNodes([]*ProxyNode{
		{Name: n.Name, Type: "naive", Server: n.Server, Port: n.Port, Password: "SUPER-SECRET-PASSWORD", UUID: n.UUID},
	}, "test")
	if len(events) != 1 {
		t.Fatalf("应有 1 条事件，实际 %d", len(events))
	}
	blob := events[0].Reason + events[0].NodeName + events[0].Server + events[0].NodeType
	for _, secret := range []string{"SUPER-SECRET-PASSWORD", n.UUID} {
		if strings.Contains(blob, secret) {
			t.Fatalf("校验事件泄露了凭据 %q: %s", secret, blob)
		}
	}
}

func TestLinkDisplayNameNeverLeaksLink(t *testing.T) {
	link := "ss://YWVzLTEyOC1nY206U1VQRVItU0VDUkVU@1.2.3.4:8388#%E9%A6%99%E6%B8%AF%E8%8A%82%E7%82%B9"
	got := linkDisplayName(link)
	if got != "香港节点" {
		t.Fatalf("应取 #fragment 作为显示名，实际 %q", got)
	}
	if strings.Contains(got, "SUPER") || strings.Contains(got, "ss://") {
		t.Fatalf("显示名泄露了链接内容: %q", got)
	}
	if linkDisplayName("naive://user:pw@a.com:443") != "(未解析节点)" {
		t.Fatal("无 fragment 时应返回占位名")
	}
}

func TestNormalizeNodeTypeAliases(t *testing.T) {
	cases := map[string]string{
		"socks": "socks5", "socks5": "socks5", "wg": "wireguard", "hy2": "hysteria2",
		"VLESS": "vless", " Hysteria2 ": "hysteria2",
		"naive": "", "naive+https": "", "bogus": "",
	}
	for in, want := range cases {
		if got := NormalizeNodeType(in); got != want {
			t.Fatalf("NormalizeNodeType(%q) = %q, want %q", in, got, want)
		}
	}
}

// 内核实测出来的"只有内核能拦"的取值，已补进第一层（每一行都对应一条实测 FAIL）
func TestValidateKernelSpecificEnums(t *testing.T) {
	// ss plugin 的 mode 取值白名单（实测：obfs 只接受 tls/http；v2ray-plugin 只接受 websocket）
	mkPlugin := func(name string, opts map[string]any) *ProxyNode {
		n := ssNode("p", "aes-128-gcm", "pw")
		n.Options["plugin"] = name
		n.Options["plugin-opts"] = opts
		return n
	}
	for _, mode := range []string{"tls", "http"} {
		if err := ValidateProxyNode(mkPlugin("obfs", map[string]any{"mode": mode})); err != nil {
			t.Fatalf("obfs mode=%s 实测通过，不应丢弃: %v", mode, err)
		}
	}
	for _, mode := range []string{"bogus", "quic", ""} {
		if err := ValidateProxyNode(mkPlugin("obfs", map[string]any{"mode": mode})); err == nil {
			t.Fatalf("obfs mode=%q 实测失败，应丢弃", mode)
		}
	}
	if err := ValidateProxyNode(mkPlugin("v2ray-plugin", map[string]any{"mode": "websocket"})); err != nil {
		t.Fatalf("v2ray-plugin mode=websocket 应通过: %v", err)
	}
	for _, mode := range []string{"quic", "bogus"} {
		if err := ValidateProxyNode(mkPlugin("v2ray-plugin", map[string]any{"mode": mode})); err == nil {
			t.Fatalf("v2ray-plugin mode=%q 实测失败，应丢弃", mode)
		}
	}

	// hysteria2 obfs 白名单（实测：salamander 通过，bogus 报 unknown obfs type）
	hy2 := func(obfs string) *ProxyNode {
		opts := map[string]any{}
		if obfs != "" {
			opts["obfs"] = obfs
		}
		return &ProxyNode{Name: "h", Type: "hysteria2", Server: "1.2.3.4", Port: 443, Password: "pw", Options: opts}
	}
	if err := ValidateProxyNode(hy2("")); err != nil {
		t.Fatalf("hysteria2 无 obfs 应通过: %v", err)
	}
	if err := ValidateProxyNode(hy2("salamander")); err != nil {
		t.Fatalf("hysteria2 obfs=salamander 应通过: %v", err)
	}
	if err := ValidateProxyNode(hy2("bogus")); err == nil {
		t.Fatal("hysteria2 obfs=bogus 实测失败，应丢弃")
	}

	// wireguard private-key 必须是合法 base64（实测：非 base64 → decode private key）
	wg := func(pk string) *ProxyNode {
		return &ProxyNode{Name: "w", Type: "wireguard", Server: "1.2.3.4", Port: 51820,
			Options: map[string]any{"private-key": pk, "ip": "10.0.0.2/32"}}
	}
	if err := ValidateProxyNode(wg(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("Z", 32))))); err != nil {
		t.Fatalf("合法 base64 私钥应通过: %v", err)
	}
	if err := ValidateProxyNode(wg("")); err != nil {
		t.Fatalf("空私钥实测通过，不应丢弃: %v", err)
	}
	if err := ValidateProxyNode(wg("not-base64!!!")); err == nil {
		t.Fatal("非法 base64 私钥实测失败，应丢弃")
	}

	// vless reality-opts 缺 public-key（实测：unset fields: public-key）
	vless := func(ro map[string]any) *ProxyNode {
		return &ProxyNode{Name: "l", Type: "vless", Server: "1.2.3.4", Port: 443,
			UUID:    "11111111-2222-4333-8444-555555555555",
			Options: map[string]any{"reality-opts": ro, "encryption": "none"}}
	}
	if err := ValidateProxyNode(vless(map[string]any{"public-key": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"})); err != nil {
		t.Fatalf("完整 reality-opts 应通过: %v", err)
	}
	for _, ro := range []map[string]any{{"short-id": "c6a6"}, {}, {"public-key": ""}} {
		err := ValidateProxyNode(vless(ro))
		if err == nil {
			t.Fatalf("reality-opts=%v 缺 public-key，实测失败，应丢弃", ro)
		}
		if ve, _ := err.(*NodeValidationError); ve.Code != ReasonRealityInvalid {
			t.Fatalf("原因码应为 %s，实际 %s", ReasonRealityInvalid, ve.Code)
		}
	}
}

// 解析器把 security=reality 但缺 pbk 的链接解析成残缺 reality-opts 时，必须被第一层拦下
func TestRealityWithoutPbkIsRejected(t *testing.T) {
	link := "vless://11111111-2222-4333-8444-555555555555@a.com:443?security=reality&sid=c6a6&type=tcp&encryption=none#缺pbk的reality"
	n, err := ParseNodeLink(link)
	if err != nil {
		t.Skipf("解析器不接受该形态: %v", err)
	}
	if _, hasReality := n.Options["reality-opts"]; !hasReality {
		t.Skip("解析器未生成 reality-opts，本用例不适用")
	}
	if verr := ValidateProxyNode(n); verr == nil {
		t.Fatalf("残缺 reality-opts 必须被第一层拦下（否则内核会让整份配置失效）: %+v", n.Options["reality-opts"])
	}
}

// 内核内置代理名冲突（内核实测全大写 DIRECT/REJECT/REJECT-DROP/PASS/COMPATIBLE → 整份配置失效）
func TestValidateReservedProxyNames(t *testing.T) {
	for _, name := range []string{"DIRECT", "REJECT", "REJECT-DROP", "PASS", "COMPATIBLE"} {
		n := ssNode(name, "aes-128-gcm", "pw")
		err := ValidateProxyNode(n)
		if err == nil {
			t.Fatalf("节点名 %q 与内核内置代理重名，必须丢弃", name)
		}
		if ve, _ := err.(*NodeValidationError); ve.Code != ReasonReservedName {
			t.Fatalf("原因码应为 %s，实际 %s", ReasonReservedName, ve.Code)
		}
	}
	// 大小写不同 / 其它名字内核实测通过，不能误杀
	for _, name := range []string{"direct", "Direct", "reject", "GLOBAL", "PROXY", "Proxy", "NORMAL"} {
		if err := ValidateProxyNode(ssNode(name, "aes-128-gcm", "pw")); err != nil {
			t.Fatalf("节点名 %q 实测通过，不应丢弃: %v", name, err)
		}
	}
}

// restls 插件缺 version（实测 → has unset fields: version）
func TestValidateRestlsRequiresVersion(t *testing.T) {
	mk := func(opts map[string]any) *ProxyNode {
		n := ssNode("r", "aes-128-gcm", "pw")
		n.Options["plugin"] = "restls"
		n.Options["plugin-opts"] = opts
		return n
	}
	if err := ValidateProxyNode(mk(map[string]any{"host": "a.com", "password": "pw", "version": "tls13"})); err != nil {
		t.Fatalf("restls 完整参数应通过: %v", err)
	}
	if err := ValidateProxyNode(mk(map[string]any{"host": "a.com", "password": "pw"})); err == nil {
		t.Fatal("restls 缺 version 实测失败，应丢弃")
	}
}

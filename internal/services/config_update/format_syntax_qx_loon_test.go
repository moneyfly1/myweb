package config_update

import (
	"strings"
	"testing"
)

func genQXT(nodes ...*ProxyNode) string {
	return (&ConfigUpdateService{}).generateQuantumultXConfig(nodes, "https://dy.moneyfly.top")
}

func genLoonT(nodes ...*ProxyNode) string {
	return (&ConfigUpdateService{}).generateLoonConfig(nodes, "https://dy.moneyfly.top")
}

func wantContains(t *testing.T, out, want, what string) {
	t.Helper()
	if !strings.Contains(out, want) {
		t.Fatalf("%s: 生成结果缺少官方语法片段\n  want: %s\n  got:\n%s", what, want, out)
	}
}

// QX vless：官方 sample.conf 要求 method=none、UUID 放在 password 字段（不是 uuid=）。
func TestQuantumultXVlessOfficialSyntax(t *testing.T) {
	n := &ProxyNode{Type: "vless", Name: "qx-vless", Server: "1.2.3.4", Port: 443, UUID: "uuid-1", TLS: true,
		Options: map[string]any{"sni": "s.example.com"}}
	out := genQXT(n)
	wantContains(t, out,
		"vless = 1.2.3.4:443, method=none, password=uuid-1, obfs=over-tls, obfs-host=s.example.com, udp-relay=false, tag=qx-vless",
		"QX vless")
}

// QX vless + Reality：键名必须是 QX 专有的 reality-base64-pubkey / reality-hex-shortid，
// 且必须带 obfs=over-tls（官方：Reality 是替换标准 TLS，不是独立传输）。
func TestQuantumultXVlessRealityKeyNames(t *testing.T) {
	n := &ProxyNode{Type: "vless", Name: "qx-reality", Server: "1.2.3.4", Port: 443, UUID: "uuid-2", TLS: true,
		Options: map[string]any{
			"sni":          "www.example.com",
			"flow":         "xtls-rprx-vision",
			"reality-opts": map[string]any{"public-key": "PBK123", "short-id": "abcd"},
		}}
	out := genQXT(n)
	for _, want := range []string{
		"reality-base64-pubkey=PBK123", "reality-hex-shortid=abcd",
		"obfs=over-tls", "vless-flow=xtls-rprx-vision",
	} {
		wantContains(t, out, want, "QX vless reality")
	}
	// 不能出现 Clash/Loon 风格的键名（两套键名互换必然解析失败）
	if strings.Contains(out, "public-key=PBK123") {
		t.Error("QX 输出里出现 Clash/Loon 的 public-key 键名；QX 只认 reality-base64-pubkey")
	}
}

// QX socks5 用 username=/password= 键；且 QX 不支持 hysteria2/tuic，绝不能出现在输出里。
func TestQuantumultXSocks5AndNoUnsupportedTypes(t *testing.T) {
	socks := &ProxyNode{Type: "socks5", Name: "qx-socks", Server: "5.6.7.8", Port: 1080,
		Options: map[string]any{"username": "u1", "password": "p1"}}
	hy := &ProxyNode{Type: "hysteria2", Name: "hy2", Server: "9.9.9.9", Port: 443, Password: "pw"}
	tuic := &ProxyNode{Type: "tuic", Name: "tuic1", Server: "9.9.9.8", Port: 443, UUID: "u", Password: "p"}
	out := genQXT(socks, hy, tuic)
	wantContains(t, out, "socks5 = 5.6.7.8:1080, username=u1, password=p1", "QX socks5")
	if strings.Contains(out, "hysteria2") || strings.Contains(out, "tuic") {
		t.Errorf("QX 不支持 hysteria2/tuic，输出中不应出现:\n%s", out)
	}
}

// Loon vless：没有加密方式字段（那是 VMess 的）；照抄 VMess 会多一个字段导致解析失败。
func TestLoonVlessOfficialSyntax(t *testing.T) {
	n := &ProxyNode{Type: "vless", Name: "loon-vless", Server: "1.2.3.4", Port: 443, UUID: "uuid-9", TLS: true,
		Options: map[string]any{"sni": "s.example.com"}}
	out := genLoonT(n)
	wantContains(t, out, `loon-vless = vless,1.2.3.4,443,"uuid-9"`, "Loon vless")
	wantContains(t, out, "transport=tcp", "Loon vless transport")
	wantContains(t, out, "over-tls=true", "Loon vless tls")
	if strings.Contains(out, `"uuid-9",chacha20`) || strings.Contains(out, `"uuid-9",auto`) {
		t.Error("Loon vless 被写成了 VMess 结构（多出加密方式字段）")
	}
}

// Loon Reality 用 public-key（带引号）/short-id；hysteria2 密码是位置参数且加引号。
func TestLoonRealityAndHysteria2OfficialSyntax(t *testing.T) {
	r := &ProxyNode{Type: "vless", Name: "loon-reality", Server: "1.2.3.4", Port: 443, UUID: "uuid-r", TLS: true,
		Options: map[string]any{
			"flow":         "xtls-rprx-vision",
			"reality-opts": map[string]any{"public-key": "PBK9", "short-id": "0123abcd"},
		}}
	hy := &ProxyNode{Type: "hysteria2", Name: "loon-hy2", Server: "2.2.2.2", Port: 443, Password: "hy-pw"}
	out := genLoonT(r, hy)
	wantContains(t, out, `public-key="PBK9"`, "Loon reality public-key")
	wantContains(t, out, "short-id=0123abcd", "Loon reality short-id")
	wantContains(t, out, "flow=xtls-rprx-vision", "Loon reality flow")
	wantContains(t, out, `loon-hy2 = hysteria2,2.2.2.2,443,"hy-pw"`, "Loon hysteria2 位置参数密码")
	if strings.Contains(out, "hysteria2,2.2.2.2,443,password=") {
		t.Error("Loon hysteria2 仍在用 password= 键值对写法；官方定义是位置参数")
	}
}

// Loon 不支持 tuic：绝不能渲染（渲染坏行有整份订阅解析失败的风险）。
func TestLoonDoesNotRenderTuic(t *testing.T) {
	tuic := &ProxyNode{Type: "tuic", Name: "loon-tuic", Server: "3.3.3.3", Port: 443, UUID: "u", Password: "p"}
	out := genLoonT(tuic)
	if strings.Contains(strings.ToLower(out), "tuic") {
		t.Errorf("Loon 不支持 tuic，输出中不应出现:\n%s", out)
	}
}

// Loon socks5/http：用户名密码是位置参数（第 4、5 位）；http 开 TLS 要把类型换成 https。
func TestLoonSocks5HttpPositionalCredentials(t *testing.T) {
	socks := &ProxyNode{Type: "socks5", Name: "loon-socks", Server: "5.6.7.8", Port: 1080,
		Options: map[string]any{"username": "u1", "password": "p1"}}
	httpNode := &ProxyNode{Type: "http", Name: "loon-http", Server: "5.6.7.9", Port: 8080, TLS: true,
		Options: map[string]any{"username": "u2", "password": "p2", "sni": "h.example.com"}}
	out := genLoonT(socks, httpNode)
	wantContains(t, out, `loon-socks = socks5,5.6.7.8,1080,u1,"p1"`, "Loon socks5 位置参数")
	wantContains(t, out, `loon-http = https,5.6.7.9,8080,u2,"p2"`, "Loon http+TLS 应为 https")
}

// 新增协议必须整份通过各格式的结构校验层（落盘前的拦截网）。
func TestNewProtocolsPassFormatValidation(t *testing.T) {
	svc := &ConfigUpdateService{}
	nodes := []*ProxyNode{
		{Type: "vless", Name: "v1", Server: "1.2.3.4", Port: 443, UUID: "uuid-1", TLS: true},
		{Type: "vless", Name: "v2", Server: "1.2.3.5", Port: 443, UUID: "uuid-2", TLS: true,
			Options: map[string]any{"reality-opts": map[string]any{"public-key": "PBK", "short-id": "ab"}}},
		{Type: "anytls", Name: "a1", Server: "1.2.3.6", Port: 443, Password: "pw", TLS: true},
		{Type: "socks5", Name: "s1", Server: "1.2.3.7", Port: 1080},
		{Type: "http", Name: "h1", Server: "1.2.3.8", Port: 8080, TLS: true},
		{Type: "ss", Name: "ss1", Server: "1.2.3.9", Port: 8388, Cipher: "aes-256-gcm", Password: "pw"},
		{Type: "hysteria2", Name: "hy1", Server: "1.2.3.10", Port: 443, Password: "pw"},
	}
	cases := []struct {
		f      OutputFormat
		render func([]*ProxyNode) string
	}{
		{FmtQuantumultX, func(ns []*ProxyNode) string { return svc.generateQuantumultXConfig(ns, "https://dy.moneyfly.top") }},
		{FmtLoon, func(ns []*ProxyNode) string { return svc.generateLoonConfig(ns, "https://dy.moneyfly.top") }},
	}
	for _, c := range cases {
		payload := c.render(nodes)
		if err := validateFormatPayload(c.f, payload); err != nil {
			t.Errorf("格式 %s 的产物未通过结构校验: %v\n%s", c.f, err, payload)
		}
	}
}

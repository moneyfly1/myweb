package config_update

import (
	"encoding/base64"
	"strings"
	"testing"
)

// ---------------- 各格式整体校验器的失败用例（证明它们真的能挡住"整份失效"）----------------

func TestFormatValidatorsRejectBrokenPayloads(t *testing.T) {
	cases := []struct {
		name    string
		format  OutputFormat
		payload string
		wantSub string
	}{
		{"Clash 空 proxies", FmtClash, "proxies: []\nproxy-groups: []\n", "proxies 为空"},
		{"Clash 缺 server", FmtClash,
			"proxies:\n  - {name: a, type: ss, port: 8388, cipher: aes-128-gcm, password: pw}\nproxy-groups: []\n", "缺少 server"},
		{"Clash 名字重复", FmtClash,
			"proxies:\n  - {name: a, type: ss, server: 1.2.3.4, port: 8388, cipher: aes-128-gcm, password: pw}\n  - {name: a, type: ss, server: 1.2.3.5, port: 8388, cipher: aes-128-gcm, password: pw}\n", "名称重复"},
		{"Clash 分组成员悬空", FmtClash,
			"proxies:\n  - {name: a, type: ss, server: 1.2.3.4, port: 8388, cipher: aes-128-gcm, password: pw}\nproxy-groups:\n  - {name: P, type: select, proxies: [\"a\", \"b\"]}\n", "成员悬空"},
		{"Clash 空分组", FmtClash,
			"proxies:\n  - {name: a, type: ss, server: 1.2.3.4, port: 8388, cipher: aes-128-gcm, password: pw}\nproxy-groups:\n  - {name: P, type: select, proxies: []}\n", "成员为空"},
		{"Clash type 越界", FmtClash,
			"proxies:\n  - {name: a, type: naive, server: 1.2.3.4, port: 443, password: pw}\n", "不在内核协议白名单"},
		{"base64 不可解码", FmtLinksBase64, "!!!not-base64!!!", "不可解码"},
		{"base64 解出来不是链接", FmtLinksBase64, base64.StdEncoding.EncodeToString([]byte("hello world")), "未知 scheme"},
		{"链接缺端口", FmtLinksPlain, "trojan://pw@a.com#n", "端口非法"},
		{"链接缺 uuid", FmtLinksPlain, "vless://@a.com:443#n", "uuid"},
		{"vmess 负载不可解码", FmtLinksPlain, "vmess://@@@#n", "不可解码"},
		{"JSON 非法", FmtSingBox, `{"outbounds": [`, "不可解析"},
		{"JSON 缺 outbounds", FmtSingBox, `{"log":{}}`, "缺少顶层 outbounds"},
		{"JSON outbound 缺 tag", FmtSingBox, `{"outbounds":[{"type":"direct"}]}`, "缺少 tag"},
		{"JSON outbound 缺 server", FmtSingBox, `{"outbounds":[{"type":"trojan","tag":"a","server_port":443}]}`, "缺少 server"},
		{"Surge 分组悬空", FmtSurge,
			"[Proxy]\nDIRECT = direct\nA = ss, 1.2.3.4, 8388, aes-128-gcm, \"pw\"\n\n[Proxy Group]\nProxy = select, AutoTest, DIRECT, A, B\nAutoTest = url-test, A, url=http://x, interval=300, tolerance=50\n\n[Rule]\nFINAL,Proxy\n", "成员悬空"},
		{"Surge 端口非法", FmtSurge,
			"[Proxy]\nDIRECT = direct\nA = ss, 1.2.3.4, abc, aes-128-gcm, \"pw\"\n\n[Proxy Group]\nProxy = select, DIRECT\n", "端口非法"},
		{"Surge 名字重复", FmtSurge,
			"[Proxy]\nDIRECT = direct\nA = ss, 1.2.3.4, 8388, x, \"p\"\nA = ss, 1.2.3.5, 8388, x, \"p\"\n\n[Proxy Group]\nProxy = select, DIRECT\n", "名称重复"},
		{"QX 缺 tag", FmtQuantumultX,
			"[server_local]\nshadowsocks = 1.2.3.4:8388, method=aes-128-gcm, password=pw\n\n[policy]\n", "缺少 tag"},
		{"QX policy 悬空", FmtQuantumultX,
			"[server_local]\nshadowsocks = 1.2.3.4:8388, method=x, password=pw, tag=A\n\n[policy]\nstatic=G, A, B\n", "成员悬空"},
		{"Loon 字段不足", FmtLoon, "[Proxy]\nA = Shadowsocks,1.2.3.4\n\n[Remote Rule]\n", "字段不足"},
		{"Loon 端口非法", FmtLoon, "[Proxy]\nA = Shadowsocks,1.2.3.4,abc,x\n\n[Remote Rule]\n", "端口非法"},
	}
	for _, c := range cases {
		err := validateFormatPayload(c.format, c.payload)
		if err == nil {
			t.Errorf("%s: 期望被拒绝，实际通过", c.name)
			continue
		}
		if !strings.Contains(err.Error(), c.wantSub) {
			t.Errorf("%s: 期望错误含 %q，实际 %v", c.name, c.wantSub, err)
		}
	}
}

// 各格式正常产物必须通过（避免校验器本身误杀）
func TestFormatValidatorsAcceptGoodPayloads(t *testing.T) {
	svc, _ := setupSelfCheckTest(t, nil)
	nodes := []*ProxyNode{
		mkSS("香港-ss"),
		{Name: "美国-trojan", Type: "trojan", Server: "a.example.com", Port: 443, Password: "pw", TLS: true, Options: map[string]any{}},
		{Name: "日本-vmess", Type: "vmess", Server: "b.example.com", Port: 443, UUID: "11111111-2222-4333-8444-555555555555", Cipher: "auto", Network: "tcp", Options: map[string]any{"alterId": 0}},
		{Name: "新加坡-vless", Type: "vless", Server: "c.example.com", Port: 443, UUID: "11111111-2222-4333-8444-555555555555", Options: map[string]any{"encryption": "none"}},
	}
	ctx := &SubscriptionContext{Status: StatusNormal}
	all := svc.addInfoNodes(nodes, ctx)

	for _, f := range []OutputFormat{FmtClash, FmtLinksBase64, FmtLinksPlain, FmtSurge, FmtSingBox, FmtQuantumultX, FmtLoon} {
		var payload string
		switch f {
		case FmtClash:
			payload = svc.generateClashYAML(all, ctx)
		case FmtLinksBase64, FmtLinksPlain:
			var links []string
			for _, n := range nodes {
				links = append(links, svc.nodeToLink(n))
			}
			payload = strings.Join(links, "\n")
			if f == FmtLinksBase64 {
				payload = base64.StdEncoding.EncodeToString([]byte(payload))
			}
		case FmtSurge:
			payload = svc.generateSurgeConfig(all, "")
		case FmtSingBox:
			payload = svc.generateSingBoxConfig(all)
		case FmtQuantumultX:
			payload = svc.generateQuantumultXConfig(all, "")
		case FmtLoon:
			payload = svc.generateLoonConfig(all, "")
		}
		if err := validateFormatPayload(f, payload); err != nil {
			t.Errorf("格式 %s 的正常产物被误判为不合法: %v", f, err)
		}
	}
}

// 逐格式隔离：坏节点只从对应格式里消失，其余节点完整（用各格式的**真实生成器**）
func TestPerFormatIsolationDropsOnlyBadNode(t *testing.T) {
	svc, _ := setupSelfCheckTest(t, nil) // 需要 s.db（generateClashYAML 会读模板配置）
	ctx := &SubscriptionContext{Status: StatusNormal}

	good := []*ProxyNode{
		mkSS("好节点-ss"),
		{Name: "好节点-trojan", Type: "trojan", Server: "a.example.com", Port: 443, Password: "pw", TLS: true, Options: map[string]any{}},
		{Name: "好节点-vmess", Type: "vmess", Server: "b.example.com", Port: 443, UUID: "11111111-2222-4333-8444-555555555555", Cipher: "auto", Network: "tcp", Options: map[string]any{"alterId": 0}},
	}
	naiveNode := &ProxyNode{Name: "坏-naive", Type: "naive", Server: "9.9.9.9", Port: 443, Password: "pw", Options: map[string]any{}}
	nodes := append(append([]*ProxyNode{}, good...), naiveNode)

	type tc struct {
		f      OutputFormat
		render RenderFunc
	}
	cases := []tc{
		{FmtClash, func(ns []*ProxyNode) string { return svc.generateClashYAML(svc.addInfoNodes(ns, ctx), ctx) }},
		{FmtLinksPlain, func(ns []*ProxyNode) string {
			var ls []string
			for _, n := range ns {
				ls = append(ls, svc.NodeToLink(n))
			}
			return strings.Join(ls, "\n")
		}},
		{FmtLinksBase64, func(ns []*ProxyNode) string {
			var ls []string
			for _, n := range ns {
				ls = append(ls, svc.NodeToLink(n))
			}
			return base64.StdEncoding.EncodeToString([]byte(strings.Join(ls, "\n")))
		}},
		{FmtSurge, func(ns []*ProxyNode) string { return svc.generateSurgeConfig(ns, "") }},
		{FmtSingBox, func(ns []*ProxyNode) string { return svc.generateSingBoxConfig(ns) }},
		{FmtQuantumultX, func(ns []*ProxyNode) string { return svc.generateQuantumultXConfig(ns, "") }},
		{FmtLoon, func(ns []*ProxyNode) string { return svc.generateLoonConfig(ns, "") }},
	}
	for _, c := range cases {
		r := svc.buildVerifiedPayload(c.f, nodes, c.render, "tok-iso-"+string(c.f))
		// ① 整体校验必须通过
		if err := validateFormatPayload(c.f, r.Payload); err != nil {
			t.Errorf("格式 %s: 隔离后整体校验仍失败: %v", c.f, err)
		}
		// ② naive 必须被剔除且原因可读
		found := false
		for _, d := range r.Dropped {
			if d.Name == "坏-naive" {
				found = true
				if d.Reason != "format-unsupported-type" {
					t.Errorf("格式 %s: naive 原因应为 format-unsupported-type，实际 %s", c.f, d.Reason)
				}
			}
		}
		if !found {
			t.Errorf("格式 %s: naive 未被剔除", c.f)
		}
		// ③ 好节点一个都不能少
		for _, d := range r.Dropped {
			if strings.HasPrefix(d.Name, "好节点") {
				t.Errorf("格式 %s: 好节点被误剔 %+v", c.f, d)
			}
		}
		if c.f != FmtLinksBase64 && c.f != FmtLinksPlain {
			for _, n := range good {
				if !strings.Contains(r.Payload, n.Name) {
					t.Errorf("格式 %s: 好节点 %s 未出现在产物里", c.f, n.Name)
				}
			}
		}
		if strings.Contains(r.Payload, "naive") {
			t.Errorf("格式 %s: 产物里仍出现 naive", c.f)
		}
		t.Logf("格式 %-14s 节点数=%d 剔除=%v", c.f, r.NodeCount, r.Dropped)
	}
}

// 名字含逗号的节点：Surge 的分组按逗号切分，极易产生"悬空成员"
// —— 正确做法是**修正名字**（替换逗号）而不是丢弃节点
func TestCommaInNameIsSanitizedNotDropped(t *testing.T) {
	svc, _ := setupSelfCheckTest(t, nil)
	nodes := []*ProxyNode{
		{Name: "香港, 节点1", Type: "ss", Server: "1.2.3.4", Port: 8388, Cipher: "aes-128-gcm", Password: "pw", Options: map[string]any{}},
		mkSS("好节点"),
	}
	for _, c := range []struct {
		f      OutputFormat
		render RenderFunc
	}{
		{FmtSurge, func(ns []*ProxyNode) string { return svc.generateSurgeConfig(ns, "") }},
		{FmtQuantumultX, func(ns []*ProxyNode) string { return svc.generateQuantumultXConfig(ns, "") }},
		{FmtLoon, func(ns []*ProxyNode) string { return svc.generateLoonConfig(ns, "") }},
	} {
		r := svc.buildVerifiedPayload(c.f, nodes, c.render, "tok-comma-"+string(c.f))
		if err := validateFormatPayload(c.f, r.Payload); err != nil {
			t.Errorf("格式 %s: 含逗号名字导致整体校验失败: %v", c.f, err)
		}
		for _, d := range r.Dropped {
			t.Errorf("格式 %s: 含逗号名字应被「修正」而不是丢弃，实际丢弃 %+v", c.f, d)
		}
		if !strings.Contains(r.Payload, "好节点") {
			t.Errorf("格式 %s: 好节点丢失", c.f)
		}
	}
}

// naive 对每个格式都不支持（它不在任何渲染白名单里）→ 必须逐个格式剔除而不是原样下发
func TestNaiveNodeExcludedFromEveryFormat(t *testing.T) {
	for f, types := range formatRenderTypes {
		if types["naive"] {
			t.Errorf("格式 %s 不应支持 naive（内核不支持该协议）", f)
		}
	}
	if linkRenderTypes["naive"] {
		t.Error("链接格式不应支持 naive")
	}
}

// 凭据规范化对所有格式同时生效（解析/入库层已规范化，出配置再兜底一次）
func TestCredentialNormalizationBenefitsEveryFormat(t *testing.T) {
	svc, _ := setupSelfCheckTest(t, nil)
	seg := "QUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBA/A="
	polluted := strings.ReplaceAll(seg, "/", "%2F") + "%3A" + strings.ReplaceAll(seg, "/", "%2F")
	decoded := seg + ":" + seg

	n := &ProxyNode{Name: "荷兰-ss2022", Type: "ss", Server: "1.2.3.4", Port: 8388,
		Cipher: "2022-blake3-aes-256-gcm", Password: polluted, Options: map[string]any{}}

	for _, f := range []OutputFormat{FmtClash, FmtLinksBase64, FmtLinksPlain, FmtSurge, FmtSingBox, FmtQuantumultX, FmtLoon} {
		var payload string
		r := svc.buildVerifiedPayload(f, []*ProxyNode{n}, func(ns []*ProxyNode) string {
			for _, x := range ns {
				if x.Name == "荷兰-ss2022" && x.Password != decoded {
					t.Errorf("格式 %s 渲染时凭据未被规范化: 长度=%d", f, len(x.Password))
				}
			}
			var b strings.Builder
			for _, x := range ns {
				b.WriteString((&ConfigUpdateService{}).NodeToLink(x) + "\n")
			}
			return base64.StdEncoding.EncodeToString([]byte(b.String()))
		}, "tok-cred-"+string(f))
		payload = r.Payload
		if strings.Contains(payload, "%2F") {
			t.Errorf("格式 %s 的下发内容里仍残留 %%2F 转义", f)
		}
	}
}

// 名字含逗号的节点：Surge 的分组按逗号切分，极易产生"悬空成员"
// —— 这是"一个节点让整份配置失效"最典型的形态之一，必须被挡住或修正
func TestCommaInNameIsHandledForLineFormats(t *testing.T) {
	setupSelfCheckTest(t, nil)
	svc := &ConfigUpdateService{}
	nodes := []*ProxyNode{mkSS("香港, 节点1"), mkSS("好节点")}

	for _, f := range []struct {
		format OutputFormat
		render RenderFunc
	}{
		{FmtSurge, func(ns []*ProxyNode) string { return svc.generateSurgeConfig(ns, "") }},
		{FmtQuantumultX, func(ns []*ProxyNode) string { return svc.generateQuantumultXConfig(ns, "") }},
		{FmtLoon, func(ns []*ProxyNode) string { return svc.generateLoonConfig(ns, "") }},
	} {
		r := svc.buildVerifiedPayload(f.format, nodes, f.render, "tok-comma-"+string(f.format))
		if err := validateFormatPayload(f.format, r.Payload); err != nil {
			t.Errorf("格式 %s: 含逗号名字后整体校验失败: %v", f.format, err)
		}
		// 理想情况是被"修正"而不是被丢弃；若丢弃则必须记录原因
		for _, d := range r.Dropped {
			t.Logf("格式 %s: 丢弃 %s 原因=%s（含逗号名字）", f.format, d.Name, d.Reason)
		}
	}
}

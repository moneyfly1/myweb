package config_update

import (
	"strings"
	"testing"
)

// hy2:// 是 Hysteria2 的通用简写 scheme，大量订阅源（如 ExampleProvider）使用它。
// 此前采集链路只识别 hysteria2://，导致 hy2:// 节点被静默丢弃。

const testHy2Link = "hy2://00000000-0000-4000-8000-000000000073@203.0.113.82:35000?sni=api.push.apple.com&insecure=0#example-node-0001"

// TestParseNodeLink_Hy2Scheme 验证 hy2:// 能被解析为 hysteria2 节点，且字段正确。
func TestParseNodeLink_Hy2Scheme(t *testing.T) {
	node, err := ParseNodeLink(testHy2Link)
	if err != nil {
		t.Fatalf("hy2:// 应可解析，实际报错: %v", err)
	}
	if node.Type != "hysteria2" {
		t.Errorf("节点类型应为 hysteria2，实际 %q", node.Type)
	}
	if node.Server != "203.0.113.82" {
		t.Errorf("服务器地址错误: %q", node.Server)
	}
	if node.Port != 35000 {
		t.Errorf("端口错误: %d", node.Port)
	}
	if node.Password != "00000000-0000-4000-8000-000000000073" {
		t.Errorf("密码/认证错误: %q", node.Password)
	}
	if !node.TLS {
		t.Error("Hysteria2 节点应启用 TLS")
	}
	if sni, _ := node.Options["servername"].(string); sni != "api.push.apple.com" {
		t.Errorf("sni 应写入 servername，实际 %v", node.Options["servername"])
	}
	if skip, _ := node.Options["skip-cert-verify"].(bool); skip {
		t.Error("insecure=0 时 skip-cert-verify 应为 false")
	}
	if node.Name != "example-node-0001" {
		t.Errorf("节点名应取 fragment 并 URL 解码，实际 %q", node.Name)
	}
}

// TestParseNodeLink_Hy2Insecure 验证 insecure=1 时跳过证书校验。
func TestParseNodeLink_Hy2Insecure(t *testing.T) {
	node, err := ParseNodeLink("hy2://pass@1.2.3.4:443?sni=a.example.com&insecure=1#N")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if skip, _ := node.Options["skip-cert-verify"].(bool); !skip {
		t.Error("insecure=1 时 skip-cert-verify 应为 true")
	}
}

// TestExtractNodeLinks_Hy2Scheme 验证采集提取阶段不丢弃 hy2:// 节点，
// 并与 hysteria2:// 混合出现时都能提取。
func TestExtractNodeLinks_Hy2Scheme(t *testing.T) {
	svc := &ConfigUpdateService{}
	content := strings.Join([]string{
		testHy2Link,
		"hysteria2://pwd@10.0.0.1:8443?sni=x.com#H2",
		"vless://uuid-1234@1.2.3.4:443?security=tls&sni=a.com#V",
		"hy2://pwd2@10.0.0.2:9000?sni=y.com#H2b",
	}, "\n")

	links := svc.extractNodeLinks(content)

	var hy2, hysteria2, vless int
	for _, l := range links {
		switch {
		case strings.HasPrefix(l, "hy2://"):
			hy2++
		case strings.HasPrefix(l, "hysteria2://"):
			hysteria2++
		case strings.HasPrefix(l, "vless://"):
			vless++
		}
	}
	if hy2 != 2 {
		t.Errorf("应提取 2 个 hy2:// 节点，实际 %d（提取结果: %v）", hy2, links)
	}
	if hysteria2 != 1 {
		t.Errorf("应提取 1 个 hysteria2:// 节点，实际 %d", hysteria2)
	}
	if vless != 1 {
		t.Errorf("应提取 1 个 vless:// 节点，实际 %d", vless)
	}
	if total := len(links); total != 4 {
		t.Errorf("应提取 4 个节点，实际 %d", total)
	}
}

// TestExtractNodeLinks_Hy2Base64Subscription 验证整包 base64 订阅（真实场景）解码后
// 的 hy2:// 节点能被完整提取，不遗漏任何协议。
func TestExtractNodeLinks_Hy2Base64Subscription(t *testing.T) {
	svc := &ConfigUpdateService{}
	plain := strings.Join([]string{
		testHy2Link,
		"hy2://pwd@203.0.113.156:60000?sni=www.apple.com&insecure=0#example-node-0002",
		"vless://uuid-1234@203.0.113.154:443?security=reality&sni=cdn.com&pbk=KEY&sid=abc#Vless-US",
	}, "\n")

	// 采集入口会先对 base64 订阅解码，这里直接模拟解码后的内容
	links := svc.extractNodeLinks(plain)
	hy2Count, vlessCount := 0, 0
	for _, l := range links {
		if strings.HasPrefix(l, "hy2://") {
			hy2Count++
		} else if strings.HasPrefix(l, "vless://") {
			vlessCount++
		}
	}
	if hy2Count != 2 || vlessCount != 1 {
		t.Fatalf("base64 订阅应提取 hy2=2 vless=1，实际 hy2=%d vless=%d", hy2Count, vlessCount)
	}

	// 每个提取出的 hy2 链接都应能成功解析入库（避免"提取成功但解析失败"）
	for _, l := range links {
		if !strings.HasPrefix(l, "hy2://") {
			continue
		}
		node, err := ParseNodeLink(l)
		if err != nil {
			t.Errorf("提取出的 hy2 链接解析失败: %v", err)
			continue
		}
		if node.Type != "hysteria2" || node.Server == "" || node.Port == 0 {
			t.Errorf("hy2 节点解析不完整: type=%s server=%s port=%d", node.Type, node.Server, node.Port)
		}
	}
}

// TestContainsNodeLinks_Hy2 验证订阅内容探测能识别 hy2://（决定是否走链接解析分支）。
func TestContainsNodeLinks_Hy2(t *testing.T) {
	if !containsNodeLinks("hy2://pass@1.2.3.4:443#N") {
		t.Error("containsNodeLinks 应识别 hy2://")
	}
	if !containsNodeLinks("hysteria2://pass@1.2.3.4:443#N") {
		t.Error("containsNodeLinks 应识别 hysteria2://")
	}
	if containsNodeLinks("这不是订阅内容") {
		t.Error("普通文本不应被误判为节点链接")
	}
}

// TestIsValidNodeLink_Hy2 验证 hy2:// 在链接有效性校验上与 hysteria2:// 完全等价
// （两者共用同一分支，不引入额外的取舍差异）。
func TestIsValidNodeLink_Hy2(t *testing.T) {
	svc := &ConfigUpdateService{}
	if !svc.isValidNodeLink(testHy2Link) {
		t.Error("有效的 hy2:// 链接应通过校验")
	}

	// hy2 与 hysteria2 对同一输入必须给出一致结果（等价 scheme 别名）
	pairs := [][2]string{
		{testHy2Link, strings.Replace(testHy2Link, "hy2://", "hysteria2://", 1)},
		{"hy2://passonly#N", "hysteria2://passonly#N"},
		{"hy2://p@1.2.3.4:443#N", "hysteria2://p@1.2.3.4:443#N"},
	}
	for _, p := range pairs {
		gotHy2 := svc.isValidNodeLink(p[0])
		gotH2 := svc.isValidNodeLink(p[1])
		if gotHy2 != gotH2 {
			t.Errorf("校验结果不一致: %q=%v 但 %q=%v", p[0], gotHy2, p[1], gotH2)
		}
	}
}

package config_update

import (
	"strings"
	"testing"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// ============================================================================
// 节点名里的非法 UTF-8 → 整份 YAML 解析失败
//
// 现网真实来源：某 ssr 链接的 remarks 经 base64 解码后含 0xFD 字节（上游用非 UTF-8
// 编码写备注），入库后 nodes.name 存下 b"$\xfd5"（本机实测 id=17856、
// source=xiaohuojian_shadowrocket_nodes.txt），渲染期 yaml 解析器报
//   yaml: invalid leading UTF-8 octet
// 导致 nodeToYAMLFlowNode 失败、该节点被跳过，而它仍在 proxy-groups 的成员表里
// → 悬空引用。客户端侧表现是 `Parse config error: yaml: line N: ...`，整份订阅起不来。
//
// 两道防线：
//   ① 采集期 NormalizeNodeName 净化（名称是展示字段，净化比丢弃更合理）
//   ② 渲染期 escapeYAMLString/escapeYAMLKey 兜底净化（所有标量输出的唯一漏斗，
//      不论数据来自采集/手工导入/自定义节点都不可能再污染产物）
// ============================================================================

// invalidUTF8Name 复刻实盘那个坏名字：'$' + 0xFD + '5'
const invalidUTF8Name = "$\xfd5"

func TestNormalizeNodeNameFixesInvalidUTF8(t *testing.T) {
	if utf8.ValidString(invalidUTF8Name) {
		t.Fatal("测试夹具本身应是非法 UTF-8")
	}
	n := &ProxyNode{Name: invalidUTF8Name, Type: "ssr", Server: "cn00.somethingstranges.com", Port: 8802}
	corrected, detail := NormalizeNodeName(n)
	if !corrected {
		t.Fatal("非法 UTF-8 名字未被净化")
	}
	if !utf8.ValidString(n.Name) {
		t.Errorf("净化后仍不是合法 UTF-8: %q", n.Name)
	}
	if !strings.Contains(n.Name, "\uFFFD") {
		t.Errorf("净化后应含替换字符 U+FFFD: %q", n.Name)
	}
	if detail == "" {
		t.Error("应给出可读的修正说明")
	}
	// 合法名字必须原样返回，不能误报
	ok := &ProxyNode{Name: "闪连-阿根廷-布宜诺斯艾利斯-1.2.3.4:443", Type: "anytls"}
	if c, _ := NormalizeNodeName(ok); c {
		t.Error("合法 UTF-8 名字被误判为需要净化")
	}
}

// 渲染期兜底：即使坏名字绕过采集期直接进来，产物也必须是合法 UTF-8 的合法 YAML。
func TestEscapeYAMLStringSanitizesInvalidUTF8(t *testing.T) {
	svc := &ConfigUpdateService{}
	got := svc.escapeYAMLString(invalidUTF8Name)
	if !utf8.ValidString(got) {
		t.Fatalf("escapeYAMLString 输出非法 UTF-8: %q", got)
	}
	if !strings.Contains(got, "\uFFFD") {
		t.Errorf("应替换为 U+FFFD: %q", got)
	}
	if k := svc.escapeYAMLKey(invalidUTF8Name); !utf8.ValidString(k) {
		t.Errorf("escapeYAMLKey 输出非法 UTF-8: %q", k)
	}
}

// 端到端：坏名字节点 + 正常节点渲染成 Clash，两条生成路径都必须产出可解析的 YAML。
func TestClashYAMLWithInvalidUTF8NameStillParses(t *testing.T) {
	svc := &ConfigUpdateService{}
	bad := &ProxyNode{
		Name: invalidUTF8Name, Type: "ssr", Server: "cn00.somethingstranges.com", Port: 8802,
		Password: "passwd", Cipher: "chacha20-ietf",
		Options: map[string]any{"obfs": "http_simple", "protocol": "origin"},
	}
	good := mkSS("正常节点")
	nodes := []*ProxyNode{bad, good}

	// 路径 A：默认生成器（手工拼接 flow 文本，最容易被坏字节打穿）
	out := svc.generateDefaultClashYAML(nodes, []string{bad.Name, good.Name}, []string{bad.Name, good.Name}, "测试")
	if !utf8.ValidString(out) {
		t.Fatalf("generateDefaultClashYAML 产物含非法 UTF-8")
	}
	var doc map[string]interface{}
	if err := yaml.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("generateDefaultClashYAML 产物 yaml.v3 解析失败: %v", err)
	}
	if n := len(doc["proxies"].([]interface{})); n != 2 {
		t.Errorf("proxies = %d, want 2", n)
	}

	// 路径 B：逐节点 flow 节点（模板路径用的就是它）
	for _, p := range nodes {
		node, err := svc.nodeToYAMLFlowNode(p)
		if err != nil {
			t.Errorf("nodeToYAMLFlowNode(%q) 失败: %v", p.Name, err)
			continue
		}
		node.Style = yaml.FlowStyle
		if b, err := yaml.Marshal(&node); err != nil || !utf8.Valid(b) {
			t.Errorf("flow 节点序列化失败/非法: %v %q", err, b)
		}
	}
}

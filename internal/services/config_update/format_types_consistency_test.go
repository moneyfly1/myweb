package config_update

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// caseLiteralsOf 用 go/ast 解析 config_update.go，取出指定函数里所有 switch
// 的 case 子句中的字符串字面量（**正确处理多值 case**，如 case "a", "b":）。
func caseLiteralsOf(t *testing.T, funcName string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "config_update.go", nil, 0)
	if err != nil {
		t.Fatalf("解析 config_update.go 失败: %v", err)
	}
	out := map[string]bool{}
	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Name.Name != funcName || fn.Body == nil {
			return true
		}
		found = true
		ast.Inspect(fn.Body, func(m ast.Node) bool {
			cc, ok := m.(*ast.CaseClause)
			if !ok {
				return true
			}
			for _, e := range cc.List {
				bl, ok := e.(*ast.BasicLit)
				if !ok || bl.Kind != token.STRING {
					continue
				}
				if v, err := strconv.Unquote(bl.Value); err == nil && v != "" {
					out[strings.ToLower(v)] = true
				}
			}
			return true
		})
		return true
	})
	if !found {
		t.Fatalf("在 config_update.go 里找不到函数 %s", funcName)
	}
	return out
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// formatRenderTypes 必须与生成器源码里的 case 完全一致。
// 这条用例就是为了防止"手工抄类型表抄漏"再次把节点静默过滤掉
// （曾把 sing-box / Surge 的 hysteria2 抄漏，93 个节点被误剔除）。
func TestFormatRenderTypesMatchGeneratorCases(t *testing.T) {
	cases := []struct {
		format OutputFormat
		funcs  []string
	}{
		{FmtSurge, []string{"proxyNodeToSurgeLine"}},
		{FmtLoon, []string{"generateLoonConfig"}},
		{FmtQuantumultX, []string{"generateQuantumultXConfig"}},
		{FmtSingBox, []string{"generateSingBoxConfig"}},
	}
	for _, c := range cases {
		actual := map[string]bool{}
		for _, fn := range c.funcs {
			for k := range caseLiteralsOf(t, fn) {
				actual[k] = true
			}
		}
		// 生成器里会出现一些非协议字面量（如 gost/其他 switch），只比较协议白名单内的键
		filtered := map[string]bool{}
		for k := range actual {
			if mihomoSupportedNodeTypes[NormalizeNodeType(k)] || k == "socks" || k == "hy2" {
				filtered[k] = true
			}
		}
		declared := formatRenderTypes[c.format]
		for k := range filtered {
			if !declared[k] {
				t.Errorf("格式 %s: 生成器支持 %q，但 formatRenderTypes 里没有 → 该类型节点会被静默剔除。请补进表。", c.format, k)
			}
		}
		for k := range declared {
			if !filtered[k] {
				t.Errorf("格式 %s: formatRenderTypes 声明支持 %q，但生成器的 case 里没有 → 会把节点渲染成空/损坏。请修正表。", c.format, k)
			}
		}
		// 校验器必须接受生成器会产出的每一种类型，否则会把合法节点判成 format-invalid-line
		if c.format == FmtSurge {
			for k := range declared {
				if !surgeTypeOK(k) {
					t.Errorf("Surge: 能力表允许 %q，但 surgeTypeOK 白名单不认 → 该校验会误剔合法节点", k)
				}
			}
		}
		t.Logf("格式 %-14s 生成器 case=%v", c.format, keys(filtered))
	}
}

// linkRenderTypes 必须与 nodeToLink 的 case 一致
func TestLinkRenderTypesMatchNodeToLink(t *testing.T) {
	actual := caseLiteralsOf(t, "nodeToLink")
	filtered := map[string]bool{}
	for k := range actual {
		if mihomoSupportedNodeTypes[NormalizeNodeType(k)] || k == "socks" || k == "hy2" {
			filtered[k] = true
		}
	}
	for k := range filtered {
		if !linkRenderTypes[k] {
			t.Errorf("nodeToLink 支持 %q，但 linkRenderTypes 里没有 → 该类型链接会被静默剔除", k)
		}
	}
	for k := range linkRenderTypes {
		if !filtered[k] {
			t.Errorf("linkRenderTypes 声明 %q，但 nodeToLink 没有对应 case", k)
		}
	}
	t.Logf("linkRenderTypes 与 nodeToLink 一致；case=%v", keys(filtered))
}

// 回归：93 个 hysteria2 节点必须出现在 sing-box 与 Surge 产物里（曾被我自己的过滤层剔除）
func TestHysteria2NotFilteredFromSingBoxOrSurge(t *testing.T) {
	svc, _ := setupSelfCheckTest(t, nil)
	ctx := &SubscriptionContext{Status: StatusNormal}
	nodes := []*ProxyNode{
		mkSS("好节点-ss"),
		{Name: "hy2节点", Type: "hysteria2", Server: "5.5.5.5", Port: 443,
			Password: "authpw", TLS: true, Options: map[string]any{"sni": "a.example.com", "skip-cert-verify": false}},
	}
	all := svc.addInfoNodes(nodes, ctx)

	for _, c := range []struct {
		f      OutputFormat
		render RenderFunc
		mark   string
	}{
		{FmtSingBox, func(ns []*ProxyNode) string { return svc.generateSingBoxConfig(ns) }, `"type": "hysteria2"`},
		{FmtSurge, func(ns []*ProxyNode) string { return svc.generateSurgeConfig(ns, "") }, "hy2节点"},
	} {
		r := svc.buildVerifiedPayload(c.f, all, c.render, "hy2-"+string(c.f))
		for _, d := range r.Dropped {
			if strings.Contains(d.Name, "hy2") {
				t.Errorf("格式 %s: hysteria2 节点被剔除 %+v（生成器本来是支持的）", c.f, d)
			}
		}
		if !strings.Contains(r.Payload, c.mark) {
			t.Errorf("格式 %s: 产物里找不到 hysteria2（期望含 %q）", c.f, c.mark)
		}
		if err := validateFormatPayload(c.f, r.Payload); err != nil {
			t.Errorf("格式 %s 整体校验失败: %v", c.f, err)
		}
		t.Logf("格式 %-14s hysteria2 已包含 ✓", c.f)
	}
}

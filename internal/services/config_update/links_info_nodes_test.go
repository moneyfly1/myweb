package config_update

import (
	"encoding/base64"
	"net/url"
	"strings"
	"testing"
)

// linkFragmentName 取分享链接的 #fragment 名字（已 URL 解码）
func linkFragmentName(t *testing.T, link string) string {
	t.Helper()
	i := strings.LastIndex(link, "#")
	if i < 0 {
		return ""
	}
	name, err := url.QueryUnescape(link[i+1:])
	if err != nil {
		t.Fatalf("链接 fragment 无法解码: %v (link=%s)", err, link)
	}
	return name
}

func decodeLinksPayload(t *testing.T, payload string) []string {
	t.Helper()
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("链接订阅不是合法 base64: %v", err)
	}
	var out []string
	for _, line := range strings.Split(strings.TrimSpace(string(decoded)), "\n") {
		if strings.TrimSpace(line) != "" {
			out = append(out, strings.TrimSpace(line))
		}
	}
	return out
}

// 回归：扫码订阅（二维码内容是 sub:// + base64(universal_url)）与 v2rayN / 小火箭走的是
// base64 链接格式，面板提示节点（📢官网 / ⏰到期 / 📱设备）必须一起下发并排在最前面。
// 曾经这里用 isPlaceholderInfoNode 跳过提示节点，导致扫码或复制「通用链接」订阅后，
// 客户端里最前面几个提示节点消失（而 Clash 链接是正常的）。
func TestRenderLinksPayloadKeepsInfoNodes(t *testing.T) {
	svc := &ConfigUpdateService{siteURL: "https://example.com"}
	ctx := &SubscriptionContext{Status: StatusNormal, DeviceLimit: 3, CurrentDevices: 1}

	real := []*ProxyNode{{Name: "香港 01", Type: "ss", Server: "hk.example.com", Port: 443, Cipher: "aes-128-gcm", Password: "pw"}}
	nodes := svc.addInfoNodes(real, ctx)

	links := decodeLinksPayload(t, svc.renderLinksPayload(nodes, false))
	if len(links) != 4 {
		t.Fatalf("期望 3 个提示节点 + 1 个真实节点，实际 %d 条链接: %v", len(links), links)
	}

	names := make([]string, 0, len(links))
	for _, l := range links {
		names = append(names, linkFragmentName(t, l))
	}
	for i, want := range []string{"📢 官网: https://example.com", "⏰ 到期: 无限期", "📱 设备: 1/3"} {
		if !strings.HasPrefix(names[i], strings.Split(want, ":")[0]) || !strings.Contains(names[i], strings.SplitN(want, ": ", 2)[1]) {
			t.Fatalf("第 %d 条不是提示节点 %q，实际 %q（全部：%v）", i+1, want, names[i], names)
		}
	}
	if names[3] != "香港 01" {
		t.Fatalf("真实节点没排在提示节点之后：%v", names)
	}
	if err := validateBase64Links(svc.renderLinksPayload(nodes, false)); err != nil {
		t.Fatalf("含提示节点的链接订阅未通过格式校验: %v", err)
	}
}

// 客服节点仅在配置了客服 QQ 时下发；配置后也应出现在链接订阅里。
func TestRenderLinksPayloadIncludesSupportNodeWhenConfigured(t *testing.T) {
	svc := &ConfigUpdateService{siteURL: "https://example.com", supportQQ: "12345678"}
	ctx := &SubscriptionContext{Status: StatusNormal, DeviceLimit: 2, CurrentDevices: 0}

	links := decodeLinksPayload(t, svc.renderLinksPayload(svc.addInfoNodes(nil, ctx), false))
	if len(links) != 4 {
		t.Fatalf("期望 4 个提示节点（含客服），实际 %d 条: %v", len(links), links)
	}
	if name := linkFragmentName(t, links[3]); name != "💬 客服: 12345678" {
		t.Fatalf("客服提示节点缺失或顺序不对：%q", name)
	}
}

// 未配置客服 QQ 时不应出现客服节点（保持既有行为）。
func TestRenderLinksPayloadWithoutSupportQQ(t *testing.T) {
	svc := &ConfigUpdateService{siteURL: "https://example.com"}
	links := decodeLinksPayload(t, svc.renderLinksPayload(svc.addInfoNodes(nil, &SubscriptionContext{Status: StatusNormal}), false))
	if len(links) != 3 {
		t.Fatalf("期望 3 个提示节点，实际 %d 条: %v", len(links), links)
	}
}

// 提示节点的链接本身必须是合法分享链接（回归保护：不要为了「让校验通过」而把提示节点剔掉，
// 正确做法是让校验层认识它们——format_verify.go 已有对应分支）。
func TestPlaceholderInfoNodesAreValidLinks(t *testing.T) {
	svc := &ConfigUpdateService{siteURL: "https://example.com", supportQQ: "12345678"}
	for _, n := range svc.addInfoNodes(nil, &SubscriptionContext{Status: StatusNormal, DeviceLimit: 1}) {
		link := svc.nodeToLink(n)
		if link == "" {
			t.Fatalf("提示节点 %q 生成了空链接", n.Name)
		}
		if err := validateOneLink(link); err != nil {
			t.Fatalf("提示节点 %q 的链接未通过校验: %v（link=%s）", n.Name, err, link)
		}
	}
}

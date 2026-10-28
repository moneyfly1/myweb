package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"cboard-go/internal/models"

	"gorm.io/gorm"
)

// 订阅失效时的"占位节点"配置。
//
// 背景（2026-10-28 用户反馈）：重置订阅地址后，旧地址在 Clash 端点返回的是
// base64(vmess://…) 文本，Clash 解析失败 → 客户端**保留旧节点**，用户以为还能用。
// 正确做法：失效地址也要返回**该客户端能解析的假节点**——客户端成功更新后
// 旧真实节点被覆盖，只剩提示节点，用户照着节点名去官网复制新地址。
//
// 约定：
//   - 节点地址用不可解析的保留域名 `subscription-expired.invalid`，端口 1/2…，
//     客户端能加载但永远连不通（"虚假节点"）；
//   - 提示语放节点名里（客户端列表直接可见），不依赖额外弹窗。

const placeholderServer = "subscription-expired.invalid"

// placeholderLabels 提示语（同时作为假节点的名字）。
// baseURL 只取主机部分，节点名太长会被客户端截断。
func placeholderLabels(baseURL, message string) []string {
	host := strings.TrimSpace(baseURL)
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimSuffix(host, "/")
	if host == "" {
		host = "官网"
	}
	if r := []rune(host); len(r) > 40 {
		host = string(r[:40])
	}
	if strings.TrimSpace(message) == "" {
		message = "订阅已失效"
	}
	if r := []rune(message); len(r) > 60 {
		message = string(r[:57]) + "..."
	}
	return []string{
		fmt.Sprintf("⚠️ %s", message),
		fmt.Sprintf("👉 请到 %s 重新复制订阅地址", host),
	}
}

// yamlQuote 双引号包裹并转义（节点名里有冒号、逗号等 YAML 特殊字符）
func yamlQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

// GeneratePlaceholderClashConfig 生成 Clash / mihomo 可解析的占位配置。
func GeneratePlaceholderClashConfig(baseURL, message string) string {
	labels := placeholderLabels(baseURL, message)
	var b strings.Builder
	b.WriteString("# 订阅已失效：以下为提示节点（虚假节点，无法连接）\n")
	b.WriteString("mixed-port: 7890\nallow-lan: true\nmode: rule\nlog-level: warning\nexternal-controller: 127.0.0.1:9090\n\n")
	b.WriteString("proxies:\n")
	for i, label := range labels {
		b.WriteString(fmt.Sprintf("  - name: %s\n    type: vmess\n    server: %s\n    port: %d\n"+
			"    uuid: 00000000-0000-0000-0000-000000000000\n    alterId: 0\n    cipher: auto\n    udp: true\n",
			yamlQuote(label), placeholderServer, i+1))
	}
	b.WriteString("\nproxy-groups:\n  - name: \"🚀 节点选择\"\n    type: select\n    proxies:\n")
	for _, label := range labels {
		b.WriteString(fmt.Sprintf("      - %s\n", yamlQuote(label)))
	}
	b.WriteString("      - DIRECT\n\nrules:\n  - MATCH,🚀 节点选择\n")
	return b.String()
}

// GeneratePlaceholderBase64Config 通用订阅占位：base64(vmess://…)，
// v2rayN / Shadowrocket / Hiddify / 各家客户端都能解析。
func GeneratePlaceholderBase64Config(baseURL, message string) string {
	labels := placeholderLabels(baseURL, message)
	links := make([]string, 0, len(labels))
	for i, label := range labels {
		node := map[string]interface{}{
			"v": "2", "ps": label,
			"add": placeholderServer, "port": fmt.Sprintf("%d", i+1),
			"id":  "00000000-0000-0000-0000-000000000000",
			"aid": "0", "scy": "auto", "net": "tcp", "type": "none",
			"host": "", "path": "", "tls": "",
		}
		raw, _ := json.Marshal(node)
		links = append(links, "vmess://"+base64.StdEncoding.EncodeToString(raw))
	}
	return base64.StdEncoding.EncodeToString([]byte(strings.Join(links, "\n")))
}

// GeneratePlaceholderSingboxConfig sing-box 占位配置（outbounds 即节点列表）。
func GeneratePlaceholderSingboxConfig(baseURL, message string) string {
	labels := placeholderLabels(baseURL, message)
	outbounds := make([]map[string]interface{}, 0, len(labels)+1)
	for i, label := range labels {
		outbounds = append(outbounds, map[string]interface{}{
			"type": "vmess", "tag": label,
			"server": placeholderServer, "server_port": i + 1,
			"uuid": "00000000-0000-0000-0000-000000000000", "security": "auto", "alter_id": 0,
		})
	}
	outbounds = append(outbounds, map[string]interface{}{"type": "direct", "tag": "direct"})
	raw, _ := json.MarshalIndent(map[string]interface{}{"outbounds": outbounds}, "", "  ")
	return string(raw)
}

// GeneratePlaceholderConfig 按客户端类型选择占位格式。
// clash 系列给 YAML（Clash/Stash 都认），sing-box 给 JSON，其余给 base64 vmess。
func GeneratePlaceholderConfig(clientType, baseURL, message string) string {
	switch strings.ToLower(strings.TrimSpace(clientType)) {
	case "clash", "clashmeta", "clash.meta", "mihomo", "stash":
		return GeneratePlaceholderClashConfig(baseURL, message)
	case "singbox", "sing-box", "sing_box", "hiddify":
		return GeneratePlaceholderSingboxConfig(baseURL, message)
	default:
		return GeneratePlaceholderBase64Config(baseURL, message)
	}
}

// revokedSubscriptionMessage 判断 token 是不是"被重置掉的旧订阅地址"。
// 依据：performSubscriptionReset 会把旧地址写进 subscription_resets.old_subscription_url，
// 这里回查一次，命中就能给出比"地址无效"更准确的提示。
func revokedSubscriptionMessage(db *gorm.DB, token string) (string, bool) {
	token = strings.TrimSpace(token)
	if token == "" || db == nil {
		return "", false
	}
	var reset models.SubscriptionReset
	if err := db.Where("old_subscription_url = ?", token).Order("created_at DESC").First(&reset).Error; err != nil {
		return "", false
	}
	return "该订阅地址已失效（订阅地址被重置），请到官网复制新地址", true
}

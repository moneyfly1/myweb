package config_update

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"cboard-go/internal/models"
)

// ============================================================
// 客户端版本自动识别 + 协议能力过滤
// 根据客户端的类型与版本，过滤掉该客户端不支持的"新协议"，
// 避免老版本客户端拿到无法解析的节点链接导致整个订阅失败。
//
// 设计原则（与现有逻辑不冲突）：
//   - 只对"明确识别出客户端 + 版本号"的请求做能力过滤；
//   - 无法识别客户端或版本号时，保持现状（全部节点下发）；
//   - 过滤只在现有 protocol_filter / exclude 过滤之后追加执行；
//   - 规则集中在本文件，可在不修改分发逻辑的前提下调整。
// ============================================================

// clientCapabilities 描述一个客户端类型+版本区间支持/不支持的协议。
type clientCapabilities struct {
	// unsupportedProtocols 该客户端始终不支持的协议（不论版本）
	unsupportedProtocols map[string]bool
	// unsupportedBefore 版本号低于该值时额外不支持的协议（新协议随版本引入）
	// 结构：协议名 -> 最低支持版本号（版本号按客户端自身的版本格式解析）
	unsupportedBefore map[string]float64
}

// detectClientVersion 从 User-Agent 解析客户端类型与版本号。
// 返回 (clientType, version, ok)；ok=false 表示无法识别（调用方保持现状）。
func detectClientVersion(ua string) (string, float64, bool) {
	uaLower := strings.ToLower(ua)

	// Clash Meta 系列（支持全部新协议）
	// ClashMetaForAndroid/2.11.24.Meta / clash.meta/alpha-de19f92 / ClashMetaforWindows/...
	if strings.Contains(uaLower, "clashmeta") || strings.Contains(uaLower, "clash.meta") {
		return "clash-meta", parseVersionFromUA(ua), true
	}

	// 已知基于 mihomo / Clash.Meta 内核的客户端：一律按"支持全部新协议"处理。
	// 必须排在下面的 clash-legacy 分支之前：ClashX Meta 的 UA 含 "clashx" 子串、
	// clash-verge 也是 mihomo 内核，若被误判为老版 Clash，会把 vless/hysteria2
	// （现网各 279/93 个）当成"客户端不支持"整批剔除。
	// 现网实测 UA：mihomo.party/v2.0.0 (clash.meta)、ClashMetaForAndroid/2.11.31.Meta、
	// clash-verge/v2.5.2、ClashX Meta/1.4.x、FlClash、Clash Nyanpasu、ClashMi 等。
	// 片段表（小写匹配）：客户端命名有连字符/空格/驼峰多种写法，逐个 Contains 易漏
	// （实测漏过 "Clash Nyanpasu"，它含空格因此命中了 legacy 分支的 "clash "）。
	metaClientHints := []string{
		"meta", "mihomo",
		"clash-verge", "clashverge", "clash verge",
		"nyanpasu", "flclash", "clashmi", "clash mi",
		"clash-rs", "clashrs", "clashx meta",
	}
	for _, kw := range metaClientHints {
		if strings.Contains(uaLower, kw) {
			return "clash-meta", parseVersionFromUA(ua), true
		}
	}

	// 真正的老版 Clash（内置 Clash Premium/开源内核，不含 Meta）：
	// ClashforWindows/0.19.23、ClashforWindows/0.20.39、ClashforAndroid/2.5.12、ClashX/1.118.0。
	// 注意：CFW 0.20.x 默认内核仍是 Clash Premium（不含 vless/hysteria2），
	// 但如果用户自行替换为 Meta 内核，UA 无法体现——这类情况由管理员在
	// 「协议过滤」页关闭 client_capability_filter_enabled 开关来放行全量节点。
	// ClashforWindows/0.19.23 / ClashforAndroid/2.5.12 / ClashX/1.118.0
	if strings.Contains(uaLower, "clashforwindows") || strings.Contains(uaLower, "clashforandroid") ||
		strings.Contains(uaLower, "clashx") || strings.Contains(uaLower, "clash/") ||
		strings.Contains(uaLower, "clash ") {
		return "clash-legacy", parseVersionFromUA(ua), true
	}

	// Shadowrocket（版本号是构建号，如 Shadowrocket/1744）
	if strings.Contains(uaLower, "shadowrocket") {
		return "shadowrocket", parseVersionFromUA(ua), true
	}

	// sing-box 系列（支持全部新协议）
	if strings.Contains(uaLower, "sing-box") || strings.Contains(uaLower, "singbox") {
		return "sing-box", parseVersionFromUA(ua), true
	}

	// Surge / Stash / Loon / QuantumultX
	if strings.Contains(uaLower, "surge") {
		return "surge", parseVersionFromUA(ua), true
	}
	if strings.Contains(uaLower, "stash") {
		return "stash", parseVersionFromUA(ua), true
	}
	if strings.Contains(uaLower, "loon") {
		return "loon", parseVersionFromUA(ua), true
	}
	if strings.Contains(uaLower, "quantumult") {
		return "quantumult", parseVersionFromUA(ua), true
	}

	// v2rayN / v2rayNG
	if strings.Contains(uaLower, "v2rayn") || strings.Contains(uaLower, "v2rayng") {
		return "v2ray", parseVersionFromUA(ua), true
	}

	return "", 0, false
}

var versionRegex = regexp.MustCompile(`([0-9]+)(?:\.([0-9]+))?(?:\.([0-9]+))?`)

// parseVersionFromUA 从 UA 中提取版本号（首个形如 x.y.z 的数字序列）。
// Shadowrocket/1744 这类纯构建号也支持（提取 1744）。
// 支持格式：0.19.23 → 0.19；2.11.24 → 2.11；1744 → 1744
func parseVersionFromUA(ua string) float64 {
	// 优先匹配 "客户端名/版本" 模式：取第一个 "/" 后的数字段。
	// 用 FirstIndex 避免 Shadowrocket/1744 CFNetwork/3860.700.1 这类多斜杠 UA 取错。
	slashIdx := strings.Index(ua, "/")
	if slashIdx >= 0 && slashIdx < len(ua)-1 {
		afterSlash := ua[slashIdx+1:]
		// 截取到第一个非版本字符（空格、下划线、- 等）
		stop := 0
		for stop < len(afterSlash) {
			ch := afterSlash[stop]
			if (ch >= '0' && ch <= '9') || ch == '.' {
				stop++
			} else {
				break
			}
		}
		versionStr := afterSlash[:stop]
		if v, ok := parseVersionString(versionStr); ok {
			return v
		}
	}

	// 回退：全 UA 中找版本号
	if m := versionRegex.FindString(ua); m != "" {
		if v, ok := parseVersionString(m); ok {
			return v
		}
	}
	return 0
}

// parseVersionString 解析形如 "0.19.23" / "2.11" / "1744" 的版本号为 float64。
// 仅取主版本.次版本（如 0.19.23 → 0.19），纯数字按原值。
func parseVersionString(s string) (float64, bool) {
	s = strings.TrimSuffix(s, ".")
	if s == "" {
		return 0, false
	}
	parts := strings.Split(s, ".")
	if len(parts) > 1 {
		// 取主版本 + 次版本（0.19.23 → "0.19"）
		s = parts[0] + "." + parts[1]
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v <= 0 {
		return 0, false
	}
	return v, true
}

// getClientCapabilities 返回指定客户端的协议能力规则。
func getClientCapabilities(clientType string) *clientCapabilities {
	legacyClashUnsupported := map[string]bool{
		"vless": true, "reality": true, "hysteria2": true, "hysteria": true,
		"tuic": true, "anytls": true, "wireguard": true, "wg": true,
	}
	switch clientType {
	case "clash-legacy":
		// 老版 Clash 不支持 VLESS/Reality/Hy2/TUIC/AnyTLS 等新协议（仅 SS/VMess/Trojan）
		return &clientCapabilities{
			unsupportedProtocols: legacyClashUnsupported,
		}
	case "clash-meta", "sing-box", "stash":
		// Meta 与 sing-box 支持全部新协议，无需过滤
		return &clientCapabilities{}
	case "shadowrocket":
		// Shadowrocket 构建号 >= 1744 支持 Reality；更老的版本不支持部分新协议
		return &clientCapabilities{
			unsupportedBefore: map[string]float64{
				"reality": 1744, "hysteria2": 1600, "tuic": 1600, "anytls": 1800,
			},
		}
	case "surge":
		// Surge 5+ 支持大部分，老版本不支持 Reality/Hy2
		return &clientCapabilities{
			unsupportedBefore: map[string]float64{
				"reality": 5.0, "hysteria2": 5.0, "tuic": 5.0,
			},
		}
	case "loon":
		return &clientCapabilities{
			unsupportedBefore: map[string]float64{
				"reality": 3.0, "hysteria2": 3.0, "tuic": 3.0,
			},
		}
	case "quantumult":
		// QuantumultX 走**独立版本号体系（1.x）**，与 Loon 的 3.x 不可共用阈值：
		// 现网实测 UA "Quantumult%20X/1.4.0" 会被解析成 1.4，若沿用 3.0 阈值，
		// 会把 QX **支持**的 reality/hysteria2/tuic 当成"客户端不支持"整批剔除
		// （1.4 < 3.0 恒成立，等于对所有 QX 用户永久生效）。
		// QX 不该支持的类型（hysteria/hysteria2/tuic）已由**格式层**拦截：
		// generateQuantumultXConfig 只渲染官方 sample.conf 列明的类型，
		// formatRenderTypes[FmtQuantumultX] 与此逐字一致（有 go/ast 机械校验）。
		// 因此这里不做版本门控，避免用错误的版本阈值误砍节点。
		return &clientCapabilities{}
	default:
		return nil
	}
}

// nodeHasReality 判断节点是否启用 Reality。
// vless 链接里 sec=reality 或带 pbk 时，解析器会写入 Options["reality-opts"]，
// 其中 public-key 是 Reality 可用性的关键字段（缺 pbk 的 reality-opts 无意义）。
func nodeHasReality(n *ProxyNode) bool {
	if n == nil || len(n.Options) == 0 {
		return false
	}
	raw, ok := n.Options["reality-opts"]
	if !ok || raw == nil {
		return false
	}
	switch v := raw.(type) {
	case map[string]any:
		pk, _ := v["public-key"].(string)
		return strings.TrimSpace(pk) != ""
	case map[string]string:
		return strings.TrimSpace(v["public-key"]) != ""
	}
	return false
}

// nodeCapabilityKeys 返回节点在"客户端能力"语义下涉及的全部协议键。
//
// 除 n.Type 本身外，还包含由节点选项隐含的能力键：
//   - "reality"：启用 Reality 的节点要求客户端支持 Reality，否则在客户端侧就是
//     无法解析的非法配置。
//
// 这正是历史缺陷所在：unsupportedProtocols/unsupportedBefore 里写了 "reality"，
// 但判定只比较 p.Type，而 ProxyNode.Type 只可能是 vless/vmess/...，永远不等于
// "reality"——于是老版 Shadowrocket(<1744)/Surge(<5.0)/Loon(<3.0) 照样会收到
// Reality 节点，本该被防住的"客户端整份订阅解析失败"并没有被防住。
func nodeCapabilityKeys(n *ProxyNode) []string {
	if n == nil {
		return nil
	}
	keys := []string{strings.ToLower(strings.TrimSpace(n.Type))}
	if nodeHasReality(n) {
		keys = append(keys, "reality")
	}
	return keys
}

// nodeUnsupported 判断该客户端（能力表 + 版本）是否不支持此节点。
// 判定遍历节点的全部能力键，任一键命中不支持即判定不支持。
func (c *clientCapabilities) nodeUnsupported(p *ProxyNode, version float64) bool {
	if c == nil || p == nil {
		return false
	}
	for _, key := range nodeCapabilityKeys(p) {
		if c.unsupportedProtocols[key] {
			return true
		}
		if minVersion, exists := c.unsupportedBefore[key]; exists && version > 0 && version < minVersion {
			return true
		}
	}
	return false
}

// filterByCapabilities 客户端能力过滤的纯函数实现（不依赖 DB/日志，便于表驱动测试）。
// 返回保留的节点与被剔除的节点数。
func filterByCapabilities(proxies []*ProxyNode, version float64, caps *clientCapabilities) ([]*ProxyNode, int) {
	if caps == nil {
		return proxies, 0
	}
	kept := make([]*ProxyNode, 0, len(proxies))
	dropped := 0
	for _, p := range proxies {
		if caps.nodeUnsupported(p, version) {
			dropped++
			continue
		}
		kept = append(kept, p)
	}
	return kept, dropped
}

// capabilityFilterLogSeen 记录已输出过能力过滤摘要日志的（UA, 客户端, 版本, 规模）组合，
// 避免每次订阅拉取都写一行日志（现网 Clash for Windows 客户端数占比很高）。
var capabilityFilterLogSeen sync.Map

// filterProxiesByClientCapability 按客户端能力过滤协议。
// 受系统设置「协议过滤」页的客户端版本过滤开关控制：
//   - 开关关闭（client_capability_filter_enabled=false）时不执行任何过滤，全部节点下发
//   - 开关开启时：仅在能识别客户端且该客户端有规则时过滤；否则原样返回
func (s *ConfigUpdateService) filterProxiesByClientCapability(proxies []*ProxyNode, userAgent string) []*ProxyNode {
	// 检查总开关（存于 system_configs: category=protocol_filter, key=client_capability_filter_enabled）
	if !s.isClientCapabilityFilterEnabled() {
		return proxies
	}

	clientType, version, ok := detectClientVersion(userAgent)
	if !ok {
		return proxies
	}
	caps := getClientCapabilities(clientType)
	if caps == nil {
		return proxies
	}

	result, dropped := filterByCapabilities(proxies, version, caps)
	s.logCapabilityFilterOnce(userAgent, clientType, version, len(proxies), len(result), dropped)
	return result
}

// logCapabilityFilterOnce 每个（UA, 客户端类型, 版本, 规模）组合只记一次过滤摘要。
// 能力过滤会让某些老客户端只拿到部分节点（如老 Clash 只支持 ss/vmess/trojan），
// 没有这行日志时管理员只看到"节点变少"却查不到原因，容易误判为节点丢失事故。
func (s *ConfigUpdateService) logCapabilityFilterOnce(userAgent, clientType string, version float64, total, kept, dropped int) {
	if s == nil || dropped <= 0 {
		return
	}
	key := fmt.Sprintf("%s|%s|%.2f|%d|%d", userAgent, clientType, version, total, kept)
	if _, loaded := capabilityFilterLogSeen.LoadOrStore(key, true); loaded {
		return
	}
	s.infof("🧭 客户端能力过滤: UA=%q 识别为 %s(v%.2f) → 下发 %d/%d 个节点（按该客户端不支持的协议剔除 %d 个）",
		truncateUA(userAgent, 64), clientType, version, kept, total, dropped)
}

// truncateUA 按 rune 截断 UA 用于日志，避免超长 UA 污染日志行。
func truncateUA(ua string, maxRunes int) string {
	ua = strings.TrimSpace(ua)
	r := []rune(ua)
	if len(r) <= maxRunes {
		return ua
	}
	return string(r[:maxRunes]) + "…"
}

// isClientCapabilityFilterEnabled 读取客户端版本过滤总开关。
// 默认开启（true）；管理员可在系统设置 → 协议过滤 页面关闭，
// 关闭后所有客户端均收到全量节点（避免老客户端订阅异常或节点过少）。
func (s *ConfigUpdateService) isClientCapabilityFilterEnabled() bool {
	if s == nil || s.db == nil {
		return true
	}
	var cfg models.SystemConfig
	if err := s.db.Where("category = ? AND key = ?", "protocol_filter", "client_capability_filter_enabled").First(&cfg).Error; err != nil {
		// 未配置时默认开启（与历史行为一致）
		return true
	}
	return cfg.Value == "true" || cfg.Value == "1"
}

// applySubscriptionFilters 应用订阅协议过滤，客户端过滤与协议白名单互斥：
//   - 客户端版本过滤开启：遵循客户端能力原则——按客户端类型/版本过滤其不支持的协议，
//     跳过 DB 协议白名单（避免白名单误删客户端实际支持的协议）；
//   - 客户端版本过滤关闭：使用协议过滤——按 DB 配置的协议白名单（clash_protocols /
//     universal_protocols）过滤。
//
// exclude 参数过滤在两种模式下都生效（用户显式指定排除的协议）。
func (s *ConfigUpdateService) applySubscriptionFilters(nodes []*ProxyNode, filterType, userAgent string, excluded map[string]bool) []*ProxyNode {
	if s.isClientCapabilityFilterEnabled() {
		// 客户端过滤优先：按客户端能力过滤，忽略 DB 协议白名单
		nodes = s.filterProxiesByExcludedProtocols(nodes, excluded)
		return s.filterProxiesByClientCapability(nodes, userAgent)
	}
	// 客户端过滤关闭：使用 DB 协议白名单
	nodes = s.filterProxiesByProtocol(nodes, s.getProtocolFilter(filterType))
	return s.filterProxiesByExcludedProtocols(nodes, excluded)
}

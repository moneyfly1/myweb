package config_update

import (
	"regexp"
	"strconv"
	"strings"

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

// 通用"新协议"集合（老客户端普遍不支持）
var newProtocols = []string{
	"vless", "reality", "hysteria2", "hysteria", "tuic", "anytls", "wireguard", "wg",
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

	// 老版 Clash（premium / open source，不含 Meta）
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
	case "loon", "quantumult":
		return &clientCapabilities{
			unsupportedBefore: map[string]float64{
				"reality": 3.0, "hysteria2": 3.0, "tuic": 3.0,
			},
		}
	default:
		return nil
	}
}

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

	var result []*ProxyNode
	for _, p := range proxies {
		if caps.unsupportedProtocols[p.Type] {
			continue
		}
		if minVersion, exists := caps.unsupportedBefore[p.Type]; exists {
			if version > 0 && version < minVersion {
				continue
			}
		}
		result = append(result, p)
	}
	return result
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

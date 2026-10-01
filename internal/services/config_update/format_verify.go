package config_update

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cboard-go/internal/models"

	"gopkg.in/yaml.v3"
)

// ============================================================================
// 全格式输出的结构/语法校验（不依赖任何客户端内核）
//
// 目标：**任何格式都不能因为一个坏节点而整份失效**。
//
// 两层做法，对每种格式统一执行：
//  1. 逐节点：先判断该格式能否渲染该节点，再对渲染结果做单条结构校验；
//     不合法就**只从该格式里剔除这一个节点**（事件里带 format=），其余节点不动。
//  2. 整体：返回前必做纯语法/结构校验，其中最关键的是**成员悬空检查**
//     （一个节点被剔除后，proxy-groups / [Proxy Group] / [policy] 里的引用会指向不存在的名字，
//      这是"一个坏节点拖垮整份配置"最常见的真实成因）。
//
// Clash 额外再走第二层内核自检（mihomo -t），本文件的结构校验是它的前置廉价闸门。
// ============================================================================

// OutputFormat 实际下发的输出格式
type OutputFormat string

const (
	FmtClash       OutputFormat = "clash-yaml"   // Clash / Clash.Meta / Stash，YAML
	FmtLinksBase64 OutputFormat = "base64-links" // 通用 / Shadowrocket / v2rayN，base64 包装的链接列表
	FmtLinksPlain  OutputFormat = "plain-links"  // 纯链接列表（内部用；也被 base64 格式复用）
	FmtSurge       OutputFormat = "surge"        // Surge，行式 [Proxy]/[Proxy Group]
	FmtSingBox     OutputFormat = "singbox-json" // sing-box，JSON
	FmtQuantumultX OutputFormat = "quantumultx"  // Quantumult X，行式 [server_local]/[policy]
	FmtLoon        OutputFormat = "loon"         // Loon，行式 [Proxy]
)

// linkRenderTypes nodeToLink 能产出链接的协议集合
var linkRenderTypes = map[string]bool{
	"vmess": true, "vless": true, "trojan": true, "ss": true, "ssr": true,
	"hysteria": true, "hysteria2": true, "tuic": true, "anytls": true,
	"socks": true, "socks5": true, "http": true,
}

// formatRenderTypes 每种格式**实际能渲染**的协议。
//
// ⚠️ 这张表必须与各生成器的 `switch n.Type` 逐字一致。曾经的教训：手工抄写时把
// `case "hysteria", "hysteria2":` 这类**多值 case** 只抄了第一个值，导致
// sing-box 与 Surge 的 93 个 hysteria2 节点被本层静默过滤掉（生成器本来是支持的）。
// 现在由 TestFormatRenderTypesMatchGeneratorCases 用 go/ast 解析生成器源码做机械比对，
// 任何一侧改动而另一侧没跟上都会直接测试失败。
//
// 必须与各生成器里的 `switch n.Type` 逐一对照，否则会出现"格式放行但生成器静默跳过"的黑洞。
//
//	Surge        : proxyNodeToSurgeLine 的 case（http/hysteria/socks/ss/trojan/tuic/vless/vmess）
//	Loon         : generateLoonConfig 的 case（ss/trojan/vmess/vless/hysteria2/socks/socks5/http/anytls）
//	QuantumultX  : generateQuantumultXConfig 的 case（ss/trojan/vmess/vless/socks/socks5/http/anytls）
//	SingBox      : generateSingBoxConfig 的 case（hysteria/ss/trojan/tuic/vless/vmess）+ direct
var formatRenderTypes = map[OutputFormat]map[string]bool{
	FmtClash:       supportedClashTypes,
	FmtLinksBase64: linkRenderTypes,
	FmtLinksPlain:  linkRenderTypes,
	FmtSurge:       {"http": true, "hysteria2": true, "socks": true, "socks5": true, "ss": true, "trojan": true, "tuic": true, "vless": true, "vmess": true},
	// SingBox：为什么**不含** wireguard / ssh / ssr（用真实内核 sing-box 1.11.15 实测得出，不是凭记忆）：
	//   · wireguard：sing-box 的 legacy wireguard **outbound** 自 1.11 起废弃、1.13 将移除
	//     （内核原话："legacy wireguard outbound is deprecated in sing-box 1.11.0 and will be removed
	//     in sing-box 1.13.0"），且缺少 private_key 时直接 FATAL "missing private key"。
	//     现网 wg:// 节点数为 0；渲染一个即将被移除的写法属于给未来埋雷，故不渲染（丢弃会记
	//     format-unsupported-type，管理员可见）。如将来要支持，应改用 1.11+ 的 endpoints 模型。
	//   · ssh：本项目**没有 ssh:// 解析器**（见 node_parser.go 的 protocolParsers），
	//     即永远不会有 ssh 类型节点进入节点池，渲染它属于死代码。
	//   · ssr：sing-box 已移除 shadowsocksr 支持。
	FmtSingBox: {"hysteria": true, "hysteria2": true, "anytls": true, "socks": true, "socks5": true, "http": true, "ss": true, "trojan": true, "tuic": true, "vless": true, "vmess": true},
	// QuantumultX：官方 sample.conf 支持 shadowsocks/vmess/vless/trojan/http/socks5/anytls；
	// 明确不支持 hysteria/hysteria2/tuic（不渲染比渲染坏行更安全——QX 遇到不支持的类型有整份失败风险）。
	FmtQuantumultX: {"ss": true, "trojan": true, "vmess": true, "vless": true, "socks": true, "socks5": true, "http": true, "anytls": true},
	// Loon：官方文档支持 ss/vmess/vless/trojan/http|https/socks5/hysteria2/anytls 等；
	// tuic 已从本表移除（官方 36 页文档零命中 + URI scheme 清单 + Sub-Store 三方一致判定不支持）。
	FmtLoon: {"ss": true, "trojan": true, "vmess": true, "vless": true, "hysteria2": true, "socks": true, "socks5": true, "http": true, "anytls": true},
}

// clashBuiltinNames Clash 里无需定义即可被分组引用的内置名
var clashBuiltinNames = map[string]bool{
	"DIRECT": true, "REJECT": true, "REJECT-DROP": true, "PASS": true, "COMPATIBLE": true, "GLOBAL": true,
}

// FormatDropped 某个格式里被剔除的节点（脱敏，不含凭据）
type FormatDropped struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

// FormatVerifyResult 单格式的校验结果
type FormatVerifyResult struct {
	Payload     string
	Dropped     []FormatDropped
	Verified    bool // 整体结构校验通过
	Fallback    bool // 是否走了上一份已验证产物
	CacheHit    bool
	Format      OutputFormat
	NodeCount   int
	Unsupported int // 该格式不支持、被正常过滤掉的协议数（非坏节点）
}

// RenderFunc 用给定节点集合渲染该格式的完整 payload
type RenderFunc func([]*ProxyNode) string

// ---------------------------------------------------------------- 缓存

type formatVerifyEntry struct {
	at time.Time
}

type formatVerifyCache struct {
	mu        sync.RWMutex
	verified  map[string]time.Time // key: format|fingerprint
	lastGood  map[string]string    // key: token|format
	lastGoodT map[string]time.Time
}

var fmtCache = &formatVerifyCache{
	verified:  make(map[string]time.Time),
	lastGood:  make(map[string]string),
	lastGoodT: make(map[string]time.Time),
}

const (
	formatVerifiedTTL = 15 * time.Minute
	formatLastGoodTTL = 60 * time.Minute
	formatBisectMax   = 24 // 结构校验很便宜，但仍给个上限防病态输入
)

func (c *formatVerifyCache) isVerified(key string) bool {
	c.mu.RLock()
	at, ok := c.verified[key]
	c.mu.RUnlock()
	return ok && time.Since(at) < formatVerifiedTTL
}

func (c *formatVerifyCache) markVerified(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.verified[key] = time.Now()
	if len(c.verified) > 8192 {
		now := time.Now()
		for k, at := range c.verified {
			if now.Sub(at) >= formatVerifiedTTL {
				delete(c.verified, k)
			}
		}
	}
}

func (c *formatVerifyCache) setLastGood(key, payload string) {
	if key == "" || strings.TrimSpace(payload) == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastGood[key] = payload
	c.lastGoodT[key] = time.Now()
	if len(c.lastGood) > 4096 {
		now := time.Now()
		for k, at := range c.lastGoodT {
			if now.Sub(at) >= formatLastGoodTTL {
				delete(c.lastGood, k)
				delete(c.lastGoodT, k)
			}
		}
	}
}

func (c *formatVerifyCache) getLastGood(key string) string {
	if key == "" {
		return ""
	}
	c.mu.RLock()
	p, ok := c.lastGood[key]
	at := c.lastGoodT[key]
	c.mu.RUnlock()
	if !ok || time.Since(at) >= formatLastGoodTTL {
		return ""
	}
	return p
}

// formatFingerprint 节点集合指纹（与内核自检同口径，保证"节点集合没变"判定一致）
func formatFingerprint(nodes []*ProxyNode) string { return kernelFingerprint(nodes) }

// ---------------------------------------------------------------- 主流程

// buildVerifiedPayload 全格式统一入口：
// 凭据规范化 → 按格式能力过滤 → 渲染 → 结构校验 → （失败）折半定位剔除 → 复检 → 缓存 / 回退。
// 绝不返回空内容；整体校验失败时回退到该格式**上一份已验证产物**。
func (s *ConfigUpdateService) buildVerifiedPayload(
	f OutputFormat, nodes []*ProxyNode, render RenderFunc,
	cacheToken string,
) FormatVerifyResult {
	res := FormatVerifyResult{Format: f}

	// ⓪ 凭据规范化（所有格式同时受益；存量未解码数据不需要回填迁移）
	normalized, _ := normalizeCredentialsForOutput(nodes)

	// ① 按格式能力过滤：不支持的协议只从**这个格式**里剔除，不影响其它格式
	candidates := make([]*ProxyNode, 0, len(normalized))
	for _, n := range normalized {
		if n == nil {
			continue
		}
		if isPlaceholderInfoNode(n) {
			candidates = append(candidates, n) // 面板提示节点单独处理，见下
			continue
		}
		// 名字为空的节点：链接格式仍然可用（无 fragment 是合法的），
		// 但**按名字组织**的格式（Clash / Surge / Loon / QX）会产出一个无名条目
		// （Surge 的分组列表里会出现空成员、Clash 的 proxies 里 name 为空）。
		// 这类节点在那些格式里直接剔除并记录原因，保证行为一致可查。
		if f != FmtLinksBase64 && f != FmtLinksPlain && strings.TrimSpace(n.Name) == "" {
			res.Dropped = append(res.Dropped, FormatDropped{Name: "(空名)", Type: n.Type, Reason: "format-invalid-line"})
			continue
		}
		if !formatRenderTypes[f][NormalizeNodeType(n.Type)] {
			// 该格式不支持这个协议（如 QuantumultX/Loon 不支持 hysteria2/vless）——
			// 这是**正常的能力差异**，不是坏节点，因此只计数不落事件，
			// 否则每个请求都会往 node_validation_logs 灌入成百上千条噪音。
			res.Unsupported++
			res.Dropped = append(res.Dropped, FormatDropped{
				Name: n.Name, Type: n.Type, Reason: "format-unsupported-type"})
			continue
		}
		candidates = append(candidates, n)
	}

	fp := formatFingerprint(candidates)
	cacheKey := string(f) + "|" + fp
	// 只在这一指纹首次计算时打印一次（缓存命中不会重复），让"哪些类型被跳过、各多少个"可见，
	// 避免节点被静默遗漏（历史上 sing-box/Surge 的 hysteria2 就是这样被漏掉的）。
	if !fmtCache.isVerified(cacheKey) && len(res.Dropped) > 0 {
		counts := map[string]int{}
		for _, d := range res.Dropped {
			counts[d.Type]++
		}
		parts := make([]string, 0, len(counts))
		for t, n := range counts {
			parts = append(parts, fmt.Sprintf("%s×%d", t, n))
		}
		sort.Strings(parts)
		// 注意：candidates 已经排除了被跳过的节点，这里不能再减一次 res.Dropped
		logf("格式 %s: 本次下发 %d 个节点；因该格式不支持而跳过 %d 个（%s）。若其中含客户端其实支持的协议，说明生成器或能力表有遗漏。",
			f, len(candidates), len(res.Dropped), strings.Join(parts, ", "))
	}
	if fmtCache.isVerified(cacheKey) {
		res.Payload = render(candidates)
		res.Verified, res.CacheHit, res.NodeCount = true, true, len(candidates)
		fmtCache.setLastGood(cacheToken+"|"+string(f), res.Payload)
		return res
	}

	// ② 渲染 + 整体结构校验
	good := candidates
	payload := render(good)
	if err := validateFormatPayload(f, payload); err == nil {
		res.Payload, res.Verified, res.NodeCount = payload, true, len(good)
		fmtCache.markVerified(cacheKey)
		fmtCache.setLastGood(cacheToken+"|"+string(f), payload)
		s.recordFormatDrops(f, res.Dropped)
		return res
	} else {
		logf("格式 %s 整体校验失败，开始折半定位坏节点：%v", f, err)
	}

	// ②' 基线探针：用 1 个该格式支持的合成节点渲染，判断"生成器本身"能否产出合法 payload。
	// 不能 → 问题出在格式/生成器（例如模板结构问题），与用户节点无关；此时**绝不能剔除任何节点**
	//（否则一次误判就会把用户的节点全删光，这是比"一个坏节点"严重得多的故障）。
	if probe := formatProbeNode(); probe != nil {
		if perr := validateFormatPayload(f, render([]*ProxyNode{probe})); perr != nil {
			logf("格式 %s 基线探针校验失败（%v）→ 判定为生成器/格式问题，不剔除任何节点", f, perr)
			if last := fmtCache.getLastGood(cacheToken + "|" + string(f)); last != "" {
				res.Payload, res.Fallback, res.NodeCount = last, true, len(candidates)
				logf("格式 %s 回退到本订阅上一份已验证产物", f)
			} else {
				res.Payload, res.NodeCount = render(candidates), len(candidates)
			}
			s.recordFormatDrops(f, res.Dropped)
			return res
		}
	}

	// ③ 折半定位（纯结构校验，代价极低）
	good, bad := s.bisectFormatNodes(f, candidates, render, 0)
	if len(bad) > 0 {
		for _, n := range bad {
			res.Dropped = append(res.Dropped, FormatDropped{
				Name: n.Name, Type: n.Type, Reason: "format-invalid-line"})
		}
	}
	payload = render(good)
	if err := validateFormatPayload(f, payload); err == nil {
		res.Payload, res.Verified, res.NodeCount = payload, true, len(good)
		fmtCache.markVerified(formatFingerprint(good) + "|" + string(f))
		fmtCache.markVerified(cacheKey)
		fmtCache.setLastGood(cacheToken+"|"+string(f), payload)
		s.recordFormatDrops(f, res.Dropped)
		return res
	} else {
		logf("格式 %s 剔除后复检仍未通过（%v），进入回退", f, err)
	}

	// ④ 回退：该格式上一份已验证产物 → 否则退回"按能力过滤后"的渲染结果（结构问题属格式自身，不再删用户节点）
	if last := fmtCache.getLastGood(cacheToken + "|" + string(f)); last != "" {
		res.Payload, res.Fallback, res.NodeCount = last, true, len(good)
		logf("格式 %s 回退到本订阅上一份已验证产物", f)
	} else {
		res.Payload, res.NodeCount = payload, len(good)
		logf("格式 %s 无历史产物，下发经静态过滤的渲染结果（不删用户节点）", f)
	}
	s.recordFormatDrops(f, res.Dropped)
	return res
}

// formatProbeNode 基线探针节点：内核实测永远合法的最小 ss 节点。
// 所有输出格式都支持 ss，因此一个探针即可判定"生成器是否健康"。
func formatProbeNode() *ProxyNode {
	return &ProxyNode{
		Name: "__mf_format_probe__", Type: "ss", Server: "1.2.3.4", Port: 8388,
		Cipher: "aes-128-gcm", Password: "probe", Options: map[string]any{},
	}
}

// bisectFormatNodes 折半递归定位"让该格式整体校验失败"的节点
func (s *ConfigUpdateService) bisectFormatNodes(f OutputFormat, nodes []*ProxyNode, render RenderFunc, depth int) (good, bad []*ProxyNode) {
	if len(nodes) == 0 || depth > formatBisectMax {
		return nodes, nil
	}
	if err := validateFormatPayload(f, render(nodes)); err == nil {
		return nodes, nil
	}
	if len(nodes) == 1 {
		return nil, nodes
	}
	mid := len(nodes) / 2
	g1, b1 := s.bisectFormatNodes(f, nodes[:mid], render, depth+1)
	g2, b2 := s.bisectFormatNodes(f, nodes[mid:], render, depth+1)
	return append(append(good, g1...), g2...), append(append(bad, b1...), b2...)
}

// recordFormatDrops 记录**真实的结构问题**剔除事件（按格式区分，后台可按 format 过滤）。
// 刻意不记录 format-unsupported-type：那是格式能力差异（QX/Loon 不支持 hysteria2/vless），
// 每个请求都会有，记进去只会淹没真正的问题。
func (s *ConfigUpdateService) recordFormatDrops(f OutputFormat, drops []FormatDropped) {
	var events []NodeValidationEvent
	for _, d := range drops {
		if d.Reason == "format-unsupported-type" {
			continue
		}
		events = append(events, NodeValidationEvent{
			Event:    models.NodeValidationPrunedAtGenerate,
			Source:   "format=" + string(f),
			Format:   string(f),
			NodeName: d.Name,
			NodeType: d.Type,
			Reason:   d.Reason,
		})
	}
	if len(events) == 0 {
		return
	}
	logf("格式 %s: 因结构问题剔除 %d 个节点（其余 %d 个是格式能力差异，不记事件）",
		f, len(events), len(drops)-len(events))
	s.recordNodeValidationEvents(events)
}

// ---------------------------------------------------------------- 整体校验

// validateFormatPayload 纯语法/结构校验（不依赖内核）
func validateFormatPayload(f OutputFormat, payload string) error {
	switch f {
	case FmtClash:
		return validateClashYAML(payload)
	case FmtLinksPlain:
		return validateLinksList(payload)
	case FmtLinksBase64:
		return validateBase64Links(payload)
	case FmtSingBox:
		return validateSingBoxJSON(payload)
	case FmtSurge:
		return validateSurge(payload)
	case FmtLoon:
		return validateLoon(payload)
	case FmtQuantumultX:
		return validateQuantumultX(payload)
	}
	return fmt.Errorf("未知格式 %s", f)
}

// validateLinksList 纯链接列表：逐行 url.Parse + 按 scheme 校验必填字段 + 内部 base64 可解码
func validateLinksList(payload string) error {
	lines := strings.Split(strings.TrimSpace(payload), "\n")
	n := 0
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if err := validateOneLink(line); err != nil {
			return fmt.Errorf("第 %d 行链接不合法: %w", i+1, err)
		}
		n++
	}
	if n == 0 {
		return fmt.Errorf("链接列表为空")
	}
	return nil
}

// validateOneLink 单条代理链接的结构校验
func validateOneLink(link string) error {
	u, err := url.Parse(link)
	if err != nil {
		return fmt.Errorf("url.Parse 失败: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if !linkRenderTypes[normalizeLinkScheme(scheme)] {
		return fmt.Errorf("未知 scheme %q", scheme)
	}
	switch normalizeLinkScheme(scheme) {
	case "vmess":
		body := strings.TrimPrefix(link, scheme+"://")
		body = strings.SplitN(body, "#", 2)[0]
		if strings.TrimSpace(body) == "" {
			return fmt.Errorf("vmess 负载为空")
		}
		dec, err := DecodeBase64(body)
		if err != nil {
			return fmt.Errorf("vmess base64 负载不可解码: %w", err)
		}
		if strings.HasPrefix(strings.TrimSpace(dec), "{") {
			if !strings.Contains(dec, "\"add\"") || !strings.Contains(dec, "\"port\"") {
				return fmt.Errorf("vmess JSON 缺少 add/port")
			}
			return nil
		}
		host, port := parseHostPort(strings.TrimSpace(dec))
		if host == "" || port < 1 || port > 65535 {
			return fmt.Errorf("vmess 非 JSON 形态缺少合法 host:port")
		}
		return nil
	case "ssr":
		body := strings.TrimPrefix(link, scheme+"://")
		body = strings.SplitN(body, "#", 2)[0]
		body = strings.SplitN(body, "?", 2)[0]
		dec, err := DecodeBase64(body)
		if err != nil {
			return fmt.Errorf("ssr 主体 base64 不可解码: %w", err)
		}
		if len(strings.Split(dec, ":")) < 6 {
			return fmt.Errorf("ssr 主体解码后字段不足 6 段")
		}
		return nil
	}

	if u.Hostname() == "" {
		return fmt.Errorf("缺少 host")
	}
	port := getPortForValidate(u)
	if port < 1 || port > 65535 {
		return fmt.Errorf("端口非法: %d", port)
	}
	switch normalizeLinkScheme(scheme) {
	case "ss":
		if u.User == nil || u.User.String() == "" {
			return fmt.Errorf("ss 缺少 userinfo(cipher:password)")
		}
		raw := u.User.String()
		if !strings.Contains(raw, ":") {
			dec, err := DecodeBase64(raw)
			if err != nil {
				return fmt.Errorf("ss userinfo 既无 ':' 也不可 base64 解码")
			}
			raw = dec
		}
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return fmt.Errorf("ss 缺少 cipher 或 password")
		}
	case "vless":
		if u.User == nil || strings.TrimSpace(u.User.Username()) == "" {
			return fmt.Errorf("vless 缺少 uuid")
		}
	case "trojan":
		if u.User == nil || strings.TrimSpace(u.User.Username()) == "" {
			return fmt.Errorf("trojan 缺少 password")
		}
	case "hysteria2", "anytls":
		if u.User == nil || strings.TrimSpace(u.User.String()) == "" {
			return fmt.Errorf("%s 缺少 auth/password", scheme)
		}
	}
	return nil
}

// normalizeLinkScheme 链接 scheme → 内核类型名
func normalizeLinkScheme(scheme string) string {
	switch scheme {
	case "hy2":
		return "hysteria2"
	case "socks":
		return "socks5"
	case "https":
		return "http"
	}
	return scheme
}

// getPortForValidate 取链接端口（不套用生成侧的默认值，用于"校验链接本身是否写了合法端口"）
func getPortForValidate(u *url.URL) int {
	p := u.Port()
	if p == "" {
		return 0
	}
	i, err := strconv.Atoi(p)
	if err != nil {
		return 0
	}
	return i
}

// validateBase64Links base64 包装的链接列表：必须能原样解码回链接文本（往返比对）
func validateBase64Links(payload string) error {
	compact := strings.TrimSpace(payload)
	if compact == "" {
		return fmt.Errorf("base64 负载为空")
	}
	// 订阅里的 base64 惯例是不带换行的标准 base64
	dec, err := base64.StdEncoding.DecodeString(compact)
	if err != nil {
		if dec2, err2 := base64.RawStdEncoding.DecodeString(compact); err2 == nil {
			dec = dec2
		} else {
			return fmt.Errorf("整份 base64 不可解码: %w", err)
		}
	}
	text := string(dec)
	// 往返比对：重新编码必须与原文一致（防止换行/非法字符导致客户端解不开）
	if re := base64.StdEncoding.EncodeToString(dec); re != compact {
		// 允许原文是 Raw 编码（无 padding）
		if base64.RawStdEncoding.EncodeToString(dec) != compact {
			return fmt.Errorf("base64 往返比对不一致（长度 原文=%d 重编码=%d）", len(compact), len(re))
		}
	}
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("base64 解码后为空")
	}
	return validateLinksList(text)
}

// validateSingBoxJSON sing-box：json.Valid + 顶层必需键 + 每个 outbound 必需字段
func validateSingBoxJSON(payload string) error {
	if strings.TrimSpace(payload) == "" {
		return fmt.Errorf("JSON 为空")
	}
	var doc map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &doc); err != nil {
		return fmt.Errorf("JSON 不可解析: %w", err)
	}
	raw, ok := doc["outbounds"]
	if !ok {
		return fmt.Errorf("缺少顶层 outbounds")
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return fmt.Errorf("outbounds 不是数组")
	}
	if len(arr) == 0 {
		return fmt.Errorf("outbounds 为空")
	}
	tags := map[string]bool{}
	proxies := 0
	for i, it := range arr {
		m, ok := it.(map[string]interface{})
		if !ok {
			return fmt.Errorf("outbounds[%d] 不是对象", i)
		}
		t, _ := m["type"].(string)
		tag, _ := m["tag"].(string)
		if strings.TrimSpace(t) == "" {
			return fmt.Errorf("outbounds[%d] 缺少 type", i)
		}
		if strings.TrimSpace(tag) == "" {
			return fmt.Errorf("outbounds[%d] 缺少 tag", i)
		}
		if tags[tag] {
			return fmt.Errorf("outbounds[%d] tag 重复: %s", i, tag)
		}
		tags[tag] = true
		switch t {
		case "direct", "block", "dns":
			continue
		case "shadowsocks", "trojan", "vmess", "vless", "hysteria", "tuic", "hysteria2", "anytls", "socks", "http":
			proxies++
			if s2, _ := m["server"].(string); strings.TrimSpace(s2) == "" {
				return fmt.Errorf("outbounds[%d](%s) 缺少 server", i, tag)
			}
			if p, okc := m["server_port"].(float64); !okc || p < 1 || p > 65535 {
				return fmt.Errorf("outbounds[%d](%s) server_port 非法", i, tag)
			}
			// sing-box 的 hysteria v1 出站**必填** up_mbps/down_mbps，缺了内核会拒绝该出站
			if t == "hysteria" {
				if up, okc := m["up_mbps"].(float64); !okc || up <= 0 {
					return fmt.Errorf("outbounds[%d](%s) hysteria v1 缺少 up_mbps", i, tag)
				}
				if dn, okc := m["down_mbps"].(float64); !okc || dn <= 0 {
					return fmt.Errorf("outbounds[%d](%s) hysteria v1 缺少 down_mbps", i, tag)
				}
			}
		default:
			return fmt.Errorf("outbounds[%d] 未知 type %q", i, t)
		}
	}
	if proxies == 0 {
		return fmt.Errorf("outbounds 里没有可用的代理节点")
	}
	return nil
}

// ---------------------------------------------------------------- 行式格式

// splitLines 去掉注释/空行
func formatNonEmptyLines(payload string) []string {
	var out []string
	for _, l := range strings.Split(payload, "\n") {
		l = strings.TrimRight(l, "\r")
		if strings.TrimSpace(l) == "" || strings.HasPrefix(strings.TrimSpace(l), "#") {
			continue
		}
		out = append(out, l)
	}
	return out
}

// sectionLines 取指定 section（形如 "surge:[Proxy]"）里的行
func sectionLines(lines []string, start string, ends ...string) []string {
	var out []string
	in := false
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
			if t == start {
				in = true
				continue
			}
			for _, e := range ends {
				if t == e {
					in = false
				}
			}
			if in {
				in = false
			}
			continue
		}
		if in {
			out = append(out, l)
		}
	}
	return out
}

// validateSurge Surge（真实产物形态）：
//
//	[Proxy]       DIRECT = direct
//	              名字 = ss, host, port, ...
//	[Proxy Group] Proxy = select, AutoTest, DIRECT, 名字...
//	              AutoTest = url-test, 名字..., url=..., interval=..., tolerance=...
//
// 关键检查：**分组成员必须能解析到已定义的代理名/分组名**（一个节点被剔除后最容易在这里悬空）。
// 分组允许前向引用（Proxy 引用后面才定义的 AutoTest），所以先收集全部名字再校验成员。
func validateSurge(payload string) error {
	lines := formatNonEmptyLines(payload)
	proxyLines := sectionLines(lines, "[Proxy]", "[Proxy Group]", "[Rule]")
	groupLines := sectionLines(lines, "[Proxy Group]", "[Rule]")
	if len(proxyLines) == 0 {
		return fmt.Errorf("[Proxy] 段为空")
	}
	defined := map[string]bool{}
	for i, l := range proxyLines {
		eq := strings.Index(l, "=")
		if eq < 0 {
			return fmt.Errorf("[Proxy] 第 %d 行缺少 '=': %.40q", i+1, l)
		}
		name := strings.TrimSpace(l[:eq])
		val := strings.TrimSpace(l[eq+1:])
		if name == "" {
			return fmt.Errorf("[Proxy] 第 %d 行名称为空", i+1)
		}
		if defined[name] {
			return fmt.Errorf("[Proxy] 名称重复: %s", name)
		}
		defined[name] = true
		if strings.EqualFold(val, "direct") {
			continue // DIRECT = direct
		}
		fields := strings.Split(val, ",")
		if len(fields) < 3 {
			return fmt.Errorf("[Proxy] %s 字段不足（需要 类型,主机,端口）", name)
		}
		typ := strings.TrimSpace(fields[0])
		if !surgeTypeOK(typ) {
			return fmt.Errorf("[Proxy] %s 类型 %q 不被 Surge 支持", name, typ)
		}
		if strings.TrimSpace(fields[1]) == "" {
			return fmt.Errorf("[Proxy] %s 缺少主机", name)
		}
		port, err := strconv.Atoi(strings.TrimSpace(fields[2]))
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("[Proxy] %s 端口非法: %q", name, strings.TrimSpace(fields[2]))
		}
	}
	// 先登记所有分组名（允许前向引用）
	groupNames := map[string]bool{}
	for _, l := range groupLines {
		eq := strings.Index(l, "=")
		if eq < 0 {
			return fmt.Errorf("[Proxy Group] 行缺少 '=': %.40q", l)
		}
		groupNames[strings.TrimSpace(l[:eq])] = true
	}
	for _, l := range groupLines {
		eq := strings.Index(l, "=")
		gname := strings.TrimSpace(l[:eq])
		for idx, m := range strings.Split(l[eq+1:], ",") {
			m = strings.TrimSpace(m)
			// 逗号分隔的第 0 项是分组**类型**（select / url-test / fallback / load-balance），不是成员
			if idx == 0 || m == "" || strings.Contains(m, "=") { // 含 '=' 的是 url=/interval=/tolerance= 参数
				continue
			}
			if m == gname || defined[m] || groupNames[m] || surgePolicyKeyword(m) {
				continue
			}
			return fmt.Errorf("[Proxy Group] %s 引用了不存在的成员 %q（成员悬空）", gname, m)
		}
	}
	return nil
}

// surgeTypeOK Surge 允许的代理类型记号。
// 先看 formatRenderTypes[FmtSurge]（由生成器的 case 机械比对保证），再补 Surge 的别名写法。
// 这样**校验器不可能拒绝生成器会产出的类型**——否则就会出现"生成器写得出来、校验器判它非法"
// 的自相矛盾（曾因白名单漏 hysteria2 把 93 个节点判成 format-invalid-line 剔除）。
func surgeTypeOK(t string) bool {
	t = strings.ToLower(strings.TrimSpace(t))
	if formatRenderTypes[FmtSurge][t] {
		return true
	}
	switch t {
	case "shadowsocks", "https", "socks5-tls", "snell", "direct":
		return true
	}
	return false
}

func surgePolicyKeyword(s string) bool {
	switch s {
	case "DIRECT", "REJECT", "REJECT-DROP", "REJECT-TINYGIF", "no-resolve", "dns-failed":
		return true
	}
	return false
}

// validateLoon Loon（真实产物形态）：[Proxy] 段每行 `名字 = 类型,主机,端口,加密,"密码"`
func validateLoon(payload string) error {
	lines := formatNonEmptyLines(payload)
	proxyLines := sectionLines(lines, "[Proxy]", "[Remote Rule]", "[Rule]", "[General]")
	if len(proxyLines) == 0 {
		return fmt.Errorf("[Proxy] 段为空")
	}
	defined := map[string]bool{}
	for i, l := range proxyLines {
		eq := strings.Index(l, "=")
		if eq < 0 {
			return fmt.Errorf("[Proxy] 第 %d 行缺少 '=': %.40q", i+1, l)
		}
		name := strings.TrimSpace(l[:eq])
		if name == "" {
			return fmt.Errorf("[Proxy] 第 %d 行名称为空", i+1)
		}
		if defined[name] {
			return fmt.Errorf("[Proxy] 名称重复: %s", name)
		}
		defined[name] = true
		fields := strings.Split(l[eq+1:], ",")
		if len(fields) < 3 {
			return fmt.Errorf("[Proxy] %s 字段不足（需要 类型,主机,端口,...）", name)
		}
		if strings.TrimSpace(fields[0]) == "" {
			return fmt.Errorf("[Proxy] %s 缺少类型", name)
		}
		if strings.TrimSpace(fields[1]) == "" {
			return fmt.Errorf("[Proxy] %s 缺少主机", name)
		}
		port, err := strconv.Atoi(strings.TrimSpace(fields[2]))
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("[Proxy] %s 端口非法: %q", name, strings.TrimSpace(fields[2]))
		}
	}
	return nil
}

// validateQuantumultX QX（真实产物形态）：
//
//	[server_local] shadowsocks = host:port, method=..., password=..., tag=名字
//	[policy]       static=组名, 成员..., img-url=...
//	               available=组名, 成员..., img-url=...
//
// [policy] 里 `=` 之后的第一项是**策略自己的名字**，其后才是成员，最后是 img-url= 之类的参数。
func validateQuantumultX(payload string) error {
	lines := formatNonEmptyLines(payload)
	serverLines := sectionLines(lines, "[server_local]", "[policy]", "[filter_remote]", "[filter_local]")
	policyLines := sectionLines(lines, "[policy]", "[filter_remote]", "[filter_local]")
	if len(serverLines) == 0 {
		return fmt.Errorf("[server_local] 段为空")
	}
	defined := map[string]bool{}
	for i, l := range serverLines {
		eq := strings.Index(l, "=")
		if eq < 0 {
			return fmt.Errorf("[server_local] 第 %d 行缺少 '='", i+1)
		}
		typ := strings.TrimSpace(strings.ToLower(l[:eq]))
		switch typ {
		case "shadowsocks", "trojan", "vmess", "vless", "http", "socks5", "anytls":
		default:
			return fmt.Errorf("[server_local] 第 %d 行类型 %q 不被 QX 支持", i+1, typ)
		}
		seg := strings.Split(l[eq+1:], ",")
		hostPort := strings.TrimSpace(seg[0])
		if !strings.Contains(hostPort, ":") {
			return fmt.Errorf("[server_local] %s 缺少 host:port", typ)
		}
		portStr := hostPort[strings.LastIndex(hostPort, ":")+1:]
		port, err := strconv.Atoi(portStr)
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("[server_local] %s 端口非法: %q", typ, portStr)
		}
		tag := ""
		for _, seg2 := range seg[1:] {
			seg2 = strings.TrimSpace(seg2)
			if strings.HasPrefix(seg2, "tag=") {
				tag = strings.TrimSpace(strings.TrimPrefix(seg2, "tag="))
			}
		}
		if tag == "" {
			return fmt.Errorf("[server_local] %s 缺少 tag=", typ)
		}
		if defined[tag] {
			return fmt.Errorf("[server_local] tag 重复: %s", tag)
		}
		defined[tag] = true
	}
	// 先登记策略名，再校验成员（允许前向引用）
	policyNames := map[string]bool{}
	for _, l := range policyLines {
		eq := strings.Index(l, "=")
		if eq < 0 {
			return fmt.Errorf("[policy] 行缺少 '=': %.40q", l)
		}
		items := strings.Split(l[eq+1:], ",")
		if len(items) > 0 {
			policyNames[strings.TrimSpace(items[0])] = true
		}
	}
	for _, l := range policyLines {
		eq := strings.Index(l, "=")
		key := strings.TrimSpace(l[:eq])
		items := strings.Split(l[eq+1:], ",")
		for idx, m := range items {
			m = strings.TrimSpace(m)
			if idx == 0 || m == "" || strings.Contains(m, "=") {
				continue // 第 0 项是策略自身名字；含 '=' 的是 img-url= 等参数
			}
			if defined[m] || policyNames[m] || surgePolicyKeyword(m) {
				continue
			}
			return fmt.Errorf("[policy] %s 引用了不存在的成员 %q（成员悬空）", key, m)
		}
	}
	return nil
}

// sortFormatDrops 稳定输出顺序（便于测试与报告）
func sortFormatDrops(d []FormatDropped) {
	sort.Slice(d, func(i, j int) bool { return d[i].Name < d[j].Name })
}

// validateClashYAML Clash 输出的结构校验（纯语法，不依赖内核）：
// YAML 可解析 + proxies 非空 + 每条含 name/type/server/port + type 在白名单 +
// 名字唯一 + **分组成员必须能解析到已存在的代理/分组名**（一个节点被剔除后最容易悬空的地方）。
func validateClashYAML(payload string) error {
	if strings.TrimSpace(payload) == "" {
		return fmt.Errorf("Clash 配置为空")
	}
	var doc struct {
		Proxies     []map[string]interface{} `yaml:"proxies"`
		ProxyGroups []map[string]interface{} `yaml:"proxy-groups"`
	}
	if err := yaml.Unmarshal([]byte(payload), &doc); err != nil {
		return fmt.Errorf("YAML 不可解析: %w", err)
	}
	if len(doc.Proxies) == 0 {
		return fmt.Errorf("proxies 为空")
	}
	names := make(map[string]bool, len(doc.Proxies)+len(doc.ProxyGroups))
	for i, p := range doc.Proxies {
		name := yamlStr(p["name"])
		typ := yamlStr(p["type"])
		server := yamlStr(p["server"])
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("proxies[%d] 缺少 name", i)
		}
		if names[name] {
			return fmt.Errorf("proxies[%d] 名称重复: %s", i, name)
		}
		names[name] = true
		if strings.TrimSpace(typ) == "" {
			return fmt.Errorf("proxies[%d](%s) 缺少 type", i, name)
		}
		if !mihomoSupportedNodeTypes[NormalizeNodeType(typ)] {
			return fmt.Errorf("proxies[%d](%s) type=%q 不在内核协议白名单内", i, name, typ)
		}
		if strings.TrimSpace(server) == "" {
			return fmt.Errorf("proxies[%d](%s) 缺少 server", i, name)
		}
		port, ok := yamlInt(p["port"])
		if !ok || port < 1 || port > 65535 {
			return fmt.Errorf("proxies[%d](%s) port 非法", i, name)
		}
	}
	for _, g := range doc.ProxyGroups {
		names[yamlStr(g["name"])] = true
	}
	for gi, g := range doc.ProxyGroups {
		gname := yamlStr(g["name"])
		if strings.TrimSpace(gname) == "" {
			return fmt.Errorf("proxy-groups[%d] 缺少 name", gi)
		}
		if _, usesProvider := g["use"]; usesProvider {
			continue
		}
		members, _ := g["proxies"].([]interface{})
		if len(members) == 0 {
			return fmt.Errorf("proxy-groups[%d](%s) 成员为空（内核报 `use` or `proxies` missing）", gi, gname)
		}
		for _, m := range members {
			mn, ok := m.(string)
			if !ok || strings.TrimSpace(mn) == "" {
				return fmt.Errorf("proxy-groups[%d](%s) 存在空成员", gi, gname)
			}
			if !names[mn] && !clashBuiltinNames[mn] {
				return fmt.Errorf("proxy-groups[%d](%s) 引用了不存在的成员 %q（成员悬空）", gi, gname, mn)
			}
		}
	}
	return nil
}

func yamlStr(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func yamlInt(v interface{}) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64:
		return int(t), true
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
			return i, true
		}
	}
	return 0, false
}

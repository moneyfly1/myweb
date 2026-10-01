package config_update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cboard-go/internal/models"
)

// ============================================================================
// 第二层防御：生成 Clash 配置后用真内核自检（mihomo -t）+ 折半二分剔除 + 回退
//
// 设计要点：
//  1. 绝不每次订阅请求都跑内核：按「节点集合指纹(sha256)」缓存已验证结论，
//     指纹未变直接复用；只有节点集合变化才重新自检。
//  2. 自检超时（5s）按失败处理，并受「请求级预算」约束（默认 2.5s），
//     预算耗尽立刻降级到静态校验通过的配置，绝不把客户卡死。
//  3. 失败时折半递归定位坏节点（最坏逐个），只剔除被定位到的那几个节点，
//     不整源丢弃。
//  4. 回退阶梯：内核判定该指纹可用 → 本订阅上一份已验证通过的 YAML →
//     静态校验通过的配置。绝不下发空配置或未校验的坏配置。
// ============================================================================

const (
	// kernelTestTimeout 单次 mihomo -t 超时
	kernelTestTimeout = 5 * time.Second
	// kernelRequestBudget 单次订阅请求允许花在自检上的总预算（超时即降级）
	kernelRequestBudget = 2500 * time.Millisecond
	// kernelMaxTests 单次请求最多允许的内核调用次数（防御性上限）
	kernelMaxTests = 64
	// kernelVerifiedTTL 「该节点集合可用」结论的缓存时长
	kernelVerifiedTTL = 15 * time.Minute
	// kernelBadNodeTTL 单个「坏节点」结论的缓存时长
	kernelBadNodeTTL = 30 * time.Minute
	// kernelLastGoodTTL 每个订阅上一份已验证通过配置的保留时长
	kernelLastGoodTTL = 60 * time.Minute
	// kernelLastGoodMax 上一份良好配置的内存条目上限
	kernelLastGoodMax = 4096
)

var errKernelUnavailable = errors.New("mihomo 内核不可用")

// kernelSelfCheckDisabled 是否关闭内核自检（运维兜底开关；关闭后退化为纯静态校验）
func kernelSelfCheckDisabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MF_KERNEL_SELFCHECK"))) {
	case "0", "false", "off", "no", "disabled":
		return true
	}
	return false
}

// mihomoBinaryPath 定位 mihomo 内核二进制：环境变量 MF_MIHOMO_BIN > 项目 bin/mihomo
func mihomoBinaryPath() string {
	if p := strings.TrimSpace(os.Getenv("MF_MIHOMO_BIN")); p != "" {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	for _, p := range []string{
		filepath.Join("bin", "mihomo"),
		filepath.Join("bin", "mihomo.exe"),
	} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

// selfCheckWorkDir 自检工作目录：内核会在此缓存 geoip.metadb，
// 必须持久化，否则每次 -t 都会重新下载 GeoIP（实测会让单次自检从 90ms 涨到 600ms+）。
func (s *ConfigUpdateService) selfCheckWorkDir() string {
	dir := filepath.Join("uploads", "config", ".selfcheck")
	if cfg, err := s.getConfig(); err == nil {
		if td, ok := cfg["target_dir"].(string); ok && td != "" {
			dir = filepath.Join(td, ".selfcheck")
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	return dir
}

// firstKernelError 从内核输出里提取第一条 level=error 信息（用于日志与原因码）
func firstKernelError(out []byte) string {
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "level=error") {
			continue
		}
		if i := strings.Index(line, `msg="`); i >= 0 {
			msg := line[i+len(`msg="`):]
			if j := strings.LastIndex(msg, `"`); j >= 0 {
				msg = msg[:j]
			}
			return strings.TrimSpace(msg)
		}
		return strings.TrimSpace(line)
	}
	return "内核未给出错误详情"
}

// kernelTestHook 测试注入点：非 nil 时替代真实 mihomo -t 调用。
// 生产环境恒为 nil，单元测试用它精确构造「哪个节点会让配置失效」的场景。
var kernelTestHook func(config string) error

// runKernelTest 用真内核校验一份 Clash YAML。返回 nil 表示通过。
func (s *ConfigUpdateService) runKernelTest(config string) error {
	if strings.TrimSpace(config) == "" {
		return errors.New("待自检配置为空")
	}
	bin := mihomoBinaryPath()
	if bin == "" {
		return errKernelUnavailable
	}
	workDir := s.selfCheckWorkDir()
	if workDir == "" {
		return fmt.Errorf("自检工作目录不可用: %s", workDir)
	}

	f, err := os.CreateTemp(workDir, "selfcheck-*.yaml")
	if err != nil {
		return fmt.Errorf("创建临时配置失败: %w", err)
	}
	name := f.Name()
	defer func() { _ = os.Remove(name) }()

	if _, err := f.WriteString(config); err != nil {
		_ = f.Close()
		return fmt.Errorf("写入临时配置失败: %w", err)
	}
	_ = f.Close()

	ctx, cancel := context.WithTimeout(context.Background(), kernelTestTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, "-t", "-d", workDir, "-f", name)
	cmd.Env = append(os.Environ(), "SAFE_PATHS="+workDir)
	out, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("内核自检超时(%s)", kernelTestTimeout)
	}
	if err != nil {
		return errors.New(firstKernelError(out))
	}
	return nil
}

// kernelFingerprint 节点集合指纹：顺序敏感（配置输出与顺序相关），
// 覆盖身份字段 + 影响内核解析的全部参数（凭据只参与哈希，不落盘）。
func kernelFingerprint(nodes []*ProxyNode) string {
	h := sha256.New()
	for _, n := range nodes {
		if n == nil {
			continue
		}
		h.Write([]byte(nodeIdentity(n)))
		h.Write([]byte{0})
		payload := struct {
			UUID     string
			Password string
			Cipher   string
			Network  string
			TLS      bool
			UDP      bool
			Options  map[string]any
		}{n.UUID, n.Password, n.Cipher, n.Network, n.TLS, n.UDP, n.Options}
		if b, err := json.Marshal(payload); err == nil { // json.Marshal 对 map key 排序，结果稳定
			h.Write(b)
		}
		h.Write([]byte{0xff})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// ---------------------------------------------------------------- 自检缓存

type badNodeEntry struct {
	Reason string
	At     time.Time
}

type lastGoodEntry struct {
	Config string
	At     time.Time
}

type kernelSelfCheckCache struct {
	mu       sync.RWMutex
	verified map[string]time.Time
	bad      map[string]badNodeEntry
	lastGood map[string]lastGoodEntry
	lastSeq  int64
}

var kernelCache = &kernelSelfCheckCache{
	verified: make(map[string]time.Time),
	bad:      make(map[string]badNodeEntry),
	lastGood: make(map[string]lastGoodEntry),
}

func (c *kernelSelfCheckCache) markVerified(fp string) {
	if fp == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.verified[fp] = time.Now()
	if len(c.verified) > 4096 {
		c.sweepLocked()
	}
}

func (c *kernelSelfCheckCache) isVerified(fp string) bool {
	if fp == "" {
		return false
	}
	c.mu.RLock()
	at, ok := c.verified[fp]
	c.mu.RUnlock()
	return ok && time.Since(at) < kernelVerifiedTTL
}

func (c *kernelSelfCheckCache) markBad(key, reason string) {
	if key == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bad[key] = badNodeEntry{Reason: reason, At: time.Now()}
}

func (c *kernelSelfCheckCache) badReason(key string) (string, bool) {
	c.mu.RLock()
	e, ok := c.bad[key]
	c.mu.RUnlock()
	if !ok || time.Since(e.At) >= kernelBadNodeTTL {
		return "", false
	}
	return e.Reason, true
}

func (c *kernelSelfCheckCache) setLastGood(token, config string) {
	if token == "" || strings.TrimSpace(config) == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastSeq++
	c.lastGood[token] = lastGoodEntry{Config: config, At: time.Now()}
	if len(c.lastGood) > kernelLastGoodMax {
		c.sweepLocked()
	}
}

func (c *kernelSelfCheckCache) getLastGood(token string) string {
	if token == "" {
		return ""
	}
	c.mu.RLock()
	e, ok := c.lastGood[token]
	c.mu.RUnlock()
	if !ok || time.Since(e.At) >= kernelLastGoodTTL {
		return ""
	}
	return e.Config
}

func (c *kernelSelfCheckCache) sweepLocked() {
	now := time.Now()
	for k, at := range c.verified {
		if now.Sub(at) >= kernelVerifiedTTL {
			delete(c.verified, k)
		}
	}
	for k, e := range c.bad {
		if now.Sub(e.At) >= kernelBadNodeTTL {
			delete(c.bad, k)
		}
	}
	for k, e := range c.lastGood {
		if now.Sub(e.At) >= kernelLastGoodTTL {
			delete(c.lastGood, k)
		}
	}
}

// ---------------------------------------------------------------- 预算控制

type kernelBudget struct {
	deadline time.Time
	tests    int
	exceeded bool
}

func newKernelBudget(d time.Duration) *kernelBudget {
	return &kernelBudget{deadline: time.Now().Add(d)}
}

func (b *kernelBudget) allow() bool {
	return !b.exceeded && b.tests < kernelMaxTests && time.Now().Before(b.deadline)
}

// ---------------------------------------------------------------- 自检主流程

// kernelTestNodes 用「信息节点 + 给定业务节点」生成配置并跑内核自检
func (s *ConfigUpdateService) kernelTestNodes(info, subset []*ProxyNode, ctx *SubscriptionContext, b *kernelBudget) (bool, error) {
	if !b.allow() {
		b.exceeded = true
		return false, fmt.Errorf("自检预算耗尽")
	}
	b.tests++
	all := make([]*ProxyNode, 0, len(info)+len(subset))
	all = append(all, info...)
	all = append(all, subset...)
	cfg := s.generateClashYAML(all, ctx)
	if kernelTestHook != nil {
		if err := kernelTestHook(cfg); err != nil {
			return false, err
		}
		return true, nil
	}
	if err := s.runKernelTest(cfg); err != nil {
		if errors.Is(err, errKernelUnavailable) {
			return false, err
		}
		return false, err
	}
	return true, nil
}

// divideConquer 折半递归定位坏节点：返回可通过内核的节点与必须剔除的节点。
// 只有「整块通过」才保留，否则继续折半，最坏逐个定位。
func (s *ConfigUpdateService) divideConquer(nodes, info []*ProxyNode, ctx *SubscriptionContext, b *kernelBudget) (good, bad []*ProxyNode) {
	if len(nodes) == 0 {
		return nil, nil
	}
	if !b.allow() {
		b.exceeded = true
		return nodes, nil // 预算耗尽：保守保留（由上层用最终整体自检兜底）
	}
	if ok, _ := s.kernelTestNodes(info, nodes, ctx, b); ok {
		return nodes, nil
	}
	if len(nodes) == 1 {
		return nil, nodes
	}
	mid := len(nodes) / 2
	g1, b1 := s.divideConquer(nodes[:mid], info, ctx, b)
	g2, b2 := s.divideConquer(nodes[mid:], info, ctx, b)
	good = append(append(good, g1...), g2...)
	bad = append(append(bad, b1...), b2...)
	return good, bad
}

// baseProbeNode 基线探针节点：内核实测永远合法的最小 ss 节点。
// 用途：判断「生成管线 + 模板」本身能否产出合法配置。
// 不能用"只有提示节点"的配置当探针——模板里的聚合分组（自动选择/故障转移/负载均衡）
// 在没有任何真实节点时本身就会让内核报错，那会把"节点问题"误判成"模板问题"。
func baseProbeNode() *ProxyNode {
	return &ProxyNode{
		Name: "__mf_selfcheck_probe__", Type: "ss", Server: "1.2.3.4", Port: 8388,
		Cipher: "aes-128-gcm", Password: "probe", Options: map[string]any{},
	}
}

// splitInfoNodes 把面板注入的提示节点（baidu.com 占位）与真实业务节点分开
func splitInfoNodes(proxies []*ProxyNode) (real, info []*ProxyNode) {
	for _, p := range proxies {
		if isPlaceholderInfoNode(p) {
			info = append(info, p)
			continue
		}
		real = append(real, p)
	}
	return real, info
}

// dropKnownBad 剔除缓存里已知会被内核拒绝的节点
func dropKnownBad(nodes []*ProxyNode) (kept []*ProxyNode, dropped []*ProxyNode, reasons map[string]string) {
	reasons = make(map[string]string)
	for _, n := range nodes {
		if n == nil {
			continue
		}
		if r, ok := kernelCache.badReason(nodeIdentity(n)); ok {
			reasons[nodeIdentity(n)] = r
			dropped = append(dropped, n)
			continue
		}
		kept = append(kept, n)
	}
	return kept, dropped, reasons
}

// buildSelfCheckedClashConfig 第二层入口：生成 Clash 配置 → 内核自检 → 剔除坏节点 → 回退。
// cacheToken 用于记住「本订阅上一份已验证通过的配置」；可为空。
func (s *ConfigUpdateService) buildSelfCheckedClashConfig(proxies []*ProxyNode, ctx *SubscriptionContext, cacheToken string) string {
	// 关闭开关或内核缺失：退化为纯静态校验（快、无外部依赖），并明确日志
	if kernelSelfCheckDisabled() {
		kept, events := StaticValidateNodes(proxies, "generate(static-only)")
		s.recordNodeValidationEvents(events)
		return s.generateClashYAML(kept, ctx)
	}
	if mihomoBinaryPath() == "" {
		kept, events := StaticValidateNodes(proxies, "generate(no-kernel)")
		s.recordNodeValidationEvents(events)
		logf("未找到 mihomo 内核二进制，本次仅做静态校验（降级模式）")
		cfg := s.generateClashYAML(kept, ctx)
		s.selfCheckCache().setLastGood(cacheToken, cfg)
		return cfg
	}

	real, info := splitInfoNodes(proxies)

	// ① 剔除缓存中的已知坏节点
	real, knownBad, knownReasons := dropKnownBad(real)
	if len(knownBad) > 0 {
		logf("生成前剔除 %d 个缓存中的已知坏节点", len(knownBad))
	}

	fp := kernelFingerprint(real)

	// ② 指纹命中：该节点集合已被内核确认可用，直接复用结论（零内核开销）
	if kernelCache.isVerified(fp) {
		cfg := s.generateClashYAML(append(append([]*ProxyNode{}, info...), real...), ctx)
		kernelCache.setLastGood(cacheToken, cfg)
		return cfg
	}

	// ③ 需要重新自检
	b := newKernelBudget(kernelRequestBudget)
	ok, terr := s.kernelTestNodes(info, real, ctx, b)
	good, bad := real, []*ProxyNode(nil)
	badReasons := map[string]string{}

	switch {
	case ok:
		// 整体通过
	case errors.Is(terr, errKernelUnavailable):
		kept, events := StaticValidateNodes(proxies, "generate(no-kernel)")
		s.recordNodeValidationEvents(events)
		cfg := s.generateClashYAML(kept, ctx)
		kernelCache.setLastGood(cacheToken, cfg)
		return cfg
	case b.exceeded:
		logf("内核自检预算耗尽（已调用 %d 次），本次降级为静态校验配置", b.tests)
		return s.fallbackClashConfig(proxies, ctx, cacheToken, "自检预算耗尽")
	default:
		// 基线检查：插一个「内核实测永远合法」的合成节点后能否通过？
		// 不能 → 问题出在生成管线/模板，而不是用户的节点，此时绝不能剔除任何业务节点。
		probe := []*ProxyNode{baseProbeNode()}
		if baseOK, baseErr := s.kernelTestNodes(info, probe, ctx, b); !baseOK {
			logf("基线探针自检失败（业务节点全量自检错误=%v；基线错误=%v），判定为模板/管线问题，不剔除任何节点", terr, baseErr)
			return s.fallbackClashConfig(proxies, ctx, cacheToken, "模板配置自检失败")
		}
		logf("内核自检失败，开始折半定位坏节点：%v", terr)
		good, bad = s.divideConquer(real, info, ctx, b)

		// 最终整体复检：只有在确认「剔除后的集合」真正通过时才认定成功
		if finalOK, ferr := s.kernelTestNodes(info, good, ctx, b); !finalOK {
			logf("剔除后复检仍未通过（%v），进入回退阶梯", ferr)
			return s.fallbackClashConfig(proxies, ctx, cacheToken, "剔除后复检失败")
		}
		for _, n := range bad {
			reason := ReasonKernelInvalid
			badReasons[nodeIdentity(n)] = reason
			kernelCache.markBad(nodeIdentity(n), reason)
		}
		if len(bad) > 0 {
			events := make([]NodeValidationEvent, 0, len(bad))
			for _, n := range bad {
				events = append(events, NewNodeValidationEvent(models.NodeValidationPrunedAtGenerate, "generate",
					n, ReasonKernelInvalid, "内核 mihomo -t 判定该节点会使整份配置失效（折半二分定位）"))
			}
			s.recordNodeValidationEvents(events)
			logf("内核自检剔除 %d 个坏节点（最小代价剔除，未整源丢弃）", len(bad))
		} else {
			logf("内核自检通过（无需剔除）")
		}
	}

	if len(knownBad) > 0 {
		events := make([]NodeValidationEvent, 0, len(knownBad))
		for _, n := range knownBad {
			r := knownReasons[nodeIdentity(n)]
			if r == "" {
				r = ReasonKernelInvalid
			}
			events = append(events, NewNodeValidationEvent(models.NodeValidationPrunedAtGenerate, "generate-cache",
				n, r, "命中「已知坏节点」缓存，直接剔除"))
		}
		s.recordNodeValidationEvents(events)
	}

	kernelCache.markVerified(kernelFingerprint(good))
	final := append(append([]*ProxyNode{}, info...), good...)
	cfg := s.generateClashYAML(final, ctx)
	kernelCache.setLastGood(cacheToken, cfg)
	return cfg
}

// sanitizeProxiesForOutput 非 Clash 输出路径（Surge / sing-box / QuantumultX / Loon / 通用链接）
// 的第一层净化：就地（在副本上）修正可修的密钥编码，并丢弃静态校验不通过的节点。
//
// 刻意不写 node_validation_logs：采集阶段已经记录过同一批事件，
// 若在这里再记一次会在每个订阅请求上重复落库。这里只负责"别把坏节点发出去"。
// 与 generateClashYAML 一致，先深拷贝再改，避免污染 GetSystemNodesCache/ParseCache 的共享节点。
func sanitizeProxiesForOutput(nodes []*ProxyNode) []*ProxyNode {
	out := make([]*ProxyNode, 0, len(nodes))
	for _, src := range nodes {
		if src == nil {
			continue
		}
		p := *src
		if src.Options != nil {
			p.Options = deepCopyOptions(src.Options)
		}
		NormalizeSS2022Key(&p)
		if ValidateProxyNode(&p) != nil {
			continue
		}
		out = append(out, &p)
	}
	return out
}

// fallbackClashConfig 回退阶梯：绝不下发空配置或坏配置
func (s *ConfigUpdateService) fallbackClashConfig(proxies []*ProxyNode, ctx *SubscriptionContext, cacheToken, reason string) string {
	// ① 本订阅上一份「内核已验证通过」的配置
	if cfg := kernelCache.getLastGood(cacheToken); cfg != "" {
		logf("回退到本订阅上一份已验证通过的配置（原因: %s）", reason)
		return cfg
	}
	// ② 静态校验通过的配置（不依赖内核的降级路径）
	kept, events := StaticValidateNodes(proxies, "generate-fallback")
	s.recordNodeValidationEvents(events)
	cfg := s.generateClashYAML(kept, ctx)
	if strings.TrimSpace(cfg) == "" {
		// 极端兜底：连静态配置都生成不出来时，仍返回无业务节点的提示配置
		_, info := splitInfoNodes(proxies)
		cfg = s.generateClashYAML(info, ctx)
	}
	logf("无历史良好配置，降级为静态校验通过的配置（原因: %s）", reason)
	return cfg
}

// recordNodeValidationEvents 记录校验事件：server.log 可见 + 落库供后台展示
func (s *ConfigUpdateService) recordNodeValidationEvents(events []NodeValidationEvent) {
	if len(events) == 0 {
		return
	}
	for _, e := range events {
		logf("[%s] 来源=%s 节点=%q 类型=%s 地址=%s:%d 原因=%s",
			e.Event, e.Source, e.NodeName, e.NodeType, e.Server, e.Port, e.Reason)
	}
	RecordNodeValidationEvents(s.db, events)
}

// PrewarmKernelSelfCheck 在采集同步结束后于后台预热内核自检缓存，
// 让「节点变化后的第一次订阅请求」也能命中缓存，不被自检拖慢。
func (s *ConfigUpdateService) PrewarmKernelSelfCheck() {
	if kernelSelfCheckDisabled() || mihomoBinaryPath() == "" {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logf("内核自检预热 panic 已恢复: %v", r)
			}
		}()
		nodes := s.loadActiveSystemNodes()
		if len(nodes) == 0 {
			return
		}
		real, info := splitInfoNodes(nodes)
		if kernelCache.isVerified(kernelFingerprint(real)) {
			return
		}
		// 用更宽松的后台预算（不占用请求路径）
		b := newKernelBudget(90 * time.Second)
		ctx := &SubscriptionContext{Status: StatusNormal}
		start := time.Now()
		if ok, err := s.kernelTestNodes(info, real, ctx, b); ok {
			kernelCache.markVerified(kernelFingerprint(real))
			logf("内核自检预热完成：%d 个节点通过，耗时 %s", len(real), time.Since(start).Round(time.Millisecond))
			return
		} else if !errors.Is(err, errKernelUnavailable) {
			logf("内核自检预热发现坏节点，开始折半定位（后台，不影响订阅请求）")
		}
		good, bad := s.divideConquer(real, info, ctx, b)
		if len(bad) > 0 {
			events := make([]NodeValidationEvent, 0, len(bad))
			for _, n := range bad {
				kernelCache.markBad(nodeIdentity(n), ReasonKernelInvalid)
				events = append(events, NewNodeValidationEvent(models.NodeValidationPrunedAtGenerate, "prewarm",
					n, ReasonKernelInvalid, "内核 mihomo -t 判定该节点会使整份配置失效（后台预热折半定位）"))
			}
			s.recordNodeValidationEvents(events)
		}
		if ok, _ := s.kernelTestNodes(info, good, ctx, b); ok {
			kernelCache.markVerified(kernelFingerprint(good))
		}
		logf("内核自检预热完成：保留 %d 个、剔除 %d 个，耗时 %s", len(good), len(bad), time.Since(start).Round(time.Millisecond))
	}()
}

// loadActiveSystemNodes 读取当前活跃的系统节点（与 appendSystemNodes 同源），
// 并跑一遍第一层静态校验，供后台预热使用。
func (s *ConfigUpdateService) loadActiveSystemNodes() []*ProxyNode {
	var nodes []models.Node
	if autoDisableTimeoutEnabled(s.db) {
		s.db.Where("is_active = ? AND status != ?", true, "timeout").Order("order_index ASC, created_at ASC").Find(&nodes)
	} else {
		s.db.Where("is_active = ?", true).Order("order_index ASC, created_at ASC").Find(&nodes)
	}
	proxies := make([]*ProxyNode, 0, len(nodes))
	seen := make(map[string]bool)
	for _, n := range nodes {
		if n.Config == nil || *n.Config == "" {
			continue
		}
		var p ProxyNode
		if json.Unmarshal([]byte(*n.Config), &p) != nil {
			continue
		}
		p.Name = n.Name
		if corrected, detail := NormalizeSS2022Key(&p); corrected {
			s.recordNodeValidationEvents([]NodeValidationEvent{
				NewNodeValidationEvent(models.NodeValidationCorrectedAtIngest, "prewarm", &p, ReasonKeyURLDecoded, detail),
			})
		}
		if err := ValidateProxyNode(&p); err != nil {
			ve, ok := err.(*NodeValidationError)
			if !ok {
				ve = &NodeValidationError{Code: ReasonKernelInvalid, Detail: err.Error()}
			}
			s.recordNodeValidationEvents([]NodeValidationEvent{
				NewNodeValidationEvent(models.NodeValidationDroppedAtIngest, "prewarm", &p, ve.Code, ve.Detail),
			})
			continue
		}
		key := fmt.Sprintf("%s:%s:%d:%s", p.Type, p.Server, p.Port, p.Name)
		if seen[key] {
			continue
		}
		seen[key] = true
		proxies = append(proxies, &p)
	}
	return proxies
}

// selfCheckCache 暴露缓存（便于测试与后续扩展）
func (s *ConfigUpdateService) selfCheckCache() *kernelSelfCheckCache { return kernelCache }

package config_update

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cboard-go/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupSelfCheckTest 建内存库 + 伪造一个存在的「内核二进制路径」，
// 并用 kernelTestHook 拦截真实 mihomo 调用（单元测试里不依赖外部二进制）。
// hook 返回值由调用方通过 returned hook 决定。
func setupSelfCheckTest(t *testing.T, hook func(cfg string) error) (*ConfigUpdateService, *int) {
	t.Helper()

	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Node{}, &models.SystemConfig{}, &models.NodeValidationLog{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})

	// 让 mihomoBinaryPath() 认为内核存在（真实调用被 hook 拦截，不会执行该文件）
	binPath := filepath.Join(t.TempDir(), "mihomo-fake")
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("写入假内核失败: %v", err)
	}
	t.Setenv("MF_MIHOMO_BIN", binPath)
	t.Setenv("MF_KERNEL_SELFCHECK", "")

	calls := 0
	kernelTestHook = func(cfg string) error {
		calls++
		return hook(cfg)
	}
	t.Cleanup(func() { kernelTestHook = nil })

	svc := &ConfigUpdateService{db: db}
	// 每个用例用干净缓存，避免相互污染
	kernelCache = &kernelSelfCheckCache{
		verified: make(map[string]time.Time),
		bad:      make(map[string]badNodeEntry),
		lastGood: make(map[string]lastGoodEntry),
	}
	return svc, &calls
}

func mkSS(name string) *ProxyNode {
	return &ProxyNode{Name: name, Type: "ss", Server: "1.2.3.4", Port: 8388,
		Cipher: "aes-128-gcm", Password: "pw", Options: map[string]any{}}
}

func mkNodes(n int, bad map[string]bool) []*ProxyNode {
	out := make([]*ProxyNode, 0, n)
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("n%02d", i)
		if bad[name] {
			name = "bad-" + name
		}
		out = append(out, mkSS(name))
	}
	return out
}

// failingOn 返回「配置里只要出现这些坏节点名就失败」的假内核
func failingOn(badSubstrings ...string) func(string) error {
	return func(cfg string) error {
		for _, b := range badSubstrings {
			if strings.Contains(cfg, b) {
				return fmt.Errorf("proxy: unsupported security type（模拟内核拒绝 %s）", b)
			}
		}
		return nil
	}
}

// 核心：一个坏节点藏在 32 个节点里，折半二分必须精确定位到它，且调用次数远小于逐个试
func TestKernelBisectLocatesSingleBadNode(t *testing.T) {
	svc, calls := setupSelfCheckTest(t, failingOn("bad-n07"))

	nodes := mkNodes(32, map[string]bool{"n07": true})
	ctx := &SubscriptionContext{Status: StatusNormal}
	b := newKernelBudget(10 * time.Second)

	// 整体先失败
	if ok, err := svc.kernelTestNodes(nil, nodes, ctx, b); ok || err == nil {
		t.Fatalf("含坏节点的整体配置应自检失败，ok=%v err=%v", ok, err)
	}
	good, bad := svc.divideConquer(nodes, nil, ctx, b)

	if len(bad) != 1 || bad[0].Name != "bad-n07" {
		names := make([]string, 0, len(bad))
		for _, n := range bad {
			names = append(names, n.Name)
		}
		t.Fatalf("应精确定位到 1 个坏节点 bad-n07，实际剔除 %d 个: %v", len(bad), names)
	}
	if len(good) != 31 {
		t.Fatalf("应保留 31 个节点，实际 %d", len(good))
	}
	// 剔除后必须能通过
	if ok, err := svc.kernelTestNodes(nil, good, ctx, b); !ok {
		t.Fatalf("剔除后应通过自检: %v", err)
	}
	if *calls > 16 {
		t.Fatalf("折半二分调用次数应远小于逐个试(32)，实际 %d 次", *calls)
	}
	t.Logf("32 个节点定位 1 个坏节点：内核调用 %d 次（逐个试需 32 次）", *calls)
}

// 多个坏节点也要全部定位
func TestKernelBisectLocatesMultipleBadNodes(t *testing.T) {
	svc, calls := setupSelfCheckTest(t, failingOn("bad-n03", "bad-n17", "bad-n28"))

	nodes := mkNodes(32, map[string]bool{"n03": true, "n17": true, "n28": true})
	ctx := &SubscriptionContext{Status: StatusNormal}
	b := newKernelBudget(10 * time.Second)

	good, bad := svc.divideConquer(nodes, nil, ctx, b)

	got := map[string]bool{}
	for _, n := range bad {
		got[n.Name] = true
	}
	for _, want := range []string{"bad-n03", "bad-n17", "bad-n28"} {
		if !got[want] {
			t.Fatalf("未定位到坏节点 %s，实际剔除 %v", want, got)
		}
	}
	if len(bad) != 3 {
		t.Fatalf("应剔除 3 个坏节点，实际 %d: %v", len(bad), got)
	}
	if len(good) != 29 {
		t.Fatalf("应保留 29 个节点，实际 %d", len(good))
	}
	if ok, err := svc.kernelTestNodes(nil, good, ctx, b); !ok {
		t.Fatalf("剔除后应通过自检: %v", err)
	}
	t.Logf("32 个节点定位 3 个坏节点：内核调用 %d 次", *calls)
}

// 指纹缓存：同一节点集合第二次请求不得再调用内核（性能硬要求）
func TestKernelSelfCheckCacheAvoidsRepeatedKernelRuns(t *testing.T) {
	svc, calls := setupSelfCheckTest(t, failingOn("bad-n05"))

	nodes := mkNodes(32, map[string]bool{"n05": true})
	ctx := &SubscriptionContext{Status: StatusNormal}

	cfg1 := svc.buildSelfCheckedClashConfig(nodes, ctx, "token-A")
	first := *calls
	if first == 0 {
		t.Fatal("首次生成应触发内核自检")
	}
	if strings.Contains(cfg1, "bad-n05") {
		t.Fatal("首次生成就应剔除坏节点")
	}

	cfg2 := svc.buildSelfCheckedClashConfig(nodes, ctx, "token-A")
	if *calls != first {
		t.Fatalf("指纹未变时不应再调用内核：首次 %d 次，第二次增加到 %d 次", first, *calls)
	}
	if cfg2 != cfg1 {
		t.Fatal("指纹未变时应复用同一份已验证配置")
	}
	t.Logf("缓存命中：首次内核调用 %d 次，第二次 0 次新增", first)
}

// 节点集合变化必须重新自检（缓存不能掩盖真实变化）
func TestKernelSelfCheckCacheInvalidatedOnNodeChange(t *testing.T) {
	svc, calls := setupSelfCheckTest(t, failingOn("bad-n05"))

	ctx := &SubscriptionContext{Status: StatusNormal}
	svc.buildSelfCheckedClashConfig(mkNodes(32, map[string]bool{"n05": true}), ctx, "token-A")
	first := *calls

	// 新增一个节点 → 指纹变化 → 必须重新自检
	changed := append(mkNodes(32, map[string]bool{"n05": true}), mkSS("n99"))
	svc.buildSelfCheckedClashConfig(changed, ctx, "token-A")
	if *calls == first {
		t.Fatal("节点集合变化后必须重新自检，但内核调用次数未增加")
	}
}

// 回退阶梯：本订阅有历史良好配置时，必须回退它而不是下发坏配置
func TestFallbackPrefersLastGoodConfig(t *testing.T) {
	svc, _ := setupSelfCheckTest(t, failingOn()) // 假内核永远通过

	ctx := &SubscriptionContext{Status: StatusNormal}
	good := mkNodes(4, nil)
	cfg := svc.buildSelfCheckedClashConfig(good, ctx, "token-X")
	if strings.TrimSpace(cfg) == "" {
		t.Fatal("正常路径不应返回空配置")
	}

	// 模拟「模板本身坏了」：基线自检也失败 → 不剔除节点，走回退
	kernelTestHook = func(string) error { return fmt.Errorf("模板配置无效") }
	fallback := svc.fallbackClashConfig(good, ctx, "token-X", "测试回退")
	if fallback != cfg {
		t.Fatal("有历史良好配置时应回退到它")
	}

	// 没有历史良好配置的订阅：退化为静态校验通过的配置，且绝不为空
	fallback2 := svc.fallbackClashConfig(good, ctx, "token-unknown", "测试回退")
	if strings.TrimSpace(fallback2) == "" {
		t.Fatal("无历史配置时也必须返回静态校验通过的配置，绝不能为空")
	}
	if !strings.Contains(fallback2, "n00") {
		t.Fatal("静态兜底配置应保留合法节点")
	}
}

// 基线（无业务节点）自检失败 = 模板问题，绝不能把用户节点全剔光
func TestBaseFailureDoesNotPruneAllNodes(t *testing.T) {
	svc, _ := setupSelfCheckTest(t, func(cfg string) error {
		return fmt.Errorf("proxy group: 'X' not found（模拟模板问题）")
	})
	ctx := &SubscriptionContext{Status: StatusNormal}
	nodes := mkNodes(8, nil)

	cfg := svc.buildSelfCheckedClashConfig(nodes, ctx, "token-Y")
	if strings.TrimSpace(cfg) == "" {
		t.Fatal("模板问题时也必须下发配置，不能为空")
	}
	if !strings.Contains(cfg, "n00") || !strings.Contains(cfg, "n07") {
		t.Fatal("模板问题不应剔除任何业务节点")
	}
}

// 已知坏节点缓存：后续请求直接跳过，不再为它跑内核
func TestKnownBadNodeIsSkippedWithoutKernelRun(t *testing.T) {
	svc, calls := setupSelfCheckTest(t, failingOn("bad-n05"))

	nodes := mkNodes(16, map[string]bool{"n05": true})
	ctx := &SubscriptionContext{Status: StatusNormal}
	svc.buildSelfCheckedClashConfig(nodes, ctx, "token-Z")
	if _, ok := kernelCache.badReason(nodeIdentity(nodes[5])); !ok {
		t.Fatal("坏节点应进入已知坏节点缓存")
	}

	// 换一个订阅（不同 token → 指纹不同，但坏节点已知）→ 额外内核调用应仅为 1 次（验证剔除后的集合）
	before := *calls
	cfg := svc.buildSelfCheckedClashConfig(nodes, ctx, "token-Z2")
	if strings.Contains(cfg, "bad-n05") {
		t.Fatal("已知坏节点不应出现在下发配置里")
	}
	if *calls-before > 2 {
		t.Fatalf("已知坏节点应直接剔除，不应该再折半定位（额外调用 %d 次）", *calls-before)
	}
}

func TestKernelFingerprintStabilityAndSensitivity(t *testing.T) {
	a := mkNodes(3, nil)
	b := mkNodes(3, nil)
	if kernelFingerprint(a) != kernelFingerprint(b) {
		t.Fatal("相同节点集合应得到相同指纹")
	}
	b[1].Password = "different"
	if kernelFingerprint(a) == kernelFingerprint(b) {
		t.Fatal("凭据变化必须改变指纹")
	}
	c := mkNodes(3, nil)
	c[0], c[1] = c[1], c[0]
	if kernelFingerprint(a) == kernelFingerprint(c) {
		t.Fatal("顺序变化必须改变指纹（配置输出与顺序相关）")
	}
}

// kernelSelfCheckDisabled 开关：关闭后退化为纯静态校验
func TestKernelSelfCheckDisableSwitch(t *testing.T) {
	for _, v := range []string{"0", "false", "OFF", "no", "disabled"} {
		t.Setenv("MF_KERNEL_SELFCHECK", v)
		if !kernelSelfCheckDisabled() {
			t.Fatalf("%q 应关闭内核自检", v)
		}
	}
	t.Setenv("MF_KERNEL_SELFCHECK", "1")
	if kernelSelfCheckDisabled() {
		t.Fatal("1 不应关闭内核自检")
	}
}

// dropKnownBad 只剔除缓存里已知的坏节点，其它原样保留
func TestDropKnownBad(t *testing.T) {
	kernelCache = &kernelSelfCheckCache{verified: map[string]time.Time{}, bad: map[string]badNodeEntry{}, lastGood: map[string]lastGoodEntry{}}
	nodes := mkNodes(4, nil)
	kernelCache.markBad(nodeIdentity(nodes[1]), ReasonKernelInvalid)

	kept, dropped, reasons := dropKnownBad(nodes)
	if len(kept) != 3 || len(dropped) != 1 {
		t.Fatalf("应剔除 1 个保留 3 个，实际剔除 %d 保留 %d", len(dropped), len(kept))
	}
	if dropped[0].Name != "n01" {
		t.Fatalf("剔除的应是 n01，实际 %s", dropped[0].Name)
	}
	if reasons[nodeIdentity(nodes[1])] != ReasonKernelInvalid {
		t.Fatal("应带出剔除原因")
	}
}

// 基线探针必须能区分「节点问题」与「模板问题」。
//
// 回归：旧实现用"只有提示节点的配置"当基线，会因为模板里的聚合分组没有真实成员
// 而被内核判定失败，于是把节点问题误判成模板问题，直接降级（既不定位也不剔除坏节点）。
func TestBaselineProbeDistinguishesNodeProblemFromTemplateProblem(t *testing.T) {
	// 假内核：① 配置里没有真实节点（既无探针也无 nXX）→ 失败（模拟聚合分组为空）；
	//          ② 配置里含 bad-n03 → 失败。
	hook := func(cfg string) error {
		hasReal := strings.Contains(cfg, "__mf_selfcheck_probe__")
		for i := 0; i < 32; i++ {
			if strings.Contains(cfg, fmt.Sprintf("n%02d", i)) {
				hasReal = true
				break
			}
		}
		if !hasReal {
			return fmt.Errorf("proxy group[1]: ♻️ 自动选择 is empty（模拟模板聚合分组无成员）")
		}
		if strings.Contains(cfg, "bad-n03") {
			return fmt.Errorf("proxy: unsupported security type")
		}
		return nil
	}
	svc, calls := setupSelfCheckTest(t, hook)

	nodes := mkNodes(32, map[string]bool{"n03": true})
	ctx := &SubscriptionContext{Status: StatusNormal}
	cfg := svc.buildSelfCheckedClashConfig(nodes, ctx, "token-probe")

	// 必须走「二分定位 + 剔除」，而不是降级
	if strings.Contains(cfg, "bad-n03") {
		t.Fatal("坏节点应被剔除")
	}
	if !strings.Contains(cfg, "n31") {
		t.Fatal("除坏节点外的节点都应保留（不能整源丢弃）")
	}
	if _, ok := kernelCache.badReason(nodeIdentity(nodes[3])); !ok {
		t.Fatal("坏节点应进入已知坏节点缓存")
	}
	t.Logf("基线探针生效：内核调用 %d 次后精确剔除坏节点", *calls)
}

// 模板真的坏掉时：绝不能剔除用户的业务节点（否则等于把客户节点全丢了）
func TestTemplateFailurePrunesNothing(t *testing.T) {
	svc, _ := setupSelfCheckTest(t, func(string) error {
		return fmt.Errorf("proxy group[0]: X: 'Y' not found（模拟模板结构错误）")
	})
	nodes := mkNodes(8, nil)
	ctx := &SubscriptionContext{Status: StatusNormal}
	cfg := svc.buildSelfCheckedClashConfig(nodes, ctx, "token-tpl")

	if strings.TrimSpace(cfg) == "" {
		t.Fatal("模板问题时也必须下发配置")
	}
	for _, want := range []string{"n00", "n07"} {
		if !strings.Contains(cfg, want) {
			t.Fatalf("模板问题不应剔除业务节点，但 %s 不见了", want)
		}
	}
	if len(kernelCache.bad) != 0 {
		t.Fatalf("模板问题不应把任何节点标记为坏节点，实际 %v", kernelCache.bad)
	}
}

// 非 Clash 输出路径的净化：修正密钥编码 + 丢弃不合法节点，且不污染共享缓存节点
func TestSanitizeProxiesForOutput(t *testing.T) {
	key := "Bw4VHCMqMTg/Rk1UW2JpcHd+hYyTmqGor7a9xMvS2eA="
	polluted := strings.ReplaceAll(key, "/", "%2F")

	src := []*ProxyNode{
		{Name: "好节点", Type: "ss", Server: "1.2.3.4", Port: 8388, Cipher: "aes-128-gcm", Password: "pw", Options: map[string]any{}},
		{Name: "待修正", Type: "ss", Server: "1.2.3.5", Port: 8388, Cipher: "2022-blake3-aes-256-gcm", Password: polluted, Options: map[string]any{}},
		{Name: "naive坏节点", Type: "naive", Server: "1.2.3.6", Port: 443, Password: "pw", Options: map[string]any{}},
		{Name: "坏cipher", Type: "ss", Server: "1.2.3.7", Port: 8388, Cipher: "aes-128-ocb", Password: "pw", Options: map[string]any{}},
		nil,
	}
	out := sanitizeProxiesForOutput(src)

	if len(out) != 2 {
		names := make([]string, 0, len(out))
		for _, n := range out {
			names = append(names, n.Name)
		}
		t.Fatalf("应保留 2 个节点（好节点 + 被修正的节点），实际 %d: %v", len(out), names)
	}
	if out[1].Password != key {
		t.Fatalf("密钥应被还原: got %q want %q", out[1].Password, key)
	}
	// 关键：不能就地污染入参（共享缓存节点的竞态与串号风险）
	if src[1].Password != polluted {
		t.Fatalf("不得修改入参节点（共享缓存），实际被改成 %q", src[1].Password)
	}
}

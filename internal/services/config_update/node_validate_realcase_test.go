package config_update

import (
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"cboard-go/internal/models"
)

// ============================================================================
// 真实线上案例验收用例：`荷兰·SS·1倍消耗`（现网 id 2243391 / 2243586）
//
// 现网现象（同日构建的 mihomo 复现原文）：
//
//	level=error msg="proxy 0: ss 203.0.113.11:20001 cipher: 2022-blake3-aes-256-gcm
//	  initialize error: decode key: illegal base64 data at input byte 41"
//	configuration file xxx.yaml test failed
//
// 即：这一个 ss 节点让**整份配置** -t 失败 → 客户内核起不来（用户所说的"核心错误"），
// 而不是"少一个节点"。这正是本次两层防御要防住的最典型场景。
//
// 现网该值长度 93、不是合法 base64；第一个 '%' 正好在下标 41——与内核报的
// "input byte 41" 精确对应，说明内核是在解析到 `%2F` 转义序列时失败的。
// 百分号解码后长度 89，形如 `serverKey:userKey` 两段式，每段解码 32 字节，
// 正好是 2022-blake3-aes-256-gcm 要求的长度。
//
// 结论：**是入库转换的问题，不是源的问题**——源里的链接按 URL 规范对 userinfo 做了
// 百分号转义（'/'→%2F、':'→%3A），而解析层用 parsed.User.String()（编码后形态）当密码，
// 既没解码、也因为 ':' 被转义而无法切分两段。
//
// 安全：仓库内不写入该节点的真实密钥（本仓有"清除真实节点信息"的安全约定）。
// 这里用**等长、同结构**的合成值复现完全相同的失败特征，并在注释里给出实测坐标。
// 真实值只在上线前的服务器端验收里使用（不落仓库）。
// ============================================================================

const (
	realCaseName   = "荷兰·SS·1倍消耗"
	realCaseServer = "203.0.113.11"
	realCasePort   = 20001
	realCaseCipher = "2022-blake3-aes-256-gcm"
)

// realCasePasswordFixture 构造与现网值**等长（93 字符）、同结构**的合成密钥：
// 两段式 base64，其中 '/'→%2F、':'→%3A，第一段的 '%' 正好落在下标 41，
// 且第二段含 base64 合法的 '+'（用于覆盖 QueryUnescape 会把 '+' 解成空格的缺陷）。
// 返回 (污染形态, 解码后形态)。
func realCasePasswordFixture(t *testing.T) (polluted, decoded string) {
	t.Helper()
	// 两段各 32 字节，字符分布刻意与现网一致：
	//   seg1 的 base64 中 '/' 恰好落在下标 41（内核报的就是 input byte 41），且不含 '+'
	//   seg2 的 base64 中含 '+'（用于覆盖 QueryUnescape 会把 '+' 解成空格的缺陷）、不含 '/'
	// 这样拼出的污染形态长度正好 93、首个 '%' 正好在下标 41——与现网实测坐标完全一致。
	seg1 := "QUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBA/A="
	seg2 := "DhsoNUJPXGl2g5CdqrfE0d7r+AUSHyw5RlNgbXqHlKE="
	for i, seg := range []string{seg1, seg2} {
		d, err := base64.StdEncoding.DecodeString(seg)
		if err != nil || len(d) != 32 {
			t.Fatalf("夹具段%d 不是 32 字节合法 base64: %v", i+1, err)
		}
	}
	if strings.Index(seg1, "/") != 41 || strings.Count(seg1, "/") != 1 || strings.Contains(seg1, "+") {
		t.Fatal("seg1 必须在且仅在下标 41 处含 '/'，且不含 '+'")
	}
	if !strings.Contains(seg2, "+") || strings.Contains(seg2, "/") {
		t.Fatal("seg2 必须含 '+' 且不含 '/'")
	}
	decoded = seg1 + ":" + seg2
	polluted = strings.ReplaceAll(seg1, "/", "%2F") + "%3A" + strings.ReplaceAll(seg2, "/", "%2F")

	if len(polluted) != 93 {
		t.Fatalf("夹具长度应与现网一致(93)，实际 %d", len(polluted))
	}
	if strings.Index(polluted, "%") != 41 {
		t.Fatalf("夹具首个 '%%' 应落在下标 41（与内核报错 input byte 41 对应），实际 %d", strings.Index(polluted, "%"))
	}
	if len(decoded) != 89 {
		t.Fatalf("解码后长度应为 89（现网实测值），实际 %d", len(decoded))
	}
	for i, s := range strings.Split(decoded, ":") {
		if d, err := base64.StdEncoding.DecodeString(s); err != nil || len(d) != 32 {
			t.Fatalf("解码后第 %d 段应为 32 字节合法 base64: %v", i+1, err)
		}
	}
	return polluted, decoded
}

func realCaseNode(password string) *ProxyNode {
	return &ProxyNode{Name: realCaseName, Type: "ss", Server: realCaseServer, Port: realCasePort,
		Cipher: realCaseCipher, Password: password, Options: map[string]any{}}
}

// kernelBinForAcceptance 定位真内核；找不到就跳过（CI 无内核时不误报失败）
func kernelBinForAcceptance(t *testing.T) string {
	t.Helper()
	for _, p := range []string{
		filepath.Join("..", "..", "..", "bin", "mihomo"),
		"/www/wwwroot/dy.moneyfly.top/bin/mihomo",
	} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	t.Skip("未找到 mihomo 内核二进制，跳过真内核验收（静态断言仍会执行）")
	return ""
}

// 用真内核对一份 Clash YAML 做 -t，返回 (通过, 原始错误行)
func runKernelOnYAML(t *testing.T, bin, yaml string) (bool, string) {
	t.Helper()
	dir := t.TempDir()
	f := filepath.Join(dir, "acceptance.yaml")
	if err := os.WriteFile(f, []byte(yaml), 0o600); err != nil {
		t.Fatalf("写入临时配置失败: %v", err)
	}
	wd := filepath.Join("..", "..", "..", "uploads", "config", ".selfcheck")
	if st, err := os.Stat(wd); err != nil || !st.IsDir() {
		wd = dir
	}
	cmd := exec.Command(bin, "-t", "-d", wd, "-f", f)
	out, err := cmd.CombinedOutput()
	var errLine string
	for _, l := range strings.Split(string(out), "\n") {
		if strings.Contains(l, "level=error") {
			errLine = l
			break
		}
	}
	return err == nil, errLine
}

// 验收 ①：静态层必须拦住它——不可还原的非法密钥被丢弃，reason 用精确子类
func TestAcceptanceRealCase_StaticLayerRejectsInvalidCipherKey(t *testing.T) {
	polluted, decoded := realCasePasswordFixture(t)

	// ① -a 现网值的"不可还原"变体：把 '%' 去掉（模拟无法自动修复的脏值）
	broken := strings.ReplaceAll(polluted, "%2F", "/") // '/' 保留但 ':' 仍是 %3A → 单段、非法 base64
	broken = strings.ReplaceAll(broken, "%3A", "/")    // 彻底去掉转义，得到一段超长非法 base64
	n := realCaseNode(broken)
	err := ValidateProxyNode(n)
	if err == nil {
		t.Fatal("不可还原的 2022 密钥必须被丢弃")
	}
	ve, ok := err.(*NodeValidationError)
	if !ok {
		t.Fatalf("应返回 *NodeValidationError，实际 %T", err)
	}
	if ve.Code != ReasonCipherKeyNotB64 && ve.Code != ReasonCipherKeyMismatch {
		t.Fatalf("原因码应为 %s 或 %s，实际 %s", ReasonCipherKeyNotB64, ReasonCipherKeyMismatch, ve.Code)
	}

	// ① -b 长度不符 → cipher-key-length-mismatch
	k16 := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", 16)))
	k32 := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("b", 32)))
	nLen := realCaseNode(k32 + ":" + k16)
	errLen := ValidateProxyNode(nLen)
	if errLen == nil {
		t.Fatal("两段中有一段长度不符，必须被丢弃")
	}
	if veLen, _ := errLen.(*NodeValidationError); veLen.Code != ReasonCipherKeyMismatch {
		t.Fatalf("原因码应为 %s，实际 %s", ReasonCipherKeyMismatch, veLen.Code)
	}

	// ① -c 非 base64 段 → cipher-key-not-base64
	nB64 := realCaseNode(k32 + ":not-base64!!")
	errB64 := ValidateProxyNode(nB64)
	if errB64 == nil {
		t.Fatal("非 base64 段必须被丢弃")
	}
	if veB64, _ := errB64.(*NodeValidationError); veB64.Code != ReasonCipherKeyNotB64 {
		t.Fatalf("原因码应为 %s，实际 %s", ReasonCipherKeyNotB64, veB64.Code)
	}

	// ① -d 合法单段 / 合法两段都必须通过（内核实测两种形态都接受）
	for _, pw := range []string{k32, k32 + ":" + k32} {
		if err := ValidateProxyNode(realCaseNode(pw)); err != nil {
			t.Fatalf("合法 2022 密钥（%d 字符）应通过: %v", len(pw), err)
		}
	}

	// ① -e 现网形态本身可被自动还原 → 保留而不是丢弃（详见验收 ②/③）
	nLive := realCaseNode(polluted)
	if code, _ := ssKeyCheck(nLive.Password, 32); code == "" {
		t.Fatal("夹具应能复现'原始形态不合法'这一特征")
	}
	if corrected, detail := NormalizeSS2022Key(nLive); !corrected {
		t.Fatalf("现网形态应被自动还原（%s）", detail)
	}
	if nLive.Password != decoded {
		t.Fatalf("还原结果应与解码形态逐字节一致:\n got %q\nwant %q", nLive.Password, decoded)
	}
	if err := ValidateProxyNode(nLive); err != nil {
		t.Fatalf("还原后应通过校验: %v", err)
	}
}

// 验收 ②：生成配置后真内核 -t 必须通过；③：日志里能看到该节点被处理及原因
func TestAcceptanceRealCase_GeneratedConfigPassesRealKernel(t *testing.T) {
	bin := kernelBinForAcceptance(t)
	polluted, _ := realCasePasswordFixture(t)
	svc, _ := setupSelfCheckTest(t, nil) // 不需要 hook：这里直接用真内核

	// 现网同款节点 + 若干正常节点
	good := mkNodes(6, nil)
	nodes := append([]*ProxyNode{realCaseNode(polluted)}, good...)

	// 走完整第一层（会就地修正该节点并记录事件）
	kept, events := StaticValidateNodes(nodes, "acceptance")

	var sawCorrect, sawDrop bool
	for _, e := range events {
		if e.NodeName != realCaseName {
			continue
		}
		if e.Event == models.NodeValidationCorrectedAtIngest {
			sawCorrect = true
			if !strings.Contains(e.Reason, ReasonKeyURLDecoded) {
				t.Errorf("修正事件 reason 应含 %s，实际 %q", ReasonKeyURLDecoded, e.Reason)
			}
		}
		if e.Event == models.NodeValidationDroppedAtIngest {
			sawDrop = true
		}
	}
	if !sawCorrect {
		t.Fatalf("应记录 %s 的修正事件（后台可见），实际 events=%+v", realCaseName, events)
	}
	if sawDrop {
		t.Fatalf("%s 应被修正保留而不是丢弃，实际被丢弃了", realCaseName)
	}
	if len(kept) != len(nodes) {
		t.Fatalf("修正后不应丢节点：期望 %d 个，实际 %d", len(nodes), len(kept))
	}

	// 生成配置并用真内核 -t
	yaml := svc.generateClashYAML(kept, &SubscriptionContext{Status: StatusNormal})
	if !strings.Contains(yaml, realCaseName) {
		t.Fatalf("生成的配置里应包含 %s", realCaseName)
	}
	if ok, errLine := runKernelOnYAML(t, bin, yaml); !ok {
		t.Fatalf("修复后的配置必须通过真内核 -t，实际失败: %s", errLine)
	}
	t.Logf("真内核验收通过：%d 个节点（含 %s）的配置 -t 成功", len(kept), realCaseName)
}

// 验收 ③（对照）：把现网原值不做校验直接塞进配置 → 真内核必须失败
// 证明这道防御是"承重"的：没有它，一个节点就让整份订阅失效。
func TestAcceptanceRealCase_ControlUnvalidatedConfigFailsRealKernel(t *testing.T) {
	bin := kernelBinForAcceptance(t)
	polluted, _ := realCasePasswordFixture(t)
	svc, _ := setupSelfCheckTest(t, nil)

	// 绕过第一层：直接把未修正的节点交给生成层
	unvalidated := append([]*ProxyNode{realCaseNode(polluted)}, mkNodes(6, nil)...)
	yaml := svc.generateClashYAML(unvalidated, &SubscriptionContext{Status: StatusNormal})

	ok, errLine := runKernelOnYAML(t, bin, yaml)
	if ok {
		t.Fatal("对照用例应当失败：未校验的现网同款节点必须让整份配置 -t 失败")
	}
	if !strings.Contains(errLine, "2022-blake3-aes-256-gcm") {
		t.Errorf("内核报错应指向该 cipher，实际: %s", errLine)
	}
	if !strings.Contains(errLine, "base64") && !strings.Contains(errLine, "bad key length") {
		t.Errorf("内核报错应是 base64/长度类错误，实际: %s", errLine)
	}
	t.Logf("对照组（未校验）确实整份失败: %s", strings.TrimSpace(errLine))
}

// 验收 ④：不加 base64 约束到非 2022 cipher 上（否则会误杀现网绝大多数正常节点）
func TestAcceptanceRealCase_Non2022CiphersNotBase64Constrained(t *testing.T) {
	// 现网真实存在的组合与口令形态（内核实测均 PASS）
	cases := []struct{ cipher, pw string }{
		{"aes-128-gcm", "pw"},
		{"chacha20-ietf-poly1305", "panel-pass-example!"},
		{"aes-256-cfb", "任意中文口令也合法"},
	}
	for _, c := range cases {
		n := ssNode("normal", c.cipher, c.pw)
		if err := ValidateProxyNode(n); err != nil {
			t.Fatalf("非 2022 cipher（%s + %q）内核实测通过，绝不能因'强制 base64'被误杀: %v", c.cipher, c.pw, err)
		}
	}
}

// 验收 ⑤：全库体检能定位并聚合这类节点（脱敏，不输出密钥）
func TestAcceptanceRealCase_AuditAggregatesInvalidCipherKeyNodes(t *testing.T) {
	bin := kernelBinForAcceptance(t)
	polluted, _ := realCasePasswordFixture(t)
	db := setupAuditTestDB(t)

	// 三个坏节点：
	//  a) 现网同款（可还原 → 应计入 corrected）
	//  b) 不可还原的非法密钥 → dropped / cipher-key-not-base64
	//  c) 长度不符 → dropped / cipher-key-length-mismatch
	k32 := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("b", 32)))
	k16 := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", 16)))
	broken := strings.ReplaceAll(strings.ReplaceAll(polluted, "%2F", "/"), "%3A", "/")
	seedAuditNode(t, db, 1, realCaseName, "ss", realCaseNode(polluted))
	seedAuditNode(t, db, 2, "坏密钥-不可还原", "ss", realCaseNode(broken))
	seedAuditNode(t, db, 3, "坏密钥-长度不符", "ss", realCaseNode(k32+":"+k16))
	seedAuditNode(t, db, 4, "正常节点", "ss", ssNode("正常节点", "aes-128-gcm", "pw"))

	report, err := AuditActiveNodes(db, bin)
	if err != nil {
		t.Fatalf("体检失败: %v", err)
	}

	if report.Total != 4 {
		t.Fatalf("活跃节点应为 4，实际 %d", report.Total)
	}
	if report.Dropped != 2 {
		t.Fatalf("应丢弃 2 个（不可还原 + 长度不符），实际 %d；明细=%+v", report.Dropped, report.Entries)
	}
	if report.Corrected != 1 {
		t.Fatalf("应修正 1 个（现网同款），实际 %d", report.Corrected)
	}

	// 聚合视图必须能回答"这类节点有多少、什么原因"
	byReason := map[string]int{}
	for _, r := range report.DroppedByReason {
		byReason[r.ReasonCode] = r.Count
	}
	if byReason[ReasonCipherKeyNotB64] != 1 || byReason[ReasonCipherKeyMismatch] != 1 {
		t.Fatalf("按原因聚合不符：%+v", report.DroppedByReason)
	}
	byTypeCipher := map[string]int{}
	for _, tc := range report.DroppedByTypeCipher {
		byTypeCipher[tc.Type+"|"+tc.Cipher] = tc.Count
	}
	if byTypeCipher["ss|"+realCaseCipher] != 2 {
		t.Fatalf("按 type+cipher 聚合应命中 ss|%s=2，实际 %+v", realCaseCipher, report.DroppedByTypeCipher)
	}
	// 聚合里绝不能出现密钥
	blob := ""
	for _, e := range report.Entries {
		blob += e.Reason + e.Name + e.Cipher + e.Server
	}
	for _, secret := range []string{polluted, k32, k16, broken} {
		if strings.Contains(blob, secret) {
			t.Fatal("体检结果泄露了密钥材料")
		}
	}
}

package handlers

import (
	"encoding/base64"
	"strings"
	"testing"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupImportValidateDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Node{}, &models.NodeValidationLog{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func ss2(name, cipher, pw string) *config_update.ProxyNode {
	return &config_update.ProxyNode{Name: name, Type: "ss", Server: "1.2.3.4", Port: 8388,
		Cipher: cipher, Password: pw, Options: map[string]any{}}
}

// 手工导入必须与采集链路共用同一套第一层判定：
// 坏节点不写库，并且给管理员可读原因（不再是"导入 N 个成功"然后节点永远不可用）。
func TestValidateAndNormalizeImportedNode(t *testing.T) {
	setupImportValidateDB(t)

	k32 := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("b", 32)))
	seg1 := "QUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBA/A="
	polluted := strings.ReplaceAll(seg1, "/", "%2F") + "%3A" + strings.ReplaceAll(seg1, "/", "%2F")

	cases := []struct {
		name     string
		node     *config_update.ProxyNode
		wantCode string
	}{
		{"naive 类型", &config_update.ProxyNode{Name: "n", Type: "naive", Server: "1.2.3.4", Port: 443, Password: "pw"}, "unsupported-type"},
		{"缺 port", &config_update.ProxyNode{Name: "n", Type: "ss", Server: "1.2.3.4", Cipher: "aes-128-gcm", Password: "pw"}, "invalid-port"},
		{"非法 cipher", ss2("n", "aes-128-ocb", "pw"), "cipher-unsupported"},
		{"2022 密钥非 base64", ss2("n", "2022-blake3-aes-256-gcm", "not-base64!!"), "cipher-key-not-base64"},
		{"合法 trojan", &config_update.ProxyNode{Name: "t", Type: "trojan", Server: "a.com", Port: 443, Password: "pw"}, ""},
		{"合法 ss", ss2("s", "aes-128-gcm", "pw"), ""},
		{"合法 2022", ss2("s2", "2022-blake3-aes-256-gcm", k32), ""},
	}
	for _, tc := range cases {
		code, detail := validateAndNormalizeImportedNode(tc.node, "test", false)
		if code != tc.wantCode {
			t.Errorf("%s: 期望原因码 %q，实际 %q (%s)", tc.name, tc.wantCode, code, detail)
		}
	}

	// URL 编码污染的 2022 密钥：手工导入也必须自动修正并保留（与采集链路一致）
	n := ss2("修好的", "2022-blake3-aes-256-gcm", polluted)
	code, _ := validateAndNormalizeImportedNode(n, "test", true)
	if code != "" {
		t.Fatalf("可修正的 URL 编码密钥不应被拒绝，实际原因码 %s", code)
	}
	if strings.Contains(n.Password, "%") {
		t.Fatalf("导入时未做凭据规范化，仍是污染形态: 长度=%d", len(n.Password))
	}
	if n.Password != seg1+":"+seg1 {
		t.Fatal("规范化结果与解码形态不一致")
	}
}

// 端到端：importNodeLinks 只写入通过校验的节点，拒绝项带可读原因，并落 node_validation_logs
func TestImportNodeLinksAppliesLayerOneGate(t *testing.T) {
	db := setupImportValidateDB(t)

	links := []string{
		// 合法
		"trojan://pw@a.example.com:443#合规节点",
		"ss://" + base64.StdEncoding.EncodeToString([]byte("aes-128-gcm:pw")) + "@b.example.com:8388#合规ss",
		// 非法：2022 密钥非 base64（手工导入最典型的错填）
		"ss://" + base64.StdEncoding.EncodeToString([]byte("2022-blake3-aes-256-gcm:not-base64!!")) + "@c.example.com:8388#坏密钥",
		// 非法：naive（内核不支持）
		"naive://user:pw@d.example.com:443#naive节点",
		// 非法：ss 缺端口
		"ss://" + base64.StdEncoding.EncodeToString([]byte("aes-128-gcm:pw")) + "@e.example.com#缺端口",
	}
	imp, skp, fail, failReasons, rejects := importNodeLinks(db, links)

	if imp != 2 {
		var got []string
		db.Model(&models.Node{}).Pluck("name", &got)
		t.Fatalf("应只导入 2 个合法节点，实际 %d，库里=%v（skp=%d fail=%d rejects=%+v）", imp, got, skp, fail, rejects)
	}
	var cnt int64
	db.Model(&models.Node{}).Count(&cnt)
	if cnt != 2 {
		t.Fatalf("库里应只有 2 个节点，实际 %d", cnt)
	}
	// 拒绝项必须有可读原因，且带上具体原因码
	if len(rejects) == 0 {
		t.Fatal("应给出拒绝明细")
	}
	blob := ""
	for _, r := range rejects {
		blob += r.Name + "|" + r.Type + "|" + r.Reason + "\n"
	}
	for _, want := range []string{"cipher-key-", "unsupported-type"} {
		if !strings.Contains(blob, want) {
			t.Errorf("拒绝原因里应包含 %q，实际:\n%s", want, blob)
		}
	}
	if len(failReasons) == 0 {
		t.Error("兼容字段 failReasons 也应有内容")
	}

	// 坏节点不得写库
	var badNames []string
	db.Model(&models.Node{}).Pluck("name", &badNames)
	for _, n := range badNames {
		if strings.Contains(n, "坏密钥") || strings.Contains(n, "naive") || strings.Contains(n, "缺端口") {
			t.Fatalf("非法节点被写进了库: %v", badNames)
		}
	}

	// 拒绝事件必须落库，后台「节点校验日志」可见
	var logs []models.NodeValidationLog
	db.Where("event = ?", models.NodeValidationDroppedAtIngest).Find(&logs)
	if len(logs) == 0 {
		t.Fatal("导入被拒的节点应写入 node_validation_logs（后台可见）")
	}
	for _, l := range logs {
		if strings.Contains(l.Reason, "pw@") || strings.Contains(l.Reason, "not-base64!!") {
			t.Fatalf("校验日志泄露了凭据: %s", l.Reason)
		}
	}
}

// 从 Clash 配置导入同样要过第一层
func TestProcessAndImportLinksAppliesLayerOneGate(t *testing.T) {
	db := setupImportValidateDB(t)
	links := []string{
		"trojan://pw@a.example.com:443#good",
		"naive://user:pw@d.example.com:443#naive节点",
	}
	got, rejects := processAndImportLinks(db, links)
	if got != 1 {
		t.Fatalf("应只导入 1 个，实际 %d", got)
	}
	if len(rejects) != 1 || rejects[0].Type != "naive" {
		t.Fatalf("应给出一条 naive 的拒绝原因，实际 %+v", rejects)
	}
}

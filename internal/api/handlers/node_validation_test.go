package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 节点校验日志接口是「只读」接口：这里同时验证
//  1. 能按 event 过滤并返回最近 N 条 + 汇总；
//  2. 返回体里绝不出现凭据（表结构本身不存 password/uuid）。
func TestGetNodeValidationLogsReadOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.NodeValidationLog{}); err != nil {
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

	rows := []models.NodeValidationLog{
		{Event: models.NodeValidationDroppedAtIngest, Source: "https://example.com/sub", NodeName: "坏节点A",
			NodeType: "naive", Server: "203.0.113.1", Port: 443, Reason: "unsupported-type: mihomo 不支持 naive 协议"},
		{Event: models.NodeValidationDroppedAtIngest, Source: "https://example.com/sub", NodeName: "坏节点B",
			NodeType: "ss", Server: "203.0.113.2", Port: 8388, Reason: "cipher-unsupported: ss cipher \"aes-128-ocb\""},
		{Event: models.NodeValidationPrunedAtGenerate, Source: "generate", NodeName: "坏节点C",
			NodeType: "ss", Server: "203.0.113.3", Port: 8388, Reason: "kernel-config-invalid: 内核判定该节点使配置失效"},
		{Event: models.NodeValidationCorrectedAtIngest, Source: "https://example.com/sub", NodeName: "修好的节点",
			NodeType: "ss", Server: "203.0.113.4", Port: 8388, Reason: "cipher-key-url-decoded: 已还原为合法 base64"},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("写入测试日志失败: %v", err)
	}

	call := func(query string) (int, map[string]interface{}) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/nodes/validation-logs"+query, nil)
		GetNodeValidationLogs(c)
		var body map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("响应不是合法 JSON: %v (%s)", err, w.Body.String())
		}
		return w.Code, body
	}

	// 全量
	code, body := call("")
	if code != http.StatusOK {
		t.Fatalf("状态码应为 200，实际 %d", code)
	}
	data, _ := body["data"].(map[string]interface{})
	if data == nil {
		t.Fatalf("响应缺少 data 字段: %s", body)
	}
	list, _ := data["list"].([]interface{})
	if len(list) != 4 {
		t.Fatalf("应返回 4 条，实际 %d", len(list))
	}
	summary, _ := data["summary"].(map[string]interface{})
	for _, ev := range []string{"dropped_at_ingest", "pruned_at_generate", "corrected_at_ingest"} {
		if summary[ev] == nil {
			t.Fatalf("汇总缺少事件类型 %s: %v", ev, summary)
		}
	}
	if summary["dropped_at_ingest"].(float64) != 2 || summary["pruned_at_generate"].(float64) != 1 {
		t.Fatalf("汇总数量不对: %v", summary)
	}

	// 按 event 过滤
	if _, body := call("?event=pruned_at_generate"); func() bool {
		d, _ := body["data"].(map[string]interface{})
		l, _ := d["list"].([]interface{})
		return len(l) != 1
	}() {
		t.Fatal("按 event=pruned_at_generate 过滤应只返回 1 条")
	}

	// 按关键字搜索
	if _, body := call("?search=naive"); func() bool {
		d, _ := body["data"].(map[string]interface{})
		l, _ := d["list"].([]interface{})
		return len(l) != 1
	}() {
		t.Fatal("search=naive 应只命中 1 条")
	}

	// limit 生效
	if _, body := call("?limit=2"); func() bool {
		d, _ := body["data"].(map[string]interface{})
		l, _ := d["list"].([]interface{})
		return len(l) != 2
	}() {
		t.Fatal("limit=2 应只返回 2 条")
	}

	// 返回体不得出现任何凭据字段
	raw, _ := json.Marshal(body)
	for _, forbidden := range []string{"password", "uuid", "private-key", "psk"} {
		if containsFold(string(raw), forbidden) {
			t.Fatalf("只读接口响应不应包含凭据字段 %q: %s", forbidden, raw)
		}
	}
}

func containsFold(haystack, needle string) bool {
	h, n := []rune(haystack), []rune(needle)
	if len(n) == 0 || len(n) > len(h) {
		return false
	}
	lower := func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + 32
		}
		return r
	}
	for i := 0; i+len(n) <= len(h); i++ {
		match := true
		for j := range n {
			if lower(h[i+j]) != lower(n[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

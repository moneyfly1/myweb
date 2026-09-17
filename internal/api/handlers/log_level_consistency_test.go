package handlers

import (
	"database/sql"
	"testing"

	"cboard-go/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 日志级别一致性测试。
//
// 背景（线上缺陷）：列表徽标用 getLogLevel（按 action_type 语义判定），
// 而"级别筛选"与"统计卡"用 response_status 阈值判定。两套规则导致自相矛盾：
//   - business_* + status=400：徽标"警告"，却出现在"错误"卡片/筛选结果里
//   - security_login_failed + status=200：徽标"错误"，却出现在"信息"里
//
// 本测试对同一批合成日志同时跑 Go 判定与 SQL 谓词，要求两者结论完全一致 ——
// 这就是"一套规则、两处表达"的回归闸门。
func setupAuditLogDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.AuditLog{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	return db
}

func nullStatus(v int64) sql.NullInt64   { return sql.NullInt64{Int64: v, Valid: true} }
func nullDesc(v string) sql.NullString   { return sql.NullString{String: v, Valid: true} }
func validStatus(v int64) *sql.NullInt64 { s := nullStatus(v); return &s }

// 覆盖各条规则分支的合成日志
func levelFixture() []models.AuditLog {
	return []models.AuditLog{
		// system_error 恒为错误，即使没有状态码
		{ActionType: "system_error", ActionDescription: nullDesc("任务失败")},
		{ActionType: "system_error", ResponseStatus: nullStatus(200), ActionDescription: nullDesc("任务失败")},

		// business_*：500+ 错误 / 400-499 警告 / 其余信息
		{ActionType: "business_subscription_pull", ResponseStatus: nullStatus(500), ActionDescription: nullDesc("拉取失败")},
		{ActionType: "business_subscription_pull", ResponseStatus: nullStatus(400), ActionDescription: nullDesc("参数错误")},
		{ActionType: "business_subscription_pull", ResponseStatus: nullStatus(200), ActionDescription: nullDesc("拉取成功")},

		// security_*：显式分类（与状态码无关）
		{ActionType: "security_login_failed", ResponseStatus: nullStatus(200), ActionDescription: nullDesc("登录失败")},
		{ActionType: "security_login_success", ResponseStatus: nullStatus(200), ActionDescription: nullDesc("登录成功")},
		{ActionType: "security_verify_code_rate_limit", ResponseStatus: nullStatus(200), ActionDescription: nullDesc("验证码限流")},
		{ActionType: "security_login_attempt", ResponseStatus: nullStatus(200), ActionDescription: nullDesc("登录尝试")},

		// security_* 未显式分类：看描述里的严重度标记
		{ActionType: "security_custom_event", ResponseStatus: nullStatus(200), ActionDescription: nullDesc("[CRITICAL] 严重事件")},
		{ActionType: "security_custom_event", ResponseStatus: nullStatus(200), ActionDescription: nullDesc("[MEDIUM] 中危事件")},
		// 未分类且无标记 → 落到通用状态码规则
		{ActionType: "security_custom_event", ResponseStatus: nullStatus(500), ActionDescription: nullDesc("无标记")},
		{ActionType: "security_custom_event", ResponseStatus: nullStatus(302), ActionDescription: nullDesc("无标记")},
		{ActionType: "security_custom_event", ResponseStatus: nullStatus(200), ActionDescription: nullDesc("无标记")},

		// login 恒为信息，即使状态码异常
		{ActionType: "login", ResponseStatus: nullStatus(500), ActionDescription: nullDesc("用户登录")},

		// 通用规则（非 business/security/system_error/login）
		{ActionType: "update_user", ResponseStatus: nullStatus(500), ActionDescription: nullDesc("更新失败")},
		{ActionType: "update_user", ResponseStatus: nullStatus(302), ActionDescription: nullDesc("重定向")},
		{ActionType: "update_user", ResponseStatus: nullStatus(200), ActionDescription: nullDesc("更新成功")},
		{ActionType: "update_user", ActionDescription: nullDesc("无状态码")},
		// scheduler_* 也走通用规则
		{ActionType: "scheduler_backup", ResponseStatus: nullStatus(500), ActionDescription: nullDesc("备份失败")},
		{ActionType: "scheduler_backup", ResponseStatus: nullStatus(200), ActionDescription: nullDesc("备份完成")},
	}
}

// TestLogLevelBadgeMatchesSQLFilter 徽标判定（Go）与筛选（SQL）必须逐条一致
func TestLogLevelBadgeMatchesSQLFilter(t *testing.T) {
	db := setupAuditLogDB(t)
	fixture := levelFixture()
	for i := range fixture {
		if err := db.Create(&fixture[i]).Error; err != nil {
			t.Fatalf("写入合成日志失败: %v", err)
		}
	}

	for _, level := range []string{"error", "warning", "info"} {
		// SQL 侧：用与筛选/统计完全相同的谓词
		var matchedIDs []uint
		if err := db.Model(&models.AuditLog{}).
			Where(logLevelWhere(level)).
			Order("id ASC").
			Pluck("id", &matchedIDs).Error; err != nil {
			t.Fatalf("按级别 %s 查询失败: %v", level, err)
		}
		sqlSet := make(map[uint]bool, len(matchedIDs))
		for _, id := range matchedIDs {
			sqlSet[id] = true
		}

		// Go 侧：徽标判定
		for i := range fixture {
			id := fixture[i].ID
			want := getLogLevel(fixture[i], false) == level
			if got := sqlSet[id]; got != want {
				t.Errorf("日志 %d (%s, status=%v, desc=%q)：SQL 判定 level=%s 为 %v，但徽标判定为 %v —— 两套规则不一致",
					id, fixture[i].ActionType, describeStatus(fixture[i].ResponseStatus),
					fixture[i].ActionDescription.String, level, got, want)
			}
		}
	}
}

// TestLogLevelSQLPartitionsAllRows 三个级别必须恰好覆盖全部日志（无遗漏、无重复计数）
func TestLogLevelSQLPartitionsAllRows(t *testing.T) {
	db := setupAuditLogDB(t)
	fixture := levelFixture()
	for i := range fixture {
		if err := db.Create(&fixture[i]).Error; err != nil {
			t.Fatalf("写入合成日志失败: %v", err)
		}
	}

	var total int64
	db.Model(&models.AuditLog{}).Count(&total)

	var sum int64
	for _, level := range []string{"error", "warning", "info"} {
		var n int64
		if err := db.Model(&models.AuditLog{}).Where(logLevelWhere(level)).Count(&n).Error; err != nil {
			t.Fatalf("统计 %s 失败: %v", level, err)
		}
		sum += n
	}
	if sum != total {
		t.Errorf("三级别人数合计 %d ≠ 日志总数 %d（说明存在落不进任何级别的日志，统计卡会漏计）", sum, total)
	}
}

// TestBusinessLogLevelBoundaries 业务日志边界：400 是警告、500 是错误
// （历史缺陷正是把 400 归进了"错误"）
func TestBusinessLogLevelBoundaries(t *testing.T) {
	cases := []struct {
		status int64
		want   string
	}{
		{200, "info"},
		{300, "info"}, // 业务日志 3xx 仍按信息（getLogLevel 只判 >=400/>=500）
		{399, "info"},
		{400, "warning"},
		{499, "warning"},
		{500, "error"},
		{503, "error"},
	}
	for _, c := range cases {
		log := models.AuditLog{ActionType: "business_x", ResponseStatus: nullStatus(c.status)}
		if got := getLogLevel(log, false); got != c.want {
			t.Errorf("business_* status=%d 徽标应为 %s，实际 %s", c.status, c.want, got)
		}
	}
}

// TestSecurityLoginFailedAlwaysError 安全类显式分类不受状态码影响
func TestSecurityLoginFailedAlwaysError(t *testing.T) {
	for _, status := range []int64{200, 401, 500} {
		log := models.AuditLog{ActionType: "security_login_failed", ResponseStatus: nullStatus(status)}
		if got := getLogLevel(log, false); got != "error" {
			t.Errorf("security_login_failed status=%d 应为 error，实际 %s", status, got)
		}
	}
	log := models.AuditLog{ActionType: "security_login_success", ResponseStatus: nullStatus(500)}
	if got := getLogLevel(log, false); got != "info" {
		t.Errorf("security_login_success 应为 info（显式分类优先），实际 %s", got)
	}
	log = models.AuditLog{ActionType: "login", ResponseStatus: nullStatus(500)}
	if got := getLogLevel(log, false); got != "info" {
		t.Errorf("login 应为 info，实际 %s", got)
	}
}

// TestLogLevelWhereEmptyForUnknown 未知级别不产生谓词（避免误筛选）
func TestLogLevelWhereEmptyForUnknown(t *testing.T) {
	if got := logLevelWhere("critical"); got != "" {
		t.Errorf("未知级别应返回空谓词，实际 %q", got)
	}
	if got := logLevelWhere(""); got != "" {
		t.Errorf("空级别应返回空谓词，实际 %q", got)
	}
}

func describeStatus(s sql.NullInt64) interface{} {
	if !s.Valid {
		return "NULL"
	}
	return s.Int64
}

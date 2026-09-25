package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestCheckinWritesAuditLogAfterCommit 是「签到固定卡 5 秒 + 审计记录 100% 丢失」的回归测试。
//
// 背景：Checkin 曾在 WithTransaction 内调用 utils.CreateBusinessLogFast，而后者用
// database.GetDB()（连接池里的另一条连接）执行 INSERT。此时外层事务已执行
// UPDATE users SET balance，SQLite 写锁尚未释放，第二条连接只能等满
// PRAGMA busy_timeout=5000 才失败并报 "database is locked"。
// 线上表现：268/268 次成功签到延迟 5005~5027ms，并伴随 268 条
// "[业务日志保存失败] user_checkin ... database is locked"。
//
// 本测试断言两件事：耗时 < 1s、审计行确实落库。旧实现两条都会失败。
func TestCheckinWritesAuditLogAfterCommit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:checkin_audit?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.CheckinRecord{}, &models.BalanceLog{}, &models.AuditLog{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	// 与生产一致的锁等待参数（internal/core/database/database.go:126）。
	if err := db.Exec("PRAGMA busy_timeout=5000").Error; err != nil {
		t.Fatalf("设置 busy_timeout 失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取 sql.DB 失败: %v", err)
	}
	// 生产 MaxOpenConns=10：必须允许多连接，旧实现的第二条连接才会真实撞锁。
	sqlDB.SetMaxOpenConns(10)

	prevDB := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = prevDB
		sqlDB.Close()
	})

	user := models.User{Username: "checkin_tester", Email: "checkin@example.com", Password: "x", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/users/checkin", nil)
	c.Set("user_id", user.ID)

	start := time.Now()
	Checkin(c)
	elapsed := time.Since(start)

	if rec.Code != http.StatusOK {
		t.Fatalf("签到应返回 200，实际 %d，body=%s", rec.Code, rec.Body.String())
	}
	if elapsed >= time.Second {
		t.Fatalf("签到耗时 %v，疑似审计日志仍在事务内写入（旧实现会等满 busy_timeout=5s 后失败）", elapsed)
	}
	var n int64
	if err := db.Model(&models.AuditLog{}).Where("action_type = ?", "business_user_checkin").Count(&n).Error; err != nil {
		t.Fatalf("查询审计日志失败: %v", err)
	}
	if n != 1 {
		t.Fatalf("签到审计日志应落库 1 条，实际 %d 条（旧实现 100%% 丢失）", n)
	}
	var cnt int64
	db.Model(&models.CheckinRecord{}).Count(&cnt)
	if cnt != 1 {
		t.Fatalf("签到记录应为 1 条，实际 %d", cnt)
	}
}

package handlers

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 「多地区拉取订阅」详情（抽屉里那一项）
//
// 用户反馈：这一类异常点开详情后信息不完整（没有明细表）。本测试锁住详情结构：
// 标题/级别/时间口径/明细列齐全，并能按 IP 反查设备补出「用什么客户端拉的」。
func TestSubscriptionMultiLocationDetail(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Device{}, &models.SubscriptionLog{}); err != nil {
		t.Fatalf("建表失败: %v", err)
	}
	prev := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = prev
		if sqlDB, e := db.DB(); e == nil {
			sqlDB.Close()
		}
	})

	now := utils.GetBeijingTime()
	ns := func(v string) sql.NullString { return sql.NullString{String: v, Valid: true} }
	loc := func(city string) string {
		return `{"country":"中国","country_code":"CN","city":"` + city + `","region":"广东","latitude":23.1,"longitude":113.2}`
	}

	// 两条拉取记录：广州（有设备）、湘潭（没有登记设备，模拟第三方客户端直连）
	for _, c := range []struct{ ip, city string }{{"1.1.1.1", "广州"}, {"2.2.2.2", "湘潭"}} {
		if err := db.Create(&models.SubscriptionLog{
			SubscriptionID: 7, UserID: 42, ActionType: "access",
			IPAddress: ns(c.ip), Location: ns(loc(c.city)), CreatedAt: now.Add(-5 * time.Minute),
		}).Error; err != nil {
			t.Fatalf("插入拉取日志失败: %v", err)
		}
	}

	uid := int64(42)
	if err := db.Create(&models.Device{
		UserID: &uid, SubscriptionID: 7, DeviceFingerprint: "fp-1",
		IPAddress:    func() *string { s := "1.1.1.1"; return &s }(),
		SoftwareName: func() *string { s := "MoneyFly"; return &s }(),
		SoftwareVersion: func() *string {
			s := "2.2.31"
			return &s
		}(),
		OSName:     func() *string { s := "iOS"; return &s }(),
		IsActive:   true,
		LastAccess: now.Add(-3 * time.Minute),
	}).Error; err != nil {
		t.Fatalf("插入设备失败: %v", err)
	}

	detail := buildSubscriptionMultiLocationDetail(db, 42, "tester", 2, 2, "广州/湘潭")

	if got := detail["type"]; got != "subscription_multi_location" {
		t.Errorf("type 应为 subscription_multi_location，实际 %v", got)
	}
	if got := detail["title"]; got != "多地区拉取订阅" {
		t.Errorf("title 应为「多地区拉取订阅」，实际 %v", got)
	}
	if got := detail["level"]; got != "danger" {
		t.Errorf("level 应为 danger，实际 %v", got)
	}
	if got := detail["period"]; got != "最近 24 小时" {
		t.Errorf("period 应为「最近 24 小时」，实际 %v", got)
	}
	if summary, _ := detail["summary"].(string); !strings.Contains(summary, "2 个不同 IP") ||
		!strings.Contains(summary, "广州/湘潭") {
		t.Errorf("summary 应包含 IP 数与城市名，实际 %q", summary)
	}
	if advice, _ := detail["advice"].(string); advice == "" {
		t.Error("应给出处置建议（重置订阅/限制设备）")
	}

	columns, _ := detail["columns"].([]gin.H)
	if len(columns) < 5 {
		t.Errorf("明细列过少（%d 列），抽屉里信息不完整", len(columns))
	}
	items, _ := detail["items"].([]gin.H)
	if len(items) != 2 {
		t.Fatalf("应有 2 条明细，实际 %d", len(items))
	}
	// 广州那条要能关联到设备并显示客户端；湘潭那条要如实标「未登记设备」
	var sawClient, sawUnknown bool
	for _, it := range items {
		switch {
		case strings.Contains(it["client"].(string), "MoneyFly"):
			sawClient = true
			if it["location"].(string) == "" {
				t.Error("明细行应有地点")
			}
		case strings.Contains(it["client"].(string), "未登记设备"):
			sawUnknown = true
		}
	}
	if !sawClient {
		t.Error("应能按 IP 关联出设备客户端（MoneyFly 2.2.31）")
	}
	if !sawUnknown {
		t.Error("没有设备记录的 IP 应标为「未登记设备」，不能留空")
	}
}

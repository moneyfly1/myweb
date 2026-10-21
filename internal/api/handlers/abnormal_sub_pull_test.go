package handlers

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 「多地区频繁订阅」检测（2026-10-21 需求）
//
// 目的：把「订阅地址分享给别人、多人多地共用」识别成异常。
// 判据：最近 24 小时内用多少个不同 IP / 城市拉取过订阅（subscription_logs.access）。
// 本测试锁住：IP/城市去重正确、非 JSON 的 location（"本地"）不会让查询报错、
// 城市名拼接可用。
func TestSubscriptionPullAbuseMetrics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.SubscriptionLog{}); err != nil {
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
	loc := func(city string) string {
		return fmt.Sprintf(`{"country":"中国","country_code":"CN","city":"%s","region":"广东","latitude":23.1,"longitude":113.2}`, city)
	}
	ns := func(v string) sql.NullString { return sql.NullString{String: v, Valid: true} }
	insert := func(userID uint, ip, location string, at time.Time) {
		row := models.SubscriptionLog{
			SubscriptionID: 1,
			UserID:         userID,
			ActionType:     "access",
			IPAddress:      ns(ip),
			Location:       ns(location),
			CreatedAt:      at,
		}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("插入拉取日志失败: %v", err)
		}
	}

	// 用户 1：24 小时内 9 个不同 IP、3 个城市（广州/济南/福州），另有一条「本地」
	for i, city := range []string{"广州", "济南", "福州", "广州", "济南", "福州", "广州", "济南", "福州"} {
		insert(1, fmt.Sprintf("1.1.1.%d", i), loc(city), now.Add(-10*time.Minute))
	}
	insert(1, "9.9.9.9", "本地", now.Add(-5*time.Minute))

	// 用户 2：只有 1 个 IP、1 个城市（正常用户，不应该被判异常）
	insert(2, "2.2.2.2", loc("北京"), now.Add(-2*time.Minute))

	// 用户 3：24 小时窗口之外的旧记录（不该被计入）
	insert(3, "3.3.3.3", loc("上海"), now.Add(-30*time.Hour))

	metrics := abnormalUserRiskMetrics{
		subPullIPCounts:   make(map[uint]int64),
		subPullCityCounts: make(map[uint]int64),
		subPullCityNames:  make(map[uint]string),
	}
	loadSubscriptionPullAbuseMetrics(db, nil, metrics)

	if got := metrics.subPullIPCounts[1]; got != 10 {
		t.Errorf("用户 1 应有 10 个不同 IP（含「本地」那条），实际 %d", got)
	}
	if got := metrics.subPullCityCounts[1]; got != 3 {
		t.Errorf("用户 1 应有 3 个城市（非 JSON 的「本地」不计），实际 %d", got)
	}
	if names := metrics.subPullCityNames[1]; names == "" {
		t.Error("用户 1 应能拼出城市名（广州/济南/福州）")
	} else {
		t.Logf("城市名拼接: %s", names)
	}

	if got := metrics.subPullIPCounts[2]; got != 1 {
		t.Errorf("用户 2 应只有 1 个 IP，实际 %d", got)
	}

	if _, ok := metrics.subPullIPCounts[3]; ok {
		t.Error("24 小时之外的拉取不应计入")
	}

	// 阈值判定：用户 1 应命中（IP ≥ 8），用户 2 不应命中
	if metrics.subPullIPCounts[1] < abnormalSubPullIPThreshold {
		t.Errorf("用户 1 应达到 IP 阈值 %d", abnormalSubPullIPThreshold)
	}
	if metrics.subPullIPCounts[2] >= abnormalSubPullIPThreshold {
		t.Error("用户 2 不应达到 IP 阈值")
	}
}

// 阈值必须能覆盖「一个用户多设备」的误伤面：24 小时窗口 + 8 个 IP 的门槛，
// 生产数据里只命中约 8% 有拉取记录的用户。
func TestSubscriptionPullThresholdsAreSane(t *testing.T) {
	if abnormalSubPullIPThreshold < 5 {
		t.Fatalf("IP 阈值 %d 太低，正常用户换网络就会误报", abnormalSubPullIPThreshold)
	}
	if abnormalSubPullCityThreshold < 3 {
		t.Fatalf("城市阈值 %d 太低，正常出差就会误报", abnormalSubPullCityThreshold)
	}
	if abnormalSubPullWindow != 24*time.Hour {
		t.Fatalf("检测窗口应为 24 小时，实际 %s", abnormalSubPullWindow)
	}
}

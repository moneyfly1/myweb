package config_update

import (
	"testing"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/device"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 设备数量限制的判定必须**按稳定设备 ID**（自研客户端上报的 X-MF-Device-Id）：
//
// 背景（用户实测反馈）：限制 7 台时，自研客户端各台设备的 UA 完全相同
// （如都是 "MoneyFly/2.2.24 (Windows NT 10.0; Win64; x64)"），而配置生成侧
// 原先用不带请求头的 FindExistingDevice（只比 UA）判断"这台设备是否已在名单里"，
// 于是第 8 台也会被当成已存在的老设备放行 —— 同一套限制在自研客户端上失效、
// 在第三方客户端（UA 各不相同）上却生效，行为不一致。
//
// 本用例锁住修复后的语义：谁超限谁被拦。
func TestDeviceLimitUsesStableDeviceID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Subscription{}, &models.Node{},
		&models.SystemConfig{}, &models.Device{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})

	now := time.Now()
	user := models.User{Username: "dl", Email: "dl@example.com", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("建用户失败: %v", err)
	}
	// 限制 1 台，便于构造"已满"场景
	sub := models.Subscription{UserID: user.ID, SubscriptionURL: "tok-devlimit",
		Status: "active", IsActive: true, DeviceLimit: 1, ExpireTime: now.Add(24 * time.Hour)}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("建订阅失败: %v", err)
	}

	mfUA := "MoneyFly/2.2.24 (Windows NT 10.0; Win64; x64)"
	hdrsA := map[string]string{"X-MF-Device-Id": "dev-A", "X-MF-Device-Model": "PC", "X-MF-OS": "Windows 10"}
	hdrsB := map[string]string{"X-MF-Device-Id": "dev-B", "X-MF-Device-Model": "PC", "X-MF-OS": "Windows 10"}

	// DeviceManager 走全局连接池，测试里把它指向内存库（结束后还原）
	prevDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prevDB })

	// 先记录设备 A（模拟它已经拉过订阅、占用了唯一名额）
	dm := device.NewDeviceManager()
	if _, err := dm.RecordDeviceAccessWithHeaders(sub.ID, user.ID, mfUA, "10.0.0.1", "clash", hdrsA); err != nil {
		t.Fatalf("记录设备 A 失败: %v", err)
	}
	cnt, _ := device.CountActiveDevices(db, sub.ID)
	if cnt != 1 {
		t.Fatalf("设备 A 应占 1 个名额，实际 %d", cnt)
	}

	svc := &ConfigUpdateService{db: db}

	// ① 已注册的设备 A 再来拉 → 名额已满但它是名单内的设备 → 放行
	if ctx := svc.GetSubscriptionContext(sub.SubscriptionURL, "10.0.0.1", mfUA, hdrsA); ctx.Status != StatusNormal {
		t.Fatalf("已注册设备应放行，实际状态 %v", ctx.Status)
	}

	// ② 同 UA、同机型、但设备 ID 不同的新设备 B → 必须判为超限
	//    （修复前只看 UA，这里会被误放行）
	ctxB := svc.GetSubscriptionContext(sub.SubscriptionURL, "10.0.0.2", mfUA, hdrsB)
	if ctxB.Status != StatusDeviceOverLimit {
		t.Fatalf("新设备在名额已满时应判超限，实际状态 %v", ctxB.Status)
	}

	// ③ 第三方客户端（没有 X-MF-Device-Id）：不同 UA 视为新设备 → 超限
	if ctx := svc.GetSubscriptionContext(sub.SubscriptionURL, "10.0.0.3", "ClashforWindows/0.20.39"); ctx.Status != StatusDeviceOverLimit {
		t.Fatalf("第三方新设备应判超限，实际状态 %v", ctx.Status)
	}
}

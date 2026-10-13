package config_update

import (
	"strings"
	"testing"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/device"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupLimitTestDB(t *testing.T, limitMode string) (*gorm.DB, models.Subscription, models.User) {
	t.Helper()
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
	if limitMode != "" {
		if err := db.Create(&models.SystemConfig{Key: "limit_mode", Value: limitMode,
			Category: "device", Type: "string", DisplayName: "设备超限处理"}).Error; err != nil {
			t.Fatalf("写入配置失败: %v", err)
		}
	}

	now := time.Now()
	user := models.User{Username: "lu", Email: "lu@example.com", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("建用户失败: %v", err)
	}
	sub := models.Subscription{UserID: user.ID, SubscriptionURL: "tok-limit",
		Status: "active", IsActive: true, DeviceLimit: 1, ExpireTime: now.Add(24 * time.Hour)}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("建订阅失败: %v", err)
	}
	node := ProxyNode{Name: "可用节点A", Type: "ss", Server: "1.1.1.1", Port: 8388,
		Cipher: "aes-128-gcm", Password: "pw", Options: map[string]any{}}
	cfg := string(mustJSON(node))
	if err := db.Create(&models.Node{Name: node.Name, Type: "ss", Region: "test", Status: "online",
		IsActive: true, Config: &cfg}).Error; err != nil {
		t.Fatalf("建节点失败: %v", err)
	}
	return db, sub, user
}

func namesOf(nodes []*ProxyNode) []string {
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, n.Name)
	}
	return out
}

// 软限制（默认）：设备数超限仍然给**完整可用节点**，只在最前面加一条提示节点 ——
// 用户反馈"设 1 台设备后换台设备就完全连不上、只提示升级设备"不合理，
// 应当是"仍然能连，只是有提示"。
func TestDeviceOverLimitSoftModeKeepsNodesUsable(t *testing.T) {
	db, sub, user := setupLimitTestDB(t, "") // 不写配置 → 默认软限制
	prevDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prevDB })

	mfUA := "MoneyFly/2.2.24 (Windows NT 10.0; Win64; x64)"
	dm := device.NewDeviceManager()
	if _, err := dm.RecordDeviceAccessWithHeaders(sub.ID, user.ID, mfUA, "10.0.0.1", "clash",
		map[string]string{"X-MF-Device-Id": "dev-A"}); err != nil {
		t.Fatalf("记录设备失败: %v", err)
	}

	svc := &ConfigUpdateService{db: db}
	// 第二台设备（不同 ID）→ 状态为超限，但节点必须仍可用
	ctx := svc.GetSubscriptionContext(sub.SubscriptionURL, "10.0.0.2", mfUA,
		map[string]string{"X-MF-Device-Id": "dev-B"})
	if ctx.Status != StatusDeviceOverLimit {
		t.Fatalf("第二台设备应判超限，实际 %v", ctx.Status)
	}
	nodes := svc.buildSubscriptionNodes(ctx)
	names := namesOf(nodes)

	hasReal := false
	for _, n := range names {
		if n == "可用节点A" {
			hasReal = true
		}
	}
	if !hasReal {
		t.Fatalf("软限制下必须仍返回真实节点，实际节点: %v", names)
	}
	hasNotice := false
	for _, n := range names {
		if strings.Contains(n, "设备数已达上限") {
			hasNotice = true
		}
	}
	if !hasNotice {
		t.Fatalf("软限制下应带一条超限提示节点，实际节点: %v", names)
	}
	if len(names) == 0 || !strings.Contains(names[0], "设备数已达上限") {
		t.Fatalf("提示节点应排在最前面，实际: %v", names)
	}
}

// 硬限制（配置 device/limit_mode=hard）：保留旧行为——只给错误节点，连不上
func TestDeviceOverLimitHardModeCanBeRestored(t *testing.T) {
	db, sub, user := setupLimitTestDB(t, "hard")
	prevDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prevDB })

	mfUA := "MoneyFly/2.2.24 (Windows NT 10.0; Win64; x64)"
	dm := device.NewDeviceManager()
	if _, err := dm.RecordDeviceAccessWithHeaders(sub.ID, user.ID, mfUA, "10.0.0.1", "clash",
		map[string]string{"X-MF-Device-Id": "dev-A"}); err != nil {
		t.Fatalf("记录设备失败: %v", err)
	}

	svc := &ConfigUpdateService{db: db}
	svc.refreshSystemConfig() // 读入 hard
	if svc.isDeviceLimitSoft() {
		t.Fatalf("配置为 hard 时不应走软限制")
	}
	ctx := svc.GetSubscriptionContext(sub.SubscriptionURL, "10.0.0.2", mfUA,
		map[string]string{"X-MF-Device-Id": "dev-B"})
	if ctx.Status != StatusDeviceOverLimit {
		t.Fatalf("第二台设备应判超限，实际 %v", ctx.Status)
	}
	for _, n := range namesOf(svc.buildSubscriptionNodes(ctx)) {
		if n == "可用节点A" {
			t.Fatalf("硬限制下不应返回真实节点，实际: %v", namesOf(svc.buildSubscriptionNodes(ctx)))
		}
	}
}

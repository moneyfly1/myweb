package device

import (
	"testing"
	"time"

	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 「删除设备 = 踢下线」后的自助恢复（2026-09-21 生产问题回归）。
//
// 事故：用户套餐到期后把设备管理里的设备全删了（包括正在用的本机），随后续费
// 一年，但软件始终「无法获取订阅节点」、设备列表里也不再出现这台设备。
//
// 根因：删除设备 = is_active=false + kicked_at=now，订阅接口对命中该软删行的
// 设备一律 403「此设备已被移除并踢下线,如需继续使用请重新登录或联系客服」；
// 而**代码里没有任何地方会清除 kicked_at**，所谓「重新登录」实际无效 ——
// 用户被永久锁死，只能找客服改库。
//
// 这里锁定修复后的行为：确实被踢过的本机，在名额允许时可以恢复；
// 名额已满则不恢复（引导先删闲置设备）；没被踢过的不动。
const testUA = "MoneyFly/2.2.15 (iOS 16.6.1; iPhone14,3)"

func setupReviveTestDB(t *testing.T) (*gorm.DB, *DeviceManager, uint) {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Device{}, &models.Subscription{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})

	sub := models.Subscription{DeviceLimit: 3, CurrentDevices: 0}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("建订阅失败: %v", err)
	}
	return db, &DeviceManager{db: db}, sub.ID
}

// seedKickedDevice 造一条「被踢下线」的设备行（与删除接口写入的字段一致）
func seedKickedDevice(t *testing.T, db *gorm.DB, dm *DeviceManager, subID uint, remark string) models.Device {
	t.Helper()
	hash := dm.GenerateDeviceHash(testUA, "1.2.3.4", "")
	ua := testUA
	kickedAt := utils.GetBeijingTime()
	ip := "1.2.3.4"
	d := models.Device{
		SubscriptionID:    subID,
		DeviceFingerprint: hash[:32],
		DeviceHash:        &hash,
		UserAgent:         &ua,
		IPAddress:         &ip,
		IsActive:          true,
		IsAllowed:         true,
		LastAccess:        kickedAt,
		Remark:            &remark,
	}
	if err := db.Create(&d).Error; err != nil {
		t.Fatalf("造设备失败: %v", err)
	}
	// 与删除接口完全一致地写入「踢下线」（它用的是 Updates：is_active 有 default 标签，
	// 直接 Create 零值会被 GORM 忽略成默认 true，写不出被踢状态）
	if err := db.Model(&models.Device{}).Where("id = ?", d.ID).
		Updates(map[string]interface{}{
			"is_active":   false,
			"kicked_at":   kickedAt,
			"last_access": kickedAt,
		}).Error; err != nil {
		t.Fatalf("写入踢下线状态失败: %v", err)
	}
	return d
}

func TestReviveKickedDevice(t *testing.T) {
	db, dm, subID := setupReviveTestDB(t)
	kicked := seedKickedDevice(t, db, dm, subID, "我的手机")

	// 前提：被踢状态下 FindKickedDevice 命中（订阅接口据此 403）
	hit, err := dm.FindKickedDevice(subID, testUA, "1.2.3.4")
	if err != nil || hit == nil {
		t.Fatalf("前置条件不成立：被踢设备应能命中，err=%v", err)
	}

	revived, reason, err := dm.ReviveKickedDevice(subID, 3, false, testUA, "1.2.3.4")
	if err != nil {
		t.Fatalf("恢复失败: %v", err)
	}
	if !revived || reason != "" {
		t.Fatalf("应当恢复成功，revived=%v reason=%s", revived, reason)
	}

	// 恢复后：行仍是原来那一行（备注保留、不新建），且不再被判定为被踢
	var after models.Device
	if err := db.First(&after, kicked.ID).Error; err != nil {
		t.Fatalf("恢复后行丢失: %v", err)
	}
	if !after.IsActive || after.KickedAt != nil {
		t.Fatalf("恢复后应 is_active=true 且 kicked_at 为空，得到 active=%v kicked=%v",
			after.IsActive, after.KickedAt)
	}
	if after.Remark == nil || *after.Remark != "我的手机" {
		t.Fatal("恢复后应保留用户备注（同一行，不重建）")
	}
	var count int64
	db.Model(&models.Device{}).Where("subscription_id = ?", subID).Count(&count)
	if count != 1 {
		t.Fatalf("不应新建设备行，期望 1 行，实际 %d", count)
	}

	// 关键：恢复后订阅接口不再把它当作"已被移除"
	stillKicked, err := dm.FindKickedDevice(subID, testUA, "1.2.3.4")
	if err != nil {
		t.Fatalf("复查失败: %v", err)
	}
	if stillKicked != nil {
		t.Fatal("恢复后不应再命中被踢判定（否则用户依旧拿不到订阅）")
	}

	// 订阅的设备数要同步
	var sub models.Subscription
	if err := db.First(&sub, subID).Error; err != nil {
		t.Fatalf("读订阅失败: %v", err)
	}
	if sub.CurrentDevices != 1 {
		t.Fatalf("current_devices 应更新为 1，实际 %d", sub.CurrentDevices)
	}
}

func TestReviveKickedDevice_NotKicked(t *testing.T) {
	_, dm, subID := setupReviveTestDB(t)

	revived, reason, err := dm.ReviveKickedDevice(subID, 3, false, testUA, "1.2.3.4")
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if revived || reason != "not-kicked" {
		t.Fatalf("未被踢的设备不应被恢复：revived=%v reason=%s", revived, reason)
	}
}

func TestReviveKickedDevice_DeviceLimitFull(t *testing.T) {
	db, dm, subID := setupReviveTestDB(t)
	seedKickedDevice(t, db, dm, subID, "")

	// 名额已用满（3/3）：此时不恢复 —— 这正是"设备超限"场景，
	// 应引导用户先删闲置设备，而不是无限放行
	for i := 0; i < 3; i++ {
		hash := dm.GenerateDeviceHash("Other/1.0 (x)", "9.9.9.9", "")
		ua := "Other/1.0 (x)"
		if err := db.Create(&models.Device{
			SubscriptionID:    subID,
			DeviceFingerprint: hash[:32],
			DeviceHash:        &hash,
			UserAgent:         &ua,
			IsActive:          true,
			IsAllowed:         true,
			LastAccess:        time.Now(),
		}).Error; err != nil {
			t.Fatalf("造活跃设备失败: %v", err)
		}
	}

	revived, reason, err := dm.ReviveKickedDevice(subID, 3, false, testUA, "1.2.3.4")
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if revived || reason != "device-limit" {
		t.Fatalf("名额已满不应恢复，revived=%v reason=%s", revived, reason)
	}
	// 仍然处于被踢状态（未被静默复活）
	hit, _ := dm.FindKickedDevice(subID, testUA, "1.2.3.4")
	if hit == nil {
		t.Fatal("名额已满时被踢状态必须保持")
	}
}

func TestReviveKickedDevice_UnlimitedUser(t *testing.T) {
	db, dm, subID := setupReviveTestDB(t)
	seedKickedDevice(t, db, dm, subID, "")
	for i := 0; i < 5; i++ {
		hash := dm.GenerateDeviceHash("Other/1.0 (x)", "9.9.9.9", "")
		ua := "Other/1.0 (x)"
		if err := db.Create(&models.Device{
			SubscriptionID:    subID,
			DeviceFingerprint: hash[:32],
			DeviceHash:        &hash,
			UserAgent:         &ua,
			IsActive:          true,
			IsAllowed:         true,
			LastAccess:        time.Now(),
		}).Error; err != nil {
			t.Fatalf("造活跃设备失败: %v", err)
		}
	}

	// 特殊用户不受设备数限制：即使已超限也应恢复
	revived, reason, err := dm.ReviveKickedDevice(subID, 3, true, testUA, "1.2.3.4")
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if !revived || reason != "" {
		t.Fatalf("不限设备数的用户应能恢复，revived=%v reason=%s", revived, reason)
	}
}

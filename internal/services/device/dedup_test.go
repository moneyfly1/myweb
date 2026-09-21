package device

import (
	"testing"
	"time"

	"cboard-go/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 「升级 App/系统就多出一台幽灵设备」的存量清理回归（2026-09-21）。
//
// 旧哈希算法把 App 版本号/系统版本号也算进设备身份，于是同一台手机每次升级都会
// 以「新设备」重新登记一行，旧行永远占着名额（用户看到设备数虚高、甚至被判定
// 超限），也让「删除设备 = 踢下线」的判定错位。
//
// 修复后：身份改用稳定特征（软件名+系统名+机型+品牌，或客户端上报的
// X-MF-Device-Id），并按稳定指纹把历史重复行合并掉。
func setupDedupDB(t *testing.T) (*gorm.DB, uint) {
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
	sub := models.Subscription{DeviceLimit: 5, CurrentDevices: 0}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("建订阅失败: %v", err)
	}
	return db, sub.ID
}

// fingerprintFrom 由测试用的短哈希造一个长度合法的 fingerprint
func fingerprintFrom(hash string) string {
	for len(hash) < 32 {
		hash += "0"
	}
	return hash[:32]
}

func mkDevice(t *testing.T, db *gorm.DB, subID uint, hash, sw, osName, model, brand, remark string,
	active bool, lastAccess time.Time) models.Device {
	t.Helper()
	d := models.Device{
		SubscriptionID:    subID,
		DeviceFingerprint: fingerprintFrom(hash),
		DeviceHash:        &hash,
		SoftwareName:      &sw,
		OSName:            &osName,
		DeviceModel:       &model,
		DeviceBrand:       &brand,
		Remark:            &remark,
		IsActive:          true,
		IsAllowed:         true,
		LastAccess:        lastAccess,
		AccessCount:       3,
	}
	if err := db.Create(&d).Error; err != nil {
		t.Fatalf("建设备失败: %v", err)
	}
	if !active {
		kick := lastAccess
		if err := db.Model(&models.Device{}).Where("id = ?", d.ID).
			Updates(map[string]interface{}{"is_active": false, "kicked_at": kick}).Error; err != nil {
			t.Fatalf("设置踢下线失败: %v", err)
		}
	}
	return d
}

func TestMergeDuplicateDevices(t *testing.T) {
	db, subID := setupDedupDB(t)
	now := time.Now()

	// 同一台 iPhone：升级 2.2.14 → 2.2.15 各登记了一行（旧算法身份含版本号）
	old := mkDevice(t, db, subID, "hashOld", "MoneyFly", "iOS", "iPhone14,3", "Apple", "", true, now.Add(-48*time.Hour))
	mkDevice(t, db, subID, "hashNew", "MoneyFly", "iOS", "iPhone14,3", "Apple", "我的主力机", true, now)

	// 另一台确实不同的设备：不能被误合并
	other := mkDevice(t, db, subID, "hashOther", "MoneyFly", "Android", "V2072A", "vivo", "", true, now)

	db.Model(&models.Subscription{}).Where("id = ?", subID).Update("current_devices", 3)

	report, err := MergeDuplicateDevices(db, true)
	if err != nil {
		t.Fatalf("合并失败: %v", err)
	}
	if report.GroupsMerged != 1 || report.RowsRemoved != 1 {
		t.Fatalf("应合并 1 组、删除 1 行，实际 group=%d removed=%d (%v)",
			report.GroupsMerged, report.RowsRemoved, report.Details)
	}

	var rows []models.Device
	db.Where("subscription_id = ?", subID).Find(&rows)
	if len(rows) != 2 {
		t.Fatalf("合并后应剩 2 台设备（iPhone + vivo），实际 %d", len(rows))
	}
	// 保留的是最近使用的那行，且备注/访问次数被合并
	var kept *models.Device
	for i := range rows {
		if *rows[i].DeviceHash == "hashNew" {
			kept = &rows[i]
		}
		if rows[i].ID == old.ID {
			t.Fatalf("旧哈希行（幽灵设备）应被删除")
		}
	}
	if kept == nil {
		t.Fatal("应保留最近使用的那一行")
	}
	if kept.Remark == nil || *kept.Remark != "我的主力机" {
		t.Fatalf("备注应保留，实际 %v", kept.Remark)
	}
	if kept.AccessCount != 6 {
		t.Fatalf("访问次数应累计为 6，实际 %d", kept.AccessCount)
	}
	// 未参与合并的设备不能被动
	var vivo models.Device
	if err := db.First(&vivo, other.ID).Error; err != nil {
		t.Fatal("不同指纹的设备不应被删除")
	}
	// current_devices 重算
	var sub models.Subscription
	db.First(&sub, subID)
	if sub.CurrentDevices != 2 {
		t.Fatalf("current_devices 应重算为 2，实际 %d", sub.CurrentDevices)
	}
}

func TestMergeDuplicateDevices_KeepsKickedState(t *testing.T) {
	db, subID := setupDedupDB(t)
	now := time.Now()
	// 同一台设备被用户删过（被踢）后又因升级多登记了一行活跃行：
	// 合并时必须保留「活跃且未踢」的那行（用户现在在用它）
	mkDevice(t, db, subID, "hashA", "MoneyFly", "iOS", "iPhone14,3", "Apple", "", false, now.Add(-2*time.Hour))
	live := mkDevice(t, db, subID, "hashB", "MoneyFly", "iOS", "iPhone14,3", "Apple", "", true, now)

	if _, err := MergeDuplicateDevices(db, true); err != nil {
		t.Fatalf("合并失败: %v", err)
	}
	var rows []models.Device
	db.Where("subscription_id = ?", subID).Find(&rows)
	if len(rows) != 1 || rows[0].ID != live.ID {
		t.Fatalf("应只保留活跃的那行（id=%d），实际 %d 行", live.ID, len(rows))
	}
	if !rows[0].IsActive || rows[0].KickedAt != nil {
		t.Fatal("保留的行必须仍是活跃且未被踢")
	}
}

func TestMergeDuplicateDevices_DryRun(t *testing.T) {
	db, subID := setupDedupDB(t)
	now := time.Now()
	mkDevice(t, db, subID, "hashA", "MoneyFly", "iOS", "iPhone14,3", "Apple", "", true, now.Add(-time.Hour))
	mkDevice(t, db, subID, "hashB", "MoneyFly", "iOS", "iPhone14,3", "Apple", "", true, now)

	report, err := MergeDuplicateDevices(db, false)
	if err != nil {
		t.Fatalf("预演失败: %v", err)
	}
	if report.GroupsMerged != 1 {
		t.Fatalf("预演应报告 1 组待合并，实际 %d", report.GroupsMerged)
	}
	var count int64
	db.Model(&models.Device{}).Where("subscription_id = ?", subID).Count(&count)
	if count != 2 {
		t.Fatalf("预演不得改库，实际剩 %d 行", count)
	}
}

func TestMergeDuplicateDevices_SkipsIncompleteFingerprint(t *testing.T) {
	db, subID := setupDedupDB(t)
	now := time.Now()
	// 机型缺失（第三方客户端 UA 解析不全）：指纹不完整 → 不合并，避免误杀两台真设备
	mkDevice(t, db, subID, "hashA", "Clash", "Windows", "", "", "", true, now.Add(-time.Hour))
	mkDevice(t, db, subID, "hashB", "Clash", "Windows", "", "", "", true, now)

	report, err := MergeDuplicateDevices(db, true)
	if err != nil {
		t.Fatalf("合并失败: %v", err)
	}
	if report.GroupsMerged != 0 {
		t.Fatalf("指纹不完整时不应合并，实际 %d", report.GroupsMerged)
	}
	var count int64
	db.Model(&models.Device{}).Where("subscription_id = ?", subID).Count(&count)
	if count != 2 {
		t.Fatalf("不应删除任何行，实际剩 %d", count)
	}
}

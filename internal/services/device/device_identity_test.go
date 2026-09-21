package device

import (
	"testing"
	"time"

	"cboard-go/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 设备身份必须**跨版本稳定**（2026-09-21 修复回归）。
//
// 旧算法把 App 版本号与系统版本号算进设备身份，于是：
//   - 用户升级 App → 设备哈希变化 → 后端当成新设备再登记一行，旧行变幽灵设备；
//   - 用户升级系统 → 同样问题；
//   - 被删除（踢下线）的设备升级一次就能绕过踢下线判定。
//
// 现在：优先用客户端上报的安装级唯一 ID（X-MF-Device-Id），否则退化为
// 「软件名+系统名+机型+品牌」的稳定特征；两者都不含版本号。
func TestDeviceHashStableAcrossVersions(t *testing.T) {
	dm := NewDeviceManager()

	uaA := "MoneyFly/2.2.14 (iOS 16.6.1; iPhone14,3)"
	uaB := "MoneyFly/2.2.15 (iOS 16.7.2; iPhone14,3)" // 升级 App + 升级系统

	if dm.GenerateDeviceHash(uaA, "1.2.3.4", "") != dm.GenerateDeviceHash(uaB, "1.2.3.4", "") {
		t.Fatal("升级 App/系统后设备哈希必须保持不变（否则会重复登记出幽灵设备）")
	}

	// 不同机型/品牌必须是不同设备
	uaC := "MoneyFly/2.2.15 (Android 13; V2072A Build/x)"
	if dm.GenerateDeviceHash(uaB, "1.2.3.4", "") == dm.GenerateDeviceHash(uaC, "1.2.3.4", "") {
		t.Fatal("不同设备不能算成同一台")
	}
}

func TestDeviceHashPrefersClientDeviceID(t *testing.T) {
	dm := NewDeviceManager()

	// 同型号两台手机：靠客户端上报的安装级 ID 区分
	h1 := dm.GenerateDeviceHashWithHeaders("MoneyFly/2.2.15 (iOS 16.6.1; iPhone14,3)", "1.1.1.1", "",
		map[string]string{"X-MF-Device-Id": "aaaa1111"})
	h2 := dm.GenerateDeviceHashWithHeaders("MoneyFly/2.2.15 (iOS 16.6.1; iPhone14,3)", "1.1.1.1", "",
		map[string]string{"X-MF-Device-Id": "bbbb2222"})
	if h1 == h2 {
		t.Fatal("不同安装实例必须算成不同设备（否则两台同型号手机只能占一个名额）")
	}

	// 同一设备 ID：换 App 版本、换网络都不变
	h3 := dm.GenerateDeviceHashWithHeaders("MoneyFly/2.2.16 (iOS 17.0; iPhone14,3)", "9.9.9.9", "",
		map[string]string{"X-MF-Device-Id": "aaaa1111"})
	if h1 != h3 {
		t.Fatal("同一安装实例跨版本/跨网络必须稳定")
	}

	// 显式 deviceID 参数优先于头
	if dm.GenerateDeviceHashWithHeaders("x", "1.1.1.1", "explicit",
		map[string]string{"X-MF-Device-Id": "header"}) !=
		dm.GenerateDeviceHashWithHeaders("x", "1.1.1.1", "explicit", nil) {
		t.Fatal("显式 deviceID 应优先于请求头")
	}
}

func setupAdoptionDB(t *testing.T) (*gorm.DB, *DeviceManager, uint) {
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
	sub := models.Subscription{DeviceLimit: 5}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("建订阅失败: %v", err)
	}
	return db, &DeviceManager{db: db}, sub.ID
}

// 存量自愈：库里按旧算法（含版本号）写的行，遇到稳定身份的新请求时应当被
// 「认领」并把哈希改写成稳定值 —— 不新增幽灵设备、也不丢设备备注。
func TestFindExistingDeviceAdoptsLegacyHashRow(t *testing.T) {
	db, dm, subID := setupAdoptionDB(t)

	const ua = "MoneyFly/2.2.15 (iOS 16.6.1; iPhone14,3)"
	legacy := dm.GenerateLegacyDeviceHash(ua, nil)
	stable := dm.GenerateDeviceHash(ua, "1.2.3.4", "")
	if legacy == stable {
		t.Fatal("前置条件不成立：新旧算法结果应不同")
	}

	remark := "我的主力机"
	device := models.Device{
		SubscriptionID:    subID,
		DeviceFingerprint: legacy[:32],
		DeviceHash:        &legacy,
		UserAgent:         strPtr(ua),
		SoftwareName:      strPtr("MoneyFly"),
		OSName:            strPtr("iOS"),
		DeviceModel:       strPtr("iPhone14,3"),
		DeviceBrand:       strPtr("Apple"),
		Remark:            &remark,
		IsActive:          true,
		IsAllowed:         true,
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("造存量设备失败: %v", err)
	}

	found, exists, err := dm.FindExistingDeviceWithHeaders(subID, ua, "1.2.3.4", nil)
	if err != nil || !exists {
		t.Fatalf("应认领旧哈希行，exists=%v err=%v", exists, err)
	}
	if found.ID != device.ID {
		t.Fatalf("应命中存量行 %d，实际 %d", device.ID, found.ID)
	}
	if found.DeviceHash == nil || *found.DeviceHash != stable {
		t.Fatalf("命中后应把 device_hash 改写为稳定值，实际 %v", found.DeviceHash)
	}

	var count int64
	db.Model(&models.Device{}).Where("subscription_id = ?", subID).Count(&count)
	if count != 1 {
		t.Fatalf("不得新增设备行，实际 %d 行", count)
	}
	var after models.Device
	db.First(&after, device.ID)
	if after.Remark == nil || *after.Remark != "我的主力机" {
		t.Fatal("认领后必须保留原有备注")
	}
}

// 被踢下线的设备升级一次 App 不能绕过判定（否则用户"删除设备"的意图被绕过）
func TestKickedStillMatchesAfterVersionUpgrade(t *testing.T) {
	db, dm, subID := setupAdoptionDB(t)

	const uaOld = "MoneyFly/2.2.14 (iOS 16.6.1; iPhone14,3)"
	legacy := dm.GenerateLegacyDeviceHash(uaOld, nil)
	// 行的字段必须与解析器真实输出一致（机型落库的是解析出的显示名，
	// 不是 UA 里的原始机型串），否则指纹兜底对不上
	info := dm.ParseUserAgent(uaOld)
	kicked := time.Now()
	device := models.Device{
		SubscriptionID:    subID,
		DeviceFingerprint: legacy[:32],
		DeviceHash:        &legacy,
		UserAgent:         strPtr(uaOld),
		SoftwareName:      strPtr(info.SoftwareName),
		OSName:            strPtr(info.OSName),
		DeviceModel:       strPtr(info.DeviceModel),
		DeviceBrand:       strPtr(info.DeviceBrand),
		IsActive:          true,
		IsAllowed:         true,
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("造设备失败: %v", err)
	}
	if err := db.Model(&models.Device{}).Where("id = ?", device.ID).
		Updates(map[string]interface{}{"is_active": false, "kicked_at": kicked}).Error; err != nil {
		t.Fatalf("踢下线失败: %v", err)
	}

	// 用户升级到 2.2.15 后再拉订阅：仍然必须判定为「已被移除并踢下线」
	uaNew := "MoneyFly/2.2.15 (iOS 16.7.2; iPhone14,3)"
	hit, err := dm.FindKickedDeviceWithHeaders(subID, uaNew, "1.2.3.4", nil)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if hit == nil {
		t.Fatal("升级 App 后仍应命中踢下线判定（否则等于绕过用户删除设备的意图）")
	}
}

func strPtr(s string) *string { return &s }

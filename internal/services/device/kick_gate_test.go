package device

import (
	"testing"

	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

// 订阅接口的「被删除设备」闸门（kick gate）回归测试。
//
// 事故（2026-09-21 生产）：删除设备写 is_active=false + kicked_at=now，订阅接口
// 对命中该软删行的设备一律 403，而没有任何代码清除 kicked_at → 永久黑名单。
// 线上实测：148 个有效订阅 / 408 台设备被锁死，被锁机器每 30 分钟重试一次、
// 永久失败，即便订阅名额根本没用满（例如 2/5）也照锁。
//
// 修复后的语义：
//  1. 设备已在活跃列表 → 放行（历史软删行不再拦已登记设备）；
//  2. 名额未满 → 自动恢复该设备并放行（自愈）；
//  3. 名额已满 → 仍然拒绝（Blocked），引导先删闲置设备。
func TestResolveKickGateAutoRevivesWhenSlotFree(t *testing.T) {
	db, dm, subID := setupReviveTestDB(t)
	kicked := seedKickedDevice(t, db, dm, subID, "被删掉的本机")

	res, err := dm.ResolveKickGate(subID, 3, false, testUA, "1.2.3.4", nil)
	if err != nil {
		t.Fatalf("ResolveKickGate 出错: %v", err)
	}
	if res.Blocked {
		t.Fatalf("名额未满（0/3）不该拦截，结果: %+v", res)
	}
	if !res.Revived {
		t.Fatalf("名额未满时应自动恢复，结果: %+v", res)
	}
	if res.Reason != "auto-revived" || res.DeviceID != kicked.ID {
		t.Fatalf("结论不符: %+v（期望 auto-revived / device %d）", res, kicked.ID)
	}

	var row models.Device
	if err := db.First(&row, kicked.ID).Error; err != nil {
		t.Fatalf("读回设备失败: %v", err)
	}
	if !row.IsActive || row.KickedAt != nil {
		t.Fatalf("设备未被真正恢复: is_active=%v kicked_at=%v", row.IsActive, row.KickedAt)
	}
	if got := subscriptionDeviceCount(t, db, subID); got != 1 {
		t.Fatalf("current_devices 未同步，期望 1，实际 %d", got)
	}
}

func TestResolveKickGateBlocksOnlyWhenFull(t *testing.T) {
	db, dm, subID := setupReviveTestDB(t)
	seedKickedDevice(t, db, dm, subID, "被删掉的本机")
	// 另外塞满 3 台活跃设备（订阅上限 3）
	for i := 0; i < 3; i++ {
		ua := []string{
			"MoneyFly/2.2.15 (Linux; Android 14; Pixel 8 Build/UQ1A)",
			"Clash Verge/v2.5.2 Windows",
			"Shadowrocket/3445 CFNetwork/3896 Darwin/23.0.0 iPhone15,3",
		}[i]
		seedActiveDevice(t, db, dm, subID, ua)
	}

	res, err := dm.ResolveKickGate(subID, 3, false, testUA, "1.2.3.4", nil)
	if err != nil {
		t.Fatalf("ResolveKickGate 出错: %v", err)
	}
	if !res.Blocked || res.Reason != "device-limit" {
		t.Fatalf("名额已满应拦截，结果: %+v", res)
	}
	// 被拦截时绝不能偷偷恢复
	var row models.Device
	if err := db.Where("subscription_id = ? AND is_active = ?", subID, false).First(&row).Error; err != nil {
		t.Fatalf("读回软删行失败: %v", err)
	}
	if row.KickedAt == nil {
		t.Fatal("名额已满却被恢复了：软删行的 kicked_at 被清空")
	}
}

func TestResolveKickGateAllowsUnlimitedDevices(t *testing.T) {
	db, dm, subID := setupReviveTestDB(t)
	seedKickedDevice(t, db, dm, subID, "被删掉的本机")
	for i := 0; i < 5; i++ {
		seedActiveDevice(t, db, dm, subID, []string{
			"MoneyFly/2.2.15 (Linux; Android 14; Pixel 8 Build/UQ1A)",
			"Clash Verge/v2.5.2 Windows",
			"Shadowrocket/3445 CFNetwork/3896 Darwin/23.0.0 iPhone15,3",
			"mihomo.party/v2.0.2 (clash.meta)",
			"v2rayN/7.14.12",
		}[i])
	}

	// unlimited=true（用户开了「不限制设备」）→ 超限也恢复
	res, err := dm.ResolveKickGate(subID, 3, true, testUA, "1.2.3.4", nil)
	if err != nil {
		t.Fatalf("ResolveKickGate 出错: %v", err)
	}
	if res.Blocked || !res.Revived {
		t.Fatalf("不限制设备时应恢复放行，结果: %+v", res)
	}
}

func TestResolveKickGateSkipsWhenDeviceAlreadyActive(t *testing.T) {
	db, dm, subID := setupReviveTestDB(t)
	headers := map[string]string{
		"X-MF-Device-Model": "Windows 10 Pro",
		"X-MF-Device-Brand": "PC",
		"X-MF-OS":           "Windows 10.0",
		"X-MF-Device-Type":  "desktop",
		"X-MF-Device-Id":    "machine-test-abcdef",
	}
	newUA := "MoneyFly/2.2.16 (Windows NT 10.0)"
	info := dm.ParseUserAgentWithHeaders(newUA, headers)

	// 同一台机器的历史软删行：特征与当前请求完全一致，但哈希是旧算法留下的
	// （isLegacyHashRow=true → 会被稳定指纹兜底命中），这是线上真实形态
	// （旧版本 App 的那一行被删掉 / 被踢下线）。
	legacyHash := "0000000000000000000000000000000000000000000000000000000000000001"
	oldUA := "MoneyFly/2.1.3 (Windows NT 10.0)"
	oldIP := "1.2.3.4"
	now := utils.GetBeijingTime()
	old := models.Device{
		SubscriptionID:    subID,
		DeviceFingerprint: legacyHash[:32],
		DeviceHash:        &legacyHash,
		UserAgent:         &oldUA,
		IPAddress:         &oldIP,
		SoftwareName:      &info.SoftwareName,
		OSName:            &info.OSName,
		DeviceModel:       &info.DeviceModel,
		DeviceBrand:       &info.DeviceBrand,
		IsActive:          true,
		IsAllowed:         true,
		LastAccess:        now,
	}
	if err := db.Create(&old).Error; err != nil {
		t.Fatalf("建设备失败: %v", err)
	}
	if err := db.Model(&models.Device{}).Where("id = ?", old.ID).
		Updates(map[string]interface{}{"is_active": false, "kicked_at": now}).Error; err != nil {
		t.Fatalf("写入踢下线状态失败: %v", err)
	}

	// 同机器的当前活跃行（稳定身份：X-MF-Device-Id）
	stableHash := dm.GenerateDeviceHashWithHeaders(newUA, "9.9.9.9", "", headers)
	activeIP := "9.9.9.9"
	active := models.Device{
		SubscriptionID:    subID,
		DeviceFingerprint: stableHash[:32],
		DeviceHash:        &stableHash,
		UserAgent:         &newUA,
		IPAddress:         &activeIP,
		SoftwareName:      &info.SoftwareName,
		OSName:            &info.OSName,
		DeviceModel:       &info.DeviceModel,
		DeviceBrand:       &info.DeviceBrand,
		IsActive:          true,
		IsAllowed:         true,
		LastAccess:        now,
	}
	if err := db.Create(&active).Error; err != nil {
		t.Fatalf("建活跃设备失败: %v", err)
	}

	res, err := dm.ResolveKickGate(subID, 3, false, newUA, "9.9.9.9", headers)
	if err != nil {
		t.Fatalf("ResolveKickGate 出错: %v", err)
	}
	if res.Blocked {
		t.Fatalf("设备已在活跃列表，不该拦截: %+v", res)
	}
	if res.Revived || res.Reason != "already-active" {
		t.Fatalf("期望 already-active（不恢复历史软删行），实际 %+v", res)
	}
	// 历史软删行应保持原样（不应被顺手恢复成第二台设备）
	var old2 models.Device
	if err := db.First(&old2, old.ID).Error; err != nil {
		t.Fatalf("读回历史行失败: %v", err)
	}
	if old2.KickedAt == nil || old2.IsActive {
		t.Fatalf("历史软删行被误恢复: is_active=%v kicked_at=%v", old2.IsActive, old2.KickedAt)
	}
}

func TestResolveKickGateNoopWhenNotKicked(t *testing.T) {
	db, dm, subID := setupReviveTestDB(t)
	seedActiveDevice(t, db, dm, subID, testUA)

	res, err := dm.ResolveKickGate(subID, 3, false, testUA, "1.2.3.4", nil)
	if err != nil {
		t.Fatalf("ResolveKickGate 出错: %v", err)
	}
	if res.Blocked || res.Revived || res.Reason != "not-kicked" {
		t.Fatalf("没被踢过的设备应为 no-op，实际 %+v", res)
	}
}

// seedActiveDevice 造一条正常活跃设备行（字段与记录访问路径写入的一致）
func seedActiveDevice(t *testing.T, db *gorm.DB, dm *DeviceManager, subID uint, ua string) models.Device {
	t.Helper()
	hash := dm.GenerateDeviceHash(ua, "9.9.9.9", "")
	now := utils.GetBeijingTime()
	ip := "9.9.9.9"
	row := models.Device{
		SubscriptionID:    subID,
		DeviceFingerprint: hash[:32],
		DeviceHash:        &hash,
		UserAgent:         &ua,
		IPAddress:         &ip,
		IsActive:          true,
		IsAllowed:         true,
		LastAccess:        now,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("建活跃设备失败: %v", err)
	}
	return row
}

// subscriptionDeviceCount 读回订阅上的 current_devices（校验恢复后计数同步）
func subscriptionDeviceCount(t *testing.T, db *gorm.DB, subID uint) int {
	t.Helper()
	var sub models.Subscription
	if err := db.First(&sub, subID).Error; err != nil {
		t.Fatalf("读回订阅失败: %v", err)
	}
	return sub.CurrentDevices
}

// 恢复必须落在**同一行**上：放行后紧跟的设备访问记录不能又新建一行，
// 否则同一台机器会重复占名额（线上 sub 638 曾出现 9890 + 10703 两行）。
func TestResolveKickGateRevivesSameRowNoDuplicate(t *testing.T) {
	db, dm, subID := setupReviveTestDB(t)
	kicked := seedKickedDevice(t, db, dm, subID, "本机")

	res, err := dm.ResolveKickGate(subID, 3, false, testUA, "1.2.3.4", nil)
	if err != nil {
		t.Fatalf("ResolveKickGate 出错: %v", err)
	}
	if !res.Revived || res.DeviceID != kicked.ID {
		t.Fatalf("应恢复命中行 %d，实际 %+v", kicked.ID, res)
	}

	if _, err := dm.RecordDeviceAccessWithHeaders(subID, 1, testUA, "1.2.3.4", "clash", nil); err != nil {
		t.Fatalf("记录设备访问失败: %v", err)
	}
	var total int64
	if err := db.Model(&models.Device{}).Where("subscription_id = ?", subID).Count(&total).Error; err != nil {
		t.Fatalf("统计设备数失败: %v", err)
	}
	if total != 1 {
		t.Fatalf("恢复后不应新增重复行，实际 %d 行", total)
	}
	if got := subscriptionDeviceCount(t, db, subID); got != 1 {
		t.Fatalf("current_devices 应为 1，实际 %d", got)
	}
}

package device

import (
	"testing"
	"time"

	"cboard-go/internal/models"
)

// 生产实证（老站用户 susanzhang1230@hotmail.com）：
// 一台 Windows 机器，客户只有 iPhone + 这台 Windows 两台设备，
// 后台却显示 3 台 —— 多出来的 11513（MoneyFly 2.2.25）与 11666（MoneyFly 2.2.28）
// 是同一台 PC。根因是 Windows 安装包升级时 [UninstallDelete] 把
// %APPDATA%\top.moneyfly\MoneyFly 删掉了，install_id 丢失，
// 而 device_hash 就是由 install_id 派生的 ⇒ 同一台机器重新登记了一行。
//
// 修复三件套：
//  1. 安装包不再删数据目录（客户端侧）；
//  2. 客户端额外上报**硬件指纹** X-MF-Device-Fp（MachineGuid / IOPlatformUUID /
//     ANDROID_ID，跨重装不变）；
//  3. 服务端在"有显式安装 ID 就到此为止"之前，先按硬件指纹在同一订阅下认回老行。
//
// 本文件锁死第 3 步：认亲必须命中、必须不新增行、必须限定在同一订阅内。
func TestAdoptRowByHardwareFingerprintOnReinstall(t *testing.T) {
	db, dm, subID := setupAdoptionDB(t)

	const (
		ua    = "MoneyFly/2.2.28 (Windows NT 10.0; Win64; x64)"
		hwFP  = "mach-uuid-9f1c4e7a-dead-beef-0001"
		oldID = "1111111111111111111111111111aaaa" // 2.2.25 时代的 install_id
		newID = "2222222222222222222222222222bbbb" // 升级后重装的 install_id
	)
	oldHash := dm.GenerateDeviceHashWithHeaders(ua, "1.1.1.1", "", map[string]string{"X-MF-Device-Id": oldID})
	newHash := dm.GenerateDeviceHashWithHeaders(ua, "2.2.2.2", "", map[string]string{"X-MF-Device-Id": newID})
	if oldHash == newHash {
		t.Fatal("前置条件不成立：两个安装实例的哈希应不同")
	}

	remark := "家里的台式机"
	device := models.Device{
		SubscriptionID:    subID,
		DeviceFingerprint: oldHash,
		DeviceHash:        &oldHash,
		DeviceHwFp:        strPtr(hwFP),
		UserAgent:         strPtr(ua),
		SoftwareName:      strPtr("MoneyFly"),
		OSName:            strPtr("Windows"),
		Remark:            &remark,
		IsActive:          true,
		IsAllowed:         true,
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("造设备失败: %v", err)
	}

	hdrs := map[string]string{"X-MF-Device-Id": newID, "X-MF-Device-Fp": hwFP}
	found, exists, err := dm.FindExistingDeviceWithHeaders(subID, ua, "2.2.2.2", hdrs)
	if err != nil {
		t.Fatalf("查找失败: %v", err)
	}
	if !exists {
		t.Fatal("同一台机器换安装实例后必须靠硬件指纹认回老行（否则又多一行幽灵设备、白占名额）")
	}
	if found == nil || found.ID != device.ID {
		t.Fatalf("应命中老行 %d，实际 %v", device.ID, found)
	}
	if found.DeviceHash == nil || *found.DeviceHash != newHash {
		t.Fatalf("认回后 device_hash 应改写为当前安装实例哈希，实际 %v", found.DeviceHash)
	}

	var count int64
	db.Model(&models.Device{}).Where("subscription_id = ?", subID).Count(&count)
	if count != 1 {
		t.Fatalf("不得新增设备行，实际 %d 行", count)
	}
	var after models.Device
	db.First(&after, device.ID)
	if after.Remark == nil || *after.Remark != "家里的台式机" {
		t.Fatal("认回后必须保留原有备注")
	}
}

// 认亲只认同一订阅：别的订阅里碰巧同指纹的行绝不能被抢过来
// （否则一个用户重装会"吃掉"另一个用户的设备行）。
func TestHardwareFingerprintAdoptionIsSubscriptionScoped(t *testing.T) {
	db, dm, subID := setupAdoptionDB(t)

	const (
		ua   = "MoneyFly/2.2.28 (Windows NT 10.0; Win64; x64)"
		hwFP = "mach-uuid-shared-fixture"
	)
	otherHash := dm.GenerateDeviceHashWithHeaders(ua, "9.9.9.9", "", map[string]string{"X-MF-Device-Id": "otherinstall"})

	otherSub := models.Subscription{DeviceLimit: 5, SubscriptionURL: "other-sub-token"}
	if err := db.Create(&otherSub).Error; err != nil {
		t.Fatalf("建第二个订阅失败: %v", err)
	}
	other := models.Device{
		SubscriptionID:    otherSub.ID,
		DeviceFingerprint: otherHash,
		DeviceHash:        &otherHash,
		DeviceHwFp:        strPtr(hwFP),
		IsActive:          true,
		IsAllowed:         true,
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("造设备失败: %v", err)
	}

	hdrs := map[string]string{"X-MF-Device-Id": "myinstall", "X-MF-Device-Fp": hwFP}
	found, exists, err := dm.FindExistingDeviceWithHeaders(subID, ua, "1.1.1.1", hdrs)
	if err != nil {
		t.Fatalf("查找失败: %v", err)
	}
	if exists {
		t.Fatalf("不得跨订阅认亲，实际命中了 %v（订阅 %d 的行）", found, otherSub.ID)
	}
}

// 不同机器同订阅：必须仍是新设备（硬件指纹不能把两台机器合并成一台）
func TestHardwareFingerprintDoesNotMergeDifferentMachines(t *testing.T) {
	db, dm, subID := setupAdoptionDB(t)

	const ua = "MoneyFly/2.2.28 (Windows NT 10.0; Win64; x64)"
	hashA := dm.GenerateDeviceHashWithHeaders(ua, "1.1.1.1", "", map[string]string{"X-MF-Device-Id": "machineA"})
	a := models.Device{
		SubscriptionID:    subID,
		DeviceFingerprint: hashA,
		DeviceHash:        &hashA,
		DeviceHwFp:        strPtr("mach-A"),
		UserAgent:         strPtr(ua),
		IsActive:          true,
		IsAllowed:         true,
	}
	if err := db.Create(&a).Error; err != nil {
		t.Fatalf("造设备失败: %v", err)
	}

	hdrs := map[string]string{"X-MF-Device-Id": "machineB", "X-MF-Device-Fp": "mach-B"}
	if _, exists, err := dm.FindExistingDeviceWithHeaders(subID, ua, "2.2.2.2", hdrs); err != nil {
		t.Fatalf("查找失败: %v", err)
	} else if exists {
		t.Fatal("不同硬件指纹必须是新设备（否则两台机器只占一个名额）")
	}
}

// 客户端没上报硬件指纹（第三方客户端 / 老版本）时行为完全不变：
// 有安装 ID 就按安装 ID，哈希没命中就是新设备。
func TestNoHardwareFingerprintKeepsLegacyBehavior(t *testing.T) {
	db, dm, subID := setupAdoptionDB(t)

	const ua = "MoneyFly/2.2.28 (Windows NT 10.0; Win64; x64)"
	hashA := dm.GenerateDeviceHashWithHeaders(ua, "1.1.1.1", "", map[string]string{"X-MF-Device-Id": "installA"})
	a := models.Device{
		SubscriptionID:    subID,
		DeviceFingerprint: hashA,
		DeviceHash:        &hashA,
		DeviceHwFp:        strPtr("mach-A"),
		UserAgent:         strPtr(ua),
		IsActive:          true,
		IsAllowed:         true,
	}
	if err := db.Create(&a).Error; err != nil {
		t.Fatalf("造设备失败: %v", err)
	}

	hdrs := map[string]string{"X-MF-Device-Id": "installB"} // 无 -Fp
	if _, exists, err := dm.FindExistingDeviceWithHeaders(subID, ua, "2.2.2.2", hdrs); err != nil {
		t.Fatalf("查找失败: %v", err)
	} else if exists {
		t.Fatal("没上报硬件指纹时不得认亲（避免把真实的新机器误判为老设备）")
	}

	// 记录访问后应落成新行，并把指纹写进去供以后认亲
	if _, err := dm.RecordDeviceAccessWithHeaders(subID, 1, ua, "2.2.2.2", "subscription",
		map[string]string{"X-MF-Device-Id": "installC", "X-MF-Device-Fp": "mach-C"}); err != nil {
		t.Fatalf("记录访问失败: %v", err)
	}
	var c models.Device
	if err := db.Where("subscription_id = ? AND device_hw_fp = ?", subID, "mach-C").First(&c).Error; err != nil {
		t.Fatal("新建设备时必须把硬件指纹落库（否则以后重装无法认亲）")
	}

	// 同一机器再次重装：这次应认回 mach-C 那一行
	again, exists, err := dm.FindExistingDeviceWithHeaders(subID, ua, "3.3.3.3",
		map[string]string{"X-MF-Device-Id": "installD", "X-MF-Device-Fp": "mach-C"})
	if err != nil || !exists {
		t.Fatalf("重装后应认回 mach-C 行，exists=%v err=%v", exists, err)
	}
	if again.ID != c.ID {
		t.Fatalf("应命中 %d，实际 %d", c.ID, again.ID)
	}
}

// 重装不能绕过「用户删除设备 = 踢下线」：同一台机器换了 install_id
// （硬件指纹不变）时，闸门仍必须命中那条软删行；而**认亲逻辑绝不能自己把它复活**。
func TestHardwareFingerprintStillHitsKickGate(t *testing.T) {
	db, dm, subID := setupAdoptionDB(t)

	const (
		ua    = "MoneyFly/2.2.28 (Windows NT 10.0; Win64; x64)"
		hwFP  = "mach-uuid-kicked-0001"
		oldID = "kickedinstall000000000000000000"
		newID = "reinstall0000000000000000000000"
	)
	oldHash := dm.GenerateDeviceHashWithHeaders(ua, "1.1.1.1", "", map[string]string{"X-MF-Device-Id": oldID})
	kickedRow := models.Device{
		SubscriptionID:    subID,
		DeviceFingerprint: oldHash,
		DeviceHash:        &oldHash,
		DeviceHwFp:        strPtr(hwFP),
		UserAgent:         strPtr(ua),
		SoftwareName:      strPtr("MoneyFly"),
		OSName:            strPtr("Windows"),
		IsActive:          false,
		IsAllowed:         true,
	}
	if err := db.Create(&kickedRow).Error; err != nil {
		t.Fatalf("造设备失败: %v", err)
	}
	// ⚠️ is_active 有 gorm default:true，插入时的零值 false 会被默认值覆盖，
	// 必须像真实删除接口那样先建行再 Update 成软删状态。
	if err := db.Model(&models.Device{}).Where("id = ?", kickedRow.ID).
		Updates(map[string]interface{}{"is_active": false, "kicked_at": time.Now()}).Error; err != nil {
		t.Fatalf("踢下线失败: %v", err)
	}

	hdrs := map[string]string{"X-MF-Device-Id": newID, "X-MF-Device-Fp": hwFP}

	hit, err := dm.FindKickedDeviceWithHeaders(subID, ua, "2.2.2.2", hdrs)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if hit == nil || hit.ID != kickedRow.ID {
		t.Fatal("重装（install_id 变化）后仍必须命中踢下线判定，否则用户「删除设备」的意图被绕过")
	}

	// 认亲绝不能复活软删行
	found, exists, err := dm.FindExistingDeviceWithHeaders(subID, ua, "2.2.2.2", hdrs)
	if err != nil {
		t.Fatalf("查找失败: %v", err)
	}
	if exists {
		t.Fatalf("认亲不得命中被踢下线的软删行，实际 %v", found)
	}
	var after models.Device
	db.First(&after, kickedRow.ID)
	if after.IsActive || after.KickedAt == nil {
		t.Fatal("认亲路径不得改写/复活软删行的状态")
	}
	if after.DeviceHash == nil || *after.DeviceHash != oldHash {
		t.Fatal("认亲路径不得改写软删行的 device_hash（会导致闸门误放行）")
	}
}

// 完整链路：被删除的机器重装后再次拉订阅 → 闸门按硬件指纹命中软删行 →
// 名额未满自动恢复（清除 kicked_at）→ 随后登记走硬件指纹认亲，
// 全程**只有一行**（不新增幽灵设备、不白占名额）。
func TestKickGateReviveThenAdoptSingleRow(t *testing.T) {
	db, dm, subID := setupAdoptionDB(t)

	const (
		ua    = "MoneyFly/2.2.28 (Windows NT 10.0; Win64; x64)"
		hwFP  = "mach-uuid-revive-0001"
		oldID = "oldinstall0000000000000000000000"
		newID = "newinstall0000000000000000000000"
	)
	oldHash := dm.GenerateDeviceHashWithHeaders(ua, "1.1.1.1", "", map[string]string{"X-MF-Device-Id": oldID})
	row := models.Device{
		SubscriptionID:    subID,
		DeviceFingerprint: oldHash,
		DeviceHash:        &oldHash,
		DeviceHwFp:        strPtr(hwFP),
		UserAgent:         strPtr(ua),
		SoftwareName:      strPtr("MoneyFly"),
		OSName:            strPtr("Windows"),
		IsActive:          true,
		IsAllowed:         true,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("造设备失败: %v", err)
	}
	if err := db.Model(&models.Device{}).Where("id = ?", row.ID).
		Updates(map[string]interface{}{"is_active": false, "kicked_at": time.Now()}).Error; err != nil {
		t.Fatalf("踢下线失败: %v", err)
	}

	hdrs := map[string]string{"X-MF-Device-Id": newID, "X-MF-Device-Fp": hwFP}
	const limit = 5
	gate, err := dm.ResolveKickGate(subID, limit, false, ua, "2.2.2.2", hdrs)
	if err != nil {
		t.Fatalf("闸门失败: %v", err)
	}
	if !gate.Revived || gate.DeviceID != row.ID {
		t.Fatalf("名额未满应自动恢复被删除的设备，得到 %+v", gate)
	}

	found, exists, err := dm.FindExistingDeviceWithHeaders(subID, ua, "2.2.2.2", hdrs)
	if err != nil || !exists {
		t.Fatalf("恢复后登记应认回同一行，exists=%v err=%v", exists, err)
	}
	if found.ID != row.ID {
		t.Fatalf("应命中 %d，实际 %d", row.ID, found.ID)
	}
	var count int64
	db.Model(&models.Device{}).Where("subscription_id = ?", subID).Count(&count)
	if count != 1 {
		t.Fatalf("全程只应有一行，实际 %d 行", count)
	}

	// 名额已满时不得恢复（仍然拦截），避免「删除无用」导致超额
	if err := db.Model(&models.Device{}).Where("id = ?", row.ID).
		Updates(map[string]interface{}{"is_active": false, "kicked_at": time.Now()}).Error; err != nil {
		t.Fatalf("再次踢下线失败: %v", err)
	}
	filler := models.Device{SubscriptionID: subID, DeviceFingerprint: "filler", IsActive: true, IsAllowed: true}
	if err := db.Create(&filler).Error; err != nil {
		t.Fatalf("造占位设备失败: %v", err)
	}
	blocked, err := dm.ResolveKickGate(subID, 1, false, ua, "2.2.2.2", hdrs)
	if err != nil {
		t.Fatalf("闸门失败: %v", err)
	}
	if !blocked.Blocked || blocked.Reason != "device-limit" {
		t.Fatalf("名额已满必须拦截，得到 %+v", blocked)
	}
}

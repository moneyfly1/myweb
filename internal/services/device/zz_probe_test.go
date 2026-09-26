package device

import (
	"testing"
	"time"

	"cboard-go/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 生产实证：老客户端（无 X-MF-Device-Id）建的行是「特征哈希」，升级到带
// 设备 ID 的新客户端后，请求哈希变成 device_id 哈希 —— 看是否还能认回同一行。
func TestProbeUpgradeToDeviceIDClient(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file:probe?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err := db.AutoMigrate(&models.Device{}, &models.Subscription{}); err != nil {
		t.Fatal(err)
	}
	dm := &DeviceManager{db: db}
	sub := models.Subscription{DeviceLimit: 5}
	db.Create(&sub)

	oldUA := "MoneyFly/2.2.14 (Windows NT 10.0)"
	headers := map[string]string{
		"X-MF-Device-Model": "Windows 10 Pro",
		"X-MF-Device-Brand": "PC",
		"X-MF-OS":           "Windows 10.0",
	}
	// 老客户端建行（无设备 ID）
	oldHash := dm.GenerateDeviceHashWithHeaders(oldUA, "1.1.1.1", "", headers)
	t.Logf("老客户端 特征哈希 = %s", oldHash)
	t.Logf("生产行 10674 哈希 = 9860ee9e41e33c7c66d484b49a7ef9c53f2608e82b301116c6cfd228d3c920ea")
	t.Logf("两者一致？%v", oldHash == "9860ee9e41e33c7c66d484b49a7ef9c53f2608e82b301116c6cfd228d3c920ea")

	info := dm.ParseUserAgentWithHeaders(oldUA, headers)
	t.Logf("解析特征: sw=%q os=%q model=%q brand=%q", info.SoftwareName, info.OSName, info.DeviceModel, info.DeviceBrand)
	t.Logf("stableHashFromFeatures = %s", stableHashFromFeatures(info.SoftwareName, info.OSName, info.DeviceModel, info.DeviceBrand))

	ua := oldUA
	ip := "1.1.1.1"
	db.Create(&models.Device{
		SubscriptionID: sub.ID, DeviceHash: &oldHash, UserAgent: &ua, IPAddress: &ip,
		SoftwareName: &info.SoftwareName, OSName: &info.OSName,
		DeviceModel: &info.DeviceModel, DeviceBrand: &info.DeviceBrand,
		IsActive: true, IsAllowed: true, LastAccess: nowPtrForProbe(),
	})

	// 升级后的新客户端：同机 + X-MF-Device-Id
	newUA := "MoneyFly/2.2.16 (Windows NT 10.0)"
	newHeaders := map[string]string{
		"X-MF-Device-Model": "Windows 10 Pro",
		"X-MF-Device-Brand": "PC",
		"X-MF-OS":           "Windows 10.0",
		"X-MF-Device-Id":    "abcd1234abcd1234",
	}
	row, found, err := dm.FindExistingDeviceWithHeaders(sub.ID, newUA, "2.2.2.2", newHeaders)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Logf("✅ 认回同一行 id=%d（升级不新增行）", row.ID)
	} else {
		t.Logf("❌ 没认回 → 升级后会新建一行（幽灵设备，白占名额）")
	}
}

func nowPtrForProbe() time.Time { return time.Now() }

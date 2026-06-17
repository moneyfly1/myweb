package device

import (
	"testing"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const hiddifyNextWindowsUA = "HiddifyNext/4.1.2 (windows) like ClashMeta v2ray sing-box"

func TestParseUserAgentPrioritizesHiddifyNextOverCoreMarkers(t *testing.T) {
	dm := NewDeviceManager()

	info := dm.ParseUserAgent(hiddifyNextWindowsUA)

	if info.SoftwareName != "Hiddify" {
		t.Fatalf("SoftwareName = %q, want Hiddify", info.SoftwareName)
	}
	if info.OSName != "Windows" {
		t.Fatalf("OSName = %q, want Windows", info.OSName)
	}
	if info.SoftwareVersion != "4.1.2" {
		t.Fatalf("SoftwareVersion = %q, want 4.1.2", info.SoftwareVersion)
	}
	if info.DeviceName != "Hiddify - Windows - v4.1.2" {
		t.Fatalf("DeviceName = %q, want Hiddify - Windows - v4.1.2", info.DeviceName)
	}
}

func TestRecordDeviceAccessRefreshesExistingParsedDeviceInfo(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:device_manager_refresh?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Subscription{}, &models.Device{}); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}

	previousDB := database.DB
	database.DB = db
	defer func() {
		database.DB = previousDB
	}()

	user := models.User{
		Username: "test-user",
		Email:    "test@example.com",
		Password: "hashed-password",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	subscription := models.Subscription{
		UserID:          user.ID,
		SubscriptionURL: "test-subscription-url",
		ExpireTime:      time.Now().Add(24 * time.Hour),
		IsActive:        true,
		Status:          "active",
		DeviceLimit:     3,
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	oldName := "V2Ray - Windows - v4.1.2"
	oldSoftware := "V2Ray"
	oldHash := "old-hash"
	deviceUA := hiddifyNextWindowsUA
	now := time.Now()
	device := models.Device{
		SubscriptionID:    subscription.ID,
		DeviceFingerprint: oldHash,
		DeviceHash:        &oldHash,
		DeviceUA:          &deviceUA,
		DeviceName:        &oldName,
		SoftwareName:      &oldSoftware,
		UserAgent:         &deviceUA,
		IsActive:          true,
		IsAllowed:         true,
		FirstSeen:         &now,
		LastAccess:        now,
		LastSeen:          &now,
		AccessCount:       1,
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}

	dm := NewDeviceManager()
	updated, err := dm.RecordDeviceAccess(subscription.ID, user.ID, hiddifyNextWindowsUA, "203.0.113.10", "clash")
	if err != nil {
		t.Fatalf("RecordDeviceAccess returned error: %v", err)
	}

	if updated.ID != device.ID {
		t.Fatalf("updated device ID = %d, want existing ID %d", updated.ID, device.ID)
	}
	if updated.DeviceName == nil || *updated.DeviceName != "Hiddify - Windows - v4.1.2" {
		t.Fatalf("DeviceName = %v, want Hiddify - Windows - v4.1.2", updated.DeviceName)
	}
	if updated.SoftwareName == nil || *updated.SoftwareName != "Hiddify" {
		t.Fatalf("SoftwareName = %v, want Hiddify", updated.SoftwareName)
	}
}

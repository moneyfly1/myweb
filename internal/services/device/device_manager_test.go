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
const genericClashWindowsUA = "Clash/0.19.23 (Windows)"
const genericClashWindowsNewVersionUA = "Clash - Windows - v0.20.0"
const clashMetaAndroidUA = "Clash Meta/2.10.4 (Linux; Android 13; Pixel 7 Build/TQ3A.230805.001)"
const clashMetaAndroidDisplayUA = "Clash Meta - Android - v2.10.4"
const clashMetaWindowsUA = "Clash Meta/2.10.4 (Windows NT 10.0; Win64; x64)"

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

func TestRecordDeviceAccessMergesGenericClashWindowsAliasWithClashMetaAndroid(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:device_manager_android_alias?mode=memory&cache=shared"), &gorm.Config{})
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
		Username: "android-user",
		Email:    "android@example.com",
		Password: "hashed-password",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	subscription := models.Subscription{
		UserID:          user.ID,
		SubscriptionURL: "android-alias-subscription-url",
		ExpireTime:      time.Now().Add(24 * time.Hour),
		IsActive:        true,
		Status:          "active",
		DeviceLimit:     1,
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	dm := NewDeviceManager()
	if !dm.IsGenericClashWindowsUA(genericClashWindowsUA) {
		t.Fatalf("IsGenericClashWindowsUA(%q) = false, want true", genericClashWindowsUA)
	}
	if !dm.IsGenericClashWindowsUA(genericClashWindowsNewVersionUA) {
		t.Fatalf("IsGenericClashWindowsUA(%q) = false, want true", genericClashWindowsNewVersionUA)
	}
	if !dm.isClashMetaAndroidUA(clashMetaAndroidDisplayUA) {
		t.Fatalf("isClashMetaAndroidUA(%q) = false, want true", clashMetaAndroidDisplayUA)
	}

	first, err := dm.RecordDeviceAccess(subscription.ID, user.ID, genericClashWindowsUA, "203.0.113.20", "clash")
	if err != nil {
		t.Fatalf("record generic Clash Windows UA: %v", err)
	}
	if first == nil {
		t.Fatal("record generic Clash Windows UA returned nil device")
	}
	if first.DeviceName == nil || *first.DeviceName != "Clash - Windows - v0.19.23" {
		t.Fatalf("DeviceName = %v, want Clash - Windows - v0.19.23", first.DeviceName)
	}

	second, err := dm.RecordDeviceAccess(subscription.ID, user.ID, clashMetaAndroidUA, "203.0.113.20", "clash")
	if err != nil {
		t.Fatalf("record Clash Meta Android UA: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("Clash Meta Android UA created device ID %d, want existing ID %d", second.ID, first.ID)
	}
	if second.DeviceName == nil || *second.DeviceName != "Clash Meta - Pixel 7 - Android 13 - v2.10.4" {
		t.Fatalf("DeviceName = %v, want Clash Meta Android device name", second.DeviceName)
	}

	var count int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscription.ID, true).Count(&count)
	if count != 1 {
		t.Fatalf("active device count = %d, want 1", count)
	}

	if _, exists, err := dm.FindExistingDevice(subscription.ID, genericClashWindowsUA, "203.0.113.20"); err != nil || !exists {
		t.Fatalf("FindExistingDevice generic alias exists = %v, err = %v; want true, nil", exists, err)
	}

	third, err := dm.RecordDeviceAccess(subscription.ID, user.ID, genericClashWindowsUA, "203.0.113.20", "clash")
	if err != nil {
		t.Fatalf("record generic Clash Windows UA after Android: %v", err)
	}
	if third.ID != first.ID {
		t.Fatalf("generic Clash Windows UA after Android updated device ID %d, want existing ID %d", third.ID, first.ID)
	}
	if third.DeviceName == nil || *third.DeviceName != "Clash Meta - Pixel 7 - Android 13 - v2.10.4" {
		t.Fatalf("DeviceName after generic alias = %v, want Clash Meta Android device name", third.DeviceName)
	}
}

func TestRecordDeviceAccessDoesNotMergeRealWindowsClashMetaWithGenericClashWindows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:device_manager_windows_clash_meta?mode=memory&cache=shared"), &gorm.Config{})
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
		Username: "windows-user",
		Email:    "windows@example.com",
		Password: "hashed-password",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	subscription := models.Subscription{
		UserID:          user.ID,
		SubscriptionURL: "windows-clash-meta-subscription-url",
		ExpireTime:      time.Now().Add(24 * time.Hour),
		IsActive:        true,
		Status:          "active",
		DeviceLimit:     3,
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	dm := NewDeviceManager()
	first, err := dm.RecordDeviceAccess(subscription.ID, user.ID, clashMetaWindowsUA, "203.0.113.30", "clash")
	if err != nil {
		t.Fatalf("record Clash Meta Windows UA: %v", err)
	}
	second, err := dm.RecordDeviceAccess(subscription.ID, user.ID, genericClashWindowsUA, "203.0.113.30", "clash")
	if err != nil {
		t.Fatalf("record generic Clash Windows UA: %v", err)
	}
	if second.ID == first.ID {
		t.Fatalf("generic Clash Windows UA merged into real Windows Clash Meta device ID %d", first.ID)
	}

	var count int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscription.ID, true).Count(&count)
	if count != 2 {
		t.Fatalf("active device count = %d, want 2", count)
	}
}

func TestRecordDeviceAccessDoesNotMergeClashMetaAndroidAliasOutsideWindow(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:device_manager_android_alias_window?mode=memory&cache=shared"), &gorm.Config{})
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
		Username: "window-user",
		Email:    "window@example.com",
		Password: "hashed-password",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	subscription := models.Subscription{
		UserID:          user.ID,
		SubscriptionURL: "android-alias-window-subscription-url",
		ExpireTime:      time.Now().Add(24 * time.Hour),
		IsActive:        true,
		Status:          "active",
		DeviceLimit:     3,
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	dm := NewDeviceManager()
	first, err := dm.RecordDeviceAccess(subscription.ID, user.ID, clashMetaAndroidUA, "203.0.113.40", "clash")
	if err != nil {
		t.Fatalf("record Clash Meta Android UA: %v", err)
	}

	oldTime := time.Now().Add(-10 * time.Minute)
	if err := db.Model(&models.Device{}).Where("id = ?", first.ID).Updates(map[string]interface{}{
		"last_access": oldTime,
		"last_seen":   oldTime,
	}).Error; err != nil {
		t.Fatalf("age Android device: %v", err)
	}

	second, err := dm.RecordDeviceAccess(subscription.ID, user.ID, genericClashWindowsUA, "203.0.113.40", "clash")
	if err != nil {
		t.Fatalf("record generic Clash Windows UA: %v", err)
	}
	if second.ID == first.ID {
		t.Fatalf("generic Clash Windows UA merged into stale Clash Meta Android device ID %d", first.ID)
	}

	var count int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscription.ID, true).Count(&count)
	if count != 2 {
		t.Fatalf("active device count = %d, want 2", count)
	}
}

func TestRecordDeviceAccessDeactivatesExistingClashMetaAndroidAliasDuplicate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:device_manager_android_alias_cleanup?mode=memory&cache=shared"), &gorm.Config{})
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
		Username: "cleanup-user",
		Email:    "cleanup@example.com",
		Password: "hashed-password",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	subscription := models.Subscription{
		UserID:          user.ID,
		SubscriptionURL: "android-alias-cleanup-subscription-url",
		ExpireTime:      time.Now().Add(24 * time.Hour),
		IsActive:        true,
		Status:          "active",
		DeviceLimit:     3,
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	dm := NewDeviceManager()
	androidHash := dm.GenerateDeviceHash(clashMetaAndroidUA, "203.0.113.50", "")
	windowsHash := dm.GenerateDeviceHash(genericClashWindowsUA, "203.0.113.50", "")
	now := time.Now()
	userID := int64(user.ID)
	androidName := "Clash Meta - Pixel 7 - Android 13 - v2.10.4"
	androidSoftware := "Clash Meta"
	androidOS := "Android"
	windowsName := "Clash - Windows - v0.19.23"
	windowsSoftware := "Clash"
	windowsOS := "Windows"
	ip := "203.0.113.50"
	androidUA := clashMetaAndroidUA
	windowsUA := genericClashWindowsUA

	androidDevice := models.Device{
		UserID:            &userID,
		SubscriptionID:    subscription.ID,
		DeviceFingerprint: androidHash,
		DeviceHash:        &androidHash,
		DeviceUA:          &androidUA,
		DeviceName:        &androidName,
		IPAddress:         &ip,
		UserAgent:         &androidUA,
		SoftwareName:      &androidSoftware,
		OSName:            &androidOS,
		IsActive:          true,
		IsAllowed:         true,
		FirstSeen:         &now,
		LastAccess:        now,
		LastSeen:          &now,
		AccessCount:       1,
	}
	if err := db.Create(&androidDevice).Error; err != nil {
		t.Fatalf("create Android device: %v", err)
	}

	windowsDevice := models.Device{
		UserID:            &userID,
		SubscriptionID:    subscription.ID,
		DeviceFingerprint: windowsHash,
		DeviceHash:        &windowsHash,
		DeviceUA:          &windowsUA,
		DeviceName:        &windowsName,
		IPAddress:         &ip,
		UserAgent:         &windowsUA,
		SoftwareName:      &windowsSoftware,
		OSName:            &windowsOS,
		IsActive:          true,
		IsAllowed:         true,
		FirstSeen:         &now,
		LastAccess:        now,
		LastSeen:          &now,
		AccessCount:       1,
	}
	if err := db.Create(&windowsDevice).Error; err != nil {
		t.Fatalf("create generic Windows alias device: %v", err)
	}

	updated, err := dm.RecordDeviceAccess(subscription.ID, user.ID, clashMetaAndroidUA, ip, "clash")
	if err != nil {
		t.Fatalf("record Clash Meta Android UA: %v", err)
	}
	if updated.ID != androidDevice.ID {
		t.Fatalf("updated device ID = %d, want Android device ID %d", updated.ID, androidDevice.ID)
	}

	var reloadedWindows models.Device
	if err := db.First(&reloadedWindows, windowsDevice.ID).Error; err != nil {
		t.Fatalf("reload generic Windows alias device: %v", err)
	}
	if reloadedWindows.IsActive {
		t.Fatal("generic Windows alias duplicate remains active, want inactive")
	}

	var count int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscription.ID, true).Count(&count)
	if count != 1 {
		t.Fatalf("active device count = %d, want 1", count)
	}
}

package software_sync

import (
	"testing"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupSoftwareConfigDB 内存 SQLite 测试库（仅 system_configs 表），
// 并把 software_sync 依赖的包级 database.DB 指向它。
func setupSoftwareConfigDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.SystemConfig{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	prev := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = prev
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

func seedSoftwareConfig(t *testing.T, db *gorm.DB, key, value string) {
	t.Helper()
	if err := db.Create(&models.SystemConfig{Key: key, Value: value, Category: "software", Type: "text"}).Error; err != nil {
		t.Fatalf("写入初始配置失败: %v", err)
	}
}

// TestMoneyFlyDeclaresVersionKey MoneyFly 必须声明展示版本号键，
// 否则「下载链接自动跟版」但「页面版本号要人工改」，用户会看到按钮下载 2.2.19、页面写 v2.2.18。
func TestMoneyFlyDeclaresVersionKey(t *testing.T) {
	sw := FindSoftwareByConfigKey("moneyfly_windows_url")
	if sw == nil {
		t.Fatal("MoneyFly 未注册进同步目录")
	}
	if sw.VersionKey != "moneyfly_version" {
		t.Fatalf("MoneyFly 的 VersionKey = %q，期望 moneyfly_version", sw.VersionKey)
	}
}

// TestWriteVersionKeyUpdatesConfig 检出新版本时应更新展示版本号，且不改变分类。
func TestWriteVersionKeyUpdatesConfig(t *testing.T) {
	db := setupSoftwareConfigDB(t)
	seedSoftwareConfig(t, db, "moneyfly_version", "2.2.10")

	sw := FindSoftwareByConfigKey("moneyfly_windows_url")
	if err := writeVersionKeyIfNeeded(sw, "2.2.19"); err != nil {
		t.Fatalf("写入展示版本号失败: %v", err)
	}
	if got := loadSoftwareValue("moneyfly_version"); got != "2.2.19" {
		t.Fatalf("展示版本号 = %q，期望 2.2.19", got)
	}

	var row models.SystemConfig
	if err := db.Where("key = ?", "moneyfly_version").First(&row).Error; err != nil {
		t.Fatalf("查询配置失败: %v", err)
	}
	if row.Category != "software" {
		t.Fatalf("分类被改成了 %q，应保持 software", row.Category)
	}
	// 只应存在一行（更新而非重复插入）
	var count int64
	db.Model(&models.SystemConfig{}).Where("key = ?", "moneyfly_version").Count(&count)
	if count != 1 {
		t.Fatalf("moneyfly_version 出现 %d 行，期望 1 行", count)
	}
}

// TestWriteVersionKeyCreatesWhenMissing 首次检出（配置键不存在）时应新建该配置。
func TestWriteVersionKeyCreatesWhenMissing(t *testing.T) {
	db := setupSoftwareConfigDB(t)
	sw := FindSoftwareByConfigKey("moneyfly_windows_url")
	if err := writeVersionKeyIfNeeded(sw, "2.2.18"); err != nil {
		t.Fatalf("写入展示版本号失败: %v", err)
	}
	if got := loadSoftwareValue("moneyfly_version"); got != "2.2.18" {
		t.Fatalf("展示版本号 = %q，期望 2.2.18", got)
	}
	var count int64
	db.Model(&models.SystemConfig{}).Where("key = ? AND category = ?", "moneyfly_version", "software").Count(&count)
	if count != 1 {
		t.Fatalf("期望新建 1 行，实际 %d 行", count)
	}
}

// TestWriteVersionKeySkipsSoftwareWithoutKey 未声明 VersionKey 的软件不得写入任何配置。
func TestWriteVersionKeySkipsSoftwareWithoutKey(t *testing.T) {
	db := setupSoftwareConfigDB(t)
	sw := FindSoftwareByConfigKey("v2rayn_url")
	if sw == nil {
		t.Fatal("V2rayN 未注册进同步目录")
	}
	if sw.VersionKey != "" {
		t.Fatalf("V2rayN 不应声明 VersionKey，实际 %q", sw.VersionKey)
	}
	if err := writeVersionKeyIfNeeded(sw, "7.24.9"); err != nil {
		t.Fatalf("无 VersionKey 时不应报错: %v", err)
	}
	var count int64
	db.Model(&models.SystemConfig{}).Count(&count)
	if count != 0 {
		t.Fatalf("无 VersionKey 的软件不应写入任何配置，实际写入 %d 行", count)
	}
}

// TestWriteVersionKeyIdempotent 版本号未变化时不重复写库（避免每次同步都改配置）。
func TestWriteVersionKeyIdempotent(t *testing.T) {
	db := setupSoftwareConfigDB(t)
	seedSoftwareConfig(t, db, "moneyfly_version", "2.2.18")

	var before models.SystemConfig
	if err := db.Where("key = ?", "moneyfly_version").First(&before).Error; err != nil {
		t.Fatal(err)
	}
	sw := FindSoftwareByConfigKey("moneyfly_windows_url")
	if err := writeVersionKeyIfNeeded(sw, "2.2.18"); err != nil {
		t.Fatalf("同版本写入不应报错: %v", err)
	}
	var after models.SystemConfig
	if err := db.Where("key = ?", "moneyfly_version").First(&after).Error; err != nil {
		t.Fatal(err)
	}
	if !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("同版本应短路不写库，UpdatedAt 却从 %v 变为 %v", before.UpdatedAt, after.UpdatedAt)
	}
	if after.Value != "2.2.18" {
		t.Fatalf("值被改动: %q", after.Value)
	}
}

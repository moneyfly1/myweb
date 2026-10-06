package handlers

import (
	"testing"

	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupDeviceDeleteGateDB 内存 SQLite（含 system_configs 表）
func setupDeviceDeleteGateDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.SystemConfig{}); err != nil {
		t.Fatalf("迁移 system_configs 失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

// 「允许用户删除设备」开关的判定：
//   - 未配置 → 允许（保持既有行为，不能让存量部署突然禁止删除）
//   - true / false（大小写、空格容错）
func TestUserDeviceDeleteAllowed(t *testing.T) {
	db := setupDeviceDeleteGateDB(t)

	setValue := func(v string) {
		db.Where("key = ? AND category = ?", "allow_user_delete_device", "registration").
			Delete(&models.SystemConfig{})
		if v != "" {
			if err := db.Create(&models.SystemConfig{
				Key: "allow_user_delete_device", Value: v, Category: "registration",
			}).Error; err != nil {
				t.Fatalf("写入配置失败: %v", err)
			}
		}
		// 配置读取有 30 秒短缓存，改完必须失效，否则测到旧值
		utils.InvalidateAllSettingCache()
	}

	setValue("")
	if !UserDeviceDeleteAllowed(db) {
		t.Error("未配置该开关时应允许用户删除设备（保持既有行为）")
	}

	setValue("false")
	if UserDeviceDeleteAllowed(db) {
		t.Error("开关为 false 时应禁止用户删除设备")
	}

	setValue("FALSE")
	if UserDeviceDeleteAllowed(db) {
		t.Error("开关判定应对大小写不敏感（FALSE 同样禁止）")
	}

	setValue(" false ")
	if UserDeviceDeleteAllowed(db) {
		t.Error("开关判定应忽略首尾空格")
	}

	setValue("true")
	if !UserDeviceDeleteAllowed(db) {
		t.Error("开关为 true 时应允许用户删除设备")
	}

	// 异常值不应把功能锁死（只有明确的 false 才禁止）
	setValue("yes")
	if !UserDeviceDeleteAllowed(db) {
		t.Error("非 false 的异常值应按允许处理")
	}
}

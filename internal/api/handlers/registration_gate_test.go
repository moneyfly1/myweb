package handlers

import (
	"testing"

	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupRegistrationGateTestDB(t *testing.T) *gorm.DB {
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
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
		utils.InvalidateAllSettingCache()
	})
	return db
}

func setRegistrationEnabled(t *testing.T, db *gorm.DB, value string) {
	t.Helper()
	var conf models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "registration_enabled", "registration").
		FirstOrInit(&conf).Error; err != nil {
		t.Fatalf("查询配置失败: %v", err)
	}
	conf.Key = "registration_enabled"
	conf.Category = "registration"
	conf.Value = value
	if err := db.Save(&conf).Error; err != nil {
		t.Fatalf("保存配置失败: %v", err)
	}
	utils.InvalidateAllSettingCache() // 与配置更新接口一致的失效动作
}

// TestRegistrationEnabled 注册开关的三态：
// 配置缺失视为允许（与代码内默认配置一致），"false" 必须判定为禁用，
// 否则管理员关闭注册后仍可直接 POST /auth/register 建号。
func TestRegistrationEnabled(t *testing.T) {
	db := setupRegistrationGateTestDB(t)

	if !registrationEnabled(db) {
		t.Error("配置项缺失时应视为允许注册")
	}

	setRegistrationEnabled(t, db, "false")
	if registrationEnabled(db) {
		t.Error("registration_enabled=false 时必须判定为已禁用")
	}

	setRegistrationEnabled(t, db, "true")
	if !registrationEnabled(db) {
		t.Error("registration_enabled=true 时应判定为允许注册")
	}
}

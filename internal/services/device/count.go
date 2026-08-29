package device

import (
	"cboard-go/internal/models"

	"gorm.io/gorm"
)

// CountActiveDevices 统计指定订阅下处于活跃状态的设备数。
// 统一各处的 "subscription_id = ? AND is_active = true" 计数查询。
func CountActiveDevices(db *gorm.DB, subscriptionID uint) (int64, error) {
	var count int64
	err := db.Model(&models.Device{}).
		Where("subscription_id = ? AND is_active = ?", subscriptionID, true).
		Count(&count).Error
	return count, err
}

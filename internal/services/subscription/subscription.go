package subscription

import (
	"fmt"
	"strconv"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

// SubscriptionService 订阅服务
type SubscriptionService struct {
	db *gorm.DB
}

// NewSubscriptionService 创建订阅服务
func NewSubscriptionService() *SubscriptionService {
	db := database.GetDB()
	if db == nil {
		// 如果数据库未初始化，记录错误但不panic
		// 在实际使用时会返回错误
		if utils.AppLogger != nil {
			utils.AppLogger.Error("SubscriptionService: 数据库未初始化")
		}
	}
	return &SubscriptionService{
		db: db,
	}
}

// GetByUserID 根据用户ID获取订阅
func (s *SubscriptionService) GetByUserID(userID uint) (*models.Subscription, error) {
	if s.db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	var subscription models.Subscription
	if err := s.db.Where("user_id = ?", userID).First(&subscription).Error; err != nil {
		return nil, err
	}
	return &subscription, nil
}

// GetBySubscriptionURL 根据订阅URL获取订阅
func (s *SubscriptionService) GetBySubscriptionURL(url string) (*models.Subscription, error) {
	if s.db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	var subscription models.Subscription
	if err := s.db.Where("subscription_url = ?", url).First(&subscription).Error; err != nil {
		return nil, err
	}
	return &subscription, nil
}

// CreateSubscription 创建订阅
func (s *SubscriptionService) CreateSubscription(userID uint, packageID uint, durationDays int) (*models.Subscription, error) {
	if s.db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	subscriptionURL := utils.GenerateSubscriptionURL()
	now := utils.GetBeijingTime()
	expireTime := now.AddDate(0, 0, durationDays)

	// 从系统设置获取默认设备数
	deviceLimit := getDefaultDeviceLimit(s.db)

	packageIDPtr := int64(packageID)
	subscription := models.Subscription{
		UserID:          userID,
		PackageID:       &packageIDPtr,
		SubscriptionURL: subscriptionURL,
		DeviceLimit:     deviceLimit,
		CurrentDevices:  0,
		IsActive:        true,
		Status:          "active",
		ExpireTime:      expireTime,
	}

	if err := s.db.Create(&subscription).Error; err != nil {
		return nil, err
	}

	return &subscription, nil
}

// getDefaultDeviceLimit 从系统设置中获取默认设备数
func getDefaultDeviceLimit(db *gorm.DB) int {
	// 默认值
	deviceLimit := 3

	// 从数据库读取配置（优先从 registration category 读取，如果没有则从 general 读取）
	var deviceLimitConfig models.SystemConfig
	// 先尝试从 registration category 读取
	if err := db.Where("key = ? AND category = ?", "default_subscription_device_limit", "registration").First(&deviceLimitConfig).Error; err != nil {
		// 如果 registration 中没有，尝试从 general category 读取
		if err := db.Where("key = ? AND category = ?", "default_subscription_device_limit", "general").First(&deviceLimitConfig).Error; err == nil {
			if limit, err := strconv.Atoi(deviceLimitConfig.Value); err == nil && limit >= 0 {
				deviceLimit = limit
			}
		}
	} else {
		if limit, err := strconv.Atoi(deviceLimitConfig.Value); err == nil && limit >= 0 {
			deviceLimit = limit
		}
	}

	return deviceLimit
}

// UpdateExpireTime 更新过期时间
func (s *SubscriptionService) UpdateExpireTime(subscriptionID uint, days int) error {
	if s.db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	var subscription models.Subscription
	if err := s.db.First(&subscription, subscriptionID).Error; err != nil {
		return err
	}

	now := utils.GetBeijingTime()
	baseTime := subscription.ExpireTime
	if baseTime.Before(now) {
		baseTime = now
	}

	subscription.ExpireTime = baseTime.AddDate(0, 0, days)
	return s.db.Save(&subscription).Error
}

// CheckExpired 检查并更新过期订阅
func (s *SubscriptionService) CheckExpired() error {
	if s.db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	now := utils.GetBeijingTime()
	return s.db.Model(&models.Subscription{}).
		Where("expire_time < ? AND status = ?", now, "active").
		Updates(map[string]interface{}{
			"status":    "expired",
			"is_active": false,
		}).Error
}

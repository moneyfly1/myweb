package subscription

import (
	"fmt"
	"strconv"
	"time"

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
	// 使用 UTC 时间进行计算，避免时区问题
	// SQLite 存储时间时可能会转换为 UTC，所以我们在 UTC 时区下计算过期时间
	nowUTC := time.Now().UTC()

	// 如果 durationDays 为 0 或负数，设置为默认值 30 天
	if durationDays <= 0 {
		durationDays = 30
	}

	// 在 UTC 时区下计算过期时间
	expireTime := nowUTC.AddDate(0, 0, durationDays)

	// 调试日志：记录订阅创建信息
	if utils.AppLogger != nil {
		utils.AppLogger.Info("创建订阅 - UserID=%d, PackageID=%d, DurationDays=%d, Now(UTC)=%s, ExpireTime(UTC)=%s",
			userID, packageID, durationDays,
			nowUTC.Format("2006-01-02 15:04:05 MST"),
			expireTime.Format("2006-01-02 15:04:05 MST"))
	}

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

	// 调试日志：记录保存后的订阅信息
	if utils.AppLogger != nil {
		utils.AppLogger.Info("订阅创建成功 - SubscriptionID=%d, ExpireTime(保存后)=%s (时区:%s)",
			subscription.ID,
			subscription.ExpireTime.Format("2006-01-02 15:04:05 MST"),
			subscription.ExpireTime.Location().String())
	}

	return &subscription, nil
}

// getDefaultDeviceLimit 从系统设置中获取默认设备数
// 如果后台没有设置配置，返回0（设备数为0）
// 只有在后台明确设置了配置值时才使用配置的值
func getDefaultDeviceLimit(db *gorm.DB) int {
	// 默认值设为0（除非后台明确配置了值）
	deviceLimit := 0

	// 从数据库读取配置（优先从 registration category 读取，如果没有则从 general 读取）
	var deviceLimitConfig models.SystemConfig
	// 先尝试从 registration category 读取
	if err := db.Where("key = ? AND category = ?", "default_subscription_device_limit", "registration").First(&deviceLimitConfig).Error; err != nil {
		// 如果 registration 中没有，尝试从 general category 读取
		if err := db.Where("key = ? AND category = ?", "default_subscription_device_limit", "general").First(&deviceLimitConfig).Error; err == nil {
			// 配置存在，尝试解析值
			if deviceLimitConfig.Value != "" {
				if limit, err := strconv.Atoi(deviceLimitConfig.Value); err == nil && limit >= 0 {
					deviceLimit = limit
				}
			}
		}
		// 如果两个地方都没有配置，保持默认值0
	} else {
		// registration category 中有配置
		if deviceLimitConfig.Value != "" {
			if limit, err := strconv.Atoi(deviceLimitConfig.Value); err == nil && limit >= 0 {
				deviceLimit = limit
			}
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
	// 确保从数据库读取的时间转换为北京时间
	baseTime := utils.ToBeijingTime(subscription.ExpireTime)
	if baseTime.Before(now) {
		baseTime = now
	}

	// 计算新的过期时间并确保转换为北京时间
	newExpireTime := baseTime.AddDate(0, 0, days)
	subscription.ExpireTime = utils.ToBeijingTime(newExpireTime)
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

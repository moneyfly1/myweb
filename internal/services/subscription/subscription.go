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

type SubscriptionService struct {
	db *gorm.DB
}

func NewSubscriptionService() *SubscriptionService {
	db := database.GetDB()
	if db == nil {
		if utils.AppLogger != nil {
			utils.AppLogger.Error("SubscriptionService: 数据库未初始化")
		}
	}
	return &SubscriptionService{
		db: db,
	}
}

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

func (s *SubscriptionService) CreateSubscription(userID uint, packageID uint, durationDays int) (*models.Subscription, error) {
	if s.db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	subscriptionURL := utils.GenerateSubscriptionURL()
	nowUTC := time.Now().UTC()

	if durationDays <= 0 {
		durationDays = 30
	}

	expireTime := nowUTC.AddDate(0, 0, durationDays)

	if utils.AppLogger != nil {
		utils.AppLogger.Info("创建订阅 - UserID=%d, PackageID=%d, DurationDays=%d, Now(UTC)=%s, ExpireTime(UTC)=%s",
			userID, packageID, durationDays,
			nowUTC.Format("2006-01-02 15:04:05 MST"),
			expireTime.Format("2006-01-02 15:04:05 MST"))
	}

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

	if utils.AppLogger != nil {
		utils.AppLogger.Info("订阅创建成功 - SubscriptionID=%d, ExpireTime(保存后)=%s (时区:%s)",
			subscription.ID,
			subscription.ExpireTime.Format("2006-01-02 15:04:05 MST"),
			subscription.ExpireTime.Location().String())
	}

	return &subscription, nil
}

func getDefaultDeviceLimit(db *gorm.DB) int {
	deviceLimit := 0

	var deviceLimitConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "default_subscription_device_limit", "registration").First(&deviceLimitConfig).Error; err != nil {
		if err := db.Where("key = ? AND category = ?", "default_subscription_device_limit", "general").First(&deviceLimitConfig).Error; err == nil {
			if deviceLimitConfig.Value != "" {
				if limit, err := strconv.Atoi(deviceLimitConfig.Value); err == nil && limit >= 0 {
					deviceLimit = limit
				}
			}
		}
	} else {
		if deviceLimitConfig.Value != "" {
			if limit, err := strconv.Atoi(deviceLimitConfig.Value); err == nil && limit >= 0 {
				deviceLimit = limit
			}
		}
	}

	return deviceLimit
}

func (s *SubscriptionService) UpdateExpireTime(subscriptionID uint, days int) error {
	if s.db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	var subscription models.Subscription
	if err := s.db.First(&subscription, subscriptionID).Error; err != nil {
		return err
	}

	now := utils.GetBeijingTime()
	baseTime := utils.ToBeijingTime(subscription.ExpireTime)
	if baseTime.Before(now) {
		baseTime = now
	}

	newExpireTime := baseTime.AddDate(0, 0, days)
	subscription.ExpireTime = utils.ToBeijingTime(newExpireTime)
	return s.db.Save(&subscription).Error
}

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

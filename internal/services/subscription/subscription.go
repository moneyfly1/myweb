package subscription

import (
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
	return &SubscriptionService{
		db: database.GetDB(),
	}
}

// GetByUserID 根据用户ID获取订阅
func (s *SubscriptionService) GetByUserID(userID uint) (*models.Subscription, error) {
	var subscription models.Subscription
	if err := s.db.Where("user_id = ?", userID).First(&subscription).Error; err != nil {
		return nil, err
	}
	return &subscription, nil
}

// GetBySubscriptionURL 根据订阅URL获取订阅
func (s *SubscriptionService) GetBySubscriptionURL(url string) (*models.Subscription, error) {
	var subscription models.Subscription
	if err := s.db.Where("subscription_url = ?", url).First(&subscription).Error; err != nil {
		return nil, err
	}
	return &subscription, nil
}

// CreateSubscription 创建订阅
func (s *SubscriptionService) CreateSubscription(userID uint, packageID uint, durationDays int) (*models.Subscription, error) {
	subscriptionURL := utils.GenerateSubscriptionURL()
	now := utils.GetBeijingTime()
	expireTime := now.AddDate(0, 0, durationDays)

	packageIDPtr := int64(packageID)
	subscription := models.Subscription{
		UserID:          userID,
		PackageID:       &packageIDPtr,
		SubscriptionURL: subscriptionURL,
		DeviceLimit:     3,
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

// UpdateExpireTime 更新过期时间
func (s *SubscriptionService) UpdateExpireTime(subscriptionID uint, days int) error {
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
	now := utils.GetBeijingTime()
	return s.db.Model(&models.Subscription{}).
		Where("expire_time < ? AND status = ?", now, "active").
		Updates(map[string]interface{}{
			"status":    "expired",
			"is_active": false,
		}).Error
}

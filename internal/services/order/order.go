package order

import (
	"database/sql"
	"fmt"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

// OrderService 订单服务
type OrderService struct {
	db *gorm.DB
}

// NewOrderService 创建订单服务
func NewOrderService() *OrderService {
	return &OrderService{
		db: database.GetDB(),
	}
}

// ProcessPaidOrder 处理已支付订单的后续逻辑（开通/续费订阅、更新消费、升级等级）
// 注意：调用此方法前，订单状态应已更新为 paid
func (s *OrderService) ProcessPaidOrder(order *models.Order) (*models.Subscription, error) {
	if order.Status != "paid" {
		return nil, fmt.Errorf("订单状态未支付")
	}

	// 获取用户信息
	var user models.User
	if err := s.db.First(&user, order.UserID).Error; err != nil {
		return nil, fmt.Errorf("用户不存在: %v", err)
	}

	// 获取套餐信息
	var pkg models.Package
	if err := s.db.First(&pkg, order.PackageID).Error; err != nil {
		return nil, fmt.Errorf("套餐不存在: %v", err)
	}

	now := utils.GetBeijingTime()

	// 1. 更新或创建订阅
	var subscription models.Subscription
	if err := s.db.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		// 创建新订阅
		subscriptionURL := utils.GenerateSubscriptionURL()
		expireTime := now.AddDate(0, 0, pkg.DurationDays)
		pkgID := int64(pkg.ID)
		subscription = models.Subscription{
			UserID:          user.ID,
			PackageID:       &pkgID,
			SubscriptionURL: subscriptionURL,
			DeviceLimit:     pkg.DeviceLimit,
			CurrentDevices:  0,
			IsActive:        true,
			Status:          "active",
			ExpireTime:      expireTime,
		}
		if err := s.db.Create(&subscription).Error; err != nil {
			return nil, fmt.Errorf("创建订阅失败: %v", err)
		}
		if utils.AppLogger != nil {
			utils.AppLogger.Info("ProcessPaidOrder: ✅ 创建新订阅成功 - user_id=%d, package_id=%d, device_limit=%d, duration_days=%d, expire_time=%s",
				user.ID, pkg.ID, pkg.DeviceLimit, pkg.DurationDays, expireTime.Format("2006-01-02 15:04:05"))
		}
	} else {
		// 续费：累加时间
		oldExpireTime := subscription.ExpireTime
		if subscription.ExpireTime.Before(now) {
			subscription.ExpireTime = now.AddDate(0, 0, pkg.DurationDays)
		} else {
			subscription.ExpireTime = subscription.ExpireTime.AddDate(0, 0, pkg.DurationDays)
		}
		oldDeviceLimit := subscription.DeviceLimit
		subscription.DeviceLimit = pkg.DeviceLimit
		subscription.IsActive = true
		subscription.Status = "active"
		pkgID := int64(pkg.ID)
		subscription.PackageID = &pkgID
		
		if err := s.db.Save(&subscription).Error; err != nil {
			return nil, fmt.Errorf("更新订阅失败: %v", err)
		}
		if utils.AppLogger != nil {
			utils.AppLogger.Info("ProcessPaidOrder: ✅ 更新订阅成功 - user_id=%d, package_id=%d, device_limit: %d->%d, expire_time: %s->%s",
				user.ID, pkg.ID, oldDeviceLimit, pkg.DeviceLimit, oldExpireTime.Format("2006-01-02 15:04:05"), subscription.ExpireTime.Format("2006-01-02 15:04:05"))
		}
	}

	// 2. 更新用户累计消费
	paidAmount := order.Amount
	if order.FinalAmount.Valid {
		paidAmount = order.FinalAmount.Float64
	}
	user.TotalConsumption += paidAmount
	if err := s.db.Save(&user).Error; err != nil {
		return nil, fmt.Errorf("更新用户累计消费失败: %v", err)
	}

	// 3. 检查并更新用户等级
	var userLevels []models.UserLevel
	if err := s.db.Where("is_active = ?", true).Order("level_order ASC").Find(&userLevels).Error; err == nil {
		for _, level := range userLevels {
			if user.TotalConsumption >= level.MinConsumption {
				// 检查是否需要升级
				if !user.UserLevelID.Valid || user.UserLevelID.Int64 != int64(level.ID) {
					// 需要升级
					var currentLevel models.UserLevel
					shouldUpgrade := true
					if user.UserLevelID.Valid {
						if err := s.db.First(&currentLevel, user.UserLevelID.Int64).Error; err == nil {
							// 如果当前等级更高（level_order 更小），不降级
							if currentLevel.LevelOrder < level.LevelOrder {
								shouldUpgrade = false
							}
						}
					}
					if shouldUpgrade {
						user.UserLevelID = sql.NullInt64{Int64: int64(level.ID), Valid: true}
						if err := s.db.Save(&user).Error; err != nil {
							if utils.AppLogger != nil {
								utils.AppLogger.Error("更新用户等级失败: %v", err)
							}
						}
					}
				}
			}
		}
	}

	return &subscription, nil
}

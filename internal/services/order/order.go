package order

import (
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

// ProcessPaidOrder 处理已支付订单
func (s *OrderService) ProcessPaidOrder(orderID uint) error {
	var order models.Order
	if err := s.db.First(&order, orderID).Error; err != nil {
		return err
	}

	if order.Status != "paid" {
		return nil // 订单已处理或状态不正确
	}

	// 获取套餐信息
	var pkg models.Package
	if err := s.db.First(&pkg, order.PackageID).Error; err != nil {
		return err
	}

	// 获取用户订阅
	var subscription models.Subscription
	s.db.Where("user_id = ?", order.UserID).First(&subscription)

	now := utils.GetBeijingTime()

	// 如果已有订阅，续费；否则创建新订阅
	if subscription.ID > 0 {
		// 续费：累加时间
		baseTime := subscription.ExpireTime
		if baseTime.Before(now) {
			baseTime = now
		}
		subscription.ExpireTime = baseTime.AddDate(0, 0, pkg.DurationDays)
		subscription.DeviceLimit = pkg.DeviceLimit
		subscription.IsActive = true
		subscription.Status = "active"
		s.db.Save(&subscription)
	} else {
		// 创建新订阅
		packageID := int64(pkg.ID)
		subscription = models.Subscription{
			UserID:          order.UserID,
			PackageID:       &packageID,
			SubscriptionURL: utils.GenerateSubscriptionURL(),
			DeviceLimit:     pkg.DeviceLimit,
			CurrentDevices:  0,
			IsActive:        true,
			Status:          "active",
			ExpireTime:      now.AddDate(0, 0, pkg.DurationDays),
		}
		s.db.Create(&subscription)
	}

	// 更新用户累计消费
	if order.FinalAmount.Valid {
		s.db.Model(&models.User{}).Where("id = ?", order.UserID).
			UpdateColumn("total_consumption", gorm.Expr("total_consumption + ?", order.FinalAmount.Float64))
	} else {
		s.db.Model(&models.User{}).Where("id = ?", order.UserID).
			UpdateColumn("total_consumption", gorm.Expr("total_consumption + ?", order.Amount))
	}

	return nil
}

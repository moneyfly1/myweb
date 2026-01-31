package order

import (
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/services/payment"
	"cboard-go/internal/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type CreateOrderParams struct {
	PackageID      uint    `json:"package_id"`
	CouponCode     string  `json:"coupon_code"`
	PaymentMethod  string  `json:"payment_method"`
	UseBalance     bool    `json:"use_balance"`
	BalanceAmount  float64 `json:"balance_amount"`
	DurationMonths int     `json:"duration_months"`
	UserAgent      string  `json:"-"`
	ClientIP       string  `json:"-"`
}

type OrderService struct {
	db *gorm.DB
}

func NewOrderService() *OrderService {
	return &OrderService{
		db: database.GetDB(),
	}
}

func (s *OrderService) CreateOrder(userID uint, params CreateOrderParams) (*models.Order, string, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, "", fmt.Errorf("用户不存在")
	}

	var pkg models.Package
	if err := s.db.First(&pkg, params.PackageID).Error; err != nil {
		return nil, "", fmt.Errorf("套餐不存在")
	}

	if !pkg.IsActive {
		return nil, "", fmt.Errorf("套餐已停用")
	}

	durationMonths := params.DurationMonths
	if durationMonths <= 0 {
		durationMonths = 1
	}
	if durationMonths > 60 {
		durationMonths = 60
	}

	baseAmount := pkg.Price * float64(durationMonths)
	levelDiscountAmount := 0.0
	couponDiscountAmount := 0.0
	finalAmount := baseAmount

	if user.UserLevelID.Valid {
		var lvl models.UserLevel
		if err := s.db.First(&lvl, user.UserLevelID.Int64).Error; err == nil {
			if lvl.DiscountRate > 0 && lvl.DiscountRate < 1.0 {
				levelDiscountAmount = baseAmount * (1.0 - lvl.DiscountRate)
				finalAmount = baseAmount * lvl.DiscountRate
			}
		}
	}

	var couponID *int64
	if params.CouponCode != "" {
		var coupon models.Coupon
		if err := s.db.Where("code = ? AND status = ?", params.CouponCode, "active").First(&coupon).Error; err == nil {
			now := utils.GetBeijingTime()
			if (now.After(coupon.ValidFrom) || now.Equal(coupon.ValidFrom)) && (now.Before(coupon.ValidUntil) || now.Equal(coupon.ValidUntil)) {
				if !coupon.MinAmount.Valid || finalAmount >= coupon.MinAmount.Float64 {
					if coupon.Type == "discount" {
						couponDiscountAmount = finalAmount * (coupon.DiscountValue / 100)
						if coupon.MaxDiscount.Valid && couponDiscountAmount > coupon.MaxDiscount.Float64 {
							couponDiscountAmount = coupon.MaxDiscount.Float64
						}
					} else if coupon.Type == "fixed" {
						couponDiscountAmount = coupon.DiscountValue
						if couponDiscountAmount > finalAmount {
							couponDiscountAmount = finalAmount
						}
					}
					finalAmount = finalAmount - couponDiscountAmount
					cID := int64(coupon.ID)
					couponID = &cID
				}
			}
		}
	}

	totalDiscountAmount := levelDiscountAmount + couponDiscountAmount
	balanceUsed := 0.0

	if params.UseBalance && params.BalanceAmount > 0 {
		if user.Balance < params.BalanceAmount {
			return nil, "", fmt.Errorf("余额不足")
		}
		if params.BalanceAmount > finalAmount {
			params.BalanceAmount = finalAmount
		}
		balanceUsed = params.BalanceAmount
		finalAmount -= balanceUsed

		user.Balance -= balanceUsed
		if err := s.db.Save(&user).Error; err != nil {
			return nil, "", fmt.Errorf("扣除余额失败")
		}
	}

	if finalAmount <= 0.01 {
		finalAmount = 0
	}

	orderNo, err := utils.GenerateOrderNo(s.db)
	if err != nil {
		return nil, "", fmt.Errorf("生成订单号失败: %v", err)
	}
	now := utils.GetBeijingTime()

	extraDataMap := map[string]interface{}{
		"duration_months": durationMonths,
	}
	if balanceUsed > 0 {
		extraDataMap["balance_used"] = balanceUsed
	}
	extraDataJSON, _ := json.Marshal(extraDataMap)

	order := models.Order{
		OrderNo:        orderNo,
		UserID:         user.ID,
		PackageID:      pkg.ID,
		Amount:         baseAmount,
		Status:         "pending",
		DiscountAmount: database.NullFloat64(totalDiscountAmount),
		FinalAmount:    database.NullFloat64(balanceUsed + finalAmount), // 记录实际价值（余额+支付）
		ExtraData:      database.NullString(string(extraDataJSON)),
		CreatedAt:      now, // 显式设置创建时间为北京时间
	}

	if finalAmount == 0 {
		order.Status = "paid"
		order.PaymentTime = database.NullTime(utils.GetBeijingTime())
		order.PaymentMethodName = database.NullString("余额支付")
	} else {
		methodName := params.PaymentMethod
		if balanceUsed > 0 {
			methodName = fmt.Sprintf("余额支付(%.2f元)+%s", balanceUsed, params.PaymentMethod)
		}
		order.PaymentMethodName = database.NullString(methodName)
	}

	if couponID != nil {
		order.CouponID = database.NullInt64(*couponID)
	}

	if err := s.db.Create(&order).Error; err != nil {
		return nil, "", fmt.Errorf("创建订单失败: %v", err)
	}

	if order.Status == "paid" {
		if _, err := s.ProcessPaidOrder(&order); err != nil {
			utils.LogError("CreateOrder: process paid order failed", err, nil)
		}

		go s.sendPaymentSuccessEmail(&user, &order, &pkg, balanceUsed+finalAmount, "余额支付")
		return &order, "", nil
	}

	var paymentURL string
	if params.PaymentMethod != "" && params.PaymentMethod != "balance" {
		utils.LogInfo("CreateOrder: 开始生成支付链接 - payment_method=%s, order_no=%s, amount=%.2f",
			params.PaymentMethod, order.OrderNo, finalAmount)
		url, err := s.generatePaymentURLWithUA(&order, params.PaymentMethod, finalAmount, params.UserAgent)
		if err != nil {
			utils.LogError("CreateOrder: 生成支付链接失败", err, map[string]interface{}{
				"payment_method": params.PaymentMethod,
				"order_no":       order.OrderNo,
			})
			return &order, "", fmt.Errorf("生成支付链接失败: %v", err)
		}
		paymentURL = url
		utils.LogInfo("CreateOrder: 支付链接生成成功 - payment_method=%s, order_no=%s, payment_url=%s",
			params.PaymentMethod, order.OrderNo, paymentURL)
	}

	return &order, paymentURL, nil
}

func (s *OrderService) generatePaymentURL(order *models.Order, payType string, amount float64) (string, error) {
	return s.generatePaymentURLWithUA(order, payType, amount, "")
}

func (s *OrderService) generatePaymentURLWithUA(order *models.Order, payType string, amount float64, userAgent string) (string, error) {
	var paymentConfig models.PaymentConfig

	if strings.HasPrefix(payType, "yipay_") {
		if err := s.db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", "yipay", 1).Order("sort_order ASC").First(&paymentConfig).Error; err == nil {
		} else {
			if err := s.db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", payType, 1).Order("sort_order ASC").First(&paymentConfig).Error; err != nil {
				return "", fmt.Errorf("未找到启用的支付配置")
			}
		}
	} else {
		if err := s.db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", payType, 1).Order("sort_order ASC").First(&paymentConfig).Error; err != nil {
			return "", fmt.Errorf("未找到启用的支付配置")
		}
	}

	transaction := models.PaymentTransaction{
		OrderID:         order.ID,
		UserID:          order.UserID,
		PaymentMethodID: paymentConfig.ID,
		Amount:          int(amount * 100),
		Currency:        "CNY",
		Status:          "pending",
	}
	s.db.Create(&transaction)

	switch paymentConfig.PayType {
	case "alipay":
		svc, err := payment.NewAlipayService(&paymentConfig)
		if err != nil {
			return "", err
		}
		return svc.CreatePayment(order, amount)
	case "wechat":
		svc, err := payment.NewWechatService(&paymentConfig)
		if err != nil {
			return "", err
		}
		return svc.CreatePayment(order, amount)
	case "paypal":
		svc, err := payment.NewPayPalService(&paymentConfig)
		if err != nil {
			return "", err
		}
		return svc.CreatePayment(order, amount)
	case "applepay":
		svc, err := payment.NewApplePayService(&paymentConfig)
		if err != nil {
			return "", err
		}
		return svc.CreatePayment(order, amount)
	case "yipay", "yipay_alipay", "yipay_wxpay", "yipay_qqpay":
		svc, err := payment.NewYipayService(&paymentConfig)
		if err != nil {
			return "", err
		}
		paymentType := extractYipayType(payType)
		utils.LogInfo("易支付生成支付链接: payType=%s, extracted_paymentType=%s, order_no=%s", payType, paymentType, order.OrderNo)
		if userAgent != "" {
			utils.LogInfo("易支付使用UserAgent: %s", userAgent)
			return svc.CreatePaymentWithDevice(order, amount, paymentType, userAgent)
		}
		return svc.CreatePayment(order, amount, paymentType)
	default:
		return "", fmt.Errorf("不支持的支付方式: %s", paymentConfig.PayType)
	}
}

func extractYipayType(payType string) string {
	if strings.HasPrefix(payType, "yipay_") {
		return strings.TrimPrefix(payType, "yipay_")
	}
	return "alipay"
}

func (s *OrderService) sendPaymentSuccessEmail(user *models.User, order *models.Order, pkg *models.Package, amount float64, paymentMethod string) {
	emailService := email.NewEmailService()
	templateBuilder := email.NewEmailTemplateBuilder()
	paymentTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")

	content := templateBuilder.GetPaymentSuccessTemplate(
		user.Username,
		order.OrderNo,
		pkg.Name,
		amount,
		paymentMethod,
		paymentTime,
	)
	_ = emailService.QueueEmail(user.Email, "支付成功通知", content, "payment_success")
}

func (s *OrderService) ProcessPaidOrder(order *models.Order) (*models.Subscription, error) {
	if order.Status != "paid" {
		return nil, fmt.Errorf("订单状态未支付")
	}

	var user models.User
	if err := s.db.First(&user, order.UserID).Error; err != nil {
		return nil, fmt.Errorf("用户不存在: %v", err)
	}

	paidAmount := order.Amount
	if order.FinalAmount.Valid {
		paidAmount = order.FinalAmount.Float64
	}

	user.TotalConsumption += order.Amount
	if err := s.db.Save(&user).Error; err != nil {
		return nil, fmt.Errorf("更新用户累计消费失败: %v", err)
	}

	s.updateUserLevel(&user)

	s.processInviteRewards(order, paidAmount)

	if order.PackageID > 0 {
		return s.processPackageOrder(order, &user)
	} else {
		return s.processDeviceUpgradeOrder(order, &user)
	}
}

func (s *OrderService) processPackageOrder(order *models.Order, user *models.User) (*models.Subscription, error) {
	var pkg models.Package
	if err := s.db.First(&pkg, order.PackageID).Error; err != nil {
		return nil, fmt.Errorf("套餐不存在: %v", err)
	}

	durationMonths := 1
	if order.ExtraData.Valid && order.ExtraData.String != "" {
		var extraData map[string]interface{}
		if err := json.Unmarshal([]byte(order.ExtraData.String), &extraData); err == nil {
			if months, ok := extraData["duration_months"].(float64); ok {
				durationMonths = int(months)
			}
		}
	}
	if durationMonths <= 0 {
		durationMonths = 1
	}
	if durationMonths > 60 {
		durationMonths = 60
	}

	totalDurationDays := pkg.DurationDays * durationMonths

	now := utils.GetBeijingTime()

	var subscription models.Subscription
	if err := s.db.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		subscriptionURL := utils.GenerateSubscriptionURL()
		expireTime := now.AddDate(0, 0, totalDurationDays)
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
			utils.AppLogger.Info("ProcessPaidOrder: ✅ 创建新订阅成功 - user_id=%d, package_id=%d, device_limit=%d, duration_months=%d, duration_days=%d, expire_time=%s",
				user.ID, pkg.ID, pkg.DeviceLimit, durationMonths, totalDurationDays, expireTime.Format("2006-01-02 15:04:05"))
		}

		go func() {
			notificationService := notification.NewNotificationService()
			createTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
			_ = notificationService.SendAdminNotification("subscription_created", map[string]interface{}{
				"username":        user.Username,
				"email":           user.Email,
				"package_name":    pkg.Name,
				"device_limit":    pkg.DeviceLimit,
				"duration_months": durationMonths,
				"duration_days":   totalDurationDays,
				"expire_time":     expireTime.Format("2006-01-02 15:04:05"),
				"create_time":     createTime,
			})
		}()
	} else {
		oldExpireTime := subscription.ExpireTime
		if subscription.ExpireTime.Before(now) {
			subscription.ExpireTime = now.AddDate(0, 0, totalDurationDays)
		} else {
			subscription.ExpireTime = subscription.ExpireTime.AddDate(0, 0, totalDurationDays)
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
			utils.AppLogger.Info("ProcessPaidOrder: ✅ 更新订阅成功 - user_id=%d, package_id=%d, device_limit: %d->%d, duration_months=%d, duration_days=%d, expire_time: %s->%s",
				user.ID, pkg.ID, oldDeviceLimit, pkg.DeviceLimit, durationMonths, totalDurationDays, oldExpireTime.Format("2006-01-02 15:04:05"), subscription.ExpireTime.Format("2006-01-02 15:04:05"))
		}
	}

	return &subscription, nil
}

func (s *OrderService) processDeviceUpgradeOrder(order *models.Order, user *models.User) (*models.Subscription, error) {
	var additionalDevices int
	var additionalDays int

	if order.ExtraData.Valid && order.ExtraData.String != "" {
		var extraData map[string]interface{}
		if err := json.Unmarshal([]byte(order.ExtraData.String), &extraData); err == nil {
			if extraData["type"] == "device_upgrade" {
				if devices, ok := extraData["additional_devices"].(float64); ok {
					additionalDevices = int(devices)
				}
				if days, ok := extraData["additional_days"].(float64); ok {
					additionalDays = int(days)
				}
			}
		}
	}

	var subscription models.Subscription
	if err := s.db.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		return nil, fmt.Errorf("订阅不存在: %v", err)
	}

	if additionalDevices > 0 {
		subscription.DeviceLimit += additionalDevices
	}

	if additionalDays > 0 {
		now := utils.GetBeijingTime()
		if subscription.ExpireTime.Before(now) {
			subscription.ExpireTime = now.AddDate(0, 0, additionalDays)
		} else {
			subscription.ExpireTime = subscription.ExpireTime.AddDate(0, 0, additionalDays)
		}
	}

	if err := s.db.Save(&subscription).Error; err != nil {
		return nil, fmt.Errorf("升级订阅失败: %v", err)
	}

	if utils.AppLogger != nil {
		utils.AppLogger.Info("ProcessPaidOrder: ✅ 设备升级成功 - user_id=%d, additional_devices=%d, additional_days=%d, device_limit=%d, expire_time=%s",
			user.ID, additionalDevices, additionalDays, subscription.DeviceLimit, subscription.ExpireTime.Format("2006-01-02 15:04:05"))
	}

	return &subscription, nil
}

func (s *OrderService) updateUserLevel(user *models.User) {
	var userLevels []models.UserLevel
	if err := s.db.Where("is_active = ?", true).Order("level_order ASC").Find(&userLevels).Error; err == nil {
		var targetLevel *models.UserLevel
		for i := range userLevels {
			level := &userLevels[i]
			if user.TotalConsumption >= level.MinConsumption {
				if targetLevel == nil || level.LevelOrder < targetLevel.LevelOrder {
					targetLevel = level
				}
			}
		}

		if targetLevel != nil {
			if !user.UserLevelID.Valid || user.UserLevelID.Int64 != int64(targetLevel.ID) {
				var currentLevel models.UserLevel
				shouldUpgrade := true
				if user.UserLevelID.Valid {
					if err := s.db.First(&currentLevel, user.UserLevelID.Int64).Error; err == nil {
						if currentLevel.LevelOrder < targetLevel.LevelOrder {
							shouldUpgrade = false
						}
					}
				}
				if shouldUpgrade {
					user.UserLevelID = sql.NullInt64{Int64: int64(targetLevel.ID), Valid: true}
					if err := s.db.Save(user).Error; err != nil {
						if utils.AppLogger != nil {
							utils.AppLogger.Error("更新用户等级失败: %v", err)
						}
					} else if utils.AppLogger != nil {
						utils.AppLogger.Info("ProcessPaidOrder: ✅ 用户等级升级 - user_id=%d, level_id=%d, level_name=%s",
							user.ID, targetLevel.ID, targetLevel.LevelName)
					}
				}
			}
		}
	}
}

func (s *OrderService) processInviteRewards(order *models.Order, paidAmount float64) {
	var inviteRelation models.InviteRelation
	if err := s.db.Where("invitee_id = ? AND (inviter_reward_given = ? OR invitee_reward_given = ?)",
		order.UserID, false, false).First(&inviteRelation).Error; err != nil {
		return
	}

	var inviteCode models.InviteCode
	if err := s.db.First(&inviteCode, inviteRelation.InviteCodeID).Error; err != nil {
		utils.LogError("processInviteRewards: invite code not found", err, map[string]interface{}{
			"invite_code_id": inviteRelation.InviteCodeID,
		})
		return
	}

	if inviteCode.MinOrderAmount > 0 && paidAmount < inviteCode.MinOrderAmount {
		if utils.AppLogger != nil {
			utils.AppLogger.Info("processInviteRewards: ⏳ 订单金额未达到最小要求 - order_id=%d, paid_amount=%.2f, min_amount=%.2f",
				order.ID, paidAmount, inviteCode.MinOrderAmount)
		}
		return
	}

	if inviteCode.NewUserOnly {
		var orderCount int64
		s.db.Model(&models.Order{}).Where("user_id = ? AND status = ?", order.UserID, "paid").Count(&orderCount)
		if orderCount > 1 {
			if utils.AppLogger != nil {
				utils.AppLogger.Info("processInviteRewards: ⏸️ 不是新用户订单，不发放奖励 - order_id=%d, order_count=%d",
					order.ID, orderCount)
			}
			return
		}
	}

	if !inviteRelation.InviteeFirstOrderID.Valid {
		inviteRelation.InviteeFirstOrderID = sql.NullInt64{Int64: int64(order.ID), Valid: true}
	}

	inviteRelation.InviteeTotalConsumption += paidAmount

	if !inviteRelation.InviterRewardGiven && inviteRelation.InviterRewardAmount > 0 {
		var inviter models.User
		if err := s.db.First(&inviter, inviteRelation.InviterID).Error; err == nil {
			inviter.Balance += inviteRelation.InviterRewardAmount
			inviter.TotalInviteReward += inviteRelation.InviterRewardAmount
			inviter.TotalInviteCount++
			if err := s.db.Save(&inviter).Error; err == nil {
				inviteRelation.InviterRewardGiven = true
				if utils.AppLogger != nil {
					utils.AppLogger.Info("processInviteRewards: ✅ 发放邀请者奖励 - inviter_id=%d, amount=%.2f, order_id=%d",
						inviter.ID, inviteRelation.InviterRewardAmount, order.ID)
				}
			} else {
				utils.LogError("processInviteRewards: failed to give inviter reward", err, map[string]interface{}{
					"inviter_id": inviter.ID,
					"amount":     inviteRelation.InviterRewardAmount,
				})
			}
		}
	}

	if !inviteRelation.InviteeRewardGiven && inviteRelation.InviteeRewardAmount > 0 {
		var invitee models.User
		if err := s.db.First(&invitee, order.UserID).Error; err == nil {
			invitee.Balance += inviteRelation.InviteeRewardAmount
			if err := s.db.Save(&invitee).Error; err == nil {
				inviteRelation.InviteeRewardGiven = true
				if utils.AppLogger != nil {
					utils.AppLogger.Info("processInviteRewards: ✅ 发放被邀请者奖励 - invitee_id=%d, amount=%.2f, order_id=%d",
						invitee.ID, inviteRelation.InviteeRewardAmount, order.ID)
				}
			} else {
				utils.LogError("processInviteRewards: failed to give invitee reward", err, map[string]interface{}{
					"invitee_id": invitee.ID,
					"amount":     inviteRelation.InviteeRewardAmount,
				})
			}
		}
	}

	if err := s.db.Save(&inviteRelation).Error; err != nil {
		utils.LogError("processInviteRewards: failed to save invite relation", err, map[string]interface{}{
			"invite_relation_id": inviteRelation.ID,
		})
	}
}

package order

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/gorm"

	"cboard-go/internal/services/email"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/services/payment"
)

// CreateOrderParams 创建订单参数
type CreateOrderParams struct {
	PackageID     uint    `json:"package_id"`
	CouponCode    string  `json:"coupon_code"`
	PaymentMethod string  `json:"payment_method"`
	UseBalance    bool    `json:"use_balance"`
	BalanceAmount float64 `json:"balance_amount"`
}

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

// CreateOrder 创建订单
func (s *OrderService) CreateOrder(userID uint, params CreateOrderParams) (*models.Order, string, error) {
	// 获取用户信息
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, "", fmt.Errorf("用户不存在")
	}

	// 获取套餐
	var pkg models.Package
	if err := s.db.First(&pkg, params.PackageID).Error; err != nil {
		return nil, "", fmt.Errorf("套餐不存在")
	}

	if !pkg.IsActive {
		return nil, "", fmt.Errorf("套餐已停用")
	}

	baseAmount := pkg.Price
	levelDiscountAmount := 0.0
	couponDiscountAmount := 0.0
	finalAmount := baseAmount

	// 计算等级折扣
	if user.UserLevelID.Valid {
		var lvl models.UserLevel
		if err := s.db.First(&lvl, user.UserLevelID.Int64).Error; err == nil {
			if lvl.DiscountRate > 0 && lvl.DiscountRate < 1.0 {
				levelDiscountAmount = baseAmount * (1.0 - lvl.DiscountRate)
				finalAmount = baseAmount * lvl.DiscountRate
			}
		}
	}

	// 计算优惠券折扣
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

	// 处理余额支付
	if params.UseBalance && params.BalanceAmount > 0 {
		if user.Balance < params.BalanceAmount {
			return nil, "", fmt.Errorf("余额不足")
		}
		if params.BalanceAmount > finalAmount {
			params.BalanceAmount = finalAmount
		}
		balanceUsed = params.BalanceAmount
		finalAmount -= balanceUsed

		// 扣除余额
		user.Balance -= balanceUsed
		if err := s.db.Save(&user).Error; err != nil {
			return nil, "", fmt.Errorf("扣除余额失败")
		}
	}

	// 修正浮点数误差
	if finalAmount <= 0.01 {
		finalAmount = 0
	}

	// 创建订单
	orderNo, err := utils.GenerateOrderNo(s.db)
	if err != nil {
		return nil, "", fmt.Errorf("生成订单号失败: %v", err)
	}
	order := models.Order{
		OrderNo:        orderNo,
		UserID:         user.ID,
		PackageID:      pkg.ID,
		Amount:         baseAmount,
		Status:         "pending",
		DiscountAmount: database.NullFloat64(totalDiscountAmount),
		FinalAmount:    database.NullFloat64(balanceUsed + finalAmount), // 记录实际价值（余额+支付）
	}

	if finalAmount == 0 {
		order.Status = "paid"
		order.PaymentTime = database.NullTime(utils.GetBeijingTime())
		order.PaymentMethodName = database.NullString("余额支付")
	} else {
		// 混合支付或其他
		methodName := params.PaymentMethod
		if balanceUsed > 0 {
			methodName = fmt.Sprintf("余额支付(%.2f元)+%s", balanceUsed, params.PaymentMethod)

			// 记录余额使用量
			extraData := fmt.Sprintf(`{"balance_used":%.2f}`, balanceUsed)
			order.ExtraData = database.NullString(extraData)
		}
		order.PaymentMethodName = database.NullString(methodName)
	}

	if couponID != nil {
		order.CouponID = database.NullInt64(*couponID)
	}

	if err := s.db.Create(&order).Error; err != nil {
		return nil, "", fmt.Errorf("创建订单失败: %v", err)
	}

	// 如果已全额支付（余额）
	if order.Status == "paid" {
		if _, err := s.ProcessPaidOrder(&order); err != nil {
			utils.LogError("CreateOrder: process paid order failed", err, nil)
			// 这里虽然处理失败，但订单已创建且钱已扣，不能简单返回错误，记录日志即可
			// 返回nil error表示订单创建成功，但后续处理有异常
		}

		// 发送邮件
		go s.sendPaymentSuccessEmail(&user, &order, &pkg, balanceUsed+finalAmount, "余额支付")
		return &order, "", nil
	}

	// 需要第三方支付
	var paymentURL string
	if params.PaymentMethod != "" && params.PaymentMethod != "balance" {
		// 生成支付链接
		url, err := s.generatePaymentURL(&order, params.PaymentMethod, finalAmount)
		if err != nil {
			// 支付链接生成失败，但订单已创建
			return &order, "", fmt.Errorf("生成支付链接失败: %v", err)
		}
		paymentURL = url
	}

	return &order, paymentURL, nil
}

// generatePaymentURL 生成支付链接
func (s *OrderService) generatePaymentURL(order *models.Order, payType string, amount float64) (string, error) {
	var paymentConfig models.PaymentConfig
	if err := s.db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", payType, 1).Order("sort_order ASC").First(&paymentConfig).Error; err != nil {
		return "", fmt.Errorf("未找到启用的支付配置")
	}

	// 创建支付交易记录
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
	default:
		return "", fmt.Errorf("不支持的支付方式: %s", paymentConfig.PayType)
	}
}

// sendPaymentSuccessEmail 发送支付成功邮件
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

// ProcessPaidOrder 处理已支付订单的后续逻辑（开通/续费订阅、更新消费、升级等级）
// 调用此方法前，订单状态应已更新为 paid
// 支持套餐订单（PackageID > 0）和设备升级订单（PackageID = 0）
func (s *OrderService) ProcessPaidOrder(order *models.Order) (*models.Subscription, error) {
	if order.Status != "paid" {
		return nil, fmt.Errorf("订单状态未支付")
	}

	// 获取用户信息
	var user models.User
	if err := s.db.First(&user, order.UserID).Error; err != nil {
		return nil, fmt.Errorf("用户不存在: %v", err)
	}

	// 计算支付金额（用于其他业务逻辑，如邀请奖励）
	paidAmount := order.Amount
	if order.FinalAmount.Valid {
		paidAmount = order.FinalAmount.Float64
	}

	// 1. 更新用户累计消费（所有订单类型都需要）
	// 注意：累计消费应该使用原价（Amount），而不是折扣后的价格（FinalAmount）
	// 这样等级升级才能正确反映用户的真实消费水平
	user.TotalConsumption += order.Amount
	if err := s.db.Save(&user).Error; err != nil {
		return nil, fmt.Errorf("更新用户累计消费失败: %v", err)
	}

	// 2. 检查并更新用户等级（所有订单类型都需要）
	s.updateUserLevel(&user)

	// 3. 处理邀请奖励（所有订单类型都需要）
	s.processInviteRewards(order, paidAmount)

	// 4. 处理订单特定的业务逻辑
	if order.PackageID > 0 {
		// 套餐订单：处理订阅开通/续费
		return s.processPackageOrder(order, &user)
	} else {
		// 设备升级订单：处理订阅升级
		return s.processDeviceUpgradeOrder(order, &user)
	}
}

// processPackageOrder 处理套餐订单
func (s *OrderService) processPackageOrder(order *models.Order, user *models.User) (*models.Subscription, error) {
	// 获取套餐信息
	var pkg models.Package
	if err := s.db.First(&pkg, order.PackageID).Error; err != nil {
		return nil, fmt.Errorf("套餐不存在: %v", err)
	}

	now := utils.GetBeijingTime()

	// 更新或创建订阅
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
		
		// 发送管理员通知（订阅创建）
		go func() {
			notificationService := notification.NewNotificationService()
			createTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
			_ = notificationService.SendAdminNotification("subscription_created", map[string]interface{}{
				"username":     user.Username,
				"email":        user.Email,
				"package_name": pkg.Name,
				"device_limit": pkg.DeviceLimit,
				"duration_days": pkg.DurationDays,
				"expire_time":  expireTime.Format("2006-01-02 15:04:05"),
				"create_time":  createTime,
			})
		}()
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

	return &subscription, nil
}

// processDeviceUpgradeOrder 处理设备升级订单
func (s *OrderService) processDeviceUpgradeOrder(order *models.Order, user *models.User) (*models.Subscription, error) {
	// 从 ExtraData 中解析升级信息
	var additionalDevices int
	var additionalDays int

	if order.ExtraData.Valid && order.ExtraData.String != "" {
		// 解析 JSON
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

	// 查找用户的订阅
	var subscription models.Subscription
	if err := s.db.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		return nil, fmt.Errorf("订阅不存在: %v", err)
	}

	// 升级设备数量
	if additionalDevices > 0 {
		subscription.DeviceLimit += additionalDevices
	}

	// 延长订阅时间
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

// updateUserLevel 更新用户等级
func (s *OrderService) updateUserLevel(user *models.User) {
	var userLevels []models.UserLevel
	if err := s.db.Where("is_active = ?", true).Order("level_order ASC").Find(&userLevels).Error; err == nil {
		// 找到所有满足条件的等级，选择 level_order 最小的（最高等级）
		var targetLevel *models.UserLevel
		for i := range userLevels {
			level := &userLevels[i]
			if user.TotalConsumption >= level.MinConsumption {
				// 如果还没有目标等级，或者当前等级的 level_order 更小（等级更高）
				if targetLevel == nil || level.LevelOrder < targetLevel.LevelOrder {
					targetLevel = level
				}
			}
		}

		// 如果找到了满足条件的等级
		if targetLevel != nil {
			// 检查是否需要升级
			if !user.UserLevelID.Valid || user.UserLevelID.Int64 != int64(targetLevel.ID) {
				// 需要升级
				var currentLevel models.UserLevel
				shouldUpgrade := true
				if user.UserLevelID.Valid {
					if err := s.db.First(&currentLevel, user.UserLevelID.Int64).Error; err == nil {
						// 如果当前等级更高（level_order 更小），不降级
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

// processInviteRewards 处理邀请奖励（订单支付成功后）
func (s *OrderService) processInviteRewards(order *models.Order, paidAmount float64) {
	// 查找该用户的邀请关系（查找未完全发放奖励的关系）
	var inviteRelation models.InviteRelation
	// 查找邀请者或被邀请者奖励未发放的关系
	if err := s.db.Where("invitee_id = ? AND (inviter_reward_given = ? OR invitee_reward_given = ?)",
		order.UserID, false, false).First(&inviteRelation).Error; err != nil {
		// 没有未发放奖励的邀请关系，直接返回
		return
	}

	// 获取邀请码信息
	var inviteCode models.InviteCode
	if err := s.db.First(&inviteCode, inviteRelation.InviteCodeID).Error; err != nil {
		utils.LogError("processInviteRewards: invite code not found", err, map[string]interface{}{
			"invite_code_id": inviteRelation.InviteCodeID,
		})
		return
	}

	// 检查是否满足最小订单金额要求
	if inviteCode.MinOrderAmount > 0 && paidAmount < inviteCode.MinOrderAmount {
		if utils.AppLogger != nil {
			utils.AppLogger.Info("processInviteRewards: ⏳ 订单金额未达到最小要求 - order_id=%d, paid_amount=%.2f, min_amount=%.2f",
				order.ID, paidAmount, inviteCode.MinOrderAmount)
		}
		return
	}

	// 检查是否是新用户订单（如果设置了 NewUserOnly）
	if inviteCode.NewUserOnly {
		// 检查这是否是被邀请者的第一个订单
		var orderCount int64
		s.db.Model(&models.Order{}).Where("user_id = ? AND status = ?", order.UserID, "paid").Count(&orderCount)
		if orderCount > 1 {
			// 不是第一个订单，不发放奖励
			if utils.AppLogger != nil {
				utils.AppLogger.Info("processInviteRewards: ⏸️ 不是新用户订单，不发放奖励 - order_id=%d, order_count=%d",
					order.ID, orderCount)
			}
			return
		}
	}

	// 更新邀请关系的首次订单ID
	if !inviteRelation.InviteeFirstOrderID.Valid {
		inviteRelation.InviteeFirstOrderID = sql.NullInt64{Int64: int64(order.ID), Valid: true}
	}

	// 更新累计消费
	inviteRelation.InviteeTotalConsumption += paidAmount

	// 发放邀请者奖励（如果还未发放）
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

	// 发放被邀请者奖励（如果还未发放）
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

	// 保存邀请关系
	if err := s.db.Save(&inviteRelation).Error; err != nil {
		utils.LogError("processInviteRewards: failed to save invite relation", err, map[string]interface{}{
			"invite_relation_id": inviteRelation.ID,
		})
	}
}

package order

import (
	"cboard-go/internal/core/cache"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/cache_service"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/services/payment"
	"cboard-go/internal/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"

	"gorm.io/gorm"
)

type CreateOrderParams struct {
	PackageID      uint   `json:"package_id"`
	CouponCode     string `json:"coupon_code"`
	PaymentMethod  string `json:"payment_method"`
	UseBalance     bool   `json:"use_balance"`
	DurationMonths int    `json:"duration_months"`
	UserAgent      string `json:"-"`
	ClientIP       string `json:"-"`
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
	promotionDiscountAmount := 0.0
	finalAmount := baseAmount

	if user.UserLevelID.Valid {
		var lvl models.UserLevel
		if err := s.db.First(&lvl, user.UserLevelID.Int64).Error; err == nil {
			if lvl.DiscountRate > 0 && lvl.DiscountRate < 1.0 {
				levelDiscountAmount = math.Round(baseAmount*(1.0-lvl.DiscountRate)*100) / 100
				finalAmount = math.Round(baseAmount*lvl.DiscountRate*100) / 100
			}
		}
	}

	// --- 优惠券验证：总量、有效期、每用户上限 ---
	var coupon *models.Coupon
	if params.CouponCode != "" {
		var c models.Coupon
		if err := s.db.Where("code = ? AND status = ?", params.CouponCode, "active").First(&c).Error; err != nil {
			return nil, "", fmt.Errorf("优惠券不存在或已失效")
		}
		now := utils.GetBeijingTime()
		if now.Before(c.ValidFrom) || now.After(c.ValidUntil) {
			return nil, "", fmt.Errorf("优惠券不在有效期内")
		}
		if c.TotalQuantity.Valid && c.UsedQuantity >= int(c.TotalQuantity.Int64) {
			return nil, "", fmt.Errorf("优惠券已被领完")
		}
		if c.MaxUsesPerUser > 0 {
			var userUsageCount int64
			s.db.Model(&models.CouponUsage{}).Where("coupon_id = ? AND user_id = ?", c.ID, userID).Count(&userUsageCount)
			if int(userUsageCount) >= c.MaxUsesPerUser {
				return nil, "", fmt.Errorf("您已达到该优惠券的使用上限")
			}
		}
		if c.MinAmount.Valid && finalAmount < c.MinAmount.Float64 {
			return nil, "", fmt.Errorf("订单金额未达到优惠券最低使用金额")
		}
		if c.Type == "discount" {
			couponDiscountAmount = math.Round(finalAmount*(c.DiscountValue/100)*100) / 100
			if c.MaxDiscount.Valid && couponDiscountAmount > c.MaxDiscount.Float64 {
				couponDiscountAmount = c.MaxDiscount.Float64
			}
		} else if c.Type == "fixed" {
			couponDiscountAmount = c.DiscountValue
			if couponDiscountAmount > finalAmount {
				couponDiscountAmount = finalAmount
			}
		}
		finalAmount = math.Round((finalAmount-couponDiscountAmount)*100) / 100
		coupon = &c
	}

	// --- 营销活动折扣 ---
	var promotionParticipation *models.PromotionParticipation
	if params.CouponCode == "" {
		// 只有在没有使用优惠券时才应用营销活动折扣
		discount, participation, err := s.applyPromotionDiscount(userID, params.PackageID, finalAmount)
		if err == nil && discount > 0 && participation != nil {
			promotionDiscountAmount = discount
			finalAmount = math.Round((finalAmount-promotionDiscountAmount)*100) / 100
			promotionParticipation = participation
		}
	}

	totalDiscountAmount := levelDiscountAmount + couponDiscountAmount + promotionDiscountAmount
	balanceUsed := 0.0

	if params.UseBalance && user.Balance > 0 {
		availableBalance := math.Round(user.Balance*100) / 100
		if availableBalance > finalAmount {
			availableBalance = finalAmount
		}
		if availableBalance > 0 {
			balanceUsed = availableBalance
			finalAmount -= balanceUsed
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
		FinalAmount:    database.NullFloat64(finalAmount),
		ExtraData:      database.NullString(string(extraDataJSON)),
		CreatedAt:      now,
	}

	if coupon != nil {
		order.CouponID = database.NullInt64(utils.MustSafeUintToInt64(coupon.ID))
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

	// --- 事务：创建订单 + 记录优惠券使用 + 递增计数器 ---
	// 注意：余额不在此处扣除，而是在支付成功回调时扣除
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&order).Error; err != nil {
			return fmt.Errorf("创建订单失败: %v", err)
		}

		// 记录优惠券使用 + 原子递增 used_quantity（事务内二次校验防并发超卖）
		if coupon != nil {
			// 事务内锁定优惠券行，防止并发超卖
			var lockedCoupon models.Coupon
			if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&lockedCoupon, coupon.ID).Error; err != nil {
				return fmt.Errorf("锁定优惠券失败: %v", err)
			}
			if lockedCoupon.TotalQuantity.Valid && lockedCoupon.UsedQuantity >= int(lockedCoupon.TotalQuantity.Int64) {
				return fmt.Errorf("优惠券已被领完")
			}

			usage := models.CouponUsage{
				CouponID:       coupon.ID,
				UserID:         userID,
				OrderID:        sql.NullInt64{Int64: utils.MustSafeUintToInt64(order.ID), Valid: true},
				DiscountAmount: couponDiscountAmount,
			}
			if err := tx.Create(&usage).Error; err != nil {
				return fmt.Errorf("记录优惠券使用失败: %v", err)
			}
			if err := tx.Model(&models.Coupon{}).Where("id = ?", coupon.ID).
				Update("used_quantity", gorm.Expr("used_quantity + 1")).Error; err != nil {
				return fmt.Errorf("更新优惠券使用次数失败: %v", err)
			}
		}

		// 关联营销活动参与记录到订单
		if promotionParticipation != nil {
			if err := tx.Model(&models.PromotionParticipation{}).Where("id = ?", promotionParticipation.ID).
				Update("order_id", sql.NullInt64{Int64: utils.MustSafeUintToInt64(order.ID), Valid: true}).Error; err != nil {
				return fmt.Errorf("关联营销活动失败: %v", err)
			}
		}

		return nil
	})
	if txErr != nil {
		return nil, "", txErr
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
	} else if strings.HasPrefix(payType, "codepay_") {
		if err := s.db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", "codepay", 1).Order("sort_order ASC").First(&paymentConfig).Error; err == nil {
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
	case "codepay", "codepay_alipay", "codepay_wxpay":
		svc, err := payment.NewCodepayService(&paymentConfig)
		if err != nil {
			return "", err
		}
		codepayType := extractCodepayType(payType)
		utils.LogInfo("码支付生成支付链接: payType=%s, extracted_paymentType=%s, order_no=%s", payType, codepayType, order.OrderNo)
		return svc.CreatePayment(order, amount, codepayType)
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

func extractCodepayType(payType string) string {
	if strings.HasPrefix(payType, "codepay_") {
		return strings.TrimPrefix(payType, "codepay_")
	}
	return "alipay"
}

func (s *OrderService) sendPaymentSuccessEmail(user *models.User, order *models.Order, pkg *models.Package, amount float64, paymentMethod string) {
	emailService := email.NewEmailService()
	templateBuilder := email.NewEmailTemplateBuilder()
	paymentTime := utils.FormatBeijingTime(utils.GetBeijingTime())

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

	// 从订单的 ExtraData 中提取余额使用金额
	var balanceUsed float64
	var extraData map[string]interface{}
	if order.ExtraData.Valid && order.ExtraData.String != "" {
		if err := json.Unmarshal([]byte(order.ExtraData.String), &extraData); err == nil {
			if balance, ok := extraData["balance_used"].(float64); ok {
				balanceUsed = balance
			}
		}
	}

	// 如果订单使用了余额，在支付成功时扣除
	orderType := ""
	if extraData != nil {
		if rawType, ok := extraData["type"].(string); ok {
			orderType = rawType
		}
	}

	if orderType == "device_upgrade" {
		var existingSubscription models.Subscription
		if err := s.db.Where("user_id = ?", user.ID).First(&existingSubscription).Error; err == nil {
			additionalDevices := 0
			additionalDays := 0
			if devices, ok := extraData["additional_devices"].(float64); ok {
				additionalDevices = int(devices)
			}
			if days, ok := extraData["additional_days"].(float64); ok {
				additionalDays = int(days)
			}
			if additionalDevices <= 0 && additionalDays <= 0 {
				return &existingSubscription, nil
			}
		}
	}

	if balanceUsed > 0 {
		result := s.db.Model(&models.User{}).Where("id = ? AND balance >= ?", user.ID, balanceUsed).
			Update("balance", gorm.Expr("balance - ?", balanceUsed))
		if result.Error != nil {
			return nil, fmt.Errorf("扣除余额失败: %v", result.Error)
		}
		if result.RowsAffected == 0 {
			return nil, fmt.Errorf("余额不足，无法完成支付")
		}
		// 重新加载用户信息
		if err := s.db.First(&user, order.UserID).Error; err != nil {
			return nil, fmt.Errorf("重新加载用户信息失败: %v", err)
		}
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

	// 更新营销活动参与记录状态为已完成
	if err := s.db.Model(&models.PromotionParticipation{}).
		Where("order_id = ? AND status = ?", order.ID, "pending").
		Updates(map[string]interface{}{
			"status":     "completed",
			"applied_at": database.NullTime(utils.GetBeijingTime()),
		}).Error; err != nil {
		utils.LogError("ProcessPaidOrder: 更新营销活动参与记录失败", err, map[string]interface{}{
			"order_id": order.ID,
		})
	}

	// 检查是否是自定义套餐订单
	isCustomPackage := false
	if order.ExtraData.Valid && order.ExtraData.String != "" {
		var extraData map[string]interface{}
		if err := json.Unmarshal([]byte(order.ExtraData.String), &extraData); err == nil {
			if orderType, ok := extraData["type"].(string); ok && orderType == "custom_package" {
				isCustomPackage = true
			}
		}
	}

	var result *models.Subscription
	var err error
	if order.PackageID > 0 || isCustomPackage {
		result, err = s.processPackageOrder(order, &user)
	} else {
		result, err = s.processDeviceUpgradeOrder(order, &user)
	}

	// 异步清除缓存
	if err == nil && result != nil {
		go func(userID uint, subscriptionURL string) {
			cs := cache_service.NewCacheService()
			if cacheErr := cs.ClearUserCache(userID); cacheErr != nil {
				log.Printf("failed to clear user cache: %v", cacheErr)
			}
			if cacheErr := cs.ClearUserSubscriptionCache(userID); cacheErr != nil {
				log.Printf("failed to clear user subscription cache: %v", cacheErr)
			}
			// 清除订阅配置缓存，确保用户立即获得最新配置
			if cacheErr := cache.ClearSubscriptionConfigCache(subscriptionURL); cacheErr != nil {
				log.Printf("failed to clear subscription config cache: %v", cacheErr)
			}
		}(user.ID, result.SubscriptionURL)
	}

	return result, err
}

func (s *OrderService) processPackageOrder(order *models.Order, user *models.User) (*models.Subscription, error) {
	// 检查是否是自定义套餐
	isCustomPackage := false
	var customDevices int
	var customMonths int

	if order.ExtraData.Valid && order.ExtraData.String != "" {
		var extraData map[string]interface{}
		if err := json.Unmarshal([]byte(order.ExtraData.String), &extraData); err == nil {
			if orderType, ok := extraData["type"].(string); ok && orderType == "custom_package" {
				isCustomPackage = true
				if devices, ok := extraData["devices"].(float64); ok {
					customDevices = int(devices)
				}
				if months, ok := extraData["months"].(float64); ok {
					customMonths = int(months)
				}
			}
		}
	}

	var pkg models.Package
	var deviceLimit int
	var durationMonths int
	var totalDurationDays int
	var packageName string

	if isCustomPackage {
		// 自定义套餐：直接使用 ExtraData 中的参数
		if customDevices <= 0 || customMonths <= 0 {
			return nil, fmt.Errorf("自定义套餐参数无效: devices=%d, months=%d", customDevices, customMonths)
		}
		deviceLimit = customDevices
		durationMonths = customMonths
		totalDurationDays = customMonths * 30
		packageName = fmt.Sprintf("自定义套餐 (%d设备/%d月)", customDevices, customMonths)
	} else {
		// 普通套餐：从数据库加载套餐信息
		if err := s.db.First(&pkg, order.PackageID).Error; err != nil {
			return nil, fmt.Errorf("套餐不存在: %v", err)
		}
		packageName = pkg.Name
		deviceLimit = pkg.DeviceLimit

		durationMonths = 1
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

		totalDurationDays = pkg.DurationDays * durationMonths
	}

	now := utils.GetBeijingTime()

	var subscription models.Subscription
	if err := s.db.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		subscriptionURL := utils.GenerateSubscriptionURL()
		expireTime := now.AddDate(0, 0, totalDurationDays)
		var pkgID *int64
		if !isCustomPackage {
			id := utils.MustSafeUintToInt64(pkg.ID)
			pkgID = &id
		}
		subscription = models.Subscription{
			UserID:          user.ID,
			PackageID:       pkgID,
			SubscriptionURL: subscriptionURL,
			DeviceLimit:     deviceLimit,
			CurrentDevices:  0,
			IsActive:        true,
			Status:          "active",
			ExpireTime:      expireTime,
		}
		if err := s.db.Create(&subscription).Error; err != nil {
			return nil, fmt.Errorf("创建订阅失败: %v", err)
		}
		if utils.AppLogger != nil {
			utils.AppLogger.Info("ProcessPaidOrder: ✅ 创建新订阅成功 - user_id=%d, package_name=%s, device_limit=%d, duration_months=%d, duration_days=%d, expire_time=%s",
				user.ID, packageName, deviceLimit, durationMonths, totalDurationDays, utils.FormatBeijingTime(expireTime))
		}

		go func() {
			notificationService := notification.NewNotificationService()
			createTime := utils.FormatBeijingTime(utils.GetBeijingTime())
			_ = notificationService.SendAdminNotification("subscription_created", map[string]interface{}{
				"username":        user.Username,
				"email":           user.Email,
				"package_name":    packageName,
				"device_limit":    deviceLimit,
				"duration_months": durationMonths,
				"duration_days":   totalDurationDays,
				"expire_time":     utils.FormatBeijingTime(expireTime),
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
		subscription.DeviceLimit = deviceLimit
		subscription.IsActive = true
		subscription.Status = "active"
		if !isCustomPackage {
			pkgID := utils.MustSafeUintToInt64(pkg.ID)
			subscription.PackageID = &pkgID
		}

		if err := s.db.Save(&subscription).Error; err != nil {
			return nil, fmt.Errorf("更新订阅失败: %v", err)
		}
		if utils.AppLogger != nil {
			utils.AppLogger.Info("ProcessPaidOrder: ✅ 更新订阅成功 - user_id=%d, package_name=%s, device_limit: %d->%d, duration_months=%d, duration_days=%d, expire_time: %s->%s",
				user.ID, packageName, oldDeviceLimit, deviceLimit, durationMonths, totalDurationDays, utils.FormatBeijingTime(oldExpireTime), utils.FormatBeijingTime(subscription.ExpireTime))
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
			user.ID, additionalDevices, additionalDays, subscription.DeviceLimit, utils.FormatBeijingTime(subscription.ExpireTime))
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
			if !user.UserLevelID.Valid || user.UserLevelID.Int64 != utils.MustSafeUintToInt64(targetLevel.ID) {
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
					user.UserLevelID = sql.NullInt64{Int64: utils.MustSafeUintToInt64(targetLevel.ID), Valid: true}
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

	// 仅首单发放：如果已有首单记录，跳过
	if inviteRelation.InviteeFirstOrderID.Valid {
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

	// 记录首单
	inviteRelation.InviteeFirstOrderID = sql.NullInt64{Int64: utils.MustSafeUintToInt64(order.ID), Valid: true}
	inviteRelation.InviteeTotalConsumption += paidAmount

	if !inviteRelation.InviterRewardGiven && inviteRelation.InviterRewardAmount > 0 {
		var inviter models.User
		if err := s.db.First(&inviter, inviteRelation.InviterID).Error; err == nil {
			oldBalance := inviter.Balance
			result := s.db.Model(&models.User{}).Where("id = ?", inviter.ID).
				Updates(map[string]interface{}{
					"balance":             gorm.Expr("balance + ?", inviteRelation.InviterRewardAmount),
					"total_invite_reward": gorm.Expr("total_invite_reward + ?", inviteRelation.InviterRewardAmount),
					"total_invite_count":  gorm.Expr("total_invite_count + 1"),
				})
			if result.Error == nil {
				inviteRelation.InviterRewardGiven = true
				var freshInviter models.User
				s.db.First(&freshInviter, inviter.ID)
				if utils.AppLogger != nil {
					utils.AppLogger.Info("processInviteRewards: ✅ 发放邀请者奖励 - inviter_id=%d, amount=%.2f, order_id=%d",
						inviter.ID, inviteRelation.InviterRewardAmount, order.ID)
				}
				go func() {
					if err := utils.CreateBalanceLog(
						inviter.ID, "commission", inviteRelation.InviterRewardAmount,
						oldBalance, freshInviter.Balance, nil, nil,
						fmt.Sprintf("邀请奖励: 邀请人奖励 (订单 %s)", order.OrderNo),
						"system", nil, "",
					); err != nil {
						log.Printf("failed to create balance log: %v", err)
					}
					relationID := uint(inviteRelation.ID)
					if err := utils.CreateCommissionLog(
						inviter.ID, order.UserID, "order_reward",
						inviteRelation.InviterRewardAmount, &relationID, nil,
						fmt.Sprintf("邀请人奖励: 订单 %s", order.OrderNo),
					); err != nil {
						log.Printf("failed to create commission log: %v", err)
					}
				}()
			} else {
				utils.LogError("processInviteRewards: failed to give inviter reward", result.Error, map[string]interface{}{
					"inviter_id": inviter.ID, "amount": inviteRelation.InviterRewardAmount,
				})
			}
		}
	}

	if !inviteRelation.InviteeRewardGiven && inviteRelation.InviteeRewardAmount > 0 {
		var invitee models.User
		if err := s.db.First(&invitee, order.UserID).Error; err == nil {
			oldBalance := invitee.Balance
			result := s.db.Model(&models.User{}).Where("id = ?", invitee.ID).
				Update("balance", gorm.Expr("balance + ?", inviteRelation.InviteeRewardAmount))
			if result.Error == nil {
				inviteRelation.InviteeRewardGiven = true
				var freshInvitee models.User
				s.db.First(&freshInvitee, invitee.ID)
				if utils.AppLogger != nil {
					utils.AppLogger.Info("processInviteRewards: ✅ 发放被邀请者奖励 - invitee_id=%d, amount=%.2f, order_id=%d",
						invitee.ID, inviteRelation.InviteeRewardAmount, order.ID)
				}
				go func() {
					if err := utils.CreateBalanceLog(
						invitee.ID, "commission", inviteRelation.InviteeRewardAmount,
						oldBalance, freshInvitee.Balance, nil, nil,
						fmt.Sprintf("邀请奖励: 被邀请人奖励 (订单 %s)", order.OrderNo),
						"system", nil, "",
					); err != nil {
						log.Printf("failed to create balance log: %v", err)
					}
					relationID := uint(inviteRelation.ID)
					if err := utils.CreateCommissionLog(
						inviteRelation.InviterID, invitee.ID, "order_reward",
						inviteRelation.InviteeRewardAmount, &relationID, nil,
						fmt.Sprintf("被邀请人奖励: 订单 %s", order.OrderNo),
					); err != nil {
						log.Printf("failed to create commission log: %v", err)
					}
				}()
			} else {
				utils.LogError("processInviteRewards: failed to give invitee reward", result.Error, map[string]interface{}{
					"invitee_id": invitee.ID, "amount": inviteRelation.InviteeRewardAmount,
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

func (s *OrderService) ProcessRefundOrder(order *models.Order) error {
	if order.Status != "paid" {
		return fmt.Errorf("只能退款已支付的订单")
	}

	var user models.User
	if err := s.db.First(&user, order.UserID).Error; err != nil {
		return fmt.Errorf("用户不存在: %v", err)
	}

	paidAmount := order.Amount
	if order.FinalAmount.Valid {
		paidAmount = order.FinalAmount.Float64
	}

	// 回退用户累计消费
	if user.TotalConsumption >= order.Amount {
		user.TotalConsumption -= order.Amount
	} else {
		user.TotalConsumption = 0
	}

	// 回退余额（如果使用了余额支付）
	var balanceUsed float64 = 0
	if order.ExtraData.Valid && order.ExtraData.String != "" {
		var extraData map[string]interface{}
		if err := json.Unmarshal([]byte(order.ExtraData.String), &extraData); err == nil {
			if balanceUsedVal, ok := extraData["balance_used"].(float64); ok {
				balanceUsed = balanceUsedVal
			}
		}
	}

	// 回退订阅或设备升级
	if order.PackageID > 0 {
		// 套餐订单：回退订阅时长和设备限制
		if err := s.rollbackPackageOrder(order, &user); err != nil {
			return fmt.Errorf("回退套餐订单失败: %v", err)
		}
	} else {
		// 设备升级订单：回退设备数量和时长
		if err := s.rollbackDeviceUpgradeOrder(order, &user); err != nil {
			return fmt.Errorf("回退设备升级订单失败: %v", err)
		}
	}

	// 回退余额
	if balanceUsed > 0 {
		user.Balance += balanceUsed
		utils.LogInfo("ProcessRefundOrder: 回退余额 - user_id=%d, balance_used=%.2f, new_balance=%.2f", user.ID, balanceUsed, user.Balance)
	}

	// 回退邀请奖励（如果已发放）
	s.rollbackInviteRewards(order, paidAmount)

	// 更新用户信息
	if err := s.db.Save(&user).Error; err != nil {
		return fmt.Errorf("更新用户信息失败: %v", err)
	}

	// 更新订单状态
	order.Status = "refunded"
	if err := s.db.Save(order).Error; err != nil {
		return fmt.Errorf("更新订单状态失败: %v", err)
	}

	utils.LogInfo("ProcessRefundOrder: 订单退款成功 - order_id=%d, order_no=%s, user_id=%d, amount=%.2f", order.ID, order.OrderNo, user.ID, paidAmount)
	return nil
}

func (s *OrderService) rollbackPackageOrder(order *models.Order, user *models.User) error {
	var subscription models.Subscription
	if err := s.db.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		// 如果订阅不存在，说明可能是新创建的，退款时不需要回退
		utils.LogWarn("ProcessRefundOrder: 订阅不存在，跳过回退 - user_id=%d", user.ID)
		return nil
	}

	var pkg models.Package
	if err := s.db.First(&pkg, order.PackageID).Error; err != nil {
		return fmt.Errorf("套餐不存在: %v", err)
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

	totalDurationDays := pkg.DurationDays * durationMonths
	now := utils.GetBeijingTime()

	// 回退订阅时长
	if subscription.ExpireTime.After(now) {
		subscription.ExpireTime = subscription.ExpireTime.AddDate(0, 0, -totalDurationDays)
		// 如果回退后到期时间早于当前时间，设置为当前时间
		if subscription.ExpireTime.Before(now) {
			subscription.ExpireTime = now
		}
	}

	// 如果订阅到期时间已过，设置为当前时间
	if subscription.ExpireTime.Before(now) || subscription.ExpireTime.Equal(now) {
		subscription.IsActive = false
		subscription.Status = "expired"
	}

	if err := s.db.Save(&subscription).Error; err != nil {
		return fmt.Errorf("回退订阅失败: %v", err)
	}

	utils.LogInfo("ProcessRefundOrder: 回退套餐订单成功 - user_id=%d, package_id=%d, duration_days=%d, expire_time=%s",
		user.ID, pkg.ID, totalDurationDays, utils.FormatBeijingTime(subscription.ExpireTime))
	return nil
}

func (s *OrderService) rollbackDeviceUpgradeOrder(order *models.Order, user *models.User) error {
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
		return fmt.Errorf("订阅不存在: %v", err)
	}

	// 回退设备数量
	if additionalDevices > 0 {
		if subscription.DeviceLimit >= additionalDevices {
			subscription.DeviceLimit -= additionalDevices
		} else {
			subscription.DeviceLimit = 0
		}
	}

	// 回退时长
	if additionalDays > 0 {
		now := utils.GetBeijingTime()
		if subscription.ExpireTime.After(now) {
			subscription.ExpireTime = subscription.ExpireTime.AddDate(0, 0, -additionalDays)
			// 如果回退后到期时间早于当前时间，设置为当前时间
			if subscription.ExpireTime.Before(now) {
				subscription.ExpireTime = now
			}
		}
	}

	if err := s.db.Save(&subscription).Error; err != nil {
		return fmt.Errorf("回退设备升级失败: %v", err)
	}

	utils.LogInfo("ProcessRefundOrder: 回退设备升级成功 - user_id=%d, additional_devices=%d, additional_days=%d, device_limit=%d, expire_time=%s",
		user.ID, additionalDevices, additionalDays, subscription.DeviceLimit, utils.FormatBeijingTime(subscription.ExpireTime))
	return nil
}

func (s *OrderService) rollbackInviteRewards(order *models.Order, paidAmount float64) {
	var inviteRelation models.InviteRelation
	if err := s.db.Where("invitee_id = ?", order.UserID).First(&inviteRelation).Error; err != nil {
		// 没有邀请关系，不需要回退
		return
	}

	// 回退被邀请者累计消费
	if inviteRelation.InviteeTotalConsumption >= paidAmount {
		inviteRelation.InviteeTotalConsumption -= paidAmount
	} else {
		inviteRelation.InviteeTotalConsumption = 0
	}

	// 如果邀请奖励已发放，需要回退（这里只记录，实际回退需要手动处理或通过其他机制）
	// 注意：回退邀请奖励比较复杂，因为可能已经使用，这里只记录日志
	if inviteRelation.InviterRewardGiven || inviteRelation.InviteeRewardGiven {
		utils.LogWarn("ProcessRefundOrder: 订单已发放邀请奖励，需要手动处理回退 - order_id=%d, invite_relation_id=%d", order.ID, inviteRelation.ID)
	}

	if err := s.db.Save(&inviteRelation).Error; err != nil {
		utils.LogError("ProcessRefundOrder: 回退邀请关系失败", err, map[string]interface{}{
			"invite_relation_id": inviteRelation.ID,
		})
	}
}

// applyPromotionDiscount 应用营销活动折扣
func (s *OrderService) applyPromotionDiscount(userID uint, packageID uint, baseAmount float64) (float64, *models.PromotionParticipation, error) {
	now := utils.GetBeijingTime()

	// 查找可用的折扣
	var participation models.PromotionParticipation
	query := s.db.Where("user_id = ? AND status = ? AND reward_type = ? AND (expire_at IS NULL OR expire_at > ?)",
		userID, "pending", "discount", now).
		Preload("Promotion")

	if err := query.Order("created_at ASC").First(&participation).Error; err != nil {
		// 没有可用折扣
		return 0, nil, nil
	}

	// 检查活动是否适用于该套餐
	if participation.Promotion.PackageIDs.Valid && participation.Promotion.PackageIDs.String != "" {
		packageIDsStr := participation.Promotion.PackageIDs.String
		var packageIDs []uint
		if err := json.Unmarshal([]byte(packageIDsStr), &packageIDs); err == nil {
			found := false
			for _, pid := range packageIDs {
				if pid == packageID {
					found = true
					break
				}
			}
			if !found {
				// 该活动不适用于此套餐
				return 0, nil, nil
			}
		}
	}

	// 检查最低金额
	if participation.Promotion.MinAmount > 0 && baseAmount < participation.Promotion.MinAmount {
		return 0, nil, nil
	}

	// 计算折扣
	var discountAmount float64
	if participation.Promotion.DiscountType == "percentage" {
		discountAmount = baseAmount * (participation.RewardValue / 100)
		if participation.Promotion.MaxDiscount > 0 && discountAmount > participation.Promotion.MaxDiscount {
			discountAmount = participation.Promotion.MaxDiscount
		}
	} else if participation.Promotion.DiscountType == "fixed" {
		discountAmount = participation.RewardValue
		if discountAmount > baseAmount {
			discountAmount = baseAmount
		}
	}

	discountAmount = utils.RoundFloat(discountAmount, 2)
	return discountAmount, &participation, nil
}

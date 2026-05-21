package order

import (
	"cboard-go/internal/core/cache"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/cache_service"
	discountService "cboard-go/internal/services/discount"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/services/payment"
	promotionService "cboard-go/internal/services/promotion"
	"cboard-go/internal/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const orderTimeLayout = "2006-01-02 15:04:05"

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

type FinalizePaidOrderOptions struct {
	PaymentMethodName     string
	ExternalTransactionID string
	PaymentMethodID       uint
	CallbackData          string
	IPAddress             string
	BalanceAmount         float64
}

func (s *OrderService) CancelPendingOrder(orderNo string, userID uint) (*models.Order, error) {
	return s.MarkPendingOrderStatus(orderNo, userID, "cancelled")
}

func (s *OrderService) MarkPendingOrderStatus(orderNo string, userID uint, status string) (*models.Order, error) {
	status = strings.TrimSpace(status)
	if status != "cancelled" && status != "expired" && status != "failed" {
		return nil, fmt.Errorf("不支持的订单状态: %s", status)
	}

	var result models.Order
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var order models.Order
		query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_no = ?", orderNo)
		if userID > 0 {
			query = query.Where("user_id = ?", userID)
		}
		if err := query.First(&order).Error; err != nil {
			return fmt.Errorf("订单不存在")
		}
		if order.Status != "pending" {
			return fmt.Errorf("订单状态不允许更新为%s", status)
		}
		if err := s.releaseDiscountReservationsTx(tx, order.ID); err != nil {
			return err
		}
		if err := s.refundFrozenBalanceTx(tx, &order); err != nil {
			return err
		}
		order.Status = status
		if err := tx.Save(&order).Error; err != nil {
			return fmt.Errorf("更新订单状态失败: %v", err)
		}
		if err := tx.Model(&models.PaymentTransaction{}).
			Where("order_id = ? AND status = ?", order.ID, "pending").
			Update("status", status).Error; err != nil {
			return fmt.Errorf("更新支付交易状态失败: %v", err)
		}
		result = order
		return nil
	})
	if err != nil {
		return nil, err
	}
	if result.UserID > 0 {
		s.clearUserCaches(result.UserID)
	}
	return &result, nil
}

func (s *OrderService) CancelPendingOrders(orderIDs []uint) (int64, error) {
	if len(orderIDs) == 0 {
		return 0, nil
	}

	var cancelled int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var orders []models.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id IN ? AND status = ?", orderIDs, "pending").
			Find(&orders).Error; err != nil {
			return err
		}
		for _, order := range orders {
			if err := s.releaseDiscountReservationsTx(tx, order.ID); err != nil {
				return err
			}
			if err := s.refundFrozenBalanceTx(tx, &order); err != nil {
				return err
			}
			if err := tx.Model(&models.Order{}).Where("id = ? AND status = ?", order.ID, "pending").
				Update("status", "cancelled").Error; err != nil {
				return err
			}
			if err := tx.Model(&models.PaymentTransaction{}).
				Where("order_id = ? AND status = ?", order.ID, "pending").
				Update("status", "cancelled").Error; err != nil {
				return err
			}
			cancelled++
		}
		return nil
	})
	return cancelled, err
}

func (s *OrderService) refundFrozenBalanceTx(tx *gorm.DB, order *models.Order) error {
	if !order.ExtraData.Valid || order.ExtraData.String == "" {
		return nil
	}
	var extraData map[string]interface{}
	if err := json.Unmarshal([]byte(order.ExtraData.String), &extraData); err != nil {
		return nil
	}
	deducted, _ := extraData["balance_deducted"].(bool)
	if !deducted {
		return nil
	}
	balanceUsed, _ := extraData["balance_used"].(float64)
	if balanceUsed <= 0 {
		return nil
	}
	var user models.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, order.UserID).Error; err != nil {
		return fmt.Errorf("退还余额时用户不存在: %v", err)
	}
	oldBalance := user.Balance
	if err := tx.Model(&models.User{}).Where("id = ?", user.ID).
		Update("balance", gorm.Expr("balance + ?", balanceUsed)).Error; err != nil {
		return fmt.Errorf("退还余额失败: %v", err)
	}
	orderID := uint(order.ID)
	userID := user.ID
	if err := utils.CreateBalanceLogWithDB(
		tx, user.ID, "refund", balanceUsed,
		oldBalance, oldBalance+balanceUsed,
		&orderID, nil,
		fmt.Sprintf("订单取消退还余额，订单号: %s", order.OrderNo),
		"system", &userID, "",
	); err != nil {
		return fmt.Errorf("记录余额退还日志失败: %v", err)
	}
	extraData["balance_deducted"] = false
	if encodedExtra, err := json.Marshal(extraData); err == nil {
		order.ExtraData = database.NullString(string(encodedExtra))
		if err := tx.Model(order).Update("extra_data", order.ExtraData).Error; err != nil {
			return fmt.Errorf("更新订单数据失败: %v", err)
		}
	}
	return nil
}

func (s *OrderService) DeleteOrders(orderIDs []uint) (int64, error) {
	if len(orderIDs) == 0 {
		return 0, nil
	}

	var deleted int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var orders []models.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id IN ?", orderIDs).
			Find(&orders).Error; err != nil {
			return err
		}
		if len(orders) == 0 {
			return nil
		}

		ids := make([]uint, 0, len(orders))
		for _, order := range orders {
			if order.Status != "paid" && order.Status != "refunded" {
				if err := s.releaseDiscountReservationsTx(tx, order.ID); err != nil {
					return err
				}
			}
			ids = append(ids, order.ID)
		}

		result := tx.Delete(&models.Order{}, ids)
		if result.Error != nil {
			return result.Error
		}
		deleted = result.RowsAffected
		return nil
	})
	return deleted, err
}

func (s *OrderService) releaseDiscountReservationsTx(tx *gorm.DB, orderID uint) error {
	if err := discountService.ReleaseCouponUsageForOrderTx(tx, orderID); err != nil {
		return fmt.Errorf("释放优惠券失败: %v", err)
	}
	if err := tx.Model(&models.PromotionParticipation{}).
		Where("order_id = ? AND status = ?", orderID, "pending").
		Updates(map[string]interface{}{
			"order_id": nil,
		}).Error; err != nil {
		return fmt.Errorf("释放营销活动折扣失败: %v", err)
	}
	return nil
}

func (s *OrderService) releaseCompletedDiscountApplicationsTx(tx *gorm.DB, orderID uint) error {
	if err := discountService.ReleaseCouponUsageForOrderTx(tx, orderID); err != nil {
		return fmt.Errorf("释放优惠券失败: %v", err)
	}
	if err := tx.Model(&models.PromotionParticipation{}).
		Where("order_id = ? AND status = ?", orderID, "completed").
		Updates(map[string]interface{}{
			"order_id":   nil,
			"status":     "pending",
			"applied_at": nil,
		}).Error; err != nil {
		return fmt.Errorf("回退营销活动折扣失败: %v", err)
	}
	return nil
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
	couponFreeDays := 0
	finalAmount := baseAmount

	levelDiscountAmount, finalAmount = discountService.CalculateLevelDiscount(s.db, userID, baseAmount)

	// --- 优惠券验证：总量、有效期、每用户上限 ---
	var coupon *models.Coupon
	couponCode := strings.TrimSpace(params.CouponCode)
	if couponCode != "" {
		quote, err := discountService.QuoteCouponForPreparedAmount(s.db, couponCode, userID, params.PackageID, finalAmount)
		if err != nil {
			return nil, "", err
		}
		couponDiscountAmount = quote.CouponDiscountAmount
		couponFreeDays = quote.FreeDays
		finalAmount = quote.FinalAmount
		coupon = quote.Coupon
	}

	// --- 营销活动折扣 ---
	var promotionParticipation *models.PromotionParticipation
	if couponCode == "" {
		// 只有在没有使用优惠券时才应用营销活动折扣
		discount, participation, err := s.applyPromotionDiscount(userID, params.PackageID, finalAmount)
		if err == nil && discount > 0 && participation != nil {
			promotionDiscountAmount = discount
			finalAmount = utils.RoundFloat(finalAmount-promotionDiscountAmount, 2)
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

	// 余额支付但余额不足时直接返回错误，避免在事务中扣除余额后无法完成支付
	if params.PaymentMethod == "balance" && finalAmount > 0 {
		return nil, "", fmt.Errorf("余额不足，请选择其他支付方式或充值后再试")
	}

	orderNo, err := utils.GenerateOrderNo(s.db)
	if err != nil {
		return nil, "", fmt.Errorf("生成订单号失败: %v", err)
	}
	now := utils.GetBeijingTime()

	extraDataMap := map[string]interface{}{
		"duration_months": durationMonths,
	}
	if couponFreeDays > 0 {
		extraDataMap["coupon_free_days"] = couponFreeDays
	}
	if balanceUsed > 0 {
		extraDataMap["balance_used"] = balanceUsed
		extraDataMap["balance_deducted"] = true
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
		if balanceUsed > 0 {
			order.PaymentMethodName = database.NullString("余额支付")
		} else {
			order.PaymentMethodName = database.NullString("优惠抵扣")
		}
	} else {
		methodName := params.PaymentMethod
		if balanceUsed > 0 {
			methodName = fmt.Sprintf("余额支付(%.2f元)+%s", balanceUsed, params.PaymentMethod)
		}
		order.PaymentMethodName = database.NullString(methodName)
	}

	// --- 事务：创建订单 + 记录优惠券使用 + 递增计数器 + 冻结余额 ---
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&order).Error; err != nil {
			return fmt.Errorf("创建订单失败: %v", err)
		}

		if balanceUsed > 0 {
			result := tx.Model(&models.User{}).Where("id = ? AND balance >= ?", user.ID, balanceUsed).
				Update("balance", gorm.Expr("balance - ?", balanceUsed))
			if result.Error != nil {
				return fmt.Errorf("扣除余额失败: %v", result.Error)
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("余额不足")
			}
			orderID := uint(order.ID)
			if err := utils.CreateBalanceLogWithDB(
				tx, user.ID, "consume", -balanceUsed,
				user.Balance, user.Balance-balanceUsed,
				&orderID, nil,
				fmt.Sprintf("订单余额冻结，订单号: %s", order.OrderNo),
				"system", &userID, "",
			); err != nil {
				return fmt.Errorf("记录余额日志失败: %v", err)
			}
		}

		// 记录优惠券使用 + 原子递增 used_quantity（事务内二次校验防并发超卖）
		if coupon != nil {
			if err := discountService.ReserveCouponUsageTx(tx, coupon.ID, userID, order.ID, couponDiscountAmount); err != nil {
				return err
			}
		}

		// 关联营销活动参与记录到订单
		if promotionParticipation != nil {
			result := tx.Model(&models.PromotionParticipation{}).Where("id = ? AND (order_id IS NULL OR order_id = 0)", promotionParticipation.ID).
				Update("order_id", sql.NullInt64{Int64: utils.MustSafeUintToInt64(order.ID), Valid: true})
			if result.Error != nil {
				return fmt.Errorf("关联营销活动失败: %v", result.Error)
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("营销活动折扣已被其他订单占用，请重试")
			}
		}

		return nil
	})
	if txErr != nil {
		return nil, "", txErr
	}

	if finalAmount == 0 {
		paymentMethodName := "优惠抵扣"
		if balanceUsed > 0 {
			paymentMethodName = "余额支付"
		}
		if _, err := s.FinalizePaidOrder(order.OrderNo, FinalizePaidOrderOptions{
			PaymentMethodName: paymentMethodName,
			IPAddress:         params.ClientIP,
		}); err != nil {
			return nil, "", fmt.Errorf("处理余额支付订单失败: %v", err)
		}
		if err := s.db.First(&order, order.ID).Error; err != nil {
			return nil, "", fmt.Errorf("刷新订单状态失败: %v", err)
		}

		go s.sendPaymentSuccessEmail(&user, &order, &pkg, balanceUsed+finalAmount, paymentMethodName)
		return &order, "", nil
	}

	var paymentURL string
	if params.PaymentMethod != "" && params.PaymentMethod != "balance" {
		utils.LogInfo("CreateOrder: 开始生成支付链接 - payment_method=%s, order_no=%s, amount=%.2f",
			params.PaymentMethod, order.OrderNo, finalAmount)
		url, err := s.generatePaymentURLWithUA(&order, params.PaymentMethod, finalAmount, params.UserAgent)
		if err != nil {
			if failedOrder, markErr := s.MarkPendingOrderStatus(order.OrderNo, userID, "failed"); markErr != nil {
				utils.LogError("CreateOrder: mark order failed after payment link error", markErr, map[string]interface{}{
					"order_no": order.OrderNo,
				})
			} else if failedOrder != nil {
				order = *failedOrder
			}
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
	paymentConfig, err := utils.FindEnabledPaymentConfig(s.db, payType)
	if err != nil {
		return "", fmt.Errorf("未找到启用的支付配置")
	}

	transaction := models.PaymentTransaction{
		OrderID:         order.ID,
		UserID:          order.UserID,
		PaymentMethodID: paymentConfig.ID,
		Amount:          int(math.Round(amount * 100)),
		Currency:        "CNY",
		Status:          "pending",
	}
	if err := s.db.Create(&transaction).Error; err != nil {
		return "", fmt.Errorf("创建支付交易失败: %v", err)
	}
	order.PaymentMethodID = database.NullInt64(utils.MustSafeUintToInt64(paymentConfig.ID))
	if err := s.db.Model(order).Updates(map[string]interface{}{
		"payment_method_id":   order.PaymentMethodID,
		"payment_method_name": order.PaymentMethodName,
	}).Error; err != nil {
		utils.LogError("generatePaymentURLWithUA: update order payment method id failed", err, map[string]interface{}{
			"order_no": order.OrderNo,
		})
		return "", fmt.Errorf("更新订单支付方式失败: %v", err)
	}

	markTransactionFailed := func(err error) (string, error) {
		if updateErr := s.db.Model(&models.PaymentTransaction{}).
			Where("id = ? AND status = ?", transaction.ID, "pending").
			Update("status", "failed").Error; updateErr != nil {
			utils.LogError("generatePaymentURLWithUA: mark payment transaction failed", updateErr, map[string]interface{}{
				"order_no":       order.OrderNo,
				"transaction_id": transaction.ID,
			})
		}
		return "", err
	}

	switch paymentConfig.PayType {
	case "alipay":
		svc, err := payment.NewAlipayService(&paymentConfig)
		if err != nil {
			return markTransactionFailed(err)
		}
		url, err := svc.CreatePayment(order, amount)
		if err != nil {
			return markTransactionFailed(err)
		}
		return url, nil
	case "wechat":
		svc, err := payment.NewWechatService(&paymentConfig)
		if err != nil {
			return markTransactionFailed(err)
		}
		url, err := svc.CreatePayment(order, amount)
		if err != nil {
			return markTransactionFailed(err)
		}
		return url, nil
	case "applepay":
		svc, err := payment.NewApplePayService(&paymentConfig)
		if err != nil {
			return markTransactionFailed(err)
		}
		url, err := svc.CreatePayment(order, amount)
		if err != nil {
			return markTransactionFailed(err)
		}
		return url, nil
	case "yipay", "yipay_alipay", "yipay_wxpay", "yipay_qqpay":
		svc, err := payment.NewYipayService(&paymentConfig)
		if err != nil {
			return markTransactionFailed(err)
		}
		paymentType := extractYipayType(payType)
		utils.LogInfo("易支付生成支付链接: payType=%s, extracted_paymentType=%s, order_no=%s", payType, paymentType, order.OrderNo)
		if userAgent != "" {
			utils.LogInfo("易支付使用UserAgent: %s", userAgent)
			url, err := svc.CreatePaymentWithDevice(order, amount, paymentType, userAgent)
			if err != nil {
				return markTransactionFailed(err)
			}
			return url, nil
		}
		url, err := svc.CreatePayment(order, amount, paymentType)
		if err != nil {
			return markTransactionFailed(err)
		}
		return url, nil
	case "codepay", "codepay_alipay", "codepay_wxpay":
		svc, err := payment.NewCodepayService(&paymentConfig)
		if err != nil {
			return markTransactionFailed(err)
		}
		codepayType := extractCodepayType(payType)
		utils.LogInfo("码支付生成支付链接: payType=%s, extracted_paymentType=%s, order_no=%s", payType, codepayType, order.OrderNo)
		url, err := svc.CreatePayment(order, amount, codepayType)
		if err != nil {
			return markTransactionFailed(err)
		}
		return url, nil
	default:
		return markTransactionFailed(fmt.Errorf("不支持的支付方式: %s", paymentConfig.PayType))
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

func (s *OrderService) FinalizePaidOrder(orderNo string, opts FinalizePaidOrderOptions) (*models.Subscription, error) {
	var result *models.Subscription
	var fulfilledNow bool
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var order models.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Package").Where("order_no = ?", orderNo).First(&order).Error; err != nil {
			return fmt.Errorf("订单不存在: %v", err)
		}

		if order.Status != "pending" && order.Status != "paid" {
			return fmt.Errorf("订单状态不允许支付: %s", order.Status)
		}

		now := utils.GetBeijingTime()
		if order.Status != "paid" {
			order.Status = "paid"
		}
		if !order.PaymentTime.Valid {
			order.PaymentTime = database.NullTime(now)
		}
		if opts.PaymentMethodName != "" && (!order.PaymentMethodName.Valid || order.PaymentMethodName.String == "") {
			order.PaymentMethodName = database.NullString(opts.PaymentMethodName)
		}
		if opts.ExternalTransactionID != "" {
			order.PaymentTransactionID = database.NullString(opts.ExternalTransactionID)
		}
		if opts.PaymentMethodID > 0 {
			order.PaymentMethodID = database.NullInt64(utils.MustSafeUintToInt64(opts.PaymentMethodID))
		}
		s.updatePaymentTransactionTx(tx, &order, opts)

		if order.FulfilledAt.Valid {
			if err := tx.Save(&order).Error; err != nil {
				return err
			}
			if sub, err := s.getUserSubscriptionTx(tx, order.UserID); err == nil {
				result = sub
			}
			return nil
		}

		subscription, err := s.processPaidOrderTx(tx, &order, opts)
		if err != nil {
			return err
		}
		order.FulfilledAt = database.NullTime(now)
		if err := tx.Save(&order).Error; err != nil {
			return err
		}

		result = subscription
		fulfilledNow = true
		return nil
	})
	if err != nil {
		return nil, err
	}

	if fulfilledNow {
		if result != nil {
			s.clearSubscriptionCaches(result.UserID, result.SubscriptionURL)
		} else {
			var order models.Order
			if err := s.db.Where("order_no = ?", orderNo).First(&order).Error; err == nil {
				s.clearUserCaches(order.UserID)
			}
		}
	}
	return result, nil
}

func (s *OrderService) updatePaymentTransactionTx(tx *gorm.DB, order *models.Order, opts FinalizePaidOrderOptions) {
	if order == nil {
		return
	}
	var paymentTx models.PaymentTransaction
	query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("order_id = ? AND status = ?", order.ID, "pending")
	if opts.PaymentMethodID > 0 {
		query = query.Where("payment_method_id = ?", opts.PaymentMethodID)
	}
	if err := query.Order("created_at DESC").First(&paymentTx).Error; err != nil {
		return
	}

	paymentTx.Status = "success"
	if opts.ExternalTransactionID != "" {
		paymentTx.ExternalTransactionID = database.NullString(opts.ExternalTransactionID)
	}
	if opts.CallbackData != "" {
		paymentTx.CallbackData = database.NullString(opts.CallbackData)
	}
	if err := tx.Save(&paymentTx).Error; err != nil {
		utils.LogError("FinalizePaidOrder: failed to update payment transaction", err, map[string]interface{}{
			"order_no": order.OrderNo,
			"order_id": order.ID,
		})
	}
}

func (s *OrderService) ProcessPaidOrder(order *models.Order) (*models.Subscription, error) {
	if order == nil {
		return nil, fmt.Errorf("订单不能为空")
	}
	if order.OrderNo != "" {
		return s.FinalizePaidOrder(order.OrderNo, FinalizePaidOrderOptions{})
	}
	return nil, fmt.Errorf("订单号不能为空")
}

func (s *OrderService) calculateOrderPaidAmount(order *models.Order, balanceUsed float64) float64 {
	if order == nil {
		return 0
	}
	paidAmount := order.Amount
	if order.FinalAmount.Valid {
		paidAmount = order.FinalAmount.Float64
	}
	if balanceUsed > 0 {
		paidAmount = utils.RoundFloat(paidAmount+balanceUsed, 2)
	}

	maxPaidAmount := order.Amount
	if order.DiscountAmount.Valid && order.DiscountAmount.Float64 > 0 {
		maxPaidAmount = utils.RoundFloat(order.Amount-order.DiscountAmount.Float64, 2)
		if maxPaidAmount < 0 {
			maxPaidAmount = 0
		}
	}
	if maxPaidAmount >= 0 && paidAmount > maxPaidAmount {
		paidAmount = maxPaidAmount
	}
	if paidAmount < 0 {
		paidAmount = 0
	}
	return utils.RoundFloat(paidAmount, 2)
}

func (s *OrderService) processPaidOrderTx(tx *gorm.DB, order *models.Order, opts FinalizePaidOrderOptions) (*models.Subscription, error) {
	if order.Status != "paid" {
		return nil, fmt.Errorf("订单状态未支付")
	}

	var user models.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, order.UserID).Error; err != nil {
		return nil, fmt.Errorf("用户不存在: %v", err)
	}

	// 从订单的 ExtraData 中提取已预留的余额抵扣；opts.BalanceAmount 表示本次支付动作额外使用余额。
	var storedBalanceUsed float64
	var balanceDeducted bool
	var extraData map[string]interface{}
	if order.ExtraData.Valid && order.ExtraData.String != "" {
		if err := json.Unmarshal([]byte(order.ExtraData.String), &extraData); err == nil {
			if balance, ok := extraData["balance_used"].(float64); ok {
				storedBalanceUsed = balance
			}
			if deducted, ok := extraData["balance_deducted"].(bool); ok {
				balanceDeducted = deducted
			}
		}
	}
	additionalBalanceUsed := utils.RoundFloat(opts.BalanceAmount, 2)
	if additionalBalanceUsed < 0 {
		additionalBalanceUsed = 0
	}
	balanceUsed := utils.RoundFloat(storedBalanceUsed+additionalBalanceUsed, 2)

	orderType := ""
	if extraData != nil {
		if rawType, ok := extraData["type"].(string); ok {
			orderType = rawType
		}
	}

	if orderType == "device_upgrade" {
		var existingSubscription models.Subscription
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", user.ID).First(&existingSubscription).Error; err == nil {
			additionalDevices := 0
			additionalDays := 0
			if devices, ok := extraData["additional_devices"].(float64); ok {
				additionalDevices = int(devices)
			}
			if days, ok := extraData["additional_days"].(float64); ok {
				additionalDays = int(days)
			}
			if additionalDevices <= 0 && additionalDays <= 0 {
				return nil, fmt.Errorf("设备升级参数无效: additional_devices=%d, additional_days=%d", additionalDevices, additionalDays)
			}
		}
	}

	if balanceDeducted {
		if additionalBalanceUsed > 0 {
			oldBalance := user.Balance
			result := tx.Model(&models.User{}).Where("id = ? AND balance >= ?", user.ID, additionalBalanceUsed).
				Update("balance", gorm.Expr("balance - ?", additionalBalanceUsed))
			if result.Error != nil {
				return nil, fmt.Errorf("扣除余额失败: %v", result.Error)
			}
			if result.RowsAffected == 0 {
				return nil, fmt.Errorf("余额不足，无法完成支付")
			}
			orderID := uint(order.ID)
			userID := user.ID
			if err := utils.CreateBalanceLogWithDB(
				tx, user.ID, "consume", -additionalBalanceUsed,
				oldBalance, oldBalance-additionalBalanceUsed,
				&orderID, nil,
				fmt.Sprintf("订单支付扣除余额，订单号: %s", order.OrderNo),
				"user", &userID, opts.IPAddress,
			); err != nil {
				return nil, fmt.Errorf("记录余额日志失败: %v", err)
			}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, order.UserID).Error; err != nil {
				return nil, fmt.Errorf("重新加载用户信息失败: %v", err)
			}
		}
	} else if balanceUsed > 0 {
		oldBalance := user.Balance
		result := tx.Model(&models.User{}).Where("id = ? AND balance >= ?", user.ID, balanceUsed).
			Update("balance", gorm.Expr("balance - ?", balanceUsed))
		if result.Error != nil {
			return nil, fmt.Errorf("扣除余额失败: %v", result.Error)
		}
		if result.RowsAffected == 0 {
			return nil, fmt.Errorf("余额不足，无法完成支付")
		}
		orderID := uint(order.ID)
		userID := user.ID
		if err := utils.CreateBalanceLogWithDB(
			tx,
			user.ID,
			"consume",
			-balanceUsed,
			oldBalance,
			oldBalance-balanceUsed,
			&orderID,
			nil,
			fmt.Sprintf("订单支付扣除余额，订单号: %s", order.OrderNo),
			"user",
			&userID,
			opts.IPAddress,
		); err != nil {
			return nil, fmt.Errorf("记录余额日志失败: %v", err)
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, order.UserID).Error; err != nil {
			return nil, fmt.Errorf("重新加载用户信息失败: %v", err)
		}
	}

	if additionalBalanceUsed > 0 {
		if extraData == nil {
			extraData = make(map[string]interface{})
		}
		extraData["balance_used"] = balanceUsed
		if encodedExtra, err := json.Marshal(extraData); err == nil {
			order.ExtraData = database.NullString(string(encodedExtra))
		}
		if opts.PaymentMethodName == "余额支付" {
			order.PaymentMethodName = database.NullString("余额支付")
		}
	}

	paidAmount := s.calculateOrderPaidAmount(order, balanceUsed)

	user.TotalConsumption = utils.RoundFloat(user.TotalConsumption+paidAmount, 2)
	if err := tx.Save(&user).Error; err != nil {
		return nil, fmt.Errorf("更新用户累计消费失败: %v", err)
	}

	s.updateUserLevelTx(tx, &user)

	s.processInviteRewardsTx(tx, order, paidAmount)

	// 更新营销活动参与记录状态为已完成
	if err := tx.Model(&models.PromotionParticipation{}).
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
		result, err = s.processPackageOrderTx(tx, order, &user)
	} else {
		result, err = s.processDeviceUpgradeOrderTx(tx, order, &user)
	}

	return result, err
}

func (s *OrderService) clearSubscriptionCaches(userID uint, subscriptionURL string) {
	go func() {
		cs := cache_service.NewCacheService()
		if cacheErr := cs.ClearUserCache(userID); cacheErr != nil {
			log.Printf("failed to clear user cache: %v", cacheErr)
		}
		if cacheErr := cs.ClearUserSubscriptionCache(userID); cacheErr != nil {
			log.Printf("failed to clear user subscription cache: %v", cacheErr)
		}
		if subscriptionURL != "" {
			if cacheErr := cache.ClearSubscriptionConfigCache(subscriptionURL); cacheErr != nil {
				log.Printf("failed to clear subscription config cache: %v", cacheErr)
			}
		}
	}()
}

func (s *OrderService) clearUserCaches(userID uint) {
	if userID == 0 {
		return
	}
	go func() {
		cs := cache_service.NewCacheService()
		if cacheErr := cs.ClearUserCache(userID); cacheErr != nil {
			log.Printf("failed to clear user cache: %v", cacheErr)
		}
	}()
}

func (s *OrderService) getUserSubscriptionTx(tx *gorm.DB, userID uint) (*models.Subscription, error) {
	var subscription models.Subscription
	if err := tx.Where("user_id = ?", userID).First(&subscription).Error; err != nil {
		return nil, err
	}
	return &subscription, nil
}

func (s *OrderService) processPackageOrder(order *models.Order, user *models.User) (*models.Subscription, error) {
	return s.processPackageOrderTx(s.db, order, user)
}

func (s *OrderService) processPackageOrderTx(tx *gorm.DB, order *models.Order, user *models.User) (*models.Subscription, error) {
	// 检查是否是自定义套餐
	isCustomPackage := false
	var customDevices int
	var customMonths int
	var couponFreeDays int
	var parsedExtraData map[string]interface{}

	if order.ExtraData.Valid && order.ExtraData.String != "" {
		var extraData map[string]interface{}
		if err := json.Unmarshal([]byte(order.ExtraData.String), &extraData); err == nil {
			parsedExtraData = extraData
			if orderType, ok := extraData["type"].(string); ok && orderType == "custom_package" {
				isCustomPackage = true
				if devices, ok := extraData["devices"].(float64); ok {
					customDevices = int(devices)
				}
				if months, ok := extraData["months"].(float64); ok {
					customMonths = int(months)
				}
			}
			if days, ok := extraData["coupon_free_days"].(float64); ok {
				couponFreeDays = int(days)
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
		if err := tx.First(&pkg, order.PackageID).Error; err != nil {
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
	if couponFreeDays > 0 {
		totalDurationDays += couponFreeDays
	}

	now := utils.GetBeijingTime()

	var subscription models.Subscription
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		subscriptionURL := utils.GenerateSubscriptionURL()
		expireTime := now.AddDate(0, 0, totalDurationDays)
		var pkgID *int64
		if !isCustomPackage {
			id := utils.MustSafeUintToInt64(pkg.ID)
			pkgID = &id
		}
		if isCustomPackage {
			if parsedExtraData == nil {
				parsedExtraData = make(map[string]interface{})
			}
			parsedExtraData["duration_days"] = totalDurationDays
			parsedExtraData["new_device_limit"] = deviceLimit
			parsedExtraData["new_expire_time"] = utils.FormatBeijingTime(expireTime)
			parsedExtraData["activation_mode"] = "create"
			parsedExtraData["had_existing_subscription"] = false
			if encodedExtra, err := json.Marshal(parsedExtraData); err == nil {
				order.ExtraData = database.NullString(string(encodedExtra))
			}
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
		if err := tx.Create(&subscription).Error; err != nil {
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
		oldDeviceLimit := subscription.DeviceLimit
		var oldPackageID int64
		if subscription.PackageID != nil {
			oldPackageID = *subscription.PackageID
		}
		if isCustomPackage {
			// 自定义套餐按本次购买时长重新开通，避免把已有剩余时间重复叠加成双倍时长。
			subscription.ExpireTime = now.AddDate(0, 0, totalDurationDays)
		} else if subscription.ExpireTime.Before(now) {
			subscription.ExpireTime = now.AddDate(0, 0, totalDurationDays)
		} else {
			subscription.ExpireTime = subscription.ExpireTime.AddDate(0, 0, totalDurationDays)
		}
		subscription.DeviceLimit = deviceLimit
		subscription.IsActive = true
		subscription.Status = "active"
		if isCustomPackage {
			subscription.PackageID = nil
			if parsedExtraData == nil {
				parsedExtraData = make(map[string]interface{})
			}
			parsedExtraData["old_device_limit"] = oldDeviceLimit
			parsedExtraData["new_device_limit"] = deviceLimit
			parsedExtraData["old_expire_time"] = utils.FormatBeijingTime(oldExpireTime)
			parsedExtraData["new_expire_time"] = utils.FormatBeijingTime(subscription.ExpireTime)
			parsedExtraData["old_expire_time_rfc3339"] = oldExpireTime.Format(time.RFC3339Nano)
			parsedExtraData["new_expire_time_rfc3339"] = subscription.ExpireTime.Format(time.RFC3339Nano)
			parsedExtraData["old_package_id"] = oldPackageID
			parsedExtraData["duration_days"] = totalDurationDays
			parsedExtraData["activation_mode"] = "replace"
			parsedExtraData["had_existing_subscription"] = true
			if encodedExtra, err := json.Marshal(parsedExtraData); err == nil {
				order.ExtraData = database.NullString(string(encodedExtra))
			}
		} else {
			pkgID := utils.MustSafeUintToInt64(pkg.ID)
			subscription.PackageID = &pkgID
		}

		if err := tx.Save(&subscription).Error; err != nil {
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
	return s.processDeviceUpgradeOrderTx(s.db, order, user)
}

func (s *OrderService) processDeviceUpgradeOrderTx(tx *gorm.DB, order *models.Order, user *models.User) (*models.Subscription, error) {
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
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
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

	if err := tx.Save(&subscription).Error; err != nil {
		return nil, fmt.Errorf("升级订阅失败: %v", err)
	}

	if utils.AppLogger != nil {
		utils.AppLogger.Info("ProcessPaidOrder: ✅ 设备升级成功 - user_id=%d, additional_devices=%d, additional_days=%d, device_limit=%d, expire_time=%s",
			user.ID, additionalDevices, additionalDays, subscription.DeviceLimit, utils.FormatBeijingTime(subscription.ExpireTime))
	}

	return &subscription, nil
}

func (s *OrderService) updateUserLevel(user *models.User) {
	s.updateUserLevelTx(s.db, user)
}

func (s *OrderService) updateUserLevelTx(tx *gorm.DB, user *models.User) {
	var userLevels []models.UserLevel
	if err := tx.Where("is_active = ?", true).Order("level_order ASC").Find(&userLevels).Error; err == nil {
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
					if err := tx.First(&currentLevel, user.UserLevelID.Int64).Error; err == nil {
						if currentLevel.LevelOrder < targetLevel.LevelOrder {
							shouldUpgrade = false
						}
					}
				}
				if shouldUpgrade {
					user.UserLevelID = sql.NullInt64{Int64: utils.MustSafeUintToInt64(targetLevel.ID), Valid: true}
					if err := tx.Save(user).Error; err != nil {
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
	s.processInviteRewardsTx(s.db, order, paidAmount)
}

func (s *OrderService) processInviteRewardsTx(tx *gorm.DB, order *models.Order, paidAmount float64) {
	var inviteRelation models.InviteRelation
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("invitee_id = ? AND (inviter_reward_given = ? OR invitee_reward_given = ?)",
		order.UserID, false, false).First(&inviteRelation).Error; err != nil {
		return
	}

	// 仅首单发放：如果已有首单记录，跳过
	if inviteRelation.InviteeFirstOrderID.Valid {
		return
	}

	var inviteCode models.InviteCode
	if err := tx.First(&inviteCode, inviteRelation.InviteCodeID).Error; err != nil {
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
		tx.Model(&models.Order{}).Where("user_id = ? AND status = ?", order.UserID, "paid").Count(&orderCount)
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
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&inviter, inviteRelation.InviterID).Error; err == nil {
			oldBalance := inviter.Balance
			result := tx.Model(&models.User{}).Where("id = ?", inviter.ID).
				Updates(map[string]interface{}{
					"balance":             gorm.Expr("balance + ?", inviteRelation.InviterRewardAmount),
					"total_invite_reward": gorm.Expr("total_invite_reward + ?", inviteRelation.InviterRewardAmount),
					"total_invite_count":  gorm.Expr("total_invite_count + 1"),
				})
			if result.Error == nil {
				inviteRelation.InviterRewardGiven = true
				var freshInviter models.User
				tx.First(&freshInviter, inviter.ID)
				if utils.AppLogger != nil {
					utils.AppLogger.Info("processInviteRewards: ✅ 发放邀请者奖励 - inviter_id=%d, amount=%.2f, order_id=%d",
						inviter.ID, inviteRelation.InviterRewardAmount, order.ID)
				}
				if err := utils.CreateBalanceLogWithDB(
					tx,
					inviter.ID, "commission", inviteRelation.InviterRewardAmount,
					oldBalance, freshInviter.Balance, nil, nil,
					fmt.Sprintf("邀请奖励: 邀请人奖励 (订单 %s)", order.OrderNo),
					"system", nil, "",
				); err != nil {
					log.Printf("failed to create balance log: %v", err)
				}
				relationID := uint(inviteRelation.ID)
				if err := utils.CreateCommissionLogWithDB(
					tx,
					inviter.ID, order.UserID, "order_reward",
					inviteRelation.InviterRewardAmount, &relationID, nil,
					fmt.Sprintf("邀请人奖励: 订单 %s", order.OrderNo),
				); err != nil {
					log.Printf("failed to create commission log: %v", err)
				}
			} else {
				utils.LogError("processInviteRewards: failed to give inviter reward", result.Error, map[string]interface{}{
					"inviter_id": inviter.ID, "amount": inviteRelation.InviterRewardAmount,
				})
			}
		}
	}

	if !inviteRelation.InviteeRewardGiven && inviteRelation.InviteeRewardAmount > 0 {
		var invitee models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&invitee, order.UserID).Error; err == nil {
			oldBalance := invitee.Balance
			result := tx.Model(&models.User{}).Where("id = ?", invitee.ID).
				Update("balance", gorm.Expr("balance + ?", inviteRelation.InviteeRewardAmount))
			if result.Error == nil {
				inviteRelation.InviteeRewardGiven = true
				var freshInvitee models.User
				tx.First(&freshInvitee, invitee.ID)
				if utils.AppLogger != nil {
					utils.AppLogger.Info("processInviteRewards: ✅ 发放被邀请者奖励 - invitee_id=%d, amount=%.2f, order_id=%d",
						invitee.ID, inviteRelation.InviteeRewardAmount, order.ID)
				}
				if err := utils.CreateBalanceLogWithDB(
					tx,
					invitee.ID, "commission", inviteRelation.InviteeRewardAmount,
					oldBalance, freshInvitee.Balance, nil, nil,
					fmt.Sprintf("邀请奖励: 被邀请人奖励 (订单 %s)", order.OrderNo),
					"system", nil, "",
				); err != nil {
					log.Printf("failed to create balance log: %v", err)
				}
				relationID := uint(inviteRelation.ID)
				if err := utils.CreateCommissionLogWithDB(
					tx,
					inviteRelation.InviterID, invitee.ID, "order_reward",
					inviteRelation.InviteeRewardAmount, &relationID, nil,
					fmt.Sprintf("被邀请人奖励: 订单 %s", order.OrderNo),
				); err != nil {
					log.Printf("failed to create commission log: %v", err)
				}
			} else {
				utils.LogError("processInviteRewards: failed to give invitee reward", result.Error, map[string]interface{}{
					"invitee_id": invitee.ID, "amount": inviteRelation.InviteeRewardAmount,
				})
			}
		}
	}

	if err := tx.Save(&inviteRelation).Error; err != nil {
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
	paidAmount := s.calculateOrderPaidAmount(order, balanceUsed)

	// 回退用户累计消费
	if user.TotalConsumption >= paidAmount {
		user.TotalConsumption = utils.RoundFloat(user.TotalConsumption-paidAmount, 2)
	} else {
		user.TotalConsumption = 0
	}

	// 回退订阅或设备升级
	isCustomPackage := false
	if order.ExtraData.Valid && order.ExtraData.String != "" {
		var extraData map[string]interface{}
		if err := json.Unmarshal([]byte(order.ExtraData.String), &extraData); err == nil {
			if orderType, ok := extraData["type"].(string); ok && orderType == "custom_package" {
				isCustomPackage = true
			}
		}
	}

	if order.PackageID > 0 || isCustomPackage {
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

	if err := s.releaseCompletedDiscountApplicationsTx(s.db, order.ID); err != nil {
		return err
	}

	// 更新用户信息
	if err := s.db.Save(&user).Error; err != nil {
		return fmt.Errorf("更新用户信息失败: %v", err)
	}

	// 更新订单状态
	order.Status = "refunded"
	if err := s.db.Save(order).Error; err != nil {
		return fmt.Errorf("更新订单状态失败: %v", err)
	}

	if sub, err := s.getUserSubscriptionTx(s.db, order.UserID); err == nil {
		s.clearSubscriptionCaches(sub.UserID, sub.SubscriptionURL)
	} else {
		s.clearUserCaches(order.UserID)
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

	isCustomPackage := false
	durationMonths := 1
	couponFreeDays := 0
	var extraData map[string]interface{}
	if order.ExtraData.Valid && order.ExtraData.String != "" {
		if err := json.Unmarshal([]byte(order.ExtraData.String), &extraData); err == nil {
			if orderType, ok := extraData["type"].(string); ok && orderType == "custom_package" {
				isCustomPackage = true
				if months, ok := extraData["months"].(float64); ok {
					durationMonths = int(months)
				}
			}
			if months, ok := extraData["duration_months"].(float64); ok {
				durationMonths = int(months)
			}
			if days, ok := extraData["coupon_free_days"].(float64); ok {
				couponFreeDays = int(days)
			}
		}
	}
	if durationMonths <= 0 {
		durationMonths = 1
	}

	var packageID uint
	var totalDurationDays int
	if isCustomPackage {
		totalDurationDays = durationMonths * 30
	} else {
		var pkg models.Package
		if err := s.db.First(&pkg, order.PackageID).Error; err != nil {
			return fmt.Errorf("套餐不存在: %v", err)
		}
		packageID = pkg.ID
		totalDurationDays = pkg.DurationDays * durationMonths
	}
	if couponFreeDays > 0 {
		totalDurationDays += couponFreeDays
	}
	now := utils.GetBeijingTime()

	if isCustomPackage && extraData != nil {
		if activationMode, _ := extraData["activation_mode"].(string); activationMode == "replace" {
			if oldExpireTimeStr, ok := extraData["old_expire_time"].(string); ok && oldExpireTimeStr != "" {
				if exactOldExpireTimeStr, ok := extraData["old_expire_time_rfc3339"].(string); ok && exactOldExpireTimeStr != "" {
					oldExpireTimeStr = exactOldExpireTimeStr
				}
				oldExpireTime, err := time.Parse(time.RFC3339Nano, oldExpireTimeStr)
				if err != nil {
					oldExpireTime, err = time.ParseInLocation(orderTimeLayout, oldExpireTimeStr, utils.BeijingTZ)
				}
				if err != nil {
					return fmt.Errorf("自定义套餐旧到期时间无效: %v", err)
				}
				subscription.ExpireTime = oldExpireTime
				if oldDeviceLimit, ok := extraData["old_device_limit"].(float64); ok {
					subscription.DeviceLimit = int(oldDeviceLimit)
				}
				if oldPackageID, ok := extraData["old_package_id"].(float64); ok && oldPackageID > 0 {
					pkgID := int64(oldPackageID)
					subscription.PackageID = &pkgID
				} else {
					subscription.PackageID = nil
				}
				if subscription.ExpireTime.After(now) {
					subscription.IsActive = true
					subscription.Status = "active"
				} else {
					subscription.IsActive = false
					subscription.Status = "expired"
				}
				if err := s.db.Save(&subscription).Error; err != nil {
					return fmt.Errorf("恢复自定义套餐前订阅失败: %v", err)
				}
				utils.LogInfo("ProcessRefundOrder: 恢复自定义套餐前订阅成功 - user_id=%d, duration_days=%d, expire_time=%s",
					user.ID, totalDurationDays, utils.FormatBeijingTime(subscription.ExpireTime))
				return nil
			}
		}
	}

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
		user.ID, packageID, totalDurationDays, utils.FormatBeijingTime(subscription.ExpireTime))
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
	return promotionService.NewService(s.db).ApplyDiscount(userID, packageID, baseAmount)
}

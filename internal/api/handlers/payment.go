package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/notification"
	orderServicePkg "cboard-go/internal/services/order"
	"cboard-go/internal/services/payment"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetPaymentMethods(c *gin.Context) {
	db := database.GetDB()
	var cfg []models.PaymentConfig
	db.Where("status = ?", 1).Order("sort_order ASC").Find(&cfg)
	res := make([]gin.H, 0, len(cfg))
	mMap := map[string]string{"alipay": "支付宝", "wechat": "微信支付", "yipay": "易支付", "paypal": "PayPal", "applepay": "Apple Pay", "stripe": "Stripe", "bank": "银行转账"}
	for _, m := range cfg {
		name := mMap[m.PayType]
		if name == "" {
			name = m.PayType
		}
		res = append(res, gin.H{"id": m.ID, "key": m.PayType, "name": name, "status": m.Status})
	}
	utils.SuccessResponse(c, http.StatusOK, "", res)
}

func CreatePayment(c *gin.Context) {
	u, _ := middleware.GetCurrentUser(c)
	var req struct {
		OrderID         uint `json:"order_id"`
		PaymentMethodID uint `json:"payment_method_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}
	db := database.GetDB()
	var order models.Order
	if err := db.Where("id = ? AND user_id = ?", req.OrderID, u.ID).First(&order).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "订单不存在", err)
		return
	}
	if order.Status != "pending" {
		utils.ErrorResponse(c, http.StatusBadRequest, "订单不可支付", nil)
		return
	}
	var cfg models.PaymentConfig
	if err := db.First(&cfg, req.PaymentMethodID).Error; err != nil || cfg.Status != 1 {
		utils.ErrorResponse(c, http.StatusNotFound, "支付方式无效", err)
		return
	}
	amt := int(order.Amount * 100)
	if order.FinalAmount.Valid {
		amt = int(order.FinalAmount.Float64 * 100)
	}
	tx := models.PaymentTransaction{OrderID: order.ID, UserID: u.ID, PaymentMethodID: cfg.ID, Amount: amt, Status: "pending"}
	db.Create(&tx)
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"transaction_id": tx.ID, "amount": float64(amt) / 100})
}

// PaymentNotify 支付回调
func PaymentNotify(c *gin.Context) {
	paymentType := c.Param("type") // alipay, wechat, etc.
	db := database.GetDB()

	params := make(map[string]string)
	if err := c.Request.ParseForm(); err == nil {
		for k, v := range c.Request.PostForm {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
	}
	if len(params) == 0 {
		for k, v := range c.Request.URL.Query() {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
	}

	var paymentConfig models.PaymentConfig
	if err := db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", paymentType, 1).First(&paymentConfig).Error; err != nil {
		utils.LogError("PaymentNotify: payment config not found", err, map[string]interface{}{
			"payment_type": paymentType,
		})
		c.String(http.StatusBadRequest, "支付配置不存在")
		return
	}

	// 验证签名
	var verified bool
	switch paymentType {
	case "alipay":
		alipayService, err := payment.NewAlipayService(&paymentConfig)
		if err == nil {
			verified = alipayService.VerifyNotify(params)
		}
	case "wechat":
		wechatService, err := payment.NewWechatService(&paymentConfig)
		if err == nil {
			verified = wechatService.VerifyNotify(params)
		}
	case "paypal":
		paypalService, err := payment.NewPayPalService(&paymentConfig)
		if err == nil {
			verified = paypalService.VerifyNotify(params)
		}
	case "applepay":
		applePayService, err := payment.NewApplePayService(&paymentConfig)
		if err == nil {
			verified = applePayService.VerifyNotify(params)
		}
	}

	if !verified {
		// 记录签名验证失败（用于安全审计）
		utils.LogError("PaymentNotify: signature verification failed", nil, map[string]interface{}{
			"payment_type": paymentType,
			"order_no":     params["out_trade_no"],
		})
		c.String(http.StatusBadRequest, "签名验证失败")
		return
	}

	orderNo := params["out_trade_no"]
	externalTransactionID := params["trade_no"] // 支付宝/微信的交易号

	// 支付宝回调中，trade_status 字段表示交易状态
	// TRADE_SUCCESS: 交易成功
	// TRADE_FINISHED: 交易完成
	// WAIT_BUYER_PAY: 等待买家付款
	// TRADE_CLOSED: 交易关闭
	if paymentType == "alipay" {
		tradeStatus := params["trade_status"]
		if tradeStatus != "TRADE_SUCCESS" && tradeStatus != "TRADE_FINISHED" {
			// 如果不是成功状态，记录日志但返回success，避免支付宝重复回调
			utils.LogError("PaymentNotify: trade status not success", nil, map[string]interface{}{
				"payment_type": paymentType,
				"order_no":     orderNo,
				"trade_status": tradeStatus,
			})
			c.String(http.StatusOK, "success")
			return
		}
	}

	if orderNo == "" {
		utils.LogError("PaymentNotify: missing order number", nil, map[string]interface{}{
			"payment_type": paymentType,
		})
		c.String(http.StatusBadRequest, "订单号不存在")
		return
	}

	// 记录支付回调日志
	utils.LogInfo("PaymentNotify: 收到支付回调 - payment_type=%s, order_no=%s, external_transaction_id=%s",
		paymentType, orderNo, externalTransactionID)

	var order models.Order
	var recharge models.RechargeRecord
	isRecharge := false

	// 先尝试查找订单
	if err := db.Preload("Package").Where("order_no = ?", orderNo).First(&order).Error; err != nil {
		// 如果不是订单，尝试查找充值记录
		if err2 := db.Where("order_no = ?", orderNo).First(&recharge).Error; err2 == nil {
			isRecharge = true
		} else {
			utils.LogError("PaymentNotify: order or recharge not found", err, map[string]interface{}{
				"order_no": orderNo,
			})
			c.String(http.StatusBadRequest, "订单或充值记录不存在")
			return
		}
	}

	if isRecharge {
		if externalTransactionID != "" {
			var existingTransaction models.PaymentTransaction
			if err := db.Where("external_transaction_id = ? AND status = ?", externalTransactionID, "success").First(&existingTransaction).Error; err == nil {
				c.String(http.StatusOK, "success")
				return
			}
		}
		// 验证充值金额
		if paymentType == "alipay" {
			if amountStr, ok := params["total_amount"]; ok {
				var callbackAmount float64
				fmt.Sscanf(amountStr, "%f", &callbackAmount)
				if callbackAmount < recharge.Amount-0.01 || callbackAmount > recharge.Amount+0.01 {
					utils.LogError("PaymentNotify: recharge amount mismatch", nil, map[string]interface{}{
						"order_no":        orderNo,
						"expected_amount": recharge.Amount,
						"callback_amount": callbackAmount,
					})
					c.String(http.StatusBadRequest, "充值金额不匹配")
					return
				}
			}
		}

		if recharge.Status == "paid" {
			c.String(http.StatusOK, "success")
			return
		}

		// 使用事务处理充值
		err := utils.WithTransaction(db, func(tx *gorm.DB) error {
			recharge.Status = "paid"
			recharge.PaidAt = database.NullTime(utils.GetBeijingTime())
			if externalTransactionID != "" {
				recharge.PaymentTransactionID = database.NullString(externalTransactionID)
			}
			if err := tx.Save(&recharge).Error; err != nil {
				utils.LogError("PaymentNotify: failed to update recharge", err, map[string]interface{}{
					"order_no": orderNo,
				})
				return err
			}

			var user models.User
			if err := tx.First(&user, recharge.UserID).Error; err == nil {
				user.Balance += recharge.Amount
				if err := tx.Save(&user).Error; err != nil {
					utils.LogError("PaymentNotify: failed to update user balance", err, map[string]interface{}{
						"order_no": orderNo,
						"user_id":  user.ID,
					})
					return err
				}
			}
			return nil
		})

		if err != nil {
			utils.LogError("PaymentNotify: failed to process recharge transaction", err, map[string]interface{}{
				"order_no": orderNo,
			})
			c.String(http.StatusInternalServerError, "处理失败")
			return
		}

		c.String(http.StatusOK, "success")
		return
	}

	// 验证订单金额（防止金额篡改）
	if paymentType == "alipay" {
		// 支付宝回调中的金额（转换为元）
		if amountStr, ok := params["total_amount"]; ok {
			var callbackAmount float64
			fmt.Sscanf(amountStr, "%f", &callbackAmount)
			// 混合支付时，回调金额可能只是第三方支付部分，需要加上余额部分
			expectedAmount := order.Amount
			if order.FinalAmount.Valid {
				expectedAmount = order.FinalAmount.Float64
			}

			var balanceUsedInOrder float64 = 0
			if order.ExtraData.Valid && order.ExtraData.String != "" {
				var extraData map[string]interface{}
				if err := json.Unmarshal([]byte(order.ExtraData.String), &extraData); err == nil {
					if balanceUsedVal, ok := extraData["balance_used"].(float64); ok {
						balanceUsedInOrder = balanceUsedVal
					}
				}
			}

			expectedCallbackAmount := expectedAmount - balanceUsedInOrder
			if balanceUsedInOrder > 0 {
				if callbackAmount < expectedCallbackAmount-0.01 || callbackAmount > expectedCallbackAmount+0.01 {
					utils.LogError("PaymentNotify: amount mismatch (mixed payment)", nil, map[string]interface{}{
						"order_no":              orderNo,
						"expected_callback":     expectedCallbackAmount,
						"callback_amount":       callbackAmount,
						"balance_used":          balanceUsedInOrder,
						"total_expected_amount": expectedAmount,
					})
					c.String(http.StatusBadRequest, "订单金额不匹配")
					return
				}
			} else {
				if callbackAmount < expectedAmount-0.01 || callbackAmount > expectedAmount+0.01 {
					utils.LogError("PaymentNotify: amount mismatch", nil, map[string]interface{}{
						"order_no":        orderNo,
						"expected_amount": expectedAmount,
						"callback_amount": callbackAmount,
					})
					c.String(http.StatusBadRequest, "订单金额不匹配")
					return
				}
			}
		}
	}

	// 幂等性检查：如果订单已支付，直接返回成功
	if order.Status == "paid" {
		utils.LogError("PaymentNotify: order already paid", nil, map[string]interface{}{
			"order_no": orderNo,
		})
		c.String(http.StatusOK, "success")
		return
	}

	err := utils.WithTransaction(db, func(tx *gorm.DB) error {
		order.Status = "paid"
		order.PaymentTime = database.NullTime(utils.GetBeijingTime())
		if err := tx.Save(&order).Error; err != nil {
			utils.LogError("PaymentNotify: failed to update order", err, map[string]interface{}{
				"order_no": orderNo,
			})
			return err
		}

		var transaction models.PaymentTransaction
		if err := tx.Where("order_id = ?", order.ID).First(&transaction).Error; err == nil {
			transaction.Status = "success"
			if externalTransactionID != "" {
				transaction.ExternalTransactionID = database.NullString(externalTransactionID)
			}
			if callbackData, err := json.Marshal(params); err == nil {
				transaction.CallbackData = database.NullString(string(callbackData))
			}
			if err := tx.Save(&transaction).Error; err != nil {
				utils.LogError("PaymentNotify: failed to update transaction", err, map[string]interface{}{
					"order_no": orderNo,
				})
				return err
			}
		}
		return nil
	})

	if err != nil {
		utils.LogError("PaymentNotify: failed to process payment transaction", err, map[string]interface{}{
			"order_no": orderNo,
		})
		c.String(http.StatusInternalServerError, "处理失败")
		return
	}

	var balanceUsed float64 = 0
	if order.ExtraData.Valid && order.ExtraData.String != "" {
		var extraData map[string]interface{}
		if err := json.Unmarshal([]byte(order.ExtraData.String), &extraData); err == nil {
			if balanceUsedVal, ok := extraData["balance_used"].(float64); ok {
				balanceUsed = balanceUsedVal
			}
		}
	}

	if balanceUsed > 0 {
		var user models.User
		if err := db.First(&user, order.UserID).Error; err == nil {
			if user.Balance >= balanceUsed {
				user.Balance -= balanceUsed
				if err := db.Save(&user).Error; err != nil {
					utils.LogError("PaymentNotify: failed to deduct balance", err, map[string]interface{}{
						"order_id":     order.ID,
						"balance_used": balanceUsed,
					})
				} else {
					utils.LogError("PaymentNotify: balance deducted", nil, map[string]interface{}{
						"order_id":     order.ID,
						"balance_used": balanceUsed,
						"user_id":      user.ID,
					})
				}
			} else {
				utils.LogError("PaymentNotify: insufficient balance", nil, map[string]interface{}{
					"order_id":     order.ID,
					"balance_used": balanceUsed,
					"user_balance": user.Balance,
				})
			}
		}
	}

	// ProcessPaidOrder 统一处理所有订单类型
	orderService := orderServicePkg.NewOrderService()
	_, processErr := orderService.ProcessPaidOrder(&order)
	if processErr != nil {
		utils.LogError("PaymentNotify: process paid order failed", processErr, map[string]interface{}{
			"order_id": order.ID,
		})
		// 支付已成功，后续可通过补偿机制修复
	}

	go func() {
		var latestOrder models.Order
		if err := db.Preload("Package").Where("id = ?", order.ID).First(&latestOrder).Error; err != nil {
			return
		}
		var latestUser models.User
		if err := db.First(&latestUser, latestOrder.UserID).Error; err != nil {
			return
		}

		paymentTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
		paidAmount := latestOrder.Amount
		if latestOrder.FinalAmount.Valid {
			paidAmount = latestOrder.FinalAmount.Float64
		}
		paymentMethod := "在线支付"
		if latestOrder.PaymentMethodName.Valid {
			paymentMethod = latestOrder.PaymentMethodName.String
		}
		packageName := "未知套餐"
		if latestOrder.Package.ID > 0 {
			packageName = latestOrder.Package.Name
		} else if latestOrder.ExtraData.Valid {
			packageName = "设备/时长升级"
		}

		// 发送客户付款成功通知邮件
		if latestOrder.PackageID > 0 && notification.ShouldSendCustomerNotification("new_order") {
			emailService := email.NewEmailService()
			templateBuilder := email.NewEmailTemplateBuilder()

			// 1. 发送付款成功通知邮件
			paymentSuccessContent := templateBuilder.GetPaymentSuccessTemplate(
				latestUser.Username,
				latestOrder.OrderNo,
				packageName,
				paidAmount,
				paymentMethod,
				paymentTime,
			)
			if err := emailService.QueueEmail(latestUser.Email, "支付成功通知", paymentSuccessContent, "payment_success"); err != nil {
				utils.LogErrorMsg("发送付款成功邮件失败: order_no=%s, email=%s, error=%v", latestOrder.OrderNo, latestUser.Email, err)
			} else {
				utils.LogInfo("付款成功邮件已加入队列: order_no=%s, email=%s", latestOrder.OrderNo, latestUser.Email)
			}

			// 2. 发送订阅配置信息邮件
			var subscriptionInfo models.Subscription
			if err := db.Where("user_id = ?", latestUser.ID).First(&subscriptionInfo).Error; err == nil {
				baseURL := templateBuilder.GetBaseURL()
				timestamp := fmt.Sprintf("%d", utils.GetBeijingTime().Unix())
				universalURL := fmt.Sprintf("%s/api/v1/subscriptions/universal/%s?t=%s", baseURL, subscriptionInfo.SubscriptionURL, timestamp)
				clashURL := fmt.Sprintf("%s/api/v1/subscriptions/clash/%s?t=%s", baseURL, subscriptionInfo.SubscriptionURL, timestamp)

				expireTime := "未设置"
				remainingDays := 0
				if !subscriptionInfo.ExpireTime.IsZero() {
					expireTime = subscriptionInfo.ExpireTime.Format("2006-01-02 15:04:05")
					diff := subscriptionInfo.ExpireTime.Sub(utils.GetBeijingTime())
					if diff > 0 {
						remainingDays = int(diff.Hours() / 24)
					}
				}

				content := templateBuilder.GetSubscriptionTemplate(
					latestUser.Username,
					universalURL,
					clashURL,
					expireTime,
					remainingDays,
					subscriptionInfo.DeviceLimit,
					subscriptionInfo.CurrentDevices,
				)
				if err := emailService.QueueEmail(latestUser.Email, "服务配置信息", content, "subscription"); err != nil {
					utils.LogErrorMsg("发送订阅配置邮件失败: order_no=%s, email=%s, error=%v", latestOrder.OrderNo, latestUser.Email, err)
				} else {
					utils.LogInfo("订阅配置邮件已加入队列: order_no=%s, email=%s", latestOrder.OrderNo, latestUser.Email)
				}
			}
		}

		notificationService := notification.NewNotificationService()
		_ = notificationService.SendAdminNotification("order_paid", map[string]interface{}{
			"order_no":       latestOrder.OrderNo,
			"username":       latestUser.Username,
			"amount":         paidAmount,
			"package_name":   packageName,
			"payment_method": paymentMethod,
			"payment_time":   paymentTime,
		})
	}()

	c.String(http.StatusOK, "success")
}

// GetPaymentStatus 查询支付状态
func GetPaymentStatus(c *gin.Context) {
	transactionID := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	var transaction models.PaymentTransaction
	if err := db.Where("id = ? AND user_id = ?", transactionID, user.ID).First(&transaction).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "支付交易不存在", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"status":   transaction.Status,
		"amount":   float64(transaction.Amount) / 100,
		"order_id": transaction.OrderID,
	})
}

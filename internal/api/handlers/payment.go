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
	c.JSON(200, gin.H{"success": true, "data": res})
}

func CreatePayment(c *gin.Context) {
	u, _ := middleware.GetCurrentUser(c)
	var req struct {
		OrderID         uint `json:"order_id"`
		PaymentMethodID uint `json:"payment_method_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "message": "参数错误"})
		return
	}
	db := database.GetDB()
	var order models.Order
	if err := db.Where("id = ? AND user_id = ?", req.OrderID, u.ID).First(&order).Error; err != nil {
		c.JSON(404, gin.H{"success": false, "message": "订单不存在"})
		return
	}
	if order.Status != "pending" {
		c.JSON(400, gin.H{"success": false, "message": "订单不可支付"})
		return
	}
	var cfg models.PaymentConfig
	if err := db.First(&cfg, req.PaymentMethodID).Error; err != nil || cfg.Status != 1 {
		c.JSON(404, gin.H{"success": false, "message": "支付方式无效"})
		return
	}
	amt := int(order.Amount * 100)
	if order.FinalAmount.Valid {
		amt = int(order.FinalAmount.Float64 * 100)
	}
	tx := models.PaymentTransaction{OrderID: order.ID, UserID: u.ID, PaymentMethodID: cfg.ID, Amount: amt, Status: "pending"}
	db.Create(&tx)
	c.JSON(200, gin.H{"success": true, "data": gin.H{"transaction_id": tx.ID, "amount": float64(amt) / 100}})
}

// PaymentNotify 支付回调
func PaymentNotify(c *gin.Context) {
	paymentType := c.Param("type") // alipay, wechat, etc.
	db := database.GetDB()

	// 获取回调参数
	params := make(map[string]string)
	// 优先从 form 获取（POST回调）
	if err := c.Request.ParseForm(); err == nil {
		for k, v := range c.Request.PostForm {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
	}
	// 如果form中没有，从query获取
	if len(params) == 0 {
		for k, v := range c.Request.URL.Query() {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
	}

	// 获取支付配置
	var paymentConfig models.PaymentConfig
	// 查找对应类型的支付配置（不区分大小写）
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

	// 获取订单号和外部交易号（用于幂等性检查）
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
	utils.LogError("PaymentNotify: received callback", nil, map[string]interface{}{
		"payment_type":            paymentType,
		"order_no":                orderNo,
		"external_transaction_id": externalTransactionID,
		"params":                  params,
	})

	// 获取订单（验证订单存在）
	var order models.Order
	if err := db.Preload("Package").Where("order_no = ?", orderNo).First(&order).Error; err != nil {
		utils.LogError("PaymentNotify: order not found", err, map[string]interface{}{
			"order_no": orderNo,
		})
		c.String(http.StatusBadRequest, "订单不存在")
		return
	}

	// 幂等性检查：如果外部交易号已存在且已处理，直接返回成功
	if externalTransactionID != "" {
		var existingTransaction models.PaymentTransaction
		if err := db.Where("external_transaction_id = ? AND status = ?", externalTransactionID, "success").First(&existingTransaction).Error; err == nil {
			// 交易已处理，直接返回成功（幂等性）
			utils.LogError("PaymentNotify: transaction already processed", nil, map[string]interface{}{
				"order_no":                orderNo,
				"external_transaction_id": externalTransactionID,
			})
			c.String(http.StatusOK, "success")
			return
		}
	}

	// 验证订单金额（防止金额篡改）
	if paymentType == "alipay" {
		// 支付宝回调中的金额（转换为元）
		if amountStr, ok := params["total_amount"]; ok {
			var callbackAmount float64
			fmt.Sscanf(amountStr, "%f", &callbackAmount)
			// 验证金额是否匹配（允许0.01的误差）
			expectedAmount := order.Amount
			if order.FinalAmount.Valid {
				expectedAmount = order.FinalAmount.Float64
			}
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

	// 如果订单已经是已支付状态，直接返回成功（避免重复处理，幂等性）
	if order.Status == "paid" {
		utils.LogError("PaymentNotify: order already paid", nil, map[string]interface{}{
			"order_no": orderNo,
		})
		c.String(http.StatusOK, "success")
		return
	}

	// 使用事务确保数据一致性
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 更新订单状态
	order.Status = "paid"
	order.PaymentTime = database.NullTime(utils.GetBeijingTime())
	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		utils.LogError("PaymentNotify: failed to update order", err, map[string]interface{}{
			"order_no": orderNo,
		})
		c.String(http.StatusInternalServerError, "更新订单失败")
		return
	}

	// 更新支付交易状态
	var transaction models.PaymentTransaction
	if err := tx.Where("order_id = ?", order.ID).First(&transaction).Error; err == nil {
		transaction.Status = "success"
		if externalTransactionID != "" {
			transaction.ExternalTransactionID = database.NullString(externalTransactionID)
		}
		// 保存回调数据（用于审计）
		if callbackData, err := json.Marshal(params); err == nil {
			transaction.CallbackData = database.NullString(string(callbackData))
		}
		if err := tx.Save(&transaction).Error; err != nil {
			tx.Rollback()
			utils.LogError("PaymentNotify: failed to update transaction", err, map[string]interface{}{
				"order_no": orderNo,
			})
			c.String(http.StatusInternalServerError, "更新交易失败")
			return
		}
	}

	// 提交事务（只有在事务成功提交后，订单状态才会真正更新为 paid）
	if err := tx.Commit().Error; err != nil {
		utils.LogError("PaymentNotify: failed to commit transaction", err, map[string]interface{}{
			"order_no": orderNo,
		})
		c.String(http.StatusInternalServerError, "处理失败")
		return
	}

	// 事务提交成功后，订单状态已更新为 paid
	// 立即同步处理订单业务逻辑（开通订阅），确保用户能立刻使用

	// 处理套餐订单
	if order.PackageID > 0 {
		orderService := orderServicePkg.NewOrderService()
		_, err := orderService.ProcessPaidOrder(&order)
		if err != nil {
			utils.LogError("PaymentNotify: process paid order failed", err, map[string]interface{}{
				"order_id": order.ID,
			})
			// 这里不返回错误，因为支付已成功，后续可通过补偿机制修复
		}
	} else {
		// 设备升级订单：从 ExtraData 中解析升级信息
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

		// 如果有升级信息，处理订阅升级
		if additionalDevices > 0 || additionalDays > 0 {
			var subscription models.Subscription
			if err := db.Where("user_id = ?", order.UserID).First(&subscription).Error; err == nil {
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
				if err := db.Save(&subscription).Error; err != nil {
					utils.LogError("PaymentNotify: upgrade devices failed", err, map[string]interface{}{
						"order_id": order.ID,
					})
				}
			}
		}
	}

	// 异步处理通知（邮件、Telegram等）
	go func() {
		// 重新加载订单和用户信息（确保获取最新关联数据）
		var latestOrder models.Order
		if err := db.Preload("Package").Where("id = ?", order.ID).First(&latestOrder).Error; err != nil {
			return
		}
		var latestUser models.User
		if err := db.First(&latestUser, latestOrder.UserID).Error; err != nil {
			return
		}

		// 准备支付信息
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

		// 发送订阅信息邮件（仅套餐订单）
		if latestOrder.PackageID > 0 && notification.ShouldSendCustomerNotification("new_order") {
			emailService := email.NewEmailService()
			templateBuilder := email.NewEmailTemplateBuilder()

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
				_ = emailService.QueueEmail(latestUser.Email, "服务配置信息", content, "subscription")
			}
		}

		// 发送管理员通知
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
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	var transaction models.PaymentTransaction
	if err := db.Where("id = ? AND user_id = ?", transactionID, user.ID).First(&transaction).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "支付交易不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"status":   transaction.Status,
			"amount":   float64(transaction.Amount) / 100,
			"order_id": transaction.OrderID,
		},
	})
}

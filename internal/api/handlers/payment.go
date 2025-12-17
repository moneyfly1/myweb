package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/services/payment"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetPaymentMethods 获取支付方式列表
func GetPaymentMethods(c *gin.Context) {
	db := database.GetDB()

	var methods []models.PaymentConfig
	if err := db.Where("status = ?", 1).Order("sort_order ASC").Find(&methods).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取支付方式失败",
		})
		return
	}

	// 只返回必要信息，不返回敏感配置
	result := make([]map[string]interface{}, 0)
	for _, method := range methods {
		// 支付方式名称映射
		nameMap := map[string]string{
			"alipay":   "支付宝",
			"wechat":   "微信支付",
			"yipay":    "易支付",
			"paypal":   "PayPal",
			"applepay": "Apple Pay",
			"stripe":   "Stripe",
			"bank":     "银行转账",
		}

		name := nameMap[method.PayType]
		if name == "" {
			name = method.PayType
		}

		result = append(result, map[string]interface{}{
			"id":       method.ID,
			"key":      method.PayType, // 前端使用的key
			"pay_type": method.PayType,
			"name":     name, // 显示名称
			"status":   method.Status,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// CreatePayment 创建支付
func CreatePayment(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	var req struct {
		OrderID         uint `json:"order_id" binding:"required"`
		PaymentMethodID uint `json:"payment_method_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()

	// 验证订单
	var order models.Order
	if err := db.Where("id = ? AND user_id = ?", req.OrderID, user.ID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "订单不存在",
		})
		return
	}

	if order.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "订单状态不允许支付",
		})
		return
	}

	// 获取支付配置
	var paymentConfig models.PaymentConfig
	if err := db.First(&paymentConfig, req.PaymentMethodID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "支付方式不存在",
		})
		return
	}

	if paymentConfig.Status != 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "支付方式已停用",
		})
		return
	}

	// 计算支付金额（分）
	amount := int(order.FinalAmount.Float64 * 100)
	if !order.FinalAmount.Valid {
		amount = int(order.Amount * 100)
	}

	// 创建支付交易
	transaction := models.PaymentTransaction{
		OrderID:         order.ID,
		UserID:          user.ID,
		PaymentMethodID: uint(req.PaymentMethodID),
		Amount:          amount,
		Currency:        "CNY",
		Status:          "pending",
	}

	if err := db.Create(&transaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建支付交易失败",
		})
		return
	}

	// 根据支付方式调用相应的支付接口
	var paymentURL string
	var paymentErr error

	paymentMethod := paymentConfig.PayType
	switch paymentMethod {
	case "alipay":
		alipayService, err := payment.NewAlipayService(&paymentConfig)
		if err == nil {
			paymentURL, paymentErr = alipayService.CreatePayment(&order, float64(amount)/100)
		} else {
			paymentErr = err
		}
	case "wechat":
		wechatService, err := payment.NewWechatService(&paymentConfig)
		if err == nil {
			paymentURL, paymentErr = wechatService.CreatePayment(&order, float64(amount)/100)
		} else {
			paymentErr = err
		}
	case "paypal":
		paypalService, err := payment.NewPayPalService(&paymentConfig)
		if err == nil {
			paymentURL, paymentErr = paypalService.CreatePayment(&order, float64(amount)/100)
		} else {
			paymentErr = err
		}
	case "applepay":
		applePayService, err := payment.NewApplePayService(&paymentConfig)
		if err == nil {
			paymentURL, paymentErr = applePayService.CreatePayment(&order, float64(amount)/100)
		} else {
			paymentErr = err
		}
	}

	if paymentErr != nil {
		// 不向客户端返回详细错误信息，防止信息泄露
		utils.LogError("CreatePayment: generate payment URL", paymentErr, map[string]interface{}{
			"order_id":       order.ID,
			"payment_method": paymentMethod,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "生成支付链接失败，请稍后重试",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"transaction_id": transaction.ID,
			"amount":         float64(amount) / 100,
			"payment_url":    paymentURL,
			"qr_code":        "", // 二维码可以通过前端生成
		},
	})
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
	if err := db.Where("pay_type = ? AND status = ?", paymentType, 1).First(&paymentConfig).Error; err != nil {
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
	// 现在可以安全地处理后续逻辑（订阅、邮件等）
	// 处理不同类型的订单（在事务外处理，避免长时间占用事务）
	go func() {
		// 获取最新的订单信息
		var latestOrder models.Order
		if err := db.Preload("Package").Where("order_no = ?", orderNo).First(&latestOrder).Error; err != nil {
			utils.LogError("PaymentNotify: failed to reload order", err, map[string]interface{}{
				"order_no": orderNo,
			})
			return
		}

		// 关键验证：只有在订单状态确实为 "paid" 时才处理
		// 这确保只有在支付回调验证成功、订单状态已更新为已支付后，才发送邮件和处理订阅
		if latestOrder.Status != "paid" {
			utils.LogError("PaymentNotify: order status is not paid, skipping processing", nil, map[string]interface{}{
				"order_no":     orderNo,
				"order_status": latestOrder.Status,
			})
			return
		}

		// 获取用户信息
		var latestUser models.User
		if err := db.First(&latestUser, latestOrder.UserID).Error; err != nil {
			utils.LogError("PaymentNotify: failed to get user", err, map[string]interface{}{
				"user_id": latestOrder.UserID,
			})
			return
		}

		// 处理套餐订单
		if latestOrder.PackageID > 0 {
			var pkg models.Package
			if err := db.First(&pkg, latestOrder.PackageID).Error; err == nil {
				subscription, err := processPaidOrderInPayment(db, &latestOrder, &pkg, &latestUser)
				if err != nil {
					utils.LogError("PaymentNotify: process subscription failed", err, map[string]interface{}{
						"order_id": latestOrder.ID,
					})
				} else if subscription != nil {
					// 再次验证订单状态为 paid（双重检查，确保安全）
					var verifyOrder models.Order
					if err := db.Where("order_no = ? AND status = ?", orderNo, "paid").First(&verifyOrder).Error; err != nil {
						utils.LogError("PaymentNotify: order status verification failed, not sending email", err, map[string]interface{}{
							"order_no": orderNo,
						})
						return
					}

					// 准备支付信息（用于管理员通知和客户邮件）
					paymentTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
					paidAmount := latestOrder.Amount
					if latestOrder.FinalAmount.Valid {
						paidAmount = latestOrder.FinalAmount.Float64
					}
					paymentMethod := "在线支付"
					if latestOrder.PaymentMethodName.Valid {
						paymentMethod = latestOrder.PaymentMethodName.String
					}

					// 发送订阅信息邮件（付款成功后直接发送订阅信息，不再发送支付成功通知）
					// 检查客户通知开关
					if notification.ShouldSendCustomerNotification("new_order") {
						go func() {
							emailService := email.NewEmailService()
							templateBuilder := email.NewEmailTemplateBuilder()

							// 获取订阅信息
							var subscriptionInfo models.Subscription
							if err := db.Where("user_id = ?", latestUser.ID).First(&subscriptionInfo).Error; err == nil {
								// 使用 EmailTemplateBuilder 的 GetBaseURL 方法获取基础URL
								baseURL := templateBuilder.GetBaseURL()
								timestamp := fmt.Sprintf("%d", utils.GetBeijingTime().Unix())
								universalURL := fmt.Sprintf("%s/api/v1/subscriptions/universal/%s?t=%s", baseURL, subscriptionInfo.SubscriptionURL, timestamp) // 通用订阅（Base64格式）
								clashURL := fmt.Sprintf("%s/api/v1/subscriptions/clash/%s?t=%s", baseURL, subscriptionInfo.SubscriptionURL, timestamp)         // 猫咪订阅（Clash YAML格式）

								// 计算到期时间和剩余天数
								expireTime := "未设置"
								remainingDays := 0
								if !subscriptionInfo.ExpireTime.IsZero() {
									expireTime = subscriptionInfo.ExpireTime.Format("2006-01-02 15:04:05")
									now := utils.GetBeijingTime()
									diff := subscriptionInfo.ExpireTime.Sub(now)
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
								subject := "服务配置信息"
								_ = emailService.QueueEmail(latestUser.Email, subject, content, "subscription")
							}
						}()
					}

					// 发送管理员通知
					go func() {
						notificationService := notification.NewNotificationService()
						_ = notificationService.SendAdminNotification("order_paid", map[string]interface{}{
							"order_no":       latestOrder.OrderNo,
							"username":       latestUser.Username,
							"amount":         paidAmount,
							"package_name":   pkg.Name,
							"payment_method": paymentMethod,
							"payment_time":   paymentTime,
						})
					}()
				}
			}
		} else {
			// 设备升级订单：从 ExtraData 中解析升级信息
			var additionalDevices int
			var additionalDays int

			if latestOrder.ExtraData.Valid && latestOrder.ExtraData.String != "" {
				// 解析 JSON
				var extraData map[string]interface{}
				if err := json.Unmarshal([]byte(latestOrder.ExtraData.String), &extraData); err == nil {
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
				if err := db.Where("user_id = ?", latestUser.ID).First(&subscription).Error; err == nil {
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
							"order_id": latestOrder.ID,
						})
					}
				}
			}
		}
	}()

	c.String(http.StatusOK, "success")
}

// processPaidOrderInPayment 处理已支付订单（在 payment.go 中，避免循环导入）
func processPaidOrderInPayment(db *gorm.DB, order *models.Order, pkg *models.Package, user *models.User) (*models.Subscription, error) {
	now := utils.GetBeijingTime()

	// 1. 更新或创建订阅
	var subscription models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
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
		if err := db.Create(&subscription).Error; err != nil {
			return nil, fmt.Errorf("创建订阅失败: %v", err)
		}
		fmt.Printf("processPaidOrderInPayment: ✅ 创建新订阅成功 - user_id=%d, package_id=%d, device_limit=%d, duration_days=%d, expire_time=%s\n",
			user.ID, pkg.ID, pkg.DeviceLimit, pkg.DurationDays, expireTime.Format("2006-01-02 15:04:05"))
	} else {
		// 延长订阅
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
		// 更新套餐ID
		pkgID := int64(pkg.ID)
		subscription.PackageID = &pkgID
		if err := db.Save(&subscription).Error; err != nil {
			return nil, fmt.Errorf("更新订阅失败: %v", err)
		}
		fmt.Printf("processPaidOrderInPayment: ✅ 更新订阅成功 - user_id=%d, package_id=%d, device_limit: %d->%d, expire_time: %s->%s\n",
			user.ID, pkg.ID, oldDeviceLimit, pkg.DeviceLimit, oldExpireTime.Format("2006-01-02 15:04:05"), subscription.ExpireTime.Format("2006-01-02 15:04:05"))
	}

	// 2. 更新用户累计消费
	paidAmount := order.Amount
	if order.FinalAmount.Valid {
		paidAmount = order.FinalAmount.Float64
	}
	user.TotalConsumption += paidAmount
	if err := db.Save(&user).Error; err != nil {
		return nil, fmt.Errorf("更新用户累计消费失败: %v", err)
	}

	// 3. 检查并更新用户等级
	var userLevels []models.UserLevel
	if err := db.Where("is_active = ?", true).Order("level_order ASC").Find(&userLevels).Error; err == nil {
		for _, level := range userLevels {
			if user.TotalConsumption >= level.MinConsumption {
				// 检查是否需要升级
				if !user.UserLevelID.Valid || user.UserLevelID.Int64 != int64(level.ID) {
					// 需要升级
					var currentLevel models.UserLevel
					shouldUpgrade := true
					if user.UserLevelID.Valid {
						if err := db.First(&currentLevel, user.UserLevelID.Int64).Error; err == nil {
							// 如果当前等级更高（level_order 更小），不降级
							if currentLevel.LevelOrder < level.LevelOrder {
								shouldUpgrade = false
							}
						}
					}
					if shouldUpgrade {
						user.UserLevelID = sql.NullInt64{Int64: int64(level.ID), Valid: true}
						if err := db.Save(&user).Error; err != nil {
							// 等级更新失败不影响订单完成，只记录错误
							fmt.Printf("更新用户等级失败: %v\n", err)
						}
					}
				}
			}
		}
	}

	return &subscription, nil
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

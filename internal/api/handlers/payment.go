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
		result = append(result, map[string]interface{}{
			"id":       method.ID,
			"pay_type": method.PayType,
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
	var err error

	paymentMethod := paymentConfig.PayType
	if paymentMethod == "alipay" {
		alipayService, err := payment.NewAlipayService(&paymentConfig)
		if err == nil {
			paymentURL, err = alipayService.CreatePayment(&order, float64(amount)/100)
		}
	} else if paymentMethod == "wechat" {
		wechatService, err := payment.NewWechatService(&paymentConfig)
		if err == nil {
			paymentURL, err = wechatService.CreatePayment(&order, float64(amount)/100)
		}
	}

	if err != nil {
		// 不向客户端返回详细错误信息，防止信息泄露
		utils.LogError("CreatePayment: generate payment URL", err, map[string]interface{}{
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
	if paymentType == "alipay" {
		// 支付宝回调参数在 query 中
		for k, v := range c.Request.URL.Query() {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
	} else if paymentType == "wechat" {
		// 微信回调参数在 body 中（XML格式）
		// 这里简化处理，实际需要解析 XML
	}

	// 获取支付配置
	var paymentConfig models.PaymentConfig
	if err := db.Where("type = ? AND is_active = ?", paymentType, true).First(&paymentConfig).Error; err != nil {
		c.String(http.StatusBadRequest, "支付配置不存在")
		return
	}

	// 验证签名
	var verified bool
	if paymentType == "alipay" {
		alipayService, err := payment.NewAlipayService(&paymentConfig)
		if err == nil {
			verified = alipayService.VerifyNotify(params)
		}
	} else if paymentType == "wechat" {
		wechatService, err := payment.NewWechatService(&paymentConfig)
		if err == nil {
			verified = wechatService.VerifyNotify(params)
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

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		utils.LogError("PaymentNotify: failed to commit transaction", err, map[string]interface{}{
			"order_no": orderNo,
		})
		c.String(http.StatusInternalServerError, "处理失败")
		return
	}

	// 获取用户信息
	var user models.User
	if err := db.First(&user, order.UserID).Error; err != nil {
		fmt.Printf("获取用户信息失败: %v\n", err)
		c.String(http.StatusOK, "success") // 仍然返回成功，避免支付平台重复回调
		return
	}

	// 处理不同类型的订单
	if order.PackageID > 0 {
		// 套餐订单：开通或延长订阅
		var pkg models.Package
		if err := db.First(&pkg, order.PackageID).Error; err == nil {
			subscription, err := processPaidOrderInPayment(db, &order, &pkg, &user)
			if err != nil {
				utils.LogError("PaymentNotify: process subscription failed", err, map[string]interface{}{
					"order_id": order.ID,
				})
			} else if subscription != nil {
				// 发送支付成功邮件
				go func() {
					emailService := email.NewEmailService()
					templateBuilder := email.NewEmailTemplateBuilder()
					paymentTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
					paidAmount := order.Amount
					if order.FinalAmount.Valid {
						paidAmount = order.FinalAmount.Float64
					}
					paymentMethod := "在线支付"
					if order.PaymentMethodName.Valid {
						paymentMethod = order.PaymentMethodName.String
					}
					content := templateBuilder.GetPaymentSuccessTemplate(
						user.Username,
						order.OrderNo,
						pkg.Name,
						paidAmount,
						paymentMethod,
						paymentTime,
					)
					subject := "支付成功通知"
					_ = emailService.QueueEmail(user.Email, subject, content, "payment_success")
				}()
			}
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
			if err := db.Where("user_id = ?", user.ID).First(&subscription).Error; err == nil {
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
	} else {
		// 延长订阅
		if subscription.ExpireTime.Before(now) {
			subscription.ExpireTime = now.AddDate(0, 0, pkg.DurationDays)
		} else {
			subscription.ExpireTime = subscription.ExpireTime.AddDate(0, 0, pkg.DurationDays)
		}
		subscription.DeviceLimit = pkg.DeviceLimit
		subscription.IsActive = true
		subscription.Status = "active"
		// 更新套餐ID
		pkgID := int64(pkg.ID)
		subscription.PackageID = &pkgID
		if err := db.Save(&subscription).Error; err != nil {
			return nil, fmt.Errorf("更新订阅失败: %v", err)
		}
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

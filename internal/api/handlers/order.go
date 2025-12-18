package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"
	orderServicePkg "cboard-go/internal/services/order"
	"cboard-go/internal/services/payment"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	PackageID     uint    `json:"package_id" binding:"required"`
	CouponCode    string  `json:"coupon_code"`
	PaymentMethod string  `json:"payment_method"`
	Amount        float64 `json:"amount"`
	UseBalance    bool    `json:"use_balance"`
	BalanceAmount float64 `json:"balance_amount"`
	Currency      string  `json:"currency"`
}

// CreateOrder 创建订单
func CreateOrder(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()

	// 获取套餐
	var pkg models.Package
	if err := db.First(&pkg, req.PackageID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "套餐不存在",
		})
		return
	}

	if !pkg.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "套餐已停用",
		})
		return
	}

	// 计算金额
	baseAmount := pkg.Price
	levelDiscountAmount := 0.0
	couponDiscountAmount := 0.0
	totalDiscountAmount := 0.0
	finalAmount := baseAmount

	// 1. 先应用用户等级折扣
	var userLevel *models.UserLevel
	if user.UserLevelID.Valid {
		var lvl models.UserLevel
		if err := db.First(&lvl, user.UserLevelID.Int64).Error; err == nil {
			userLevel = &lvl
			if userLevel.DiscountRate > 0 && userLevel.DiscountRate < 1.0 {
				// 应用等级折扣（例如：0.9 表示 9 折，即 10% 折扣）
				levelDiscountAmount = baseAmount * (1.0 - userLevel.DiscountRate)
				finalAmount = baseAmount * userLevel.DiscountRate
			}
		}
	}

	// 2. 在等级折扣后的金额上应用优惠券
	var couponID *int64
	if req.CouponCode != "" {
		var coupon models.Coupon
		if err := db.Where("code = ? AND status = ?", req.CouponCode, "active").First(&coupon).Error; err == nil {
			now := utils.GetBeijingTime()
			// 检查有效期：now >= ValidFrom && now <= ValidUntil
			if (now.After(coupon.ValidFrom) || now.Equal(coupon.ValidFrom)) && (now.Before(coupon.ValidUntil) || now.Equal(coupon.ValidUntil)) {
				// 检查最低消费金额（使用等级折扣后的金额）
				if !coupon.MinAmount.Valid || finalAmount >= coupon.MinAmount.Float64 {
					// 应用优惠券（在等级折扣后的金额上）
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
					couponIDVal := int64(coupon.ID)
					couponID = &couponIDVal
				}
			}
		}
	}

	// 计算总折扣金额
	totalDiscountAmount = levelDiscountAmount + couponDiscountAmount

	// 如果前端提供了金额，验证是否匹配（允许小的浮点数误差）
	if req.Amount > 0 {
		if req.Amount < finalAmount-0.01 || req.Amount > finalAmount+0.01 {
			// 前端金额与后端计算不一致，使用后端计算的金额
			// 但记录警告（可选）
		}
		// 使用后端计算的金额，确保一致性
	}

	// 处理余额支付
	balanceUsed := 0.0
	if req.UseBalance && req.BalanceAmount > 0 {
		// 检查用户余额
		if user.Balance < req.BalanceAmount {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "余额不足",
			})
			return
		}
		// 限制余额使用不超过最终金额
		if req.BalanceAmount > finalAmount {
			req.BalanceAmount = finalAmount
		}
		balanceUsed = req.BalanceAmount
		finalAmount -= balanceUsed
		// 扣除余额
		user.Balance -= balanceUsed
		if err := db.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "扣除余额失败",
			})
			return
		}
	}

	// 如果余额已支付全部金额，直接完成订单
	if finalAmount <= 0.01 { // 允许小的浮点数误差
		finalAmount = 0
		// 创建已支付的订单
		orderNo := utils.GenerateOrderNo(user.ID)
		order := models.Order{
			OrderNo:           orderNo,
			UserID:            user.ID,
			PackageID:         pkg.ID,
			Amount:            baseAmount,
			Status:            "paid",
			DiscountAmount:    database.NullFloat64(totalDiscountAmount),
			FinalAmount:       database.NullFloat64(balanceUsed),
			PaymentMethodName: database.NullString("余额支付"),
		}
		if couponID != nil {
			order.CouponID = database.NullInt64(*couponID)
		}
		order.PaymentTime = database.NullTime(utils.GetBeijingTime())

		if err := db.Create(&order).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "创建订单失败",
			})
			return
		}

		// 处理订单支付成功后的逻辑（开通订阅、更新累计消费等）
		svc := orderServicePkg.NewOrderService()
		subscription, err := svc.ProcessPaidOrder(&order)
		if err != nil {
			// 不向客户端返回详细错误信息，防止信息泄露
			utils.LogError("CreateOrder: process paid order", err, map[string]interface{}{
				"order_id": order.ID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "处理订单失败，请稍后重试",
			})
			return
		}

		// 记录创建订单审计日志（余额支付）
		utils.SetResponseStatus(c, http.StatusCreated)
		utils.CreateAuditLogSimple(c, "create_order", "order", order.ID, fmt.Sprintf("创建订单(余额支付): %s, 金额: %.2f元", order.OrderNo, order.Amount))

		// 发送支付成功邮件（只有在订单状态确实为"paid"时才发送）
		// 余额支付是立即完成的，所以订单状态已经是"paid"，可以发送支付成功邮件
		if order.Status == "paid" {
			go func() {
				emailService := email.NewEmailService()
				templateBuilder := email.NewEmailTemplateBuilder()
				paymentTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
				paidAmount := order.Amount
				if order.FinalAmount.Valid {
					paidAmount = order.FinalAmount.Float64
				}
				content := templateBuilder.GetPaymentSuccessTemplate(
					user.Username,
					order.OrderNo,
					pkg.Name,
					paidAmount,
					"余额支付",
					paymentTime,
				)
				subject := "支付成功通知"
				_ = emailService.QueueEmail(user.Email, subject, content, "payment_success")
				fmt.Printf("CreateOrder: ✅ 发送支付成功邮件（余额支付）- order_no=%s, status=%s, amount=%.2f\n", order.OrderNo, order.Status, paidAmount)
			}()
		} else {
			fmt.Printf("CreateOrder: ⚠️ 不发送支付成功邮件（余额支付）- order_no=%s, status=%s（不是paid状态）\n", order.OrderNo, order.Status)
		}

		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"message": "订单已支付成功",
			"data": gin.H{
				"order":        order,
				"subscription": subscription,
			},
		})
		return
	}

	// 生成订单号
	orderNo := utils.GenerateOrderNo(user.ID)

	// 创建订单
	order := models.Order{
		OrderNo:        orderNo,
		UserID:         user.ID,
		PackageID:      pkg.ID,
		Amount:         baseAmount,
		Status:         "pending",
		DiscountAmount: database.NullFloat64(totalDiscountAmount),
		FinalAmount:    database.NullFloat64(finalAmount),
	}

	if couponID != nil {
		order.CouponID = database.NullInt64(*couponID)
	}

	// 设置支付方式
	if balanceUsed > 0 {
		if finalAmount > 0.01 {
			// 混合支付：余额+其他支付方式
			order.PaymentMethodName = database.NullString(fmt.Sprintf("余额支付(%.2f元)+%s", balanceUsed, req.PaymentMethod))
		} else {
			// 纯余额支付
			order.PaymentMethodName = database.NullString("余额支付")
		}
	} else if req.PaymentMethod != "" && req.PaymentMethod != "balance" {
		order.PaymentMethodName = database.NullString(req.PaymentMethod)
	}

	if err := db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建订单失败",
		})
		return
	}

	// 记录创建订单审计日志
	utils.SetResponseStatus(c, http.StatusCreated)
	utils.CreateAuditLogSimple(c, "create_order", "order", order.ID, fmt.Sprintf("创建订单: %s, 金额: %.2f元", order.OrderNo, order.Amount))

	// 不再发送订单确认邮件（提醒付款），付款成功后直接发送订阅信息

	// 如果订单需要其他支付方式（非余额支付），立即生成支付URL
	var paymentURL string
	var paymentError string
	if finalAmount > 0.01 && req.PaymentMethod != "" && req.PaymentMethod != "balance" {
		// 获取支付配置
		var paymentConfig models.PaymentConfig
		// 根据支付方式查找对应的支付配置
		payType := req.PaymentMethod
		fmt.Printf("CreateOrder: 开始处理支付方式 %s, 订单金额: %.2f, 订单号: %s\n", payType, finalAmount, order.OrderNo)

		if payType == "alipay" || payType == "wechat" || payType == "paypal" || payType == "applepay" {
			// 查找对应类型的支付配置（不区分大小写）
			// 使用 LOWER() 函数确保大小写不敏感匹配
			if err := db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", payType, 1).Order("sort_order ASC").First(&paymentConfig).Error; err != nil {
				// 如果查询失败，尝试查询所有该类型的配置（包括禁用的），以便提供更详细的错误信息
				var allConfigs []models.PaymentConfig
				db.Where("LOWER(pay_type) = LOWER(?)", payType).Find(&allConfigs)

				if len(allConfigs) > 0 {
					// 找到了配置，但可能状态不是启用状态
					var statusInfo []string
					for _, cfg := range allConfigs {
						statusInfo = append(statusInfo, fmt.Sprintf("ID:%d,status:%d", cfg.ID, cfg.Status))
					}
					paymentError = fmt.Sprintf("未找到启用的%s支付配置。找到%d个配置，但状态都不是启用(1)。配置状态: %s。请管理员检查支付配置的状态设置。", payType, len(allConfigs), strings.Join(statusInfo, "; "))
					fmt.Printf("CreateOrder: ❌ 支付配置状态不正确 - pay_type=%s, found_configs=%d, statuses=%v\n", payType, len(allConfigs), statusInfo)
				} else {
					// 完全没有找到该类型的配置
					paymentError = fmt.Sprintf("未找到%s支付配置，请管理员先创建支付配置", payType)
					fmt.Printf("CreateOrder: ❌ 支付配置不存在 - pay_type=%s\n", payType)
				}

				utils.LogError("CreateOrder: payment config not found or not enabled", err, map[string]interface{}{
					"pay_type":      payType,
					"user_id":       user.ID,
					"found_configs": len(allConfigs),
				})
			} else {
				fmt.Printf("CreateOrder: ✅ 找到支付配置 - pay_type=%s, config_id=%d, app_id=%s, has_notify_url=%v\n",
					payType, paymentConfig.ID, paymentConfig.AppID.String, paymentConfig.NotifyURL.Valid)

				// 检查支付宝配置是否完整（特别是 NotifyURL）
				if payType == "alipay" {
					if !paymentConfig.NotifyURL.Valid || paymentConfig.NotifyURL.String == "" {
						paymentError = "支付宝回调地址未配置，请在支付配置中设置 NotifyURL（异步回调地址）"
						fmt.Printf("CreateOrder: ❌ 支付宝回调地址未配置 - config_id=%d\n", paymentConfig.ID)
						utils.LogError("CreateOrder: alipay notify URL not configured", nil, map[string]interface{}{
							"payment_config_id": paymentConfig.ID,
							"order_id":          order.ID,
						})
					}
				}

				// 如果配置检查通过，继续创建支付交易
				if paymentError == "" {
					// 创建支付交易
					transaction := models.PaymentTransaction{
						OrderID:         order.ID,
						UserID:          user.ID,
						PaymentMethodID: paymentConfig.ID,
						Amount:          int(finalAmount * 100), // 转换为分
						Currency:        "CNY",
						Status:          "pending",
					}
					if err := db.Create(&transaction).Error; err != nil {
						paymentError = fmt.Sprintf("创建支付交易失败: %v", err)
						utils.LogError("CreateOrder: create payment transaction failed", err, map[string]interface{}{
							"order_id": order.ID,
							"user_id":  user.ID,
						})
					} else {
						// 生成支付URL
						switch paymentConfig.PayType {
						case "alipay":
							fmt.Printf("CreateOrder: 开始初始化支付宝服务 - order_id=%d, order_no=%s, amount=%.2f, notify_url=%s\n",
								order.ID, order.OrderNo, finalAmount, paymentConfig.NotifyURL.String)
							alipayService, err := payment.NewAlipayService(&paymentConfig)
							if err != nil {
								paymentError = fmt.Sprintf("初始化支付宝服务失败: %v", err)
								fmt.Printf("CreateOrder: ❌ 支付宝服务初始化失败 - order_id=%d, error=%v\n", order.ID, err)
								utils.LogError("CreateOrder: init alipay service failed", err, map[string]interface{}{
									"payment_config_id": paymentConfig.ID,
									"order_id":          order.ID,
									"app_id":            paymentConfig.AppID.String,
									"has_private_key":   paymentConfig.MerchantPrivateKey.Valid,
									"has_public_key":    paymentConfig.AlipayPublicKey.Valid,
									"has_notify_url":    paymentConfig.NotifyURL.Valid,
								})
							} else {
								fmt.Printf("CreateOrder: ✅ 支付宝服务初始化成功，开始创建支付 - order_id=%d, order_no=%s\n",
									order.ID, order.OrderNo)
								paymentURL, err = alipayService.CreatePayment(&order, finalAmount)
								if err != nil {
									paymentError = fmt.Sprintf("生成支付宝支付URL失败: %v", err)
									fmt.Printf("CreateOrder: ❌ 支付宝支付URL生成失败 - order_id=%d, order_no=%s, error=%v\n",
										order.ID, order.OrderNo, err)
									utils.LogError("CreateOrder: create alipay payment failed", err, map[string]interface{}{
										"order_id":     order.ID,
										"order_no":     order.OrderNo,
										"amount":       finalAmount,
										"app_id":       paymentConfig.AppID.String,
										"is_prod":      paymentConfig.ConfigJSON.Valid,
										"error_detail": err.Error(),
									})
								} else {
									// 支付URL生成成功，记录日志
									urlPreview := paymentURL
									if len(urlPreview) > 100 {
										urlPreview = urlPreview[:100] + "..."
									}
									fmt.Printf("CreateOrder: ✅ 支付宝支付URL生成成功 - order_id=%d, order_no=%s, amount=%.2f, url_length=%d, url_preview=%s\n",
										order.ID, order.OrderNo, finalAmount, len(paymentURL), urlPreview)
								}
							}
						case "wechat":
							wechatService, err := payment.NewWechatService(&paymentConfig)
							if err != nil {
								paymentError = fmt.Sprintf("初始化微信支付服务失败: %v", err)
								utils.LogError("CreateOrder: init wechat service failed", err, map[string]interface{}{
									"payment_config_id": paymentConfig.ID,
									"order_id":          order.ID,
								})
							} else {
								paymentURL, err = wechatService.CreatePayment(&order, finalAmount)
								if err != nil {
									paymentError = fmt.Sprintf("生成微信支付URL失败: %v", err)
									utils.LogError("CreateOrder: create wechat payment failed", err, map[string]interface{}{
										"order_id": order.ID,
										"amount":   finalAmount,
									})
								}
							}
						case "paypal":
							paypalService, err := payment.NewPayPalService(&paymentConfig)
							if err != nil {
								paymentError = fmt.Sprintf("初始化PayPal服务失败: %v", err)
								utils.LogError("CreateOrder: init paypal service failed", err, map[string]interface{}{
									"payment_config_id": paymentConfig.ID,
									"order_id":          order.ID,
								})
							} else {
								paymentURL, err = paypalService.CreatePayment(&order, finalAmount)
								if err != nil {
									paymentError = fmt.Sprintf("生成PayPal支付URL失败: %v", err)
									utils.LogError("CreateOrder: create paypal payment failed", err, map[string]interface{}{
										"order_id": order.ID,
										"amount":   finalAmount,
									})
								}
							}
						case "applepay":
							applePayService, err := payment.NewApplePayService(&paymentConfig)
							if err != nil {
								paymentError = fmt.Sprintf("初始化Apple Pay服务失败: %v", err)
								utils.LogError("CreateOrder: init applepay service failed", err, map[string]interface{}{
									"payment_config_id": paymentConfig.ID,
									"order_id":          order.ID,
								})
							} else {
								paymentURL, err = applePayService.CreatePayment(&order, finalAmount)
								if err != nil {
									paymentError = fmt.Sprintf("生成Apple Pay支付URL失败: %v", err)
									utils.LogError("CreateOrder: create applepay payment failed", err, map[string]interface{}{
										"order_id": order.ID,
										"amount":   finalAmount,
									})
								}
							}
						}
					}
				}
			}
		}
	}

	// 返回订单和支付URL
	responseData := gin.H{
		"order_no":        order.OrderNo,
		"id":              order.ID,
		"user_id":         order.UserID,
		"package_id":      order.PackageID,
		"amount":          order.Amount,
		"final_amount":    finalAmount,
		"discount_amount": totalDiscountAmount,
		"status":          order.Status,
		"created_at":      order.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	if paymentURL != "" {
		responseData["payment_url"] = paymentURL
		responseData["payment_qr_code"] = paymentURL // 兼容前端可能使用的字段名
		// 记录支付URL（截取前100个字符以避免日志过长）
		urlPreview := paymentURL
		if len(urlPreview) > 100 {
			urlPreview = urlPreview[:100] + "..."
		}
		fmt.Printf("CreateOrder: payment URL generated successfully, order_id=%d, order_no=%s, url_preview=%s\n",
			order.ID, order.OrderNo, urlPreview)
	} else if paymentError != "" {
		// 如果支付URL生成失败，返回错误信息
		responseData["payment_error"] = paymentError
		responseData["note"] = paymentError // 兼容字段
		fmt.Printf("CreateOrder: payment URL generation failed, order_id=%d, order_no=%s, error=%s\n",
			order.ID, order.OrderNo, paymentError)
	} else {
		// 既没有支付URL也没有错误信息（可能是余额支付或其他情况）
		fmt.Printf("CreateOrder: no payment URL needed, order_id=%d, order_no=%s, status=%s\n",
			order.ID, order.OrderNo, order.Status)
	}

	if couponID != nil {
		responseData["coupon_id"] = *couponID
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    responseData,
	})
}

// GetOrders 获取订单列表
func GetOrders(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	// 获取分页参数
	page := 1
	size := 20
	if pageStr := c.Query("page"); pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if sizeStr := c.Query("size"); sizeStr != "" {
		fmt.Sscanf(sizeStr, "%d", &size)
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	// 获取筛选参数
	status := c.Query("status")
	paymentMethod := c.Query("payment_method")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	isAdmin, exists := c.Get("is_admin")
	admin := exists && isAdmin.(bool)

	db := database.GetDB()
	var orders []models.Order
	var total int64

	// 构建查询
	query := db.Model(&models.Order{}).Preload("Package").Preload("Coupon")

	// 非管理员只能查看自己的订单
	if !admin {
		query = query.Where("user_id = ?", user.ID)
	}

	// 状态筛选（精确匹配）
	if status != "" && status != "all" {
		// 支持多种状态值格式
		statusMap := map[string]string{
			"pending":   "pending",
			"paid":      "paid",
			"cancelled": "cancelled",
			"expired":   "expired",
			"待支付":       "pending",
			"已支付":       "paid",
			"已取消":       "cancelled",
			"已过期":       "expired",
		}

		if mappedStatus, ok := statusMap[status]; ok {
			query = query.Where("status = ?", mappedStatus)
		} else {
			// 如果不在映射表中，直接使用原值（兼容其他状态值）
			query = query.Where("status = ?", status)
		}
	}

	// 支付方式筛选（精确匹配）
	if paymentMethod != "" && paymentMethod != "all" {
		query = query.Where("payment_method_name = ?", paymentMethod)
	}

	// 日期范围筛选
	if startDate != "" {
		query = query.Where("DATE(created_at) >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("DATE(created_at) <= ?", endDate)
	}

	// 计算总数
	query.Count(&total)

	// 分页和排序
	offset := (page - 1) * size
	if err := query.Order("created_at DESC").Offset(offset).Limit(size).Find(&orders).Error; err != nil {
		// 不向客户端返回详细错误信息，防止信息泄露
		utils.LogError("GetOrders: query orders", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取订单列表失败，请稍后重试",
		})
		return
	}

	// 计算总页数
	pages := (int(total) + size - 1) / size

	// 格式化订单数据，处理支付方式显示
	formattedOrders := make([]gin.H, len(orders))
	for i, order := range orders {
		paymentMethod := ""
		if order.PaymentMethodName.Valid {
			paymentMethod = order.PaymentMethodName.String
		}

		formattedOrders[i] = gin.H{
			"id":                     order.ID,
			"order_no":               order.OrderNo,
			"user_id":                order.UserID,
			"package_id":             order.PackageID,
			"amount":                 order.Amount,
			"status":                 order.Status,
			"payment_method":         paymentMethod, // 精简显示
			"payment_method_name":    paymentMethod, // 兼容字段
			"payment_method_id":      getNullInt64Value(order.PaymentMethodID),
			"payment_time":           getNullTimeValue(order.PaymentTime),
			"payment_transaction_id": getNullStringValue(order.PaymentTransactionID),
			"expire_time":            getNullTimeValue(order.ExpireTime),
			"coupon_id":              getNullInt64Value(order.CouponID),
			"discount_amount":        getNullFloat64Value(order.DiscountAmount),
			"final_amount":           getNullFloat64Value(order.FinalAmount),
			"created_at":             order.CreatedAt,
			"updated_at":             order.UpdatedAt,
		}

		// 添加套餐信息
		if order.Package.ID > 0 {
			formattedOrders[i]["package"] = gin.H{
				"id":           order.Package.ID,
				"name":         order.Package.Name,
				"price":        order.Package.Price,
				"device_limit": order.Package.DeviceLimit,
			}
		}

		// 添加优惠券信息
		if order.Coupon.ID > 0 {
			formattedOrders[i]["coupon"] = gin.H{
				"id":   order.Coupon.ID,
				"code": order.Coupon.Code,
				"name": order.Coupon.Name,
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"orders": formattedOrders,
			"total":  total,
			"page":   page,
			"size":   size,
			"pages":  pages,
		},
	})
}

// 辅助函数：处理NullString
func getNullStringValue(ns sql.NullString) interface{} {
	if ns.Valid {
		return ns.String
	}
	return nil
}

// 辅助函数：处理NullInt64
func getNullInt64Value(ni sql.NullInt64) interface{} {
	if ni.Valid {
		return ni.Int64
	}
	return nil
}

// 辅助函数：处理NullFloat64
func getNullFloat64Value(nf sql.NullFloat64) interface{} {
	if nf.Valid {
		return nf.Float64
	}
	return nil
}

// 辅助函数：处理NullTime
func getNullTimeValue(nt sql.NullTime) interface{} {
	if nt.Valid {
		return nt.Time.Format("2006-01-02 15:04:05")
	}
	return nil
}

// GetOrder 获取单个订单
func GetOrder(c *gin.Context) {
	id := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	isAdmin, _ := c.Get("is_admin")
	admin := isAdmin.(bool)

	db := database.GetDB()
	var order models.Order
	query := db.Preload("Package").Preload("Coupon").Where("id = ?", id)

	if !admin {
		query = query.Where("user_id = ?", user.ID)
	}

	if err := query.First(&order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "订单不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "获取订单失败",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    order,
	})
}

// CancelOrder 取消订单（通过 ID，保留用于兼容）
func CancelOrder(c *gin.Context) {
	id := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	var order models.Order
	if err := db.Where("id = ? AND user_id = ?", id, user.ID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "订单不存在",
		})
		return
	}

	if order.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "订单状态不允许取消",
		})
		return
	}

	order.Status = "cancelled"
	if err := db.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "取消订单失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "订单已取消",
		"data":    order,
	})
}

// CancelOrderByNo 通过订单号取消订单
func CancelOrderByNo(c *gin.Context) {
	orderNo := c.Param("orderNo")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	var order models.Order
	if err := db.Where("order_no = ? AND user_id = ?", orderNo, user.ID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "订单不存在",
		})
		return
	}

	if order.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "订单状态不允许取消",
		})
		return
	}

	order.Status = "cancelled"
	if err := db.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "取消订单失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "订单已取消",
		"data":    order,
	})
}

// GetAdminOrders 管理员获取订单列表
func GetAdminOrders(c *gin.Context) {
	db := database.GetDB()
	var orders []models.Order
	query := db.Preload("User").Preload("Package").Preload("Coupon")

	// 分页参数（支持 page/size 和 skip/limit）
	page := 1
	size := 20
	if pageStr := c.Query("page"); pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if sizeStr := c.Query("size"); sizeStr != "" {
		fmt.Sscanf(sizeStr, "%d", &size)
	}
	// 兼容 skip/limit
	if skipStr := c.Query("skip"); skipStr != "" {
		var skip int
		fmt.Sscanf(skipStr, "%d", &skip)
		if page == 1 && size == 20 {
			page = (skip / size) + 1
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		var limit int
		fmt.Sscanf(limitStr, "%d", &limit)
		if size == 20 {
			size = limit
		}
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}

	// 搜索参数（支持 keyword 和 search）
	keyword := c.Query("keyword")
	if keyword == "" {
		keyword = c.Query("search")
	}
	if keyword != "" {
		// 清理和验证搜索关键词，防止SQL注入
		sanitizedKeyword := utils.SanitizeSearchKeyword(keyword)
		if sanitizedKeyword != "" {
			query = query.Where("order_no LIKE ? OR user_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ?)",
				"%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%")
		}
	}

	// 状态筛选
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	// 使用相同的查询条件计算总数
	countQuery := db.Model(&models.Order{})
	if keyword != "" {
		countQuery = countQuery.Where("order_no LIKE ? OR user_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ?)",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status := c.Query("status"); status != "" {
		countQuery = countQuery.Where("status = ?", status)
	}
	countQuery.Count(&total)

	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取订单列表失败",
		})
		return
	}

	// 转换为前端需要的格式
	orderList := make([]gin.H, 0)
	for _, order := range orders {
		amount := order.Amount
		if order.FinalAmount.Valid {
			amount = order.FinalAmount.Float64
		}

		// 获取支付方式名称
		paymentMethod := ""
		if order.PaymentMethodName.Valid {
			paymentMethod = order.PaymentMethodName.String
		}

		// 获取支付时间
		paymentTime := ""
		if order.PaymentTime.Valid {
			paymentTime = order.PaymentTime.Time.Format("2006-01-02 15:04:05")
		}

		// 构建用户对象（前端期望嵌套的 user 对象）
		userInfo := gin.H{
			"id":       order.User.ID,
			"username": order.User.Username,
			"email":    order.User.Email,
		}

		orderList = append(orderList, gin.H{
			"id":             order.ID,
			"order_no":       order.OrderNo,
			"user_id":        order.UserID,
			"user":           userInfo,            // 嵌套用户信息
			"username":       order.User.Username, // 保留顶层字段以兼容
			"email":          order.User.Email,    // 保留顶层字段以兼容
			"package_id":     order.PackageID,
			"package_name":   order.Package.Name,
			"amount":         amount,
			"payment_method": paymentMethod,
			"payment_time":   paymentTime,
			"status":         order.Status,
			"created_at":     order.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"orders": orderList,
			"items":  orderList, // 兼容前端可能使用的 items 字段
			"total":  total,
			"page":   page,
			"size":   size,
		},
	})
}

// UpdateAdminOrder 管理员更新订单状态
func UpdateAdminOrder(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误，请检查输入格式",
		})
		return
	}

	db := database.GetDB()
	var order models.Order
	if err := db.Preload("Package").Preload("User").First(&order, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "订单不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "获取订单失败",
			})
		}
		return
	}

	oldStatus := order.Status
	order.Status = req.Status

	// 如果订单状态从非paid变为paid，需要处理订阅
	if oldStatus != "paid" && req.Status == "paid" {
		// 设置支付时间
		now := utils.GetBeijingTime()
		order.PaymentTime = database.NullTime(now)

		// 获取用户信息
		var user models.User
		if err := db.First(&user, order.UserID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "获取用户信息失败",
			})
			return
		}

		// 使用统一的处理函数
		svc := orderServicePkg.NewOrderService()
		_, err := svc.ProcessPaidOrder(&order)
		if err != nil {
			// 不向客户端返回详细错误信息，防止信息泄露
			utils.LogError("BulkMarkOrdersPaid: process paid order", err, map[string]interface{}{
				"order_id": order.ID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "处理订单失败，请稍后重试",
			})
			return
		}
	}

	if err := db.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新订单失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "订单已更新",
		"data":    order,
	})
}

// DeleteAdminOrder 管理员删除订单
func DeleteAdminOrder(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()

	// 先检查订单是否存在
	var order models.Order
	if err := db.First(&order, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "订单不存在",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "查询订单失败",
		})
		return
	}

	// 删除订单
	if err := db.Delete(&order).Error; err != nil {
		utils.LogError("DeleteOrder: delete order failed", err, map[string]interface{}{
			"order_id": order.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除订单失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "订单已删除",
	})
}

// GetOrderStatistics 获取订单统计
func GetOrderStatistics(c *gin.Context) {
	db := database.GetDB()

	var totalOrders int64
	var pendingOrders int64
	var paidOrders int64
	var totalRevenue float64

	db.Model(&models.Order{}).Count(&totalOrders)
	db.Model(&models.Order{}).Where("status = ?", "pending").Count(&pendingOrders)
	db.Model(&models.Order{}).Where("status = ?", "paid").Count(&paidOrders)

	// 统计总收入（使用final_amount，如果为NULL则使用amount）
	// 使用原生SQL查询，兼容SQLite和MySQL
	var result struct {
		Total sql.NullFloat64
	}
	db.Raw(`
		SELECT COALESCE(SUM(
			CASE 
				WHEN final_amount IS NOT NULL AND final_amount != 0 THEN final_amount
				ELSE amount
			END
		), 0) as total
		FROM orders 
		WHERE status = ?
	`, "paid").Scan(&result)

	if result.Total.Valid {
		totalRevenue = result.Total.Float64
	} else {
		totalRevenue = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total_orders":   totalOrders,
			"pending_orders": pendingOrders,
			"paid_orders":    paidOrders,
			"total_revenue":  totalRevenue,
		},
	})
}

// BulkMarkOrdersPaid 批量标记订单为已支付
func BulkMarkOrdersPaid(c *gin.Context) {
	var req struct {
		OrderIDs []uint `json:"order_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	if err := db.Model(&models.Order{}).Where("id IN ?", req.OrderIDs).Update("status", "paid").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "批量更新失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "批量标记成功",
	})
}

// BulkCancelOrders 批量取消订单
func BulkCancelOrders(c *gin.Context) {
	var req struct {
		OrderIDs []uint `json:"order_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	if err := db.Model(&models.Order{}).Where("id IN ? AND status = ?", req.OrderIDs, "pending").Update("status", "cancelled").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "批量取消失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "批量取消成功",
	})
}

// BatchDeleteOrders 批量删除订单
func BatchDeleteOrders(c *gin.Context) {
	var req struct {
		OrderIDs []uint `json:"order_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	if err := db.Delete(&models.Order{}, req.OrderIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "批量删除失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "批量删除成功",
	})
}

// ExportOrders 导出订单（CSV格式）
func ExportOrders(c *gin.Context) {
	db := database.GetDB()
	query := db.Preload("User").Preload("Package").Preload("Coupon").Model(&models.Order{})

	// 搜索参数（支持 keyword 和 search）
	keyword := c.Query("keyword")
	if keyword == "" {
		keyword = c.Query("search")
	}
	if keyword != "" {
		// 清理和验证搜索关键词，防止SQL注入
		sanitizedKeyword := utils.SanitizeSearchKeyword(keyword)
		if sanitizedKeyword != "" {
			query = query.Where("order_no LIKE ? OR user_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ?)",
				"%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%")
		}
	}

	// 状态筛选
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var orders []models.Order
	if err := query.Order("created_at DESC").Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取订单列表失败",
		})
		return
	}

	// 生成CSV内容
	var csvContent strings.Builder
	csvContent.WriteString("\xEF\xBB\xBF") // UTF-8 BOM，确保Excel正确显示中文
	csvContent.WriteString("订单号,用户ID,用户名,邮箱,套餐ID,套餐名称,订单金额,支付方式,订单状态,创建时间,支付时间,更新时间\n")

	for _, order := range orders {
		amount := order.Amount
		if order.FinalAmount.Valid {
			amount = order.FinalAmount.Float64
		}

		paymentMethod := ""
		if order.PaymentMethodName.Valid {
			paymentMethod = order.PaymentMethodName.String
		}

		paymentTime := ""
		if order.PaymentTime.Valid {
			paymentTime = order.PaymentTime.Time.Format("2006-01-02 15:04:05")
		}

		username := ""
		email := ""
		if order.User.ID > 0 {
			username = order.User.Username
			email = order.User.Email
		}

		packageName := ""
		if order.Package.ID > 0 {
			packageName = order.Package.Name
		}

		statusText := order.Status
		switch order.Status {
		case "pending":
			statusText = "待支付"
		case "paid":
			statusText = "已支付"
		case "cancelled":
			statusText = "已取消"
		}

		csvContent.WriteString(fmt.Sprintf("%s,%d,%s,%s,%d,%s,%.2f,%s,%s,%s,%s,%s\n",
			order.OrderNo,
			order.UserID,
			username,
			email,
			order.PackageID,
			packageName,
			amount,
			paymentMethod,
			statusText,
			order.CreatedAt.Format("2006-01-02 15:04:05"),
			paymentTime,
			order.UpdatedAt.Format("2006-01-02 15:04:05"),
		))
	}

	// 设置响应头
	filename := fmt.Sprintf("orders_export_%s.csv", time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", filename))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", []byte(csvContent.String()))
}

// GetOrderStats 获取订单统计（用户）
func GetOrderStats(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()

	var stats struct {
		TotalOrders     int64   `json:"total_orders"`
		PendingOrders   int64   `json:"pending_orders"`
		PaidOrders      int64   `json:"paid_orders"`
		CancelledOrders int64   `json:"cancelled_orders"`
		TotalAmount     float64 `json:"total_amount"`
		PaidAmount      float64 `json:"paid_amount"`
	}

	// 统计订单数量
	db.Model(&models.Order{}).Where("user_id = ?", user.ID).Count(&stats.TotalOrders)
	db.Model(&models.Order{}).Where("user_id = ? AND status = ?", user.ID, "pending").Count(&stats.PendingOrders)
	db.Model(&models.Order{}).Where("user_id = ? AND status = ?", user.ID, "paid").Count(&stats.PaidOrders)
	db.Model(&models.Order{}).Where("user_id = ? AND status = ?", user.ID, "cancelled").Count(&stats.CancelledOrders)

	// 统计总金额（所有订单，使用final_amount如果存在，否则使用amount）
	// 使用绝对值，因为金额可能为负数（退款等情况）
	var totalAmountResult struct {
		Total sql.NullFloat64
	}
	// 兼容SQLite和MySQL的查询
	if err := db.Raw(`
		SELECT COALESCE(SUM(
			CASE 
				WHEN final_amount IS NOT NULL AND final_amount != 0 THEN ABS(final_amount)
				ELSE ABS(amount)
			END
		), 0) as total
		FROM orders 
		WHERE user_id = ?
	`, user.ID).Scan(&totalAmountResult).Error; err != nil {
		utils.LogError("GetOrderStats: calculate total amount", err, nil)
		stats.TotalAmount = 0
	} else if totalAmountResult.Total.Valid {
		stats.TotalAmount = totalAmountResult.Total.Float64
	} else {
		stats.TotalAmount = 0
	}

	// 统计已支付金额（只统计已支付订单，使用final_amount如果存在，否则使用amount）
	// 使用绝对值
	var paidAmountResult struct {
		Total sql.NullFloat64
	}
	if err := db.Raw(`
		SELECT COALESCE(SUM(
			CASE 
				WHEN final_amount IS NOT NULL AND final_amount != 0 THEN ABS(final_amount)
				ELSE ABS(amount)
			END
		), 0) as total
		FROM orders 
		WHERE user_id = ? AND status = ?
	`, user.ID, "paid").Scan(&paidAmountResult).Error; err != nil {
		utils.LogError("GetOrderStats: calculate paid amount", err, nil)
		stats.PaidAmount = 0
	} else if paidAmountResult.Total.Valid {
		stats.PaidAmount = paidAmountResult.Total.Float64
	} else {
		stats.PaidAmount = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			// 兼容前端期望的字段名
			"total":       stats.TotalOrders,
			"pending":     stats.PendingOrders,
			"paid":        stats.PaidOrders,
			"cancelled":   stats.CancelledOrders,
			"totalAmount": stats.TotalAmount,
			"paidAmount":  stats.PaidAmount,
			// 保留原有字段名（兼容性）
			"total_orders":     stats.TotalOrders,
			"pending_orders":   stats.PendingOrders,
			"paid_orders":      stats.PaidOrders,
			"cancelled_orders": stats.CancelledOrders,
			"total_amount":     stats.TotalAmount,
			"paid_amount":      stats.PaidAmount,
		},
	})
}

// GetOrderStatusByNo 通过订单号获取订单状态
func GetOrderStatusByNo(c *gin.Context) {
	orderNo := c.Param("orderNo")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	var order models.Order
	if err := db.Where("order_no = ? AND user_id = ?", orderNo, user.ID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "订单不存在",
		})
		return
	}

	// 如果订单是 pending 状态，且创建时间超过5秒，主动查询支付状态
	if order.Status == "pending" {
		// 检查订单创建时间，避免频繁查询
		timeSinceCreated := time.Since(order.CreatedAt)
		if timeSinceCreated > 5*time.Second {
			// 获取支付交易记录
			var transaction models.PaymentTransaction
			if err := db.Where("order_id = ?", order.ID).First(&transaction).Error; err == nil {
				// 获取支付配置
				var paymentConfig models.PaymentConfig
				if err := db.First(&paymentConfig, transaction.PaymentMethodID).Error; err == nil {
					// 根据支付方式查询状态
					if paymentConfig.PayType == "alipay" {
						alipayService, err := payment.NewAlipayService(&paymentConfig)
						if err == nil {
							queryResult, err := alipayService.QueryOrder(orderNo)
							if err == nil && queryResult != nil && queryResult.IsPaid() {
								// 支付成功，更新订单状态（模拟回调处理）
								utils.LogError("GetOrderStatusByNo: payment query success, updating order", nil, map[string]interface{}{
									"order_no":     orderNo,
									"trade_no":     queryResult.TradeNo,
									"trade_status": queryResult.TradeStatus,
								})

								// 使用事务更新订单状态
								tx := db.Begin()
								success := false
								defer func() {
									if !success {
										tx.Rollback()
									}
								}()

								// 重新加载订单（获取最新状态，避免重复处理）
								var latestOrder models.Order
								if err := tx.Where("order_no = ? AND status = ?", orderNo, "pending").First(&latestOrder).Error; err == nil {
									latestOrder.Status = "paid"
									latestOrder.PaymentTime = database.NullTime(utils.GetBeijingTime())
									if err := tx.Save(&latestOrder).Error; err == nil {
										// 更新交易状态
										var latestTransaction models.PaymentTransaction
										if err := tx.Where("order_id = ?", latestOrder.ID).First(&latestTransaction).Error; err == nil {
											latestTransaction.Status = "success"
											latestTransaction.ExternalTransactionID = database.NullString(queryResult.TradeNo)
											if err := tx.Save(&latestTransaction).Error; err == nil {
												if err := tx.Commit().Error; err == nil {
													success = true
													// 订单状态已更新，处理后续逻辑（异步）
													go func() {
														var processedOrder models.Order
														if err := db.Preload("Package").Where("order_no = ?", orderNo).First(&processedOrder).Error; err == nil {
															if processedOrder.Status == "paid" {
																if processedOrder.PackageID > 0 {
																	// 套餐订单
																	var pkg models.Package
																	var processedUser models.User
																	if db.First(&pkg, processedOrder.PackageID).Error == nil &&
																		db.First(&processedUser, processedOrder.UserID).Error == nil {
																		svc := orderServicePkg.NewOrderService()
																		svc.ProcessPaidOrder(&processedOrder)
																	}
																} else {
																	// 设备升级订单，处理逻辑在 PaymentNotify 中
																	// 这里可以触发类似的逻辑
																}
															}
														}
													}()

													// 重新加载订单以返回最新状态
													db.Where("order_no = ?", orderNo).First(&order)
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"order_no": order.OrderNo,
			"status":   order.Status,
			"amount":   order.Amount,
		},
	})
}

// UpgradeDevices 升级设备数（支持支付）
func UpgradeDevices(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	var req struct {
		AdditionalDevices int     `json:"additional_devices" binding:"required,min=5"` // 至少增加5个设备
		AdditionalDays    int     `json:"additional_days"`                             // 可选：延长天数
		PaymentMethod     string  `json:"payment_method"`                              // 支付方式：balance, alipay, wechat, mixed
		UseBalance        bool    `json:"use_balance"`                                 // 是否使用余额
		BalanceAmount     float64 `json:"balance_amount"`                              // 使用的余额金额
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// 不向客户端返回详细错误信息，防止信息泄露
		utils.LogError("UpdateAdminOrder: bind request", err, nil)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误，请检查输入格式",
		})
		return
	}

	db := database.GetDB()

	// 获取用户的订阅
	var subscription models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "订阅不存在",
		})
		return
	}

	// 计算升级费用（假设每个设备每月10元，每天约0.33元）
	devicePricePerMonth := 10.0
	devicePricePerDay := devicePricePerMonth / 30.0
	deviceCost := float64(req.AdditionalDevices) * devicePricePerMonth
	daysCost := float64(req.AdditionalDays) * devicePricePerDay
	totalAmount := deviceCost + daysCost

	// 应用用户等级折扣
	var userLevel models.UserLevel
	if user.UserLevelID.Valid {
		if err := db.First(&userLevel, user.UserLevelID.Int64).Error; err == nil {
			if userLevel.DiscountRate > 0 && userLevel.DiscountRate < 1.0 {
				totalAmount *= userLevel.DiscountRate
			}
		}
	}

	// 处理余额支付
	balanceUsed := 0.0
	finalAmount := totalAmount
	if req.UseBalance {
		if user.Balance <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "您的余额为0，无法使用余额支付",
			})
			return
		}

		if req.BalanceAmount > user.Balance {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "余额不足",
			})
			return
		}

		if req.BalanceAmount > finalAmount {
			balanceUsed = finalAmount
		} else {
			balanceUsed = req.BalanceAmount
		}

		finalAmount -= balanceUsed
	}

	// 如果最终金额为0（被余额完全抵扣），直接升级
	if finalAmount <= 0.01 {
		// 扣除余额
		if balanceUsed > 0 {
			user.Balance -= balanceUsed
			if err := db.Save(&user).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": "扣除余额失败",
				})
				return
			}
		}

		// 升级设备数量
		subscription.DeviceLimit += req.AdditionalDevices
		if req.AdditionalDays > 0 {
			now := utils.GetBeijingTime()
			if subscription.ExpireTime.Before(now) {
				subscription.ExpireTime = now.AddDate(0, 0, req.AdditionalDays)
			} else {
				subscription.ExpireTime = subscription.ExpireTime.AddDate(0, 0, req.AdditionalDays)
			}
		}
		if err := db.Save(&subscription).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "升级设备数失败",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "设备数量升级成功",
			"data": gin.H{
				"status":             "paid",
				"subscription":       subscription,
				"additional_devices": req.AdditionalDevices,
				"additional_days":    req.AdditionalDays,
			},
		})
		return
	}

	// 需要支付，创建订单
	orderNo := utils.GenerateOrderNo(user.ID)
	order := models.Order{
		OrderNo:           orderNo,
		UserID:            user.ID,
		PackageID:         0, // 设备升级订单，PackageID 为 0
		Amount:            totalAmount,
		FinalAmount:       database.NullFloat64(finalAmount),
		DiscountAmount:    database.NullFloat64(totalAmount - finalAmount),
		Status:            "pending",
		PaymentMethodName: database.NullString(req.PaymentMethod),
		// 订单类型标记为设备升级
		// 注意：这里 PackageID 为 0，表示这是设备升级订单，不是套餐订单
	}

	// 扣除已使用的余额
	if balanceUsed > 0 {
		user.Balance -= balanceUsed
		if err := db.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "扣除余额失败",
			})
			return
		}
	}

	if err := db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建订单失败",
		})
		return
	}

	// 生成支付URL（如果需要其他支付方式）
	var paymentURL string
	if finalAmount > 0.01 && req.PaymentMethod != "" && req.PaymentMethod != "balance" {
		var paymentConfig models.PaymentConfig
		payType := req.PaymentMethod
		if payType == "alipay" || payType == "wechat" {
			// 查找对应类型的支付配置（不区分大小写）
			if err := db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", payType, 1).Order("sort_order ASC").First(&paymentConfig).Error; err == nil {
				// 创建支付交易
				transaction := models.PaymentTransaction{
					OrderID:         order.ID,
					UserID:          user.ID,
					PaymentMethodID: paymentConfig.ID,
					Amount:          int(finalAmount * 100),
					Currency:        "CNY",
					Status:          "pending",
				}
				if err := db.Create(&transaction).Error; err == nil {
					// 生成支付URL
					if paymentConfig.PayType == "alipay" {
						alipayService, err := payment.NewAlipayService(&paymentConfig)
						if err == nil {
							paymentURL, err = alipayService.CreatePayment(&order, finalAmount)
							if err != nil {
								fmt.Printf("生成支付宝支付URL失败: %v\n", err)
							}
						}
					} else if paymentConfig.PayType == "wechat" {
						wechatService, err := payment.NewWechatService(&paymentConfig)
						if err == nil {
							paymentURL, err = wechatService.CreatePayment(&order, finalAmount)
							if err != nil {
								fmt.Printf("生成微信支付URL失败: %v\n", err)
							}
						}
					}
				}
			}
		}
	}

	// 在订单 ExtraData 中记录升级信息（用于支付回调后处理）
	extraData := fmt.Sprintf(`{"type":"device_upgrade","additional_devices":%d,"additional_days":%d}`, req.AdditionalDevices, req.AdditionalDays)
	order.ExtraData = database.NullString(extraData)
	if err := db.Save(&order).Error; err != nil {
		fmt.Printf("保存订单ExtraData失败: %v\n", err)
	}

	// 返回订单和支付URL
	responseData := gin.H{
		"order_no":           order.OrderNo,
		"id":                 order.ID,
		"status":             order.Status,
		"amount":             totalAmount,
		"final_amount":       finalAmount,
		"additional_devices": req.AdditionalDevices,
		"additional_days":    req.AdditionalDays,
	}

	if paymentURL != "" {
		responseData["payment_url"] = paymentURL
		responseData["payment_qr_code"] = paymentURL
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    responseData,
	})
}

// PayOrder 支付订单
func PayOrder(c *gin.Context) {
	orderNo := c.Param("orderNo")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	var req struct {
		PaymentMethodID uint `json:"payment_method_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误: 缺少 payment_method_id 参数",
			"error":   err.Error(),
		})
		return
	}

	db := database.GetDB()
	var order models.Order
	if err := db.Where("order_no = ? AND user_id = ?", orderNo, user.ID).First(&order).Error; err != nil {
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

	// 计算支付金额
	amount := order.Amount
	if order.FinalAmount.Valid {
		amount = order.FinalAmount.Float64
	}

	// 创建支付交易
	transaction := models.PaymentTransaction{
		OrderID:         order.ID,
		UserID:          user.ID,
		PaymentMethodID: req.PaymentMethodID,
		Amount:          int(amount * 100), // 转换为分
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

	// 根据支付方式生成支付 URL
	var paymentURL string
	var payErr error

	if paymentConfig.PayType == "alipay" {
		alipayService, err := payment.NewAlipayService(&paymentConfig)
		if err != nil {
			payErr = fmt.Errorf("初始化支付宝服务失败: %v", err)
		} else {
			paymentURL, payErr = alipayService.CreatePayment(&order, amount)
		}
	} else if paymentConfig.PayType == "wechat" {
		wechatService, err := payment.NewWechatService(&paymentConfig)
		if err != nil {
			payErr = fmt.Errorf("初始化微信支付服务失败: %v", err)
		} else {
			paymentURL, payErr = wechatService.CreatePayment(&order, amount)
		}
	} else {
		payErr = fmt.Errorf("不支持的支付方式: %s", paymentConfig.PayType)
	}

	if payErr != nil {
		utils.LogError("CreateOrder: create payment failed", payErr, map[string]interface{}{
			"order_id": order.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建支付失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "支付订单已创建",
		"data": gin.H{
			"payment_url":    paymentURL,
			"order_no":       order.OrderNo,
			"amount":         amount,
			"transaction_id": transaction.ID,
		},
	})
}

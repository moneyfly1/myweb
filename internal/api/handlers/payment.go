package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
	res := make([]gin.H, 0)
	mMap := map[string]string{
		"alipay":       "支付宝",
		"wechat":       "微信支付",
		"yipay_alipay": "易支付-支付宝",
		"yipay_wxpay":  "易支付-微信",
		"yipay_qqpay":  "易支付-QQ钱包",
		"paypal":       "PayPal",
		"applepay":     "Apple Pay",
		"stripe":       "Stripe",
		"bank":         "银行转账",
	}

	yipaySubTypeMap := map[string]string{
		"alipay": "支付宝",
		"wxpay":  "微信支付",
		"qqpay":  "QQ钱包",
	}

	for _, m := range cfg {
		if m.PayType == "yipay" {
			supportedTypes := payment.GetYipaySupportedTypes(&m)
			for _, t := range supportedTypes {
				subTypeKey := fmt.Sprintf("yipay_%s", t)
				subTypeName := yipaySubTypeMap[t]
				if subTypeName == "" {
					subTypeName = t
				}
				res = append(res, gin.H{
					"id":     m.ID,
					"key":    subTypeKey,
					"name":   fmt.Sprintf("易支付-%s", subTypeName),
					"status": m.Status,
				})
			}
		} else if strings.HasPrefix(m.PayType, "yipay_") {
			name := mMap[m.PayType]
			if name == "" {
				name = m.PayType
			}
			res = append(res, gin.H{
				"id":     m.ID,
				"key":    m.PayType,
				"name":   name,
				"status": m.Status,
			})
		} else {
			name := mMap[m.PayType]
			if name == "" {
				name = m.PayType
			}
			res = append(res, gin.H{
				"id":     m.ID,
				"key":    m.PayType,
				"name":   name,
				"status": m.Status,
			})
		}
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

func PaymentNotify(c *gin.Context) {
	paymentType := c.Param("type")
	db := database.GetDB()

	params := make(map[string]string)

	if c.Request.Method == "POST" {
		contentType := c.Request.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/x-www-form-urlencoded") {
			if err := c.Request.ParseForm(); err == nil {
				for k, v := range c.Request.PostForm {
					if len(v) > 0 {
						params[k] = v[0]
					}
				}
			}
		} else if strings.Contains(contentType, "multipart/form-data") {
			if err := c.Request.ParseMultipartForm(32 << 20); err == nil { // 32MB max
				if c.Request.MultipartForm != nil && c.Request.MultipartForm.Value != nil {
					for k, v := range c.Request.MultipartForm.Value {
						if len(v) > 0 {
							params[k] = v[0]
						}
					}
				}
			}
		} else if strings.Contains(contentType, "application/json") {
			var jsonParams map[string]interface{}
			if err := c.ShouldBindJSON(&jsonParams); err == nil {
				for k, v := range jsonParams {
					if str, ok := v.(string); ok {
						params[k] = str
					} else {
						params[k] = fmt.Sprintf("%v", v)
					}
				}
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

	utils.LogInfo("PaymentNotify: 解析后的参数 - 原始参数数量=%d", len(params))
	for k, v := range params {
		if k != "sign" && k != "rsa_sign" {
			utils.LogInfo("PaymentNotify: 参数[%s]=%s (长度=%d)", k, v, len(v))
		} else {
			utils.LogInfo("PaymentNotify: 参数[%s]=*** (隐藏)", k)
		}
	}

	utils.LogInfo("PaymentNotify: 收到回调请求 - payment_type=%s, method=%s, content_type=%s, params_count=%d, url=%s, remote_addr=%s",
		paymentType, c.Request.Method, c.Request.Header.Get("Content-Type"), len(params), c.Request.URL.String(), c.ClientIP())

	safeParams := make(map[string]string)
	for k, v := range params {
		if k != "sign" && k != "rsa_sign" {
			safeParams[k] = v
		} else {
			safeParams[k] = "***"
		}
	}
	utils.LogInfo("PaymentNotify: 回调参数 - %+v", safeParams)

	var paymentConfig models.PaymentConfig
	queryPayType := paymentType
	if paymentType == "yipay" || strings.HasPrefix(paymentType, "yipay_") {
		if err := db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", "yipay", 1).First(&paymentConfig).Error; err == nil {
			queryPayType = "yipay"
		} else {
			if err := db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", paymentType, 1).First(&paymentConfig).Error; err != nil {
				utils.LogError("PaymentNotify: payment config not found", err, map[string]interface{}{
					"payment_type": paymentType,
					"query_type":   queryPayType,
				})
				c.String(http.StatusBadRequest, "支付配置不存在")
				return
			}
		}
	} else {
		if err := db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", paymentType, 1).First(&paymentConfig).Error; err != nil {
			utils.LogError("PaymentNotify: payment config not found", err, map[string]interface{}{
				"payment_type": paymentType,
			})
			c.String(http.StatusBadRequest, "支付配置不存在")
			return
		}
	}

	utils.LogInfo("PaymentNotify: 找到支付配置 - payment_type=%s, config_id=%d, config_pay_type=%s",
		paymentType, paymentConfig.ID, paymentConfig.PayType)

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
	case "yipay", "yipay_alipay", "yipay_wxpay", "yipay_qqpay":
		yipayService, err := payment.NewYipayService(&paymentConfig)
		if err != nil {
			utils.LogError("PaymentNotify: 创建易支付服务失败", err, map[string]interface{}{
				"payment_type": paymentType,
			})
		} else {
			verified = yipayService.VerifyNotify(params)
			utils.LogInfo("PaymentNotify: 易支付签名验证结果 - verified=%v, payment_type=%s", verified, paymentType)
		}
	}

	if !verified {
		utils.LogError("PaymentNotify: signature verification failed", nil, map[string]interface{}{
			"payment_type": paymentType,
			"order_no":     params["out_trade_no"],
			"params_count": len(params),
		})
		if paymentType == "yipay" || strings.HasPrefix(paymentType, "yipay_") {
			c.String(http.StatusOK, "fail")
		} else {
			c.String(http.StatusBadRequest, "签名验证失败")
		}
		return
	}

	orderNo := params["out_trade_no"]
	externalTransactionID := params["trade_no"]

	if paymentType == "yipay" || strings.HasPrefix(paymentType, "yipay_") {
		tradeStatus := params["trade_status"]
		if tradeStatus == "" {
			tradeStatus = params["status"]
		}

		if tradeStatus == "TRADE_FREEZE" {
			utils.LogWarn("PaymentNotify: yipay trade frozen", map[string]interface{}{
				"payment_type": paymentType,
				"order_no":     orderNo,
				"trade_status": tradeStatus,
			})
			c.String(http.StatusOK, "success")
			return
		}

		if tradeStatus == "TRADE_UNFREEZE" {
			utils.LogInfo("PaymentNotify: yipay trade unfrozen", map[string]interface{}{
				"payment_type": paymentType,
				"order_no":     orderNo,
				"trade_status": tradeStatus,
			})
			c.String(http.StatusOK, "success")
			return
		}

		if tradeStatus != "TRADE_SUCCESS" {
			utils.LogWarn("PaymentNotify: yipay trade status not success", map[string]interface{}{
				"payment_type": paymentType,
				"order_no":     orderNo,
				"trade_status": tradeStatus,
			})
			c.String(http.StatusOK, "success")
			return
		}

		utils.LogInfo("PaymentNotify: 易支付订单状态为TRADE_SUCCESS，继续处理 - order_no=%s, trade_status=%s", orderNo, tradeStatus)
	}

	if paymentType == "alipay" {
		tradeStatus := params["trade_status"]
		if tradeStatus != "TRADE_SUCCESS" && tradeStatus != "TRADE_FINISHED" {
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

	utils.LogInfo("PaymentNotify: 收到支付回调 - payment_type=%s, order_no=%s, external_transaction_id=%s",
		paymentType, orderNo, externalTransactionID)

	var order models.Order
	var recharge models.RechargeRecord
	isRecharge := false

	if err := db.Preload("Package").Where("order_no = ?", orderNo).First(&order).Error; err != nil {
		if err2 := db.Where("order_no = ?", orderNo).First(&recharge).Error; err2 == nil {
			isRecharge = true
		} else {
			utils.LogError("PaymentNotify: order or recharge not found", err, map[string]interface{}{
				"order_no": orderNo,
			})
			if paymentType == "yipay" || strings.HasPrefix(paymentType, "yipay_") {
				c.String(http.StatusOK, "success")
			} else {
				c.String(http.StatusBadRequest, "订单或充值记录不存在")
			}
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

		err := utils.WithTransaction(db, func(tx *gorm.DB) error {
			recharge.Status = "paid"
			recharge.PaidAt = database.NullTime(utils.GetBeijingTime())
			if externalTransactionID != "" {
				recharge.PaymentTransactionID = database.NullString(externalTransactionID)
			}
			if !recharge.PaymentMethod.Valid || recharge.PaymentMethod.String == "" {
				recharge.PaymentMethod = database.NullString(paymentType)
			}
			if err := tx.Save(&recharge).Error; err != nil {
				utils.LogError("PaymentNotify: failed to update recharge", err, map[string]interface{}{
					"order_no": orderNo,
				})
				return err
			}

			var user models.User
			if err := tx.First(&user, recharge.UserID).Error; err == nil {
				oldBalance := user.Balance
				user.Balance += recharge.Amount
				if err := tx.Save(&user).Error; err != nil {
					utils.LogError("PaymentNotify: failed to update user balance", err, map[string]interface{}{
						"order_no": orderNo,
						"user_id":  user.ID,
					})
					return err
				}
				utils.LogInfo("PaymentNotify: 充值成功 - order_no=%s, user_id=%d, amount=%.2f, old_balance=%.2f, new_balance=%.2f",
					orderNo, user.ID, recharge.Amount, oldBalance, user.Balance)
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

		utils.LogInfo("PaymentNotify: 充值回调处理成功 - order_no=%s, user_id=%d, amount=%.2f, payment_type=%s",
			orderNo, recharge.UserID, recharge.Amount, paymentType)

		c.String(http.StatusOK, "success")
		return
	}

	if paymentType == "alipay" {
		if amountStr, ok := params["total_amount"]; ok {
			var callbackAmount float64
			fmt.Sscanf(amountStr, "%f", &callbackAmount)
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

	if order.Status == "paid" {
		utils.LogInfo("PaymentNotify: order already paid, ensuring subscription is activated - order_no=%s", orderNo)
		go func(orderNoParam string) {
			defer func() {
				if r := recover(); r != nil {
					utils.LogError("PaymentNotify: panic in async processing (already paid)", fmt.Errorf("%v", r), map[string]interface{}{
						"order_no": orderNoParam,
					})
				}
			}()

			var freshOrder models.Order
			if err := db.Preload("Package").Where("order_no = ?", orderNoParam).First(&freshOrder).Error; err != nil {
				utils.LogError("PaymentNotify: 重新加载订单失败 (already paid)", err, map[string]interface{}{
					"order_no": orderNoParam,
				})
				return
			}

			var subscription models.Subscription
			if err := db.Where("user_id = ?", freshOrder.UserID).First(&subscription).Error; err != nil {
				utils.LogInfo("PaymentNotify: subscription not found, reprocessing order to activate - order_no=%s", orderNoParam)
				orderService := orderServicePkg.NewOrderService()
				if _, err := orderService.ProcessPaidOrder(&freshOrder); err != nil {
					utils.LogError("PaymentNotify: failed to reprocess order for subscription activation", err, map[string]interface{}{
						"order_no": orderNoParam,
					})
				} else {
					utils.LogInfo("PaymentNotify: subscription activated successfully - order_no=%s, package_id=%d", orderNoParam, freshOrder.PackageID)
				}
			} else {
				utils.LogInfo("PaymentNotify: subscription already exists, sending notifications - order_no=%s", orderNoParam)
			}

			sendPaymentNotifications(db, orderNoParam)
		}(orderNo)
		c.String(http.StatusOK, "success")
		return
	}

	err := utils.WithTransaction(db, func(tx *gorm.DB) error {
		var freshOrder models.Order
		if err := tx.Preload("Package").Where("order_no = ?", orderNo).First(&freshOrder).Error; err != nil {
			utils.LogError("PaymentNotify: 事务中重新加载订单失败", err, map[string]interface{}{
				"order_no": orderNo,
			})
			return err
		}

		if freshOrder.Status == "paid" {
			utils.LogInfo("PaymentNotify: 订单已经是paid状态，跳过更新 - order_no=%s", orderNo)
			order = freshOrder
			return nil
		}

		freshOrder.Status = "paid"
		freshOrder.PaymentTime = database.NullTime(utils.GetBeijingTime())
		if err := tx.Save(&freshOrder).Error; err != nil {
			utils.LogError("PaymentNotify: failed to update order", err, map[string]interface{}{
				"order_no": orderNo,
			})
			return err
		}
		order = freshOrder

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
		if paymentType == "yipay" || strings.HasPrefix(paymentType, "yipay_") {
			c.String(http.StatusOK, "fail")
		} else {
			c.String(http.StatusInternalServerError, "处理失败")
		}
		return
	}

	utils.LogInfo("PaymentNotify: 订单状态已更新为paid - order_no=%s, order_id=%d, status=%s", orderNo, order.ID, order.Status)

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

	utils.LogInfo("PaymentNotify: 订单状态已更新为paid，开始处理订单 - order_no=%s, order_id=%d", orderNo, order.ID)

	go func(orderNoParam string) {
		defer func() {
			if r := recover(); r != nil {
				utils.LogError("PaymentNotify: panic in async processing", fmt.Errorf("%v", r), map[string]interface{}{
					"order_no": orderNoParam,
				})
			}
		}()

		utils.LogInfo("PaymentNotify: 开始处理已支付订单 - order_no=%s", orderNoParam)

		var freshOrder models.Order
		if err := db.Preload("Package").Where("order_no = ?", orderNoParam).First(&freshOrder).Error; err != nil {
			utils.LogError("PaymentNotify: 重新加载订单失败", err, map[string]interface{}{
				"order_no": orderNoParam,
			})
			return
		}

		if freshOrder.Status != "paid" {
			utils.LogWarn("PaymentNotify: 订单状态不是paid，跳过处理 - order_no=%s, status=%s", orderNoParam, freshOrder.Status)
			return
		}

		orderService := orderServicePkg.NewOrderService()
		_, processErr := orderService.ProcessPaidOrder(&freshOrder)
		if processErr != nil {
			utils.LogError("PaymentNotify: process paid order failed", processErr, map[string]interface{}{
				"order_id": freshOrder.ID,
				"order_no": orderNoParam,
			})
		} else {
			utils.LogInfo("PaymentNotify: 订单处理成功，套餐已开通 - order_no=%s, package_id=%d", orderNoParam, freshOrder.PackageID)
		}

		sendPaymentNotifications(db, orderNoParam)
	}(orderNo)

	c.String(http.StatusOK, "success")
}

func sendPaymentNotifications(db *gorm.DB, orderNo string) {
	var latestOrder models.Order
	if err := db.Preload("Package").Where("order_no = ?", orderNo).First(&latestOrder).Error; err != nil {
		utils.LogErrorMsg("sendPaymentNotifications: 查询订单失败: order_no=%s, error=%v", orderNo, err)
		return
	}

	var latestUser models.User
	if err := db.First(&latestUser, latestOrder.UserID).Error; err != nil {
		utils.LogErrorMsg("sendPaymentNotifications: 查询用户失败: order_no=%s, user_id=%d, error=%v", orderNo, latestOrder.UserID, err)
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

	if notification.ShouldSendCustomerNotification("new_order") {
		emailService := email.NewEmailService()
		templateBuilder := email.NewEmailTemplateBuilder()

		paymentSuccessContent := templateBuilder.GetPaymentSuccessTemplate(
			latestUser.Username,
			latestOrder.OrderNo,
			packageName,
			paidAmount,
			paymentMethod,
			paymentTime,
		)
		if err := emailService.QueueEmail(latestUser.Email, "支付成功通知", paymentSuccessContent, "payment_success"); err != nil {
			utils.LogErrorMsg("sendPaymentNotifications: 发送付款成功邮件失败: order_no=%s, email=%s, error=%v", latestOrder.OrderNo, latestUser.Email, err)
		} else {
			utils.LogInfo("sendPaymentNotifications: 付款成功邮件已加入队列: order_no=%s, email=%s", latestOrder.OrderNo, latestUser.Email)
		}

		if latestOrder.PackageID > 0 {
			var subscriptionInfo models.Subscription
			if err := db.Where("user_id = ?", latestUser.ID).First(&subscriptionInfo).Error; err == nil {
				baseURL := templateBuilder.GetBaseURL()
				universalURL := fmt.Sprintf("%s/api/v1/subscriptions/universal/%s", baseURL, subscriptionInfo.SubscriptionURL)
				clashURL := fmt.Sprintf("%s/api/v1/subscriptions/clash/%s", baseURL, subscriptionInfo.SubscriptionURL)

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
					utils.LogErrorMsg("sendPaymentNotifications: 发送订阅配置邮件失败: order_no=%s, email=%s, error=%v", latestOrder.OrderNo, latestUser.Email, err)
				} else {
					utils.LogInfo("sendPaymentNotifications: 订阅配置邮件已加入队列: order_no=%s, email=%s", latestOrder.OrderNo, latestUser.Email)
				}
			} else {
				utils.LogErrorMsg("sendPaymentNotifications: 查询订阅信息失败: order_no=%s, user_id=%d, error=%v", latestOrder.OrderNo, latestUser.ID, err)
			}
		}
	} else {
		utils.LogInfo("sendPaymentNotifications: 客户通知已禁用，跳过发送: order_no=%s", latestOrder.OrderNo)
	}

	notificationService := notification.NewNotificationService()
	if err := notificationService.SendAdminNotification("order_paid", map[string]interface{}{
		"order_no":       latestOrder.OrderNo,
		"username":       latestUser.Username,
		"amount":         paidAmount,
		"package_name":   packageName,
		"payment_method": paymentMethod,
		"payment_time":   paymentTime,
	}); err != nil {
		utils.LogErrorMsg("sendPaymentNotifications: 发送管理员通知失败: order_no=%s, error=%v", latestOrder.OrderNo, err)
	} else {
		utils.LogInfo("sendPaymentNotifications: 管理员通知已发送: order_no=%s", latestOrder.OrderNo)
	}
}

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

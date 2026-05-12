package handlers

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/cache_service"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/notification"
	orderServicePkg "cboard-go/internal/services/order"
	"cboard-go/internal/services/payment"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func GetPaymentMethods(c *gin.Context) {
	cacheService := cache_service.NewCacheService()

	// 尝试从缓存获取
	if cached, ok := cacheService.GetPaymentMethodsCache(); ok {
		utils.SuccessResponse(c, http.StatusOK, "", cached)
		return
	}

	db := database.GetDB()
	var cfg []models.PaymentConfig
	db.Where("status = ?", 1).Order("sort_order ASC").Find(&cfg)
	res := make([]gin.H, 0)
	mMap := map[string]string{
		"alipay":         "支付宝",
		"wechat":         "微信支付",
		"yipay_alipay":   "易支付-支付宝",
		"yipay_wxpay":    "易支付-微信",
		"yipay_qqpay":    "易支付-QQ钱包",
		"applepay":       "Apple Pay",
		"codepay_alipay": "码支付-支付宝",
		"codepay_wxpay":  "码支付-微信",
	}

	yipaySubTypeMap := map[string]string{
		"alipay": "支付宝",
		"wxpay":  "微信支付",
		"qqpay":  "QQ钱包",
	}

	codepaySubTypeMap := map[string]string{
		"alipay": "支付宝",
		"wxpay":  "微信支付",
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
		} else if m.PayType == "codepay" {
			supportedTypes := payment.GetCodepaySupportedTypes(&m)
			for _, t := range supportedTypes {
				subTypeKey := fmt.Sprintf("codepay_%s", t)
				subTypeName := codepaySubTypeMap[t]
				if subTypeName == "" {
					subTypeName = t
				}
				res = append(res, gin.H{
					"id":     m.ID,
					"key":    subTypeKey,
					"name":   fmt.Sprintf("码支付-%s", subTypeName),
					"status": m.Status,
				})
			}
		} else if strings.HasPrefix(m.PayType, "codepay_") {
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

	// 转换为 []map[string]interface{} 以便缓存
	cacheData := make([]map[string]interface{}, len(res))
	for i, item := range res {
		cacheData[i] = map[string]interface{}(item)
	}

	// 异步写入缓存
	go cacheService.SetPaymentMethodsCache(cacheData)

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
	amount := order.Amount
	if order.FinalAmount.Valid {
		amount = order.FinalAmount.Float64
	}
	if amount <= 0.01 {
		if _, err := orderServicePkg.NewOrderService().FinalizePaidOrder(order.OrderNo, orderServicePkg.FinalizePaidOrderOptions{
			PaymentMethodName: "优惠抵扣",
			IPAddress:         utils.GetRealClientIP(c),
		}); err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "处理订单失败", err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "支付成功", gin.H{
			"order_no": order.OrderNo,
			"status":   "paid",
			"amount":   0,
		})
		return
	}
	if cfg.PayType != "" && !order.PaymentMethodID.Valid {
		order.PaymentMethodID = database.NullInt64(utils.MustSafeUintToInt64(cfg.ID))
		if !order.PaymentMethodName.Valid || strings.TrimSpace(order.PaymentMethodName.String) == "" {
			order.PaymentMethodName = database.NullString(cfg.PayType)
		}
		if err := db.Model(&order).Updates(map[string]interface{}{
			"payment_method_id":   order.PaymentMethodID,
			"payment_method_name": order.PaymentMethodName,
		}).Error; err != nil {
			utils.LogError("CreatePayment: update order payment method failed", err, map[string]interface{}{"order_no": order.OrderNo})
		}
	}

	amt := int(math.Round(amount * 100))
	tx := models.PaymentTransaction{
		OrderID:         order.ID,
		UserID:          u.ID,
		PaymentMethodID: cfg.ID,
		Amount:          amt,
		Currency:        "CNY",
		Status:          "pending",
	}
	if err := db.Create(&tx).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建支付交易失败", err)
		return
	}

	paymentURL, err := generatePaymentURL(db, &order, &cfg, cfg.PayType, amount)
	if err != nil {
		if failedOrder, markErr := orderServicePkg.NewOrderService().MarkPendingOrderStatus(order.OrderNo, u.ID, "failed"); markErr != nil {
			utils.LogError("CreatePayment: mark order failed after payment link error", markErr, map[string]interface{}{
				"order_no": order.OrderNo,
			})
		} else if failedOrder != nil {
			order = *failedOrder
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建支付链接失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"transaction_id":        tx.ID,
		"amount":                amount,
		"final_amount":          amount,
		"actual_payment_amount": amount,
		"order_no":              order.OrderNo,
		"status":                order.Status,
		"payment_url":           paymentURL,
		"payment_qr_code":       paymentURL,
		"payment_method":        cfg.PayType,
	})
}

func parseCallbackAmount(paymentType string, params map[string]string) (float64, bool) {
	var callbackAmount float64
	if paymentType == "alipay" || paymentType == "applepay" {
		if amountStr := params["total_amount"]; amountStr != "" {
			_, _ = fmt.Sscanf(amountStr, "%f", &callbackAmount)
			return callbackAmount, true
		}
		if amountStr := params["amount"]; amountStr != "" {
			_, _ = fmt.Sscanf(amountStr, "%f", &callbackAmount)
			return callbackAmount, true
		}
		return 0, false
	}

	if paymentType == "wechat" {
		if amountStr := params["total_fee"]; amountStr != "" {
			var amountInCents int
			_, _ = fmt.Sscanf(amountStr, "%d", &amountInCents)
			return float64(amountInCents) / 100.0, true
		}
		return 0, false
	}

	if strings.HasPrefix(paymentType, "yipay") || strings.HasPrefix(paymentType, "codepay") {
		amountStr := params["money"]
		if amountStr == "" {
			amountStr = params["price"]
		}
		if amountStr == "" {
			amountStr = params["amount"]
		}
		if amountStr != "" {
			_, _ = fmt.Sscanf(amountStr, "%f", &callbackAmount)
			return callbackAmount, true
		}
	}

	return 0, false
}

func amountMatches(expected, actual float64) bool {
	return actual >= expected-0.01 && actual <= expected+0.01
}

func isAsyncAckPaymentType(paymentType string) bool {
	return paymentType == "yipay" || strings.HasPrefix(paymentType, "yipay_") ||
		paymentType == "codepay" || strings.HasPrefix(paymentType, "codepay_")
}

func paymentCallbackAck(c *gin.Context, paymentType string, success bool, message string) {
	if paymentType == "wechat" {
		returnCode := "SUCCESS"
		returnMsg := "OK"
		if !success {
			returnCode = "FAIL"
			returnMsg = message
			if returnMsg == "" {
				returnMsg = "处理失败"
			}
		}
		c.Header("Content-Type", "text/xml; charset=utf-8")
		c.String(http.StatusOK, "<xml><return_code><![CDATA[%s]]></return_code><return_msg><![CDATA[%s]]></return_msg></xml>", returnCode, returnMsg)
		return
	}
	if isAsyncAckPaymentType(paymentType) {
		if success {
			c.String(http.StatusOK, "success")
		} else {
			c.String(http.StatusOK, "fail")
		}
		return
	}
	if success {
		c.String(http.StatusOK, "success")
		return
	}
	c.String(http.StatusInternalServerError, message)
}

func parseXMLPaymentParams(body []byte) (map[string]string, error) {
	params := make(map[string]string)
	decoder := xml.NewDecoder(strings.NewReader(string(body)))
	var currentKey string
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := token.(type) {
		case xml.StartElement:
			currentKey = t.Name.Local
		case xml.CharData:
			if currentKey != "" && currentKey != "xml" {
				value := strings.TrimSpace(string(t))
				if value != "" {
					params[currentKey] = value
				}
			}
		case xml.EndElement:
			if currentKey == t.Name.Local {
				currentKey = ""
			}
		}
	}
	return params, nil
}

func updatePaymentTransactionTx(tx *gorm.DB, orderID uint, userID uint, amountCents int, externalTransactionID string, paymentMethodID uint, callbackData string, transactionID string) error {
	var paymentTx models.PaymentTransaction
	query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("status = ?", "pending")
	if orderID > 0 {
		query = query.Where("order_id = ?", orderID)
	} else if transactionID != "" {
		query = query.Where("order_id = ? AND transaction_id = ?", 0, transactionID)
	} else {
		query = query.Where("order_id = ? AND user_id = ? AND amount = ?", 0, userID, amountCents)
	}
	if paymentMethodID > 0 {
		query = query.Where("payment_method_id = ?", paymentMethodID)
	}
	if err := query.Order("created_at DESC").First(&paymentTx).Error; err != nil {
		return err
	}
	paymentTx.Status = "success"
	if externalTransactionID != "" {
		paymentTx.ExternalTransactionID = database.NullString(externalTransactionID)
	}
	if callbackData != "" {
		paymentTx.CallbackData = database.NullString(callbackData)
	}
	return tx.Save(&paymentTx).Error
}

func processPaidRecharge(db *gorm.DB, orderNo string, paymentType string, paymentConfigID uint, externalTransactionID string, params map[string]string, ipAddress string) (*models.RechargeRecord, error) {
	var result models.RechargeRecord
	var freshUser models.User
	paidNow := false
	callbackData := ""
	if data, err := json.Marshal(params); err == nil {
		callbackData = string(data)
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		var recharge models.RechargeRecord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_no = ?", orderNo).First(&recharge).Error; err != nil {
			return err
		}

		if recharge.Status == "paid" {
			result = recharge
			return nil
		}
		if recharge.Status != "pending" {
			return fmt.Errorf("充值订单状态不允许入账: %s", recharge.Status)
		}

		if callbackAmount, ok := parseCallbackAmount(paymentType, params); ok {
			if !amountMatches(recharge.Amount, callbackAmount) {
				return fmt.Errorf("充值金额不匹配: 预期%.2f元, 回调%.2f元", recharge.Amount, callbackAmount)
			}
		} else {
			utils.LogWarn("PaymentNotify: unable to verify recharge amount for payment type: %+v", map[string]interface{}{
				"order_no":     orderNo,
				"payment_type": paymentType,
			})
		}

		recharge.Status = "paid"
		recharge.PaidAt = database.NullTime(utils.GetBeijingTime())
		if externalTransactionID != "" {
			recharge.PaymentTransactionID = database.NullString(externalTransactionID)
		}
		if !recharge.PaymentMethod.Valid || recharge.PaymentMethod.String == "" {
			recharge.PaymentMethod = database.NullString(paymentType)
		}
		if err := tx.Save(&recharge).Error; err != nil {
			return err
		}

		if err := updatePaymentTransactionTx(tx, 0, recharge.UserID, int(math.Round(recharge.Amount*100)), externalTransactionID, paymentConfigID, callbackData, orderNo); err != nil {
			if fallbackErr := updatePaymentTransactionTx(tx, 0, recharge.UserID, int(math.Round(recharge.Amount*100)), externalTransactionID, paymentConfigID, callbackData, ""); fallbackErr != nil {
				utils.LogWarn("PaymentNotify: payment transaction not found for recharge: %+v", map[string]interface{}{
					"order_no": orderNo,
					"user_id":  recharge.UserID,
					"amount":   recharge.Amount,
				})
			}
		}

		var user models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, recharge.UserID).Error; err != nil {
			return err
		}
		oldBalance := user.Balance
		if err := tx.Model(&models.User{}).Where("id = ?", user.ID).
			Update("balance", gorm.Expr("balance + ?", recharge.Amount)).Error; err != nil {
			return err
		}
		if err := tx.First(&freshUser, user.ID).Error; err != nil {
			return err
		}

		rechargeID := uint(recharge.ID)
		userID := user.ID
		if err := utils.CreateBalanceLogWithDB(
			tx,
			user.ID,
			"recharge",
			recharge.Amount,
			oldBalance,
			freshUser.Balance,
			nil,
			&rechargeID,
			fmt.Sprintf("充值成功，订单号: %s", orderNo),
			"user",
			&userID,
			ipAddress,
		); err != nil {
			return err
		}

		result = recharge
		paidNow = true
		return nil
	})

	if err == nil && paidNow && result.ID > 0 && result.Status == "paid" {
		sendRechargePaidNotifications(db, &result, &freshUser, paymentType)
	}

	return &result, err
}

func sendRechargePaidNotifications(db *gorm.DB, recharge *models.RechargeRecord, user *models.User, paymentType string) {
	if recharge == nil || recharge.ID == 0 || user == nil || user.ID == 0 {
		return
	}

	paymentMethod := paymentType
	if recharge.PaymentMethod.Valid && recharge.PaymentMethod.String != "" {
		paymentMethod = recharge.PaymentMethod.String
	}
	if paymentMethod == "" {
		paymentMethod = "在线支付"
	}
	paymentTime := utils.FormatBeijingTime(utils.GetBeijingTime())
	if recharge.PaidAt.Valid {
		paymentTime = utils.FormatBeijingTime(recharge.PaidAt.Time)
	}

	data := map[string]interface{}{
		"order_no":       recharge.OrderNo,
		"username":       user.Username,
		"email":          user.Email,
		"amount":         recharge.Amount,
		"balance":        user.Balance,
		"payment_method": paymentMethod,
		"payment_time":   paymentTime,
	}

	emailService := email.NewEmailService()
	templateBuilder := email.NewEmailTemplateBuilder()
	if notification.ShouldSendCustomerNotificationToUser(user, "recharge_paid", notification.ChannelEmail) {
		content := templateBuilder.GetAdminNotificationTemplate("recharge_paid", "充值到账通知", "您的充值已到账。", data)
		if err := emailService.QueueEmail(user.Email, "充值到账通知", content, "recharge_success"); err != nil {
			utils.LogErrorMsg("sendRechargePaidNotifications: 充值到账邮件失败: order_no=%s, email=%s, error=%v", recharge.OrderNo, user.Email, err)
		}
	}

	notificationService := notification.NewNotificationService()
	if notification.ShouldSendCustomerNotificationToUser(user, "recharge_paid", notification.ChannelSystem) {
		content := fmt.Sprintf("您的充值 ¥%.2f 已到账，当前余额 ¥%.2f。", recharge.Amount, user.Balance)
		if err := notificationService.CreateUserSystemNotification(user, "recharge_paid", "充值到账", content); err != nil {
			utils.LogErrorMsg("sendRechargePaidNotifications: 充值站内通知失败: order_no=%s, user_id=%d, error=%v", recharge.OrderNo, user.ID, err)
		}
	}

	if err := notificationService.SendAdminNotification("recharge_paid", data); err != nil {
		utils.LogErrorMsg("sendRechargePaidNotifications: 管理员充值通知失败: order_no=%s, error=%v", recharge.OrderNo, err)
	}

	if db != nil {
		utils.LogInfo("sendRechargePaidNotifications: 充值通知处理完成: order_no=%s", recharge.OrderNo)
	}
}

func PaymentNotify(c *gin.Context) {
	paymentType := c.Param("type")

	// 立即记录回调到达（用于诊断）
	utils.LogInfo("========== PaymentNotify 函数被调用 ==========")
	utils.LogInfo("PaymentNotify: 回调到达 - payment_type=%s, method=%s, remote_addr=%s, url=%s",
		paymentType, c.Request.Method, c.ClientIP(), c.Request.URL.String())

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
		} else if strings.Contains(contentType, "xml") {
			if body, err := io.ReadAll(c.Request.Body); err == nil && len(body) > 0 {
				if xmlParams, parseErr := parseXMLPaymentParams(body); parseErr == nil {
					for k, v := range xmlParams {
						params[k] = v
					}
				} else {
					utils.LogError("PaymentNotify: XML参数解析失败", parseErr, map[string]interface{}{
						"payment_type": paymentType,
					})
				}
			}
		}
	}

	// 处理 GET 请求参数（包括同步回调以及附带在URL上的参数）
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			paramValue := v[0]
			// 如果参数值包含逗号，说明可能是重复参数，取第一个
			if strings.Contains(paramValue, ",") {
				paramValue = strings.Split(paramValue, ",")[0]
				utils.LogWarn("PaymentNotify: 检测到参数 %s 包含逗号，已修正为: %s", k, paramValue)
			}
			// 如果参数已存在且值不同，取第一个（避免重复参数，优先使用POST Body中的参数）
			if existingVal, exists := params[k]; exists && existingVal != paramValue {
				utils.LogWarn("PaymentNotify: 检测到重复参数 %s，原值=%s，新值=%s，使用原值", k, existingVal, paramValue)
			} else if !exists {
				params[k] = paramValue
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

	paymentConfig, err := utils.FindEnabledPaymentConfig(db, paymentType)
	if err != nil {
		utils.LogError("PaymentNotify: payment config not found", err, map[string]interface{}{
			"payment_type": paymentType,
		})
		c.String(http.StatusBadRequest, "支付配置不存在")
		return
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
	case "codepay", "codepay_alipay", "codepay_wxpay":
		codepayService, err := payment.NewCodepayService(&paymentConfig)
		if err != nil {
			utils.LogError("PaymentNotify: 创建码支付服务失败", err, map[string]interface{}{
				"payment_type": paymentType,
			})
		} else {
			verified = codepayService.VerifyNotify(params)
			utils.LogInfo("PaymentNotify: 码支付签名验证结果 - verified=%v, payment_type=%s, order_no=%s", verified, paymentType, params["out_trade_no"])
		}
	}

	if !verified {
		utils.LogError("PaymentNotify: signature verification failed", nil, map[string]interface{}{
			"payment_type": paymentType,
			"order_no":     params["out_trade_no"],
			"params_count": len(params),
		})
		utils.CreateBusinessLog(c, "payment_callback_signature_failed", "支付回调签名验证失败（疑似伪造或篡改）", "error", map[string]interface{}{
			"payment_type": paymentType,
			"order_no":     params["out_trade_no"],
		})
		paymentCallbackAck(c, paymentType, false, "签名验证失败")
		return
	}

	orderNo := params["out_trade_no"]
	if orderNo == "" {
		orderNo = params["order_id"] // 兼容部分码支付平台
	}
	if orderNo == "" {
		orderNo = params["out_trade_id"]
	}

	// 处理订单号参数重复的情况（易支付平台可能会在 return_url 中自动添加参数）
	if orderNo != "" && strings.Contains(orderNo, ",") {
		// 如果订单号包含逗号，说明参数重复了，取第一个
		orderNo = strings.Split(orderNo, ",")[0]
		utils.LogWarn("PaymentNotify: 检测到重复的订单号参数，已修正为: %s", orderNo)
	}
	externalTransactionID := params["trade_no"]
	if externalTransactionID == "" {
		externalTransactionID = params["pay_no"] // 兼容部分码支付平台
	}
	if externalTransactionID == "" {
		externalTransactionID = params["transaction_id"] // 微信支付交易号
	}
	// 处理交易号参数重复的情况
	if externalTransactionID != "" && strings.Contains(externalTransactionID, ",") {
		externalTransactionID = strings.Split(externalTransactionID, ",")[0]
		utils.LogWarn("PaymentNotify: 检测到重复的交易号参数，已修正为: %s", externalTransactionID)
	}

	if paymentType == "yipay" || strings.HasPrefix(paymentType, "yipay_") ||
		paymentType == "codepay" || strings.HasPrefix(paymentType, "codepay_") {
		tradeStatus := params["trade_status"]
		if tradeStatus == "" {
			tradeStatus = params["status"]
		}

		if tradeStatus == "TRADE_FREEZE" {
			utils.LogWarn("PaymentNotify: yipay trade frozen: %+v", map[string]interface{}{
				"payment_type": paymentType,
				"order_no":     orderNo,
				"trade_status": tradeStatus,
			})
			c.String(http.StatusOK, "success")
			return
		}

		if tradeStatus == "TRADE_UNFREEZE" {
			utils.LogInfo("PaymentNotify: yipay trade unfrozen: %+v", map[string]interface{}{
				"payment_type": paymentType,
				"order_no":     orderNo,
				"trade_status": tradeStatus,
			})
			c.String(http.StatusOK, "success")
			return
		}

		// 兼容不同平台的成功状态标识：TRADE_SUCCESS, SUCCESS, 1, 2, 200，或者如果平台不传状态字段也视为成功（依赖签名）
		if tradeStatus != "TRADE_SUCCESS" && tradeStatus != "SUCCESS" && tradeStatus != "1" && tradeStatus != "2" && tradeStatus != "200" && tradeStatus != "" {
			utils.LogWarn("PaymentNotify: yipay trade status not success: %+v", map[string]interface{}{
				"payment_type": paymentType,
				"order_no":     orderNo,
				"trade_status": tradeStatus,
			})
			c.String(http.StatusOK, "success")
			return
		}

		utils.LogInfo("PaymentNotify: 支付订单状态为成功，继续处理 - order_no=%s, trade_status=%s", orderNo, tradeStatus)
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

	if paymentType == "wechat" {
		returnCode := params["return_code"]
		resultCode := params["result_code"]
		if (returnCode != "" && returnCode != "SUCCESS") || (resultCode != "" && resultCode != "SUCCESS") {
			utils.LogWarn("PaymentNotify: wechat trade status not success: %+v", map[string]interface{}{
				"payment_type": paymentType,
				"order_no":     orderNo,
				"return_code":  returnCode,
				"result_code":  resultCode,
			})
			paymentCallbackAck(c, paymentType, true, "OK")
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
			utils.CreateBusinessLog(c, "payment_callback_order_not_found", "支付回调订单或充值记录不存在", "error", map[string]interface{}{
				"order_no": orderNo, "payment_type": paymentType,
			})
			paymentCallbackAck(c, paymentType, false, "订单或充值记录不存在")
			return
		}
	}

	if isRecharge {
		processedRecharge, err := processPaidRecharge(db, orderNo, paymentType, paymentConfig.ID, externalTransactionID, params, utils.GetRealClientIP(c))
		if err != nil {
			utils.LogError("PaymentNotify: failed to process recharge transaction", err, map[string]interface{}{
				"order_no": orderNo,
			})
			utils.CreateBusinessLog(c, "payment_callback_process_failed", "支付回调处理失败（充值入账失败）", "error", map[string]interface{}{
				"order_no": orderNo, "payment_type": paymentType, "reason": err.Error(),
			})
			paymentCallbackAck(c, paymentType, false, "处理失败")
			return
		}

		utils.LogInfo("PaymentNotify: 充值回调处理成功 - order_no=%s, user_id=%d, amount=%.2f, payment_type=%s",
			orderNo, processedRecharge.UserID, processedRecharge.Amount, paymentType)

		utils.CreateBusinessLog(c, "payment_callback_success", "支付回调处理成功（充值）", "info", map[string]interface{}{
			"order_no":     orderNo,
			"amount":       processedRecharge.Amount,
			"payment_type": paymentType,
		})

		paymentCallbackAck(c, paymentType, true, "OK")
		return
	}

	if callbackAmount, amountVerified := parseCallbackAmount(paymentType, params); amountVerified {
		expectedAmount := order.Amount
		if order.FinalAmount.Valid {
			expectedAmount = order.FinalAmount.Float64
		}

		if !amountMatches(expectedAmount, callbackAmount) {
			utils.LogError("PaymentNotify: amount mismatch", nil, map[string]interface{}{
				"order_no":        orderNo,
				"payment_type":    paymentType,
				"expected_amount": expectedAmount,
				"callback_amount": callbackAmount,
			})
			utils.CreateBusinessLog(c, "payment_callback_amount_mismatch", "支付回调订单金额与实付不一致", "error", map[string]interface{}{
				"order_no": orderNo, "payment_type": paymentType, "expected": expectedAmount, "callback_amount": callbackAmount,
			})
			c.String(http.StatusBadRequest, "订单金额不匹配")
			return
		}
	} else {
		utils.LogWarn("PaymentNotify: unable to verify order amount for payment type: %+v", map[string]interface{}{
			"order_no":     orderNo,
			"payment_type": paymentType,
		})
	}

	callbackData := ""
	if data, err := json.Marshal(params); err == nil {
		callbackData = string(data)
	}
	orderService := orderServicePkg.NewOrderService()
	_, err = orderService.FinalizePaidOrder(orderNo, orderServicePkg.FinalizePaidOrderOptions{
		PaymentMethodName:     paymentType,
		PaymentMethodID:       paymentConfig.ID,
		ExternalTransactionID: externalTransactionID,
		CallbackData:          callbackData,
		IPAddress:             utils.GetRealClientIP(c),
	})
	if err != nil {
		utils.LogError("PaymentNotify: failed to finalize paid order", err, map[string]interface{}{
			"order_no": orderNo,
		})
		utils.CreateBusinessLog(c, "payment_callback_process_failed", "支付回调处理失败（订单履约失败）", "error", map[string]interface{}{
			"order_no": orderNo, "payment_type": paymentType, "reason": err.Error(),
		})
		paymentCallbackAck(c, paymentType, false, "处理失败")
		return
	}

	utils.LogInfo("PaymentNotify: 订单状态已更新为paid - order_no=%s, order_id=%d, status=%s", orderNo, order.ID, order.Status)

	sendPaymentNotifications(db, orderNo)

	utils.CreateBusinessLog(c, "payment_callback_success", "支付回调处理成功（订单）", "info", map[string]interface{}{
		"order_no":     orderNo,
		"order_id":     order.ID,
		"payment_type": paymentType,
	})

	paymentCallbackAck(c, paymentType, true, "OK")
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

	paymentTime := utils.FormatBeijingTime(utils.GetBeijingTime())
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

	if notification.ShouldSendCustomerNotificationToUser(&latestUser, "order_paid", notification.ChannelEmail) {
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
					expireTime = utils.FormatBeijingTime(subscriptionInfo.ExpireTime)
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
	if notification.ShouldSendCustomerNotificationToUser(&latestUser, "order_paid", notification.ChannelSystem) {
		content := fmt.Sprintf("您的订单 %s 已支付成功，金额 ¥%.2f。", latestOrder.OrderNo, paidAmount)
		if err := notificationService.CreateUserSystemNotification(&latestUser, "order_paid", "支付成功", content); err != nil {
			utils.LogErrorMsg("sendPaymentNotifications: 创建站内支付通知失败: order_no=%s, error=%v", latestOrder.OrderNo, err)
		}
	}

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

	if transaction.Status == "pending" && shouldQueryPaymentStatus(transaction.CreatedAt) {
		orderNo := ""
		isRecharge := false
		if transaction.OrderID > 0 {
			var order models.Order
			if err := db.Where("id = ? AND user_id = ?", transaction.OrderID, user.ID).First(&order).Error; err == nil {
				orderNo = order.OrderNo
			}
		} else if transaction.TransactionID.Valid && transaction.TransactionID.String != "" {
			orderNo = transaction.TransactionID.String
			isRecharge = true
		}
		if orderNo != "" {
			if success, err := performPaymentStatusQuery(db, orderNo, isRecharge); success {
				db.Where("id = ? AND user_id = ?", transactionID, user.ID).First(&transaction)
			} else if err != nil {
				utils.LogWarn("GetPaymentStatus: active payment query failed: %+v", map[string]interface{}{
					"transaction_id": transactionID,
					"order_no":       orderNo,
					"error":          err.Error(),
				})
			}
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"status":   transaction.Status,
		"amount":   float64(transaction.Amount) / 100,
		"order_id": transaction.OrderID,
	})
}

package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	orderServicePkg "cboard-go/internal/services/order"
	"cboard-go/internal/services/payment"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func resolveUserInfo(user models.User) gin.H {
	if user.ID > 0 {
		return gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		}
	}
	return gin.H{
		"id":       0,
		"username": "已删除",
		"email":    "deleted",
	}
}

func resolvePackageInfo(order models.Order) (uint, string, interface{}) {
	if order.PackageID == 0 {
		name := "设备升级"
		itemType := "device_upgrade"
		if order.ExtraData.Valid && order.ExtraData.String != "" {
			var extraData map[string]interface{}
			if err := json.Unmarshal([]byte(order.ExtraData.String), &extraData); err == nil {
				if orderType, ok := extraData["type"].(string); ok && orderType == "custom_package" {
					devices := 0
					months := 0
					if devicesVal, ok := extraData["devices"].(float64); ok {
						devices = int(devicesVal)
					}
					if monthsVal, ok := extraData["months"].(float64); ok {
						months = int(monthsVal)
					}
					name = fmt.Sprintf("自定义套餐 (%d设备/%d月)", devices, months)
					itemType = "custom_package"
					return 0, name, gin.H{"id": 0, "name": name, "type": itemType}
				}
			}
			name = "设备升级订单"
		}
		return 0, name, gin.H{"id": 0, "name": name, "type": itemType}
	}

	if order.Package.ID > 0 && order.Package.ID == order.PackageID {
		return order.PackageID, order.Package.Name, gin.H{
			"id":           order.Package.ID,
			"name":         order.Package.Name,
			"price":        order.Package.Price,
			"device_limit": order.Package.DeviceLimit,
		}
	}

	// 如果Package未预加载，返回基本 info
	return order.PackageID, "套餐未加载", gin.H{"id": order.PackageID, "name": "套餐未加载"}
}

func formatOrderData(order models.Order) gin.H {
	amount := order.Amount
	if order.FinalAmount.Valid {
		amount = order.FinalAmount.Float64
	}

	var balanceUsed float64
	if order.ExtraData.Valid && order.ExtraData.String != "" {
		var extraData map[string]interface{}
		if err := json.Unmarshal([]byte(order.ExtraData.String), &extraData); err == nil {
			if balanceUsedVal, ok := extraData["balance_used"].(float64); ok {
				balanceUsed = balanceUsedVal
			}
		}
	}
	if balanceUsed > 0 {
		amount = utils.RoundFloat(amount+balanceUsed, 2)
	}

	paymentMethod := ""
	if order.PaymentMethodName.Valid {
		paymentMethod = order.PaymentMethodName.String
	}

	paymentTime := ""
	if order.PaymentTime.Valid {
		paymentTime = utils.FormatBeijingTime(order.PaymentTime.Time)
	}

	userInfo := resolveUserInfo(order.User)
	pkgID, pkgName, pkgInfo := resolvePackageInfo(order)

	return gin.H{
		"id":                     order.ID,
		"order_no":               order.OrderNo,
		"user_id":                order.UserID,
		"user":                   userInfo,
		"package_id":             pkgID,
		"package_name":           pkgName,
		"package":                pkgInfo,
		"amount":                 amount,
		"final_amount":           utils.GetNullFloat64Value(order.FinalAmount),
		"discount_amount":        utils.GetNullFloat64Value(order.DiscountAmount),
		"payment_method":         paymentMethod,
		"payment_method_id":      utils.GetNullInt64Value(order.PaymentMethodID),
		"payment_time":           paymentTime,
		"payment_transaction_id": utils.GetNullStringValue(order.PaymentTransactionID),
		"status":                 order.Status,
		"created_at":             utils.FormatBeijingTime(order.CreatedAt),
		"updated_at":             utils.FormatBeijingTime(order.UpdatedAt),
		"expire_time":            utils.GetNullTimeValue(order.ExpireTime),
		"coupon_id":              utils.GetNullInt64Value(order.CouponID),
	}
}

func extractYipayPaymentType(payType string, orderPaymentMethodName string) string {
	if strings.HasPrefix(payType, "yipay_") {
		return strings.TrimPrefix(payType, "yipay_")
	}

	if orderPaymentMethodName != "" {
		if strings.Contains(orderPaymentMethodName, "yipay_wxpay") || strings.Contains(orderPaymentMethodName, "易支付-微信") {
			return "wxpay"
		}
		if strings.Contains(orderPaymentMethodName, "yipay_alipay") || strings.Contains(orderPaymentMethodName, "易支付-支付宝") {
			return "alipay"
		}
		if strings.Contains(orderPaymentMethodName, "yipay_qqpay") || strings.Contains(orderPaymentMethodName, "易支付-QQ") {
			return "qqpay"
		}
		if strings.HasPrefix(orderPaymentMethodName, "yipay_") {
			parts := strings.Split(orderPaymentMethodName, "yipay_")
			if len(parts) > 1 {
				for _, part := range parts {
					if strings.HasPrefix(part, "wxpay") {
						return "wxpay"
					} else if strings.HasPrefix(part, "alipay") {
						return "alipay"
					} else if strings.HasPrefix(part, "qqpay") {
						return "qqpay"
					}
				}
			}
		}
	}
	return "alipay"
}

func extractCodepayPaymentType(payType string, reqMethod string) string {
	if reqMethod != "" && strings.HasPrefix(reqMethod, "codepay_") {
		return strings.TrimPrefix(reqMethod, "codepay_")
	}
	if strings.HasPrefix(payType, "codepay_") {
		return strings.TrimPrefix(payType, "codepay_")
	}
	return "alipay"
}

func generatePaymentURL(db *gorm.DB, order *models.Order, paymentConfig *models.PaymentConfig, reqMethod string, amount float64) (string, error) {
	switch paymentConfig.PayType {
	case "alipay":
		service, err := payment.NewAlipayService(paymentConfig)
		if err != nil {
			return "", err
		}
		return service.CreatePayment(order, amount)
	case "wechat":
		service, err := payment.NewWechatService(paymentConfig)
		if err != nil {
			return "", err
		}
		return service.CreatePayment(order, amount)
	default:
		if paymentConfig.PayType == "yipay" || strings.HasPrefix(paymentConfig.PayType, "yipay_") {
			service, err := payment.NewYipayService(paymentConfig)
			if err != nil {
				return "", err
			}
			pType := extractYipayPaymentType(paymentConfig.PayType, "")
			if reqMethod != "" && strings.HasPrefix(reqMethod, "yipay_") {
				pType = strings.TrimPrefix(reqMethod, "yipay_")
			} else if order.PaymentMethodName.Valid {
				pType = extractYipayPaymentType(paymentConfig.PayType, order.PaymentMethodName.String)
			}
			return service.CreatePayment(order, amount, pType)
		}
		if paymentConfig.PayType == "codepay" || strings.HasPrefix(paymentConfig.PayType, "codepay_") {
			service, err := payment.NewCodepayService(paymentConfig)
			if err != nil {
				return "", err
			}
			pType := extractCodepayPaymentType(paymentConfig.PayType, reqMethod)
			return service.CreatePayment(order, amount, pType)
		}
		return "", fmt.Errorf("不支持的支付方式: %s", paymentConfig.PayType)
	}
}

func shouldQueryPaymentStatus(createdAt time.Time) bool {
	timeSince := time.Since(createdAt)
	if timeSince >= 3*time.Second && timeSince < 10*time.Second {
		return true
	} else if timeSince >= 10*time.Second && timeSince < 60*time.Second {
		return int(timeSince.Seconds())%5 < 2
	} else if timeSince >= 60*time.Second {
		return int(timeSince.Seconds())%30 < 2
	}
	return false
}

func performAlipayQuery(db *gorm.DB, orderNo string, isRecharge bool) (bool, *payment.AlipayQueryResult, error) {
	var configModel models.PaymentConfig
	if err := db.Where("LOWER(pay_type) = 'alipay' AND status = 1").First(&configModel).Error; err != nil {
		return false, nil, err
	}

	service, err := payment.NewAlipayService(&configModel)
	if err != nil {
		return false, nil, err
	}

	result, err := service.QueryOrder(orderNo)
	if err != nil || result == nil || !result.IsPaid() {
		return false, result, err
	}

	// 验证订单号匹配（防止订单号碰撞）
	if result.OutTradeNo != orderNo {
		utils.LogError("performAlipayQuery: order number mismatch", nil, map[string]interface{}{
			"expected_order_no": orderNo,
			"alipay_order_no":   result.OutTradeNo,
			"trade_no":          result.TradeNo,
		})
		return false, result, fmt.Errorf("订单号不匹配")
	}

	// 解析支付宝返回的金额
	var alipayAmount float64
	if result.TotalAmount != "" {
		_, _ = fmt.Sscanf(result.TotalAmount, "%f", &alipayAmount) // Ignore error, use default value
	}

	err = utils.WithTransaction(db, func(tx *gorm.DB) error {
		if isRecharge {
			var record models.RechargeRecord
			if err := tx.Where("order_no = ? AND status = ?", orderNo, "pending").First(&record).Error; err != nil {
				return err
			}

			// 验证充值金额（防止金额篡改）
			if alipayAmount > 0 {
				if alipayAmount < record.Amount-0.01 || alipayAmount > record.Amount+0.01 {
					utils.LogError("performAlipayQuery: recharge amount mismatch", nil, map[string]interface{}{
						"order_no":        orderNo,
						"expected_amount": record.Amount,
						"alipay_amount":   alipayAmount,
						"trade_no":        result.TradeNo,
					})
					return fmt.Errorf("充值金额不匹配: 预期%.2f元, 支付宝返回%.2f元", record.Amount, alipayAmount)
				}
			}

			record.Status = "paid"
			record.PaidAt = database.NullTime(utils.GetBeijingTime())
			if result.TradeNo != "" {
				record.PaymentTransactionID = database.NullString(result.TradeNo)
			}
			if err := tx.Save(&record).Error; err != nil {
				return err
			}
			var user models.User
			if err := tx.First(&user, record.UserID).Error; err == nil {
				user.Balance += record.Amount
				return tx.Save(&user).Error
			}
			return nil
		}

		var order models.Order
		if err := tx.Where("order_no = ? AND status = ?", orderNo, "pending").First(&order).Error; err != nil {
			return err
		}

		// 验证订单金额（防止金额篡改）
		if alipayAmount > 0 {
			expectedAmount := order.Amount
			if order.FinalAmount.Valid {
				expectedAmount = order.FinalAmount.Float64
			}

			// 检查是否使用了余额支付
			var balanceUsed float64 = 0
			if order.ExtraData.Valid && order.ExtraData.String != "" {
				var extraData map[string]interface{}
				if err := json.Unmarshal([]byte(order.ExtraData.String), &extraData); err == nil {
					if balanceUsedVal, ok := extraData["balance_used"].(float64); ok {
						balanceUsed = balanceUsedVal
					}
				}
			}

			// FinalAmount 仅表示第三方应付金额
			expectedPaymentAmount := expectedAmount

			if alipayAmount < expectedPaymentAmount-0.01 || alipayAmount > expectedPaymentAmount+0.01 {
				utils.LogError("performAlipayQuery: order amount mismatch", nil, map[string]interface{}{
					"order_no":         orderNo,
					"expected_amount":  expectedAmount,
					"balance_used":     balanceUsed,
					"expected_payment": expectedPaymentAmount,
					"alipay_amount":    alipayAmount,
					"trade_no":         result.TradeNo,
				})
				return fmt.Errorf("订单金额不匹配: 预期支付%.2f元, 支付宝返回%.2f元", expectedPaymentAmount, alipayAmount)
			}
		}

		order.Status = "paid"
		order.PaymentTime = database.NullTime(utils.GetBeijingTime())
		if err := tx.Save(&order).Error; err != nil {
			return err
		}

		var transaction models.PaymentTransaction
		if err := tx.Where("order_id = ?", order.ID).First(&transaction).Error; err == nil {
			transaction.Status = "success"
			transaction.ExternalTransactionID = database.NullString(result.TradeNo)
			tx.Save(&transaction)
		}
		return nil
	})

	return err == nil, result, err
}

type CreateOrderRequest struct {
	PackageID      uint    `json:"package_id" binding:"required"`
	CouponCode     string  `json:"coupon_code"`
	PaymentMethod  string  `json:"payment_method"`
	Amount         float64 `json:"amount"`
	UseBalance     bool    `json:"use_balance"`
	BalanceAmount  float64 `json:"balance_amount"`
	Currency       string  `json:"currency"`
	DurationMonths int     `json:"duration_months"`
}

func CreateOrder(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	var req orderServicePkg.CreateOrderParams
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	req.UserAgent = c.GetHeader("User-Agent")
	req.ClientIP = utils.GetRealClientIP(c)

	svc := orderServicePkg.NewOrderService()
	order, paymentURL, err := svc.CreateOrder(user.ID, req)
	if err != nil {
		if strings.Contains(err.Error(), "生成支付链接失败") {
			utils.CreateBusinessLog(c, "order_payment_url_failed", "创建订单生成支付链接失败", "error", map[string]interface{}{
				"user_id": user.ID, "order_no": order.OrderNo, "reason": err.Error(),
			})
		}
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	data := gin.H{
		"order_no":            order.OrderNo,
		"id":                  order.ID,
		"user_id":             order.UserID,
		"package_id":          order.PackageID,
		"amount":              order.Amount,
		"final_amount":        utils.GetNullFloat64Value(order.FinalAmount),
		"discount_amount":     utils.GetNullFloat64Value(order.DiscountAmount),
		"status":              order.Status,
		"payment_method":      utils.GetNullStringValue(order.PaymentMethodName),
		"payment_method_name": utils.GetNullStringValue(order.PaymentMethodName),
		"created_at":          utils.FormatBeijingTime(order.CreatedAt),
	}

	if order.PaymentMethodName.Valid {
		data["payment_method_name"] = order.PaymentMethodName.String
	}
	if paymentURL != "" {
		data["payment_url"] = paymentURL
		data["payment_qr_code"] = paymentURL
	}
	if order.Status == "paid" {
		data["message"] = "订单已支付成功"
	}
	if order.CouponID.Valid {
		data["coupon_id"] = order.CouponID.Int64
	}

	utils.SuccessResponse(c, http.StatusCreated, "订单创建成功", data)
}

func GetOrders(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	page, size, _ := getPagination(c)
	status := c.Query("status")
	paymentMethod := c.Query("payment_method")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	isAdmin, exists := c.Get("is_admin")
	admin := exists && isAdmin.(bool)

	db := database.GetDB()
	var orders []models.Order
	var total int64

	query := db.Model(&models.Order{}).Preload("User").Preload("Package").Preload("Coupon")

	if !admin {
		query = query.Where("user_id = ?", user.ID)
	}

	if status != "" && status != "all" {
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
			query = query.Where("status = ?", status)
		}
	}

	if paymentMethod != "" && paymentMethod != "all" {
		query = query.Where("payment_method_name = ?", paymentMethod)
	}
	var startParsed time.Time
	if startDate != "" {
		parsed, err := time.ParseInLocation("2006-01-02", startDate, utils.BeijingTZ)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "start_date格式错误，应为YYYY-MM-DD", err)
			return
		}
		startParsed = parsed
		query = query.Where("created_at >= ?", startParsed)
	}
	if endDate != "" {
		parsed, err := time.ParseInLocation("2006-01-02", endDate, utils.BeijingTZ)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "end_date格式错误，应为YYYY-MM-DD", err)
			return
		}
		if !startParsed.IsZero() && parsed.Before(startParsed) {
			utils.ErrorResponse(c, http.StatusBadRequest, "end_date不能早于start_date", nil)
			return
		}
		endTime := parsed.Add(24 * time.Hour)
		query = query.Where("created_at < ?", endTime)
	}

	query.Count(&total)

	offset := (page - 1) * size
	if err := query.Order("created_at DESC").Offset(offset).Limit(size).Find(&orders).Error; err != nil {
		utils.LogError("GetOrders: query orders", err, nil)
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取订单列表失败，请稍后重试", err)
		return
	}

	pages := (int(total) + size - 1) / size
	formattedOrders := make([]gin.H, len(orders))
	for i, order := range orders {
		formattedOrders[i] = formatOrderData(order)
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"orders": formattedOrders,
		"total":  total,
		"page":   page,
		"size":   size,
		"pages":  pages,
	})
}

func GetOrder(c *gin.Context) {
	id := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
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
			utils.ErrorResponse(c, http.StatusNotFound, "订单不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订单失败", err)
		}
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", order)
}

func CancelOrder(c *gin.Context) {
	id := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	var order models.Order
	if err := db.Where("id = ? AND user_id = ?", id, user.ID).First(&order).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "订单不存在", nil)
		return
	}

	if order.Status != "pending" {
		utils.ErrorResponse(c, http.StatusBadRequest, "订单状态不允许取消", nil)
		return
	}

	order.Status = "cancelled"
	if err := db.Save(&order).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "取消订单失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "订单已取消", order)
}

func CancelOrderByNo(c *gin.Context) {
	orderNo := c.Param("orderNo")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	var order models.Order
	if err := db.Where("order_no = ? AND user_id = ?", orderNo, user.ID).First(&order).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "订单不存在", err)
		return
	}

	if order.Status != "pending" {
		utils.ErrorResponse(c, http.StatusBadRequest, "订单状态不允许取消", nil)
		return
	}

	order.Status = "cancelled"
	if err := db.Save(&order).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "取消订单失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "订单已取消", order)
}

func GetAdminOrders(c *gin.Context) {
	db := database.GetDB()
	includeRecharges := c.Query("include_recharges") == "true"
	page, size, _ := getPagination(c)
	keyword := c.Query("keyword")
	if keyword == "" {
		keyword = c.Query("search")
	}
	status := c.Query("status")

	if includeRecharges {
		var orderCount, rechargeCount int64
		orderQuery := db.Model(&models.Order{}).Joins("LEFT JOIN users ON orders.user_id = users.id")
		rechargeQuery := db.Model(&models.RechargeRecord{}).Joins("LEFT JOIN users ON recharge_records.user_id = users.id")

		if keyword != "" {
			sanitizedKeyword := utils.SanitizeSearchKeyword(keyword)
			if sanitizedKeyword != "" {
				orderClause := "orders.order_no LIKE ? OR orders.order_no LIKE ? OR users.username LIKE ? OR users.email LIKE ?"
				rechargeClause := "recharge_records.order_no LIKE ? OR recharge_records.order_no LIKE ? OR users.username LIKE ? OR users.email LIKE ?"
				params := []interface{}{"%" + sanitizedKeyword + "%", "%ORD%" + sanitizedKeyword + "%", "%" + sanitizedKeyword + "%", "%" + sanitizedKeyword + "%"}
				rParams := []interface{}{"%" + sanitizedKeyword + "%", "%RCH%" + sanitizedKeyword + "%", "%" + sanitizedKeyword + "%", "%" + sanitizedKeyword + "%"}

				orderQuery = orderQuery.Where(orderClause, params...)
				rechargeQuery = rechargeQuery.Where(rechargeClause, rParams...)
			}
		}

		if status != "" && status != "all" {
			orderQuery = orderQuery.Where("orders.status = ?", status)
			rechargeQuery = rechargeQuery.Where("recharge_records.status = ?", status)
		}

		orderQuery.Count(&orderCount)
		rechargeQuery.Count(&rechargeCount)
		total := orderCount + rechargeCount

		// 只取当前页需要的最大数据量（page*size），而非固定500
		limit := page * size
		if limit > 200 {
			limit = 200
		}

		type mergedRecord struct {
			recordType string
			createdAt  time.Time
			data       gin.H
		}
		allRecords := make([]mergedRecord, 0, limit*2)

		var orders []models.Order
		if err := orderQuery.Preload("User").Preload("Package").Order("orders.created_at DESC").Limit(limit).Find(&orders).Error; err == nil {
			for _, order := range orders {
				formatted := formatOrderData(order)
				allRecords = append(allRecords, mergedRecord{
					recordType: "order",
					createdAt:  order.CreatedAt,
					data:       formatted,
				})
			}
		}

		var recharges []models.RechargeRecord
		if err := rechargeQuery.Order("created_at DESC").Limit(limit).Find(&recharges).Error; err == nil {
			for _, record := range recharges {
				userInfo := resolveUserInfo(record.User)
				rechargeData := gin.H{
					"id":                     record.ID,
					"user_id":                record.UserID,
					"user":                   userInfo,
					"order_no":               record.OrderNo,
					"amount":                 record.Amount,
					"status":                 record.Status,
					"payment_method":         utils.GetNullStringValue(record.PaymentMethod),
					"payment_transaction_id": utils.GetNullStringValue(record.PaymentTransactionID),
					"paid_at":                utils.GetNullTimeValue(record.PaidAt),
					"created_at":             utils.FormatBeijingTime(record.CreatedAt),
				}
				allRecords = append(allRecords, mergedRecord{
					recordType: "recharge",
					createdAt:  record.CreatedAt,
					data:       rechargeData,
				})
			}
		}

		// 用 time.Time 直接比较，避免重复字符串解析
		sort.Slice(allRecords, func(i, j int) bool {
			return allRecords[i].createdAt.After(allRecords[j].createdAt)
		})

		offset := (page - 1) * size
		end := offset + size
		if end > len(allRecords) {
			end = len(allRecords)
		}

		mergedList := make([]gin.H, 0, size)
		if offset < len(allRecords) {
			for i := offset; i < end; i++ {
				record := allRecords[i].data
				record["record_type"] = allRecords[i].recordType
				mergedList = append(mergedList, record)
			}
		}

		utils.SuccessResponse(c, http.StatusOK, "", gin.H{
			"orders": mergedList,
			"total":  total,
			"page":   page,
			"size":   size,
		})
		return
	}

	var orders []models.Order
	query := db.Model(&models.Order{}).Joins("LEFT JOIN users ON orders.user_id = users.id")

	if keyword != "" {
		sanitizedKeyword := utils.SanitizeSearchKeyword(keyword)
		if sanitizedKeyword != "" {
			query = query.Where(
				"orders.order_no LIKE ? OR orders.order_no LIKE ? OR users.username LIKE ? OR users.email LIKE ?",
				"%"+sanitizedKeyword+"%", "%ORD%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%",
			)
		}
	}

	if status != "" && status != "all" {
		query = query.Where("orders.status = ?", status)
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * size
	if err := query.Preload("User").Preload("Package").Offset(offset).Limit(size).Order("orders.created_at DESC").Find(&orders).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取订单列表失败", err)
		return
	}

	orderList := make([]gin.H, len(orders))
	for i, order := range orders {
		orderList[i] = formatOrderData(order)
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"orders": orderList,
		"total":  total,
		"page":   page,
		"size":   size,
	})
}

func UpdateAdminOrder(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误，请检查输入格式", err)
		return
	}

	db := database.GetDB()
	var order models.Order
	if err := db.Preload("Package").Preload("User").First(&order, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订单不存在", nil)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订单失败", err)
		}
		return
	}

	oldStatus := order.Status
	order.Status = req.Status

	if oldStatus != "paid" && req.Status == "paid" {
		now := utils.GetBeijingTime()
		order.PaymentTime = database.NullTime(now)
		svc := orderServicePkg.NewOrderService()
		if _, err := svc.ProcessPaidOrder(&order); err != nil {
			utils.LogError("BulkMarkOrdersPaid: process paid order", err, map[string]interface{}{"order_id": order.ID})
			utils.ErrorResponse(c, http.StatusInternalServerError, "处理订单失败，请稍后重试", nil)
			return
		}
	}

	if err := db.Save(&order).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新订单失败", err)
		return
	}
	utils.CreateAuditLogSimple(c, "update_admin_order", "order", order.ID, fmt.Sprintf("管理员操作: 更新订单 %s 状态为 %s", order.OrderNo, order.Status))
	utils.SuccessResponse(c, http.StatusOK, "订单已更新", order)
}

func RefundAdminOrder(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var order models.Order
	if err := db.Preload("Package").Preload("User").First(&order, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订单不存在", nil)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订单失败", err)
		}
		return
	}

	// 只能退款已支付的订单
	if order.Status != "paid" {
		utils.ErrorResponse(c, http.StatusBadRequest, "只能退款已支付的订单", nil)
		return
	}

	// 检查支付方式，只有易支付订单才能退款
	paymentMethodName := ""
	if order.PaymentMethodName.Valid {
		paymentMethodName = order.PaymentMethodName.String
	}

	isYipay := strings.Contains(paymentMethodName, "易支付") || strings.Contains(paymentMethodName, "yipay")
	if !isYipay && order.PaymentMethodID.Valid {
		var paymentConfig models.PaymentConfig
		if err := db.First(&paymentConfig, order.PaymentMethodID.Int64).Error; err == nil {
			if paymentConfig.PayType == "yipay" || strings.HasPrefix(paymentConfig.PayType, "yipay_") {
				isYipay = true
			}
		}
	}

	if !isYipay {
		utils.ErrorResponse(c, http.StatusBadRequest, "只有易支付订单才能退款", nil)
		return
	}

	// 获取支付配置
	var paymentConfig models.PaymentConfig
	if order.PaymentMethodID.Valid {
		if err := db.First(&paymentConfig, order.PaymentMethodID.Int64).Error; err != nil {
			// 如果找不到支付配置，尝试查找易支付配置
			if err := db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", "yipay", 1).First(&paymentConfig).Error; err != nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "未找到易支付配置", nil)
				return
			}
		}
	} else {
		// 如果没有支付方式ID，尝试查找易支付配置
		if err := db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", "yipay", 1).First(&paymentConfig).Error; err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "未找到易支付配置", nil)
			return
		}
	}

	// 创建易支付服务
	yipayService, err := payment.NewYipayService(&paymentConfig)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "初始化易支付服务失败: "+err.Error(), nil)
		return
	}

	// 计算退款金额
	refundAmount := order.Amount
	if order.FinalAmount.Valid {
		refundAmount = order.FinalAmount.Float64
	}

	// 获取易支付交易号
	tradeNo := ""
	if order.PaymentTransactionID.Valid {
		tradeNo = order.PaymentTransactionID.String
	}

	// 调用易支付退款API
	utils.LogInfo("RefundAdminOrder: 开始退款 - order_id=%d, order_no=%s, trade_no=%s, refund_amount=%.2f", order.ID, order.OrderNo, tradeNo, refundAmount)
	if err := yipayService.RefundOrder(order.OrderNo, tradeNo, refundAmount); err != nil {
		utils.LogError("RefundAdminOrder: 易支付退款失败", err, map[string]interface{}{
			"order_id":      order.ID,
			"order_no":      order.OrderNo,
			"trade_no":      tradeNo,
			"refund_amount": refundAmount,
		})
		utils.CreateBusinessLog(c, "refund_failed", "管理员退款失败（易支付接口）", "error", map[string]interface{}{
			"order_id": order.ID, "order_no": order.OrderNo, "trade_no": tradeNo, "refund_amount": refundAmount, "reason": err.Error(),
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "退款失败: "+err.Error(), nil)
		return
	}

	// 退款成功，处理订单回退逻辑
	orderService := orderServicePkg.NewOrderService()
	if err := orderService.ProcessRefundOrder(&order); err != nil {
		utils.LogError("RefundAdminOrder: 处理退款订单失败", err, map[string]interface{}{
			"order_id": order.ID,
		})
		utils.CreateBusinessLog(c, "refund_process_failed", "管理员退款后订单回退处理失败", "error", map[string]interface{}{
			"order_id": order.ID, "order_no": order.OrderNo, "reason": err.Error(),
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "处理退款订单失败: "+err.Error(), nil)
		return
	}

	utils.LogInfo("RefundAdminOrder: 订单退款成功 - order_id=%d, order_no=%s, refund_amount=%.2f", order.ID, order.OrderNo, refundAmount)
	utils.CreateAuditLogSimple(c, "refund_admin_order", "order", order.ID, fmt.Sprintf("管理员操作: 退款订单 %s 金额 %.2f", order.OrderNo, refundAmount))
	utils.SuccessResponse(c, http.StatusOK, "订单退款成功", order)
}

func DeleteAdminOrder(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var order models.Order
	if err := db.First(&order, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订单不存在", nil)
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询订单失败", err)
		return
	}

	if err := db.Delete(&order).Error; err != nil {
		utils.LogError("DeleteOrder: delete order failed", err, map[string]interface{}{"order_id": order.ID})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除订单失败", err)
		return
	}
	utils.CreateAuditLogSimple(c, "delete_admin_order", "order", order.ID, fmt.Sprintf("管理员操作: 删除订单 %s", order.OrderNo))
	utils.SuccessResponse(c, http.StatusOK, "订单已删除", nil)
}

func GetOrderStatistics(c *gin.Context) {
	db := database.GetDB()
	var orderTotal, orderPending, orderPaid int64
	var orderRevenue float64

	db.Model(&models.Order{}).Count(&orderTotal)
	db.Model(&models.Order{}).Where("status = ?", "pending").Count(&orderPending)
	db.Model(&models.Order{}).Where("status = ?", "paid").Count(&orderPaid)
	orderRevenue = utils.CalculateTotalRevenue(db, "paid")

	var rechargeTotal, rechargePending, rechargePaid int64
	var rechargeRevenue float64
	db.Model(&models.RechargeRecord{}).Count(&rechargeTotal)
	db.Model(&models.RechargeRecord{}).Where("status = ?", "pending").Count(&rechargePending)
	db.Model(&models.RechargeRecord{}).Where("status = ?", "paid").Count(&rechargePaid)

	var paidRecharges []models.RechargeRecord
	if err := db.Model(&models.RechargeRecord{}).Where("status = ?", "paid").Find(&paidRecharges).Error; err == nil {
		for _, recharge := range paidRecharges {
			rechargeRevenue += recharge.Amount
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"total_orders":   orderTotal + rechargeTotal,
		"pending_orders": orderPending + rechargePending,
		"paid_orders":    orderPaid + rechargePaid,
		"total_revenue":  orderRevenue + rechargeRevenue,
	})
}

func BulkMarkOrdersPaid(c *gin.Context) {
	var req struct {
		OrderIDs []uint `json:"order_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	svc := orderServicePkg.NewOrderService()
	successCount := 0
	failCount := 0

	// 批量查出所有待处理订单
	var orders []models.Order
	db.Where("id IN ? AND status = ?", req.OrderIDs, "pending").Find(&orders)
	orderMap := make(map[uint]*models.Order)
	for i := range orders {
		orderMap[orders[i].ID] = &orders[i]
	}

	for _, orderID := range req.OrderIDs {
		order, ok := orderMap[orderID]
		if !ok {
			failCount++
			continue
		}
		order.PaymentTime = database.NullTime(utils.GetBeijingTime())
		if _, err := svc.ProcessPaidOrder(order); err != nil {
			utils.LogError("BulkMarkOrdersPaid: process order failed", err, map[string]interface{}{"order_id": orderID})
			failCount++
		} else {
			successCount++
			go sendPaymentNotifications(db, order.OrderNo)
		}
	}
	utils.CreateAuditLogSimple(c, "bulk_mark_orders_paid", "order", 0, fmt.Sprintf("管理员操作: 批量标记订单已支付 成功 %d 失败 %d", successCount, failCount))
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("处理完成: 成功 %d, 失败 %d", successCount, failCount), nil)
}

func BulkCancelOrders(c *gin.Context) {
	var req struct {
		OrderIDs []uint `json:"order_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}
	db := database.GetDB()
	if err := db.Model(&models.Order{}).Where("id IN ? AND status = ?", req.OrderIDs, "pending").Update("status", "cancelled").Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "批量取消失败", err)
		return
	}
	utils.CreateAuditLogSimple(c, "bulk_cancel_orders", "order", 0, fmt.Sprintf("管理员操作: 批量取消订单 %d 个", len(req.OrderIDs)))
	utils.SuccessResponse(c, http.StatusOK, "批量取消成功", nil)
}

func BatchDeleteOrders(c *gin.Context) {
	var req struct {
		OrderIDs []uint `json:"order_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}
	db := database.GetDB()
	if err := db.Delete(&models.Order{}, req.OrderIDs).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "批量删除失败", err)
		return
	}
	utils.CreateAuditLogSimple(c, "batch_delete_orders", "order", 0, fmt.Sprintf("管理员操作: 批量删除订单 %d 个", len(req.OrderIDs)))
	utils.SuccessResponse(c, http.StatusOK, "批量删除成功", nil)
}

func ExportOrders(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.Order{})
	keyword := c.Query("keyword")
	if keyword == "" {
		keyword = c.Query("search")
	}
	if keyword != "" {
		sanitizedKeyword := utils.SanitizeSearchKeyword(keyword)
		if sanitizedKeyword != "" {
			query = query.Where("order_no LIKE ? OR user_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ?)",
				"%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%")
		}
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var orders []models.Order
	if err := query.Order("created_at DESC").Find(&orders).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取订单列表失败", err)
		return
	}

	var csvContent strings.Builder
	csvContent.WriteString("\xEF\xBB\xBF")
	csvContent.WriteString("订单号,用户ID,用户名,邮箱,套餐ID,套餐名称,订单金额,支付方式,订单状态,创建时间,支付时间,更新时间\n")

	for _, order := range orders {
		formatted := formatOrderData(order)
		userMap := formatted["user"].(gin.H)
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
			userMap["username"],
			userMap["email"],
			formatted["package_id"],
			formatted["package_name"],
			formatted["amount"],
			formatted["payment_method"],
			statusText,
			utils.FormatBeijingTime(order.CreatedAt),
			formatted["payment_time"],
			utils.FormatBeijingTime(order.UpdatedAt),
		))
	}

	filename := fmt.Sprintf("orders_export_%s.csv", utils.GetBeijingTime().Format("20060102_150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", filename))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", []byte(csvContent.String()))
}

func GetOrderStats(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}
	db := database.GetDB()
	var orderTotal, orderPending, orderPaid, orderCancelled int64
	var orderPaidAmount float64

	db.Model(&models.Order{}).Where("user_id = ?", user.ID).Count(&orderTotal)
	db.Model(&models.Order{}).Where("user_id = ? AND LOWER(status) = ?", user.ID, "pending").Count(&orderPending)
	db.Model(&models.Order{}).Where("user_id = ? AND LOWER(status) = ?", user.ID, "paid").Count(&orderPaid)
	db.Model(&models.Order{}).Where("user_id = ? AND LOWER(status) = ?", user.ID, "cancelled").Count(&orderCancelled)
	orderPaidAmount = utils.CalculateUserOrderAmount(db, user.ID, "paid", true)

	var rechargeTotal, rechargePending, rechargePaid int64
	var rechargePaidAmount float64
	db.Model(&models.RechargeRecord{}).Where("user_id = ?", user.ID).Count(&rechargeTotal)
	db.Model(&models.RechargeRecord{}).Where("user_id = ? AND LOWER(status) = ?", user.ID, "pending").Count(&rechargePending)
	db.Model(&models.RechargeRecord{}).Where("user_id = ? AND LOWER(status) = ?", user.ID, "paid").Count(&rechargePaid)

	var paidRecharges []models.RechargeRecord
	if err := db.Model(&models.RechargeRecord{}).Where("user_id = ? AND LOWER(status) = ?", user.ID, "paid").Find(&paidRecharges).Error; err == nil {
		for _, recharge := range paidRecharges {
			rechargePaidAmount += recharge.Amount
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"total":       orderTotal + rechargeTotal,
		"pending":     orderPending + rechargePending,
		"paid":        orderPaid + rechargePaid,
		"cancelled":   orderCancelled,
		"totalAmount": orderPaidAmount + rechargePaidAmount,
		"paidAmount":  orderPaidAmount + rechargePaidAmount,
	})
}

func GetOrderStatusByNo(c *gin.Context) {
	orderNo := c.Param("orderNo")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}
	db := database.GetDB()

	if strings.HasPrefix(orderNo, "RCH") {
		var recharge models.RechargeRecord
		if err := db.Where("order_no = ? AND user_id = ?", orderNo, user.ID).First(&recharge).Error; err != nil {
			utils.ErrorResponse(c, http.StatusNotFound, "充值记录不存在", err)
			return
		}
		if recharge.Status == "pending" && shouldQueryPaymentStatus(recharge.CreatedAt) {
			success, _, _ := performAlipayQuery(db, orderNo, true)
			if success {
				db.Where("order_no = ?", orderNo).First(&recharge)
			}
		}
		utils.SuccessResponse(c, http.StatusOK, "", gin.H{
			"order_no": recharge.OrderNo,
			"status":   recharge.Status,
			"amount":   recharge.Amount,
			"type":     "recharge",
		})
		return
	}

	var order models.Order
	if err := db.Where("order_no = ? AND user_id = ?", orderNo, user.ID).First(&order).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "订单不存在", err)
		return
	}

	if order.Status == "pending" && shouldQueryPaymentStatus(order.CreatedAt) {
		success, _, _ := performAlipayQuery(db, orderNo, false)
		if success {
			go func() {
				var processedOrder models.Order
				if err := db.Preload("Package").Where("order_no = ?", orderNo).First(&processedOrder).Error; err == nil && processedOrder.Status == "paid" {
					svc := orderServicePkg.NewOrderService()
					if _, err := svc.ProcessPaidOrder(&processedOrder); err != nil {
						log.Printf("failed to process paid order: %v", err)
					}
					sendPaymentNotifications(db, orderNo)
				}
			}()
			db.Where("order_no = ?", orderNo).First(&order)
		}
	}

	orderType := "order"
	if order.PackageID == 0 {
		orderType = "device_upgrade"
		if order.ExtraData.Valid && order.ExtraData.String != "" {
			var extraData map[string]interface{}
			if err := json.Unmarshal([]byte(order.ExtraData.String), &extraData); err == nil {
				if extraType, ok := extraData["type"].(string); ok && extraType != "" {
					orderType = extraType
				}
			}
		}
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"order_no": order.OrderNo,
		"status":   order.Status,
		"amount":   order.Amount,
		"type":     orderType,
	})
}

// calcDeviceUpgradeAmount 按剩余时间与可选续期计算设备升级费用（年度价格法）
// 规则：年度价 200 对应 baseDevices 台；仅加设备时按剩余天数比例；若同时加时长则 = 原设备续期费 + 新增设备在(剩余+续期)内的费用
func calcDeviceUpgradeAmount(expireTime time.Time, currentDeviceLimit, additionalDevices, additionalDays int, yearlyPrice float64, baseDevices int) float64 {
	if yearlyPrice <= 0 || baseDevices <= 0 {
		return 0
	}
	now := utils.GetBeijingTime()
	remainingDays := expireTime.Sub(now).Hours() / 24
	if remainingDays < 0 {
		remainingDays = 0
	}

	var cost float64
	if additionalDays > 0 {
		// 原设备续期费用 = 年度价 × (续期天数/365) × (当前设备数/基准设备数)
		cost += yearlyPrice * (float64(additionalDays) / 365) * (float64(currentDeviceLimit) / float64(baseDevices))
		// 新增设备在（剩余+续期）内的费用 = 年度价 × ((剩余+续期)天数/365) × (新增设备数/基准设备数)
		cost += yearlyPrice * ((remainingDays + float64(additionalDays)) / 365) * (float64(additionalDevices) / float64(baseDevices))
	} else {
		// 仅增加设备：按剩余时间比例计费
		if remainingDays <= 0 {
			return 0 // 已过期且未续期，无法仅加设备
		}
		cost = yearlyPrice * (remainingDays / 365) * (float64(additionalDevices) / float64(baseDevices))
	}
	return cost
}

func UpgradeDevices(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	var req struct {
		AdditionalDevices int     `json:"additional_devices" binding:"required,min=1"`
		AdditionalDays    int     `json:"additional_days"`
		PaymentMethod     string  `json:"payment_method"`
		UseBalance        bool    `json:"use_balance"`
		BalanceAmount     float64 `json:"balance_amount"`
		PreviewOnly       bool    `json:"preview_only"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	var subscription models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		return
	}

	yearlyPrice := config.AppConfig.DeviceUpgradePricePerYear
	baseDevices := config.AppConfig.DeviceUpgradeBaseDevices
	if baseDevices <= 0 {
		baseDevices = 5
	}

	var totalAmount float64
	if yearlyPrice > 0 && baseDevices > 0 {
		totalAmount = calcDeviceUpgradeAmount(
			subscription.ExpireTime,
			subscription.DeviceLimit,
			req.AdditionalDevices,
			req.AdditionalDays,
			yearlyPrice,
			baseDevices,
		)
		// 已过期且未选续期时不允许仅加设备
		if totalAmount == 0 && req.AdditionalDays <= 0 {
			utils.ErrorResponse(c, http.StatusBadRequest, "订阅已过期，请同时选择增加时长后再升级设备数量", nil)
			return
		}
	}
	// 未配置年度价时回退到按月的旧逻辑
	if totalAmount == 0 {
		devicePricePerMonth := config.AppConfig.DeviceUpgradePricePerMonth
		if devicePricePerMonth <= 0 {
			devicePricePerMonth = 10.0
		}
		totalAmount = (float64(req.AdditionalDevices) * devicePricePerMonth) + (float64(req.AdditionalDays) * (devicePricePerMonth / 30.0))
	}

	var userLevel models.UserLevel
	levelDiscount := 1.0
	if user.UserLevelID.Valid {
		if err := db.First(&userLevel, user.UserLevelID.Int64).Error; err == nil && userLevel.DiscountRate > 0 && userLevel.DiscountRate < 1.0 {
			levelDiscount = userLevel.DiscountRate
			totalAmount *= userLevel.DiscountRate
		}
	}

	// 仅预览：返回费用不创建订单
	if req.PreviewOnly {
		utils.SuccessResponse(c, http.StatusOK, "", gin.H{
			"upgrade_cost":   totalAmount,
			"level_discount": levelDiscount,
			"amount":         totalAmount,
		})
		return
	}

	balanceUsed := 0.0
	finalAmount := totalAmount
	if req.UseBalance {
		availableBalance := math.Round(user.Balance*100) / 100
		requestedBalance := math.Round(req.BalanceAmount*100) / 100
		if requestedBalance <= 0 || requestedBalance > availableBalance {
			requestedBalance = availableBalance
		}
		if requestedBalance > finalAmount {
			requestedBalance = finalAmount
		}
		if requestedBalance > 0 {
			balanceUsed = requestedBalance
			finalAmount -= balanceUsed
		}
	}

	orderNo, err := utils.GenerateDeviceUpgradeOrderNo(db)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成订单号失败", err)
		return
	}
	extraData := fmt.Sprintf(`{"type":"device_upgrade","additional_devices":%d,"additional_days":%d,"balance_used":%.2f}`, req.AdditionalDevices, req.AdditionalDays, balanceUsed)

	actualPaidAmount := balanceUsed + finalAmount
	order := models.Order{
		OrderNo:           orderNo,
		UserID:            user.ID,
		PackageID:         0,
		Amount:            totalAmount,
		FinalAmount:       database.NullFloat64(finalAmount),
		DiscountAmount:    database.NullFloat64(totalAmount - actualPaidAmount),
		Status:            "pending",
		ExtraData:         database.NullString(extraData),
		PaymentMethodName: database.NullString(req.PaymentMethod),
	}

	if balanceUsed > 0 {
		if finalAmount > 0.01 {
			order.PaymentMethodName = database.NullString(fmt.Sprintf("余额支付(%.2f元)+%s", balanceUsed, req.PaymentMethod))
		} else {
			order.PaymentMethodName = database.NullString("余额支付")
		}
	}

	if err := db.Create(&order).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建订单失败", err)
		return
	}

	if finalAmount <= 0.01 {
		// --- 事务：扣余额 + 创建订单 + 标记已支付 ---
		txErr := db.Transaction(func(tx *gorm.DB) error {
			if balanceUsed > 0 {
				result := tx.Model(&models.User{}).Where("id = ? AND balance >= ?", user.ID, balanceUsed).
					Update("balance", gorm.Expr("balance - ?", balanceUsed))
				if result.Error != nil {
					return fmt.Errorf("扣除余额失败: %v", result.Error)
				}
				if result.RowsAffected == 0 {
					return fmt.Errorf("余额不足")
				}
			}
			order.Status = "paid"
			order.PaymentTime = database.NullTime(utils.GetBeijingTime())
			return tx.Save(&order).Error
		})
		if txErr != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, txErr.Error(), txErr)
			return
		}

		if balanceUsed > 0 {
			go func() {
				orderID := uint(order.ID)
				userID := user.ID
				if err := utils.CreateBalanceLog(
					user.ID,
					"consume",
					-balanceUsed,
					user.Balance,
					user.Balance-balanceUsed,
					&orderID,
					nil,
					fmt.Sprintf("订单支付扣除余额，订单号: %s", order.OrderNo),
					"user",
					&userID,
					utils.GetRealClientIP(c),
				); err != nil {
					log.Printf("failed to create balance log: %v", err)
				}
			}()
		}
		go func() {
			if _, err := orderServicePkg.NewOrderService().ProcessPaidOrder(&order); err != nil {
				log.Printf("failed to process paid order: %v", err)
			}
		}()
		db.Where("user_id = ?", user.ID).First(&subscription)
		utils.SuccessResponse(c, http.StatusOK, "设备数量升级成功", gin.H{
			"order_no":           order.OrderNo,
			"status":             "paid",
			"subscription":       subscription,
			"additional_devices": req.AdditionalDevices,
			"additional_days":    req.AdditionalDays,
		})
		return
	}

	var paymentURL string
	if finalAmount > 0.01 && req.PaymentMethod != "" && req.PaymentMethod != "balance" {
		queryPayType := req.PaymentMethod
		if queryPayType == "mixed" {
			queryPayType = "alipay"
		} else if strings.HasPrefix(queryPayType, "yipay_") {
			queryPayType = "yipay"
		} else if strings.HasPrefix(queryPayType, "codepay_") {
			queryPayType = "codepay"
		}

		var paymentConfig models.PaymentConfig
		escapedPayType := utils.EscapeLikePattern(queryPayType)
		if err := db.Where("LOWER(pay_type) LIKE ? AND status = 1", "%"+escapedPayType+"%").Order("sort_order ASC").First(&paymentConfig).Error; err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "未找到启用的支付配置", nil)
			return
		}

		transaction := models.PaymentTransaction{
			OrderID:         order.ID,
			UserID:          user.ID,
			PaymentMethodID: paymentConfig.ID,
			Amount:          int(finalAmount * 100),
			Currency:        "CNY",
			Status:          "pending",
		}
		if err := db.Create(&transaction).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "创建支付记录失败", err)
			return
		}
		paymentURL, err = generatePaymentURL(db, &order, &paymentConfig, req.PaymentMethod, finalAmount)
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "创建支付链接失败", err)
			return
		}
	}

	var extraDataMap map[string]interface{}
	_ = json.Unmarshal([]byte(order.ExtraData.String), &extraDataMap) // Ignore error, use empty if invalid
	extraDataMap["type"] = "device_upgrade"
	extraDataMap["additional_devices"] = req.AdditionalDevices
	extraDataMap["additional_days"] = req.AdditionalDays
	if balanceUsed > 0 {
		extraDataMap["balance_used"] = balanceUsed
	}
	extraDataBytes, _ := json.Marshal(extraDataMap)
	order.ExtraData = database.NullString(string(extraDataBytes))
	if err := db.Save(&order).Error; err != nil {
		log.Printf("failed to update order extra data: %v", err)
	}

		actualTotalAmount := utils.RoundFloat(balanceUsed+finalAmount, 2)
		responseData := gin.H{
			"order_no":            order.OrderNo,
			"id":                  order.ID,
			"status":              order.Status,
			"amount":              totalAmount,
			"final_amount":        finalAmount,
			"actual_total_amount": actualTotalAmount,
			"balance_used":        balanceUsed,
			"additional_devices":  req.AdditionalDevices,
			"additional_days":     req.AdditionalDays,
		}
	if paymentURL != "" {
		responseData["payment_url"] = paymentURL
		responseData["payment_qr_code"] = paymentURL
	}
	utils.SuccessResponse(c, http.StatusOK, "", responseData)
}

func PayOrder(c *gin.Context) {
	orderNo := c.Param("orderNo")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	var req struct {
		PaymentMethodID uint   `json:"payment_method_id"`
		PaymentMethod   string `json:"payment_method"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	var order models.Order
	if err := db.Where("order_no = ? AND user_id = ?", orderNo, user.ID).First(&order).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "订单不存在", nil)
		return
	}
	if order.Status != "pending" {
		utils.ErrorResponse(c, http.StatusBadRequest, "订单状态不允许支付", nil)
		return
	}

	amount := order.Amount
	if order.FinalAmount.Valid {
		amount = order.FinalAmount.Float64
	}

	// 余额支付
	if req.PaymentMethod == "balance" {
		// 检查余额是否充足
		if user.Balance < amount {
			utils.ErrorResponse(c, http.StatusBadRequest, "余额不足", nil)
			return
		}

		// 扣除余额
		tx := db.Begin()
		if err := tx.Model(user).UpdateColumn("balance", gorm.Expr("balance - ?", amount)).Error; err != nil {
			tx.Rollback()
			utils.ErrorResponse(c, http.StatusInternalServerError, "扣减余额失败", err)
			return
		}

		// 更新订单状态
		now := time.Now()
		balanceStr := "balance"
		if err := tx.Model(&order).Updates(map[string]interface{}{
			"status":              "paid",
			"payment_method_name": &balanceStr,
			"payment_time":        &now,
		}).Error; err != nil {
			tx.Rollback()
			utils.ErrorResponse(c, http.StatusInternalServerError, "更新订单状态失败", err)
			return
		}

		// 激活订阅
		orderService := orderServicePkg.NewOrderService()
		if _, err := orderService.ProcessPaidOrder(&order); err != nil {
			tx.Rollback()
			utils.ErrorResponse(c, http.StatusInternalServerError, "激活订阅失败", err)
			return
		}

		if err := tx.Commit().Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "支付事务提交失败", err)
			return
		}

		utils.SuccessResponse(c, http.StatusOK, "支付成功", gin.H{
			"status":   "paid",
			"order_no": order.OrderNo,
			"amount":   amount,
		})
		return
	}

	// 第三方支付
	if req.PaymentMethodID == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "请选择支付方式", nil)
		return
	}

	var paymentConfig models.PaymentConfig
	if err := db.First(&paymentConfig, req.PaymentMethodID).Error; err != nil || paymentConfig.Status != 1 {
		utils.ErrorResponse(c, http.StatusBadRequest, "支付方式无效", nil)
		return
	}

	transaction := models.PaymentTransaction{
		OrderID:         order.ID,
		UserID:          user.ID,
		PaymentMethodID: req.PaymentMethodID,
		Amount:          int(amount * 100),
		Currency:        "CNY",
		Status:          "pending",
	}
	if err := db.Create(&transaction).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建支付交易失败", err)
		return
	}

	paymentURL, err := generatePaymentURL(db, &order, &paymentConfig, req.PaymentMethod, amount)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建支付失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "支付订单已创建", gin.H{
		"payment_url":    paymentURL,
		"order_no":       order.OrderNo,
		"amount":         amount,
		"transaction_id": transaction.ID,
	})
}

// CreateCustomOrder 创建自定义套餐订单
func CreateCustomOrder(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	var req struct {
		Devices    int    `json:"devices" binding:"required,min=1"`
		Months     int    `json:"months" binding:"required,min=1"`
		CouponCode string `json:"coupon_code"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	// 检查自定义套餐是否启用
	db := database.GetDB()
	var enabledConfig models.SystemConfig
	if err := db.Where("`key` = ? AND category = ?", "custom_package_enabled", "custom_package").First(&enabledConfig).Error; err != nil || enabledConfig.Value != "true" {
		utils.ErrorResponse(c, http.StatusBadRequest, "自定义套餐功能未启用", nil)
		return
	}

	// 获取配置
	var configs []models.SystemConfig
	db.Where("category = ?", "custom_package").Find(&configs)
	configMap := make(map[string]string)
	for _, cfg := range configs {
		configMap[cfg.Key] = cfg.Value
	}

	pricePerDeviceYear := utils.ParseFloat(configMap["custom_package_price_per_device_year"], 40.0)
	minDevices := utils.ParseInt(configMap["custom_package_min_devices"], 5)
	maxDevices := utils.ParseInt(configMap["custom_package_max_devices"], 100)
	minMonths := utils.ParseInt(configMap["custom_package_min_months"], 6)

	// 验证参数
	if req.Devices < minDevices || req.Devices > maxDevices {
		utils.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("设备数量需在 %d ~ %d 之间", minDevices, maxDevices), nil)
		return
	}
	if req.Months < minMonths {
		utils.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("最少购买 %d 个月", minMonths), nil)
		return
	}
	if req.Months > 120 {
		utils.ErrorResponse(c, http.StatusBadRequest, "最多购买 120 个月", nil)
		return
	}

	// 计算价格
	basePrice := pricePerDeviceYear * float64(req.Devices) * (float64(req.Months) / 12.0)
	basePrice = utils.RoundFloat(basePrice, 2)

	// 解析时长折扣
	var discountTiers []struct {
		Months   int     `json:"months"`
		Discount float64 `json:"discount"`
	}
	if discountsJSON := configMap["custom_package_duration_discounts"]; discountsJSON != "" {
		_ = json.Unmarshal([]byte(discountsJSON), &discountTiers) // Ignore error, use empty if invalid
	}

	// 找到最佳折扣
	var discountPercent float64
	for _, tier := range discountTiers {
		if req.Months >= tier.Months && tier.Discount > discountPercent {
			discountPercent = tier.Discount
		}
	}

	finalPrice := basePrice * (1 - discountPercent/100)
	finalPrice = utils.RoundFloat(finalPrice, 2)

	// 应用优惠券
	var couponDiscount float64
	var couponID *int64
	if req.CouponCode != "" {
		var coupon models.Coupon
		if err := db.Where("code = ? AND status = ?", req.CouponCode, "active").First(&coupon).Error; err == nil {
			now := time.Now()
			if now.After(coupon.ValidFrom) && now.Before(coupon.ValidUntil) {
				// 检查总数量
				if coupon.TotalQuantity.Valid && coupon.UsedQuantity >= int(coupon.TotalQuantity.Int64) {
					utils.ErrorResponse(c, http.StatusBadRequest, "优惠券已被领完", nil)
					return
				}
				// 检查用户使用次数
				var usageCount int64
				db.Model(&models.CouponUsage{}).Where("coupon_id = ? AND user_id = ?", coupon.ID, user.ID).Count(&usageCount)
				if int(usageCount) >= coupon.MaxUsesPerUser {
					utils.ErrorResponse(c, http.StatusBadRequest, "您已达到该优惠券的使用上限", nil)
					return
				}

				// 计算折扣
				switch coupon.Type {
				case "discount":
					couponDiscount = utils.RoundFloat(finalPrice*coupon.DiscountValue/100, 2)
				case "fixed":
					couponDiscount = coupon.DiscountValue
				}

				if coupon.MaxDiscount.Valid && couponDiscount > coupon.MaxDiscount.Float64 {
					couponDiscount = coupon.MaxDiscount.Float64
				}
				if couponDiscount > finalPrice {
					couponDiscount = finalPrice
				}
				couponDiscount = utils.RoundFloat(couponDiscount, 2)
				cid := utils.MustSafeUintToInt64(coupon.ID)
				couponID = &cid
			}
		}
	}

	finalPrice = finalPrice - couponDiscount
	if finalPrice < 0 {
		finalPrice = 0
	}
	finalPrice = utils.RoundFloat(finalPrice, 2)

	// 构建额外数据
	extraData := map[string]interface{}{
		"type":             "custom_package",
		"devices":          req.Devices,
		"months":           req.Months,
		"discount_percent": discountPercent,
	}
	extraDataJSON, _ := json.Marshal(extraData)
	extraStr := string(extraDataJSON)

	// 创建订单号
	orderNo, err := utils.GenerateOrderNo(db)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成订单号失败", err)
		return
	}
	expireTime := time.Now().Add(30 * time.Minute)
	totalDiscount := basePrice - finalPrice

	order := models.Order{
		OrderNo:        orderNo,
		UserID:         user.ID,
		PackageID:      0,
		Amount:         basePrice,
		Status:         "pending",
		ExpireTime:     database.NullTime(expireTime),
		DiscountAmount: database.NullFloat64(totalDiscount),
		FinalAmount:    database.NullFloat64(finalPrice),
		ExtraData:      database.NullString(extraStr),
	}

	if couponID != nil {
		order.CouponID = database.NullInt64(*couponID)
	}

	if err := db.Create(&order).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建订单失败", err)
		return
	}

	// 记录优惠券使用
	if couponID != nil {
		db.Create(&models.CouponUsage{
			CouponID:       utils.MustSafeInt64ToUint(*couponID),
			UserID:         user.ID,
			OrderID:        database.NullInt64(utils.MustSafeUintToInt64(order.ID)),
			DiscountAmount: couponDiscount,
		})
		db.Model(&models.Coupon{}).Where("id = ?", *couponID).UpdateColumn("used_quantity", gorm.Expr("used_quantity + 1"))
	}

	// 创建审计日志
	pkgName := fmt.Sprintf("自定义套餐 (%d设备/%d月)", req.Devices, req.Months)
	utils.CreateAuditLogSimple(c, "create_custom_order", "order", order.ID, fmt.Sprintf("用户创建自定义套餐订单: %s, 金额: %.2f", pkgName, finalPrice))

	utils.SuccessResponse(c, http.StatusCreated, "订单创建成功", gin.H{
		"order_no":        order.OrderNo,
		"id":              order.ID,
		"user_id":         order.UserID,
		"package_id":      0,
		"package_name":    pkgName,
		"amount":          basePrice,
		"final_amount":    finalPrice,
		"discount_amount": totalDiscount,
		"status":          order.Status,
		"created_at":      utils.FormatBeijingTime(order.CreatedAt),
	})
}

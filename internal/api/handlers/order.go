package handlers

import (
	"encoding/json"
	"fmt"
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

func resolveUserInfo(db *gorm.DB, userID uint, user models.User) gin.H {
	if user.ID > 0 {
		return gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		}
	}
	var u models.User
	if err := db.First(&u, userID).Error; err == nil {
		return gin.H{
			"id":       u.ID,
			"username": u.Username,
			"email":    u.Email,
		}
	}
	return gin.H{
		"id":       userID,
		"username": "已删除",
		"email":    "deleted",
	}
}

func resolvePackageInfo(db *gorm.DB, order models.Order) (uint, string, interface{}) {
	if order.PackageID == 0 {
		name := "设备升级"
		if order.ExtraData.Valid && order.ExtraData.String != "" {
			name = "设备升级订单"
		}
		return 0, name, gin.H{"id": 0, "name": name}
	}

	if order.Package.ID > 0 && order.Package.ID == order.PackageID {
		return order.PackageID, order.Package.Name, gin.H{
			"id":           order.Package.ID,
			"name":         order.Package.Name,
			"price":        order.Package.Price,
			"device_limit": order.Package.DeviceLimit,
		}
	}

	var pkg models.Package
	if err := db.First(&pkg, order.PackageID).Error; err == nil {
		return pkg.ID, pkg.Name, gin.H{
			"id":           pkg.ID,
			"name":         pkg.Name,
			"price":        pkg.Price,
			"device_limit": pkg.DeviceLimit,
		}
	}
	return order.PackageID, "未知套餐", gin.H{"id": order.PackageID, "name": "未知套餐"}
}

func formatOrderData(db *gorm.DB, order models.Order) gin.H {
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

	userInfo := resolveUserInfo(db, order.UserID, order.User)
	pkgID, pkgName, pkgInfo := resolvePackageInfo(db, order)

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
		"created_at":             order.CreatedAt.Format("2006-01-02 15:04:05"),
		"updated_at":             order.UpdatedAt.Format("2006-01-02 15:04:05"),
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

func generatePaymentURL(db *gorm.DB, order *models.Order, paymentConfig *models.PaymentConfig, reqMethod string) (string, error) {
	amount := order.Amount
	if order.FinalAmount.Valid {
		amount = order.FinalAmount.Float64
	}

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

	err = utils.WithTransaction(db, func(tx *gorm.DB) error {
		if isRecharge {
			var record models.RechargeRecord
			if err := tx.Where("order_no = ? AND status = ?", orderNo, "pending").First(&record).Error; err != nil {
				return err
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
	req.ClientIP = c.ClientIP()

	svc := orderServicePkg.NewOrderService()
	order, paymentURL, err := svc.CreateOrder(user.ID, req)
	if err != nil {
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
		"created_at":          order.CreatedAt.Format("2006-01-02 15:04:05"),
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

	query := db.Model(&models.Order{}).Preload("Package").Preload("Coupon")

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
	if startDate != "" {
		query = query.Where("DATE(created_at) >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("DATE(created_at) <= ?", endDate)
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
		formattedOrders[i] = formatOrderData(db, order)
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

		limit := page*size + size
		if limit > 500 {
			limit = 500
		}

		allRecords := make([]gin.H, 0)
		var orders []models.Order
		if err := orderQuery.Preload("User").Preload("Package").Order("orders.created_at DESC").Limit(limit).Find(&orders).Error; err == nil {
			for _, order := range orders {
				formatted := formatOrderData(db, order)
				allRecords = append(allRecords, gin.H{
					"record_type": "order",
					"created_at":  formatted["created_at"],
					"data":        formatted,
				})
			}
		}

		var recharges []models.RechargeRecord
		if err := rechargeQuery.Order("created_at DESC").Limit(limit).Find(&recharges).Error; err == nil {
			for _, record := range recharges {
				userInfo := resolveUserInfo(db, record.UserID, record.User)
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
					"created_at":             record.CreatedAt.Format("2006-01-02 15:04:05"),
				}
				allRecords = append(allRecords, gin.H{
					"record_type": "recharge",
					"created_at":  rechargeData["created_at"],
					"data":        rechargeData,
				})
			}
		}

		sort.Slice(allRecords, func(i, j int) bool {
			timeI, _ := time.Parse("2006-01-02 15:04:05", allRecords[i]["created_at"].(string))
			timeJ, _ := time.Parse("2006-01-02 15:04:05", allRecords[j]["created_at"].(string))
			return timeI.After(timeJ)
		})

		offset := (page - 1) * size
		end := offset + size
		if end > len(allRecords) {
			end = len(allRecords)
		}

		mergedList := make([]gin.H, 0)
		if offset < len(allRecords) {
			for i := offset; i < end; i++ {
				record := allRecords[i]["data"].(gin.H)
				record["record_type"] = allRecords[i]["record_type"]
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
		orderList[i] = formatOrderData(db, order)
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

	utils.SuccessResponse(c, http.StatusOK, "订单已更新", order)
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

	for _, orderID := range req.OrderIDs {
		var order models.Order
		if err := db.Where("id = ? AND status = ?", orderID, "pending").First(&order).Error; err != nil {
			failCount++
			continue
		}
		order.PaymentTime = database.NullTime(utils.GetBeijingTime())
		if _, err := svc.ProcessPaidOrder(&order); err != nil {
			utils.LogError("BulkMarkOrdersPaid: process order failed", err, map[string]interface{}{"order_id": orderID})
			failCount++
		} else {
			successCount++
			go sendPaymentNotifications(db, order.OrderNo)
		}
	}
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
		formatted := formatOrderData(db, order)
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
			order.CreatedAt.Format("2006-01-02 15:04:05"),
			formatted["payment_time"],
			order.UpdatedAt.Format("2006-01-02 15:04:05"),
		))
	}

	filename := fmt.Sprintf("orders_export_%s.csv", time.Now().Format("20060102_150405"))
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
					svc.ProcessPaidOrder(&processedOrder)
					sendPaymentNotifications(db, orderNo)
				}
			}()
			db.Where("order_no = ?", orderNo).First(&order)
		}
	}

	orderType := "order"
	if order.PackageID == 0 {
		orderType = "device_upgrade"
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"order_no": order.OrderNo,
		"status":   order.Status,
		"amount":   order.Amount,
		"type":     orderType,
	})
}

func UpgradeDevices(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	var req struct {
		AdditionalDevices int     `json:"additional_devices" binding:"required,min=5"`
		AdditionalDays    int     `json:"additional_days"`
		PaymentMethod     string  `json:"payment_method"`
		UseBalance        bool    `json:"use_balance"`
		BalanceAmount     float64 `json:"balance_amount"`
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

	devicePricePerMonth := config.AppConfig.DeviceUpgradePricePerMonth
	if devicePricePerMonth <= 0 {
		devicePricePerMonth = 10.0
	}
	totalAmount := (float64(req.AdditionalDevices) * devicePricePerMonth) + (float64(req.AdditionalDays) * (devicePricePerMonth / 30.0))

	var userLevel models.UserLevel
	if user.UserLevelID.Valid {
		if err := db.First(&userLevel, user.UserLevelID.Int64).Error; err == nil && userLevel.DiscountRate > 0 && userLevel.DiscountRate < 1.0 {
			totalAmount *= userLevel.DiscountRate
		}
	}

	balanceUsed := 0.0
	finalAmount := totalAmount
	if req.UseBalance {
		if user.Balance <= 0 {
			utils.ErrorResponse(c, http.StatusBadRequest, "余额不足", nil)
			return
		}
		availableBalance := user.Balance
		if req.BalanceAmount > 0 && req.BalanceAmount < user.Balance {
			availableBalance = req.BalanceAmount
		}
		if availableBalance > finalAmount {
			balanceUsed = finalAmount
		} else {
			balanceUsed = availableBalance
		}
		finalAmount -= balanceUsed
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
		FinalAmount:       database.NullFloat64(actualPaidAmount),
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
		if balanceUsed > 0 {
			user.Balance -= balanceUsed
			if err := db.Save(&user).Error; err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, "扣除余额失败", err)
				return
			}
		}
		order.Status = "paid"
		order.PaymentTime = database.NullTime(utils.GetBeijingTime())
		db.Save(&order)
		orderServicePkg.NewOrderService().ProcessPaidOrder(&order)
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
		}

		var paymentConfig models.PaymentConfig
		if err := db.Where("LOWER(pay_type) LIKE ? AND status = 1", "%"+queryPayType+"%").Order("sort_order ASC").First(&paymentConfig).Error; err != nil {
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
		db.Create(&transaction)
		paymentURL, err = generatePaymentURL(db, &order, &paymentConfig, req.PaymentMethod)
		if err != nil {
			utils.LogError("UpgradeDevices: create payment failed", err, nil)
		}
	}

	var extraDataMap map[string]interface{}
	json.Unmarshal([]byte(order.ExtraData.String), &extraDataMap)
	extraDataMap["type"] = "device_upgrade"
	extraDataMap["additional_devices"] = req.AdditionalDevices
	extraDataMap["additional_days"] = req.AdditionalDays
	if balanceUsed > 0 {
		extraDataMap["balance_used"] = balanceUsed
	}
	extraDataBytes, _ := json.Marshal(extraDataMap)
	order.ExtraData = database.NullString(string(extraDataBytes))
	db.Save(&order)

	responseData := gin.H{
		"order_no":           order.OrderNo,
		"id":                 order.ID,
		"status":             order.Status,
		"amount":             totalAmount,
		"final_amount":       finalAmount,
		"balance_used":       balanceUsed,
		"additional_devices": req.AdditionalDevices,
		"additional_days":    req.AdditionalDays,
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
		PaymentMethodID uint   `json:"payment_method_id" binding:"required"`
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

	var paymentConfig models.PaymentConfig
	if err := db.First(&paymentConfig, req.PaymentMethodID).Error; err != nil || paymentConfig.Status != 1 {
		utils.ErrorResponse(c, http.StatusBadRequest, "支付方式无效", nil)
		return
	}

	amount := order.Amount
	if order.FinalAmount.Valid {
		amount = order.FinalAmount.Float64
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

	paymentURL, err := generatePaymentURL(db, &order, &paymentConfig, req.PaymentMethod)
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

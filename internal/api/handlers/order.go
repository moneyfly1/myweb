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

func buildOrderListData(db *gorm.DB, orders []models.Order) []gin.H {
	orderList := make([]gin.H, 0, len(orders))
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

		var username, email string
		var userID uint
		if order.User.ID > 0 {
			userID = order.User.ID
			username = order.User.Username
			email = order.User.Email
		} else {
			var user models.User
			if err := db.First(&user, order.UserID).Error; err == nil {
				userID = user.ID
				username = user.Username
				email = user.Email
			} else {
				userID = order.UserID
				username = "已删除"
				email = "deleted"
			}
		}

		userInfo := gin.H{
			"id":       userID,
			"username": username,
			"email":    email,
		}

		var packageName string
		if order.PackageID == 0 {
			packageName = "设备升级"
			if order.ExtraData.Valid && order.ExtraData.String != "" {
				packageName = "设备升级订单"
			}
		} else {
			if order.Package.ID > 0 && order.Package.ID == order.PackageID {
				packageName = order.Package.Name
			} else {
				var pkg models.Package
				if err := db.First(&pkg, order.PackageID).Error; err == nil {
					packageName = pkg.Name
				} else {
					packageName = "未知套餐"
				}
			}
		}

		orderList = append(orderList, gin.H{
			"id":             order.ID,
			"order_no":       order.OrderNo,
			"user_id":        order.UserID,
			"user":           userInfo,
			"package_id":     order.PackageID,
			"package_name":   packageName,
			"amount":         amount,
			"payment_method": paymentMethod,
			"payment_time":   paymentTime,
			"status":         order.Status,
			"created_at":     order.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return orderList
}

type CreateOrderRequest struct {
	PackageID     uint    `json:"package_id" binding:"required"`
	CouponCode    string  `json:"coupon_code"`
	PaymentMethod string  `json:"payment_method"`
	Amount        float64 `json:"amount"`
	UseBalance    bool    `json:"use_balance"`
	BalanceAmount float64 `json:"balance_amount"`
	Currency      string  `json:"currency"`
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

	svc := orderServicePkg.NewOrderService()
	order, paymentURL, err := svc.CreateOrder(user.ID, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	data := gin.H{
		"order_no":        order.OrderNo,
		"id":              order.ID,
		"user_id":         order.UserID,
		"package_id":      order.PackageID,
		"amount":          order.Amount,
		"final_amount":    utils.GetNullFloat64Value(order.FinalAmount),
		"discount_amount": utils.GetNullFloat64Value(order.DiscountAmount),
		"status":          order.Status,
		"created_at":      order.CreatedAt.Format("2006-01-02 15:04:05"),
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
			"payment_method":         paymentMethod,
			"payment_method_id":      utils.GetNullInt64Value(order.PaymentMethodID),
			"payment_time":           utils.GetNullTimeValue(order.PaymentTime),
			"payment_transaction_id": utils.GetNullStringValue(order.PaymentTransactionID),
			"expire_time":            utils.GetNullTimeValue(order.ExpireTime),
			"coupon_id":              utils.GetNullInt64Value(order.CouponID),
			"discount_amount":        utils.GetNullFloat64Value(order.DiscountAmount),
			"final_amount":           utils.GetNullFloat64Value(order.FinalAmount),
			"created_at":             order.CreatedAt,
			"updated_at":             order.UpdatedAt,
		}

		if order.PackageID == 0 {
			formattedOrders[i]["package"] = gin.H{
				"id":   0,
				"name": "设备升级",
			}
			formattedOrders[i]["package_name"] = "设备升级"
		} else if order.Package.ID > 0 {
			formattedOrders[i]["package"] = gin.H{
				"id":           order.Package.ID,
				"name":         order.Package.Name,
				"price":        order.Package.Price,
				"device_limit": order.Package.DeviceLimit,
			}
			formattedOrders[i]["package_name"] = order.Package.Name
		} else {
			var pkg models.Package
			if err := db.First(&pkg, order.PackageID).Error; err == nil {
				formattedOrders[i]["package"] = gin.H{
					"id":           pkg.ID,
					"name":         pkg.Name,
					"price":        pkg.Price,
					"device_limit": pkg.DeviceLimit,
				}
				formattedOrders[i]["package_name"] = pkg.Name
			} else {
				formattedOrders[i]["package_name"] = "未知套餐"
			}
		}

		if order.Coupon.ID > 0 {
			formattedOrders[i]["coupon"] = gin.H{
				"id":   order.Coupon.ID,
				"code": order.Coupon.Code,
				"name": order.Coupon.Name,
			}
		}
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

	page := 1
	size := 20
	if pageStr := c.Query("page"); pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if sizeStr := c.Query("size"); sizeStr != "" {
		fmt.Sscanf(sizeStr, "%d", &size)
	}
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

	keyword := c.Query("keyword")
	if keyword == "" {
		keyword = c.Query("search")
	}

	status := c.Query("status")

	if includeRecharges {
		var orderCount, rechargeCount int64
		orderCountQuery := db.Model(&models.Order{})
		rechargeCountQuery := db.Model(&models.RechargeRecord{})

		if keyword != "" {
			sanitizedKeyword := utils.SanitizeSearchKeyword(keyword)
			if sanitizedKeyword != "" {
				orderCountQuery = orderCountQuery.Joins("LEFT JOIN users ON orders.user_id = users.id").Where(
					"orders.order_no LIKE ? OR orders.order_no LIKE ? OR users.username LIKE ? OR users.email LIKE ?",
					"%"+sanitizedKeyword+"%", "%ORD%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%",
				)
				rechargeCountQuery = rechargeCountQuery.Joins("LEFT JOIN users ON recharge_records.user_id = users.id").Where(
					"recharge_records.order_no LIKE ? OR recharge_records.order_no LIKE ? OR users.username LIKE ? OR users.email LIKE ?",
					"%"+sanitizedKeyword+"%", "%RCH%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%",
				)
			}
		}
		if status != "" && status != "all" {
			orderCountQuery = orderCountQuery.Where("orders.status = ?", status)
			rechargeCountQuery = rechargeCountQuery.Where("recharge_records.status = ?", status)
		}
		orderCountQuery.Count(&orderCount)
		rechargeCountQuery.Count(&rechargeCount)
		total := orderCount + rechargeCount

		limit := page*size + size
		if limit > 500 {
			limit = 500
		}

		allRecords := make([]gin.H, 0)

		orderQuery := db.Model(&models.Order{}).Joins("LEFT JOIN users ON orders.user_id = users.id")
		if keyword != "" {
			sanitizedKeyword := utils.SanitizeSearchKeyword(keyword)
			if sanitizedKeyword != "" {
				orderQuery = orderQuery.Where(
					"orders.order_no LIKE ? OR orders.order_no LIKE ? OR users.username LIKE ? OR users.email LIKE ?",
					"%"+sanitizedKeyword+"%", "%ORD%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%",
				)
			}
		}
		if status != "" && status != "all" {
			orderQuery = orderQuery.Where("orders.status = ?", status)
		}

		var orders []models.Order
		orderQuery = orderQuery.Preload("User").Preload("Package")
		if err := orderQuery.Order("orders.created_at DESC").Limit(limit).Find(&orders).Error; err == nil {
			orderList := buildOrderListData(db, orders)
			for _, order := range orderList {
				allRecords = append(allRecords, gin.H{
					"record_type": "order",
					"created_at":  order["created_at"],
					"data":        order,
				})
			}
		}

		rechargeQuery := db.Model(&models.RechargeRecord{}).Joins("LEFT JOIN users ON recharge_records.user_id = users.id")
		if keyword != "" {
			sanitizedKeyword := utils.SanitizeSearchKeyword(keyword)
			if sanitizedKeyword != "" {
				rechargeQuery = rechargeQuery.Where(
					"recharge_records.order_no LIKE ? OR recharge_records.order_no LIKE ? OR users.username LIKE ? OR users.email LIKE ?",
					"%"+sanitizedKeyword+"%", "%RCH%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%",
				)
			}
		}
		if status != "" && status != "all" {
			rechargeQuery = rechargeQuery.Where("recharge_records.status = ?", status)
		}

		var recharges []models.RechargeRecord
		if err := rechargeQuery.Order("created_at DESC").Limit(limit).Find(&recharges).Error; err == nil {
			for _, record := range recharges {
				userInfo := gin.H{}
				if record.User.ID > 0 {
					userInfo = gin.H{
						"id":       record.User.ID,
						"username": record.User.Username,
						"email":    record.User.Email,
					}
				}

				rechargeData := gin.H{
					"id":                     record.ID,
					"user_id":                record.UserID,
					"user":                   userInfo,
					"order_no":               record.OrderNo,
					"amount":                 record.Amount,
					"status":                 record.Status,
					"payment_method":         utils.GetNullStringValue(record.PaymentMethod),
					"payment_transaction_id": utils.GetNullStringValue(record.PaymentTransactionID),
					"paid_at": func() interface{} {
						if record.PaidAt.Valid {
							return record.PaidAt.Time.Format("2006-01-02 15:04:05")
						}
						return nil
					}(),
					"created_at": record.CreatedAt.Format("2006-01-02 15:04:05"),
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
				"%"+sanitizedKeyword+"%",     // 完整订单号匹配
				"%ORD%"+sanitizedKeyword+"%", // 时间戳匹配（订单号格式：ORD + timestamp + 随机数）
				"%"+sanitizedKeyword+"%",     // 用户名匹配
				"%"+sanitizedKeyword+"%",     // 邮箱匹配
			)
		}
	}

	if status != "" && status != "all" {
		query = query.Where("orders.status = ?", status)
	}

	var total int64
	countQuery := db.Model(&models.Order{}).Joins("LEFT JOIN users ON orders.user_id = users.id")
	if keyword != "" {
		sanitizedKeyword := utils.SanitizeSearchKeyword(keyword)
		if sanitizedKeyword != "" {
			countQuery = countQuery.Where(
				"orders.order_no LIKE ? OR orders.order_no LIKE ? OR users.username LIKE ? OR users.email LIKE ?",
				"%"+sanitizedKeyword+"%",     // 完整订单号匹配
				"%ORD%"+sanitizedKeyword+"%", // 时间戳匹配
				"%"+sanitizedKeyword+"%",     // 用户名匹配
				"%"+sanitizedKeyword+"%",     // 邮箱匹配
			)
		}
	}
	if status != "" && status != "all" {
		countQuery = countQuery.Where("orders.status = ?", status)
	}
	countQuery.Count(&total)

	query = query.Preload("User").Preload("Package")

	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("orders.created_at DESC").Find(&orders).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取订单列表失败", err)
		return
	}

	orderList := buildOrderListData(db, orders)
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

		var user models.User
		if err := db.First(&user, order.UserID).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取用户信息失败", err)
			return
		}

		svc := orderServicePkg.NewOrderService()
		_, err := svc.ProcessPaidOrder(&order)
		if err != nil {
			utils.LogError("BulkMarkOrdersPaid: process paid order", err, map[string]interface{}{
				"order_id": order.ID,
			})
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
		utils.LogError("DeleteOrder: delete order failed", err, map[string]interface{}{
			"order_id": order.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除订单失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "订单已删除", nil)
}

func GetOrderStatistics(c *gin.Context) {
	db := database.GetDB()

	var orderTotal int64
	var orderPending int64
	var orderPaid int64
	var orderRevenue float64

	db.Model(&models.Order{}).Count(&orderTotal)
	db.Model(&models.Order{}).Where("status = ?", "pending").Count(&orderPending)
	db.Model(&models.Order{}).Where("status = ?", "paid").Count(&orderPaid)

	orderRevenue = utils.CalculateTotalRevenue(db, "paid")

	var rechargeTotal int64
	var rechargePending int64
	var rechargePaid int64
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

	totalOrders := orderTotal + rechargeTotal
	pendingOrders := orderPending + rechargePending
	paidOrders := orderPaid + rechargePaid
	totalRevenue := orderRevenue + rechargeRevenue

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"total_orders":   totalOrders,
		"pending_orders": pendingOrders,
		"paid_orders":    paidOrders,
		"total_revenue":  totalRevenue,
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

		now := utils.GetBeijingTime()
		order.PaymentTime = database.NullTime(now)

		if _, err := svc.ProcessPaidOrder(&order); err != nil {
			utils.LogError("BulkMarkOrdersPaid: process order failed", err, map[string]interface{}{
				"order_id": orderID,
			})
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
		} else {
			var user models.User
			if err := db.First(&user, order.UserID).Error; err == nil {
				username = user.Username
				email = user.Email
			} else {
				username = "已删除"
				email = "deleted"
			}
		}

		packageName := ""
		if order.PackageID == 0 {
			packageName = "设备升级"
		} else if order.Package.ID > 0 && order.Package.ID == order.PackageID {
			packageName = order.Package.Name
		} else {
			var pkg models.Package
			if err := db.First(&pkg, order.PackageID).Error; err == nil {
				packageName = pkg.Name
			} else {
				packageName = "未知套餐"
			}
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

	totalOrders := orderTotal + rechargeTotal
	pendingOrders := orderPending + rechargePending
	paidOrders := orderPaid + rechargePaid
	totalAmount := orderPaidAmount + rechargePaidAmount

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"total":       totalOrders,
		"pending":     pendingOrders,
		"paid":        paidOrders,
		"cancelled":   orderCancelled,
		"totalAmount": totalAmount,
		"paidAmount":  totalAmount,
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
	var order models.Order
	if err := db.Where("order_no = ? AND user_id = ?", orderNo, user.ID).First(&order).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "订单不存在", err)
		return
	}

	if order.Status == "pending" {
		timeSinceCreated := time.Since(order.CreatedAt)

		var shouldQuery bool
		if timeSinceCreated >= 3*time.Second && timeSinceCreated < 10*time.Second {
			shouldQuery = true
			utils.LogInfo("订单状态查询(快速模式): order_no=%s, time_since_created=%.1fs", orderNo, timeSinceCreated.Seconds())
		} else if timeSinceCreated >= 10*time.Second && timeSinceCreated < 60*time.Second {
			shouldQuery = int(timeSinceCreated.Seconds())%5 < 2
			if shouldQuery {
				utils.LogInfo("订单状态查询(常规模式): order_no=%s, time_since_created=%.1fs", orderNo, timeSinceCreated.Seconds())
			}
		} else if timeSinceCreated >= 60*time.Second {
			shouldQuery = int(timeSinceCreated.Seconds())%30 < 2
			if shouldQuery {
				utils.LogInfo("订单状态查询(慢速模式): order_no=%s, time_since_created=%.1fs", orderNo, timeSinceCreated.Seconds())
			}
		}

		if shouldQuery {
			var transaction models.PaymentTransaction
			if err := db.Where("order_id = ?", order.ID).First(&transaction).Error; err == nil {
				var paymentConfig models.PaymentConfig
				if err := db.First(&paymentConfig, transaction.PaymentMethodID).Error; err == nil {
					if paymentConfig.PayType == "alipay" {
						alipayService, err := payment.NewAlipayService(&paymentConfig)
						if err == nil {
							queryResult, err := alipayService.QueryOrder(orderNo)
							if err == nil && queryResult != nil {
								utils.LogInfo("支付宝查询结果: order_no=%s, status=%s, paid=%v", orderNo, queryResult.TradeStatus, queryResult.IsPaid())

								if queryResult.IsPaid() {

									tx := db.Begin()
									success := false
									defer func() {
										if !success {
											tx.Rollback()
										}
									}()

									var latestOrder models.Order
									if err := tx.Where("order_no = ? AND status = ?", orderNo, "pending").First(&latestOrder).Error; err == nil {
										latestOrder.Status = "paid"
										latestOrder.PaymentTime = database.NullTime(utils.GetBeijingTime())
										if err := tx.Save(&latestOrder).Error; err == nil {
											var latestTransaction models.PaymentTransaction
											if err := tx.Where("order_id = ?", latestOrder.ID).First(&latestTransaction).Error; err == nil {
												latestTransaction.Status = "success"
												latestTransaction.ExternalTransactionID = database.NullString(queryResult.TradeNo)
												if err := tx.Save(&latestTransaction).Error; err == nil {
													if err := tx.Commit().Error; err == nil {
														success = true
														utils.LogInfo("主动查询发现支付成功: order_no=%s, trade_no=%s", orderNo, queryResult.TradeNo)

														go func() {
															var processedOrder models.Order
															if err := db.Preload("Package").Where("order_no = ?", orderNo).First(&processedOrder).Error; err == nil {
																if processedOrder.Status == "paid" {
																	var processedUser models.User
																	if db.First(&processedUser, processedOrder.UserID).Error == nil {
																		svc := orderServicePkg.NewOrderService()
																		svc.ProcessPaidOrder(&processedOrder)
																	}
																	sendPaymentNotifications(db, orderNo)
																}
															}
														}()

														db.Where("order_no = ?", orderNo).First(&order)
													}
												}
											}
										}
									}
								}
							} else if err != nil {
								utils.LogWarn("查询支付宝订单状态失败: order_no=%s, error=%v", orderNo, err)
							}
						}
					}
				}
			}
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"order_no": order.OrderNo,
		"status":   order.Status,
		"amount":   order.Amount,
	})
}

func UpgradeDevices(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
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
		utils.LogError("UpdateAdminOrder: bind request", err, nil)
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误，请检查输入格式", err)
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
		devicePricePerMonth = 10.0 // 默认值
	}
	devicePricePerDay := devicePricePerMonth / 30.0
	deviceCost := float64(req.AdditionalDevices) * devicePricePerMonth
	daysCost := float64(req.AdditionalDays) * devicePricePerDay
	totalAmount := deviceCost + daysCost

	var userLevel models.UserLevel
	if user.UserLevelID.Valid {
		if err := db.First(&userLevel, user.UserLevelID.Int64).Error; err == nil {
			if userLevel.DiscountRate > 0 && userLevel.DiscountRate < 1.0 {
				totalAmount *= userLevel.DiscountRate
			}
		}
	}

	balanceUsed := 0.0
	finalAmount := totalAmount
	if req.UseBalance {
		if user.Balance <= 0 {
			utils.ErrorResponse(c, http.StatusBadRequest, "您的余额为0，无法使用余额支付", nil)
			return
		}

		availableBalance := user.Balance
		if req.BalanceAmount > 0 {
			availableBalance = req.BalanceAmount
			if availableBalance > user.Balance {
				availableBalance = user.Balance
			}
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
		OrderNo:        orderNo,
		UserID:         user.ID,
		PackageID:      0, // 设备升级订单
		Amount:         totalAmount,
		FinalAmount:    database.NullFloat64(actualPaidAmount), // 记录实际支付金额
		DiscountAmount: database.NullFloat64(totalAmount - actualPaidAmount),
		Status:         "pending",
		ExtraData:      database.NullString(extraData),
	}

	if balanceUsed > 0 && finalAmount > 0.01 {
		order.PaymentMethodName = database.NullString(fmt.Sprintf("余额支付(%.2f元)+%s", balanceUsed, req.PaymentMethod))
	} else if balanceUsed > 0 {
		order.PaymentMethodName = database.NullString("余额支付")
	} else if req.PaymentMethod != "" {
		order.PaymentMethodName = database.NullString(req.PaymentMethod)
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
		if err := db.Save(&order).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "更新订单状态失败", err)
			return
		}

		orderService := orderServicePkg.NewOrderService()
		_, err := orderService.ProcessPaidOrder(&order)
		if err != nil {
			utils.LogError("UpgradeDevices: process paid order failed", err, map[string]interface{}{
				"order_id": order.ID,
			})
			utils.ErrorResponse(c, http.StatusInternalServerError, "处理订单失败", err)
			return
		}

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
		var paymentConfig models.PaymentConfig
		payType := req.PaymentMethod
		queryPayType := payType

		if payType == "mixed" {
			payType = "alipay"
			queryPayType = "alipay"
		}

		if strings.HasPrefix(payType, "yipay_") {
			if err := db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", "yipay", 1).Order("sort_order ASC").First(&paymentConfig).Error; err == nil {
				queryPayType = "yipay"
			} else {
				if err := db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", payType, 1).Order("sort_order ASC").First(&paymentConfig).Error; err != nil {
					utils.ErrorResponse(c, http.StatusBadRequest, "未找到启用的支付配置", nil)
					return
				}
			}
		} else if payType == "alipay" || payType == "wechat" {
			if err := db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", payType, 1).Order("sort_order ASC").First(&paymentConfig).Error; err != nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "未找到启用的支付配置", nil)
				return
			}
		} else {
			utils.ErrorResponse(c, http.StatusBadRequest, "不支持的支付方式", nil)
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
		if err := db.Create(&transaction).Error; err == nil {
			if paymentConfig.PayType == "alipay" {
				alipayService, err := payment.NewAlipayService(&paymentConfig)
				if err == nil {
					paymentURL, err = alipayService.CreatePayment(&order, finalAmount)
					if err != nil {
						utils.LogError("UpgradeDevices: create alipay payment failed", err, nil)
					}
				}
			} else if paymentConfig.PayType == "wechat" {
				wechatService, err := payment.NewWechatService(&paymentConfig)
				if err == nil {
					paymentURL, err = wechatService.CreatePayment(&order, finalAmount)
					if err != nil {
						utils.LogError("UpgradeDevices: create wechat payment failed", err, nil)
					}
				}
			} else if queryPayType == "yipay" || strings.HasPrefix(payType, "yipay_") {
				yipayService, err := payment.NewYipayService(&paymentConfig)
				if err == nil {
					paymentType := extractYipayPaymentType(payType)
					paymentURL, err = yipayService.CreatePayment(&order, finalAmount, paymentType)
					if err != nil {
						utils.LogError("UpgradeDevices: create yipay payment failed", err, nil)
					}
				}
			}
		}
	}

	var extraDataMap map[string]interface{}
	if order.ExtraData.Valid && order.ExtraData.String != "" {
		json.Unmarshal([]byte(order.ExtraData.String), &extraDataMap)
	} else {
		extraDataMap = make(map[string]interface{})
	}
	extraDataMap["type"] = "device_upgrade"
	extraDataMap["additional_devices"] = req.AdditionalDevices
	extraDataMap["additional_days"] = req.AdditionalDays
	if balanceUsed > 0 {
		extraDataMap["balance_used"] = balanceUsed
	}
	extraDataBytes, _ := json.Marshal(extraDataMap)
	order.ExtraData = database.NullString(string(extraDataBytes))
	if err := db.Save(&order).Error; err != nil {
		utils.LogError("UpgradeDevices: save order extra data failed", err, nil)
	}

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
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误: 缺少 payment_method_id 参数", err)
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
	if err := db.First(&paymentConfig, req.PaymentMethodID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "支付方式不存在", nil)
		return
	}

	if paymentConfig.Status != 1 {
		utils.ErrorResponse(c, http.StatusBadRequest, "支付方式已停用", nil)
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
		Amount:          int(amount * 100), // 转换为分
		Currency:        "CNY",
		Status:          "pending",
	}

	if err := db.Create(&transaction).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建支付交易失败", err)
		return
	}

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
	} else if paymentConfig.PayType == "yipay" || strings.HasPrefix(paymentConfig.PayType, "yipay_") {
		yipayService, err := payment.NewYipayService(&paymentConfig)
		if err != nil {
			payErr = fmt.Errorf("初始化易支付服务失败: %v", err)
		} else {
			paymentType := "alipay"
			if req.PaymentMethod != "" && strings.HasPrefix(req.PaymentMethod, "yipay_") {
				paymentType = strings.TrimPrefix(req.PaymentMethod, "yipay_")
			} else if order.PaymentMethodName.Valid {
				paymentMethodName := order.PaymentMethodName.String
				if strings.Contains(paymentMethodName, "yipay_wxpay") || strings.Contains(paymentMethodName, "易支付-微信") {
					paymentType = "wxpay"
				} else if strings.Contains(paymentMethodName, "yipay_alipay") || strings.Contains(paymentMethodName, "易支付-支付宝") {
					paymentType = "alipay"
				} else if strings.Contains(paymentMethodName, "yipay_qqpay") || strings.Contains(paymentMethodName, "易支付-QQ") {
					paymentType = "qqpay"
				} else if strings.HasPrefix(paymentMethodName, "yipay_") {
					parts := strings.Split(paymentMethodName, "yipay_")
					if len(parts) > 1 {
						for _, part := range parts {
							if strings.HasPrefix(part, "wxpay") {
								paymentType = "wxpay"
								break
							} else if strings.HasPrefix(part, "alipay") {
								paymentType = "alipay"
								break
							} else if strings.HasPrefix(part, "qqpay") {
								paymentType = "qqpay"
								break
							}
						}
					}
				}
			} else {
				paymentType = extractYipayPaymentType(paymentConfig.PayType)
			}
			utils.LogInfo("PayOrder: 易支付支付类型提取 - payment_method=%s, payment_method_name=%s, extracted_type=%s",
				req.PaymentMethod, order.PaymentMethodName.String, paymentType)
			paymentURL, payErr = yipayService.CreatePayment(&order, amount, paymentType)
		}
	} else {
		payErr = fmt.Errorf("不支持的支付方式: %s", paymentConfig.PayType)
	}

	if payErr != nil {
		utils.LogError("CreateOrder: create payment failed", payErr, map[string]interface{}{
			"order_id": order.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建支付失败", payErr)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "支付订单已创建", gin.H{
		"payment_url":    paymentURL,
		"order_no":       order.OrderNo,
		"amount":         amount,
		"transaction_id": transaction.ID,
	})
}

func extractYipayPaymentType(payType string) string {
	if strings.HasPrefix(payType, "yipay_") {
		return strings.TrimPrefix(payType, "yipay_")
	}
	return "alipay"
}

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
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

		// 添加套餐信息（处理设备升级订单 PackageID=0）
		if order.PackageID == 0 {
			// 设备升级订单
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
			// Preload 失败，尝试单独查询
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

// CancelOrder 取消订单
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
		utils.ErrorResponse(c, http.StatusNotFound, "订单不存在", nil)
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
	// 不 Preload，避免 PackageID=0 的问题，在循环中单独查询
	query := db.Model(&models.Order{})

	// 分页参数（支持 page/size 和 skip/limit）
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

	// 使用 Preload 预加载 User 和 Package，避免 N+1 查询
	query = query.Preload("User").Preload("Package")

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

		// 处理用户信息 - 如果 Preload 失败，单独查询
		var username, email string
		var userID uint
		if order.User.ID > 0 {
			userID = order.User.ID
			username = order.User.Username
			email = order.User.Email
		} else {
			// Preload 失败，单独查询用户
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

		// 构建用户对象（前端期望嵌套的 user 对象）
		userInfo := gin.H{
			"id":       userID,
			"username": username,
			"email":    email,
		}

		// 处理 Package 信息（设备升级订单 PackageID 为 0）
		var packageName string
		if order.PackageID == 0 {
			packageName = "设备升级"
			// 如果有 ExtraData，尝试解析显示升级详情
			if order.ExtraData.Valid && order.ExtraData.String != "" {
				packageName = "设备升级订单"
			}
		} else {
			// 检查 Package 是否已加载
			if order.Package.ID > 0 && order.Package.ID == order.PackageID {
				packageName = order.Package.Name
			} else {
				// Preload 失败，单独查询 Package
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

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"orders": orderList,
		"total":  total,
		"page":   page,
		"size":   size,
	})
}

// UpdateAdminOrder 管理员更新订单状态
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

	// 如果订单状态从非paid变为paid，需要处理订阅
	if oldStatus != "paid" && req.Status == "paid" {
		// 设置支付时间
		now := utils.GetBeijingTime()
		order.PaymentTime = database.NullTime(now)

		// 获取用户信息
		var user models.User
		if err := db.First(&user, order.UserID).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取用户信息失败", err)
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

// DeleteAdminOrder 管理员删除订单
func DeleteAdminOrder(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()

	// 先检查订单是否存在
	var order models.Order
	if err := db.First(&order, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订单不存在", nil)
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询订单失败", err)
		return
	}

	// 删除订单
	if err := db.Delete(&order).Error; err != nil {
		utils.LogError("DeleteOrder: delete order failed", err, map[string]interface{}{
			"order_id": order.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除订单失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "订单已删除", nil)
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

	// 统计总收入（使用公共函数）
	totalRevenue = utils.CalculateTotalRevenue(db, "paid")

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"total_orders":   totalOrders,
		"pending_orders": pendingOrders,
		"paid_orders":    paidOrders,
		"total_revenue":  totalRevenue,
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

	utils.SuccessResponse(c, http.StatusOK, "批量标记成功", nil)
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

	utils.SuccessResponse(c, http.StatusOK, "批量取消成功", nil)
}

// BatchDeleteOrders 批量删除订单
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

// ExportOrders 导出订单（CSV格式）
func ExportOrders(c *gin.Context) {
	db := database.GetDB()
	// 不 Preload，避免 PackageID=0 的问题，在循环中单独查询
	query := db.Model(&models.Order{})

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
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取订单列表失败", err)
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

		// 处理用户信息
		username := ""
		email := ""
		if order.User.ID > 0 {
			username = order.User.Username
			email = order.User.Email
		} else {
			// Preload 失败，单独查询
			var user models.User
			if err := db.First(&user, order.UserID).Error; err == nil {
				username = user.Username
				email = user.Email
			} else {
				username = "已删除"
				email = "deleted"
			}
		}

		// 处理套餐信息（设备升级订单 PackageID=0）
		packageName := ""
		if order.PackageID == 0 {
			packageName = "设备升级"
		} else if order.Package.ID > 0 && order.Package.ID == order.PackageID {
			packageName = order.Package.Name
		} else {
			// Preload 失败，单独查询
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

	// 统计总金额（所有订单，使用公共函数，使用绝对值）
	stats.TotalAmount = utils.CalculateUserOrderAmount(db, user.ID, "", true)

	// 统计已支付金额（只统计已支付订单，使用公共函数，使用绝对值）
	stats.PaidAmount = utils.CalculateUserOrderAmount(db, user.ID, "paid", true)

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"total_orders":     stats.TotalOrders,
		"pending_orders":   stats.PendingOrders,
		"paid_orders":      stats.PaidOrders,
		"cancelled_orders": stats.CancelledOrders,
		"total_amount":     stats.TotalAmount,
		"paid_amount":      stats.PaidAmount,
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
																// 统一使用 ProcessPaidOrder 处理所有订单类型（套餐订单和设备升级订单）
																var processedUser models.User
																if db.First(&processedUser, processedOrder.UserID).Error == nil {
																	svc := orderServicePkg.NewOrderService()
																	svc.ProcessPaidOrder(&processedOrder)
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

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"order_no": order.OrderNo,
		"status":   order.Status,
		"amount":   order.Amount,
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
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误，请检查输入格式", err)
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

	// 计算余额使用量（但不立即扣除，等支付成功后再扣除）
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

		// 计算可使用的余额（不超过订单金额和用户余额）
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

	// 创建订单（无论是否完全用余额支付）
	orderNo := utils.GenerateOrderNo(user.ID)
	extraData := fmt.Sprintf(`{"type":"device_upgrade","additional_devices":%d,"additional_days":%d,"balance_used":%.2f}`, req.AdditionalDevices, req.AdditionalDays, balanceUsed)

	// 计算实际支付金额（余额+第三方支付）
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

	// 设置支付方式名称
	if balanceUsed > 0 && finalAmount > 0.01 {
		order.PaymentMethodName = database.NullString(fmt.Sprintf("余额支付(%.2f元)+%s", balanceUsed, req.PaymentMethod))
	} else if balanceUsed > 0 {
		order.PaymentMethodName = database.NullString("余额支付")
	} else if req.PaymentMethod != "" {
		order.PaymentMethodName = database.NullString(req.PaymentMethod)
	}

	if err := db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建订单失败",
		})
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

		// 标记订单为已支付
		order.Status = "paid"
		order.PaymentTime = database.NullTime(utils.GetBeijingTime())
		if err := db.Save(&order).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "更新订单状态失败",
			})
			return
		}

		// 使用统一的订单处理逻辑
		orderService := orderServicePkg.NewOrderService()
		_, err := orderService.ProcessPaidOrder(&order)
		if err != nil {
			utils.LogError("UpgradeDevices: process paid order failed", err, map[string]interface{}{
				"order_id": order.ID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "处理订单失败",
			})
			return
		}

		// 重新加载订阅以获取最新数据
		db.Where("user_id = ?", user.ID).First(&subscription)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "设备数量升级成功",
			"data": gin.H{
				"order_no":           order.OrderNo,
				"status":             "paid",
				"subscription":       subscription,
				"additional_devices": req.AdditionalDevices,
				"additional_days":    req.AdditionalDays,
			},
		})
		return
	}

	// 需要第三方支付，先不扣除余额（等支付成功后再扣除）
	// 生成支付URL（如果需要其他支付方式）
	var paymentURL string
	if finalAmount > 0.01 && req.PaymentMethod != "" && req.PaymentMethod != "balance" {
		var paymentConfig models.PaymentConfig
		payType := req.PaymentMethod
		if payType == "alipay" || payType == "wechat" {
			// 查找对应类型的支付配置（不区分大小写）
			if err := db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", payType, 1).Order("sort_order ASC").First(&paymentConfig).Error; err == nil {
				// 创建支付交易（注意：金额是第三方支付部分，不包括余额）
				transaction := models.PaymentTransaction{
					OrderID:         order.ID,
					UserID:          user.ID,
					PaymentMethodID: paymentConfig.ID,
					Amount:          int(finalAmount * 100), // 只记录第三方支付部分
					Currency:        "CNY",
					Status:          "pending",
				}
				if err := db.Create(&transaction).Error; err == nil {
					// 生成支付URL（只传递第三方支付部分）
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
					}
				}
			}
		}
	}

	// 在订单 ExtraData 中记录升级信息和余额使用量（用于支付回调后处理）
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

	// 返回订单和支付URL
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

	// 获取支付配置
	var paymentConfig models.PaymentConfig
	if err := db.First(&paymentConfig, req.PaymentMethodID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "支付方式不存在", nil)
		return
	}

	if paymentConfig.Status != 1 {
		utils.ErrorResponse(c, http.StatusBadRequest, "支付方式已停用", nil)
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
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建支付交易失败", err)
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

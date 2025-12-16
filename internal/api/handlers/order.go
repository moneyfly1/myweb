package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
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
	amount := pkg.Price
	discountAmount := 0.0
	finalAmount := amount

	// 如果前端提供了金额，使用前端金额（已应用用户等级折扣等）
	if req.Amount > 0 {
		finalAmount = req.Amount
		discountAmount = amount - finalAmount
	}

	// 处理优惠券
	if req.CouponCode != "" {
		var coupon models.Coupon
		if err := db.Where("code = ? AND status = ?", req.CouponCode, "active").First(&coupon).Error; err == nil {
			now := utils.GetBeijingTime()
			if now.After(coupon.ValidFrom) && now.Before(coupon.ValidUntil) {
				// 应用优惠券
				if coupon.Type == "discount" {
					discountAmount = amount * (coupon.DiscountValue / 100)
					if coupon.MaxDiscount.Valid && discountAmount > coupon.MaxDiscount.Float64 {
						discountAmount = coupon.MaxDiscount.Float64
					}
				} else if coupon.Type == "fixed" {
					discountAmount = coupon.DiscountValue
					if discountAmount > amount {
						discountAmount = amount
					}
				}
				finalAmount = amount - discountAmount
			}
		}
	}

	// 处理余额支付
	if req.UseBalance && req.BalanceAmount > 0 {
		// 检查用户余额
		if user.Balance < req.BalanceAmount {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "余额不足",
			})
			return
		}
		// 如果余额足够支付全部金额，直接扣除余额并完成订单
		if req.BalanceAmount >= finalAmount {
			user.Balance -= finalAmount
			if err := db.Save(&user).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": "扣除余额失败",
				})
				return
			}
			// 创建已支付的订单
			orderNo := utils.GenerateOrderNo(user.ID)
			order := models.Order{
				OrderNo:           orderNo,
				UserID:            user.ID,
				PackageID:         pkg.ID,
				Amount:            amount,
				Status:            "paid",
				DiscountAmount:    database.NullFloat64(discountAmount),
				FinalAmount:       database.NullFloat64(finalAmount),
				PaymentMethodName: database.NullString("余额支付"),
			}
			order.PaymentTime = database.NullTime(utils.GetBeijingTime())

			if err := db.Create(&order).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": "创建订单失败",
				})
				return
			}

			// 更新或创建订阅
			var subscription models.Subscription
			if err := db.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
				// 创建新订阅
				subscriptionURL := utils.GenerateSubscriptionURL()
				expireTime := utils.GetBeijingTime().AddDate(0, 0, pkg.DurationDays)
				subscription = models.Subscription{
					UserID:          user.ID,
					PackageID:       database.NullInt64(int64(pkg.ID)),
					SubscriptionURL: subscriptionURL,
					DeviceLimit:     pkg.DeviceLimit,
					IsActive:        true,
					Status:          "active",
					ExpireTime:      expireTime,
				}
				db.Create(&subscription)
			} else {
				// 延长订阅
				if subscription.ExpireTime.Before(utils.GetBeijingTime()) {
					subscription.ExpireTime = utils.GetBeijingTime().AddDate(0, 0, pkg.DurationDays)
				} else {
					subscription.ExpireTime = subscription.ExpireTime.AddDate(0, 0, pkg.DurationDays)
				}
				subscription.DeviceLimit = pkg.DeviceLimit
				subscription.IsActive = true
				subscription.Status = "active"
				db.Save(&subscription)
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
		// 部分余额支付，需要继续支付剩余金额
		finalAmount -= req.BalanceAmount
		user.Balance -= req.BalanceAmount
		if err := db.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "扣除余额失败",
			})
			return
		}
	}

	// 生成订单号
	orderNo := utils.GenerateOrderNo(user.ID)

	// 创建订单
	order := models.Order{
		OrderNo:        orderNo,
		UserID:         user.ID,
		PackageID:      pkg.ID,
		Amount:         amount,
		Status:         "pending",
		DiscountAmount: database.NullFloat64(discountAmount),
		FinalAmount:    database.NullFloat64(finalAmount),
	}

	if req.PaymentMethod != "" && req.PaymentMethod != "balance" {
		order.PaymentMethodName = database.NullString(req.PaymentMethod)
	}

	if err := db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建订单失败",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    order,
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

	// 状态筛选
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 支付方式筛选
	if paymentMethod != "" {
		query = query.Where("payment_method_name = ?", paymentMethod)
	}

	// 计算总数
	query.Count(&total)

	// 分页和排序
	offset := (page - 1) * size
	if err := query.Order("created_at DESC").Offset(offset).Limit(size).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取订单列表失败: " + err.Error(),
		})
		return
	}

	// 计算总页数
	pages := (int(total) + size - 1) / size

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"orders": orders,
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
		query = query.Where("order_no LIKE ? OR user_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ?)",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
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
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	var order models.Order
	if err := db.First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "订单不存在",
		})
		return
	}

	order.Status = req.Status
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
	if err := db.Delete(&models.Order{}, id).Error; err != nil {
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
	db.Model(&models.Order{}).Where("status = ?", "paid").Select("COALESCE(SUM(final_amount), 0)").Scan(&totalRevenue)
	if totalRevenue == 0 {
		db.Model(&models.Order{}).Where("status = ?", "paid").Select("COALESCE(SUM(amount), 0)").Scan(&totalRevenue)
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
		query = query.Where("order_no LIKE ? OR user_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ?)",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
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

	// 统计金额
	var totalAmountResult struct {
		Total float64
	}
	db.Model(&models.Order{}).Where("user_id = ?", user.ID).
		Select("COALESCE(SUM(amount), 0) as total").Scan(&totalAmountResult)
	stats.TotalAmount = totalAmountResult.Total

	var paidAmountResult struct {
		Total float64
	}
	db.Model(&models.Order{}).Where("user_id = ? AND status = ?", user.ID, "paid").
		Select("COALESCE(SUM(final_amount), 0) as total").Scan(&paidAmountResult)
	stats.PaidAmount = paidAmountResult.Total

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"order_no": order.OrderNo,
			"status":   order.Status,
			"amount":   order.Amount,
		},
	})
}

// UpgradeDevices 升级设备数
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
		SubscriptionID uint `json:"subscription_id" binding:"required"`
		NewDeviceLimit int  `json:"new_device_limit" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	var subscription models.Subscription
	if err := db.Where("id = ? AND user_id = ?", req.SubscriptionID, user.ID).First(&subscription).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "订阅不存在",
		})
		return
	}

	// 检查新设备数是否大于当前设备数
	if req.NewDeviceLimit < subscription.DeviceLimit {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "新设备数不能小于当前设备数",
		})
		return
	}

	// 更新设备限制
	subscription.DeviceLimit = req.NewDeviceLimit
	if err := db.Save(&subscription).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新设备数失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "设备数已升级",
		"data":    subscription,
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
			"message": "请求参数错误",
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": fmt.Sprintf("创建支付失败: %v", payErr),
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

package handlers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetUserDashboard 获取用户仪表盘信息
func GetUserDashboard(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	db := database.GetDB()

	// 从数据库重新加载用户信息，确保获取最新的余额
	var freshUser models.User
	if err := db.First(&freshUser, user.ID).Error; err == nil {
		user = &freshUser
	}

	// 取用户等级
	var userLevel *models.UserLevel
	if user.UserLevelID.Valid {
		var lvl models.UserLevel
		if err := db.First(&lvl, user.UserLevelID.Int64).Error; err == nil {
			userLevel = &lvl
		}
	}

	// 获取用户订阅（取最新的有效订阅）
	var subscription models.Subscription
	db.Where("user_id = ?", user.ID).Order("created_at DESC").First(&subscription)

	// 计算剩余天数/到期时间（使用北京时区）
	remainingDays := 0
	expiryDate := "未设置"
	if subscription.ID > 0 && !subscription.ExpireTime.IsZero() {
		now := utils.GetBeijingTime()
		// 将数据库中的时间转换为北京时区
		beijingTime := utils.ToBeijingTime(subscription.ExpireTime)
		diff := beijingTime.Sub(now)
		if diff > 0 {
			// 使用更精确的天数计算：将时间差转换为天数（向上取整）
			days := diff.Hours() / 24.0
			remainingDays = int(days)
			// 如果有小数部分（即使只有1小时），也算作1天
			if days > float64(remainingDays) {
				remainingDays++
			}
		} else {
			remainingDays = 0
		}
		// 格式化为北京时区的日期时间字符串
		expiryDate = beijingTime.Format("2006-01-02 15:04:05")
	}

	// 在线设备数
	var deviceCount int64
	if subscription.ID > 0 {
		db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscription.ID, true).Count(&deviceCount)
	}

	// 订阅链接（统一格式）
	baseURL := utils.GetBuildBaseURL(c.Request, database.GetDB())
	clashURL := ""
	universalURL := ""
	qrcodeURL := ""
	if subscription.ID > 0 && subscription.SubscriptionURL != "" {
		// 猫咪订阅地址（Clash YAML格式）
		clashURL = fmt.Sprintf("%s/api/v1/subscriptions/clash/%s", baseURL, subscription.SubscriptionURL)
		// 通用订阅地址（Base64格式，适用于小火煎、v2ray等）
		universalURL = fmt.Sprintf("%s/api/v1/subscriptions/universal/%s", baseURL, subscription.SubscriptionURL)

		// 生成二维码 URL（sub://格式，包含到期时间）
		encodedURL := base64.StdEncoding.EncodeToString([]byte(universalURL))
		expiryDisplay := expiryDate
		if expiryDisplay == "未设置" {
			expiryDisplay = subscription.SubscriptionURL
		}
		qrcodeURL = fmt.Sprintf("sub://%s#%s", encodedURL, url.QueryEscape(expiryDisplay))
	}

	// 订阅状态
	subStatus := subscription.Status
	if subStatus == "" {
		if subscription.ID > 0 && subscription.IsActive {
			subStatus = "active"
		} else {
			subStatus = "inactive"
		}
	}

	// 组装 user_level
	var userLevelInfo gin.H
	if userLevel != nil {
		userLevelInfo = gin.H{
			"id":              userLevel.ID,
			"name":            userLevel.LevelName,
			"discount_rate":   userLevel.DiscountRate,
			"device_limit":    userLevel.DeviceLimit,
			"color":           userLevel.Color,
			"benefits":        userLevel.Benefits.String,
			"level_order":     userLevel.LevelOrder,
			"min_consumption": userLevel.MinConsumption,
		}
	}

	dashboard := gin.H{
		"username":            user.Username,
		"email":               user.Email,
		"is_verified":         user.IsVerified,
		"is_active":           user.IsActive,
		"is_admin":            user.IsAdmin,
		"balance":             fmt.Sprintf("%.2f", user.Balance),
		"membership":          userLevelInfo["name"],
		"user_level":          userLevelInfo,
		"online_devices":      deviceCount,
		"total_devices":       subscription.DeviceLimit,
		"subscription_url":    subscription.SubscriptionURL,
		"clashUrl":            clashURL,
		"universalUrl":        universalURL, // 通用订阅（Base64格式）
		"qrcodeUrl":           qrcodeURL,    // 通用订阅地址生成的二维码（用于 Shadowrocket 扫码）
		"subscription_status": subStatus,
		"expire_time":         expiryDate,
		"expiryDate":          expiryDate,
		"remaining_days":      remainingDays,
		// 订阅信息
		"subscription": gin.H{
			"status":           subStatus,
			"remaining_days":   remainingDays,
			"expiryDate":       expiryDate,
			"expire_time":      expiryDate,
			"currentDevices":   deviceCount,
			"maxDevices":       subscription.DeviceLimit,
			"subscription_url": subscription.SubscriptionURL,
			"clashUrl":         clashURL,
			"universalUrl":     universalURL, // 通用订阅（Base64格式）
			"qrcodeUrl":        qrcodeURL,    // 通用订阅地址生成的二维码（用于 Shadowrocket 扫码）
		},
	}

	utils.SuccessResponse(c, http.StatusOK, "", dashboard)
}

// GetDashboard 获取管理员仪表盘统计
func GetDashboard(c *gin.Context) {
	db := database.GetDB()

	// 统计总用户数
	var totalUsers int64
	db.Model(&models.User{}).Count(&totalUsers)

	// 统计活跃订阅数（状态为active且is_active为true，并且未过期）
	var activeSubscriptions int64
	now := utils.GetBeijingTime()
	// 查询活跃订阅：is_active为true，状态为active（或空字符串），并且expire_time大于当前时间
	db.Model(&models.Subscription{}).
		Where("is_active = ?", true).
		Where("(status = ? OR status = '' OR status IS NULL)", "active").
		Where("expire_time > ?", now).
		Count(&activeSubscriptions)

	// 统计总订单数
	var totalOrders int64
	db.Model(&models.Order{}).Count(&totalOrders)

	// 统计总收入（使用公共函数）
	totalRevenue := utils.CalculateTotalRevenue(db, "paid")

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"totalUsers":          totalUsers,
		"activeSubscriptions": activeSubscriptions,
		"totalOrders":         totalOrders,
		"totalRevenue":        totalRevenue,
	})
}

// GetRecentUsers 获取最近注册的用户
func GetRecentUsers(c *gin.Context) {
	db := database.GetDB()
	var users []models.User
	db.Order("created_at DESC").Limit(10).Find(&users)

	// 转换为前端需要的格式
	userList := make([]gin.H, 0)
	for _, user := range users {
		// 计算状态
		status := "inactive"
		if user.IsActive {
			status = "active"
		}

		userList = append(userList, gin.H{
			"id":          user.ID,
			"username":    user.Username,
			"email":       user.Email,
			"is_active":   user.IsActive,
			"is_verified": user.IsVerified,
			"status":      status,
			"created_at":  user.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", userList)
}

// GetRecentOrders 获取最近的订单
func GetRecentOrders(c *gin.Context) {
	db := database.GetDB()
	var orders []models.Order
	db.Preload("User").Order("created_at DESC").Limit(10).Find(&orders)

	// 转换为前端需要的格式
	orderList := make([]gin.H, 0)
	for _, order := range orders {
		amount := order.Amount
		if order.FinalAmount.Valid {
			amount = order.FinalAmount.Float64
		}
		orderList = append(orderList, gin.H{
			"id":         order.ID,
			"order_no":   order.OrderNo,
			"user_id":    order.UserID,
			"username":   order.User.Username,
			"amount":     amount,
			"status":     order.Status,
			"created_at": order.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", orderList)
}

// GetAbnormalUsers 获取异常用户（账户禁用、频繁重置、频繁订阅、长期未登录等）
func GetAbnormalUsers(c *gin.Context) {
	db := database.GetDB()
	now := utils.GetBeijingTime()

	// 前端筛选参数
	dateRange := c.QueryArray("date_range[]")
	if len(dateRange) == 0 {
		// 有些客户端会用 date_range 传递
		dateRange = c.QueryArray("date_range")
	}
	// 也支持 start_date 和 end_date 参数
	if len(dateRange) == 0 {
		startDate := c.Query("start_date")
		endDate := c.Query("end_date")
		if startDate != "" && endDate != "" {
			dateRange = []string{startDate, endDate}
		}
	}

	// 如果没有提供日期范围，默认使用本月1号到今天
	var startTime, endTime time.Time
	if len(dateRange) == 2 {
		var err error
		startTime, err = time.Parse("2006-01-02", dateRange[0])
		if err != nil {
			startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		}
		endTime, err = time.Parse("2006-01-02", dateRange[1])
		if err != nil {
			endTime = now
		}
		// 确保结束时间包含当天的23:59:59
		endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 23, 59, 59, 0, endTime.Location())
	} else {
		// 默认：本月1号 00:00:00 到今天 23:59:59
		startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endTime = now
	}

	subscriptionCountFilter := c.DefaultQuery("subscription_count", "10") // 默认10次
	resetCountFilter := c.DefaultQuery("reset_count", "3")                // 默认3次

	// 基础筛选：异常用户包括：
	// 1. 账户被禁用
	// 2. 频繁重置订阅（在时间范围内>=3次，默认）
	// 3. 频繁创建订阅（在时间范围内>=10次，默认）
	// 4. 长期未登录（注册超过1个月且从未登录）
	oneMonthAgo := now.AddDate(0, -1, 0)

	// 构建查询：查找异常用户
	// 使用 OR 条件组合所有异常情况
	// 注意：订阅次数和重置次数是"或"的关系，不是"和"的关系
	var minSub, minReset int
	fmt.Sscanf(subscriptionCountFilter, "%d", &minSub)
	fmt.Sscanf(resetCountFilter, "%d", &minReset)

	// 如果未指定，使用默认值
	if minSub <= 0 {
		minSub = 10
	}
	if minReset <= 0 {
		minReset = 3
	}

	// 构建子查询：在时间范围内统计订阅次数
	subscriptionSubQuery := db.Model(&models.Subscription{}).
		Select("user_id").
		Where("created_at >= ? AND created_at <= ?", startTime, endTime).
		Group("user_id").
		Having("COUNT(*) >= ?", minSub)

	// 构建子查询：在时间范围内统计重置次数
	resetSubQuery := db.Model(&models.SubscriptionReset{}).
		Select("user_id").
		Where("created_at >= ? AND created_at <= ?", startTime, endTime).
		Group("user_id").
		Having("COUNT(*) >= ?", minReset)

	// 构建主查询：订阅次数和重置次数是"或"的关系
	query := db.Model(&models.User{}).
		Where("is_active = ? OR (last_login IS NULL AND created_at < ?) OR id IN (?) OR id IN (?)",
			false, oneMonthAgo, subscriptionSubQuery, resetSubQuery)

	// 时间范围（注册时间）- 如果提供了日期范围，也限制注册时间
	if len(dateRange) == 2 {
		query = query.Where("created_at BETWEEN ? AND ?", startTime, endTime)
	}

	// 获取用户
	var users []models.User
	query.Order("created_at DESC").Limit(200).Find(&users)

	// 使用辅助函数构建异常用户数据（需要传入时间范围用于统计）
	userList := buildAbnormalUserDataWithDateRange(db, users, startTime, endTime, minSub, minReset)
	utils.SuccessResponse(c, http.StatusOK, "", userList)
}

// MarkUserNormal 将用户标记为正常（简单实现：标记已验证且激活）
func MarkUserNormal(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "用户不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取用户失败", err)
		}
		return
	}
	user.IsActive = true
	user.IsVerified = true
	if err := db.Save(&user).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新用户失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "已标记为正常", nil)
}

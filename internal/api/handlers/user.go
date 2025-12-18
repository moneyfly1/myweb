package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cboard-go/internal/core/auth"
	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// getDefaultSubscriptionSettings 从系统设置中获取默认订阅配置
func getDefaultSubscriptionSettings(db *gorm.DB) (deviceLimit int, durationMonths int) {
	// 默认值
	deviceLimit = 3
	durationMonths = 1

	// 从数据库读取配置
	var deviceLimitConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "default_subscription_device_limit", "registration").First(&deviceLimitConfig).Error; err == nil {
		if limit, err := strconv.Atoi(deviceLimitConfig.Value); err == nil && limit > 0 {
			deviceLimit = limit
		}
	}

	var durationConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "default_subscription_duration_months", "registration").First(&durationConfig).Error; err == nil {
		if months, err := strconv.Atoi(durationConfig.Value); err == nil && months > 0 {
			durationMonths = months
		}
	}

	return deviceLimit, durationMonths
}

// createDefaultSubscription 为用户创建默认订阅（如果不存在）
func createDefaultSubscription(db *gorm.DB, userID uint) error {
	// 检查是否已存在订阅
	var existing models.Subscription
	if err := db.Where("user_id = ?", userID).First(&existing).Error; err == nil {
		return nil
	}

	// 从系统设置获取默认配置
	deviceLimit, durationMonths := getDefaultSubscriptionSettings(db)

	// 生成订阅 URL
	subscriptionURL := utils.GenerateSubscriptionURL()

	sub := models.Subscription{
		UserID:          userID,
		SubscriptionURL: subscriptionURL,
		DeviceLimit:     deviceLimit,
		CurrentDevices:  0,
		IsActive:        true,
		Status:          "active",
		ExpireTime:      utils.GetBeijingTime().AddDate(0, durationMonths, 0),
	}

	if err := db.Create(&sub).Error; err != nil {
		return err
	}
	return nil
}

// GetCurrentUser 获取当前用户
func GetCurrentUser(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	// 格式化时间字段
	lastLoginStr := ""
	if user.LastLogin.Valid {
		lastLoginStr = user.LastLogin.Time.Format("2006-01-02 15:04:05")
	}

	// 构建响应数据，确保包含所有必要字段
	responseData := gin.H{
		"id":                  user.ID,
		"username":            user.Username,
		"email":               user.Email,
		"is_active":           user.IsActive,
		"is_verified":         user.IsVerified,
		"is_admin":            user.IsAdmin,
		"created_at":          user.CreatedAt.Format("2006-01-02 15:04:05"),
		"last_login":          lastLoginStr,
		"theme":               user.Theme,
		"language":            user.Language,
		"timezone":            user.Timezone,
		"email_notifications": user.EmailNotifications,
		"notification_types":  user.NotificationTypes,
		"sms_notifications":   user.SMSNotifications,
		"push_notifications":  user.PushNotifications,
		"data_sharing":        user.DataSharing,
		"analytics":           user.Analytics,
		"balance":             user.Balance,
	}

	// 添加昵称（如果存在）
	if user.Nickname.Valid {
		responseData["nickname"] = user.Nickname.String
	}

	// 添加头像（如果存在）
	if user.Avatar.Valid {
		responseData["avatar"] = user.Avatar.String
		responseData["avatar_url"] = user.Avatar.String
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    responseData,
	})
}

// UpdateCurrentUser 更新当前用户
func UpdateCurrentUser(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	var req struct {
		Username string `json:"username"`
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
		Theme    string `json:"theme"`
		Language string `json:"language"`
		Timezone string `json:"timezone"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()

	if req.Username != "" {
		// 检查用户名是否已被其他用户使用
		var existingUser models.User
		if err := db.Where("username = ? AND id != ?", req.Username, user.ID).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "用户名已被使用",
			})
			return
		}
		user.Username = req.Username
	}
	if req.Nickname != "" {
		user.Nickname = database.NullString(req.Nickname)
	} else if req.Nickname == "" {
		// 如果传入空字符串，清空昵称
		user.Nickname = database.NullString("")
	}
	if req.Avatar != "" {
		user.Avatar = database.NullString(req.Avatar)
	}
	if req.Theme != "" {
		user.Theme = req.Theme
	}
	if req.Language != "" {
		user.Language = req.Language
	}
	if req.Timezone != "" {
		user.Timezone = req.Timezone
	}

	if err := db.Save(user).Error; err != nil {
		utils.LogError("UpdateUser: save user failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新失败",
		})
		return
	}

	// 构建响应数据
	responseData := gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"theme":    user.Theme,
		"language": user.Language,
		"timezone": user.Timezone,
	}
	if user.Avatar.Valid {
		responseData["avatar"] = user.Avatar.String
		responseData["avatar_url"] = user.Avatar.String
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "更新成功",
		"data":    responseData,
	})
}

// GetUsers 获取用户列表（管理员）
func GetUsers(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.User{})

	// 分页参数
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
	if size < 1 {
		size = 20
	}

	// 搜索参数
	if keyword := c.Query("keyword"); keyword != "" {
		// 清理和验证搜索关键词，防止SQL注入
		sanitizedKeyword := utils.SanitizeSearchKeyword(keyword)
		if sanitizedKeyword != "" {
			query = query.Where("username LIKE ? OR email LIKE ?", "%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%")
		}
	}

	// 状态筛选
	if status := c.Query("status"); status != "" {
		switch status {
		case "active":
			query = query.Where("is_active = ?", true)
		case "inactive":
			query = query.Where("is_active = ?", false)
		case "verified":
			query = query.Where("is_verified = ?", true)
		case "unverified":
			query = query.Where("is_verified = ?", false)
		case "admin":
			query = query.Where("is_admin = ?", true)
		}
	}

	// 日期范围筛选
	// 支持两种格式：
	// 1. date_range 作为数组参数：date_range[]=2024-01-01&date_range[]=2024-12-31
	// 2. start_date 和 end_date 作为独立参数
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// 如果 date_range 是数组格式
	dateRangeArray := c.QueryArray("date_range[]")
	if len(dateRangeArray) == 0 {
		dateRangeArray = c.QueryArray("date_range")
	}

	if len(dateRangeArray) == 2 {
		startDate = dateRangeArray[0]
		endDate = dateRangeArray[1]
	} else if dateRangeStr := c.Query("date_range"); dateRangeStr != "" {
		// 尝试解析 JSON 数组格式的字符串
		// 前端可能传递类似 "[\"2024-01-01\",\"2024-12-31\"]" 的格式
		// 这里简化处理，假设是逗号分隔的格式
		parts := strings.Split(dateRangeStr, ",")
		if len(parts) == 2 {
			startDate = strings.TrimSpace(parts[0])
			endDate = strings.TrimSpace(parts[1])
		}
	}

	// 应用日期范围筛选
	if startDate != "" && endDate != "" {
		// 解析日期
		startTime, err1 := time.Parse("2006-01-02", startDate)
		endTime, err2 := time.Parse("2006-01-02", endDate)

		if err1 == nil && err2 == nil {
			// 设置开始时间为当天的 00:00:00
			startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, startTime.Location())
			// 设置结束时间为当天的 23:59:59
			endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 23, 59, 59, 999999999, endTime.Location())
			query = query.Where("created_at >= ? AND created_at <= ?", startTime, endTime)
		}
	}

	// 计算总数
	var total int64
	query.Count(&total)

	// 获取用户列表
	var users []models.User
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("created_at DESC").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取用户列表失败",
		})
		return
	}

	// 转换为前端需要的格式
	userList := make([]gin.H, 0)
	for _, user := range users {
		// 获取用户订阅
		var subscription models.Subscription
		_ = db.Where("user_id = ?", user.ID).First(&subscription) // 忽略错误，如果没有订阅就使用默认值

		// 计算在线设备数
		var onlineDevices int64
		if subscription.ID > 0 {
			db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscription.ID, true).Count(&onlineDevices)
		}

		// 计算状态
		status := "inactive"
		if user.IsActive {
			status = "active"
		}

		// 计算订阅到期信息
		var subscriptionInfo gin.H
		if subscription.ID > 0 {
			now := utils.GetBeijingTime()
			daysUntilExpire := 0
			isExpired := false
			if !subscription.ExpireTime.IsZero() {
				diff := subscription.ExpireTime.Sub(now)
				if diff > 0 {
					daysUntilExpire = int(diff.Hours() / 24)
				} else {
					isExpired = true
				}
			}
			subscriptionInfo = gin.H{
				"id":                subscription.ID,
				"device_limit":      subscription.DeviceLimit,
				"status":            subscription.Status,
				"is_active":         subscription.IsActive,
				"expire_time":       subscription.ExpireTime.Format("2006-01-02 15:04:05"),
				"days_until_expire": daysUntilExpire,
				"is_expired":        isExpired,
			}
		} else {
			subscriptionInfo = nil
		}

		userList = append(userList, gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"avatar": func() string {
				if user.Avatar.Valid {
					return user.Avatar.String
				}
				return ""
			}(),
			"is_active":      user.IsActive,
			"is_verified":    user.IsVerified,
			"is_admin":       user.IsAdmin,
			"balance":        user.Balance,
			"status":         status,
			"online_devices": onlineDevices,
			"subscription":   subscriptionInfo,
			"created_at":     user.CreatedAt.Format("2006-01-02 15:04:05"),
			"last_login": func() string {
				if user.LastLogin.Valid {
					return user.LastLogin.Time.Format("2006-01-02 15:04:05")
				}
				return ""
			}(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"users": userList,
			"total": total,
			"page":  page,
			"size":  size,
		},
	})
}

// GetUser 获取单个用户（管理员）
func GetUser(c *gin.Context) {
	id := c.Param("id")

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "用户不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}

// GetUserDetails 获取用户详细信息（包含订阅、订单、充值记录、活动记录）
func GetUserDetails(c *gin.Context) {
	id := c.Param("id")

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "用户不存在",
		})
		return
	}

	// 获取用户订阅
	var subscriptions []models.Subscription
	db.Where("user_id = ?", user.ID).Find(&subscriptions)

	// 获取用户订单
	var orders []models.Order
	db.Where("user_id = ?", user.ID).Order("created_at DESC").Limit(50).Find(&orders)

	// 获取充值记录
	var rechargeRecords []models.RechargeRecord
	db.Where("user_id = ?", user.ID).Order("created_at DESC").Limit(50).Find(&rechargeRecords)

	// 获取最近活动
	var activities []models.UserActivity
	db.Where("user_id = ?", user.ID).Order("created_at DESC").Limit(50).Find(&activities)

	// 统计信息
	var totalSpent float64
	db.Model(&models.Order{}).Where("user_id = ? AND status = ?", user.ID, "paid").
		Select("COALESCE(SUM(final_amount), 0)").Scan(&totalSpent)
	if totalSpent == 0 {
		db.Model(&models.Order{}).Where("user_id = ? AND status = ?", user.ID, "paid").
			Select("COALESCE(SUM(amount), 0)").Scan(&totalSpent)
	}

	var totalResets int64
	db.Model(&models.UserActivity{}).Where("user_id = ? AND activity_type = ?", user.ID, "subscription_reset").Count(&totalResets)

	var recentResets30d int64
	thirtyDaysAgo := utils.GetBeijingTime().AddDate(0, 0, -30)
	db.Model(&models.UserActivity{}).Where("user_id = ? AND activity_type = ? AND created_at >= ?", user.ID, "subscription_reset", thirtyDaysAgo).Count(&recentResets30d)

	// 格式化订阅列表
	subscriptionList := make([]gin.H, 0)
	var totalAppleCount int64
	var totalClashCount int64

	for _, sub := range subscriptions {
		// 统计该订阅的订阅次数
		var appleCount int64
		db.Model(&models.Device{}).Where("subscription_id = ? AND subscription_type IN ?", sub.ID, []string{"v2ray", "ssr"}).Count(&appleCount)

		var clashCount int64
		db.Model(&models.Device{}).Where("subscription_id = ? AND subscription_type = ?", sub.ID, "clash").Count(&clashCount)

		totalAppleCount += appleCount
		totalClashCount += clashCount

		subscriptionList = append(subscriptionList, gin.H{
			"id":               sub.ID,
			"subscription_url": sub.SubscriptionURL,
			"device_limit":     sub.DeviceLimit,
			"current_devices":  sub.CurrentDevices,
			"is_active":        sub.IsActive,
			"status":           sub.Status,
			"expire_time":      sub.ExpireTime.Format("2006-01-02 15:04:05"),
			"created_at":       sub.CreatedAt.Format("2006-01-02 15:04:05"),
			"apple_count":      appleCount,
			"clash_count":      clashCount,
		})
	}

	// 格式化订单列表
	orderList := make([]gin.H, 0)
	for _, order := range orders {
		amount := order.Amount
		if order.FinalAmount.Valid {
			amount = order.FinalAmount.Float64
		}
		paymentMethod := ""
		if order.PaymentMethodName.Valid {
			paymentMethod = order.PaymentMethodName.String
		}
		orderList = append(orderList, gin.H{
			"id":                  order.ID,
			"order_no":            order.OrderNo,
			"amount":              amount,
			"status":              order.Status,
			"payment_method":      paymentMethod,
			"payment_method_name": paymentMethod, // 兼容前端
			"payment_time": func() string {
				if order.PaymentTime.Valid {
					return order.PaymentTime.Time.Format("2006-01-02 15:04:05")
				}
				return ""
			}(),
			"package_name": func() string {
				if order.PackageID > 0 {
					var pkg models.Package
					if db.First(&pkg, order.PackageID).Error == nil {
						return pkg.Name
					}
				}
				return "未知"
			}(),
			"created_at": order.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	// 格式化充值记录
	rechargeList := make([]gin.H, 0)
	for _, record := range rechargeRecords {
		rechargeList = append(rechargeList, gin.H{
			"id":             record.ID,
			"order_no":       record.OrderNo,
			"amount":         record.Amount,
			"status":         record.Status,
			"payment_method": record.PaymentMethod.String,
			"ip_address":     record.IPAddress.String,
			"paid_at": func() string {
				if record.PaidAt.Valid {
					return record.PaidAt.Time.Format("2006-01-02 15:04:05")
				}
				return ""
			}(),
			"created_at": record.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	// 格式化活动记录
	activityList := make([]gin.H, 0)
	for _, activity := range activities {
		activityList = append(activityList, gin.H{
			"id":            activity.ID,
			"activity_type": activity.ActivityType,
			"description":   activity.Description.String,
			"ip_address":    activity.IPAddress.String,
			"created_at":    activity.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	// 获取用户设备列表（用于生成 UA 记录）
	var devices []models.Device
	db.Where("user_id = ?", user.ID).Order("created_at DESC").Find(&devices)

	// 生成 UA 记录（从设备记录中提取）
	uaRecords := make([]gin.H, 0)
	for _, device := range devices {
		getStringValue := func(ptr *string) string {
			if ptr != nil {
				return *ptr
			}
			return ""
		}

		uaRecords = append(uaRecords, gin.H{
			"id":          device.ID,
			"user_agent":  getStringValue(device.UserAgent),
			"device_type": getStringValue(device.DeviceType),
			"device_name": getStringValue(device.DeviceName),
			"ip_address":  getStringValue(device.IPAddress),
			"os_name":     getStringValue(device.OSName),
			"os_version":  getStringValue(device.OSVersion),
			"created_at":  device.CreatedAt.Format("2006-01-02 15:04:05"),
			"last_access": device.LastAccess.Format("2006-01-02 15:04:05"),
		})
	}

	// 构建响应
	response := gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"user_info": gin.H{
			"id":          user.ID,
			"username":    user.Username,
			"email":       user.Email,
			"is_active":   user.IsActive,
			"is_verified": user.IsVerified,
			"is_admin":    user.IsAdmin,
			"balance":     user.Balance,
			"created_at":  user.CreatedAt.Format("2006-01-02 15:04:05"),
			"last_login": func() string {
				if user.LastLogin.Valid {
					return user.LastLogin.Time.Format("2006-01-02 15:04:05")
				}
				return ""
			}(),
		},
		"subscriptions":     subscriptionList,
		"orders":            orderList,
		"recharge_records":  rechargeList,
		"recent_activities": activityList,
		"ua_records":        uaRecords, // 添加 UA 记录
		"subscription_resets": func() []gin.H {
			// 获取订阅重置记录
			var resets []models.SubscriptionReset
			db.Where("user_id = ?", user.ID).Order("created_at DESC").Limit(50).Find(&resets)
			resetList := make([]gin.H, 0)
			for _, reset := range resets {
				resetData := gin.H{
					"id":                  reset.ID,
					"subscription_id":     reset.SubscriptionID,
					"reset_type":          reset.ResetType,
					"reason":              reset.Reason,
					"device_count_before": reset.DeviceCountBefore,
					"device_count_after":  reset.DeviceCountAfter,
					"created_at":          reset.CreatedAt.Format("2006-01-02 15:04:05"),
				}
				// 添加旧订阅地址和新订阅地址
				if reset.OldSubscriptionURL != nil {
					resetData["old_subscription_url"] = *reset.OldSubscriptionURL
				}
				if reset.NewSubscriptionURL != nil {
					resetData["new_subscription_url"] = *reset.NewSubscriptionURL
				}
				if reset.ResetBy != nil {
					resetData["reset_by"] = *reset.ResetBy
				}
				resetList = append(resetList, resetData)
			}
			return resetList
		}(),
		"statistics": gin.H{
			"total_spent":         totalSpent,
			"total_resets":        totalResets,
			"recent_resets_30d":   recentResets30d,
			"total_subscriptions": len(subscriptions),
			"subscription_count":  len(subscriptions),
		},
		// 添加订阅次数统计（兼容前端字段名）
		"apple_count": totalAppleCount,
		"clash_count": totalClashCount,
		"v2ray_count": totalAppleCount, // 兼容字段
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// CreateUser 创建用户（管理员）
func CreateUser(c *gin.Context) {
	var req struct {
		Username    string  `json:"username" binding:"required"`
		Email       string  `json:"email" binding:"required,email"`
		Password    string  `json:"password" binding:"required,min=8"`
		IsActive    bool    `json:"is_active"`
		IsVerified  bool    `json:"is_verified"`
		IsAdmin     bool    `json:"is_admin"`
		Balance     float64 `json:"balance"`
		DeviceLimit int     `json:"device_limit"` // 设备限制
		ExpireTime  string  `json:"expire_time"`  // 到期时间，格式：YYYY-MM-DDTHH:mm:ss
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// 不向客户端返回详细错误信息，防止信息泄露
		utils.LogError("CreateUser: bind request", err, nil)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误，请检查输入格式",
		})
		return
	}

	db := database.GetDB()

	// 检查用户是否已存在
	var existingUser models.User
	if err := db.Where("email = ? OR username = ?", req.Email, req.Username).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "邮箱或用户名已存在",
		})
		return
	}

	// 验证密码强度
	valid, msg := auth.ValidatePasswordStrength(req.Password, 8)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": msg,
		})
		return
	}

	// 哈希密码
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "密码加密失败",
		})
		return
	}

	// 创建用户
	user := models.User{
		Username:   req.Username,
		Email:      req.Email,
		Password:   hashedPassword,
		IsActive:   req.IsActive,
		IsVerified: req.IsVerified,
		IsAdmin:    req.IsAdmin,
		Balance:    req.Balance,
	}

	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建用户失败",
		})
		return
	}

	// 创建订阅（使用请求中的 device_limit 和 expire_time，或使用系统默认值）
	deviceLimit := req.DeviceLimit
	defaultDeviceLimit, defaultDurationMonths := getDefaultSubscriptionSettings(db)
	if deviceLimit == 0 {
		deviceLimit = defaultDeviceLimit
	}

	var expireTime time.Time
	if req.ExpireTime != "" {
		// 解析时间字符串
		parsedTime, err := time.Parse("2006-01-02T15:04:05", req.ExpireTime)
		if err != nil {
			// 尝试其他格式
			parsedTime, err = time.Parse("2006-01-02 15:04:05", req.ExpireTime)
			if err != nil {
				// 使用系统默认值
				expireTime = utils.GetBeijingTime().AddDate(0, defaultDurationMonths, 0)
			} else {
				expireTime = parsedTime
			}
		} else {
			expireTime = parsedTime
		}
	} else {
		// 使用系统默认值
		expireTime = utils.GetBeijingTime().AddDate(0, defaultDurationMonths, 0)
	}

	subscription := models.Subscription{
		UserID:          user.ID,
		SubscriptionURL: utils.GenerateSubscriptionURL(),
		DeviceLimit:     deviceLimit,
		CurrentDevices:  0,
		IsActive:        true,
		Status:          "active",
		ExpireTime:      expireTime,
	}

	if err := db.Create(&subscription).Error; err != nil {
		// 订阅创建失败不影响用户创建，但记录错误
		if utils.AppLogger != nil {
			utils.AppLogger.Error("创建用户订阅失败: %v", err)
		}
	}

	// 记录审计日志
	utils.CreateAuditLogSimple(c, "create_user", "user", user.ID,
		fmt.Sprintf("管理员创建用户: %s (%s), 管理员权限: %v", user.Username, user.Email, user.IsAdmin))

	// 发送管理员通知
	go func() {
		notificationService := notification.NewNotificationService()
		adminUser, _ := middleware.GetCurrentUser(c)
		createdBy := "系统"
		if adminUser != nil {
			createdBy = adminUser.Username
		}
		createTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
		_ = notificationService.SendAdminNotification("user_created", map[string]interface{}{
			"username":    user.Username,
			"email":       user.Email,
			"created_by":  createdBy,
			"create_time": createTime,
		})
	}()

	utils.SetResponseStatus(c, http.StatusCreated)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "创建成功",
		"data":    user,
	})
}

// UpdateUser 更新用户（管理员）
func UpdateUser(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Username   string   `json:"username"`
		Email      string   `json:"email"`
		IsActive   *bool    `json:"is_active"`
		IsVerified *bool    `json:"is_verified"`
		IsAdmin    *bool    `json:"is_admin"`
		Balance    *float64 `json:"balance"`
		Password   string   `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "用户不存在",
		})
		return
	}

	// 保存更新前的数据用于审计日志
	beforeData := map[string]interface{}{
		"username":    user.Username,
		"email":       user.Email,
		"is_active":   user.IsActive,
		"is_verified": user.IsVerified,
		"is_admin":    user.IsAdmin,
		"balance":     user.Balance,
	}

	// 更新字段
	if req.Username != "" {
		// 检查用户名是否已被其他用户使用
		var existing models.User
		if err := db.Where("username = ? AND id != ?", req.Username, id).First(&existing).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "用户名已被使用",
			})
			return
		}
		user.Username = req.Username
	}

	if req.Email != "" {
		// 检查邮箱是否已被其他用户使用
		var existing models.User
		if err := db.Where("email = ? AND id != ?", req.Email, id).First(&existing).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "邮箱已被使用",
			})
			return
		}
		user.Email = req.Email
	}

	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if req.IsVerified != nil {
		user.IsVerified = *req.IsVerified
	}
	if req.IsAdmin != nil {
		user.IsAdmin = *req.IsAdmin
	}
	if req.Balance != nil {
		user.Balance = *req.Balance
	}
	if req.Password != "" {
		hashedPassword, err := auth.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "密码加密失败",
			})
			return
		}
		user.Password = hashedPassword
	}

	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新失败",
		})
		return
	}

	// 保存更新后的数据用于审计日志
	afterData := map[string]interface{}{
		"username":    user.Username,
		"email":       user.Email,
		"is_active":   user.IsActive,
		"is_verified": user.IsVerified,
		"is_admin":    user.IsAdmin,
		"balance":     user.Balance,
	}

	// 记录审计日志
	description := fmt.Sprintf("管理员更新用户: %s (%s)", user.Username, user.Email)
	if req.Password != "" {
		description += " (包含密码重置)"
	}
	utils.CreateAuditLogWithData(c, "update_user", "user", user.ID, description, beforeData, afterData)

	utils.SetResponseStatus(c, http.StatusOK)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "更新成功",
		"data":    user,
	})
}

// DeleteUser 删除用户（管理员）
func DeleteUser(c *gin.Context) {
	id := c.Param("id")

	// 验证ID是否有效
	if id == "" || id == "0" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的用户ID",
		})
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "用户不存在",
		})
		return
	}

	// 保存用户信息用于审计日志
	userData := map[string]interface{}{
		"id":          user.ID,
		"username":    user.Username,
		"email":       user.Email,
		"is_admin":    user.IsAdmin,
		"is_active":   user.IsActive,
		"is_verified": user.IsVerified,
	}

	// 检查是否是管理员
	if user.IsAdmin {
		// 检查是否还有其他管理员
		var adminCount int64
		db.Model(&models.User{}).Where("is_admin = ? AND id != ?", true, id).Count(&adminCount)
		if adminCount == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "不能删除最后一个管理员",
			})
			return
		}
	}

	// 开始事务，删除用户的所有相关数据
	tx := db.Begin()

	// 1. 删除用户的订阅（同时会通过外键约束删除相关设备）
	if err := tx.Where("user_id = ?", user.ID).Delete(&models.Subscription{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete subscriptions failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户订阅失败",
		})
		return
	}

	// 2. 删除用户的设备（通过 subscription_id 关联的设备）
	if err := tx.Where("subscription_id IN (SELECT id FROM subscriptions WHERE user_id = ?)", user.ID).Delete(&models.Device{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete devices by subscription failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户设备失败",
		})
		return
	}

	// 3. 删除用户直接关联的设备（通过 user_id）
	if err := tx.Where("user_id = ?", user.ID).Delete(&models.Device{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete devices by user_id failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户设备失败",
		})
		return
	}

	// 4. 删除用户的订阅重置记录
	if err := tx.Where("user_id = ?", user.ID).Delete(&models.SubscriptionReset{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete subscription resets failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户订阅重置记录失败",
		})
		return
	}

	// 5. 删除用户的订单
	if err := tx.Where("user_id = ?", user.ID).Delete(&models.Order{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete orders failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户订单失败",
		})
		return
	}

	// 6. 删除用户的支付交易记录
	if err := tx.Where("user_id = ?", user.ID).Delete(&models.PaymentTransaction{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete payment transactions failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户支付记录失败",
		})
		return
	}

	// 7. 删除用户的充值记录
	if err := tx.Where("user_id = ?", user.ID).Delete(&models.RechargeRecord{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete recharge records failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户充值记录失败",
		})
		return
	}

	// 8. 删除用户的工单回复
	if err := tx.Where("user_id = ?", user.ID).Delete(&models.TicketReply{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete ticket replies failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户工单回复失败",
		})
		return
	}

	// 9. 删除用户的工单
	if err := tx.Where("user_id = ?", user.ID).Delete(&models.Ticket{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete tickets failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户工单失败",
		})
		return
	}

	// 10. 删除用户的通知
	if err := tx.Where("user_id = ?", user.ID).Delete(&models.Notification{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete notifications failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户通知失败",
		})
		return
	}

	// 11. 删除用户的活动记录
	if err := tx.Where("user_id = ?", user.ID).Delete(&models.UserActivity{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete user activities failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户活动记录失败",
		})
		return
	}

	// 12. 删除用户的登录历史
	if err := tx.Where("user_id = ?", user.ID).Delete(&models.LoginHistory{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete login history failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户登录历史失败",
		})
		return
	}

	// 13. 删除用户的邀请码（设置为禁用状态，而不是删除，因为可能已被使用）
	// 如果邀请码已被使用，只禁用；如果未使用，则删除
	if err := tx.Model(&models.InviteCode{}).Where("user_id = ? AND used_count = 0", user.ID).Delete(&models.InviteCode{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete invite codes failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户邀请码失败",
		})
		return
	}
	// 禁用已使用的邀请码
	if err := tx.Model(&models.InviteCode{}).Where("user_id = ? AND used_count > 0", user.ID).Update("is_active", false).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: disable invite codes failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "禁用用户邀请码失败",
		})
		return
	}

	// 14. 删除用户作为邀请人的邀请关系
	if err := tx.Where("inviter_id = ?", user.ID).Delete(&models.InviteRelation{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete invite relations as inviter failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户邀请关系失败",
		})
		return
	}

	// 15. 删除用户作为被邀请人的邀请关系
	if err := tx.Where("invitee_id = ?", user.ID).Delete(&models.InviteRelation{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete invite relations as invitee failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户被邀请关系失败",
		})
		return
	}

	// 16. 删除用户的优惠券使用记录（如果有 CouponUsage 表）
	// 注意：这里假设有 CouponUsage 表，如果没有可以忽略

	// 17. 最后删除用户本身
	if err := tx.Delete(&user).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete user failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户失败",
		})
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		utils.LogError("DeleteUser: commit transaction failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除操作失败",
		})
		return
	}

	// 记录审计日志
	utils.CreateAuditLogWithData(c, "delete_user", "user", user.ID,
		fmt.Sprintf("管理员删除用户: %s (%s)", user.Username, user.Email), userData, nil)

	// 发送账户删除确认邮件（在删除前发送）
	go func() {
		emailService := email.NewEmailService()
		templateBuilder := email.NewEmailTemplateBuilder()
		deletionDate := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
		reason := "管理员删除"
		dataRetentionPeriod := "30天"
		content := templateBuilder.GetAccountDeletionTemplate(user.Username, deletionDate, reason, dataRetentionPeriod)
		subject := "账号删除确认"
		_ = emailService.QueueEmail(user.Email, subject, content, "account_deletion")
	}()

	utils.SetResponseStatus(c, http.StatusOK)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "用户及其所有相关数据已成功删除",
	})
}

// LoginAsUser 管理员以用户身份登录
func LoginAsUser(c *gin.Context) {
	userID := c.Param("id")
	db := database.GetDB()

	var targetUser models.User
	if err := db.First(&targetUser, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "用户不存在",
		})
		return
	}

	// 生成令牌
	accessToken, err := utils.CreateAccessToken(targetUser.ID, targetUser.Email, targetUser.IsAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "生成令牌失败",
		})
		return
	}

	refreshToken, err := utils.CreateRefreshToken(targetUser.ID, targetUser.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "生成刷新令牌失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "登录成功",
		"data": gin.H{
			"token":         accessToken, // 兼容前端期望的字段名
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"token_type":    "bearer",
			"user": gin.H{
				"id":       targetUser.ID,
				"username": targetUser.Username,
				"email":    targetUser.Email,
				"is_admin": targetUser.IsAdmin,
			},
		},
	})
}

// UpdateUserStatus 更新用户状态
func UpdateUserStatus(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status     string `json:"status"`    // "active", "inactive", "disabled"
		IsActive   *bool  `json:"is_active"` // 兼容旧格式
		IsVerified *bool  `json:"is_verified"`
		IsAdmin    *bool  `json:"is_admin"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "用户不存在",
		})
		return
	}

	// 优先使用 status 字段，如果没有则使用 is_active
	if req.Status != "" {
		switch req.Status {
		case "active":
			user.IsActive = true
		case "inactive", "disabled":
			user.IsActive = false
		}
	} else if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if req.IsVerified != nil {
		user.IsVerified = *req.IsVerified
	}
	if req.IsAdmin != nil {
		user.IsAdmin = *req.IsAdmin
	}

	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新用户状态失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "用户状态已更新",
		"data":    user,
	})
}

// UnlockUserLogin 解锁用户登录
func UnlockUserLogin(c *gin.Context) {
	id := c.Param("id")

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "用户不存在",
		})
		return
	}

	// 清除该用户的所有登录失败记录
	result := db.Where("username = ? OR username = ?", user.Username, user.Email).
		Where("success = ?", false).
		Delete(&models.LoginAttempt{})

	// 确保用户是激活状态
	user.IsActive = true

	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "解锁用户失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("用户已解锁，清除了 %d 条登录失败记录", result.RowsAffected),
	})
}

// BatchDeleteUsers 批量删除用户
func BatchDeleteUsers(c *gin.Context) {
	var req struct {
		UserIDs []uint `json:"user_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	if len(req.UserIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请选择要删除的用户",
		})
		return
	}

	db := database.GetDB()

	// 检查是否包含管理员用户
	var adminUsers []models.User
	if err := db.Where("id IN ? AND is_admin = ?", req.UserIDs, true).Find(&adminUsers).Error; err == nil && len(adminUsers) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "不能删除管理员用户",
		})
		return
	}

	// 开始事务
	tx := db.Begin()

	// 删除用户的订阅
	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.Subscription{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户订阅失败",
		})
		return
	}

	// 删除用户的设备（通过 subscription_id 关联的设备）
	if err := tx.Where("subscription_id IN (SELECT id FROM subscriptions WHERE user_id IN ?)", req.UserIDs).Delete(&models.Device{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户设备失败",
		})
		return
	}

	// 删除用户直接关联的设备（通过 user_id）
	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.Device{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户设备失败",
		})
		return
	}

	// 删除用户的订阅重置记录
	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.SubscriptionReset{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户订阅重置记录失败",
		})
		return
	}

	// 删除用户的订单
	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.Order{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户订单失败",
		})
		return
	}

	// 删除用户的支付交易记录
	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.PaymentTransaction{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户支付记录失败",
		})
		return
	}

	// 删除用户的充值记录
	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.RechargeRecord{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户充值记录失败",
		})
		return
	}

	// 删除用户的工单回复
	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.TicketReply{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户工单回复失败",
		})
		return
	}

	// 删除用户的工单
	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.Ticket{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户工单失败",
		})
		return
	}

	// 删除用户的通知
	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.Notification{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户通知失败",
		})
		return
	}

	// 删除用户的活动记录
	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.UserActivity{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户活动记录失败",
		})
		return
	}

	// 删除用户的登录历史
	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.LoginHistory{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户登录历史失败",
		})
		return
	}

	// 删除用户的邀请码（未使用的删除，已使用的禁用）
	if err := tx.Where("user_id IN ? AND used_count = 0", req.UserIDs).Delete(&models.InviteCode{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户邀请码失败",
		})
		return
	}
	// 禁用已使用的邀请码
	if err := tx.Model(&models.InviteCode{}).Where("user_id IN ? AND used_count > 0", req.UserIDs).Update("is_active", false).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "禁用用户邀请码失败",
		})
		return
	}

	// 删除用户作为邀请人的邀请关系
	if err := tx.Where("inviter_id IN ?", req.UserIDs).Delete(&models.InviteRelation{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户邀请关系失败",
		})
		return
	}

	// 删除用户作为被邀请人的邀请关系
	if err := tx.Where("invitee_id IN ?", req.UserIDs).Delete(&models.InviteRelation{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户被邀请关系失败",
		})
		return
	}

	// 删除用户
	if err := tx.Where("id IN ?", req.UserIDs).Delete(&models.User{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除用户失败",
		})
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除操作失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("成功删除 %d 个用户", len(req.UserIDs)),
	})
}

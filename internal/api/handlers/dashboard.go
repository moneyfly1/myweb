package handlers

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetUserDashboard 获取用户仪表盘信息
func GetUserDashboard(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()

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
	baseURL := buildBaseURL(c)
	clashURL := ""
	universalURL := ""
	qrcodeURL := ""
	if subscription.ID > 0 && subscription.SubscriptionURL != "" {
		// 使用时间戳避免缓存
		timestamp := fmt.Sprintf("%d", utils.GetBeijingTime().Unix())
		// 猫咪订阅地址（Clash YAML格式）
		clashURL = fmt.Sprintf("%s/api/v1/subscriptions/clash/%s?t=%s", baseURL, subscription.SubscriptionURL, timestamp)
		// 通用订阅地址（Base64格式，适用于小火煎、v2ray等）
		universalURL = fmt.Sprintf("%s/api/v1/subscriptions/universal/%s?t=%s", baseURL, subscription.SubscriptionURL, timestamp)

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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dashboard,
	})
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

	// 统计总收入（使用final_amount，如果为NULL则使用amount）
	var totalRevenue float64
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
			"totalUsers":          totalUsers,
			"activeSubscriptions": activeSubscriptions,
			"totalOrders":         totalOrders,
			"totalRevenue":        totalRevenue,
		},
	})
}

// GetAdminStats 获取管理员统计（别名，兼容前端 /admin/stats）
func GetAdminStats(c *gin.Context) {
	GetDashboard(c)
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    userList,
	})
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    orderList,
	})
}

// GetAbnormalUsers 获取异常用户（未验证、长期未登录、频繁重置等）
func GetAbnormalUsers(c *gin.Context) {
	db := database.GetDB()
	// 前端筛选参数
	dateRange := c.QueryArray("date_range[]")
	if len(dateRange) == 0 {
		// 有些客户端会用 date_range 传递
		dateRange = c.QueryArray("date_range")
	}
	subscriptionCountFilter := c.DefaultQuery("subscription_count", "")
	resetCountFilter := c.DefaultQuery("reset_count", "")

	// 基础筛选：未验证或长期未登录
	query := db.Model(&models.User{}).
		Where("is_verified = ? OR (last_login IS NULL AND created_at < ?)", false, utils.GetBeijingTime().AddDate(0, -1, 0))

	// 时间范围（注册时间）
	if len(dateRange) == 2 {
		start := dateRange[0]
		end := dateRange[1]
		query = query.Where("created_at BETWEEN ? AND ?", start, end)
	}

	// 订阅次数过滤
	if subscriptionCountFilter != "" {
		var minSub int
		fmt.Sscanf(subscriptionCountFilter, "%d", &minSub)
		if minSub > 0 {
			query = query.Where("id IN (SELECT user_id FROM subscriptions GROUP BY user_id HAVING COUNT(*) >= ?)", minSub)
		}
	}

	// 重置次数过滤
	if resetCountFilter != "" {
		var minReset int
		fmt.Sscanf(resetCountFilter, "%d", &minReset)
		if minReset > 0 {
			query = query.Where("id IN (SELECT user_id FROM subscription_resets GROUP BY user_id HAVING COUNT(*) >= ?)", minReset)
		}
	}

	// 获取用户
	var users []models.User
	query.Order("created_at DESC").Limit(200).Find(&users)

	// 转换为前端需要的格式
	userList := make([]gin.H, 0)
	for _, user := range users {
		lastLogin := "从未登录"
		if user.LastLogin.Valid {
			lastLogin = user.LastLogin.Time.Format("2006-01-02 15:04:05")
		}

		// 计算状态
		status := "inactive"
		if user.IsActive {
			status = "active"
		}

		// 统计用户异常行为
		var resetCount int64
		db.Model(&models.UserActivity{}).Where("user_id = ? AND activity_type = ?", user.ID, "subscription_reset").Count(&resetCount)

		var subscriptionCount int64
		db.Model(&models.Subscription{}).Where("user_id = ?", user.ID).Count(&subscriptionCount)

		// 判断异常类型和次数
		abnormalType := "multiple_abnormal"
		abnormalCount := 0
		description := ""

		if resetCount >= 5 {
			abnormalType = "frequent_reset"
			abnormalCount = int(resetCount)
			description = fmt.Sprintf("频繁重置订阅 %d 次", resetCount)
		} else if subscriptionCount >= 10 {
			abnormalType = "frequent_subscription"
			abnormalCount = int(subscriptionCount)
			description = fmt.Sprintf("频繁创建订阅 %d 次", subscriptionCount)
		} else if !user.IsVerified {
			abnormalType = "unverified"
			abnormalCount = 1
			description = "未验证邮箱"
		} else {
			abnormalType = "inactive"
			abnormalCount = 1
			description = "长期未登录"
		}

		// 获取最后活动时间
		var lastActivity string
		var lastActivityRecord models.UserActivity
		if err := db.Where("user_id = ?", user.ID).Order("created_at DESC").First(&lastActivityRecord).Error; err != nil {
			// 如果没有活动记录，使用创建时间
			if err == gorm.ErrRecordNotFound {
				lastActivity = user.CreatedAt.Format("2006-01-02 15:04:05")
			} else {
				// 其他错误也使用创建时间
				lastActivity = user.CreatedAt.Format("2006-01-02 15:04:05")
			}
		} else {
			lastActivity = lastActivityRecord.CreatedAt.Format("2006-01-02 15:04:05")
		}

		userList = append(userList, gin.H{
			"id":                 user.ID,
			"user_id":            user.ID,
			"username":           user.Username,
			"email":              user.Email,
			"is_active":          user.IsActive,
			"is_verified":        user.IsVerified,
			"status":             status,
			"last_login":         lastLogin,
			"created_at":         user.CreatedAt.Format("2006-01-02 15:04:05"),
			"abnormal_type":      abnormalType,
			"abnormal_count":     abnormalCount,
			"reset_count":        resetCount,
			"subscription_count": subscriptionCount,
			"description":        description,
			"last_activity":      lastActivity,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    userList,
	})
}

// MarkUserNormal 将用户标记为正常（简单实现：标记已验证且激活）
func MarkUserNormal(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户不存在"})
		return
	}
	user.IsActive = true
	user.IsVerified = true
	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "更新用户失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已标记为正常"})
}

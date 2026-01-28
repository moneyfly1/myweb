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

func GetUserDashboard(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	db := database.GetDB()

	var freshUser models.User
	if err := db.First(&freshUser, user.ID).Error; err == nil {
		user = &freshUser
	}

	var userLevel *models.UserLevel
	if user.UserLevelID.Valid {
		var lvl models.UserLevel
		if err := db.First(&lvl, user.UserLevelID.Int64).Error; err == nil {
			userLevel = &lvl
		}
	}

	var subscription models.Subscription
	db.Where("user_id = ?", user.ID).Order("created_at DESC").First(&subscription)

	remainingDays := 0
	expiryDate := "未设置"
	if subscription.ID > 0 && !subscription.ExpireTime.IsZero() {
		now := utils.GetBeijingTime()
		beijingTime := utils.ToBeijingTime(subscription.ExpireTime)
		diff := beijingTime.Sub(now)
		if diff > 0 {
			days := diff.Hours() / 24.0
			remainingDays = int(days)
			if days > float64(remainingDays) {
				remainingDays++
			}
		} else {
			remainingDays = 0
		}
		expiryDate = beijingTime.Format("2006-01-02 15:04:05")
	}

	var deviceCount int64
	if subscription.ID > 0 {
		db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscription.ID, true).Count(&deviceCount)
	}

	baseURL := utils.GetBuildBaseURL(c.Request, database.GetDB())
	clashURL := ""
	universalURL := ""
	qrcodeURL := ""
	if subscription.ID > 0 && subscription.SubscriptionURL != "" {
		clashURL = fmt.Sprintf("%s/api/v1/subscriptions/clash/%s", baseURL, subscription.SubscriptionURL)
		universalURL = fmt.Sprintf("%s/api/v1/subscriptions/universal/%s", baseURL, subscription.SubscriptionURL)

		encodedURL := base64.StdEncoding.EncodeToString([]byte(universalURL))
		expiryDisplay := expiryDate
		if expiryDisplay == "未设置" {
			expiryDisplay = subscription.SubscriptionURL
		}
		qrcodeURL = fmt.Sprintf("sub://%s#%s", encodedURL, url.QueryEscape(expiryDisplay))
	}

	subStatus := subscription.Status
	if subStatus == "" {
		if subscription.ID > 0 && subscription.IsActive {
			subStatus = "active"
		} else {
			subStatus = "inactive"
		}
	}

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

	var orderCount int64
	var totalSpent float64
	db.Model(&models.Order{}).Where("user_id = ? AND status = ?", user.ID, "paid").Count(&orderCount)
	db.Model(&models.Order{}).
		Where("user_id = ? AND status = ?", user.ID, "paid").
		Select("COALESCE(SUM(final_amount), SUM(amount), 0)").
		Scan(&totalSpent)

	var announcementConfig models.SystemConfig
	var announcementEnabled bool
	var announcementContent string
	if err := db.Where("key = ? AND category = ?", "announcement_enabled", "system").First(&announcementConfig).Error; err == nil {
		announcementEnabled = announcementConfig.Value == "true"
	}
	if announcementEnabled {
		var contentConfig models.SystemConfig
		if err := db.Where("key = ? AND category = ?", "announcement_content", "system").First(&contentConfig).Error; err == nil {
			announcementContent = contentConfig.Value
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
		"universalUrl":        universalURL,
		"qrcodeUrl":           qrcodeURL,
		"subscription_status": subStatus,
		"expire_time":         expiryDate,
		"expiryDate":          expiryDate,
		"remaining_days":      remainingDays,
		"subscription": gin.H{
			"status":           subStatus,
			"remaining_days":   remainingDays,
			"expiryDate":       expiryDate,
			"expire_time":      expiryDate,
			"currentDevices":   deviceCount,
			"maxDevices":       subscription.DeviceLimit,
			"subscription_url": subscription.SubscriptionURL,
			"clashUrl":         clashURL,
			"universalUrl":     universalURL,
			"qrcodeUrl":        qrcodeURL,
		},
		"stat": gin.H{
			"order_count":  orderCount,
			"total_spent":  totalSpent,
			"device_count": deviceCount,
		},
		"notice": gin.H{
			"enabled": announcementEnabled,
			"content": announcementContent,
		},
	}

	utils.SuccessResponse(c, http.StatusOK, "", dashboard)
}

func GetDashboard(c *gin.Context) {
	db := database.GetDB()

	var totalUsers int64
	db.Model(&models.User{}).Count(&totalUsers)

	var activeSubscriptions int64
	now := utils.GetBeijingTime()
	db.Model(&models.Subscription{}).
		Where("is_active = ?", true).
		Where("(status = ? OR status = '' OR status IS NULL)", "active").
		Where("expire_time > ?", now).
		Count(&activeSubscriptions)

	var totalOrders int64
	db.Model(&models.Order{}).Count(&totalOrders)

	totalRevenue := utils.CalculateTotalRevenue(db, "paid")

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"totalUsers":          totalUsers,
		"activeSubscriptions": activeSubscriptions,
		"totalOrders":         totalOrders,
		"totalRevenue":        totalRevenue,
	})
}

func GetRecentUsers(c *gin.Context) {
	db := database.GetDB()
	var users []models.User
	db.Order("created_at DESC").Limit(10).Find(&users)

	userList := make([]gin.H, 0)
	for _, user := range users {
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

func GetRecentOrders(c *gin.Context) {
	db := database.GetDB()
	var orders []models.Order
	db.Preload("User").Order("created_at DESC").Limit(10).Find(&orders)

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

func GetAbnormalUsers(c *gin.Context) {
	db := database.GetDB()
	now := utils.GetBeijingTime()

	dateRange := c.QueryArray("date_range[]")
	if len(dateRange) == 0 {
		dateRange = c.QueryArray("date_range")
	}
	if len(dateRange) == 0 {
		startDate := c.Query("start_date")
		endDate := c.Query("end_date")
		if startDate != "" && endDate != "" {
			dateRange = []string{startDate, endDate}
		}
	}

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
		endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 23, 59, 59, 0, endTime.Location())
	} else {
		startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endTime = now
	}

	subscriptionCountFilter := c.DefaultQuery("subscription_count", "10") // 默认10次
	resetCountFilter := c.DefaultQuery("reset_count", "3")                // 默认3次

	oneMonthAgo := now.AddDate(0, -1, 0)

	var minSub, minReset int
	fmt.Sscanf(subscriptionCountFilter, "%d", &minSub)
	fmt.Sscanf(resetCountFilter, "%d", &minReset)

	if minSub <= 0 {
		minSub = 10
	}
	if minReset <= 0 {
		minReset = 3
	}

	subscriptionSubQuery := db.Model(&models.Subscription{}).
		Select("user_id").
		Where("created_at >= ? AND created_at <= ?", startTime, endTime).
		Group("user_id").
		Having("COUNT(*) >= ?", minSub)

	resetSubQuery := db.Model(&models.SubscriptionReset{}).
		Select("user_id").
		Where("created_at >= ? AND created_at <= ?", startTime, endTime).
		Group("user_id").
		Having("COUNT(*) >= ?", minReset)

	query := db.Model(&models.User{}).
		Where("is_active = ? OR (last_login IS NULL AND created_at < ?) OR id IN (?) OR id IN (?)",
			false, oneMonthAgo, subscriptionSubQuery, resetSubQuery)

	if len(dateRange) == 2 {
		query = query.Where("created_at BETWEEN ? AND ?", startTime, endTime)
	}

	var users []models.User
	query.Order("created_at DESC").Limit(200).Find(&users)

	userList := buildAbnormalUserDataWithDateRange(db, users, startTime, endTime, minSub, minReset)
	utils.SuccessResponse(c, http.StatusOK, "", userList)
}

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

func buildAbnormalUserData(db *gorm.DB, users []models.User) []gin.H {
	now := utils.GetBeijingTime()
	startTime := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endTime := now
	return buildAbnormalUserDataWithDateRange(db, users, startTime, endTime, 10, 3)
}

func buildAbnormalUserDataWithDateRange(db *gorm.DB, users []models.User, startTime, endTime time.Time, minSub, minReset int) []gin.H {
	now := utils.GetBeijingTime()
	oneMonthAgo := now.AddDate(0, -1, 0)
	userList := make([]gin.H, 0, len(users))

	for _, user := range users {
		lastLogin := "从未登录"
		if user.LastLogin.Valid {
			lastLogin = user.LastLogin.Time.Format("2006-01-02 15:04:05")
		}

		status := "inactive"
		if user.IsActive {
			status = "active"
		}

		var resetCount int64
		db.Model(&models.SubscriptionReset{}).
			Where("user_id = ? AND created_at >= ? AND created_at <= ?", user.ID, startTime, endTime).
			Count(&resetCount)

		var subscriptionCount int64
		db.Model(&models.Subscription{}).
			Where("user_id = ? AND created_at >= ? AND created_at <= ?", user.ID, startTime, endTime).
			Count(&subscriptionCount)

		abnormalType := "unknown"
		abnormalCount := 0
		description := ""

		if !user.IsActive {
			abnormalType = "disabled"
			abnormalCount = 1
			description = "账户已被禁用"
		} else if resetCount >= int64(minReset) {
			abnormalType = "frequent_reset"
			abnormalCount = int(resetCount)
			description = fmt.Sprintf("频繁重置订阅 %d 次（时间范围内）", resetCount)
		} else if subscriptionCount >= int64(minSub) {
			abnormalType = "frequent_subscription"
			abnormalCount = int(subscriptionCount)
			description = fmt.Sprintf("频繁创建订阅 %d 次（时间范围内）", subscriptionCount)
		} else if !user.LastLogin.Valid && user.CreatedAt.Before(oneMonthAgo) {
			abnormalType = "inactive"
			abnormalCount = 1
			description = "长期未登录（注册超过1个月且从未登录）"
		} else {
			abnormalType = "multiple_abnormal"
			abnormalCount = 1
			description = "存在多种异常行为"
		}

		var lastActivity string
		var lastActivityRecord models.UserActivity
		if err := db.Where("user_id = ?", user.ID).Order("created_at DESC").First(&lastActivityRecord).Error; err != nil {
			lastActivity = user.CreatedAt.Format("2006-01-02 15:04:05")
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

	return userList
}

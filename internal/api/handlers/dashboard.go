package handlers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
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
		expiryDate = utils.FormatBeijingTime(beijingTime)
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
	var membershipName interface{}
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
		membershipName = userLevel.LevelName
	} else {
		membershipName = nil
	}

	userPaymentSummary := utils.CalculateUserPaymentSummary(db, user.ID)

	var announcementEnabled bool
	var announcementContent string
	var announcementConfigs []models.SystemConfig
	if err := db.Where("category = ? AND key IN ?", "system", []string{"announcement_enabled", "announcement_content"}).
		Find(&announcementConfigs).Error; err == nil {
		for _, cfg := range announcementConfigs {
			switch cfg.Key {
			case "announcement_enabled":
				announcementEnabled = cfg.Value == "true"
			case "announcement_content":
				announcementContent = cfg.Value
			}
		}
	}

	dashboard := gin.H{
		"username":            user.Username,
		"email":               user.Email,
		"is_verified":         user.IsVerified,
		"is_active":           user.IsActive,
		"is_admin":            user.IsAdmin,
		"balance":             fmt.Sprintf("%.2f", user.Balance),
		"membership":          membershipName,
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
			"order_count":  userPaymentSummary.Paid,
			"total_spent":  userPaymentSummary.PaidAmount,
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

	now := utils.GetBeijingTime()
	var dashboardStats struct {
		TotalUsers          int64
		ActiveSubscriptions int64
	}
	db.Raw(`
		SELECT
			(SELECT COUNT(*) FROM users) AS total_users,
			(SELECT COUNT(*) FROM subscriptions WHERE is_active = ? AND (status = ? OR status = '' OR status IS NULL) AND expire_time > ?) AS active_subscriptions
	`, true, "active", now).Scan(&dashboardStats)

	dayStart, dayEnd := utils.GetDayRange(now)
	paymentSummary := utils.CalculatePaymentSummary(db, dayStart, dayEnd)

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"totalUsers":          dashboardStats.TotalUsers,
		"activeSubscriptions": dashboardStats.ActiveSubscriptions,
		"totalOrders":         paymentSummary.Total,
		"totalRevenue":        paymentSummary.PaidRevenue,
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
			"created_at":  utils.FormatBeijingTime(user.CreatedAt),
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
			"created_at": utils.FormatBeijingTime(order.CreatedAt),
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
	_, _ = fmt.Sscanf(subscriptionCountFilter, "%d", &minSub) // Ignore error, use default value
	_, _ = fmt.Sscanf(resetCountFilter, "%d", &minReset)      // Ignore error, use default value

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

	// 注意：日期范围只用于统计订阅/重置次数的时间范围，不用于限制用户的创建时间
	// 因为一个用户可能在上个月创建，但在本月有异常行为，应该被识别为异常用户

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
	middleware.InvalidateAuthUserCache(user.ID)
	utils.SuccessResponse(c, http.StatusOK, "已标记为正常", nil)
}

func buildAbnormalUserData(db *gorm.DB, users []models.User) []gin.H {
	now := utils.GetBeijingTime()
	startTime := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endTime := now
	return buildAbnormalUserDataWithDateRange(db, users, startTime, endTime, 10, 3)
}

func buildAbnormalUserDataWithDateRange(db *gorm.DB, users []models.User, startTime, endTime time.Time, minSub, minReset int) []gin.H {
	if len(users) == 0 {
		return []gin.H{}
	}

	now := utils.GetBeijingTime()
	oneMonthAgo := now.AddDate(0, -1, 0)

	// 收集所有用户ID
	userIDs := make([]uint, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}

	// 批量查询重置次数
	type UserCount struct {
		UserID uint  `gorm:"column:user_id"`
		Count  int64 `gorm:"column:count"`
	}
	var resetCounts []UserCount
	db.Model(&models.SubscriptionReset{}).
		Select("user_id, COUNT(*) as count").
		Where("user_id IN ? AND created_at >= ? AND created_at <= ?", userIDs, startTime, endTime).
		Group("user_id").Scan(&resetCounts)
	resetMap := make(map[uint]int64)
	for _, rc := range resetCounts {
		resetMap[rc.UserID] = rc.Count
	}

	// 批量查询订阅次数
	var subCounts []UserCount
	db.Model(&models.Subscription{}).
		Select("user_id, COUNT(*) as count").
		Where("user_id IN ? AND created_at >= ? AND created_at <= ?", userIDs, startTime, endTime).
		Group("user_id").Scan(&subCounts)
	subMap := make(map[uint]int64)
	for _, sc := range subCounts {
		subMap[sc.UserID] = sc.Count
	}

	// 批量查询最后活动时间
	type UserActivity struct {
		UserID    uint      `gorm:"column:user_id"`
		CreatedAt time.Time `gorm:"column:created_at"`
	}
	var activities []UserActivity
	db.Raw("SELECT user_id, MAX(created_at) as created_at FROM user_activities WHERE user_id IN ? GROUP BY user_id", userIDs).Scan(&activities)
	activityMap := make(map[uint]time.Time)
	for _, a := range activities {
		activityMap[a.UserID] = a.CreatedAt
	}

	userList := make([]gin.H, 0, len(users))
	for _, user := range users {
		lastLogin := "从未登录"
		if user.LastLogin.Valid {
			lastLogin = utils.FormatBeijingTime(user.LastLogin.Time)
		}

		status := "inactive"
		if user.IsActive {
			status = "active"
		}

		resetCount := resetMap[user.ID]
		subscriptionCount := subMap[user.ID]

		var abnormalTypes []string
		var abnormalDescriptions []string
		abnormalCount := 0

		if !user.IsActive {
			abnormalTypes = append(abnormalTypes, "账户已禁用")
			abnormalDescriptions = append(abnormalDescriptions, "账户已被禁用")
			abnormalCount++
		}
		if resetCount >= int64(minReset) {
			abnormalTypes = append(abnormalTypes, "频繁重置")
			abnormalDescriptions = append(abnormalDescriptions, fmt.Sprintf("频繁重置订阅 %d 次", resetCount))
			abnormalCount++
		}
		if subscriptionCount >= int64(minSub) {
			abnormalTypes = append(abnormalTypes, "频繁创建订阅")
			abnormalDescriptions = append(abnormalDescriptions, fmt.Sprintf("频繁创建订阅 %d 次", subscriptionCount))
			abnormalCount++
		}
		if !user.LastLogin.Valid && user.CreatedAt.Before(oneMonthAgo) {
			abnormalTypes = append(abnormalTypes, "长期未登录")
			abnormalDescriptions = append(abnormalDescriptions, "注册超过1个月且从未登录")
			abnormalCount++
		}

		if abnormalCount == 0 {
			continue
		}

		abnormalType := "unknown"
		description := ""
		if abnormalCount == 1 {
			if !user.IsActive {
				abnormalType = "disabled"
			} else if resetCount >= int64(minReset) {
				abnormalType = "frequent_reset"
			} else if subscriptionCount >= int64(minSub) {
				abnormalType = "frequent_subscription"
			} else {
				abnormalType = "inactive"
			}
			description = abnormalDescriptions[0]
		} else {
			abnormalType = "multiple_abnormal"
			description = fmt.Sprintf("存在 %d 种异常：%s", abnormalCount, strings.Join(abnormalTypes, "、"))
		}

		lastActivity := utils.FormatBeijingTime(user.CreatedAt)
		if t, ok := activityMap[user.ID]; ok {
			lastActivity = utils.FormatBeijingTime(t)
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
			"created_at":         utils.FormatBeijingTime(user.CreatedAt),
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

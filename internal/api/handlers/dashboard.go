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
	var specialNodeCount int64
	db.Model(&models.UserCustomNode{}).Where("user_id = ?", user.ID).Count(&specialNodeCount)

	clashURL := ""
	universalURL := ""
	qrcodeURL := ""
	multiURLs := gin.H{}
	if subscription.ID > 0 && subscription.SubscriptionURL != "" {
		multiURLs = getMultiClientSubscriptionURLs(c, subscription.SubscriptionURL)
		clashURL = multiURLs["clash_url"].(string)
		universalURL = multiURLs["universal_url"].(string)

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
		"username":                       user.Username,
		"email":                          user.Email,
		"is_verified":                    user.IsVerified,
		"is_active":                      user.IsActive,
		"is_admin":                       user.IsAdmin,
		"balance":                        fmt.Sprintf("%.2f", user.Balance),
		"membership":                     membershipName,
		"user_level":                     userLevelInfo,
		"online_devices":                 deviceCount,
		"total_devices":                  subscription.DeviceLimit,
		"subscription_url":               subscription.SubscriptionURL,
		"clashUrl":                       clashURL,
		"universalUrl":                   universalURL,
		"stashUrl":                       multiURLs["stash_url"],
		"surgeUrl":                       multiURLs["surge_url"],
		"quantumultxUrl":                 multiURLs["quantumultx_url"],
		"loonUrl":                        multiURLs["loon_url"],
		"singboxUrl":                     multiURLs["singbox_url"],
		"shadowrocketUrl":                multiURLs["shadowrocket_url"],
		"qrcodeUrl":                      qrcodeURL,
		"subscription_status":            subStatus,
		"expire_time":                    expiryDate,
		"expiryDate":                     expiryDate,
		"remaining_days":                 remainingDays,
		"has_special_nodes":              specialNodeCount > 0,
		"special_node_count":             specialNodeCount,
		"special_node_subscription_type": user.SpecialNodeSubscriptionType,
		"special_node_unlimited_devices": user.SpecialNodeUnlimitedDevices,
		"subscription": gin.H{
			"status":                         subStatus,
			"remaining_days":                 remainingDays,
			"expiryDate":                     expiryDate,
			"expire_time":                    expiryDate,
			"currentDevices":                 deviceCount,
			"maxDevices":                     subscription.DeviceLimit,
			"subscription_url":               subscription.SubscriptionURL,
			"clashUrl":                       clashURL,
			"universalUrl":                   universalURL,
			"stashUrl":                       multiURLs["stash_url"],
			"surgeUrl":                       multiURLs["surge_url"],
			"quantumultxUrl":                 multiURLs["quantumultx_url"],
			"loonUrl":                        multiURLs["loon_url"],
			"singboxUrl":                     multiURLs["singbox_url"],
			"shadowrocketUrl":                multiURLs["shadowrocket_url"],
			"qrcodeUrl":                      qrcodeURL,
			"has_special_nodes":              specialNodeCount > 0,
			"special_node_count":             specialNodeCount,
			"special_node_subscription_type": user.SpecialNodeSubscriptionType,
			"special_node_unlimited_devices": user.SpecialNodeUnlimitedDevices,
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

const (
	abnormalIPThreshold          = 5
	abnormalLocationThreshold    = 3
	abnormalLoginFailedThreshold = 5
	unverifiedAccountAgeDays     = 7
)

type abnormalUserMetric struct {
	UserID uint  `gorm:"column:user_id"`
	Count  int64 `gorm:"column:count"`
}

type abnormalDeviceLimitMetric struct {
	UserID            uint  `gorm:"column:user_id"`
	ActiveDeviceCount int64 `gorm:"column:active_device_count"`
	DeviceLimit       int   `gorm:"column:device_limit"`
}

type abnormalUserRiskMetrics struct {
	resetCounts        map[uint]int64
	subscriptionCounts map[uint]int64
	ipCounts           map[uint]int64
	locationCounts     map[uint]int64
	loginFailedCounts  map[uint]int64
	activeDeviceCounts map[uint]int64
	deviceLimits       map[uint]int
}

func GetAbnormalUsers(c *gin.Context) {
	db := database.GetDB()
	now := utils.GetBeijingTime()
	pagination := utils.ParsePagination(c)
	page := pagination.Page
	size := pagination.Size

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

	riskLevelFilter := strings.TrimSpace(c.Query("risk_level"))
	abnormalTypeFilter := strings.TrimSpace(c.Query("abnormal_type"))

	candidateIDs := collectAbnormalUserCandidateIDs(db, startTime, endTime, oneMonthAgo, minSub, minReset)
	if len(candidateIDs) == 0 {
		utils.SuccessResponse(c, http.StatusOK, "", gin.H{"users": []gin.H{}, "total": 0, "page": page, "size": size})
		return
	}

	// 注意：日期范围只用于统计订阅/重置次数的时间范围，不用于限制用户的创建时间
	// 因为一个用户可能在上个月创建，但在本月有异常行为，应该被识别为异常用户
	query := db.Model(&models.User{}).Where("id IN ?", candidateIDs)

	var total int64
	query.Count(&total)

	var users []models.User
	query.Order("created_at DESC").Find(&users)

	userList := buildAbnormalUserDataWithDateRange(db, users, startTime, endTime, minSub, minReset)
	userList = filterAbnormalUserData(userList, riskLevelFilter, abnormalTypeFilter)
	total = int64(len(userList))
	userList = paginateAbnormalUserData(userList, pagination.GetOffset(), pagination.Size)

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"users": userList, "total": total, "page": page, "size": size})
}

func collectAbnormalUserCandidateIDs(db *gorm.DB, startTime, endTime, oneMonthAgo time.Time, minSub, minReset int) []uint {
	candidateSet := make(map[uint]struct{})
	addCandidateID := func(id uint) {
		if id > 0 {
			candidateSet[id] = struct{}{}
		}
	}

	type userIDRow struct {
		ID uint `gorm:"column:id"`
	}
	var accountRows []userIDRow
	db.Model(&models.User{}).
		Select("id").
		Where("is_active = ? OR (last_login IS NULL AND created_at < ?) OR (is_verified = ? AND created_at < ?)",
			false, oneMonthAgo, false, utils.GetBeijingTime().AddDate(0, 0, -unverifiedAccountAgeDays)).
		Scan(&accountRows)
	for _, row := range accountRows {
		addCandidateID(row.ID)
	}

	metrics := loadAbnormalUserRiskMetrics(db, nil, startTime, endTime)
	for userID, count := range metrics.subscriptionCounts {
		if count >= int64(minSub) {
			addCandidateID(userID)
		}
	}
	for userID, count := range metrics.resetCounts {
		if count >= int64(minReset) {
			addCandidateID(userID)
		}
	}
	for userID, count := range metrics.ipCounts {
		if count >= abnormalIPThreshold {
			addCandidateID(userID)
		}
	}
	for userID, count := range metrics.locationCounts {
		if count >= abnormalLocationThreshold {
			addCandidateID(userID)
		}
	}
	for userID, count := range metrics.loginFailedCounts {
		if count >= abnormalLoginFailedThreshold {
			addCandidateID(userID)
		}
	}
	for userID, activeCount := range metrics.activeDeviceCounts {
		if deviceLimit := metrics.deviceLimits[userID]; deviceLimit > 0 && activeCount > int64(deviceLimit) {
			addCandidateID(userID)
		}
	}

	candidateIDs := make([]uint, 0, len(candidateSet))
	for id := range candidateSet {
		candidateIDs = append(candidateIDs, id)
	}
	return candidateIDs
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

	metrics := loadAbnormalUserRiskMetrics(db, userIDs, startTime, endTime)

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

		resetCount := metrics.resetCounts[user.ID]
		subscriptionCount := metrics.subscriptionCounts[user.ID]
		ipCount := metrics.ipCounts[user.ID]
		locationCount := metrics.locationCounts[user.ID]
		loginFailedCount := metrics.loginFailedCounts[user.ID]
		activeDeviceCount := metrics.activeDeviceCounts[user.ID]
		deviceLimit := metrics.deviceLimits[user.ID]

		var abnormalTypes []string
		var abnormalDescriptions []string
		riskReasons := make([]string, 0)
		abnormalCount := 0
		riskScore := 0
		riskLevel := "low"

		if !user.IsActive {
			abnormalTypes = append(abnormalTypes, "账户已禁用")
			abnormalDescriptions = append(abnormalDescriptions, "账户已被禁用")
			riskReasons = append(riskReasons, "账户已禁用")
			abnormalCount++
			riskScore += 80
		}
		if deviceLimit > 0 && activeDeviceCount > int64(deviceLimit) {
			abnormalTypes = append(abnormalTypes, "设备超限")
			abnormalDescriptions = append(abnormalDescriptions, fmt.Sprintf("活跃设备 %d 台，超过限制 %d 台", activeDeviceCount, deviceLimit))
			riskReasons = append(riskReasons, "设备数量超过订阅限制")
			abnormalCount++
			riskScore += 50
		}
		if ipCount >= abnormalIPThreshold {
			abnormalTypes = append(abnormalTypes, "多IP访问")
			abnormalDescriptions = append(abnormalDescriptions, fmt.Sprintf("时间段内出现 %d 个不同IP", ipCount))
			riskReasons = append(riskReasons, "短期内多IP访问")
			abnormalCount++
			riskScore += 45
		}
		if locationCount >= abnormalLocationThreshold {
			abnormalTypes = append(abnormalTypes, "多地区访问")
			abnormalDescriptions = append(abnormalDescriptions, fmt.Sprintf("时间段内出现 %d 个不同地区", locationCount))
			riskReasons = append(riskReasons, "短期内多地区访问")
			abnormalCount++
			riskScore += 40
		}
		if loginFailedCount >= abnormalLoginFailedThreshold {
			abnormalTypes = append(abnormalTypes, "登录失败过多")
			abnormalDescriptions = append(abnormalDescriptions, fmt.Sprintf("登录失败 %d 次", loginFailedCount))
			riskReasons = append(riskReasons, "登录失败次数过多")
			abnormalCount++
			riskScore += 35
		}
		if resetCount >= int64(minReset) {
			abnormalTypes = append(abnormalTypes, "频繁重置")
			abnormalDescriptions = append(abnormalDescriptions, fmt.Sprintf("频繁重置订阅 %d 次", resetCount))
			riskReasons = append(riskReasons, "频繁重置订阅")
			abnormalCount++
			riskScore += 30
		}
		if subscriptionCount >= int64(minSub) {
			abnormalTypes = append(abnormalTypes, "频繁创建订阅")
			abnormalDescriptions = append(abnormalDescriptions, fmt.Sprintf("频繁创建订阅 %d 次", subscriptionCount))
			riskReasons = append(riskReasons, "频繁创建订阅")
			abnormalCount++
			riskScore += 30
		}
		if !user.IsVerified && user.CreatedAt.Before(now.AddDate(0, 0, -unverifiedAccountAgeDays)) {
			abnormalTypes = append(abnormalTypes, "邮箱未验证")
			abnormalDescriptions = append(abnormalDescriptions, fmt.Sprintf("注册超过%d天仍未验证邮箱", unverifiedAccountAgeDays))
			riskReasons = append(riskReasons, "长期未验证邮箱")
			abnormalCount++
			riskScore += 15
		}
		if !user.LastLogin.Valid && user.CreatedAt.Before(oneMonthAgo) {
			abnormalTypes = append(abnormalTypes, "长期未登录")
			abnormalDescriptions = append(abnormalDescriptions, "注册超过1个月且从未登录")
			riskReasons = append(riskReasons, "长期未登录")
			abnormalCount++
			riskScore += 10
		}

		if abnormalCount == 0 {
			continue
		}

		if riskScore >= 70 {
			riskLevel = "high"
		} else if riskScore >= 30 {
			riskLevel = "medium"
		}

		abnormalType := "unknown"
		description := ""
		if abnormalCount == 1 {
			if !user.IsActive {
				abnormalType = "disabled"
			} else if deviceLimit > 0 && activeDeviceCount > int64(deviceLimit) {
				abnormalType = "device_over_limit"
			} else if ipCount >= abnormalIPThreshold {
				abnormalType = "multi_ip"
			} else if locationCount >= abnormalLocationThreshold {
				abnormalType = "multi_location"
			} else if loginFailedCount >= abnormalLoginFailedThreshold {
				abnormalType = "login_failed"
			} else if resetCount >= int64(minReset) {
				abnormalType = "frequent_reset"
			} else if subscriptionCount >= int64(minSub) {
				abnormalType = "frequent_subscription"
			} else if !user.IsVerified {
				abnormalType = "unverified"
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
			"id":                  user.ID,
			"user_id":             user.ID,
			"username":            user.Username,
			"email":               user.Email,
			"is_active":           user.IsActive,
			"is_verified":         user.IsVerified,
			"status":              status,
			"last_login":          lastLogin,
			"created_at":          utils.FormatBeijingTime(user.CreatedAt),
			"abnormal_type":       abnormalType,
			"abnormal_count":      abnormalCount,
			"risk_level":          riskLevel,
			"risk_score":          riskScore,
			"risk_reasons":        riskReasons,
			"reset_count":         resetCount,
			"subscription_count":  subscriptionCount,
			"ip_count":            ipCount,
			"location_count":      locationCount,
			"active_device_count": activeDeviceCount,
			"device_limit":        deviceLimit,
			"login_failed_count":  loginFailedCount,
			"description":         description,
			"last_activity":       lastActivity,
		})
	}

	return userList
}

func loadAbnormalUserRiskMetrics(db *gorm.DB, userIDs []uint, startTime, endTime time.Time) abnormalUserRiskMetrics {
	metrics := abnormalUserRiskMetrics{
		resetCounts:        make(map[uint]int64),
		subscriptionCounts: make(map[uint]int64),
		ipCounts:           make(map[uint]int64),
		locationCounts:     make(map[uint]int64),
		loginFailedCounts:  make(map[uint]int64),
		activeDeviceCounts: make(map[uint]int64),
		deviceLimits:       make(map[uint]int),
	}

	applyUserFilter := func(query *gorm.DB, column string) *gorm.DB {
		if len(userIDs) == 0 {
			return query
		}
		return query.Where(column+" IN ?", userIDs)
	}

	var resetCounts []abnormalUserMetric
	resetQuery := db.Model(&models.SubscriptionReset{}).
		Select("user_id, COUNT(*) as count").
		Where("created_at >= ? AND created_at <= ?", startTime, endTime)
	applyUserFilter(resetQuery, "user_id").Group("user_id").Scan(&resetCounts)
	for _, row := range resetCounts {
		metrics.resetCounts[row.UserID] = row.Count
	}

	var subscriptionCounts []abnormalUserMetric
	subscriptionQuery := db.Model(&models.Subscription{}).
		Select("user_id, COUNT(*) as count").
		Where("created_at >= ? AND created_at <= ?", startTime, endTime)
	applyUserFilter(subscriptionQuery, "user_id").Group("user_id").Scan(&subscriptionCounts)
	for _, row := range subscriptionCounts {
		metrics.subscriptionCounts[row.UserID] = row.Count
	}

	var deviceLimitRows []abnormalDeviceLimitMetric
	deviceLimitQuery := db.Table("subscriptions").
		Select("subscriptions.user_id, COUNT(devices.id) AS active_device_count, subscriptions.device_limit AS device_limit").
		Joins("LEFT JOIN devices ON devices.subscription_id = subscriptions.id AND devices.is_active = ?", true).
		Where("subscriptions.device_limit > 0 AND subscriptions.is_active = ?", true)
	applyUserFilter(deviceLimitQuery, "subscriptions.user_id").
		Group("subscriptions.id, subscriptions.user_id, subscriptions.device_limit").
		Scan(&deviceLimitRows)
	for _, row := range deviceLimitRows {
		if row.ActiveDeviceCount > metrics.activeDeviceCounts[row.UserID] {
			metrics.activeDeviceCounts[row.UserID] = row.ActiveDeviceCount
			metrics.deviceLimits[row.UserID] = row.DeviceLimit
		}
	}

	loadDistinctCountMap(db.Model(&models.Device{}), userIDs, "user_id", "ip_address", "last_access", startTime, endTime, metrics.ipCounts, "user_id IS NOT NULL AND is_active = ?", true)
	loadDistinctCountMap(db.Model(&models.LoginHistory{}), userIDs, "user_id", "ip_address", "login_time", startTime, endTime, metrics.ipCounts, "login_status = ?", "success")
	loadDistinctCountMap(db.Model(&models.Device{}), userIDs, "user_id", "location", "last_access", startTime, endTime, metrics.locationCounts, "user_id IS NOT NULL AND is_active = ?", true)
	loadDistinctCountMap(db.Model(&models.LoginHistory{}), userIDs, "user_id", "location", "login_time", startTime, endTime, metrics.locationCounts, "login_status = ?", "success")

	var loginFailedCounts []abnormalUserMetric
	loginFailedQuery := db.Table("login_attempts").
		Select("users.id AS user_id, COUNT(login_attempts.id) AS count").
		Joins("JOIN users ON lower(login_attempts.username) = lower(users.email) OR lower(login_attempts.username) = lower(users.username)").
		Where("login_attempts.success = ? AND login_attempts.created_at >= ? AND login_attempts.created_at <= ?", false, startTime, endTime)
	applyUserFilter(loginFailedQuery, "users.id").Group("users.id").Scan(&loginFailedCounts)
	for _, row := range loginFailedCounts {
		metrics.loginFailedCounts[row.UserID] = row.Count
	}

	return metrics
}

func loadDistinctCountMap(query *gorm.DB, userIDs []uint, userColumn, distinctColumn, timeColumn string, startTime, endTime time.Time, target map[uint]int64, extraCondition string, extraArgs ...interface{}) {
	var rows []abnormalUserMetric
	query = query.Select(fmt.Sprintf("%s AS user_id, COUNT(DISTINCT %s) AS count", userColumn, distinctColumn)).
		Where(fmt.Sprintf("%s IS NOT NULL AND %s != ''", distinctColumn, distinctColumn)).
		Where(fmt.Sprintf("%s >= ? AND %s <= ?", timeColumn, timeColumn), startTime, endTime).
		Where(extraCondition, extraArgs...)
	if len(userIDs) > 0 {
		query = query.Where(userColumn+" IN ?", userIDs)
	}
	query.Group(userColumn).Scan(&rows)
	for _, row := range rows {
		if row.Count > target[row.UserID] {
			target[row.UserID] = row.Count
		}
	}
}

func filterAbnormalUserData(users []gin.H, riskLevelFilter, abnormalTypeFilter string) []gin.H {
	if riskLevelFilter == "" && abnormalTypeFilter == "" {
		return users
	}

	filtered := make([]gin.H, 0, len(users))
	for _, user := range users {
		if riskLevelFilter != "" && user["risk_level"] != riskLevelFilter {
			continue
		}
		if abnormalTypeFilter != "" && user["abnormal_type"] != abnormalTypeFilter {
			continue
		}
		filtered = append(filtered, user)
	}
	return filtered
}

func paginateAbnormalUserData(users []gin.H, offset, size int) []gin.H {
	if offset >= len(users) {
		return []gin.H{}
	}
	end := offset + size
	if end > len(users) {
		end = len(users)
	}
	return users[offset:end]
}

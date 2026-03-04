package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
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
	"cboard-go/internal/services/geoip"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func getDefaultSubscriptionSettings(db *gorm.DB) (deviceLimit int, durationMonths int) {
	deviceLimit = 0
	durationMonths = 0

	var deviceLimitConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "default_subscription_device_limit", "registration").First(&deviceLimitConfig).Error; err != nil {
		if err := db.Where("key = ? AND category = ?", "default_subscription_device_limit", "general").First(&deviceLimitConfig).Error; err == nil {
			if deviceLimitConfig.Value != "" {
				if limit, err := strconv.Atoi(deviceLimitConfig.Value); err == nil && limit >= 0 {
					deviceLimit = limit
				}
			}
		}
	} else {
		if deviceLimitConfig.Value != "" {
			if limit, err := strconv.Atoi(deviceLimitConfig.Value); err == nil && limit >= 0 {
				deviceLimit = limit
			}
		}
	}

	var durationConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "default_subscription_duration_months", "registration").First(&durationConfig).Error; err != nil {
		if err := db.Where("key = ? AND category = ?", "default_subscription_duration_months", "general").First(&durationConfig).Error; err == nil {
			if durationConfig.Value != "" {
				if months, err := strconv.Atoi(durationConfig.Value); err == nil && months >= 0 {
					durationMonths = months
				}
			}
		}
	} else {
		if durationConfig.Value != "" {
			if months, err := strconv.Atoi(durationConfig.Value); err == nil && months >= 0 {
				durationMonths = months
			}
		}
	}

	return deviceLimit, durationMonths
}

func createDefaultSubscription(db *gorm.DB, userID uint) error {
	var existing models.Subscription
	err := db.Where("user_id = ?", userID).First(&existing).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	deviceLimit, durationMonths := getDefaultSubscriptionSettings(db)

	subscriptionURL := utils.GenerateSubscriptionURL()

	now := utils.GetBeijingTime()
	var expireTime time.Time
	if durationMonths <= 0 {
		expireTime = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	} else {
		expireTime = now.AddDate(0, durationMonths, 0)
	}

	sub := models.Subscription{
		UserID:          userID,
		SubscriptionURL: subscriptionURL,
		DeviceLimit:     deviceLimit,
		CurrentDevices:  0,
		IsActive:        true,
		Status:          "active",
		ExpireTime:      expireTime,
	}

	if err := db.Create(&sub).Error; err != nil {
		return err
	}
	return nil
}

func GetCurrentUser(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	lastLoginStr := utils.FormatNullTimeBeijing(user.LastLogin)

	responseData := gin.H{
		"id":                  user.ID,
		"username":            user.Username,
		"email":               user.Email,
		"is_active":           user.IsActive,
		"is_verified":         user.IsVerified,
		"is_admin":            user.IsAdmin,
		"created_at":          utils.FormatBeijingTime(user.CreatedAt),
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

	if user.Nickname.Valid {
		responseData["nickname"] = user.Nickname.String
	}

	if user.Avatar.Valid {
		responseData["avatar"] = user.Avatar.String
		responseData["avatar_url"] = user.Avatar.String
	}

	utils.SuccessResponse(c, http.StatusOK, "", responseData)
}

func UpdateCurrentUser(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
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
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()

	if req.Username != "" {
		var existingUser models.User
		if err := db.Where("username = ? AND id != ?", req.Username, user.ID).First(&existingUser).Error; err == nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "用户名已被使用", nil)
			return
		}
		user.Username = req.Username
	}
	if req.Nickname != "" {
		user.Nickname = database.NullString(req.Nickname)
	} else if req.Nickname == "" {
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
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新失败", err)
		return
	}

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

	utils.SuccessResponse(c, http.StatusOK, "更新成功", responseData)
}

func GetUsers(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.User{})
	pagination := utils.ParsePagination(c)
	page := pagination.Page
	size := pagination.Size
	if kw := c.Query("keyword"); kw != "" {
		escapedKw := utils.EscapeLikePattern(utils.SanitizeSearchKeyword(kw))
		searchPattern := "%" + escapedKw + "%"
		// 使用 COALESCE 处理 NULL 值，确保备注搜索能正常工作
		query = query.Where("username LIKE ? OR email LIKE ? OR COALESCE(notes, '') LIKE ?", searchPattern, searchPattern, searchPattern)
	}
	if st := c.Query("status"); st != "" {
		switch st {
		case "active":
			query = query.Where("is_active = ?", true)
		case "inactive":
			query = query.Where("is_active = ?", false)
		case "admin":
			query = query.Where("is_admin = ?", true)
		}
	}
	// 处理排序
	sortField := strings.TrimSpace(c.Query("sort"))
	sortOrder := strings.TrimSpace(c.Query("order"))
	orderBy := "created_at DESC" // 默认排序

	if sortField != "" {
		// 验证排序字段，防止 SQL 注入
		allowedSortFields := map[string]string{
			"balance":    "balance",
			"created_at": "created_at",
			"username":   "username",
			"email":      "email",
		}

		if dbField, ok := allowedSortFields[sortField]; ok {
			if sortOrder == "asc" {
				orderBy = dbField + " ASC"
			} else if sortOrder == "desc" {
				orderBy = dbField + " DESC"
			} else {
				// 如果没有指定排序方向，默认降序
				orderBy = dbField + " DESC"
			}
		}
	}

	var total int64
	query.Count(&total)
	var users []models.User
	query.Offset(pagination.GetOffset()).Limit(pagination.Size).Order(orderBy).Find(&users)

	userIDs := make([]uint, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}

	var subscriptions []models.Subscription
	if len(userIDs) > 0 {
		db.Raw(`
			SELECT s1.* FROM subscriptions s1
			INNER JOIN (
				SELECT user_id, MAX(created_at) as max_created_at
				FROM subscriptions
				WHERE user_id IN ?
				GROUP BY user_id
			) s2 ON s1.user_id = s2.user_id AND s1.created_at = s2.max_created_at
			WHERE s1.user_id IN ?
		`, userIDs, userIDs).Scan(&subscriptions)
	}

	subMap := make(map[uint]*models.Subscription)
	for i := range subscriptions {
		subMap[subscriptions[i].UserID] = &subscriptions[i]
	}

	subIDs := make([]uint, 0)
	for _, sub := range subscriptions {
		if sub.ID > 0 {
			subIDs = append(subIDs, sub.ID)
		}
	}

	var deviceCounts []struct {
		SubscriptionID uint
		Count          int64
	}
	if len(subIDs) > 0 {
		db.Model(&models.Device{}).
			Select("subscription_id, COUNT(*) as count").
			Where("subscription_id IN ? AND is_active = ?", subIDs, true).
			Group("subscription_id").
			Scan(&deviceCounts)
	}

	deviceCountMap := make(map[uint]int64)
	for _, dc := range deviceCounts {
		deviceCountMap[dc.SubscriptionID] = dc.Count
	}

	list := make([]gin.H, 0, len(users))
	now := utils.GetBeijingTime()
	for _, u := range users {
		sub := subMap[u.ID]

		var online int64
		var deviceLimit int
		var currentDevices int
		if sub != nil && sub.ID > 0 {
			online = deviceCountMap[sub.ID]
			deviceLimit = sub.DeviceLimit
			currentDevices = sub.CurrentDevices
			if currentDevices < int(online) {
				currentDevices = int(online)
			}
		}

		var subscriptionInfo gin.H
		if sub != nil && sub.ID > 0 {
			daysUntilExpire := 0
			isExpired := false
			if !sub.ExpireTime.IsZero() {
				diff := sub.ExpireTime.Sub(now)
				if diff > 0 {
					daysUntilExpire = int(diff.Hours() / 24)
				} else {
					isExpired = true
				}
			}

			subscriptionInfo = gin.H{
				"id":                sub.ID,
				"status":            sub.Status,
				"is_active":         sub.IsActive,
				"device_limit":      deviceLimit,
				"current_devices":   currentDevices,
				"expire_time":       utils.FormatBeijingTime(sub.ExpireTime),
				"days_until_expire": daysUntilExpire,
				"is_expired":        isExpired,
			}
		} else {
			subscriptionInfo = nil
		}

		lastLogin := ""
		if u.LastLogin.Valid {
			lastLogin = utils.FormatBeijingTime(u.LastLogin.Time)
		}

		notes := ""
		if u.Notes.Valid {
			notes = u.Notes.String
		}
		list = append(list, gin.H{
			"id":        u.ID,
			"username":  u.Username,
			"email":     u.Email,
			"balance":   u.Balance,
			"is_active": u.IsActive,
			"is_admin":  u.IsAdmin,
			"status": func() string {
				if !u.IsActive {
					return "inactive"
				}
				return "active"
			}(),
			"online_devices": online,
			"created_at":     utils.FormatBeijingTime(u.CreatedAt),
			"last_login":     lastLogin,
			"subscription":   subscriptionInfo,
			"notes":          notes,
		})
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"users": list, "total": total, "page": page, "size": size})
}

func GetUser(c *gin.Context) {
	currentUser, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未授权", nil)
		return
	}

	requestedUserID := c.Param("id")
	db := database.GetDB()
	var u models.User
	if err := db.First(&u, requestedUserID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "不存在", err)
		return
	}

	// 权限检查：只能查看自己的信息，除非是管理员
	if u.ID != currentUser.ID && !currentUser.IsAdmin {
		utils.ErrorResponse(c, http.StatusForbidden, "无权访问其他用户信息", nil)
		utils.CreateBusinessLog(c, "unauthorized_user_access", "尝试越权访问用户信息", "warning", map[string]interface{}{
			"current_user_id":   currentUser.ID,
			"requested_user_id": u.ID,
		})
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", u)
}

func GetUserDetails(c *gin.Context) {
	currentUser, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未授权", nil)
		return
	}

	requestedUserID := c.Param("id")
	db := database.GetDB()
	var u models.User
	if err := db.First(&u, requestedUserID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "不存在", err)
		return
	}

	// 权限检查：只能查看自己的详细信息，除非是管理员
	if u.ID != currentUser.ID && !currentUser.IsAdmin {
		utils.ErrorResponse(c, http.StatusForbidden, "无权访问其他用户详细信息", nil)
		utils.CreateBusinessLog(c, "unauthorized_user_details_access", "尝试越权访问用户详细信息", "warning", map[string]interface{}{
			"current_user_id":   currentUser.ID,
			"requested_user_id": u.ID,
		})
		return
	}

	lastLogin := ""
	if u.LastLogin.Valid {
		lastLogin = utils.FormatBeijingTime(u.LastLogin.Time)
	}

	// 根据用户权限返回不同的信息
	var userInfo gin.H
	if currentUser.IsAdmin {
		// 管理员可以看到所有信息
		userInfo = gin.H{
			"id":          u.ID,
			"username":    u.Username,
			"email":       u.Email,
			"balance":     u.Balance,
			"is_active":   u.IsActive,
			"is_verified": u.IsVerified,
			"is_admin":    u.IsAdmin,
			"created_at":  utils.FormatBeijingTime(u.CreatedAt),
			"last_login":  lastLogin,
			"theme":       u.Theme,
			"language":    u.Language,
			"timezone":    u.Timezone,
		}
	} else {
		// 普通用户只能看到自己的基本信息（不包括敏感字段）
		userInfo = gin.H{
			"id":          u.ID,
			"username":    u.Username,
			"email":       u.Email,
			"balance":     u.Balance,
			"is_active":   u.IsActive,
			"is_verified": u.IsVerified,
			"created_at":  utils.FormatBeijingTime(u.CreatedAt),
			"last_login":  lastLogin,
			"theme":       u.Theme,
			"language":    u.Language,
			"timezone":    u.Timezone,
		}
	}

	if u.Nickname.Valid {
		userInfo["nickname"] = u.Nickname.String
	}
	if u.Avatar.Valid {
		userInfo["avatar"] = u.Avatar.String
		userInfo["avatar_url"] = u.Avatar.String
	}

	var subs []models.Subscription
	db.Where("user_id = ?", u.ID).Find(&subs)

	formattedSubs := make([]gin.H, 0, len(subs))
	for _, sub := range subs {
		var online int64
		db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", sub.ID, true).Count(&online)

		daysUntilExpire := 0
		isExpired := false
		now := utils.GetBeijingTime()
		if !sub.ExpireTime.IsZero() {
			diff := sub.ExpireTime.Sub(now)
			if diff > 0 {
				daysUntilExpire = int(diff.Hours() / 24)
			} else {
				isExpired = true
			}
		}

		universalCount := sub.UniversalCount
		clashCount := sub.ClashCount

		// 生成通用订阅和Clash订阅URL
		universalURL, clashURL := getSubscriptionURLs(c, sub.SubscriptionURL)

		formattedSubs = append(formattedSubs, gin.H{
			"id":                sub.ID,
			"subscription_url":  sub.SubscriptionURL,
			"universal_url":     universalURL,
			"clash_url":         clashURL,
			"status":            sub.Status,
			"is_active":         sub.IsActive,
			"device_limit":      sub.DeviceLimit,
			"current_devices":   sub.CurrentDevices,
			"online_devices":    online,
			"expire_time":       utils.FormatBeijingTime(sub.ExpireTime),
			"days_until_expire": daysUntilExpire,
			"is_expired":        isExpired,
			"created_at":        utils.FormatBeijingTime(sub.CreatedAt),
			"apple_count":       universalCount,
			"clash_count":       clashCount,
			"package_name":      sub.Package.Name,
		})
	}

	var orders []models.Order
	db.Preload("Package").Where("user_id = ?", u.ID).Order("created_at DESC").Limit(50).Find(&orders)

	formattedOrders := make([]gin.H, 0, len(orders))
	for _, order := range orders {
		formattedOrder := gin.H{
			"id":         order.ID,
			"order_no":   order.OrderNo,
			"user_id":    order.UserID,
			"package_id": order.PackageID,
			"amount":     order.Amount,
			"status":     order.Status,
			"created_at": utils.FormatBeijingTime(order.CreatedAt),
			"updated_at": utils.FormatBeijingTime(order.UpdatedAt),
		}

		if order.PaymentMethodName.Valid {
			formattedOrder["payment_method"] = order.PaymentMethodName.String
			formattedOrder["payment_method_name"] = order.PaymentMethodName.String
		} else {
			formattedOrder["payment_method"] = nil
			formattedOrder["payment_method_name"] = nil
		}

		if order.PaymentTime.Valid {
			formattedOrder["payment_time"] = utils.FormatBeijingTime(order.PaymentTime.Time)
		} else {
			formattedOrder["payment_time"] = nil
		}

		formattedOrder["payment_transaction_id"] = utils.GetNullStringValue(order.PaymentTransactionID)

		if order.ExpireTime.Valid {
			formattedOrder["expire_time"] = utils.FormatBeijingTime(order.ExpireTime.Time)
		} else {
			formattedOrder["expire_time"] = nil
		}

		if order.Package.ID > 0 {
			formattedOrder["package_name"] = order.Package.Name
		} else {
			formattedOrder["package_name"] = ""
		}

		if order.DiscountAmount.Valid {
			formattedOrder["discount_amount"] = order.DiscountAmount.Float64
		} else {
			formattedOrder["discount_amount"] = 0
		}

		if order.FinalAmount.Valid {
			formattedOrder["final_amount"] = order.FinalAmount.Float64
		} else {
			formattedOrder["final_amount"] = order.Amount
		}

		formattedOrders = append(formattedOrders, formattedOrder)
	}

	var recharges []models.RechargeRecord
	db.Where("user_id = ?", u.ID).Order("created_at DESC").Limit(50).Find(&recharges)

	formattedRecharges := make([]gin.H, 0, len(recharges))
	formatIPForRecharge := func(ip string) string {
		if ip == "" {
			return "-"
		}
		if ip == "::1" {
			return "127.0.0.1"
		}
		if strings.HasPrefix(ip, "::ffff:") {
			return strings.TrimPrefix(ip, "::ffff:")
		}
		return ip
	}
	for _, record := range recharges {
		ipValue := utils.GetNullStringValue(record.IPAddress)
		var ipStr string
		if ipValue != nil {
			ipStr = ipValue.(string)
		}
		ipAddress := formatIPForRecharge(ipStr)
		location := ""
		if ipAddress != "" && ipAddress != "-" && geoip.IsEnabled() {
			locationStr := geoip.GetLocationString(ipAddress)
			if locationStr.Valid {
				location = locationStr.String
			}
		}

		formattedRecharges = append(formattedRecharges, gin.H{
			"id":                     record.ID,
			"user_id":                record.UserID,
			"order_no":               record.OrderNo,
			"amount":                 record.Amount,
			"status":                 record.Status,
			"payment_method":         utils.GetNullStringValue(record.PaymentMethod),
			"payment_transaction_id": utils.GetNullStringValue(record.PaymentTransactionID),
			"payment_qr_code":        utils.GetNullStringValue(record.PaymentQRCode),
			"payment_url":            utils.GetNullStringValue(record.PaymentURL),
			"ip_address":             ipAddress,
			"location":               location, // 添加归属地信息
			"user_agent":             utils.GetNullStringValue(record.UserAgent),
			"paid_at": func() interface{} {
				if record.PaidAt.Valid {
					return utils.FormatBeijingTime(record.PaidAt.Time)
				}
				return nil
			}(),
			"created_at": utils.FormatBeijingTime(record.CreatedAt),
			"updated_at": utils.FormatBeijingTime(record.UpdatedAt),
		})
	}

	var checkins []models.CheckinRecord
	db.Where("user_id = ?", u.ID).Order("created_at DESC").Limit(100).Find(&checkins)
	formattedCheckins := make([]gin.H, 0, len(checkins))
	for _, record := range checkins {
		formattedCheckins = append(formattedCheckins, gin.H{
			"id":         record.ID,
			"user_id":    record.UserID,
			"amount":     record.Amount,
			"created_at": utils.FormatBeijingTime(record.CreatedAt),
		})
	}

	var totalOrders int64
	db.Model(&models.Order{}).Where("user_id = ?", u.ID).Count(&totalOrders)

	var totalSpent float64
	db.Model(&models.Order{}).Where("user_id = ? AND status = 'paid'", u.ID).Select("COALESCE(SUM(final_amount), SUM(amount), 0)").Scan(&totalSpent)

	var totalResets int64
	db.Model(&models.SubscriptionReset{}).Where("user_id = ?", u.ID).Count(&totalResets)

	var resets []models.SubscriptionReset
	db.Where("user_id = ?", u.ID).Order("created_at DESC").Find(&resets)
	formattedResets := make([]gin.H, 0, len(resets))
	getStringPtr := func(ptr *string) string {
		if ptr != nil {
			return *ptr
		}
		return ""
	}
	for _, reset := range resets {
		formattedResets = append(formattedResets, gin.H{
			"id":                   reset.ID,
			"subscription_id":      reset.SubscriptionID,
			"reset_type":           reset.ResetType,
			"reason":               reset.Reason,
			"old_subscription_url": getStringPtr(reset.OldSubscriptionURL),
			"new_subscription_url": getStringPtr(reset.NewSubscriptionURL),
			"device_count_before":  reset.DeviceCountBefore,
			"device_count_after":   reset.DeviceCountAfter,
			"reset_by":             getStringPtr(reset.ResetBy),
			"created_at":           utils.FormatBeijingTime(reset.CreatedAt),
		})
	}

	var subIDs []uint
	for _, sub := range subs {
		subIDs = append(subIDs, sub.ID)
	}
	uaRecords := make([]gin.H, 0)
	getString := func(ptr *string) string {
		if ptr != nil {
			return *ptr
		}
		return ""
	}
	formatIPForUA := func(ip string) string {
		if ip == "" {
			return "-"
		}
		if ip == "::1" {
			return "127.0.0.1"
		}
		if strings.HasPrefix(ip, "::ffff:") {
			return strings.TrimPrefix(ip, "::ffff:")
		}
		return ip
	}
	if len(subIDs) > 0 {
		var devices []models.Device
		db.Where("subscription_id IN ?", subIDs).
			Where("user_agent IS NOT NULL AND user_agent != ''").
			Order("last_access DESC").
			Find(&devices)

		uaMap := make(map[string]*models.Device)
		for i := range devices {
			if devices[i].UserAgent != nil && *devices[i].UserAgent != "" {
				ua := *devices[i].UserAgent
				if existing, exists := uaMap[ua]; !exists {
					uaMap[ua] = &devices[i]
				} else {
					if devices[i].LastAccess.After(existing.LastAccess) {
						uaMap[ua] = &devices[i]
					}
				}
			}
		}

		for _, d := range uaMap {
			ipAddress := formatIPForUA(getString(d.IPAddress))
			location := ""
			if ipAddress != "" && ipAddress != "-" && geoip.IsEnabled() {
				locationStr := geoip.GetLocationString(ipAddress)
				if locationStr.Valid {
					location = locationStr.String
				}
			}

			uaRecords = append(uaRecords, gin.H{
				"user_agent":   *d.UserAgent,
				"device_type":  getString(d.DeviceType),
				"device_name":  getString(d.DeviceName),
				"ip_address":   ipAddress,
				"location":     location,
				"created_at":   utils.FormatBeijingTime(d.CreatedAt),
				"last_access":  utils.FormatBeijingTime(d.LastAccess),
				"access_count": d.AccessCount,
			})
		}
	}

	var loginHistory []models.LoginHistory
	db.Where("user_id = ?", u.ID).Order("login_time DESC").Limit(50).Find(&loginHistory)
	formattedLoginHistory := make([]gin.H, 0, len(loginHistory))
	for _, lh := range loginHistory {
		ipAddr := ""
		if lh.IPAddress.Valid {
			ipAddr = lh.IPAddress.String
		}
		if ipAddr == "::1" {
			ipAddr = "127.0.0.1"
		} else if strings.HasPrefix(ipAddr, "::ffff:") {
			ipAddr = strings.TrimPrefix(ipAddr, "::ffff:")
		}
		location := ""
		if lh.Location.Valid {
			location = lh.Location.String
		} else if ipAddr != "" && geoip.IsEnabled() {
			if loc := geoip.GetLocationString(ipAddr); loc.Valid {
				location = loc.String
			}
		}
		entry := gin.H{
			"id":           lh.ID,
			"login_time":   utils.FormatBeijingTime(lh.LoginTime),
			"ip_address":   ipAddr,
			"location":     location,
			"login_status": lh.LoginStatus,
		}
		if lh.UserAgent.Valid {
			entry["user_agent"] = lh.UserAgent.String
		}
		if lh.FailureReason.Valid {
			entry["failure_reason"] = lh.FailureReason.String
		}
		formattedLoginHistory = append(formattedLoginHistory, entry)
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"user_info":        userInfo,
		"subscriptions":    formattedSubs,
		"orders":           formattedOrders,
		"recharge_records": formattedRecharges,
		"checkin_records":  formattedCheckins,
		"statistics": gin.H{
			"total_subscriptions": len(subs),
			"total_orders":        totalOrders,
			"total_resets":        totalResets,
			"total_spent":         totalSpent,
		},
		"subscription_resets": formattedResets,
		"ua_records":          uaRecords,
		"login_history":       formattedLoginHistory,
	})
}

func buildUserCheckinLogsQuery(db *gorm.DB, c *gin.Context, userID uint) (*gorm.DB, error) {
	query := db.Model(&models.CheckinRecord{}).Where("user_id = ?", userID)

	if startTime := strings.TrimSpace(c.Query("start_time")); startTime != "" {
		t, err := time.ParseInLocation(TimeLayout, startTime, utils.BeijingTZ)
		if err != nil {
			return nil, fmt.Errorf("开始时间格式错误，请使用 %s", TimeLayout)
		}
		query = query.Where("created_at >= ?", t)
	}
	if endTime := strings.TrimSpace(c.Query("end_time")); endTime != "" {
		t, err := time.ParseInLocation(TimeLayout, endTime, utils.BeijingTZ)
		if err != nil {
			return nil, fmt.Errorf("结束时间格式错误，请使用 %s", TimeLayout)
		}
		query = query.Where("created_at <= ?", t)
	}

	return query, nil
}

func GetUserCheckinLogs(c *gin.Context) {
	currentUser, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未授权", nil)
		return
	}
	if !currentUser.IsAdmin {
		utils.ErrorResponse(c, http.StatusForbidden, "权限不足", nil)
		return
	}

	db := database.GetDB()
	userID := c.Param("id")

	var user models.User
	if err := db.Select("id").First(&user, userID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "用户不存在", err)
		return
	}

	query, err := buildUserCheckinLogsQuery(db, c, user.ID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), err)
		return
	}

	pagination := utils.ParsePagination(c)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取签到日志总数失败", err)
		return
	}

	var records []models.CheckinRecord
	if err := query.Order("created_at DESC").Offset(pagination.GetOffset()).Limit(pagination.Size).Find(&records).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取签到日志失败", err)
		return
	}

	logs := make([]gin.H, 0, len(records))
	for _, record := range records {
		logs = append(logs, gin.H{
			"id":         record.ID,
			"user_id":    record.UserID,
			"amount":     record.Amount,
			"created_at": utils.FormatBeijingTime(record.CreatedAt),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"logs":  logs,
		"total": total,
		"page":  pagination.Page,
		"size":  pagination.Size,
	})
}

func ExportUserCheckinLogs(c *gin.Context) {
	currentUser, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未授权", nil)
		return
	}
	if !currentUser.IsAdmin {
		utils.ErrorResponse(c, http.StatusForbidden, "权限不足", nil)
		return
	}

	db := database.GetDB()
	userID := c.Param("id")

	var user models.User
	if err := db.Select("id").First(&user, userID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "用户不存在", err)
		return
	}

	query, err := buildUserCheckinLogsQuery(db, c, user.ID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), err)
		return
	}

	var records []models.CheckinRecord
	if err := query.Order("created_at DESC").Limit(20000).Find(&records).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "导出签到日志失败", err)
		return
	}

	var csvContent strings.Builder
	csvContent.WriteString("\xEF\xBB\xBF")
	csvContent.WriteString("签到时间,奖励金额,用户ID,备注\n")
	for _, record := range records {
		csvContent.WriteString(fmt.Sprintf("%s,%.2f,%d,每日签到奖励\n",
			utils.FormatBeijingTime(record.CreatedAt), record.Amount, record.UserID))
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=user_%d_checkin_logs_%s.csv",
		user.ID, utils.GetBeijingTime().Format("20060102")))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", []byte(csvContent.String()))
}

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
		Notes       string  `json:"notes"`        // 备注
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError("CreateUser: bind request", err, nil)
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误，请检查输入格式", err)
		return
	}

	req.Email = utils.NormalizeEmail(req.Email)
	db := database.GetDB()

	var existingUser models.User
	if err := db.Where("LOWER(email) = ? OR username = ?", req.Email, req.Username).First(&existingUser).Error; err == nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "邮箱或用户名已存在", nil)
		return
	}

	valid, msg := auth.ValidatePasswordStrength(req.Password, 8)
	if !valid {
		utils.ErrorResponse(c, http.StatusBadRequest, msg, nil)
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "密码加密失败", err)
		return
	}

	user := models.User{
		Username:   req.Username,
		Email:      req.Email,
		Password:   hashedPassword,
		IsActive:   req.IsActive,
		IsVerified: req.IsVerified,
		IsAdmin:    req.IsAdmin,
		Balance:    req.Balance,
	}
	if req.Notes != "" {
		user.Notes = database.NullString(req.Notes)
	}

	if err := db.Create(&user).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建用户失败", err)
		return
	}

	deviceLimit := req.DeviceLimit
	defaultDeviceLimit, defaultDurationMonths := getDefaultSubscriptionSettings(db)
	if deviceLimit == 0 {
		deviceLimit = defaultDeviceLimit
	}

	var expireTime time.Time
	if req.ExpireTime != "" {
		parsedTime, err := time.Parse("2006-01-02T15:04:05", req.ExpireTime)
		if err != nil {
			parsedTime, err = time.Parse("2006-01-02 15:04:05", req.ExpireTime)
			if err != nil {
				months := defaultDurationMonths
				if months <= 0 {
					months = 1
				}
				expireTime = utils.GetBeijingTime().AddDate(0, months, 0)
			} else {
				expireTime = parsedTime.In(utils.BeijingTZ)
			}
		} else {
			expireTime = parsedTime.In(utils.BeijingTZ)
		}
	} else {
		months := defaultDurationMonths
		if months <= 0 {
			months = 1
		}
		expireTime = utils.GetBeijingTime().AddDate(0, months, 0)
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
		if utils.AppLogger != nil {
			utils.AppLogger.Error("创建用户订阅失败: %v", err)
		}
	} else {
		// 记录订阅日志
		go func() {
			ipAddress := utils.GetRealClientIP(c)
			adminUser, _ := middleware.GetCurrentUser(c)
			var actionByUserID *uint
			actionBy := "admin"
			if adminUser != nil {
				actionByUserID = &adminUser.ID
			}
			afterData := map[string]interface{}{
				"subscription_id": subscription.ID,
				"device_limit":    subscription.DeviceLimit,
				"expire_time":     utils.FormatBeijingTime(subscription.ExpireTime),
				"status":          subscription.Status,
			}
			utils.CreateSubscriptionLog(subscription.ID, user.ID, "create", actionBy, actionByUserID, ipAddress, nil, afterData, "管理员创建用户时自动创建订阅")
		}()
	}

	utils.CreateAuditLogSimple(c, "create_user", "user", user.ID,
		fmt.Sprintf("管理员创建用户: %s (%s), 管理员权限: %v", user.Username, user.Email, user.IsAdmin))

	go func() {
		notificationService := notification.NewNotificationService()
		adminUser, _ := middleware.GetCurrentUser(c)
		createdBy := "系统"
		if adminUser != nil {
			createdBy = adminUser.Username
		}
		createTime := utils.FormatBeijingTime(utils.GetBeijingTime())

		expireTimeStr := "未设置"
		if !expireTime.IsZero() {
			expireTimeStr = utils.FormatBeijingTime(expireTime)
		}

		plainPassword := req.Password

		_ = notificationService.SendAdminNotification("user_created", map[string]interface{}{
			"username":     user.Username,
			"email":        user.Email,
			"password":     plainPassword, // 明文密码
			"created_by":   createdBy,
			"create_time":  createTime,
			"expire_time":  expireTimeStr,
			"device_limit": deviceLimit,
		})
	}()

	go func() {
		plainPassword := req.Password
		userEmail := user.Email
		userUsername := user.Username

		emailService := email.NewEmailService()
		templateBuilder := email.NewEmailTemplateBuilder()

		expireTimeStr := "未设置"
		if !expireTime.IsZero() {
			expireTimeStr = utils.FormatBeijingTime(expireTime)
		}

		content := templateBuilder.GetUserCreatedTemplate(
			userUsername,
			userEmail,
			plainPassword, // 明文密码
			expireTimeStr,
			deviceLimit,
		)

		_ = emailService.QueueEmail(userEmail, "账户创建通知", content, "user_created")
	}()

	utils.SetResponseStatus(c, http.StatusCreated)
	utils.SuccessResponse(c, http.StatusCreated, "创建成功", user)
}

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
		Notes      *string  `json:"notes"` // 备注
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "用户不存在", err)
		return
	}

	beforeData := map[string]interface{}{
		"username":    user.Username,
		"email":       user.Email,
		"is_active":   user.IsActive,
		"is_verified": user.IsVerified,
		"is_admin":    user.IsAdmin,
		"balance":     user.Balance,
	}

	if req.Username != "" {
		var existing models.User
		if err := db.Where("username = ? AND id != ?", req.Username, id).First(&existing).Error; err == nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "用户名已被使用", nil)
			return
		}
		user.Username = req.Username
	}

	if req.Email != "" {
		req.Email = utils.NormalizeEmail(req.Email)
		var existing models.User
		if err := db.Where("LOWER(email) = ? AND id != ?", req.Email, id).First(&existing).Error; err == nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "邮箱已被使用", nil)
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
	var oldBalance float64
	if req.Balance != nil {
		oldBalance = user.Balance
		user.Balance = *req.Balance
	}
	if req.Password != "" {
		hashedPassword, err := auth.HashPassword(req.Password)
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "密码加密失败", err)
			return
		}
		user.Password = hashedPassword
	}
	if req.Notes != nil {
		if *req.Notes == "" {
			user.Notes = sql.NullString{Valid: false}
		} else {
			user.Notes = database.NullString(*req.Notes)
		}
	}

	if err := db.Save(&user).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新失败", err)
		return
	}

	afterData := map[string]interface{}{
		"username":    user.Username,
		"email":       user.Email,
		"is_active":   user.IsActive,
		"is_verified": user.IsVerified,
		"is_admin":    user.IsAdmin,
		"balance":     user.Balance,
	}

	description := fmt.Sprintf("管理员更新用户: %s (%s)", user.Username, user.Email)
	if req.Password != "" {
		description += " (包含密码重置)"
	}
	utils.CreateAuditLogWithData(c, "update_user", "user", user.ID, description, beforeData, afterData)

	// 如果余额有变更，记录余额日志
	if req.Balance != nil && oldBalance != user.Balance {
		go func() {
			adminUser, _ := middleware.GetCurrentUser(c)
			var operatorUserID *uint
			operator := "system"
			if adminUser != nil {
				operator = adminUser.Username
				operatorUserID = &adminUser.ID
			}
			amount := user.Balance - oldBalance
			ipAddress := utils.GetRealClientIP(c)
			utils.CreateBalanceLog(
				user.ID,
				"admin_adjust",
				amount,
				oldBalance,
				user.Balance,
				nil,
				nil,
				fmt.Sprintf("管理员调整余额: %s", operator),
				operator,
				operatorUserID,
				ipAddress,
			)
		}()
	}

	utils.SetResponseStatus(c, http.StatusOK)
	utils.SuccessResponse(c, http.StatusOK, "更新成功", user)
}

func DeleteUser(c *gin.Context) {
	id := c.Param("id")

	if id == "" || id == "0" {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的用户ID", nil)
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "用户不存在", err)
		return
	}

	userData := map[string]interface{}{
		"id":          user.ID,
		"username":    user.Username,
		"email":       user.Email,
		"is_admin":    user.IsAdmin,
		"is_active":   user.IsActive,
		"is_verified": user.IsVerified,
	}

	if user.IsAdmin {
		var adminCount int64
		db.Model(&models.User{}).Where("is_admin = ? AND id != ?", true, id).Count(&adminCount)
		if adminCount == 0 {
			utils.ErrorResponse(c, http.StatusBadRequest, "不能删除最后一个管理员", nil)
			return
		}
	}

	tx := db.Begin()
	if err := tx.Where("user_id = ?", user.ID).Delete(&models.Subscription{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete subscriptions failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		utils.CreateBusinessLog(c, "delete_user_failed", "删除用户失败: 删除用户订阅失败", "error", map[string]interface{}{
			"target_user_id": user.ID, "step": "subscriptions", "reason": err.Error(),
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户订阅失败", err)
		return
	}

	if err := tx.Where("subscription_id IN (SELECT id FROM subscriptions WHERE user_id = ?)", user.ID).Delete(&models.Device{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete devices by subscription failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户设备失败", err)
		return
	}

	if err := tx.Where("user_id = ?", user.ID).Delete(&models.Device{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete devices by user_id failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户设备失败", err)
		return
	}

	if err := tx.Where("user_id = ?", user.ID).Delete(&models.SubscriptionReset{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete subscription resets failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户订阅重置记录失败", err)
		return
	}

	if err := tx.Where("user_id = ?", user.ID).Delete(&models.Order{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete orders failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户订单失败", err)
		return
	}

	if err := tx.Where("user_id = ?", user.ID).Delete(&models.PaymentTransaction{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete payment transactions failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户支付记录失败", err)
		return
	}

	if err := tx.Where("user_id = ?", user.ID).Delete(&models.RechargeRecord{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete recharge records failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户充值记录失败", err)
		return
	}

	if err := tx.Where("user_id = ?", user.ID).Delete(&models.TicketReply{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete ticket replies failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户工单回复失败", err)
		return
	}

	if err := tx.Where("user_id = ?", user.ID).Delete(&models.Ticket{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete tickets failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户工单失败", err)
		return
	}

	if err := tx.Where("user_id = ?", user.ID).Delete(&models.Notification{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete notifications failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户通知失败", err)
		return
	}

	if err := tx.Where("user_id = ?", user.ID).Delete(&models.UserActivity{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete user activities failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户活动记录失败", err)
		return
	}

	if err := tx.Where("user_id = ?", user.ID).Delete(&models.LoginHistory{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete login history failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户登录历史失败", err)
		return
	}

	if err := tx.Model(&models.InviteCode{}).Where("user_id = ? AND used_count = 0", user.ID).Delete(&models.InviteCode{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete invite codes failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户邀请码失败", err)
		return
	}
	if err := tx.Model(&models.InviteCode{}).Where("user_id = ? AND used_count > 0", user.ID).Update("is_active", false).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: disable invite codes failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "禁用用户邀请码失败", err)
		return
	}

	if err := tx.Where("inviter_id = ?", user.ID).Delete(&models.InviteRelation{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete invite relations as inviter failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户邀请关系失败", err)
		return
	}

	if err := tx.Where("invitee_id = ?", user.ID).Delete(&models.InviteRelation{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete invite relations as invitee failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户被邀请关系失败", err)
		return
	}

	if err := tx.Delete(&user).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete user failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		utils.CreateBusinessLog(c, "delete_user_failed", "删除用户失败: 删除用户记录失败", "error", map[string]interface{}{
			"target_user_id": user.ID, "step": "delete_user", "reason": err.Error(),
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户失败", err)
		return
	}

	if err := tx.Commit().Error; err != nil {
		utils.LogError("DeleteUser: commit transaction failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		utils.CreateBusinessLog(c, "delete_user_failed", "删除用户失败: 提交事务失败", "error", map[string]interface{}{
			"target_user_id": user.ID, "step": "commit", "reason": err.Error(),
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除操作失败", err)
		return
	}

	utils.CreateAuditLogWithData(c, "delete_user", "user", user.ID,
		fmt.Sprintf("管理员删除用户: %s (%s)", user.Username, user.Email), userData, nil)

	go func() {
		emailService := email.NewEmailService()
		templateBuilder := email.NewEmailTemplateBuilder()
		deletionDate := utils.FormatBeijingTime(utils.GetBeijingTime())
		reason := "管理员删除"
		dataRetentionPeriod := "30天"
		content := templateBuilder.GetAccountDeletionTemplate(user.Username, deletionDate, reason, dataRetentionPeriod)
		subject := "账号删除确认"
		_ = emailService.QueueEmail(user.Email, subject, content, "account_deletion")
	}()

	utils.SetResponseStatus(c, http.StatusOK)
	utils.SuccessResponse(c, http.StatusOK, "用户及其所有相关数据已成功删除", nil)
}

func LoginAsUser(c *gin.Context) {
	userID := c.Param("id")
	db := database.GetDB()

	var targetUser models.User
	if err := db.First(&targetUser, userID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "用户不存在", err)
		return
	}

	accessToken, err := utils.CreateAccessToken(targetUser.ID, targetUser.Email, targetUser.IsAdmin)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成令牌失败", err)
		return
	}

	refreshToken, err := utils.CreateRefreshToken(targetUser.ID, targetUser.Email)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成刷新令牌失败", err)
		return
	}

	utils.CreateSecurityLog(c, "admin_login_as", "MEDIUM",
		fmt.Sprintf("管理员以用户身份登录: %s (ID: %d)", targetUser.Username, targetUser.ID),
		map[string]interface{}{"target_user_id": targetUser.ID, "target_username": targetUser.Username})

	utils.SuccessResponse(c, http.StatusOK, "登录成功", gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "bearer",
		"user": gin.H{
			"id":       targetUser.ID,
			"username": targetUser.Username,
			"email":    targetUser.Email,
			"is_admin": targetUser.IsAdmin,
		},
	})
}

func UpdateUserStatus(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status     string `json:"status"`
		IsActive   *bool  `json:"is_active"`
		IsVerified *bool  `json:"is_verified"`
		IsAdmin    *bool  `json:"is_admin"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "用户不存在", err)
		return
	}

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
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新用户状态失败", err)
		return
	}

	// 记录启用/禁用操作到系统日志
	if req.IsActive != nil {
		if *req.IsActive {
			utils.CreateSecurityLog(c, "user_enabled", "INFO",
				fmt.Sprintf("管理员启用用户: %s (ID: %d)", user.Username, user.ID),
				map[string]interface{}{"target_user_id": user.ID, "target_username": user.Username})
		} else {
			utils.CreateSecurityLog(c, "user_disabled", "MEDIUM",
				fmt.Sprintf("管理员禁用用户: %s (ID: %d)", user.Username, user.ID),
				map[string]interface{}{"target_user_id": user.ID, "target_username": user.Username})
		}
	} else if req.Status != "" {
		if req.Status == "active" {
			utils.CreateSecurityLog(c, "user_enabled", "INFO",
				fmt.Sprintf("管理员启用用户: %s (ID: %d)", user.Username, user.ID),
				map[string]interface{}{"target_user_id": user.ID, "target_username": user.Username})
		} else if req.Status == "inactive" || req.Status == "disabled" {
			utils.CreateSecurityLog(c, "user_disabled", "MEDIUM",
				fmt.Sprintf("管理员禁用用户: %s (ID: %d)", user.Username, user.ID),
				map[string]interface{}{"target_user_id": user.ID, "target_username": user.Username})
		}
	}
	utils.CreateAuditLogSimple(c, "update_user_status", "user", user.ID, fmt.Sprintf("管理员操作: 更新用户状态 %s (ID=%d)", user.Username, user.ID))
	utils.SuccessResponse(c, http.StatusOK, "用户状态已更新", user)
}

func UnlockUserLogin(c *gin.Context) {
	id := c.Param("id")

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "用户不存在", err)
		return
	}

	result := db.Where("username = ? OR username = ?", user.Username, user.Email).
		Where("success = ?", false).
		Delete(&models.LoginAttempt{})

	var loginHistories []models.LoginHistory
	db.Where("user_id = ? AND ip_address IS NOT NULL", user.ID).
		Order("login_time DESC").
		Limit(10).
		Find(&loginHistories)

	var auditLogs []models.AuditLog
	db.Where("user_id = ? AND ip_address IS NOT NULL AND action_type LIKE ?",
		user.ID, "security_login%").
		Order("created_at DESC").
		Limit(10).
		Find(&auditLogs)

	ipSet := make(map[string]bool)
	for _, history := range loginHistories {
		if history.IPAddress.Valid && history.IPAddress.String != "" {
			ipSet[history.IPAddress.String] = true
		}
	}
	for _, log := range auditLogs {
		if log.IPAddress.Valid && log.IPAddress.String != "" {
			ipSet[log.IPAddress.String] = true
		}
	}

	ipCount := 0
	for ip := range ipSet {
		middleware.ResetLoginAttempt(ip)
		ipCount++
	}

	user.IsActive = true

	if err := db.Save(&user).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "解锁用户失败", err)
		return
	}

	message := fmt.Sprintf("用户已解锁，清除了 %d 条登录失败记录", result.RowsAffected)
	if ipCount > 0 {
		message += fmt.Sprintf("，已清除 %d 个IP地址的速率限制", ipCount)
	}

	utils.CreateSecurityLog(c, "user_unlock", "INFO",
		fmt.Sprintf("管理员解禁用户: %s (ID: %d)，清除 %d 条失败记录，%d 个IP限流", user.Username, user.ID, result.RowsAffected, ipCount),
		map[string]interface{}{"target_user_id": user.ID, "target_username": user.Username, "cleared_attempts": result.RowsAffected, "ips_reset": ipCount})

	utils.SuccessResponse(c, http.StatusOK, message, nil)
}

func BatchDeleteUsers(c *gin.Context) {
	var req struct {
		UserIDs []uint `json:"user_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	if len(req.UserIDs) == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "请选择要删除的用户", nil)
		return
	}

	currentUser, _ := middleware.GetCurrentUser(c)
	if currentUser != nil {
		for _, id := range req.UserIDs {
			if id == currentUser.ID {
				utils.ErrorResponse(c, http.StatusBadRequest, "不能删除当前登录的管理员账户", nil)
				return
			}
		}
	}

	db := database.GetDB()

	var adminUsers []models.User
	if err := db.Where("id IN ? AND is_admin = ?", req.UserIDs, true).Find(&adminUsers).Error; err == nil && len(adminUsers) > 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "不能删除管理员用户", nil)
		return
	}

	tx := db.Begin()

	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.Subscription{}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户订阅失败", err)
		return
	}

	if err := tx.Where("subscription_id IN (SELECT id FROM subscriptions WHERE user_id IN ?)", req.UserIDs).Delete(&models.Device{}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户设备失败", err)
		return
	}

	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.Device{}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户设备失败", err)
		return
	}

	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.SubscriptionReset{}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户订阅重置记录失败", err)
		return
	}

	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.Order{}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户订单失败", err)
		return
	}

	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.PaymentTransaction{}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户支付记录失败", err)
		return
	}

	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.RechargeRecord{}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户充值记录失败", err)
		return
	}

	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.TicketReply{}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户工单回复失败", err)
		return
	}

	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.Ticket{}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户工单失败", err)
		return
	}

	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.Notification{}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户通知失败", err)
		return
	}

	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.UserActivity{}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户活动记录失败", err)
		return
	}

	if err := tx.Where("user_id IN ?", req.UserIDs).Delete(&models.LoginHistory{}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户登录历史失败", err)
		return
	}

	if err := tx.Where("user_id IN ? AND used_count = 0", req.UserIDs).Delete(&models.InviteCode{}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户邀请码失败", err)
		return
	}
	if err := tx.Model(&models.InviteCode{}).Where("user_id IN ? AND used_count > 0", req.UserIDs).Update("is_active", false).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "禁用用户邀请码失败", err)
		return
	}

	if err := tx.Where("inviter_id IN ?", req.UserIDs).Delete(&models.InviteRelation{}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户邀请关系失败", err)
		return
	}

	if err := tx.Where("invitee_id IN ?", req.UserIDs).Delete(&models.InviteRelation{}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户被邀请关系失败", err)
		return
	}

	if err := tx.Where("id IN ?", req.UserIDs).Delete(&models.User{}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户失败", err)
		return
	}

	if err := tx.Commit().Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除操作失败", err)
		return
	}
	utils.CreateAuditLogSimple(c, "batch_delete_users", "user", 0, fmt.Sprintf("管理员操作: 批量删除用户 %d 个", len(req.UserIDs)))
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功删除 %d 个用户", len(req.UserIDs)), nil)
}

func BatchEnableUsers(c *gin.Context) {
	var req struct {
		UserIDs []uint `json:"user_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	if len(req.UserIDs) == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "请选择要启用的用户", nil)
		return
	}

	db := database.GetDB()
	result := db.Model(&models.User{}).Where("id IN ?", req.UserIDs).Update("is_active", true)

	if result.Error != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "启用用户失败", result.Error)
		return
	}
	utils.CreateAuditLogSimple(c, "batch_enable_users", "user", 0, fmt.Sprintf("管理员操作: 批量启用用户 %d 个", result.RowsAffected))
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功启用 %d 个用户", result.RowsAffected), nil)
}

func BatchDisableUsers(c *gin.Context) {
	var req struct {
		UserIDs []uint `json:"user_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	if len(req.UserIDs) == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "请选择要禁用的用户", nil)
		return
	}

	currentUser, _ := middleware.GetCurrentUser(c)
	if currentUser != nil {
		for _, id := range req.UserIDs {
			if id == currentUser.ID {
				utils.ErrorResponse(c, http.StatusBadRequest, "不能禁用当前登录的管理员账户", nil)
				return
			}
		}
	}

	db := database.GetDB()

	var adminUsers []models.User
	if err := db.Where("id IN ? AND is_admin = ?", req.UserIDs, true).Find(&adminUsers).Error; err == nil && len(adminUsers) > 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "不能禁用管理员用户", nil)
		return
	}

	result := db.Model(&models.User{}).Where("id IN ?", req.UserIDs).Update("is_active", false)

	if result.Error != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "禁用用户失败", result.Error)
		return
	}
	utils.CreateAuditLogSimple(c, "batch_disable_users", "user", 0, fmt.Sprintf("管理员操作: 批量禁用用户 %d 个", result.RowsAffected))
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功禁用 %d 个用户", result.RowsAffected), nil)
}

func BatchSendSubEmail(c *gin.Context) {
	var req struct {
		UserIDs []uint `json:"user_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	if len(req.UserIDs) == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "请选择要发送邮件的用户", nil)
		return
	}

	db := database.GetDB()
	var users []models.User
	if err := db.Where("id IN ?", req.UserIDs).Find(&users).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取用户信息失败", err)
		return
	}

	successCount := 0
	failCount := 0

	for _, user := range users {
		var sub models.Subscription
		if err := db.Where("user_id = ?", user.ID).First(&sub).Error; err != nil {
			failCount++
			continue
		}

		if err := queueSubEmail(c, sub, user); err != nil {
			failCount++
			continue
		}
		successCount++
	}
	utils.CreateAuditLogSimple(c, "batch_send_sub_email", "user", 0, fmt.Sprintf("管理员操作: 批量发送订阅邮件 成功 %d 失败 %d", successCount, failCount))
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功发送 %d 封邮件，失败 %d 封", successCount, failCount), gin.H{
		"success_count": successCount,
		"fail_count":    failCount,
	})
}

func BatchSendExpireReminder(c *gin.Context) {
	var req struct {
		UserIDs []uint `json:"user_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	if len(req.UserIDs) == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "请选择要发送提醒的用户", nil)
		return
	}

	db := database.GetDB()
	var users []models.User
	if err := db.Where("id IN ?", req.UserIDs).Preload("Subscriptions").Find(&users).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取用户信息失败", err)
		return
	}

	emailService := email.NewEmailService()
	templateBuilder := email.NewEmailTemplateBuilder()
	successCount := 0
	failCount := 0
	now := utils.GetBeijingTime()

	for _, user := range users {
		if len(user.Subscriptions) == 0 {
			failCount++
			continue
		}

		sub := user.Subscriptions[0]
		if sub.ExpireTime.IsZero() {
			failCount++
			continue
		}

		daysUntilExpire := int(sub.ExpireTime.Sub(now).Hours() / 24)
		if daysUntilExpire < 0 {
			daysUntilExpire = 0
		}

		subject := "订阅即将到期提醒"
		pkgName := "默认套餐"
		if sub.PackageID != nil {
			var pkg models.Package
			if err := db.First(&pkg, *sub.PackageID).Error; err == nil {
				pkgName = pkg.Name
			}
		}
		isExpired := daysUntilExpire <= 0
		content := templateBuilder.GetExpirationReminderTemplate(
			user.Username,
			pkgName,
			utils.FormatBeijingDate(sub.ExpireTime),
			daysUntilExpire,
			sub.DeviceLimit,
			sub.CurrentDevices,
			isExpired,
		)

		if err := emailService.QueueEmail(user.Email, subject, content, "expiration_reminder"); err != nil {
			failCount++
			continue
		}
		successCount++
	}
	utils.CreateAuditLogSimple(c, "batch_send_expire_reminder", "user", 0, fmt.Sprintf("管理员操作: 批量发送到期提醒 成功 %d 失败 %d", successCount, failCount))
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功发送 %d 封提醒邮件，失败 %d 封", successCount, failCount), gin.H{
		"success_count": successCount,
		"fail_count":    failCount,
	})
}

// getCurrentUserOrError 辅助函数：获取当前用户或返回错误
func getCurrentUserOrError(c *gin.Context) (*models.User, bool) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return nil, false
	}
	return user, true
}

// ==========================================
// 用户个人资料管理（从 user_profile.go 合并）
// ==========================================

type UpdatePreferencesRequest struct {
	Theme              string `json:"theme"`
	Language           string `json:"language"`
	Timezone           string `json:"timezone"`
	EmailNotifications *bool  `json:"email_notifications"`
	SMSNotifications   *bool  `json:"sms_notifications"`
	PushNotifications  *bool  `json:"push_notifications"`
}

func UpdatePreferences(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()

	if req.Theme != "" {
		user.Theme = req.Theme
	}
	if req.Language != "" {
		user.Language = req.Language
	}
	if req.Timezone != "" {
		user.Timezone = req.Timezone
	}
	if req.EmailNotifications != nil {
		user.EmailNotifications = *req.EmailNotifications
	}
	if req.SMSNotifications != nil {
		user.SMSNotifications = *req.SMSNotifications
	}
	if req.PushNotifications != nil {
		user.PushNotifications = *req.PushNotifications
	}

	if err := db.Save(&user).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "更新成功", user)
}

func getUserConfigs(db *gorm.DB, userID uint, category string, keys []string) map[string]string {
	configs := make(map[string]string, len(keys))

	if len(keys) == 0 {
		return configs
	}

	keyPatterns := make([]string, len(keys))
	prefix := fmt.Sprintf("user_%d_", userID)
	for i, key := range keys {
		keyPatterns[i] = prefix + key
	}

	var dbConfigs []models.SystemConfig
	db.Where("category = ? AND key IN (?)", category, keyPatterns).Find(&dbConfigs)

	for _, config := range dbConfigs {
		key := strings.TrimPrefix(config.Key, prefix)
		if key != config.Key {
			configs[key] = config.Value
		}
	}

	return configs
}

func updateUserConfig(db *gorm.DB, userID uint, category, key, value string) error {
	configKey := fmt.Sprintf("user_%d_%s", userID, key)
	var config models.SystemConfig

	err := db.Where("key = ? AND category = ?", configKey, category).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			config = models.SystemConfig{
				Key:      configKey,
				Category: category,
				Value:    value,
			}
			return db.Create(&config).Error
		}
		return err
	}

	config.Value = value
	return db.Save(&config).Error
}

func buildProfileResponse(user *models.User, configs map[string]string) gin.H {
	displayName := configs["display_name"]
	if displayName == "" {
		displayName = user.Username
	}

	return gin.H{
		"id":           user.ID,
		"username":     user.Username,
		"email":        user.Email,
		"is_admin":     user.IsAdmin,
		"avatar_url":   user.Avatar.String,
		"avatar":       user.Avatar.String,
		"display_name": displayName,
		"phone":        configs["phone"],
		"bio":          configs["bio"],
		"theme":        user.Theme,
		"language":     user.Language,
	}
}

func GetAdminProfile(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	db := database.GetDB()
	configs := getUserConfigs(db, user.ID, "admin_profile", []string{"display_name", "phone", "bio"})

	utils.SuccessResponse(c, http.StatusOK, "", buildProfileResponse(user, configs))
}

func UpdateAdminProfile(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	var req struct {
		DisplayName string `json:"display_name"`
		AvatarURL   string `json:"avatar_url"`
		Avatar      string `json:"avatar"`
		Phone       string `json:"phone"`
		Bio         string `json:"bio"`
		Theme       string `json:"theme"`
		Language    string `json:"language"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()

	if req.AvatarURL != "" {
		user.Avatar = database.NullString(req.AvatarURL)
	} else if req.Avatar != "" {
		user.Avatar = database.NullString(req.Avatar)
	}

	if req.Theme != "" {
		user.Theme = req.Theme
	}
	if req.Language != "" {
		user.Language = req.Language
	}

	if err := db.Save(user).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新失败", err)
		return
	}

	configUpdates := map[string]string{
		"display_name": req.DisplayName,
		"phone":        req.Phone,
		"bio":          req.Bio,
	}

	for key, value := range configUpdates {
		if value != "" {
			if err := updateUserConfig(db, user.ID, "admin_profile", key, value); err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("更新%s失败", key), err)
				return
			}
		}
	}

	responseConfigs := map[string]string{
		"display_name": req.DisplayName,
		"phone":        req.Phone,
		"bio":          req.Bio,
	}

	utils.SuccessResponse(c, http.StatusOK, "个人资料更新成功", buildProfileResponse(user, responseConfigs))
}

func GetLoginHistory(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	db := database.GetDB()
	var history []models.LoginHistory
	db.Where("user_id = ?", user.ID).Order("login_time DESC").Limit(50).Find(&history)

	historyList := make([]gin.H, 0, len(history))
	for _, h := range history {
		country, city := h.GetLocationInfo()
		status := "success"
		if h.LoginStatus != "" {
			status = h.LoginStatus
		}

		ipAddr := utils.GetNullStringValue(h.IPAddress)
		userAgent := utils.GetNullStringValue(h.UserAgent)
		loginTime := utils.FormatBeijingTime(h.LoginTime)

		historyList = append(historyList, gin.H{
			"id":           h.ID,
			"ip_address":   ipAddr,
			"user_agent":   userAgent,
			"login_time":   loginTime,
			"login_status": status,
			"country":      country,
			"city":         city,
			"location":     h.Location.String,
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", historyList)
}

func GetSecuritySettings(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	db := database.GetDB()
	var configs []models.SystemConfig
	db.Where("category = ? AND key LIKE ?", "user_security", fmt.Sprintf("user_%d_%%", user.ID)).Find(&configs)

	settings := make(map[string]interface{})
	prefix := fmt.Sprintf("user_%d_", user.ID)

	for _, config := range configs {
		key := strings.TrimPrefix(config.Key, prefix)
		if config.Value == "true" || config.Value == "false" {
			settings[key] = config.Value == "true"
		} else {
			settings[key] = config.Value
		}
	}

	if settings["login_notification"] == nil {
		settings["login_notification"] = true
	}
	if settings["notification_email"] == nil {
		settings["notification_email"] = user.Email
	}
	if settings["session_timeout"] == nil {
		settings["session_timeout"] = "120"
	}

	utils.SuccessResponse(c, http.StatusOK, "", settings)
}

func UpdateAdminSecuritySettings(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	for key, value := range req {
		valueStr := fmt.Sprintf("%v", value)
		if err := updateUserConfig(db, user.ID, "user_security", key, valueStr); err != nil {
			utils.LogError("UpdateAdminSecuritySettings: update config failed", err, map[string]interface{}{"key": key})
			utils.ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("更新配置 %s 失败", key), err)
			return
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "安全设置已保存", nil)
}

func GetNotificationSettings(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"email_enabled":         user.EmailNotifications,
		"email_notifications":   user.EmailNotifications,
		"abnormal_login_alert":  user.AbnormalLoginAlertEnabled,
		"system_notification":   true,
		"security_notification": true,
		"frequency":             "realtime",
		"sms_notifications":     user.SMSNotifications,
		"push_notifications":    user.PushNotifications,
		"notification_types":    user.NotificationTypes,
	})
}

func UpdateUserNotificationSettings(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()

	if emailNotifications, ok := req["email_notifications"].(bool); ok {
		user.EmailNotifications = emailNotifications
	} else if emailEnabled, ok := req["email_enabled"].(bool); ok {
		user.EmailNotifications = emailEnabled
	}
	if abnormalLoginAlert, ok := req["abnormal_login_alert"].(bool); ok {
		user.AbnormalLoginAlertEnabled = abnormalLoginAlert
	}

	if notificationTypes, ok := req["notification_types"].([]interface{}); ok {
		typesJSON := ""
		if len(notificationTypes) > 0 {
			typesBytes, _ := json.Marshal(notificationTypes)
			typesJSON = string(typesBytes)
		}
		user.NotificationTypes = typesJSON
	}

	if smsNotifications, ok := req["sms_notifications"].(bool); ok {
		user.SMSNotifications = smsNotifications
	}

	if pushNotifications, ok := req["push_notifications"].(bool); ok {
		user.PushNotifications = pushNotifications
	}

	if err := db.Save(user).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "通知设置已保存", nil)
}

func UpdateAdminNotificationSettings(c *gin.Context) {
	UpdateUserNotificationSettings(c)
}

func GetUserActivities(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	db := database.GetDB()
	var activities []models.UserActivity
	db.Where("user_id = ?", user.ID).Order("created_at DESC").Limit(100).Find(&activities)

	activityList := make([]gin.H, 0, len(activities))
	for _, act := range activities {
		activityList = append(activityList, gin.H{
			"id":            act.ID,
			"activity_type": act.ActivityType,
			"description":   act.Description.String,
			"ip_address":    act.IPAddress.String,
			"created_at":    utils.FormatBeijingTime(act.CreatedAt),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", activityList)
}

func GetSubscriptionResets(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	db := database.GetDB()
	var resets []models.SubscriptionReset
	db.Where("user_id = ?", user.ID).Order("created_at DESC").Limit(50).Find(&resets)

	resetList := make([]gin.H, 0, len(resets))
	for _, reset := range resets {
		resetList = append(resetList, gin.H{
			"id":                  reset.ID,
			"subscription_id":     reset.SubscriptionID,
			"reset_type":          reset.ResetType,
			"reason":              reset.Reason,
			"device_count_before": reset.DeviceCountBefore,
			"device_count_after":  reset.DeviceCountAfter,
			"created_at":          utils.FormatBeijingTime(reset.CreatedAt),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", resetList)
}

func GetUserDevices(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	db := database.GetDB()
	var devices []models.Device
	db.Where("user_id = ?", user.ID).Order("last_access DESC").Find(&devices)

	deviceList := make([]gin.H, 0, len(devices))
	for _, device := range devices {
		deviceList = append(deviceList, gin.H{
			"id":              device.ID,
			"subscription_id": device.SubscriptionID,
			"device_name":     utils.GetStringValue(device.DeviceName),
			"device_type":     utils.GetStringValue(device.DeviceType),
			"ip_address":      utils.GetStringValue(device.IPAddress),
			"is_active":       device.IsActive,
			"last_access":     utils.FormatBeijingTime(device.LastAccess),
			"created_at":      utils.FormatBeijingTime(device.CreatedAt),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", deviceList)
}

func GetPrivacySettings(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"data_sharing": user.DataSharing,
		"analytics":    user.Analytics,
	})
}

func UpdatePrivacySettings(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()

	if dataSharing, ok := req["data_sharing"].(bool); ok {
		user.DataSharing = dataSharing
	}

	if analytics, ok := req["analytics"].(bool); ok {
		user.Analytics = analytics
	}

	if err := db.Save(user).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "隐私设置已保存", nil)
}

// SendEmailToUser 发送邮件给用户
func SendEmailToUser(c *gin.Context) {
	var req struct {
		UserID       uint   `json:"user_id" binding:"required"`
		Email        string `json:"email" binding:"required,email"`
		Subject      string `json:"subject"`
		Content      string `json:"content"`
		EmailType    string `json:"email_type"`
		TemplateID   uint   `json:"template_id"`
		TemplateName string `json:"template_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()

	// 验证用户是否存在
	var user models.User
	if err := db.First(&user, req.UserID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "用户不存在", err)
		return
	}

	subject := req.Subject
	content := req.Content

	// 如果指定了模板，使用模板
	if req.TemplateID > 0 || req.TemplateName != "" {
		var template models.EmailTemplate
		var err error

		if req.TemplateID > 0 {
			err = db.Where("id = ? AND is_active = ?", req.TemplateID, true).First(&template).Error
		} else {
			err = db.Where("name = ? AND is_active = ?", req.TemplateName, true).First(&template).Error
		}

		if err != nil {
			utils.ErrorResponse(c, http.StatusNotFound, "邮件模板不存在或未启用", err)
			return
		}

		subject = template.Subject
		content = template.Content

		// 替换模板变量
		content = strings.ReplaceAll(content, "{username}", user.Username)
		content = strings.ReplaceAll(content, "{email}", user.Email)

		// 获取用户订阅信息
		var subscription models.Subscription
		if err := db.Where("user_id = ? AND is_active = ?", user.ID, true).First(&subscription).Error; err == nil {
			expireDate := subscription.ExpireTime.Format("2006-01-02")
			content = strings.ReplaceAll(content, "{expire_date}", expireDate)

			daysLeft := int(subscription.ExpireTime.Sub(time.Now()).Hours() / 24)
			content = strings.ReplaceAll(content, "{days_left}", fmt.Sprintf("%d", daysLeft))
		}
	}

	// 统一走邮件队列，确保日志可追踪并避免同步阻塞
	emailService := email.NewEmailService()
	emailType := strings.TrimSpace(req.EmailType)
	if emailType == "" {
		emailType = strings.TrimSpace(req.TemplateName)
	}
	if emailType == "" {
		emailType = "admin_manual"
	}
	if err := emailService.QueueEmail(req.Email, subject, content, emailType); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "邮件加入队列失败", err)
		return
	}

	// 记录审计日志
	utils.CreateAuditLogSimple(c, "send_email", "user", req.UserID,
		fmt.Sprintf("向用户 %s 加入邮件队列: %s (模板: %s, 类型: %s)", user.Username, subject, req.TemplateName, emailType))

	utils.SuccessResponse(c, http.StatusOK, "邮件已加入队列", nil)
}

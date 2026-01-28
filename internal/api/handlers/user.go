package handlers

import (
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

	nowUTC := time.Now().UTC()
	var expireTime time.Time
	if durationMonths <= 0 {
		expireTime = time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 23, 59, 59, 0, nowUTC.Location())
	} else {
		expireTime = nowUTC.AddDate(0, durationMonths, 0)
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

	lastLoginStr := ""
	if user.LastLogin.Valid {
		lastLoginStr = user.LastLogin.Time.Format("2006-01-02 15:04:05")
	}

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
		sk := "%" + utils.SanitizeSearchKeyword(kw) + "%"
		query = query.Where("username LIKE ? OR email LIKE ?", sk, sk)
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
	var total int64
	query.Count(&total)
	var users []models.User
	query.Offset(pagination.GetOffset()).Limit(pagination.Size).Order("created_at DESC").Find(&users)

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
				"expire_time":       sub.ExpireTime.Format("2006-01-02 15:04:05"),
				"days_until_expire": daysUntilExpire,
				"is_expired":        isExpired,
			}
		} else {
			subscriptionInfo = nil
		}

		lastLogin := ""
		if u.LastLogin.Valid {
			lastLogin = u.LastLogin.Time.Format("2006-01-02 15:04:05")
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
			"created_at":     u.CreatedAt.Format("2006-01-02 15:04:05"),
			"last_login":     lastLogin,
			"subscription":   subscriptionInfo,
		})
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"users": list, "total": total, "page": page, "size": size})
}

func GetUser(c *gin.Context) {
	db := database.GetDB()
	var u models.User
	if err := db.First(&u, c.Param("id")).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "不存在", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", u)
}

func GetUserDetails(c *gin.Context) {
	db := database.GetDB()
	var u models.User
	if err := db.First(&u, c.Param("id")).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "不存在", err)
		return
	}

	lastLogin := ""
	if u.LastLogin.Valid {
		lastLogin = u.LastLogin.Time.Format("2006-01-02 15:04:05")
	}

	userInfo := gin.H{
		"id":          u.ID,
		"username":    u.Username,
		"email":       u.Email,
		"balance":     u.Balance,
		"is_active":   u.IsActive,
		"is_verified": u.IsVerified,
		"is_admin":    u.IsAdmin,
		"created_at":  u.CreatedAt.Format("2006-01-02 15:04:05"),
		"last_login":  lastLogin,
		"theme":       u.Theme,
		"language":    u.Language,
		"timezone":    u.Timezone,
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

		formattedSubs = append(formattedSubs, gin.H{
			"id":                sub.ID,
			"subscription_url":  sub.SubscriptionURL,
			"status":            sub.Status,
			"is_active":         sub.IsActive,
			"device_limit":      sub.DeviceLimit,
			"current_devices":   sub.CurrentDevices,
			"online_devices":    online,
			"expire_time":       sub.ExpireTime.Format("2006-01-02 15:04:05"),
			"days_until_expire": daysUntilExpire,
			"is_expired":        isExpired,
			"created_at":        sub.CreatedAt.Format("2006-01-02 15:04:05"),
			"apple_count":       universalCount,
			"clash_count":       clashCount,
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
			"created_at": order.CreatedAt.Format("2006-01-02 15:04:05"),
			"updated_at": order.UpdatedAt.Format("2006-01-02 15:04:05"),
		}

		if order.PaymentMethodName.Valid {
			formattedOrder["payment_method"] = order.PaymentMethodName.String
			formattedOrder["payment_method_name"] = order.PaymentMethodName.String
		} else {
			formattedOrder["payment_method"] = nil
			formattedOrder["payment_method_name"] = nil
		}

		if order.PaymentTime.Valid {
			formattedOrder["payment_time"] = order.PaymentTime.Time.Format("2006-01-02 15:04:05")
		} else {
			formattedOrder["payment_time"] = nil
		}

		formattedOrder["payment_transaction_id"] = utils.GetNullStringValue(order.PaymentTransactionID)

		if order.ExpireTime.Valid {
			formattedOrder["expire_time"] = order.ExpireTime.Time.Format("2006-01-02 15:04:05")
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
					return record.PaidAt.Time.Format("2006-01-02 15:04:05")
				}
				return nil
			}(),
			"created_at": record.CreatedAt.Format("2006-01-02 15:04:05"),
			"updated_at": record.UpdatedAt.Format("2006-01-02 15:04:05"),
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
			"created_at":           reset.CreatedAt.Format("2006-01-02 15:04:05"),
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
				"created_at":   d.CreatedAt.Format("2006-01-02 15:04:05"),
				"last_access":  d.LastAccess.Format("2006-01-02 15:04:05"),
				"access_count": d.AccessCount,
			})
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"user_info":        userInfo,
		"subscriptions":    formattedSubs,
		"orders":           formattedOrders,
		"recharge_records": formattedRecharges,
		"statistics": gin.H{
			"total_subscriptions": len(subs),
			"total_orders":        totalOrders,
			"total_resets":        totalResets,
			"total_spent":         totalSpent,
		},
		"subscription_resets": formattedResets,
		"ua_records":          uaRecords,
		"recent_activities":   []gin.H{}, // 预留字段，后续可以添加最近活动记录
	})
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
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError("CreateUser: bind request", err, nil)
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误，请检查输入格式", err)
		return
	}

	db := database.GetDB()

	var existingUser models.User
	if err := db.Where("email = ? OR username = ?", req.Email, req.Username).First(&existingUser).Error; err == nil {
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
				expireTime = time.Now().UTC().AddDate(0, months, 0)
			} else {
				expireTime = parsedTime.UTC()
			}
		} else {
			expireTime = parsedTime.UTC()
		}
	} else {
		months := defaultDurationMonths
		if months <= 0 {
			months = 1
		}
		expireTime = time.Now().UTC().AddDate(0, months, 0)
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
		createTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")

		expireTimeStr := "未设置"
		if !expireTime.IsZero() {
			expireTimeStr = expireTime.Format("2006-01-02 15:04:05")
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
			expireTimeStr = expireTime.Format("2006-01-02 15:04:05")
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
		var existing models.User
		if err := db.Where("email = ? AND id != ?", req.Email, id).First(&existing).Error; err == nil {
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
	if req.Balance != nil {
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
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户失败", err)
		return
	}

	if err := tx.Commit().Error; err != nil {
		utils.LogError("DeleteUser: commit transaction failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除操作失败", err)
		return
	}

	utils.CreateAuditLogWithData(c, "delete_user", "user", user.ID,
		fmt.Sprintf("管理员删除用户: %s (%s)", user.Username, user.Email), userData, nil)

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
			sub.ExpireTime.Format("2006-01-02"),
			daysUntilExpire,
			sub.DeviceLimit,
			sub.CurrentDevices,
			isExpired,
		)

		if err := emailService.QueueEmail(user.Email, subject, content, "expiry_reminder"); err != nil {
			failCount++
			continue
		}
		successCount++
	}

	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功发送 %d 封提醒邮件，失败 %d 封", successCount, failCount), gin.H{
		"success_count": successCount,
		"fail_count":    failCount,
	})
}

package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cboard-go/internal/core/auth"
	"cboard-go/internal/core/cache"
	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func getDefaultSubscriptionSettings(db *gorm.DB) (deviceLimit int, durationMonths int) {
	deviceLimit = 0
	durationMonths = 0

	// 走短 TTL 配置缓存，避免每次创建用户/订阅重复查库（最多 4 次查询/调用）
	if v, err := utils.GetCachedSetting(db, "default_subscription_device_limit", "registration"); err == nil && v != "" {
		if limit, err := strconv.Atoi(v); err == nil && limit >= 0 {
			deviceLimit = limit
		}
	} else if v, err := utils.GetCachedSetting(db, "default_subscription_device_limit", "general"); err == nil && v != "" {
		if limit, err := strconv.Atoi(v); err == nil && limit >= 0 {
			deviceLimit = limit
		}
	}

	if v, err := utils.GetCachedSetting(db, "default_subscription_duration_months", "registration"); err == nil && v != "" {
		if months, err := strconv.Atoi(v); err == nil && months >= 0 {
			durationMonths = months
		}
	} else if v, err := utils.GetCachedSetting(db, "default_subscription_duration_months", "general"); err == nil && v != "" {
		if months, err := strconv.Atoi(v); err == nil && months >= 0 {
			durationMonths = months
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

	c.Header("Cache-Control", "private, max-age=60, must-revalidate")
	utils.SuccessResponse(c, http.StatusOK, "", responseData)
}

func UpdateCurrentUser(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	var req struct {
		Username         string `json:"username"`
		Nickname         string `json:"nickname"`
		Avatar           string `json:"avatar"`
		Theme            string `json:"theme"`
		Language         string `json:"language"`
		Timezone         string `json:"timezone"`
		Email            string `json:"email"`
		VerificationCode string `json:"verification_code"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()

	// 邮箱换绑：需先向新邮箱发送验证码，校验通过后才更新
	if req.Email != "" {
		newEmail := utils.NormalizeEmail(req.Email)
		if newEmail == user.Email {
			utils.ErrorResponse(c, http.StatusBadRequest, "新邮箱与当前邮箱相同", nil)
			return
		}
		if req.VerificationCode == "" {
			utils.ErrorResponse(c, http.StatusBadRequest, "请先获取邮箱验证码", nil)
			return
		}
		var existingUser models.User
		if err := db.Where("LOWER(email) = ? AND id != ?", newEmail, user.ID).First(&existingUser).Error; err == nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "该邮箱已被其他账号使用", nil)
			return
		}
		// 校验验证码（purpose=email_change，未使用且未过期）
		var verificationCode models.VerificationCode
		if err := db.Where("LOWER(email) = ? AND code = ? AND used = ? AND purpose = ?", newEmail, req.VerificationCode, 0, "email_change").
			Order("created_at DESC").First(&verificationCode).Error; err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "验证码错误，请重新获取", nil)
			return
		}
		if verificationCode.IsExpired() {
			utils.ErrorResponse(c, http.StatusBadRequest, "验证码已过期，请重新获取", nil)
			return
		}
		// 事务：更新邮箱 + 标记验证码已用
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&models.User{}).Where("id = ?", user.ID).Update("email", newEmail).Error; err != nil {
				return err
			}
			return tx.Model(&models.VerificationCode{}).Where("id = ?", verificationCode.ID).Update("used", 1).Error
		})
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "邮箱修改失败", err)
			return
		}
		user.Email = newEmail
		utils.CreateAuditLogSimple(c, "change_email", "user", user.ID, fmt.Sprintf("用户修改邮箱: %s", newEmail))
	}

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

	var customNodeCounts []struct {
		UserID          uint
		CustomNodeCount int64
	}
	if len(userIDs) > 0 {
		db.Model(&models.UserCustomNode{}).
			Select("user_id, COUNT(*) as custom_node_count").
			Where("user_id IN ?", userIDs).
			Group("user_id").
			Scan(&customNodeCounts)
	}
	customNodeCountMap := make(map[uint]int64)
	for _, cc := range customNodeCounts {
		customNodeCountMap[cc.UserID] = cc.CustomNodeCount
	}

	list := make([]gin.H, 0, len(users))
	now := utils.GetBeijingTime()
	for _, u := range users {
		sub := subMap[u.ID]
		customNodeCount := customNodeCountMap[u.ID]

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
				daysUntilExpire, isExpired = utils.RemainingDays(sub.ExpireTime, now)
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
			"online_devices":                 online,
			"custom_node_count":              customNodeCount,
			"is_special_node_user":           customNodeCount > 0,
			"special_node_subscription_type": u.SpecialNodeSubscriptionType,
			"created_at":                     utils.FormatBeijingTime(u.CreatedAt),
			"last_login":                     lastLogin,
			"subscription":                   subscriptionInfo,
			"notes":                          notes,
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
		utils.CreateBusinessLogAsync(c, "unauthorized_user_access", "尝试越权访问用户信息", "warning", map[string]interface{}{
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
		utils.CreateBusinessLogAsync(c, "unauthorized_user_details_access", "尝试越权访问用户详细信息", "warning", map[string]interface{}{
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
		var specialNodeCount int64
		db.Model(&models.UserCustomNode{}).Where("user_id = ?", u.ID).Count(&specialNodeCount)
		// 管理员可以看到所有信息
		userInfo = gin.H{
			"id":                             u.ID,
			"username":                       u.Username,
			"email":                          u.Email,
			"balance":                        u.Balance,
			"is_active":                      u.IsActive,
			"is_verified":                    u.IsVerified,
			"is_admin":                       u.IsAdmin,
			"created_at":                     utils.FormatBeijingTime(u.CreatedAt),
			"last_login":                     lastLogin,
			"theme":                          u.Theme,
			"language":                       u.Language,
			"timezone":                       u.Timezone,
			"custom_node_count":              specialNodeCount,
			"is_special_node_user":           specialNodeCount > 0,
			"special_node_subscription_type": u.SpecialNodeSubscriptionType,
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
	db.Where("user_id = ?", u.ID).Preload("Package").Find(&subs)

	// 批量查询在线设备数
	subIDs := make([]uint, len(subs))
	for i, s := range subs {
		subIDs[i] = s.ID
	}
	type SubCount struct {
		SubscriptionID uint  `gorm:"column:subscription_id"`
		Count          int64 `gorm:"column:count"`
	}
	var onlineCounts []SubCount
	if len(subIDs) > 0 {
		db.Model(&models.Device{}).Select("subscription_id, COUNT(*) as count").
			Where("subscription_id IN ? AND is_active = ?", subIDs, true).
			Group("subscription_id").Scan(&onlineCounts)
	}
	onlineMap := make(map[uint]int64)
	for _, oc := range onlineCounts {
		onlineMap[oc.SubscriptionID] = oc.Count
	}

	formattedSubs := make([]gin.H, 0, len(subs))
	for _, sub := range subs {
		online := onlineMap[sub.ID]

		daysUntilExpire := 0
		isExpired := false
		now := utils.GetBeijingTime()
		if !sub.ExpireTime.IsZero() {
			daysUntilExpire, isExpired = utils.RemainingDays(sub.ExpireTime, now)
		}

		universalCount := sub.UniversalCount
		clashCount := sub.ClashCount

		// 生成通用订阅和Clash订阅URL
		universalURL, clashURL := getSubscriptionURLs(c, sub.SubscriptionURL)
		// 多客户端订阅地址
		multiURLs := getMultiClientSubscriptionURLs(c, sub.SubscriptionURL)

		formattedSubs = append(formattedSubs, gin.H{
			"id":                sub.ID,
			"subscription_url":  sub.SubscriptionURL,
			"universal_url":     universalURL,
			"clash_url":         clashURL,
			"stash_url":         multiURLs["stash_url"],
			"surge_url":         multiURLs["surge_url"],
			"quantumultx_url":   multiURLs["quantumultx_url"],
			"loon_url":          multiURLs["loon_url"],
			"singbox_url":       multiURLs["singbox_url"],
			"shadowrocket_url":  multiURLs["shadowrocket_url"],
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
	for _, record := range recharges {
		ipValue := utils.GetNullStringValue(record.IPAddress)
		var ipStr string
		if ipValue != nil {
			ipStr = ipValue.(string)
		}
		ipAddress := utils.FormatIP(ipStr)
		// 列表查询不查询 GeoIP，提升性能
		location := ""
		// if ipAddress != "" && ipAddress != "-" && geoip.IsEnabled() {
		// 	locationStr := geoip.GetLocationString(ipAddress)
		// 	if locationStr.Valid {
		// 		location = locationStr.String
		// 	}
		// }

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

	paymentSummary := utils.CalculateUserPaymentSummary(db, u.ID)

	var totalResets int64
	db.Model(&models.SubscriptionReset{}).Where("user_id = ?", u.ID).Count(&totalResets)

	var resets []models.SubscriptionReset
	db.Where("user_id = ?", u.ID).Order("created_at DESC").Find(&resets)
	formattedResets := make([]gin.H, 0, len(resets))
	for _, reset := range resets {
		formattedResets = append(formattedResets, gin.H{
			"id":                   reset.ID,
			"subscription_id":      reset.SubscriptionID,
			"reset_type":           reset.ResetType,
			"reason":               reset.Reason,
			"old_subscription_url": utils.GetStringValue(reset.OldSubscriptionURL),
			"new_subscription_url": utils.GetStringValue(reset.NewSubscriptionURL),
			"device_count_before":  reset.DeviceCountBefore,
			"device_count_after":   reset.DeviceCountAfter,
			"reset_by":             utils.GetStringValue(reset.ResetBy),
			"created_at":           utils.FormatBeijingTime(reset.CreatedAt),
		})
	}

	uaRecords := make([]gin.H, 0)
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
			ipAddress := utils.FormatIP(utils.GetStringValue(d.IPAddress))
			// 使用数据库中已存储的位置信息，避免实时查询 GeoIP
			location := utils.GetStringValue(d.Location)

			uaRecords = append(uaRecords, gin.H{
				"user_agent":   *d.UserAgent,
				"device_type":  utils.GetStringValue(d.DeviceType),
				"device_name":  utils.GetStringValue(d.DeviceName),
				"ip_address":   ipAddress,
				"location":     utils.FormatLocation(location),
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
		ipAddr = utils.NormalizeIP(ipAddr)
		location := ""
		if lh.Location.Valid {
			location = lh.Location.String
		}
		// 列表查询不查询 GeoIP，提升性能
		// else if ipAddr != "" && geoip.IsEnabled() {
		// 	if loc := geoip.GetLocationString(ipAddr); loc.Valid {
		// 		location = loc.String
		// 	}
		// }
		entry := gin.H{
			"id":           lh.ID,
			"login_time":   utils.FormatBeijingTime(lh.LoginTime),
			"ip_address":   ipAddr,
			"location":     utils.FormatLocation(location),
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

	abnormalDetails := []gin.H{}
	if currentUser.IsAdmin {
		startTime, endTime, minSub, minReset := parseAbnormalDetailFilters(c)
		abnormalDetails = buildUserAbnormalDetails(db, &u, startTime, endTime, minSub, minReset)
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"user_info":        userInfo,
		"subscriptions":    formattedSubs,
		"orders":           formattedOrders,
		"recharge_records": formattedRecharges,
		"checkin_records":  formattedCheckins,
		"statistics": gin.H{
			"total_subscriptions": len(subs),
			"total_orders":        paymentSummary.Total,
			"total_resets":        totalResets,
			"total_spent":         paymentSummary.PaidAmount,
		},
		"subscription_resets": formattedResets,
		"ua_records":          uaRecords,
		"login_history":       formattedLoginHistory,
		"abnormal_details":    abnormalDetails,
	})
}

func parseAbnormalDetailFilters(c *gin.Context) (time.Time, time.Time, int, int) {
	now := utils.GetBeijingTime()
	startTime := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endTime := now

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
	if len(dateRange) == 2 {
		if parsedStart, err := time.ParseInLocation("2006-01-02", dateRange[0], now.Location()); err == nil {
			startTime = parsedStart
		}
		if parsedEnd, err := time.ParseInLocation("2006-01-02", dateRange[1], now.Location()); err == nil {
			endTime = time.Date(parsedEnd.Year(), parsedEnd.Month(), parsedEnd.Day(), 23, 59, 59, 0, parsedEnd.Location())
		}
	}

	minSub, minReset := 10, 3
	if value, err := strconv.Atoi(c.DefaultQuery("subscription_count", "10")); err == nil && value > 0 {
		minSub = value
	}
	if value, err := strconv.Atoi(c.DefaultQuery("reset_count", "3")); err == nil && value > 0 {
		minReset = value
	}

	return startTime, endTime, minSub, minReset
}

func buildUserAbnormalDetails(db *gorm.DB, user *models.User, startTime, endTime time.Time, minSub, minReset int) []gin.H {
	metrics := loadAbnormalUserRiskMetrics(db, []uint{user.ID}, startTime, endTime)
	resetCount := metrics.resetCounts[user.ID]
	subscriptionCount := metrics.subscriptionCounts[user.ID]
	ipCount := metrics.ipCounts[user.ID]
	locationCount := metrics.locationCounts[user.ID]
	loginFailedCount := metrics.loginFailedCounts[user.ID]
	activeDeviceCount := metrics.activeDeviceCounts[user.ID]
	deviceLimit := metrics.deviceLimits[user.ID]

	period := fmt.Sprintf("%s 至 %s", utils.FormatBeijingTime(startTime), utils.FormatBeijingTime(endTime))
	details := make([]gin.H, 0)

	if !user.IsActive {
		details = append(details, gin.H{
			"type":    "disabled",
			"title":   "账户禁用",
			"level":   "danger",
			"summary": "账户当前处于禁用状态",
			"period":  "账户状态",
			"columns": abnormalDetailColumns("字段", "内容"),
			"items": []gin.H{
				{"label": "账户状态", "value": "禁用"},
				{"label": "注册时间", "value": utils.FormatBeijingTime(user.CreatedAt)},
			},
		})
	}
	if deviceLimit > 0 && activeDeviceCount > int64(deviceLimit) {
		details = append(details, buildDeviceOverLimitDetail(db, user.ID, activeDeviceCount, deviceLimit))
	}
	if ipCount >= abnormalIPThreshold {
		details = append(details, buildMultiIPDetail(db, user.ID, ipCount, startTime, endTime, period))
	}
	if locationCount >= abnormalLocationThreshold {
		details = append(details, buildMultiLocationDetail(db, user.ID, locationCount, startTime, endTime, period))
	}
	if loginFailedCount >= abnormalLoginFailedThreshold {
		details = append(details, buildLoginFailedDetail(db, user, loginFailedCount, startTime, endTime, period))
	}
	if resetCount >= int64(minReset) {
		details = append(details, buildFrequentResetDetail(db, user.ID, resetCount, startTime, endTime, period))
	}
	if subscriptionCount >= int64(minSub) {
		details = append(details, buildFrequentSubscriptionDetail(db, user.ID, subscriptionCount, startTime, endTime, period))
	}
	if !user.IsVerified && user.CreatedAt.Before(utils.GetBeijingTime().AddDate(0, 0, -unverifiedAccountAgeDays)) {
		details = append(details, gin.H{
			"type":    "unverified",
			"title":   "邮箱未验证",
			"level":   "warning",
			"summary": fmt.Sprintf("注册超过 %d 天仍未验证邮箱", unverifiedAccountAgeDays),
			"period":  "账户状态",
			"columns": abnormalDetailColumns("字段", "内容"),
			"items": []gin.H{
				{"label": "邮箱", "value": user.Email},
				{"label": "注册时间", "value": utils.FormatBeijingTime(user.CreatedAt)},
				{"label": "验证状态", "value": "未验证"},
			},
		})
	}
	if !user.LastLogin.Valid && user.CreatedAt.Before(utils.GetBeijingTime().AddDate(0, -1, 0)) {
		details = append(details, gin.H{
			"type":    "inactive",
			"title":   "长期未登录",
			"level":   "info",
			"summary": "注册超过 1 个月且从未登录",
			"period":  "账户状态",
			"columns": abnormalDetailColumns("字段", "内容"),
			"items": []gin.H{
				{"label": "注册时间", "value": utils.FormatBeijingTime(user.CreatedAt)},
				{"label": "最后登录", "value": "从未登录"},
			},
		})
	}

	return details
}

func abnormalDetailColumns(firstLabel, secondLabel string) []gin.H {
	return []gin.H{
		{"prop": "label", "label": firstLabel, "width": 140},
		{"prop": "value", "label": secondLabel, "min_width": 240},
	}
}

func buildMultiIPDetail(db *gorm.DB, userID uint, count int64, startTime, endTime time.Time, period string) gin.H {
	var rows []models.LoginHistory
	db.Where("user_id = ? AND login_status = ? AND ip_address IS NOT NULL AND ip_address != '' AND login_time >= ? AND login_time <= ?", userID, "success", startTime, endTime).
		Order("login_time DESC").
		Limit(100).
		Find(&rows)

	var deviceRows []models.Device
	db.Where("user_id = ? AND is_active = ? AND ip_address IS NOT NULL AND ip_address != '' AND last_access >= ? AND last_access <= ?", userID, true, startTime, endTime).
		Order("last_access DESC").
		Limit(100).
		Find(&deviceRows)

	items := make([]gin.H, 0, len(rows)+len(deviceRows))
	for _, row := range rows {
		items = append(items, gin.H{
			"time":       utils.FormatBeijingTime(row.LoginTime),
			"source":     "登录",
			"ip_address": normalizeNullableIP(row.IPAddress),
			"location":   utils.FormatLocation(utils.NullStringValue(row.Location)),
			"user_agent": utils.NullStringValue(row.UserAgent),
		})
	}
	for _, row := range deviceRows {
		items = append(items, gin.H{
			"time":       utils.FormatBeijingTime(row.LastAccess),
			"source":     "设备访问",
			"ip_address": normalizePointerIP(row.IPAddress),
			"location":   utils.FormatLocation(utils.GetStringValue(row.Location)),
			"user_agent": utils.GetStringValue(row.UserAgent),
		})
	}

	return gin.H{
		"type":    "multi_ip",
		"title":   "多 IP 访问",
		"level":   "danger",
		"summary": fmt.Sprintf("时间段内出现 %d 个不同 IP，以下为最近成功登录记录", count),
		"period":  period,
		"columns": []gin.H{
			{"prop": "time", "label": "时间", "width": 170},
			{"prop": "source", "label": "来源", "width": 100},
			{"prop": "ip_address", "label": "IP 地址", "width": 150},
			{"prop": "location", "label": "地点", "min_width": 180},
			{"prop": "user_agent", "label": "User-Agent", "min_width": 240},
		},
		"items": items,
	}
}

func buildMultiLocationDetail(db *gorm.DB, userID uint, count int64, startTime, endTime time.Time, period string) gin.H {
	var rows []models.LoginHistory
	db.Where("user_id = ? AND login_status = ? AND location IS NOT NULL AND location != '' AND login_time >= ? AND login_time <= ?", userID, "success", startTime, endTime).
		Order("login_time DESC").
		Limit(100).
		Find(&rows)

	var deviceRows []models.Device
	db.Where("user_id = ? AND is_active = ? AND location IS NOT NULL AND location != '' AND last_access >= ? AND last_access <= ?", userID, true, startTime, endTime).
		Order("last_access DESC").
		Limit(100).
		Find(&deviceRows)

	items := make([]gin.H, 0, len(rows)+len(deviceRows))
	for _, row := range rows {
		items = append(items, gin.H{
			"time":       utils.FormatBeijingTime(row.LoginTime),
			"source":     "登录",
			"location":   utils.FormatLocation(utils.NullStringValue(row.Location)),
			"ip_address": normalizeNullableIP(row.IPAddress),
			"user_agent": utils.NullStringValue(row.UserAgent),
		})
	}
	for _, row := range deviceRows {
		items = append(items, gin.H{
			"time":       utils.FormatBeijingTime(row.LastAccess),
			"source":     "设备访问",
			"location":   utils.FormatLocation(utils.GetStringValue(row.Location)),
			"ip_address": normalizePointerIP(row.IPAddress),
			"user_agent": utils.GetStringValue(row.UserAgent),
		})
	}

	return gin.H{
		"type":    "multi_location",
		"title":   "多地区访问",
		"level":   "warning",
		"summary": fmt.Sprintf("时间段内出现 %d 个不同地区，以下为具体登录地点和时间", count),
		"period":  period,
		"columns": []gin.H{
			{"prop": "time", "label": "时间", "width": 170},
			{"prop": "source", "label": "来源", "width": 100},
			{"prop": "location", "label": "地点", "min_width": 180},
			{"prop": "ip_address", "label": "IP 地址", "width": 150},
			{"prop": "user_agent", "label": "User-Agent", "min_width": 240},
		},
		"items": items,
	}
}

func buildLoginFailedDetail(db *gorm.DB, user *models.User, count int64, startTime, endTime time.Time, period string) gin.H {
	var rows []models.LoginAttempt
	db.Where("(lower(username) = lower(?) OR lower(username) = lower(?)) AND success = ? AND created_at >= ? AND created_at <= ?",
		user.Email, user.Username, false, startTime, endTime).
		Order("created_at DESC").
		Limit(100).
		Find(&rows)

	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, gin.H{
			"time":       utils.FormatBeijingTime(row.CreatedAt),
			"username":   row.Username,
			"ip_address": normalizeNullableIP(row.IPAddress),
			"user_agent": utils.NullStringValue(row.UserAgent),
		})
	}

	return gin.H{
		"type":    "login_failed",
		"title":   "登录失败过多",
		"level":   "warning",
		"summary": fmt.Sprintf("时间段内登录失败 %d 次", count),
		"period":  period,
		"columns": []gin.H{
			{"prop": "time", "label": "失败时间", "width": 170},
			{"prop": "username", "label": "登录账号", "min_width": 160},
			{"prop": "ip_address", "label": "IP 地址", "width": 150},
			{"prop": "user_agent", "label": "User-Agent", "min_width": 240},
		},
		"items": items,
	}
}

func buildFrequentResetDetail(db *gorm.DB, userID uint, count int64, startTime, endTime time.Time, period string) gin.H {
	var rows []models.SubscriptionReset
	db.Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, startTime, endTime).
		Order("created_at DESC").
		Limit(100).
		Find(&rows)

	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, gin.H{
			"time":                utils.FormatBeijingTime(row.CreatedAt),
			"subscription_id":     row.SubscriptionID,
			"reset_type":          row.ResetType,
			"reason":              row.Reason,
			"device_count_before": row.DeviceCountBefore,
			"device_count_after":  row.DeviceCountAfter,
			"reset_by":            utils.GetStringValue(row.ResetBy),
		})
	}

	return gin.H{
		"type":    "frequent_reset",
		"title":   "频繁重置",
		"level":   "warning",
		"summary": fmt.Sprintf("时间段内重置订阅 %d 次", count),
		"period":  period,
		"columns": []gin.H{
			{"prop": "time", "label": "重置时间", "width": 170},
			{"prop": "subscription_id", "label": "订阅ID", "width": 90},
			{"prop": "reset_type", "label": "类型", "width": 120},
			{"prop": "reason", "label": "原因", "min_width": 180},
			{"prop": "device_count_before", "label": "重置前设备", "width": 110},
			{"prop": "device_count_after", "label": "重置后设备", "width": 110},
			{"prop": "reset_by", "label": "操作者", "width": 120},
		},
		"items": items,
	}
}

func buildFrequentSubscriptionDetail(db *gorm.DB, userID uint, count int64, startTime, endTime time.Time, period string) gin.H {
	var rows []models.Subscription
	db.Preload("Package").
		Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, startTime, endTime).
		Order("created_at DESC").
		Limit(100).
		Find(&rows)

	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		packageName := ""
		if row.Package.ID > 0 {
			packageName = row.Package.Name
		}
		items = append(items, gin.H{
			"time":            utils.FormatBeijingTime(row.CreatedAt),
			"subscription_id": row.ID,
			"package_name":    packageName,
			"status":          row.Status,
			"device_limit":    row.DeviceLimit,
			"expire_time":     utils.FormatBeijingTime(row.ExpireTime),
		})
	}

	return gin.H{
		"type":    "frequent_subscription",
		"title":   "频繁创建订阅",
		"level":   "danger",
		"summary": fmt.Sprintf("时间段内创建订阅 %d 次", count),
		"period":  period,
		"columns": []gin.H{
			{"prop": "time", "label": "创建时间", "width": 170},
			{"prop": "subscription_id", "label": "订阅ID", "width": 90},
			{"prop": "package_name", "label": "套餐", "min_width": 140},
			{"prop": "status", "label": "状态", "width": 100},
			{"prop": "device_limit", "label": "设备限制", "width": 100},
			{"prop": "expire_time", "label": "到期时间", "width": 170},
		},
		"items": items,
	}
}

func buildDeviceOverLimitDetail(db *gorm.DB, userID uint, activeDeviceCount int64, deviceLimit int) gin.H {
	var rows []models.Device
	db.Table("devices").
		Select("devices.*").
		Joins("JOIN subscriptions ON subscriptions.id = devices.subscription_id").
		Where("subscriptions.user_id = ? AND subscriptions.is_active = ? AND devices.is_active = ?", userID, true, true).
		Order("devices.last_access DESC").
		Limit(100).
		Find(&rows)

	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, gin.H{
			"subscription_id": row.SubscriptionID,
			"device_name":     utils.GetStringValue(row.DeviceName),
			"device_type":     utils.GetStringValue(row.DeviceType),
			"ip_address":      normalizePointerIP(row.IPAddress),
			"location":        utils.FormatLocation(utils.GetStringValue(row.Location)),
			"last_access":     utils.FormatBeijingTime(row.LastAccess),
			"access_count":    row.AccessCount,
		})
	}

	return gin.H{
		"type":    "device_over_limit",
		"title":   "设备超限",
		"level":   "danger",
		"summary": fmt.Sprintf("活跃设备 %d 台，超过限制 %d 台", activeDeviceCount, deviceLimit),
		"period":  "当前活跃设备",
		"columns": []gin.H{
			{"prop": "subscription_id", "label": "订阅ID", "width": 90},
			{"prop": "device_name", "label": "设备名称", "min_width": 140},
			{"prop": "device_type", "label": "设备类型", "width": 120},
			{"prop": "ip_address", "label": "IP 地址", "width": 150},
			{"prop": "location", "label": "地点", "min_width": 160},
			{"prop": "last_access", "label": "最后访问", "width": 170},
			{"prop": "access_count", "label": "访问次数", "width": 100},
		},
		"items": items,
	}
}

func normalizeNullableIP(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return utils.NormalizeIP(value.String)
}

func normalizePointerIP(value *string) string {
	if value == nil {
		return ""
	}
	return utils.NormalizeIP(*value)
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
		Password    string  `json:"password" binding:"required"`
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

	valid, msg := auth.ValidatePasswordStrength(req.Password, getMinPasswordLength(db))
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
			if err := utils.CreateSubscriptionLog(subscription.ID, user.ID, "create", actionBy, actionByUserID, ipAddress, nil, afterData, "管理员创建用户时自动创建订阅"); err != nil {
				log.Printf("failed to create subscription log: %v", err)
			}
		}()
	}

	utils.CreateAuditLog(c, "create_user", "user", user.ID,
		fmt.Sprintf("管理员创建用户: %s (%s), 管理员权限:%v", user.Username, user.Email, user.IsAdmin), nil, map[string]interface{}{"target_user_id": user.ID, "target_username": user.Username, "target_email": user.Email, "is_admin": user.IsAdmin, "is_active": user.IsActive})

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

		// 管理员通知不含明文密码：管理员本人设置的密码无需回执，
		// 且通知可能流经 Telegram/Bark/邮件等多个渠道，明文密码泄露面过大
		_ = notificationService.SendAdminNotification("user_created", map[string]interface{}{
			"username":     user.Username,
			"email":        user.Email,
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
		Username                    string   `json:"username"`
		Email                       string   `json:"email"`
		IsActive                    *bool    `json:"is_active"`
		IsVerified                  *bool    `json:"is_verified"`
		IsAdmin                     *bool    `json:"is_admin"`
		Balance                     *float64 `json:"balance"`
		Password                    string   `json:"password"`
		Notes                       *string  `json:"notes"` // 备注
		DeviceLimit                 *int     `json:"device_limit"`
		ExpireTime                  *string  `json:"expire_time"`
		SpecialNodeSubscriptionType *string  `json:"special_node_subscription_type"`
	}

	body, err := c.GetRawData()
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}
	if strings.TrimSpace(string(body)) == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", nil)
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}
	var rawPayload map[string]json.RawMessage
	if err := json.Unmarshal(body, &rawPayload); err != nil {
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
		"username":                       user.Username,
		"email":                          user.Email,
		"is_active":                      user.IsActive,
		"is_verified":                    user.IsVerified,
		"is_admin":                       user.IsAdmin,
		"balance":                        user.Balance,
		"special_node_subscription_type": user.SpecialNodeSubscriptionType,
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
	if rawNotes, notesProvided := rawPayload["notes"]; notesProvided {
		if string(rawNotes) == "null" {
			user.Notes = sql.NullString{Valid: false}
		} else {
			var notes string
			if err := json.Unmarshal(rawNotes, &notes); err != nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "备注格式错误", err)
				return
			}
			if notes == "" {
				user.Notes = sql.NullString{Valid: false}
			} else {
				user.Notes = database.NullString(notes)
			}
		}
	}
	if req.SpecialNodeSubscriptionType != nil {
		mode := strings.TrimSpace(*req.SpecialNodeSubscriptionType)
		switch mode {
		case "", "normal":
			user.SpecialNodeSubscriptionType = "normal"
		case "both", "special_only":
			user.SpecialNodeSubscriptionType = mode
		default:
			utils.ErrorResponse(c, http.StatusBadRequest, "线路模式无效", nil)
			return
		}
	}

	if err := db.Save(&user).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新失败", err)
		return
	}

	// 如果设备数量或到期时间有变更，更新订阅信息
	if req.DeviceLimit != nil || req.ExpireTime != nil {
		var subscription models.Subscription
		if err := db.Where("user_id = ?", user.ID).Order("created_at DESC").First(&subscription).Error; err == nil {
			// 记录订阅变更前数据
			beforeSubData := map[string]interface{}{
				"device_limit": subscription.DeviceLimit,
				"expire_time":  utils.FormatBeijingTime(subscription.ExpireTime),
			}

			if req.DeviceLimit != nil {
				subscription.DeviceLimit = *req.DeviceLimit
			}
			if req.ExpireTime != nil && *req.ExpireTime != "" {
				if t, err := time.Parse("2006-01-02T15:04:05", *req.ExpireTime); err == nil {
					subscription.ExpireTime = t
				} else if t, err := time.Parse("2006-01-02 15:04:05", *req.ExpireTime); err == nil {
					subscription.ExpireTime = t
				} else if t, err := time.Parse(time.RFC3339, *req.ExpireTime); err == nil {
					subscription.ExpireTime = t
				}
			}

			if err := db.Save(&subscription).Error; err == nil {
				// 记录订阅变更日志
				afterSubData := map[string]interface{}{
					"device_limit": subscription.DeviceLimit,
					"expire_time":  utils.FormatBeijingTime(subscription.ExpireTime),
				}

				go func() {
					adminUser, _ := middleware.GetCurrentUser(c)
					var actionByUserID *uint
					actionBy := "admin"
					if adminUser != nil {
						actionByUserID = &adminUser.ID
						actionBy = adminUser.Username
					}
					ipAddress := utils.GetRealClientIP(c)
					if err := utils.CreateSubscriptionLog(subscription.ID, user.ID, "update", actionBy, actionByUserID, ipAddress, beforeSubData, afterSubData, "管理员通过编辑用户更新订阅信息"); err != nil {
						log.Printf("failed to create subscription log: %v", err)
					}
				}()

				// 清除订阅配置缓存
				go func(subURL string) {
					if err := cache.ClearSubscriptionConfigCache(subURL); err != nil {
						log.Printf("failed to clear subscription config cache: %v", err)
					}
				}(subscription.SubscriptionURL)
			}
		}
	}

	afterData := map[string]interface{}{
		"username":                       user.Username,
		"email":                          user.Email,
		"is_active":                      user.IsActive,
		"is_verified":                    user.IsVerified,
		"is_admin":                       user.IsAdmin,
		"balance":                        user.Balance,
		"special_node_subscription_type": user.SpecialNodeSubscriptionType,
	}

	description := fmt.Sprintf("管理员更新用户: %s (%s)", user.Username, user.Email)
	if req.Password != "" {
		description += " (包含密码重置)"
	}
	utils.CreateAuditLogWithData(c, "update_user", "user", user.ID, description, beforeData, afterData)
	if req.SpecialNodeSubscriptionType != nil {
		clearUserCustomNodeCache(user.ID)
	}

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
			if err := utils.CreateBalanceLog(
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
			); err != nil {
				log.Printf("failed to create balance log: %v", err)
			}
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

	// 设备按 user_id 直接删除（此前“按订阅子查询删除”在订阅已删后恒为空，属死代码）
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

	// 补充清理遗漏的关联数据，避免残留孤儿记录
	if err := tx.Where("user_id = ?", user.ID).Delete(&models.CheckinRecord{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete checkin records failed", err, map[string]interface{}{"user_id": user.ID})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户签到记录失败", err)
		return
	}
	if err := tx.Where("user_id = ?", user.ID).Delete(&models.UserCustomNode{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete user custom nodes failed", err, map[string]interface{}{"user_id": user.ID})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户专线节点分配失败", err)
		return
	}
	if err := tx.Where("username = ?", user.Username).Delete(&models.LoginAttempt{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete login attempts failed", err, map[string]interface{}{"username": user.Username})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户登录尝试记录失败", err)
		return
	}
	if err := tx.Where("email = ?", user.Email).Delete(&models.VerificationCode{}).Error; err != nil {
		tx.Rollback()
		utils.LogError("DeleteUser: delete verification codes failed", err, map[string]interface{}{"email": user.Email})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除用户验证码记录失败", err)
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
	if targetUser.IsAdmin {
		utils.ErrorResponse(c, http.StatusForbidden, "不能以管理员账号进入用户后台", nil)
		return
	}
	if !targetUser.IsActive {
		utils.ErrorResponse(c, http.StatusBadRequest, "该用户已被禁用，无法登录", nil)
		return
	}

	accessToken, err := utils.CreateAccessToken(targetUser.ID, targetUser.Email, false)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成令牌失败", err)
		return
	}

	refreshToken, err := utils.CreateRefreshToken(targetUser.ID, targetUser.Email, false)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成刷新令牌失败", err)
		return
	}

	// 设置用户的 refresh token，通过响应体返回（不再使用 cookie）

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
			"is_admin": false,
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

	// 自我保护：禁止管理员禁用自己或取消自己的管理员身份
	currentUser, _ := middleware.GetCurrentUser(c)
	if currentUser != nil && currentUser.ID == user.ID {
		if (req.IsActive != nil && !*req.IsActive) || req.Status == "inactive" || req.Status == "disabled" {
			utils.ErrorResponse(c, http.StatusBadRequest, "不能禁用当前登录的管理员账号", nil)
			return
		}
		if req.IsAdmin != nil && !*req.IsAdmin {
			utils.ErrorResponse(c, http.StatusBadRequest, "不能取消当前登录账号的管理员身份", nil)
			return
		}
	}

	// 保存变更前的状态
	oldIsActive := user.IsActive
	oldIsVerified := user.IsVerified
	oldIsAdmin := user.IsAdmin

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

	// 构建变更描述
	changes := []string{}
	if oldIsActive != user.IsActive {
		if user.IsActive {
			changes = append(changes, "启用账户")
		} else {
			changes = append(changes, "禁用账户")
		}
	}
	if oldIsVerified != user.IsVerified {
		if user.IsVerified {
			changes = append(changes, "验证邮箱")
		} else {
			changes = append(changes, "取消验证")
		}
	}
	if oldIsAdmin != user.IsAdmin {
		if user.IsAdmin {
			changes = append(changes, "设为管理员")
		} else {
			changes = append(changes, "取消管理员")
		}
	}
	changeDesc := strings.Join(changes, "，")
	if changeDesc == "" {
		changeDesc = "无变更"
	}

	utils.CreateAuditLog(c, "update_user_status", "user", user.ID,
		fmt.Sprintf("管理员更新用户 %s(ID:%d, 邮箱:%s) 状态: %s", user.Username, user.ID, user.Email, changeDesc),
		map[string]interface{}{
			"target_user_id":  user.ID,
			"target_username": user.Username,
			"target_email":    user.Email,
			"is_active":       oldIsActive,
			"is_verified":     oldIsVerified,
			"is_admin":        oldIsAdmin,
		},
		map[string]interface{}{
			"is_active":   user.IsActive,
			"is_verified": user.IsVerified,
			"is_admin":    user.IsAdmin,
		})
	middleware.InvalidateAuthUserCache(user.ID)
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
	utils.CreateAuditLog(c, "batch_delete_users", "user", 0,
		fmt.Sprintf("管理员批量删除 %d 个用户", len(req.UserIDs)),
		map[string]interface{}{"user_ids": req.UserIDs, "count": len(req.UserIDs)}, nil)
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功删除 %d 个用户", len(req.UserIDs)), nil)
}

func BatchEnableUsers(c *gin.Context) {
	batchUpdateUsersActive(c, true)
}

func BatchDisableUsers(c *gin.Context) {
	batchUpdateUsersActive(c, false)
}

// batchUpdateUsersActive 批量启用/禁用用户的公共实现（Enable/Disable 此前约 120 行复制粘贴）
func batchUpdateUsersActive(c *gin.Context, active bool) {
	var req struct {
		UserIDs []uint `json:"user_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	actionName := "启用"
	actionKey := "batch_enable_users"
	if !active {
		actionName = "禁用"
		actionKey = "batch_disable_users"
	}

	if len(req.UserIDs) == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("请选择要%s的用户", actionName), nil)
		return
	}

	db := database.GetDB()

	if !active {
		// 自我保护 + 管理员保护（仅禁用时）
		currentUser, _ := middleware.GetCurrentUser(c)
		if currentUser != nil {
			for _, id := range req.UserIDs {
				if id == currentUser.ID {
					utils.ErrorResponse(c, http.StatusBadRequest, "不能禁用当前登录的管理员账户", nil)
					return
				}
			}
		}
		var adminUsers []models.User
		if err := db.Where("id IN ? AND is_admin = ?", req.UserIDs, true).Find(&adminUsers).Error; err == nil && len(adminUsers) > 0 {
			utils.ErrorResponse(c, http.StatusBadRequest, "不能禁用管理员用户", nil)
			return
		}
	}

	var targetUsers []models.User
	db.Where("id IN ?", req.UserIDs).Select("id, username, email").Find(&targetUsers)
	result := db.Model(&models.User{}).Where("id IN ?", req.UserIDs).Update("is_active", active)

	if result.Error != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("%s用户失败", actionName), result.Error)
		return
	}

	// 清除被操作用户的订阅配置缓存
	go func(userIDs []uint) {
		var subs []models.Subscription
		db.Select("subscription_url").Where("user_id IN ?", userIDs).Find(&subs)
		for _, sub := range subs {
			if err := cache.ClearSubscriptionConfigCache(sub.SubscriptionURL); err != nil {
				log.Printf("failed to clear subscription config cache: %v", err)
			}
		}
	}(req.UserIDs)

	userDetails := make([]map[string]interface{}, 0, len(targetUsers))
	for _, u := range targetUsers {
		userDetails = append(userDetails, map[string]interface{}{
			"user_id":  u.ID,
			"username": u.Username,
			"email":    u.Email,
		})
	}
	utils.CreateAuditLog(c, actionKey, "user", 0,
		fmt.Sprintf("管理员批量%s %d 个用户", actionName, result.RowsAffected),
		map[string]interface{}{"user_ids": req.UserIDs, "users": userDetails},
		map[string]interface{}{"is_active": active, "count": result.RowsAffected})
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功%s %d 个用户", actionName, result.RowsAffected), nil)
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

	// 批量查询所有用户的订阅
	var subs []models.Subscription
	db.Where("user_id IN ?", req.UserIDs).Find(&subs)
	subMap := make(map[uint]models.Subscription)
	for _, s := range subs {
		subMap[s.UserID] = s
	}

	successCount := 0
	failCount := 0
	emailTargets := make([]map[string]interface{}, 0, len(users))

	for _, user := range users {
		sub, ok := subMap[user.ID]
		if !ok {
			failCount++
			continue
		}

		if err := queueSubEmail(c, sub, user); err != nil {
			failCount++
			continue
		}
		emailTargets = append(emailTargets, map[string]interface{}{
			"user_id":  user.ID,
			"username": user.Username,
			"email":    user.Email,
		})
		successCount++
	}
	utils.CreateAuditLog(c, "batch_send_sub_email", "user", 0,
		fmt.Sprintf("管理员批量发送订阅邮件 %d 封，成功 %d 失败 %d", len(users), successCount, failCount),
		map[string]interface{}{
			"user_ids": req.UserIDs,
			"total":    len(users),
			"targets":  emailTargets,
		}, nil)
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

	// 批量查询所有相关的 Package
	pkgIDs := make([]int64, 0)
	for _, user := range users {
		if len(user.Subscriptions) > 0 && user.Subscriptions[0].PackageID != nil {
			pkgIDs = append(pkgIDs, *user.Subscriptions[0].PackageID)
		}
	}
	pkgMap := make(map[int64]string)
	if len(pkgIDs) > 0 {
		var pkgs []models.Package
		db.Where("id IN ?", pkgIDs).Select("id, name").Find(&pkgs)
		for _, p := range pkgs {
			pkgMap[int64(p.ID)] = p.Name
		}
	}

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

		daysUntilExpire, isExpired := utils.RemainingDays(sub.ExpireTime, now)

		subject := "订阅即将到期提醒"
		pkgName := "默认套餐"
		if sub.PackageID != nil {
			if name, ok := pkgMap[*sub.PackageID]; ok {
				pkgName = name
			}
		}
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
			"location":     utils.FormatLocation(h.Location.String),
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

			daysLeft, _ := utils.RemainingDays(subscription.ExpireTime, utils.GetBeijingTime())
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

package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 注意：getCurrentUserOrError 和 getDefaultSubscriptionSettings 已在 user_profile.go 和 user.go 中定义

func getSubscriptionURLs(c *gin.Context, subURL string) (string, string) {
	baseURL := utils.GetBuildBaseURL(c.Request, database.GetDB())
	timestamp := fmt.Sprintf("%d", utils.GetBeijingTime().Unix())
	universal := fmt.Sprintf("%s/api/v1/subscriptions/universal/%s?t=%s", baseURL, subURL, timestamp)
	clash := fmt.Sprintf("%s/api/v1/subscriptions/clash/%s?t=%s", baseURL, subURL, timestamp)
	return universal, clash
}

func getCurrentAdminUsername(c *gin.Context) *string {
	user, ok := middleware.GetCurrentUser(c)
	if ok && user != nil {
		return &user.Username
	}
	return nil
}

func formatDeviceList(devices []models.Device) []gin.H {
	deviceList := make([]gin.H, 0)
	getString := func(ptr *string) string {
		if ptr != nil {
			return *ptr
		}
		return ""
	}
	formatIP := func(ip string) string {
		if ip == "" {
			return "-"
		}
		// 将 IPv6 本地地址转换为 IPv4 本地地址
		if ip == "::1" {
			return "127.0.0.1"
		}
		// 将 IPv6 映射的 IPv4 地址转换
		if strings.HasPrefix(ip, "::ffff:") {
			return strings.TrimPrefix(ip, "::ffff:")
		}
		return ip
	}
	for _, d := range devices {
		lastSeen := d.LastAccess.Format("2006-01-02 15:04:05")
		if d.LastSeen != nil {
			lastSeen = d.LastSeen.Format("2006-01-02 15:04:05")
		}
		ipAddress := formatIP(getString(d.IPAddress))
		deviceList = append(deviceList, gin.H{
			"id":                 d.ID,
			"device_name":        getString(d.DeviceName),
			"name":               getString(d.DeviceName),
			"device_fingerprint": d.DeviceFingerprint,
			"device_type":        getString(d.DeviceType),
			"type":               getString(d.DeviceType),
			"ip_address":         ipAddress,
			"ip":                 ipAddress,
			"os_name":            getString(d.OSName),
			"os_version":         getString(d.OSVersion),
			"last_access":        d.LastAccess.Format("2006-01-02 15:04:05"),
			"last_seen":          lastSeen,
			"created_at":         d.CreatedAt.Format("2006-01-02 15:04:05"),
			"is_active":          d.IsActive,
			"is_allowed":         d.IsAllowed,
			"user_agent":         getString(d.UserAgent),
			"software_name":      getString(d.SoftwareName),
			"software_version":   getString(d.SoftwareVersion),
			"device_model":       getString(d.DeviceModel),
			"device_brand":       getString(d.DeviceBrand),
			"access_count":       d.AccessCount,
		})
	}
	return deviceList
}

// GetSubscriptions 获取当前用户的订阅列表
func GetSubscriptions(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}
	var subscriptions []models.Subscription
	if err := database.GetDB().Where("user_id = ?", user.ID).Find(&subscriptions).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅列表失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", subscriptions)
}

// GetSubscription 获取指定ID的订阅详情
func GetSubscription(c *gin.Context) {
	id := c.Param("id")
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}
	var sub models.Subscription
	if err := database.GetDB().Where("id = ? AND user_id = ?", id, user.ID).First(&sub).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅失败", err)
		}
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", sub)
}

// CreateSubscription 创建新订阅
func CreateSubscription(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}
	db := database.GetDB()
	// 从系统设置中获取默认订阅配置
	deviceLimit, durationMonths := getDefaultSubscriptionSettings(db)
	sub := models.Subscription{
		UserID:          user.ID,
		SubscriptionURL: utils.GenerateSubscriptionURL(),
		DeviceLimit:     deviceLimit,
		CurrentDevices:  0,
		IsActive:        true,
		Status:          utils.SubscriptionStatusActive,
		ExpireTime:      utils.GetBeijingTime().AddDate(0, durationMonths, 0),
	}
	if err := db.Create(&sub).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建订阅失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusCreated, "", sub)
}

func GetAdminSubscriptions(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.Subscription{})
	page, size := 1, 20
	fmt.Sscanf(c.Query("page"), "%d", &page)
	fmt.Sscanf(c.Query("size"), "%d", &size)

	if keyword := utils.SanitizeSearchKeyword(c.DefaultQuery("search", c.Query("keyword"))); keyword != "" {
		query = query.Where(
			"subscription_url LIKE ? OR user_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ?) OR user_id IN (SELECT DISTINCT user_id FROM subscription_resets WHERE old_subscription_url LIKE ?)",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	if status := c.Query("status"); status != "" {
		switch status {
		case "active":
			query = query.Where("status = ? AND is_active = ?", "active", true)
		case "expired":
			query = query.Where("expire_time < ?", utils.GetBeijingTime())
		case "inactive":
			query = query.Where("is_active = ?", false)
		}
	}

	sort := c.DefaultQuery("sort", "add_time_desc")
	sortMap := map[string]string{
		"add_time_desc": "created_at DESC", "add_time_asc": "created_at ASC",
		"expire_time_desc": "expire_time DESC", "expire_time_asc": "expire_time ASC",
		"device_count_desc": "current_devices DESC", "device_count_asc": "current_devices ASC",
	}
	if order, ok := sortMap[sort]; ok {
		query = query.Order(order)
	} else {
		query = query.Order("created_at DESC")
	}

	var total int64
	query.Count(&total)
	// 使用 Preload 预加载 User 和 Package，避免 N+1 查询
	query = query.Preload("User").Preload("Package")

	var subscriptions []models.Subscription
	if err := query.Offset((page - 1) * size).Limit(size).Find(&subscriptions).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅列表失败", err)
		return
	}

	list := make([]gin.H, 0)
	if len(subscriptions) > 0 {
		subIDs := make([]uint, len(subscriptions))
		userIDs := make([]uint, 0, len(subscriptions))
		userIDSet := make(map[uint]bool)
		for i, s := range subscriptions {
			subIDs[i] = s.ID
			if !userIDSet[s.UserID] {
				userIDs = append(userIDs, s.UserID)
				userIDSet[s.UserID] = true
			}
		}

		// 批量查询所有用户，避免 N+1 查询
		var users []models.User
		userMap := make(map[uint]*models.User)
		if len(userIDs) > 0 {
			db.Where("id IN ?", userIDs).Find(&users)
			for i := range users {
				userMap[users[i].ID] = &users[i]
			}
		}

		type Stat struct {
			SubID uint
			Type  *string
			Count int64
		}
		var onlineStats []Stat
		db.Model(&models.Device{}).Select("subscription_id as sub_id, count(*) as count").Where("subscription_id IN ? AND is_active = ?", subIDs, true).Group("subscription_id").Scan(&onlineStats)

		var typeStats []Stat
		db.Model(&models.Device{}).Select("subscription_id as sub_id, subscription_type as type, count(*) as count").Where("subscription_id IN ?", subIDs).Group("subscription_id, subscription_type").Scan(&typeStats)

		onlineMap, appleMap, clashMap := make(map[uint]int64), make(map[uint]int64), make(map[uint]int64)
		for _, s := range onlineStats {
			onlineMap[s.SubID] = s.Count
		}
		for _, s := range typeStats {
			if s.Type == nil {
				continue
			}
			if *s.Type == "v2ray" || *s.Type == "ssr" {
				appleMap[s.SubID] += s.Count
			}
			if *s.Type == "clash" {
				clashMap[s.SubID] += s.Count
			}
		}

		for _, sub := range subscriptions {
			online := onlineMap[sub.ID]
			curr := sub.CurrentDevices
			if curr < int(online) {
				curr = int(online)
			}

			universal, clash := getSubscriptionURLs(c, sub.SubscriptionURL)
			// 使用预加载的 User 或从 userMap 获取，避免 N+1 查询
			var userInfo gin.H
			if sub.User.ID > 0 {
				userInfo = gin.H{"id": sub.User.ID, "username": sub.User.Username, "email": sub.User.Email}
			} else if user, ok := userMap[sub.UserID]; ok {
				userInfo = gin.H{"id": user.ID, "username": user.Username, "email": user.Email}
			} else {
				userInfo = gin.H{"id": 0, "username": fmt.Sprintf("用户已删除 (ID: %d)", sub.UserID), "email": fmt.Sprintf("deleted_user_%d", sub.UserID), "deleted": true}
			}

			daysUntil, isExpired, now := 0, false, utils.GetBeijingTime()
			if !sub.ExpireTime.IsZero() {
				if diff := sub.ExpireTime.Sub(now); diff > 0 {
					daysUntil = int(diff.Hours() / 24)
				} else {
					isExpired = true
				}
			}

			// 使用数据库中的订阅次数字段，如果没有则使用统计值作为后备
			universalCount := sub.UniversalCount
			clashCount := sub.ClashCount
			if universalCount == 0 && appleMap[sub.ID] > 0 {
				universalCount = int(appleMap[sub.ID])
			}
			if clashCount == 0 && clashMap[sub.ID] > 0 {
				clashCount = int(clashMap[sub.ID])
			}

			list = append(list, gin.H{
				"id": sub.ID, "user_id": sub.UserID, "user": userInfo, "username": userInfo["username"], "email": userInfo["email"],
				"subscription_url": sub.SubscriptionURL, "universal_url": universal, "clash_url": clash,
				"status": sub.Status, "is_active": sub.IsActive, "device_limit": sub.DeviceLimit,
				"current_devices": curr, "online_devices": online, "apple_count": universalCount, "clash_count": clashCount,
				"expire_time": sub.ExpireTime.Format("2006-01-02 15:04:05"), "days_until_expire": daysUntil, "is_expired": isExpired,
				"created_at": sub.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"subscriptions": list, "total": total, "page": page, "size": size})
}

// GetUserSubscriptionDevices 获取当前用户的订阅设备列表
func GetUserSubscriptionDevices(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}
	db := database.GetDB()
	var sub models.Subscription
	if err := db.Where("user_id = ?", user.ID).Order("created_at DESC").First(&sub).Error; err != nil {
		utils.SuccessResponse(c, http.StatusOK, "", []gin.H{})
		return
	}
	var devices []models.Device
	db.Where("subscription_id = ?", sub.ID).Find(&devices)
	utils.SuccessResponse(c, http.StatusOK, "", formatDeviceList(devices))
}

// GetSubscriptionDevices 获取指定订阅的设备列表（管理员）
func GetSubscriptionDevices(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var sub models.Subscription
	if err := db.First(&sub, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅失败", err)
		}
		return
	}
	var devices []models.Device
	db.Where("subscription_id = ?", sub.ID).Find(&devices)
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"devices":        formatDeviceList(devices),
		"device_limit":   sub.DeviceLimit,
		"current_devices": sub.CurrentDevices,
	})
}

// BatchClearDevices 批量清除订阅设备
func BatchClearDevices(c *gin.Context) {
	var req struct {
		SubscriptionIDs []uint `json:"subscription_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}
	db := database.GetDB()
	db.Where("subscription_id IN ?", req.SubscriptionIDs).Delete(&models.Device{})
	db.Model(&models.Subscription{}).Where("id IN ?", req.SubscriptionIDs).Update("current_devices", 0)
	utils.SuccessResponse(c, http.StatusOK, "设备已清除", nil)
}

// UpdateSubscription 更新订阅信息
func UpdateSubscription(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		DeviceLimit *int    `json:"device_limit"`
		ExpireTime  *string `json:"expire_time"`
		IsActive    *bool   `json:"is_active"`
		Status      string  `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}
	db := database.GetDB()
	var sub models.Subscription
	if err := db.First(&sub, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅失败", err)
		}
		return
	}
	if req.DeviceLimit != nil {
		sub.DeviceLimit = *req.DeviceLimit
	}
	if req.IsActive != nil {
		sub.IsActive = *req.IsActive
	}
	if req.Status != "" {
		sub.Status = req.Status
	}
	if req.ExpireTime != nil && *req.ExpireTime != "" {
		if t, err := time.Parse("2006-01-02", *req.ExpireTime); err == nil {
			sub.ExpireTime = t
		} else if t, err := time.Parse("2006-01-02 15:04:05", *req.ExpireTime); err == nil {
			sub.ExpireTime = t
		}
	}
	if err := db.Save(&sub).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "更新成功", nil)
}

// ResetSubscription 重置订阅地址
func ResetSubscription(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var sub models.Subscription
	if err := db.First(&sub, id).Preload("User").Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅失败", err)
		}
		return
	}

	// 记录旧订阅地址
	oldURL := sub.SubscriptionURL
	var deviceCountBefore int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", sub.ID, true).Count(&deviceCountBefore)

	// 生成新订阅地址
	newURL := utils.GenerateSubscriptionURL()
	sub.SubscriptionURL = newURL
	sub.CurrentDevices = 0
	db.Save(&sub)

	// 记录订阅重置
	reset := models.SubscriptionReset{
		UserID:             sub.UserID,
		SubscriptionID:     sub.ID,
		ResetType:          "admin_reset",
		Reason:             "管理员重置订阅地址",
		OldSubscriptionURL: &oldURL,
		NewSubscriptionURL: &newURL,
		DeviceCountBefore:  int(deviceCountBefore),
		DeviceCountAfter:   0,
		ResetBy:            getCurrentAdminUsername(c),
	}
	db.Create(&reset)

	// 清理设备记录
	db.Where("subscription_id = ?", sub.ID).Delete(&models.Device{})

	go sendResetEmail(c, sub, sub.User, "管理员重置")
	utils.SuccessResponse(c, http.StatusOK, "订阅已重置", sub)
}

// ExtendSubscription 延长订阅有效期
func ExtendSubscription(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Days int `json:"days" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "days 必填", err)
		return
	}
	db := database.GetDB()
	var sub models.Subscription
	if err := db.First(&sub, id).Preload("User").Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅失败", err)
		}
		return
	}
	oldExp := "未设置"
	if !sub.ExpireTime.IsZero() {
		oldExp = sub.ExpireTime.Format("2006-01-02 15:04:05")
	}
	if sub.ExpireTime.IsZero() {
		sub.ExpireTime = utils.GetBeijingTime()
	}
	sub.ExpireTime = sub.ExpireTime.AddDate(0, 0, req.Days)
	db.Save(&sub)

	go func() {
		pkgName := "默认套餐"
		if sub.PackageID != nil {
			var pkg models.Package
			if err := db.First(&pkg, *sub.PackageID).Error; err == nil {
				pkgName = pkg.Name
			}
		}
		email.NewEmailService().QueueEmail(sub.User.Email, "续费成功",
			email.NewEmailTemplateBuilder().GetRenewalConfirmationTemplate(sub.User.Username, pkgName, oldExp, sub.ExpireTime.Format("2006-01-02 15:04:05"), utils.GetBeijingTime().Format("2006-01-02 15:04:05"), 0), "renewal_confirmation")
	}()
	utils.SuccessResponse(c, http.StatusOK, "订阅已延长", sub)
}

func ResetUserSubscription(c *gin.Context) {
	userID := c.Param("id")
	db := database.GetDB()
	var subs []models.Subscription
	db.Where("user_id = ?", userID).Find(&subs)

	for _, sub := range subs {
		// 记录旧订阅地址
		oldURL := sub.SubscriptionURL
		var deviceCountBefore int64
		db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", sub.ID, true).Count(&deviceCountBefore)

		// 生成新订阅地址
		newURL := utils.GenerateSubscriptionURL()
		sub.SubscriptionURL = newURL
		sub.CurrentDevices = 0
		db.Save(&sub)

		// 记录订阅重置
		reset := models.SubscriptionReset{
			UserID:             sub.UserID,
			SubscriptionID:     sub.ID,
			ResetType:          "admin_reset",
			Reason:             "管理员重置用户订阅地址",
			OldSubscriptionURL: &oldURL,
			NewSubscriptionURL: &newURL,
			DeviceCountBefore:  int(deviceCountBefore),
			DeviceCountAfter:   0,
			ResetBy:            getCurrentAdminUsername(c),
		}
		db.Create(&reset)

		// 清理设备记录
		db.Where("subscription_id = ?", sub.ID).Delete(&models.Device{})
	}

	utils.SuccessResponse(c, http.StatusOK, "用户订阅已重置", nil)
}

// SendSubscriptionEmail 发送订阅邮件给用户
func SendSubscriptionEmail(c *gin.Context) {
	userID := c.Param("id")
	db := database.GetDB()
	var user models.User
	var sub models.Subscription
	if err := db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "用户不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取用户失败", err)
		}
		return
	}
	if err := db.Where("user_id = ?", user.ID).First(&sub).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "用户没有订阅", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅失败", err)
		}
		return
	}
	if err := queueSubEmail(c, sub, user); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "发送邮件失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "订阅邮件已加入队列", nil)
}

// ClearUserDevices 清除用户的所有设备
func ClearUserDevices(c *gin.Context) {
	userID := c.Param("id")
	db := database.GetDB()
	var subIDs []uint
	db.Model(&models.Subscription{}).Where("user_id = ?", userID).Pluck("id", &subIDs)
	if len(subIDs) > 0 {
		db.Where("subscription_id IN ?", subIDs).Delete(&models.Device{})
		db.Model(&models.Subscription{}).Where("id IN ?", subIDs).Update("current_devices", 0)
	}
	utils.SuccessResponse(c, http.StatusOK, "设备已清理", nil)
}

// ResetUserSubscriptionSelf 用户重置自己的订阅
func ResetUserSubscriptionSelf(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}
	db := database.GetDB()
	var sub models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&sub).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅失败", err)
		}
		return
	}

	// 记录旧订阅地址
	oldURL := sub.SubscriptionURL
	var deviceCountBefore int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", sub.ID, true).Count(&deviceCountBefore)

	// 生成新订阅地址
	newURL := utils.GenerateSubscriptionURL()
	sub.SubscriptionURL = newURL
	sub.CurrentDevices = 0
	db.Save(&sub)

	// 记录订阅重置
	reason := "用户主动重置订阅地址"
	reset := models.SubscriptionReset{
		UserID:             sub.UserID,
		SubscriptionID:     sub.ID,
		ResetType:          "user_reset",
		Reason:             reason,
		OldSubscriptionURL: &oldURL,
		NewSubscriptionURL: &newURL,
		DeviceCountBefore:  int(deviceCountBefore),
		DeviceCountAfter:   0,
		ResetBy:            &user.Username,
	}
	db.Create(&reset)

	// 清理设备记录
	db.Where("subscription_id = ?", sub.ID).Delete(&models.Device{})

	go sendResetEmail(c, sub, *user, reason)
	utils.SuccessResponse(c, http.StatusOK, "订阅已重置", sub)
}

// SendSubscriptionEmailSelf 用户发送自己的订阅邮件
func SendSubscriptionEmailSelf(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}
	var sub models.Subscription
	if err := database.GetDB().Where("user_id = ?", user.ID).First(&sub).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "您还没有订阅", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅失败", err)
		}
		return
	}
	go notification.NewNotificationService().SendAdminNotification("subscription_sent", map[string]interface{}{"username": user.Username, "email": user.Email, "send_time": utils.GetBeijingTime().Format("2006-01-02 15:04:05")})
	if err := queueSubEmail(c, sub, *user); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "发送邮件失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "订阅邮件已加入队列", nil)
}

// ConvertSubscriptionToBalance 将订阅转换为余额
func ConvertSubscriptionToBalance(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}
	db := database.GetDB()
	var sub models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&sub).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅失败", err)
		}
		return
	}
	now := utils.GetBeijingTime()
	if sub.ExpireTime.After(now) {
		// 计算剩余天数（向上取整）
		diff := sub.ExpireTime.Sub(now)
		days := int(diff.Hours() / 24)
		if diff.Hours() > float64(days*24) {
			days++ // 如果有小数部分，向上取整
		}

		// 计算折算金额
		// 公式：折算金额 = 剩余天数 × (原套餐价格 ÷ 原套餐天数)
		var originalPackagePrice float64 = 0
		var originalPackageDays int = 0

		// 方法1：从订阅的 PackageID 获取原套餐信息
		if sub.PackageID != nil {
			var pkg models.Package
			if err := db.First(&pkg, *sub.PackageID).Error; err == nil {
				originalPackagePrice = pkg.Price
				originalPackageDays = pkg.DurationDays
			}
		}

		// 方法2：如果订阅没有 PackageID（管理员直接开通），尝试从订单记录中查找
		if originalPackageDays <= 0 {
			// 计算订阅的总时长（从创建时间到到期时间）
			totalDuration := int(sub.ExpireTime.Sub(sub.CreatedAt).Hours() / 24)
			if totalDuration <= 0 {
				totalDuration = 30 // 默认30天
			}

			// 查找用户最近一次购买该订阅相关的订单（已支付）
			var recentOrder models.Order
			if err := db.Where("user_id = ? AND status = ?", user.ID, "paid").
				Order("created_at DESC").
				First(&recentOrder).Error; err == nil {
				// 找到了订单，使用订单的套餐信息
				var pkg models.Package
				if err := db.First(&pkg, recentOrder.PackageID).Error; err == nil {
					originalPackagePrice = recentOrder.Amount // 使用订单原价（未折扣）
					originalPackageDays = pkg.DurationDays
				}
			}

			// 方法3：如果还是找不到，根据订阅总时长查找系统中相同时长的套餐
			if originalPackageDays <= 0 {
				var similarPackage models.Package
				// 查找时长最接近的套餐（允许±5天的误差）
				// 使用原生SQL计算差值并排序
				if err := db.Where("duration_days BETWEEN ? AND ? AND is_active = ?",
					totalDuration-5, totalDuration+5, true).
					Order(fmt.Sprintf("ABS(duration_days - %d) ASC", totalDuration)).
					First(&similarPackage).Error; err == nil {
					originalPackagePrice = similarPackage.Price
					originalPackageDays = similarPackage.DurationDays
				}
			}

			// 方法4：如果仍然找不到，根据订阅总时长估算价格（使用默认每天1元）
			if originalPackageDays <= 0 {
				originalPackageDays = totalDuration
				if originalPackageDays <= 0 {
					originalPackageDays = 30 // 默认30天
				}
				originalPackagePrice = float64(originalPackageDays) * 1.0 // 默认每天1元
			}
		}

		// 计算每天单价
		dailyPrice := originalPackagePrice / float64(originalPackageDays)

		// 计算折算金额
		convertedAmount := float64(days) * dailyPrice

		// 保留两位小数
		convertedAmount = float64(int(convertedAmount*100+0.5)) / 100

		user.Balance += convertedAmount
		if err := db.Save(&user).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "更新余额失败", err)
			return
		}

		// 删除订阅
		if err := db.Delete(&sub).Error; err != nil {
			utils.LogError("ConvertSubscriptionToBalance: failed to delete subscription", err, map[string]interface{}{
				"user_id":         user.ID,
				"subscription_id": sub.ID,
			})
		}

		utils.SuccessResponse(c, http.StatusOK, "已转换为余额", gin.H{
			"converted_amount":       convertedAmount,
			"balance_added":          convertedAmount,
			"new_balance":            user.Balance,
			"remaining_days":         days,
			"daily_price":            dailyPrice,
			"original_package_price": originalPackagePrice,
			"original_package_days":  originalPackageDays,
		})
	} else {
		utils.ErrorResponse(c, http.StatusBadRequest, "订阅已过期", nil)
	}
}

// ExportSubscriptions 导出订阅列表为 CSV
func ExportSubscriptions(c *gin.Context) {
	var subs []models.Subscription
	if err := database.GetDB().Preload("User").Find(&subs).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取列表失败", err)
		return
	}
	var csv strings.Builder
	csv.WriteString("\xEF\xBB\xBFID,用户ID,用户名,邮箱,订阅地址,状态,是否激活,设备限制,当前设备,到期时间,创建时间\n")
	for _, s := range subs {
		active := "是"
		if !s.IsActive {
			active = "否"
		}
		csv.WriteString(fmt.Sprintf("%d,%d,%s,%s,%s,%s,%s,%d,%d,%s,%s\n", s.ID, s.UserID, s.User.Username, s.User.Email, s.SubscriptionURL, s.Status, active, s.DeviceLimit, s.CurrentDevices, s.ExpireTime.Format("2006-01-02 15:04:05"), s.CreatedAt.Format("2006-01-02 15:04:05")))
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=subscriptions_%s.csv", time.Now().Format("20060102")))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", []byte(csv.String()))
}

func sendResetEmail(c *gin.Context, sub models.Subscription, user models.User, reason string) {
	univ, clash := getSubscriptionURLs(c, sub.SubscriptionURL)
	exp := "未设置"
	if !sub.ExpireTime.IsZero() {
		exp = sub.ExpireTime.Format("2006-01-02 15:04:05")
	}
	resetTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
	content := email.NewEmailTemplateBuilder().GetSubscriptionResetTemplate(user.Username, univ, clash, exp, resetTime, reason)
	_ = email.NewEmailService().QueueEmail(user.Email, "订阅重置通知", content, "subscription_reset")
	_ = notification.NewNotificationService().SendAdminNotification("subscription_reset", map[string]interface{}{"username": user.Username, "email": user.Email, "reset_time": resetTime})
}

func queueSubEmail(c *gin.Context, sub models.Subscription, user models.User) error {
	univ, clash := getSubscriptionURLs(c, sub.SubscriptionURL)
	exp, days := "未设置", 0
	if !sub.ExpireTime.IsZero() {
		exp = sub.ExpireTime.Format("2006-01-02 15:04:05")
		if diff := sub.ExpireTime.Sub(utils.GetBeijingTime()); diff > 0 {
			days = int(diff.Hours() / 24)
		}
	}
	content := email.NewEmailTemplateBuilder().GetSubscriptionTemplate(user.Username, univ, clash, exp, days, sub.DeviceLimit, sub.CurrentDevices)
	return email.NewEmailService().QueueEmail(user.Email, "服务配置信息", content, "subscription")
}

// BatchDeleteSubscriptions 批量删除订阅
func BatchDeleteSubscriptions(c *gin.Context) {
	var req struct {
		SubscriptionIDs []uint `json:"subscription_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	if len(req.SubscriptionIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请选择要删除的订阅",
		})
		return
	}

	db := database.GetDB()
	tx := db.Begin()

	// 删除订阅相关的设备
	if err := tx.Where("subscription_id IN ?", req.SubscriptionIDs).Delete(&models.Device{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除订阅设备失败",
		})
		return
	}

	// 删除订阅重置记录
	if err := tx.Where("subscription_id IN ?", req.SubscriptionIDs).Delete(&models.SubscriptionReset{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除订阅重置记录失败",
		})
		return
	}

	// 删除订阅
	if err := tx.Where("id IN ?", req.SubscriptionIDs).Delete(&models.Subscription{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除订阅失败",
		})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除操作失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("成功删除 %d 个订阅", len(req.SubscriptionIDs)),
	})
}

// BatchEnableSubscriptions 批量启用订阅
func BatchEnableSubscriptions(c *gin.Context) {
	var req struct {
		SubscriptionIDs []uint `json:"subscription_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	if len(req.SubscriptionIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请选择要启用的订阅",
		})
		return
	}

	db := database.GetDB()
	result := db.Model(&models.Subscription{}).Where("id IN ?", req.SubscriptionIDs).Updates(map[string]interface{}{
		"is_active": true,
		"status":    "active",
	})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "启用订阅失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("成功启用 %d 个订阅", result.RowsAffected),
	})
}

// BatchDisableSubscriptions 批量禁用订阅
func BatchDisableSubscriptions(c *gin.Context) {
	var req struct {
		SubscriptionIDs []uint `json:"subscription_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	if len(req.SubscriptionIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请选择要禁用的订阅",
		})
		return
	}

	db := database.GetDB()
	result := db.Model(&models.Subscription{}).Where("id IN ?", req.SubscriptionIDs).Updates(map[string]interface{}{
		"is_active": false,
		"status":    "inactive",
	})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "禁用订阅失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("成功禁用 %d 个订阅", result.RowsAffected),
	})
}

// BatchResetSubscriptions 批量重置订阅
func BatchResetSubscriptions(c *gin.Context) {
	var req struct {
		SubscriptionIDs []uint `json:"subscription_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	if len(req.SubscriptionIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请选择要重置的订阅",
		})
		return
	}

	db := database.GetDB()
	var subscriptions []models.Subscription
	if err := db.Where("id IN ?", req.SubscriptionIDs).Preload("User").Find(&subscriptions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取订阅信息失败",
		})
		return
	}

	successCount := 0
	failCount := 0
	adminUsername := getCurrentAdminUsername(c)

	for _, sub := range subscriptions {
		// 记录旧订阅地址
		oldURL := sub.SubscriptionURL
		var deviceCountBefore int64
		db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", sub.ID, true).Count(&deviceCountBefore)

		// 生成新订阅地址
		newURL := utils.GenerateSubscriptionURL()
		sub.SubscriptionURL = newURL
		sub.CurrentDevices = 0

		if err := db.Save(&sub).Error; err != nil {
			failCount++
			continue
		}

		// 记录订阅重置
		reset := models.SubscriptionReset{
			UserID:             sub.UserID,
			SubscriptionID:     sub.ID,
			ResetType:          "admin_batch_reset",
			Reason:             "管理员批量重置订阅地址",
			OldSubscriptionURL: &oldURL,
			NewSubscriptionURL: &newURL,
			DeviceCountBefore:  int(deviceCountBefore),
			DeviceCountAfter:   0,
			ResetBy:            adminUsername,
		}
		if err := db.Create(&reset).Error; err != nil {
			failCount++
			continue
		}

		// 清理设备记录
		db.Where("subscription_id = ?", sub.ID).Delete(&models.Device{})

		// 发送重置邮件
		go sendResetEmail(c, sub, sub.User, "管理员批量重置")

		successCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("成功重置 %d 个订阅，失败 %d 个", successCount, failCount),
		"data": gin.H{
			"success_count": successCount,
			"fail_count":    failCount,
		},
	})
}

// BatchSendAdminSubEmail 批量发送订阅邮件（管理员）
func BatchSendAdminSubEmail(c *gin.Context) {
	var req struct {
		SubscriptionIDs []uint `json:"subscription_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	if len(req.SubscriptionIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请选择要发送邮件的订阅",
		})
		return
	}

	db := database.GetDB()
	var subscriptions []models.Subscription
	if err := db.Where("id IN ?", req.SubscriptionIDs).Preload("User").Find(&subscriptions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取订阅信息失败",
		})
		return
	}

	successCount := 0
	failCount := 0

	for _, sub := range subscriptions {
		if err := queueSubEmail(c, sub, sub.User); err != nil {
			failCount++
			continue
		}
		successCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("成功发送 %d 封邮件，失败 %d 封", successCount, failCount),
		"data": gin.H{
			"success_count": successCount,
			"fail_count":    failCount,
		},
	})
}

// GetExpiringSubscriptions 获取即将到期的订阅
func GetExpiringSubscriptions(c *gin.Context) {
	db := database.GetDB()

	// 获取参数
	daysStr := c.DefaultQuery("days", "7")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 {
		days = 7
	}

	filter := c.Query("filter")

	now := utils.GetBeijingTime()
	endDate := now.AddDate(0, 0, days)

	// 查询即将到期的订阅
	var subscriptions []models.Subscription
	query := db.Where("expire_time IS NOT NULL AND expire_time > ? AND expire_time <= ?", now, endDate).
		Where("is_active = ?", true).
		Preload("User").
		Order("expire_time ASC")

	// 根据筛选条件过滤
	if filter != "" && filter != "all" {
		switch filter {
		case "today":
			todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			todayEnd := todayStart.AddDate(0, 0, 1)
			query = query.Where("expire_time >= ? AND expire_time < ?", todayStart, todayEnd)
		case "1-3":
			day3End := now.AddDate(0, 0, 3)
			query = query.Where("expire_time > ? AND expire_time <= ?", now, day3End)
		case "4-7":
			day3End := now.AddDate(0, 0, 3)
			day7End := now.AddDate(0, 0, 7)
			query = query.Where("expire_time > ? AND expire_time <= ?", day3End, day7End)
		}
	}

	if err := query.Find(&subscriptions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "查询失败",
		})
		return
	}

	// 格式化数据
	result := make([]gin.H, 0, len(subscriptions))
	for _, sub := range subscriptions {
		daysUntilExpire := 0
		if !sub.ExpireTime.IsZero() {
			diff := sub.ExpireTime.Sub(now)
			if diff > 0 {
				daysUntilExpire = int(diff.Hours() / 24)
			}
		}

		userInfo := gin.H{
			"id":       0,
			"username": "用户已删除",
			"email":    "",
			"qq":       "",
		}
		// 检查 User 是否已加载（通过检查 ID 是否为 0）
		if sub.User.ID > 0 {
			// 注意：User 模型中目前没有 QQ 字段，如需使用请先在模型中添加
			userInfo = gin.H{
				"id":       sub.User.ID,
				"username": sub.User.Username,
				"email":    sub.User.Email,
				"qq":       "", // User 模型中暂无 QQ 字段
			}
		}

		result = append(result, gin.H{
			"id":                sub.ID,
			"user_id":           sub.UserID,
			"username":          userInfo["username"],
			"email":             userInfo["email"],
			"qq":                userInfo["qq"],
			"expire_time":       sub.ExpireTime.Format("2006-01-02 15:04:05"),
			"days_until_expire": daysUntilExpire,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

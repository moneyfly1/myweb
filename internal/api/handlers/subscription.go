package handlers

import (
	"fmt"
	"net/http"
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

// GetSubscriptions 获取订阅列表
func GetSubscriptions(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	var subscriptions []models.Subscription
	if err := db.Where("user_id = ?", user.ID).Find(&subscriptions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取订阅列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    subscriptions,
	})
}

// GetSubscription 获取单个订阅
func GetSubscription(c *gin.Context) {
	id := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	var subscription models.Subscription
	if err := db.Where("id = ? AND user_id = ?", id, user.ID).First(&subscription).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "订阅不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "获取订阅失败",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    subscription,
	})
}

// CreateSubscription 创建订阅
func CreateSubscription(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	// 生成订阅 URL
	subscriptionURL := utils.GenerateSubscriptionURL()

	db := database.GetDB()
	subscription := models.Subscription{
		UserID:          user.ID,
		SubscriptionURL: subscriptionURL,
		DeviceLimit:     3,
		CurrentDevices:  0,
		IsActive:        true,
		Status:          "active",
		ExpireTime:      utils.GetBeijingTime().AddDate(0, 1, 0), // 默认1个月
	}

	if err := db.Create(&subscription).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建订阅失败",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    subscription,
	})
}

// GetAdminSubscriptions 管理员获取订阅列表
func GetAdminSubscriptions(c *gin.Context) {
	db := database.GetDB()
	query := db.Preload("User").Model(&models.Subscription{})

	// 分页参数
	page := 1
	size := 20
	if pageStr := c.Query("page"); pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if sizeStr := c.Query("size"); sizeStr != "" {
		fmt.Sscanf(sizeStr, "%d", &size)
	}

	// 搜索参数
	keyword := c.Query("search")
	if keyword == "" {
		keyword = c.Query("keyword") // 兼容两种参数名
	}
	if keyword != "" {
		// 清理和验证搜索关键词，防止SQL注入
		keyword = utils.SanitizeSearchKeyword(keyword)
	}
	if keyword != "" {
		// 搜索逻辑：
		// 1. 当前订阅地址匹配
		// 2. 用户名或邮箱匹配
		// 3. 失效的订阅地址匹配（通过 subscription_resets 表的 old_subscription_url）
		query = query.Where(
			"subscription_url LIKE ? OR user_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ?) OR user_id IN (SELECT DISTINCT user_id FROM subscription_resets WHERE old_subscription_url LIKE ?)",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 状态筛选
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

	// 排序
	sort := c.Query("sort")
	switch sort {
	case "add_time_desc":
		query = query.Order("created_at DESC")
	case "add_time_asc":
		query = query.Order("created_at ASC")
	case "expire_time_desc":
		query = query.Order("expire_time DESC")
	case "expire_time_asc":
		query = query.Order("expire_time ASC")
	case "device_count_desc":
		query = query.Order("current_devices DESC")
	case "device_count_asc":
		query = query.Order("current_devices ASC")
	default:
		query = query.Order("created_at DESC")
	}

	var total int64
	query.Count(&total)

	var subscriptions []models.Subscription
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Find(&subscriptions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取订阅列表失败",
		})
		return
	}

	// 转换为前端需要的格式
	subscriptionList := make([]gin.H, 0)
	for _, sub := range subscriptions {
		// 计算在线设备数，并同步 current_devices 返回值（不落库）
		var onlineDevices int64
		db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", sub.ID, true).Count(&onlineDevices)
		currentDevices := sub.CurrentDevices
		if currentDevices < int(onlineDevices) {
			currentDevices = int(onlineDevices)
		}

		// 统计订阅次数
		// apple_count: v2ray + ssr 订阅次数（通用订阅）
		var appleCount int64
		db.Model(&models.Device{}).Where("subscription_id = ? AND subscription_type IN ?", sub.ID, []string{"v2ray", "ssr"}).Count(&appleCount)

		// clash_count: clash 订阅次数（猫咪订阅）
		var clashCount int64
		db.Model(&models.Device{}).Where("subscription_id = ? AND subscription_type = ?", sub.ID, "clash").Count(&clashCount)

		// 计算到期状态
		now := utils.GetBeijingTime()
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

		// 构造订阅链接（统一格式，与用户端和邮件保持一致）
		baseURL := buildBaseURL(c)
		timestamp := fmt.Sprintf("%d", utils.GetBeijingTime().Unix())
		universalURL := fmt.Sprintf("%s/api/v1/subscriptions/universal/%s?t=%s", baseURL, sub.SubscriptionURL, timestamp) // 通用订阅（Base64格式，适用于小火煎、v2ray等）
		clashURL := fmt.Sprintf("%s/api/v1/subscriptions/clash/%s?t=%s", baseURL, sub.SubscriptionURL, timestamp)         // 猫咪订阅（Clash YAML格式）

		// 构建用户信息对象（前端期望嵌套在 user 中）
		// 检查用户是否存在（如果 Preload 失败，User.ID 会是 0）
		var userInfo gin.H
		var username, email string

		if sub.User.ID == 0 {
			// Preload 失败，尝试从数据库重新查询用户
			var user models.User
			if err := db.First(&user, sub.UserID).Error; err != nil {
				// 用户确实不存在（可能已被删除）
				username = fmt.Sprintf("用户已删除 (ID: %d)", sub.UserID)
				email = fmt.Sprintf("deleted_user_%d", sub.UserID)
				userInfo = gin.H{
					"id":       0,
					"username": username,
					"email":    email,
					"deleted":  true, // 标记用户已删除
				}
			} else {
				// 用户存在，使用查询到的用户信息
				username = user.Username
				email = user.Email
				userInfo = gin.H{
					"id":       user.ID,
					"username": username,
					"email":    email,
				}
			}
		} else {
			// 用户存在，使用 Preload 的用户信息
			username = sub.User.Username
			email = sub.User.Email
			userInfo = gin.H{
				"id":       sub.User.ID,
				"username": username,
				"email":    email,
			}
		}

		subscriptionList = append(subscriptionList, gin.H{
			"id":                sub.ID,
			"user_id":           sub.UserID,
			"user":              userInfo, // 嵌套用户信息
			"username":          username, // 保留顶层字段以兼容
			"email":             email,    // 保留顶层字段以兼容
			"subscription_url":  sub.SubscriptionURL,
			"universal_url":     universalURL, // 通用订阅（Base64格式，适用于小火煎、v2ray等）
			"clash_url":          clashURL,    // 猫咪订阅（Clash YAML格式）
			"status":            sub.Status,
			"is_active":         sub.IsActive,
			"device_limit":      sub.DeviceLimit,
			"current_devices":   currentDevices,
			"online_devices":    onlineDevices,
			"apple_count":       appleCount,
			"clash_count":       clashCount,
			"expire_time":       sub.ExpireTime.Format("2006-01-02 15:04:05"),
			"days_until_expire": daysUntilExpire,
			"is_expired":        isExpired,
			"created_at":        sub.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"subscriptions": subscriptionList,
			"total":         total,
			"page":          page,
			"size":          size,
		},
	})
}

// GetUserSubscriptionDevices 获取当前用户的订阅设备列表（不需要订阅ID）
func GetUserSubscriptionDevices(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()

	// 获取用户的订阅
	var subscription models.Subscription
	if err := db.Where("user_id = ?", user.ID).Order("created_at DESC").First(&subscription).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    []gin.H{},
		})
		return
	}

	// 获取设备列表
	var devices []models.Device
	if err := db.Where("subscription_id = ?", subscription.ID).Find(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取设备列表失败",
		})
		return
	}

	deviceList := make([]gin.H, 0)
	for _, device := range devices {
		// Helper function to safely get string value from pointer
		getStringValue := func(ptr *string) string {
			if ptr != nil {
				return *ptr
			}
			return ""
		}

		deviceName := getStringValue(device.DeviceName)
		deviceType := getStringValue(device.DeviceType)
		ipAddress := getStringValue(device.IPAddress)
		osName := getStringValue(device.OSName)
		osVersion := getStringValue(device.OSVersion)
		userAgent := getStringValue(device.UserAgent)
		softwareName := getStringValue(device.SoftwareName)
		softwareVersion := getStringValue(device.SoftwareVersion)
		deviceModel := getStringValue(device.DeviceModel)
		deviceBrand := getStringValue(device.DeviceBrand)

		deviceList = append(deviceList, gin.H{
			"id":                 device.ID,
			"device_name":        deviceName,
			"name":               deviceName, // 兼容字段
			"device_fingerprint": device.DeviceFingerprint,
			"device_type":        deviceType,
			"type":               deviceType, // 兼容字段
			"ip_address":         ipAddress,
			"ip":                 ipAddress, // 兼容字段
			"os_name":            osName,
			"os_version":         osVersion,
			"last_access":        device.LastAccess.Format("2006-01-02 15:04:05"),
			"last_seen": func() string {
				if device.LastSeen != nil {
					return device.LastSeen.Format("2006-01-02 15:04:05")
				}
				return device.LastAccess.Format("2006-01-02 15:04:05")
			}(), // 兼容字段
			"created_at":       device.CreatedAt.Format("2006-01-02 15:04:05"),
			"is_active":        device.IsActive,
			"is_allowed":       device.IsAllowed,
			"user_agent":       userAgent,
			"software_name":    softwareName,
			"software_version": softwareVersion,
			"device_model":     deviceModel,
			"device_brand":     deviceBrand,
			"access_count":     device.AccessCount, // 使用实际的访问次数
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    deviceList,
	})
}

// GetSubscriptionDevices 获取订阅设备列表（需要订阅ID）
func GetSubscriptionDevices(c *gin.Context) {
	subscriptionID := c.Param("id")

	db := database.GetDB()
	var subscription models.Subscription
	if err := db.First(&subscription, subscriptionID).Error; err != nil {
		status := http.StatusInternalServerError
		msg := "获取订阅失败"
		if err == gorm.ErrRecordNotFound {
			status = http.StatusNotFound
			msg = "订阅不存在"
		}
		c.JSON(status, gin.H{"success": false, "message": msg})
		return
	}

	// 获取设备列表
	var devices []models.Device
	if err := db.Where("subscription_id = ?", subscription.ID).Find(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取设备列表失败",
		})
		return
	}

	deviceList := make([]gin.H, 0)
	for _, device := range devices {
		// Helper function to safely get string value from pointer
		getStringValue := func(ptr *string) string {
			if ptr != nil {
				return *ptr
			}
			return ""
		}

		deviceName := getStringValue(device.DeviceName)
		deviceType := getStringValue(device.DeviceType)
		ipAddress := getStringValue(device.IPAddress)
		osName := getStringValue(device.OSName)
		osVersion := getStringValue(device.OSVersion)
		userAgent := getStringValue(device.UserAgent)
		softwareName := getStringValue(device.SoftwareName)
		softwareVersion := getStringValue(device.SoftwareVersion)
		deviceModel := getStringValue(device.DeviceModel)
		deviceBrand := getStringValue(device.DeviceBrand)

		deviceList = append(deviceList, gin.H{
			"id":                 device.ID,
			"device_name":        deviceName,
			"name":               deviceName, // 兼容字段
			"device_fingerprint": device.DeviceFingerprint,
			"device_type":        deviceType,
			"type":               deviceType, // 兼容字段
			"ip_address":         ipAddress,
			"ip":                 ipAddress, // 兼容字段
			"os_name":            osName,
			"os_version":         osVersion,
			"last_access":        device.LastAccess.Format("2006-01-02 15:04:05"),
			"last_seen":          device.LastAccess.Format("2006-01-02 15:04:05"), // 兼容字段
			"is_active":          device.IsActive,
			"is_allowed":         device.IsAllowed,
			"user_agent":         userAgent,
			"software_name":      softwareName,
			"software_version":   softwareVersion,
			"device_model":       deviceModel,
			"device_brand":       deviceBrand,
			"access_count":       device.AccessCount, // 使用实际的访问次数
			"created_at":         device.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"devices":         deviceList,
			"device_limit":    subscription.DeviceLimit,
			"current_devices": subscription.CurrentDevices,
		},
	})
}

// BatchClearDevices 批量清除设备（管理员）
func BatchClearDevices(c *gin.Context) {
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

	db := database.GetDB()

	// 删除设备
	if err := db.Where("subscription_id IN ?", req.SubscriptionIDs).Delete(&models.Device{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "清除设备失败",
		})
		return
	}

	// 重置订阅的设备计数
	for _, subID := range req.SubscriptionIDs {
		db.Model(&models.Subscription{}).Where("id = ?", subID).Update("current_devices", 0)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "设备已清除",
	})
}

// UpdateSubscription 管理员更新订阅（设备数、到期时间、状态）
func UpdateSubscription(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		DeviceLimit *int    `json:"device_limit"`
		ExpireTime  *string `json:"expire_time"`
		IsActive    *bool   `json:"is_active"`
		Status      string  `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数错误"})
		return
	}

	db := database.GetDB()
	var sub models.Subscription
	if err := db.First(&sub, id).Error; err != nil {
		status := http.StatusInternalServerError
		msg := "订阅不存在"
		if err == gorm.ErrRecordNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"success": false, "message": msg})
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
		t, err := time.Parse("2006-01-02", *req.ExpireTime)
		if err != nil {
			// 尝试带时间
			t, err = time.Parse("2006-01-02 15:04:05", *req.ExpireTime)
		}
		if err == nil {
			sub.ExpireTime = t
		}
	}

	if err := db.Save(&sub).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "更新订阅失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "更新成功"})
}

// ResetSubscription 重置订阅地址
func ResetSubscription(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var sub models.Subscription
	if err := db.First(&sub, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订阅不存在"})
		return
	}

	// 获取用户信息
	var user models.User
	if err := db.First(&user, sub.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户不存在"})
		return
	}

	sub.SubscriptionURL = utils.GenerateSubscriptionURL()
	sub.CurrentDevices = 0

	if err := db.Save(&sub).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "重置订阅失败"})
		return
	}

	// 发送订阅重置邮件
	go func() {
		emailService := email.NewEmailService()
		templateBuilder := email.NewEmailTemplateBuilder()
		baseURL := buildBaseURL(c)
		timestamp := fmt.Sprintf("%d", utils.GetBeijingTime().Unix())
		universalURL := fmt.Sprintf("%s/api/v1/subscriptions/universal/%s?t=%s", baseURL, sub.SubscriptionURL, timestamp) // 通用订阅（Base64格式）
		clashURL := fmt.Sprintf("%s/api/v1/subscriptions/clash/%s?t=%s", baseURL, sub.SubscriptionURL, timestamp)         // 猫咪订阅（Clash YAML格式）
		expireTime := "未设置"
		if !sub.ExpireTime.IsZero() {
			expireTime = sub.ExpireTime.Format("2006-01-02 15:04:05")
		}
		resetTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
		resetReason := "管理员重置"
		content := templateBuilder.GetSubscriptionResetTemplate(user.Username, universalURL, clashURL, expireTime, resetTime, resetReason)
		subject := "订阅重置通知"
		_ = emailService.QueueEmail(user.Email, subject, content, "subscription_reset")

		// 发送管理员通知
		notificationService := notification.NewNotificationService()
		_ = notificationService.SendAdminNotification("subscription_reset", map[string]interface{}{
			"username":   user.Username,
			"email":      user.Email,
			"reset_time": resetTime,
		})
	}()

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "订阅已重置", "data": sub})
}

// ExtendSubscription 延长订阅
func ExtendSubscription(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Days int `json:"days" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Days == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "days 参数必填"})
		return
	}

	db := database.GetDB()
	var sub models.Subscription
	if err := db.First(&sub, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订阅不存在"})
		return
	}

	// 获取用户信息
	var user models.User
	if err := db.First(&user, sub.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户不存在"})
		return
	}

	// 获取套餐信息
	var packageName string
	if sub.PackageID != nil {
		var pkg models.Package
		if err := db.First(&pkg, *sub.PackageID).Error; err == nil {
			packageName = pkg.Name
		}
	}
	if packageName == "" {
		packageName = "默认套餐"
	}

	oldExpiryDate := "未设置"
	if !sub.ExpireTime.IsZero() {
		oldExpiryDate = sub.ExpireTime.Format("2006-01-02 15:04:05")
	}

	if sub.ExpireTime.IsZero() {
		sub.ExpireTime = utils.GetBeijingTime()
	}
	sub.ExpireTime = sub.ExpireTime.AddDate(0, 0, req.Days)

	newExpiryDate := sub.ExpireTime.Format("2006-01-02 15:04:05")

	if err := db.Save(&sub).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "延长订阅失败"})
		return
	}

	// 发送续费成功邮件
	go func() {
		emailService := email.NewEmailService()
		templateBuilder := email.NewEmailTemplateBuilder()
		renewalDate := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
		// 这里假设续费金额为0（管理员延长），实际应该从请求中获取
		amount := 0.0
		content := templateBuilder.GetRenewalConfirmationTemplate(
			user.Username,
			packageName,
			oldExpiryDate,
			newExpiryDate,
			renewalDate,
			amount,
		)
		subject := "续费成功"
		_ = emailService.QueueEmail(user.Email, subject, content, "renewal_confirmation")
	}()

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "订阅已延长", "data": sub})
}

// ResetUserSubscription 重置用户的所有订阅
func ResetUserSubscription(c *gin.Context) {
	userID := c.Param("id")
	db := database.GetDB()
	var subs []models.Subscription
	if err := db.Where("user_id = ?", userID).Find(&subs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取订阅失败"})
		return
	}

	for i := range subs {
		subs[i].SubscriptionURL = utils.GenerateSubscriptionURL()
		subs[i].CurrentDevices = 0
		db.Save(&subs[i])
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "用户订阅已重置"})
}

// SendSubscriptionEmail 发送订阅邮件（管理员）
func SendSubscriptionEmail(c *gin.Context) {
	userID := c.Param("id")
	db := database.GetDB()

	// 获取用户信息
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户不存在"})
		return
	}

	// 获取用户的订阅
	var subscription models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户没有订阅"})
		return
	}

	// 生成订阅链接
	baseURL := buildBaseURL(c)
	timestamp := fmt.Sprintf("%d", utils.GetBeijingTime().Unix())
	universalURL := fmt.Sprintf("%s/api/v1/subscriptions/universal/%s?t=%s", baseURL, subscription.SubscriptionURL, timestamp) // 通用订阅（Base64格式）
	clashURL := fmt.Sprintf("%s/api/v1/subscriptions/clash/%s?t=%s", baseURL, subscription.SubscriptionURL, timestamp)         // 猫咪订阅（Clash YAML格式）

	// 计算到期时间和剩余天数
	expireTime := "未设置"
	remainingDays := 0
	if !subscription.ExpireTime.IsZero() {
		expireTime = subscription.ExpireTime.Format("2006-01-02 15:04:05")
		now := utils.GetBeijingTime()
		diff := subscription.ExpireTime.Sub(now)
		if diff > 0 {
			remainingDays = int(diff.Hours() / 24)
		}
	}

	// 使用新的邮件模板
	emailService := email.NewEmailService()
	templateBuilder := email.NewEmailTemplateBuilder()
	content := templateBuilder.GetSubscriptionTemplate(
		user.Username,
		universalURL,
		clashURL,
		expireTime,
		remainingDays,
		subscription.DeviceLimit,
		subscription.CurrentDevices,
	)
	subject := "服务配置信息"

	// 加入邮件队列
	if err := emailService.QueueEmail(user.Email, subject, content, "subscription"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "发送邮件失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "订阅邮件已加入发送队列"})
}

// ClearUserDevices 清空某用户的设备
func ClearUserDevices(c *gin.Context) {
	userID := c.Param("id")
	db := database.GetDB()

	// 获取用户的所有订阅
	var subscriptions []models.Subscription
	if err := db.Where("user_id = ?", userID).Find(&subscriptions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取订阅失败"})
		return
	}

	// 删除所有设备
	var subscriptionIDs []uint
	for _, sub := range subscriptions {
		subscriptionIDs = append(subscriptionIDs, sub.ID)
	}

	if len(subscriptionIDs) > 0 {
		if err := db.Where("subscription_id IN ?", subscriptionIDs).Delete(&models.Device{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "清理设备失败"})
			return
		}

		// 重置所有订阅的设备计数
		if err := db.Model(&models.Subscription{}).Where("id IN ?", subscriptionIDs).Update("current_devices", 0).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "更新设备计数失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "设备已清理"})
}

// ResetUserSubscriptionSelf 用户重置自己的订阅
func ResetUserSubscriptionSelf(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}

	db := database.GetDB()
	var subscription models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订阅不存在"})
		return
	}

	subscription.SubscriptionURL = utils.GenerateSubscriptionURL()
	subscription.CurrentDevices = 0

	if err := db.Save(&subscription).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "重置订阅失败"})
		return
	}

	// 发送订阅重置邮件
	go func() {
		emailService := email.NewEmailService()
		templateBuilder := email.NewEmailTemplateBuilder()
		baseURL := buildBaseURL(c)
		timestamp := fmt.Sprintf("%d", utils.GetBeijingTime().Unix())
		universalURL := fmt.Sprintf("%s/api/v1/subscriptions/universal/%s?t=%s", baseURL, subscription.SubscriptionURL, timestamp) // 通用订阅（Base64格式）
		clashURL := fmt.Sprintf("%s/api/v1/subscriptions/clash/%s?t=%s", baseURL, subscription.SubscriptionURL, timestamp)
		expireTime := "未设置"
		if !subscription.ExpireTime.IsZero() {
			expireTime = subscription.ExpireTime.Format("2006-01-02 15:04:05")
		}
		resetTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
		resetReason := "用户主动重置"
		content := templateBuilder.GetSubscriptionResetTemplate(user.Username, universalURL, clashURL, expireTime, resetTime, resetReason)
		subject := "订阅重置通知"
		_ = emailService.QueueEmail(user.Email, subject, content, "subscription_reset")

		// 发送管理员通知
		notificationService := notification.NewNotificationService()
		_ = notificationService.SendAdminNotification("subscription_reset", map[string]interface{}{
			"username":   user.Username,
			"email":      user.Email,
			"reset_time": resetTime,
		})
	}()

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "订阅已重置", "data": subscription})
}

// SendSubscriptionEmailSelf 用户发送订阅邮件给自己
func SendSubscriptionEmailSelf(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}

	db := database.GetDB()

	// 获取用户的订阅
	var subscription models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "您还没有订阅"})
		return
	}

	// 生成订阅链接
	baseURL := buildBaseURL(c)
	timestamp := fmt.Sprintf("%d", utils.GetBeijingTime().Unix())
	universalURL := fmt.Sprintf("%s/api/v1/subscriptions/universal/%s?t=%s", baseURL, subscription.SubscriptionURL, timestamp) // 通用订阅（Base64格式）
	clashURL := fmt.Sprintf("%s/api/v1/subscriptions/clash/%s?t=%s", baseURL, subscription.SubscriptionURL, timestamp)         // 猫咪订阅（Clash YAML格式）

	// 计算到期时间和剩余天数
	expireTime := "未设置"
	remainingDays := 0
	if !subscription.ExpireTime.IsZero() {
		expireTime = subscription.ExpireTime.Format("2006-01-02 15:04:05")
		now := utils.GetBeijingTime()
		diff := subscription.ExpireTime.Sub(now)
		if diff > 0 {
			remainingDays = int(diff.Hours() / 24)
		}
	}

	// 使用新的邮件模板
	emailService := email.NewEmailService()
	templateBuilder := email.NewEmailTemplateBuilder()
	content := templateBuilder.GetSubscriptionTemplate(
		user.Username,
		universalURL,
		clashURL,
		expireTime,
		remainingDays,
		subscription.DeviceLimit,
		subscription.CurrentDevices,
	)
	subject := "服务配置信息"

	// 发送管理员通知
	go func() {
		notificationService := notification.NewNotificationService()
		sendTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
		_ = notificationService.SendAdminNotification("subscription_sent", map[string]interface{}{
			"username":  user.Username,
			"email":     user.Email,
			"send_time": sendTime,
		})
	}()

	// 加入邮件队列
	if err := emailService.QueueEmail(user.Email, subject, content, "subscription"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "发送邮件失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "订阅邮件已加入发送队列"})
}

// ConvertSubscriptionToBalance 将订阅转换为余额
func ConvertSubscriptionToBalance(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}

	db := database.GetDB()
	var subscription models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订阅不存在"})
		return
	}

	// 计算剩余天数对应的余额（这里简化处理，实际应该根据套餐价格计算）
	now := utils.GetBeijingTime()
	if subscription.ExpireTime.After(now) {
		daysRemaining := int(subscription.ExpireTime.Sub(now).Hours() / 24)
		// 假设每天价值 1 元（实际应该从套餐价格计算）
		balanceToAdd := float64(daysRemaining) * 1.0

		// 更新用户余额
		user.Balance += balanceToAdd
		if err := db.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "更新余额失败"})
			return
		}

		// 删除订阅
		if err := db.Delete(&subscription).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "删除订阅失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "订阅已转换为余额",
			"data": gin.H{
				"balance_added": balanceToAdd,
				"new_balance":   user.Balance,
			},
		})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "订阅已过期，无法转换"})
	}
}

// RemoveDevice 删除单个设备（管理员）
func RemoveDevice(c *gin.Context) {
	deviceID := c.Param("id")
	db := database.GetDB()

	// 先获取设备信息，以便更新订阅的设备计数
	var device models.Device
	if err := db.First(&device, deviceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "设备不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取设备失败"})
		}
		return
	}

	// 验证设备所属的订阅是否存在（防止IDOR）
	var subscription models.Subscription
	if err := db.First(&subscription, device.SubscriptionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订阅不存在"})
		return
	}

	subscriptionID := device.SubscriptionID

	// 删除设备
	if err := db.Delete(&device).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "删除设备失败"})
		return
	}

	// 更新订阅的设备计数
	var deviceCount int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscriptionID, true).Count(&deviceCount)
	db.Model(&models.Subscription{}).Where("id = ?", subscriptionID).Update("current_devices", deviceCount)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "设备已删除"})
}

// buildBaseURL 根据请求构造带协议的基础 URL
// buildBaseURL 优先使用数据库配置的域名，如果没有则使用请求的 Host
// 统一所有订阅地址生成逻辑，确保管理员后台、用户端和邮件中的订阅地址一致
func buildBaseURL(c *gin.Context) string {
	// 优先从数据库配置获取域名（category = "general"）
	db := database.GetDB()
	if db != nil {
		var config models.SystemConfig
		if err := db.Where("key = ? AND category = ?", "domain_name", "general").First(&config).Error; err == nil && config.Value != "" {
			domain := strings.TrimSpace(config.Value)
			// 如果配置的域名包含协议，直接使用
			if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
				return strings.TrimSuffix(domain, "/")
			}
			// 否则使用当前请求的协议
			scheme := "https"
			if proto := c.Request.Header.Get("X-Forwarded-Proto"); proto != "" {
				scheme = proto
			} else if c.Request.TLS == nil {
				scheme = "http"
			}
			return fmt.Sprintf("%s://%s", scheme, domain)
		}
		// 兼容旧配置（不限制 category）
		if err := db.Where("key = ?", "domain_name").First(&config).Error; err == nil && config.Value != "" {
			domain := strings.TrimSpace(config.Value)
			if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
				return strings.TrimSuffix(domain, "/")
			}
			scheme := "https"
			if proto := c.Request.Header.Get("X-Forwarded-Proto"); proto != "" {
				scheme = proto
			} else if c.Request.TLS == nil {
				scheme = "http"
			}
			return fmt.Sprintf("%s://%s", scheme, domain)
		}
	}

	// 如果没有配置域名，使用请求的 Host
	scheme := "http"
	if proto := c.Request.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if c.Request.TLS != nil {
		scheme = "https"
	}
	host := c.Request.Host
	return fmt.Sprintf("%s://%s", scheme, host)
}

// ExportSubscriptions 导出订阅列表（管理员）- 返回CSV格式
func ExportSubscriptions(c *gin.Context) {
	db := database.GetDB()
	var subscriptions []models.Subscription
	if err := db.Preload("User").Find(&subscriptions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取订阅列表失败",
		})
		return
	}

	// 生成CSV内容
	var csvContent strings.Builder
	csvContent.WriteString("\xEF\xBB\xBF") // UTF-8 BOM，确保Excel正确显示中文
	csvContent.WriteString("ID,用户ID,用户名,邮箱,订阅地址,状态,是否激活,设备限制,当前设备,到期时间,创建时间\n")

	for _, sub := range subscriptions {
		username := sub.User.Username
		email := sub.User.Email
		status := sub.Status
		isActive := "是"
		if !sub.IsActive {
			isActive = "否"
		}

		csvContent.WriteString(fmt.Sprintf("%d,%d,%s,%s,%s,%s,%s,%d,%d,%s,%s\n",
			sub.ID,
			sub.UserID,
			username,
			email,
			sub.SubscriptionURL,
			status,
			isActive,
			sub.DeviceLimit,
			sub.CurrentDevices,
			sub.ExpireTime.Format("2006-01-02 15:04:05"),
			sub.CreatedAt.Format("2006-01-02 15:04:05"),
		))
	}

	// 设置响应头
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=subscriptions_%s.csv", time.Now().Format("20060102")))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", []byte(csvContent.String()))
}

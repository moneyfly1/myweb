package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetAdminProfile 获取管理员个人资料
func GetAdminProfile(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()

	// 从SystemConfig中读取display_name, phone, bio
	displayName := user.Username // 默认使用username
	phone := ""
	bio := ""

	var configs []models.SystemConfig
	db.Where("category = ? AND key IN (?, ?, ?)", "admin_profile",
		fmt.Sprintf("user_%d_display_name", user.ID),
		fmt.Sprintf("user_%d_phone", user.ID),
		fmt.Sprintf("user_%d_bio", user.ID)).Find(&configs)

	for _, config := range configs {
		if strings.Contains(config.Key, "display_name") {
			displayName = config.Value
		} else if strings.Contains(config.Key, "phone") {
			phone = config.Value
		} else if strings.Contains(config.Key, "bio") {
			bio = config.Value
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"id":           user.ID,
		"username":     user.Username,
		"email":        user.Email,
		"is_admin":     user.IsAdmin,
		"avatar_url":   user.Avatar.String,
		"avatar":       user.Avatar.String,
		"display_name": displayName,
		"phone":        phone,
		"bio":          bio,
		"theme":        user.Theme,
		"language":     user.Language,
	})
}

// UpdateAdminProfile 更新管理员个人资料
func UpdateAdminProfile(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
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

	// 更新头像
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

	// 辅助函数：更新配置
	updateConfig := func(key, value string) error {
		configKey := fmt.Sprintf("user_%d_%s", user.ID, key)
		var config models.SystemConfig
		err := db.Where("key = ? AND category = ?", configKey, "admin_profile").First(&config).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				config = models.SystemConfig{
					Key:      configKey,
					Category: "admin_profile",
					Value:    value,
				}
				return db.Create(&config).Error
			}
			return err
		}
		config.Value = value
		return db.Save(&config).Error
	}

	// 更新额外信息
	configs := map[string]string{
		"display_name": req.DisplayName,
		"phone":        req.Phone,
		"bio":          req.Bio,
	}

	for key, value := range configs {
		if value != "" { // 只更新非空值，原逻辑看似允许空值覆盖但语义模糊，此处优化为非空更新，或根据需求调整
			if err := updateConfig(key, value); err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, "更新"+key+"失败", err)
				return
			}
		}
	}

	// 重新读取配置
	displayName := user.Username
	phone := ""
	bio := ""
	if req.DisplayName != "" {
		displayName = req.DisplayName
	}
	if req.Phone != "" {
		phone = req.Phone
	}
	if req.Bio != "" {
		bio = req.Bio
	}

	utils.SuccessResponse(c, http.StatusOK, "个人资料更新成功", gin.H{
		"id":           user.ID,
		"username":     user.Username,
		"email":        user.Email,
		"display_name": displayName,
		"avatar_url":   user.Avatar.String,
		"phone":        phone,
		"bio":          bio,
		"theme":        user.Theme,
		"language":     user.Language,
	})
}

// GetLoginHistory 获取登录历史
func GetLoginHistory(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	var history []models.LoginHistory
	db.Where("user_id = ?", user.ID).Order("login_time DESC").Limit(50).Find(&history)

	historyList := make([]gin.H, 0)
	for _, h := range history {
		country, city := h.GetLocationInfo()
		status := "success"
		if h.LoginStatus != "" {
			status = h.LoginStatus
		}

		historyList = append(historyList, gin.H{
			"id":           h.ID,
			"ip_address":   utils.GetNullStringValue(h.IPAddress),
			"ipAddress":    utils.GetNullStringValue(h.IPAddress), // 兼容字段
			"user_agent":   utils.GetNullStringValue(h.UserAgent),
			"userAgent":    utils.GetNullStringValue(h.UserAgent), // 兼容字段
			"login_time":   h.LoginTime.Format("2006-01-02 15:04:05"),
			"loginTime":    h.LoginTime.Format("2006-01-02 15:04:05"), // 兼容字段
			"login_status": status,
			"status":       status, // 兼容字段
			"country":      country,
			"city":         city,
			"location":     h.Location.String, // 原始位置信息
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", historyList)
}

// GetSecuritySettings 获取安全设置
func GetSecuritySettings(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	var configs []models.SystemConfig
	// 从SystemConfig中读取用户的安全设置
	db.Where("category = ? AND key LIKE ?", "user_security", fmt.Sprintf("user_%d_%%", user.ID)).Find(&configs)

	settings := make(map[string]interface{})
	for _, config := range configs {
		key := strings.TrimPrefix(config.Key, fmt.Sprintf("user_%d_", user.ID))
		if config.Value == "true" || config.Value == "false" {
			settings[key] = config.Value == "true"
		} else {
			settings[key] = config.Value
		}
	}

	// 设置默认值
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

// UpdateAdminSecuritySettings 更新安全设置（管理员个人）
func UpdateAdminSecuritySettings(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	for key, value := range req {
		configKey := fmt.Sprintf("user_%d_%s", user.ID, key)
		var config models.SystemConfig
		if err := db.Where("key = ? AND category = ?", configKey, "user_security").First(&config).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				config = models.SystemConfig{
					Key:      configKey,
					Category: "user_security",
					Value:    fmt.Sprintf("%v", value),
				}
				if err := db.Create(&config).Error; err != nil {
					utils.LogError("UpdateUserSecuritySettings: create config failed", err, map[string]interface{}{"key": key})
					utils.ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("保存配置 %s 失败", key), err)
					return
				}
			} else {
				utils.LogError("UpdateUserSecuritySettings: query config failed", err, map[string]interface{}{"key": key})
				utils.ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("查询配置 %s 失败", key), err)
				return
			}
		} else {
			config.Value = fmt.Sprintf("%v", value)
			if err := db.Save(&config).Error; err != nil {
				utils.LogError("UpdateUserSecuritySettings: update config failed", err, map[string]interface{}{"key": key})
				utils.ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("更新配置 %s 失败", key), err)
				return
			}
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "安全设置已保存", nil)
}

// GetNotificationSettings 获取通知设置
func GetNotificationSettings(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"email_enabled":         user.EmailNotifications,
		"email_notifications":   user.EmailNotifications,
		"system_notification":   true,       // 默认值
		"security_notification": true,       // 默认值
		"frequency":             "realtime", // 默认值
		"sms_notifications":     user.SMSNotifications,
		"push_notifications":    user.PushNotifications,
		"notification_types":    user.NotificationTypes,
	})
}

// UpdateUserNotificationSettings 更新通知设置（用户端）
func UpdateUserNotificationSettings(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()

	// 更新用户表中的通知设置
	if emailNotifications, ok := req["email_notifications"].(bool); ok {
		user.EmailNotifications = emailNotifications
	} else if emailEnabled, ok := req["email_enabled"].(bool); ok {
		user.EmailNotifications = emailEnabled
	}

	if notificationTypes, ok := req["notification_types"].([]interface{}); ok {
		// 将通知类型数组转换为JSON字符串
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

// UpdateAdminNotificationSettings 更新通知设置（管理员个人）
func UpdateAdminNotificationSettings(c *gin.Context) {
	UpdateUserNotificationSettings(c)
}

// GetUserActivities 获取用户活动记录
func GetUserActivities(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	var activities []models.UserActivity
	db.Where("user_id = ?", user.ID).Order("created_at DESC").Limit(100).Find(&activities)

	activityList := make([]gin.H, 0)
	for _, act := range activities {
		activityList = append(activityList, gin.H{
			"id":            act.ID,
			"activity_type": act.ActivityType,
			"description":   act.Description.String,
			"ip_address":    act.IPAddress.String,
			"created_at":    act.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", activityList)
}

// GetSubscriptionResets 获取订阅重置记录
func GetSubscriptionResets(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	var resets []models.SubscriptionReset
	db.Where("user_id = ?", user.ID).Order("created_at DESC").Limit(50).Find(&resets)

	resetList := make([]gin.H, 0)
	for _, reset := range resets {
		resetList = append(resetList, gin.H{
			"id":                  reset.ID,
			"subscription_id":     reset.SubscriptionID,
			"reset_type":          reset.ResetType,
			"reason":              reset.Reason, // Reason 是 string 类型
			"device_count_before": reset.DeviceCountBefore,
			"device_count_after":  reset.DeviceCountAfter,
			"created_at":          reset.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", resetList)
}

// GetUserDevices 获取用户设备列表
func GetUserDevices(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	var devices []models.Device
	db.Where("user_id = ?", user.ID).Order("last_access DESC").Find(&devices)

	deviceList := make([]gin.H, 0)
	for _, device := range devices {
		deviceList = append(deviceList, gin.H{
			"id":              device.ID,
			"subscription_id": device.SubscriptionID,
			"device_name":     utils.GetStringValue(device.DeviceName),
			"device_type":     utils.GetStringValue(device.DeviceType),
			"ip_address":      utils.GetStringValue(device.IPAddress),
			"is_active":       device.IsActive,
			"last_access":     device.LastAccess.Format("2006-01-02 15:04:05"),
			"created_at":      device.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", deviceList)
}

// GetPrivacySettings 获取隐私设置
func GetPrivacySettings(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"data_sharing": user.DataSharing,
		"analytics":    user.Analytics,
	})
}

// UpdatePrivacySettings 更新隐私设置
func UpdatePrivacySettings(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()

	// 更新用户表中的隐私设置
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

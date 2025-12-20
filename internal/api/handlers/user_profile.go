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

// ==================== 公共辅助函数 ====================

// getCurrentUserOrError 获取当前用户，如果未登录则返回错误响应
func getCurrentUserOrError(c *gin.Context) (*models.User, bool) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return nil, false
	}
	return user, true
}

// getUserConfigs 获取用户配置（从 SystemConfig 表）
func getUserConfigs(db *gorm.DB, userID uint, category string, keys []string) map[string]string {
	configs := make(map[string]string, len(keys))

	if len(keys) == 0 {
		return configs
	}

	// 构建查询条件
	keyPatterns := make([]string, len(keys))
	prefix := fmt.Sprintf("user_%d_", userID)
	for i, key := range keys {
		keyPatterns[i] = prefix + key
	}

	var dbConfigs []models.SystemConfig
	db.Where("category = ? AND key IN (?)", category, keyPatterns).Find(&dbConfigs)

	// 提取配置键名（去掉 user_{id}_ 前缀）
	for _, config := range dbConfigs {
		key := strings.TrimPrefix(config.Key, prefix)
		if key != config.Key { // 确保前缀被成功移除
			configs[key] = config.Value
		}
	}

	return configs
}

// updateUserConfig 更新或创建用户配置
func updateUserConfig(db *gorm.DB, userID uint, category, key, value string) error {
	configKey := fmt.Sprintf("user_%d_%s", userID, key)
	var config models.SystemConfig

	err := db.Where("key = ? AND category = ?", configKey, category).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建新配置
			config = models.SystemConfig{
				Key:      configKey,
				Category: category,
				Value:    value,
			}
			return db.Create(&config).Error
		}
		return err
	}

	// 更新现有配置
	config.Value = value
	return db.Save(&config).Error
}

// buildProfileResponse 构建个人资料响应
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

// ==================== 个人资料相关 ====================

// GetAdminProfile 获取管理员个人资料
func GetAdminProfile(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	db := database.GetDB()
	configs := getUserConfigs(db, user.ID, "admin_profile", []string{"display_name", "phone", "bio"})

	utils.SuccessResponse(c, http.StatusOK, "", buildProfileResponse(user, configs))
}

// UpdateAdminProfile 更新管理员个人资料
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

	// 更新用户表中的字段
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

	// 更新配置表中的字段
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

	// 构建响应（使用更新后的值）
	responseConfigs := map[string]string{
		"display_name": req.DisplayName,
		"phone":        req.Phone,
		"bio":          req.Bio,
	}

	utils.SuccessResponse(c, http.StatusOK, "个人资料更新成功", buildProfileResponse(user, responseConfigs))
}

// ==================== 登录历史相关 ====================

// GetLoginHistory 获取登录历史
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
		loginTime := h.LoginTime.Format("2006-01-02 15:04:05")

		historyList = append(historyList, gin.H{
			"id":           h.ID,
			"ip_address":   ipAddr,
			"ipAddress":    ipAddr, // 兼容字段
			"user_agent":   userAgent,
			"userAgent":    userAgent, // 兼容字段
			"login_time":   loginTime,
			"loginTime":    loginTime, // 兼容字段
			"login_status": status,
			"status":       status, // 兼容字段
			"country":      country,
			"city":         city,
			"location":     h.Location.String,
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", historyList)
}

// ==================== 安全设置相关 ====================

// GetSecuritySettings 获取安全设置
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

// ==================== 通知设置相关 ====================

// GetNotificationSettings 获取通知设置
func GetNotificationSettings(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"email_enabled":         user.EmailNotifications,
		"email_notifications":   user.EmailNotifications,
		"system_notification":   true,
		"security_notification": true,
		"frequency":             "realtime",
		"sms_notifications":     user.SMSNotifications,
		"push_notifications":    user.PushNotifications,
		"notification_types":    user.NotificationTypes,
	})
}

// UpdateUserNotificationSettings 更新通知设置（用户端）
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

	// 更新用户表中的通知设置
	if emailNotifications, ok := req["email_notifications"].(bool); ok {
		user.EmailNotifications = emailNotifications
	} else if emailEnabled, ok := req["email_enabled"].(bool); ok {
		user.EmailNotifications = emailEnabled
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

// UpdateAdminNotificationSettings 更新通知设置（管理员个人）
func UpdateAdminNotificationSettings(c *gin.Context) {
	UpdateUserNotificationSettings(c)
}

// ==================== 用户活动相关 ====================

// GetUserActivities 获取用户活动记录
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
			"created_at":    act.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", activityList)
}

// GetSubscriptionResets 获取订阅重置记录
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
			"created_at":          reset.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", resetList)
}

// GetUserDevices 获取用户设备列表
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
			"last_access":     device.LastAccess.Format("2006-01-02 15:04:05"),
			"created_at":      device.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", deviceList)
}

// ==================== 隐私设置相关 ====================

// GetPrivacySettings 获取隐私设置
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

// UpdatePrivacySettings 更新隐私设置
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

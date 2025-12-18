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
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
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
		},
	})
}

// UpdateAdminProfile 更新管理员个人资料
func UpdateAdminProfile(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
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
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()

	// 更新头像（支持 avatar_url 和 avatar 两种字段名）
	if req.AvatarURL != "" {
		user.Avatar = database.NullString(req.AvatarURL)
	} else if req.Avatar != "" {
		user.Avatar = database.NullString(req.Avatar)
	}

	// 更新主题
	if req.Theme != "" {
		user.Theme = req.Theme
	}

	// 更新语言
	if req.Language != "" {
		user.Language = req.Language
	}

	// 保存用户基本信息
	if err := db.Save(user).Error; err != nil {
		utils.LogError("UpdateAdminProfile: save user failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新失败",
		})
		return
	}

	// 将display_name, phone, bio存储在SystemConfig中
	updateConfig := func(key, value string) error {
		var config models.SystemConfig
		configKey := fmt.Sprintf("user_%d_%s", user.ID, key)
		if err := db.Where("key = ? AND category = ?", configKey, "admin_profile").First(&config).Error; err != nil {
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

	// 更新display_name
	if req.DisplayName != "" {
		if err := updateConfig("display_name", req.DisplayName); err != nil {
			utils.LogError("UpdateAdminProfile: update display_name failed", err, map[string]interface{}{
				"user_id": user.ID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "更新显示名称失败",
			})
			return
		}
	}

	// 更新phone
	if req.Phone != "" || req.Phone == "" {
		if err := updateConfig("phone", req.Phone); err != nil {
			utils.LogError("UpdateAdminProfile: update phone failed", err, map[string]interface{}{
				"user_id": user.ID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "更新手机号码失败",
			})
			return
		}
	}

	// 更新bio
	if req.Bio != "" || req.Bio == "" {
		if err := updateConfig("bio", req.Bio); err != nil {
			utils.LogError("UpdateAdminProfile: update bio failed", err, map[string]interface{}{
				"user_id": user.ID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "更新个人简介失败",
			})
			return
		}
	}

	// 重新读取配置以返回最新值
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "个人资料更新成功",
		"data": gin.H{
			"id":           user.ID,
			"username":     user.Username,
			"email":        user.Email,
			"display_name": displayName,
			"avatar_url":   user.Avatar.String,
			"phone":        phone,
			"bio":          bio,
			"theme":        user.Theme,
			"language":     user.Language,
		},
	})
}

// GetLoginHistory 获取登录历史
func GetLoginHistory(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	var history []models.LoginHistory
	db.Where("user_id = ?", user.ID).Order("login_time DESC").Limit(50).Find(&history)

	historyList := make([]gin.H, 0)
	for _, h := range history {
		ipAddress := ""
		if h.IPAddress.Valid {
			ipAddress = h.IPAddress.String
		}
		userAgent := ""
		if h.UserAgent.Valid {
			userAgent = h.UserAgent.String
		}

		// 解析地理位置信息
		country := ""
		city := ""
		if h.Location.Valid && h.Location.String != "" {
			// Location 格式可能是 "国家,城市" 或 JSON 格式
			locationStr := h.Location.String
			if strings.Contains(locationStr, ",") {
				parts := strings.Split(locationStr, ",")
				if len(parts) >= 1 {
					country = strings.TrimSpace(parts[0])
				}
				if len(parts) >= 2 {
					city = strings.TrimSpace(parts[1])
				}
			} else {
				// 尝试解析为JSON
				var locationData map[string]interface{}
				if err := json.Unmarshal([]byte(locationStr), &locationData); err == nil {
					if c, ok := locationData["country"].(string); ok {
						country = c
					}
					if c, ok := locationData["city"].(string); ok {
						city = c
					}
				}
			}
		}

		status := "success"
		if h.LoginStatus != "" {
			status = h.LoginStatus
		}

		historyList = append(historyList, gin.H{
			"id":           h.ID,
			"ip_address":   ipAddress,
			"ipAddress":    ipAddress, // 兼容字段
			"user_agent":   userAgent,
			"userAgent":    userAgent, // 兼容字段
			"login_time":   h.LoginTime.Format("2006-01-02 15:04:05"),
			"loginTime":    h.LoginTime.Format("2006-01-02 15:04:05"), // 兼容字段
			"login_status": status,
			"status":       status, // 兼容字段
			"country":      country,
			"city":         city,
			"location":     h.Location.String, // 原始位置信息
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    historyList,
	})
}

// GetSecuritySettings 获取安全设置
func GetSecuritySettings(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    settings,
	})
}

// UpdateAdminSecuritySettings 更新安全设置（管理员个人）
func UpdateAdminSecuritySettings(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	for key, value := range req {
		var config models.SystemConfig
		configKey := fmt.Sprintf("user_%d_%s", user.ID, key)
		if err := db.Where("key = ? AND category = ?", configKey, "user_security").First(&config).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				config = models.SystemConfig{
					Key:      configKey,
					Category: "user_security",
					Value:    fmt.Sprintf("%v", value),
				}
				if err := db.Create(&config).Error; err != nil {
					utils.LogError("UpdateUserSecuritySettings: create config failed", err, map[string]interface{}{
						"key": key,
					})
					c.JSON(http.StatusInternalServerError, gin.H{
						"success": false,
						"message": fmt.Sprintf("保存配置 %s 失败", key),
					})
					return
				}
			} else {
				utils.LogError("UpdateUserSecuritySettings: query config failed", err, map[string]interface{}{
					"key": key,
				})
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": fmt.Sprintf("查询配置 %s 失败", key),
				})
				return
			}
		} else {
			config.Value = fmt.Sprintf("%v", value)
			if err := db.Save(&config).Error; err != nil {
				utils.LogError("UpdateUserSecuritySettings: update config failed", err, map[string]interface{}{
					"key": key,
				})
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": fmt.Sprintf("更新配置 %s 失败", key),
				})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "安全设置已保存",
	})
}

// GetNotificationSettings 获取通知设置
func GetNotificationSettings(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"email_enabled":         user.EmailNotifications,
			"email_notifications":   user.EmailNotifications,
			"system_notification":   true,       // 默认值
			"security_notification": true,       // 默认值
			"frequency":             "realtime", // 默认值
			"sms_notifications":     user.SMSNotifications,
			"push_notifications":    user.PushNotifications,
			"notification_types":    user.NotificationTypes,
		},
	})
}

// UpdateUserNotificationSettings 更新通知设置（用户端）
func UpdateUserNotificationSettings(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
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
		utils.LogError("UpdateNotificationSettings: save user failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "通知设置已保存",
	})
}

// UpdateAdminNotificationSettings 更新通知设置（管理员个人）
func UpdateAdminNotificationSettings(c *gin.Context) {
	UpdateUserNotificationSettings(c)
}

// GetUserActivities 获取用户活动记录
func GetUserActivities(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    activityList,
	})
}

// GetSubscriptionResets 获取订阅重置记录
func GetSubscriptionResets(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    resetList,
	})
}

// GetUserDevices 获取用户设备列表
func GetUserDevices(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	db := database.GetDB()
	var devices []models.Device
	db.Where("user_id = ?", user.ID).Order("last_access DESC").Find(&devices)

	deviceList := make([]gin.H, 0)
	for _, device := range devices {
		// Helper function to safely get string value from pointer
		getStringValue := func(ptr *string) string {
			if ptr != nil {
				return *ptr
			}
			return ""
		}

		deviceList = append(deviceList, gin.H{
			"id":              device.ID,
			"subscription_id": device.SubscriptionID,
			"device_name":     getStringValue(device.DeviceName),
			"device_type":     getStringValue(device.DeviceType),
			"ip_address":      getStringValue(device.IPAddress),
			"is_active":       device.IsActive,
			"last_access":     device.LastAccess.Format("2006-01-02 15:04:05"),
			"created_at":      device.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    deviceList,
	})
}

// GetPrivacySettings 获取隐私设置
func GetPrivacySettings(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"data_sharing": user.DataSharing,
			"analytics":    user.Analytics,
		},
	})
}

// UpdatePrivacySettings 更新隐私设置
func UpdatePrivacySettings(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
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
		utils.LogError("UpdatePrivacySettings: save user failed", err, map[string]interface{}{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "隐私设置已保存",
	})
}

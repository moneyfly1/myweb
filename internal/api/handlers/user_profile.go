package handlers

import (
	"net/http"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"

	"github.com/gin-gonic/gin"
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"is_admin": user.IsAdmin,
			"avatar":   user.Avatar.String,
			"theme":    user.Theme,
			"language": user.Language,
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
		historyList = append(historyList, gin.H{
			"id":           h.ID,
			"ip_address":   ipAddress,
			"user_agent":   userAgent,
			"login_time":   h.LoginTime.Format("2006-01-02 15:04:05"),
			"login_status": h.LoginStatus,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    historyList,
	})
}

// GetSecuritySettings 获取安全设置
func GetSecuritySettings(c *gin.Context) {
	// 返回安全相关的设置
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"two_factor_enabled":  false,
			"password_changed_at": "",
		},
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
			"email_notifications": user.EmailNotifications,
			"sms_notifications":   user.SMSNotifications,
			"push_notifications":  user.PushNotifications,
			"notification_types":  user.NotificationTypes,
		},
	})
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
			"id":                reset.ID,
			"subscription_id":   reset.SubscriptionID,
			"reset_type":       reset.ResetType,
			"reason":            reset.Reason, // Reason 是 string 类型
			"device_count_before": reset.DeviceCountBefore,
			"device_count_after":  reset.DeviceCountAfter,
			"created_at":        reset.CreatedAt.Format("2006-01-02 15:04:05"),
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
		deviceList = append(deviceList, gin.H{
			"id":              device.ID,
			"subscription_id":  device.SubscriptionID,
			"device_name":     device.DeviceName.String,
			"device_type":     device.DeviceType.String,
			"ip_address":      device.IPAddress.String,
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

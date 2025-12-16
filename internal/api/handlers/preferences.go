package handlers

import (
	"net/http"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"github.com/gin-gonic/gin"
)

// UpdatePreferencesRequest 更新偏好设置请求
type UpdatePreferencesRequest struct {
	Theme              string `json:"theme"`
	Language           string `json:"language"`
	Timezone           string `json:"timezone"`
	EmailNotifications *bool  `json:"email_notifications"`
	SMSNotifications   *bool  `json:"sms_notifications"`
	PushNotifications  *bool  `json:"push_notifications"`
}

// UpdatePreferences 更新用户偏好设置
func UpdatePreferences(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录",
		})
		return
	}

	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "更新成功",
		"data":    user,
	})
}


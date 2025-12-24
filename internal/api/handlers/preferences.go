package handlers

import (
	"net/http"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/utils"

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

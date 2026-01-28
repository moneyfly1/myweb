package handlers

import (
	"fmt"
	"net/http"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

func GetCurrentUserXBoardCompat(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	lastLoginStr := ""
	if user.LastLogin.Valid {
		lastLoginStr = user.LastLogin.Time.Format("2006-01-02 15:04:05")
	}

	responseData := gin.H{
		"id":          user.ID,
		"username":    user.Username,
		"email":       user.Email,
		"is_active":   user.IsActive,
		"is_verified": user.IsVerified,
		"is_admin":    user.IsAdmin,
		"balance":     user.Balance,
		"created_at":  user.CreatedAt.Format("2006-01-02 15:04:05"),
		"last_login":  lastLoginStr,
	}

	if user.Nickname.Valid {
		responseData["nickname"] = user.Nickname.String
	}
	if user.Avatar.Valid {
		responseData["avatar"] = user.Avatar.String
		responseData["avatar_url"] = user.Avatar.String
	}

	c.JSON(http.StatusOK, responseData)
}

func GetUserSubscriptionXBoardCompat(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	var subscription models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}

	baseURL := utils.GetBuildBaseURL(c.Request, db)
	clashURL := fmt.Sprintf("%s/api/v1/subscriptions/clash/%s", baseURL, subscription.SubscriptionURL)
	universalURL := fmt.Sprintf("%s/api/v1/subscriptions/universal/%s", baseURL, subscription.SubscriptionURL)

	expiryDate := ""
	if !subscription.ExpireTime.IsZero() {
		expiryDate = subscription.ExpireTime.Format("2006-01-02T15:04:05Z")
	}

	remainingDays := 0
	isExpired := false
	if !subscription.ExpireTime.IsZero() {
		now := utils.GetBeijingTime()
		diff := subscription.ExpireTime.Sub(now)
		if diff > 0 {
			remainingDays = int(diff.Hours() / 24)
			if diff.Hours() > float64(remainingDays*24) {
				remainingDays++
			}
		} else {
			isExpired = true
		}
	}

	var onlineDevices int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscription.ID, true).Count(&onlineDevices)

	responseData := gin.H{
		"subscribe_url":   clashURL,     // XBoard 期望的字段名
		"universal_url":   universalURL, // 通用订阅 URL
		"expire_time":     expiryDate,   // ISO 8601 格式
		"expiryDate":      expiryDate,   // 兼容字段
		"device_limit":    subscription.DeviceLimit,
		"current_devices": int(onlineDevices),
		"remaining_days":  remainingDays,
		"is_expired":      isExpired,
		"status":          subscription.Status,
		"is_active":       subscription.IsActive,
	}

	c.JSON(http.StatusOK, responseData)
}

func GetClientSubscribeXBoardCompat(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "缺少 token 参数", nil)
		return
	}

	db := database.GetDB()
	var subscription models.Subscription
	if err := db.Where("subscription_url = ?", token).First(&subscription).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		return
	}

	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		var subUser models.User
		if err := db.First(&subUser, subscription.UserID).Error; err != nil {
			utils.ErrorResponse(c, http.StatusNotFound, "订阅关联的用户不存在", err)
			return
		}
		if !subUser.IsActive {
			utils.ErrorResponse(c, http.StatusForbidden, "用户账户已禁用", nil)
			return
		}
	} else {
		if subscription.UserID != user.ID {
			utils.ErrorResponse(c, http.StatusForbidden, "无权访问此订阅", nil)
			return
		}
	}

	now := utils.GetBeijingTime()
	if !subscription.ExpireTime.IsZero() && subscription.ExpireTime.Before(now) {
		utils.ErrorResponse(c, http.StatusForbidden, "订阅已过期", nil)
		return
	}

	if !subscription.IsActive || subscription.Status != "active" {
		utils.ErrorResponse(c, http.StatusForbidden, "订阅未激活", nil)
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/api/v1/subscriptions/clash/%s", subscription.SubscriptionURL))
}

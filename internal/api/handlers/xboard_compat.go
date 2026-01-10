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

// GetCurrentUserXBoardCompat XBoard 兼容的用户信息接口
//
// 路径: GET /api/v1/user/info
//
// 这个函数只是调用现有的 GetCurrentUser，然后转换响应格式
// 不会影响原有的 /api/v1/users/me 接口
func GetCurrentUserXBoardCompat(c *gin.Context) {
	// 1. 获取当前用户（使用现有的逻辑）
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	// 2. 格式化时间字段
	lastLoginStr := ""
	if user.LastLogin.Valid {
		lastLoginStr = user.LastLogin.Time.Format("2006-01-02 15:04:05")
	}

	// 3. 构建 XBoard 期望的响应格式
	// 注意：这里直接返回数据，不使用 utils.SuccessResponse 的包装
	// 因为 XBoard 可能期望不同的响应格式
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

	// 添加可选字段
	if user.Nickname.Valid {
		responseData["nickname"] = user.Nickname.String
	}
	if user.Avatar.Valid {
		responseData["avatar"] = user.Avatar.String
		responseData["avatar_url"] = user.Avatar.String
	}

	// 4. 返回响应（XBoard 可能期望直接返回数据，而不是包装在 data 字段中）
	// 如果 XBoard SDK 期望 {code, message, data} 格式，则使用 utils.SuccessResponse
	// 如果期望直接返回数据，则使用下面的代码
	c.JSON(http.StatusOK, responseData)
}

// GetUserSubscriptionXBoardCompat XBoard 兼容的订阅信息接口
//
// 路径: GET /api/v1/user/subscribe
//
// 这个函数调用现有的 GetUserSubscription 逻辑，然后转换响应格式
// 不会影响原有的 /api/v1/subscriptions/user-subscription 接口
func GetUserSubscriptionXBoardCompat(c *gin.Context) {
	// 1. 获取当前用户
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	// 2. 获取订阅信息（使用现有的逻辑）
	db := database.GetDB()
	var subscription models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		// 用户没有订阅，返回空数据（XBoard 可能期望 null 或空对象）
		c.JSON(http.StatusOK, gin.H{})
		return
	}

	// 3. 生成订阅地址
	baseURL := utils.GetBuildBaseURL(c.Request, db)
	timestamp := fmt.Sprintf("%d", utils.GetBeijingTime().Unix())
	clashURL := fmt.Sprintf("%s/api/v1/subscriptions/clash/%s?t=%s", baseURL, subscription.SubscriptionURL, timestamp)
	universalURL := fmt.Sprintf("%s/api/v1/subscriptions/universal/%s?t=%s", baseURL, subscription.SubscriptionURL, timestamp)

	// 4. 计算到期时间
	expiryDate := ""
	if !subscription.ExpireTime.IsZero() {
		expiryDate = subscription.ExpireTime.Format("2006-01-02T15:04:05Z")
	}

	// 5. 计算剩余天数
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

	// 6. 获取在线设备数
	var onlineDevices int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscription.ID, true).Count(&onlineDevices)

	// 7. 构建 XBoard 期望的响应格式
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

	// 8. 返回响应（XBoard 可能期望直接返回数据）
	c.JSON(http.StatusOK, responseData)
}

// GetClientSubscribeXBoardCompat XBoard 兼容的客户端订阅接口
//
// 路径: GET /api/v1/client/subscribe?token=xxx
//
// 这个函数将 token 参数转换为订阅 URL，然后调用现有的订阅配置接口
func GetClientSubscribeXBoardCompat(c *gin.Context) {
	// 1. 从查询参数获取 token
	token := c.Query("token")
	if token == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "缺少 token 参数", nil)
		return
	}

	// 2. 根据 token 查找订阅
	db := database.GetDB()
	var subscription models.Subscription
	if err := db.Where("subscription_url = ?", token).First(&subscription).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		return
	}

	// 3. 检查订阅状态
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		// 如果没有认证，检查订阅是否属于某个用户
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
		// 如果已认证，检查订阅是否属于当前用户
		if subscription.UserID != user.ID {
			utils.ErrorResponse(c, http.StatusForbidden, "无权访问此订阅", nil)
			return
		}
	}

	// 4. 检查订阅是否过期
	now := utils.GetBeijingTime()
	if !subscription.ExpireTime.IsZero() && subscription.ExpireTime.Before(now) {
		utils.ErrorResponse(c, http.StatusForbidden, "订阅已过期", nil)
		return
	}

	// 5. 检查订阅是否激活
	if !subscription.IsActive || subscription.Status != "active" {
		utils.ErrorResponse(c, http.StatusForbidden, "订阅未激活", nil)
		return
	}

	// 6. 重定向到 Clash 订阅接口（使用现有的处理函数）
	// 或者直接调用 GetSubscriptionConfig
	c.Redirect(http.StatusFound, fmt.Sprintf("/api/v1/subscriptions/clash/%s", subscription.SubscriptionURL))
}

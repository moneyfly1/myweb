package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"
	"cboard-go/internal/services/device"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetCurrentUserXBoardCompat(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	lastLoginStr := utils.FormatNullTimeBeijing(user.LastLogin)

	responseData := gin.H{
		"id":          user.ID,
		"username":    user.Username,
		"email":       user.Email,
		"is_active":   user.IsActive,
		"is_verified": user.IsVerified,
		"is_admin":    user.IsAdmin,
		"balance":     user.Balance,
		"created_at":  utils.FormatBeijingTime(user.CreatedAt),
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
	// 订阅内容必须实时反映节点增删，禁止任何客户端/代理缓存
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	c.Header("Pragma", "no-cache")

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

	urls := getMultiClientSubscriptionURLs(c, subscription.SubscriptionURL)

	clashURL := urls["clash_url"].(string)
	universalURL := urls["universal_url"].(string)

	expiryDate := ""
	if !subscription.ExpireTime.IsZero() {
		expiryDate = utils.FormatBeijingRFC3339(subscription.ExpireTime)
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
	// 订阅内容必须实时反映节点增删，禁止任何客户端/代理缓存
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	c.Header("Pragma", "no-cache")

	if shouldBlockBrowserSubscriptionAccess(c) {
		respondEmptySubscriptionForBrowser(c)
		return
	}

	token := c.Query("token")
	if token == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "缺少 token 参数", nil)
		return
	}

	subType := c.Query("type")
	if subType == "" {
		subType = c.Query("format")
	}
	if subType == "" {
		// 根据 User-Agent 自动检测客户端类型
		ua := c.GetHeader("User-Agent")
		subType = detectClientType(ua)
	}

	db := database.GetDB()
	var subscription models.Subscription
	if err := db.Where("subscription_url = ?", token).First(&subscription).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", nil)
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询订阅失败", err)
		return
	}

	// 验证订阅状态
	now := utils.GetBeijingTime()
	isExpired := subscription.ExpireTime.Before(now)
	isInactive := !subscription.IsActive || subscription.Status != "active"

	var user models.User
	var isSpecialValid bool
	if err := db.First(&user, subscription.UserID).Error; err == nil {
		isSpecialValid = user.SpecialNodeExpiresAt.Valid && user.SpecialNodeExpiresAt.Time.After(now)
	}

	if isExpired && !isSpecialValid {
		baseURL := utils.GetBuildBaseURL(c.Request, db)
		content, _, _ := config_update.NewConfigUpdateService().GenerateClientConfig(token, utils.GetRealClientIP(c), c.GetHeader("User-Agent"), subType)
		if content != "" {
			c.String(200, content)
			return
		}
		c.String(200, generateErrorConfigBase64("订阅已过期", fmt.Sprintf("到期时间: %s，请续费", subscription.ExpireTime.Format(DateFormat)), baseURL))
		return
	}
	if isInactive {
		baseURL := utils.GetBuildBaseURL(c.Request, db)
		c.String(200, generateErrorConfigBase64("订阅已失效", "订阅已被禁用或状态异常，请联系客服", baseURL))
		return
	}

	clientIP := utils.GetRealClientIP(c)
	userAgent := c.GetHeader("User-Agent")

	// 设备管理
	deviceManager := device.NewDeviceManager()
	_, deviceExists, _ := deviceManager.FindExistingDevice(subscription.ID, userAgent, clientIP)

	var count int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscription.ID, true).Count(&count)

	shouldRecord := true
	if !deviceExists && !user.SpecialNodeUnlimitedDevices {
		if subscription.DeviceLimit == 0 || (subscription.DeviceLimit > 0 && int(count) >= subscription.DeviceLimit) {
			shouldRecord = false
		}
	}

	if shouldRecord {
		go func(subID, userID uint, ua, ip string) {
			deviceManager.RecordDeviceAccess(subID, userID, ua, ip, subType)
		}(subscription.ID, subscription.UserID, userAgent, clientIP)

		go func(subID uint) {
			db.Model(&models.Subscription{}).Where("id = ?", subID).
				UpdateColumn(fmt.Sprintf("%s_count", safeSubTypeForDB(subType)), gorm.Expr(fmt.Sprintf("%s_count + 1", safeSubTypeForDB(subType))))
		}(subscription.ID)
	}

	// 生成配置
	configService := config_update.NewConfigUpdateService()
	var config, contentType, fileName string
	if excludedProtocols := parseExcludedProtocols(c.Query("exclude")); len(excludedProtocols) > 0 {
		config, contentType, fileName = configService.GenerateClientConfigWithExcludedProtocols(token, clientIP, userAgent, subType, excludedProtocols)
	} else {
		config, contentType, fileName = configService.GenerateClientConfig(token, clientIP, userAgent, subType)
	}

	subscriptionName := configService.GenerateSubscriptionName(
		configService.GetSubscriptionContext(token, clientIP, userAgent),
	)
	encodedName := url.QueryEscape(fileName)

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", encodedName))
	c.Header("Subscription-Title", subscriptionName)
	c.Header("Profile-Title", subscriptionName)
	// 不发送 Subscription-Userinfo 头：去掉流量与到期时间显示，客户端只显示"更新时间: X 分钟前"
	// 更新间隔（单位：分钟）：60 = 1 小时，客户端导入后按此间隔自动更新订阅
	c.Header("Profile-Update-Interval", "60")
	// 最后更新时间（Unix 时间戳）= 本次配置生成时间
	c.Header("Profile-Update-Time", fmt.Sprintf("%d", time.Now().Unix()))

	c.String(200, config)
}

// detectClientType 根据 User-Agent 检测客户端类型
func detectClientType(ua string) string {
	uaLower := strings.ToLower(ua)
	switch {
	case strings.Contains(uaLower, "clash"):
		return "clash"
	case strings.Contains(uaLower, "stash"):
		return "stash"
	case strings.Contains(uaLower, "surge"):
		return "surge"
	case strings.Contains(uaLower, "quantumult"):
		return "quantumultx"
	case strings.Contains(uaLower, "loon"):
		return "loon"
	case strings.Contains(uaLower, "sing-box") || strings.Contains(uaLower, "singbox"):
		return "singbox"
	case strings.Contains(uaLower, "shadowrocket"):
		return "shadowrocket"
	default:
		return "universal"
	}
}

func parseExcludedProtocols(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	protocols := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		protocol := strings.ToLower(strings.TrimSpace(part))
		if protocol == "" || seen[protocol] {
			continue
		}
		seen[protocol] = true
		protocols = append(protocols, protocol)
	}
	return protocols
}

// safeSubTypeForDB maps client type to the DB counter column suffix
func safeSubTypeForDB(t string) string {
	switch t {
	case "clash", "clashmeta", "stash":
		return "clash"
	case "surge":
		return "surge"
	case "shadowrocket":
		return "universal"
	case "quantumultx", "quantumult":
		return "universal"
	case "singbox", "sing-box":
		return "universal"
	case "loon":
		return "universal"
	default:
		return "universal"
	}
}

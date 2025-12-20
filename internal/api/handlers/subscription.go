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

// --- 辅助私有函数 ---

func getSubscriptionURLs(c *gin.Context, subURL string) (string, string) {
	baseURL := buildBaseURL(c)
	timestamp := fmt.Sprintf("%d", utils.GetBeijingTime().Unix())
	universal := fmt.Sprintf("%s/api/v1/subscriptions/universal/%s?t=%s", baseURL, subURL, timestamp)
	clash := fmt.Sprintf("%s/api/v1/subscriptions/clash/%s?t=%s", baseURL, subURL, timestamp)
	return universal, clash
}

func formatDeviceList(devices []models.Device) []gin.H {
	deviceList := make([]gin.H, 0)
	getString := func(ptr *string) string {
		if ptr != nil {
			return *ptr
		}
		return ""
	}
	for _, d := range devices {
		lastSeen := d.LastAccess.Format("2006-01-02 15:04:05")
		if d.LastSeen != nil {
			lastSeen = d.LastSeen.Format("2006-01-02 15:04:05")
		}
		deviceList = append(deviceList, gin.H{
			"id":                 d.ID,
			"device_name":        getString(d.DeviceName),
			"name":               getString(d.DeviceName),
			"device_fingerprint": d.DeviceFingerprint,
			"device_type":        getString(d.DeviceType),
			"type":               getString(d.DeviceType),
			"ip_address":         getString(d.IPAddress),
			"ip":                 getString(d.IPAddress),
			"os_name":            getString(d.OSName),
			"os_version":         getString(d.OSVersion),
			"last_access":        d.LastAccess.Format("2006-01-02 15:04:05"),
			"last_seen":          lastSeen,
			"created_at":         d.CreatedAt.Format("2006-01-02 15:04:05"),
			"is_active":          d.IsActive,
			"is_allowed":         d.IsAllowed,
			"user_agent":         getString(d.UserAgent),
			"software_name":      getString(d.SoftwareName),
			"software_version":   getString(d.SoftwareVersion),
			"device_model":       getString(d.DeviceModel),
			"device_brand":       getString(d.DeviceBrand),
			"access_count":       d.AccessCount,
		})
	}
	return deviceList
}

// --- 处理函数 ---

func GetSubscriptions(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}
	var subscriptions []models.Subscription
	if err := database.GetDB().Where("user_id = ?", user.ID).Find(&subscriptions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取订阅列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": subscriptions})
}

func GetSubscription(c *gin.Context) {
	id := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}
	var sub models.Subscription
	if err := database.GetDB().Where("id = ? AND user_id = ?", id, user.ID).First(&sub).Error; err != nil {
		msg := "获取订阅失败"
		if err == gorm.ErrRecordNotFound {
			msg = "订阅不存在"
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": sub})
}

func CreateSubscription(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}
	db := database.GetDB()
	deviceLimit, durationMonths := getDefaultSubscriptionSettings(db)
	sub := models.Subscription{
		UserID:          user.ID,
		SubscriptionURL: utils.GenerateSubscriptionURL(),
		DeviceLimit:     deviceLimit,
		CurrentDevices:  0,
		IsActive:        true,
		Status:          "active",
		ExpireTime:      utils.GetBeijingTime().AddDate(0, durationMonths, 0),
	}
	if err := db.Create(&sub).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "创建订阅失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": sub})
}

func GetAdminSubscriptions(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.Subscription{})
	page, size := 1, 20
	fmt.Sscanf(c.Query("page"), "%d", &page)
	fmt.Sscanf(c.Query("size"), "%d", &size)

	if keyword := utils.SanitizeSearchKeyword(c.DefaultQuery("search", c.Query("keyword"))); keyword != "" {
		query = query.Where(
			"subscription_url LIKE ? OR user_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ?) OR user_id IN (SELECT DISTINCT user_id FROM subscription_resets WHERE old_subscription_url LIKE ?)",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

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

	sort := c.DefaultQuery("sort", "add_time_desc")
	sortMap := map[string]string{
		"add_time_desc": "created_at DESC", "add_time_asc": "created_at ASC",
		"expire_time_desc": "expire_time DESC", "expire_time_asc": "expire_time ASC",
		"device_count_desc": "current_devices DESC", "device_count_asc": "current_devices ASC",
	}
	if order, ok := sortMap[sort]; ok {
		query = query.Order(order)
	} else {
		query = query.Order("created_at DESC")
	}

	var total int64
	query.Count(&total)
	var subscriptions []models.Subscription
	if err := query.Offset((page - 1) * size).Limit(size).Find(&subscriptions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取订阅列表失败"})
		return
	}

	list := make([]gin.H, 0)
	if len(subscriptions) > 0 {
		subIDs := make([]uint, len(subscriptions))
		for i, s := range subscriptions {
			subIDs[i] = s.ID
		}

		type Stat struct {
			SubID uint
			Type  *string
			Count int64
		}
		var onlineStats []Stat
		db.Model(&models.Device{}).Select("subscription_id as sub_id, count(*) as count").Where("subscription_id IN ? AND is_active = ?", subIDs, true).Group("subscription_id").Scan(&onlineStats)

		var typeStats []Stat
		db.Model(&models.Device{}).Select("subscription_id as sub_id, subscription_type as type, count(*) as count").Where("subscription_id IN ?", subIDs).Group("subscription_id, subscription_type").Scan(&typeStats)

		onlineMap, appleMap, clashMap := make(map[uint]int64), make(map[uint]int64), make(map[uint]int64)
		for _, s := range onlineStats {
			onlineMap[s.SubID] = s.Count
		}
		for _, s := range typeStats {
			if s.Type == nil {
				continue
			}
			if *s.Type == "v2ray" || *s.Type == "ssr" {
				appleMap[s.SubID] += s.Count
			}
			if *s.Type == "clash" {
				clashMap[s.SubID] += s.Count
			}
		}

		for _, sub := range subscriptions {
			online := onlineMap[sub.ID]
			curr := sub.CurrentDevices
			if curr < int(online) {
				curr = int(online)
			}

			universal, clash := getSubscriptionURLs(c, sub.SubscriptionURL)
			var user models.User
			userInfo := gin.H{"id": 0, "username": fmt.Sprintf("用户已删除 (ID: %d)", sub.UserID), "email": fmt.Sprintf("deleted_user_%d", sub.UserID), "deleted": true}
			if db.First(&user, sub.UserID).Error == nil {
				userInfo = gin.H{"id": user.ID, "username": user.Username, "email": user.Email}
			}

			daysUntil, isExpired, now := 0, false, utils.GetBeijingTime()
			if !sub.ExpireTime.IsZero() {
				if diff := sub.ExpireTime.Sub(now); diff > 0 {
					daysUntil = int(diff.Hours() / 24)
				} else {
					isExpired = true
				}
			}

			list = append(list, gin.H{
				"id": sub.ID, "user_id": sub.UserID, "user": userInfo, "username": userInfo["username"], "email": userInfo["email"],
				"subscription_url": sub.SubscriptionURL, "universal_url": universal, "clash_url": clash,
				"status": sub.Status, "is_active": sub.IsActive, "device_limit": sub.DeviceLimit,
				"current_devices": curr, "online_devices": online, "apple_count": appleMap[sub.ID], "clash_count": clashMap[sub.ID],
				"expire_time": sub.ExpireTime.Format("2006-01-02 15:04:05"), "days_until_expire": daysUntil, "is_expired": isExpired,
				"created_at": sub.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"subscriptions": list, "total": total, "page": page, "size": size}})
}

func GetUserSubscriptionDevices(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}
	db := database.GetDB()
	var sub models.Subscription
	if err := db.Where("user_id = ?", user.ID).Order("created_at DESC").First(&sub).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": []gin.H{}})
		return
	}
	var devices []models.Device
	db.Where("subscription_id = ?", sub.ID).Find(&devices)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": formatDeviceList(devices)})
}

func GetSubscriptionDevices(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var sub models.Subscription
	if err := db.First(&sub, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订阅不存在"})
		return
	}
	var devices []models.Device
	db.Where("subscription_id = ?", sub.ID).Find(&devices)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"devices": formatDeviceList(devices), "device_limit": sub.DeviceLimit, "current_devices": sub.CurrentDevices,
	}})
}

func BatchClearDevices(c *gin.Context) {
	var req struct {
		SubscriptionIDs []uint `json:"subscription_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	db := database.GetDB()
	db.Where("subscription_id IN ?", req.SubscriptionIDs).Delete(&models.Device{})
	db.Model(&models.Subscription{}).Where("id IN ?", req.SubscriptionIDs).Update("current_devices", 0)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "设备已清除"})
}

func UpdateSubscription(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		DeviceLimit *int    `json:"device_limit"`
		ExpireTime  *string `json:"expire_time"`
		IsActive    *bool   `json:"is_active"`
		Status      string  `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	db := database.GetDB()
	var sub models.Subscription
	if err := db.First(&sub, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订阅不存在"})
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
		if t, err := time.Parse("2006-01-02", *req.ExpireTime); err == nil {
			sub.ExpireTime = t
		} else if t, err := time.Parse("2006-01-02 15:04:05", *req.ExpireTime); err == nil {
			sub.ExpireTime = t
		}
	}
	db.Save(&sub)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "更新成功"})
}

func ResetSubscription(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var sub models.Subscription
	if err := db.First(&sub, id).Preload("User").Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订阅不存在"})
		return
	}
	sub.SubscriptionURL = utils.GenerateSubscriptionURL()
	sub.CurrentDevices = 0
	db.Save(&sub)
	go sendResetEmail(c, sub, sub.User, "管理员重置")
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "订阅已重置", "data": sub})
}

func ExtendSubscription(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Days int `json:"days" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "days 必填"})
		return
	}
	db := database.GetDB()
	var sub models.Subscription
	if err := db.First(&sub, id).Preload("User").Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订阅不存在"})
		return
	}
	oldExp := "未设置"
	if !sub.ExpireTime.IsZero() {
		oldExp = sub.ExpireTime.Format("2006-01-02 15:04:05")
	}
	if sub.ExpireTime.IsZero() {
		sub.ExpireTime = utils.GetBeijingTime()
	}
	sub.ExpireTime = sub.ExpireTime.AddDate(0, 0, req.Days)
	db.Save(&sub)

	go func() {
		pkgName := "默认套餐"
		if sub.PackageID != nil {
			var pkg models.Package
			if err := db.First(&pkg, *sub.PackageID).Error; err == nil {
				pkgName = pkg.Name
			}
		}
		email.NewEmailService().QueueEmail(sub.User.Email, "续费成功",
			email.NewEmailTemplateBuilder().GetRenewalConfirmationTemplate(sub.User.Username, pkgName, oldExp, sub.ExpireTime.Format("2006-01-02 15:04:05"), utils.GetBeijingTime().Format("2006-01-02 15:04:05"), 0), "renewal_confirmation")
	}()
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "订阅已延长", "data": sub})
}

func ResetUserSubscription(c *gin.Context) {
	userID := c.Param("id")
	db := database.GetDB()
	db.Model(&models.Subscription{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"subscription_url": gorm.Expr("?", utils.GenerateSubscriptionURL()), "current_devices": 0,
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "用户订阅已重置"})
}

func SendSubscriptionEmail(c *gin.Context) {
	userID := c.Param("id")
	db := database.GetDB()
	var user models.User
	var sub models.Subscription
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户不存在"})
		return
	}
	if err := db.Where("user_id = ?", user.ID).First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户没有订阅"})
		return
	}
	if err := queueSubEmail(c, sub, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "发送邮件失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "订阅邮件已加入队列"})
}

func ClearUserDevices(c *gin.Context) {
	userID := c.Param("id")
	db := database.GetDB()
	var subIDs []uint
	db.Model(&models.Subscription{}).Where("user_id = ?", userID).Pluck("id", &subIDs)
	if len(subIDs) > 0 {
		db.Where("subscription_id IN ?", subIDs).Delete(&models.Device{})
		db.Model(&models.Subscription{}).Where("id IN ?", subIDs).Update("current_devices", 0)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "设备已清理"})
}

func ResetUserSubscriptionSelf(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}
	db := database.GetDB()
	var sub models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订阅不存在"})
		return
	}
	sub.SubscriptionURL = utils.GenerateSubscriptionURL()
	sub.CurrentDevices = 0
	db.Save(&sub)
	go sendResetEmail(c, sub, *user, "用户主动重置")
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "订阅已重置", "data": sub})
}

func SendSubscriptionEmailSelf(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}
	var sub models.Subscription
	if err := database.GetDB().Where("user_id = ?", user.ID).First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "您还没有订阅"})
		return
	}
	go notification.NewNotificationService().SendAdminNotification("subscription_sent", map[string]interface{}{"username": user.Username, "email": user.Email, "send_time": utils.GetBeijingTime().Format("2006-01-02 15:04:05")})
	if err := queueSubEmail(c, sub, *user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "发送邮件失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "订阅邮件已加入队列"})
}

func ConvertSubscriptionToBalance(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}
	db := database.GetDB()
	var sub models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订阅不存在"})
		return
	}
	now := utils.GetBeijingTime()
	if sub.ExpireTime.After(now) {
		days := int(sub.ExpireTime.Sub(now).Hours() / 24)
		added := float64(days) * 1.0
		user.Balance += added
		db.Save(user)
		db.Delete(&sub)
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "已转换为余额", "data": gin.H{"balance_added": added, "new_balance": user.Balance}})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "订阅已过期"})
	}
}

func RemoveDevice(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var d models.Device
	if err := db.First(&d, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "设备不存在"})
		return
	}
	subID := d.SubscriptionID
	db.Delete(&d)
	var count int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subID, true).Count(&count)
	db.Model(&models.Subscription{}).Where("id = ?", subID).Update("current_devices", count)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "设备已删除"})
}

func ExportSubscriptions(c *gin.Context) {
	var subs []models.Subscription
	if err := database.GetDB().Preload("User").Find(&subs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取列表失败"})
		return
	}
	var csv strings.Builder
	csv.WriteString("\xEF\xBB\xBFID,用户ID,用户名,邮箱,订阅地址,状态,是否激活,设备限制,当前设备,到期时间,创建时间\n")
	for _, s := range subs {
		active := "是"
		if !s.IsActive {
			active = "否"
		}
		csv.WriteString(fmt.Sprintf("%d,%d,%s,%s,%s,%s,%s,%d,%d,%s,%s\n", s.ID, s.UserID, s.User.Username, s.User.Email, s.SubscriptionURL, s.Status, active, s.DeviceLimit, s.CurrentDevices, s.ExpireTime.Format("2006-01-02 15:04:05"), s.CreatedAt.Format("2006-01-02 15:04:05")))
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=subscriptions_%s.csv", time.Now().Format("20060102")))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", []byte(csv.String()))
}

// --- 内部邮件工具 ---

func sendResetEmail(c *gin.Context, sub models.Subscription, user models.User, reason string) {
	univ, clash := getSubscriptionURLs(c, sub.SubscriptionURL)
	exp := "未设置"
	if !sub.ExpireTime.IsZero() {
		exp = sub.ExpireTime.Format("2006-01-02 15:04:05")
	}
	resetTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
	content := email.NewEmailTemplateBuilder().GetSubscriptionResetTemplate(user.Username, univ, clash, exp, resetTime, reason)
	_ = email.NewEmailService().QueueEmail(user.Email, "订阅重置通知", content, "subscription_reset")
	_ = notification.NewNotificationService().SendAdminNotification("subscription_reset", map[string]interface{}{"username": user.Username, "email": user.Email, "reset_time": resetTime})
}

func queueSubEmail(c *gin.Context, sub models.Subscription, user models.User) error {
	univ, clash := getSubscriptionURLs(c, sub.SubscriptionURL)
	exp, days := "未设置", 0
	if !sub.ExpireTime.IsZero() {
		exp = sub.ExpireTime.Format("2006-01-02 15:04:05")
		if diff := sub.ExpireTime.Sub(utils.GetBeijingTime()); diff > 0 {
			days = int(diff.Hours() / 24)
		}
	}
	content := email.NewEmailTemplateBuilder().GetSubscriptionTemplate(user.Username, univ, clash, exp, days, sub.DeviceLimit, sub.CurrentDevices)
	return email.NewEmailService().QueueEmail(user.Email, "服务配置信息", content, "subscription")
}

func buildBaseURL(c *gin.Context) string {
	db := database.GetDB()
	var config models.SystemConfig
	if db != nil && db.Where("key = ? AND category = ?", "domain_name", "general").First(&config).Error == nil && config.Value != "" {
		domain := strings.TrimSuffix(strings.TrimSpace(config.Value), "/")
		if strings.HasPrefix(domain, "http") {
			return domain
		}
		scheme := "https"
		if proto := c.Request.Header.Get("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else if c.Request.TLS == nil {
			scheme = "http"
		}
		return fmt.Sprintf("%s://%s", scheme, domain)
	}
	scheme := "http"
	if proto := c.Request.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if c.Request.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, c.Request.Host)
}

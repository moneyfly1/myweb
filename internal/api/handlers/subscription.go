package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/geoip"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const timeLayout = "2006-01-02 15:04:05"

// --- Helper Functions ---

func getSubscriptionURLs(c *gin.Context, subURL string) (string, string) {
	baseURL := utils.GetBuildBaseURL(c.Request, database.GetDB())
	return fmt.Sprintf("%s/api/v1/subscriptions/universal/%s", baseURL, subURL),
		fmt.Sprintf("%s/api/v1/subscriptions/clash/%s", baseURL, subURL)
}

func getCurrentAdminUsername(c *gin.Context) *string {
	if user, ok := middleware.GetCurrentUser(c); ok && user != nil {
		return &user.Username
	}
	return nil
}

func getString(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

func formatIP(ip string) string {
	if ip == "" {
		return "-"
	}
	if ip == "::1" {
		return "127.0.0.1"
	}
	if strings.HasPrefix(ip, "::ffff:") {
		return strings.TrimPrefix(ip, "::ffff:")
	}
	return ip
}

func formatDeviceList(devices []models.Device) []gin.H {
	list := make([]gin.H, 0, len(devices))
	geoEnabled := geoip.IsEnabled()

	for _, d := range devices {
		lastSeen := d.LastAccess.Format(timeLayout)
		if d.LastSeen != nil {
			lastSeen = d.LastSeen.Format(timeLayout)
		}
		ipAddress := formatIP(getString(d.IPAddress))
		location := ""
		if ipAddress != "" && ipAddress != "-" && geoEnabled {
			if loc := geoip.GetLocationString(ipAddress); loc.Valid {
				location = loc.String
			}
		}

		list = append(list, gin.H{
			"id":                 d.ID,
			"device_name":        getString(d.DeviceName),
			"name":               getString(d.DeviceName),
			"device_fingerprint": d.DeviceFingerprint,
			"device_type":        getString(d.DeviceType),
			"type":               getString(d.DeviceType),
			"ip_address":         ipAddress,
			"ip":                 ipAddress,
			"location":           location,
			"os_name":            getString(d.OSName),
			"os_version":         getString(d.OSVersion),
			"last_access":        d.LastAccess.Format(timeLayout),
			"last_seen":          lastSeen,
			"created_at":         d.CreatedAt.Format(timeLayout),
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
	return list
}

func getSubscriptionByID(db *gorm.DB, id string, userID uint) (*models.Subscription, error) {
	var sub models.Subscription
	query := db.Where("id = ?", id)
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if err := query.Preload("User").First(&sub).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

// performSubscriptionReset 封装重置订阅的核心逻辑
func performSubscriptionReset(db *gorm.DB, sub *models.Subscription, resetType, reason string, resetBy *string) error {
	oldURL := sub.SubscriptionURL
	var deviceCountBefore int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", sub.ID, true).Count(&deviceCountBefore)

	newURL := utils.GenerateSubscriptionURL()
	sub.SubscriptionURL = newURL
	sub.CurrentDevices = 0

	if err := db.Save(sub).Error; err != nil {
		return err
	}

	reset := models.SubscriptionReset{
		UserID:             sub.UserID,
		SubscriptionID:     sub.ID,
		ResetType:          resetType,
		Reason:             reason,
		OldSubscriptionURL: &oldURL,
		NewSubscriptionURL: &newURL,
		DeviceCountBefore:  int(deviceCountBefore),
		DeviceCountAfter:   0,
		ResetBy:            resetBy,
	}
	if err := db.Create(&reset).Error; err != nil {
		return err
	}
	return db.Where("subscription_id = ?", sub.ID).Delete(&models.Device{}).Error
}

// --- Handlers ---

func GetSubscriptions(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}
	var subscriptions []models.Subscription
	if err := database.GetDB().Where("user_id = ?", user.ID).Find(&subscriptions).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅列表失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", subscriptions)
}

func GetSubscription(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}
	sub, err := getSubscriptionByID(database.GetDB(), c.Param("id"), user.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅失败", err)
		}
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", sub)
}

func CreateSubscription(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}
	db := database.GetDB()
	deviceLimit, durationMonths := getDefaultSubscriptionSettings(db)

	nowUTC := time.Now().UTC()
	var expireTime time.Time
	if durationMonths <= 0 {
		expireTime = time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 23, 59, 59, 0, nowUTC.Location())
	} else {
		expireTime = nowUTC.AddDate(0, durationMonths, 0)
	}

	sub := models.Subscription{
		UserID:          user.ID,
		SubscriptionURL: utils.GenerateSubscriptionURL(),
		DeviceLimit:     deviceLimit,
		CurrentDevices:  0,
		IsActive:        true,
		Status:          utils.SubscriptionStatusActive,
		ExpireTime:      expireTime,
	}
	if err := db.Create(&sub).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建订阅失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusCreated, "", sub)
}

func GetAdminSubscriptions(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.Subscription{})
	page, size := 1, 20
	fmt.Sscanf(c.Query("page"), "%d", &page)
	fmt.Sscanf(c.Query("size"), "%d", &size)

	if keyword := utils.SanitizeSearchKeyword(c.DefaultQuery("search", c.Query("keyword"))); keyword != "" {
		likeKey := "%" + keyword + "%"
		query = query.Where(
			"subscription_url LIKE ? OR user_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ?) OR user_id IN (SELECT DISTINCT user_id FROM subscription_resets WHERE old_subscription_url LIKE ?)",
			likeKey, likeKey, likeKey, likeKey)
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
		"add_time_desc":       "created_at DESC",
		"add_time_asc":        "created_at ASC",
		"expire_time_desc":    "expire_time DESC",
		"expire_time_asc":     "expire_time ASC",
		"device_count_desc":   "current_devices DESC",
		"device_count_asc":    "current_devices ASC",
		"device_limit_desc":   "device_limit DESC",
		"device_limit_asc":    "device_limit ASC",
		"apple_count_desc":    "universal_count DESC",
		"apple_count_asc":     "universal_count ASC",
		"online_devices_desc": "(SELECT COUNT(*) FROM devices WHERE devices.subscription_id = subscriptions.id AND devices.is_active = 1) DESC",
		"online_devices_asc":  "(SELECT COUNT(*) FROM devices WHERE devices.subscription_id = subscriptions.id AND devices.is_active = 1) ASC",
	}

	order, ok := sortMap[sort]
	if !ok {
		order = "created_at DESC"
	}
	query = query.Order(order)

	var total int64
	query.Count(&total)
	query = query.Preload("User").Preload("Package")

	var subscriptions []models.Subscription
	if err := query.Offset((page - 1) * size).Limit(size).Find(&subscriptions).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅列表失败", err)
		return
	}

	list := buildSubscriptionListData(db, subscriptions, c)
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"subscriptions": list, "total": total, "page": page, "size": size})
}

func GetUserSubscriptionDevices(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}
	db := database.GetDB()
	var sub models.Subscription
	if err := db.Where("user_id = ?", user.ID).Order("created_at DESC").First(&sub).Error; err != nil {
		utils.SuccessResponse(c, http.StatusOK, "", []gin.H{})
		return
	}
	var devices []models.Device
	db.Where("subscription_id = ?", sub.ID).Find(&devices)
	utils.SuccessResponse(c, http.StatusOK, "", formatDeviceList(devices))
}

func GetSubscriptionDevices(c *gin.Context) {
	sub, err := getSubscriptionByID(database.GetDB(), c.Param("id"), 0)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅失败", err)
		}
		return
	}
	var devices []models.Device
	database.GetDB().Where("subscription_id = ?", sub.ID).Find(&devices)
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"devices":         formatDeviceList(devices),
		"device_limit":    sub.DeviceLimit,
		"current_devices": sub.CurrentDevices,
	})
}

func BatchClearDevices(c *gin.Context) {
	var req struct {
		SubscriptionIDs []uint `json:"subscription_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}
	db := database.GetDB()
	db.Where("subscription_id IN ?", req.SubscriptionIDs).Delete(&models.Device{})
	db.Model(&models.Subscription{}).Where("id IN ?", req.SubscriptionIDs).Update("current_devices", 0)
	utils.SuccessResponse(c, http.StatusOK, "设备已清除", nil)
}

func UpdateSubscription(c *gin.Context) {
	var req struct {
		DeviceLimit *int    `json:"device_limit"`
		ExpireTime  *string `json:"expire_time"`
		IsActive    *bool   `json:"is_active"`
		Status      string  `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	db := database.GetDB()
	sub, err := getSubscriptionByID(db, c.Param("id"), 0)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅失败", err)
		}
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
		} else if t, err := time.Parse(timeLayout, *req.ExpireTime); err == nil {
			sub.ExpireTime = t
		}
	}
	if err := db.Save(sub).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "更新成功", nil)
}

func ResetSubscription(c *gin.Context) {
	db := database.GetDB()
	sub, err := getSubscriptionByID(db, c.Param("id"), 0)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅失败", err)
		}
		return
	}

	if err := performSubscriptionReset(db, sub, "admin_reset", "管理员重置订阅地址", getCurrentAdminUsername(c)); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "重置失败", err)
		return
	}

	go sendResetEmail(c, *sub, sub.User, "管理员重置")
	utils.SuccessResponse(c, http.StatusOK, "订阅已重置", sub)
}

func ExtendSubscription(c *gin.Context) {
	var req struct {
		Days int `json:"days" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "days 必填", err)
		return
	}
	db := database.GetDB()
	sub, err := getSubscriptionByID(db, c.Param("id"), 0)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅失败", err)
		}
		return
	}

	oldExp := "未设置"
	if !sub.ExpireTime.IsZero() {
		oldExp = sub.ExpireTime.Format(timeLayout)
	} else {
		sub.ExpireTime = utils.GetBeijingTime()
	}
	sub.ExpireTime = sub.ExpireTime.AddDate(0, 0, req.Days)
	db.Save(sub)

	go func() {
		pkgName := "默认套餐"
		if sub.PackageID != nil {
			var pkg models.Package
			if err := db.First(&pkg, *sub.PackageID).Error; err == nil {
				pkgName = pkg.Name
			}
		}
		email.NewEmailService().QueueEmail(sub.User.Email, "续费成功",
			email.NewEmailTemplateBuilder().GetRenewalConfirmationTemplate(sub.User.Username, pkgName, oldExp, sub.ExpireTime.Format(timeLayout), utils.GetBeijingTime().Format(timeLayout), 0), "renewal_confirmation")
	}()
	utils.SuccessResponse(c, http.StatusOK, "订阅已延长", sub)
}

func ResetUserSubscription(c *gin.Context) {
	userID := c.Param("id")
	db := database.GetDB()
	var subs []models.Subscription
	db.Where("user_id = ?", userID).Find(&subs)
	adminName := getCurrentAdminUsername(c)

	for _, sub := range subs {
		// 这里虽然在循环中，但使用 shared helper 依然比重复代码好
		// 注意：subs 循环出来的对象是指针副本，需要小心处理
		subCopy := sub
		_ = performSubscriptionReset(db, &subCopy, "admin_reset", "管理员重置用户订阅地址", adminName)
	}
	utils.SuccessResponse(c, http.StatusOK, "用户订阅已重置", nil)
}

func SendSubscriptionEmail(c *gin.Context) {
	db := database.GetDB()
	var user models.User
	var sub models.Subscription
	if err := db.First(&user, c.Param("id")).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "用户不存在", err)
		return
	}
	if err := db.Where("user_id = ?", user.ID).First(&sub).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "用户没有订阅", err)
		return
	}
	if err := queueSubEmail(c, sub, user); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "发送邮件失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "订阅邮件已加入队列", nil)
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
	utils.SuccessResponse(c, http.StatusOK, "设备已清理", nil)
}

func ResetUserSubscriptionSelf(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}
	db := database.GetDB()
	var sub models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&sub).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		return
	}

	reason := "用户主动重置订阅地址"
	if err := performSubscriptionReset(db, &sub, "user_reset", reason, &user.Username); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "重置失败", err)
		return
	}

	go sendResetEmail(c, sub, *user, reason)
	utils.SuccessResponse(c, http.StatusOK, "订阅已重置", sub)
}

func SendSubscriptionEmailSelf(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}
	var sub models.Subscription
	if err := database.GetDB().Where("user_id = ?", user.ID).First(&sub).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "您还没有订阅", err)
		return
	}
	go notification.NewNotificationService().SendAdminNotification("subscription_sent", map[string]interface{}{"username": user.Username, "email": user.Email, "send_time": utils.GetBeijingTime().Format(timeLayout)})
	if err := queueSubEmail(c, sub, *user); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "发送邮件失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "订阅邮件已加入队列", nil)
}

func ConvertSubscriptionToBalance(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}
	db := database.GetDB()
	var sub models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&sub).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		return
	}
	now := utils.GetBeijingTime()
	if !sub.ExpireTime.After(now) {
		utils.ErrorResponse(c, http.StatusBadRequest, "订阅已过期", nil)
		return
	}

	diff := sub.ExpireTime.Sub(now)
	days := int(diff.Hours() / 24)
	if diff.Hours() > float64(days*24) {
		days++
	}

	var originalPkgPrice float64 = 0
	var originalPkgDays int = 0

	// 1. 尝试从订阅的 PackageID 获取
	if sub.PackageID != nil {
		var pkg models.Package
		if err := db.First(&pkg, *sub.PackageID).Error; err == nil {
			originalPkgPrice = pkg.Price
			originalPkgDays = pkg.DurationDays
		}
	}

	// 2. 尝试从订单记录获取
	if originalPkgDays <= 0 {
		totalDuration := int(sub.ExpireTime.Sub(sub.CreatedAt).Hours() / 24)
		if totalDuration <= 0 {
			totalDuration = 30
		}
		var recentOrder models.Order
		if err := db.Where("user_id = ? AND status = ?", user.ID, "paid").Order("created_at DESC").First(&recentOrder).Error; err == nil {
			var pkg models.Package
			if err := db.First(&pkg, recentOrder.PackageID).Error; err == nil {
				originalPkgPrice = recentOrder.Amount
				originalPkgDays = pkg.DurationDays
			}
		}
		// 3. 查找相似时长套餐
		if originalPkgDays <= 0 {
			var similarPkg models.Package
			if err := db.Where("duration_days BETWEEN ? AND ? AND is_active = ?", totalDuration-5, totalDuration+5, true).
				Order(fmt.Sprintf("ABS(duration_days - %d) ASC", totalDuration)).
				First(&similarPkg).Error; err == nil {
				originalPkgPrice = similarPkg.Price
				originalPkgDays = similarPkg.DurationDays
			}
		}
		// 4. 估算
		if originalPkgDays <= 0 {
			originalPkgDays = totalDuration
			if originalPkgDays <= 0 {
				originalPkgDays = 30
			}
			originalPkgPrice = float64(originalPkgDays) * 1.0
		}
	}

	dailyPrice := originalPkgPrice / float64(originalPkgDays)
	convertedAmount := float64(days) * dailyPrice
	convertedAmount = float64(int(convertedAmount*100+0.5)) / 100

	user.Balance += convertedAmount
	if err := db.Save(&user).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新余额失败", err)
		return
	}

	if err := db.Delete(&sub).Error; err != nil {
		utils.LogError("ConvertSubscriptionToBalance: failed to delete subscription", err, map[string]interface{}{"user_id": user.ID, "sub_id": sub.ID})
	}

	utils.SuccessResponse(c, http.StatusOK, "已转换为余额", gin.H{
		"converted_amount":       convertedAmount,
		"balance_added":          convertedAmount,
		"new_balance":            user.Balance,
		"remaining_days":         days,
		"daily_price":            dailyPrice,
		"original_package_price": originalPkgPrice,
		"original_package_days":  originalPkgDays,
	})
}

func ExportSubscriptions(c *gin.Context) {
	var subs []models.Subscription
	if err := database.GetDB().Preload("User").Find(&subs).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取列表失败", err)
		return
	}
	var csv strings.Builder
	csv.WriteString("\xEF\xBB\xBFID,用户ID,用户名,邮箱,订阅地址,状态,是否激活,设备限制,当前设备,到期时间,创建时间\n")
	for _, s := range subs {
		active := "是"
		if !s.IsActive {
			active = "否"
		}
		csv.WriteString(fmt.Sprintf("%d,%d,%s,%s,%s,%s,%s,%d,%d,%s,%s\n",
			s.ID, s.UserID, s.User.Username, s.User.Email, s.SubscriptionURL, s.Status, active,
			s.DeviceLimit, s.CurrentDevices, s.ExpireTime.Format(timeLayout), s.CreatedAt.Format(timeLayout)))
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=subscriptions_%s.csv", time.Now().Format("20060102")))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", []byte(csv.String()))
}

func sendResetEmail(c *gin.Context, sub models.Subscription, user models.User, reason string) {
	univ, clash := getSubscriptionURLs(c, sub.SubscriptionURL)
	exp := "未设置"
	if !sub.ExpireTime.IsZero() {
		exp = sub.ExpireTime.Format(timeLayout)
	}
	resetTime := utils.GetBeijingTime().Format(timeLayout)
	content := email.NewEmailTemplateBuilder().GetSubscriptionResetTemplate(user.Username, univ, clash, exp, resetTime, reason)
	_ = email.NewEmailService().QueueEmail(user.Email, "订阅重置通知", content, "subscription_reset")
	_ = notification.NewNotificationService().SendAdminNotification("subscription_reset", map[string]interface{}{"username": user.Username, "email": user.Email, "reset_time": resetTime})
}

func queueSubEmail(c *gin.Context, sub models.Subscription, user models.User) error {
	univ, clash := getSubscriptionURLs(c, sub.SubscriptionURL)
	exp, days := "未设置", 0
	if !sub.ExpireTime.IsZero() {
		exp = sub.ExpireTime.Format(timeLayout)
		if diff := sub.ExpireTime.Sub(utils.GetBeijingTime()); diff > 0 {
			days = int(diff.Hours() / 24)
		}
	}
	content := email.NewEmailTemplateBuilder().GetSubscriptionTemplate(user.Username, univ, clash, exp, days, sub.DeviceLimit, sub.CurrentDevices)
	return email.NewEmailService().QueueEmail(user.Email, "服务配置信息", content, "subscription")
}

func BatchDeleteSubscriptions(c *gin.Context) {
	var req struct {
		SubscriptionIDs []uint `json:"subscription_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}
	if len(req.SubscriptionIDs) == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "请选择要删除的订阅", nil)
		return
	}

	db := database.GetDB()
	tx := db.Begin()
	if err := tx.Where("subscription_id IN ?", req.SubscriptionIDs).Delete(&models.Device{}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除订阅设备失败", err)
		return
	}
	if err := tx.Where("subscription_id IN ?", req.SubscriptionIDs).Delete(&models.SubscriptionReset{}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除订阅重置记录失败", err)
		return
	}
	if err := tx.Where("id IN ?", req.SubscriptionIDs).Delete(&models.Subscription{}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除订阅失败", err)
		return
	}
	if err := tx.Commit().Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除操作失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功删除 %d 个订阅", len(req.SubscriptionIDs)), nil)
}

func BatchEnableSubscriptions(c *gin.Context) {
	batchUpdateSubscriptionStatus(c, true, "active")
}

func BatchDisableSubscriptions(c *gin.Context) {
	batchUpdateSubscriptionStatus(c, false, "inactive")
}

func batchUpdateSubscriptionStatus(c *gin.Context, isActive bool, status string) {
	var req struct {
		SubscriptionIDs []uint `json:"subscription_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}
	if len(req.SubscriptionIDs) == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "请选择订阅", nil)
		return
	}
	res := database.GetDB().Model(&models.Subscription{}).Where("id IN ?", req.SubscriptionIDs).Updates(map[string]interface{}{
		"is_active": isActive,
		"status":    status,
	})
	if res.Error != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "操作失败", res.Error)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功操作 %d 个订阅", res.RowsAffected), nil)
}

func BatchResetSubscriptions(c *gin.Context) {
	var req struct {
		SubscriptionIDs []uint `json:"subscription_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}
	if len(req.SubscriptionIDs) == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "请选择要重置的订阅", nil)
		return
	}

	db := database.GetDB()
	var subscriptions []models.Subscription
	if err := db.Where("id IN ?", req.SubscriptionIDs).Preload("User").Find(&subscriptions).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅信息失败", err)
		return
	}

	successCount, failCount := 0, 0
	adminUsername := getCurrentAdminUsername(c)

	for _, sub := range subscriptions {
		subCopy := sub
		if err := performSubscriptionReset(db, &subCopy, "admin_batch_reset", "管理员批量重置订阅地址", adminUsername); err != nil {
			failCount++
			continue
		}
		go sendResetEmail(c, subCopy, subCopy.User, "管理员批量重置")
		successCount++
	}
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功重置 %d 个订阅，失败 %d 个", successCount, failCount), gin.H{
		"success_count": successCount,
		"fail_count":    failCount,
	})
}

func BatchSendAdminSubEmail(c *gin.Context) {
	var req struct {
		SubscriptionIDs []uint `json:"subscription_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}
	if len(req.SubscriptionIDs) == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "请选择订阅", nil)
		return
	}

	var subscriptions []models.Subscription
	if err := database.GetDB().Where("id IN ?", req.SubscriptionIDs).Preload("User").Find(&subscriptions).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅信息失败", err)
		return
	}

	successCount, failCount := 0, 0
	for _, sub := range subscriptions {
		if err := queueSubEmail(c, sub, sub.User); err != nil {
			failCount++
			continue
		}
		successCount++
	}
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功发送 %d 封邮件，失败 %d 封", successCount, failCount), gin.H{
		"success_count": successCount,
		"fail_count":    failCount,
	})
}

func GetExpiringSubscriptions(c *gin.Context) {
	db := database.GetDB()
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days <= 0 {
		days = 7
	}
	filter := c.Query("filter")
	now := utils.GetBeijingTime()
	endDate := now.AddDate(0, 0, days)

	query := db.Where("expire_time IS NOT NULL AND expire_time > ? AND expire_time <= ?", now, endDate).
		Where("is_active = ?", true).Preload("User").Order("expire_time ASC")

	if filter != "" && filter != "all" {
		switch filter {
		case "today":
			todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			query = query.Where("expire_time >= ? AND expire_time < ?", todayStart, todayStart.AddDate(0, 0, 1))
		case "1-3":
			query = query.Where("expire_time > ? AND expire_time <= ?", now, now.AddDate(0, 0, 3))
		case "4-7":
			query = query.Where("expire_time > ? AND expire_time <= ?", now.AddDate(0, 0, 3), now.AddDate(0, 0, 7))
		}
	}

	var subscriptions []models.Subscription
	if err := query.Find(&subscriptions).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询失败", err)
		return
	}

	result := make([]gin.H, 0, len(subscriptions))
	for _, sub := range subscriptions {
		daysUntilExpire := 0
		if !sub.ExpireTime.IsZero() {
			if diff := sub.ExpireTime.Sub(now); diff > 0 {
				daysUntilExpire = int(diff.Hours() / 24)
			}
		}
		userInfo := gin.H{"id": 0, "username": "用户已删除", "email": "", "qq": ""}
		if sub.User.ID > 0 {
			userInfo["id"] = sub.User.ID
			userInfo["username"] = sub.User.Username
			userInfo["email"] = sub.User.Email
		}

		result = append(result, gin.H{
			"id":                sub.ID,
			"user_id":           sub.UserID,
			"username":          userInfo["username"],
			"email":             userInfo["email"],
			"qq":                userInfo["qq"],
			"expire_time":       sub.ExpireTime.Format(timeLayout),
			"days_until_expire": daysUntilExpire,
		})
	}
	utils.SuccessResponse(c, http.StatusOK, "", result)
}

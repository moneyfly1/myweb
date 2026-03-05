package handlers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"
	"cboard-go/internal/services/device"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	TimeLayout = "2006-01-02 15:04:05"
	DateFormat = "2006-01-02"
)

// ==========================================
// 辅助函数 (Helpers)
// ==========================================

// fetchSubscription 统一获取订阅逻辑，自动处理 404/500 响应（用户端，强制校验 userID）
func fetchSubscription(c *gin.Context, db *gorm.DB, subID string, userID uint) (*models.Subscription, bool) {
	sub, err := getSubscriptionByID(db, subID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅失败", err)
		}
		return nil, false
	}
	return sub, true
}

// fetchSubscriptionAdmin 管理端获取订阅，不限制 userID
func fetchSubscriptionAdmin(c *gin.Context, db *gorm.DB, subID string) (*models.Subscription, bool) {
	sub, err := getSubscriptionByIDAdmin(db, subID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅失败", err)
		}
		return nil, false
	}
	return sub, true
}

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

// asyncSubscriptionLog 异步记录订阅日志，安全处理 Context
func asyncSubscriptionLog(
	subID, userID uint,
	actionType, actionBy string,
	actionByUserID *uint,
	clientIP string, // 必须在 handler 中获取，不能在 goroutine 中从 ctx 获取
	beforeData, afterData map[string]interface{},
	reason string,
) {
	go func() {
		utils.CreateSubscriptionLog(subID, userID, actionType, actionBy, actionByUserID, clientIP, beforeData, afterData, reason)
	}()
}

// ==========================================
// 核心逻辑函数
// ==========================================

func formatDeviceList(devices []models.Device) []gin.H {
	list := make([]gin.H, 0, len(devices))
	// 列表查询不查询 GeoIP，提升性能
	// geoEnabled := geoip.IsEnabled()

	for _, d := range devices {
		lastSeen := d.LastAccess.Format(TimeLayout)
		if d.LastSeen != nil {
			lastSeen = d.LastSeen.Format(TimeLayout)
		}
		ipAddress := formatIP(getString(d.IPAddress))
		location := ""
		// if ipAddress != "" && ipAddress != "-" && geoEnabled {
		// 	if loc := geoip.GetLocationString(ipAddress); loc.Valid {
		// 		location = loc.String
		// 	}
		// }

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
			"last_access":        d.LastAccess.Format(TimeLayout),
			"last_seen":          lastSeen,
			"created_at":         d.CreatedAt.Format(TimeLayout),
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
	// userID 必须参与过滤，防止 IDOR
	if err := db.Where("id = ? AND user_id = ?", id, userID).Preload("User").First(&sub).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

// getSubscriptionByIDAdmin 管理端专用，不限制 userID
func getSubscriptionByIDAdmin(db *gorm.DB, id string) (*models.Subscription, error) {
	var sub models.Subscription
	if err := db.Where("id = ?", id).Preload("User").First(&sub).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

func performSubscriptionReset(db *gorm.DB, sub *models.Subscription, resetType, reason string, resetBy *string, resetByUserID *uint, ipAddress string) error {
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

	// 记录日志
	beforeData := map[string]interface{}{
		"subscription_url": oldURL,
		"device_count":     deviceCountBefore,
	}
	afterData := map[string]interface{}{
		"subscription_url": newURL,
		"device_count":     0,
	}
	actionBy := "user"
	var actionByUserID *uint
	if resetBy != nil && resetByUserID != nil {
		// 管理员重置
		actionBy = "admin"
		actionByUserID = resetByUserID
	} else if resetBy != nil {
		// 用户自己重置（通过用户名）
		actionBy = "user"
		actionByUserID = &sub.UserID
	} else {
		// 系统重置
		actionBy = "system"
	}

	// 使用传入的 IP 地址记录日志
	asyncSubscriptionLog(sub.ID, sub.UserID, "reset", actionBy, actionByUserID, ipAddress, beforeData, afterData, reason)

	return db.Where("subscription_id = ?", sub.ID).Delete(&models.Device{}).Error
}

// calculateSubscriptionValue 计算订阅剩余价值
func calculateSubscriptionValue(db *gorm.DB, sub models.Subscription, userID uint) (float64, int, float64, int, float64) {
	now := utils.GetBeijingTime()
	diff := sub.ExpireTime.Sub(now)
	days := int(math.Ceil(diff.Hours() / 24))
	if days < 0 {
		days = 0
	}

	var originalPkgPrice float64 = 0
	var originalPkgDays int = 0

	// 1. 尝试从关联套餐获取
	if sub.PackageID != nil {
		var pkg models.Package
		if err := db.First(&pkg, *sub.PackageID).Error; err == nil {
			originalPkgPrice = pkg.Price
			originalPkgDays = pkg.DurationDays
		}
	}

	// 2. 尝试从最近订单或推断
	if originalPkgDays <= 0 {
		totalDuration := int(sub.ExpireTime.Sub(sub.CreatedAt).Hours() / 24)
		if totalDuration <= 0 {
			totalDuration = 30
		}

		// 尝试查找最近的支付订单
		var recentOrder models.Order
		if err := db.Where("user_id = ? AND status = ?", userID, "paid").Order("created_at DESC").First(&recentOrder).Error; err == nil {
			var pkg models.Package
			if err := db.First(&pkg, recentOrder.PackageID).Error; err == nil {
				originalPkgPrice = recentOrder.Amount
				originalPkgDays = pkg.DurationDays
			}
		}

		// 尝试匹配相似时长的套餐
		if originalPkgDays <= 0 {
			var similarPkg models.Package
			if err := db.Where("duration_days BETWEEN ? AND ? AND is_active = ?", totalDuration-5, totalDuration+5, true).
				Order(fmt.Sprintf("ABS(duration_days - %d) ASC", totalDuration)).
				First(&similarPkg).Error; err == nil {
				originalPkgPrice = similarPkg.Price
				originalPkgDays = similarPkg.DurationDays
			}
		}

		// 兜底策略
		if originalPkgDays <= 0 {
			originalPkgDays = totalDuration
			if originalPkgDays <= 0 {
				originalPkgDays = 30
			}
			originalPkgPrice = float64(originalPkgDays) * 1.0 // 默认一天一元
		}
	}

	dailyPrice := originalPkgPrice / float64(originalPkgDays)
	convertedAmount := float64(days) * dailyPrice
	convertedAmount = math.Round(convertedAmount*100) / 100 // 保留两位小数

	return convertedAmount, days, dailyPrice, originalPkgDays, originalPkgPrice
}

// ==========================================
// Handlers
// ==========================================

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
	sub, ok := fetchSubscription(c, database.GetDB(), c.Param("id"), user.ID)
	if !ok {
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

	now := utils.GetBeijingTime()
	var expireTime time.Time
	if durationMonths <= 0 {
		expireTime = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	} else {
		expireTime = now.AddDate(0, durationMonths, 0)
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

	// 记录日志
	afterData := map[string]interface{}{
		"subscription_id": sub.ID,
		"device_limit":    sub.DeviceLimit,
		"expire_time":     sub.ExpireTime.Format(TimeLayout),
		"status":          sub.Status,
	}
	asyncSubscriptionLog(sub.ID, user.ID, "create", "user", &user.ID, utils.GetRealClientIP(c), nil, afterData, "用户创建订阅")

	utils.SuccessResponse(c, http.StatusCreated, "", sub)
}

func GetAdminSubscriptions(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.Subscription{})
	p := utils.ParsePagination(c)
	page, size := p.Page, p.Size

	if keyword := utils.SanitizeSearchKeyword(c.DefaultQuery("search", c.Query("keyword"))); keyword != "" {
		likeKey := "%" + keyword + "%"
		query = query.Where(
			"subscription_url LIKE ? OR user_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ? OR notes LIKE ?) OR user_id IN (SELECT DISTINCT user_id FROM subscription_resets WHERE old_subscription_url LIKE ?)",
			likeKey, likeKey, likeKey, likeKey, likeKey)
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
		"clash_count_desc":    "clash_count DESC",
		"clash_count_asc":     "clash_count ASC",
		"online_devices_desc": "(SELECT COUNT(*) FROM devices WHERE devices.subscription_id = subscriptions.id AND devices.is_active = 1) DESC",
		"online_devices_asc":  "(SELECT COUNT(*) FROM devices WHERE devices.subscription_id = subscriptions.id AND devices.is_active = 1) ASC",
	}

	if order, ok := sortMap[sort]; ok {
		query = query.Order(order)
	} else {
		query = query.Order("created_at DESC")
	}

	var total int64
	query.Count(&total)

	var subscriptions []models.Subscription
	if err := query.Preload("User").Preload("Package").Offset((page - 1) * size).Limit(size).Find(&subscriptions).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅列表失败", err)
		return
	}

	list := buildSubscriptionListData(db, subscriptions, c)
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"subscriptions": list, "total": total, "page": page, "size": size})
}

func buildSubscriptionListData(db *gorm.DB, subscriptions []models.Subscription, c *gin.Context) []gin.H {
	if len(subscriptions) == 0 {
		return []gin.H{}
	}

	subIDs := make([]uint, len(subscriptions))
	userIDs := make([]uint, 0, len(subscriptions))
	userIDSet := make(map[uint]bool)
	for i, s := range subscriptions {
		subIDs[i] = s.ID
		if !userIDSet[s.UserID] {
			userIDs = append(userIDs, s.UserID)
			userIDSet[s.UserID] = true
		}
	}

	var users []models.User
	userMap := make(map[uint]*models.User)
	if len(userIDs) > 0 {
		db.Where("id IN ?", userIDs).Find(&users)
		for i := range users {
			userMap[users[i].ID] = &users[i]
		}
	}

	// 优化：一次性查询统计信息
	type Stat struct {
		SubID uint
		Type  *string
		Count int64
	}
	var onlineStats, typeStats []Stat

	db.Model(&models.Device{}).Select("subscription_id as sub_id, count(*) as count").
		Where("subscription_id IN ? AND is_active = ?", subIDs, true).
		Group("subscription_id").Scan(&onlineStats)

	db.Model(&models.Device{}).Select("subscription_id as sub_id, subscription_type as type, count(*) as count").
		Where("subscription_id IN ?", subIDs).
		Group("subscription_id, subscription_type").Scan(&typeStats)

	onlineMap := make(map[uint]int64)
	appleMap := make(map[uint]int64)
	clashMap := make(map[uint]int64)

	for _, s := range onlineStats {
		onlineMap[s.SubID] = s.Count
	}
	for _, s := range typeStats {
		if s.Type == nil {
			continue
		}
		if *s.Type == "v2ray" || *s.Type == "ssr" {
			appleMap[s.SubID] += s.Count
		} else if *s.Type == "clash" {
			clashMap[s.SubID] += s.Count
		}
	}

	now := utils.GetBeijingTime()
	list := make([]gin.H, 0, len(subscriptions))

	for _, sub := range subscriptions {
		online := onlineMap[sub.ID]
		curr := sub.CurrentDevices
		if curr < int(online) {
			curr = int(online)
		}

		universal, clash := getSubscriptionURLs(c, sub.SubscriptionURL)

		var userInfo gin.H
		if sub.User.ID > 0 {
			userInfo = gin.H{"id": sub.User.ID, "username": sub.User.Username, "email": sub.User.Email}
		} else if user, ok := userMap[sub.UserID]; ok {
			userInfo = gin.H{"id": user.ID, "username": user.Username, "email": user.Email}
		} else {
			userInfo = gin.H{
				"id":       0,
				"username": fmt.Sprintf("用户已删除 (ID: %d)", sub.UserID),
				"email":    fmt.Sprintf("deleted_user_%d", sub.UserID),
				"deleted":  true,
			}
		}

		daysUntil, isExpired := 0, false
		if !sub.ExpireTime.IsZero() {
			if diff := sub.ExpireTime.Sub(now); diff > 0 {
				daysUntil = int(diff.Hours() / 24)
			} else {
				isExpired = true
			}
		}

		universalCount := sub.UniversalCount
		if universalCount == 0 && appleMap[sub.ID] > 0 {
			universalCount = int(appleMap[sub.ID])
		}
		clashCount := sub.ClashCount
		if clashCount == 0 && clashMap[sub.ID] > 0 {
			clashCount = int(clashMap[sub.ID])
		}

		list = append(list, gin.H{
			"id":                sub.ID,
			"user_id":           sub.UserID,
			"user":              userInfo,
			"username":          userInfo["username"],
			"email":             userInfo["email"],
			"subscription_url":  sub.SubscriptionURL,
			"universal_url":     universal,
			"clash_url":         clash,
			"status":            sub.Status,
			"is_active":         sub.IsActive,
			"device_limit":      sub.DeviceLimit,
			"current_devices":   curr,
			"online_devices":    online,
			"apple_count":       universalCount,
			"clash_count":       clashCount,
			"expire_time":       sub.ExpireTime.Format(TimeLayout),
			"days_until_expire": daysUntil,
			"is_expired":        isExpired,
			"created_at":        sub.CreatedAt.Format(TimeLayout),
		})
	}

	return list
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
	sub, ok := fetchSubscriptionAdmin(c, database.GetDB(), c.Param("id"))
	if !ok {
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
	if len(req.SubscriptionIDs) == 0 {
		utils.SuccessResponse(c, http.StatusOK, "未选择订阅", nil)
		return
	}
	db := database.GetDB()
	db.Where("subscription_id IN ?", req.SubscriptionIDs).Delete(&models.Device{})
	db.Model(&models.Subscription{}).Where("id IN ?", req.SubscriptionIDs).Update("current_devices", 0)
	utils.CreateAuditLogSimple(c, "batch_clear_devices", "subscription", 0, fmt.Sprintf("管理员操作: 批量清除订阅设备 %d 个", len(req.SubscriptionIDs)))
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
	sub, ok := fetchSubscriptionAdmin(c, db, c.Param("id"))
	if !ok {
		return
	}

	// 记录变更前数据
	beforeData := map[string]interface{}{
		"device_limit": sub.DeviceLimit,
		"is_active":    sub.IsActive,
		"status":       sub.Status,
		"expire_time":  sub.ExpireTime.Format(TimeLayout),
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
		if t, err := time.Parse(DateFormat, *req.ExpireTime); err == nil {
			sub.ExpireTime = t
		} else if t, err := time.Parse(TimeLayout, *req.ExpireTime); err == nil {
			sub.ExpireTime = t
		}
	}
	if err := db.Save(sub).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新失败", err)
		return
	}

	// 准备日志数据
	actionType := "update"
	if req.IsActive != nil {
		if *req.IsActive {
			actionType = "activate"
		} else {
			actionType = "deactivate"
		}
	}
	actionBy := "user"
	var actionByUserID *uint
	currentUser, ok := middleware.GetCurrentUser(c)
	if ok && currentUser != nil {
		if currentUser.IsAdmin {
			// 管理员更新
			actionBy = "admin"
			actionByUserID = &currentUser.ID
		} else {
			// 用户自己更新
			actionBy = "user"
			actionByUserID = &currentUser.ID
		}
	}
	afterData := map[string]interface{}{
		"device_limit": sub.DeviceLimit,
		"is_active":    sub.IsActive,
		"status":       sub.Status,
		"expire_time":  sub.ExpireTime.Format(TimeLayout),
	}

	asyncSubscriptionLog(sub.ID, sub.UserID, actionType, actionBy, actionByUserID, utils.GetRealClientIP(c), beforeData, afterData, "更新订阅")
	if actionBy == "admin" {
		utils.CreateAuditLogSimple(c, "update_subscription", "subscription", sub.ID, fmt.Sprintf("管理员操作: 更新订阅 subscription_id=%d", sub.ID))
	}
	utils.SuccessResponse(c, http.StatusOK, "更新成功", nil)
}

func ResetSubscription(c *gin.Context) {
	db := database.GetDB()
	sub, ok := fetchSubscriptionAdmin(c, db, c.Param("id"))
	if !ok {
		return
	}

	ipAddress := utils.GetRealClientIP(c)
	adminUser, _ := middleware.GetCurrentUser(c)
	var adminUserID *uint
	if adminUser != nil {
		adminUserID = &adminUser.ID
	}
	if err := performSubscriptionReset(db, sub, "admin_reset", "管理员重置订阅地址", getCurrentAdminUsername(c), adminUserID, ipAddress); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "重置失败", err)
		return
	}
	utils.CreateAuditLogSimple(c, "reset_subscription", "subscription", sub.ID, fmt.Sprintf("管理员操作: 重置订阅 subscription_id=%d", sub.ID))
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
	sub, ok := fetchSubscriptionAdmin(c, db, c.Param("id"))
	if !ok {
		return
	}

	oldExp := "未设置"
	if !sub.ExpireTime.IsZero() {
		oldExp = sub.ExpireTime.Format(TimeLayout)
	} else {
		sub.ExpireTime = utils.GetBeijingTime()
	}
	sub.ExpireTime = sub.ExpireTime.AddDate(0, 0, req.Days)
	db.Save(sub)
	utils.CreateAuditLogSimple(c, "extend_subscription", "subscription", sub.ID, fmt.Sprintf("管理员操作: 延长订阅 %d 天 subscription_id=%d", req.Days, sub.ID))
	// 异步发送通知
	go func() {
		pkgName := "默认套餐"
		if sub.PackageID != nil {
			var pkg models.Package
			if err := db.First(&pkg, *sub.PackageID).Error; err == nil {
				pkgName = pkg.Name
			}
		}
		email.NewEmailService().QueueEmail(sub.User.Email, "续费成功",
			email.NewEmailTemplateBuilder().GetRenewalConfirmationTemplate(sub.User.Username, pkgName, oldExp, sub.ExpireTime.Format(TimeLayout), utils.GetBeijingTime().Format(TimeLayout), 0), "renewal_confirmation")
	}()
	utils.SuccessResponse(c, http.StatusOK, "订阅已延长", sub)
}

func ResetUserSubscription(c *gin.Context) {
	userID := c.Param("id")
	db := database.GetDB()
	var subs []models.Subscription
	db.Where("user_id = ?", userID).Find(&subs)
	adminName := getCurrentAdminUsername(c)
	adminUser, _ := middleware.GetCurrentUser(c)
	var adminUserID *uint
	if adminUser != nil {
		adminUserID = &adminUser.ID
	}

	ipAddress := utils.GetRealClientIP(c)
	for _, sub := range subs {
		subCopy := sub
		_ = performSubscriptionReset(db, &subCopy, "admin_reset", "管理员重置用户订阅地址", adminName, adminUserID, ipAddress)
	}
	utils.CreateAuditLogSimple(c, "reset_user_subscription", "user", 0, fmt.Sprintf("管理员操作: 重置用户订阅 user_id=%s", userID))
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
	utils.CreateAuditLogSimple(c, "send_subscription_email", "subscription", sub.ID, fmt.Sprintf("管理员操作: 发送订阅邮件 user_id=%d", user.ID))
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
	utils.CreateAuditLogSimple(c, "clear_user_devices", "user", 0, fmt.Sprintf("管理员操作: 清理用户设备 user_id=%s", userID))
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
	ipAddress := utils.GetRealClientIP(c)
	userID := user.ID
	if err := performSubscriptionReset(db, &sub, "user_reset", reason, &user.Username, &userID, ipAddress); err != nil {
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
	go notification.NewNotificationService().SendAdminNotification("subscription_sent", map[string]interface{}{"username": user.Username, "email": user.Email, "send_time": utils.GetBeijingTime().Format(TimeLayout)})
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

	if !sub.ExpireTime.After(utils.GetBeijingTime()) {
		utils.ErrorResponse(c, http.StatusBadRequest, "订阅已过期", nil)
		return
	}

	convertedAmount, days, dailyPrice, originalPkgDays, originalPkgPrice := calculateSubscriptionValue(db, sub, user.ID)

	ipAddress := utils.GetRealClientIP(c)
	var oldBalance, newBalance float64

	txErr := db.Transaction(func(tx *gorm.DB) error {
		// 行锁：防止并发双花
		var lockedUser models.User
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&lockedUser, user.ID).Error; err != nil {
			return fmt.Errorf("锁定用户失败: %v", err)
		}
		oldBalance = lockedUser.Balance
		newBalance = lockedUser.Balance + convertedAmount

		if err := tx.Model(&models.User{}).Where("id = ?", user.ID).Update("balance", newBalance).Error; err != nil {
			return fmt.Errorf("更新余额失败: %v", err)
		}
		if err := tx.Delete(&sub).Error; err != nil {
			return fmt.Errorf("删除订阅失败: %v", err)
		}
		return nil
	})
	if txErr != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, txErr.Error(), txErr)
		return
	}

	// 记录余额日志
	userID := user.ID
	go func() {
		utils.CreateBalanceLog(
			user.ID,
			"refund",
			convertedAmount,
			oldBalance,
			newBalance,
			nil,
			nil,
			fmt.Sprintf("订阅转换为余额，订阅ID: %d", sub.ID),
			"user",
			&userID,
			ipAddress,
		)
	}()

	// 记录订阅日志
	beforeData := map[string]interface{}{
		"subscription_id": sub.ID,
		"expire_time":     sub.ExpireTime.Format(TimeLayout),
	}
	asyncSubscriptionLog(sub.ID, user.ID, "delete", "user", &user.ID, ipAddress, beforeData, nil, "订阅转换为余额")

	utils.SuccessResponse(c, http.StatusOK, "已转换为余额", gin.H{
		"converted_amount":       convertedAmount,
		"balance_added":          convertedAmount,
		"new_balance":            newBalance,
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
			s.DeviceLimit, s.CurrentDevices, s.ExpireTime.Format(TimeLayout), s.CreatedAt.Format(TimeLayout)))
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=subscriptions_%s.csv", utils.GetBeijingTime().Format("20060102")))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", []byte(csv.String()))
}

func sendResetEmail(c *gin.Context, sub models.Subscription, user models.User, reason string) {
	univ, clash := getSubscriptionURLs(c, sub.SubscriptionURL)
	exp := "未设置"
	if !sub.ExpireTime.IsZero() {
		exp = sub.ExpireTime.Format(TimeLayout)
	}
	resetTime := utils.GetBeijingTime().Format(TimeLayout)
	content := email.NewEmailTemplateBuilder().GetSubscriptionResetTemplate(user.Username, univ, clash, exp, resetTime, reason)
	_ = email.NewEmailService().QueueEmail(user.Email, "订阅重置通知", content, "subscription_reset")
	_ = notification.NewNotificationService().SendAdminNotification("subscription_reset", map[string]interface{}{"username": user.Username, "email": user.Email, "reset_time": resetTime})
}

func queueSubEmail(c *gin.Context, sub models.Subscription, user models.User) error {
	univ, clash := getSubscriptionURLs(c, sub.SubscriptionURL)
	exp, days := "未设置", 0
	if !sub.ExpireTime.IsZero() {
		exp = sub.ExpireTime.Format(TimeLayout)
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

	// 获取要删除的订阅信息（用于日志）
	var subsToDelete []models.Subscription
	db.Where("id IN ?", req.SubscriptionIDs).Find(&subsToDelete)

	tx := db.Begin()
	// 使用 Where("subscription_id IN ?") 优化批量删除逻辑
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

	// 记录日志
	ipAddress := utils.GetRealClientIP(c)
	adminUser, _ := middleware.GetCurrentUser(c)
	var actionByUserID *uint
	actionBy := "admin"
	if adminUser != nil {
		actionByUserID = &adminUser.ID
	}

	for _, sub := range subsToDelete {
		beforeData := map[string]interface{}{
			"subscription_id": sub.ID,
			"user_id":         sub.UserID,
			"expire_time":     sub.ExpireTime.Format(TimeLayout),
		}
		asyncSubscriptionLog(sub.ID, sub.UserID, "delete", actionBy, actionByUserID, ipAddress, beforeData, nil, "批量删除订阅")
	}
	utils.CreateAuditLogSimple(c, "batch_delete_subscriptions", "subscription", 0, fmt.Sprintf("管理员操作: 批量删除订阅 %d 个", len(req.SubscriptionIDs)))
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

	db := database.GetDB()
	var subsToUpdate []models.Subscription
	db.Where("id IN ?", req.SubscriptionIDs).Find(&subsToUpdate)

	res := db.Model(&models.Subscription{}).Where("id IN ?", req.SubscriptionIDs).Updates(map[string]interface{}{
		"is_active": isActive,
		"status":    status,
	})
	if res.Error != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "操作失败", res.Error)
		return
	}

	// 记录日志
	ipAddress := utils.GetRealClientIP(c)
	adminUser, _ := middleware.GetCurrentUser(c)
	var actionByUserID *uint
	actionBy := "admin"
	if adminUser != nil {
		actionByUserID = &adminUser.ID
	}
	actionType := "activate"
	if !isActive {
		actionType = "deactivate"
	}

	actionName := "激活"
	if !isActive {
		actionName = "停用"
	}

	for _, sub := range subsToUpdate {
		beforeData := map[string]interface{}{
			"is_active": sub.IsActive,
			"status":    sub.Status,
		}
		afterData := map[string]interface{}{
			"is_active": isActive,
			"status":    status,
		}
		asyncSubscriptionLog(sub.ID, sub.UserID, actionType, actionBy, actionByUserID, ipAddress, beforeData, afterData, fmt.Sprintf("批量%s订阅", actionName))
	}
	utils.CreateAuditLogSimple(c, "batch_update_subscriptions_status", "subscription", 0, fmt.Sprintf("管理员操作: 批量%s订阅 %d 个", actionName, res.RowsAffected))
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

	ipAddress := utils.GetRealClientIP(c)
	adminUser, _ := middleware.GetCurrentUser(c)
	var adminUserID *uint
	if adminUser != nil {
		adminUserID = &adminUser.ID
	}
	for _, sub := range subscriptions {
		subCopy := sub
		if err := performSubscriptionReset(db, &subCopy, "admin_batch_reset", "管理员批量重置订阅地址", adminUsername, adminUserID, ipAddress); err != nil {
			failCount++
			continue
		}
		go sendResetEmail(c, subCopy, subCopy.User, "管理员批量重置")
		successCount++
	}
	utils.CreateAuditLogSimple(c, "batch_reset_subscriptions", "subscription", 0, fmt.Sprintf("管理员操作: 批量重置订阅 成功 %d 失败 %d", successCount, failCount))
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
	utils.CreateAuditLogSimple(c, "batch_send_subscription_email", "subscription", 0, fmt.Sprintf("管理员操作: 批量发送订阅邮件 成功 %d 失败 %d", successCount, failCount))
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
		userInfo := gin.H{"id": 0, "username": "用户已删除", "email": "", "last_login": ""}
		if sub.User.ID > 0 {
			userInfo["id"] = sub.User.ID
			userInfo["username"] = sub.User.Username
			userInfo["email"] = sub.User.Email
			if sub.User.LastLogin.Valid {
				userInfo["last_login"] = sub.User.LastLogin.Time.Format(TimeLayout)
			}
		}

		result = append(result, gin.H{
			"id":                sub.ID,
			"user_id":           sub.UserID,
			"username":          userInfo["username"],
			"email":             userInfo["email"],
			"last_login":        userInfo["last_login"],
			"expire_time":       sub.ExpireTime.Format(TimeLayout),
			"days_until_expire": daysUntilExpire,
		})
	}
	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// ==========================================
// 订阅配置管理
// ==========================================

func validateSubscription(subscription *models.Subscription, user *models.User, db *gorm.DB, clientIP, userAgent string) (string, int, int, bool) {
	now := utils.GetBeijingTime()

	isExpired := subscription.ExpireTime.Before(now)
	isInactive := !subscription.IsActive || subscription.Status != "active"
	isSpecialValid := user.SpecialNodeExpiresAt.Valid && user.SpecialNodeExpiresAt.Time.After(now)

	if isExpired && !isSpecialValid {
		return fmt.Sprintf("订阅已过期(到期时间:%s)，请续费", subscription.ExpireTime.Format(DateFormat)), 0, subscription.DeviceLimit, false
	}
	if isInactive {
		return "订阅已失效或被禁用，请联系客服", 0, subscription.DeviceLimit, false
	}

	var count int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscription.ID, true).Count(&count)

	if subscription.DeviceLimit == 0 {
		return "设备数量限制为0，无法使用服务", int(count), subscription.DeviceLimit, false
	}

	if subscription.DeviceLimit > 0 && int(count) >= subscription.DeviceLimit {
		hash := device.NewDeviceManager().GenerateDeviceHash(userAgent, clientIP, "")
		var currentDevice models.Device
		isCurrentDeviceExists := db.Where("device_hash = ? AND subscription_id = ?", hash, subscription.ID).First(&currentDevice).Error == nil

		if !isCurrentDeviceExists {
			return fmt.Sprintf("设备数量超过限制(当前%d/限制%d)，无法添加新设备", count, subscription.DeviceLimit), int(count), subscription.DeviceLimit, false
		}

		var allowedDevices []models.Device
		db.Where("subscription_id = ? AND is_active = ?", subscription.ID, true).
			Order("last_access DESC").
			Limit(subscription.DeviceLimit).
			Find(&allowedDevices)

		isAllowed := false
		for _, allowedDevice := range allowedDevices {
			if allowedDevice.ID == currentDevice.ID {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			return fmt.Sprintf("设备数量超过限制(当前%d/限制%d)，此设备不在允许范围内", count, subscription.DeviceLimit), int(count), subscription.DeviceLimit, false
		}
	}

	return "", int(count), subscription.DeviceLimit, true
}

func GetSubscriptionConfig(c *gin.Context) {
	// Clash 订阅是公开的，通过 subscription_url 访问，不需要登录
	clashURL := c.Param("url")
	db := database.GetDB()
	baseURL := utils.GetBuildBaseURL(c.Request, db)
	var subscription models.Subscription

	// 通过 subscription_url 查找订阅
	if err := db.Where("subscription_url = ?", clashURL).First(&subscription).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.CreateBusinessLogAsync(c, "subscription_pull_not_found", "订阅拉取: token 无效或订阅不存在", "warning", nil)
			c.String(200, generateErrorConfigBase64("错误", "订阅不存在", baseURL))
			return
		}
		utils.CreateBusinessLogAsync(c, "subscription_pull_query_failed", "订阅拉取: 查询订阅失败", "error", map[string]interface{}{"reason": err.Error()})
		c.String(200, generateErrorConfigBase64("错误", "查询订阅失败", baseURL))
		return
	}

	// 验证订阅状态
	now := utils.GetBeijingTime()
	isExpired := subscription.ExpireTime.Before(now)
	isInactive := !subscription.IsActive || subscription.Status != "active"

	// 检查用户是否有特殊节点权限
	var user models.User
	var isSpecialValid bool
	if err := db.First(&user, subscription.UserID).Error; err == nil {
		isSpecialValid = user.SpecialNodeExpiresAt.Valid && user.SpecialNodeExpiresAt.Time.After(now)
	}

	if isExpired && !isSpecialValid {
		c.String(200, generateErrorConfigBase64("订阅已过期", fmt.Sprintf("到期时间: %s，请续费", subscription.ExpireTime.Format(DateFormat)), baseURL))
		return
	}
	if isInactive {
		c.String(200, generateErrorConfigBase64("订阅已失效", "订阅已被禁用或状态异常，请联系客服", baseURL))
		return
	}

	clientIP := utils.GetRealClientIP(c)
	userAgent := c.GetHeader("User-Agent")

	// 设备管理逻辑（类似 GetUniversalSubscription）
	deviceManager := device.NewDeviceManager()
	hash := deviceManager.GenerateDeviceHash(userAgent, clientIP, "")

	var currentDevice models.Device
	deviceExists := db.Where("device_hash = ? AND subscription_id = ?", hash, subscription.ID).First(&currentDevice).Error == nil

	if !deviceExists {
		var sameUADevice models.Device
		if err := db.Where("subscription_id = ? AND user_agent = ? AND is_active = ?", subscription.ID, userAgent, true).
			Order("last_access DESC").
			First(&sameUADevice).Error; err == nil {
			sameUADevice.IPAddress = &clientIP
			sameUADevice.DeviceHash = &hash
			sameUADevice.LastAccess = utils.GetBeijingTime()
			if err := db.Save(&sameUADevice).Error; err == nil {
				deviceExists = true
				currentDevice = sameUADevice
			}
		}
	}

	var count int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscription.ID, true).Count(&count)

	// 检查设备限制
	if subscription.DeviceLimit == 0 {
		c.String(200, generateErrorConfigBase64("设备限制", "设备数量限制为0，无法使用服务", baseURL))
		return
	}

	shouldRecord := true
	if !deviceExists {
		if (subscription.DeviceLimit > 0 && int(count) >= subscription.DeviceLimit) || subscription.DeviceLimit == 0 {
			shouldRecord = false
		}
	}

	// 如果设备数量超过限制且当前设备不在允许范围内
	if subscription.DeviceLimit > 0 && int(count) >= subscription.DeviceLimit {
		if !deviceExists {
			// 设备不存在且已达到限制，不允许添加新设备
			c.String(200, generateErrorConfigBase64("设备限制", fmt.Sprintf("设备数量超过限制(当前%d/限制%d)，无法添加新设备", count, subscription.DeviceLimit), baseURL))
			return
		}
		// 设备存在，检查是否在允许的设备列表中
		var allowedDevices []models.Device
		db.Where("subscription_id = ? AND is_active = ?", subscription.ID, true).
			Order("last_access DESC").
			Limit(subscription.DeviceLimit).
			Find(&allowedDevices)

		isAllowed := false
		for _, allowedDevice := range allowedDevices {
			if allowedDevice.ID == currentDevice.ID {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			c.String(200, generateErrorConfigBase64("设备限制", fmt.Sprintf("设备数量超过限制(当前%d/限制%d)，此设备不在允许范围内", count, subscription.DeviceLimit), baseURL))
			return
		}
	}

	// 异步记录设备访问和更新计数（不阻塞配置返回）
	if shouldRecord {
		go func(subID, userID uint, ua, ip string) {
			deviceManager.RecordDeviceAccess(subID, userID, ua, ip, "clash")
		}(subscription.ID, subscription.UserID, userAgent, clientIP)

		go func(subID uint) {
			db := database.GetDB()
			db.Model(&models.Subscription{}).Where("id = ?", subID).
				Update("clash_count", gorm.Expr("clash_count + ?", 1))
		}(subscription.ID)
	}

	// 生成 Clash 配置
	configService := config_update.NewConfigUpdateService()
	config, err := configService.GenerateClashConfig(clashURL, clientIP, userAgent)
	if err != nil {
		c.String(200, generateErrorConfigBase64("错误", "生成配置失败", baseURL))
		return
	}

	// 生成订阅名称（用于 HTTP 响应头）
	ctx := configService.GetSubscriptionContext(clashURL, clientIP, userAgent)
	subscriptionName := configService.GenerateSubscriptionName(ctx)

	// 生成文件名（格式：到期时间2026-02-XX 或 无限期订阅）
	fileName := subscriptionName
	if strings.HasPrefix(subscriptionName, "到期: ") {
		// 将 "到期: 2026-02-XX" 转换为 "到期时间2026-02-XX"
		fileName = "到期时间" + strings.TrimPrefix(subscriptionName, "到期: ")
	} else if subscriptionName == "无限期订阅" {
		fileName = "无限期订阅"
	}

	// URL 编码文件名（用于 Content-Disposition 头部）
	encodedName := url.QueryEscape(fileName)

	// 返回 YAML 格式的 Clash 配置
	c.Header("Content-Type", "text/yaml; charset=utf-8")

	// 1. 标准头部 (告诉浏览器和客户端这是个文件，且指定文件名)
	// 注意：filename*=UTF-8'' 这种格式能最好地兼容中文
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s.yaml", encodedName))

	// 2. 兼容性头部 (部分客户端读取此字段)
	c.Header("Subscription-Title", subscriptionName)
	c.Header("Profile-Title", subscriptionName)

	// 3. 订阅信息头部（某些客户端可能使用）
	// 设置流量为无限（0 表示无限制）
	userinfoParts := []string{
		"upload=0",
		"download=0",
		"total=0",
	}
	if !subscription.ExpireTime.IsZero() {
		expireUnix := subscription.ExpireTime.Unix()
		userinfoParts = append(userinfoParts, fmt.Sprintf("expire=%d", expireUnix))
	}
	c.Header("Subscription-Userinfo", strings.Join(userinfoParts, "; "))

	// 4. 更新间隔头部（部分客户端可能使用）
	c.Header("Profile-Update-Interval", "24")

	c.String(200, config)
}

func UpdateSubscriptionConfig(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	db := database.GetDB()
	var subscription models.Subscription
	if err := db.Where("user_id = ? AND is_active = ?", user.ID, true).Order("created_at DESC").First(&subscription).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, "未找到有效订阅", nil)
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询订阅失败", err)
		return
	}

	clientIP := utils.GetRealClientIP(c)
	userAgent := c.GetHeader("User-Agent")

	message, deviceCount, deviceLimit, isValid := validateSubscription(&subscription, user, db, clientIP, userAgent)
	if !isValid {
		utils.CreateBusinessLogAsync(c, "subscription_validation_failed", "订阅校验未通过: "+message, "warning", map[string]interface{}{
			"user_id": user.ID, "subscription_id": subscription.ID, "reason": message,
		})
		utils.ErrorResponse(c, http.StatusBadRequest, message, nil)
		return
	}

	config, err := config_update.NewConfigUpdateService().GenerateClashConfig(subscription.SubscriptionURL, clientIP, userAgent)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成订阅配置失败", err)
		return
	}

	subscriptionURL := utils.GenerateSubscriptionURL()
	subscription.SubscriptionURL = subscriptionURL
	if err := db.Save(&subscription).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新订阅URL失败", err)
		return
	}

	encodedConfig := base64.StdEncoding.EncodeToString([]byte(config))
	utils.SuccessResponse(c, http.StatusOK, "订阅配置更新成功", gin.H{
		"config":           encodedConfig,
		"subscription_id":  subscription.ID,
		"subscription_url": subscriptionURL,
		"device_count":     deviceCount,
		"device_limit":     deviceLimit,
	})
}

// generateErrorConfigBase64 生成Base64编码的错误配置
func generateErrorConfigBase64(title, message string, baseURL string) string {
	cleanMessage := strings.ReplaceAll(message, "\n", " ")

	if baseURL == "" {
		baseURL = "请登录官网"
	} else if len(baseURL) > 30 {
		baseURL = baseURL[:27] + "..."
	}

	errorReason := cleanMessage
	if len(errorReason) > 30 {
		errorReason = errorReason[:27] + "..."
	}

	errorNodes := []string{
		fmt.Sprintf("🌐 %s", baseURL),
		fmt.Sprintf("⚠️ %s", errorReason),
		"💡 请登录官网查看详情",
		"📞 联系管理员获取帮助",
	}

	var nodeLinks []string
	for i, nodeName := range errorNodes {
		errorData := map[string]interface{}{
			"v":    "2",
			"ps":   nodeName,
			"add":  "baidu.com",
			"port": i,
			"id":   "00000000-0000-0000-0000-000000000000",
			"net":  "tcp",
			"type": "none",
		}

		jsonData, _ := json.Marshal(errorData)
		encoded := base64.StdEncoding.EncodeToString(jsonData)
		nodeLinks = append(nodeLinks, "vmess://"+encoded)
	}

	content := strings.Join(nodeLinks, "\n")
	return base64.StdEncoding.EncodeToString([]byte(content))
}

func GetUniversalSubscription(c *gin.Context) {
	uurl := c.Param("url")
	db := database.GetDB()
	baseURL := utils.GetBuildBaseURL(c.Request, db)
	var sub models.Subscription

	// 统一处理订阅查找和设备记录逻辑
	if err := db.Where("subscription_url = ?", uurl).First(&sub).Error; err == nil {
		deviceIP := utils.GetRealClientIP(c)
		deviceUA := c.GetHeader("User-Agent")
		deviceManager := device.NewDeviceManager()
		hash := deviceManager.GenerateDeviceHash(deviceUA, deviceIP, "")

		var currentDevice models.Device
		deviceExists := db.Where("device_hash = ? AND subscription_id = ?", hash, sub.ID).First(&currentDevice).Error == nil

		if !deviceExists {
			var sameUADevice models.Device
			if err := db.Where("subscription_id = ? AND user_agent = ? AND is_active = ?", sub.ID, deviceUA, true).
				Order("last_access DESC").
				First(&sameUADevice).Error; err == nil {

				sameUADevice.IPAddress = &deviceIP
				sameUADevice.DeviceHash = &hash
				sameUADevice.LastAccess = utils.GetBeijingTime()

				if err := db.Save(&sameUADevice).Error; err == nil {
					deviceExists = true
					currentDevice = sameUADevice
				}
			}
		}

		var count int64
		db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", sub.ID, true).Count(&count)

		shouldRecord := true
		if !deviceExists {
			if (sub.DeviceLimit > 0 && int(count) >= sub.DeviceLimit) || sub.DeviceLimit == 0 {
				shouldRecord = false
			}
		}

		// 异步记录设备访问和更新计数（不阻塞配置返回）
		if shouldRecord {
			go func(subID, userID uint, ua, ip string) {
				deviceManager.RecordDeviceAccess(subID, userID, ua, ip, "universal")
			}(sub.ID, sub.UserID, deviceUA, deviceIP)

			go func(subID uint) {
				db := database.GetDB()
				db.Model(&models.Subscription{}).Where("id = ?", subID).
					Update("universal_count", gorm.Expr("universal_count + ?", 1))
			}(sub.ID)
		}

		// 正常生成配置
		cfg, err := config_update.NewConfigUpdateService().GenerateUniversalConfig(uurl, deviceIP, deviceUA, "base64")
		if err != nil {
			c.String(200, generateErrorConfigBase64("错误", "生成配置失败", baseURL))
			return
		}
		c.String(200, cfg)
		return
	}

	// 订阅未找到
	c.String(200, generateErrorConfigBase64("错误", "订阅不存在", baseURL))
}

func GetConfigUpdateStatus(c *gin.Context) {
	service := config_update.NewConfigUpdateService()
	status := service.GetStatus()
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"is_running":  status["is_running"],
		"last_update": status["last_update"],
		"next_update": status["next_update"],
	})
}

func GetConfigUpdateConfig(c *gin.Context) {
	db := database.GetDB()
	var configs []models.SystemConfig
	db.Where("category = ?", "config_update").Find(&configs)

	configMap := make(map[string]interface{})
	defaultConfig := map[string]interface{}{
		"urls":              []string{},
		"filter_keywords":   []string{},
		"enable_schedule":   false,
		"schedule_interval": 3600,
	}

	var urlsConfig *models.SystemConfig

	for _, config := range configs {
		key := config.Key
		value := config.Value

		switch key {
		case "urls":
			urlsConfig = &config
		case "filter_keywords":
			urls := strings.Split(value, "\n")
			filtered := make([]string, 0)
			for _, url := range urls {
				if s := strings.TrimSpace(url); s != "" {
					filtered = append(filtered, s)
				}
			}
			configMap[key] = filtered
		case "enable_schedule":
			configMap[key] = value == "true" || value == "1"
		case "schedule_interval":
			var interval int
			fmt.Sscanf(value, "%d", &interval)
			configMap[key] = interval
		default:
			configMap[key] = value
		}
	}

	if urlsConfig != nil && strings.TrimSpace(urlsConfig.Value) != "" {
		urls := strings.Split(urlsConfig.Value, "\n")
		filtered := make([]string, 0)
		for _, url := range urls {
			if s := strings.TrimSpace(url); s != "" {
				filtered = append(filtered, s)
			}
		}
		configMap["urls"] = filtered
	}

	for key, defaultValue := range defaultConfig {
		if _, exists := configMap[key]; !exists {
			configMap[key] = defaultValue
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", configMap)
}

func GetConfigUpdateFiles(c *gin.Context) {
	service := config_update.NewConfigUpdateService()
	config, err := service.GetConfig()
	if err != nil {
		utils.SuccessResponse(c, http.StatusOK, "", []gin.H{})
		return
	}

	targetDir, _ := config["target_dir"].(string)
	v2rayFile, _ := config["v2ray_file"].(string)
	clashFile, _ := config["clash_file"].(string)

	if targetDir == "" {
		targetDir = "./uploads/config"
	}
	if v2rayFile == "" {
		v2rayFile = "xr"
	}
	clashFile = filepath.Base(clashFile)

	targetDir = filepath.Clean(targetDir)
	v2rayPath := filepath.Join(targetDir, v2rayFile)
	clashPath := filepath.Join(targetDir, clashFile)

	getFileInfo := func(name, path string) gin.H {
		res := gin.H{"name": name, "path": path, "size": 0, "exists": false}
		if info, err := os.Stat(path); err == nil {
			res["size"] = info.Size()
			res["modified"] = info.ModTime().Format(TimeLayout)
			res["exists"] = true
		}
		return res
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"v2ray": getFileInfo(v2rayFile, v2rayPath),
		"clash": getFileInfo(clashFile, clashPath),
	})
}

func GetConfigUpdateLogs(c *gin.Context) {
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}
	service := config_update.NewConfigUpdateService()
	utils.SuccessResponse(c, http.StatusOK, "", service.GetLogs(limit))
}

func ClearConfigUpdateLogs(c *gin.Context) {
	service := config_update.NewConfigUpdateService()
	if err := service.ClearLogs(); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "清理失败", err)
		return
	}
	utils.CreateAuditLogSimple(c, "clear_config_update_logs", "config_update", 0, "管理员操作: 清理配置更新日志")
	utils.SuccessResponse(c, http.StatusOK, "日志已清理", nil)
}

// formatConfigValue 将任意类型转换为字符串存储格式
func formatConfigValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []interface{}:
		urls := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				urls = append(urls, s)
			}
		}
		return strings.Join(urls, "\n")
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		return fmt.Sprintf("%.0f", v)
	default:
		j, _ := json.Marshal(v)
		return string(j)
	}
}

func UpdateConfigUpdateConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	db := database.GetDB()

	// 特殊处理 urls 字段
	if urlsValue, ok := req["urls"]; ok {
		req["urls"] = formatConfigValue(urlsValue)
	}

	for key, value := range req {
		valueStr := formatConfigValue(value)

		var config models.SystemConfig
		err := db.Where("key = ? AND category = ?", key, "config_update").First(&config).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			config = models.SystemConfig{
				Key:      key,
				Value:    valueStr,
				Category: "config_update",
				Type:     "config_update",
			}
			if err := db.Create(&config).Error; err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("保存配置 %s 失败", key), err)
				return
			}
		} else if err == nil {
			config.Value = valueStr
			if err := db.Save(&config).Error; err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("更新配置 %s 失败", key), err)
				return
			}
		} else {
			// 其他 DB 错误
			continue
		}
	}
	utils.CreateAuditLogSimple(c, "update_config_update_config", "config_update", 0, "管理员操作: 更新配置更新设置")
	utils.SuccessResponse(c, http.StatusOK, "配置保存成功", nil)
}

func StartConfigUpdate(c *gin.Context) {
	service := config_update.NewConfigUpdateService()
	go func() {
		_ = service.RunUpdateTask()
	}()
	utils.CreateAuditLogSimple(c, "start_config_update", "config_update", 0, "管理员操作: 启动配置更新任务")
	utils.SuccessResponse(c, http.StatusOK, "配置更新任务已启动", nil)
}

func StopConfigUpdate(c *gin.Context) {
	// 这里假设有一个 Stop 方法或类似的机制，原代码只有响应
	utils.CreateAuditLogSimple(c, "stop_config_update", "config_update", 0, "管理员操作: 停止配置更新任务")
	utils.SuccessResponse(c, http.StatusOK, "配置更新任务停止指令已发送", nil)
}

func TestConfigUpdate(c *gin.Context) {
	service := config_update.NewConfigUpdateService()
	go func() {
		_ = service.RunUpdateTask()
	}()
	utils.SuccessResponse(c, http.StatusOK, "测试任务已启动", nil)
}

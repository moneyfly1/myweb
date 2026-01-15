package handlers

import (
	"fmt"
	"net/http"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/geoip"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetDevices 获取用户设备列表
func GetDevices(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	db := database.GetDB()
	var devices []models.Device

	// 获取用户的订阅ID列表
	var subscriptionIDs []uint
	db.Model(&models.Subscription{}).Where("user_id = ?", user.ID).Pluck("id", &subscriptionIDs)

	if len(subscriptionIDs) == 0 {
		utils.SuccessResponse(c, http.StatusOK, "", []gin.H{})
		return
	}

	// 查询设备，按最后访问时间排序
	if err := db.Where("subscription_id IN ?", subscriptionIDs).
		Order("last_access DESC").
		Find(&devices).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取设备列表失败", err)
		return
	}

	// 格式化设备信息
	deviceList := make([]gin.H, 0, len(devices))
	for _, d := range devices {
		getString := func(ptr *string) string {
			if ptr != nil {
				return *ptr
			}
			return ""
		}

		lastSeen := d.LastAccess.Format("2006-01-02 15:04:05")
		if d.LastSeen != nil {
			lastSeen = d.LastSeen.Format("2006-01-02 15:04:05")
		}

		firstSeen := ""
		if d.FirstSeen != nil {
			firstSeen = d.FirstSeen.Format("2006-01-02 15:04:05")
		}

		// IP地址格式化函数
		formatIP := func(ip string) string {
			if ip == "" {
				return "-"
			}
			// 将 IPv6 本地地址转换为 IPv4 本地地址
			if ip == "::1" {
				return "127.0.0.1"
			}
			// 将 IPv6 映射的 IPv4 地址转换
			if len(ip) >= 7 && ip[:7] == "::ffff:" {
				return ip[7:]
			}
			return ip
		}

		ipStr := getString(d.IPAddress)
		ipAddress := formatIP(ipStr)

		// 使用GeoIP解析地理位置
		location := ""
		if ipAddress != "" && ipAddress != "-" && geoip.IsEnabled() {
			locationStr := geoip.GetLocationString(ipAddress)
			if locationStr.Valid {
				location = locationStr.String
			}
		}

		deviceList = append(deviceList, gin.H{
			"id":                 d.ID,
			"subscription_id":    d.SubscriptionID,
			"device_name":        getString(d.DeviceName),
			"device_type":        getString(d.DeviceType),
			"device_model":       getString(d.DeviceModel),
			"device_brand":       getString(d.DeviceBrand),
			"device_fingerprint": d.DeviceFingerprint,
			"ip_address":         ipAddress,
			"location":           location,
			"user_agent":         getString(d.UserAgent),
			"software_name":      getString(d.SoftwareName),
			"software_version":   getString(d.SoftwareVersion),
			"os_name":            getString(d.OSName),
			"os_version":         getString(d.OSVersion),
			"subscription_type":  getString(d.SubscriptionType),
			"is_active":          d.IsActive,
			"is_allowed":         d.IsAllowed,
			"first_seen":         firstSeen,
			"last_access":        d.LastAccess.Format("2006-01-02 15:04:05"),
			"last_seen":          lastSeen,
			"access_count":       d.AccessCount,
			"created_at":         d.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", deviceList)
}

// DeleteDevice 用户删除自己的设备
func DeleteDevice(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	db := database.GetDB()
	deviceID := c.Param("id")

	var device models.Device
	// 验证设备是否属于当前用户
	if err := db.Where("devices.id = ?", deviceID).
		Joins("JOIN subscriptions ON devices.subscription_id = subscriptions.id").
		Where("subscriptions.user_id = ?", user.ID).
		First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "设备不存在或无权限", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "查询设备失败", err)
		}
		return
	}

	// 删除设备
	if err := db.Delete(&device).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除设备失败", err)
		return
	}

	// 更新订阅的设备计数
	var count int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", device.SubscriptionID, true).Count(&count)
	db.Model(&models.Subscription{}).Where("id = ?", device.SubscriptionID).Update("current_devices", count)

	// 记录审计日志
	utils.CreateAuditLogSimple(c, "delete_device", "device", device.ID, fmt.Sprintf("用户删除设备: %s", getDeviceDisplayName(&device)))

	utils.SuccessResponse(c, http.StatusOK, "设备已删除", nil)
}

// RemoveDevice 管理员删除设备
func RemoveDevice(c *gin.Context) {
	db := database.GetDB()
	deviceID := c.Param("id")

	var device models.Device
	if err := db.First(&device, deviceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "设备不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "查询设备失败", err)
		}
		return
	}

	// 保存设备信息用于审计日志
	deviceInfo := getDeviceDisplayName(&device)
	subscriptionID := device.SubscriptionID

	// 删除设备
	if err := db.Delete(&device).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除设备失败", err)
		return
	}

	// 更新订阅的设备计数
	var count int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscriptionID, true).Count(&count)
	db.Model(&models.Subscription{}).Where("id = ?", subscriptionID).Update("current_devices", count)

	// 记录审计日志
	utils.CreateAuditLogSimple(c, "admin_delete_device", "device", device.ID, fmt.Sprintf("管理员删除设备: %s", deviceInfo))

	utils.SuccessResponse(c, http.StatusOK, "设备已删除", nil)
}

// BatchDeleteDevices 批量删除设备（管理员）
func BatchDeleteDevices(c *gin.Context) {
	var req struct {
		DeviceIDs []uint `json:"device_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	if len(req.DeviceIDs) == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "设备ID列表不能为空", nil)
		return
	}

	db := database.GetDB()

	// 查询要删除的设备
	var devices []models.Device
	if err := db.Where("id IN ?", req.DeviceIDs).Find(&devices).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询设备失败", err)
		return
	}

	if len(devices) == 0 {
		utils.ErrorResponse(c, http.StatusNotFound, "未找到要删除的设备", nil)
		return
	}

	// 收集需要更新计数的订阅ID
	subscriptionIDMap := make(map[uint]bool)
	for _, d := range devices {
		subscriptionIDMap[d.SubscriptionID] = true
	}

	// 批量删除设备
	if err := db.Where("id IN ?", req.DeviceIDs).Delete(&models.Device{}).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "批量删除设备失败", err)
		return
	}

	// 更新相关订阅的设备计数
	for subID := range subscriptionIDMap {
		var count int64
		db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subID, true).Count(&count)
		db.Model(&models.Subscription{}).Where("id = ?", subID).Update("current_devices", count)
	}

	// 记录审计日志
	utils.CreateAuditLogSimple(c, "batch_delete_devices", "device", 0, fmt.Sprintf("管理员批量删除设备: %d 个", len(devices)))

	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功删除 %d 个设备", len(devices)), gin.H{"deleted_count": len(devices)})
}

// GetDeviceStats 获取设备统计（管理员）
func GetDeviceStats(c *gin.Context) {
	db := database.GetDB()

	var stats struct {
		TotalDevices       int64            `json:"total_devices"`
		ActiveDevices      int64            `json:"active_devices"`
		InactiveDevices    int64            `json:"inactive_devices"`
		TotalSubscriptions int64            `json:"total_subscriptions"`
		DevicesByType      map[string]int64 `json:"devices_by_type"`
	}

	db.Model(&models.Device{}).Count(&stats.TotalDevices)
	db.Model(&models.Device{}).Where("is_active = ?", true).Count(&stats.ActiveDevices)
	db.Model(&models.Device{}).Where("is_active = ?", false).Count(&stats.InactiveDevices)
	db.Model(&models.Subscription{}).Count(&stats.TotalSubscriptions)

	// 按设备类型统计
	stats.DevicesByType = make(map[string]int64)
	var typeStats []struct {
		DeviceType string
		Count      int64
	}
	db.Model(&models.Device{}).
		Select("COALESCE(device_type, 'unknown') as device_type, count(*) as count").
		Group("device_type").
		Scan(&typeStats)

	for _, ts := range typeStats {
		stats.DevicesByType[ts.DeviceType] = ts.Count
	}

	utils.SuccessResponse(c, http.StatusOK, "", stats)
}

// getDeviceDisplayName 获取设备显示名称
func getDeviceDisplayName(device *models.Device) string {
	if device.DeviceName != nil && *device.DeviceName != "" {
		return *device.DeviceName
	}
	if device.DeviceModel != nil && *device.DeviceModel != "" {
		return *device.DeviceModel
	}
	if device.SoftwareName != nil && *device.SoftwareName != "" {
		return *device.SoftwareName
	}
	if device.UserAgent != nil && *device.UserAgent != "" {
		ua := *device.UserAgent
		if len(ua) > 50 {
			return ua[:50] + "..."
		}
		return ua
	}
	return fmt.Sprintf("设备 #%d", device.ID)
}

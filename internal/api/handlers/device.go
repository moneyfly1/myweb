package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	devicesvc "cboard-go/internal/services/device"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetDevices(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	db := database.GetDB()
	var devices []models.Device

	var subscriptionIDs []uint
	db.Model(&models.Subscription{}).Where("user_id = ?", user.ID).Pluck("id", &subscriptionIDs)

	if len(subscriptionIDs) == 0 {
		utils.SuccessResponse(c, http.StatusOK, "", []gin.H{})
		return
	}

	if err := db.Where("subscription_id IN ?", subscriptionIDs).
		Order("last_access DESC").
		Find(&devices).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取设备列表失败", err)
		return
	}

	deviceList := make([]gin.H, 0, len(devices))
	for _, d := range devices {
		lastSeen := utils.FormatBeijingTime(d.LastAccess)
		if d.LastSeen != nil {
			lastSeen = utils.FormatBeijingTime(*d.LastSeen)
		}

		firstSeen := ""
		if d.FirstSeen != nil {
			firstSeen = utils.FormatBeijingTime(*d.FirstSeen)
		}

		devAct := devicesvc.ActivityOf(d, utils.GetBeijingTime())

		ipStr := utils.GetStringValue(d.IPAddress)
		ipAddress := utils.FormatIP(ipStr)

		// 列表查询不查询 GeoIP，提升性能
		location := ""

		deviceList = append(deviceList, gin.H{
			"id":                        d.ID,
			"subscription_id":           d.SubscriptionID,
			"device_name":               utils.GetStringValue(d.DeviceName),
			"device_type":               utils.GetStringValue(d.DeviceType),
			"device_model":              utils.GetStringValue(d.DeviceModel),
			"device_brand":              utils.GetStringValue(d.DeviceBrand),
			"device_fingerprint":        d.DeviceFingerprint,
			"ip_address":                ipAddress,
			"location":                  location,
			"user_agent":                utils.GetStringValue(d.UserAgent),
			"software_name":             utils.GetStringValue(d.SoftwareName),
			"software_version":          utils.GetStringValue(d.SoftwareVersion),
			"os_name":                   utils.GetStringValue(d.OSName),
			"os_version":                utils.GetStringValue(d.OSVersion),
			"subscription_type":         utils.GetStringValue(d.SubscriptionType),
			"is_active":                 d.IsActive,
			"is_allowed":                d.IsAllowed,
			"activity_state":            devAct.State,
			"activity_state_label":      devAct.StateLabel,
			"activity_ago_seconds":      devAct.AgoSeconds,
			"activity_avg_interval_sec": devAct.AvgIntervalSec,
			"activity_threshold_sec":    devAct.ThresholdSec,
			"activity_basis":            devAct.Basis,
			"first_seen":                firstSeen,
			"last_access":               utils.FormatBeijingTime(d.LastAccess),
			"last_seen":                 lastSeen,
			"access_count":              d.AccessCount,
			"created_at":                utils.FormatBeijingTime(d.CreatedAt),
			"remark":                    utils.GetStringValue(d.Remark),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", deviceList)
}

// UserDeviceDeleteAllowed 是否允许用户自行删除（踢下线）设备。
//
// 由「系统设置 → 用户与注册 → 允许用户删除设备」控制（config key:
// allow_user_delete_device，category: registration）：
//   - 允许（默认）：用户端设备管理显示删除按钮；
//   - 不允许：用户端不显示删除按钮，只能升级设备数量后再连接新设备。
//
// 未配置时按「允许」处理，保持既有行为。
func UserDeviceDeleteAllowed(db *gorm.DB) bool {
	v, err := utils.GetCachedSetting(db, "allow_user_delete_device", "registration")
	if err != nil || strings.TrimSpace(v) == "" {
		return true
	}
	return !strings.EqualFold(strings.TrimSpace(v), "false")
}

func DeleteDevice(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	db := database.GetDB()

	// 管理员可关闭「用户删除设备」：此时用户端不显示删除按钮，这里再做服务端拦截，
	// 避免直接调接口绕过（提示引导用户去升级设备数量）。
	if !UserDeviceDeleteAllowed(db) {
		utils.ErrorResponse(c, http.StatusForbidden, "管理员已关闭设备删除，如需更换设备请升级设备数量", nil)
		return
	}
	deviceID := c.Param("id")

	var device models.Device
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

	// 删除设备 = 踢下线（软删除）：
	// - IsActive=false + KickedAt=now：设备立即从活跃列表移除、名额释放，
	//   该设备再次拉订阅时被拒并提示「已被移除/踢下线」；
	// - 行保留（不物理删除）：按 device_hash/UA 匹配识别，防止被删设备
	//   静默重新注册复活（若真删除则下次订阅访问又会新建一行，等于没踢）。
	now := utils.GetBeijingTime()
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Device{}).Where("id = ?", device.ID).
			Updates(map[string]interface{}{
				"is_active":   false,
				"kicked_at":   now,
				"last_access": now,
			}).Error; err != nil {
			return err
		}
		count, _ := devicesvc.CountRecentDevices(tx, device.SubscriptionID, devicesvc.DisplayWindow)
		return tx.Model(&models.Subscription{}).Where("id = ?", device.SubscriptionID).Update("current_devices", count).Error
	}); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除设备失败", err)
		return
	}

	utils.CreateAuditLogSimpleFast(c, "delete_device", "device", device.ID, fmt.Sprintf("用户删除设备(踢下线): %s", getDeviceDisplayName(&device)))

	utils.SuccessResponse(c, http.StatusOK, "设备已删除并下线", nil)
}

// RebindCurrentDevice 把「被踢下线」的本机重新登记回设备列表（用户自助恢复）。
//
// 背景：删除设备 = 软删 + kicked_at，此后该设备拉订阅一律 403
// 「此设备已被移除并踢下线,如需继续使用请重新登录或联系客服」；而此前没有任何
// 代码会清除 kicked_at（详见 DeviceManager.ReviveKickedDevice 的注释），
// 用户删掉本机后就永久失联，只能找客服改库。
//
// 客户端在收到「已被移除」提示时调用本接口一次即可恢复，无需重新登录、
// 也不需要重新输入密码（token 仍然有效，被踢只影响订阅接口）。
// 名额已满时返回 403 并提示先删除闲置设备（与设备超限的引导一致）。
func RebindCurrentDevice(c *gin.Context) {
	db := database.GetDB()
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	var sub models.Subscription
	if err := db.Where("user_id = ?", user.ID).Order("created_at DESC").First(&sub).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "当前账号没有订阅", nil)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "查询订阅失败", err)
		}
		return
	}

	clientIP := utils.GetRealClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	dm := devicesvc.NewDeviceManager() // 内部使用全局数据库连接
	subID := sub.ID
	revived, reason, err := dm.ReviveKickedDeviceWithHeaders(
		sub.ID, sub.DeviceLimit, user.SpecialNodeUnlimitedDevices, userAgent, clientIP, extractMFHeaders(c))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "重新绑定设备失败", err)
		return
	}
	if !revived {
		if reason == "device-limit" {
			utils.ErrorResponse(c, http.StatusForbidden,
				"设备名额已满，请先在设备管理里删除不再使用的设备后再试", nil)
			return
		}
		// 本机并未处于被踢状态：直接告诉客户端可以去更新订阅了
		utils.SuccessResponse(c, http.StatusOK, "本机设备正常，无需重新绑定", nil)
		return
	}

	utils.CreateAuditLogSimpleFast(c, "rebind_device", "subscription", subID,
		fmt.Sprintf("用户自助重新绑定被踢设备(UA: %s, IP: %s)", userAgent, clientIP))
	utils.SuccessResponse(c, http.StatusOK, "本机设备已重新绑定", nil)
}

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

	deviceInfo := getDeviceDisplayName(&device)
	subscriptionID := device.SubscriptionID

	// 管理员删除 = 软删 + kicked_at，与用户端 DeleteDevice 完全一致。
	//
	// 旧实现用 tx.Delete(&device) 做**物理删除**：行一消失，该设备下次拉订阅时
	// 就再也匹配不到历史行（FindExistingDeviceWithHeaders 找不到记录），于是
	// 被判为新设备**重新登记一行** —— 管理员在后台「删了设备」，用户那边过几分钟
	// 又冒出一条同名设备，等于没删（2026-10-30 生产排查：订阅 386 的 4 台设备
	// 被硬删后，用户重新连接又生成新行）。
	//
	// 软删保留了 device_hash 指纹，既能立即释放名额，又能让该设备再次请求时命中
	// kickGate 走「已移除」判定，而不是静默复活成一条新设备。
	now := utils.GetBeijingTime()
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Device{}).Where("id = ?", device.ID).
			Updates(map[string]interface{}{
				"is_active":   false,
				"kicked_at":   now,
				"last_access": now,
			}).Error; err != nil {
			return err
		}
		count, _ := devicesvc.CountRecentDevices(tx, subscriptionID, devicesvc.DisplayWindow)
		return tx.Model(&models.Subscription{}).Where("id = ?", subscriptionID).Update("current_devices", count).Error
	}); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除设备失败", err)
		return
	}

	utils.CreateAuditLogSimple(c, "admin_delete_device", "device", device.ID, fmt.Sprintf("管理员删除设备(踢下线): %s", deviceInfo))

	utils.SuccessResponse(c, http.StatusOK, "设备已删除并下线", nil)
}

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

	var devices []models.Device
	if err := db.Where("id IN ?", req.DeviceIDs).Find(&devices).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询设备失败", err)
		return
	}

	if len(devices) == 0 {
		utils.ErrorResponse(c, http.StatusNotFound, "未找到要删除的设备", nil)
		return
	}

	subscriptionIDMap := make(map[uint]bool)
	for _, d := range devices {
		subscriptionIDMap[d.SubscriptionID] = true
	}

	now := utils.GetBeijingTime()
	if err := db.Transaction(func(tx *gorm.DB) error {
		// 与单个删除一致：软删 + kicked_at（保留指纹行），不做物理删除。
		// 物理删除会让这些设备下次拉订阅时匹配不到历史行 → 静默重建为「新设备」。
		if err := tx.Model(&models.Device{}).Where("id IN ?", req.DeviceIDs).
			Updates(map[string]interface{}{
				"is_active":   false,
				"kicked_at":   now,
				"last_access": now,
			}).Error; err != nil {
			return err
		}
		// 一次 GROUP BY 拿到所有受影响订阅的在线设备数，再逐条回写：
		// 此前是「每个订阅一次 COUNT + 一次 UPDATE」，批量删除涉及几十个订阅就是上百条 SQL。
		subIDs := make([]uint, 0, len(subscriptionIDMap))
		for subID := range subscriptionIDMap {
			subIDs = append(subIDs, subID)
		}
		type countRow struct {
			SubscriptionID uint  `gorm:"column:subscription_id"`
			Cnt            int64 `gorm:"column:cnt"`
		}
		var counts []countRow
		if len(subIDs) > 0 {
			if err := tx.Model(&models.Device{}).
				Select("subscription_id, COUNT(*) AS cnt").
				Where("subscription_id IN ? AND is_active = ?", subIDs, true).
				Group("subscription_id").
				Scan(&counts).Error; err != nil {
				return err
			}
		}
		bySub := make(map[uint]int64, len(counts))
		for _, c := range counts {
			bySub[c.SubscriptionID] = c.Cnt
		}
		for _, subID := range subIDs {
			if err := tx.Model(&models.Subscription{}).Where("id = ?", subID).
				Update("current_devices", bySub[subID]).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "批量删除设备失败", err)
		return
	}

	utils.CreateAuditLogSimple(c, "batch_delete_devices", "device", 0, fmt.Sprintf("管理员批量删除设备: %d 个", len(devices)))

	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功删除 %d 个设备", len(devices)), gin.H{"deleted_count": len(devices)})
}

func GetDeviceStats(c *gin.Context) {
	db := database.GetDB()

	var stats struct {
		TotalDevices       int64            `json:"total_devices"`
		ActiveDevices      int64            `json:"active_devices"`
		InactiveDevices    int64            `json:"inactive_devices"`
		TotalSubscriptions int64            `json:"total_subscriptions"`
		DevicesByType      map[string]int64 `json:"devices_by_type"`
	}

	db.Raw(`
		SELECT
			COUNT(*) AS total_devices,
			COALESCE(SUM(CASE WHEN is_active = ? THEN 1 ELSE 0 END), 0) AS active_devices,
			COALESCE(SUM(CASE WHEN is_active = ? THEN 1 ELSE 0 END), 0) AS inactive_devices
		FROM devices
	`, true, false).Scan(&stats)
	db.Model(&models.Subscription{}).Count(&stats.TotalSubscriptions)

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

func UpdateDeviceRemark(c *gin.Context) {
	user, ok := getCurrentUserOrError(c)
	if !ok {
		return
	}

	db := database.GetDB()
	deviceID := c.Param("id")

	var req struct {
		Remark string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	// 限制备注长度
	if len(req.Remark) > 200 {
		utils.ErrorResponse(c, http.StatusBadRequest, "备注长度不能超过200个字符", nil)
		return
	}

	var device models.Device
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

	remark := req.Remark
	if remark == "" {
		// 允许清空备注
		if err := db.Model(&device).Update("remark", nil).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "更新备注失败", err)
			return
		}
	} else {
		if err := db.Model(&device).Update("remark", remark).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "更新备注失败", err)
			return
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "备注已更新", gin.H{"remark": remark})
}

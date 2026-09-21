package device

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/geoip"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

const clashMetaAndroidAliasWindow = 30 * time.Second

// 预编译常用正则，避免每次 UA 解析在热路径上重新编译
var (
	reIPhoneID       = regexp.MustCompile(`iPhone\d+,\d+`)
	reOpenWrt        = regexp.MustCompile(`OpenWrt[/\s]+(\d+[.\d]*)`)
	reRouterOS       = regexp.MustCompile(`RouterOS[/\s]+(\d+[.\d]*)`)
	reAndroidVersion = regexp.MustCompile(`Android\s+(\d+[.\d]*)`)
	reWindowsNT      = regexp.MustCompile(`Windows\s+NT\s+(\d+\.\d+)`)
	reMacOSX         = regexp.MustCompile(`Mac OS X\s+(\d+[._]\d+)`)
	reIPhoneModel    = regexp.MustCompile(`iPhone(\d+,\d+)`)
	reIPhoneProMax   = regexp.MustCompile(`iPhone\s+(\d+)\s+Pro\s+Max`)
	reIPhonePro      = regexp.MustCompile(`iPhone\s+(\d+)\s+Pro`)
	reIPhoneMini     = regexp.MustCompile(`iPhone\s+(\d+)\s+mini`)
	reIPhonePlain    = regexp.MustCompile(`iPhone\s+(\d+)`)
	reIPadModel      = regexp.MustCompile(`iPad(\d+,\d+)`)
	reIPadPlain      = regexp.MustCompile(`iPad`)
	reAndroidBuild   = regexp.MustCompile(`(?i);\s*([^;]+)\s*build`)
)

type DeviceManager struct {
	db *gorm.DB
}

func NewDeviceManager() *DeviceManager {
	return &DeviceManager{
		db: database.GetDB(),
	}
}

type DeviceInfo struct {
	SoftwareName    string
	SoftwareVersion string
	OSName          string
	OSVersion       string
	DeviceModel     string
	DeviceBrand     string
	DeviceType      string
	DeviceName      string
}

func (dm *DeviceManager) ParseUserAgent(userAgent string) *DeviceInfo {
	return dm.ParseUserAgentWithHeaders(userAgent, nil)
}

// ParseUserAgentWithHeaders 解析 UA 并用 X-MF-* 自定义头补充缺失字段。
// MoneyFly 客户端每个请求都会携带 X-MF-Device-Model / X-MF-Device-Brand /
// X-MF-OS / X-MF-Device-Type 头，确保即使 UA 信息不全也能准确识别设备。
func (dm *DeviceManager) ParseUserAgentWithHeaders(userAgent string, headers map[string]string) *DeviceInfo {
	info := &DeviceInfo{
		SoftwareName:    "Unknown",
		SoftwareVersion: "",
		OSName:          "Unknown",
		OSVersion:       "",
		DeviceModel:     "",
		DeviceBrand:     "",
		DeviceType:      "unknown",
		DeviceName:      "Unknown Device",
	}

	if userAgent == "" {
		return info
	}

	uaLower := strings.ToLower(userAgent)

	info.SoftwareName = dm.matchSoftware(userAgent, uaLower)

	osInfo := dm.parseOSInfo(userAgent, uaLower)
	info.OSName = osInfo["os_name"]
	info.OSVersion = osInfo["os_version"]

	if info.OSName == "Unknown" && info.SoftwareName != "Unknown" {
		inferredOS := dm.inferOSFromSoftware(info.SoftwareName)
		if inferredOS != nil {
			info.OSName = inferredOS["os_name"]
			info.OSVersion = inferredOS["os_version"]
		}
	}

	deviceInfo := dm.parseDeviceInfo(userAgent, info.OSName)
	info.DeviceModel = deviceInfo["device_model"]
	info.DeviceBrand = deviceInfo["device_brand"]

	if info.DeviceModel == "" && info.SoftwareName != "Unknown" {
		inferredDevice := dm.inferDeviceFromSoftware(info.SoftwareName)
		if inferredDevice != nil {
			info.DeviceBrand = inferredDevice["device_brand"]
		}
	}

	info.SoftwareVersion = dm.parseVersion(userAgent)

	info.DeviceType = dm.determineDeviceType(userAgent, info)

	info.DeviceName = dm.generateDeviceName(info)

	// X-MF-* 头补充：MoneyFly 客户端发送的设备详情头，填补 UA 解析不全的字段
	if headers != nil {
		if info.OSName == "Unknown" {
			if osStr := headers["X-MF-OS"]; osStr != "" {
				parts := strings.SplitN(osStr, " ", 2)
				info.OSName = parts[0]
				if len(parts) > 1 {
					info.OSVersion = parts[1]
				}
			}
		}
		if info.DeviceModel == "" {
			if model := headers["X-MF-Device-Model"]; model != "" {
				info.DeviceModel = model
			}
		}
		if info.DeviceBrand == "" {
			if brand := headers["X-MF-Device-Brand"]; brand != "" {
				info.DeviceBrand = brand
			}
		}
		if info.DeviceType == "unknown" {
			if dtype := headers["X-MF-Device-Type"]; dtype != "" {
				info.DeviceType = dtype
			}
		}
		// 补充后重新生成设备名
		if headers["X-MF-OS"] != "" || headers["X-MF-Device-Model"] != "" {
			info.DeviceName = dm.generateDeviceName(info)
		}
	}

	return info
}

func (dm *DeviceManager) IsGenericClashWindowsUA(userAgent string) bool {
	info := dm.ParseUserAgent(userAgent)
	if info.SoftwareName != "Clash" || info.OSName != "Windows" {
		return false
	}

	uaLower := strings.ToLower(userAgent)
	return strings.Contains(uaLower, "windows") &&
		strings.Contains(uaLower, "clash") &&
		!strings.Contains(uaLower, "clash for windows") &&
		!strings.Contains(uaLower, "clash meta") &&
		!strings.Contains(uaLower, "clashmeta") &&
		!strings.Contains(uaLower, "clash-verge") &&
		!strings.Contains(uaLower, "clash verge")
}

func (dm *DeviceManager) isClashMetaAndroidUA(userAgent string) bool {
	info := dm.ParseUserAgent(userAgent)
	return info.SoftwareName == "Clash Meta" && info.OSName == "Android"
}

func (dm *DeviceManager) areSameClashMetaAndroidAlias(a, b string) bool {
	return (dm.IsGenericClashWindowsUA(a) && dm.isClashMetaAndroidUA(b)) ||
		(dm.IsGenericClashWindowsUA(b) && dm.isClashMetaAndroidUA(a))
}

func (dm *DeviceManager) deviceUA(device *models.Device) string {
	if device.UserAgent != nil {
		return *device.UserAgent
	}
	if device.DeviceUA != nil {
		return *device.DeviceUA
	}
	return ""
}

func (dm *DeviceManager) isClashMetaAndroidDevice(device *models.Device) bool {
	if dm.isClashMetaAndroidUA(dm.deviceUA(device)) {
		return true
	}
	return device.SoftwareName != nil && *device.SoftwareName == "Clash Meta" &&
		device.OSName != nil && *device.OSName == "Android"
}

func closeInTime(a, b time.Time, window time.Duration) bool {
	if a.IsZero() || b.IsZero() {
		return false
	}
	if a.After(b) {
		return a.Sub(b) <= window
	}
	return b.Sub(a) <= window
}

func firstSeenTime(device *models.Device) time.Time {
	if device.FirstSeen != nil && !device.FirstSeen.IsZero() {
		return *device.FirstSeen
	}
	return device.CreatedAt
}

func (dm *DeviceManager) findClashMetaAndroidAliasDevice(subscriptionID uint, userAgent, ipAddress string) (*models.Device, error) {
	if ipAddress == "" || (!dm.IsGenericClashWindowsUA(userAgent) && !dm.isClashMetaAndroidUA(userAgent)) {
		return nil, gorm.ErrRecordNotFound
	}

	var candidates []models.Device
	if err := dm.db.Where("subscription_id = ? AND ip_address = ? AND is_active = ? AND last_access >= ?", subscriptionID, ipAddress, true, utils.GetBeijingTime().Add(-clashMetaAndroidAliasWindow)).
		Order("last_access DESC").
		Limit(20).
		Find(&candidates).Error; err != nil {
		return nil, err
	}

	for i := range candidates {
		if dm.areSameClashMetaAndroidAlias(dm.deviceUA(&candidates[i]), userAgent) {
			return &candidates[i], nil
		}
	}

	return nil, gorm.ErrRecordNotFound
}

func (dm *DeviceManager) FindExistingDevice(subscriptionID uint, userAgent, ipAddress string) (*models.Device, bool, error) {
	return dm.FindExistingDeviceWithHeaders(subscriptionID, userAgent, ipAddress, nil)
}

// FindExistingDeviceWithHeaders 同 FindExistingDevice，但带上 X-MF-* 头：
// 设备身份优先用客户端上报的稳定设备 ID（X-MF-Device-Id），并支持把按旧算法
// （含版本号）写入的存量行「认领」过来 —— 见 GenerateLegacyDeviceHash。
func (dm *DeviceManager) FindExistingDeviceWithHeaders(subscriptionID uint, userAgent, ipAddress string, headers map[string]string) (*models.Device, bool, error) {
	deviceHash := dm.GenerateDeviceHashWithHeaders(userAgent, ipAddress, "", headers)

	var device models.Device
	err := dm.db.Where("device_hash = ? AND subscription_id = ?", deviceHash, subscriptionID).First(&device).Error
	if err == nil {
		return &device, true, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, false, err
	}

	// 存量自愈：这一行是旧算法（含版本号）写的 → 认领它并改写为新稳定哈希，
	// 避免升级 App/系统后重复注册出幽灵设备（旧行会一直占着名额）
	if legacyHash := dm.GenerateLegacyDeviceHash(userAgent, headers); legacyHash != "" && legacyHash != deviceHash {
		if err := dm.db.Where("device_hash = ? AND subscription_id = ?", legacyHash, subscriptionID).
			First(&device).Error; err == nil {
			if updErr := dm.db.Model(&models.Device{}).Where("id = ?", device.ID).
				Update("device_hash", deviceHash).Error; updErr != nil {
				return nil, false, updErr
			}
			device.DeviceHash = &deviceHash
			return &device, true, nil
		} else if err != gorm.ErrRecordNotFound {
			return nil, false, err
		}
	}

	if userAgent != "" {
		err = dm.db.Where("subscription_id = ? AND user_agent = ? AND is_active = ?", subscriptionID, userAgent, true).
			Order("last_access DESC").
			First(&device).Error
		if err == nil {
			return &device, true, nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, false, err
		}
	}

	// 历史行兜底：跨版本升级后哈希与 UA 都对不上（旧算法把版本号算进身份，
	// 客户端又换成了稳定设备 ID），按**稳定指纹**（软件名+系统名+机型+品牌）
	// 把同一台设备认回来，并把它的 device_hash 改写成稳定值 —— 否则每次升级
	// 都会多出一行幽灵设备，设备名额被悄悄吃掉。
	if adopted, err := dm.adoptLegacyRowByFingerprint(subscriptionID, userAgent, ipAddress, headers, deviceHash); err != nil {
		return nil, false, err
	} else if adopted != nil {
		return adopted, true, nil
	}

	if aliasDevice, aliasErr := dm.findClashMetaAndroidAliasDevice(subscriptionID, userAgent, ipAddress); aliasErr == nil {
		return aliasDevice, true, nil
	} else if aliasErr != gorm.ErrRecordNotFound {
		return nil, false, aliasErr
	}

	return nil, false, nil
}

// FindKickedDevice 查找被「踢下线」的设备（软删行：is_active=false 且
// KickedAt 非空）。命中说明该设备此前被设备所有者从设备列表删除，当前正在
// 尝试重新拉取订阅 —— 订阅接口应拒绝并提示「已被移除/踢下线」，
// 防止其静默重新注册复活（设备名额已释放，不再放行）。
func (dm *DeviceManager) FindKickedDevice(subscriptionID uint, userAgent, ipAddress string) (*models.Device, error) {
	return dm.FindKickedDeviceWithHeaders(subscriptionID, userAgent, ipAddress, nil)
}

// FindKickedDeviceWithHeaders 同 FindKickedDevice，但带上 X-MF-* 头：设备身份改用
// 稳定特征后，必须同时用**旧算法哈希**再找一遍，否则升级一次 App 就能绕过
// 「已被移除并踢下线」的判定（也就等于绕过用户"删除设备"的意图）。
func (dm *DeviceManager) FindKickedDeviceWithHeaders(subscriptionID uint, userAgent, ipAddress string, headers map[string]string) (*models.Device, error) {
	deviceHash := dm.GenerateDeviceHashWithHeaders(userAgent, ipAddress, "", headers)

	for _, hash := range dm.kickCandidateHashes(userAgent, headers, deviceHash) {
		var device models.Device
		err := dm.db.Where("device_hash = ? AND subscription_id = ? AND is_active = ? AND kicked_at IS NOT NULL",
			hash, subscriptionID, false).First(&device).Error
		if err == nil {
			return &device, nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	if userAgent != "" {
		var byUA models.Device
		err := dm.db.Where("subscription_id = ? AND user_agent = ? AND is_active = ? AND kicked_at IS NOT NULL",
			subscriptionID, userAgent, false).
			Order("last_access DESC").
			First(&byUA).Error
		if err == nil {
			return &byUA, nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	// 历史行兜底：被删除（踢下线）的设备在用户升级 App/系统后哈希与 UA 都变了，
	// 必须按稳定指纹再找一遍，否则「删除设备」的意图可以被一次升级绕过。
	if kicked, err := dm.findKickedLegacyRowByFingerprint(subscriptionID, userAgent, headers); err != nil {
		return nil, err
	} else if kicked != nil {
		return kicked, nil
	}
	return nil, nil
}

// adoptLegacyRowByFingerprint 在订阅内按稳定指纹找到「历史算法写下」的同设备行，
// 把 device_hash 改写为新的稳定值后返回（自愈迁移，不新增行、不丢备注）。
func (dm *DeviceManager) adoptLegacyRowByFingerprint(subscriptionID uint, userAgent, ipAddress string, headers map[string]string, stableHash string) (*models.Device, error) {
	rows, err := dm.legacyRowsByFingerprint(subscriptionID, userAgent, headers, false)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	row := rows[0]
	if err := dm.db.Model(&models.Device{}).Where("id = ?", row.ID).
		Update("device_hash", stableHash).Error; err != nil {
		return nil, err
	}
	row.DeviceHash = &stableHash
	if ipAddress != "" {
		row.IPAddress = &ipAddress
	}
	return &row, nil
}

// findKickedLegacyRowByFingerprint 同上，但只找**被踢下线**的历史行。
func (dm *DeviceManager) findKickedLegacyRowByFingerprint(subscriptionID uint, userAgent string, headers map[string]string) (*models.Device, error) {
	rows, err := dm.legacyRowsByFingerprint(subscriptionID, userAgent, headers, true)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return &rows[0], nil
}

// legacyRowsByFingerprint 按稳定指纹列出候选行（仅历史行，见 isLegacyHashRow）。
func (dm *DeviceManager) legacyRowsByFingerprint(subscriptionID uint, userAgent string, headers map[string]string, onlyKicked bool) ([]models.Device, error) {
	info := dm.ParseUserAgentWithHeaders(userAgent, headers)
	if _, ok := fingerprintKeyOf(info.SoftwareName, info.OSName, info.DeviceModel, info.DeviceBrand); !ok {
		return nil, nil // 指纹不完整（机型/品牌缺失）→ 不做模糊匹配，避免误伤
	}

	q := dm.db.Where(
		"subscription_id = ? AND software_name = ? AND os_name = ? AND device_model = ? AND device_brand = ?",
		subscriptionID, info.SoftwareName, info.OSName, info.DeviceModel, info.DeviceBrand)
	if onlyKicked {
		q = q.Where("is_active = ? AND kicked_at IS NOT NULL", false)
	}
	var rows []models.Device
	if err := q.Order("last_access DESC").Limit(5).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]models.Device, 0, len(rows))
	for i := range rows {
		if isLegacyHashRow(&rows[i]) {
			out = append(out, rows[i])
		}
	}
	return out, nil
}

// kickCandidateHashes 返回该请求可能对应的设备哈希（稳定哈希 + 旧算法哈希）。
// 去重且跳过空值。
func (dm *DeviceManager) kickCandidateHashes(userAgent string, headers map[string]string, stableHash string) []string {
	out := []string{stableHash}
	if legacy := dm.GenerateLegacyDeviceHash(userAgent, headers); legacy != "" && legacy != stableHash {
		out = append(out, legacy)
	}
	return out
}

// ReviveKickedDevice 解除「踢下线」：把被软删的设备重新登记为活跃设备。
//
// 为什么需要（2026-09-21 生产排查结论）：
//   删除设备 = is_active=false + kicked_at=now，此后该设备拉订阅一律 403
//   「此设备已被移除并踢下线,如需继续使用请重新登录或联系客服」。
//   但**代码里没有任何地方会清除 kicked_at**（FindKickedDevice 只读、删除接口只写），
//   于是「请重新登录」实际无效：用户只要删掉了本机（例如到期后清理设备、或先删光
//   再续费），就永久拿不到订阅、设备列表里也不再出现这台设备，只能找客服改库。
//
// 语义（与「防静默复活」的设计意图兼容）：
//   - 只有**确实被踢过的这一台**（按设备哈希/UA 命中软删行）才允许恢复；
//   - **名额未满**才恢复：名额已满说明是"用满额度"场景，应引导用户先删闲置设备；
//   - 恢复保留原行（备注、首次出现时间等都在），只清 kicked_at 并置回活跃；
//   - 恢复后同步订阅的 current_devices。
//
// 返回 (是否恢复, 未恢复原因, 错误)；原因取值："not-kicked" / "device-limit"。
func (dm *DeviceManager) ReviveKickedDevice(subscriptionID uint, deviceLimit int, unlimited bool, userAgent, ipAddress string) (bool, string, error) {
	return dm.ReviveKickedDeviceWithHeaders(subscriptionID, deviceLimit, unlimited, userAgent, ipAddress, nil)
}

// ReviveKickedDeviceWithHeaders 同 ReviveKickedDevice，但带上 X-MF-* 头：
// 设备身份与旧算法哈希都要用同一份头来算，才能精确命中「这台设备」的历史行
// （例如 Windows 客户端的机型来自 X-MF-Device-Model，光靠 UA 算不出同样的身份）。
func (dm *DeviceManager) ReviveKickedDeviceWithHeaders(subscriptionID uint, deviceLimit int, unlimited bool, userAgent, ipAddress string, headers map[string]string) (bool, string, error) {
	kicked, err := dm.FindKickedDeviceWithHeaders(subscriptionID, userAgent, ipAddress, headers)
	if err != nil {
		return false, "", err
	}
	if kicked == nil {
		return false, "not-kicked", nil
	}

	count, err := CountActiveDevices(dm.db, subscriptionID)
	if err != nil {
		return false, "", err
	}
	if !unlimited && deviceLimit > 0 && int(count) >= deviceLimit {
		return false, "device-limit", nil
	}

	now := utils.GetBeijingTime()
	err = dm.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"is_active":   true,
			"kicked_at":   nil,
			"last_access": now,
			"last_seen":   now,
		}
		if ipAddress != "" {
			updates["ip_address"] = ipAddress
		}
		if err := tx.Model(&models.Device{}).Where("id = ?", kicked.ID).
			Updates(updates).Error; err != nil {
			return err
		}
		newCount, cErr := CountActiveDevices(tx, subscriptionID)
		if cErr != nil {
			return cErr
		}
		return tx.Model(&models.Subscription{}).Where("id = ?", subscriptionID).
			Update("current_devices", newCount).Error
	})
	if err != nil {
		return false, "", err
	}
	return true, "", nil
}

func (dm *DeviceManager) deactivateClashMetaAndroidAliasDuplicates(canonical *models.Device, ipAddress string) error {
	if canonical == nil || canonical.ID == 0 || ipAddress == "" || !dm.isClashMetaAndroidDevice(canonical) {
		return nil
	}

	var candidates []models.Device
	if err := dm.db.Where("subscription_id = ? AND ip_address = ? AND is_active = ? AND id <> ?", canonical.SubscriptionID, ipAddress, true, canonical.ID).
		Find(&candidates).Error; err != nil {
		return err
	}

	canonicalFirstSeen := firstSeenTime(canonical)
	deactivated := false
	for i := range candidates {
		if !dm.IsGenericClashWindowsUA(dm.deviceUA(&candidates[i])) {
			continue
		}
		if !closeInTime(canonicalFirstSeen, firstSeenTime(&candidates[i]), clashMetaAndroidAliasWindow) {
			continue
		}
		candidates[i].IsActive = false
		if err := dm.db.Save(&candidates[i]).Error; err != nil {
			return err
		}
		deactivated = true
	}

	if deactivated {
		deviceCount, _ := CountActiveDevices(dm.db, canonical.SubscriptionID)
		return dm.db.Model(&models.Subscription{}).Where("id = ?", canonical.SubscriptionID).Update("current_devices", deviceCount).Error
	}

	return nil
}

func (dm *DeviceManager) matchSoftware(userAgent, uaLower string) string {
	// Shadowrocket
	if strings.Contains(uaLower, "shadowrocket") {
		return "Shadowrocket"
	}

	// iOS 设备特征识别
	hasIPhoneID := reIPhoneID.MatchString(userAgent)
	if hasIPhoneID && (strings.Contains(uaLower, "cfnetwork") || strings.Contains(uaLower, "darwin")) {
		if strings.Contains(uaLower, "quantumult") {
			return "Quantumult"
		}
		if strings.Contains(uaLower, "surge") {
			return "Surge"
		}
		if strings.Contains(uaLower, "loon") {
			return "Loon"
		}
		if strings.Contains(uaLower, "stash") {
			return "Stash"
		}
		return "Shadowrocket"
	}

	// Windows 客户端
	if strings.Contains(uaLower, "v2rayn") {
		return "v2rayN"
	}
	if strings.Contains(uaLower, "clash for windows") || strings.Contains(uaLower, "clash-windows") {
		return "Clash for Windows"
	}
	if strings.Contains(uaLower, "clash verge") || strings.Contains(uaLower, "clash-verge") {
		return "Clash Verge"
	}

	// Clash Part 系列
	// 路由器和软路由客户端
	if strings.Contains(uaLower, "openwrt") {
		if strings.Contains(uaLower, "clash") {
			return "OpenClash"
		}
		if strings.Contains(uaLower, "passwall") {
			return "PassWall"
		}
		if strings.Contains(uaLower, "ssr+") || strings.Contains(uaLower, "ssrplus") {
			return "SSR Plus+"
		}
		return "OpenWrt"
	}

	// Android 客户端
	if strings.Contains(uaLower, "clash for android") || strings.Contains(uaLower, "clashforandroid") || strings.Contains(uaLower, "cfa") {
		return "Clash for Android"
	}
	if strings.Contains(uaLower, "surfboard") {
		return "Surfboard"
	}
	if strings.Contains(uaLower, "v2rayng") {
		return "v2rayNG"
	}

	// 通用软件识别。这里必须保持有序：一些客户端 UA 会同时带上内核/兼容标识，
	// 例如 HiddifyNext/... like ClashMeta v2ray sing-box，应优先识别真实客户端。
	softwares := []struct {
		key  string
		name string
	}{
		{"moneyfly", "MoneyFly"},
		{"hiddifynext", "Hiddify"},
		{"hiddify-next", "Hiddify"},
		{"hiddify next", "Hiddify"},
		{"hiddify", "Hiddify"},
		{"quantumult", "Quantumult"},
		{"clash meta", "Clash Meta"},
		{"clashmeta", "Clash Meta"},
		{"sing-box", "sing-box"},
		{"karing", "Karing"},
		{"nekobox", "NekoBox"},
		{"shadowsocks", "Shadowsocks"},
		{"clash", "Clash"},
		{"v2ray", "V2Ray"},
		{"xray", "Xray"},
		{"loon", "Loon"},
		{"surge", "Surge"},
		{"stash", "Stash"},
	}

	for _, software := range softwares {
		if strings.Contains(uaLower, software.key) {
			return software.name
		}
	}

	return "Unknown"
}

func (dm *DeviceManager) parseOSInfo(userAgent, uaLower string) map[string]string {
	result := map[string]string{
		"os_name":    "Unknown",
		"os_version": "",
	}

	// 路由器系统识别（优先级最高）
	if strings.Contains(uaLower, "openwrt") {
		result["os_name"] = "OpenWrt"
		if match := reOpenWrt.FindStringSubmatch(userAgent); len(match) > 1 {
			result["os_version"] = match[1]
		}
		return result
	}
	if strings.Contains(uaLower, "routeros") {
		result["os_name"] = "RouterOS"
		if match := reRouterOS.FindStringSubmatch(userAgent); len(match) > 1 {
			result["os_version"] = match[1]
		}
		return result
	}
	if strings.Contains(uaLower, "padavan") {
		result["os_name"] = "Padavan"
		return result
	}
	if strings.Contains(uaLower, "merlin") || strings.Contains(uaLower, "asuswrt") {
		result["os_name"] = "Asuswrt-Merlin"
		return result
	}

	// iOS/iPadOS 识别
	if strings.Contains(uaLower, "iphone") || strings.Contains(uaLower, "ipad") || strings.Contains(uaLower, "ipod") {
		if strings.Contains(uaLower, "ipad") {
			result["os_name"] = "iPadOS"
		} else {
			result["os_name"] = "iOS"
		}
		patterns := []string{
			`OS\s+(\d+)[._](\d+)(?:[._](\d+))?`,          // OS 16_6_1, OS 16.6.1
			`iPhone\s+OS\s+(\d+)[._](\d+)(?:[._](\d+))?`, // iPhone OS 16_6_1
			`Version/(\d+)[._](\d+)(?:[._](\d+))?`,       // Version/16.6.1
			`iOS\s+(\d+)[._](\d+)(?:[._](\d+))?`,         // iOS 16.6.1
		}
		for _, pattern := range patterns {
			if match := regexp.MustCompile(pattern).FindStringSubmatch(userAgent); len(match) > 1 {
				version := match[1] + "." + match[2]
				if len(match) > 3 && match[3] != "" {
					version += "." + match[3]
				}
				result["os_version"] = version
				break
			}
		}
		return result
	}

	// Android 识别
	if strings.Contains(uaLower, "android") {
		result["os_name"] = "Android"
		if match := reAndroidVersion.FindStringSubmatch(userAgent); len(match) > 1 {
			result["os_version"] = match[1]
		}
		return result
	}

	// Windows 识别
	if strings.Contains(uaLower, "windows") {
		result["os_name"] = "Windows"
		if match := reWindowsNT.FindStringSubmatch(userAgent); len(match) > 1 {
			result["os_version"] = match[1]
		}
		return result
	}

	// macOS 识别
	if strings.Contains(uaLower, "macintosh") || strings.Contains(uaLower, "mac os") {
		result["os_name"] = "macOS"
		if match := reMacOSX.FindStringSubmatch(userAgent); len(match) > 1 {
			result["os_version"] = strings.Replace(match[1], "_", ".", -1)
		}
		return result
	}

	// Linux 识别
	if strings.Contains(uaLower, "linux") {
		result["os_name"] = "Linux"
		// 尝试识别具体发行版
		if strings.Contains(uaLower, "ubuntu") {
			result["os_name"] = "Ubuntu"
		} else if strings.Contains(uaLower, "debian") {
			result["os_name"] = "Debian"
		} else if strings.Contains(uaLower, "centos") {
			result["os_name"] = "CentOS"
		} else if strings.Contains(uaLower, "fedora") {
			result["os_name"] = "Fedora"
		}
		return result
	}

	return result
}

func (dm *DeviceManager) inferOSFromSoftware(softwareName string) map[string]string {
	iosSoftware := []string{"shadowrocket", "quantumult", "surge", "loon", "stash", "anx", "anxray", "karing", "kitsunebi", "pharos", "potatso"}
	androidSoftware := []string{"clash for android", "clashandroid", "shadowsocks", "v2rayng", "surfboard"}
	windowsSoftware := []string{"clash for windows", "clash-verge", "clash verge", "v2rayn", "qv2ray", "clash part"}
	macosSoftware := []string{"clash for mac", "clashx", "clashx pro", "surge", "v2rayu"}
	routerSoftware := []string{"openclash", "passwall", "ssr plus+", "ssrplus"}

	swLower := strings.ToLower(softwareName)

	// 路由器软件
	for _, sw := range routerSoftware {
		if strings.Contains(swLower, sw) {
			return map[string]string{"os_name": "OpenWrt", "os_version": ""}
		}
	}

	// iOS 软件
	for _, sw := range iosSoftware {
		if strings.Contains(swLower, sw) {
			return map[string]string{"os_name": "iOS", "os_version": ""}
		}
	}

	// Android 软件
	for _, sw := range androidSoftware {
		if strings.Contains(swLower, sw) {
			return map[string]string{"os_name": "Android", "os_version": ""}
		}
	}

	// Windows 软件
	for _, sw := range windowsSoftware {
		if strings.Contains(swLower, sw) {
			return map[string]string{"os_name": "Windows", "os_version": ""}
		}
	}

	// macOS 软件
	for _, sw := range macosSoftware {
		if strings.Contains(swLower, sw) {
			return map[string]string{"os_name": "macOS", "os_version": ""}
		}
	}

	return nil
}

func (dm *DeviceManager) parseDeviceInfo(userAgent, osName string) map[string]string {
	result := map[string]string{
		"device_model": "",
		"device_brand": "",
	}

	uaLower := strings.ToLower(userAgent)

	if strings.Contains(uaLower, "iphone") || strings.Contains(uaLower, "ipad") || strings.Contains(uaLower, "ipod") {
		result["device_brand"] = "Apple"

		iphoneModelMap := map[string]string{
			"iPhone14,2": "iPhone 13 Pro",
			"iPhone14,3": "iPhone 13 Pro Max",
			"iPhone14,4": "iPhone 13 mini",
			"iPhone14,5": "iPhone 13",
			"iPhone15,2": "iPhone 14 Pro",
			"iPhone15,3": "iPhone 14 Pro Max",
			"iPhone15,4": "iPhone 14",
			"iPhone15,5": "iPhone 14 Plus",
			"iPhone16,1": "iPhone 15 Pro",
			"iPhone16,2": "iPhone 15 Pro Max",
			"iPhone16,3": "iPhone 15",
			"iPhone16,4": "iPhone 15 Plus",
		}

		if match := reIPhoneModel.FindStringSubmatch(userAgent); len(match) > 1 {
			modelID := "iPhone" + match[1]
			if modelName, exists := iphoneModelMap[modelID]; exists {
				result["device_model"] = modelName
			} else {
				result["device_model"] = fmt.Sprintf("iPhone %s", strings.Replace(match[1], ",", ".", -1))
			}
		} else if match := reIPhoneProMax.FindStringSubmatch(userAgent); len(match) > 1 {
			result["device_model"] = fmt.Sprintf("iPhone %s Pro Max", match[1])
		} else if match := reIPhonePro.FindStringSubmatch(userAgent); len(match) > 1 {
			result["device_model"] = fmt.Sprintf("iPhone %s Pro", match[1])
		} else if match := reIPhoneMini.FindStringSubmatch(userAgent); len(match) > 1 {
			result["device_model"] = fmt.Sprintf("iPhone %s mini", match[1])
		} else if match := reIPhonePlain.FindStringSubmatch(userAgent); len(match) > 1 {
			result["device_model"] = fmt.Sprintf("iPhone %s", match[1])
		}

		if match := reIPadModel.FindStringSubmatch(userAgent); len(match) > 1 {
			modelID := "iPad" + match[1]
			// iPad 型号映射表
			iPadModelMap := map[string]string{
				"iPad13,1":  "iPad Air (第5代)",
				"iPad13,2":  "iPad Air (第5代)",
				"iPad13,4":  "iPad Pro 11英寸 (第4代)",
				"iPad13,5":  "iPad Pro 11英寸 (第4代)",
				"iPad13,6":  "iPad Pro 11英寸 (第4代)",
				"iPad13,7":  "iPad Pro 11英寸 (第4代)",
				"iPad13,8":  "iPad Pro 12.9英寸 (第6代)",
				"iPad13,9":  "iPad Pro 12.9英寸 (第6代)",
				"iPad13,10": "iPad Pro 12.9英寸 (第6代)",
				"iPad13,11": "iPad Pro 12.9英寸 (第6代)",
				"iPad13,16": "iPad Air (第5代)",
				"iPad13,17": "iPad Air (第5代)",
				"iPad13,18": "iPad (第10代)",
				"iPad13,19": "iPad (第10代)",
				"iPad14,1":  "iPad mini (第6代)",
				"iPad14,2":  "iPad mini (第6代)",
				"iPad14,3":  "iPad Pro 11英寸 (第5代)",
				"iPad14,4":  "iPad Pro 11英寸 (第5代)",
				"iPad14,5":  "iPad Pro 12.9英寸 (第7代)",
				"iPad14,6":  "iPad Pro 12.9英寸 (第7代)",
			}
			if modelName, exists := iPadModelMap[modelID]; exists {
				result["device_model"] = modelName
			} else {
				result["device_model"] = fmt.Sprintf("iPad %s", strings.Replace(match[1], ",", ".", -1))
			}
		} else if strings.Contains(userAgent, "iPad Pro") {
			if strings.Contains(userAgent, "12.9") {
				result["device_model"] = "iPad Pro 12.9英寸"
			} else if strings.Contains(userAgent, "11") {
				result["device_model"] = "iPad Pro 11英寸"
			} else {
				result["device_model"] = "iPad Pro"
			}
		} else if strings.Contains(userAgent, "iPad Air") {
			result["device_model"] = "iPad Air"
		} else if strings.Contains(userAgent, "iPad mini") {
			result["device_model"] = "iPad mini"
		} else if match := reIPadPlain.FindStringSubmatch(userAgent); len(match) > 0 {
			result["device_model"] = "iPad"
		}

		return result
	}

	if strings.Contains(uaLower, "android") {
		if match := reAndroidBuild.FindStringSubmatch(userAgent); len(match) > 1 {
			name := strings.TrimSpace(match[1])
			result["device_model"] = name
			brands := map[string][]string{
				"Samsung":  {"samsung", "galaxy", "sm-"},
				"Huawei":   {"huawei", "honor", "hma-", "ane-", "vog-", "ele-"},
				"Xiaomi":   {"xiaomi", "redmi", "mi ", "poco"},
				"OPPO":     {"oppo", "oneplus", "realme"},
				"vivo":     {"vivo", "iqoo"},
				"Meizu":    {"meizu", "m1"},
				"Lenovo":   {"lenovo", "zuk"},
				"Motorola": {"motorola", "moto"},
				"Sony":     {"sony", "xperia"},
				"LG":       {"lg-", "lge"},
				"Google":   {"pixel", "nexus"},
				"OnePlus":  {"oneplus"},
				"Realme":   {"realme"},
				"Nothing":  {"nothing"},
			}
			nameLower := strings.ToLower(name)
			for brand, keywords := range brands {
				for _, keyword := range keywords {
					if strings.Contains(nameLower, keyword) {
						result["device_brand"] = brand
						return result
					}
				}
			}
		}
	}

	return result
}

func (dm *DeviceManager) inferDeviceFromSoftware(softwareName string) map[string]string {
	iosSoftware := []string{"shadowrocket", "quantumult", "surge", "loon", "stash", "anx", "anxray", "karing", "kitsunebi", "pharos", "potatso"}
	swLower := strings.ToLower(softwareName)
	for _, sw := range iosSoftware {
		if strings.Contains(swLower, sw) {
			return map[string]string{"device_brand": "Apple", "device_model": ""}
		}
	}
	return nil
}

func (dm *DeviceManager) parseVersion(userAgent string) string {
	patterns := []string{
		`(\d+\.\d+\.\d+)`,
		`(\d+\.\d+)`,
		`v(\d+\.\d+\.\d+)`,
		`version\s*(\d+\.\d+\.\d+)`,
		`(\d+\.\d+\.\d+\.\d+)`,
	}

	for _, pattern := range patterns {
		if match := regexp.MustCompile(pattern).FindStringSubmatch(userAgent); len(match) > 1 {
			return match[1]
		}
	}
	return ""
}

func (dm *DeviceManager) determineDeviceType(userAgent string, info *DeviceInfo) string {
	uaLower := strings.ToLower(userAgent)
	osName := strings.ToLower(info.OSName)
	swName := strings.ToLower(info.SoftwareName)

	// 路由器和软路由识别（优先级最高）
	if dm.isRouter(userAgent, uaLower, osName) {
		return "router"
	}

	// 电视盒子识别
	if dm.isTVBox(userAgent, uaLower, osName) {
		return "tv_box"
	}

	// iPad 识别（区分不同型号）
	if strings.Contains(osName, "ipad") || strings.Contains(uaLower, "ipad") {
		return "tablet"
	}

	// 手机识别
	if strings.Contains(osName, "ios") || strings.Contains(osName, "android") || strings.Contains(uaLower, "iphone") {
		return "mobile"
	}

	// 桌面系统识别
	if strings.Contains(osName, "windows") || strings.Contains(osName, "macos") {
		return "desktop"
	}

	// Linux 系统需要进一步判断（可能是桌面、服务器或路由器）
	if strings.Contains(osName, "linux") {
		// 如果是常见的桌面 Linux 发行版
		if strings.Contains(uaLower, "ubuntu") || strings.Contains(uaLower, "debian") ||
			strings.Contains(uaLower, "fedora") || strings.Contains(uaLower, "arch") {
			return "desktop"
		}
		// 如果有桌面浏览器特征
		if strings.Contains(uaLower, "chrome") || strings.Contains(uaLower, "firefox") ||
			strings.Contains(uaLower, "electron") {
			return "desktop"
		}
		// 否则可能是服务器或路由器
		return "server"
	}

	// 基于软件名称推断设备类型
	if strings.Contains(swName, "shadowrocket") || strings.Contains(swName, "quantumult") || strings.Contains(swName, "surge") {
		if strings.Contains(uaLower, "ipad") {
			return "tablet"
		}
		return "mobile"
	}
	if strings.Contains(swName, "clash part") || strings.Contains(swName, "clash for windows") || strings.Contains(swName, "v2rayn") {
		return "desktop"
	}

	return "unknown"
}

// isRouter 判断是否为路由器或软路由
func (dm *DeviceManager) isRouter(userAgent, uaLower, osName string) bool {
	// OpenWrt 路由器系统
	if strings.Contains(uaLower, "openwrt") {
		return true
	}

	// RouterOS (MikroTik)
	if strings.Contains(uaLower, "routeros") || strings.Contains(uaLower, "mikrotik") {
		return true
	}

	// Padavan 固件
	if strings.Contains(uaLower, "padavan") {
		return true
	}

	// Merlin 固件 (华硕路由器)
	if strings.Contains(uaLower, "merlin") || strings.Contains(uaLower, "asuswrt") {
		return true
	}

	// DD-WRT 固件
	if strings.Contains(uaLower, "dd-wrt") {
		return true
	}

	// Tomato 固件
	if strings.Contains(uaLower, "tomato") {
		return true
	}

	// iKuai 爱快路由
	if strings.Contains(uaLower, "ikuai") {
		return true
	}

	// 软路由常见特征：clash 系列 + mips/arm/aarch64 架构
	if strings.Contains(uaLower, "clash") &&
		(strings.Contains(uaLower, "mips") || strings.Contains(uaLower, "arm") ||
			strings.Contains(uaLower, "aarch64") || strings.Contains(uaLower, "armv7")) {
		return true
	}

	// 常见路由器品牌特征
	routerBrands := []string{"netgear", "tp-link", "asus router", "xiaomi router", "huawei router"}
	for _, brand := range routerBrands {
		if strings.Contains(uaLower, brand) {
			return true
		}
	}

	return false
}

// isTVBox 判断是否为电视盒子
func (dm *DeviceManager) isTVBox(userAgent, uaLower, osName string) bool {
	// Android TV
	if strings.Contains(uaLower, "android tv") || strings.Contains(uaLower, "androidtv") {
		return true
	}

	// Apple TV
	if strings.Contains(uaLower, "apple tv") || strings.Contains(uaLower, "appletv") {
		return true
	}

	// 小米盒子
	if strings.Contains(uaLower, "mi box") || strings.Contains(uaLower, "mibox") {
		return true
	}

	// Fire TV
	if strings.Contains(uaLower, "fire tv") || strings.Contains(uaLower, "firetv") {
		return true
	}

	// Nvidia Shield
	if strings.Contains(uaLower, "shield") && strings.Contains(uaLower, "android") {
		return true
	}

	return false
}

func (dm *DeviceManager) generateDeviceName(info *DeviceInfo) string {
	parts := []string{}

	if info.SoftwareName != "Unknown" {
		parts = append(parts, info.SoftwareName)
	}

	if info.DeviceModel != "" {
		parts = append(parts, info.DeviceModel)
	} else if info.DeviceBrand != "" {
		parts = append(parts, info.DeviceBrand)
	}

	if info.OSName != "Unknown" {
		osName := info.OSName
		if info.OSVersion != "" {
			osName += " " + info.OSVersion
		}
		parts = append(parts, osName)
	}

	if info.SoftwareVersion != "" {
		parts = append(parts, "v"+info.SoftwareVersion)
	}

	if len(parts) > 0 {
		return strings.Join(parts, " - ")
	}
	return "Unknown Device"
}

func (dm *DeviceManager) refreshDeviceInfo(device *models.Device, info *DeviceInfo) {
	if info.DeviceName != "Unknown Device" {
		device.DeviceName = &info.DeviceName
	}
	if info.DeviceType != "unknown" {
		device.DeviceType = &info.DeviceType
	}
	if info.DeviceModel != "" {
		device.DeviceModel = &info.DeviceModel
	}
	if info.DeviceBrand != "" {
		device.DeviceBrand = &info.DeviceBrand
	}
	if info.SoftwareName != "Unknown" {
		device.SoftwareName = &info.SoftwareName
	}
	if info.SoftwareVersion != "" {
		device.SoftwareVersion = &info.SoftwareVersion
	}
	if info.OSName != "Unknown" {
		device.OSName = &info.OSName
	}
	if info.OSVersion != "" {
		device.OSVersion = &info.OSVersion
	}
}

func (dm *DeviceManager) GenerateDeviceHash(userAgent, ipAddress, deviceID string) string {
	return dm.GenerateDeviceHashWithHeaders(userAgent, ipAddress, deviceID, nil)
}

// fingerprintKeyOf 稳定指纹：软件名 + 系统名 + 机型 + 品牌（四者必须齐备且非 Unknown）。
// **不含版本号** —— 见 GenerateDeviceHashWithHeaders 的说明。
func fingerprintKeyOf(sw, osName, model, brand string) (string, bool) {
	sw = strings.TrimSpace(sw)
	osName = strings.TrimSpace(osName)
	model = strings.TrimSpace(model)
	brand = strings.TrimSpace(brand)
	if sw == "" || osName == "" || model == "" || brand == "" {
		return "", false
	}
	if strings.EqualFold(sw, "Unknown") || strings.EqualFold(osName, "Unknown") {
		return "", false
	}
	if !IsSpecificDeviceModel(model) {
		return "", false
	}
	return strings.ToLower(sw) + "|" + strings.ToLower(osName) + "|" +
		strings.ToLower(model) + "|" + strings.ToLower(brand), true
}

// genericDeviceModels 无法区分「哪一台设备」的机型值（UA 解析只给到品类级别）。
// 用它们做指纹合并会把同一用户的多台同品类设备误判成一台（每个都删掉，等于
// 无故剥夺用户的设备名额），所以这类值一律不参与指纹匹配/合并。
var genericDeviceModels = map[string]bool{
	"iphone": true, "ipad": true, "ipod": true, "iphone simulator": true,
	"android": true, "android phone": true, "phone": true, "mobile": true,
	"windows": true, "windows nt": true, "pc": true, "desktop": true,
	"macintosh": true, "mac": true, "macos": true, "linux": true,
	"unknown": true, "unknown device": true, "generic": true, "web": true,
}

// IsSpecificDeviceModel 机型字段是否**足够具体**（能唯一指认一台设备）。
//
// 两类不算具体（线上真实数据踩过）：
//  1. 纯品类名：`iPhone` / `Android` / `PC` / `Windows` —— 同用户多台同型号设备会撞在一起；
//  2. 被解析串味的「系统版本」：如 `iPhone 18.1`（Shadowrocket 之类 UA 的机型位拿了
//     OS 版本）—— 同一台手机升级系统就会变，把它当身份会把真实设备误合并。
func IsSpecificDeviceModel(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" || genericDeviceModels[m] {
		return false
	}
	// 形如 "iphone 18.1" / "android 13" / "windows 10.0"：品类 + 纯版本号 → 不具体
	if matched, _ := regexp.MatchString(`^(iphone|ipad|ipod|android|windows|macos|mac|harmonyos|ios)[\s_-]?[0-9]+(\.[0-9]+)*$`, m); matched {
		return false
	}
	return true
}

// stableHashFromFeatures 由稳定特征（无客户端设备 ID 时）计算哈希，
// 必须与 GenerateDeviceHashWithHeaders 的特征顺序完全一致。
func stableHashFromFeatures(sw, osName, model, brand string) string {
	features := []string{}
	if sw != "" && !strings.EqualFold(sw, "Unknown") {
		features = append(features, "software:"+sw)
	}
	if osName != "" && !strings.EqualFold(osName, "Unknown") {
		features = append(features, "os:"+osName)
	}
	if model != "" {
		features = append(features, "model:"+model)
	}
	if brand != "" {
		features = append(features, "brand:"+brand)
	}
	deviceString := strings.Join(features, "|")
	if deviceString == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(deviceString))
	return hex.EncodeToString(hash[:])
}

// isLegacyHashRow 该行的 device_hash 是否还是旧算法（含版本号）写下的值。
// 判断方式：用该行自己的字段算一遍稳定哈希，和存的值不一致 → 说明是历史行。
// 只有历史行才允许被「稳定指纹」兜底认领/命中，避免对已是稳定身份的行做模糊匹配。
func isLegacyHashRow(d *models.Device) bool {
	if d.DeviceHash == nil || *d.DeviceHash == "" {
		return true
	}
	get := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	expected := stableHashFromFeatures(get(d.SoftwareName), get(d.OSName), get(d.DeviceModel), get(d.DeviceBrand))
	if expected == "" {
		return false // 字段不全，无法判断 → 不当作历史行
	}
	return expected != *d.DeviceHash
}

// GenerateDeviceHashWithHeaders 计算设备身份哈希（**稳定身份**）。
//
// 身份来源优先级：
//  1. 显式设备 ID（`deviceID` 参数，或客户端请求头 `X-MF-Device-Id`）——
//     MoneyFly 客户端会带安装级唯一 ID：既唯一（同型号两台手机不会互相顶号），
//     又稳定（升级 App/系统都不变，只有重装才变）。
//  2. 退化到 UA/请求头解析出的**稳定特征**：软件名 + 系统名 + 机型 + 品牌。
//
// ⚠️ 特征集里**刻意不含 App 版本号与系统版本号**（2026-09-21 修复）：
// 旧算法把 `version:` / `os_version:` 也算进身份，于是用户每次升级 App 或系统，
// 设备哈希就变了 —— 同一台手机会以「新设备」重复注册，旧行变成永久占用名额的
// 幽灵设备；被踢下线（删除设备）的判定也会随之漂移（升级即可绕过，或者反过来
// 永远命中旧行）。升级不应该改变"我是哪台设备"。
func (dm *DeviceManager) GenerateDeviceHashWithHeaders(userAgent, ipAddress, deviceID string, headers map[string]string) string {
	if deviceID == "" && headers != nil {
		deviceID = headers["X-MF-Device-Id"]
	}
	if strings.TrimSpace(deviceID) != "" {
		hash := sha256.Sum256([]byte("device_id:" + strings.TrimSpace(deviceID)))
		return hex.EncodeToString(hash[:])
	}

	info := dm.ParseUserAgentWithHeaders(userAgent, headers)
	features := []string{}

	if info.SoftwareName != "Unknown" {
		features = append(features, "software:"+info.SoftwareName)
	}

	if info.OSName != "Unknown" {
		features = append(features, "os:"+info.OSName)
	}

	if info.DeviceModel != "" {
		features = append(features, "model:"+info.DeviceModel)
	}
	if info.DeviceBrand != "" {
		features = append(features, "brand:"+info.DeviceBrand)
	}

	// X-MF-* 头补充（MoneyFly 客户端携带的设备详情，增强设备区分度）
	if headers != nil {
		if v := headers["X-MF-Device-Model"]; v != "" && info.DeviceModel == "" {
			features = append(features, "mf_model:"+v)
		}
		if v := headers["X-MF-OS"]; v != "" && info.OSName == "Unknown" {
			features = append(features, "mf_os:"+v)
		}
	}

	deviceString := strings.Join(features, "|")
	if deviceString == "" {
		deviceString = userAgent
	}

	hash := sha256.Sum256([]byte(deviceString))
	return hex.EncodeToString(hash[:])
}

// GenerateLegacyDeviceHash 旧算法哈希（含 App 版本号 / 系统版本号）。
//
// 只用于**存量迁移**：库里已有的行是按旧算法写的，客户端换成稳定身份后第一请求
// 必然对不上；用旧算法再算一次能精确认领「就是这台设备」的那一行，把它的
// device_hash 改写成新的稳定值（自愈，不新增幽灵设备、也不丢设备备注）。
// 新代码不要再用它写库。
func (dm *DeviceManager) GenerateLegacyDeviceHash(userAgent string, headers map[string]string) string {
	info := dm.ParseUserAgentWithHeaders(userAgent, headers)
	features := []string{}

	if info.SoftwareName != "Unknown" {
		features = append(features, "software:"+info.SoftwareName)
		if info.SoftwareVersion != "" {
			features = append(features, "version:"+info.SoftwareVersion)
		}
	}
	if info.OSName != "Unknown" {
		features = append(features, "os:"+info.OSName)
		if info.OSVersion != "" {
			features = append(features, "os_version:"+info.OSVersion)
		}
	}
	if info.DeviceModel != "" {
		features = append(features, "model:"+info.DeviceModel)
	}
	if info.DeviceBrand != "" {
		features = append(features, "brand:"+info.DeviceBrand)
	}
	if headers != nil {
		if v := headers["X-MF-Device-Model"]; v != "" && info.DeviceModel == "" {
			features = append(features, "mf_model:"+v)
		}
		if v := headers["X-MF-OS"]; v != "" && info.OSName == "Unknown" {
			features = append(features, "mf_os:"+v)
		}
	}

	deviceString := strings.Join(features, "|")
	if deviceString == "" {
		deviceString = userAgent
	}
	hash := sha256.Sum256([]byte(deviceString))
	return hex.EncodeToString(hash[:])
}

func (dm *DeviceManager) updateExistingDeviceAccess(device *models.Device, deviceInfo *DeviceInfo, deviceHash, userAgent, ipAddress, subscriptionType string) error {
	now := utils.GetBeijingTime()
	device.IPAddress = &ipAddress
	device.LastAccess = now
	device.LastSeen = &now
	device.AccessCount++
	device.IsActive = true

	// 查询并更新位置信息（使用缓存）
	if ipAddress != "" {
		location := geoip.GetLocationWithCache(ipAddress)
		if location.Valid && location.String != "" {
			device.Location = &location.String
		}
	}

	if subscriptionType != "" {
		subscriptionTypeStr := subscriptionType
		device.SubscriptionType = &subscriptionTypeStr
	}

	existingClashMetaAndroid := dm.isClashMetaAndroidDevice(device)
	if dm.IsGenericClashWindowsUA(userAgent) && existingClashMetaAndroid {
		if err := dm.db.Save(device).Error; err != nil {
			return err
		}
		return dm.deactivateClashMetaAndroidAliasDuplicates(device, ipAddress)
	}

	device.DeviceHash = &deviceHash
	device.DeviceFingerprint = deviceHash
	device.DeviceUA = &userAgent
	device.UserAgent = &userAgent
	dm.refreshDeviceInfo(device, deviceInfo)

	if err := dm.db.Save(device).Error; err != nil {
		return err
	}
	return dm.deactivateClashMetaAndroidAliasDuplicates(device, ipAddress)
}

// RecordDeviceAccessWithHeaders 记录设备访问（带 X-MF-* 头，MoneyFly 客户端专用）
func (dm *DeviceManager) RecordDeviceAccessWithHeaders(subscriptionID uint, userID uint, userAgent, ipAddress, subscriptionType string, headers map[string]string) (*models.Device, error) {
	deviceInfo := dm.ParseUserAgentWithHeaders(userAgent, headers)

	if deviceInfo.SoftwareName == "Unknown" {
		uaLower := strings.ToLower(userAgent)
		browserKeywords := []string{
			"mozilla", "chrome", "safari", "firefox", "edge", "opera", "msie",
			"webkit", "gecko", "trident", "presto", "blink",
		}
		isBrowser := false
		for _, keyword := range browserKeywords {
			if strings.Contains(uaLower, keyword) {
				subscriptionSoftwareKeywords := []string{
					"shadowrocket", "quantumult", "surge", "loon", "stash",
					"v2rayn", "clash", "hiddify", "v2ray", "moneyfly",
				}
				hasSubscriptionSoftware := false
				for _, swKeyword := range subscriptionSoftwareKeywords {
					if strings.Contains(uaLower, swKeyword) {
						hasSubscriptionSoftware = true
						break
					}
				}
				if !hasSubscriptionSoftware {
					isBrowser = true
					break
				}
			}
		}
		if isBrowser {
			return nil, nil
		}
	}

	deviceHash := dm.GenerateDeviceHashWithHeaders(userAgent, ipAddress, "", headers)

	if existingDevice, exists, err := dm.FindExistingDeviceWithHeaders(subscriptionID, userAgent, ipAddress, headers); err != nil {
		return nil, err
	} else if exists {
		if err := dm.updateExistingDeviceAccess(existingDevice, deviceInfo, deviceHash, userAgent, ipAddress, subscriptionType); err != nil {
			return nil, err
		}
		return existingDevice, nil
	}

	{
		now := utils.GetBeijingTime()
		userIDInt64 := utils.MustSafeUintToInt64(userID)
		subscriptionTypeStr := subscriptionType

		// 查询位置信息（使用缓存）
		var locationStr *string
		if ipAddress != "" {
			location := geoip.GetLocationWithCache(ipAddress)
			if location.Valid && location.String != "" {
				locationStr = &location.String
			}
		}

		device := models.Device{
			UserID:            &userIDInt64,
			SubscriptionID:    subscriptionID,
			DeviceFingerprint: deviceHash,
			DeviceHash:        &deviceHash,
			DeviceUA:          &userAgent,
			DeviceName:        &deviceInfo.DeviceName,
			DeviceType:        &deviceInfo.DeviceType,
			DeviceModel:       &deviceInfo.DeviceModel,
			DeviceBrand:       &deviceInfo.DeviceBrand,
			IPAddress:         &ipAddress,
			Location:          locationStr,
			UserAgent:         &userAgent,
			SoftwareName:      &deviceInfo.SoftwareName,
			SoftwareVersion:   &deviceInfo.SoftwareVersion,
			OSName:            &deviceInfo.OSName,
			OSVersion:         &deviceInfo.OSVersion,
			SubscriptionType:  &subscriptionTypeStr,
			IsActive:          true,
			IsAllowed:         true,
			FirstSeen:         &now,
			LastAccess:        now,
			LastSeen:          &now,
			AccessCount:       1,
		}

		if err := dm.db.Create(&device).Error; err != nil {
			return nil, err
		}

		deviceCount, _ := CountActiveDevices(dm.db, subscriptionID)
		dm.db.Model(&models.Subscription{}).Where("id = ?", subscriptionID).Update("current_devices", deviceCount)

		return &device, nil
	}
}

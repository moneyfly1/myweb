package device

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

// DeviceManager 设备管理器
type DeviceManager struct {
	db *gorm.DB
}

// NewDeviceManager 创建设备管理器
func NewDeviceManager() *DeviceManager {
	return &DeviceManager{
		db: database.GetDB(),
	}
}

// DeviceInfo 设备信息
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

// ParseUserAgent 解析 User-Agent
func (dm *DeviceManager) ParseUserAgent(userAgent string) *DeviceInfo {
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

	// 识别软件
	info.SoftwareName = dm.matchSoftware(userAgent, uaLower)

	// 解析操作系统
	osInfo := dm.parseOSInfo(userAgent, uaLower)
	info.OSName = osInfo["os_name"]
	info.OSVersion = osInfo["os_version"]

	// 如果 OS 未知，从软件推断
	if info.OSName == "Unknown" && info.SoftwareName != "Unknown" {
		inferredOS := dm.inferOSFromSoftware(info.SoftwareName)
		if inferredOS != nil {
			info.OSName = inferredOS["os_name"]
			info.OSVersion = inferredOS["os_version"]
		}
	}

	// 解析设备信息
	deviceInfo := dm.parseDeviceInfo(userAgent, info.OSName)
	info.DeviceModel = deviceInfo["device_model"]
	info.DeviceBrand = deviceInfo["device_brand"]

	// 从软件推断设备
	if info.DeviceModel == "" && info.SoftwareName != "Unknown" {
		inferredDevice := dm.inferDeviceFromSoftware(info.SoftwareName)
		if inferredDevice != nil {
			info.DeviceBrand = inferredDevice["device_brand"]
		}
	}

	// 解析版本
	info.SoftwareVersion = dm.parseVersion(userAgent)

	// 确定设备类型
	info.DeviceType = dm.determineDeviceType(userAgent, info)

	// 生成设备名称
	info.DeviceName = dm.generateDeviceName(info)

	return info
}

// matchSoftware 匹配软件
func (dm *DeviceManager) matchSoftware(userAgent, uaLower string) string {
	// Shadowrocket
	if strings.Contains(uaLower, "shadowrocket") {
		return "Shadowrocket"
	}

	// iOS 代理应用
	hasIPhoneID := regexp.MustCompile(`iPhone\d+,\d+`).MatchString(userAgent)
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

	// v2rayN
	if strings.Contains(uaLower, "v2rayn") {
		return "v2rayN"
	}

	// 其他常见软件
	softwares := map[string]string{
		"quantumult": "Quantumult",
		"hiddify":    "Hiddify",
		"clash":      "Clash",
		"v2ray":      "V2Ray",
		"loon":       "Loon",
		"surge":      "Surge",
	}

	for key, name := range softwares {
		if strings.Contains(uaLower, key) {
			return name
		}
	}

	return "Unknown"
}

// parseOSInfo 解析操作系统信息
func (dm *DeviceManager) parseOSInfo(userAgent, uaLower string) map[string]string {
	result := map[string]string{
		"os_name":    "Unknown",
		"os_version": "",
	}

	// iOS
	if strings.Contains(uaLower, "iphone") || strings.Contains(uaLower, "ipad") {
		result["os_name"] = "iOS"
		if match := regexp.MustCompile(`OS\s+(\d+[._]\d+)`).FindStringSubmatch(userAgent); len(match) > 1 {
			result["os_version"] = strings.Replace(match[1], "_", ".", -1)
		}
		return result
	}

	// Android
	if strings.Contains(uaLower, "android") {
		result["os_name"] = "Android"
		if match := regexp.MustCompile(`Android\s+(\d+[.\d]*)`).FindStringSubmatch(userAgent); len(match) > 1 {
			result["os_version"] = match[1]
		}
		return result
	}

	// Windows
	if strings.Contains(uaLower, "windows") {
		result["os_name"] = "Windows"
		if match := regexp.MustCompile(`Windows\s+NT\s+(\d+\.\d+)`).FindStringSubmatch(userAgent); len(match) > 1 {
			result["os_version"] = match[1]
		}
		return result
	}

	// macOS
	if strings.Contains(uaLower, "macintosh") || strings.Contains(uaLower, "mac os") {
		result["os_name"] = "macOS"
		if match := regexp.MustCompile(`Mac OS X\s+(\d+[._]\d+)`).FindStringSubmatch(userAgent); len(match) > 1 {
			result["os_version"] = strings.Replace(match[1], "_", ".", -1)
		}
		return result
	}

	// Linux
	if strings.Contains(uaLower, "linux") {
		result["os_name"] = "Linux"
		return result
	}

	return result
}

// inferOSFromSoftware 从软件推断操作系统
func (dm *DeviceManager) inferOSFromSoftware(softwareName string) map[string]string {
	iosSoftware := []string{"shadowrocket", "quantumult", "surge", "loon", "stash", "anx", "anxray", "karing", "kitsunebi", "pharos", "potatso"}
	androidSoftware := []string{"clash for android", "clashandroid", "shadowsocks", "v2rayng"}
	windowsSoftware := []string{"clash for windows", "clash-verge", "v2rayn", "qv2ray"}
	macosSoftware := []string{"clash for mac", "clashx", "clashx pro", "surge", "v2rayu"}

	swLower := strings.ToLower(softwareName)
	for _, sw := range iosSoftware {
		if strings.Contains(swLower, sw) {
			return map[string]string{"os_name": "iOS", "os_version": ""}
		}
	}
	for _, sw := range androidSoftware {
		if strings.Contains(swLower, sw) {
			return map[string]string{"os_name": "Android", "os_version": ""}
		}
	}
	for _, sw := range windowsSoftware {
		if strings.Contains(swLower, sw) {
			return map[string]string{"os_name": "Windows", "os_version": ""}
		}
	}
	for _, sw := range macosSoftware {
		if strings.Contains(swLower, sw) {
			return map[string]string{"os_name": "macOS", "os_version": ""}
		}
	}
	return nil
}

// parseDeviceInfo 解析设备信息
func (dm *DeviceManager) parseDeviceInfo(userAgent, osName string) map[string]string {
	result := map[string]string{
		"device_model": "",
		"device_brand": "",
	}

	uaLower := strings.ToLower(userAgent)

	// Apple 设备
	if strings.Contains(uaLower, "iphone") || strings.Contains(uaLower, "ipad") {
		result["device_brand"] = "Apple"
		if match := regexp.MustCompile(`iPhone(\d+,\d+)`).FindStringSubmatch(userAgent); len(match) > 1 {
			result["device_model"] = fmt.Sprintf("iPhone %s", strings.Replace(match[1], ",", ".", -1))
		} else if match := regexp.MustCompile(`iPad(\d+,\d+)`).FindStringSubmatch(userAgent); len(match) > 1 {
			result["device_model"] = fmt.Sprintf("iPad %s", strings.Replace(match[1], ",", ".", -1))
		}
		return result
	}

	// Android 设备
	if strings.Contains(uaLower, "android") {
		if match := regexp.MustCompile(`;\s*([^;]+)\s*build`).FindStringSubmatch(userAgent); len(match) > 1 {
			name := strings.TrimSpace(match[1])
			result["device_model"] = name
			// 识别品牌
			brands := map[string][]string{
				"Samsung": {"samsung", "galaxy"},
				"Huawei":  {"huawei", "honor"},
				"Xiaomi":  {"xiaomi", "redmi", "mi "},
				"OPPO":    {"oppo", "oneplus"},
				"vivo":    {"vivo", "iqoo"},
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

// inferDeviceFromSoftware 从软件推断设备
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

// parseVersion 解析版本号
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

// determineDeviceType 确定设备类型
func (dm *DeviceManager) determineDeviceType(userAgent string, info *DeviceInfo) string {
	uaLower := strings.ToLower(userAgent)
	osName := strings.ToLower(info.OSName)
	swName := strings.ToLower(info.SoftwareName)

	if strings.Contains(osName, "ipad") || strings.Contains(uaLower, "ipad") {
		return "tablet"
	}
	if strings.Contains(osName, "ios") || strings.Contains(osName, "android") || strings.Contains(uaLower, "iphone") {
		return "mobile"
	}
	if strings.Contains(osName, "windows") || strings.Contains(osName, "macos") || strings.Contains(osName, "linux") {
		return "desktop"
	}

	// 从软件推断
	if strings.Contains(swName, "shadowrocket") || strings.Contains(swName, "quantumult") || strings.Contains(swName, "surge") {
		if strings.Contains(uaLower, "ipad") {
			return "tablet"
		}
		return "mobile"
	}
	if strings.Contains(swName, "clash for windows") || strings.Contains(swName, "v2rayn") {
		return "desktop"
	}

	return "unknown"
}

// generateDeviceName 生成设备名称
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

// GenerateDeviceHash 生成设备哈希
func (dm *DeviceManager) GenerateDeviceHash(userAgent, ipAddress, deviceID string) string {
	if deviceID != "" {
		hash := sha256.Sum256([]byte("device_id:" + strings.TrimSpace(deviceID)))
		return hex.EncodeToString(hash[:])
	}

	info := dm.ParseUserAgent(userAgent)
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

	deviceString := strings.Join(features, "|")
	if deviceString == "" {
		deviceString = userAgent
	}

	hash := sha256.Sum256([]byte(deviceString))
	return hex.EncodeToString(hash[:])
}

// RecordDeviceAccess 记录设备访问
func (dm *DeviceManager) RecordDeviceAccess(subscriptionID uint, userID uint, userAgent, ipAddress, subscriptionType string) (*models.Device, error) {
	// 解析设备信息
	deviceInfo := dm.ParseUserAgent(userAgent)
	deviceHash := dm.GenerateDeviceHash(userAgent, ipAddress, "")

	// 查找现有设备
	var existingDevice models.Device
	err := dm.db.Where("device_hash = ? AND subscription_id = ?", deviceHash, subscriptionID).First(&existingDevice).Error

	if err == nil {
		// 更新现有设备
		now := utils.GetBeijingTime()
		existingDevice.LastAccess = now
		existingDevice.LastSeen = database.NullTime(now)
		existingDevice.AccessCount++
		existingDevice.IPAddress = database.NullString(ipAddress)
		existingDevice.UserAgent = database.NullString(userAgent)

		// 更新设备信息（如果之前没有）
		if !existingDevice.DeviceName.Valid || existingDevice.DeviceName.String == "" {
			existingDevice.DeviceName = database.NullString(deviceInfo.DeviceName)
		}
		if !existingDevice.DeviceType.Valid || existingDevice.DeviceType.String == "" {
			existingDevice.DeviceType = database.NullString(deviceInfo.DeviceType)
		}
		if !existingDevice.DeviceModel.Valid || existingDevice.DeviceModel.String == "" {
			existingDevice.DeviceModel = database.NullString(deviceInfo.DeviceModel)
		}
		if !existingDevice.DeviceBrand.Valid || existingDevice.DeviceBrand.String == "" {
			existingDevice.DeviceBrand = database.NullString(deviceInfo.DeviceBrand)
		}
		if !existingDevice.SoftwareName.Valid || existingDevice.SoftwareName.String == "" {
			existingDevice.SoftwareName = database.NullString(deviceInfo.SoftwareName)
		}
		if !existingDevice.SoftwareVersion.Valid || existingDevice.SoftwareVersion.String == "" {
			existingDevice.SoftwareVersion = database.NullString(deviceInfo.SoftwareVersion)
		}
		if !existingDevice.OSName.Valid || existingDevice.OSName.String == "" {
			existingDevice.OSName = database.NullString(deviceInfo.OSName)
		}
		if !existingDevice.OSVersion.Valid || existingDevice.OSVersion.String == "" {
			existingDevice.OSVersion = database.NullString(deviceInfo.OSVersion)
		}

		if err := dm.db.Save(&existingDevice).Error; err != nil {
			return nil, err
		}
		return &existingDevice, nil
	} else if err == gorm.ErrRecordNotFound {
		// 创建新设备
		now := utils.GetBeijingTime()
		device := models.Device{
			UserID:            database.NullInt64(int64(userID)),
			SubscriptionID:    subscriptionID,
			DeviceFingerprint: deviceHash,
			DeviceHash:        database.NullString(deviceHash),
			DeviceUA:          database.NullString(userAgent),
			DeviceName:        database.NullString(deviceInfo.DeviceName),
			DeviceType:        database.NullString(deviceInfo.DeviceType),
			DeviceModel:       database.NullString(deviceInfo.DeviceModel),
			DeviceBrand:       database.NullString(deviceInfo.DeviceBrand),
			IPAddress:         database.NullString(ipAddress),
			UserAgent:         database.NullString(userAgent),
			SoftwareName:      database.NullString(deviceInfo.SoftwareName),
			SoftwareVersion:   database.NullString(deviceInfo.SoftwareVersion),
			OSName:            database.NullString(deviceInfo.OSName),
			OSVersion:         database.NullString(deviceInfo.OSVersion),
			IsActive:          true,
			IsAllowed:         true,
			FirstSeen:         database.NullTime(now),
			LastAccess:        now,
			LastSeen:          database.NullTime(now),
			AccessCount:       1,
		}

		if err := dm.db.Create(&device).Error; err != nil {
			return nil, err
		}

		// 更新订阅的设备计数
		var deviceCount int64
		dm.db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscriptionID, true).Count(&deviceCount)
		dm.db.Model(&models.Subscription{}).Where("id = ?", subscriptionID).Update("current_devices", deviceCount)

		return &device, nil
	}

	return nil, err
}

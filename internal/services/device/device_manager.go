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

	return info
}

func (dm *DeviceManager) matchSoftware(userAgent, uaLower string) string {
	if strings.Contains(uaLower, "shadowrocket") {
		return "Shadowrocket"
	}

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

	if strings.Contains(uaLower, "v2rayn") {
		return "v2rayN"
	}

	if strings.Contains(uaLower, "mihomo.party") || strings.Contains(uaLower, "mihomo/") {
		return "Mihomo Party"
	}
	if strings.Contains(uaLower, "mihomo") {
		return "Mihomo"
	}

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

func (dm *DeviceManager) parseOSInfo(userAgent, uaLower string) map[string]string {
	result := map[string]string{
		"os_name":    "Unknown",
		"os_version": "",
	}

	if strings.Contains(uaLower, "iphone") || strings.Contains(uaLower, "ipad") || strings.Contains(uaLower, "ipod") {
		result["os_name"] = "iOS"
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

	if strings.Contains(uaLower, "android") {
		result["os_name"] = "Android"
		if match := regexp.MustCompile(`Android\s+(\d+[.\d]*)`).FindStringSubmatch(userAgent); len(match) > 1 {
			result["os_version"] = match[1]
		}
		return result
	}

	if strings.Contains(uaLower, "windows") {
		result["os_name"] = "Windows"
		if match := regexp.MustCompile(`Windows\s+NT\s+(\d+\.\d+)`).FindStringSubmatch(userAgent); len(match) > 1 {
			result["os_version"] = match[1]
		}
		return result
	}

	if strings.Contains(uaLower, "macintosh") || strings.Contains(uaLower, "mac os") {
		result["os_name"] = "macOS"
		if match := regexp.MustCompile(`Mac OS X\s+(\d+[._]\d+)`).FindStringSubmatch(userAgent); len(match) > 1 {
			result["os_version"] = strings.Replace(match[1], "_", ".", -1)
		}
		return result
	}

	if strings.Contains(uaLower, "linux") {
		result["os_name"] = "Linux"
		return result
	}

	return result
}

func (dm *DeviceManager) inferOSFromSoftware(softwareName string) map[string]string {
	iosSoftware := []string{"shadowrocket", "quantumult", "surge", "loon", "stash", "anx", "anxray", "karing", "kitsunebi", "pharos", "potatso"}
	androidSoftware := []string{"clash for android", "clashandroid", "shadowsocks", "v2rayng"}
	windowsSoftware := []string{"clash for windows", "clash-verge", "v2rayn", "qv2ray", "mihomo", "mihomo party"}
	macosSoftware := []string{"clash for mac", "clashx", "clashx pro", "surge", "v2rayu", "mihomo", "mihomo party"}
	linuxSoftware := []string{"mihomo", "mihomo party"}

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
	for _, sw := range linuxSoftware {
		if strings.Contains(swLower, sw) {
			return map[string]string{"os_name": "Linux", "os_version": ""}
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

		if match := regexp.MustCompile(`iPhone(\d+,\d+)`).FindStringSubmatch(userAgent); len(match) > 1 {
			modelID := "iPhone" + match[1]
			if modelName, exists := iphoneModelMap[modelID]; exists {
				result["device_model"] = modelName
			} else {
				result["device_model"] = fmt.Sprintf("iPhone %s", strings.Replace(match[1], ",", ".", -1))
			}
		} else if match := regexp.MustCompile(`iPhone\s+(\d+)\s+Pro\s+Max`).FindStringSubmatch(userAgent); len(match) > 1 {
			result["device_model"] = fmt.Sprintf("iPhone %s Pro Max", match[1])
		} else if match := regexp.MustCompile(`iPhone\s+(\d+)\s+Pro`).FindStringSubmatch(userAgent); len(match) > 1 {
			result["device_model"] = fmt.Sprintf("iPhone %s Pro", match[1])
		} else if match := regexp.MustCompile(`iPhone\s+(\d+)\s+mini`).FindStringSubmatch(userAgent); len(match) > 1 {
			result["device_model"] = fmt.Sprintf("iPhone %s mini", match[1])
		} else if match := regexp.MustCompile(`iPhone\s+(\d+)`).FindStringSubmatch(userAgent); len(match) > 1 {
			result["device_model"] = fmt.Sprintf("iPhone %s", match[1])
		}

		if match := regexp.MustCompile(`iPad(\d+,\d+)`).FindStringSubmatch(userAgent); len(match) > 1 {
			result["device_model"] = fmt.Sprintf("iPad %s", strings.Replace(match[1], ",", ".", -1))
		} else if match := regexp.MustCompile(`iPad`).FindStringSubmatch(userAgent); len(match) > 0 {
			result["device_model"] = "iPad"
		}

		return result
	}

	if strings.Contains(uaLower, "android") {
		if match := regexp.MustCompile(`;\s*([^;]+)\s*build`).FindStringSubmatch(userAgent); len(match) > 1 {
			name := strings.TrimSpace(match[1])
			result["device_model"] = name
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

	if strings.Contains(osName, "ipad") || strings.Contains(uaLower, "ipad") {
		return "tablet"
	}
	if strings.Contains(osName, "ios") || strings.Contains(osName, "android") || strings.Contains(uaLower, "iphone") {
		return "mobile"
	}
	if strings.Contains(osName, "windows") || strings.Contains(osName, "macos") || strings.Contains(osName, "linux") {
		return "desktop"
	}

	if strings.Contains(swName, "shadowrocket") || strings.Contains(swName, "quantumult") || strings.Contains(swName, "surge") {
		if strings.Contains(uaLower, "ipad") {
			return "tablet"
		}
		return "mobile"
	}
	if strings.Contains(swName, "mihomo") {
		return "desktop"
	}
	if strings.Contains(swName, "clash for windows") || strings.Contains(swName, "v2rayn") {
		return "desktop"
	}

	return "unknown"
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

func (dm *DeviceManager) RecordDeviceAccess(subscriptionID uint, userID uint, userAgent, ipAddress, subscriptionType string) (*models.Device, error) {
	deviceInfo := dm.ParseUserAgent(userAgent)

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
					"v2rayn", "clash", "hiddify", "v2ray", "mihomo",
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

	deviceHash := dm.GenerateDeviceHash(userAgent, ipAddress, "")

	var existingDevice models.Device
	err := dm.db.Where("device_hash = ? AND subscription_id = ?", deviceHash, subscriptionID).First(&existingDevice).Error

	if err == nil {
		now := utils.GetBeijingTime()
		existingDevice.LastAccess = now
		existingDevice.LastSeen = &now
		existingDevice.AccessCount++
		existingDevice.IPAddress = &ipAddress
		existingDevice.UserAgent = &userAgent
		existingDevice.IsActive = true // 确保设备标记为活跃

		if subscriptionType != "" {
			subscriptionTypeStr := subscriptionType
			existingDevice.SubscriptionType = &subscriptionTypeStr
		}

		if deviceInfo.DeviceName != "Unknown Device" && (existingDevice.DeviceName == nil || *existingDevice.DeviceName == "" || *existingDevice.DeviceName == "Unknown Device") {
			existingDevice.DeviceName = &deviceInfo.DeviceName
		}
		if deviceInfo.DeviceType != "unknown" && (existingDevice.DeviceType == nil || *existingDevice.DeviceType == "" || *existingDevice.DeviceType == "unknown") {
			existingDevice.DeviceType = &deviceInfo.DeviceType
		}
		if deviceInfo.DeviceModel != "" && (existingDevice.DeviceModel == nil || *existingDevice.DeviceModel == "") {
			existingDevice.DeviceModel = &deviceInfo.DeviceModel
		}
		if deviceInfo.DeviceBrand != "" && (existingDevice.DeviceBrand == nil || *existingDevice.DeviceBrand == "") {
			existingDevice.DeviceBrand = &deviceInfo.DeviceBrand
		}
		if deviceInfo.SoftwareName != "Unknown" && (existingDevice.SoftwareName == nil || *existingDevice.SoftwareName == "" || *existingDevice.SoftwareName == "Unknown") {
			existingDevice.SoftwareName = &deviceInfo.SoftwareName
		}
		if deviceInfo.SoftwareVersion != "" && (existingDevice.SoftwareVersion == nil || *existingDevice.SoftwareVersion == "") {
			existingDevice.SoftwareVersion = &deviceInfo.SoftwareVersion
		}
		if deviceInfo.OSName != "Unknown" && (existingDevice.OSName == nil || *existingDevice.OSName == "" || *existingDevice.OSName == "Unknown") {
			existingDevice.OSName = &deviceInfo.OSName
		}
		if deviceInfo.OSVersion != "" && (existingDevice.OSVersion == nil || *existingDevice.OSVersion == "") {
			existingDevice.OSVersion = &deviceInfo.OSVersion
		}

		if err := dm.db.Save(&existingDevice).Error; err != nil {
			return nil, err
		}
		return &existingDevice, nil
	} else if err == gorm.ErrRecordNotFound {
		now := utils.GetBeijingTime()
		userIDInt64 := int64(userID)
		subscriptionTypeStr := subscriptionType
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

		var deviceCount int64
		dm.db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscriptionID, true).Count(&deviceCount)
		dm.db.Model(&models.Subscription{}).Where("id = ?", subscriptionID).Update("current_devices", deviceCount)

		return &device, nil
	}

	return nil, err
}

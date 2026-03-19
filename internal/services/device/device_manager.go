package device

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/geoip"
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
	// Shadowrocket
	if strings.Contains(uaLower, "shadowrocket") {
		return "Shadowrocket"
	}

	// iOS 设备特征识别
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

	// Mihomo 系列
	if strings.Contains(uaLower, "mihomo.party") || strings.Contains(uaLower, "mihomo/") {
		return "Mihomo Party"
	}
	if strings.Contains(uaLower, "mihomo") {
		return "Mihomo"
	}

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
	if strings.Contains(uaLower, "clash for android") || strings.Contains(uaLower, "cfa") {
		return "Clash for Android"
	}
	if strings.Contains(uaLower, "surfboard") {
		return "Surfboard"
	}
	if strings.Contains(uaLower, "v2rayng") {
		return "v2rayNG"
	}

	// 通用软件识别
	softwares := map[string]string{
		"quantumult":  "Quantumult",
		"hiddify":     "Hiddify",
		"clash meta":  "Clash Meta",
		"clash":       "Clash",
		"v2ray":       "V2Ray",
		"xray":        "Xray",
		"loon":        "Loon",
		"surge":       "Surge",
		"stash":       "Stash",
		"shadowsocks": "Shadowsocks",
		"sing-box":    "sing-box",
		"karing":      "Karing",
		"nekobox":     "NekoBox",
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

	// 路由器系统识别（优先级最高）
	if strings.Contains(uaLower, "openwrt") {
		result["os_name"] = "OpenWrt"
		if match := regexp.MustCompile(`OpenWrt[/\s]+(\d+[.\d]*)`).FindStringSubmatch(userAgent); len(match) > 1 {
			result["os_version"] = match[1]
		}
		return result
	}
	if strings.Contains(uaLower, "routeros") {
		result["os_name"] = "RouterOS"
		if match := regexp.MustCompile(`RouterOS[/\s]+(\d+[.\d]*)`).FindStringSubmatch(userAgent); len(match) > 1 {
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
		if match := regexp.MustCompile(`Android\s+(\d+[.\d]*)`).FindStringSubmatch(userAgent); len(match) > 1 {
			result["os_version"] = match[1]
		}
		return result
	}

	// Windows 识别
	if strings.Contains(uaLower, "windows") {
		result["os_name"] = "Windows"
		if match := regexp.MustCompile(`Windows\s+NT\s+(\d+\.\d+)`).FindStringSubmatch(userAgent); len(match) > 1 {
			result["os_version"] = match[1]
		}
		return result
	}

	// macOS 识别
	if strings.Contains(uaLower, "macintosh") || strings.Contains(uaLower, "mac os") {
		result["os_name"] = "macOS"
		if match := regexp.MustCompile(`Mac OS X\s+(\d+[._]\d+)`).FindStringSubmatch(userAgent); len(match) > 1 {
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
	windowsSoftware := []string{"clash for windows", "clash-verge", "clash verge", "v2rayn", "qv2ray", "mihomo party"}
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

	// Mihomo 可能在多个平台
	if strings.Contains(swLower, "mihomo") {
		return map[string]string{"os_name": "Linux", "os_version": ""}
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
	if strings.Contains(swName, "mihomo") || strings.Contains(swName, "clash for windows") || strings.Contains(swName, "v2rayn") {
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

	// 软路由常见特征：clash/mihomo + mips/arm/aarch64 架构
	if (strings.Contains(uaLower, "clash") || strings.Contains(uaLower, "mihomo")) &&
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

	// 如果通过 device_hash 找不到设备，尝试通过相同的 User-Agent 查找
	if err == gorm.ErrRecordNotFound {
		var sameUADevice models.Device
		if uaErr := dm.db.Where("subscription_id = ? AND user_agent = ? AND is_active = ?", subscriptionID, userAgent, true).
			Order("last_access DESC").
			First(&sameUADevice).Error; uaErr == nil {
			// 找到相同 User-Agent 的设备，更新其 hash 和其他信息
			now := utils.GetBeijingTime()
			sameUADevice.DeviceHash = &deviceHash
			sameUADevice.IPAddress = &ipAddress
			sameUADevice.LastAccess = now
			sameUADevice.LastSeen = &now
			sameUADevice.AccessCount++
			sameUADevice.UserAgent = &userAgent
			sameUADevice.IsActive = true

			// 查询并更新位置信息（使用缓存）
			if ipAddress != "" {
				location := geoip.GetLocationWithCache(ipAddress)
				if location.Valid && location.String != "" {
					sameUADevice.Location = &location.String
				}
			}

			if subscriptionType != "" {
				subscriptionTypeStr := subscriptionType
				sameUADevice.SubscriptionType = &subscriptionTypeStr
			}

			if deviceInfo.DeviceName != "Unknown Device" && (sameUADevice.DeviceName == nil || *sameUADevice.DeviceName == "" || *sameUADevice.DeviceName == "Unknown Device") {
				sameUADevice.DeviceName = &deviceInfo.DeviceName
			}
			if deviceInfo.DeviceType != "unknown" && (sameUADevice.DeviceType == nil || *sameUADevice.DeviceType == "" || *sameUADevice.DeviceType == "unknown") {
				sameUADevice.DeviceType = &deviceInfo.DeviceType
			}
			if deviceInfo.DeviceModel != "" && (sameUADevice.DeviceModel == nil || *sameUADevice.DeviceModel == "") {
				sameUADevice.DeviceModel = &deviceInfo.DeviceModel
			}
			if deviceInfo.DeviceBrand != "" && (sameUADevice.DeviceBrand == nil || *sameUADevice.DeviceBrand == "") {
				sameUADevice.DeviceBrand = &deviceInfo.DeviceBrand
			}
			if deviceInfo.SoftwareName != "Unknown" && (sameUADevice.SoftwareName == nil || *sameUADevice.SoftwareName == "" || *sameUADevice.SoftwareName == "Unknown") {
				sameUADevice.SoftwareName = &deviceInfo.SoftwareName
			}
			if deviceInfo.SoftwareVersion != "" && (sameUADevice.SoftwareVersion == nil || *sameUADevice.SoftwareVersion == "") {
				sameUADevice.SoftwareVersion = &deviceInfo.SoftwareVersion
			}
			if deviceInfo.OSName != "Unknown" && (sameUADevice.OSName == nil || *sameUADevice.OSName == "" || *sameUADevice.OSName == "Unknown") {
				sameUADevice.OSName = &deviceInfo.OSName
			}
			if deviceInfo.OSVersion != "" && (sameUADevice.OSVersion == nil || *sameUADevice.OSVersion == "") {
				sameUADevice.OSVersion = &deviceInfo.OSVersion
			}

			if err := dm.db.Save(&sameUADevice).Error; err != nil {
				return nil, err
			}
			return &sameUADevice, nil
		}
	}

	if err == nil {
		now := utils.GetBeijingTime()
		existingDevice.LastAccess = now
		existingDevice.LastSeen = &now
		existingDevice.AccessCount++
		existingDevice.IPAddress = &ipAddress
		existingDevice.UserAgent = &userAgent
		existingDevice.IsActive = true // 确保设备标记为活跃

		// 查询并更新位置信息（使用缓存）
		if ipAddress != "" {
			location := geoip.GetLocationWithCache(ipAddress)
			if location.Valid && location.String != "" {
				existingDevice.Location = &location.String
			}
		}

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

		var deviceCount int64
		dm.db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscriptionID, true).Count(&deviceCount)
		dm.db.Model(&models.Subscription{}).Where("id = ?", subscriptionID).Update("current_devices", deviceCount)

		return &device, nil
	}

	return nil, err
}

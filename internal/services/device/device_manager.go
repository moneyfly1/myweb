package device

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
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
	// 只写 iOS 不写 iPhone 标识的客户端 UA（如 "V2Box 10.1.5/iOS 26.5.2"）
	reIOSToken = regexp.MustCompile(`(?i)(?:^|[/\s(])ios(?:[\s/),]|$)`)
	// macOS 版本号（"macOS 12.6.9" / "Mac OS X 10.15.7" 之外的写法）
	reMacOSVersion = regexp.MustCompile(`(?i)mac(?:\s?os\s?x?)\s*(\d+[._]\d+(?:[._]\d+)?)`)
	// iOS 系统版本号（含旧式 "iPhone OS 16_6_1"）
	iosVersionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`OS\s+(\d+)[._](\d+)(?:[._](\d+))?`),          // OS 16_6_1, OS 16.6.1
		regexp.MustCompile(`iPhone\s+OS\s+(\d+)[._](\d+)(?:[._](\d+))?`), // iPhone OS 16_6_1
		regexp.MustCompile(`Version/(\d+)[._](\d+)(?:[._](\d+))?`),       // Version/16.6.1
		regexp.MustCompile(`iOS\s+(\d+)[._](\d+)(?:[._](\d+))?`),         // iOS 16.6.1
	}
	// UA 里显式的 iOS 版本号（"iOS 26.5.2"）
	reIOSVersion = regexp.MustCompile(`(?i)ios\s+(\d+)[._](\d+)(?:[._](\d+))?`)
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
		// 无 UA：不是正常客户端（订阅访问控制也会拦掉）。历史行里这类多半是脚本/内部探测，
		// 明确标出来，避免继续显示成 "Unknown" 让人以为是识别失败。
		info.SoftwareName = "（无 UA）"
		info.DeviceName = "未上报 UA 的请求"
		return info
	}

	uaLower := strings.ToLower(userAgent)

	// Dalvik = Android 的运行时。原生 App 用 HttpClient 发请求时默认就带
	// "Dalvik/2.1.0 (Linux; U; Android 14; …)"，里面没有任何客户端名。
	// 这是**真实设备**，只是软件名未知：识别成 Android 平台，
	// 而不是丢进 "Unknown"（线上 27 台未知设备里有 17 台是这种）。
	trimmedUA := strings.TrimSpace(uaLower)
	if strings.HasPrefix(trimmedUA, "dalvik/") || trimmedUA == "android" {
		osVersion := ""
		if m := reAndroidVersion.FindStringSubmatch(userAgent); len(m) > 1 {
			osVersion = m[1]
		}
		// 软件名保持 Unknown：UA 里确实没有任何客户端标识，不能瞎猜（诚实标注）。
		// 但平台能确定 → 设备名与 OS 显示成 Android，避免列表里只看到干巴巴的 "Unknown"。
		info.OSName = "Android"
		info.OSVersion = osVersion
		info.DeviceType = "mobile"
		info.DeviceName = "Android 设备（客户端未上报 UA）"
		return info
	}

	info.SoftwareName = dm.matchSoftware(userAgent, uaLower)

	osInfo := dm.parseOSInfo(userAgent, uaLower)
	info.OSName = osInfo["os_name"]
	info.OSVersion = osInfo["os_version"]

	// 平台专属客户端的归属强于 UA 里的通用段：
	// Shadowrocket/Quantumult 只可能是 iOS，V2rayU/ClashX 只可能是 macOS。
	// iOS 客户端 UA 常只带 "CFNetwork/… Darwin/…"（没有 iPhone 段），
	// 光看 UA 会判不出，所以这里按客户端名覆盖。
	if osName, ok := strongPlatformOS(info.SoftwareName); ok {
		info.OSName = osName
	}

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

	// 客户端是否上报了**稳定设备 ID**（自研客户端 MoneyFly 的 X-MF-Device-Id）。
	// 这个标记决定了下面能不能用"指纹/UA 兜底"去找设备：
	//
	// 实测缺陷（用户反馈"最大设备数 7，但自研客户端第 7 台不算数"）：
	// MoneyFly 各台设备的 UA 完全相同（如都是 "MoneyFly/2.2.24 (Windows NT 10.0; Win64; x64)"），
	// 而哈希没命中时会走「按 UA 匹配」的兜底 —— 于是 8 台不同设备（8 个不同 X-MF-Device-Id）
	// 全部被当成同一台，库里只有 1 行、access_count 累加到 8，设备数限制对自研客户端形同失效。
	// 结论：客户端既然给了稳定 ID，就以 ID 为权威身份；哈希没命中就是**新设备**，
	// 不能再退化成 UA/机型指纹匹配（那是给「没上报 ID 的历史行」用的）。
	explicitDeviceID := ""
	if headers != nil {
		explicitDeviceID = strings.TrimSpace(headers["X-MF-Device-Id"])
	}

	var device models.Device
	err := dm.db.Where("device_hash = ? AND subscription_id = ?", deviceHash, subscriptionID).First(&device).Error
	if err == nil {
		return &device, true, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, false, err
	}

	// 存量迁移：新算法把「网络段」并入了身份，老行的 hash 是旧算法（不含网络段）。
	// 先用旧算法精确认回那一行并改写为新哈希 —— 否则每个存量设备上线后
	// 首次拉订阅都会多出一行幽灵设备、白占一个名额。
	if prevHash := dm.GenerateDeviceHashWithoutNetwork(userAgent, headers); prevHash != "" && prevHash != deviceHash {
		if err := dm.db.Where("device_hash = ? AND subscription_id = ?", prevHash, subscriptionID).
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

	// 有显式设备 ID 时先尝试**硬件指纹认亲**，再判定为新设备。
	//
	// 为什么需要：安装实例 ID（device_hash）只在"重装 App / 清数据 / 桌面安装包
	// 升级时把数据目录删了"之后会变，而这几件事都会让同一台物理机被当成新设备
	// 重复登记（用户实测 susanzhang1230：一台 Windows 2.2.25→2.2.28 升级后
	// 变成两行，白占一个名额）。硬件指纹（MachineGuid / IOPlatformUUID /
	// ANDROID_ID）跨重装不变，用它在同一订阅下把老行认回来，
	// 并把 device_hash 改写成新的安装实例哈希（身份仍以安装实例为准）。
	if explicitDeviceID != "" {
		if adopted, err := dm.adoptRowByHardwareFingerprint(subscriptionID, headers, deviceHash); err != nil {
			return nil, false, err
		} else if adopted != nil {
			return adopted, true, nil
		}
		return nil, false, nil
	}

	if userAgent != "" {
		var candidates []models.Device
		if err := dm.db.Where("subscription_id = ? AND user_agent = ? AND is_active = ?", subscriptionID, userAgent, true).
			Order("last_access DESC").
			Limit(10).
			Find(&candidates).Error; err != nil {
			return nil, false, err
		}
		// 同 UA 只说明「同型号同版本客户端」，不等于同一台设备。判定顺序：
		//   ① 当前网络段在该设备的**历史画像**里 → 同一台（换回家里/公司等旧网络）；
		//   ② 当前网络段与该行当前 IP 同段 → 同一台（IP 微变）；
		//   ③ 都不命中：名额未满 → 视为**另一台设备**（同型号两台手机要能分开）；
		//      名额已满 → 复用最近活跃行（视作这台设备换了新网络，不额外占名额、
		//      也不让用户因为换网被"设备超限"挡住）。
		if ipAddress == "" {
			return &candidates[0], true, nil
		}
		for i := range candidates {
			if knowsNetwork(&candidates[i], ipAddress) {
				return &candidates[i], true, nil
			}
		}
		for i := range candidates {
			rowIP := ""
			if candidates[i].IPAddress != nil {
				rowIP = *candidates[i].IPAddress
			}
			if rowIP == "" || sameNetworkSegment(rowIP, ipAddress) {
				return &candidates[i], true, nil
			}
		}
		if len(candidates) > 0 && dm.quotaExhausted(subscriptionID) {
			return &candidates[0], true, nil
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

// automationSoftwareNames 自动化/抓取类客户端（不是真实用户设备）。
//
// 这些 UA 来自脚本、爬虫、监控探针、第三方测速与转发服务：
// curl / wget / Go-http-client（Go 程序默认 UA）/ python-requests / OkHttp / axios …
// 它们既不是"用户的设备"，也不该占用设备名额 —— 线上案例：某用户订阅地址泄露后，
// 被大量海外 VPS 用 Go 脚本轮询，设备列表里堆了几十条 Go-http-client，
// 每条 IP 都算一台设备，把设备名额吃满。
var automationSoftwareNames = map[string]bool{
	"curl": true, "wget": true, "go-http-client": true,
	"python-requests": true, "python-urllib": true,
	"okhttp": true, "postman": true, "axios": true, "node-fetch": true,
	"apache-httpclient": true, "libwww-perl": true, "got": true,
	"undici": true, "java-http-client": true, "java": true,
	"直接抓取订阅": true, "链接预览": true, "协议链接请求": true, "反垃圾检测": true,
}

// IsAutomationSoftware 该软件名是否属于「自动化/抓取类客户端」。
// 这类请求**不登记为设备**（但访问日志照常记录，供滥用分析）。
func IsAutomationSoftware(softwareName string) bool {
	name := strings.ToLower(strings.TrimSpace(softwareName))
	if name == "" || name == "unknown" {
		return false
	}
	if automationSoftwareNames[name] {
		return true
	}
	// 预览/抓取类的软件名带来源前缀（如 "WhatsApp 链接预览"、"钉钉链接预览"），
	// 精确匹配会漏掉它们，这里再按关键词包含判断一次。
	for _, marker := range []string{"链接预览", "直接抓取订阅", "协议链接请求", "反垃圾检测"} {
		if strings.Contains(name, strings.ToLower(marker)) {
			return true
		}
	}
	return false
}

// HardwareFingerprintOf 取出客户端上报的硬件指纹（X-MF-Device-Fp）。
// 客户端没上报（旧版本/第三方客户端）时返回空串，匹配逻辑保持原样。
func HardwareFingerprintOf(headers map[string]string) string {
	if headers == nil {
		return ""
	}
	return strings.TrimSpace(headers["X-MF-Device-Fp"])
}

// adoptRowByHardwareFingerprint 用硬件指纹在同一订阅下认回老行。
//
// 命中条件：同一订阅 + device_hw_fp 相同 + **当前活跃**。
// 认回后：
//   - device_hash 改写成**当前安装实例**的哈希（之后按安装 ID 直接命中，快路径）；
//   - device_hw_fp 补齐（老行可能没写过）。
//
// 只看活跃行是刻意的：被「踢下线」的软删行（is_active=false + kicked_at 非空）
// 绝不能在这里被悄悄复活 —— 那等于绕过用户"删除设备"的意图。踢下线语义由
// FindKickedDeviceWithHeaders / ResolveKickGate 统一裁决（它同样按硬件指纹找，
// 所以重装后仍然会命中闸门，要么按现有规则自动恢复、要么明确拒绝）。
//
// 只在客户端**确实上报了硬件指纹**时才会命中；没上报的历史行不受影响。
func (dm *DeviceManager) adoptRowByHardwareFingerprint(
	subscriptionID uint, headers map[string]string, newInstallHash string) (*models.Device, error) {
	hwFP := HardwareFingerprintOf(headers)
	if hwFP == "" {
		return nil, nil
	}

	var device models.Device
	err := dm.db.Where("subscription_id = ? AND device_hw_fp = ? AND is_active = ? AND kicked_at IS NULL",
		subscriptionID, hwFP, true).
		Order("last_access DESC").
		First(&device).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// 同一台物理机的新安装实例：沿用老行（名额不变），只更新身份与指纹
	if err := dm.db.Model(&models.Device{}).Where("id = ?", device.ID).Updates(map[string]any{
		"device_hash":  newInstallHash,
		"device_hw_fp": hwFP,
	}).Error; err != nil {
		return nil, err
	}
	device.DeviceHash = &newInstallHash
	device.DeviceHwFp = &hwFP
	return &device, nil
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

	if hwFP := HardwareFingerprintOf(headers); hwFP != "" {
		// 硬件指纹命中软删行：同一台机器重装/升级（install_id 变了）后仍要命中
		// 踢下线判定，否则「删除设备」的意图会被一次重装绕过。
		// ⚠️ 只读，绝不改写行 —— 改写会让后续指纹兜底失效、闸门误放行（线上 sub 638 实测）。
		var byHwFp models.Device
		err := dm.db.Where("subscription_id = ? AND device_hw_fp = ? AND is_active = ? AND kicked_at IS NOT NULL",
			subscriptionID, hwFP, false).
			Order("last_access DESC").
			First(&byHwFp).Error
		if err == nil {
			return &byHwFp, nil
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
//
//	删除设备 = is_active=false + kicked_at=now，此后该设备拉订阅一律 403
//	「此设备已被移除并踢下线,如需继续使用请重新登录或联系客服」。
//	但**代码里没有任何地方会清除 kicked_at**（FindKickedDevice 只读、删除接口只写），
//	于是「请重新登录」实际无效：用户只要删掉了本机（例如到期后清理设备、或先删光
//	再续费），就永久拿不到订阅、设备列表里也不再出现这台设备，只能找客服改库。
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

	count, err := CountOnlineDevices(dm.db, subscriptionID)
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
		newCount, cErr := CountRecentDevices(tx, subscriptionID, DisplayWindow)
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
		deviceCount, _ := CountRecentDevices(dm.db, canonical.SubscriptionID, DisplayWindow)
		return dm.db.Model(&models.Subscription{}).Where("id = ?", canonical.SubscriptionID).Update("current_devices", deviceCount).Error
	}

	return nil
}

func (dm *DeviceManager) matchSoftware(userAgent, uaLower string) string {
	// ① UA 本身就是订阅链接：脚本 / 中间层把地址当作 UA 发出来，没有任何客户端标识。
	//    这类请求既不是用户设备，也不属于任何已知客户端，必须单独归类，
	//    否则全部堆进"未知设备"（实测生产库一次就有十几条）。
	if strings.HasPrefix(uaLower, "http://") || strings.HasPrefix(uaLower, "https://") {
		return "直接抓取订阅"
	}
	// 节点协议链接被当成 UA 发过来（vmess://、vless://、ss:// …），同上：不是设备
	if strings.HasPrefix(uaLower, "vmess://") || strings.HasPrefix(uaLower, "vless://") ||
		strings.HasPrefix(uaLower, "ss://") || strings.HasPrefix(uaLower, "ssr://") ||
		strings.HasPrefix(uaLower, "trojan://") {
		return "协议链接请求"
	}

	// ② 社交 / IM 的链接预览爬虫：用户把订阅链接发进聊天窗口，由服务端来抓一次。
	//    它们不是设备（没有 OS、没有硬件指纹），归为「链接预览」，从设备统计里剔除。
	previewBots := []struct{ key, name string }{
		{"dingtalkbot", "钉钉链接预览"},
		{"wxwork", "企业微信链接预览"},
		{"micromessenger", "微信链接预览"},
		{"whatsapp/", "WhatsApp 链接预览"},
		{"googlemessages", "Google 链接预览"},
		{"telegrambot", "Telegram 链接预览"},
		{"telegram", "Telegram 链接预览"},
		{"facebookexternalhit", "Facebook 链接预览"},
		{"facebookbot", "Facebook 链接预览"},
		{"twitterbot", "Twitter 链接预览"},
		{"slackbot", "Slack 链接预览"},
		{"discordbot", "Discord 链接预览"},
		{"linkedinbot", "LinkedIn 链接预览"},
		{"pinterestbot", "Pinterest 链接预览"},
		{"redditbot", "Reddit 链接预览"},
		{"bingbot", "Bing 链接预览"},
		{"googlebot", "Google 链接预览"},
		{"applebot", "Apple 链接预览"},
		{"vkshare", "VK 链接预览"},
	}
	for _, b := range previewBots {
		if strings.Contains(uaLower, b.key) {
			return b.name
		}
	}

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

	// 具体客户端优先于"clash verge / clash for windows"等泛化分支 ——
	// 实测 FlClash 的 UA 是 "FlClash/v0.9.1 clash-verge Platform/macos"，
	// 先判 clash-verge 会把它误认成 Clash Verge。
	if strings.Contains(uaLower, "flclash") {
		return "FlClash"
	}
	if strings.Contains(uaLower, "mihomo.party") || strings.Contains(uaLower, "mihomo-party") ||
		strings.Contains(uaLower, "mihomo_party") {
		return "Mihomo Party"
	}

	// Android 的 v2rayNG 名字里含 "v2rayn"，必须先判，否则会被当成 Windows 的
	// v2rayN（老 bug：Android 设备被统计成 Windows）
	if strings.Contains(uaLower, "v2rayng") {
		return "v2rayNG"
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

	// 自动化/抓取类 UA：**不是用户设备**（面板健康检查、订阅探测、脚本抓取、
	// 第三方监控）。以前它们落到 "Unknown"，让"未知设备"数字虚高。
	// 单独归类后，分析页可以把它们排除在设备统计之外。
	automationUA := []struct {
		key  string
		name string
	}{
		{"curl/", "curl"},
		{"wget/", "Wget"},
		{"go-http-client", "Go-http-client"},
		{"python-requests", "python-requests"},
		{"python-urllib", "python-urllib"},
		{"okhttp", "OkHttp"},
		{"postmanruntime", "Postman"},
		{"axios/", "axios"},
		{"node-fetch", "node-fetch"},
		{"java/", "Java"},
		{"apache-httpclient", "Apache HttpClient"},
		{"libwww-perl", "libwww-perl"},
		{"got (", "got"},
		{"got/", "got"},
		{"undici", "undici"},
		{"java-http-client", "Java HttpClient"},
		// 容器/反垃圾等健康检查脚本
		{"container_curl", "curl"},
		{"antispam/", "反垃圾检测"},
	}
	for _, a := range automationUA {
		if strings.Contains(uaLower, a.key) {
			return a.name
		}
	}

	// 通用软件识别。这里必须保持有序：一些客户端 UA 会同时带上内核/兼容标识，
	// 例如 HiddifyNext/... like ClashMeta v2ray sing-box，应优先识别真实客户端。
	// ⚠️ 顺序原则：**具体名在前、泛化名（clash/v2ray）在后** ——
	//    "mihomo.party/v2.0.0 (clash.meta)" 必须识别成 Mihomo Party 而不是 Clash。
	softwares := []struct {
		key  string
		name string
	}{
		{"mihomo.party", "Mihomo Party"},
		{"mihomo-party", "Mihomo Party"},
		{"mihomo_party", "Mihomo Party"},
		{"flclash", "FlClash"},
		// FlClash 在 Android 上的短 UA："FlC/0.8.61/meta/verge/Android 10; HJC-AN90"
		{"flc/", "FlClash"},
		// macOS / iOS 上的具体客户端（UA 通常带 macos/ios 段，早识别避免落到 Unknown）
		{"clashx meta", "ClashX Meta"},
		{"clashx", "ClashX"},
		{"v2rayu", "V2rayU"},
		{"v2box", "V2Box"},
		// 线上剩余的零散未知客户端（实测 UA）：Happ/NexVPN、SFMainland_Store_Pro、meta/nextin
		{"happ/", "Happ"},
		{"nexvpn", "NexVPN"},
		{"meta/nextin", "Nextin"},
		{"sagernet", "SagerNet"},
		{"matsuri", "Matsuri"},
		{"tomoonclash", "ToMoon Clash"},
		{"axxray", "ANXRay"},
		{"anxray", "ANXRay"},
		// Clash Meta for Android 的短 UA："CMFAndroid/2.11.1x"
		{"cmfandroid", "Clash Meta"},
		{"clash.meta", "Clash Meta"},
		{"clash-meta", "Clash Meta"},
		{"clashmetaforandroid", "Clash Meta"},
		{"mihomo", "Mihomo"},
		{"karing", "Karing"},
		{"sfi/", "sing-box"},
		{"sfa/", "sing-box"},
		{"sfm/", "sing-box"},
		{"nekoray", "NekoRay"},
		{"nekobox", "NekoBox"},
		// OpenWrt PassWall 的 UA 就是 "passwall/26.8.26"（路由器插件，不是手机/桌面）
		{"passwall", "PassWall"},
		{"v2guard", "V2Guard"},
		{"clash party", "Clash Party"},
		{"clash-party", "Clash Party"},
		{"moneyfly", "MoneyFly"},
		// MBox：sing-box 内核的自有客户端，UA 形如
		// `MBox/0.0.1 (Macintosh; Mac OS X 26_6)`。必须排在 sing-box / mihomo /
		// clash 之前 —— 它的 UA 里可能带内核串，排后面会被认成内核名
		// （和 MoneyFly 当年被认成 Mihomo 是同一个坑）。
		{"mbox", "MBox"},
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

	// iOS/iPadOS 识别。除了标准 iPhone/iPad 标识，还要认只写 "iOS" 的客户端 UA
	// （"V2Box 10.1.5/iOS 26.5.2"、"HiddifyNext/4.0.0 (ios) like ClashMeta ..."），
	// 否则这些真实 iOS 设备会落到"未知系统"。
	if strings.Contains(uaLower, "iphone") || strings.Contains(uaLower, "ipad") ||
		strings.Contains(uaLower, "ipod") || reIOSToken.MatchString(userAgent) {
		if strings.Contains(uaLower, "ipad") {
			result["os_name"] = "iPadOS"
		} else {
			result["os_name"] = "iOS"
		}
		for _, re := range iosVersionPatterns {
			if match := re.FindStringSubmatch(userAgent); len(match) > 1 {
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

	// macOS 识别。除标准 "Macintosh"/"Mac OS X"，还要认 "macOS 12.6.9"（ClashX 系列 UA）。
	// ⚠️ 不能用裸 "CFNetwork/Darwin" 判 macOS：iOS 客户端（Shadowrocket 等）也带这段，
	//    误判会把它们从 iOS 挪到 macOS。桌面专属客户端（ClashX/V2rayU）由
	//    strongPlatformOS 覆盖成 macOS，所以下面这段只兜底 Apple 平台的其余情况。
	if strings.Contains(uaLower, "macintosh") || strings.Contains(uaLower, "mac os") ||
		strings.Contains(uaLower, "macos") {
		result["os_name"] = "macOS"
		if match := reMacOSX.FindStringSubmatch(userAgent); len(match) > 1 {
			result["os_version"] = strings.Replace(match[1], "_", ".", -1)
		} else if match := reMacOSVersion.FindStringSubmatch(userAgent); len(match) > 1 {
			result["os_version"] = strings.Replace(match[1], "_", ".", -1)
		}
		return result
	}

	// Apple 平台兜底：带 CFNetwork/Darwin 又没有 Mac 标识的 UA
	// （App 内嵌 webview 抓订阅、只报 CFNetwork 的 iOS 客户端）→ 按 iOS 归类。
	if strings.Contains(uaLower, "cfnetwork") || strings.Contains(uaLower, "darwin/") {
		result["os_name"] = "iOS"
		if match := reIOSVersion.FindStringSubmatch(userAgent); len(match) > 1 {
			version := match[1] + "." + match[2]
			if len(match) > 3 && match[3] != "" {
				version += "." + match[3]
			}
			result["os_version"] = version
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

// strongPlatformOS 返回「只存在于单一平台」的客户端所属系统。
//
// 用途：覆盖 UA 里缺失/笼统的 OS 段。iOS 客户端（Shadowrocket、V2Box）常只报
// "CFNetwork/… Darwin/…" 而没有 iPhone 标识，光看 UA 判不出平台；反过来
// 桌面客户端（V2rayU）也是同样的 UA 形状。按客户端名定性比猜 UA 可靠。
// 跨平台客户端（Clash Meta、sing-box、Karing、Surge、MoneyFly）不在此列。
func strongPlatformOS(softwareName string) (string, bool) {
	sw := strings.ToLower(softwareName)
	iosOnly := []string{"shadowrocket", "quantumult", "loon", "stash", "kitsunebi",
		"pharos", "potatso", "v2box", "anxray", "anx"}
	macosOnly := []string{"clashx meta", "clashx pro", "clashx", "v2rayu"}
	androidOnly := []string{"clash for android", "v2rayng", "surfboard", "nekobox", "sagernet", "matsuri"}
	routerOnly := []string{"passwall", "openclash", "ssr plus+", "ssrplus"}
	for _, s := range iosOnly {
		if strings.Contains(sw, s) {
			return "iOS", true
		}
	}
	for _, s := range macosOnly {
		if strings.Contains(sw, s) {
			return "macOS", true
		}
	}
	for _, s := range androidOnly {
		if strings.Contains(sw, s) {
			return "Android", true
		}
	}
	for _, s := range routerOnly {
		if strings.Contains(sw, s) {
			return "OpenWrt", true
		}
	}
	return "", false
}

func (dm *DeviceManager) inferOSFromSoftware(softwareName string) map[string]string {
	iosSoftware := []string{"shadowrocket", "quantumult", "surge", "loon", "stash", "anx", "anxray", "karing", "kitsunebi", "pharos", "potatso", "v2box"}
	androidSoftware := []string{"clash for android", "clashandroid", "shadowsocks", "v2rayng", "surfboard", "nekobox", "sfa/",
		"sagernet", "matsuri"}
	windowsSoftware := []string{"clash for windows", "clash-verge", "clash verge", "v2rayn", "qv2ray", "clash part", "nekoray"}
	macosSoftware := []string{"clash for mac", "clashx", "clashx pro", "v2rayu"}
	routerSoftware := []string{"openclash", "passwall", "ssr plus+", "ssrplus"}
	// 跨平台桌面客户端：UA 里只有客户端名、没有 OS 段（实测 "mihomo.party/v2.0.0
	// (clash.meta)" 就是这种）→ 明确标成桌面，而不是留一个"未知"让人以为识别失败。
	desktopSoftware := []string{"mihomo party", "mihomo.party", "mihomo", "clash verge", "clash-verge",
		"flclash", "clash meta", "clashmeta", "sing-box", "nekoray"}
	// 自动化/抓取：不是用户设备
	automationSoftware := []string{"curl", "wget", "go-http-client", "python-requests", "python-urllib",
		"okhttp", "postman", "axios", "node-fetch", "apache-httpclient", "libwww-perl",
		"got", "undici", "java-http-client", "直接抓取订阅", "链接预览", "协议链接请求", "反垃圾检测"}

	swLower := strings.ToLower(softwareName)

	// 自动化/抓取：归为「非客户端」，OS 未知但不再算"用户设备"
	for _, sw := range automationSoftware {
		if strings.Contains(swLower, sw) {
			return map[string]string{"os_name": "自动化/非客户端", "os_version": ""}
		}
	}

	// 跨平台桌面客户端（UA 无 OS 段）
	for _, sw := range desktopSoftware {
		if strings.Contains(swLower, sw) {
			return map[string]string{"os_name": "Desktop（Win/macOS）", "os_version": ""}
		}
	}

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
			// 老机型（生产库里仍有一批，补全后抽屉里不再显示 iPhone13,2 这种标识）
			"iPhone10,2": "iPhone 8 Plus",
			"iPhone10,3": "iPhone X",
			"iPhone10,5": "iPhone 8 Plus",
			"iPhone11,2": "iPhone XS",
			"iPhone11,4": "iPhone XS Max",
			"iPhone11,6": "iPhone XS Max",
			"iPhone11,8": "iPhone XR",
			"iPhone12,1": "iPhone 11",
			"iPhone12,3": "iPhone 11 Pro",
			"iPhone12,5": "iPhone 11 Pro Max",
			"iPhone12,8": "iPhone SE (第2代)",
			"iPhone13,1": "iPhone 12 mini",
			"iPhone13,2": "iPhone 12",
			"iPhone13,3": "iPhone 12 Pro",
			"iPhone13,4": "iPhone 12 Pro Max",
			"iPhone14,6": "iPhone SE (第3代)",
			"iPhone14,7": "iPhone 14",
			"iPhone14,8": "iPhone 14 Plus",
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
			"iPhone17,1": "iPhone 16 Pro",
			"iPhone17,2": "iPhone 16 Pro Max",
			"iPhone17,3": "iPhone 16",
			"iPhone17,4": "iPhone 16 Plus",
			"iPhone17,5": "iPhone 16e",
			"iPhone18,1": "iPhone 17 Pro",
			"iPhone18,2": "iPhone 17 Pro Max",
			"iPhone18,3": "iPhone 17",
			"iPhone18,4": "iPhone Air",
		}

		if match := reIPhoneModel.FindStringSubmatch(userAgent); len(match) > 1 {
			modelID := "iPhone" + match[1]
			if modelName, exists := iphoneModelMap[modelID]; exists {
				result["device_model"] = modelName
			} else {
				// 未收录的机型标识原样保留（iPhone19,7）。
				// ⚠️ 以前这里把逗号换成点，得到 "iPhone 18.2" —— 看起来像系统版本号，
				//    抽屉里就会显示成「iPhone 18.2」，用户以为机型识别错了（实测 1800+ 行）。
				result["device_model"] = modelID
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
				"iPad14,8":  "iPad Air 11英寸 (M2)",
				"iPad14,9":  "iPad Air 11英寸 (M2)",
				"iPad14,10": "iPad Air 13英寸 (M2)",
				"iPad14,11": "iPad Air 13英寸 (M2)",
				"iPad8,6":   "iPad Pro 11英寸 (第1代)",
				"iPad11,6":  "iPad (第8代)",
				"iPad12,1":  "iPad (第9代)",
				"iPad15,3":  "iPad Air 11英寸 (M3)",
				"iPad15,5":  "iPad Air 13英寸 (M3)",
				"iPad15,7":  "iPad (A16)",
				"iPad16,1":  "iPad mini (第7代)",
				"iPad16,2":  "iPad mini (第7代)",
				"iPad16,3":  "iPad Pro 11英寸 (M4)",
				"iPad16,4":  "iPad Pro 11英寸 (M4)",
				"iPad16,5":  "iPad Pro 13英寸 (M4)",
				"iPad16,6":  "iPad Pro 13英寸 (M4)",
			}
			if modelName, exists := iPadModelMap[modelID]; exists {
				result["device_model"] = modelName
			} else {
				result["device_model"] = modelID
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

// ipSegmentOf 取网络段（IPv4 前三段 / IPv6 前四组），用于把「同一网络」作为设备区分特征。
//
// 为什么需要：第三方客户端（Shadowrocket 等）的 UA 只有「软件名+系统名+机型」，
// 同型号同版本的两台手机 UA 完全相同；此前设备哈希也不含任何网络信息，于是
// 第二台设备直接命中第一台的哈希，被当成同一台（用户实际 2 台，面板只显示 1 台）。
// 把出口网络段并入身份后，不同网络下的同型号设备就能区分。
//
// 取 /24（IPv4）与 /64（IPv6）而不是完整 IP：同一网络内 DHCP 换地址、运营商
// 短暂换 IP 归属同一段时，身份仍保持稳定，不会每次都变成"新设备"。
func ipSegmentOf(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}
	if v4 := parsed.To4(); v4 != nil {
		return fmt.Sprintf("%d.%d.%d", v4[0], v4[1], v4[2])
	}
	// IPv6 取前 4 组（/64）
	groups := strings.Split(parsed.String(), ":")
	if len(groups) < 4 {
		return parsed.String()
	}
	return strings.Join(groups[:4], ":")
}

// networksOf 解析设备的历史网络段列表（JSON 数组，坏数据按空处理）
func networksOf(d *models.Device) []string {
	if d == nil || d.NetworkSegments == nil || strings.TrimSpace(*d.NetworkSegments) == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(*d.NetworkSegments), &out); err != nil {
		return nil
	}
	return out
}

// addNetworkSegment 把当前网络段并进画像（去重、最多保留 5 个、最近的排最后）
func addNetworkSegment(d *models.Device, seg string) bool {
	if d == nil || seg == "" {
		return false
	}
	list := networksOf(d)
	for _, existing := range list {
		if existing == seg {
			// 已存在：移到末尾表示"最近用过"
			reordered := make([]string, 0, len(list))
			for _, v := range list {
				if v != seg {
					reordered = append(reordered, v)
				}
			}
			reordered = append(reordered, seg)
			if len(reordered) == len(list) {
				// 顺序未变也写回，保证调用方不需要额外判断
				encoded, _ := json.Marshal(reordered)
				val := string(encoded)
				d.NetworkSegments = &val
				return false
			}
			encoded, _ := json.Marshal(reordered)
			val := string(encoded)
			d.NetworkSegments = &val
			return true
		}
	}
	list = append(list, seg)
	if len(list) > 5 {
		list = list[len(list)-5:]
	}
	encoded, err := json.Marshal(list)
	if err != nil {
		return false
	}
	val := string(encoded)
	d.NetworkSegments = &val
	return true
}

// quotaExhausted 该订阅的活跃设备数是否已经达到设备名额上限。
//
// 只影响「同 UA + 全新网络段」这一种情形的判定：名额没满时允许登记新设备
// （同型号的两台手机要能分开），名额已满时把新网络段当作「这台设备换了网络」
// 复用旧行，避免用户换个 WiFi 就被判成新设备、直接超额断连。
func (dm *DeviceManager) quotaExhausted(subscriptionID uint) bool {
	var sub models.Subscription
	if err := dm.db.Select("id, device_limit").First(&sub, subscriptionID).Error; err != nil {
		return false
	}
	if sub.DeviceLimit <= 0 {
		return false // 0 = 该订阅不开放设备（由上层按"超限"处理），这里不算名额已满
	}
	var count int64
	if err := dm.db.Model(&models.Device{}).
		Where("subscription_id = ? AND is_active = ?", subscriptionID, true).
		Count(&count).Error; err != nil {
		return false
	}
	return int(count) >= sub.DeviceLimit
}

// knowsNetwork 该设备历史上是否用过这个网络段（换回旧网络时认回同一台）
func knowsNetwork(d *models.Device, ip string) bool {
	seg := ipSegmentOf(ip)
	if seg == "" {
		return false
	}
	for _, s := range networksOf(d) {
		if s == seg {
			return true
		}
	}
	return false
}

// sameNetworkSegment 两个 IP 是否属于同一网络段（用于 UA 兜底匹配时的同设备判定）
func sameNetworkSegment(ipA, ipB string) bool {
	segA, segB := ipSegmentOf(ipA), ipSegmentOf(ipB)
	return segA != "" && segA == segB
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

	// 网络段：第三方客户端的 UA 常常完全一致（同型号同版本），
	// 只看 UA 无法区分「同一用户的第二台设备」，补上网络段作为区分特征。
	if seg := ipSegmentOf(ipAddress); seg != "" {
		features = append(features, "net:"+seg)
	}

	deviceString := strings.Join(features, "|")
	if deviceString == "" {
		deviceString = userAgent
	}

	hash := sha256.Sum256([]byte(deviceString))
	return hex.EncodeToString(hash[:])
}

// GenerateDeviceHashWithoutNetwork 上一版算法（不含网络段），仅用于**存量行迁移**：
// 部署「含网络段」的新算法后，老行的 device_hash 仍是旧值；先用它精确认回那一行，
// 再改写成新哈希，避免每个存量设备都多出一行幽灵设备。
// 新代码不要用它写库。
func (dm *DeviceManager) GenerateDeviceHashWithoutNetwork(userAgent string, headers map[string]string) string {
	if headers != nil {
		if id := strings.TrimSpace(headers["X-MF-Device-Id"]); id != "" {
			hash := sha256.Sum256([]byte("device_id:" + id))
			return hex.EncodeToString(hash[:])
		}
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
	// 记住用过的网络段：用户换回旧网络时能认回同一台（第三方客户端没有设备 ID，
	// 这是能拿到的唯一"稳定"特征）
	addNetworkSegment(device, ipSegmentOf(ipAddress))
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

	// 自动化/抓取类请求（curl / Go-http-client / python-requests …）不是用户设备：
	// 不登记、不占设备名额。否则订阅地址一旦泄露被脚本轮询，设备列表会被几十条
	// 同用途的行塞满，用户真实设备反而加不进来。访问日志仍由调用方记录。
	if IsAutomationSoftware(deviceInfo.SoftwareName) {
		return nil, nil
	}

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
		// 老行补齐硬件指纹：首次由新版客户端访问时把 MachineGuid/IOPlatformUUID/
		// ANDROID_ID 落库，之后同一台机器重装/升级就能靠它认回这一行（不再多占名额）。
		// updateExistingDeviceAccess 内部是 db.Save(device)，整体落盘，这里改结构体即可。
		if hwFP := HardwareFingerprintOf(headers); hwFP != "" && (existingDevice.DeviceHwFp == nil || *existingDevice.DeviceHwFp != hwFP) {
			existingDevice.DeviceHwFp = &hwFP
		}
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

		hwFPSet := HardwareFingerprintOf(headers)
		var hwFPPtr *string
		if hwFPSet != "" {
			hwFPPtr = &hwFPSet
		}

		device := models.Device{
			UserID:            &userIDInt64,
			SubscriptionID:    subscriptionID,
			DeviceFingerprint: deviceHash,
			DeviceHash:        &deviceHash,
			DeviceHwFp:        hwFPPtr,
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

		// 记下首次出现的网络段，供之后「换回旧网络」时认回同一台设备
		addNetworkSegment(&device, ipSegmentOf(ipAddress))

		if err := dm.db.Create(&device).Error; err != nil {
			return nil, err
		}

		deviceCount, _ := CountRecentDevices(dm.db, subscriptionID, DisplayWindow)
		dm.db.Model(&models.Subscription{}).Where("id = ?", subscriptionID).Update("current_devices", deviceCount)

		return &device, nil
	}
}

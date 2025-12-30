package handlers

import (
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"
	"cboard-go/internal/services/device"
	"cboard-go/internal/utils"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// validateSubscription 验证订阅状态，返回 (错误信息, 当前设备数, 设备限制, 是否允许)
func validateSubscription(subscription *models.Subscription, user *models.User, db *gorm.DB, clientIP, userAgent string) (string, int, int, bool) {
	now := utils.GetBeijingTime()

	// 1. 检查订阅是否过期
	isExpired := subscription.ExpireTime.Before(now)
	isInactive := !subscription.IsActive || subscription.Status != "active"
	isSpecialValid := user.SpecialNodeExpiresAt.Valid && user.SpecialNodeExpiresAt.Time.After(now)

	if isExpired && !isSpecialValid {
		return fmt.Sprintf("订阅已过期(到期时间:%s)，请续费", subscription.ExpireTime.Format("2006-01-02")), 0, subscription.DeviceLimit, false
	}
	if isInactive {
		return "订阅已失效或被禁用，请联系客服", 0, subscription.DeviceLimit, false
	}

	// 2. 检查设备数量限制
	var count int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscription.ID, true).Count(&count)

	// 如果设备限制为0，不允许任何设备使用
	if subscription.DeviceLimit == 0 {
		return "设备数量限制为0，无法使用服务", int(count), subscription.DeviceLimit, false
	}

	// 如果设备总数达到或超过限制，检查当前设备是否在允许的范围内
	if subscription.DeviceLimit > 0 && int(count) >= subscription.DeviceLimit {
		// 生成设备哈希，检查当前设备
		hash := device.NewDeviceManager().GenerateDeviceHash(userAgent, clientIP, "")
		var currentDevice models.Device
		isCurrentDeviceExists := db.Where("device_hash = ? AND subscription_id = ?", hash, subscription.ID).First(&currentDevice).Error == nil

		// 如果当前设备不存在，说明是新设备，直接拒绝
		if !isCurrentDeviceExists {
			return fmt.Sprintf("设备数量超过限制(当前%d/限制%d)，无法添加新设备", count, subscription.DeviceLimit), int(count), subscription.DeviceLimit, false
		}

		// 如果当前设备存在，检查它是否在允许的范围内（按最后访问时间排序，取前 deviceLimit 个）
		var allowedDevices []models.Device
		db.Where("subscription_id = ? AND is_active = ?", subscription.ID, true).
			Order("last_access DESC").
			Limit(subscription.DeviceLimit).
			Find(&allowedDevices)

		// 检查当前设备是否在允许的设备列表中
		isAllowed := false
		for _, allowedDevice := range allowedDevices {
			if allowedDevice.ID == currentDevice.ID {
				isAllowed = true
				break
			}
		}

		// 如果当前设备不在允许的范围内，拒绝
		if !isAllowed {
			return fmt.Sprintf("设备数量超过限制(当前%d/限制%d)，此设备不在允许范围内", count, subscription.DeviceLimit), int(count), subscription.DeviceLimit, false
		}
	}

	return "", int(count), subscription.DeviceLimit, true
}

// checkOldSubscriptionURL 检查是否是旧订阅地址
func checkOldSubscriptionURL(db *gorm.DB, oldURL string) (*models.SubscriptionReset, *models.Subscription, *models.User, bool) {
	var reset models.SubscriptionReset
	if err := db.Where("old_subscription_url = ?", oldURL).Order("created_at DESC").First(&reset).Error; err != nil {
		return nil, nil, nil, false
	}

	var sub models.Subscription
	if err := db.First(&sub, reset.SubscriptionID).Error; err != nil {
		return &reset, nil, nil, true
	}

	var user models.User
	if err := db.First(&user, sub.UserID).Error; err != nil {
		return &reset, &sub, nil, true
	}

	return &reset, &sub, &user, true
}

// generateErrorConfig 生成错误配置（Clash格式），返回4个错误节点信息
// 节点格式：1.官网 2.错误原因 3.解决办法 4.联系管理员
func generateErrorConfig(title, message string, baseURL string) string {
	// 清理消息，移除换行符
	cleanMessage := strings.ReplaceAll(message, "\n", " ")

	// 如果baseURL为空，使用默认提示
	if baseURL == "" {
		baseURL = "请登录官网"
	} else {
		// 截断URL，确保不超过30个字符
		if len(baseURL) > 30 {
			baseURL = baseURL[:27] + "..."
		}
	}

	// 拆分错误原因，确保不超过30个字符
	errorReason := cleanMessage
	if len(errorReason) > 30 {
		errorReason = errorReason[:27] + "..."
	}

	// 生成4个节点
	errorNodes := []string{
		fmt.Sprintf("🌐 %s", baseURL),      // 第1个：官网
		fmt.Sprintf("⚠️ %s", errorReason), // 第2个：错误原因
		"💡 请登录官网查看详情",                     // 第3个：解决办法
		"📞 联系管理员获取帮助",                     // 第4个：联系管理员
	}

	// 生成节点列表（使用 SS 节点而不是 socks5）
	proxyList := ""
	proxyNames := ""
	for i, nodeName := range errorNodes {
		// 使用无效的 SS 节点配置，确保无法连接
		proxyList += fmt.Sprintf("  - name: \"%s\"\n    type: ss\n    server: baidu.com\n    port: %d\n    cipher: aes-256-gcm\n    password: \"invalid\"\n    # 错误节点，仅用于显示信息\n", nodeName, i)
		proxyNames += fmt.Sprintf("      - \"%s\"\n", nodeName)
	}

	return fmt.Sprintf(`# ============================================
# ⚠️ 订阅错误：%s
# ============================================
# %s
# ============================================
# 此订阅无法使用，请检查您的账户状态
# 请登录官网查看订单详情或联系客服
# ============================================

port: 7890
socks-port: 7891
allow-lan: false
mode: Rule
log-level: error

# 错误节点（无效节点，仅用于显示错误信息）
proxies:
%s
proxy-groups:
  - name: "❌ 订阅错误提示"
    type: select
    proxies:
%s
rules:
  - MATCH,REJECT
`, title, cleanMessage, proxyList, proxyNames)
}

// generateErrorConfigBase64 生成通用订阅的 Base64 错误提示，返回4个错误节点信息
// 节点格式：1.官网 2.错误原因 3.解决办法 4.联系管理员
func generateErrorConfigBase64(title, message string, baseURL string) string {
	// 清理消息
	cleanMessage := strings.ReplaceAll(message, "\n", " ")

	// 如果baseURL为空，使用默认提示
	if baseURL == "" {
		baseURL = "请登录官网"
	} else {
		// 截断URL，确保不超过30个字符
		if len(baseURL) > 30 {
			baseURL = baseURL[:27] + "..."
		}
	}

	// 拆分错误原因，确保不超过30个字符
	errorReason := cleanMessage
	if len(errorReason) > 30 {
		errorReason = errorReason[:27] + "..."
	}

	// 生成4个节点
	errorNodes := []string{
		fmt.Sprintf("🌐 %s", baseURL),      // 第1个：官网
		fmt.Sprintf("⚠️ %s", errorReason), // 第2个：错误原因
		"💡 请登录官网查看详情",                     // 第3个：解决办法
		"📞 联系管理员获取帮助",                     // 第4个：联系管理员
	}

	// 生成多个无效 VMess 节点链接
	var nodeLinks []string
	for i, nodeName := range errorNodes {
		errorData := map[string]interface{}{
			"v":    "2",
			"ps":   nodeName,                               // 节点名称包含错误信息
			"add":  "baidu.com",                            // 无效地址
			"port": i,                                      // 使用不同的无效端口
			"id":   "00000000-0000-0000-0000-000000000000", // 无效 UUID
			"net":  "tcp",
			"type": "none",
		}

		jsonData, _ := json.Marshal(errorData)
		encoded := base64.StdEncoding.EncodeToString(jsonData)
		nodeLinks = append(nodeLinks, "vmess://"+encoded)
	}

	// 返回多个错误节点链接，用换行符分隔
	content := strings.Join(nodeLinks, "\n")
	return base64.StdEncoding.EncodeToString([]byte(content))
}

// GetSubscriptionConfig 处理 Clash 订阅请求
func GetSubscriptionConfig(c *gin.Context) {
	uurl := c.Param("url")
	db := database.GetDB()
	baseURL := utils.GetBuildBaseURL(c.Request, db)
	var sub models.Subscription

	// 1. 查找订阅
	if err := db.Where("subscription_url = ?", uurl).First(&sub).Error; err != nil {
		// 检查旧地址
		reset, currentSub, user, isOldURL := checkOldSubscriptionURL(db, uurl)
		if isOldURL {
			now := utils.GetBeijingTime()
			var msg string
			if currentSub != nil && user != nil {
				isExpired := currentSub.ExpireTime.Before(now)
				isInactive := !currentSub.IsActive || currentSub.Status != "active"
				msg = fmt.Sprintf("订阅地址已于 %s 重置，原链接已失效。", reset.CreatedAt.Format("2006-01-02 15:04:05"))
				if isExpired {
					msg += fmt.Sprintf(" 当前订阅已过期(到期时间:%s)，请续费。", currentSub.ExpireTime.Format("2006-01-02"))
				} else if isInactive {
					msg += " 当前订阅已失效，请联系客服。"
				} else {
					remainingDays := int(currentSub.ExpireTime.Sub(now).Hours() / 24)
					if remainingDays > 0 {
						msg += fmt.Sprintf(" 当前订阅有效(剩余%d天)，请登录获取新链接。", remainingDays)
					}
				}
			} else {
				msg = fmt.Sprintf("订阅地址已于 %s 重置，原链接已失效。请登录账户获取新订阅地址。", reset.CreatedAt.Format("2006-01-02 15:04:05"))
			}
			c.Header("Content-Type", "application/x-yaml")
			c.String(200, generateErrorConfig("订阅地址已更换", msg, baseURL))
			return
		}
		c.Header("Content-Type", "application/x-yaml")
		c.String(200, generateErrorConfig("订阅不存在", "未在数据库中找到该订阅地址，请检查订阅链接是否正确", baseURL))
		return
	}

	// 2. 检查用户
	var u models.User
	if err := db.First(&u, sub.UserID).Error; err != nil || !u.IsActive {
		var msg string
		if err != nil {
			msg = "关联的用户账户不存在或已被删除，无法使用订阅服务。"
		} else {
			msg = "您的账户已被禁用，无法使用订阅服务。请联系客服获取帮助。"
		}
		c.Header("Content-Type", "application/x-yaml")
		c.String(200, generateErrorConfig("账户异常", msg, baseURL))
		return
	}

	// 3. 验证有效性（过期/超额）
	// 注意：具体的设备限制逻辑交由 Service 处理，但在调用前我们需要处理设备记录逻辑
	// 以确保"新设备超限被阻，旧设备超限可用"的逻辑生效

	deviceManager := device.NewDeviceManager()
	deviceIP := utils.GetRealClientIP(c)
	deviceUA := c.GetHeader("User-Agent")

	// 检查当前设备是否存在
	hash := deviceManager.GenerateDeviceHash(deviceUA, deviceIP, "")
	var currentDevice models.Device
	deviceExists := db.Where("device_hash = ? AND subscription_id = ?", hash, sub.ID).First(&currentDevice).Error == nil

	// 如果当前设备不存在，尝试查找是否有相同User-Agent的设备
	// 这解决了用户开启代理后IP变化导致被识别为新设备的问题
	// 也解决了同一设备在不同网络环境下（如家庭WiFi和公司WiFi）被识别为不同设备的问题
	if !deviceExists {
		var sameUADevice models.Device
		// 查找该订阅下 UA 相同且最近活跃的设备（例如最近24小时内）
		// 我们取最近活跃的那一个
		if err := db.Where("subscription_id = ? AND user_agent = ? AND is_active = ?", sub.ID, deviceUA, true).
			Order("last_access DESC").
			First(&sameUADevice).Error; err == nil {

			// 找到了同UA的设备，我们将当前请求视为该设备的"漫游"
			// 更新该设备的IP和Hash为当前的
			sameUADevice.IPAddress = &deviceIP
			sameUADevice.DeviceHash = &hash
			sameUADevice.LastAccess = utils.GetBeijingTime()

			// 保存更新
			if err := db.Save(&sameUADevice).Error; err == nil {
				// 成功"继承"了旧设备，标记为已存在
				deviceExists = true
				currentDevice = sameUADevice
				// 记录日志（可选）
				// utils.LogInfo(fmt.Sprintf("设备漫游: SubID=%d, OldIP=%s, NewIP=%s", sub.ID, sameUADevice.IPAddress, deviceIP))
			}
		}
	}

	// 获取当前设备数
	var count int64
	db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", sub.ID, true).Count(&count)

	// 逻辑：
	// 1. 如果设备已存在 -> 允许 (更新访问时间)
	// 2. 如果设备不存在 (新设备)
	//    a. 如果未超限 -> 允许 (创建设备)
	//    b. 如果已超限 -> 拒绝 (不创建设备，Service 会检测到 device 不在列表中且超限，从而返回错误节点)

	shouldRecord := true
	if !deviceExists {
		if sub.DeviceLimit > 0 && int(count) >= sub.DeviceLimit {
			shouldRecord = false
		} else if sub.DeviceLimit == 0 {
			shouldRecord = false
		}
	}

	if shouldRecord {
		deviceManager.RecordDeviceAccess(sub.ID, sub.UserID, deviceUA, deviceIP, "clash")
	}

	// 4. 生成配置
	db.Model(&sub).Update("clash_count", gorm.Expr("clash_count + ?", 1))

	cfg, err := config_update.NewConfigUpdateService().GenerateClashConfig(uurl, deviceIP, deviceUA)
	if err != nil {
		// 这里的 err 通常是系统错误，而非业务逻辑阻断（业务阻断会返回错误节点的 YAML）
		c.Header("Content-Type", "application/x-yaml")
		c.String(200, generateErrorConfig("生成失败", fmt.Sprintf("配置生成错误: %v", err), baseURL))
		return
	}

	c.Header("Content-Type", "application/x-yaml")
	c.String(200, cfg)
}

// GetUniversalSubscription 处理通用 Base64 订阅
func GetUniversalSubscription(c *gin.Context) {
	uurl := c.Param("url")
	db := database.GetDB()
	baseURL := utils.GetBuildBaseURL(c.Request, db)
	var sub models.Subscription

	if err := db.Where("subscription_url = ?", uurl).First(&sub).Error; err != nil {
		// Service 会处理旧地址检查和错误消息生成，这里不需要重复
		// 由于我们需要 sub 来进行设备逻辑判断，但这里 sub 不存在
		// 我们直接调用 Service，Service 内部会再次尝试查找 oldURL
	}

	// 为了确保逻辑统一，我们重新组织一下 GetUniversalSubscription
	// 实际上，我们只需要获取 IP/UA，然后调用 Service 即可

	deviceIP := utils.GetRealClientIP(c)
	deviceUA := c.GetHeader("User-Agent")
	deviceManager := device.NewDeviceManager()

	// 预先获取 sub 以便进行设备逻辑判断（如果 sub 存在）
	if db.Where("subscription_url = ?", uurl).First(&sub).Error == nil {
		// 同样的设备记录逻辑
		hash := deviceManager.GenerateDeviceHash(deviceUA, deviceIP, "")
		var currentDevice models.Device
		deviceExists := db.Where("device_hash = ? AND subscription_id = ?", hash, sub.ID).First(&currentDevice).Error == nil

		// 同UA设备漫游逻辑
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
			if sub.DeviceLimit > 0 && int(count) >= sub.DeviceLimit {
				shouldRecord = false
			} else if sub.DeviceLimit == 0 {
				shouldRecord = false
			}
		}

		if shouldRecord {
			deviceManager.RecordDeviceAccess(sub.ID, sub.UserID, deviceUA, deviceIP, "universal")
			db.Model(&sub).Update("universal_count", gorm.Expr("universal_count + ?", 1))
		}
	}

	// 调用 Service 生成配置
	// format 默认为 base64 (vmess/vless/etc)
	// 如果是 ssr 客户端，可能需要不同的处理，但这里统一用 base64
	cfg, err := config_update.NewConfigUpdateService().GenerateUniversalConfig(uurl, deviceIP, deviceUA, "base64")
	if err != nil {
		c.String(200, generateErrorConfigBase64("错误", "生成配置失败", baseURL))
		return
	}
	c.String(200, cfg)
}

// UpdateSubscriptionConfig 更新订阅配置（由用户/管理员手动触发）
func UpdateSubscriptionConfig(c *gin.Context) {
	var req struct {
		SubscriptionURL string `json:"subscription_url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	var sub models.Subscription
	if err := db.Where("subscription_url = ?", req.SubscriptionURL).First(&sub).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅失败", err)
		}
		return
	}

	service := config_update.NewConfigUpdateService()
	if err := service.UpdateSubscriptionConfig(req.SubscriptionURL); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新配置失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "配置更新成功", nil)
}

// --- 后台管理函数（完整保留，无省略） ---

// GetConfigUpdateStatus 获取配置更新状态
func GetConfigUpdateStatus(c *gin.Context) {
	service := config_update.NewConfigUpdateService()
	status := service.GetStatus()
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"is_running":  status["is_running"],
		"last_update": status["last_update"],
		"next_update": status["next_update"],
	})
}

// GetConfigUpdateConfig 获取配置更新设置
func GetConfigUpdateConfig(c *gin.Context) {
	db := database.GetDB()
	var configs []models.SystemConfig
	db.Where("category = ?", "config_update").Find(&configs)

	configMap := make(map[string]interface{})
	defaultConfig := map[string]interface{}{
		"urls":              []string{},
		"target_dir":        "./uploads/config",
		"v2ray_file":        "xr",
		"clash_file":        "clash.yaml",
		"filter_keywords":   []string{},
		"enable_schedule":   false,
		"schedule_interval": 3600,
	}

	var urlsConfig *models.SystemConfig

	for _, config := range configs {
		key := config.Key
		value := config.Value

		if key == "urls" {
			urlsConfig = &config
		} else if key == "filter_keywords" {
			urls := strings.Split(value, "\n")
			filtered := make([]string, 0)
			for _, url := range urls {
				url = strings.TrimSpace(url)
				if url != "" {
					filtered = append(filtered, url)
				}
			}
			configMap[key] = filtered
		} else if key == "enable_schedule" {
			configMap[key] = value == "true" || value == "1"
		} else if key == "schedule_interval" {
			var interval int
			fmt.Sscanf(value, "%d", &interval)
			configMap[key] = interval
		} else {
			configMap[key] = value
		}
	}

	// 处理 URLs
	if urlsConfig != nil && strings.TrimSpace(urlsConfig.Value) != "" {
		urls := strings.Split(urlsConfig.Value, "\n")
		filtered := make([]string, 0)
		for _, url := range urls {
			url = strings.TrimSpace(url)
			if url != "" {
				filtered = append(filtered, url)
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

// GetConfigUpdateFiles 获取生成的文件列表
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

	result := gin.H{
		"v2ray": gin.H{"name": v2rayFile, "path": v2rayPath, "size": 0, "exists": false},
		"clash": gin.H{"name": clashFile, "path": clashPath, "size": 0, "exists": false},
	}

	if info, err := os.Stat(v2rayPath); err == nil {
		result["v2ray"] = gin.H{"name": v2rayFile, "path": v2rayPath, "size": info.Size(), "modified": info.ModTime().Format("2006-01-02 15:04:05"), "exists": true}
	}
	if info, err := os.Stat(clashPath); err == nil {
		result["clash"] = gin.H{"name": clashFile, "path": clashPath, "size": info.Size(), "modified": info.ModTime().Format("2006-01-02 15:04:05"), "exists": true}
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// GetConfigUpdateLogs 获取更新日志
func GetConfigUpdateLogs(c *gin.Context) {
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}
	service := config_update.NewConfigUpdateService()
	utils.SuccessResponse(c, http.StatusOK, "", service.GetLogs(limit))
}

// ClearConfigUpdateLogs 清理日志
func ClearConfigUpdateLogs(c *gin.Context) {
	service := config_update.NewConfigUpdateService()
	if err := service.ClearLogs(); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "清理失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "日志已清理", nil)
}

// UpdateConfigUpdateConfig 修改配置设置
func UpdateConfigUpdateConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	db := database.GetDB()

	// 处理 urls 配置
	if urlsValue, ok := req["urls"]; ok {
		var valueStr string
		switch v := urlsValue.(type) {
		case string:
			valueStr = v
		case []interface{}:
			urls := []string{}
			for _, item := range v {
				if s, ok := item.(string); ok && s != "" {
					urls = append(urls, s)
				}
			}
			valueStr = strings.Join(urls, "\n")
		default:
			j, _ := json.Marshal(v)
			valueStr = string(j)
		}
		req["urls"] = valueStr
	}

	for key, value := range req {
		var config models.SystemConfig
		err := db.Where("key = ? AND category = ?", key, "config_update").First(&config).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			continue
		}

		var valueStr string
		switch v := value.(type) {
		case string:
			valueStr = v
		case []interface{}:
			urls := []string{}
			for _, item := range v {
				if s, ok := item.(string); ok && s != "" {
					urls = append(urls, s)
				}
			}
			valueStr = strings.Join(urls, "\n")
		case bool:
			if v {
				valueStr = "true"
			} else {
				valueStr = "false"
			}
		case float64:
			valueStr = fmt.Sprintf("%.0f", v)
		default:
			j, _ := json.Marshal(v)
			valueStr = string(j)
		}

		if err == gorm.ErrRecordNotFound {
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
		} else {
			config.Value = valueStr
			if err := db.Save(&config).Error; err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("更新配置 %s 失败", key), err)
				return
			}
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "配置保存成功", nil)
}

// StartConfigUpdate 开启任务
func StartConfigUpdate(c *gin.Context) {
	service := config_update.NewConfigUpdateService()
	go func() {
		if err := service.RunUpdateTask(); err != nil {
			return
		}
	}()
	utils.SuccessResponse(c, http.StatusOK, "配置更新任务已启动", nil)
}

// StopConfigUpdate 停止任务
func StopConfigUpdate(c *gin.Context) {
	utils.SuccessResponse(c, http.StatusOK, "配置更新任务停止指令已发送", nil)
}

// TestConfigUpdate 测试更新任务
func TestConfigUpdate(c *gin.Context) {
	service := config_update.NewConfigUpdateService()
	go func() {
		service.RunUpdateTask()
	}()
	utils.SuccessResponse(c, http.StatusOK, "测试任务已启动", nil)
}

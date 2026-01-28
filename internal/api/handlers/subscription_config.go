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

func validateSubscription(subscription *models.Subscription, user *models.User, db *gorm.DB, clientIP, userAgent string) (string, int, int, bool) {
	now := utils.GetBeijingTime()

	isExpired := subscription.ExpireTime.Before(now)
	isInactive := !subscription.IsActive || subscription.Status != "active"
	isSpecialValid := user.SpecialNodeExpiresAt.Valid && user.SpecialNodeExpiresAt.Time.After(now)

	if isExpired && !isSpecialValid {
		return fmt.Sprintf("订阅已过期(到期时间:%s)，请续费", subscription.ExpireTime.Format("2006-01-02")), 0, subscription.DeviceLimit, false
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

func generateErrorConfig(title, message string, baseURL string) string {
	cleanMessage := strings.ReplaceAll(message, "\n", " ")

	if baseURL == "" {
		baseURL = "请登录官网"
	} else {
		if len(baseURL) > 30 {
			baseURL = baseURL[:27] + "..."
		}
	}

	errorReason := cleanMessage
	if len(errorReason) > 30 {
		errorReason = errorReason[:27] + "..."
	}

	errorNodes := []string{
		fmt.Sprintf("🌐 %s", baseURL),      // 第1个：官网
		fmt.Sprintf("⚠️ %s", errorReason), // 第2个：错误原因
		"💡 请登录官网查看详情",                     // 第3个：解决办法
		"📞 联系管理员获取帮助",                     // 第4个：联系管理员
	}

	proxyList := ""
	proxyNames := ""
	for i, nodeName := range errorNodes {
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

func generateErrorConfigBase64(title, message string, baseURL string) string {
	cleanMessage := strings.ReplaceAll(message, "\n", " ")

	if baseURL == "" {
		baseURL = "请登录官网"
	} else {
		if len(baseURL) > 30 {
			baseURL = baseURL[:27] + "..."
		}
	}

	errorReason := cleanMessage
	if len(errorReason) > 30 {
		errorReason = errorReason[:27] + "..."
	}

	errorNodes := []string{
		fmt.Sprintf("🌐 %s", baseURL),      // 第1个：官网
		fmt.Sprintf("⚠️ %s", errorReason), // 第2个：错误原因
		"💡 请登录官网查看详情",                     // 第3个：解决办法
		"📞 联系管理员获取帮助",                     // 第4个：联系管理员
	}

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

	content := strings.Join(nodeLinks, "\n")
	return base64.StdEncoding.EncodeToString([]byte(content))
}

func GetSubscriptionConfig(c *gin.Context) {
	uurl := c.Param("url")
	db := database.GetDB()
	baseURL := utils.GetBuildBaseURL(c.Request, db)
	var sub models.Subscription

	if err := db.Where("subscription_url = ?", uurl).First(&sub).Error; err != nil {
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

	deviceManager := device.NewDeviceManager()
	deviceIP := utils.GetRealClientIP(c)
	deviceUA := c.GetHeader("User-Agent")

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
		if sub.DeviceLimit > 0 && int(count) >= sub.DeviceLimit {
			shouldRecord = false
		} else if sub.DeviceLimit == 0 {
			shouldRecord = false
		}
	}

	if shouldRecord {
		deviceManager.RecordDeviceAccess(sub.ID, sub.UserID, deviceUA, deviceIP, "clash")
	}

	db.Model(&sub).Update("clash_count", gorm.Expr("clash_count + ?", 1))

	cfg, err := config_update.NewConfigUpdateService().GenerateClashConfig(uurl, deviceIP, deviceUA)
	if err != nil {
		c.Header("Content-Type", "application/x-yaml")
		c.String(200, generateErrorConfig("生成失败", fmt.Sprintf("配置生成错误: %v", err), baseURL))
		return
	}

	c.Header("Content-Type", "application/x-yaml")
	c.String(200, cfg)
}

func GetUniversalSubscription(c *gin.Context) {
	uurl := c.Param("url")
	db := database.GetDB()
	baseURL := utils.GetBuildBaseURL(c.Request, db)
	var sub models.Subscription

	if err := db.Where("subscription_url = ?", uurl).First(&sub).Error; err != nil {
	}

	deviceIP := utils.GetRealClientIP(c)
	deviceUA := c.GetHeader("User-Agent")
	deviceManager := device.NewDeviceManager()

	if db.Where("subscription_url = ?", uurl).First(&sub).Error == nil {
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

	cfg, err := config_update.NewConfigUpdateService().GenerateUniversalConfig(uurl, deviceIP, deviceUA, "base64")
	if err != nil {
		c.String(200, generateErrorConfigBase64("错误", "生成配置失败", baseURL))
		return
	}
	c.String(200, cfg)
}

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
	utils.SuccessResponse(c, http.StatusOK, "日志已清理", nil)
}

func UpdateConfigUpdateConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	db := database.GetDB()

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

func StartConfigUpdate(c *gin.Context) {
	service := config_update.NewConfigUpdateService()
	go func() {
		if err := service.RunUpdateTask(); err != nil {
			return
		}
	}()
	utils.SuccessResponse(c, http.StatusOK, "配置更新任务已启动", nil)
}

func StopConfigUpdate(c *gin.Context) {
	utils.SuccessResponse(c, http.StatusOK, "配置更新任务停止指令已发送", nil)
}

func TestConfigUpdate(c *gin.Context) {
	service := config_update.NewConfigUpdateService()
	go func() {
		service.RunUpdateTask()
	}()
	utils.SuccessResponse(c, http.StatusOK, "测试任务已启动", nil)
}

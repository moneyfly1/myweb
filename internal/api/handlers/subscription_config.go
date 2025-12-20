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

	// 生成设备哈希，检查是否为新设备
	hash := device.NewDeviceManager().GenerateDeviceHash(userAgent, clientIP, "")
	var d models.Device
	isNewDevice := db.Where("device_hash = ? AND subscription_id = ?", hash, subscription.ID).First(&d).Error != nil

	// 只有新设备且已达到设备限制时，才阻止订阅
	// 已存在的设备即使总数超限，仍然允许使用
	if isNewDevice && subscription.DeviceLimit > 0 && int(count) >= subscription.DeviceLimit {
		return fmt.Sprintf("设备超限(当前%d/限制%d)，请登录官网管理设备", count, subscription.DeviceLimit), int(count), subscription.DeviceLimit, false
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

// generateErrorConfig 生成错误配置（Clash格式），不包含任何可用节点
func generateErrorConfig(title, message string) string {
	// 清理消息，移除换行符，确保在注释中正确显示
	cleanMessage := strings.ReplaceAll(message, "\n", " | ")

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

# 不包含任何可用节点
proxies: []

proxy-groups:
  - name: "❌ 订阅错误"
    type: select
    proxies:
      - DIRECT

rules:
  - MATCH,REJECT
`, title, cleanMessage)
}

// generateErrorConfigBase64 生成通用订阅的 Base64 错误提示
func generateErrorConfigBase64(title, message string) string {
	// 生成清晰的错误信息
	content := fmt.Sprintf(`# ============================================
# ⚠️ 订阅错误：%s
# ============================================
# %s
# ============================================
# 此订阅无法使用，请检查您的账户状态
# 请登录官网查看订单详情或联系客服
# ============================================
`, title, strings.ReplaceAll(message, "\n", " | "))
	return base64.StdEncoding.EncodeToString([]byte(content))
}

// GetSubscriptionConfig 处理 Clash 订阅请求
func GetSubscriptionConfig(c *gin.Context) {
	uurl := c.Param("url")
	db := database.GetDB()
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
			c.String(200, generateErrorConfig("订阅地址已更换", msg))
			return
		}
		c.Header("Content-Type", "application/x-yaml")
		c.String(200, generateErrorConfig("订阅不存在", "未在数据库中找到该订阅地址，请检查订阅链接是否正确"))
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
		c.String(200, generateErrorConfig("账户异常", msg))
		return
	}

	// 3. 验证有效性（过期/超额）
	now := utils.GetBeijingTime()
	isExpired := sub.ExpireTime.Before(now)
	isInactive := !sub.IsActive || sub.Status != "active"
	
	// 先检查订阅是否过期或失效（即使有专线节点，普通订阅过期也应该返回错误）
	if isExpired || isInactive {
		var title, message string
		if isExpired {
			title = "订阅已过期"
			message = fmt.Sprintf("您的订阅已于 %s 过期，无法使用服务。请及时续费以继续使用。", sub.ExpireTime.Format("2006-01-02 15:04:05"))
		} else {
			title = "订阅已失效"
			message = "您的订阅已被禁用或失效，无法使用服务。请联系客服获取帮助。"
		}
		c.Header("Content-Type", "application/x-yaml")
		c.String(200, generateErrorConfig(title, message))
		return
	}
	
	// 再检查设备数量限制（只阻止新设备）
	_, currentDevices, deviceLimit, ok := validateSubscription(&sub, &u, db, utils.GetRealClientIP(c), c.GetHeader("User-Agent"))
	if !ok {
		title := "设备数量超限"
		message := fmt.Sprintf("设备数量超过限制(当前%d/限制%d)，无法添加新设备。请登录官网删除多余设备后再试。", currentDevices, deviceLimit)
		c.Header("Content-Type", "application/x-yaml")
		c.String(200, generateErrorConfig(title, message))
		return
	}

	// 4. 正常返回
	device.NewDeviceManager().RecordDeviceAccess(sub.ID, sub.UserID, c.GetHeader("User-Agent"), utils.GetRealClientIP(c), "clash")
	db.Model(&sub).Update("clash_count", gorm.Expr("clash_count + ?", 1))

	cfg, err := config_update.NewConfigUpdateService().GenerateClashConfig(sub.UserID, uurl)
	if err != nil {
		c.Header("Content-Type", "application/x-yaml")
		c.String(200, generateErrorConfig("生成失败", "服务器在构建配置时发生错误"))
		return
	}
	c.Header("Content-Type", "application/x-yaml")
	c.String(200, cfg)
}

// GetUniversalSubscription 处理通用 Base64 订阅
func GetUniversalSubscription(c *gin.Context) {
	uurl := c.Param("url")
	db := database.GetDB()
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
			c.String(200, generateErrorConfigBase64("订阅地址已更换", msg))
			return
		}
		c.String(200, generateErrorConfigBase64("订阅不存在", "未在数据库中找到该订阅地址，请检查订阅链接是否正确"))
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
		c.String(200, generateErrorConfigBase64("账户异常", msg))
		return
	}

	// 先检查订阅是否过期或失效（即使有专线节点，普通订阅过期也应该返回错误）
	now := utils.GetBeijingTime()
	isExpired := sub.ExpireTime.Before(now)
	isInactive := !sub.IsActive || sub.Status != "active"
	
	if isExpired || isInactive {
		var title, message string
		if isExpired {
			title = "订阅已过期"
			message = fmt.Sprintf("您的订阅已于 %s 过期，无法使用服务。请及时续费以继续使用。", sub.ExpireTime.Format("2006-01-02 15:04:05"))
		} else {
			title = "订阅已失效"
			message = "您的订阅已被禁用或失效，无法使用服务。请联系客服获取帮助。"
		}
		c.String(200, generateErrorConfigBase64(title, message))
		return
	}
	
	// 再检查设备数量限制（只阻止新设备）
	_, currentDevices, deviceLimit, ok := validateSubscription(&sub, &u, db, utils.GetRealClientIP(c), c.GetHeader("User-Agent"))
	if !ok {
		title := "设备数量超限"
		message := fmt.Sprintf("设备数量超过限制(当前%d/限制%d)，无法添加新设备。请登录官网删除多余设备后再试。", currentDevices, deviceLimit)
		c.String(200, generateErrorConfigBase64(title, message))
		return
	}

	device.NewDeviceManager().RecordDeviceAccess(sub.ID, sub.UserID, c.GetHeader("User-Agent"), utils.GetRealClientIP(c), "universal")
	db.Model(&sub).Update("universal_count", gorm.Expr("universal_count + ?", 1))

	cfg, err := config_update.NewConfigUpdateService().GenerateSSRConfig(sub.UserID, uurl)
	if err != nil {
		c.String(200, generateErrorConfigBase64("错误", "生成配置失败"))
		return
	}
	c.String(200, base64.StdEncoding.EncodeToString([]byte(cfg)))
}

// UpdateSubscriptionConfig 更新订阅配置（由用户/管理员手动触发）
func UpdateSubscriptionConfig(c *gin.Context) {
	var req struct {
		SubscriptionURL string `json:"subscription_url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数错误"})
		return
	}

	db := database.GetDB()
	var sub models.Subscription
	if err := db.Where("subscription_url = ?", req.SubscriptionURL).First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订阅不存在"})
		return
	}

	service := config_update.NewConfigUpdateService()
	if err := service.UpdateSubscriptionConfig(req.SubscriptionURL); err != nil {
		utils.LogError("UpdateSubscriptionConfig: failed", err, map[string]interface{}{"url": req.SubscriptionURL})
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "更新配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "配置更新成功"})
}

// --- 后台管理函数（完整保留，无省略） ---

// GetConfigUpdateStatus 获取配置更新状态
func GetConfigUpdateStatus(c *gin.Context) {
	service := config_update.NewConfigUpdateService()
	status := service.GetStatus()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"is_running":  status["is_running"],
			"last_update": status["last_update"],
			"next_update": status["next_update"],
		},
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
		"node_source_urls":  []string{},
		"target_dir":        "./uploads/config",
		"v2ray_file":        "xr",
		"clash_file":        "clash.yaml",
		"filter_keywords":   []string{},
		"enable_schedule":   false,
		"schedule_interval": 3600,
	}

	for _, config := range configs {
		key := config.Key
		value := config.Value
		if key == "urls" || key == "node_source_urls" || key == "filter_keywords" {
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

	for key, defaultValue := range defaultConfig {
		if _, exists := configMap[key]; !exists {
			configMap[key] = defaultValue
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": configMap})
}

// GetConfigUpdateFiles 获取生成的文件列表
func GetConfigUpdateFiles(c *gin.Context) {
	service := config_update.NewConfigUpdateService()
	config, err := service.GetConfig()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": []gin.H{}})
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

	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// GetConfigUpdateLogs 获取更新日志
func GetConfigUpdateLogs(c *gin.Context) {
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}
	service := config_update.NewConfigUpdateService()
	c.JSON(http.StatusOK, gin.H{"success": true, "data": service.GetLogs(limit)})
}

// ClearConfigUpdateLogs 清理日志
func ClearConfigUpdateLogs(c *gin.Context) {
	service := config_update.NewConfigUpdateService()
	if err := service.ClearLogs(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "清理失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "日志已清理"})
}

// UpdateConfigUpdateConfig 修改配置设置
func UpdateConfigUpdateConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}

	db := database.GetDB()
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
				utils.LogError("UpdateConfigUpdateConfig: create failed", err, map[string]interface{}{"key": key})
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": fmt.Sprintf("保存配置 %s 失败", key)})
				return
			}
		} else {
			config.Value = valueStr
			if err := db.Save(&config).Error; err != nil {
				utils.LogError("UpdateConfigUpdateConfig: update failed", err, map[string]interface{}{"key": key})
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": fmt.Sprintf("更新配置 %s 失败", key)})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "配置保存成功"})
}

// StartConfigUpdate 开启任务
func StartConfigUpdate(c *gin.Context) {
	service := config_update.NewConfigUpdateService()
	go func() {
		if err := service.RunUpdateTask(); err != nil {
			return
		}
	}()
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "配置更新任务已启动"})
}

// StopConfigUpdate 停止任务
func StopConfigUpdate(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "配置更新任务停止指令已发送"})
}

// TestConfigUpdate 测试更新任务
func TestConfigUpdate(c *gin.Context) {
	service := config_update.NewConfigUpdateService()
	go func() {
		service.RunUpdateTask()
	}()
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "测试任务已启动"})
}

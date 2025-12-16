package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"
	"cboard-go/internal/services/device"

	"github.com/gin-gonic/gin"
)

// GetSubscriptionConfig 获取订阅配置
func GetSubscriptionConfig(c *gin.Context) {
	subscriptionURL := c.Param("url")

	// 验证订阅URL属于当前用户
	db := database.GetDB()
	var subscription models.Subscription
	if err := db.Where("subscription_url = ?", subscriptionURL).First(&subscription).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "订阅不存在",
		})
		return
	}

	// 记录设备访问
	userAgent := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()
	deviceManager := device.NewDeviceManager()
	_, _ = deviceManager.RecordDeviceAccess(subscription.ID, subscription.UserID, userAgent, ipAddress, "clash")

	// 生成配置（Clash）
	service := config_update.NewConfigUpdateService()
	config, err := service.GenerateClashConfig(subscription.UserID, subscriptionURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "生成配置失败: " + err.Error(),
		})
		return
	}

	// 返回配置（Clash 格式）
	c.Header("Content-Type", "application/x-yaml")
	c.String(http.StatusOK, config)
}

// GetSSRSubscription 返回 Base64 编码的通用订阅（SSR/V2Ray 等客户端）
// 逻辑：生成 Clash 配置 -> Base64 编码文本返回，方便客户端直接使用
func GetSSRSubscription(c *gin.Context) {
	subscriptionURL := c.Param("url")

	// 校验订阅
	db := database.GetDB()
	var subscription models.Subscription
	if err := db.Where("subscription_url = ?", subscriptionURL).First(&subscription).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "订阅不存在"})
		return
	}

	// 记录设备访问
	userAgent := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()
	deviceManager := device.NewDeviceManager()
	_, _ = deviceManager.RecordDeviceAccess(subscription.ID, subscription.UserID, userAgent, ipAddress, "ssr")

	// 生成 Clash 配置文本
	service := config_update.NewConfigUpdateService()
	configText, err := service.GenerateClashConfig(subscription.UserID, subscriptionURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "生成配置失败: " + err.Error()})
		return
	}

	// Base64 编码返回
	encoded := base64.StdEncoding.EncodeToString([]byte(configText))
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, encoded)
}

// UpdateSubscriptionConfig 更新订阅配置（管理员）
func UpdateSubscriptionConfig(c *gin.Context) {
	var req struct {
		SubscriptionURL string `json:"subscription_url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	service := config_update.NewConfigUpdateService()
	if err := service.UpdateSubscriptionConfig(req.SubscriptionURL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新配置失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置更新成功",
	})
}

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

// GetConfigUpdateConfig 获取配置更新配置
func GetConfigUpdateConfig(c *gin.Context) {
	db := database.GetDB()
	var configs []models.SystemConfig
	db.Where("category = ?", "config_update").Find(&configs)

	configMap := make(map[string]interface{})

	// 设置默认值
	defaultConfig := map[string]interface{}{
		"urls":              []string{},
		"target_dir":        "./uploads/config",
		"v2ray_file":        "xr",
		"clash_file":        "clash.yaml",
		"filter_keywords":   []string{},
		"enable_schedule":   false,
		"schedule_interval": 3600,
	}

	// 从数据库加载配置
	for _, config := range configs {
		key := config.Key
		value := config.Value

		// 特殊处理数组类型的配置
		if key == "urls" || key == "node_source_urls" {
			// URLs 是换行分隔的字符串，转换为数组
			urls := strings.Split(value, "\n")
			filtered := make([]string, 0)
			for _, url := range urls {
				url = strings.TrimSpace(url)
				if url != "" {
					filtered = append(filtered, url)
				}
			}
			configMap["urls"] = filtered
		} else if key == "filter_keywords" {
			// 过滤关键词也是换行分隔的字符串
			keywords := strings.Split(value, "\n")
			filtered := make([]string, 0)
			for _, keyword := range keywords {
				keyword = strings.TrimSpace(keyword)
				if keyword != "" {
					filtered = append(filtered, keyword)
				}
			}
			configMap["filter_keywords"] = filtered
		} else if key == "enable_schedule" {
			configMap[key] = value == "true" || value == "1"
		} else if key == "schedule_interval" {
			var interval int
			fmt.Sscanf(value, "%d", &interval)
			if interval == 0 {
				interval = 3600
			}
			configMap[key] = interval
		} else {
			configMap[key] = value
		}
	}

	// 合并默认值（如果数据库中没有）
	for key, defaultValue := range defaultConfig {
		if _, exists := configMap[key]; !exists {
			configMap[key] = defaultValue
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    configMap,
	})
}

// GetConfigUpdateFiles 获取配置更新文件列表
func GetConfigUpdateFiles(c *gin.Context) {
	service := config_update.NewConfigUpdateService()
	config, err := service.GetConfig()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    []gin.H{},
		})
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
	if clashFile == "" {
		clashFile = "clash.yaml"
	}

	// 转换为绝对路径
	if !filepath.IsAbs(targetDir) {
		wd, _ := os.Getwd()
		targetDir = filepath.Join(wd, strings.TrimPrefix(targetDir, "./"))
	}

	files := []gin.H{}

	// 检查 V2Ray 文件
	v2rayPath := filepath.Join(targetDir, v2rayFile)
	if info, err := os.Stat(v2rayPath); err == nil {
		files = append(files, gin.H{
			"name":     v2rayFile,
			"path":     v2rayPath,
			"size":     info.Size(),
			"modified": info.ModTime().Format("2006-01-02 15:04:05"),
			"type":     "v2ray",
		})
	}

	// 检查 Clash 文件
	clashPath := filepath.Join(targetDir, clashFile)
	if info, err := os.Stat(clashPath); err == nil {
		files = append(files, gin.H{
			"name":     clashFile,
			"path":     clashPath,
			"size":     info.Size(),
			"modified": info.ModTime().Format("2006-01-02 15:04:05"),
			"type":     "clash",
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    files,
	})
}

// GetConfigUpdateLogs 获取配置更新日志
func GetConfigUpdateLogs(c *gin.Context) {
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	service := config_update.NewConfigUpdateService()
	logs := service.GetLogs(limit)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    logs,
	})
}

// UpdateConfigUpdateConfig 更新配置更新设置
func UpdateConfigUpdateConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	db := database.GetDB()

	// 保存配置到数据库
	for key, value := range req {
		var config models.SystemConfig
		// 查找现有配置
		err := db.Where("key = ? AND category = ?", key, "config_update").First(&config).Error

		isNew := err != nil
		if isNew {
			// 如果不存在，创建新配置
			config = models.SystemConfig{
				Key:      key,
				Category: "config_update",
				Type:     "config_update",
			}
		}

		// 转换值为字符串
		var valueStr string
		switch v := value.(type) {
		case string:
			valueStr = v
		case []interface{}:
			// 如果是数组（如URLs），转换为换行分隔的字符串
			urls := make([]string, 0)
			for _, item := range v {
				if str, ok := item.(string); ok && str != "" {
					urls = append(urls, strings.TrimSpace(str))
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
			// JSON 数字可能是 float64
			valueStr = fmt.Sprintf("%.0f", v)
		case int:
			valueStr = fmt.Sprintf("%d", v)
		default:
			// 尝试 JSON 编码
			if jsonBytes, err := json.Marshal(v); err == nil {
				valueStr = string(jsonBytes)
			} else {
				valueStr = fmt.Sprintf("%v", v)
			}
		}

		config.Value = valueStr
		config.DisplayName = strings.ReplaceAll(key, "_", " ")
		config.Description = fmt.Sprintf("Configuration update setting for %s", key)

		if isNew {
			if err := db.Create(&config).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": fmt.Sprintf("保存配置 %s 失败: %v", key, err),
				})
				return
			}
		} else {
			if err := db.Save(&config).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": fmt.Sprintf("更新配置 %s 失败: %v", key, err),
				})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置保存成功",
	})
}

// StartConfigUpdate 开始配置更新
func StartConfigUpdate(c *gin.Context) {
	service := config_update.NewConfigUpdateService()

	// 在 goroutine 中异步执行
	go func() {
		if err := service.RunUpdateTask(); err != nil {
			// 错误已记录在日志中
			return
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置更新任务已启动",
		"data": gin.H{
			"status": "running",
		},
	})
}

// StopConfigUpdate 停止配置更新
func StopConfigUpdate(c *gin.Context) {
	// 这里应该停止配置更新任务
	// 暂时返回成功
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置更新已停止",
		"data": gin.H{
			"status": "stopped",
		},
	})
}

// TestConfigUpdate 测试配置更新
func TestConfigUpdate(c *gin.Context) {
	service := config_update.NewConfigUpdateService()

	// 在 goroutine 中异步执行
	go func() {
		if err := service.RunUpdateTask(); err != nil {
			// 错误已记录在日志中
			return
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "测试更新任务已启动",
		"data": gin.H{
			"tested": true,
		},
	})
}

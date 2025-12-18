package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetSystemConfigs 获取系统配置
func GetSystemConfigs(c *gin.Context) {
	category := c.Query("category")
	isPublic := c.Query("is_public") == "true"

	db := database.GetDB()
	var configs []models.SystemConfig
	query := db

	if category != "" {
		query = query.Where("category = ?", category)
	}
	if isPublic {
		query = query.Where("is_public = ?", true)
	}

	if err := query.Order("sort_order ASC").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取配置失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    configs,
	})
}

// GetSystemConfig 获取单个配置
func GetSystemConfig(c *gin.Context) {
	key := c.Param("key")

	db := database.GetDB()
	var config models.SystemConfig
	if err := db.Where("key = ?", key).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "配置不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
	})
}

// UpdateSystemConfig 更新系统配置（管理员）- 支持单个和批量更新
func UpdateSystemConfig(c *gin.Context) {
	key := c.Param("key")

	// 如果 key 是 "batch"，则批量更新
	if key == "batch" {
		var req map[string]interface{}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "请求参数错误",
			})
			return
		}

		db := database.GetDB()
		for k, v := range req {
			var config models.SystemConfig
			if err := db.Where("key = ?", k).First(&config).Error; err != nil {
				// 如果不存在，创建新配置
				config = models.SystemConfig{
					Key:      k,
					Value:    fmt.Sprintf("%v", v),
					Category: "system",
				}
				db.Create(&config)
			} else {
				// 更新现有配置
				config.Value = fmt.Sprintf("%v", v)
				db.Save(&config)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "批量更新成功",
		})
		return
	}

	// 单个更新
	var req struct {
		Value string `json:"value" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	var config models.SystemConfig
	if err := db.Where("key = ?", key).First(&config).Error; err != nil {
		// 如果不存在，创建新配置
		config = models.SystemConfig{
			Key:      key,
			Value:    req.Value,
			Category: "system",
		}
		if err := db.Create(&config).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "创建配置失败",
			})
			return
		}
	} else {
		// 更新现有配置
		config.Value = req.Value
		if err := db.Save(&config).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "更新配置失败",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "更新成功",
		"data":    config,
	})
}

// CreateSystemConfig 创建系统配置（管理员）
func CreateSystemConfig(c *gin.Context) {
	var req struct {
		Key      string `json:"key" binding:"required"`
		Value    string `json:"value" binding:"required"`
		Category string `json:"category"`
		IsPublic bool   `json:"is_public"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()

	// 检查是否已存在（基于 key 和 category 的组合）
	var existing models.SystemConfig
	query := db.Where("key = ?", req.Key)
	if req.Category != "" {
		query = query.Where("category = ?", req.Category)
	}
	if err := query.First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "配置已存在",
		})
		return
	}

	config := models.SystemConfig{
		Key:      req.Key,
		Value:    req.Value,
		Category: req.Category,
		IsPublic: req.IsPublic,
	}

	if err := db.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建配置失败",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    config,
	})
}

// UpdateSoftwareConfig 更新软件配置
func UpdateSoftwareConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	for key, value := range req {
		var config models.SystemConfig
		if err := db.Where("key = ? AND category = ?", key, "software").First(&config).Error; err != nil {
			config = models.SystemConfig{
				Key:      key,
				Category: "software",
				Value:    fmt.Sprintf("%v", value),
			}
			db.Create(&config)
		} else {
			config.Value = fmt.Sprintf("%v", value)
			db.Save(&config)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "软件配置已更新",
	})
}

// ExportConfig 导出配置
func ExportConfig(c *gin.Context) {
	db := database.GetDB()
	var configs []models.SystemConfig
	if err := db.Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取配置失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    configs,
	})
}

// GetAnnouncements 获取公告列表
func GetAnnouncements(c *gin.Context) {
	db := database.GetDB()

	var announcements []models.Announcement
	now := time.Now()
	if err := db.Where("is_active = ? AND (start_time IS NULL OR start_time <= ?) AND (end_time IS NULL OR end_time >= ?)", true, now, now).Order("is_pinned DESC, created_at DESC").Find(&announcements).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取公告列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    announcements,
	})
}

// CreateAnnouncement 创建公告（管理员）
func CreateAnnouncement(c *gin.Context) {
	var req struct {
		Title       string     `json:"title" binding:"required"`
		Content     string     `json:"content" binding:"required"`
		Type        string     `json:"type"`
		IsActive    bool       `json:"is_active"`
		IsPinned    bool       `json:"is_pinned"`
		StartTime   *time.Time `json:"start_time"`
		EndTime     *time.Time `json:"end_time"`
		TargetUsers string     `json:"target_users"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	user, _ := middleware.GetCurrentUser(c)

	db := database.GetDB()
	announcement := models.Announcement{
		Title:       req.Title,
		Content:     req.Content,
		Type:        req.Type,
		IsActive:    req.IsActive,
		IsPinned:    req.IsPinned,
		TargetUsers: req.TargetUsers,
		CreatedBy:   user.ID,
	}

	if req.StartTime != nil {
		announcement.StartTime = req.StartTime
	} else {
		announcement.StartTime = nil
	}
	if req.EndTime != nil {
		announcement.EndTime = req.EndTime
	} else {
		announcement.EndTime = nil
	}

	if err := db.Create(&announcement).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建公告失败",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    announcement,
	})
}

// GetAdminSettings 获取管理员设置
func GetAdminSettings(c *gin.Context) {
	db := database.GetDB()

	// 默认设置
	defaultSettings := map[string]map[string]interface{}{
		"general": {
			"site_name":        "CBoard Modern",
			"site_description": "现代化的代理服务管理平台",
			"site_logo":        "",
			"default_theme":    "default",
		},
		"registration": {
			"registration_enabled":                 "true",
			"email_verification_required":          "true",
			"min_password_length":                  "8",
			"invite_code_required":                 "false",
			"default_subscription_device_limit":    3,
			"default_subscription_duration_months": 1,
		},
		"notification": {
			"system_notifications":              "true",
			"email_notifications":               "true",
			"subscription_expiry_notifications": "true",
			"new_user_notifications":            "true",
			"new_order_notifications":           "true",
		},
		"security": {
			"login_fail_limit":           "5",
			"login_lock_time":            "30",
			"session_timeout":            "120",
			"device_fingerprint_enabled": "true",
			"ip_whitelist_enabled":       "false",
			"ip_whitelist":               "",
		},
		"theme": {
			"default_theme":    "light",
			"allow_user_theme": "true",
			"available_themes": "[\"light\",\"dark\",\"blue\",\"green\",\"purple\",\"orange\",\"red\",\"cyan\",\"luck\",\"aurora\",\"auto\"]",
		},
		"admin_notification": {
			"admin_notification_enabled":        "false",
			"admin_email_notification":          "false",
			"admin_telegram_notification":       "false",
			"admin_bark_notification":           "false",
			"admin_telegram_bot_token":          "",
			"admin_telegram_chat_id":            "",
			"admin_bark_server_url":             "https://api.day.app",
			"admin_bark_device_key":             "",
			"admin_notification_email":          "",
			"admin_notify_order_paid":           "false",
			"admin_notify_user_registered":      "false",
			"admin_notify_password_reset":       "false",
			"admin_notify_subscription_sent":    "false",
			"admin_notify_subscription_reset":   "false",
			"admin_notify_subscription_expired": "false",
			"admin_notify_user_created":         "false",
			"admin_notify_subscription_created": "false",
		},
	}

	// 从 SystemConfig 表中读取各种设置
	settings := make(map[string]interface{})

	// 遍历所有类别
	for category, defaults := range defaultSettings {
		categorySettings := make(map[string]interface{})

		// 先设置默认值
		for key, value := range defaults {
			categorySettings[key] = value
		}

		// 从数据库读取配置并覆盖默认值
		var configs []models.SystemConfig
		db.Where("category = ?", category).Find(&configs)
		for _, config := range configs {
			// 尝试解析布尔值和数字
			value := config.Value
			if value == "true" || value == "false" {
				categorySettings[config.Key] = value == "true"
			} else if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
				// 尝试解析数组
				var arr []string
				if err := json.Unmarshal([]byte(value), &arr); err == nil {
					categorySettings[config.Key] = arr
				} else {
					categorySettings[config.Key] = value
				}
			} else {
				// 尝试解析数字
				if num, err := strconv.Atoi(value); err == nil {
					categorySettings[config.Key] = num
				} else {
					categorySettings[config.Key] = value
				}
			}
		}

		settings[category] = categorySettings
	}

	// 单独读取 domain_name（属于 system 类别，但需要在 general 中显示）
	var domainConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "domain_name", "system").First(&domainConfig).Error; err == nil {
		if generalSettings, ok := settings["general"].(map[string]interface{}); ok {
			generalSettings["domain_name"] = domainConfig.Value
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    settings,
	})
}

// UpdateGeneralSettings 更新基本设置
func UpdateGeneralSettings(c *gin.Context) {
	var settings map[string]interface{}
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	for key, value := range settings {
		// domain_name 应该保存在 system 类别中
		category := "general"
		if key == "domain_name" {
			category = "system"
		}

		var config models.SystemConfig
		if err := db.Where("key = ? AND category = ?", key, category).First(&config).Error; err != nil {
			// 如果不存在，创建新配置
			config = models.SystemConfig{
				Key:      key,
				Category: category,
				Value:    fmt.Sprintf("%v", value),
			}
			db.Create(&config)
		} else {
			config.Value = fmt.Sprintf("%v", value)
			db.Save(&config)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "基本设置已保存",
	})
}

// UpdateRegistrationSettings 更新注册设置
func UpdateRegistrationSettings(c *gin.Context) {
	var settings map[string]interface{}
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	for key, value := range settings {
		var config models.SystemConfig
		if err := db.Where("key = ? AND category = ?", key, "registration").First(&config).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// 如果不存在，创建新配置
				config = models.SystemConfig{
					Key:      key,
					Category: "registration",
					Value:    fmt.Sprintf("%v", value),
				}
				if err := db.Create(&config).Error; err != nil {
					utils.LogError("UpdateRegistrationSettings: create config failed", err, map[string]interface{}{
						"key": key,
					})
					c.JSON(http.StatusInternalServerError, gin.H{
						"success": false,
						"message": fmt.Sprintf("保存配置 %s 失败", key),
					})
					return
				}
			} else {
				utils.LogError("UpdateRegistrationSettings: query config failed", err, map[string]interface{}{
					"key": key,
				})
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": fmt.Sprintf("查询配置 %s 失败", key),
				})
				return
			}
		} else {
			config.Value = fmt.Sprintf("%v", value)
			if err := db.Save(&config).Error; err != nil {
				utils.LogError("UpdateRegistrationSettings: update config failed", err, map[string]interface{}{
					"key": key,
				})
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": fmt.Sprintf("更新配置 %s 失败", key),
				})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "注册设置已保存",
	})
}

// UpdateNotificationSettings 更新通知设置
func UpdateNotificationSettings(c *gin.Context) {
	var settings map[string]interface{}
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	for key, value := range settings {
		var config models.SystemConfig
		if err := db.Where("key = ? AND category = ?", key, "notification").First(&config).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// 如果不存在，创建新配置
				config = models.SystemConfig{
					Key:      key,
					Category: "notification",
					Value:    fmt.Sprintf("%v", value),
				}
				if err := db.Create(&config).Error; err != nil {
					utils.LogError("UpdateNotificationSettings: create config failed", err, map[string]interface{}{
						"key": key,
					})
					c.JSON(http.StatusInternalServerError, gin.H{
						"success": false,
						"message": fmt.Sprintf("保存配置 %s 失败", key),
					})
					return
				}
			} else {
				utils.LogError("UpdateNotificationSettings: query config failed", err, map[string]interface{}{
					"key": key,
				})
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": fmt.Sprintf("查询配置 %s 失败", key),
				})
				return
			}
		} else {
			config.Value = fmt.Sprintf("%v", value)
			if err := db.Save(&config).Error; err != nil {
				utils.LogError("UpdateNotificationSettings: update config failed", err, map[string]interface{}{
					"key": key,
				})
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": fmt.Sprintf("更新配置 %s 失败", key),
				})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "通知设置已保存",
	})
}

// UpdateSecuritySettings 更新安全设置
func UpdateSecuritySettings(c *gin.Context) {
	var settings map[string]interface{}
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	for key, value := range settings {
		var config models.SystemConfig
		if err := db.Where("key = ? AND category = ?", key, "security").First(&config).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// 如果不存在，创建新配置
				config = models.SystemConfig{
					Key:      key,
					Category: "security",
					Value:    fmt.Sprintf("%v", value),
				}
				if err := db.Create(&config).Error; err != nil {
					utils.LogError("UpdateSecuritySettings: create config failed", err, map[string]interface{}{
						"key": key,
					})
					c.JSON(http.StatusInternalServerError, gin.H{
						"success": false,
						"message": fmt.Sprintf("保存配置 %s 失败", key),
					})
					return
				}
			} else {
				utils.LogError("UpdateSecuritySettings: query config failed", err, map[string]interface{}{
					"key": key,
				})
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": fmt.Sprintf("查询配置 %s 失败", key),
				})
				return
			}
		} else {
			config.Value = fmt.Sprintf("%v", value)
			if err := db.Save(&config).Error; err != nil {
				utils.LogError("UpdateSecuritySettings: update config failed", err, map[string]interface{}{
					"key": key,
				})
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": fmt.Sprintf("更新配置 %s 失败", key),
				})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "安全设置已保存",
	})
}

// UpdateThemeSettings 更新主题设置
func UpdateThemeSettings(c *gin.Context) {
	var settings map[string]interface{}
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	for key, value := range settings {
		var config models.SystemConfig
		// 处理数组类型（如 available_themes）
		var valueStr string
		if arr, ok := value.([]interface{}); ok {
			// 将数组转换为JSON字符串
			jsonBytes, err := json.Marshal(arr)
			if err == nil {
				valueStr = string(jsonBytes)
			} else {
				valueStr = fmt.Sprintf("%v", value)
			}
		} else {
			valueStr = fmt.Sprintf("%v", value)
		}

		if err := db.Where("key = ? AND category = ?", key, "theme").First(&config).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				config = models.SystemConfig{
					Key:      key,
					Category: "theme",
					Value:    valueStr,
				}
				if err := db.Create(&config).Error; err != nil {
					utils.LogError("UpdateThemeSettings: create config failed", err, map[string]interface{}{
						"key": key,
					})
					c.JSON(http.StatusInternalServerError, gin.H{
						"success": false,
						"message": fmt.Sprintf("保存配置 %s 失败", key),
					})
					return
				}
			} else {
				utils.LogError("UpdateThemeSettings: query config failed", err, map[string]interface{}{
					"key": key,
				})
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": fmt.Sprintf("查询配置 %s 失败", key),
				})
				return
			}
		} else {
			config.Value = valueStr
			if err := db.Save(&config).Error; err != nil {
				utils.LogError("UpdateThemeSettings: update config failed", err, map[string]interface{}{
					"key": key,
				})
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": fmt.Sprintf("更新配置 %s 失败", key),
				})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "主题设置已保存",
	})
}

// UpdateInviteSettings 更新邀请设置
func UpdateInviteSettings(c *gin.Context) {
	var settings map[string]interface{}
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	for key, value := range settings {
		var config models.SystemConfig
		if err := db.Where("key = ? AND category = ?", key, "invite").First(&config).Error; err != nil {
			config = models.SystemConfig{
				Key:      key,
				Category: "invite",
				Value:    fmt.Sprintf("%v", value),
			}
			db.Create(&config)
		} else {
			config.Value = fmt.Sprintf("%v", value)
			db.Save(&config)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "邀请设置已保存",
	})
}

// UpdateAdminNotificationSystemSettings 更新管理员通知设置（系统设置）
func UpdateAdminNotificationSystemSettings(c *gin.Context) {
	var settings map[string]interface{}
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	for key, value := range settings {
		var config models.SystemConfig
		if err := db.Where("key = ? AND category = ?", key, "admin_notification").First(&config).Error; err != nil {
			config = models.SystemConfig{
				Key:      key,
				Category: "admin_notification",
				Value:    fmt.Sprintf("%v", value),
			}
			db.Create(&config)
		} else {
			config.Value = fmt.Sprintf("%v", value)
			db.Save(&config)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "管理员通知设置已保存",
	})
}

// TestAdminEmailNotification 测试管理员邮件通知
func TestAdminEmailNotification(c *gin.Context) {
	db := database.GetDB()

	// 获取管理员通知邮箱
	var emailConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "admin_notification_email", "admin_notification").First(&emailConfig).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "管理员通知邮箱未配置",
		})
		return
	}

	adminEmail := emailConfig.Value
	if adminEmail == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "管理员通知邮箱未配置",
		})
		return
	}

	// 发送测试邮件（使用邮件模板）
	emailService := email.NewEmailService()
	templateBuilder := email.NewEmailTemplateBuilder()
	subject := "🧪 测试消息"
	content := templateBuilder.GetBroadcastNotificationTemplate(
		subject,
		"这是一条测试消息，如果您收到此消息，说明邮件通知配置正确。",
	)

	// 将邮件加入队列
	if err := emailService.QueueEmail(adminEmail, subject, content, "admin_notification"); err != nil {
		utils.LogError("TestAdminEmailNotification: queue email failed", err, map[string]interface{}{
			"admin_email": adminEmail,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "测试消息发送失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "测试消息已加入邮件队列，请检查您的邮箱",
	})
}

// TestAdminTelegramNotification 测试管理员 Telegram 通知
func TestAdminTelegramNotification(c *gin.Context) {
	db := database.GetDB()

	// 获取 Telegram 配置
	var botTokenConfig, chatIDConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "admin_telegram_bot_token", "admin_notification").First(&botTokenConfig).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Telegram Bot Token 未配置",
		})
		return
	}
	if err := db.Where("key = ? AND category = ?", "admin_telegram_chat_id", "admin_notification").First(&chatIDConfig).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Telegram Chat ID 未配置",
		})
		return
	}

	botToken := botTokenConfig.Value
	chatID := chatIDConfig.Value

	if botToken == "" || chatID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Telegram 配置不完整",
		})
		return
	}

	// 发送测试消息（使用模板构建器）
	templateBuilder := notification.NewMessageTemplateBuilder()
	testData := map[string]interface{}{
		"title":   "测试消息",
		"message": "这是一条测试消息，如果您收到此消息，说明 Telegram 通知配置正确。",
	}
	testMessage := templateBuilder.BuildTelegramMessage("default", testData)
	success, err := sendTelegramMessage(botToken, chatID, testMessage)
	if err != nil {
		utils.LogError("TestAdminTelegramNotification: send message failed", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "测试消息发送失败",
		})
		return
	}

	if !success {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "测试消息发送失败，请检查配置",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "测试消息发送成功，请检查您的 Telegram",
	})
}

// TestAdminBarkNotification 测试管理员 Bark 通知
func TestAdminBarkNotification(c *gin.Context) {
	db := database.GetDB()

	// 获取 Bark 配置
	var serverURLConfig, deviceKeyConfig models.SystemConfig
	serverURL := "https://api.day.app" // 默认值
	if err := db.Where("key = ? AND category = ?", "admin_bark_server_url", "admin_notification").First(&serverURLConfig).Error; err == nil {
		if serverURLConfig.Value != "" {
			serverURL = serverURLConfig.Value
		}
	}

	if err := db.Where("key = ? AND category = ?", "admin_bark_device_key", "admin_notification").First(&deviceKeyConfig).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Bark Device Key 未配置",
		})
		return
	}

	deviceKey := deviceKeyConfig.Value
	if deviceKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Bark Device Key 未配置",
		})
		return
	}

	// 发送测试消息（使用模板构建器）
	templateBuilder := notification.NewMessageTemplateBuilder()
	testData := map[string]interface{}{
		"title":   "测试消息",
		"message": "这是一条测试消息，如果您收到此消息，说明 Bark 通知配置正确。",
	}
	barkTitle, barkBody := templateBuilder.BuildBarkMessage("default", testData)
	success, err := sendBarkMessage(serverURL, deviceKey, barkTitle, barkBody)
	if err != nil {
		utils.LogError("TestAdminBarkNotification: send message failed", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "测试消息发送失败",
		})
		return
	}

	if !success {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "测试消息发送失败，请检查配置",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "测试消息发送成功，请检查您的设备",
	})
}

// sendTelegramMessage 发送 Telegram 消息
func sendTelegramMessage(botToken, chatID, message string) (bool, error) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "HTML",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return result["ok"] == true, nil
}

// sendBarkMessage 发送 Bark 消息
func sendBarkMessage(serverURL, deviceKey, title, body string) (bool, error) {
	// 移除末尾的斜杠
	serverURL = strings.TrimSuffix(serverURL, "/")
	apiURL := fmt.Sprintf("%s/push", serverURL)

	payload := map[string]interface{}{
		"device_key": deviceKey,
		"title":      title,
		"body":       body,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return result["code"] == float64(200), nil
}

// UploadFile 文件上传（管理员）
func UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "文件上传失败",
		})
		return
	}

	// 获取配置
	cfg := config.AppConfig
	if cfg == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "系统配置错误",
		})
		return
	}

	// 1. 验证文件大小
	maxSize := cfg.MaxFileSize
	if maxSize == 0 {
		maxSize = 10 * 1024 * 1024 // 默认 10MB
	}
	if file.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": fmt.Sprintf("文件大小超过限制（最大 %d MB）", maxSize/(1024*1024)),
		})
		return
	}

	// 2. 验证文件类型（白名单）
	allowedExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".pdf", ".txt", ".doc", ".docx", ".xls", ".xlsx", ".zip", ".rar"}
	originalFilename := file.Filename
	ext := strings.ToLower(filepath.Ext(originalFilename))

	allowed := false
	for _, allowedExt := range allowedExtensions {
		if ext == allowedExt {
			allowed = true
			break
		}
	}

	if !allowed {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "不支持的文件类型，仅支持: " + strings.Join(allowedExtensions, ", "),
		})
		return
	}

	// 3. 防止路径遍历攻击
	// 清理文件名，移除危险字符
	safeFilename := utils.SanitizeInput(strings.TrimSpace(originalFilename))
	if safeFilename == "" {
		safeFilename = "uploaded_file"
	}
	// 移除路径分隔符
	safeFilename = strings.ReplaceAll(safeFilename, "/", "_")
	safeFilename = strings.ReplaceAll(safeFilename, "\\", "_")
	safeFilename = strings.ReplaceAll(safeFilename, "..", "_")

	// 如果清理后没有扩展名，添加原始扩展名
	if filepath.Ext(safeFilename) == "" && ext != "" {
		safeFilename += ext
	}

	// 4. 生成唯一文件名（防止文件名冲突和覆盖）
	timestamp := time.Now().Unix()
	uniqueFilename := fmt.Sprintf("%d_%s", timestamp, safeFilename)

	// 5. 确保上传目录存在
	uploadDir := cfg.UploadDir
	if uploadDir == "" {
		uploadDir = "uploads"
	}

	// 使用 filepath.Join 安全地构建路径
	safePath := filepath.Join(uploadDir, uniqueFilename)

	// 6. 验证最终路径在允许的目录内（防止路径遍历）
	absUploadDir, err := filepath.Abs(uploadDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "系统错误",
		})
		utils.LogError("UploadFile: get absolute path failed", err, nil)
		return
	}

	absSafePath, err := filepath.Abs(safePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "系统错误",
		})
		utils.LogError("UploadFile: get absolute path failed", err, nil)
		return
	}

	// 确保文件路径在允许的目录内
	if !strings.HasPrefix(absSafePath, absUploadDir) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的文件路径",
		})
		utils.LogError("UploadFile: path traversal detected", nil, map[string]interface{}{
			"original_filename": originalFilename,
			"safe_path":         safePath,
		})
		return
	}

	// 7. 创建上传目录
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "系统错误",
		})
		utils.LogError("UploadFile: create upload directory failed", err, nil)
		return
	}

	// 8. 保存文件
	if err := c.SaveUploadedFile(file, safePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "保存文件失败",
		})
		utils.LogError("UploadFile: save file failed", err, map[string]interface{}{
			"safe_path": safePath,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "文件上传成功",
		"data": gin.H{
			"url":      "/" + safePath,
			"filename": uniqueFilename,
		},
	})
}

// GetPublicSettings 获取公开设置
func GetPublicSettings(c *gin.Context) {
	db := database.GetDB()
	var configs []models.SystemConfig
	db.Where("is_public = ?", true).Find(&configs)

	settings := make(map[string]interface{})
	for _, config := range configs {
		settings[config.Key] = config.Value
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    settings,
	})
}

// UpdateAnnouncement 更新公告（管理员）
func UpdateAnnouncement(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Title       string     `json:"title"`
		Content     string     `json:"content"`
		Type        string     `json:"type"`
		IsActive    *bool      `json:"is_active"`
		IsPinned    *bool      `json:"is_pinned"`
		StartTime   *time.Time `json:"start_time"`
		EndTime     *time.Time `json:"end_time"`
		TargetUsers string     `json:"target_users"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	var announcement models.Announcement
	if err := db.First(&announcement, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "公告不存在",
		})
		return
	}

	if req.Title != "" {
		announcement.Title = req.Title
	}
	if req.Content != "" {
		announcement.Content = req.Content
	}
	if req.Type != "" {
		announcement.Type = req.Type
	}
	if req.IsActive != nil {
		announcement.IsActive = *req.IsActive
	}
	if req.IsPinned != nil {
		announcement.IsPinned = *req.IsPinned
	}
	if req.TargetUsers != "" {
		announcement.TargetUsers = req.TargetUsers
	}
	if req.StartTime != nil {
		announcement.StartTime = req.StartTime
	}
	if req.EndTime != nil {
		announcement.EndTime = req.EndTime
	}

	if err := db.Save(&announcement).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新公告失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "更新成功",
		"data":    announcement,
	})
}

// DeleteAnnouncement 删除公告（管理员）
func DeleteAnnouncement(c *gin.Context) {
	id := c.Param("id")

	db := database.GetDB()
	if err := db.Delete(&models.Announcement{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除公告失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "删除成功",
	})
}

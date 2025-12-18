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
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// --- Helper Functions ---

// jsonResponse 统一响应格式
func jsonResponse(c *gin.Context, code int, success bool, msg string, data interface{}) {
	c.JSON(code, gin.H{"success": success, "message": msg, "data": data})
}

// updateSettingsCommon 通用配置更新逻辑
func updateSettingsCommon(c *gin.Context, category string) {
	var settings map[string]interface{}
	if err := c.ShouldBindJSON(&settings); err != nil {
		jsonResponse(c, http.StatusBadRequest, false, "请求参数错误", nil)
		return
	}

	err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		for key, val := range settings {
			targetCat := category
			if key == "domain_name" && category == "general" {
				targetCat = "system" // 特殊处理
			}

			// 处理数组/切片转JSON字符串
			valStr := fmt.Sprintf("%v", val)
			if _, ok := val.([]interface{}); ok {
				if jsonBytes, err := json.Marshal(val); err == nil {
					valStr = string(jsonBytes)
				}
			}

			// 使用 Upsert (OnConflict) 减少查询次数，或先查后改
			var conf models.SystemConfig
			if err := tx.Where("key = ? AND category = ?", key, targetCat).First(&conf).Error; err != nil {
				conf = models.SystemConfig{Key: key, Category: targetCat, Value: valStr}
				if err := tx.Create(&conf).Error; err != nil {
					return err
				}
			} else {
				conf.Value = valStr
				if err := tx.Save(&conf).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		utils.LogError(fmt.Sprintf("UpdateSettings (%s)", category), err, nil)
		jsonResponse(c, http.StatusInternalServerError, false, "保存设置失败", nil)
		return
	}
	jsonResponse(c, http.StatusOK, true, "设置已保存", nil)
}

// --- Handlers ---

// GetSystemConfigs 获取系统配置
func GetSystemConfigs(c *gin.Context) {
	db := database.GetDB()
	var configs []models.SystemConfig
	query := db.Order("sort_order ASC")

	if cat := c.Query("category"); cat != "" {
		query = query.Where("category = ?", cat)
	}
	if c.Query("is_public") == "true" {
		query = query.Where("is_public = ?", true)
	}

	if err := query.Find(&configs).Error; err != nil {
		jsonResponse(c, http.StatusInternalServerError, false, "获取配置失败", nil)
		return
	}
	jsonResponse(c, http.StatusOK, true, "", configs)
}

// GetSystemConfig 获取单个配置
func GetSystemConfig(c *gin.Context) {
	var config models.SystemConfig
	if err := database.GetDB().Where("key = ?", c.Param("key")).First(&config).Error; err != nil {
		jsonResponse(c, http.StatusNotFound, false, "配置不存在", nil)
		return
	}
	jsonResponse(c, http.StatusOK, true, "", config)
}

// CreateSystemConfig 创建系统配置
func CreateSystemConfig(c *gin.Context) {
	var req models.SystemConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonResponse(c, http.StatusBadRequest, false, "请求参数错误", nil)
		return
	}

	db := database.GetDB()
	var exist models.SystemConfig
	q := db.Where("key = ?", req.Key)
	if req.Category != "" {
		q = q.Where("category = ?", req.Category)
	}

	if q.First(&exist).Error == nil {
		jsonResponse(c, http.StatusBadRequest, false, "配置已存在", nil)
		return
	}

	if err := db.Create(&req).Error; err != nil {
		jsonResponse(c, http.StatusInternalServerError, false, "创建配置失败", nil)
		return
	}
	jsonResponse(c, http.StatusCreated, true, "", req)
}

// UpdateSystemConfig 更新配置 (单条或批量)
func UpdateSystemConfig(c *gin.Context) {
	key := c.Param("key")
	db := database.GetDB()

	// 批量更新
	if key == "batch" {
		var req map[string]interface{}
		if err := c.ShouldBindJSON(&req); err != nil {
			jsonResponse(c, http.StatusBadRequest, false, "请求参数错误", nil)
			return
		}
		// 复用通用逻辑，但 category 设为 system (或者根据业务需求调整)
		// 这里保留原逻辑：先查后改
		for k, v := range req {
			val := fmt.Sprintf("%v", v)
			db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "key"}}, // 假设 key 是唯一索引
				DoUpdates: clause.Assignments(map[string]interface{}{"value": val}),
			}).Create(&models.SystemConfig{Key: k, Value: val, Category: "system"})
		}
		jsonResponse(c, http.StatusOK, true, "批量更新成功", nil)
		return
	}

	// 单个更新
	var req struct {
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonResponse(c, http.StatusBadRequest, false, "请求参数错误", nil)
		return
	}

	var config models.SystemConfig
	err := db.Where("key = ?", key).First(&config).Error
	if err != nil {
		config = models.SystemConfig{Key: key, Value: req.Value, Category: "system"}
		if err := db.Create(&config).Error; err != nil {
			jsonResponse(c, http.StatusInternalServerError, false, "创建配置失败", nil)
			return
		}
	} else {
		config.Value = req.Value
		if err := db.Save(&config).Error; err != nil {
			jsonResponse(c, http.StatusInternalServerError, false, "更新配置失败", nil)
			return
		}
	}
	jsonResponse(c, http.StatusOK, true, "更新成功", config)
}

// ExportConfig 导出配置
func ExportConfig(c *gin.Context) {
	var configs []models.SystemConfig
	if err := database.GetDB().Find(&configs).Error; err != nil {
		jsonResponse(c, http.StatusInternalServerError, false, "获取配置失败", nil)
		return
	}
	jsonResponse(c, http.StatusOK, true, "", configs)
}

// GetAdminSettings 获取聚合的管理员设置
func GetAdminSettings(c *gin.Context) {
	// 定义默认值
	settings := map[string]map[string]interface{}{
		"general": {
			"site_name": "CBoard Modern", "site_description": "现代化的代理服务管理平台", "site_logo": "", "default_theme": "default",
		},
		"registration": {
			"registration_enabled": "true", "email_verification_required": "true", "min_password_length": 8,
			"invite_code_required": "false", "default_subscription_device_limit": 3, "default_subscription_duration_months": 1,
		},
		"notification": {
			"system_notifications": "true", "email_notifications": "true", "subscription_expiry_notifications": "true",
			"new_user_notifications": "true", "new_order_notifications": "true",
		},
		"security": {
			"login_fail_limit": 5, "login_lock_time": 30, "session_timeout": 120,
			"device_fingerprint_enabled": "true", "ip_whitelist_enabled": "false", "ip_whitelist": "",
		},
		"theme": {
			"default_theme": "light", "allow_user_theme": "true",
			"available_themes": []string{"light", "dark", "blue", "green", "purple", "orange", "red", "cyan", "luck", "aurora", "auto"},
		},
		"admin_notification": {
			"admin_notification_enabled": "false", "admin_email_notification": "false", "admin_telegram_notification": "false",
			"admin_bark_notification": "false", "admin_telegram_bot_token": "", "admin_telegram_chat_id": "",
			"admin_bark_server_url": "https://api.day.app", "admin_bark_device_key": "", "admin_notification_email": "",
			"admin_notify_order_paid": "false", "admin_notify_user_registered": "false", "admin_notify_password_reset": "false",
			"admin_notify_subscription_sent": "false", "admin_notify_subscription_reset": "false", "admin_notify_subscription_expired": "false",
			"admin_notify_user_created": "false", "admin_notify_subscription_created": "false",
		},
		"announcement": {
			"announcement_enabled": "false",
			"announcement_content": "",
		},
	}

	db := database.GetDB()
	var configs []models.SystemConfig
	// 一次性查出所有相关配置，减少数据库往返
	cats := make([]string, 0, len(settings)+1)
	for k := range settings {
		cats = append(cats, k)
	}
	cats = append(cats, "system") // 用于 domain_name
	db.Where("category IN ?", cats).Find(&configs)

	configMap := make(map[string]map[string]string)
	for _, conf := range configs {
		if _, ok := configMap[conf.Category]; !ok {
			configMap[conf.Category] = make(map[string]string)
		}
		configMap[conf.Category][conf.Key] = conf.Value
	}

	// 填充数据
	for cat, catDefaults := range settings {
		for key := range catDefaults {
			if val, ok := configMap[cat][key]; ok {
				// 尝试转换类型
				if val == "true" || val == "false" {
					settings[cat][key] = (val == "true")
				} else if strings.HasPrefix(val, "[") {
					var arr []string
					if json.Unmarshal([]byte(val), &arr) == nil {
						settings[cat][key] = arr
					} else {
						settings[cat][key] = val
					}
				} else if num, err := strconv.Atoi(val); err == nil {
					settings[cat][key] = num
				} else {
					settings[cat][key] = val
				}
			}
		}
	}
	// 特殊处理 domain_name
	if val, ok := configMap["system"]["domain_name"]; ok {
		settings["general"]["domain_name"] = val
	}

	jsonResponse(c, http.StatusOK, true, "", settings)
}

// 统一的 Update Handlers
func UpdateGeneralSettings(c *gin.Context)      { updateSettingsCommon(c, "general") }
func UpdateRegistrationSettings(c *gin.Context) { updateSettingsCommon(c, "registration") }
func UpdateNotificationSettings(c *gin.Context) { updateSettingsCommon(c, "notification") }
func UpdateSecuritySettings(c *gin.Context)     { updateSettingsCommon(c, "security") }
func UpdateThemeSettings(c *gin.Context)        { updateSettingsCommon(c, "theme") }
func UpdateInviteSettings(c *gin.Context)       { updateSettingsCommon(c, "invite") }
func UpdateSoftwareConfig(c *gin.Context)       { updateSettingsCommon(c, "software") }
func UpdateAnnouncementSettings(c *gin.Context) { updateSettingsCommon(c, "announcement") }
func UpdateAdminNotificationSystemSettings(c *gin.Context) {
	updateSettingsCommon(c, "admin_notification")
}

// --- Notification Tests ---

func TestAdminEmailNotification(c *gin.Context) {
	var conf models.SystemConfig
	if err := database.GetDB().Where("key = ? AND category = ?", "admin_notification_email", "admin_notification").First(&conf).Error; err != nil || conf.Value == "" {
		jsonResponse(c, http.StatusBadRequest, false, "管理员通知邮箱未配置", nil)
		return
	}

	subject := "🧪 测试消息"
	content := email.NewEmailTemplateBuilder().GetBroadcastNotificationTemplate(subject, "这是一条测试消息，说明邮件通知配置正确。")
	if err := email.NewEmailService().QueueEmail(conf.Value, subject, content, "admin_notification"); err != nil {
		utils.LogError("TestAdminEmail", err, nil)
		jsonResponse(c, http.StatusInternalServerError, false, "测试消息发送失败", nil)
		return
	}
	jsonResponse(c, http.StatusOK, true, "测试消息已加入邮件队列", nil)
}

func TestAdminTelegramNotification(c *gin.Context) {
	db := database.GetDB()
	var token, chatID models.SystemConfig
	if db.Where("key = ?", "admin_telegram_bot_token").First(&token).Error != nil ||
		db.Where("key = ?", "admin_telegram_chat_id").First(&chatID).Error != nil ||
		token.Value == "" || chatID.Value == "" {
		jsonResponse(c, http.StatusBadRequest, false, "Telegram 配置不完整", nil)
		return
	}

	msg := notification.NewMessageTemplateBuilder().BuildTelegramMessage("default", map[string]interface{}{"title": "测试消息", "message": "配置正确。"})
	if ok, _ := sendTelegramMessage(token.Value, chatID.Value, msg); !ok {
		jsonResponse(c, http.StatusInternalServerError, false, "测试消息发送失败", nil)
		return
	}
	jsonResponse(c, http.StatusOK, true, "测试消息发送成功", nil)
}

func TestAdminBarkNotification(c *gin.Context) {
	db := database.GetDB()
	serverURL := "https://api.day.app"
	var urlConf, keyConf models.SystemConfig
	if db.Where("key = ?", "admin_bark_server_url").First(&urlConf).Error == nil && urlConf.Value != "" {
		serverURL = urlConf.Value
	}
	if db.Where("key = ?", "admin_bark_device_key").First(&keyConf).Error != nil || keyConf.Value == "" {
		jsonResponse(c, http.StatusBadRequest, false, "Bark Device Key 未配置", nil)
		return
	}

	title, body := notification.NewMessageTemplateBuilder().BuildBarkMessage("default", map[string]interface{}{"title": "测试消息", "message": "配置正确。"})
	if ok, _ := sendBarkMessage(serverURL, keyConf.Value, title, body); !ok {
		jsonResponse(c, http.StatusInternalServerError, false, "测试消息发送失败", nil)
		return
	}
	jsonResponse(c, http.StatusOK, true, "测试消息发送成功", nil)
}

func sendTelegramMessage(token, chatID, msg string) (bool, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer([]byte(fmt.Sprintf(`{"chat_id":"%s","text":%q,"parse_mode":"HTML"}`, chatID, msg))))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	return res["ok"] == true, nil
}

func sendBarkMessage(server, key, title, body string) (bool, error) {
	url := fmt.Sprintf("%s/push", strings.TrimSuffix(server, "/"))
	data, _ := json.Marshal(map[string]string{"device_key": key, "title": title, "body": body})
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	return res["code"] == float64(200), nil
}

// UploadFile 文件上传
func UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		jsonResponse(c, http.StatusBadRequest, false, "文件上传失败", nil)
		return
	}

	cfg := config.AppConfig
	maxSize := int64(10 * 1024 * 1024)
	if cfg != nil && cfg.MaxFileSize > 0 {
		maxSize = cfg.MaxFileSize
	}

	if file.Size > maxSize {
		jsonResponse(c, http.StatusBadRequest, false, fmt.Sprintf("文件超限 (Max %d MB)", maxSize>>20), nil)
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".pdf": true, ".txt": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".zip": true, ".rar": true}
	if !allowed[ext] {
		jsonResponse(c, http.StatusBadRequest, false, "不支持的文件类型", nil)
		return
	}

	// 路径安全处理
	safeName := utils.SanitizeInput(file.Filename)
	if safeName == "" {
		safeName = "file" + ext
	}
	safeName = fmt.Sprintf("%d_%s", time.Now().Unix(), strings.NewReplacer("/", "_", "\\", "_", "..", "_").Replace(safeName))

	uploadDir := "uploads"
	if cfg != nil && cfg.UploadDir != "" {
		uploadDir = cfg.UploadDir
	}

	absDir, _ := filepath.Abs(uploadDir)
	absPath, _ := filepath.Abs(filepath.Join(uploadDir, safeName))
	if !strings.HasPrefix(absPath, absDir) {
		jsonResponse(c, http.StatusBadRequest, false, "非法路径", nil)
		return
	}

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		jsonResponse(c, http.StatusInternalServerError, false, "系统错误", nil)
		return
	}

	if err := c.SaveUploadedFile(file, filepath.Join(uploadDir, safeName)); err != nil {
		utils.LogError("UploadFile", err, nil)
		jsonResponse(c, http.StatusInternalServerError, false, "保存失败", nil)
		return
	}
	jsonResponse(c, http.StatusOK, true, "上传成功", gin.H{"url": "/" + filepath.Join(uploadDir, safeName), "filename": safeName})
}

// GetPublicSettings 获取公开设置
func GetPublicSettings(c *gin.Context) {
	var configs []models.SystemConfig
	db := database.GetDB()
	// 获取所有公开设置
	db.Where("is_public = ?", true).Find(&configs)
	settings := make(map[string]interface{})
	for _, conf := range configs {
		settings[conf.Key] = conf.Value
	}
	// 获取公告设置（如果启用）
	var announcementEnabled models.SystemConfig
	var announcementContent models.SystemConfig
	if db.Where("key = ? AND category = ?", "announcement_enabled", "announcement").First(&announcementEnabled).Error == nil {
		if announcementEnabled.Value == "true" {
			settings["announcement_enabled"] = true
			if db.Where("key = ? AND category = ?", "announcement_content", "announcement").First(&announcementContent).Error == nil {
				settings["announcement_content"] = announcementContent.Value
			}
		} else {
			settings["announcement_enabled"] = false
		}
	} else {
		settings["announcement_enabled"] = false
	}
	jsonResponse(c, http.StatusOK, true, "", settings)
}

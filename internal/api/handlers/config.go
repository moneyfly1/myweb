package handlers

import (
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

	// 单个更新 - 支持完整的配置对象（包括 category）
	var req models.SystemConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果绑定失败，尝试只绑定 value（向后兼容）
		var simpleReq struct {
			Value string `json:"value"`
		}
		if err2 := c.ShouldBindJSON(&simpleReq); err2 != nil {
			jsonResponse(c, http.StatusBadRequest, false, "请求参数错误", nil)
			return
		}
		req.Value = simpleReq.Value
	}

	// 确定 category
	category := req.Category
	if category == "" {
		category = "system" // 默认 category
	}

	// 查找配置（根据 key 和 category）
	var config models.SystemConfig
	err := db.Where("key = ? AND category = ?", key, category).First(&config).Error
	if err != nil {
		// 配置不存在，创建新配置
		config = models.SystemConfig{
			Key:         key,
			Value:       req.Value,
			Category:    category,
			Type:        req.Type,
			DisplayName: req.DisplayName,
		}
		if config.Type == "" {
			config.Type = "string"
		}
		if err := db.Create(&config).Error; err != nil {
			jsonResponse(c, http.StatusInternalServerError, false, "创建配置失败", nil)
			return
		}
	} else {
		// 配置存在，更新
		config.Value = req.Value
		if req.Type != "" {
			config.Type = req.Type
		}
		if req.DisplayName != "" {
			config.DisplayName = req.DisplayName
		}
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
		"security": {
			"login_fail_limit": 5, "login_lock_time": 30, "session_timeout": 120,
			"ip_whitelist_enabled": "false", "ip_whitelist": "",
		},
		"theme": {
			"default_theme": "light", "allow_user_theme": "true",
			"available_themes": []string{"light", "dark", "blue", "green", "purple", "orange", "red", "cyan", "luck", "aurora", "auto"},
		},
		"announcement": {
			"announcement_enabled": "false",
			"announcement_content": "",
		},
		"node_health": {
			"node_health_check_interval": "300",
			"node_max_latency":           "3000",
			"node_test_timeout":          "5",
			"test_url":                   "https://ping.pe",
		},
		"custom_node": {},
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
func UpdateSecuritySettings(c *gin.Context)     { updateSettingsCommon(c, "security") }
func UpdateThemeSettings(c *gin.Context)        { updateSettingsCommon(c, "theme") }
func UpdateInviteSettings(c *gin.Context)       { updateSettingsCommon(c, "invite") }
func UpdateSoftwareConfig(c *gin.Context)       { updateSettingsCommon(c, "software") }
func UpdateAnnouncementSettings(c *gin.Context) { updateSettingsCommon(c, "announcement") }
func UpdateNotificationSettings(c *gin.Context) { updateSettingsCommon(c, "notification") }
func UpdateAdminNotificationSystemSettings(c *gin.Context) {
	updateSettingsCommon(c, "admin_notification")
}
func UpdateNodeHealthSettings(c *gin.Context) {
	updateSettingsCommon(c, "node_health")
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
	var announcementEnabled models.SystemConfig
	var announcementContent models.SystemConfig
	err := db.Where("key = ? AND category = ?", "announcement_enabled", "announcement").First(&announcementEnabled).Error
	if err == nil {
		if announcementEnabled.Value == "true" {
			settings["announcement_enabled"] = true
			err2 := db.Where("key = ? AND category = ?", "announcement_content", "announcement").First(&announcementContent).Error
			if err2 == nil {
				settings["announcement_content"] = announcementContent.Value
			}
		} else {
			settings["announcement_enabled"] = false
		}
	} else if err != gorm.ErrRecordNotFound {
		utils.LogError("GetPublicSettings: query announcement_enabled failed", err, nil)
		settings["announcement_enabled"] = false
	} else {
		settings["announcement_enabled"] = false
	}
	jsonResponse(c, http.StatusOK, true, "", settings)
}

// TestAdminEmailNotification 测试管理员邮件通知
func TestAdminEmailNotification(c *gin.Context) {
	db := database.GetDB()
	var configs []models.SystemConfig
	db.Where("category = ?", "admin_notification").Find(&configs)

	configMap := make(map[string]string)
	for _, config := range configs {
		configMap[config.Key] = config.Value
	}

	adminEmail := configMap["admin_notification_email"]
	if adminEmail == "" {
		jsonResponse(c, http.StatusBadRequest, false, "管理员邮箱未配置", nil)
		return
	}

	emailService := email.NewEmailService()
	templateBuilder := email.NewEmailTemplateBuilder()
	subject := "测试邮件通知"
	content := templateBuilder.GetAdminNotificationTemplate("test", "测试通知", "这是一条测试消息，用于验证邮件通知功能是否正常工作。", map[string]interface{}{})

	if err := emailService.QueueEmail(adminEmail, subject, content, "admin_notification"); err != nil {
		utils.LogError("TestAdminEmailNotification", err, nil)
		jsonResponse(c, http.StatusInternalServerError, false, "发送测试邮件失败", nil)
		return
	}

	jsonResponse(c, http.StatusOK, true, "测试邮件已加入队列，请检查您的邮箱", nil)
}

// TestAdminTelegramNotification 测试管理员 Telegram 通知
func TestAdminTelegramNotification(c *gin.Context) {
	db := database.GetDB()
	var configs []models.SystemConfig
	db.Where("category = ?", "admin_notification").Find(&configs)

	configMap := make(map[string]string)
	for _, config := range configs {
		configMap[config.Key] = config.Value
	}

	botToken := configMap["admin_telegram_bot_token"]
	chatID := configMap["admin_telegram_chat_id"]

	if botToken == "" || chatID == "" {
		jsonResponse(c, http.StatusBadRequest, false, "Telegram Bot Token 或 Chat ID 未配置", nil)
		return
	}

	notificationService := notification.NewNotificationService()
	testData := map[string]interface{}{
		"type":    "test",
		"message": "这是一条测试消息，用于验证 Telegram 通知功能是否正常工作。",
	}

	// 发送测试通知
	go func() {
		_ = notificationService.SendAdminNotification("test", testData)
	}()

	jsonResponse(c, http.StatusOK, true, "测试消息已发送，请检查您的 Telegram", nil)
}

// TestAdminBarkNotification 测试管理员 Bark 通知
func TestAdminBarkNotification(c *gin.Context) {
	db := database.GetDB()
	var configs []models.SystemConfig
	db.Where("category = ?", "admin_notification").Find(&configs)

	configMap := make(map[string]string)
	for _, config := range configs {
		configMap[config.Key] = config.Value
	}

	serverURL := configMap["admin_bark_server_url"]
	deviceKey := configMap["admin_bark_device_key"]

	if deviceKey == "" {
		jsonResponse(c, http.StatusBadRequest, false, "Bark Device Key 未配置", nil)
		return
	}

	if serverURL == "" {
		serverURL = "https://api.day.app"
	}

	notificationService := notification.NewNotificationService()
	testData := map[string]interface{}{
		"type":    "test",
		"message": "这是一条测试消息，用于验证 Bark 通知功能是否正常工作。",
	}

	// 发送测试通知
	go func() {
		_ = notificationService.SendAdminNotification("test", testData)
	}()

	jsonResponse(c, http.StatusOK, true, "测试消息已发送，请检查您的设备", nil)
}

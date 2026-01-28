package handlers

import (
	"encoding/json"
	"fmt"
	"io"
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
	"cboard-go/internal/services/geoip"
	"cboard-go/internal/services/notification"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func updateSettingsCommon(c *gin.Context, category string) {
	var settings map[string]interface{}
	if err := c.ShouldBindJSON(&settings); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		for key, val := range settings {
			targetCat := category
			if key == "domain_name" && category == "general" {
				targetCat = "system" // 特殊处理
			}

			valStr := fmt.Sprintf("%v", val)
			if _, ok := val.([]interface{}); ok {
				if jsonBytes, err := json.Marshal(val); err == nil {
					valStr = string(jsonBytes)
				}
			}

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
		utils.ErrorResponse(c, http.StatusInternalServerError, "保存设置失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "设置已保存", nil)
}

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
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取配置失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", configs)
}

func GetSystemConfig(c *gin.Context) {
	var config models.SystemConfig
	if err := database.GetDB().Where("key = ?", c.Param("key")).First(&config).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "配置不存在", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", config)
}

func CreateSystemConfig(c *gin.Context) {
	var req models.SystemConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	var exist models.SystemConfig
	q := db.Where("key = ?", req.Key)
	if req.Category != "" {
		q = q.Where("category = ?", req.Category)
	}

	if q.First(&exist).Error == nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "配置已存在", nil)
		return
	}

	if err := db.Create(&req).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建配置失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusCreated, "", req)
}

func UpdateSystemConfig(c *gin.Context) {
	key := c.Param("key")
	db := database.GetDB()

	if key == "batch" {
		var req map[string]interface{}
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
			return
		}
		for k, v := range req {
			val := fmt.Sprintf("%v", v)
			db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "key"}}, // 假设 key 是唯一索引
				DoUpdates: clause.Assignments(map[string]interface{}{"value": val}),
			}).Create(&models.SystemConfig{Key: k, Value: val, Category: "system"})
		}
		utils.SuccessResponse(c, http.StatusOK, "批量更新成功", nil)
		return
	}

	var req models.SystemConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	category := req.Category
	if category == "" {
		category = "system" // 默认 category
	}

	var config models.SystemConfig
	err := db.Where("key = ? AND category = ?", key, category).First(&config).Error
	if err != nil {
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
			utils.ErrorResponse(c, http.StatusInternalServerError, "创建配置失败", err)
			return
		}
	} else {
		config.Value = req.Value
		if req.Type != "" {
			config.Type = req.Type
		}
		if req.DisplayName != "" {
			config.DisplayName = req.DisplayName
		}
		if err := db.Save(&config).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "更新配置失败", err)
			return
		}
	}
	utils.SuccessResponse(c, http.StatusOK, "更新成功", config)
}

func GetAdminSettings(c *gin.Context) {
	settings := map[string]map[string]interface{}{
		"general": {
			"site_name": "CBoard Modern", "site_description": "现代化的代理服务管理平台", "site_logo": "", "default_theme": "default",
			"support_qq": "", "support_email": "",
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
		"notification": {
			"system_notifications":              "true",
			"email_notifications":               "true",
			"subscription_expiry_notifications": "true",
			"new_user_notifications":            "true",
			"new_order_notifications":           "true",
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

	db := database.GetDB()
	var configs []models.SystemConfig
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

	stringOnlyFields := map[string]bool{
		"admin_telegram_chat_id":   true,
		"admin_telegram_bot_token": true,
		"admin_bark_device_key":    true,
		"admin_notification_email": true,
		"admin_bark_server_url":    true,
		"support_qq":               true,
		"support_email":            true,
		"domain_name":              true,
	}

	for cat, catDefaults := range settings {
		for key := range catDefaults {
			if val, ok := configMap[cat][key]; ok {
				if val == "true" || val == "false" {
					settings[cat][key] = (val == "true")
				} else if strings.HasPrefix(val, "[") {
					var arr []string
					if json.Unmarshal([]byte(val), &arr) == nil {
						settings[cat][key] = arr
					} else {
						settings[cat][key] = val
					}
				} else if stringOnlyFields[key] {
					settings[cat][key] = val
				} else if num, err := strconv.Atoi(val); err == nil {
					settings[cat][key] = num
				} else {
					settings[cat][key] = val
				}
			}
		}
	}
	if val, ok := configMap["system"]["domain_name"]; ok {
		settings["general"]["domain_name"] = val
	}

	utils.SuccessResponse(c, http.StatusOK, "", settings)
}

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

func UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "文件上传失败", err)
		return
	}

	cfg := config.AppConfig
	maxSize := int64(10 * 1024 * 1024)
	if cfg != nil && cfg.MaxFileSize > 0 {
		maxSize = cfg.MaxFileSize
	}

	if file.Size > maxSize {
		utils.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("文件超限 (Max %d MB)", maxSize>>20), nil)
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".pdf": true, ".txt": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".zip": true, ".rar": true}
	if !allowed[ext] {
		utils.ErrorResponse(c, http.StatusBadRequest, "不支持的文件类型", nil)
		return
	}

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
		utils.ErrorResponse(c, http.StatusBadRequest, "非法路径", nil)
		return
	}

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "系统错误", err)
		return
	}

	if err := c.SaveUploadedFile(file, filepath.Join(uploadDir, safeName)); err != nil {
		utils.LogError("UploadFile", err, nil)
		utils.ErrorResponse(c, http.StatusInternalServerError, "保存失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "上传成功", gin.H{"url": "/" + filepath.Join(uploadDir, safeName), "filename": safeName})
}

func GetPublicSettings(c *gin.Context) {
	var configs []models.SystemConfig
	db := database.GetDB()
	db.Where("is_public = ?", true).Find(&configs)
	settings := make(map[string]interface{})
	for _, conf := range configs {
		settings[conf.Key] = conf.Value
	}

	var registrationConfigs []models.SystemConfig
	db.Where("category = ?", "registration").Find(&registrationConfigs)
	for _, conf := range registrationConfigs {
		if conf.Key == "email_verification_required" || conf.Key == "registration_enabled" || conf.Key == "invite_code_required" {
			settings[conf.Key] = conf.Value == "true"
		} else if conf.Key == "min_password_length" || conf.Key == "default_subscription_device_limit" || conf.Key == "default_subscription_duration_months" {
			if val, err := strconv.Atoi(conf.Value); err == nil {
				settings[conf.Key] = val
			} else {
				settings[conf.Key] = conf.Value
			}
		} else {
			settings[conf.Key] = conf.Value
		}
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

	var supportQQConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "support_qq", "general").First(&supportQQConfig).Error; err == nil && supportQQConfig.Value != "" {
		settings["support_qq"] = strings.TrimSpace(supportQQConfig.Value)
	} else {
		settings["support_qq"] = "" // 不设置默认值
	}

	var supportEmailConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "support_email", "general").First(&supportEmailConfig).Error; err == nil && supportEmailConfig.Value != "" {
		settings["support_email"] = strings.TrimSpace(supportEmailConfig.Value)
	} else {
		settings["support_email"] = "" // 不设置默认值
	}

	utils.SuccessResponse(c, http.StatusOK, "", settings)
}

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
		utils.ErrorResponse(c, http.StatusBadRequest, "管理员邮箱未配置", nil)
		return
	}

	emailService := email.NewEmailService()
	templateBuilder := email.NewEmailTemplateBuilder()
	subject := "测试邮件通知"
	content := templateBuilder.GetAdminNotificationTemplate("test", "测试通知", "这是一条测试消息，用于验证邮件通知功能是否正常工作。", map[string]interface{}{})

	if err := emailService.QueueEmail(adminEmail, subject, content, "admin_notification"); err != nil {
		utils.LogError("TestAdminEmailNotification", err, nil)
		utils.ErrorResponse(c, http.StatusInternalServerError, "发送测试邮件失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "测试邮件已加入队列，请检查您的邮箱", nil)
}

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
		utils.ErrorResponse(c, http.StatusBadRequest, "Telegram Bot Token 或 Chat ID 未配置", nil)
		return
	}

	notificationService := notification.NewNotificationService()
	testTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
	testData := map[string]interface{}{
		"type":      "test",
		"test_time": testTime,
	}

	go func() {
		_ = notificationService.SendAdminNotification("test", testData)
	}()

	utils.SuccessResponse(c, http.StatusOK, "测试消息已发送，请检查您的 Telegram", nil)
}

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
		utils.ErrorResponse(c, http.StatusBadRequest, "Bark Device Key 未配置", nil)
		return
	}

	if serverURL == "" {
		serverURL = "https://api.day.app"
	}

	notificationService := notification.NewNotificationService()
	testTime := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
	testData := map[string]interface{}{
		"type":      "test",
		"test_time": testTime,
	}

	go func() {
		_ = notificationService.SendAdminNotification("test", testData)
	}()

	utils.SuccessResponse(c, http.StatusOK, "测试消息已发送，请检查您的设备", nil)
}

func UpdateGeoIPDatabase(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未授权", nil)
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, userID).Error; err != nil || !user.IsAdmin {
		utils.ErrorResponse(c, http.StatusForbidden, "需要管理员权限", err)
		return
	}

	geoipPath := os.Getenv("GEOIP_DB_PATH")
	if geoipPath == "" {
		geoipPath = "./GeoLite2-City.mmdb"
	}

	geoipURL := "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb"

	tmpFile := geoipPath + ".tmp"

	resp, err := http.Get(geoipURL)
	if err != nil {
		utils.LogError("UpdateGeoIPDatabase: 下载失败", err, nil)
		utils.ErrorResponse(c, http.StatusInternalServerError, "下载 GeoIP 数据库失败: "+err.Error(), err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		utils.ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("下载失败，状态码: %d", resp.StatusCode), nil)
		return
	}

	out, err := os.Create(tmpFile)
	if err != nil {
		utils.LogError("UpdateGeoIPDatabase: 创建临时文件失败", err, nil)
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建临时文件失败: "+err.Error(), err)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(tmpFile)
		utils.LogError("UpdateGeoIPDatabase: 保存文件失败", err, nil)
		utils.ErrorResponse(c, http.StatusInternalServerError, "保存文件失败: "+err.Error(), err)
		return
	}
	out.Close()

	if err := os.Rename(tmpFile, geoipPath); err != nil {
		os.Remove(tmpFile)
		utils.LogError("UpdateGeoIPDatabase: 替换文件失败", err, nil)
		utils.ErrorResponse(c, http.StatusInternalServerError, "替换文件失败: "+err.Error(), err)
		return
	}

	if err := geoip.InitGeoIP(geoipPath); err != nil {
		utils.SuccessResponse(c, http.StatusOK, "文件下载成功，但重新加载失败: "+err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "GeoIP 数据库更新成功", nil)
}

func GetGeoIPStatus(c *gin.Context) {
	geoipPath := os.Getenv("GEOIP_DB_PATH")
	if geoipPath == "" {
		geoipPath = "./GeoLite2-City.mmdb"
	}

	status := map[string]interface{}{
		"enabled":     geoip.IsEnabled(),
		"db_path":     geoipPath,
		"db_exists":   false,
		"db_size":     int64(0),
		"db_modified": "",
	}

	if info, err := os.Stat(geoipPath); err == nil {
		status["db_exists"] = true
		status["db_size"] = info.Size()
		status["db_modified"] = info.ModTime().Format("2006-01-02 15:04:05")
	}

	utils.SuccessResponse(c, http.StatusOK, "", status)
}

func GetMobileConfig(c *gin.Context) {
	db := database.GetDB()
	baseURL := utils.GetBuildBaseURL(c.Request, db)

	var configs []models.SystemConfig
	db.Where("category IN (?)", []string{"software", "general"}).Find(&configs)

	configMap := make(map[string]string)
	for _, config := range configs {
		configMap[config.Key] = config.Value
	}

	var banners []string
	bannersConfig := configMap["mobile_banners"]
	if bannersConfig != "" {
		if err := json.Unmarshal([]byte(bannersConfig), &banners); err != nil {
			bannerList := strings.Split(bannersConfig, ",")
			for _, banner := range bannerList {
				if trimmed := strings.TrimSpace(banner); trimmed != "" {
					banners = append(banners, trimmed)
				}
			}
		}
	}

	configData := gin.H{
		"baseURL":         baseURL + "/api/v1/",
		"baseDYURL":       configMap["mobile_base_dy_url"], // 可选：用于测试节点的 URL
		"mainregisterURL": baseURL + "/#/register?code=",
		"paymentURL":      baseURL + "/#/payment",
		"telegramurl":     configMap["telegram_url"],
		"kefuurl":         baseURL + "/#/tickets",
		"websiteURL":      baseURL,
		"crisptoken":      configMap["crisp_token"], // 如果使用 Crisp 客服
		"banners":         banners,
		"message":         "OK",
		"code":            1,
	}

	utils.SuccessResponse(c, http.StatusOK, "Mobile config fetched successfully", configData)
}

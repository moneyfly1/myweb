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

const (
	CatSystem            = "system"
	CatGeneral           = "general"
	CatRegistration      = "registration"
	CatAnnouncement      = "announcement"
	CatAdminNotification = "admin_notification"
)

// Helper: Fetch configs into a map[key]value for given categories
func getConfigMap(categories ...string) map[string]string {
	db := database.GetDB()
	var configs []models.SystemConfig
	query := db.Model(&models.SystemConfig{})
	if len(categories) > 0 {
		query = query.Where("category IN ?", categories)
	}
	query.Find(&configs)

	res := make(map[string]string, len(configs))
	for _, c := range configs {
		res[c.Key] = c.Value
	}
	return res
}

func updateSettingsCommon(c *gin.Context, category string) {
	var settings map[string]interface{}
	if err := c.ShouldBindJSON(&settings); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		for key, val := range settings {
			targetCat := category
			if key == "domain_name" && category == CatGeneral {
				targetCat = CatSystem
			}

			valStr := fmt.Sprintf("%v", val)
			if arr, ok := val.([]interface{}); ok {
				if b, err := json.Marshal(arr); err == nil {
					valStr = string(b)
				}
			}

			var conf models.SystemConfig
			// Check existence by Key + Category
			if err := tx.Where("key = ? AND category = ?", key, targetCat).FirstOrInit(&conf).Error; err != nil {
				return err
			}

			conf.Key = key
			conf.Category = targetCat
			conf.Value = valStr

			if err := tx.Save(&conf).Error; err != nil {
				return err
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
	query := database.GetDB().Order("sort_order ASC")
	if cat := c.Query("category"); cat != "" {
		query = query.Where("category = ?", cat)
	}
	if c.Query("is_public") == "true" {
		query = query.Where("is_public = ?", true)
	}

	var configs []models.SystemConfig
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
	q := db.Where("key = ?", req.Key)
	if req.Category != "" {
		q = q.Where("category = ?", req.Category)
	}

	var count int64
	q.Model(&models.SystemConfig{}).Count(&count)
	if count > 0 {
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

	// Batch Update Mode
	if key == "batch" {
		var req map[string]interface{}
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
			return
		}

		// Use transaction for batch upsert correctness
		err := db.Transaction(func(tx *gorm.DB) error {
			for k, v := range req {
				val := fmt.Sprintf("%v", v)
				// Assuming 'key' is unique enough or schema allows this Upsert
				if err := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "key"}},
					DoUpdates: clause.Assignments(map[string]interface{}{"value": val}),
				}).Create(&models.SystemConfig{Key: k, Value: val, Category: CatSystem}).Error; err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "批量更新失败", err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "批量更新成功", nil)
		return
	}

	// Single Update Mode
	var req models.SystemConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	category := req.Category
	if category == "" {
		category = CatSystem
	}

	var config models.SystemConfig
	if err := db.Where("key = ? AND category = ?", key, category).FirstOrInit(&config).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询配置失败", err)
		return
	}

	config.Key = key
	config.Category = category
	config.Value = req.Value
	if req.Type != "" {
		config.Type = req.Type
	}
	if req.DisplayName != "" {
		config.DisplayName = req.DisplayName
	}
	if config.Type == "" {
		config.Type = "string"
	}

	if err := db.Save(&config).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "保存配置失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "更新成功", config)
}

func GetAdminSettings(c *gin.Context) {
	// Define defaults
	settings := map[string]map[string]interface{}{
		CatGeneral: {
			"site_name": "CBoard Modern", "site_description": "现代化的代理服务管理平台", "site_logo": "", "default_theme": "default",
			"support_qq": "", "support_email": "", "unified_auth_enabled": "false", "domain_name": "",
		},
		CatRegistration: {
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
		CatAnnouncement: {
			"announcement_enabled": "false", "announcement_content": "",
		},
		"node_health": {
			"node_health_check_interval": "300", "node_max_latency": "3000", "node_test_timeout": "5", "test_url": "https://ping.pe",
		},
		"custom_node": {},
		"notification": {
			"system_notifications": "true", "email_notifications": "true", "subscription_expiry_notifications": "true",
			"new_user_notifications": "true", "new_order_notifications": "true",
		},
		CatAdminNotification: {
			"admin_notification_enabled": "false", "admin_email_notification": "false", "admin_telegram_notification": "false",
			"admin_bark_notification": "false", "admin_telegram_bot_token": "", "admin_telegram_chat_id": "",
			"admin_bark_server_url": "https://api.day.app", "admin_bark_device_key": "", "admin_notification_email": "",
			"admin_notify_order_paid": "false", "admin_notify_user_registered": "false", "admin_notify_password_reset": "false",
			"admin_notify_subscription_sent": "false", "admin_notify_subscription_reset": "false", "admin_notify_subscription_expired": "false",
			"admin_notify_user_created": "false", "admin_notify_subscription_created": "false",
		},
		"backup": {
			"backup_target":        "gitee",
			"backup_gitee_enabled": "false", "backup_gitee_token": "", "backup_gitee_owner": "moneyfly",
			"backup_gitee_repo":     "backup",
			"backup_github_enabled": "false", "backup_github_token": "", "backup_github_owner": "moneyfly1",
			"backup_github_repo":  "backup",
			"backup_auto_enabled": "false", "backup_auto_interval": "24",
		},
	}

	// Fetch all relevant configs
	cats := make([]string, 0, len(settings)+1)
	for k := range settings {
		cats = append(cats, k)
	}
	cats = append(cats, CatSystem)

	// Always treat these fields as strings regardless of content
	stringOnlyFields := map[string]bool{
		"admin_telegram_chat_id": true, "admin_telegram_bot_token": true,
		"admin_bark_device_key": true, "admin_notification_email": true,
		"admin_bark_server_url": true, "support_qq": true, "support_email": true, "domain_name": true,
	}

	// Optimized merge logic
	var allConfigs []models.SystemConfig
	database.GetDB().Where("category IN ?", cats).Find(&allConfigs)

	// Transform slice to nested map for precise lookup: map[category][key]value
	preciseMap := make(map[string]map[string]string)
	for _, c := range allConfigs {
		if _, ok := preciseMap[c.Category]; !ok {
			preciseMap[c.Category] = make(map[string]string)
		}
		preciseMap[c.Category][c.Key] = c.Value
	}

	// Apply values
	for cat, catDefaults := range settings {
		for key, defaultVal := range catDefaults {
			val, exists := preciseMap[cat][key]

			// Special handle: domain_name in 'general' might be stored in 'system'
			if !exists && key == "domain_name" && cat == CatGeneral {
				val, exists = preciseMap[CatSystem][key]
			}

			if exists {
				if stringOnlyFields[key] {
					settings[cat][key] = val
					continue
				}

				// Type conversion based on value content
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
			} else {
				// Convert default value string booleans to actual booleans
				if s, ok := defaultVal.(string); ok && (s == "true" || s == "false") {
					settings[cat][key] = (s == "true")
				}
			}
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", settings)
}

func UpdateGeneralSettings(c *gin.Context)      { updateSettingsCommon(c, CatGeneral) }
func UpdateRegistrationSettings(c *gin.Context) { updateSettingsCommon(c, CatRegistration) }
func UpdateSecuritySettings(c *gin.Context)     { updateSettingsCommon(c, "security") }
func UpdateThemeSettings(c *gin.Context)        { updateSettingsCommon(c, "theme") }
func UpdateInviteSettings(c *gin.Context)       { updateSettingsCommon(c, "invite") }
func UpdateSoftwareConfig(c *gin.Context)       { updateSettingsCommon(c, "software") }
func UpdateAnnouncementSettings(c *gin.Context) { updateSettingsCommon(c, CatAnnouncement) }
func UpdateNotificationSettings(c *gin.Context) { updateSettingsCommon(c, "notification") }
func UpdateAdminNotificationSystemSettings(c *gin.Context) {
	updateSettingsCommon(c, CatAdminNotification)
}
func UpdateNodeHealthSettings(c *gin.Context) { updateSettingsCommon(c, "node_health") }
func UpdateBackupSettings(c *gin.Context)     { updateSettingsCommon(c, "backup") }

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
	allowedExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".pdf": true, ".txt": true, ".doc": true, ".docx": true,
		".xls": true, ".xlsx": true, ".zip": true, ".rar": true,
	}
	if !allowedExts[ext] {
		utils.ErrorResponse(c, http.StatusBadRequest, "不支持的文件类型", nil)
		return
	}

	// Validate content
	f, err := file.Open()
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无法读取文件", err)
		return
	}
	defer f.Close()

	buffer := make([]byte, 512)
	n, err := f.Read(buffer)
	if err != nil && err != io.EOF {
		utils.ErrorResponse(c, http.StatusBadRequest, "文件读取失败", err)
		return
	}

	contentType := http.DetectContentType(buffer[:n])
	allowedMimeTypes := map[string]bool{
		"image/jpeg": true, "image/png": true, "image/gif": true,
		"application/pdf": true, "text/plain": true, "application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/vnd.ms-excel": true, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
		"application/zip": true, "application/x-rar-compressed": true, "application/x-zip-compressed": true,
	}

	if !allowedMimeTypes[contentType] && !strings.HasPrefix(contentType, "application/octet-stream") {
		// Allow generic binaries if extensions match archives, otherwise block
		if ext != ".zip" && ext != ".rar" {
			utils.ErrorResponse(c, http.StatusBadRequest, "文件类型验证失败", nil)
			return
		}
	}

	// Reset pointer
	f.Seek(0, 0)

	safeName := fmt.Sprintf("%d_%s", time.Now().Unix(), utils.SanitizeInput(file.Filename))
	if utils.SanitizeInput(file.Filename) == "" {
		safeName = fmt.Sprintf("%d_file%s", time.Now().Unix(), ext)
	}

	uploadDir := "uploads"
	if cfg != nil && cfg.UploadDir != "" {
		uploadDir = cfg.UploadDir
	}

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "系统错误", err)
		return
	}

	fullPath := filepath.Join(uploadDir, safeName)
	if err := c.SaveUploadedFile(file, fullPath); err != nil {
		utils.LogError("UploadFile", err, nil)
		utils.ErrorResponse(c, http.StatusInternalServerError, "保存失败", err)
		return
	}

	_ = os.Chmod(fullPath, 0644) // Best effort chmod

	utils.SuccessResponse(c, http.StatusOK, "上传成功", gin.H{
		"url":      "/" + filepath.ToSlash(fullPath), // Ensure forward slashes for URLs
		"filename": safeName,
	})
}

func GetPublicSettings(c *gin.Context) {
	// Consolidate queries
	db := database.GetDB()
	var configs []models.SystemConfig

	// Fetch all public configs + specific categories we need internally
	db.Where("is_public = ? OR category IN ?", true, []string{CatRegistration, CatAnnouncement, CatGeneral}).Find(&configs)

	// Index by Key (last one wins if duplicates, but category logic below clarifies)
	configMap := make(map[string]models.SystemConfig)
	for _, c := range configs {
		configMap[c.Key] = c // This stores the whole object, allowing category checks
	}

	settings := make(map[string]interface{})

	// 1. Populate explicitly public settings
	for _, c := range configs {
		if c.IsPublic {
			settings[c.Key] = c.Value
		}
	}

	// 2. Registration Logic
	regKeysBool := []string{"email_verification_required", "registration_enabled", "invite_code_required"}
	regKeysInt := []string{"min_password_length", "default_subscription_device_limit", "default_subscription_duration_months"}

	for _, k := range regKeysBool {
		if conf, ok := configMap[k]; ok && conf.Category == CatRegistration {
			settings[k] = (conf.Value == "true")
		}
	}
	for _, k := range regKeysInt {
		if conf, ok := configMap[k]; ok && conf.Category == CatRegistration {
			if val, err := strconv.Atoi(conf.Value); err == nil {
				settings[k] = val
			} else {
				settings[k] = conf.Value
			}
		}
	}

	// 3. Announcement Logic
	if conf, ok := configMap["announcement_enabled"]; ok && conf.Category == CatAnnouncement && conf.Value == "true" {
		settings["announcement_enabled"] = true
		if content, ok := configMap["announcement_content"]; ok && content.Category == CatAnnouncement {
			settings["announcement_content"] = content.Value
		}
	} else {
		settings["announcement_enabled"] = false
	}

	// 4. Support & Auth Logic (General Category)
	generalKeys := []string{"support_qq", "support_email"}
	for _, k := range generalKeys {
		if conf, ok := configMap[k]; ok && conf.Category == CatGeneral {
			settings[k] = strings.TrimSpace(conf.Value)
		} else {
			settings[k] = ""
		}
	}

	if conf, ok := configMap["unified_auth_enabled"]; ok && conf.Category == CatGeneral {
		settings["unified_auth_enabled"] = (conf.Value == "true")
	} else {
		settings["unified_auth_enabled"] = false
	}

	utils.SuccessResponse(c, http.StatusOK, "", settings)
}

func TestAdminEmailNotification(c *gin.Context) {
	configMap := getConfigMap(CatAdminNotification)
	adminEmail := configMap["admin_notification_email"]
	if adminEmail == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "管理员邮箱未配置", nil)
		return
	}

	subject := "测试邮件通知"
	content := email.NewEmailTemplateBuilder().GetAdminNotificationTemplate("test", "测试通知", "这是一条测试消息，用于验证邮件通知功能是否正常工作。", nil)

	if err := email.NewEmailService().QueueEmail(adminEmail, subject, content, CatAdminNotification); err != nil {
		utils.LogError("TestAdminEmailNotification", err, nil)
		utils.ErrorResponse(c, http.StatusInternalServerError, "发送测试邮件失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "测试邮件已加入队列，请检查您的邮箱", nil)
}

func TestAdminTelegramNotification(c *gin.Context) {
	configMap := getConfigMap(CatAdminNotification)
	if configMap["admin_telegram_bot_token"] == "" || configMap["admin_telegram_chat_id"] == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Telegram Bot Token 或 Chat ID 未配置", nil)
		return
	}

	sendTestNotification("Telegram")
	utils.SuccessResponse(c, http.StatusOK, "测试消息已发送，请检查您的 Telegram", nil)
}

func TestAdminBarkNotification(c *gin.Context) {
	configMap := getConfigMap(CatAdminNotification)
	if configMap["admin_bark_device_key"] == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Bark Device Key 未配置", nil)
		return
	}

	sendTestNotification("Bark")
	utils.SuccessResponse(c, http.StatusOK, "测试消息已发送，请检查您的设备", nil)
}

// Helper to fire and forget test notification
func sendTestNotification(logTag string) {
	go func() {
		testData := map[string]interface{}{
			"type":      "test",
			"test_time": utils.GetBeijingTime().Format("2006-01-02 15:04:05"),
		}
		_ = notification.NewNotificationService().SendAdminNotification("test", testData)
	}()
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

	tmpFile := geoipPath + ".tmp"
	resp, err := http.Get("https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb")
	if err != nil || resp.StatusCode != http.StatusOK {
		utils.LogError("UpdateGeoIPDatabase: Download failed", err, nil)
		utils.ErrorResponse(c, http.StatusInternalServerError, "下载失败", err)
		return
	}
	defer resp.Body.Close()

	out, err := os.Create(tmpFile)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建临时文件失败", err)
		return
	}

	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		os.Remove(tmpFile)
		utils.ErrorResponse(c, http.StatusInternalServerError, "保存文件失败", err)
		return
	}
	out.Close()

	if err := os.Rename(tmpFile, geoipPath); err != nil {
		os.Remove(tmpFile)
		utils.ErrorResponse(c, http.StatusInternalServerError, "替换文件失败", err)
		return
	}

	if err := geoip.InitGeoIP(geoipPath); err != nil {
		utils.SuccessResponse(c, http.StatusOK, "更新成功，但重载失败: "+err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "GeoIP 数据库更新成功", nil)
}

func GetGeoIPStatus(c *gin.Context) {
	geoipPath := os.Getenv("GEOIP_DB_PATH")
	if geoipPath == "" {
		geoipPath = "./GeoLite2-City.mmdb"
	}

	status := gin.H{
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
	configMap := getConfigMap("software", "general")

	var banners []string
	if bannersConfig := configMap["mobile_banners"]; bannersConfig != "" {
		if err := json.Unmarshal([]byte(bannersConfig), &banners); err != nil {
			for _, b := range strings.Split(bannersConfig, ",") {
				if t := strings.TrimSpace(b); t != "" {
					banners = append(banners, t)
				}
			}
		}
	}

	baseURL := utils.GetBuildBaseURL(c.Request, db)
	utils.SuccessResponse(c, http.StatusOK, "Mobile config fetched", gin.H{
		"baseURL":         baseURL + "/api/v1/",
		"baseDYURL":       configMap["mobile_base_dy_url"],
		"mainregisterURL": baseURL + "/#/register?code=",
		"paymentURL":      baseURL + "/#/payment",
		"telegramurl":     configMap["telegram_url"],
		"kefuurl":         baseURL + "/#/tickets",
		"websiteURL":      baseURL,
		"crisptoken":      configMap["crisp_token"],
		"banners":         banners,
		"message":         "OK",
		"code":            1,
	})
}

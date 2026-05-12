package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cboard-go/internal/core/cache"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"
	"cboard-go/internal/utils"
)

var (
	adminNotificationEvents = []string{
		"user_registered",
		"user_created",
		"password_reset",
		"password_changed",
		"order_paid",
		"recharge_paid",
		"subscription_created",
		"subscription_sent",
		"subscription_reset",
		"subscription_expired",
		"ticket_created",
		"ticket_replied",
		"abnormal_login",
	}

	// adminNotificationKeyMap 将事件类型映射到配置键
	adminNotificationKeyMap = map[string]string{
		"order_paid":           "admin_notify_order_paid",
		"recharge_paid":        "admin_notify_recharge_paid",
		"user_registered":      "admin_notify_user_registered",
		"password_reset":       "admin_notify_password_reset",
		"password_changed":     "admin_notify_password_changed",
		"subscription_sent":    "admin_notify_subscription_sent",
		"subscription_reset":   "admin_notify_subscription_reset",
		"subscription_expired": "admin_notify_subscription_expired",
		"user_created":         "admin_notify_user_created",
		"subscription_created": "admin_notify_subscription_created",
		"ticket_created":       "admin_notify_ticket_created",
		"ticket_replied":       "admin_notify_ticket_replied",
		"abnormal_login":       "admin_notify_abnormal_login",
	}

	// #nosec G101 - 这些是邮件/通知的主题模板，不是敏感凭证
	notificationSubjects = map[string]string{
		"order_paid":           "💰 新订单支付成功",
		"recharge_paid":        "💳 用户充值成功",
		"user_registered":      "👤 新用户注册",
		"password_reset":       "🔐 用户重置密码",
		"password_changed":     "🔐 用户修改密码",
		"subscription_sent":    "📧 用户发送订阅",
		"subscription_reset":   "🔄 用户重置订阅",
		"subscription_expired": "⏰ 订阅已过期",
		"user_created":         "📋 管理员创建用户",
		"subscription_created": "📦 订阅创建",
		"ticket_created":       "🎫 用户提交工单",
		"ticket_replied":       "💬 工单新回复",
		"abnormal_login":       "⚠️ 异常登录告警",
	}
)

const (
	ChannelSystem   = "system"
	ChannelEmail    = "email"
	ChannelTelegram = "telegram"
	ChannelBark     = "bark"
)

var customerNotificationKeyMap = map[string]string{
	"new_user":             "new_user_notifications",
	"user_registered":      "new_user_notifications",
	"new_order":            "new_order_notifications",
	"order_paid":           "new_order_notifications",
	"payment_success":      "new_order_notifications",
	"recharge_paid":        "recharge_success_notifications",
	"recharge_success":     "recharge_success_notifications",
	"subscription_created": "subscription_created_notifications",
	"subscription_sent":    "subscription_sent_notifications",
	"subscription_reset":   "subscription_reset_notifications",
	"subscription_expiry":  "subscription_expiry_notifications",
	"subscription_expired": "subscription_expiry_notifications",
	"ticket_reply":         "ticket_reply_notifications",
	"ticket_replied":       "ticket_reply_notifications",
	"password_changed":     "password_changed_notifications",
	"password_reset":       "password_reset_notifications",
	"abnormal_login":       "abnormal_login_notifications",
	"system":               "system_notifications",
	"email":                "email_notifications",
}

var customerNotificationLegacyTypeMap = map[string]string{
	"new_user":             "system",
	"user_registered":      "system",
	"new_order":            "payment",
	"order_paid":           "payment",
	"payment_success":      "payment",
	"recharge_paid":        "payment",
	"recharge_success":     "payment",
	"subscription_created": "subscription",
	"subscription_sent":    "subscription",
	"subscription_reset":   "subscription",
	"subscription_expiry":  "subscription",
	"subscription_expired": "subscription",
	"ticket_reply":         "ticket",
	"ticket_replied":       "ticket",
	"password_changed":     "security",
	"password_reset":       "security",
	"abnormal_login":       "security",
}

func AdminNotificationDefaultSettings() map[string]interface{} {
	defaults := map[string]interface{}{
		"admin_notification_enabled":         "false",
		"admin_email_notification":           "false",
		"admin_telegram_notification":        "false",
		"admin_bark_notification":            "false",
		"admin_telegram_bot_token":           "",
		"admin_telegram_chat_id":             "",
		"admin_bark_server_url":              "https://api.day.app",
		"admin_bark_device_key":              "",
		"admin_notification_email":           "",
		"admin_abnormal_login_alert_enabled": true,
		"admin_notify_user_registered":       "false",
		"admin_notify_user_created":          "false",
		"admin_notify_password_reset":        "false",
		"admin_notify_password_changed":      "false",
		"admin_notify_order_paid":            "false",
		"admin_notify_recharge_paid":         "false",
		"admin_notify_subscription_created":  "false",
		"admin_notify_subscription_sent":     "false",
		"admin_notify_subscription_reset":    "false",
		"admin_notify_subscription_expired":  "false",
		"admin_notify_ticket_created":        "false",
		"admin_notify_ticket_replied":        "false",
		"admin_notify_abnormal_login":        "false",
	}
	for _, event := range adminNotificationEvents {
		baseKey, ok := adminNotificationKeyMap[event]
		if !ok {
			continue
		}
		defaults[baseKey+"_"+ChannelEmail] = "false"
		defaults[baseKey+"_"+ChannelTelegram] = "false"
		defaults[baseKey+"_"+ChannelBark] = "false"
	}
	return defaults
}

func AdminNotificationEventKeys() []string {
	keys := make([]string, 0, len(adminNotificationEvents))
	for _, event := range adminNotificationEvents {
		if key, ok := adminNotificationKeyMap[event]; ok {
			keys = append(keys, key)
		}
	}
	return keys
}

func CustomerNotificationDefaultSettings() map[string]interface{} {
	return map[string]interface{}{
		"email_notifications":                         "true",
		"subscription_expiry_notifications":           "true",
		"subscription_created_notifications":          "true",
		"subscription_sent_notifications":             "true",
		"subscription_reset_notifications":            "true",
		"new_user_notifications":                      "true",
		"new_order_notifications":                     "true",
		"recharge_success_notifications":              "true",
		"ticket_reply_notifications":                  "true",
		"password_changed_notifications":              "true",
		"password_reset_notifications":                "true",
		"abnormal_login_notifications":                "true",
		"subscription_expiry_reminder_cooldown_hours": 24,
		"subscription_expiry_reminder_daily_limit":    1,
		"user_registered_email_notifications":         "true",
		"order_paid_email_notifications":              "true",
		"recharge_paid_email_notifications":           "true",
		"subscription_created_email_notifications":    "true",
		"subscription_sent_email_notifications":       "true",
		"subscription_reset_email_notifications":      "true",
		"subscription_expiry_email_notifications":     "true",
		"ticket_reply_email_notifications":            "true",
		"password_changed_email_notifications":        "true",
		"password_reset_email_notifications":          "true",
		"abnormal_login_email_notifications":          "true",
	}
}

// loadConfigFromDB 提取了从数据库获取配置并转换为 Map 的公共逻辑
func loadConfigFromDB(category string) map[string]string {
	db := database.GetDB()
	if db == nil {
		return make(map[string]string)
	}

	var configs []models.SystemConfig
	db.Where("category = ?", category).Find(&configs)

	configMap := make(map[string]string, len(configs))
	for _, config := range configs {
		configMap[config.Key] = config.Value
	}
	return configMap
}

func configBool(configMap map[string]string, key string, defaultValue bool) bool {
	value, ok := configMap[key]
	if !ok || value == "" {
		return defaultValue
	}
	return value == "true" || value == "1"
}

func ShouldSendCustomerNotification(notificationType string) bool {
	return ShouldSendCustomerNotificationChannel(notificationType, ChannelEmail)
}

func ShouldSendCustomerNotificationChannel(notificationType, channel string) bool {
	// 客户通知只支持邮件。站内通知属于产品内功能提示，不作为外部通知渠道配置。
	if channel == ChannelSystem {
		return true
	}
	if channel != ChannelEmail {
		return false
	}

	db := database.GetDB()
	if db == nil {
		return true // 默认发送
	}

	configMap := loadConfigFromDB("notification")

	if !configBool(configMap, "email_notifications", true) {
		return false
	}

	eventKey := customerNotificationKeyMap[notificationType]
	if eventKey == "" {
		return true
	}
	if !configBool(configMap, eventKey, true) {
		return false
	}

	channelEventKey := fmt.Sprintf("%s_%s_notifications", notificationType, channel)
	if _, ok := configMap[channelEventKey]; ok {
		return configBool(configMap, channelEventKey, channel == ChannelEmail)
	}

	return channel == ChannelEmail
}

func UserAllowsCustomerNotification(user *models.User, notificationType, channel string) bool {
	if user == nil {
		return false
	}
	switch channel {
	case ChannelSystem:
		return true
	case ChannelEmail:
		if !user.EmailNotifications || user.Email == "" {
			return false
		}
	default:
		return false
	}

	if user.NotificationTypes == "" {
		return true
	}

	var types []string
	if err := json.Unmarshal([]byte(user.NotificationTypes), &types); err != nil || len(types) == 0 {
		return true
	}

	category := customerNotificationLegacyTypeMap[notificationType]
	if category == "" {
		category = notificationType
	}
	for _, item := range types {
		if item == category || item == notificationType {
			return true
		}
	}
	return false
}

func ShouldSendCustomerNotificationToUser(user *models.User, notificationType, channel string) bool {
	return ShouldSendCustomerNotificationChannel(notificationType, channel) && UserAllowsCustomerNotification(user, notificationType, channel)
}

type NotificationService struct{}

func NewNotificationService() *NotificationService {
	return &NotificationService{}
}

// getAdminConfig 获取并缓存管理员通知配置
func (s *NotificationService) getAdminConfig() map[string]string {
	const cacheKey = "admin_notification_config"
	const cacheTTL = 2 * time.Minute

	var configMap map[string]string
	if cache.IsRedisEnabled() {
		if cached, err := cache.Get(cacheKey); err == nil && cached != "" {
			_ = json.Unmarshal([]byte(cached), &configMap)
			if configMap != nil {
				return configMap
			}
		}
	}

	configMap = loadConfigFromDB("admin_notification")

	if cache.IsRedisEnabled() && len(configMap) > 0 {
		if b, err := json.Marshal(configMap); err == nil {
			_ = cache.Set(cacheKey, string(b), cacheTTL)
		}
	}

	return configMap
}

func ClearAdminNotificationCache() {
	if cache.IsRedisEnabled() {
		_ = cache.Del("admin_notification_config")
	}
}

func adminChannelEnabled(configMap map[string]string, notificationType, channel string) bool {
	if notificationType == "test" {
		if testChannel := configMap["test_channel"]; testChannel != "" {
			return testChannel == channel
		}
		return configBool(configMap, fmt.Sprintf("admin_%s_notification", channel), false)
	}

	baseKey, ok := adminNotificationKeyMap[notificationType]
	if !ok {
		return configBool(configMap, fmt.Sprintf("admin_%s_notification", channel), false)
	}

	channelKey := fmt.Sprintf("%s_%s", baseKey, channel)
	if _, exists := configMap[channelKey]; exists {
		return configBool(configMap, channelKey, false) && configBool(configMap, fmt.Sprintf("admin_%s_notification", channel), false)
	}

	return configBool(configMap, baseKey, false) && configBool(configMap, fmt.Sprintf("admin_%s_notification", channel), false)
}

func anyAdminChannelEnabled(configMap map[string]string, notificationType string) bool {
	return adminChannelEnabled(configMap, notificationType, ChannelEmail) ||
		adminChannelEnabled(configMap, notificationType, ChannelTelegram) ||
		adminChannelEnabled(configMap, notificationType, ChannelBark)
}

func (s *NotificationService) SendAdminNotification(notificationType string, data map[string]interface{}) error {
	configMap := s.getAdminConfig()
	if data != nil {
		if testChannel, ok := data["test_channel"].(string); ok {
			configMap["test_channel"] = testChannel
		}
	}

	if notificationType != "test" && configMap["admin_notification_enabled"] != "true" {
		return nil
	}

	if !anyAdminChannelEnabled(configMap, notificationType) {
		return nil
	}

	templateBuilder := NewMessageTemplateBuilder()
	telegramMsg := templateBuilder.BuildTelegramMessage(notificationType, data)
	barkTitle, barkBody := templateBuilder.BuildBarkMessage(notificationType, data)

	s.notifyTelegram(configMap, notificationType, telegramMsg)
	s.notifyBark(configMap, notificationType, barkTitle, barkBody)
	s.notifyEmail(configMap, notificationType, barkTitle, barkBody, data)

	return nil
}

func (s *NotificationService) notifyTelegram(configMap map[string]string, notificationType, msg string) {
	if !adminChannelEnabled(configMap, notificationType, ChannelTelegram) {
		return
	}

	botToken := configMap["admin_telegram_bot_token"]
	chatID := configMap["admin_telegram_chat_id"]
	if botToken != "" && chatID != "" {
		go func() {
			success, err := sendTelegramMessage(botToken, chatID, msg)
			if err != nil {
				utils.LogErrorMsg("发送 Telegram 通知失败: type=%s, error=%v", notificationType, err)
			} else if success {
				utils.LogInfo("Telegram 通知发送成功: type=%s", notificationType)
			} else {
				utils.LogErrorMsg("Telegram 通知发送失败: type=%s, API返回失败", notificationType)
			}
		}()
	} else {
		utils.LogWarn("Telegram 通知未发送: type=%s, bot_token=%v, chat_id=%v (需要两者都配置)",
			notificationType, botToken != "", chatID != "")
	}
}

func (s *NotificationService) notifyBark(configMap map[string]string, notificationType, title, body string) {
	if !adminChannelEnabled(configMap, notificationType, ChannelBark) {
		return
	}

	serverURL := configMap["admin_bark_server_url"]
	deviceKey := configMap["admin_bark_device_key"]
	if serverURL == "" {
		serverURL = "https://api.day.app"
	}

	if serverURL != "" && deviceKey != "" {
		go func() {
			success, err := sendBarkMessage(serverURL, deviceKey, title, body)
			if err != nil {
				utils.LogErrorMsg("发送 Bark 通知失败: type=%s, server=%s, error=%v", notificationType, serverURL, err)
			} else if success {
				utils.LogInfo("Bark 通知发送成功: type=%s, server=%s", notificationType, serverURL)
			} else {
				utils.LogErrorMsg("Bark 通知发送失败: type=%s, server=%s, API返回失败", notificationType, serverURL)
			}
		}()
	} else {
		utils.LogWarn("Bark 通知未发送: type=%s, server_url或device_key未配置", notificationType)
	}
}

func (s *NotificationService) notifyEmail(configMap map[string]string, notificationType, barkTitle, barkBody string, data map[string]interface{}) {
	if !adminChannelEnabled(configMap, notificationType, ChannelEmail) {
		return
	}

	adminEmail := configMap["admin_notification_email"]
	if adminEmail != "" && strings.Contains(adminEmail, "@") {
		emailService := email.NewEmailService()
		templateBuilder := email.NewEmailTemplateBuilder()
		subject := getNotificationSubject(notificationType)
		content := templateBuilder.GetAdminNotificationTemplate(notificationType, barkTitle, barkBody, data)

		if err := emailService.QueueEmail(adminEmail, subject, content, "admin_notification"); err != nil {
			utils.LogErrorMsg("发送管理员邮件通知失败: type=%s, email=%s, error=%v", notificationType, adminEmail, err)
		} else {
			utils.LogInfo("管理员邮件通知已加入队列: type=%s, email=%s", notificationType, adminEmail)
		}
	} else {
		utils.LogWarn("管理员邮件通知未发送: type=%s, admin_email未配置或格式无效 (当前值: %s)", notificationType, adminEmail)
	}
}

func (s *NotificationService) CreateUserSystemNotification(user *models.User, notificationType, title, content string) error {
	if user == nil || user.ID == 0 {
		return nil
	}
	if !ShouldSendCustomerNotificationChannel(notificationType, "system") {
		return nil
	}
	db := database.GetDB()
	if db == nil {
		return nil
	}
	return db.Create(&models.Notification{
		UserID:  database.NullInt64(int64(user.ID)),
		Title:   title,
		Content: content,
		Type:    notificationType,
	}).Error
}

// doJSONPost 提取公共的 HTTP JSON POST 请求逻辑，减少重复代码
func doJSONPost(apiURL string, payload interface{}, result interface{}) error {
	// 验证URL以防止SSRF攻击
	if err := utils.ValidateHTTPURL(apiURL); err != nil {
		return fmt.Errorf("URL验证失败: %w", err)
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// #nosec G107 - URL is validated above with ValidateHTTPURL
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return err
		}
	}

	return nil
}

func sendTelegramMessage(botToken, chatID, message string) (bool, error) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "HTML",
	}

	var result map[string]interface{}
	if err := doJSONPost(apiURL, payload, &result); err != nil {
		return false, err
	}

	return result["ok"] == true, nil
}

func sendBarkMessage(serverURL, deviceKey, title, body string) (bool, error) {
	serverURL = strings.TrimSuffix(serverURL, "/")
	apiURL := fmt.Sprintf("%s/push", serverURL)

	payload := map[string]interface{}{
		"device_key": deviceKey,
		"title":      title,
		"body":       body,
	}

	var result map[string]interface{}
	if err := doJSONPost(apiURL, payload, &result); err != nil {
		return false, err
	}

	code, _ := result["code"].(float64)
	return code == 200, nil
}

func getNotificationSubject(notificationType string) string {
	if subject, ok := notificationSubjects[notificationType]; ok {
		return subject
	}
	return "系统通知"
}

func getString(data map[string]interface{}, key string, defaultValue string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", val)
	}
	return defaultValue
}

func getFloat(data map[string]interface{}, key string, defaultValue float64) float64 {
	if val, ok := data[key]; ok {
		if f, ok := val.(float64); ok {
			return f
		}
	}
	return defaultValue
}

func getInt(data map[string]interface{}, key string, defaultValue int) int {
	if val, ok := data[key]; ok {
		if i, ok := val.(int); ok {
			return i
		}
		if f, ok := val.(float64); ok {
			return int(f)
		}
	}
	return defaultValue
}

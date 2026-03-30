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
	// adminNotificationKeyMap 将事件类型映射到配置键
	adminNotificationKeyMap = map[string]string{
		"order_paid":           "admin_notify_order_paid",
		"user_registered":      "admin_notify_user_registered",
		"password_reset":       "admin_notify_password_reset",
		"subscription_sent":    "admin_notify_subscription_sent",
		"subscription_reset":   "admin_notify_subscription_reset",
		"subscription_expired": "admin_notify_subscription_expired",
		"user_created":         "admin_notify_user_created",
		"subscription_created": "admin_notify_subscription_created",
		"ticket_created":       "admin_notify_ticket_created",
		"ticket_replied":       "admin_notify_ticket_replied",
	}

	// #nosec G101 - 这些是邮件/通知的主题模板，不是敏感凭证
	notificationSubjects = map[string]string{
		"order_paid":           "💰 新订单支付成功",
		"user_registered":      "👤 新用户注册",
		"password_reset":       "🔐 用户重置密码",
		"subscription_sent":    "📧 用户发送订阅",
		"subscription_reset":   "🔄 用户重置订阅",
		"subscription_expired": "⏰ 订阅已过期",
		"user_created":         "📋 管理员创建用户",
		"subscription_created": "📦 订阅创建",
		"ticket_created":       "🎫 用户提交工单",
		"ticket_replied":       "💬 工单新回复",
	}
)

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

func ShouldSendCustomerNotification(notificationType string) bool {
	db := database.GetDB()
	if db == nil {
		return true // 默认发送
	}

	configMap := loadConfigFromDB("notification")

	if configMap["email_notifications"] != "true" || configMap["system_notifications"] != "true" {
		return false
	}

	switch notificationType {
	case "subscription_expiry":
		return configMap["subscription_expiry_notifications"] == "true"
	case "new_user":
		return configMap["new_user_notifications"] == "true"
	case "new_order":
		return configMap["new_order_notifications"] == "true"
	case "ticket_reply":
		return configMap["ticket_reply_notifications"] != "false" // 默认开启
	case "system", "email":
		return true
	default:
		return true
	}
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

func (s *NotificationService) SendAdminNotification(notificationType string, data map[string]interface{}) error {
	configMap := s.getAdminConfig()

	if configMap["admin_notification_enabled"] != "true" {
		return nil
	}

	if key, ok := adminNotificationKeyMap[notificationType]; ok {
		if configMap[key] != "true" {
			return nil
		}
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
	if configMap["admin_telegram_notification"] != "true" {
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
	if configMap["admin_bark_notification"] != "true" {
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
	if configMap["admin_email_notification"] != "true" {
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

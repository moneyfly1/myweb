package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"
	"cboard-go/internal/utils"
)

// ShouldSendCustomerNotification 检查是否应该发送客户通知
// notificationType: "system", "email", "subscription_expiry", "new_user", "new_order"
func ShouldSendCustomerNotification(notificationType string) bool {
	db := database.GetDB()
	if db == nil {
		return true // 默认发送
	}

	// 获取客户通知配置
	var configs []models.SystemConfig
	db.Where("category = ?", "notification").Find(&configs)

	configMap := make(map[string]string)
	for _, config := range configs {
		configMap[config.Key] = config.Value
	}

	// 检查是否启用邮件通知
	if configMap["email_notifications"] != "true" {
		return false
	}

	// 检查系统通知是否启用
	if configMap["system_notifications"] != "true" {
		return false
	}

	// 根据通知类型检查对应的开关
	switch notificationType {
	case "subscription_expiry":
		return configMap["subscription_expiry_notifications"] == "true"
	case "new_user":
		return configMap["new_user_notifications"] == "true"
	case "new_order":
		return configMap["new_order_notifications"] == "true"
	case "system", "email":
		// 系统通知和邮件通知已经通过上面的检查
		return true
	default:
		return true // 默认发送
	}
}

// NotificationService 通知服务
type NotificationService struct {
}

// NewNotificationService 创建通知服务
func NewNotificationService() *NotificationService {
	return &NotificationService{}
}

// SendAdminNotification 发送管理员通知
func (s *NotificationService) SendAdminNotification(notificationType string, data map[string]interface{}) error {
	db := database.GetDB()

	// 获取管理员通知配置
	var configs []models.SystemConfig
	db.Where("category = ?", "admin_notification").Find(&configs)

	configMap := make(map[string]string)
	for _, config := range configs {
		configMap[config.Key] = config.Value
	}

	// 检查是否启用管理员通知
	if configMap["admin_notification_enabled"] != "true" {
		return nil
	}

	// 检查该通知类型是否启用
	notificationKeyMap := map[string]string{
		"order_paid":           "admin_notify_order_paid",
		"user_registered":      "admin_notify_user_registered",
		"password_reset":       "admin_notify_password_reset",
		"subscription_sent":    "admin_notify_subscription_sent",
		"subscription_reset":   "admin_notify_subscription_reset",
		"subscription_expired": "admin_notify_subscription_expired",
		"user_created":         "admin_notify_user_created",
		"subscription_created": "admin_notify_subscription_created",
	}

	if key, ok := notificationKeyMap[notificationType]; ok {
		if configMap[key] != "true" {
			// 该通知类型未启用，直接返回
			return nil
		}
	}

	// 使用模板构建器格式化消息
	templateBuilder := NewMessageTemplateBuilder()
	telegramMsg := templateBuilder.BuildTelegramMessage(notificationType, data)
	barkTitle, barkBody := templateBuilder.BuildBarkMessage(notificationType, data)

	// 发送 Telegram 通知
	if configMap["admin_telegram_notification"] == "true" {
		botToken := configMap["admin_telegram_bot_token"]
		chatID := configMap["admin_telegram_chat_id"]
		if botToken != "" && chatID != "" {
			go func() {
				success, err := sendTelegramMessage(botToken, chatID, telegramMsg)
				if err != nil {
					utils.LogErrorMsg("发送 Telegram 通知失败: type=%s, error=%v", notificationType, err)
				} else if success {
					utils.LogInfo("Telegram 通知发送成功: type=%s", notificationType)
				} else {
					utils.LogErrorMsg("Telegram 通知发送失败: type=%s, API返回失败", notificationType)
				}
			}()
		} else {
			hasToken := botToken != ""
			hasChatID := chatID != ""
			utils.LogWarn("Telegram 通知未发送: type=%s, bot_token=%v, chat_id=%v (需要两者都配置)",
				notificationType, hasToken, hasChatID)
		}
	}

	// 发送 Bark 通知
	if configMap["admin_bark_notification"] == "true" {
		serverURL := configMap["admin_bark_server_url"]
		deviceKey := configMap["admin_bark_device_key"]
		if serverURL == "" {
			serverURL = "https://api.day.app"
		}
		if serverURL != "" && deviceKey != "" {
			go func() {
				success, err := sendBarkMessage(serverURL, deviceKey, barkTitle, barkBody)
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

	// 发送邮件通知（使用邮件模板）
	if configMap["admin_email_notification"] == "true" {
		adminEmail := configMap["admin_notification_email"]
		// 验证邮箱格式（简单验证：包含@符号）
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

	return nil
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

// getNotificationSubject 获取通知邮件主题
func getNotificationSubject(notificationType string) string {
	subjectMap := map[string]string{
		"order_paid":           "💰 新订单支付成功",
		"user_registered":      "👤 新用户注册",
		"password_reset":       "🔐 用户重置密码",
		"subscription_sent":    "📧 用户发送订阅",
		"subscription_reset":   "🔄 用户重置订阅",
		"subscription_expired": "⏰ 订阅已过期",
		"user_created":         "📋 管理员创建用户",
		"subscription_created": "📦 订阅创建",
	}
	if subject, ok := subjectMap[notificationType]; ok {
		return subject
	}
	return "系统通知"
}

// Helper functions
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

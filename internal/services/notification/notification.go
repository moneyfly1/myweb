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

// FormatTelegramMessage 格式化 Telegram 消息
func FormatTelegramMessage(notificationType string, data map[string]interface{}) string {
	switch notificationType {
	case "order_paid":
		orderNo := getString(data, "order_no", "N/A")
		username := getString(data, "username", "N/A")
		amount := getFloat(data, "amount", 0)
		packageName := getString(data, "package_name", "未知套餐")
		paymentMethod := getString(data, "payment_method", "未知")
		paymentTime := getString(data, "payment_time", "N/A")
		return fmt.Sprintf(`💰 <b>新订单支付成功</b>

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 <b>订单号</b>: <code>%s</code>
👤 <b>用户账号</b>: %s
📦 <b>套餐名称</b>: %s
💵 <b>支付金额</b>: ¥%.2f
💳 <b>支付方式</b>: %s
⏰ <b>支付时间</b>: %s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ 订单已自动处理，订阅已激活`, orderNo, username, packageName, amount, paymentMethod, paymentTime)

	case "user_registered":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		registerTime := getString(data, "register_time", "N/A")
		return fmt.Sprintf(`👤 <b>新用户注册</b>

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👤 <b>用户账号</b>: %s
📧 <b>注册邮箱</b>: %s
⏰ <b>注册时间</b>: %s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ 新用户已自动创建默认订阅`, username, email, registerTime)

	case "password_reset":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		resetTime := getString(data, "reset_time", "N/A")
		return fmt.Sprintf(`🔐 <b>用户重置密码</b>

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👤 <b>用户账号</b>: %s
📧 <b>用户邮箱</b>: %s
⏰ <b>重置时间</b>: %s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━

⚠️ 如非用户本人操作，请及时检查账户安全`, username, email, resetTime)

	case "subscription_sent":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		sendTime := getString(data, "send_time", "N/A")
		return fmt.Sprintf(`📧 <b>用户发送订阅</b>

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👤 <b>用户账号</b>: %s
📧 <b>用户邮箱</b>: %s
⏰ <b>发送时间</b>: %s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━`, username, email, sendTime)

	case "subscription_reset":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		resetTime := getString(data, "reset_time", "N/A")
		return fmt.Sprintf(`🔄 <b>用户重置订阅</b>

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👤 <b>用户账号</b>: %s
📧 <b>用户邮箱</b>: %s
⏰ <b>重置时间</b>: %s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ 订阅地址已重置，旧地址已失效`, username, email, resetTime)

	case "subscription_expired":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		expireTime := getString(data, "expire_time", "N/A")
		return fmt.Sprintf(`⏰ <b>订阅已过期</b>

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👤 <b>用户账号</b>: %s
📧 <b>用户邮箱</b>: %s
⏰ <b>过期时间</b>: %s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━

⚠️ 建议引导用户续费以恢复服务`, username, email, expireTime)

	case "user_created":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		createdBy := getString(data, "created_by", "N/A")
		createTime := getString(data, "create_time", "N/A")
		return fmt.Sprintf(`📋 <b>管理员创建用户</b>

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👤 <b>用户账号</b>: <code>%s</code>
📧 <b>注册邮箱</b>: %s
👨‍💼 <b>创建者</b>: %s
⏰ <b>创建时间</b>: %s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ 用户账户已成功创建`, username, email, createdBy, createTime)

	case "subscription_created":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		packageName := getString(data, "package_name", "未知套餐")
		createTime := getString(data, "create_time", "N/A")
		return fmt.Sprintf(`📦 <b>订阅创建</b>

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👤 <b>用户账号</b>: %s
📧 <b>用户邮箱</b>: %s
📦 <b>套餐名称</b>: %s
⏰ <b>创建时间</b>: %s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ 订阅已创建并激活，用户可立即使用服务`, username, email, packageName, createTime)

	default:
		title := getString(data, "title", "系统通知")
		message := getString(data, "message", "")
		return fmt.Sprintf("<b>%s</b>\n\n%s", title, message)
	}
}

// FormatBarkMessage 格式化 Bark 消息
func FormatBarkMessage(notificationType string, data map[string]interface{}) (string, string) {
	var title, body string

	switch notificationType {
	case "order_paid":
		orderNo := getString(data, "order_no", "N/A")
		username := getString(data, "username", "N/A")
		amount := getFloat(data, "amount", 0)
		packageName := getString(data, "package_name", "未知套餐")
		paymentMethod := getString(data, "payment_method", "未知")
		paymentTime := getString(data, "payment_time", "N/A")
		title = "💰 新订单支付成功"
		body = fmt.Sprintf(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 订单号: %s
👤 用户账号: %s
📦 套餐名称: %s
💵 支付金额: ¥%.2f
💳 支付方式: %s
⏰ 支付时间: %s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ 订单已自动处理，订阅已激活`, orderNo, username, packageName, amount, paymentMethod, paymentTime)

	case "user_registered":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		registerTime := getString(data, "register_time", "N/A")
		title = "👤 新用户注册"
		body = fmt.Sprintf(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👤 用户账号: %s
📧 注册邮箱: %s
⏰ 注册时间: %s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ 新用户已自动创建默认订阅`, username, email, registerTime)

	case "password_reset":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		resetTime := getString(data, "reset_time", "N/A")
		title = "🔐 用户重置密码"
		body = fmt.Sprintf(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👤 用户账号: %s
📧 用户邮箱: %s
⏰ 重置时间: %s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━

⚠️ 如非用户本人操作，请及时检查账户安全`, username, email, resetTime)

	case "subscription_sent":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		sendTime := getString(data, "send_time", "N/A")
		title = "📧 用户发送订阅"
		body = fmt.Sprintf(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👤 用户账号: %s
📧 用户邮箱: %s
⏰ 发送时间: %s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━`, username, email, sendTime)

	case "subscription_reset":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		resetTime := getString(data, "reset_time", "N/A")
		title = "🔄 用户重置订阅"
		body = fmt.Sprintf(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👤 用户账号: %s
📧 用户邮箱: %s
⏰ 重置时间: %s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ 订阅地址已重置，旧地址已失效`, username, email, resetTime)

	case "subscription_expired":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		expireTime := getString(data, "expire_time", "N/A")
		title = "⏰ 订阅已过期"
		body = fmt.Sprintf(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👤 用户账号: %s
📧 用户邮箱: %s
⏰ 过期时间: %s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━

⚠️ 建议引导用户续费以恢复服务`, username, email, expireTime)

	case "user_created":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		createdBy := getString(data, "created_by", "N/A")
		createTime := getString(data, "create_time", "N/A")
		title = "📋 管理员创建用户"
		body = fmt.Sprintf(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👤 用户账号: %s
📧 注册邮箱: %s
👨‍💼 创建者: %s
⏰ 创建时间: %s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ 用户账户已成功创建`, username, email, createdBy, createTime)

	case "subscription_created":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		packageName := getString(data, "package_name", "未知套餐")
		createTime := getString(data, "create_time", "N/A")
		title = "📦 订阅创建"
		body = fmt.Sprintf(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👤 用户账号: %s
📧 用户邮箱: %s
📦 套餐名称: %s
⏰ 创建时间: %s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ 订阅已创建并激活，用户可立即使用服务`, username, email, packageName, createTime)

	default:
		title = getString(data, "title", "系统通知")
		body = getString(data, "message", "")
	}

	return title, body
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

	// 格式化消息
	telegramMsg := FormatTelegramMessage(notificationType, data)
	barkTitle, barkBody := FormatBarkMessage(notificationType, data)

	// 发送 Telegram 通知
	if configMap["admin_telegram_notification"] == "true" {
		botToken := configMap["admin_telegram_bot_token"]
		chatID := configMap["admin_telegram_chat_id"]
		if botToken != "" && chatID != "" {
			go func() {
				_, _ = sendTelegramMessage(botToken, chatID, telegramMsg)
			}()
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
				_, _ = sendBarkMessage(serverURL, deviceKey, barkTitle, barkBody)
			}()
		}
	}

	// 发送邮件通知（使用邮件模板）
	if configMap["admin_email_notification"] == "true" {
		adminEmail := configMap["admin_notification_email"]
		if adminEmail != "" {
			emailService := email.NewEmailService()
			templateBuilder := email.NewEmailTemplateBuilder()
			subject := getNotificationSubject(notificationType)
			content := templateBuilder.GetAdminNotificationTemplate(notificationType, barkTitle, barkBody, data)
			_ = emailService.QueueEmail(adminEmail, subject, content, "admin_notification")
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

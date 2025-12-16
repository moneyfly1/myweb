package notification

import (
	"fmt"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/email"
)

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

📋 订单号: <code>%s</code>
👤 用户: %s
📦 套餐: %s
💵 金额: ¥%.2f
💳 支付方式: %s
⏰ 支付时间: %s`, orderNo, username, packageName, amount, paymentMethod, paymentTime)

	case "user_registered":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		registerTime := getString(data, "register_time", "N/A")
		return fmt.Sprintf(`👤 <b>新用户注册</b>

用户名: %s
邮箱: %s
注册时间: %s`, username, email, registerTime)

	case "password_reset":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		resetTime := getString(data, "reset_time", "N/A")
		return fmt.Sprintf(`🔐 <b>用户重置密码</b>

用户名: %s
邮箱: %s
重置时间: %s`, username, email, resetTime)

	case "subscription_sent":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		sendTime := getString(data, "send_time", "N/A")
		return fmt.Sprintf(`📧 <b>用户发送订阅</b>

用户名: %s
邮箱: %s
发送时间: %s`, username, email, sendTime)

	case "subscription_reset":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		resetTime := getString(data, "reset_time", "N/A")
		return fmt.Sprintf(`🔄 <b>用户重置订阅</b>

用户名: %s
邮箱: %s
重置时间: %s`, username, email, resetTime)

	case "subscription_expired":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		expireTime := getString(data, "expire_time", "N/A")
		return fmt.Sprintf(`⏰ <b>订阅已过期</b>

用户名: %s
邮箱: %s
过期时间: %s`, username, email, expireTime)

	case "user_created":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		createdBy := getString(data, "created_by", "N/A")
		createTime := getString(data, "create_time", "N/A")
		return fmt.Sprintf(`👤 <b>管理员创建用户</b>

用户账号: <code>%s</code>
注册邮箱: %s
创建者: 👤 %s
创建时间: ⏰ %s`, username, email, createdBy, createTime)

	case "subscription_created":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		packageName := getString(data, "package_name", "未知套餐")
		createTime := getString(data, "create_time", "N/A")
		return fmt.Sprintf(`📦 <b>订阅创建</b>

用户名: %s
邮箱: %s
套餐: %s
创建时间: %s`, username, email, packageName, createTime)

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
		title = fmt.Sprintf("💰 新订单支付成功 - %s", orderNo)
		body = fmt.Sprintf(`订单号: %s
用户: %s
套餐: %s
金额: ¥%.2f
支付方式: %s
支付时间: %s`, orderNo, username, packageName, amount, paymentMethod, paymentTime)

	case "user_registered":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		registerTime := getString(data, "register_time", "N/A")
		title = fmt.Sprintf("👤 新用户注册 - %s", username)
		body = fmt.Sprintf(`用户名: %s
邮箱: %s
注册时间: %s`, username, email, registerTime)

	case "password_reset":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		resetTime := getString(data, "reset_time", "N/A")
		title = fmt.Sprintf("🔐 用户重置密码 - %s", username)
		body = fmt.Sprintf(`用户名: %s
邮箱: %s
重置时间: %s`, username, email, resetTime)

	case "subscription_sent":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		sendTime := getString(data, "send_time", "N/A")
		title = fmt.Sprintf("📧 用户发送订阅 - %s", username)
		body = fmt.Sprintf(`用户名: %s
邮箱: %s
发送时间: %s`, username, email, sendTime)

	case "subscription_reset":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		resetTime := getString(data, "reset_time", "N/A")
		title = fmt.Sprintf("🔄 用户重置订阅 - %s", username)
		body = fmt.Sprintf(`用户名: %s
邮箱: %s
重置时间: %s`, username, email, resetTime)

	case "subscription_expired":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		expireTime := getString(data, "expire_time", "N/A")
		title = fmt.Sprintf("⏰ 订阅已过期 - %s", username)
		body = fmt.Sprintf(`用户名: %s
邮箱: %s
过期时间: %s`, username, email, expireTime)

	case "user_created":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		createdBy := getString(data, "created_by", "N/A")
		createTime := getString(data, "create_time", "N/A")
		title = "📋 管理员创建用户"
		body = fmt.Sprintf(`📋 **账户信息**

**用户账号**
`+"`%s`"+`

**注册邮箱**
%s

**创建者**
👤 %s

**创建时间**
⏰ %s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ 用户账户已成功创建`, username, email, createdBy, createTime)

	case "subscription_created":
		username := getString(data, "username", "N/A")
		email := getString(data, "email", "N/A")
		packageName := getString(data, "package_name", "未知套餐")
		createTime := getString(data, "create_time", "N/A")
		title = fmt.Sprintf("📦 订阅创建 - %s", username)
		body = fmt.Sprintf(`用户名: %s
邮箱: %s
套餐: %s
创建时间: %s`, username, email, packageName, createTime)

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

	// 检查是否启用
	if configMap["admin_notification_enabled"] != "true" {
		return nil
	}

	// 格式化消息
	telegramMsg := FormatTelegramMessage(notificationType, data)
	barkTitle, barkBody := FormatBarkMessage(notificationType, data)

	// 发送 Telegram 通知
	if configMap["admin_telegram_notification"] == "true" {
		botToken := configMap["admin_telegram_bot_token"]
		chatID := configMap["admin_telegram_chat_id"]
		if botToken != "" && chatID != "" {
			// 这里需要调用 Telegram API，暂时跳过
			_ = botToken
			_ = chatID
			_ = telegramMsg
		}
	}

	// 发送 Bark 通知
	if configMap["admin_bark_notification"] == "true" {
		serverURL := configMap["admin_bark_server_url"]
		deviceKey := configMap["admin_bark_device_key"]
		if serverURL != "" && deviceKey != "" {
			// 这里需要调用 Bark API，暂时跳过
			_ = serverURL
			_ = deviceKey
			_ = barkTitle
			_ = barkBody
		}
	}

	// 发送邮件通知（使用邮件模板）
	if configMap["admin_email_notification"] == "true" {
		adminEmail := configMap["admin_notification_email"]
		if adminEmail != "" {
			emailService := email.NewEmailService()
			templateBuilder := email.NewEmailTemplateBuilder()
			subject := fmt.Sprintf("系统通知 - %s", notificationType)
			content := templateBuilder.GetBroadcastNotificationTemplate(barkTitle, barkBody)
			_ = emailService.QueueEmail(adminEmail, subject, content, "admin_notification")
		}
	}

	return nil
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

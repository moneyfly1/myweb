package notification

import (
	"fmt"
)

// MessageTemplateBuilder 消息模板构建器
type MessageTemplateBuilder struct {
}

// NewMessageTemplateBuilder 创建消息模板构建器
func NewMessageTemplateBuilder() *MessageTemplateBuilder {
	return &MessageTemplateBuilder{}
}

// BuildTelegramMessage 构建 Telegram 消息
func (b *MessageTemplateBuilder) BuildTelegramMessage(notificationType string, data map[string]interface{}) string {
	switch notificationType {
	case "order_paid":
		return b.buildOrderPaidTelegram(data)
	case "user_registered":
		return b.buildUserRegisteredTelegram(data)
	case "password_reset":
		return b.buildPasswordResetTelegram(data)
	case "subscription_sent":
		return b.buildSubscriptionSentTelegram(data)
	case "subscription_reset":
		return b.buildSubscriptionResetTelegram(data)
	case "subscription_expired":
		return b.buildSubscriptionExpiredTelegram(data)
	case "user_created":
		return b.buildUserCreatedTelegram(data)
	case "subscription_created":
		return b.buildSubscriptionCreatedTelegram(data)
	default:
		return b.buildDefaultTelegram(data)
	}
}

// BuildBarkMessage 构建 Bark 消息
func (b *MessageTemplateBuilder) BuildBarkMessage(notificationType string, data map[string]interface{}) (string, string) {
	switch notificationType {
	case "order_paid":
		return b.buildOrderPaidBark(data)
	case "user_registered":
		return b.buildUserRegisteredBark(data)
	case "password_reset":
		return b.buildPasswordResetBark(data)
	case "subscription_sent":
		return b.buildSubscriptionSentBark(data)
	case "subscription_reset":
		return b.buildSubscriptionResetBark(data)
	case "subscription_expired":
		return b.buildSubscriptionExpiredBark(data)
	case "user_created":
		return b.buildUserCreatedBark(data)
	case "subscription_created":
		return b.buildSubscriptionCreatedBark(data)
	default:
		return b.buildDefaultBark(data)
	}
}

// ==================== Telegram 消息模板 ====================

func (b *MessageTemplateBuilder) buildOrderPaidTelegram(data map[string]interface{}) string {
	orderNo := getString(data, "order_no", "N/A")
	username := getString(data, "username", "N/A")
	amount := getFloat(data, "amount", 0)
	packageName := getString(data, "package_name", "未知套餐")
	paymentMethod := getString(data, "payment_method", "未知")
	paymentTime := getString(data, "payment_time", "N/A")

	return fmt.Sprintf(`🎉 <b>订单支付成功</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  📋 <b>订单信息</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

🆔 <b>订单号</b>: <code>%s</code>
👤 <b>用户账号</b>: <code>%s</code>
📦 <b>套餐名称</b>: %s
💰 <b>支付金额</b>: <b>¥%.2f</b>
💳 <b>支付方式</b>: %s
🕐 <b>支付时间</b>: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ✅ <b>订单已自动处理</b>
┃  📦 <b>订阅已激活</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, orderNo, username, packageName, amount, paymentMethod, paymentTime)
}

func (b *MessageTemplateBuilder) buildUserRegisteredTelegram(data map[string]interface{}) string {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	registerTime := getString(data, "register_time", "N/A")

	return fmt.Sprintf(`👋 <b>新用户注册</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  👤 <b>用户信息</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 <b>用户账号</b>: <code>%s</code>
📧 <b>注册邮箱</b>: %s
🕐 <b>注册时间</b>: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ✅ <b>新用户已自动创建默认订阅</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, registerTime)
}

func (b *MessageTemplateBuilder) buildPasswordResetTelegram(data map[string]interface{}) string {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	resetTime := getString(data, "reset_time", "N/A")

	return fmt.Sprintf(`🔐 <b>密码重置通知</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ⚠️ <b>安全提醒</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 <b>用户账号</b>: <code>%s</code>
📧 <b>用户邮箱</b>: %s
🕐 <b>重置时间</b>: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ⚠️ <b>如非用户本人操作</b>
┃  <b>请及时检查账户安全</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, resetTime)
}

func (b *MessageTemplateBuilder) buildSubscriptionSentTelegram(data map[string]interface{}) string {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	sendTime := getString(data, "send_time", "N/A")

	return fmt.Sprintf(`📧 <b>订阅邮件发送</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  📋 <b>发送信息</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 <b>用户账号</b>: <code>%s</code>
📧 <b>用户邮箱</b>: %s
🕐 <b>发送时间</b>: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ✅ <b>订阅信息已发送至用户邮箱</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, sendTime)
}

func (b *MessageTemplateBuilder) buildSubscriptionResetTelegram(data map[string]interface{}) string {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	resetTime := getString(data, "reset_time", "N/A")

	return fmt.Sprintf(`🔄 <b>订阅重置</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  📋 <b>重置信息</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 <b>用户账号</b>: <code>%s</code>
📧 <b>用户邮箱</b>: %s
🕐 <b>重置时间</b>: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ✅ <b>订阅地址已重置</b>
┃  ⚠️ <b>旧地址已失效</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, resetTime)
}

func (b *MessageTemplateBuilder) buildSubscriptionExpiredTelegram(data map[string]interface{}) string {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	expireTime := getString(data, "expire_time", "N/A")

	return fmt.Sprintf(`⏰ <b>订阅已过期</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ⚠️ <b>过期提醒</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 <b>用户账号</b>: <code>%s</code>
📧 <b>用户邮箱</b>: %s
🕐 <b>过期时间</b>: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  💡 <b>建议引导用户续费</b>
┃  <b>以恢复服务</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, expireTime)
}

func (b *MessageTemplateBuilder) buildUserCreatedTelegram(data map[string]interface{}) string {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	createdBy := getString(data, "created_by", "N/A")
	createTime := getString(data, "create_time", "N/A")

	return fmt.Sprintf(`📋 <b>管理员创建用户</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  👤 <b>用户信息</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 <b>用户账号</b>: <code>%s</code>
📧 <b>注册邮箱</b>: %s
👨‍💼 <b>创建者</b>: <code>%s</code>
🕐 <b>创建时间</b>: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ✅ <b>用户账户已成功创建</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, createdBy, createTime)
}

func (b *MessageTemplateBuilder) buildSubscriptionCreatedTelegram(data map[string]interface{}) string {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	packageName := getString(data, "package_name", "未知套餐")
	createTime := getString(data, "create_time", "N/A")

	return fmt.Sprintf(`📦 <b>订阅创建</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  📋 <b>订阅信息</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 <b>用户账号</b>: <code>%s</code>
📧 <b>用户邮箱</b>: %s
📦 <b>套餐名称</b>: %s
🕐 <b>创建时间</b>: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ✅ <b>订阅已创建并激活</b>
┃  🚀 <b>用户可立即使用服务</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, packageName, createTime)
}

func (b *MessageTemplateBuilder) buildDefaultTelegram(data map[string]interface{}) string {
	title := getString(data, "title", "系统通知")
	message := getString(data, "message", "")

	return fmt.Sprintf(`📢 <b>%s</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  %s
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, title, message)
}

// ==================== Bark 消息模板 ====================

func (b *MessageTemplateBuilder) buildOrderPaidBark(data map[string]interface{}) (string, string) {
	orderNo := getString(data, "order_no", "N/A")
	username := getString(data, "username", "N/A")
	amount := getFloat(data, "amount", 0)
	packageName := getString(data, "package_name", "未知套餐")
	paymentMethod := getString(data, "payment_method", "未知")
	paymentTime := getString(data, "payment_time", "N/A")

	title := "🎉 订单支付成功"
	body := fmt.Sprintf(`┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  📋 订单信息
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

🆔 订单号: %s
👤 用户账号: %s
📦 套餐名称: %s
💰 支付金额: ¥%.2f
💳 支付方式: %s
🕐 支付时间: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ✅ 订单已自动处理
┃  📦 订阅已激活
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, orderNo, username, packageName, amount, paymentMethod, paymentTime)

	return title, body
}

func (b *MessageTemplateBuilder) buildUserRegisteredBark(data map[string]interface{}) (string, string) {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	registerTime := getString(data, "register_time", "N/A")

	title := "👋 新用户注册"
	body := fmt.Sprintf(`┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  👤 用户信息
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 用户账号: %s
📧 注册邮箱: %s
🕐 注册时间: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ✅ 新用户已自动创建默认订阅
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, registerTime)

	return title, body
}

func (b *MessageTemplateBuilder) buildPasswordResetBark(data map[string]interface{}) (string, string) {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	resetTime := getString(data, "reset_time", "N/A")

	title := "🔐 密码重置通知"
	body := fmt.Sprintf(`┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ⚠️ 安全提醒
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 用户账号: %s
📧 用户邮箱: %s
🕐 重置时间: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ⚠️ 如非用户本人操作
┃  请及时检查账户安全
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, resetTime)

	return title, body
}

func (b *MessageTemplateBuilder) buildSubscriptionSentBark(data map[string]interface{}) (string, string) {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	sendTime := getString(data, "send_time", "N/A")

	title := "📧 订阅邮件发送"
	body := fmt.Sprintf(`┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  📋 发送信息
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 用户账号: %s
📧 用户邮箱: %s
🕐 发送时间: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ✅ 订阅信息已发送至用户邮箱
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, sendTime)

	return title, body
}

func (b *MessageTemplateBuilder) buildSubscriptionResetBark(data map[string]interface{}) (string, string) {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	resetTime := getString(data, "reset_time", "N/A")

	title := "🔄 订阅重置"
	body := fmt.Sprintf(`┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  📋 重置信息
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 用户账号: %s
📧 用户邮箱: %s
🕐 重置时间: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ✅ 订阅地址已重置
┃  ⚠️ 旧地址已失效
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, resetTime)

	return title, body
}

func (b *MessageTemplateBuilder) buildSubscriptionExpiredBark(data map[string]interface{}) (string, string) {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	expireTime := getString(data, "expire_time", "N/A")

	title := "⏰ 订阅已过期"
	body := fmt.Sprintf(`┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ⚠️ 过期提醒
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 用户账号: %s
📧 用户邮箱: %s
🕐 过期时间: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  💡 建议引导用户续费
┃  以恢复服务
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, expireTime)

	return title, body
}

func (b *MessageTemplateBuilder) buildUserCreatedBark(data map[string]interface{}) (string, string) {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	createdBy := getString(data, "created_by", "N/A")
	createTime := getString(data, "create_time", "N/A")

	title := "📋 管理员创建用户"
	body := fmt.Sprintf(`┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  👤 用户信息
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 用户账号: %s
📧 注册邮箱: %s
👨‍💼 创建者: %s
🕐 创建时间: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ✅ 用户账户已成功创建
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, createdBy, createTime)

	return title, body
}

func (b *MessageTemplateBuilder) buildSubscriptionCreatedBark(data map[string]interface{}) (string, string) {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	packageName := getString(data, "package_name", "未知套餐")
	createTime := getString(data, "create_time", "N/A")

	title := "📦 订阅创建"
	body := fmt.Sprintf(`┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  📋 订阅信息
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 用户账号: %s
📧 用户邮箱: %s
📦 套餐名称: %s
🕐 创建时间: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ✅ 订阅已创建并激活
┃  🚀 用户可立即使用服务
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, packageName, createTime)

	return title, body
}

func (b *MessageTemplateBuilder) buildDefaultBark(data map[string]interface{}) (string, string) {
	title := getString(data, "title", "系统通知")
	message := getString(data, "message", "")

	body := fmt.Sprintf(`┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  %s
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, message)

	return title, body
}

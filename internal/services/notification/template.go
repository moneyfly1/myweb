package notification

import (
	"fmt"
)

type MessageTemplateBuilder struct {
}

func NewMessageTemplateBuilder() *MessageTemplateBuilder {
	return &MessageTemplateBuilder{}
}

func (b *MessageTemplateBuilder) BuildTelegramMessage(notificationType string, data map[string]interface{}) string {
	switch notificationType {
	case "order_paid":
		return b.buildOrderPaidTelegram(data)
	case "recharge_paid":
		return b.buildRechargePaidTelegram(data)
	case "user_registered":
		return b.buildUserRegisteredTelegram(data)
	case "password_reset":
		return b.buildPasswordResetTelegram(data)
	case "password_changed":
		return b.buildPasswordChangedTelegram(data)
	case "abnormal_login":
		return b.buildAbnormalLoginTelegram(data)
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
	case "ticket_created":
		return b.buildTicketCreatedTelegram(data)
	case "ticket_replied":
		return b.buildTicketRepliedTelegram(data)
	case "test":
		return b.buildTestTelegram(data)
	default:
		return b.buildDefaultTelegram(data)
	}
}

func (b *MessageTemplateBuilder) BuildBarkMessage(notificationType string, data map[string]interface{}) (string, string) {
	switch notificationType {
	case "order_paid":
		return b.buildOrderPaidBark(data)
	case "recharge_paid":
		return b.buildRechargePaidBark(data)
	case "user_registered":
		return b.buildUserRegisteredBark(data)
	case "password_reset":
		return b.buildPasswordResetBark(data)
	case "password_changed":
		return b.buildPasswordChangedBark(data)
	case "abnormal_login":
		return b.buildAbnormalLoginBark(data)
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
	case "ticket_created":
		return b.buildTicketCreatedBark(data)
	case "ticket_replied":
		return b.buildTicketRepliedBark(data)
	case "test":
		return b.buildTestBark(data)
	default:
		return b.buildDefaultBark(data)
	}
}

func (b *MessageTemplateBuilder) buildOrderPaidTelegram(data map[string]interface{}) string {
	orderNo := getString(data, "order_no", "N/A")
	username := getString(data, "username", "N/A")
	amount := getFloat(data, "amount", 0)
	packageName := getString(data, "package_name", "未知套餐")
	paymentMethod := getString(data, "payment_method", "未知")
	paymentTime := getString(data, "payment_time", "N/A")

	return fmt.Sprintf(`🎉 <b>订单支付成功</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  📋 <b>订单详情</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

🆔 <b>订单号</b>: <code>%s</code>
👤 <b>用户账号</b>: <code>%s</code>
📦 <b>套餐名称</b>: <b>%s</b>
💰 <b>支付金额</b>: <b>¥%.2f</b>
💳 <b>支付方式</b>: %s
🕐 <b>支付时间</b>: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ✅ <b>订单已自动处理</b>
┃  📦 <b>订阅已激活</b>
┃  🚀 <b>用户可立即使用服务</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, orderNo, username, packageName, amount, paymentMethod, paymentTime)
}

func (b *MessageTemplateBuilder) buildRechargePaidTelegram(data map[string]interface{}) string {
	orderNo := getString(data, "order_no", "N/A")
	username := getString(data, "username", "N/A")
	amount := getFloat(data, "amount", 0)
	balance := getFloat(data, "balance", 0)
	paymentMethod := getString(data, "payment_method", "未知")
	paymentTime := getString(data, "payment_time", "N/A")

	return fmt.Sprintf(`💳 <b>用户充值成功</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  💰 <b>充值详情</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

🆔 <b>充值单号</b>: <code>%s</code>
👤 <b>用户账号</b>: <code>%s</code>
💰 <b>充值金额</b>: <b>¥%.2f</b>
💼 <b>当前余额</b>: <b>¥%.2f</b>
💳 <b>支付方式</b>: %s
🕐 <b>到账时间</b>: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ✅ <b>充值金额已到账</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, orderNo, username, amount, balance, paymentMethod, paymentTime)
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
┃  💡 <b>可引导用户购买套餐激活服务</b>
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
┃  💡 <b>建议联系用户确认</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, resetTime)
}

func (b *MessageTemplateBuilder) buildPasswordChangedTelegram(data map[string]interface{}) string {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	changeTime := getString(data, "change_time", "N/A")

	return fmt.Sprintf(`🔐 <b>用户修改密码</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ⚠️ <b>安全提醒</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 <b>用户账号</b>: <code>%s</code>
📧 <b>用户邮箱</b>: %s
🕐 <b>修改时间</b>: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  💡 <b>如非本人操作，请及时处理</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, changeTime)
}

func (b *MessageTemplateBuilder) buildAbnormalLoginTelegram(data map[string]interface{}) string {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	ipAddress := getString(data, "ip_address", "N/A")
	location := getString(data, "location", "未知")
	loginTime := getString(data, "login_time", "N/A")

	return fmt.Sprintf(`⚠️ <b>异常登录告警</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  🔐 <b>登录信息</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 <b>用户账号</b>: <code>%s</code>
📧 <b>用户邮箱</b>: %s
🌐 <b>登录 IP</b>: <code>%s</code>
📍 <b>登录位置</b>: %s
🕐 <b>登录时间</b>: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ⚠️ <b>请确认是否为用户本人操作</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, ipAddress, location, loginTime)
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
┃  📡 <b>包含订阅地址和配置信息</b>
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
┃  📧 <b>重置通知已发送至用户邮箱</b>
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
┃  📧 <b>过期提醒已发送至用户邮箱</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, expireTime)
}

func (b *MessageTemplateBuilder) buildUserCreatedTelegram(data map[string]interface{}) string {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	password := getString(data, "password", "N/A")
	createdBy := getString(data, "created_by", "N/A")
	createTime := getString(data, "create_time", "N/A")
	expireTime := getString(data, "expire_time", "未设置")
	deviceLimit := getInt(data, "device_limit", 0)

	return fmt.Sprintf(`📋 <b>管理员创建用户</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  👤 <b>账户信息</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 <b>用户账号</b>: <code>%s</code>
📧 <b>注册邮箱</b>: %s
🔑 <b>登录密码</b>: <code>%s</code>
👨‍💼 <b>创建者</b>: <code>%s</code>
🕐 <b>创建时间</b>: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  📡 <b>服务信息</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

⏰ <b>有效期</b>: %s
📱 <b>设备限制</b>: <b>%d 台设备</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ✅ <b>用户账户已成功创建</b>
┃  📧 <b>账户信息已发送至用户邮箱</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, password, createdBy, createTime, expireTime, deviceLimit)
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
📦 <b>套餐名称</b>: <b>%s</b>
🕐 <b>创建时间</b>: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ✅ <b>订阅已创建并激活</b>
┃  🚀 <b>用户可立即使用服务</b>
┃  📧 <b>订阅信息已发送至用户邮箱</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, packageName, createTime)
}

func (b *MessageTemplateBuilder) buildTestTelegram(data map[string]interface{}) string {
	testTime := getString(data, "test_time", "")
	if testTime == "" {
		testTime = "刚刚"
	}

	return fmt.Sprintf(`🧪 <b>通知功能测试</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ✅ <b>测试成功</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

📱 <b>通知类型</b>: Telegram
🕐 <b>测试时间</b>: %s
📡 <b>状态</b>: <b>连接正常</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  💡 <b>提示信息</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

✅ <b>Telegram 通知功能正常工作</b>
📧 <b>您将收到所有管理员通知</b>
🔔 <b>包括订单、用户、订阅等事件</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  🎉 <b>配置完成，可以开始使用</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, testTime)
}

func (b *MessageTemplateBuilder) buildDefaultTelegram(data map[string]interface{}) string {
	title := getString(data, "title", "系统通知")
	message := getString(data, "message", "")
	if message == "" {
		message = "这是一条系统通知消息"
	}

	return fmt.Sprintf(`📢 <b>%s</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  <b>通知内容</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

%s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  💡 <b>系统自动发送</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, title, message)
}

func (b *MessageTemplateBuilder) buildOrderPaidBark(data map[string]interface{}) (string, string) {
	orderNo := getString(data, "order_no", "N/A")
	username := getString(data, "username", "N/A")
	amount := getFloat(data, "amount", 0)
	packageName := getString(data, "package_name", "未知套餐")
	paymentMethod := getString(data, "payment_method", "未知")
	paymentTime := getString(data, "payment_time", "N/A")

	title := "🎉 订单支付成功"
	body := fmt.Sprintf(`┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  📋 订单详情
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
┃  🚀 用户可立即使用服务
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, orderNo, username, packageName, amount, paymentMethod, paymentTime)

	return title, body
}

func (b *MessageTemplateBuilder) buildRechargePaidBark(data map[string]interface{}) (string, string) {
	orderNo := getString(data, "order_no", "N/A")
	username := getString(data, "username", "N/A")
	amount := getFloat(data, "amount", 0)
	balance := getFloat(data, "balance", 0)
	paymentMethod := getString(data, "payment_method", "未知")
	paymentTime := getString(data, "payment_time", "N/A")

	title := "💳 用户充值成功"
	body := fmt.Sprintf(`┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  💰 充值详情
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

🆔 充值单号: %s
👤 用户账号: %s
💰 充值金额: ¥%.2f
💼 当前余额: ¥%.2f
💳 支付方式: %s
🕐 到账时间: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ✅ 充值金额已到账
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, orderNo, username, amount, balance, paymentMethod, paymentTime)

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
┃  💡 可引导用户购买套餐激活服务
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
┃  💡 建议联系用户确认
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, resetTime)

	return title, body
}

func (b *MessageTemplateBuilder) buildPasswordChangedBark(data map[string]interface{}) (string, string) {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	changeTime := getString(data, "change_time", "N/A")

	title := "🔐 用户修改密码"
	body := fmt.Sprintf(`┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ⚠️ 安全提醒
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 用户账号: %s
📧 用户邮箱: %s
🕐 修改时间: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  💡 如非本人操作，请及时处理
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, changeTime)

	return title, body
}

func (b *MessageTemplateBuilder) buildAbnormalLoginBark(data map[string]interface{}) (string, string) {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	ipAddress := getString(data, "ip_address", "N/A")
	location := getString(data, "location", "未知")
	loginTime := getString(data, "login_time", "N/A")

	title := "⚠️ 异常登录告警"
	body := fmt.Sprintf(`┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  🔐 登录信息
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 用户账号: %s
📧 用户邮箱: %s
🌐 登录 IP: %s
📍 登录位置: %s
🕐 登录时间: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ⚠️ 请确认是否为用户本人操作
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, ipAddress, location, loginTime)

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
┃  📡 包含订阅地址和配置信息
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
┃  📧 重置通知已发送至用户邮箱
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
┃  📧 过期提醒已发送至用户邮箱
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, expireTime)

	return title, body
}

func (b *MessageTemplateBuilder) buildUserCreatedBark(data map[string]interface{}) (string, string) {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	password := getString(data, "password", "N/A")
	createdBy := getString(data, "created_by", "N/A")
	createTime := getString(data, "create_time", "N/A")
	expireTime := getString(data, "expire_time", "未设置")
	deviceLimit := getInt(data, "device_limit", 0)

	title := "📋 管理员创建用户"
	body := fmt.Sprintf(`┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  👤 账户信息
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 用户账号: %s
📧 注册邮箱: %s
🔑 登录密码: %s
👨‍💼 创建者: %s
🕐 创建时间: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  📡 服务信息
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

⏰ 有效期: %s
📱 设备限制: %d 台设备

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ✅ 用户账户已成功创建
┃  📧 账户信息已发送至用户邮箱
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, password, createdBy, createTime, expireTime, deviceLimit)

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
┃  📧 订阅信息已发送至用户邮箱
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, packageName, createTime)

	return title, body
}

func (b *MessageTemplateBuilder) buildTestBark(data map[string]interface{}) (string, string) {
	testTime := getString(data, "test_time", "")
	if testTime == "" {
		testTime = "刚刚"
	}

	title := "🧪 通知功能测试"
	body := fmt.Sprintf(`┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  ✅ 测试成功
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

📱 通知类型: Bark
🕐 测试时间: %s
📡 状态: 连接正常

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  💡 提示信息
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

✅ Bark 通知功能正常工作
📧 您将收到所有管理员通知
🔔 包括订单、用户、订阅等事件

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  🎉 配置完成，可以开始使用
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, testTime)

	return title, body
}

func (b *MessageTemplateBuilder) buildDefaultBark(data map[string]interface{}) (string, string) {
	title := getString(data, "title", "系统通知")
	message := getString(data, "message", "")
	if message == "" {
		message = "这是一条系统通知消息"
	}

	body := fmt.Sprintf(`┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  通知内容
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

%s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  💡 系统自动发送
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, message)

	return title, body
}

func (b *MessageTemplateBuilder) buildTicketCreatedTelegram(data map[string]interface{}) string {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	ticketNo := getString(data, "ticket_no", "N/A")
	title := getString(data, "title", "N/A")
	ticketType := getString(data, "type", "N/A")
	priority := getString(data, "priority", "N/A")
	createTime := getString(data, "create_time", "N/A")

	return fmt.Sprintf(`🎫 <b>用户提交工单</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  👤 <b>用户信息</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 <b>用户账号</b>: <code>%s</code>
📧 <b>用户邮箱</b>: %s
🕐 <b>提交时间</b>: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  📋 <b>工单信息</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

🎫 <b>工单编号</b>: <code>%s</code>
📝 <b>工单标题</b>: %s
🏷️ <b>工单类型</b>: %s
⚡ <b>优先级</b>: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  💡 <b>请及时处理用户工单</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, createTime, ticketNo, title, ticketType, priority)
}

func (b *MessageTemplateBuilder) buildTicketRepliedTelegram(data map[string]interface{}) string {
	username := getString(data, "username", "N/A")
	email := getString(data, "email", "N/A")
	ticketNo := getString(data, "ticket_no", "N/A")
	title := getString(data, "title", "N/A")
	replyTime := getString(data, "reply_time", "N/A")

	return fmt.Sprintf(`💬 <b>工单新回复</b>

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  👤 <b>用户信息</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 <b>用户账号</b>: <code>%s</code>
📧 <b>用户邮箱</b>: %s
🕐 <b>回复时间</b>: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  📋 <b>工单信息</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

🎫 <b>工单编号</b>: <code>%s</code>
📝 <b>工单标题</b>: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  💡 <b>请及时查看用户回复</b>
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, email, replyTime, ticketNo, title)
}

func (b *MessageTemplateBuilder) buildTicketCreatedBark(data map[string]interface{}) (string, string) {
	username := getString(data, "username", "N/A")
	ticketNo := getString(data, "ticket_no", "N/A")
	title := getString(data, "title", "N/A")
	ticketType := getString(data, "type", "N/A")
	priority := getString(data, "priority", "N/A")
	createTime := getString(data, "create_time", "N/A")

	barkTitle := "🎫 用户提交工单"
	body := fmt.Sprintf(`┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  📋 工单信息
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 用户账号: %s
🎫 工单编号: %s
📝 工单标题: %s
🏷️ 工单类型: %s
⚡ 优先级: %s
🕐 提交时间: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  💡 请及时处理用户工单
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, ticketNo, title, ticketType, priority, createTime)

	return barkTitle, body
}

func (b *MessageTemplateBuilder) buildTicketRepliedBark(data map[string]interface{}) (string, string) {
	username := getString(data, "username", "N/A")
	ticketNo := getString(data, "ticket_no", "N/A")
	title := getString(data, "title", "N/A")
	replyTime := getString(data, "reply_time", "N/A")

	barkTitle := "💬 工单新回复"
	body := fmt.Sprintf(`┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  📋 工单信息
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

👤 用户账号: %s
🎫 工单编号: %s
📝 工单标题: %s
🕐 回复时间: %s

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  💡 请及时查看用户回复
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`, username, ticketNo, title, replyTime)

	return barkTitle, body
}

package handlers

import "testing"

// 敏感配置键判定测试。
//
// 背景：/admin/configs 此前完全不脱敏（返回 smtp_password、alipay_private_key 等明文），
// 而 /admin/settings 已脱敏 —— 同一后台两个接口一脱一裸。
// 修复时还必须避免"过度脱敏"：把 min_password_length、admin_notify_password_reset
// 这类非密钥配置也打码会让后台无法正常使用（显示 ****** 且无法编辑）。
func TestIsSensitiveConfigKey(t *testing.T) {
	sensitive := []string{
		// 显式清单
		"admin_telegram_bot_token", "admin_bark_device_key", "repo_sync_token",
		"backup_gitee_token", "backup_github_token", "smtp_password",
		"alipay_private_key", "alipay_public_key", "wechat_api_key",
		"paypal_secret", "stripe_secret_key", "merchant_private_key",
		"secret_key", "jwt_secret_key", "api_token", "access_token",
		"refresh_token", "webhook_secret", "telegram_bot_token", "bark_device_key",
		// 线上实际存在、此前漏掉的
		"email_password", "password", "token",
		"aliyun_refresh_token", "backup_aliyundrive_token",
		// 后缀规则命中
		"some_new_service_password", "another_api_key", "pan123_token",
		"third_party_private_key", "custom_device_key",
	}
	for _, key := range sensitive {
		if !isSensitiveConfigKey(key) {
			t.Errorf("%s 应判定为敏感键（需脱敏）", key)
		}
	}

	notSensitive := []string{
		// 长度/开关/列表类配置：绝不能打码，否则后台无法使用
		"min_password_length", "password_changed_email_notifications",
		"password_reset_notifications", "password_changed_notifications",
		"admin_notify_password_reset", "admin_notify_password_changed",
		"admin_notify_password_reset_email", "admin_notify_password_changed_telegram",
		"filter_keywords", "site_name", "domain_name", "announcement_content",
		"registration_enabled", "invite_inviter_reward",
	}
	for _, key := range notSensitive {
		if isSensitiveConfigKey(key) {
			t.Errorf("%s 属于普通配置，不应被脱敏（会导致后台显示 ****** 无法编辑）", key)
		}
	}
}

func TestIsMaskedSecretPlaceholder(t *testing.T) {
	if !isMaskedSecretPlaceholder(maskedSecretValue) {
		t.Errorf("%q 应识别为脱敏占位符", maskedSecretValue)
	}
	if !isMaskedSecretPlaceholder("  " + maskedSecretValue + "  ") {
		t.Error("带空格的占位符也应识别")
	}
	for _, v := range []string{"", "real-secret", "*****", "*******", "abcdef"} {
		if isMaskedSecretPlaceholder(v) {
			t.Errorf("%q 不应被识别为脱敏占位符", v)
		}
	}
}

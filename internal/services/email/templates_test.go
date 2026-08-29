package email

import (
	"strings"
	"testing"
)

// TestEmailTemplatesEscapeUserInput 验证所有接收用户可控字段的模板构建器都会转义 HTML，
// 防止恶意用户名/邮箱注入邮件 HTML（存储型 XSS）。
func TestEmailTemplatesEscapeUserInput(t *testing.T) {
	// GetBaseURL 在无 DB 时会读 BASE_URL，避免触碰未初始化的全局配置
	t.Setenv("BASE_URL", "https://example.com")
	b := NewEmailTemplateBuilder()
	evil := `<img src=x onerror=alert(1)>`
	evilWithQuote := `<script>"alert(1)"</script>`

	cases := []struct {
		name   string
		render func() string
	}{
		{"VerificationCode", func() string { return b.GetVerificationCodeTemplate(evil, "123456") }},
		{"PasswordReset", func() string { return b.GetPasswordResetTemplate(evil, "https://example.com/reset?token=abc") }},
		{"PasswordResetVerificationCode", func() string { return b.GetPasswordResetVerificationCodeTemplate(evil, "123456") }},
		{"Subscription", func() string { return b.GetSubscriptionTemplate(evil, "u", "c", "2026-01-01", 3, 3, 1) }},
		{"OrderConfirmation", func() string {
			return b.GetOrderConfirmationTemplate(evil, "ORD1", "套餐", 10, "支付宝", "2026-01-01")
		}},
		{"PaymentSuccess", func() string {
			return b.GetPaymentSuccessTemplate(evil, "ORD1", "套餐", 10, "支付宝", "2026-01-01")
		}},
		{"DeviceUpgrade", func() string {
			return b.GetDeviceUpgradePaymentSuccessTemplate(evil, "ORD1", 10, "支付宝", "2026-01-01", 3, 5, 2, 30, "2026-01-01", "2026-02-01")
		}},
		{"AbnormalLogin", func() string {
			return b.GetAbnormalLoginAlertTemplate(evil, "2026-01-01", "1.2.3.4", evil, true, false)
		}},
		{"Welcome", func() string {
			return b.GetWelcomeTemplate(evil, evilWithQuote, "https://example.com/login", true, evil)
		}},
		{"UserCreated", func() string { return b.GetUserCreatedTemplate(evil, evilWithQuote, evil, "2026-01-01", 3) }},
		{"PasswordChanged", func() string { return b.GetPasswordChangedTemplate(evil, "2026-01-01", "https://example.com") }},
		{"SubscriptionReset", func() string { return b.GetSubscriptionResetTemplate(evil, "u", "c", "2026-01-01", "2026-01-01", evil) }},
		{"AccountDeletion", func() string { return b.GetAccountDeletionTemplate(evil, "2026-01-01", evil, "30天") }},
		{"ExpirationReminder", func() string { return b.GetExpirationReminderTemplate(evil, evil, "2026-01-01", 3, 3, 1, false) }},
		{"RenewalConfirmation", func() string {
			return b.GetRenewalConfirmationTemplate(evil, evil, "2026-01-01", "2026-02-01", "2026-01-01", 10)
		}},
		{"Marketing", func() string { return b.GetMarketingEmailTemplate(evil, evil) }},
		{"Broadcast", func() string { return b.GetBroadcastNotificationTemplate(evil, evil) }},
		{"AdminNotificationOrder", func() string {
			return b.GetAdminNotificationTemplate("order_created", "订单", "", map[string]interface{}{
				"username": evil, "email": evil, "order_no": evil, "package_name": evil, "payment_method": evil,
			})
		}},
		{"AdminNotificationRegistered", func() string {
			return b.GetAdminNotificationTemplate("user_registered", "注册", "", map[string]interface{}{
				"username": evil, "email": evil, "register_time": evil,
			})
		}},
	}

	for _, tc := range cases {
		out := tc.render()
		if strings.Contains(out, "<img") || strings.Contains(out, "<script") {
			t.Errorf("%s: 输出包含未转义的原始 HTML 注入: %q", tc.name, snippet(out))
		}
		if !strings.Contains(out, "&lt;img") && !strings.Contains(out, "&lt;script") {
			t.Errorf("%s: 未找到转义后的标记（&lt;img 或 &lt;script），可能输入未被插入", tc.name)
		}
	}
}

func snippet(s string) string {
	idx := strings.Index(s, "<img")
	if idx < 0 {
		idx = strings.Index(s, "<script")
	}
	if idx < 0 {
		return s[:min(len(s), 80)]
	}
	start := idx - 40
	if start < 0 {
		start = 0
	}
	end := idx + 60
	if end > len(s) {
		end = len(s)
	}
	return s[start:end]
}

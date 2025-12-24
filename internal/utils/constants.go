package utils

// 默认订阅配置常量
const (
	// DefaultDeviceLimit 默认设备数量限制（0表示不允许链接）
	DefaultDeviceLimit = 0
	// DefaultDurationMonths 默认订阅时长（月，0表示当天到期）
	DefaultDurationMonths = 0
)

// 订阅状态常量
const (
	SubscriptionStatusActive   = "active"
	SubscriptionStatusInactive = "inactive"
	SubscriptionStatusExpired  = "expired"
)

// 订单状态常量
const (
	OrderStatusPending  = "pending"
	OrderStatusPaid     = "paid"
	OrderStatusFailed   = "failed"
	OrderStatusCanceled = "canceled"
)

// 验证码用途常量
const (
	VerificationPurposeRegister      = "register"
	VerificationPurposeResetPassword = "reset_password"
	VerificationPurposeChangeEmail   = "change_email"
)

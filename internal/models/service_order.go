package models

import (
	"database/sql"
	"time"
)

// ServiceOrder 验证码服务订单（一次性短信验证码，来自 HeroSMS 兼容平台）。
// 与套餐订单（orders）完全独立：套餐订单是「买订阅」，这里是「买一次短信验证码」。
type ServiceOrder struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	OrderNo          string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"order_no"`
	UserID           uint           `gorm:"index;index:idx_service_orders_user_status,priority:1;not null" json:"user_id"`
	ServiceCode      string         `gorm:"type:varchar(50);not null" json:"service_code"`
	ServiceName      string         `gorm:"type:varchar(100)" json:"service_name"`
	CountryID        int            `gorm:"not null" json:"country_id"`
	CountryName      string         `gorm:"type:varchar(100)" json:"country_name"`
	OfficialPriceUSD float64        `gorm:"type:decimal(10,4);not null" json:"official_price_usd"`
	Multiplier       float64        `gorm:"type:decimal(6,2);not null" json:"multiplier"`
	ExchangeRate     float64        `gorm:"type:decimal(10,4);not null" json:"exchange_rate"`
	PriceCNY         float64        `gorm:"type:decimal(10,2);not null" json:"price_cny"`
	PaymentMethod    sql.NullString `gorm:"type:varchar(100)" json:"payment_method,omitempty"`
	PaymentStatus    string         `gorm:"type:varchar(20);default:pending;index" json:"payment_status"` // pending/paid/refunded
	Status           string         `gorm:"type:varchar(20);default:pending;index;index:idx_service_orders_user_status,priority:2" json:"status"` // pending/active/completed/cancelled/failed
	ActivationID     sql.NullString `gorm:"type:varchar(64)" json:"activation_id,omitempty"`
	PhoneNumber      sql.NullString `gorm:"type:varchar(50)" json:"phone_number,omitempty"`
	OTP              sql.NullString `gorm:"type:varchar(50)" json:"otp,omitempty"`
	Refunded         bool           `gorm:"default:false" json:"refunded"`
	RefundReason     sql.NullString `gorm:"type:varchar(255)" json:"refund_reason,omitempty"`
	ErrorMsg         sql.NullString `gorm:"type:text" json:"error_msg,omitempty"`
	APIResponse      sql.NullString `gorm:"type:text" json:"api_response,omitempty"`
	PaidAt           sql.NullTime   `gorm:"index" json:"paid_at,omitempty"`
	CompletedAt      sql.NullTime   `json:"completed_at,omitempty"`
	CreatedAt        time.Time      `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (ServiceOrder) TableName() string {
	return "service_orders"
}

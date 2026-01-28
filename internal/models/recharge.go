package models

import (
	"database/sql"
	"time"
)

type RechargeRecord struct {
	ID                   uint           `gorm:"primaryKey" json:"id"`
	UserID               uint           `gorm:"index;not null" json:"user_id"`
	OrderNo              string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"order_no"`
	Amount               float64        `gorm:"type:decimal(10,2);not null" json:"amount"`
	Status               string         `gorm:"type:varchar(20);default:pending" json:"status"`
	PaymentMethod        sql.NullString `gorm:"type:varchar(50)" json:"payment_method,omitempty"`
	PaymentTransactionID sql.NullString `gorm:"type:varchar(100)" json:"payment_transaction_id,omitempty"`
	PaymentQRCode        sql.NullString `gorm:"type:text" json:"payment_qr_code,omitempty"`
	PaymentURL           sql.NullString `gorm:"type:text" json:"payment_url,omitempty"`
	IPAddress            sql.NullString `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent            sql.NullString `gorm:"type:text" json:"user_agent,omitempty"`
	PaidAt               sql.NullTime   `json:"paid_at,omitempty"`
	CreatedAt            time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (RechargeRecord) TableName() string {
	return "recharge_records"
}

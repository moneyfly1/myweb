package models

import (
	"database/sql"
	"time"
)

type PaymentTransaction struct {
	ID                    uint           `gorm:"primaryKey" json:"id"`
	OrderID               uint           `gorm:"index;not null" json:"order_id"`
	UserID                uint           `gorm:"index;not null" json:"user_id"`
	PaymentMethodID       uint           `gorm:"index;not null" json:"payment_method_id"`
	Amount                int            `gorm:"not null" json:"amount"` // 金额（分）
	Currency              string         `gorm:"type:varchar(10);default:CNY" json:"currency"`
	TransactionID         sql.NullString `gorm:"type:varchar(100);uniqueIndex" json:"transaction_id,omitempty"`
	ExternalTransactionID sql.NullString `gorm:"type:varchar(100)" json:"external_transaction_id,omitempty"`
	Status                string         `gorm:"type:varchar(20);default:pending" json:"status"`
	PaymentData           sql.NullString `gorm:"type:json" json:"payment_data,omitempty"`
	CallbackData          sql.NullString `gorm:"type:json" json:"callback_data,omitempty"`
	CreatedAt             time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	User  User  `gorm:"foreignKey:UserID" json:"-"`
	Order Order `gorm:"foreignKey:OrderID" json:"-"`
}

func (PaymentTransaction) TableName() string {
	return "payment_transactions"
}

type PaymentCallback struct {
	ID                   uint           `gorm:"primaryKey" json:"id"`
	PaymentTransactionID uint           `gorm:"not null" json:"payment_transaction_id"`
	CallbackType         string         `gorm:"type:varchar(50);not null" json:"callback_type"`
	CallbackData         string         `gorm:"type:json;not null" json:"callback_data"`
	RawRequest           sql.NullString `gorm:"type:text" json:"raw_request,omitempty"`
	Processed            bool           `gorm:"default:false" json:"processed"`
	ProcessingResult     sql.NullString `gorm:"type:varchar(50)" json:"processing_result,omitempty"`
	ErrorMessage         sql.NullString `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt            time.Time      `gorm:"autoCreateTime" json:"created_at"`
}

func (PaymentCallback) TableName() string {
	return "payment_callbacks"
}

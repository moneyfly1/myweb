package models

import (
	"database/sql"
	"time"
)

// Order 订单模型
type Order struct {
	ID                   uint            `gorm:"primaryKey" json:"id"`
	OrderNo              string          `gorm:"type:varchar(50);uniqueIndex;not null" json:"order_no"`
	UserID               uint            `gorm:"index;not null" json:"user_id"`
	PackageID            uint            `gorm:"index;not null" json:"package_id"`
	Amount               float64         `gorm:"type:decimal(10,2);not null" json:"amount"`
	Status               string          `gorm:"type:varchar(20);default:pending;index" json:"status"`
	PaymentMethodID      sql.NullInt64   `json:"payment_method_id,omitempty"`
	PaymentMethodName    sql.NullString  `gorm:"type:varchar(100)" json:"payment_method_name,omitempty"`
	PaymentTime          sql.NullTime    `json:"payment_time,omitempty"`
	PaymentTransactionID sql.NullString  `gorm:"type:varchar(100)" json:"payment_transaction_id,omitempty"`
	ExpireTime           sql.NullTime    `json:"expire_time,omitempty"`
	CouponID             sql.NullInt64   `gorm:"index" json:"coupon_id,omitempty"`
	DiscountAmount       sql.NullFloat64 `gorm:"type:decimal(10,2);default:0" json:"discount_amount,omitempty"`
	FinalAmount          sql.NullFloat64 `gorm:"type:decimal(10,2)" json:"final_amount,omitempty"`
	ExtraData            sql.NullString  `gorm:"type:text" json:"extra_data,omitempty"`
	CreatedAt            time.Time       `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt            time.Time       `gorm:"autoUpdateTime" json:"updated_at"`

	// 关系
	User    User    `gorm:"foreignKey:UserID" json:"-"`
	Package Package `gorm:"foreignKey:PackageID" json:"-"`
	Coupon  Coupon  `gorm:"foreignKey:CouponID" json:"-"`
}

// TableName 指定表名
func (Order) TableName() string {
	return "orders"
}

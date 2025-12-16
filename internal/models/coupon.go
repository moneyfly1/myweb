package models

import (
	"database/sql"
	"time"
)

// CouponType 优惠券类型
type CouponType string

const (
	CouponTypeDiscount CouponType = "discount"  // 折扣
	CouponTypeFixed    CouponType = "fixed"     // 固定金额
	CouponTypeFreeDays CouponType = "free_days" // 赠送天数
)

// CouponStatus 优惠券状态
type CouponStatus string

const (
	CouponStatusActive   CouponStatus = "active"   // 有效
	CouponStatusInactive CouponStatus = "inactive" // 无效
	CouponStatusExpired  CouponStatus = "expired"  // 已过期
)

// Coupon 优惠券模型
type Coupon struct {
	ID                 uint            `gorm:"primaryKey" json:"id"`
	Code               string          `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"`
	Name               string          `gorm:"type:varchar(100);not null" json:"name"`
	Description        string          `gorm:"type:text" json:"description"`
	Type               string          `gorm:"type:varchar(20);not null" json:"type"`
	DiscountValue      float64         `gorm:"type:decimal(10,2);not null" json:"discount_value"`
	MinAmount          sql.NullFloat64 `gorm:"type:decimal(10,2);default:0" json:"min_amount,omitempty"`
	MaxDiscount        sql.NullFloat64 `gorm:"type:decimal(10,2)" json:"max_discount,omitempty"`
	ValidFrom          time.Time       `gorm:"not null" json:"valid_from"`
	ValidUntil         time.Time       `gorm:"not null" json:"valid_until"`
	TotalQuantity      sql.NullInt64   `json:"total_quantity,omitempty"`
	UsedQuantity       int             `gorm:"default:0" json:"used_quantity"`
	MaxUsesPerUser     int             `gorm:"default:1" json:"max_uses_per_user"`
	Status             string          `gorm:"type:varchar(20);default:active" json:"status"`
	ApplicablePackages string          `gorm:"type:text" json:"applicable_packages"`
	CreatedBy          sql.NullInt64   `gorm:"index" json:"created_by,omitempty"`
	CreatedAt          time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time       `gorm:"autoUpdateTime" json:"updated_at"`

	// 关系
	Usages []CouponUsage `gorm:"foreignKey:CouponID" json:"-"`
}

// TableName 指定表名
func (Coupon) TableName() string {
	return "coupons"
}

// CouponUsage 优惠券使用记录
type CouponUsage struct {
	ID             uint          `gorm:"primaryKey" json:"id"`
	CouponID       uint          `gorm:"index;not null" json:"coupon_id"`
	UserID         uint          `gorm:"index;not null" json:"user_id"`
	OrderID        sql.NullInt64 `gorm:"index" json:"order_id,omitempty"`
	DiscountAmount float64       `gorm:"type:decimal(10,2);not null" json:"discount_amount"`
	UsedAt         time.Time     `gorm:"autoCreateTime" json:"used_at"`

	// 关系
	Coupon Coupon `gorm:"foreignKey:CouponID" json:"coupon,omitempty"`
	User   User   `gorm:"foreignKey:UserID" json:"-"`
	Order  Order  `gorm:"foreignKey:OrderID" json:"-"`
}

// TableName 指定表名
func (CouponUsage) TableName() string {
	return "coupon_usages"
}

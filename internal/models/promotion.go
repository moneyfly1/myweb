package models

import (
	"database/sql"
	"time"
)

type Promotion struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Name          string         `gorm:"not null" json:"name"`
	Type          string         `gorm:"not null;index" json:"type"` // flash_sale, new_user, recall, member_day
	DiscountType  string         `gorm:"not null" json:"discount_type"` // percentage, fixed, free_days
	DiscountValue float64        `gorm:"not null" json:"discount_value"`
	MinAmount     float64        `json:"min_amount"`
	MaxDiscount   float64        `json:"max_discount"`
	PackageIDs    sql.NullString `json:"package_ids"`
	StartTime     time.Time      `gorm:"not null" json:"start_time"`
	EndTime       time.Time      `gorm:"not null" json:"end_time"`
	IsActive      bool           `json:"is_active"`
	Description   sql.NullString `json:"description"`
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Promotion) TableName() string {
	return "promotions"
}

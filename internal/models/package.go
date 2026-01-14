package models

import (
	"database/sql"
	"time"
)

// Package 套餐模型
type Package struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Name          string         `gorm:"type:varchar(100);not null" json:"name"`
	Description   sql.NullString `gorm:"type:text" json:"description,omitempty"`
	Price         float64        `gorm:"type:decimal(10,2);not null" json:"price"`
	DurationDays  int            `gorm:"not null" json:"duration_days"`
	DeviceLimit   int            `gorm:"default:3" json:"device_limit"`
	SortOrder     int            `gorm:"default:1" json:"sort_order"`
	IsActive      bool           `gorm:"default:true" json:"is_active"`
	IsRecommended bool           `gorm:"default:false" json:"is_recommended"`
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	// 关系
	Orders        []Order        `gorm:"foreignKey:PackageID" json:"-"`
	Subscriptions []Subscription `gorm:"foreignKey:PackageID" json:"-"`
}

// TableName 指定表名
func (Package) TableName() string {
	return "packages"
}

package models

import (
	"time"
)

// Subscription 订阅模型
type Subscription struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          uint      `gorm:"index;not null" json:"user_id"`
	PackageID       *int64    `gorm:"index" json:"package_id,omitempty"`
	SubscriptionURL string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"subscription_url"`
	DeviceLimit     int       `gorm:"default:3" json:"device_limit"`
	CurrentDevices  int       `gorm:"default:0" json:"current_devices"`
	UniversalCount  int       `gorm:"default:0" json:"universal_count"` // 通用订阅次数
	ClashCount      int       `gorm:"default:0" json:"clash_count"`      // 猫咪订阅次数
	IsActive        bool      `gorm:"default:true;index" json:"is_active"`
	Status          string    `gorm:"type:varchar(20);default:active;index" json:"status"`
	ExpireTime      time.Time `gorm:"not null;index" json:"expire_time"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// 关系
	User    User                `gorm:"foreignKey:UserID" json:"-"`
	Package Package             `gorm:"foreignKey:PackageID" json:"-"`
	Devices []Device            `gorm:"foreignKey:SubscriptionID" json:"-"`
	Resets  []SubscriptionReset `gorm:"foreignKey:SubscriptionID" json:"-"`
}

// TableName 指定表名
func (Subscription) TableName() string {
	return "subscriptions"
}

// SubscriptionReset 订阅重置记录
type SubscriptionReset struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	UserID             uint      `gorm:"index;not null" json:"user_id"`
	SubscriptionID     uint      `gorm:"index;not null" json:"subscription_id"`
	ResetType          string    `gorm:"type:varchar(50);not null" json:"reset_type"`
	Reason             string    `gorm:"type:text" json:"reason"`
	OldSubscriptionURL *string   `gorm:"type:varchar(255)" json:"old_subscription_url,omitempty"`
	NewSubscriptionURL *string   `gorm:"type:varchar(255)" json:"new_subscription_url,omitempty"`
	DeviceCountBefore  int       `gorm:"default:0" json:"device_count_before"`
	DeviceCountAfter   int       `gorm:"default:0" json:"device_count_after"`
	ResetBy            *string   `gorm:"type:varchar(50)" json:"reset_by,omitempty"`
	CreatedAt          time.Time `gorm:"autoCreateTime" json:"created_at"`

	// 关系
	User         User         `gorm:"foreignKey:UserID" json:"-"`
	Subscription Subscription `gorm:"foreignKey:SubscriptionID" json:"-"`
}

// TableName 指定表名
func (SubscriptionReset) TableName() string {
	return "subscription_resets"
}

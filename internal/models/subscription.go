package models

import (
	"database/sql"
	"time"
)

// Subscription 订阅模型
type Subscription struct {
	ID              uint          `gorm:"primaryKey" json:"id"`
	UserID          uint          `gorm:"index;not null" json:"user_id"`
	PackageID       sql.NullInt64 `gorm:"index" json:"package_id,omitempty"`
	SubscriptionURL string        `gorm:"type:varchar(100);uniqueIndex;not null" json:"subscription_url"`
	DeviceLimit     int           `gorm:"default:3" json:"device_limit"`
	CurrentDevices  int           `gorm:"default:0" json:"current_devices"`
	IsActive        bool          `gorm:"default:true" json:"is_active"`
	Status          string        `gorm:"type:varchar(20);default:active" json:"status"`
	ExpireTime      time.Time     `gorm:"not null" json:"expire_time"`
	CreatedAt       time.Time     `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time     `gorm:"autoUpdateTime" json:"updated_at"`

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

// Device 设备模型
type Device struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	UserID            sql.NullInt64  `gorm:"index" json:"user_id,omitempty"`
	SubscriptionID    uint           `gorm:"index;not null" json:"subscription_id"`
	DeviceFingerprint string         `gorm:"type:varchar(255);not null" json:"device_fingerprint"`
	DeviceHash        sql.NullString `gorm:"type:varchar(255)" json:"device_hash,omitempty"`
	DeviceUA          sql.NullString `gorm:"type:varchar(255)" json:"device_ua,omitempty"`
	DeviceName        sql.NullString `gorm:"type:varchar(100)" json:"device_name,omitempty"`
	DeviceType        sql.NullString `gorm:"type:varchar(50)" json:"device_type,omitempty"`
	IPAddress         sql.NullString `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent         sql.NullString `gorm:"type:text" json:"user_agent,omitempty"`
	SoftwareName      sql.NullString `gorm:"type:varchar(100)" json:"software_name,omitempty"`
	SoftwareVersion   sql.NullString `gorm:"type:varchar(50)" json:"software_version,omitempty"`
	OSName            sql.NullString `gorm:"type:varchar(50)" json:"os_name,omitempty"`
	OSVersion         sql.NullString `gorm:"type:varchar(50)" json:"os_version,omitempty"`
	DeviceModel       sql.NullString `gorm:"type:varchar(100)" json:"device_model,omitempty"`
	DeviceBrand       sql.NullString `gorm:"type:varchar(50)" json:"device_brand,omitempty"`
	IsActive          bool           `gorm:"default:true" json:"is_active"`
	IsAllowed         bool           `gorm:"default:true" json:"is_allowed"`
	FirstSeen         sql.NullTime   `json:"first_seen,omitempty"`
	LastAccess        time.Time      `gorm:"autoCreateTime" json:"last_access"`
	LastSeen          sql.NullTime   `json:"last_seen,omitempty"`
	AccessCount       int            `gorm:"default:0" json:"access_count"`
	CreatedAt         time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	// 关系
	User         User         `gorm:"foreignKey:UserID" json:"-"`
	Subscription Subscription `gorm:"foreignKey:SubscriptionID" json:"-"`
}

// TableName 指定表名
func (Device) TableName() string {
	return "devices"
}

// SubscriptionReset 订阅重置记录
type SubscriptionReset struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	UserID             uint           `gorm:"index;not null" json:"user_id"`
	SubscriptionID     uint           `gorm:"index;not null" json:"subscription_id"`
	ResetType          string         `gorm:"type:varchar(50);not null" json:"reset_type"`
	Reason             string         `gorm:"type:text" json:"reason"`
	OldSubscriptionURL sql.NullString `gorm:"type:varchar(255)" json:"old_subscription_url,omitempty"`
	NewSubscriptionURL sql.NullString `gorm:"type:varchar(255)" json:"new_subscription_url,omitempty"`
	DeviceCountBefore  int            `gorm:"default:0" json:"device_count_before"`
	DeviceCountAfter   int            `gorm:"default:0" json:"device_count_after"`
	ResetBy            sql.NullString `gorm:"type:varchar(50)" json:"reset_by,omitempty"`
	CreatedAt          time.Time      `gorm:"autoCreateTime" json:"created_at"`

	// 关系
	User         User         `gorm:"foreignKey:UserID" json:"-"`
	Subscription Subscription `gorm:"foreignKey:SubscriptionID" json:"-"`
}

// TableName 指定表名
func (SubscriptionReset) TableName() string {
	return "subscription_resets"
}

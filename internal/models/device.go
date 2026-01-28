package models

import (
	"time"
)

type Device struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	UserID            *int64     `gorm:"index" json:"user_id,omitempty"`
	SubscriptionID    uint       `gorm:"index;not null" json:"subscription_id"`
	DeviceFingerprint string     `gorm:"type:varchar(255);not null" json:"device_fingerprint"`
	DeviceHash        *string    `gorm:"type:varchar(255)" json:"device_hash,omitempty"`
	DeviceUA          *string    `gorm:"type:varchar(255)" json:"device_ua,omitempty"`
	DeviceName        *string    `gorm:"type:varchar(100)" json:"device_name,omitempty"`
	DeviceType        *string    `gorm:"type:varchar(50)" json:"device_type,omitempty"`
	IPAddress         *string    `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent         *string    `gorm:"type:text" json:"user_agent,omitempty"`
	SoftwareName      *string    `gorm:"type:varchar(100)" json:"software_name,omitempty"`
	SoftwareVersion   *string    `gorm:"type:varchar(50)" json:"software_version,omitempty"`
	OSName            *string    `gorm:"type:varchar(50)" json:"os_name,omitempty"`
	OSVersion         *string    `gorm:"type:varchar(50)" json:"os_version,omitempty"`
	DeviceModel       *string    `gorm:"type:varchar(100)" json:"device_model,omitempty"`
	DeviceBrand       *string    `gorm:"type:varchar(50)" json:"device_brand,omitempty"`
	SubscriptionType  *string    `gorm:"type:varchar(20);index" json:"subscription_type,omitempty"` // 订阅类型: clash, v2ray, ssr
	IsActive          bool       `gorm:"default:true;index" json:"is_active"`
	IsAllowed         bool       `gorm:"default:true" json:"is_allowed"`
	FirstSeen         *time.Time `json:"first_seen,omitempty"`
	LastAccess        time.Time  `gorm:"autoCreateTime" json:"last_access"`
	LastSeen          *time.Time `json:"last_seen,omitempty"`
	AccessCount       int        `gorm:"default:0" json:"access_count"`
	CreatedAt         time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	User         User         `gorm:"foreignKey:UserID" json:"-"`
	Subscription Subscription `gorm:"foreignKey:SubscriptionID" json:"-"`
}

func (Device) TableName() string {
	return "devices"
}

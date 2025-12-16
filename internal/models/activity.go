package models

import (
	"database/sql"
	"time"
)

// UserActivity 用户活动模型
type UserActivity struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	UserID           uint           `gorm:"index;not null" json:"user_id"`
	ActivityType     string         `gorm:"type:varchar(50);not null" json:"activity_type"`
	Description      sql.NullString `gorm:"type:text" json:"description,omitempty"`
	IPAddress        sql.NullString `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent        sql.NullString `gorm:"type:text" json:"user_agent,omitempty"`
	Location         sql.NullString `gorm:"type:varchar(100)" json:"location,omitempty"`
	ActivityMetadata sql.NullString `gorm:"type:json" json:"activity_metadata,omitempty"`
	CreatedAt        time.Time      `gorm:"autoCreateTime" json:"created_at"`

	// 关系
	User User `gorm:"foreignKey:UserID" json:"-"`
}

// TableName 指定表名
func (UserActivity) TableName() string {
	return "user_activities"
}

// LoginHistory 登录历史记录
type LoginHistory struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	UserID            uint           `gorm:"index;not null" json:"user_id"`
	LoginTime         time.Time      `gorm:"autoCreateTime" json:"login_time"`
	LogoutTime        sql.NullTime   `json:"logout_time,omitempty"`
	IPAddress         sql.NullString `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent         sql.NullString `gorm:"type:text" json:"user_agent,omitempty"`
	Location          sql.NullString `gorm:"type:varchar(100)" json:"location,omitempty"`
	DeviceFingerprint sql.NullString `gorm:"type:varchar(255)" json:"device_fingerprint,omitempty"`
	LoginStatus       string         `gorm:"type:varchar(20);default:success" json:"login_status"`
	FailureReason     sql.NullString `gorm:"type:text" json:"failure_reason,omitempty"`
	SessionDuration   sql.NullInt64  `json:"session_duration,omitempty"`

	// 关系
	User User `gorm:"foreignKey:UserID" json:"-"`
}

// TableName 指定表名
func (LoginHistory) TableName() string {
	return "login_history"
}

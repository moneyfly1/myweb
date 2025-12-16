package models

import (
	"database/sql"
	"time"
)

// LoginAttempt 登录尝试记录
type LoginAttempt struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"type:varchar(100);index;not null" json:"username"`
	IPAddress sql.NullString `gorm:"type:varchar(45);index" json:"ip_address,omitempty"`
	Success   bool           `gorm:"default:false" json:"success"`
	UserAgent sql.NullString `gorm:"type:varchar(500)" json:"user_agent,omitempty"`
	CreatedAt time.Time      `gorm:"autoCreateTime;not null" json:"created_at"`
}

// TableName 指定表名
func (LoginAttempt) TableName() string {
	return "login_attempts"
}

// VerificationAttempt 验证码尝试记录
type VerificationAttempt struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Email     string         `gorm:"type:varchar(100);index;not null" json:"email"`
	IPAddress sql.NullString `gorm:"type:varchar(45);index" json:"ip_address,omitempty"`
	Success   bool           `gorm:"default:false" json:"success"`
	Purpose   string         `gorm:"type:varchar(50);default:register" json:"purpose"`
	CreatedAt time.Time      `gorm:"autoCreateTime;not null" json:"created_at"`
}

// TableName 指定表名
func (VerificationAttempt) TableName() string {
	return "verification_attempts"
}

// VerificationCode 验证码模型
type VerificationCode struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Email     string    `gorm:"type:varchar(100);index;not null" json:"email"`
	Code      string    `gorm:"type:varchar(6);not null" json:"code"`
	CreatedAt time.Time `gorm:"autoCreateTime;not null" json:"created_at"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	Used      int       `gorm:"default:0" json:"used"`
	Purpose   string    `gorm:"type:varchar(50);default:register" json:"purpose"`
}

// TableName 指定表名
func (VerificationCode) TableName() string {
	return "verification_codes"
}

// IsExpired 检查验证码是否过期
func (v *VerificationCode) IsExpired() bool {
	return time.Now().After(v.ExpiresAt)
}

// IsUsed 检查验证码是否已使用
func (v *VerificationCode) IsUsed() bool {
	return v.Used == 1
}

// MarkAsUsed 标记验证码为已使用
func (v *VerificationCode) MarkAsUsed() {
	v.Used = 1
}

package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// TokenBlacklist Token黑名单（用于Token撤销）
type TokenBlacklist struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TokenHash string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"token_hash"` // Token的哈希值
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"` // Token的过期时间
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName 指定表名
func (TokenBlacklist) TableName() string {
	return "token_blacklist"
}

// IsExpired 检查Token是否已过期
func (t *TokenBlacklist) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// AddToBlacklist 将Token添加到黑名单
func AddToBlacklist(db *gorm.DB, tokenHash string, userID uint, expiresAt time.Time) error {
	blacklist := TokenBlacklist{
		TokenHash: tokenHash,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}
	return db.Create(&blacklist).Error
}

// IsTokenBlacklisted 检查Token是否在黑名单中
func IsTokenBlacklisted(db *gorm.DB, tokenHash string) bool {
	var blacklist TokenBlacklist
	err := db.Where("token_hash = ? AND expires_at > ?", tokenHash, time.Now()).First(&blacklist).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false
		}
		return false
	}
	return true
}

// CleanExpiredTokens 清理过期的Token
func CleanExpiredTokens(db *gorm.DB) error {
	return db.Where("expires_at < ?", time.Now()).Delete(&TokenBlacklist{}).Error
}

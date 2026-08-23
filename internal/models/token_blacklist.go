package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type TokenBlacklist struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TokenHash string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"token_hash"` // Token的哈希值
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"` // Token的过期时间
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (TokenBlacklist) TableName() string {
	return "token_blacklist"
}

func (t *TokenBlacklist) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

func AddToBlacklist(db *gorm.DB, tokenHash string, userID uint, expiresAt time.Time) error {
	blacklist := TokenBlacklist{
		TokenHash: tokenHash,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}
	return db.Create(&blacklist).Error
}

// IsTokenBlacklisted 判断 token 是否已被加入黑名单。
// 注意：数据库错误时按“已拉黑”（fail-closed）处理，宁可拒绝请求也不放行可能已登出的令牌。
func IsTokenBlacklisted(db *gorm.DB, tokenHash string) bool {
	var blacklist TokenBlacklist
	err := db.Where("token_hash = ? AND expires_at > ?", tokenHash, time.Now()).First(&blacklist).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false
		}
		// 数据库错误：fail-closed
		return true
	}
	return true
}

func CleanExpiredTokens(db *gorm.DB) error {
	return db.Where("expires_at < ?", time.Now()).Delete(&TokenBlacklist{}).Error
}

package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TokenBlacklist struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TokenHash string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"token_hash"` // Token的哈希值
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"` // Token的过期时间
	// Reason 拉黑原因：rotate（刷新轮换产生的新旧更替）/ logout（显式登出）/ revoke。
	// 刷新接口据此区分「轮换重放」与「真的失效」：轮换产物在宽限期内可以重放换新令牌，
	// 登出/吊销的令牌一律拒绝（否则登出在窗口内形同虚设）。
	Reason    string    `gorm:"type:varchar(20);index;default:rotate" json:"reason"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (TokenBlacklist) TableName() string {
	return "token_blacklist"
}

func (t *TokenBlacklist) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// 拉黑原因常量
const (
	BlacklistReasonRotate = "rotate" // 刷新令牌轮换（旧令牌被新令牌替换）
	BlacklistReasonLogout = "logout" // 用户显式登出
	BlacklistReasonRevoke = "revoke" // 管理员/风控吊销
)

func AddToBlacklist(db *gorm.DB, tokenHash string, userID uint, expiresAt time.Time) error {
	return AddToBlacklistWithReason(db, tokenHash, userID, expiresAt, BlacklistReasonRotate)
}

// AddToBlacklistWithReason 拉黑并记录原因（按 token_hash 幂等 upsert）。
//
// ⚠️ 必须 upsert 而不是 Create：同一个 refresh_token 会先因「轮换」被拉黑
// （reason=rotate，允许宽限重放），之后用户又显式登出时再次拉黑
// （reason=logout，必须立刻失效）。若第二次 Create 撞上 uniqueIndex 静默失败，
// 记录会一直停在 rotate —— 登出后旧 token 仍能换到新令牌（实测过的 bug）。
func AddToBlacklistWithReason(db *gorm.DB, tokenHash string, userID uint, expiresAt time.Time, reason string) error {
	blacklist := TokenBlacklist{
		TokenHash: tokenHash,
		UserID:    userID,
		ExpiresAt: expiresAt,
		Reason:    reason,
	}
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "token_hash"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"user_id", "expires_at", "reason", "created_at",
		}),
	}).Create(&blacklist).Error
}

// BlacklistEntry 查一条未过期的黑名单记录（用于区分拉黑原因与时间）
func GetBlacklistEntry(db *gorm.DB, tokenHash string) (TokenBlacklist, bool) {
	var rec TokenBlacklist
	err := db.Where("token_hash = ? AND expires_at > ?", tokenHash, time.Now()).First(&rec).Error
	if err != nil {
		return TokenBlacklist{}, false
	}
	return rec, true
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

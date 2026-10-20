package handlers

import (
	"testing"
	"time"

	"cboard-go/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 同一个 refresh_token 会经历两次拉黑：先「轮换」（rotate，可宽限重放），
// 后「登出」（logout，必须立刻失效）。第二次必须是可靠覆盖，否则登出形同虚设。
//
// 线上实测（2026-10-20 部署后自测）：登出后再用同一 refresh 续期仍然 200 ——
// 原因就是第二次拉黑撞 uniqueIndex 静默失败，记录一直停在 rotate。
func TestBlacklistReasonOverwrittenByLogout(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.TokenBlacklist{}); err != nil {
		t.Fatalf("建表失败: %v", err)
	}

	const hash = "refresh-token-hash-for-test"
	expires := time.Now().Add(30 * 24 * time.Hour)

	// ① 轮换拉黑：允许重放
	if err := models.AddToBlacklistWithReason(db, hash, 42, expires, models.BlacklistReasonRotate); err != nil {
		t.Fatalf("轮换拉黑失败: %v", err)
	}
	entry, ok := models.GetBlacklistEntry(db, hash)
	if !ok {
		t.Fatal("应能查到轮换记录")
	}
	if entry.Reason != models.BlacklistReasonRotate {
		t.Fatalf("轮换后 reason 应为 rotate，实际 %q", entry.Reason)
	}
	if !canReplayRotatedToken(entry, time.Now()) {
		t.Fatal("轮换记录应允许重放")
	}

	// ② 用户显式登出：同一 token 再次拉黑，必须覆盖成 logout
	if err := models.AddToBlacklistWithReason(db, hash, 42, expires, models.BlacklistReasonLogout); err != nil {
		t.Fatalf("登出拉黑失败（upsert 不应报错）: %v", err)
	}
	entry, ok = models.GetBlacklistEntry(db, hash)
	if !ok {
		t.Fatal("登出后应仍能查到记录")
	}
	if entry.Reason != models.BlacklistReasonLogout {
		t.Fatalf("登出后 reason 应被覆盖为 logout，实际 %q", entry.Reason)
	}
	if canReplayRotatedToken(entry, time.Now()) {
		t.Fatal("登出后的令牌绝不能重放换新令牌")
	}

	// ③ 记录不能重复堆积（token_hash 唯一）
	var n int64
	db.Model(&models.TokenBlacklist{}).Where("token_hash = ?", hash).Count(&n)
	if n != 1 {
		t.Fatalf("同一 token 应只有 1 条黑名单记录，实际 %d", n)
	}
}

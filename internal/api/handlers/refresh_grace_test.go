package handlers

import (
	"testing"
	"time"
)

// 钉住「轮换宽限期」的语义：旧 refresh_token 在轮换后 60 秒内重复提交
// 必须重放上一次签发的同一对令牌（客户端丢响应/并发重试不再被迫重新登录），
// 超出窗口后必须回到原行为（拒绝），避免旧 token 无限可用。
func TestRotatedRefreshGraceReplaysSamePair(t *testing.T) {
	oldHash := "hash-old-1"
	storeRotatedRefreshResult(oldHash, "access-A", "refresh-A")

	a, r, ok := lookupRotatedRefreshResult(oldHash)
	if !ok || a != "access-A" || r != "refresh-A" {
		t.Fatalf("宽限期内应重放同一对令牌，实际 ok=%v access=%q refresh=%q", ok, a, r)
	}

	// 重复提交多次都应命中（幂等）
	for i := 0; i < 3; i++ {
		if _, _, ok := lookupRotatedRefreshResult(oldHash); !ok {
			t.Fatalf("第 %d 次重复提交应命中", i+1)
		}
	}
}

func TestRotatedRefreshGraceMissesUnknownToken(t *testing.T) {
	if _, _, ok := lookupRotatedRefreshResult("never-stored"); ok {
		t.Fatal("未记录过的 token 不应命中宽限期")
	}
}

func TestRotatedRefreshGraceExpires(t *testing.T) {
	oldHash := "hash-old-2"
	storeRotatedRefreshResult(oldHash, "access-B", "refresh-B")

	// 手工把过期时间提前（同包内可直接改缓存内容）
	rotatedRefreshCache.Store(oldHash, &rotatedTokenPair{
		accessToken:  "access-B",
		refreshToken: "refresh-B",
		expireAt:     time.Now().Add(-time.Second),
	})
	if _, _, ok := lookupRotatedRefreshResult(oldHash); ok {
		t.Fatal("超出宽限期后必须不再命中（否则旧 token 变相永久有效）")
	}
	if _, found := rotatedRefreshCache.Load(oldHash); found {
		t.Fatal("过期条目应被删除，避免缓存无限增长")
	}
}

func TestRotatedRefreshGraceWindowIsOneMinute(t *testing.T) {
	if refreshRotateGracePeriod != 60*time.Second {
		t.Fatalf("宽限期应为 60s，实际 %v", refreshRotateGracePeriod)
	}
}

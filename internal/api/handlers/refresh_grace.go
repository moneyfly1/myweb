package handlers

import (
	"sync"
	"time"
)

// 刷新令牌「轮换宽限期」。
//
// 背景（2026-09-25 线上问题：客户反馈「软件无故退出」）：
// /auth/refresh 一旦成功就会把**旧 refresh_token 立刻拉黑**。正常情况下没问题，
// 但只要第一次刷新的响应在途中丢了（客户端进程被杀、断网、切换网络/域名，或客户端
// 并发重试），客户端手里就只剩一个已拉黑的旧 token —— 下一次刷新必然 401
// 「刷新令牌已失效，请重新登录」，用户被迫重新登录（老客户端甚至会直接登出回登录页）。
//
// 这里让轮换**幂等**：旧 token 在轮换后 refreshRotateGracePeriod 内再次提交时，
// 直接返回上一次签发的**同一对令牌**（既不签发新的、也不报 401）。窗口一过，旧 token
// 依然按黑名单拒绝，安全性不变（旧 token 本身仍受 30 天有效期约束）。
const refreshRotateGracePeriod = 60 * time.Second

// 缓存上限：超出后整体清空，避免异常流量把内存撑爆。宽限期只有 60 秒，
// 清空的代价仅是那批请求按原逻辑报 401，不影响正确性。
const refreshRotateMaxEntries = 20000

type rotatedTokenPair struct {
	accessToken  string
	refreshToken string
	expireAt     time.Time
}

var (
	rotatedRefreshCache sync.Map
	rotatedJanitorOnce  sync.Once
)

// storeRotatedRefreshResult 记录「旧 token 哈希 → 刚签发的新令牌对」。
func storeRotatedRefreshResult(oldHash, accessToken, newRefreshToken string) {
	rotatedJanitorOnce.Do(startRotatedRefreshJanitor)

	if countRotatedRefreshEntries() >= refreshRotateMaxEntries {
		rotatedRefreshCache.Range(func(k, _ any) bool {
			rotatedRefreshCache.Delete(k)
			return true
		})
	}
	rotatedRefreshCache.Store(oldHash, &rotatedTokenPair{
		accessToken:  accessToken,
		refreshToken: newRefreshToken,
		expireAt:     time.Now().Add(refreshRotateGracePeriod),
	})
}

// lookupRotatedRefreshResult 命中 = 宽限期内的重复提交 → 重放上次结果。
func lookupRotatedRefreshResult(oldHash string) (accessToken, refreshToken string, ok bool) {
	val, found := rotatedRefreshCache.Load(oldHash)
	if !found {
		return "", "", false
	}
	pair, isPair := val.(*rotatedTokenPair)
	if !isPair || time.Now().After(pair.expireAt) {
		rotatedRefreshCache.Delete(oldHash)
		return "", "", false
	}
	return pair.accessToken, pair.refreshToken, true
}

func countRotatedRefreshEntries() int {
	n := 0
	rotatedRefreshCache.Range(func(_, _ any) bool {
		n++
		return n < refreshRotateMaxEntries
	})
	return n
}

// 定期清理过期条目（与黑名单缓存同风格：读时懒过期 + 后台定期清扫）。
func startRotatedRefreshJanitor() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			rotatedRefreshCache.Range(func(k, v any) bool {
				if pair, ok := v.(*rotatedTokenPair); ok && now.After(pair.expireAt) {
					rotatedRefreshCache.Delete(k)
				}
				return true
			})
		}
	}()
}

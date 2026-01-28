package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
	visitors map[string]*Visitor
	mu       sync.RWMutex
	rate     int           // 允许的请求次数
	window   time.Duration // 时间窗口
}

type Visitor struct {
	Count    int
	ResetAt  time.Time
	Locked   bool
	LockedAt time.Time
}

func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     rate,
		window:   window,
	}

	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, visitor := range rl.visitors {
			if !visitor.Locked && now.After(visitor.ResetAt.Add(rl.window)) {
				delete(rl.visitors, key)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Allow(key string) (allowed bool, resetAt time.Time, locked bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	visitor, exists := rl.visitors[key]

	if !exists {
		rl.visitors[key] = &Visitor{
			Count:   1,
			ResetAt: now.Add(rl.window),
		}
		return true, now.Add(rl.window), false
	}

	if visitor.Locked {
		if now.After(visitor.LockedAt.Add(15 * time.Minute)) {
			visitor.Locked = false
			visitor.Count = 0
			visitor.ResetAt = now.Add(rl.window)
			return true, visitor.ResetAt, false
		}
		return false, visitor.LockedAt.Add(15 * time.Minute), true
	}

	if now.After(visitor.ResetAt) {
		visitor.Count = 1
		visitor.ResetAt = now.Add(rl.window)
		return true, visitor.ResetAt, false
	}

	if visitor.Count >= rl.rate {
		visitor.Locked = true
		visitor.LockedAt = now
		return false, visitor.LockedAt.Add(15 * time.Minute), true
	}

	visitor.Count++
	return true, visitor.ResetAt, false
}

func (rl *RateLimiter) Check(key string) (allowed bool, resetAt time.Time, locked bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	visitor, exists := rl.visitors[key]

	if !exists {
		return true, now.Add(rl.window), false
	}

	if visitor.Locked {
		if now.After(visitor.LockedAt.Add(15 * time.Minute)) {
			visitor.Locked = false
			visitor.Count = 0
			visitor.ResetAt = now.Add(rl.window)
			return true, visitor.ResetAt, false
		}
		return false, visitor.LockedAt.Add(15 * time.Minute), true
	}

	if now.After(visitor.ResetAt) {
		return true, now.Add(rl.window), false
	}

	if visitor.Count >= rl.rate {
		return false, visitor.ResetAt, false
	}

	return true, visitor.ResetAt, false
}

func (rl *RateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	visitor, exists := rl.visitors[key]

	if exists {
		visitor.Locked = false
		visitor.Count = 0
		visitor.ResetAt = now.Add(rl.window)
	}
}

var (
	loginRateLimiter    = NewRateLimiter(5, 15*time.Minute)  // 登录：15分钟内最多5次
	registerRateLimiter = NewRateLimiter(3, 1*time.Hour)     // 注册：1小时内最多3次
	verifyCodeLimiter   = NewRateLimiter(5, 1*time.Hour)     // 验证码：1小时内最多5次
	generalRateLimiter  = NewRateLimiter(100, 1*time.Minute) // 通用：1分钟内最多100次
)

func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := utils.GetRealClientIP(c)
		if key == "" {
			key = c.ClientIP() // 如果获取不到，使用 Gin 的默认方法
		}

		if userID, exists := c.Get("user_id"); exists {
			key = key + ":" + fmt.Sprintf("%d", userID.(uint))
		}

		allowed, resetAt, locked := limiter.Allow(key)

		if !allowed {
			if locked {
				utils.ErrorResponse(c, http.StatusTooManyRequests, "请求过于频繁，账户已被临时锁定，请稍后再试", nil)
			} else {
				c.Header("X-RateLimit-Limit", "100")
				c.Header("X-RateLimit-Remaining", "0")
				c.Header("X-RateLimit-Reset", resetAt.Format(time.RFC1123))
				utils.ErrorResponse(c, http.StatusTooManyRequests, "请求过于频繁，请稍后再试", nil)
			}
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", "100")
		c.Header("X-RateLimit-Reset", resetAt.Format(time.RFC1123))

		c.Next()
	}
}

func LoginRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := utils.GetRealClientIP(c)
		if key == "" {
			key = c.ClientIP() // 如果获取不到，使用 Gin 的默认方法
		}

		allowed, resetAt, locked := loginRateLimiter.Check(key)

		if !allowed {
			if locked {
				utils.CreateSecurityLog(c, "ip_blocked", "HIGH",
					fmt.Sprintf("IP被封禁: %s (登录失败次数过多，已锁定15分钟)", key),
					map[string]interface{}{
						"ip":        key,
						"reason":    "登录失败次数过多",
						"lock_time": "15分钟",
						"reset_at":  resetAt.Format("2006-01-02 15:04:05"),
					})
			} else {
				utils.CreateSecurityLog(c, "login_rate_limit", "MEDIUM",
					fmt.Sprintf("登录速率限制: IP %s 接近限制", key),
					map[string]interface{}{
						"ip":       key,
						"reason":   "登录失败次数过多",
						"reset_at": resetAt.Format("2006-01-02 15:04:05"),
					})
			}

			if locked {
				utils.ErrorResponse(c, http.StatusTooManyRequests, "登录失败次数过多，账户已被临时锁定15分钟，请稍后再试", nil)
			} else {
				c.Header("X-RateLimit-Limit", "5")
				c.Header("X-RateLimit-Remaining", "0")
				c.Header("X-RateLimit-Reset", resetAt.Format(time.RFC1123))
				utils.ErrorResponse(c, http.StatusTooManyRequests, "登录失败次数过多，请稍后再试", nil)
			}
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", "5")
		c.Header("X-RateLimit-Reset", resetAt.Format(time.RFC1123))

		c.Set("rate_limit_key", key)

		c.Next()
	}
}

func IncrementLoginAttempt(ip string) {
	loginRateLimiter.Allow(ip)
}

func ResetLoginAttempt(ip string) {
	loginRateLimiter.Reset(ip)
}

func GetLoginAttemptStatus(ip string) (allowed bool, resetAt time.Time, locked bool) {
	return loginRateLimiter.Check(ip)
}

func RegisterRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := utils.GetRealClientIP(c)
		if key == "" {
			key = c.ClientIP() // 如果获取不到，使用 Gin 的默认方法
		}

		allowed, resetAt, locked := registerRateLimiter.Allow(key)

		if !allowed {
			if locked {
				utils.ErrorResponse(c, http.StatusTooManyRequests, "注册请求过于频繁，账户已被临时锁定，请稍后再试", nil)
			} else {
				c.Header("X-RateLimit-Limit", "3")
				c.Header("X-RateLimit-Remaining", "0")
				c.Header("X-RateLimit-Reset", resetAt.Format(time.RFC1123))
				utils.ErrorResponse(c, http.StatusTooManyRequests, "注册请求过于频繁，请稍后再试", nil)
			}
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", "3")
		c.Header("X-RateLimit-Reset", resetAt.Format(time.RFC1123))

		c.Next()
	}
}

func VerifyCodeRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := utils.GetRealClientIP(c)
		if key == "" {
			key = c.ClientIP() // 如果获取不到，使用 Gin 的默认方法
		}

		allowed, resetAt, locked := verifyCodeLimiter.Allow(key)

		if !allowed {
			if locked {
				utils.ErrorResponse(c, http.StatusTooManyRequests, "验证码发送过于频繁，已被临时锁定，请稍后再试", nil)
			} else {
				c.Header("X-RateLimit-Limit", "5")
				c.Header("X-RateLimit-Remaining", "0")
				c.Header("X-RateLimit-Reset", resetAt.Format(time.RFC1123))
				utils.ErrorResponse(c, http.StatusTooManyRequests, "验证码发送过于频繁，请稍后再试", nil)
			}
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", "5")
		c.Header("X-RateLimit-Reset", resetAt.Format(time.RFC1123))

		c.Next()
	}
}

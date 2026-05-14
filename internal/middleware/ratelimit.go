package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
	visitors     map[string]*Visitor
	mu           sync.RWMutex
	rate         int
	window       time.Duration
	lockDuration time.Duration
}

type Visitor struct {
	Count    int
	ResetAt  time.Time
	Locked   bool
	LockedAt time.Time
}

func NewRateLimiter(rate int, window time.Duration, lockDuration time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors:     make(map[string]*Visitor),
		rate:         rate,
		window:       window,
		lockDuration: lockDuration,
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
			if visitor.Locked {
				if now.After(visitor.LockedAt.Add(rl.lockDuration)) {
					delete(rl.visitors, key)
				}
				continue
			}

			if now.After(visitor.ResetAt.Add(rl.window)) {
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
		if now.After(visitor.LockedAt.Add(rl.lockDuration)) {
			visitor.Locked = false
			visitor.Count = 0
			visitor.ResetAt = now.Add(rl.window)
			return true, visitor.ResetAt, false
		}
		return false, visitor.LockedAt.Add(rl.lockDuration), true
	}

	if now.After(visitor.ResetAt) {
		visitor.Count = 1
		visitor.ResetAt = now.Add(rl.window)
		return true, visitor.ResetAt, false
	}

	if visitor.Count >= rl.rate {
		visitor.Locked = true
		visitor.LockedAt = now
		return false, visitor.LockedAt.Add(rl.lockDuration), true
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
		if now.After(visitor.LockedAt.Add(rl.lockDuration)) {
			visitor.Locked = false
			visitor.Count = 0
			visitor.ResetAt = now.Add(rl.window)
			return true, visitor.ResetAt, false
		}
		return false, visitor.LockedAt.Add(rl.lockDuration), true
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

func (rl *RateLimiter) UpdateConfig(rate int, lockDuration time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rate > 0 {
		rl.rate = rate
	}
	if lockDuration > 0 {
		rl.lockDuration = lockDuration
	}
}

func (rl *RateLimiter) GetConfig() (rate int, lockDuration time.Duration) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.rate, rl.lockDuration
}

var (
	loginRateLimiter    = NewRateLimiter(5, 15*time.Minute, 15*time.Minute)
	registerRateLimiter = NewRateLimiter(3, 1*time.Hour, 1*time.Hour)
	verifyCodeLimiter   = NewRateLimiter(5, 1*time.Hour, 1*time.Hour)
	generalRateLimiter  = NewRateLimiter(100, 1*time.Minute, 5*time.Minute)
)

// ReloadLoginRateLimiter 从数据库读取 login_fail_limit 和 login_lock_time 并更新限流器
func ReloadLoginRateLimiter() {
	db := database.GetDB()
	if db == nil {
		return
	}

	var cfg models.SystemConfig
	rate := 5
	if err := db.Where("category = ? AND key = ?", "security", "login_fail_limit").First(&cfg).Error; err == nil {
		if v, err := strconv.Atoi(cfg.Value); err == nil && v > 0 {
			rate = v
		}
	}

	lockMinutes := 15
	if err := db.Where("category = ? AND key = ?", "security", "login_lock_time").First(&cfg).Error; err == nil {
		if v, err := strconv.Atoi(cfg.Value); err == nil && v > 0 {
			lockMinutes = v
		}
	}

	loginRateLimiter.UpdateConfig(rate, time.Duration(lockMinutes)*time.Minute)

	if utils.AppLogger != nil {
		utils.AppLogger.Info("登录限流器配置已加载: 最大失败次数=%d, 锁定时间=%d分钟", rate, lockMinutes)
	}
}

func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := utils.GetRealClientIP(c)
		if key == "" {
			key = c.ClientIP()
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
			key = c.ClientIP()
		}

		allowed, resetAt, locked := loginRateLimiter.Check(key)

		rate, lockDuration := loginRateLimiter.GetConfig()
		lockMinutes := int(lockDuration.Minutes())
		rateLimitStr := fmt.Sprintf("%d", rate)
		lockStr := fmt.Sprintf("%d分钟", lockMinutes)

		if !allowed {
			if locked {
				utils.CreateSecurityLog(c, "ip_blocked", "HIGH",
					fmt.Sprintf("IP被封禁: %s (登录失败次数过多，已锁定%s)", key, lockStr),
					map[string]interface{}{
						"ip":        key,
						"reason":    "登录失败次数过多",
						"lock_time": lockStr,
						"reset_at":  utils.FormatBeijingTime(resetAt),
					})
			} else {
				utils.CreateSecurityLog(c, "login_rate_limit", "MEDIUM",
					fmt.Sprintf("登录速率限制: IP %s 接近限制", key),
					map[string]interface{}{
						"ip":       key,
						"reason":   "登录失败次数过多",
						"reset_at": utils.FormatBeijingTime(resetAt),
					})
			}

			if locked {
				utils.ErrorResponse(c, http.StatusTooManyRequests,
					fmt.Sprintf("登录失败次数过多，账户已被临时锁定%s，请稍后再试", lockStr), nil)
			} else {
				c.Header("X-RateLimit-Limit", rateLimitStr)
				c.Header("X-RateLimit-Remaining", "0")
				c.Header("X-RateLimit-Reset", resetAt.Format(time.RFC1123))
				utils.ErrorResponse(c, http.StatusTooManyRequests, "登录失败次数过多，请稍后再试", nil)
			}
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", rateLimitStr)
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
			key = c.ClientIP()
		}

		allowed, resetAt, locked := registerRateLimiter.Allow(key)

		if !allowed {
			if locked {
				utils.CreateSecurityLog(c, "register_ip_blocked", "HIGH",
					fmt.Sprintf("注册IP被封禁: %s (请求过于频繁，已临时锁定)", key),
					map[string]interface{}{"ip": key, "reason": "注册请求过于频繁", "reset_at": utils.FormatBeijingTime(resetAt)})
				utils.ErrorResponse(c, http.StatusTooManyRequests, "注册请求过于频繁，账户已被临时锁定，请稍后再试", nil)
			} else {
				utils.CreateSecurityLog(c, "register_rate_limit", "MEDIUM",
					fmt.Sprintf("注册速率限制: IP %s 接近限制", key),
					map[string]interface{}{"ip": key, "reset_at": utils.FormatBeijingTime(resetAt)})
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
			key = c.ClientIP()
		}

		allowed, resetAt, locked := verifyCodeLimiter.Allow(key)

		if !allowed {
			if locked {
				utils.CreateSecurityLog(c, "verify_code_rate_limit", "MEDIUM",
					fmt.Sprintf("验证码发送IP被限流/锁定: %s (请求过于频繁)", key),
					map[string]interface{}{"ip": key, "reason": "验证码发送过于频繁", "reset_at": utils.FormatBeijingTime(resetAt), "locked": true})
				utils.ErrorResponse(c, http.StatusTooManyRequests, "验证码发送过于频繁，已被临时锁定，请稍后再试", nil)
			} else {
				utils.CreateSecurityLog(c, "verify_code_rate_limit", "MEDIUM",
					fmt.Sprintf("验证码发送速率限制: IP %s 接近限制", key),
					map[string]interface{}{"ip": key, "reset_at": utils.FormatBeijingTime(resetAt)})
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

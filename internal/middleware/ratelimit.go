package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// RateLimiter 速率限制器
type RateLimiter struct {
	visitors map[string]*Visitor
	mu       sync.RWMutex
	rate     int           // 允许的请求次数
	window   time.Duration // 时间窗口
}

// Visitor 访问者信息
type Visitor struct {
	Count    int
	ResetAt  time.Time
	Locked   bool
	LockedAt time.Time
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     rate,
		window:   window,
	}

	// 定期清理过期的访问者记录
	go rl.cleanup()

	return rl
}

// cleanup 定期清理过期的访问者记录
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, visitor := range rl.visitors {
			// 如果记录已过期且未锁定，删除它
			if !visitor.Locked && now.After(visitor.ResetAt.Add(rl.window)) {
				delete(rl.visitors, key)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow 检查是否允许请求（会增加计数）
func (rl *RateLimiter) Allow(key string) (allowed bool, resetAt time.Time, locked bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	visitor, exists := rl.visitors[key]

	if !exists {
		// 创建新访问者
		rl.visitors[key] = &Visitor{
			Count:   1,
			ResetAt: now.Add(rl.window),
		}
		return true, now.Add(rl.window), false
	}

	// 检查是否被锁定
	if visitor.Locked {
		// 检查锁定是否已过期（15分钟）
		if now.After(visitor.LockedAt.Add(15 * time.Minute)) {
			// 解锁并重置
			visitor.Locked = false
			visitor.Count = 0
			visitor.ResetAt = now.Add(rl.window)
			return true, visitor.ResetAt, false
		}
		return false, visitor.LockedAt.Add(15 * time.Minute), true
	}

	// 检查时间窗口是否已过期
	if now.After(visitor.ResetAt) {
		// 重置计数
		visitor.Count = 1
		visitor.ResetAt = now.Add(rl.window)
		return true, visitor.ResetAt, false
	}

	// 检查是否超过限制
	if visitor.Count >= rl.rate {
		// 锁定15分钟
		visitor.Locked = true
		visitor.LockedAt = now
		return false, visitor.LockedAt.Add(15 * time.Minute), true
	}

	// 增加计数
	visitor.Count++
	return true, visitor.ResetAt, false
}

// Check 检查是否允许请求（不增加计数）
func (rl *RateLimiter) Check(key string) (allowed bool, resetAt time.Time, locked bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	visitor, exists := rl.visitors[key]

	if !exists {
		return true, now.Add(rl.window), false
	}

	// 检查是否被锁定
	if visitor.Locked {
		// 检查锁定是否已过期（15分钟）
		if now.After(visitor.LockedAt.Add(15 * time.Minute)) {
			// 解锁并重置
			visitor.Locked = false
			visitor.Count = 0
			visitor.ResetAt = now.Add(rl.window)
			return true, visitor.ResetAt, false
		}
		return false, visitor.LockedAt.Add(15 * time.Minute), true
	}

	// 检查时间窗口是否已过期
	if now.After(visitor.ResetAt) {
		return true, now.Add(rl.window), false
	}

	// 检查是否超过限制
	if visitor.Count >= rl.rate {
		return false, visitor.ResetAt, false
	}

	return true, visitor.ResetAt, false
}

// Reset 重置指定key的计数
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

// 全局速率限制器实例
var (
	loginRateLimiter    = NewRateLimiter(5, 15*time.Minute)  // 登录：15分钟内最多5次
	registerRateLimiter = NewRateLimiter(3, 1*time.Hour)     // 注册：1小时内最多3次
	verifyCodeLimiter   = NewRateLimiter(5, 1*time.Hour)     // 验证码：1小时内最多5次
	generalRateLimiter  = NewRateLimiter(100, 1*time.Minute) // 通用：1分钟内最多100次
)

// RateLimitMiddleware 通用速率限制中间件
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取客户端真实IP
		key := utils.GetRealClientIP(c)
		if key == "" {
			key = c.ClientIP() // 如果获取不到，使用 Gin 的默认方法
		}

		// 如果用户已登录，也使用用户ID作为key的一部分
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

		// 设置响应头
		c.Header("X-RateLimit-Limit", "100")
		c.Header("X-RateLimit-Reset", resetAt.Format(time.RFC1123))

		c.Next()
	}
}

// LoginRateLimitMiddleware 登录速率限制中间件
// 注意：此中间件只检查是否允许，不增加计数
// 计数增加和重置由登录handler根据登录结果决定
func LoginRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取客户端真实IP
		key := utils.GetRealClientIP(c)
		if key == "" {
			key = c.ClientIP() // 如果获取不到，使用 Gin 的默认方法
		}

		// 只检查，不增加计数
		allowed, resetAt, locked := loginRateLimiter.Check(key)

		if !allowed {
			if locked {
				// IP被封禁，记录详细日志
				utils.CreateSecurityLog(c, "ip_blocked", "HIGH",
					fmt.Sprintf("IP被封禁: %s (登录失败次数过多，已锁定15分钟)", key),
					map[string]interface{}{
						"ip":        key,
						"reason":    "登录失败次数过多",
						"lock_time": "15分钟",
						"reset_at":  resetAt.Format("2006-01-02 15:04:05"),
					})
			} else {
				// 接近限制，记录警告
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

		// 设置响应头
		c.Header("X-RateLimit-Limit", "5")
		c.Header("X-RateLimit-Reset", resetAt.Format(time.RFC1123))

		// 将key存储到上下文中，以便handler可以访问
		c.Set("rate_limit_key", key)

		c.Next()
	}
}

// IncrementLoginAttempt 增加登录尝试计数（登录失败时调用）
func IncrementLoginAttempt(ip string) {
	loginRateLimiter.Allow(ip)
}

// ResetLoginAttempt 重置登录尝试计数（登录成功时调用）
func ResetLoginAttempt(ip string) {
	loginRateLimiter.Reset(ip)
}

// GetLoginAttemptStatus 获取登录尝试状态（用于日志记录）
func GetLoginAttemptStatus(ip string) (allowed bool, resetAt time.Time, locked bool) {
	return loginRateLimiter.Check(ip)
}

// RegisterRateLimitMiddleware 注册速率限制中间件
func RegisterRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取客户端真实IP
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

		// 设置响应头
		c.Header("X-RateLimit-Limit", "3")
		c.Header("X-RateLimit-Reset", resetAt.Format(time.RFC1123))

		c.Next()
	}
}

// VerifyCodeRateLimitMiddleware 验证码速率限制中间件
func VerifyCodeRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取客户端真实IP
		key := utils.GetRealClientIP(c)
		if key == "" {
			key = c.ClientIP() // 如果获取不到，使用 Gin 的默认方法
		}

		// 注意：邮箱/手机号的限制在handler中实现，这里只做IP级别的限制
		// 更细粒度的限制（同一邮箱/手机号）在SendVerificationCode handler中处理

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

		// 设置响应头
		c.Header("X-RateLimit-Limit", "5")
		c.Header("X-RateLimit-Reset", resetAt.Format(time.RFC1123))

		c.Next()
	}
}

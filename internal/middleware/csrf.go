package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CSRFManager CSRF管理器
type CSRFManager struct {
	tokens    map[string]*CSRFToken
	mu        sync.RWMutex
	secretKey string
}

// CSRFToken CSRF Token
type CSRFToken struct {
	Token     string
	ExpiresAt time.Time
}

// 全局CSRF管理器
var csrfManager *CSRFManager
var csrfOnce sync.Once

// GetCSRFManager 获取CSRF管理器单例
func GetCSRFManager() *CSRFManager {
	csrfOnce.Do(func() {
		csrfManager = &CSRFManager{
			tokens:    make(map[string]*CSRFToken),
			secretKey: generateSecretKey(),
		}
		// 定期清理过期的token
		go csrfManager.cleanup()
	})
	return csrfManager
}

// generateSecretKey 生成密钥
func generateSecretKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// cleanup 定期清理过期的token
func (cm *CSRFManager) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cm.mu.Lock()
		now := time.Now()
		for key, token := range cm.tokens {
			if now.After(token.ExpiresAt) {
				delete(cm.tokens, key)
			}
		}
		cm.mu.Unlock()
	}
}

// GenerateToken 生成CSRF Token
func (cm *CSRFManager) GenerateToken(sessionID string) (string, error) {
	// 生成随机token
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.URLEncoding.EncodeToString(b)

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 存储token（24小时有效期）
	cm.tokens[sessionID] = &CSRFToken{
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	return token, nil
}

// ValidateToken 验证CSRF Token
func (cm *CSRFManager) ValidateToken(sessionID, token string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	storedToken, exists := cm.tokens[sessionID]
	if !exists {
		return false
	}

	// 检查是否过期
	if time.Now().After(storedToken.ExpiresAt) {
		return false
	}

	// 验证token
	return storedToken.Token == token
}

// getSessionID 获取会话ID（从Cookie或Header）
func getSessionID(c *gin.Context) string {
	// 优先从Cookie获取
	if cookie, err := c.Cookie("session_id"); err == nil && cookie != "" {
		return cookie
	}

	// 如果没有 session_id Cookie，生成一个随机的 sessionID 并设置
	// 使用随机字符串作为 sessionID，确保稳定性（不依赖 IP 或 User-Agent）
	b := make([]byte, 32)
	rand.Read(b)
	sessionID := base64.URLEncoding.EncodeToString(b)

	// 设置 session_id Cookie（30天有效期）
	isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetCookie("session_id", sessionID, 30*24*3600, "/", "", isSecure, false)

	return sessionID
}

// CSRFMiddleware CSRF保护中间件
func CSRFMiddleware() gin.HandlerFunc {
	manager := GetCSRFManager()

	return func(c *gin.Context) {
		// 只对状态变更操作进行CSRF保护
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			// GET请求生成或更新token（即使已存在也更新，确保token不会过期）
			sessionID := getSessionID(c)
			token, err := manager.GenerateToken(sessionID)
			if err == nil {
				c.Header("X-CSRF-Token", token)
				// HttpOnly 设置为 false，允许前端 JavaScript 读取 CSRF Token
				// Secure 根据请求协议动态设置（HTTPS 时设置为 true）
				isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
				c.SetCookie("csrf_token", token, 86400, "/", "", isSecure, false) // HttpOnly=false, Secure=动态
			}
			c.Next()
			return
		}

		// POST/PUT/DELETE/PATCH请求验证token
		sessionID := getSessionID(c)

		// 从Header或Form获取token
		token := c.GetHeader("X-CSRF-Token")
		if token == "" {
			token = c.PostForm("csrf_token")
		}
		if token == "" {
			// 尝试从Cookie获取
			if cookie, err := c.Cookie("csrf_token"); err == nil {
				token = cookie
			}
		}

		if token == "" || !manager.ValidateToken(sessionID, token) {
			// 检查Origin和Referer头（额外的保护）
			origin := c.GetHeader("Origin")
			referer := c.GetHeader("Referer")
			host := c.Request.Host

			// 如果Origin或Referer存在，验证它们是否匹配当前域名
			if origin != "" && !isValidOrigin(origin, host) {
				c.JSON(http.StatusForbidden, gin.H{
					"success": false,
					"message": "CSRF验证失败：无效的请求来源",
				})
				c.Abort()
				return
			}

			if referer != "" && !isValidReferer(referer, host) {
				c.JSON(http.StatusForbidden, gin.H{
					"success": false,
					"message": "CSRF验证失败：无效的请求来源",
				})
				c.Abort()
				return
			}

			// 如果token验证失败，尝试生成新token并返回，让前端自动重试
			// 这可以解决token过期或会话ID变化的问题
			newToken, err := manager.GenerateToken(sessionID)
			if err == nil {
				isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
				c.SetCookie("csrf_token", newToken, 86400, "/", "", isSecure, false)
				c.Header("X-CSRF-Token", newToken)
			}

			// 如果token验证失败，返回错误（包含新token，让前端可以重试）
			if token == "" {
				c.JSON(http.StatusForbidden, gin.H{
					"success":    false,
					"message":    "CSRF Token缺失，请刷新页面后重试",
					"csrf_token": newToken, // 返回新token，方便前端自动重试
				})
				c.Abort()
				return
			}

			c.JSON(http.StatusForbidden, gin.H{
				"success":    false,
				"message":    "CSRF验证失败，请刷新页面后重试",
				"csrf_token": newToken, // 返回新token，方便前端自动重试
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// isValidOrigin 验证Origin是否有效
func isValidOrigin(origin, host string) bool {
	if origin == "" {
		return false
	}
	// 支持localhost开发环境
	if origin == "http://localhost" || origin == "https://localhost" ||
		origin == "http://localhost:5173" || origin == "https://localhost:5173" ||
		origin == "http://127.0.0.1" || origin == "https://127.0.0.1" ||
		origin == "http://127.0.0.1:5173" || origin == "https://127.0.0.1:5173" {
		return true
	}
	// 检查origin是否匹配host
	return origin == "https://"+host || origin == "http://"+host ||
		origin == "https://"+host+"/" || origin == "http://"+host+"/" ||
		strings.HasPrefix(origin, "https://"+host+":") || strings.HasPrefix(origin, "http://"+host+":")
}

// isValidReferer 验证Referer是否有效
func isValidReferer(referer, host string) bool {
	if referer == "" {
		return false
	}
	// 支持localhost开发环境
	if strings.HasPrefix(referer, "http://localhost") || strings.HasPrefix(referer, "https://localhost") ||
		strings.HasPrefix(referer, "http://127.0.0.1") || strings.HasPrefix(referer, "https://127.0.0.1") {
		return true
	}
	// 检查referer是否匹配host
	return referer == "https://"+host || referer == "http://"+host ||
		referer == "https://"+host+"/" || referer == "http://"+host+"/" ||
		strings.HasPrefix(referer, "https://"+host+":") || strings.HasPrefix(referer, "http://"+host+":") ||
		strings.HasPrefix(referer, "https://"+host+"/") || strings.HasPrefix(referer, "http://"+host+"/")
}

// CSRFExemptMiddleware 豁免CSRF检查的中间件（用于某些公开API）
func CSRFExemptMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("csrf_exempt", true)
		c.Next()
	}
}

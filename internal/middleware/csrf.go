package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
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

	// 如果没有Cookie，使用IP+User-Agent作为会话标识
	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")
	return fmt.Sprintf("%s:%s", ip, ua)
}

// CSRFMiddleware CSRF保护中间件
func CSRFMiddleware() gin.HandlerFunc {
	manager := GetCSRFManager()

	return func(c *gin.Context) {
		// 只对状态变更操作进行CSRF保护
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			// GET请求生成token
			sessionID := getSessionID(c)
			token, err := manager.GenerateToken(sessionID)
			if err == nil {
				c.Header("X-CSRF-Token", token)
				c.SetCookie("csrf_token", token, 86400, "/", "", false, true) // HttpOnly, Secure
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

			// 如果token验证失败，返回错误
			if token == "" {
				c.JSON(http.StatusForbidden, gin.H{
					"success": false,
					"message": "CSRF Token缺失，请刷新页面后重试",
				})
				c.Abort()
				return
			}

			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "CSRF验证失败，请刷新页面后重试",
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


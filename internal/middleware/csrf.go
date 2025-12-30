package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
	"time"

	"cboard-go/internal/utils"

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

		// 对于移动应用（使用 Bearer token 认证），豁免 CSRF 检查
		// 移动应用使用 Bearer token 认证，不受 CSRF 攻击影响
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" && parts[1] != "" {
				// 有有效的 Bearer token，豁免 CSRF 检查（移动应用）
				c.Next()
				return
			}
		}

		// POST/PUT/DELETE/PATCH请求验证token
		// 确保使用相同的sessionID（如果不存在则生成，但会导致验证失败）
		sessionID := getSessionID(c)

		// 从Header获取token（优先）
		token := c.GetHeader("X-CSRF-Token")
		if token == "" {
			// 尝试从Cookie获取
			if cookie, err := c.Cookie("csrf_token"); err == nil {
				token = cookie
			}
		}

		// 验证token
		if token == "" || !manager.ValidateToken(sessionID, token) {
			// 检查Origin和Referer
			origin := c.GetHeader("Origin")
			referer := c.GetHeader("Referer")
			host := c.Request.Host
			if (origin != "" && !isValidOrigin(origin, host)) || (referer != "" && !isValidReferer(referer, host)) {
				utils.ErrorResponse(c, http.StatusForbidden, "CSRF验证失败：无效的请求来源", nil)
				c.Abort()
				return
			}

			// 生成新token（基于当前sessionID）并返回，让前端自动重试
			newToken, _ := manager.GenerateToken(sessionID)
			isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
			c.SetCookie("csrf_token", newToken, 86400, "/", "", isSecure, false)
			c.Header("X-CSRF-Token", newToken)
			utils.ErrorResponse(c, http.StatusForbidden, "CSRF验证失败，请刷新页面后重试", nil)
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
	// 支持局域网IP访问（开发环境）
	if strings.HasPrefix(origin, "http://192.168.") || strings.HasPrefix(origin, "https://192.168.") ||
		strings.HasPrefix(origin, "http://10.") || strings.HasPrefix(origin, "https://10.") ||
		strings.HasPrefix(origin, "http://172.") || strings.HasPrefix(origin, "https://172.") {
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
	// 支持局域网IP访问（开发环境）
	if strings.HasPrefix(referer, "http://192.168.") || strings.HasPrefix(referer, "https://192.168.") ||
		strings.HasPrefix(referer, "http://10.") || strings.HasPrefix(referer, "https://10.") ||
		strings.HasPrefix(referer, "http://172.") || strings.HasPrefix(referer, "https://172.") {
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

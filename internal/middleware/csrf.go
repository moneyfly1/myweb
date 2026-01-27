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

type CSRFManager struct {
	tokens    map[string]*CSRFToken
	mu        sync.RWMutex
	secretKey string
}

type CSRFToken struct {
	Token     string
	ExpiresAt time.Time
}

var csrfManager *CSRFManager
var csrfOnce sync.Once

func GetCSRFManager() *CSRFManager {
	csrfOnce.Do(func() {
		csrfManager = &CSRFManager{
			tokens:    make(map[string]*CSRFToken),
			secretKey: generateSecretKey(),
		}
		go csrfManager.cleanup()
	})
	return csrfManager
}

func generateSecretKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

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

func (cm *CSRFManager) GenerateToken(sessionID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.URLEncoding.EncodeToString(b)

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.tokens[sessionID] = &CSRFToken{
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	return token, nil
}

func (cm *CSRFManager) ValidateToken(sessionID, token string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	storedToken, exists := cm.tokens[sessionID]
	if !exists {
		return false
	}

	if time.Now().After(storedToken.ExpiresAt) {
		return false
	}

	return storedToken.Token == token
}

func getSessionID(c *gin.Context) string {
	if cookie, err := c.Cookie("session_id"); err == nil && cookie != "" {
		return cookie
	}

	b := make([]byte, 32)
	rand.Read(b)
	sessionID := base64.URLEncoding.EncodeToString(b)

	isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetCookie("session_id", sessionID, 30*24*3600, "/", "", isSecure, false)

	return sessionID
}

func CSRFMiddleware() gin.HandlerFunc {
	manager := GetCSRFManager()

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/v1/payment/notify/") {
			c.Next()
			return
		}

		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			sessionID := getSessionID(c)
			token, err := manager.GenerateToken(sessionID)
			if err == nil {
				c.Header("X-CSRF-Token", token)
				isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
				c.SetCookie("csrf_token", token, 86400, "/", "", isSecure, false)
			}
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" && parts[1] != "" {
				c.Next()
				return
			}
		}

		sessionID := getSessionID(c)

		token := c.GetHeader("X-CSRF-Token")
		if token == "" {
			if cookie, err := c.Cookie("csrf_token"); err == nil {
				token = cookie
			}
		}

		if token == "" || !manager.ValidateToken(sessionID, token) {
			origin := c.GetHeader("Origin")
			referer := c.GetHeader("Referer")
			host := c.Request.Host
			if (origin != "" && !isValidOrigin(origin, host)) || (referer != "" && !isValidReferer(referer, host)) {
				utils.ErrorResponse(c, http.StatusForbidden, "CSRF验证失败：无效的请求来源", nil)
				c.Abort()
				return
			}

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

func isValidOrigin(origin, host string) bool {
	if origin == "" {
		return false
	}
	return origin == "https://"+host || origin == "http://"+host ||
		origin == "https://"+host+"/" || origin == "http://"+host+"/" ||
		strings.HasPrefix(origin, "https://"+host+":") || strings.HasPrefix(origin, "http://"+host+":")
}

func isValidReferer(referer, host string) bool {
	if referer == "" {
		return false
	}
	return referer == "https://"+host || referer == "http://"+host ||
		referer == "https://"+host+"/" || referer == "http://"+host+"/" ||
		strings.HasPrefix(referer, "https://"+host+":") || strings.HasPrefix(referer, "http://"+host+":") ||
		strings.HasPrefix(referer, "https://"+host+"/") || strings.HasPrefix(referer, "http://"+host+"/")
}

func CSRFExemptMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("csrf_exempt", true)
		c.Next()
	}
}

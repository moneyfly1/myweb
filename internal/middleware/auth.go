package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// authUserCache 轻量内存缓存，避免每次请求都查数据库
var authUserCache sync.Map

type cachedUser struct {
	user      models.User
	expireAt  time.Time
}

const authCacheTTL = 2 * time.Minute

func getAuthCachedUser(userID uint) (*models.User, bool) {
	val, ok := authUserCache.Load(userID)
	if !ok {
		return nil, false
	}
	cu := val.(*cachedUser)
	if time.Now().After(cu.expireAt) {
		authUserCache.Delete(userID)
		return nil, false
	}
	// 返回拷贝，避免外部修改污染缓存
	copy := cu.user
	return &copy, true
}

func setAuthCachedUser(user *models.User) {
	authUserCache.Store(user.ID, &cachedUser{
		user:     *user,
		expireAt: time.Now().Add(authCacheTTL),
	})
}

// InvalidateAuthUserCache 供外部在用户状态变更时清除缓存
func InvalidateAuthUserCache(userID uint) {
	authUserCache.Delete(userID)
}

// tokenBlacklistCache 黑名单 token 缓存，避免每次请求查 DB
// key: tokenHash(string), value: *cachedBlacklist
var tokenBlacklistCache sync.Map

type cachedBlacklist struct {
	blacklisted bool
	expireAt    time.Time
}

const blacklistCacheTTL = 5 * time.Minute

func isTokenBlacklistedCached(tokenHash string) bool {
	if val, ok := tokenBlacklistCache.Load(tokenHash); ok {
		cb := val.(*cachedBlacklist)
		if time.Now().Before(cb.expireAt) {
			return cb.blacklisted
		}
		tokenBlacklistCache.Delete(tokenHash)
	}
	result := models.IsTokenBlacklisted(database.GetDB(), tokenHash)
	tokenBlacklistCache.Store(tokenHash, &cachedBlacklist{
		blacklisted: result,
		expireAt:    time.Now().Add(blacklistCacheTTL),
	})
	return result
}

// AddTokenToBlacklistCache 登出时主动写入缓存
func AddTokenToBlacklistCache(tokenHash string) {
	tokenBlacklistCache.Store(tokenHash, &cachedBlacklist{
		blacklisted: true,
		expireAt:    time.Now().Add(blacklistCacheTTL),
	})
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "未提供认证令牌", nil)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "无效的认证格式", nil)
			c.Abort()
			return
		}

		token := parts[1]

		db := database.GetDB()
		tokenHash := utils.HashToken(token)
		if isTokenBlacklistedCached(tokenHash) {
			utils.CreateSecurityLog(c, "auth_token_blacklisted", "LOW",
				"访问被拒绝: 令牌已失效（已登出或已加入黑名单）",
				map[string]interface{}{"path": c.Request.URL.Path})
			utils.ErrorResponse(c, http.StatusUnauthorized, "令牌已失效，请重新登录", nil)
			c.Abort()
			return
		}

		claims, err := utils.VerifyToken(token)
		if err != nil {
			utils.CreateSecurityLog(c, "auth_token_invalid", "MEDIUM",
				"访问被拒绝: 无效或过期的访问令牌",
				map[string]interface{}{"path": c.Request.URL.Path, "reason": err.Error()})
			utils.ErrorResponse(c, http.StatusUnauthorized, "无效或过期的令牌", err)
			c.Abort()
			return
		}

		if claims.Type != "access" {
			utils.CreateSecurityLog(c, "auth_token_invalid", "MEDIUM",
				"访问被拒绝: 刷新令牌不能用于访问",
				map[string]interface{}{"path": c.Request.URL.Path})
			utils.ErrorResponse(c, http.StatusUnauthorized, "刷新令牌不能用于访问", nil)
			c.Abort()
			return
		}

		var user models.User
		if cached, ok := getAuthCachedUser(claims.UserID); ok {
			user = *cached
		} else {
			if err := db.First(&user, claims.UserID).Error; err != nil {
				utils.CreateSecurityLog(c, "auth_token_invalid", "MEDIUM",
					"访问被拒绝: 令牌对应用户不存在",
					map[string]interface{}{"path": c.Request.URL.Path, "user_id": claims.UserID})
				utils.ErrorResponse(c, http.StatusUnauthorized, "用户不存在", err)
				c.Abort()
				return
			}
			setAuthCachedUser(&user)
		}

		if !user.IsActive {
			utils.CreateSecurityLog(c, "auth_token_invalid", "MEDIUM",
				"访问被拒绝: 账户已被禁用",
				map[string]interface{}{"path": c.Request.URL.Path, "user_id": user.ID})
			utils.ErrorResponse(c, http.StatusForbidden, "账户已被禁用，无法使用服务。如有疑问，请联系管理员。", nil)
			c.Abort()
			return
		}

		c.Set("user", &user)
		c.Set("user_id", user.ID)
		c.Set("is_admin", user.IsAdmin)

		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, exists := c.Get("is_admin")
		if !exists {
			utils.ErrorResponse(c, http.StatusUnauthorized, "请先登录", nil)
			c.Abort()
			return
		}

		admin, ok := isAdmin.(bool)
		if !ok || !admin {
			utils.CreateSecurityLog(c, "admin_forbidden", "MEDIUM",
				"非管理员尝试访问管理接口",
				map[string]interface{}{"path": c.Request.URL.Path})
			utils.ErrorResponse(c, http.StatusForbidden, "权限不足，需要管理员权限", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

func GetCurrentUser(c *gin.Context) (*models.User, bool) {
	user, exists := c.Get("user")
	if !exists {
		return nil, false
	}

	u, ok := user.(*models.User)
	return u, ok
}

func TryAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		token := parts[1]
		db := database.GetDB()
		tokenHash := utils.HashToken(token)
		if isTokenBlacklistedCached(tokenHash) {
			c.Next()
			return
		}

		claims, err := utils.VerifyToken(token)
		if err != nil {
			c.Next()
			return
		}

		if claims.Type != "access" {
			c.Next()
			return
		}

		var user models.User
		if cached, ok := getAuthCachedUser(claims.UserID); ok {
			user = *cached
		} else {
			if err := db.First(&user, claims.UserID).Error; err != nil {
				c.Next()
				return
			}
			setAuthCachedUser(&user)
		}

		if !user.IsActive {
			c.Next()
			return
		}

		c.Set("user", &user)
		c.Set("user_id", user.ID)
		c.Set("is_admin", user.IsAdmin)
		c.Next()
	}
}

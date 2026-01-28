package middleware

import (
	"net/http"
	"strings"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

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
		if models.IsTokenBlacklisted(db, tokenHash) {
			utils.ErrorResponse(c, http.StatusUnauthorized, "令牌已失效，请重新登录", nil)
			c.Abort()
			return
		}

		claims, err := utils.VerifyToken(token)
		if err != nil {
			utils.ErrorResponse(c, http.StatusUnauthorized, "无效或过期的令牌", err)
			c.Abort()
			return
		}

		if claims.Type != "access" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "刷新令牌不能用于访问", nil)
			c.Abort()
			return
		}

		var user models.User
		if err := db.First(&user, claims.UserID).Error; err != nil {
			utils.ErrorResponse(c, http.StatusUnauthorized, "用户不存在", err)
			c.Abort()
			return
		}

		if !user.IsActive {
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
		if models.IsTokenBlacklisted(db, tokenHash) {
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
		if err := db.First(&user, claims.UserID).Error; err != nil {
			c.Next()
			return
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

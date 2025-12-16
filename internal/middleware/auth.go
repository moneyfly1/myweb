package middleware

import (
	"net/http"
	"strings"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "未提供认证令牌",
			})
			c.Abort()
			return
		}

		// 提取 Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "无效的认证格式",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// 检查Token是否在黑名单中（已撤销）
		db := database.GetDB()
		tokenHash := utils.HashToken(token)
		if models.IsTokenBlacklisted(db, tokenHash) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "令牌已失效，请重新登录",
			})
			c.Abort()
			return
		}

		claims, err := utils.VerifyToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "无效或过期的令牌",
			})
			c.Abort()
			return
		}

		// 检查令牌类型
		if claims.Type != "access" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "刷新令牌不能用于访问",
			})
			c.Abort()
			return
		}

		// 从数据库获取用户
		var user models.User
		if err := db.First(&user, claims.UserID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "用户不存在",
			})
			c.Abort()
			return
		}

		// 检查用户是否激活
		if !user.IsActive {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "账户已被禁用，无法使用服务。如有疑问，请联系管理员。",
			})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文
		c.Set("user", &user)
		c.Set("user_id", user.ID)
		c.Set("is_admin", user.IsAdmin)

		c.Next()
	}
}

// AdminMiddleware 管理员中间件
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, exists := c.Get("is_admin")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "请先登录",
			})
			c.Abort()
			return
		}

		admin, ok := isAdmin.(bool)
		if !ok || !admin {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "权限不足，需要管理员权限",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetCurrentUser 获取当前用户
func GetCurrentUser(c *gin.Context) (*models.User, bool) {
	user, exists := c.Get("user")
	if !exists {
		return nil, false
	}

	u, ok := user.(*models.User)
	return u, ok
}

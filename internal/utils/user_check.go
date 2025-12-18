package utils

import (
	"net/http"

	"cboard-go/internal/models"

	"github.com/gin-gonic/gin"
)

// CheckUserActive 检查用户是否激活，如果未激活则返回错误响应
func CheckUserActive(user *models.User) (bool, gin.H) {
	if !user.IsActive {
		return false, gin.H{
			"success": false,
			"message": "账户已被禁用，无法使用服务。如有疑问，请联系管理员。",
		}
	}
	return true, nil
}

// CheckUserActiveWithResponse 检查用户是否激活，如果未激活则直接返回响应并中止请求
func CheckUserActiveWithResponse(c *gin.Context, user *models.User) bool {
	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "账户已被禁用，无法使用服务。如有疑问，请联系管理员。",
		})
		c.Abort()
		return false
	}
	return true
}


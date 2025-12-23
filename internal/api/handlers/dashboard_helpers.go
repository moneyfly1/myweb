package handlers

import (
	"fmt"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// buildAbnormalUserData 构建异常用户数据（从 GetAbnormalUsers 提取）
func buildAbnormalUserData(db *gorm.DB, users []models.User) []gin.H {
	now := utils.GetBeijingTime()
	oneMonthAgo := now.AddDate(0, -1, 0)
	userList := make([]gin.H, 0, len(users))

	for _, user := range users {
		lastLogin := "从未登录"
		if user.LastLogin.Valid {
			lastLogin = user.LastLogin.Time.Format("2006-01-02 15:04:05")
		}

		// 计算状态
		status := "inactive"
		if user.IsActive {
			status = "active"
		}

		// 统计用户异常行为
		var resetCount int64
		db.Model(&models.SubscriptionReset{}).Where("user_id = ?", user.ID).Count(&resetCount)

		var subscriptionCount int64
		db.Model(&models.Subscription{}).Where("user_id = ?", user.ID).Count(&subscriptionCount)

		// 判断异常类型和次数（按优先级）
		abnormalType := "unknown"
		abnormalCount := 0
		description := ""

		// 优先级1：账户被禁用
		if !user.IsActive {
			abnormalType = "disabled"
			abnormalCount = 1
			description = "账户已被禁用"
		} else if resetCount >= 5 {
			// 优先级2：频繁重置订阅
			abnormalType = "frequent_reset"
			abnormalCount = int(resetCount)
			description = fmt.Sprintf("频繁重置订阅 %d 次", resetCount)
		} else if subscriptionCount >= 10 {
			// 优先级3：频繁创建订阅
			abnormalType = "frequent_subscription"
			abnormalCount = int(subscriptionCount)
			description = fmt.Sprintf("频繁创建订阅 %d 次", subscriptionCount)
		} else if !user.LastLogin.Valid && user.CreatedAt.Before(oneMonthAgo) {
			// 优先级4：长期未登录
			abnormalType = "inactive"
			abnormalCount = 1
			description = "长期未登录（注册超过1个月且从未登录）"
		} else {
			// 其他情况：多重异常
			abnormalType = "multiple_abnormal"
			abnormalCount = 1
			description = "存在多种异常行为"
		}

		// 获取最后活动时间
		var lastActivity string
		var lastActivityRecord models.UserActivity
		if err := db.Where("user_id = ?", user.ID).Order("created_at DESC").First(&lastActivityRecord).Error; err != nil {
			// 如果没有活动记录，使用创建时间
			lastActivity = user.CreatedAt.Format("2006-01-02 15:04:05")
		} else {
			lastActivity = lastActivityRecord.CreatedAt.Format("2006-01-02 15:04:05")
		}

		userList = append(userList, gin.H{
			"id":                 user.ID,
			"user_id":            user.ID,
			"username":           user.Username,
			"email":              user.Email,
			"is_active":          user.IsActive,
			"is_verified":        user.IsVerified,
			"status":             status,
			"last_login":         lastLogin,
			"created_at":         user.CreatedAt.Format("2006-01-02 15:04:05"),
			"abnormal_type":     abnormalType,
			"abnormal_count":    abnormalCount,
			"reset_count":        resetCount,
			"subscription_count": subscriptionCount,
			"description":       description,
			"last_activity":      lastActivity,
		})
	}

	return userList
}


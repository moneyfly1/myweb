package handlers

import (
	"fmt"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// buildSubscriptionListData 构建订阅列表数据（从 GetAdminSubscriptions 提取）
func buildSubscriptionListData(db *gorm.DB, subscriptions []models.Subscription, c *gin.Context) []gin.H {
	if len(subscriptions) == 0 {
		return []gin.H{}
	}

	// 收集订阅ID和用户ID
	subIDs := make([]uint, len(subscriptions))
	userIDs := make([]uint, 0, len(subscriptions))
	userIDSet := make(map[uint]bool)
	for i, s := range subscriptions {
		subIDs[i] = s.ID
		if !userIDSet[s.UserID] {
			userIDs = append(userIDs, s.UserID)
			userIDSet[s.UserID] = true
		}
	}

	// 批量查询所有用户，避免 N+1 查询
	var users []models.User
	userMap := make(map[uint]*models.User)
	if len(userIDs) > 0 {
		db.Where("id IN ?", userIDs).Find(&users)
		for i := range users {
			userMap[users[i].ID] = &users[i]
		}
	}

	// 批量查询设备统计
	type Stat struct {
		SubID uint
		Type  *string
		Count int64
	}
	var onlineStats []Stat
	db.Model(&models.Device{}).Select("subscription_id as sub_id, count(*) as count").
		Where("subscription_id IN ? AND is_active = ?", subIDs, true).
		Group("subscription_id").Scan(&onlineStats)

	var typeStats []Stat
	db.Model(&models.Device{}).Select("subscription_id as sub_id, subscription_type as type, count(*) as count").
		Where("subscription_id IN ?", subIDs).
		Group("subscription_id, subscription_type").Scan(&typeStats)

	// 构建统计映射
	onlineMap, appleMap, clashMap := make(map[uint]int64), make(map[uint]int64), make(map[uint]int64)
	for _, s := range onlineStats {
		onlineMap[s.SubID] = s.Count
	}
	for _, s := range typeStats {
		if s.Type == nil {
			continue
		}
		if *s.Type == "v2ray" || *s.Type == "ssr" {
			appleMap[s.SubID] += s.Count
		}
		if *s.Type == "clash" {
			clashMap[s.SubID] += s.Count
		}
	}

	// 构建列表数据
	list := make([]gin.H, 0, len(subscriptions))
	for _, sub := range subscriptions {
		online := onlineMap[sub.ID]
		curr := sub.CurrentDevices
		if curr < int(online) {
			curr = int(online)
		}

		universal, clash := getSubscriptionURLs(c, sub.SubscriptionURL)
		
		// 使用预加载的 User 或从 userMap 获取，避免 N+1 查询
		var userInfo gin.H
		if sub.User.ID > 0 {
			userInfo = gin.H{"id": sub.User.ID, "username": sub.User.Username, "email": sub.User.Email}
		} else if user, ok := userMap[sub.UserID]; ok {
			userInfo = gin.H{"id": user.ID, "username": user.Username, "email": user.Email}
		} else {
			userInfo = gin.H{
				"id":       0,
				"username": fmt.Sprintf("用户已删除 (ID: %d)", sub.UserID),
				"email":    fmt.Sprintf("deleted_user_%d", sub.UserID),
				"deleted":  true,
			}
		}

		daysUntil, isExpired, now := 0, false, utils.GetBeijingTime()
		if !sub.ExpireTime.IsZero() {
			if diff := sub.ExpireTime.Sub(now); diff > 0 {
				daysUntil = int(diff.Hours() / 24)
			} else {
				isExpired = true
			}
		}

		// 使用数据库中的订阅次数字段，如果没有则使用统计值作为后备
		universalCount := sub.UniversalCount
		clashCount := sub.ClashCount
		if universalCount == 0 && appleMap[sub.ID] > 0 {
			universalCount = int(appleMap[sub.ID])
		}
		if clashCount == 0 && clashMap[sub.ID] > 0 {
			clashCount = int(clashMap[sub.ID])
		}

		list = append(list, gin.H{
			"id":                 sub.ID,
			"user_id":            sub.UserID,
			"user":               userInfo,
			"username":           userInfo["username"],
			"email":              userInfo["email"],
			"subscription_url":   sub.SubscriptionURL,
			"universal_url":      universal,
			"clash_url":          clash,
			"status":             sub.Status,
			"is_active":          sub.IsActive,
			"device_limit":       sub.DeviceLimit,
			"current_devices":    curr,
			"online_devices":     online,
			"apple_count":        universalCount,
			"clash_count":        clashCount,
			"expire_time":        sub.ExpireTime.Format("2006-01-02 15:04:05"),
			"days_until_expire":  daysUntil,
			"is_expired":         isExpired,
			"created_at":         sub.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return list
}


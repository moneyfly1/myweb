package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"cboard-go/internal/models"

	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"
)

// UserCache 用户缓存相关操作
type UserCache struct{}

var userCache = &UserCache{}

const (
	userCachePrefix    = "user:"
	userCacheExpire    = 10 * time.Minute
	userListCacheKey   = "users:list"
	userListCacheExpire = 5 * time.Minute
)

// GetUser 从缓存获取用户信息
func (uc *UserCache) GetUser(userID uint) (*models.User, error) {
	if !IsRedisEnabled() {
		return nil, fmt.Errorf("redis not enabled")
	}

	key := fmt.Sprintf("%s%d", userCachePrefix, userID)
	data, err := redisClient.Get(context.Background(), key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 缓存未命中，返回nil而不是错误
		}
		return nil, err
	}

	var user models.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// SetUser 缓存用户信息
func (uc *UserCache) SetUser(user *models.User) error {
	if !IsRedisEnabled() || user == nil {
		return nil
	}

	key := fmt.Sprintf("%s%d", userCachePrefix, user.ID)
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return redisClient.Set(context.Background(), key, data, userCacheExpire).Err()
}

// DeleteUser 删除用户缓存
func (uc *UserCache) DeleteUser(userID uint) error {
	if !IsRedisEnabled() {
		return nil
	}

	key := fmt.Sprintf("%s%d", userCachePrefix, userID)
	return redisClient.Del(context.Background(), key).Err()
}

// GetUsersBatch 批量获取用户（优先从缓存）
func (uc *UserCache) GetUsersBatch(userIDs []uint) (map[uint]*models.User, error) {
	if !IsRedisEnabled() || len(userIDs) == 0 {
		return nil, fmt.Errorf("redis not enabled or empty ids")
	}

	userMap := make(map[uint]*models.User)
	
	// 逐个获取，简化实现
	for _, id := range userIDs {
		key := fmt.Sprintf("%s%d", userCachePrefix, id)
		data, err := redisClient.Get(context.Background(), key).Bytes()
		if err != nil {
			if err == redis.Nil {
				continue // 缓存未命中
			}
			// 其他错误，跳过这个key
			continue
		}
		var user models.User
		if err := json.Unmarshal(data, &user); err != nil {
			continue
		}
		userMap[id] = &user
	}

	return userMap, nil
}

// SetUsersBatch 批量缓存用户
func (uc *UserCache) SetUsersBatch(users []models.User) error {
	if !IsRedisEnabled() || len(users) == 0 {
		return nil
	}

	pipe := redisClient.Pipeline()
	for _, user := range users {
		key := fmt.Sprintf("%s%d", userCachePrefix, user.ID)
		data, err := json.Marshal(user)
		if err != nil {
			continue
		}
		pipe.Set(context.Background(), key, data, userCacheExpire)
	}

	_, err := pipe.Exec(context.Background())
	return err
}

// InvalidateUserCache 使指定用户缓存失效
func (uc *UserCache) InvalidateUser(userID uint) error {
	return uc.DeleteUser(userID)
}

// InvalidateUsersCache 使多个用户缓存失效
func (uc *UserCache) InvalidateUsers(userIDs []uint) error {
	if !IsRedisEnabled() || len(userIDs) == 0 {
		return nil
	}

	keys := make([]string, len(userIDs))
	for i, id := range userIDs {
		keys[i] = fmt.Sprintf("%s%d", userCachePrefix, id)
	}

	return redisClient.Del(context.Background(), keys...).Err()
}

// GetOrSetUser 获取用户，如果缓存中没有则从数据库查询并缓存
func (uc *UserCache) GetOrSetUser(db *gorm.DB, userID uint) (*models.User, error) {
	// 尝试从缓存获取
	if user, err := uc.GetUser(userID); err == nil && user != nil {
		return user, nil
	}

	// 从数据库查询
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	// 写入缓存
	if err := uc.SetUser(&user); err != nil {
		// 缓存失败不影响主流程，只记录日志
		log.Printf("缓存用户失败 (user_id=%d): %v\n", userID, err)
	}

	return &user, nil
}

// GetUserStats 获取用户统计信息缓存
const userStatsCacheKey = "user:stats:%d"
const userStatsCacheExpire = 2 * time.Minute

func (uc *UserCache) GetUserStats(userID uint) (*map[string]interface{}, error) {
	if !IsRedisEnabled() {
		return nil, fmt.Errorf("redis not enabled")
	}

	key := fmt.Sprintf(userStatsCacheKey, userID)
	data, err := redisClient.Get(context.Background(), key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var stats map[string]interface{}
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

func (uc *UserCache) SetUserStats(userID uint, stats map[string]interface{}) error {
	if !IsRedisEnabled() {
		return nil
	}

	key := fmt.Sprintf(userStatsCacheKey, userID)
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}

	return redisClient.Set(context.Background(), key, data, userStatsCacheExpire).Err()
}

func (uc *UserCache) DeleteUserStats(userID uint) error {
	if !IsRedisEnabled() {
		return nil
	}

	key := fmt.Sprintf(userStatsCacheKey, userID)
	return redisClient.Del(context.Background(), key).Err()
}

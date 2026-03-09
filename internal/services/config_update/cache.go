package config_update

import (
	"cboard-go/internal/core/cache"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// CacheService 订阅配置缓存服务
type CacheService struct{}

// GetSystemNodesCache 获取系统节点缓存
func (cs *CacheService) GetSystemNodesCache() ([]*ProxyNode, bool) {
	if !cache.IsRedisEnabled() {
		return nil, false
	}

	cacheKey := "nodes:system:all"
	cached, err := cache.Get(cacheKey)
	if err != nil || cached == "" {
		return nil, false
	}

	var nodes []*ProxyNode
	if err := json.Unmarshal([]byte(cached), &nodes); err != nil {
		return nil, false
	}

	return nodes, true
}

// SetSystemNodesCache 设置系统节点缓存
func (cs *CacheService) SetSystemNodesCache(nodes []*ProxyNode) error {
	if !cache.IsRedisEnabled() {
		return nil
	}

	// #nosec G117 - Password field is proxy node password, not user credential
	data, err := json.Marshal(nodes) // #nosec G117
	if err != nil {
		return err
	}

	cacheKey := "nodes:system:all"
	return cache.Set(cacheKey, string(data), 1*time.Hour)
}

// GetCustomNodesCache 获取用户自定义节点缓存
func (cs *CacheService) GetCustomNodesCache(userID uint) ([]*ProxyNode, bool) {
	if !cache.IsRedisEnabled() {
		return nil, false
	}

	cacheKey := fmt.Sprintf("nodes:custom:user:%d", userID)
	cached, err := cache.Get(cacheKey)
	if err != nil || cached == "" {
		return nil, false
	}

	var nodes []*ProxyNode
	if err := json.Unmarshal([]byte(cached), &nodes); err != nil {
		return nil, false
	}

	return nodes, true
}

// SetCustomNodesCache 设置用户自定义节点缓存
func (cs *CacheService) SetCustomNodesCache(userID uint, nodes []*ProxyNode) error {
	if !cache.IsRedisEnabled() {
		return nil
	}

	// #nosec G117 - Password field is proxy node password, not user credential
	data, err := json.Marshal(nodes) // #nosec G117
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("nodes:custom:user:%d", userID)
	return cache.Set(cacheKey, string(data), 10*time.Minute)
}

// ClearSystemNodesCache 清除系统节点缓存
func (cs *CacheService) ClearSystemNodesCache() error {
	if !cache.IsRedisEnabled() {
		return nil
	}
	return cache.Del("nodes:system:all")
}

// ClearCustomNodesCache 清除用户自定义节点缓存
func (cs *CacheService) ClearCustomNodesCache(userID uint) error {
	if !cache.IsRedisEnabled() {
		return nil
	}
	cacheKey := fmt.Sprintf("nodes:custom:user:%d", userID)
	return cache.Del(cacheKey)
}

// ClearAllSubscriptionCache 清除所有订阅配置缓存（节点变更时调用）
func (cs *CacheService) ClearAllSubscriptionCache() error {
	if !cache.IsRedisEnabled() {
		return nil
	}

	// 清除系统节点缓存
	if err := cache.Del("nodes:system:all"); err != nil {
		log.Printf("failed to delete nodes cache: %v", err)
	}

	// 清除所有订阅配置缓存（节点变更时，所有订阅配置都应失效）
	// 使用 SCAN 命令批量删除，避免 KEYS 命令阻塞
	client := cache.GetRedisClient()
	if client != nil {
		ctx := context.Background()
		var cursor uint64
		for {
			var keys []string
			var err error
			keys, cursor, err = client.Scan(ctx, cursor, "subscription:config:*", 100).Result()
			if err != nil {
				break
			}
			if len(keys) > 0 {
				if delErr := client.Del(ctx, keys...).Err(); delErr != nil {
					log.Printf("failed to delete subscription config cache keys: %v", delErr)
				}
			}
			if cursor == 0 {
				break
			}
		}
	}

	return nil
}

// GetSubscriptionConfigCache 获取订阅配置缓存
func (cs *CacheService) GetSubscriptionConfigCache(subscriptionURL, format string) (string, bool) {
	if !cache.IsRedisEnabled() {
		return "", false
	}

	cacheKey := fmt.Sprintf("subscription:config:%s:%s", subscriptionURL, format)
	cached, err := cache.Get(cacheKey)
	if err != nil || cached == "" {
		return "", false
	}

	return cached, true
}

// SetSubscriptionConfigCache 设置订阅配置缓存（智能TTL）
func (cs *CacheService) SetSubscriptionConfigCache(subscriptionURL, format, config string, ttl time.Duration) error {
	if !cache.IsRedisEnabled() {
		return nil
	}

	cacheKey := fmt.Sprintf("subscription:config:%s:%s", subscriptionURL, format)
	return cache.Set(cacheKey, config, ttl)
}

// ClearSubscriptionConfigCache 清除指定订阅的配置缓存
func (cs *CacheService) ClearSubscriptionConfigCache(subscriptionURL string) error {
	if !cache.IsRedisEnabled() {
		return nil
	}

	// 清除该订阅的所有格式缓存
	if err := cache.Del(fmt.Sprintf("subscription:config:%s:clash", subscriptionURL)); err != nil {
		log.Printf("failed to delete clash cache: %v", err)
	}
	if err := cache.Del(fmt.Sprintf("subscription:config:%s:base64", subscriptionURL)); err != nil {
		log.Printf("failed to delete base64 cache: %v", err)
	}

	return nil
}

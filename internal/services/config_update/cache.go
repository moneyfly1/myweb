package config_update

import (
	"cboard-go/internal/core/cache"
	"encoding/json"
	"fmt"
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

	data, err := json.Marshal(nodes)
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

	data, err := json.Marshal(nodes)
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
	cache.Del("nodes:system:all")

	// 注意：这里不清除用户自定义节点缓存，因为系统节点变更不影响自定义节点
	// 如果需要清除所有缓存，可以使用 Redis 的 KEYS 命令（生产环境慎用）

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
	cache.Del(fmt.Sprintf("subscription:config:%s:base64", subscriptionURL))
	cache.Del(fmt.Sprintf("subscription:config:%s:clash", subscriptionURL))

	return nil
}

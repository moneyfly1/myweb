package cache_service

import (
	"cboard-go/internal/core/cache"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// CacheService 通用缓存服务
type CacheService struct{}

// NewCacheService 创建缓存服务实例
func NewCacheService() *CacheService {
	return &CacheService{}
}

// Get 获取缓存（泛型）
func (cs *CacheService) Get(key string, result interface{}) (bool, error) {
	if !cache.IsRedisEnabled() {
		return false, nil
	}

	cached, err := cache.Get(key)
	if err != nil || cached == "" {
		return false, nil
	}

	if err := json.Unmarshal([]byte(cached), result); err != nil {
		// 缓存数据异常，删除
		if delErr := cache.Del(key); delErr != nil {
			log.Printf("failed to delete invalid cache: %v", delErr)
		}
		return false, err
	}

	return true, nil
}

// Set 设置缓存（泛型）
func (cs *CacheService) Set(key string, value interface{}, ttl time.Duration) error {
	if !cache.IsRedisEnabled() {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return cache.Set(key, string(data), ttl)
}

// Del 删除缓存
func (cs *CacheService) Del(key string) error {
	if !cache.IsRedisEnabled() {
		return nil
	}
	return cache.Del(key)
}

// ==========================================
// 用户信息缓存
// ==========================================

// ClearUserCache 清除用户信息缓存
func (cs *CacheService) ClearUserCache(userID uint) error {
	key := fmt.Sprintf("user:info:%d", userID)
	return cs.Del(key)
}

// ==========================================
// 套餐列表缓存
// ==========================================

// GetPackagesCache 获取套餐列表缓存
func (cs *CacheService) GetPackagesCache() ([]map[string]interface{}, bool) {
	var packages []map[string]interface{}
	key := "packages:list:active"

	ok, err := cs.Get(key, &packages)
	if err != nil || !ok {
		return nil, false
	}

	return packages, true
}

// SetPackagesCache 设置套餐列表缓存
func (cs *CacheService) SetPackagesCache(packages []map[string]interface{}) error {
	key := "packages:list:active"
	return cs.Set(key, packages, 30*time.Minute)
}

// ClearPackagesCache 清除套餐列表缓存
func (cs *CacheService) ClearPackagesCache() error {
	key := "packages:list:active"
	return cs.Del(key)
}

// ==========================================
// 公告列表缓存
// ==========================================

// SetAnnouncementsCache 设置公告列表缓存
func (cs *CacheService) SetAnnouncementsCache(announcements []map[string]interface{}) error {
	key := "announcements:list:active"
	return cs.Set(key, announcements, 10*time.Minute)
}

// ClearAnnouncementsCache 清除公告列表缓存
func (cs *CacheService) ClearAnnouncementsCache() error {
	key := "announcements:list:active"
	return cs.Del(key)
}

// ==========================================
// 系统配置缓存
// ==========================================

// SetSystemConfigCache 设置系统配置缓存
func (cs *CacheService) SetSystemConfigCache(category string, configs []map[string]interface{}) error {
	key := fmt.Sprintf("system:config:%s", category)
	return cs.Set(key, configs, 1*time.Hour)
}

// ClearSystemConfigCache 清除系统配置缓存
func (cs *CacheService) ClearSystemConfigCache(category string) error {
	key := fmt.Sprintf("system:config:%s", category)
	return cs.Del(key)
}

// ==========================================
// 支付方式缓存
// ==========================================

// GetPaymentMethodsCache 获取支付方式列表缓存
func (cs *CacheService) GetPaymentMethodsCache() ([]map[string]interface{}, bool) {
	var methods []map[string]interface{}
	key := "payment:methods:active"

	ok, err := cs.Get(key, &methods)
	if err != nil || !ok {
		return nil, false
	}

	return methods, true
}

// SetPaymentMethodsCache 设置支付方式列表缓存
func (cs *CacheService) SetPaymentMethodsCache(methods []map[string]interface{}) error {
	key := "payment:methods:active"
	return cs.Set(key, methods, 1*time.Hour)
}

// ClearPaymentMethodsCache 清除支付方式列表缓存
func (cs *CacheService) ClearPaymentMethodsCache() error {
	key := "payment:methods:active"
	return cs.Del(key)
}

// ==========================================
// 节点列表缓存（保留但不推荐使用 - 节点更新频繁）
// ==========================================

// ClearNodesCache 清除节点列表缓存
func (cs *CacheService) ClearNodesCache() error {
	key := "nodes:list:active"
	return cs.Del(key)
}

// ==========================================
// 用户订阅缓存
// ==========================================

// ClearUserSubscriptionCache 清除用户订阅缓存
func (cs *CacheService) ClearUserSubscriptionCache(userID uint) error {
	key := fmt.Sprintf("user:subscription:%d", userID)
	return cs.Del(key)
}

// ==========================================
// 统计数据缓存
// ==========================================

// GetStatisticsCache 获取统计数据缓存
func (cs *CacheService) GetStatisticsCache(cacheKey string) (map[string]interface{}, bool) {
	var stats map[string]interface{}
	key := fmt.Sprintf("statistics:%s", cacheKey)

	ok, err := cs.Get(key, &stats)
	if err != nil || !ok {
		return nil, false
	}

	return stats, true
}

// SetStatisticsCache 设置统计数据缓存
func (cs *CacheService) SetStatisticsCache(cacheKey string, stats map[string]interface{}, ttl time.Duration) error {
	key := fmt.Sprintf("statistics:%s", cacheKey)
	return cs.Set(key, stats, ttl)
}

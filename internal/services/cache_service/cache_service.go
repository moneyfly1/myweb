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

// Generation 读取缓存键版本号（防陈旧回填用，详见 core/cache.SetIfGeneration）
func (cs *CacheService) Generation(key string) int64 {
	return cache.Generation(key)
}

// SetIfUnchanged 版本号未变才写入缓存；返回 false 表示期间缓存已被清除（数据已变更），本次写入被丢弃
func (cs *CacheService) SetIfUnchanged(key string, generation int64, value interface{}, ttl time.Duration) (bool, error) {
	if !cache.IsRedisEnabled() {
		return false, nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return false, err
	}
	return cache.SetIfGeneration(key, generation, string(data), ttl)
}

// Invalidate 删除缓存并递增版本号：数据变更后调用，
// 既清掉当前缓存，也让并发的"陈旧回填"写入失效
func (cs *CacheService) Invalidate(key string) error {
	if !cache.IsRedisEnabled() {
		return nil
	}
	return cache.DelWithGeneration(key)
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

// 套餐列表缓存
const packagesCacheKey = "packages:list:active"

// GetPackagesCache 获取套餐列表缓存
func (cs *CacheService) GetPackagesCache() ([]map[string]interface{}, bool) {
	var packages []map[string]interface{}

	ok, err := cs.Get(packagesCacheKey, &packages)
	if err != nil || !ok {
		return nil, false
	}

	return packages, true
}

// PackagesCacheGeneration 查库前取一次版本号，写回时用 SetPackagesCacheIfUnchanged 校验
func (cs *CacheService) PackagesCacheGeneration() int64 {
	return cs.Generation(packagesCacheKey)
}

// SetPackagesCacheIfUnchanged 版本号未变才写回套餐缓存（避免把改价前的旧列表写回缓存）
func (cs *CacheService) SetPackagesCacheIfUnchanged(generation int64, packages []map[string]interface{}) (bool, error) {
	return cs.SetIfUnchanged(packagesCacheKey, generation, packages, 30*time.Minute)
}

// ClearPackagesCache 清除套餐列表缓存（同时递增版本号，使并发陈旧回填失效）
func (cs *CacheService) ClearPackagesCache() error {
	return cs.Invalidate(packagesCacheKey)
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

const paymentMethodsCacheKey = "payment:methods:active"

// GetPaymentMethodsCache 获取支付方式列表缓存
func (cs *CacheService) GetPaymentMethodsCache() ([]map[string]interface{}, bool) {
	var methods []map[string]interface{}

	ok, err := cs.Get(paymentMethodsCacheKey, &methods)
	if err != nil || !ok {
		return nil, false
	}

	return methods, true
}

// PaymentMethodsCacheGeneration 查库前取一次版本号，写回时校验
func (cs *CacheService) PaymentMethodsCacheGeneration() int64 {
	return cs.Generation(paymentMethodsCacheKey)
}

// SetPaymentMethodsCacheIfUnchanged 版本号未变才写回支付方式缓存
// （避免管理员停用某支付方式后，并发的旧数据回填让已停用的方式继续可选）
func (cs *CacheService) SetPaymentMethodsCacheIfUnchanged(generation int64, methods []map[string]interface{}) (bool, error) {
	return cs.SetIfUnchanged(paymentMethodsCacheKey, generation, methods, 1*time.Hour)
}

// ClearPaymentMethodsCache 清除支付方式列表缓存（同时递增版本号）
func (cs *CacheService) ClearPaymentMethodsCache() error {
	return cs.Invalidate(paymentMethodsCacheKey)
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

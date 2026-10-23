package cache_service

import (
	"cboard-go/internal/core/cache"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// CacheService 通用缓存服务。
//
// 约定：**缓存只写不读 = 纯负担**。此前这里还有 user:info / user:subscription /
// announcements:list / nodes:list / system:config 五组 key，只有写与删、没有任何
// 读取方：每个写操作后都白跑一次 Redis 往返，还给维护者"这里有缓存"的错觉。
// 系统配置与公开设置现在走 utils 的进程内 30s 缓存（config_cache.go），
// 公告/节点/用户信息本身查询很轻，不需要 Redis 层缓存。
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

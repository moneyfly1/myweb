package cache_service

import (
	"cboard-go/internal/core/cache"
	"encoding/json"
	"fmt"
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
		cache.Del(key)
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

// GetUserCache 获取用户信息缓存
func (cs *CacheService) GetUserCache(userID uint) (map[string]interface{}, bool) {
	var user map[string]interface{}
	key := fmt.Sprintf("user:info:%d", userID)
	
	ok, err := cs.Get(key, &user)
	if err != nil || !ok {
		return nil, false
	}
	
	return user, true
}

// SetUserCache 设置用户信息缓存
func (cs *CacheService) SetUserCache(userID uint, user map[string]interface{}) error {
	key := fmt.Sprintf("user:info:%d", userID)
	return cs.Set(key, user, 10*time.Minute)
}

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

// GetAnnouncementsCache 获取公告列表缓存
func (cs *CacheService) GetAnnouncementsCache() ([]map[string]interface{}, bool) {
	var announcements []map[string]interface{}
	key := "announcements:list:active"
	
	ok, err := cs.Get(key, &announcements)
	if err != nil || !ok {
		return nil, false
	}
	
	return announcements, true
}

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

// GetSystemConfigCache 获取系统配置缓存
func (cs *CacheService) GetSystemConfigCache(category string) ([]map[string]interface{}, bool) {
	var configs []map[string]interface{}
	key := fmt.Sprintf("system:config:%s", category)
	
	ok, err := cs.Get(key, &configs)
	if err != nil || !ok {
		return nil, false
	}
	
	return configs, true
}

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

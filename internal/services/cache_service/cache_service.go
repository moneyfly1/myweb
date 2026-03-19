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
// 知识库缓存
// ==========================================

// GetKnowledgeCategoriesCache 获取知识库分类缓存
func (cs *CacheService) GetKnowledgeCategoriesCache() ([]map[string]interface{}, bool) {
	var categories []map[string]interface{}
	key := "knowledge:categories:active"

	ok, err := cs.Get(key, &categories)
	if err != nil || !ok {
		return nil, false
	}

	return categories, true
}

// SetKnowledgeCategoriesCache 设置知识库分类缓存
func (cs *CacheService) SetKnowledgeCategoriesCache(categories []map[string]interface{}) error {
	key := "knowledge:categories:active"
	return cs.Set(key, categories, 1*time.Hour)
}

// ClearKnowledgeCategoriesCache 清除知识库分类缓存
func (cs *CacheService) ClearKnowledgeCategoriesCache() error {
	key := "knowledge:categories:active"
	return cs.Del(key)
}

// GetKnowledgeArticlesCache 获取知识库文章缓存
func (cs *CacheService) GetKnowledgeArticlesCache(categoryID string) ([]map[string]interface{}, bool) {
	var articles []map[string]interface{}
	var key string
	if categoryID == "" {
		key = "knowledge:articles:active"
	} else {
		key = fmt.Sprintf("knowledge:articles:category:%s", categoryID)
	}

	ok, err := cs.Get(key, &articles)
	if err != nil || !ok {
		return nil, false
	}

	return articles, true
}

// SetKnowledgeArticlesCache 设置知识库文章缓存
func (cs *CacheService) SetKnowledgeArticlesCache(categoryID string, articles []map[string]interface{}) error {
	var key string
	if categoryID == "" {
		key = "knowledge:articles:active"
	} else {
		key = fmt.Sprintf("knowledge:articles:category:%s", categoryID)
	}
	return cs.Set(key, articles, 1*time.Hour)
}

// ClearKnowledgeArticlesCache 清除知识库文章缓存
func (cs *CacheService) ClearKnowledgeArticlesCache() error {
	// 清除所有知识库相关缓存
	if err := cs.Del("knowledge:articles:active"); err != nil {
		log.Printf("failed to delete knowledge articles cache: %v", err)
	}
	if err := cs.Del("knowledge:categories:active"); err != nil {
		log.Printf("failed to delete knowledge categories cache: %v", err)
	}
	// 注意：这里无法清除所有分类的缓存，需要在具体操作时清除
	return nil
}

// ==========================================
// 节点列表缓存（保留但不推荐使用 - 节点更新频繁）
// ==========================================

// GetNodesCache 获取节点列表缓存
func (cs *CacheService) GetNodesCache() ([]map[string]interface{}, bool) {
	var nodes []map[string]interface{}
	key := "nodes:list:active"

	ok, err := cs.Get(key, &nodes)
	if err != nil || !ok {
		return nil, false
	}

	return nodes, true
}

// SetNodesCache 设置节点列表缓存
func (cs *CacheService) SetNodesCache(nodes []map[string]interface{}) error {
	key := "nodes:list:active"
	return cs.Set(key, nodes, 5*time.Minute)
}

// ClearNodesCache 清除节点列表缓存
func (cs *CacheService) ClearNodesCache() error {
	key := "nodes:list:active"
	return cs.Del(key)
}

// ==========================================
// 用户订阅缓存
// ==========================================

// GetUserSubscriptionCache 获取用户订阅缓存
func (cs *CacheService) GetUserSubscriptionCache(userID uint) (map[string]interface{}, bool) {
	var sub map[string]interface{}
	key := fmt.Sprintf("user:subscription:%d", userID)

	ok, err := cs.Get(key, &sub)
	if err != nil || !ok {
		return nil, false
	}

	return sub, true
}

// SetUserSubscriptionCache 设置用户订阅缓存
func (cs *CacheService) SetUserSubscriptionCache(userID uint, sub map[string]interface{}) error {
	key := fmt.Sprintf("user:subscription:%d", userID)
	return cs.Set(key, sub, 5*time.Minute)
}

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

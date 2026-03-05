package cache_service

import (
	"cboard-go/internal/core/cache"
	"cboard-go/internal/utils"
)

// FlushAllCache 清空所有 Redis 缓存
func FlushAllCache() error {
	if !cache.IsRedisEnabled() {
		utils.LogInfo("Redis 未启用，跳过缓存清除")
		return nil
	}

	if err := cache.FlushAll(); err != nil {
		utils.LogError("清空 Redis 缓存失败", err, nil)
		return err
	}

	utils.LogInfo("✅ Redis 缓存已全部清空")
	return nil
}

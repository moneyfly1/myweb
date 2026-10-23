package cache_service

import (
	"cboard-go/internal/core/cache"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"
	"log"
	"time"
)

// WarmupCache 预热缓存
func WarmupCache() {
	if !cache.IsRedisEnabled() {
		utils.LogInfo("缓存预热: Redis 未启用，跳过")
		return
	}

	utils.LogInfo("缓存预热: 开始预热缓存...")
	start := time.Now()

	// 只预热真正有读路径的缓存（套餐）：公告/系统配置的查询本身很轻，
	// 且系统配置已有进程内 30s 缓存，再写一份 Redis 无人读取纯属空转。
	go warmupPackages()

	utils.LogInfo("缓存预热: 完成，耗时 %v", time.Since(start))
}

func warmupPackages() {
	cs := NewCacheService()
	// 预热同样是"查库后写缓存"，会与管理员改套餐后的清缓存乱序，
	// 因此先取版本号、写回时校验（与读路径一致）
	generation := cs.PackagesCacheGeneration()

	db := database.GetDB()
	var packages []models.Package
	if err := db.Where("is_active = ?", true).Order("sort_order ASC").Find(&packages).Error; err != nil {
		return
	}

	result := make([]map[string]interface{}, 0)
	for _, pkg := range packages {
		result = append(result, map[string]interface{}{
			"id":             pkg.ID,
			"name":           pkg.Name,
			"description":    pkg.Description.String,
			"price":          pkg.Price,
			"duration_days":  pkg.DurationDays,
			"device_limit":   pkg.DeviceLimit,
			"sort_order":     pkg.SortOrder,
			"is_active":      pkg.IsActive,
			"is_recommended": pkg.IsRecommended,
			"created_at":     utils.FormatBeijingTime(pkg.CreatedAt),
			"updated_at":     utils.FormatBeijingTime(pkg.UpdatedAt),
		})
	}

	if written, err := cs.SetPackagesCacheIfUnchanged(generation, result); err != nil {
		log.Printf("failed to set packages cache: %v", err)
	} else if !written {
		utils.LogInfo("缓存预热: 套餐缓存已被更新，跳过本次写入 (%d 条)", len(result))
		return
	}
	utils.LogInfo("缓存预热: 套餐列表已预热 (%d 条)", len(result))
}

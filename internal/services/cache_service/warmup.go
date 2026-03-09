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

	go warmupPackages()
	go warmupAnnouncements()
	go warmupSystemConfig()

	utils.LogInfo("缓存预热: 完成，耗时 %v", time.Since(start))
}

func warmupPackages() {
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

	cs := NewCacheService()
	if err := cs.SetPackagesCache(result); err != nil {
		log.Printf("failed to set packages cache: %v", err)
	}
	utils.LogInfo("缓存预热: 套餐列表已预热 (%d 条)", len(result))
}

func warmupAnnouncements() {
	db := database.GetDB()
	var announcements []models.Notification
	if err := db.Where("type = ? AND is_active = ?", "announcement", true).
		Order("created_at DESC").Limit(10).Find(&announcements).Error; err != nil {
		return
	}

	result := make([]map[string]interface{}, 0)
	for _, ann := range announcements {
		result = append(result, map[string]interface{}{
			"id":         ann.ID,
			"title":      ann.Title,
			"content":    ann.Content,
			"type":       ann.Type,
			"is_active":  ann.IsActive,
			"created_at": utils.FormatBeijingTime(ann.CreatedAt),
		})
	}

	cs := NewCacheService()
	if err := cs.SetAnnouncementsCache(result); err != nil {
		log.Printf("failed to set announcements cache: %v", err)
	}
	utils.LogInfo("缓存预热: 公告列表已预热 (%d 条)", len(result))
}

func warmupSystemConfig() {
	db := database.GetDB()
	categories := []string{"general", "payment", "email", "security"}

	cs := NewCacheService()
	for _, category := range categories {
		var configs []models.SystemConfig
		if err := db.Where("category = ?", category).Find(&configs).Error; err != nil {
			continue
		}

		result := make([]map[string]interface{}, 0)
		for _, cfg := range configs {
			result = append(result, map[string]interface{}{
				"id":       cfg.ID,
				"key":      cfg.Key,
				"value":    cfg.Value,
				"category": cfg.Category,
			})
		}

		if err := cs.SetSystemConfigCache(category, result); err != nil {
			log.Printf("failed to set system config cache for category %s: %v", category, err)
		}
	}
	utils.LogInfo("缓存预热: 系统配置已预热")
}

// 软件库自动同步（GitHub 版本检测）管理接口。
// 分发方式为「GitHub Releases + 国内加速镜像直链」（/download/gh），
// 本模块仅负责：定时版本检测、版本对照展示、手动触发检测。
package handlers

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/ghrelease"
	"cboard-go/internal/services/software_sync"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// cloudCategory 同步配置存储分类（沿用旧库分类名，兼容历史数据）
const cloudCategory = "pan123"

type cloudConfig struct {
	SyncEnabled       bool
	SyncIntervalHours int
}

func loadCloudConfig() (cloudConfig, error) {
	var cfg cloudConfig
	cfg.SyncEnabled = true
	cfg.SyncIntervalHours = 12
	db := database.GetDB()
	var configs []models.SystemConfig
	if err := db.Where("category = ?", cloudCategory).Find(&configs).Error; err != nil {
		return cfg, err
	}
	for _, c := range configs {
		switch c.Key {
		case "sync_enabled":
			cfg.SyncEnabled = c.Value == "" || c.Value == "true" || c.Value == "1"
		case "sync_interval_hours":
			if v, err2 := strconv.Atoi(c.Value); err2 == nil && v >= 1 {
				cfg.SyncIntervalHours = v
			}
		}
	}
	return cfg, nil
}

func saveCloudConfigValue(key, value string) error {
	db := database.GetDB()
	var conf models.SystemConfig
	if err := db.Where("key = ? AND category = ?", key, cloudCategory).FirstOrInit(&conf).Error; err != nil {
		return err
	}
	conf.Key = key
	conf.Category = cloudCategory
	conf.Value = value
	conf.Type = "text"
	return db.Save(&conf).Error
}

// GetCloudConfig 读取同步配置
func GetCloudConfig(c *gin.Context) {
	cfg, err := loadCloudConfig()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取配置失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"sync_enabled":        cfg.SyncEnabled,
		"sync_interval_hours": cfg.SyncIntervalHours,
	})
}

// SaveCloudConfig 保存同步配置
func SaveCloudConfig(c *gin.Context) {
	var req struct {
		SyncEnabled       *bool `json:"sync_enabled"`
		SyncIntervalHours *int  `json:"sync_interval_hours"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}
	if req.SyncEnabled != nil {
		_ = saveCloudConfigValue("sync_enabled", strconv.FormatBool(*req.SyncEnabled))
	}
	if req.SyncIntervalHours != nil && *req.SyncIntervalHours >= 1 && *req.SyncIntervalHours <= 168 {
		_ = saveCloudConfigValue("sync_interval_hours", strconv.Itoa(*req.SyncIntervalHours))
	}
	utils.InvalidateAllSettingCache()
	utils.CreateAuditLogSimple(c, "cloud_config_save", "settings", 0, "管理员操作: 保存软件库自动同步配置")
	utils.SuccessResponse(c, http.StatusOK, "保存成功", nil)
}

// CloudSync 触发一次版本检测（异步执行，前端轮询状态）
func CloudSync(c *gin.Context) {
	if !software_sync.TriggerAsync() {
		utils.SuccessResponse(c, http.StatusOK, "检测任务正在进行中", gin.H{"started": false, "running": true})
		return
	}
	utils.CreateAuditLogSimple(c, "cloud_sync", "settings", 0, "管理员操作: 手动触发 GitHub 软件库版本检测")
	utils.SuccessResponse(c, http.StatusOK, "检测已开始，请稍候查看结果", gin.H{"started": true, "running": true})
}

// CloudSyncStatus 同步状态
func CloudSyncStatus(c *gin.Context) {
	utils.SuccessResponse(c, http.StatusOK, "", software_sync.GetStatus())
}

// GitHub 版本对照缓存（30 分钟，避免每次打开页面都打 GitHub API）
type ghVersionCacheEntry struct {
	Version string
	Expire  time.Time
}

var ghVersionCache sync.Map // repo → ghVersionCacheEntry

func cachedGitHubVersion(repo, token string) (string, bool) {
	if v, ok := ghVersionCache.Load(repo); ok {
		e := v.(ghVersionCacheEntry)
		if time.Now().Before(e.Expire) {
			return e.Version, true
		}
		ghVersionCache.Delete(repo)
	}
	release, err := ghrelease.Latest(repo, loadDownloadProxyPrefixes(), token)
	if err != nil {
		return "", false
	}
	ver := release.Version()
	ghVersionCache.Store(repo, ghVersionCacheEntry{Version: ver, Expire: time.Now().Add(30 * time.Minute)})
	return ver, true
}

// CloudVersions 版本对照：GitHub 最新版本 vs 已检出版本
func CloudVersions(c *gin.Context) {
	fileIDMap, _ := software_sync.LoadFileIDMap()
	token := software_sync.LoadGitHubToken()
	softwareConfig := getConfigMap("software")

	rows := make([]gin.H, 0)
	for _, sw := range software_sync.Catalog {
		ghVersion := ""
		ghErr := ""
		if v, ok := cachedGitHubVersion(sw.Repo, token); ok {
			ghVersion = v
		} else {
			ghErr = "GitHub 版本获取失败（可能限流，稍后自动重试）"
		}
		for _, t := range sw.Targets {
			row := gin.H{
				"key":            t.ConfigKey,
				"name":           sw.Name,
				"label":          t.Label,
				"os":             t.OS,
				"arch":           t.Arch,
				"github_version": ghVersion,
				"cloud_version":  "",
				"file_name":      "",
				"synced":         false,
				"custom":         false,
				"gh_error":       ghErr,
			}
			// 配置了手工自定义外部链接的入口：不使用自动分发，明确标注
			if current := strings.TrimSpace(softwareConfig[t.ConfigKey]); current != "" && !strings.HasPrefix(current, "pan://") {
				row["custom"] = true
				row["file_name"] = current
				rows = append(rows, row)
				continue
			}
			if entry, ok := fileIDMap[t.ConfigKey]; ok && entry.Version != "" {
				row["cloud_version"] = entry.Version
				row["file_name"] = entry.FileName
				row["synced"] = entry.Version == ghVersion
			}
			rows = append(rows, row)
		}
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"configured": true,
		"list":       rows,
	})
}

// GetSoftwareVersions 用户端：已检出软件版本信息（公共接口，供下载页展示）
func GetSoftwareVersions(c *gin.Context) {
	fileIDMap, err := software_sync.LoadFileIDMap()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取版本信息失败", err)
		return
	}
	out := make([]gin.H, 0)
	for key, entry := range fileIDMap {
		if entry.Version == "" {
			continue
		}
		out = append(out, gin.H{
			"key":        key,
			"version":    entry.Version,
			"file_name":  entry.FileName,
			"size":       entry.Size,
			"size_text":  formatFileSize(entry.Size),
			"updated_at": entry.UpdatedAt,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i]["key"].(string) < out[j]["key"].(string)
	})
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"list": out})
}

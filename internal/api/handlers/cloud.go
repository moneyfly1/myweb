package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/aliyundrive"
	"cboard-go/internal/services/ghrelease"
	"cboard-go/internal/services/software_sync"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

const (
	cloudCategory = "pan123" // 沿用旧库分类名（兼容历史数据），仅阿里云盘后端使用
	cloudLinkTTL  = 10 * time.Minute
)

func init() {
	// 同步完成后清理直链缓存，避免旧链接在缓存期内继续生效
	software_sync.SetOnSyncComplete(clearCloudLinkCache)
}

// 公共解析接口全局限流：防止滥用（每次请求会消耗云盘接口额度）
var resolveGate = struct {
	mu   sync.Mutex
	last time.Time
}{}

func maskIfNonEmpty(v string) string {
	if strings.TrimSpace(v) == "" {
		return ""
	}
	return maskedSecretValue
}

func resolveThrottled() bool {
	resolveGate.mu.Lock()
	defer resolveGate.mu.Unlock()
	if time.Since(resolveGate.last) < 200*time.Millisecond {
		return false
	}
	resolveGate.last = time.Now()
	return true
}

// ---------------------------------------------------------------------------
// 配置读写
// ---------------------------------------------------------------------------

type cloudConfig struct {
	AliyunRefreshToken string
	SyncEnabled        bool
	SyncIntervalHours  int
	Folder             string // 云盘上传目录（/ 分隔多级；空 = 根目录）
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
		case "aliyun_refresh_token":
			cfg.AliyunRefreshToken = strings.TrimSpace(c.Value)
		case "sync_enabled":
			cfg.SyncEnabled = c.Value == "" || c.Value == "true" || c.Value == "1"
		case "sync_interval_hours":
			if v, err2 := strconv.Atoi(c.Value); err2 == nil && v >= 1 {
				cfg.SyncIntervalHours = v
			}
		case "aliyun_folder":
			cfg.Folder = strings.TrimSpace(c.Value)
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

// GetCloudConfig 获取云盘配置（refresh_token 脱敏）
func GetCloudConfig(c *gin.Context) {
	cfg, err := loadCloudConfig()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取配置失败", err)
		return
	}
	fileIDMap, _ := software_sync.LoadFileIDMap()
	syncedFiles := map[string]string{}
	for key, entry := range fileIDMap {
		if entry.AliyunFileID != "" {
			syncedFiles[key] = entry.FileName
		}
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"aliyun_refresh_token": maskIfNonEmpty(cfg.AliyunRefreshToken),
		"aliyun_folder":        cfg.Folder,
		"sync_enabled":         cfg.SyncEnabled,
		"sync_interval_hours":  cfg.SyncIntervalHours,
		"synced_files":         syncedFiles,
		"configured":           cfg.AliyunRefreshToken != "",
	})
}

// SaveCloudConfig 保存云盘配置
func SaveCloudConfig(c *gin.Context) {
	var req struct {
		AliyunRefreshToken string `json:"aliyun_refresh_token"`
		AliyunFolder       string `json:"aliyun_folder"`
		SyncEnabled        *bool  `json:"sync_enabled"`
		SyncIntervalHours  *int   `json:"sync_interval_hours"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}
	cfg, _ := loadCloudConfig()

	rt := strings.TrimSpace(req.AliyunRefreshToken)
	if rt == maskedSecretValue {
		rt = cfg.AliyunRefreshToken
	}
	if err := saveCloudConfigValue("aliyun_refresh_token", rt); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "保存失败", err)
		return
	}
	folder := strings.Trim(strings.TrimSpace(req.AliyunFolder), "/")
	_ = saveCloudConfigValue("aliyun_folder", folder)
	if req.SyncEnabled != nil {
		_ = saveCloudConfigValue("sync_enabled", strconv.FormatBool(*req.SyncEnabled))
	}
	if req.SyncIntervalHours != nil && *req.SyncIntervalHours >= 1 && *req.SyncIntervalHours <= 168 {
		_ = saveCloudConfigValue("sync_interval_hours", strconv.Itoa(*req.SyncIntervalHours))
	}
	utils.InvalidateAllSettingCache()
	utils.CreateAuditLogSimple(c, "cloud_config_save", "settings", 0, "管理员操作: 保存阿里云盘自动填充配置")
	utils.SuccessResponse(c, http.StatusOK, "保存成功", nil)
}

func newAliyunClientFromConfig(cfg cloudConfig) (*aliyundrive.Client, error) {
	if cfg.AliyunRefreshToken == "" {
		return nil, fmt.Errorf("未配置阿里云盘 refresh_token")
	}
	return aliyundrive.New(cfg.AliyunRefreshToken), nil
}

// ---------------------------------------------------------------------------
// 阿里云盘连接测试 / 搜索
// ---------------------------------------------------------------------------

// CloudAliyunTest 测试阿里云盘 refresh_token（刷新 + 列根目录；自动轮换保存新 token）
func CloudAliyunTest(c *gin.Context) {
	cfg, err := loadCloudConfig()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取配置失败", err)
		return
	}
	ali, cerr := newAliyunClientFromConfig(cfg)
	if cerr != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, cerr.Error(), nil)
		return
	}
	newRT, rerr := ali.Refresh()
	if rerr != nil {
		utils.ErrorResponse(c, http.StatusBadGateway, "刷新 token 失败: "+rerr.Error(), nil)
		return
	}
	if newRT != "" && newRT != cfg.AliyunRefreshToken {
		if err := saveCloudConfigValue("aliyun_refresh_token", newRT); err == nil {
			cfg.AliyunRefreshToken = newRT
		}
	}
	// 解析上传目录（不存在会自动创建），未配置则列根目录
	folderID := "root"
	folderName := "根目录"
	if cfg.Folder != "" {
		fid, derr := ali.EnsureDir(cfg.Folder)
		if derr != nil {
			utils.ErrorResponse(c, http.StatusBadGateway, "连接成功但解析上传目录失败: "+derr.Error(), nil)
			return
		}
		folderID = fid
		folderName = cfg.Folder
	}
	files, lerr := ali.List(folderID, 5)
	if lerr != nil {
		utils.ErrorResponse(c, http.StatusBadGateway, "连接成功但列目录失败: "+lerr.Error(), nil)
		return
	}
	firstName := ""
	if len(files) > 0 {
		firstName = files[0].Name
	}
	utils.SuccessResponse(c, http.StatusOK, "阿里云盘连接成功", gin.H{
		"refresh_token_rotated": newRT != "",
		"folder":                folderName,
		"root_files":            len(files),
		"first":                 firstName,
	})
}

// CloudAliyunSearch 搜索阿里云盘文件
func CloudAliyunSearch(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	if keyword == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "缺少 keyword 参数", nil)
		return
	}
	cfg, err := loadCloudConfig()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取配置失败", err)
		return
	}
	ali, cerr := newAliyunClientFromConfig(cfg)
	if cerr != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, cerr.Error(), nil)
		return
	}
	files, serr := ali.Search(keyword, 20)
	if serr != nil {
		utils.ErrorResponse(c, http.StatusBadGateway, "搜索失败: "+serr.Error(), nil)
		return
	}
	list := make([]gin.H, 0, len(files))
	for _, f := range files {
		list = append(list, gin.H{
			"file_id":   f.FileID,
			"file_name": f.Name,
			"size":      f.Size,
			"size_text": f.DisplaySize(),
			"type":      f.Type,
			"update_at": f.UpdatedAt,
		})
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"list": list, "total": len(list)})
}

// ---------------------------------------------------------------------------
// 动态解析：pan://<配置键> → 阿里云盘直链
// ---------------------------------------------------------------------------

type cloudLinkCacheEntry struct {
	URL    string
	Expire time.Time
}

var cloudLinkCache sync.Map // key → entry

func getCachedCloudLink(key string) (string, bool) {
	if v, ok := cloudLinkCache.Load(key); ok {
		e := v.(cloudLinkCacheEntry)
		if time.Now().Before(e.Expire) {
			return e.URL, true
		}
		cloudLinkCache.Delete(key)
	}
	return "", false
}

func setCachedCloudLink(key, rawURL string) {
	cloudLinkCache.Store(key, cloudLinkCacheEntry{URL: rawURL, Expire: time.Now().Add(cloudLinkTTL)})
}

func clearCloudLinkCache() {
	cloudLinkCache.Range(func(k, _ interface{}) bool {
		cloudLinkCache.Delete(k)
		return true
	})
}

// CloudResolve 根据软件配置 key 实时生成阿里云盘直链并 302 跳转。
// 支持 ?key=<softwareConfigKey> 或 ?q=<关键词>
func CloudResolve(c *gin.Context) {
	if !resolveThrottled() {
		utils.ErrorResponse(c, http.StatusTooManyRequests, "请求过于频繁，请稍后再试", nil)
		return
	}
	cfg, err := loadCloudConfig()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取配置失败", err)
		return
	}
	if cfg.AliyunRefreshToken == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "管理员尚未配置阿里云盘", nil)
		return
	}
	ali, cerr := newAliyunClientFromConfig(cfg)
	if cerr != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, cerr.Error(), nil)
		return
	}

	keyword := ""
	rawKey := ""
	if key := strings.TrimSpace(c.Query("key")); key != "" {
		rawKey = key
		if entry, ok := cloudFileIDMapOf(key); ok && entry.AliyunFileID != "" {
			cacheKey := "ali:" + entry.AliyunFileID
			if cached, ok := getCachedCloudLink(cacheKey); ok {
				if validateDownloadURL(cached) == nil {
					c.Redirect(http.StatusFound, cached)
					return
				}
				cloudLinkCache.Delete(cacheKey)
			}
			link, lerr := ali.DownloadURL(entry.AliyunFileID)
			if lerr == nil {
				if verr := validateDownloadURL(link); verr == nil {
					setCachedCloudLink(cacheKey, link)
					c.Redirect(http.StatusFound, link)
					return
				}
			}
		}
	}
	if keyword == "" {
		keyword = strings.TrimSpace(c.Query("q"))
	}
	if keyword == "" {
		keyword = rawKey
	}
	if keyword == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "缺少 key 或 q 参数", nil)
		return
	}

	// 按关键词搜索云盘（兜底）
	files, serr := ali.Search(keyword, 10)
	if serr != nil {
		utils.ErrorResponse(c, http.StatusBadGateway, "云盘搜索失败: "+serr.Error(), nil)
		return
	}
	var target *aliyundrive.File
	for i := range files {
		if files[i].Type == "file" {
			target = &files[i]
			break
		}
	}
	if target == nil {
		utils.ErrorResponse(c, http.StatusNotFound, "未找到匹配的文件", nil)
		return
	}
	cacheKey := "ali:" + target.FileID
	if cached, ok := getCachedCloudLink(cacheKey); ok {
		if validateDownloadURL(cached) == nil {
			c.Redirect(http.StatusFound, cached)
			return
		}
		cloudLinkCache.Delete(cacheKey)
	}
	link, lerr := ali.DownloadURL(target.FileID)
	if lerr != nil {
		utils.ErrorResponse(c, http.StatusBadGateway, "生成下载链接失败: "+lerr.Error(), nil)
		return
	}
	if verr := validateDownloadURL(link); verr != nil {
		utils.ErrorResponse(c, http.StatusBadGateway, "生成的下载链接不合法: "+verr.Error(), nil)
		return
	}
	setCachedCloudLink(cacheKey, link)
	c.Redirect(http.StatusFound, link)
}

func cloudFileIDMapOf(key string) (software_sync.FileEntry, bool) {
	m, err := software_sync.LoadFileIDMap()
	if err != nil {
		return software_sync.FileEntry{}, false
	}
	e, ok := m[key]
	return e, ok
}

// GitHub 版本对照缓存（30 分钟，避免每次打开页面都打 GitHub API）
type ghVersionCacheEntry struct {
	Version string
	Expire  time.Time
}

var ghVersionCache sync.Map // repo → entry

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

// ---------------------------------------------------------------------------
// 软件库同步（GitHub → 阿里云盘）
// ---------------------------------------------------------------------------

// CloudSync 触发一次软件库同步（异步执行，前端轮询状态）
func CloudSync(c *gin.Context) {
	if !software_sync.TriggerAsync() {
		utils.SuccessResponse(c, http.StatusOK, "同步任务正在进行中", gin.H{"started": false, "running": true})
		return
	}
	utils.CreateAuditLogSimple(c, "cloud_sync", "settings", 0, "管理员操作: 手动触发 GitHub→阿里云盘 软件库同步")
	utils.SuccessResponse(c, http.StatusOK, "同步已开始，请稍候查看结果", gin.H{"started": true, "running": true})
}

// CloudSyncStatus 同步状态
func CloudSyncStatus(c *gin.Context) {
	utils.SuccessResponse(c, http.StatusOK, "", software_sync.GetStatus())
}

// CloudVersions 版本对照：GitHub 最新版本 vs 云盘已同步版本
func CloudVersions(c *gin.Context) {
	cfg, err := loadCloudConfig()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取配置失败", err)
		return
	}
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
			// 配置了手工自定义外部链接的入口：不使用云盘，明确标注
			if current := strings.TrimSpace(softwareConfig[t.ConfigKey]); current != "" && !strings.HasPrefix(current, "pan://") {
				row["custom"] = true
				row["file_name"] = current
				rows = append(rows, row)
				continue
			}
			if entry, ok := fileIDMap[t.ConfigKey]; ok && entry.AliyunFileID != "" {
				row["cloud_version"] = entry.Version
				row["file_name"] = entry.FileName
				row["synced"] = entry.Version == ghVersion
			}
			rows = append(rows, row)
		}
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"configured": cfg.AliyunRefreshToken != "",
		"list":       rows,
	})
}

// GetSoftwareVersions 用户端：已同步软件版本信息（公共接口，供下载页展示）
func GetSoftwareVersions(c *gin.Context) {
	fileIDMap, err := software_sync.LoadFileIDMap()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取版本信息失败", err)
		return
	}
	out := make([]gin.H, 0)
	for key, entry := range fileIDMap {
		if entry.AliyunFileID == "" {
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

var _ = errors.New
var _ = json.Marshal
var _ = models.SystemConfig{}
var _ = database.GetDB

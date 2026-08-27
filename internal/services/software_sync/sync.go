package software_sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/aliyundrive"
	"cboard-go/internal/services/ghrelease"
)

// cfgCategory 云盘配置存储分类（沿用旧库分类名，避免迁移；与 123 云盘已无任何关联）
const cfgCategory = "pan123"

// FileEntry 云盘文件映射条目（存于 file_id_map 配置）
type FileEntry struct {
	// FileId 旧版（123 云盘）遗留字段，已不使用
	FileId    int64  `json:"fileId,omitempty"`
	FileName  string `json:"fileName"`
	Size      int64  `json:"size"`
	Version   string `json:"version"`
	UpdatedAt string `json:"updatedAt"`
	// AliyunFileID 阿里云盘文件 id（当前唯一存储后端）
	AliyunFileID string `json:"aliyunFileId,omitempty"`
}

// HasMeta 是否已具备可用文件 id
func (e FileEntry) HasMeta() bool {
	return e.AliyunFileID != ""
}

// ReportItem 单目标同步结果
type ReportItem struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Label    string `json:"label"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
	FileName string `json:"file_name,omitempty"`
	Status   string `json:"status"` // ok / skip / error
	Message  string `json:"message,omitempty"`
}

// SyncStatus 同步状态（供管理接口/前端轮询）
type SyncStatus struct {
	Running       bool         `json:"running"`
	Enabled       bool         `json:"enabled"`
	IntervalHours int          `json:"interval_hours"`
	LastRun       string       `json:"last_run"`
	LastReport    []ReportItem `json:"last_report"`
	TotalUploaded int          `json:"total_uploaded"`
}

var (
	statusMu     sync.Mutex
	syncRunning  bool
	lastRun      string
	lastReport   []ReportItem
	lastUploaded int

	// OnSyncComplete 同步结束后回调（由 handlers 注册用于清理直链缓存等）
	OnSyncComplete func()
)

// SetOnSyncComplete 注册同步完成回调（幂等，重复注册以最后一次为准）
func SetOnSyncComplete(fn func()) {
	statusMu.Lock()
	defer statusMu.Unlock()
	OnSyncComplete = fn
}

func fireOnSyncComplete() {
	statusMu.Lock()
	fn := OnSyncComplete
	statusMu.Unlock()
	if fn != nil {
		defer func() {
			if r := recover(); r != nil {
				// 回调异常不影响同步结果
			}
		}()
		fn()
	}
}

// ---------------------------------------------------------------------------
// 配置读写
// ---------------------------------------------------------------------------

type syncConfig struct {
	Enabled            bool
	Interval           time.Duration
	AliyunRefreshToken string
}

func loadSyncConfig() (syncConfig, error) {
	var cfg syncConfig
	cfg.Enabled = true
	cfg.Interval = 12 * time.Hour
	db := database.GetDB()
	var configs []models.SystemConfig
	if err := db.Where("category = ?", cfgCategory).Find(&configs).Error; err != nil {
		return cfg, err
	}
	for _, c := range configs {
		switch c.Key {
		case "sync_enabled":
			cfg.Enabled = c.Value == "" || c.Value == "true" || c.Value == "1"
		case "sync_interval_hours":
			var h int
			if n, _ := fmt.Sscanf(c.Value, "%d", &h); n == 1 && h >= 1 {
				cfg.Interval = time.Duration(h) * time.Hour
			}
		case "aliyun_refresh_token":
			cfg.AliyunRefreshToken = strings.TrimSpace(c.Value)
		}
	}
	return cfg, nil
}

// LoadGitHubToken 复用备份功能的 GitHub token（若已配置），提高 API 限额（5000 次/小时）
func LoadGitHubToken() string {
	db := database.GetDB()
	var conf models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "backup_github_token", "backup").First(&conf).Error; err != nil {
		return ""
	}
	return strings.TrimSpace(conf.Value)
}

func loadProxyPrefixes() []string {
	db := database.GetDB()
	var conf models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "download_proxy_prefixes", "software").First(&conf).Error; err != nil {
		return ghrelease.DefaultProxyPrefixes()
	}
	var prefixes []string
	if err := json.Unmarshal([]byte(conf.Value), &prefixes); err != nil || len(prefixes) == 0 {
		prefixes = strings.FieldsFunc(conf.Value, func(r rune) bool { return r == '\n' || r == ',' })
	}
	if len(prefixes) == 0 {
		return ghrelease.DefaultProxyPrefixes()
	}
	return prefixes
}

// LoadFileIDMap 读取云盘文件映射
func LoadFileIDMap() (map[string]FileEntry, error) {
	out := map[string]FileEntry{}
	db := database.GetDB()
	var conf models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "file_id_map", cfgCategory).First(&conf).Error; err != nil {
		return out, nil // 不存在则空
	}
	if strings.TrimSpace(conf.Value) != "" {
		_ = json.Unmarshal([]byte(conf.Value), &out)
	}
	return out, nil
}

// SaveFileIDMap 保存云盘文件映射
func SaveFileIDMap(m map[string]FileEntry) error {
	b, _ := json.Marshal(m)
	db := database.GetDB()
	var conf models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "file_id_map", cfgCategory).FirstOrInit(&conf).Error; err != nil {
		return err
	}
	conf.Key = "file_id_map"
	conf.Category = cfgCategory
	conf.Value = string(b)
	conf.Type = "text"
	return db.Save(&conf).Error
}

func saveCfgValue(key, value string) error {
	db := database.GetDB()
	var conf models.SystemConfig
	if err := db.Where("key = ? AND category = ?", key, cfgCategory).FirstOrInit(&conf).Error; err != nil {
		return err
	}
	conf.Key = key
	conf.Category = cfgCategory
	conf.Value = value
	conf.Type = "text"
	return db.Save(&conf).Error
}

func loadSoftwareValue(key string) string {
	db := database.GetDB()
	var conf models.SystemConfig
	if err := db.Where("key = ? AND category = ?", key, "software").First(&conf).Error; err != nil {
		return ""
	}
	return conf.Value
}

func saveSoftwareValue(key, value string) error {
	db := database.GetDB()
	var conf models.SystemConfig
	if err := db.Where("key = ? AND category = ?", key, "software").FirstOrInit(&conf).Error; err != nil {
		return err
	}
	conf.Key = key
	conf.Category = "software"
	conf.Value = value
	conf.Type = "text"
	return db.Save(&conf).Error
}

// GetStatus 当前同步状态（内存优先，重启后从数据库回退读取上次运行信息）
func GetStatus() SyncStatus {
	cfg, _ := loadSyncConfig()
	statusMu.Lock()
	report := make([]ReportItem, len(lastReport))
	copy(report, lastReport)
	runAt := lastRun
	uploaded := lastUploaded
	statusMu.Unlock()

	if runAt == "" {
		runAt = LastSyncAt().Format(time.RFC3339)
	}
	if len(report) == 0 {
		// 从数据库回退读取上次报告
		db := database.GetDB()
		var conf models.SystemConfig
		if err := db.Where("key = ? AND category = ?", "sync_last_report", cfgCategory).First(&conf).Error; err == nil {
			_ = json.Unmarshal([]byte(conf.Value), &report)
		}
	}
	return SyncStatus{
		Running:       IsRunning(),
		Enabled:       cfg.Enabled,
		IntervalHours: int(cfg.Interval.Hours()),
		LastRun:       runAt,
		LastReport:    report,
		TotalUploaded: uploaded,
	}
}

// LastSyncAt 上次同步时间（供调度器判断是否到期）
func LastSyncAt() time.Time {
	db := database.GetDB()
	var conf models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "sync_last_run", cfgCategory).First(&conf).Error; err != nil {
		return time.Time{}
	}
	t, err := time.ParseInLocation(time.RFC3339, conf.Value, time.Local)
	if err != nil {
		return time.Time{}
	}
	return t
}

// Due 判断是否到定时同步时间（开启、至少手动/定时运行过一次、且距上次运行超过间隔）。
// 从未运行过时返回 false，避免服务器重启后自动触发全量同步（可能数 GB）。
func Due() bool {
	cfg, err := loadSyncConfig()
	if err != nil || !cfg.Enabled {
		return false
	}
	last := LastSyncAt()
	if last.IsZero() {
		return false
	}
	return time.Since(last) >= cfg.Interval
}

// IsRunning 是否正在同步
func IsRunning() bool {
	statusMu.Lock()
	defer statusMu.Unlock()
	return syncRunning
}

// TriggerAsync 后台触发一次同步（已在运行则忽略）
func TriggerAsync() bool {
	return triggerAsync(nil)
}

func triggerAsync(only []string) bool {
	statusMu.Lock()
	if syncRunning {
		statusMu.Unlock()
		return false
	}
	syncRunning = true
	statusMu.Unlock()
	go func() {
		report, uploaded := run(only)
		statusMu.Lock()
		syncRunning = false
		lastRun = time.Now().Format(time.RFC3339)
		lastReport = report
		lastUploaded = uploaded
		statusMu.Unlock()
		fireOnSyncComplete()
	}()
	return true
}

// RunSync 前台执行同步（only 非空时仅同步指定配置键，用于测试/按需同步）
func RunSync(only []string) ([]ReportItem, error) {
	if IsRunning() {
		return nil, fmt.Errorf("同步任务正在进行中")
	}
	report, _ := run(only)
	return report, nil
}

// ---------------------------------------------------------------------------
// 同步主流程（GitHub 最新版 → 下载 → 上传阿里云盘 → 设置动态直链）
// ---------------------------------------------------------------------------

func run(only []string) ([]ReportItem, int) {
	report := make([]ReportItem, 0)
	uploaded := 0
	onlySet := map[string]bool{}
	for _, k := range only {
		onlySet[k] = true
	}

	cfg, err := loadSyncConfig()
	if err != nil {
		return append(report, ReportItem{Status: "error", Message: "读取配置失败: " + err.Error()}), 0
	}
	if cfg.AliyunRefreshToken == "" {
		return append(report, ReportItem{Status: "error", Message: "未配置阿里云盘 refresh_token，无法同步"}), 0
	}
	ali := aliyundrive.New(cfg.AliyunRefreshToken)

	fileIDMap, err := LoadFileIDMap()
	if err != nil {
		return append(report, ReportItem{Status: "error", Message: "读取文件映射失败: " + err.Error()}), 0
	}
	prefixes := loadProxyPrefixes()
	ghToken := LoadGitHubToken()
	releaseCache := map[string]*ghrelease.Release{}
	tmpDir, err := os.MkdirTemp("", "cboard-sync-*")
	if err != nil {
		return append(report, ReportItem{Status: "error", Message: "创建临时目录失败: " + err.Error()}), 0
	}
	defer os.RemoveAll(tmpDir)

	for _, sw := range Catalog {
		release, ok := releaseCache[sw.Repo]
		if !ok {
			release, err = ghrelease.Latest(sw.Repo, prefixes, ghToken)
			if err != nil {
				for _, t := range sw.Targets {
					report = append(report, ReportItem{Key: t.ConfigKey, Name: sw.Name, Label: t.Label, OS: t.OS, Arch: t.Arch, Status: "error", Message: "获取 GitHub 版本失败: " + err.Error()})
				}
				continue
			}
			releaseCache[sw.Repo] = release
		}
		version := release.Version()

		for _, t := range sw.Targets {
			if len(onlySet) > 0 && !onlySet[t.ConfigKey] {
				continue
			}
			item := ReportItem{Key: t.ConfigKey, Name: sw.Name, Label: t.Label, OS: t.OS, Arch: t.Arch, Version: version}

			// 该入口配置了手工自定义外部链接（非 pan://）→ 不使用云盘，整项跳过
			if current := loadSoftwareValue(t.ConfigKey); current != "" && !strings.HasPrefix(current, "pan://") {
				item.Status = "skip"
				item.Message = "该入口使用自定义链接，跳过云盘同步"
				report = append(report, item)
				continue
			}

			asset, aerr := findAsset(release, &t)
			if aerr != nil {
				item.Status = "error"
				item.Message = aerr.Error()
				report = append(report, item)
				continue
			}

			// 版本与匹配到的安装包名都一致才跳过（同版本更换构建/命名时仍会更新）
			entry, hasEntry := fileIDMap[t.ConfigKey]
			if hasEntry && entry.AliyunFileID != "" && entry.Version == version && entry.FileName == asset.Name {
				item.Status = "skip"
				item.FileName = entry.FileName
				item.Message = "已是最新版本"
				report = append(report, item)
				continue
			}

			localPath := filepath.Join(tmpDir, asset.Name)
			if derr := ghrelease.Download(asset, localPath, prefixes, ghToken); derr != nil {
				item.Status = "error"
				item.Message = "下载失败: " + derr.Error()
				report = append(report, item)
				continue
			}

			aliFileID, uerr := ali.Upload(localPath, asset.Name, "root")
			if uerr != nil {
				item.Status = "error"
				item.Message = "上传阿里云盘失败: " + uerr.Error()
				report = append(report, item)
				continue
			}

			newEntry := FileEntry{
				AliyunFileID: aliFileID,
				FileName:     asset.Name,
				Size:         asset.Size,
				Version:      version,
				UpdatedAt:    time.Now().Format(time.RFC3339),
			}
			fileIDMap[t.ConfigKey] = newEntry
			uploaded++

			// 更新软件下载配置：动态链接 pan://<配置键>
			if err := ensurePanManagedURL(t.ConfigKey); err != nil {
				item.Message = "上传成功，但更新下载链接失败: " + err.Error()
			} else {
				item.Message = "已上传阿里云盘并设置直链"
			}
			// 有新版本时删除云盘中的旧版本文件
			if hasEntry && entry.AliyunFileID != "" && entry.AliyunFileID != aliFileID {
				if terr := ali.Trash(entry.AliyunFileID); terr != nil {
					item.Message = "新版本已上传，但删除旧版本失败: " + terr.Error()
				} else {
					item.Message = "已上传阿里云盘并替换删除旧版本"
				}
			}
			item.Status = "ok"
			item.FileName = asset.Name
			report = append(report, item)
		}
	}

	if err := SaveFileIDMap(fileIDMap); err != nil {
		report = append(report, ReportItem{Status: "error", Message: "保存文件映射失败: " + err.Error()})
	}
	_ = saveCfgValue("sync_last_run", time.Now().Format(time.RFC3339))
	if b, err := json.Marshal(report); err == nil {
		_ = saveCfgValue("sync_last_report", string(b))
	}
	sort.SliceStable(report, func(i, j int) bool {
		if report[i].Name != report[j].Name {
			return report[i].Name < report[j].Name
		}
		return report[i].Label < report[j].Label
	})
	return report, uploaded
}

// ensurePanManagedURL 若该键当前为空或已是 pan://，则写入 pan://<键>；手工外部链接不覆盖
func ensurePanManagedURL(configKey string) error {
	current := loadSoftwareValue(configKey)
	if current != "" && !strings.HasPrefix(current, "pan://") {
		return nil // 管理员手填的外部链接，不覆盖
	}
	return saveSoftwareValue(configKey, "pan://"+configKey)
}

func findAsset(release *ghrelease.Release, t *Target) (*ghrelease.Asset, error) {
	if len(t.Preferred) > 0 {
		if a := findAssetWithPref(release, t.Preferred); a != nil {
			return a, nil
		}
	}
	return release.FindAsset(t.Patterns)
}

// FindAssetFor 供外部（校验/测试）按目标匹配资产
func FindAssetFor(release *ghrelease.Release, t *Target) (*ghrelease.Asset, error) {
	return findAsset(release, t)
}

// findAssetWithPref 优先匹配 preferred；同架构下跳过 fdroid 构建（如 v2rayNG 常规版优先）
func findAssetWithPref(release *ghrelease.Release, preferred []*regexp.Regexp) *ghrelease.Asset {
	var fdroidFallback *ghrelease.Asset
	for i := range release.Assets {
		for _, p := range preferred {
			if !p.MatchString(release.Assets[i].Name) {
				continue
			}
			if !strings.Contains(strings.ToLower(release.Assets[i].Name), "fdroid") {
				return &release.Assets[i]
			}
			if fdroidFallback == nil {
				fdroidFallback = &release.Assets[i]
			}
		}
	}
	return fdroidFallback
}

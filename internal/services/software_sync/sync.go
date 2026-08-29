package software_sync

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/ghrelease"
)

// cfgCategory 云盘配置存储分类（沿用旧库分类名，避免迁移；与 123 云盘已无任何关联）
const cfgCategory = "pan123"

// FileEntry 版本记录条目（存于 file_id_map 配置，记录各目标已检出的最新版本与资产名）
type FileEntry struct {
	FileName  string `json:"fileName"`
	Size      int64  `json:"size"`
	Version   string `json:"version"`
	UpdatedAt string `json:"updatedAt"`
}

// HasMeta 是否已具备版本信息
func (e FileEntry) HasMeta() bool {
	return e.Version != ""
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
	// 实时进度（同步运行中有效）
	Progress SyncProgress `json:"progress"`
}

// SyncProgress 同步实时进度
type SyncProgress struct {
	Done        int    `json:"done"`         // 已处理目标数
	Total       int    `json:"total"`        // 总目标数
	Item        string `json:"item"`         // 当前处理目标，如 "v2rayN (macOS Apple 芯片)"
	Stage       string `json:"stage"`        // 阶段：获取版本 / 下载中 / 上传中
	CurrentFile string `json:"current_file"` // 当前文件（下载/上传中的安装包名）
	Folder      string `json:"folder"`       // 上传目录（云盘路径）
}

var (
	statusMu     sync.Mutex
	syncRunning  bool
	lastRun      string
	lastReport   []ReportItem
	lastUploaded int

	// 实时进度
	progressDone   int
	progressTotal  int
	progressItem   string
	progressStage  string
	progressFile   string
	progressFolder string

	// OnSyncComplete 同步结束后回调（由 handlers 注册用于清理直链缓存等）
	OnSyncComplete func()
)

// SetProgress 更新实时进度（由 run 主流程调用）
func SetProgress(done int, item, stage, file string) {
	statusMu.Lock()
	defer statusMu.Unlock()
	if done >= 0 {
		progressDone = done
	}
	progressItem = item
	progressStage = stage
	progressFile = file
}

func SetProgressTotal(total int) {
	statusMu.Lock()
	defer statusMu.Unlock()
	progressTotal = total
}

func SetProgressFolder(folder string) {
	statusMu.Lock()
	defer statusMu.Unlock()
	progressFolder = folder
}

func ResetProgress() {
	statusMu.Lock()
	defer statusMu.Unlock()
	progressDone = 0
	progressTotal = 0
	progressItem = ""
	progressStage = ""
	progressFile = ""
}

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
	Enabled  bool
	Interval time.Duration
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
	statusMu.Lock()
	pr := SyncProgress{
		Done:        progressDone,
		Total:       progressTotal,
		Item:        progressItem,
		Stage:       progressStage,
		CurrentFile: progressFile,
		Folder:      progressFolder,
	}
	statusMu.Unlock()

	return SyncStatus{
		Running:       IsRunning(),
		Enabled:       cfg.Enabled,
		IntervalHours: int(cfg.Interval.Hours()),
		LastRun:       runAt,
		LastReport:    report,
		TotalUploaded: uploaded,
		Progress:      pr,
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

// Due 判断是否到定时同步时间。
// 同步引擎为轻量版本检测（仅查 GitHub API + 写 pan:// 链接，不下载/上传），
// 因此「从未运行过」视为立即到期：服务器启动/升级后自动检测一次并填充下载链接。
func Due() bool {
	cfg, err := loadSyncConfig()
	if err != nil || !cfg.Enabled {
		return false
	}
	last := LastSyncAt()
	if last.IsZero() {
		return true
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
	newVersions := 0
	ResetProgress()
	defer ResetProgress()
	onlySet := map[string]bool{}
	for _, k := range only {
		onlySet[k] = true
	}

	prefixes := loadProxyPrefixes()
	ghToken := LoadGitHubToken()
	releaseCache := map[string]*ghrelease.Release{}

	fileIDMap, err := LoadFileIDMap()
	if err != nil {
		return append(report, ReportItem{Status: "error", Message: "读取版本记录失败: " + err.Error()}), 0
	}

	// 计算本次将检查的目标总数（用于进度显示）
	totalTargets := 0
	for _, sw := range Catalog {
		for _, t := range sw.Targets {
			if len(onlySet) == 0 || onlySet[t.ConfigKey] {
				totalTargets++
			}
		}
	}
	SetProgressTotal(totalTargets)

	done := 0
	for _, sw := range Catalog {
		release, ok := releaseCache[sw.Repo]
		if !ok {
			release, err = ghrelease.Latest(sw.Repo, prefixes, ghToken)
			if err != nil {
				for _, t := range sw.Targets {
					if len(onlySet) > 0 && !onlySet[t.ConfigKey] {
						continue
					}
					done++
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
			SetProgress(done, sw.Name+" "+t.Label, "检查中", "")

			// 该入口配置了手工自定义外部链接（非 pan://）→ 不使用自动分发，整项跳过
			if current := loadSoftwareValue(t.ConfigKey); current != "" && !strings.HasPrefix(current, "pan://") {
				item.Status = "skip"
				item.Message = "该入口使用自定义链接，跳过自动分发"
				done++
				report = append(report, item)
				continue
			}

			asset, aerr := findAsset(release, &t)
			if aerr != nil {
				item.Status = "error"
				item.Message = aerr.Error()
				done++
				report = append(report, item)
				continue
			}
			SetProgress(done, sw.Name+" "+t.Label, "比对版本", asset.Name)

			// 版本与资产名一致 → 已是最新，跳过
			entry, hasEntry := fileIDMap[t.ConfigKey]
			if hasEntry && entry.Version == version && entry.FileName == asset.Name {
				item.Status = "skip"
				item.FileName = asset.Name
				item.Message = "已是最新版本"
				done++
				report = append(report, item)
				continue
			}

			// 有新版本（或首次检出）：更新版本记录，下载链接始终指向最新版
			fileIDMap[t.ConfigKey] = FileEntry{
				FileName:  asset.Name,
				Size:      asset.Size,
				Version:   version,
				UpdatedAt: time.Now().Format(time.RFC3339),
			}
			if err := ensurePanManagedURL(t.ConfigKey); err != nil {
				item.Message = "已检出新版本，但更新下载链接失败: " + err.Error()
			} else {
				item.Message = "已检出最新版 v" + version + "，下载链接将指向该版本（国内镜像直链）"
			}
			newVersions++
			item.Status = "ok"
			item.FileName = asset.Name
			done++
			report = append(report, item)
		}
	}

	if err := SaveFileIDMap(fileIDMap); err != nil {
		report = append(report, ReportItem{Status: "error", Message: "保存版本记录失败: " + err.Error()})
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
	return report, newVersions
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

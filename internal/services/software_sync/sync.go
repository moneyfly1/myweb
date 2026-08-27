package software_sync

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

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
	Folder             string // 云盘上传目录（/ 分隔多级；空 = 根目录）
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
		case "aliyun_folder":
			cfg.Folder = strings.TrimSpace(c.Value)
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
	// token 到期会自动轮换（新 refresh_token 在客户端内存中），
	// 无论同步结果如何都回存到 DB，避免只有定时同步在跑时轮换丢失导致 90 天后同步断掉。
	ali.OnRotate = func(newRT string) {
		if newRT != "" && newRT != cfg.AliyunRefreshToken {
			_ = saveCfgValue("aliyun_refresh_token", newRT)
		}
	}
	defer func() {
		if ali.RefreshToken != "" && ali.RefreshToken != cfg.AliyunRefreshToken {
			_ = saveCfgValue("aliyun_refresh_token", ali.RefreshToken)
		}
	}()

	// 解析上传目录（可配置，支持多级路径；空 = 根目录）
	folderID := "root"
	if cfg.Folder != "" {
		folderID, err = ali.EnsureDir(cfg.Folder)
		if err != nil {
			return append(report, ReportItem{Status: "error", Message: "创建/解析云盘上传目录失败: " + err.Error()}), 0
		}
	}

	fileIDMap, err := LoadFileIDMap()
	if err != nil {
		return append(report, ReportItem{Status: "error", Message: "读取文件映射失败: " + err.Error()}), 0
	}
	prefixes := loadProxyPrefixes()
	ghToken := LoadGitHubToken()
	releaseCache := map[string]*ghrelease.Release{}
	// 下载暂存目录：放在项目 uploads/synctmp 下（与数据库同盘、容量可预测），
	// 避免系统 /tmp 若是 tmpfs（内存盘）时大文件安装包占满内存导致生产不稳定。
	cleanStaleSyncTemp()
	tmpDir, err := newSyncTempDir()
	if err != nil {
		return append(report, ReportItem{Status: "error", Message: "创建下载暂存目录失败: " + err.Error()}), 0
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

		// 每个软件一个子文件夹（如 软件下载/v2rayN/），失败时回退到上传目录
		swFolderID := folderID
		subID, ferr := ali.EnsureFolder(folderID, sanitizeFolderName(sw.Name))
		if ferr != nil {
			// 子文件夹创建失败不阻断同步，回退到上传目录
			log.Printf("软件 %s 子文件夹创建失败（回退到上传目录）: %v", sw.Name, ferr)
		} else {
			swFolderID = subID
		}

		// 同一软件内同名资产（如 Hiddify 官方 macOS 通用包 Intel/Apple 同一文件）只上传一次，第二个目标复用
		uploadedByName := map[string]string{} // asset.Name → 本次已就绪的 file_id
		trashedIDs := map[string]bool{}       // 本次已删除的旧文件 id（多个目标共享同一旧 id 时避免重复删除）

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
				// 校验云盘文件仍存在（用户可能手动删除）：文件不存在时不跳过，走重新上传
				_, gerr := ali.GetFile(entry.AliyunFileID)
				if gerr == nil || !aliyundrive.IsFileNotFound(gerr) {
					item.Status = "skip"
					item.FileName = entry.FileName
					if gerr != nil {
						item.Message = "已是最新版本（云盘文件校验暂时不可用）"
					} else {
						item.Message = "已是最新版本"
					}
					report = append(report, item)
					continue
				}
				// 云盘文件已被删除 → 继续执行重新上传
			}

			localPath := filepath.Join(tmpDir, asset.Name)

			// 同一软件内同名资产复用本次已上传的文件（避免重复上传/重复占空间）
			if reuseID, ok := uploadedByName[asset.Name]; ok {
				fileIDMap[t.ConfigKey] = FileEntry{
					AliyunFileID: reuseID,
					FileName:     asset.Name,
					Size:         asset.Size,
					Version:      version,
					UpdatedAt:    time.Now().Format(time.RFC3339),
				}
				if err := ensurePanManagedURL(t.ConfigKey); err != nil {
					item.Message = "复用上传成功，但更新下载链接失败: " + err.Error()
				} else {
					item.Message = "复用已上传的同一安装包"
				}
				trashOldVersion(ali, entry, hasEntry, reuseID, &item, trashedIDs)
				item.Status = "ok"
				item.FileName = asset.Name
				report = append(report, item)
				continue
			}

			if derr := ghrelease.Download(asset, localPath, prefixes, ghToken); derr != nil {
				item.Status = "error"
				item.Message = "下载失败: " + derr.Error()
				report = append(report, item)
				continue
			}

			aliFileID, uerr := ali.Upload(localPath, asset.Name, swFolderID)
			if uerr != nil {
				item.Status = "error"
				item.Message = "上传阿里云盘失败: " + uerr.Error()
				report = append(report, item)
				continue
			}
			uploadedByName[asset.Name] = aliFileID

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
			trashOldVersion(ali, entry, hasEntry, aliFileID, &item, trashedIDs)
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

// trashOldVersion 删除该配置键对应的旧版云盘文件；多个目标共享同一旧文件时只删一次。
// newFileID 为本次新上传（或复用）的文件 id，相同则跳过，避免误删刚就绪的文件。
func trashOldVersion(ali *aliyundrive.Client, entry FileEntry, hasEntry bool, newFileID string, item *ReportItem, trashedIDs map[string]bool) {
	if !hasEntry || entry.AliyunFileID == "" || entry.AliyunFileID == newFileID || trashedIDs[entry.AliyunFileID] {
		return
	}
	if terr := ali.Trash(entry.AliyunFileID); terr != nil {
		item.Message = "新版本已上传，但删除旧版本失败: " + terr.Error()
		return
	}
	trashedIDs[entry.AliyunFileID] = true
	item.Message = "已上传阿里云盘并替换删除旧版本"
}

// syncTempBase 下载暂存根目录（相对项目工作目录；uploads 已 gitignore，与数据库同盘）
const syncTempBase = "uploads/synctmp"

// newSyncTempDir 创建本次同步的下载暂存目录
func newSyncTempDir() (string, error) {
	if err := os.MkdirAll(syncTempBase, 0o755); err != nil {
		return "", err
	}
	return os.MkdirTemp(syncTempBase, "sync-*")
}

// cleanStaleSyncTemp 清理超过 24 小时的遗留暂存目录（同步异常中断时防止堆积占空间）
func cleanStaleSyncTemp() {
	entries, err := os.ReadDir(syncTempBase)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, ierr := e.Info()
		if ierr != nil {
			continue
		}
		if time.Since(info.ModTime()) > 24*time.Hour {
			_ = os.RemoveAll(filepath.Join(syncTempBase, e.Name()))
		}
	}
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

// sanitizeFolderName 清理文件夹名中的非法字符（云盘禁止 / \ : * ? " < > | 等）
func sanitizeFolderName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "软件下载"
	}
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		}
		if unicode.IsControl(r) {
			return '_'
		}
		return r
	}, name)
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

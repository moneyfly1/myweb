package repo_sync

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/git"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Category 系统配置分类
const Category = "repo_sync"

// 配置键
const (
	KeyEnabled         = "repo_sync_enabled"
	KeyToken           = "repo_sync_token"
	KeyOwner           = "repo_sync_owner"
	KeyRepo            = "repo_sync_repo"
	KeyPath            = "repo_sync_path"
	KeyIntervalMinutes = "repo_sync_interval_minutes"

	// 状态键
	KeyLastTime    = "repo_sync_last_time"
	KeyLastStatus  = "repo_sync_last_status"
	KeyLastMessage = "repo_sync_last_message"
	KeyFileCount   = "repo_sync_file_count"
)

// 默认同步间隔（分钟）
const DefaultIntervalMinutes = 10

// 递归目录最大深度
const maxDirDepth = 8

// Config 同步配置
type Config struct {
	Enabled         bool
	Token           string
	Owner           string
	Repo            string
	Path            string
	IntervalMinutes int
}

// SyncResult 一次同步的结果
type SyncResult struct {
	FilesDownloaded int      `json:"files_downloaded"`
	FilesRemoved    int      `json:"files_removed"`
	TotalSize       int64    `json:"total_size"`
	Errors          []string `json:"errors"`
}

// Service 仓库文件同步服务：将 GitHub 私有仓库目录定时下载到本地公开目录
type Service struct {
	db *gorm.DB
	mu sync.Mutex
}

// NewService 创建同步服务
func NewService() *Service {
	return &Service{db: database.GetDB()}
}

// LoadConfig 从 system_configs 读取同步配置
func (s *Service) LoadConfig() Config {
	getValue := func(key, defaultVal string) string {
		var cfg models.SystemConfig
		if err := s.db.Where("key = ? AND category = ?", key, Category).First(&cfg).Error; err == nil {
			return cfg.Value
		}
		return defaultVal
	}

	cfg := Config{
		Token:           getValue(KeyToken, ""),
		Owner:           getValue(KeyOwner, ""),
		Repo:            getValue(KeyRepo, ""),
		Path:            getValue(KeyPath, ""),
		IntervalMinutes: DefaultIntervalMinutes,
	}
	cfg.Enabled = getValue(KeyEnabled, "false") == "true"

	if minutes, err := strconv.Atoi(getValue(KeyIntervalMinutes, strconv.Itoa(DefaultIntervalMinutes))); err == nil && minutes > 0 {
		cfg.IntervalMinutes = minutes
	}
	if cfg.IntervalMinutes <= 0 {
		cfg.IntervalMinutes = DefaultIntervalMinutes
	}

	return cfg
}

// LocalDirPath 本地存储目录: <工作目录>/<UploadDir>/repo_sync
func (s *Service) LocalDirPath() string {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	uploadDir := "uploads"
	if config.AppConfig != nil && config.AppConfig.UploadDir != "" {
		uploadDir = config.AppConfig.UploadDir
	}
	return filepath.Join(wd, uploadDir, "repo_sync")
}

// IsRunning 是否正在同步
func (s *Service) IsRunning() bool {
	if s.mu.TryLock() {
		s.mu.Unlock()
		return false
	}
	return true
}

// ShouldRunNow 根据配置（启用开关 + 间隔 + 上次执行时间）判断是否到了执行时间
func (s *Service) ShouldRunNow() bool {
	cfg := s.LoadConfig()
	if !cfg.Enabled || cfg.Token == "" {
		return false
	}

	lastStr := s.getConfigValue(KeyLastTime)
	if lastStr == "" {
		return true // 从未同步过
	}

	last, err := time.ParseInLocation("2006-01-02T15:04:05", lastStr, utils.BeijingTZ)
	if err != nil {
		return true
	}

	elapsed := utils.GetBeijingTime().Sub(last)
	return elapsed >= time.Duration(cfg.IntervalMinutes)*time.Minute
}

// Tick 定时任务入口：到点则执行一次同步并记录调度日志
func (s *Service) Tick() {
	if !s.ShouldRunNow() {
		return
	}

	utils.LogInfo("开始执行 GitHub 仓库文件同步任务")
	if err := utils.CreateSchedulerLog("repo_sync", "started", "开始执行 GitHub 仓库文件同步任务", nil); err != nil {
		log.Printf("failed to create scheduler log: %v", err)
	}

	start := time.Now()
	result, err := s.SyncNow()
	if err != nil {
		utils.LogErrorMsg("GitHub 仓库文件同步失败: %v", err)
		if logErr := utils.CreateSchedulerLog("repo_sync", "error", fmt.Sprintf("GitHub 仓库文件同步失败: %v", err), map[string]interface{}{
			"error": err.Error(),
		}); logErr != nil {
			log.Printf("failed to create scheduler log: %v", logErr)
		}
		return
	}

	msg := fmt.Sprintf("同步完成: 下载 %d 个文件, 清理 %d 个文件, 耗时 %.1fs", result.FilesDownloaded, result.FilesRemoved, time.Since(start).Seconds())
	if len(result.Errors) > 0 {
		msg += fmt.Sprintf(", %d 个文件失败", len(result.Errors))
	}
	utils.LogInfo("%s", msg)
	if err := utils.CreateSchedulerLog("repo_sync", "success", msg, map[string]interface{}{
		"files_downloaded": result.FilesDownloaded,
		"files_removed":    result.FilesRemoved,
		"errors":           result.Errors,
	}); err != nil {
		log.Printf("failed to create scheduler log: %v", err)
	}
}

// SyncNow 立即执行一次同步：列出远程目录、逐个下载、清理本地多余文件
func (s *Service) SyncNow() (*SyncResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg := s.LoadConfig()
	if cfg.Token == "" || cfg.Owner == "" || cfg.Repo == "" {
		err := errors.New("请先配置 GitHub Token、仓库所有者和仓库名称")
		s.saveStatus("failed", err.Error(), 0)
		return nil, err
	}

	client := git.NewClient(git.PlatformGitHub, cfg.Token, cfg.Owner, cfg.Repo)

	// 递归列出远程目录下所有文件
	var files []git.RemoteEntry
	if err := s.listRemoteRecursive(client, cfg.Path, 0, &files); err != nil {
		s.saveStatus("failed", err.Error(), 0)
		return nil, err
	}

	localDir := s.LocalDirPath()
	if err := os.MkdirAll(localDir, 0750); err != nil {
		err = fmt.Errorf("创建本地目录失败: %w", err)
		s.saveStatus("failed", err.Error(), 0)
		return nil, err
	}

	result := &SyncResult{}
	remoteRelPaths := make(map[string]bool, len(files))

	for _, f := range files {
		rel, err := s.relLocalPath(cfg.Path, f.Name)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", f.Name, err))
			continue
		}
		remoteRelPaths[rel] = true

		if err := s.downloadOne(client, f.Name, filepath.Join(localDir, rel)); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", f.Name, err))
			continue
		}
		result.FilesDownloaded++
		result.TotalSize += f.Size
	}

	// 清理本地已不存在于远程的文件及残留临时文件
	if removed, err := s.removeStaleFiles(localDir, remoteRelPaths); err != nil {
		utils.LogErrorMsg("清理本地多余文件失败: %v", err)
	} else {
		result.FilesRemoved = removed
	}

	status := "success"
	if len(result.Errors) > 0 && result.FilesDownloaded == 0 {
		status = "failed"
	} else if len(result.Errors) > 0 {
		status = "partial"
	}

	message := fmt.Sprintf("同步完成: 下载 %d 个文件, 清理 %d 个文件", result.FilesDownloaded, result.FilesRemoved)
	if len(result.Errors) > 0 {
		message += fmt.Sprintf(", %d 个文件失败", len(result.Errors))
	}
	s.saveStatus(status, message, len(files))

	return result, nil
}

// TestConnectionWith 测试连接并列出远程目录文件。参数为空时回退到已保存的配置。
func (s *Service) TestConnectionWith(token, owner, repo, path string) ([]git.RemoteEntry, error) {
	cfg := s.LoadConfig()
	if token != "" {
		cfg.Token = token
	}
	if owner != "" {
		cfg.Owner = owner
	}
	if repo != "" {
		cfg.Repo = repo
	}
	if path != "" {
		cfg.Path = path
	}

	if cfg.Token == "" || cfg.Owner == "" || cfg.Repo == "" {
		return nil, errors.New("请先填写 GitHub Token、仓库所有者和仓库名称")
	}

	client := git.NewClient(git.PlatformGitHub, cfg.Token, cfg.Owner, cfg.Repo)
	if err := client.TestConnection(); err != nil {
		return nil, fmt.Errorf("连接 GitHub 失败: %w", err)
	}

	var files []git.RemoteEntry
	if err := s.listRemoteRecursive(client, cfg.Path, 0, &files); err != nil {
		return nil, err
	}
	return files, nil
}

// GetStatus 返回同步状态与本地文件列表（供管理页展示）
func (s *Service) GetStatus() map[string]interface{} {
	cfg := s.LoadConfig()
	return map[string]interface{}{
		"enabled":          cfg.Enabled,
		"interval_minutes": cfg.IntervalMinutes,
		"is_running":       s.IsRunning(),
		"last_time":        s.getConfigValue(KeyLastTime),
		"last_status":      s.getConfigValue(KeyLastStatus),
		"last_message":     s.getConfigValue(KeyLastMessage),
		"file_count":       s.getConfigValue(KeyFileCount),
		"files":            s.listLocalFiles(),
	}
}

// listRemoteRecursive 递归列出远程目录下所有文件
func (s *Service) listRemoteRecursive(client *git.GitClient, path string, depth int, out *[]git.RemoteEntry) error {
	if depth > maxDirDepth {
		return fmt.Errorf("目录层级超过 %d 层: %s", maxDirDepth, path)
	}

	entries, err := client.ListContents(path)
	if err != nil {
		return fmt.Errorf("列出目录 %s 失败: %w", path, err)
	}

	for _, entry := range entries {
		if !validRemoteName(entry.Name) {
			continue
		}
		childPath := strings.Trim(strings.TrimSuffix(path, "/")+"/"+entry.Name, "/")
		switch entry.Type {
		case "dir":
			if err := s.listRemoteRecursive(client, childPath, depth+1, out); err != nil {
				return err
			}
		case "file":
			*out = append(*out, git.RemoteEntry{Name: childPath, Type: "file", Size: entry.Size})
		}
	}
	return nil
}

// relLocalPath 将远程完整路径转换为相对配置目录的本地相对路径
func (s *Service) relLocalPath(basePath, remotePath string) (string, error) {
	base := strings.Trim(strings.TrimSpace(basePath), "/")
	remote := strings.TrimPrefix(strings.TrimSpace(remotePath), "/")

	if base != "" {
		if remote == base {
			return "", fmt.Errorf("路径不是文件: %s", remotePath)
		}
		prefix := base + "/"
		if !strings.HasPrefix(remote, prefix) {
			return "", fmt.Errorf("路径超出配置目录: %s", remotePath)
		}
		remote = strings.TrimPrefix(remote, prefix)
	}

	if remote == "" {
		return "", fmt.Errorf("非法路径: %s", remotePath)
	}
	for _, part := range strings.Split(remote, "/") {
		if !validRemoteName(part) {
			return "", fmt.Errorf("非法路径: %s", remotePath)
		}
	}
	return filepath.FromSlash(remote), nil
}

// downloadOne 下载单个文件（先写临时文件再原子替换，失败时保留旧文件）
func (s *Service) downloadOne(client *git.GitClient, remotePath, localPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0750); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	tmpPath := localPath + ".tmp"
	_ = os.Remove(tmpPath) // 清理可能残留的临时文件

	if err := client.DownloadFile(remotePath, tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, localPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("替换文件失败: %w", err)
	}
	return nil
}

// removeStaleFiles 删除本地目录中远程已不存在（或残留 .tmp）的文件
func (s *Service) removeStaleFiles(localDir string, keep map[string]bool) (int, error) {
	removed := 0
	err := filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(localDir, path)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)
		if strings.HasSuffix(relSlash, ".tmp") || !keep[relSlash] {
			if os.Remove(path) == nil {
				removed++
			}
		}
		return nil
	})
	return removed, err
}

// listLocalFiles 列出本地目录文件（含公开访问 URL）
func (s *Service) listLocalFiles() []map[string]interface{} {
	localDir := s.LocalDirPath()
	files := []map[string]interface{}{}

	_ = filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(localDir, path)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)
		if strings.HasSuffix(relSlash, ".tmp") {
			return nil
		}
		files = append(files, map[string]interface{}{
			"name": relSlash,
			"size": info.Size(),
			"url":  "/repo-sync/" + relSlash,
		})
		return nil
	})

	sort.Slice(files, func(i, j int) bool {
		return files[i]["name"].(string) < files[j]["name"].(string)
	})
	return files
}

// saveStatus 将状态写入 system_configs（upsert）
func (s *Service) saveStatus(status, message string, fileCount int) {
	now := utils.GetBeijingTime().Format("2006-01-02T15:04:05")
	rows := []models.SystemConfig{
		{Key: KeyLastTime, Value: now, Category: Category},
		{Key: KeyLastStatus, Value: status, Category: Category},
		{Key: KeyLastMessage, Value: message, Category: Category},
		{Key: KeyFileCount, Value: strconv.Itoa(fileCount), Category: Category},
	}

	for _, row := range rows {
		if err := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}, {Name: "category"}},
			DoUpdates: clause.Assignments(map[string]interface{}{"value": row.Value}),
		}).Create(&row).Error; err != nil {
			log.Printf("failed to save repo_sync status: %v", err)
		}
	}
}

// getConfigValue 读取单个配置值
func (s *Service) getConfigValue(key string) string {
	var cfg models.SystemConfig
	if err := s.db.Where("key = ? AND category = ?", key, Category).First(&cfg).Error; err == nil {
		return cfg.Value
	}
	return ""
}

// validRemoteName 校验单个路径片段
func validRemoteName(name string) bool {
	return name != "" && name != "." && name != ".." &&
		!strings.Contains(name, "/") && !strings.Contains(name, "\\") &&
		!strings.Contains(name, "\x00")
}

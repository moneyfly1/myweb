package backup_service

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"time"

	"cboard-go/internal/models"
	"cboard-go/internal/services/git"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

type RemoteBackupConfig struct {
	Target       string
	Enabled      bool
	Token        string
	Owner        string
	Repo         string
	PlatformType git.PlatformType
	PlatformName string
}

type PlatformBackupConfig struct {
	Target       string
	Token        string
	Owner        string
	Repo         string
	PlatformType git.PlatformType
	PlatformName string
}

func LoadRemoteBackupConfig(db *gorm.DB) RemoteBackupConfig {
	base := DefaultPlatformConfig("gitee")
	cfg := RemoteBackupConfig{
		Target:       base.Target,
		Owner:        base.Owner,
		Repo:         base.Repo,
		PlatformType: base.PlatformType,
		PlatformName: base.PlatformName,
	}

	var targetConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "backup_target", "backup").First(&targetConfig).Error; err == nil && targetConfig.Value == "github" {
		cfg.Target = "github"
		cfg.Owner = "moneyfly1"
		cfg.PlatformType = git.PlatformGitHub
		cfg.PlatformName = "GitHub"
	}

	enabledKey, tokenKey, ownerKey, repoKey := keysByTarget(cfg.Target)
	var enabledConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", enabledKey, "backup").First(&enabledConfig).Error; err == nil && enabledConfig.Value == "true" {
		cfg.Enabled = true
	}
	if !cfg.Enabled {
		return cfg
	}

	var tokenConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", tokenKey, "backup").First(&tokenConfig).Error; err == nil {
		cfg.Token = tokenConfig.Value
	}

	var ownerConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", ownerKey, "backup").First(&ownerConfig).Error; err == nil && ownerConfig.Value != "" {
		cfg.Owner = ownerConfig.Value
	}

	var repoConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", repoKey, "backup").First(&repoConfig).Error; err == nil && repoConfig.Value != "" {
		cfg.Repo = repoConfig.Value
	}

	return cfg
}

func LoadPlatformConfig(db *gorm.DB, target string) PlatformBackupConfig {
	cfg := DefaultPlatformConfig(target)
	_, tokenKey, ownerKey, repoKey := keysByTarget(cfg.Target)

	var tokenConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", tokenKey, "backup").First(&tokenConfig).Error; err == nil {
		cfg.Token = tokenConfig.Value
	}
	var ownerConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", ownerKey, "backup").First(&ownerConfig).Error; err == nil && ownerConfig.Value != "" {
		cfg.Owner = ownerConfig.Value
	}
	var repoConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", repoKey, "backup").First(&repoConfig).Error; err == nil && repoConfig.Value != "" {
		cfg.Repo = repoConfig.Value
	}
	return cfg
}

func DefaultPlatformConfig(target string) PlatformBackupConfig {
	if target == "github" {
		return PlatformBackupConfig{
			Target:       "github",
			Owner:        "moneyfly1",
			Repo:         "backup",
			PlatformType: git.PlatformGitHub,
			PlatformName: "GitHub",
		}
	}
	return PlatformBackupConfig{
		Target:       "gitee",
		Owner:        "moneyfly",
		Repo:         "backup",
		PlatformType: git.PlatformGitee,
		PlatformName: "Gitee",
	}
}

func BuildDBOnlyBackupZip(wd, backupDir string, now time.Time) (string, string, int64, error) {
	backupFileName := fmt.Sprintf("backup_db_%s.zip", now.Format("20060102_150405"))
	backupFilePath, ok := utils.JoinWithinBaseDir(backupDir, backupFileName)
	if !ok {
		return "", "", 0, fmt.Errorf("invalid backup path")
	}

	zipFile, err := os.Create(backupFilePath)
	if err != nil {
		return "", "", 0, err
	}

	zipWriter := zip.NewWriter(zipFile)
	dbPath, ok := utils.JoinWithinBaseDir(wd, "cboard.db")
	if ok {
		if _, statErr := os.Stat(dbPath); statErr == nil {
			dbFile, openErr := os.Open(dbPath)
			if openErr != nil {
				_ = zipWriter.Close()
				_ = zipFile.Close()
				return "", "", 0, openErr
			}
			writer, createErr := zipWriter.Create("cboard.db")
			if createErr != nil {
				_ = dbFile.Close()
				_ = zipWriter.Close()
				_ = zipFile.Close()
				return "", "", 0, createErr
			}
			if _, copyErr := io.Copy(writer, dbFile); copyErr != nil {
				_ = dbFile.Close()
				_ = zipWriter.Close()
				_ = zipFile.Close()
				return "", "", 0, copyErr
			}
			if closeErr := dbFile.Close(); closeErr != nil {
				_ = zipWriter.Close()
				_ = zipFile.Close()
				return "", "", 0, closeErr
			}
		}
	}

	if closeErr := zipWriter.Close(); closeErr != nil {
		_ = zipFile.Close()
		return "", "", 0, closeErr
	}
	if closeErr := zipFile.Close(); closeErr != nil {
		return "", "", 0, closeErr
	}

	var fileSize int64
	if fileInfo, statErr := os.Stat(backupFilePath); statErr == nil {
		fileSize = fileInfo.Size()
	}

	return backupFileName, backupFilePath, fileSize, nil
}

func keysByTarget(target string) (enabledKey, tokenKey, ownerKey, repoKey string) {
	if target == "github" {
		return "backup_github_enabled", "backup_github_token", "backup_github_owner", "backup_github_repo"
	}
	return "backup_gitee_enabled", "backup_gitee_token", "backup_gitee_owner", "backup_gitee_repo"
}

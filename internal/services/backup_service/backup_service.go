package backup_service

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cboard-go/internal/models"
	"cboard-go/internal/services/git"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

// 敏感配置键：备份 .env / config.yaml 时值将被替换为 ***REDACTED***
var sensitiveConfigKeys = []string{
	"SECRET_KEY", "SMTP_PASSWORD", "REDIS_PASSWORD", "JWT_SECRET_KEY",
	"ALIPAY_PRIVATE_KEY", "ALIPAY_PUBLIC_KEY", "WECHAT_API_KEY", "WECHAT_API_SECRET",
	"PAYPAL_SECRET", "STRIPE_SECRET", "PAYMENT_KEY", "PAYMENT_SECRET",
	"APP_KEY", "API_SECRET", "TOKEN", "PASSWORD", "PRIVATE_KEY", "SECRET",
}

var sensitiveConfigValueRe = regexp.MustCompile(`(?i)((?:secret|password|token|key|private)[^=]*)=(.+)$`)

type RemoteBackupConfig struct {
	Target       string
	Enabled      bool
	Token        string
	Owner        string
	Repo         string
	PlatformType git.PlatformType
	PlatformName string
}

// CanPush 判断是否具备推送远端备份的条件：
// 必须显式配置 Owner/Repo，绝不默认推送到第三方/作者仓库。
func (c RemoteBackupConfig) CanPush() bool {
	return c.Enabled && c.Token != "" && c.Owner != "" && c.Repo != ""
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
		// 不默认填充 Owner：仓库必须是用户自己的，未配置时 CanPush() 为 false，推送安全失败
		cfg.Owner = ""
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
			Owner:        "",
			Repo:         "",
			PlatformType: git.PlatformGitHub,
			PlatformName: "GitHub",
		}
	}
	return PlatformBackupConfig{
		Target:       "gitee",
		Owner:        "",
		Repo:         "",
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

// AddSanitizedConfigFile 将 .env / config.yaml 脱敏后写入 zip。
// 敏感键（SECRET_KEY/SMTP_PASSWORD/各支付密钥等）的值一律替换为 ***REDACTED***，
// 防止备份文件外泄后凭据被直接复用。文件不存在时静默跳过。
func AddSanitizedConfigFile(zipWriter *zip.Writer, wd, configFile string) {
	if strings.Contains(configFile, "..") || strings.Contains(configFile, "/") ||
		strings.Contains(configFile, "\\") || strings.Contains(configFile, "~") {
		return
	}
	configPath, inBase := utils.JoinWithinBaseDir(wd, configFile)
	if !inBase {
		return
	}
	file, err := os.Open(configPath)
	if err != nil {
		return
	}
	defer file.Close()

	writer, err := zipWriter.Create(filepath.Base(configFile))
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if _, err := io.WriteString(writer, sanitizeConfigLine(line)+"\n"); err != nil {
			return
		}
	}
}

// sanitizeConfigLine 对单行配置做脱敏：key 含敏感词时替换 value；YAML 的 key: value 同规则。
func sanitizeConfigLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return line
	}
	// 支持 KEY=value 与 KEY: value 两种格式
	sepIndex := strings.IndexAny(trimmed, "=:")
	if sepIndex <= 0 {
		return line
	}
	key := strings.TrimSpace(trimmed[:sepIndex])
	if isSensitiveKey(key) {
		// 保留 key 与分隔符，值替换为掩码
		prefix := line[:len(line)-len(trimmed)]
		rest := line[len(line)-len(trimmed):]
		sep := string(rest[sepIndex])
		// 定位行内真实分隔符（考虑行首空白）
		sepIdxInRest := strings.IndexAny(rest, "=:")
		return prefix + rest[:sepIdxInRest] + sep + "***REDACTED***"
	}
	return line
}

func isSensitiveKey(key string) bool {
	upper := strings.ToUpper(key)
	for _, k := range sensitiveConfigKeys {
		if upper == k || strings.Contains(upper, k) {
			return true
		}
	}
	return false
}

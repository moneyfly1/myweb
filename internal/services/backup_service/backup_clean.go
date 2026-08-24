package backup_service

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// BackupCleanConfig 备份前数据库清理配置
type BackupCleanConfig struct {
	Enabled       bool
	RetentionDays int // 钳制到 [1,365]
}

// LoadBackupCleanConfig 从 system_configs(category=backup) 读取清理配置，默认启用、保留 7 天
func LoadBackupCleanConfig(db *gorm.DB) BackupCleanConfig {
	cfg := BackupCleanConfig{Enabled: true, RetentionDays: 7}

	var row models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "backup_clean_enabled", "backup").First(&row).Error; err == nil {
		cfg.Enabled = row.Value == "true"
	}
	if err := db.Where("key = ? AND category = ?", "backup_log_retention_days", "backup").First(&row).Error; err == nil {
		if v, err2 := strconv.Atoi(row.Value); err2 == nil && v >= 1 && v <= 365 {
			cfg.RetentionDays = v
		}
	}
	return cfg
}

// cleanClearAllTables 备份副本中完全清空的临时数据表（恢复后无任何意义）
var cleanClearAllTables = []string{
	"verification_codes",
	"verification_attempts",
	"token_blacklist",
	"ticket_reads",
	"email_queue",
	"login_attempts",
}

// cleanRetentionTables 仅保留最近 N 天的日志表；值为时间列名
// 注意 login_history 的时间列是 login_time 而非 created_at
var cleanRetentionTables = map[string]string{
	"audit_logs":          "created_at",
	"login_history":       "login_time",
	"user_activities":     "created_at",
	"registration_logs":   "created_at",
	"subscription_logs":   "created_at",
	"subscription_resets": "created_at",
	"checkin_records":     "created_at",
	"notifications":       "created_at",
}

// PrepareBackupDB 生成清理 + 压缩后的 SQLite 数据库副本（临时文件）。
// 全程只操作临时文件，线上数据库不受任何影响。
// 返回临时文件路径（调用方必须负责删除）与清理后大小；失败返回 error，由调用方回退为原始文件打包。
func PrepareBackupDB(wd string, retentionDays int) (string, int64, error) {
	if retentionDays < 1 {
		retentionDays = 7
	}
	if retentionDays > 365 {
		retentionDays = 365
	}

	dbPath, ok := utils.JoinWithinBaseDir(wd, "cboard.db")
	if !ok {
		return "", 0, fmt.Errorf("invalid db path")
	}
	if _, err := os.Stat(dbPath); err != nil {
		return "", 0, err
	}

	// 1) 占位临时文件：CreateTemp 后立即删除，供 VACUUM INTO 覆写（SQLite 要求目标不存在）
	tmpFile, err := os.CreateTemp("", "cboard_clean_*.db")
	if err != nil {
		return "", 0, err
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	_ = os.Remove(tmpPath)

	// 2) 一致性快照 + 首次压缩：VACUUM INTO 取 WAL 读快照，不阻塞线上写入，
	//    目标文件是全新的 DELETE 日志模式数据库（不会继承源库 WAL 头）
	live := database.GetDB()
	if live == nil {
		return "", 0, fmt.Errorf("database not initialized")
	}
	if err := live.Exec("VACUUM INTO '" + strings.ReplaceAll(tmpPath, "'", "''") + "'").Error; err != nil {
		return "", 0, fmt.Errorf("vacuum into: %w", err)
	}

	// 3) 打开临时库执行裁剪（静默日志，避免 SQL 刷屏）
	cleanDB, err := gorm.Open(sqlite.Open(tmpPath), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", 0, err
	}
	sqlDB, err := cleanDB.DB()
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", 0, err
	}
	sqlDB.SetMaxOpenConns(1) // 单连接串行执行，VACUUM 不受其他池连接读锁干扰
	sqlDB.SetMaxIdleConns(1)
	_ = cleanDB.Exec("PRAGMA busy_timeout=5000").Error
	_ = cleanDB.Exec("PRAGMA journal_mode=DELETE").Error

	closeWithCleanup := func() {
		_ = sqlDB.Close()
		_ = os.Remove(tmpPath)
	}

	cutoff := utils.GetBeijingTime().Add(-time.Duration(retentionDays) * 24 * time.Hour)

	// 4) 逐表裁剪：尽力而为，单表失败仅告警并继续
	for _, table := range cleanClearAllTables {
		if !tableExists(cleanDB, table) {
			continue
		}
		if err := cleanDB.Exec("DELETE FROM " + table).Error; err != nil {
			utils.LogWarn("备份清理: 清空 %s 失败(忽略): %v", table, err)
		}
	}
	for table, col := range cleanRetentionTables {
		if !tableExists(cleanDB, table) {
			continue
		}
		if err := cleanDB.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s < ?", table, col), cutoff).Error; err != nil {
			utils.LogWarn("备份清理: 裁剪 %s 失败(忽略): %v", table, err)
		}
	}

	// 5) 压缩回收被删除行占用的页
	if err := cleanDB.Exec("VACUUM").Error; err != nil {
		closeWithCleanup()
		return "", 0, fmt.Errorf("vacuum temp db: %w", err)
	}

	// 关闭连接池后再打包/删除，确保 -wal/-shm 已合并清理
	if err := sqlDB.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", 0, err
	}

	info, err := os.Stat(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", 0, err
	}
	return tmpPath, info.Size(), nil
}

// tableExists 检查临时库中是否存在指定表（兼容旧 schema）
func tableExists(db *gorm.DB, table string) bool {
	var n int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&n)
	return n > 0
}

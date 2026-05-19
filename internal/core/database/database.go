package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"cboard-go/internal/core/config"
	"cboard-go/internal/models"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDatabase() error {
	cfg := config.AppConfig
	if cfg == nil {
		return fmt.Errorf("配置未初始化")
	}

	var dialector gorm.Dialector
	var err error
	if strings.Contains(cfg.DatabaseURL, "sqlite") {
		dbPath := strings.Replace(cfg.DatabaseURL, "sqlite:///./", "", 1)
		dbPath = strings.Replace(dbPath, "sqlite:///", "", 1)
		if !filepath.IsAbs(dbPath) {
			dbPath = filepath.Join(".", dbPath)
		}

		dialector = sqlite.Open(dbPath)
	} else if strings.Contains(cfg.DatabaseURL, "mysql") ||
		os.Getenv("USE_MYSQL") == "true" {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Asia%%2FShanghai",
			cfg.MySQLUser,
			cfg.MySQLPassword,
			cfg.MySQLHost,
			cfg.MySQLPort,
			cfg.MySQLDatabase,
		)
		dialector = mysql.Open(dsn)
	} else if strings.Contains(cfg.DatabaseURL, "postgresql") ||
		os.Getenv("USE_POSTGRES") == "true" {
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=5432 sslmode=disable TimeZone=Asia/Shanghai",
			cfg.PostgresServer,
			cfg.PostgresUser,
			cfg.PostgresPass,
			cfg.PostgresDB,
		)
		dialector = postgres.Open(dsn)
	} else {
		dbPath := "cboard.db"
		if !filepath.IsAbs(dbPath) {
			dbPath = filepath.Join(".", dbPath)
		}
		dialector = sqlite.Open(dbPath)
	}
	customLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Silent,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	if cfg.Debug {
		customLogger = logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Info,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		)
	}

	gormConfig := &gorm.Config{
		Logger: customLogger,
	}
	DB, err = gorm.Open(dialector, gormConfig)
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}
	if strings.Contains(cfg.DatabaseURL, "sqlite") {
		DB.Exec("PRAGMA journal_mode=WAL")
		DB.Exec("PRAGMA busy_timeout=5000")
		DB.Exec("PRAGMA synchronous=NORMAL")
		sqlDB.SetMaxOpenConns(10)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(time.Hour)
	} else {
		// 根据CPU核心数动态调整连接池配置
		numCPU := runtime.NumCPU()
		maxOpenConns := numCPU * 5
		if maxOpenConns < 25 {
			maxOpenConns = 25
		}
		if maxOpenConns > 100 {
			maxOpenConns = 100
		}

		maxIdleConns := numCPU * 2
		if maxIdleConns < 5 {
			maxIdleConns = 5
		}
		if maxIdleConns > 20 {
			maxIdleConns = 20
		}

		sqlDB.SetMaxOpenConns(maxOpenConns)
		sqlDB.SetMaxIdleConns(maxIdleConns)
		sqlDB.SetConnMaxLifetime(30 * time.Minute)

		log.Printf("数据库连接池配置: MaxOpenConns=%d, MaxIdleConns=%d, ConnMaxLifetime=30m (CPU核心数: %d)",
			maxOpenConns, maxIdleConns, numCPU)
	}
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	if strings.Contains(cfg.DatabaseURL, "mysql") || os.Getenv("USE_MYSQL") == "true" {
		if err := DB.Exec("SET time_zone = '+08:00'").Error; err != nil {
			log.Printf("警告: 设置 MySQL 时区失败: %v", err)
		} else {
			log.Println("MySQL 会话时区已设置为 Asia/Shanghai (+08:00)")
		}
	}

	log.Println("数据库连接成功")
	return nil
}

func AutoMigrate() error {
	if DB == nil {
		return fmt.Errorf("数据库未初始化")
	}
	fulfilledAtExisted := DB.Migrator().HasColumn(&models.Order{}, "FulfilledAt")
	fulfilledAtAdded := false
	if strings.Contains(DB.Dialector.Name(), "sqlite") {
		var ordersExists int64
		DB.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='orders'").Scan(&ordersExists)
		if ordersExists > 0 {
			var hasFulfilledAt int64
			DB.Raw("SELECT COUNT(*) FROM pragma_table_info('orders') WHERE name='fulfilled_at'").Scan(&hasFulfilledAt)
			if hasFulfilledAt == 0 {
				if err := DB.Exec("ALTER TABLE orders ADD COLUMN fulfilled_at datetime").Error; err != nil {
					log.Printf("警告: 添加 orders.fulfilled_at 列失败（可能已存在）: %v", err)
				} else {
					fulfilledAtAdded = true
				}
			}
		}

		var tableExists int64
		DB.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='custom_nodes'").Scan(&tableExists)
		if tableExists > 0 {
			var hasProtocol, hasDomain, hasDisplayName int64
			DB.Raw("SELECT COUNT(*) FROM pragma_table_info('custom_nodes') WHERE name='protocol'").Scan(&hasProtocol)
			DB.Raw("SELECT COUNT(*) FROM pragma_table_info('custom_nodes') WHERE name='domain'").Scan(&hasDomain)
			DB.Raw("SELECT COUNT(*) FROM pragma_table_info('custom_nodes') WHERE name='display_name'").Scan(&hasDisplayName)
			var hasServerID, hasXrayRNodeID, hasTrafficLimit, hasTrafficUsed, hasTrafficResetAt, hasCertPath, hasKeyPath, hasCertExpireAt int64
			DB.Raw("SELECT COUNT(*) FROM pragma_table_info('custom_nodes') WHERE name='server_id'").Scan(&hasServerID)
			DB.Raw("SELECT COUNT(*) FROM pragma_table_info('custom_nodes') WHERE name='xray_r_node_id'").Scan(&hasXrayRNodeID)
			DB.Raw("SELECT COUNT(*) FROM pragma_table_info('custom_nodes') WHERE name='traffic_limit'").Scan(&hasTrafficLimit)
			DB.Raw("SELECT COUNT(*) FROM pragma_table_info('custom_nodes') WHERE name='traffic_used'").Scan(&hasTrafficUsed)
			DB.Raw("SELECT COUNT(*) FROM pragma_table_info('custom_nodes') WHERE name='traffic_reset_at'").Scan(&hasTrafficResetAt)
			DB.Raw("SELECT COUNT(*) FROM pragma_table_info('custom_nodes') WHERE name='cert_path'").Scan(&hasCertPath)
			DB.Raw("SELECT COUNT(*) FROM pragma_table_info('custom_nodes') WHERE name='key_path'").Scan(&hasKeyPath)
			DB.Raw("SELECT COUNT(*) FROM pragma_table_info('custom_nodes') WHERE name='cert_expire_at'").Scan(&hasCertExpireAt)
			hasOldFields := hasServerID > 0 || hasXrayRNodeID > 0 || hasTrafficLimit > 0 || hasTrafficUsed > 0 || hasTrafficResetAt > 0 || hasCertPath > 0 || hasKeyPath > 0 || hasCertExpireAt > 0
			if hasDomain == 0 || hasOldFields {
				log.Println("检测到旧版 custom_nodes 表结构，开始重建表...")
				var nodeCount int64
				DB.Raw("SELECT COUNT(*) FROM custom_nodes").Scan(&nodeCount)
				if nodeCount > 0 {
					log.Printf("发现 %d 条旧数据，将备份到 custom_nodes_backup 表", nodeCount)
					var backupExists int64
					DB.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='custom_nodes_backup'").Scan(&backupExists)
					if backupExists > 0 {
						DB.Exec("DROP TABLE custom_nodes_backup")
					}
					DB.Exec("CREATE TABLE custom_nodes_backup AS SELECT * FROM custom_nodes")
					log.Println("旧表已备份为 custom_nodes_backup")
				}
				DB.Exec("DROP TABLE custom_nodes")
				log.Println("已删除旧表，将在后续创建新表")
			} else {
				if hasDisplayName == 0 {
					err := DB.Exec("ALTER TABLE custom_nodes ADD COLUMN display_name VARCHAR(100) DEFAULT ''").Error
					if err != nil {
						log.Printf("警告: 添加 display_name 列失败（可能已存在）: %v", err)
					}
				}
				if hasProtocol > 0 {
					var protocolNotNull int64
					DB.Raw("SELECT COUNT(*) FROM pragma_table_info('custom_nodes') WHERE name='protocol' AND \"notnull\"=1").Scan(&protocolNotNull)
					if protocolNotNull > 0 {
						log.Println("Protocol 字段为 NOT NULL，需要重建表以移除约束...")
						var nodeCount int64
						DB.Raw("SELECT COUNT(*) FROM custom_nodes").Scan(&nodeCount)
						if nodeCount > 0 {
							var backupExists int64
							DB.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='custom_nodes_protocol_backup'").Scan(&backupExists)
							if backupExists > 0 {
								DB.Exec("DROP TABLE custom_nodes_protocol_backup")
							}
							DB.Exec("CREATE TABLE custom_nodes_protocol_backup AS SELECT * FROM custom_nodes")
						}
						DB.Exec("DROP TABLE custom_nodes")
						log.Println("已删除旧表以修复 Protocol 字段约束，将在后续创建新表")
					}
				}
			}
		}
		if err := repairSQLitePromotionParticipationsDDL(); err != nil {
			return err
		}
	}
	err := DB.AutoMigrate(
		&models.User{},
		&models.UserLevel{},
		&models.InviteCode{},
		&models.InviteRelation{},
		&models.Subscription{},
		&models.Device{},
		&models.SubscriptionReset{},
		&models.Order{},
		&models.Package{},
		&models.PaymentTransaction{},
		&models.PaymentConfig{},
		&models.PaymentCallback{},
		&models.RegistrationLog{},
		&models.SubscriptionLog{},
		&models.BalanceLog{},
		&models.CommissionLog{},
		&models.Node{},
		&models.SystemConfig{},
		&models.CustomNode{},
		&models.UserCustomNode{},
		&models.Notification{},
		&models.EmailQueue{},
		&models.EmailTemplate{},
		&models.Announcement{},
		&models.Ticket{},
		&models.TicketReply{},
		&models.TicketAttachment{},
		&models.TicketRead{},
		&models.Coupon{},
		&models.CouponUsage{},
		&models.RechargeRecord{},
		&models.LoginAttempt{},
		&models.VerificationAttempt{},
		&models.VerificationCode{},
		&models.UserActivity{},
		&models.LoginHistory{},
		&models.AuditLog{},
		&models.TokenBlacklist{},
		&models.CheckinRecord{},
		&models.KnowledgeCategory{},
		&models.KnowledgeArticle{},
		&models.Promotion{},
		&models.PromotionParticipation{},
	)

	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			log.Printf("警告: 迁移过程中检测到已存在的索引，这通常不是问题: %v", err)
		} else {
			return fmt.Errorf("数据库迁移失败: %w", err)
		}
	}
	if !fulfilledAtExisted && DB.Migrator().HasColumn(&models.Order{}, "FulfilledAt") {
		fulfilledAtAdded = true
	}

	if fulfilledAtAdded {
		if err := DB.Exec("UPDATE orders SET fulfilled_at = COALESCE(payment_time, updated_at, created_at) WHERE status = ? AND fulfilled_at IS NULL", "paid").Error; err != nil {
			log.Printf("警告: 回填 orders.fulfilled_at 失败: %v", err)
		}
	}

	log.Println("数据库迁移成功")
	return nil
}

func repairSQLitePromotionParticipationsDDL() error {
	const tableName = "promotion_participations"

	var tableExists int64
	if err := DB.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&tableExists).Error; err != nil {
		return fmt.Errorf("检查 promotion_participations 表失败: %w", err)
	}
	if tableExists == 0 {
		return nil
	}

	var createSQL string
	if err := DB.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&createSQL).Error; err != nil {
		return fmt.Errorf("读取 promotion_participations 表结构失败: %w", err)
	}
	if strings.Contains(createSQL, "`reward_type`") &&
		strings.Contains(createSQL, "CONSTRAINT `fk_promotion_participations_promotion`") &&
		strings.Contains(createSQL, "DEFAULT \"pending\"") {
		return nil
	}

	requiredColumns := []string{
		"id",
		"promotion_id",
		"user_id",
		"order_id",
		"reward_type",
		"reward_value",
		"status",
		"applied_at",
		"expire_at",
		"created_at",
		"updated_at",
	}
	for _, column := range requiredColumns {
		var columnExists int64
		if err := DB.Raw("SELECT COUNT(*) FROM pragma_table_info('promotion_participations') WHERE name=?", column).Scan(&columnExists).Error; err != nil {
			return fmt.Errorf("检查 promotion_participations.%s 列失败: %w", column, err)
		}
		if columnExists == 0 {
			return nil
		}
	}

	log.Println("检测到旧版 promotion_participations 表结构，开始修复 SQLite DDL...")

	var foreignKeys int
	if err := DB.Raw("PRAGMA foreign_keys").Scan(&foreignKeys).Error; err != nil {
		return fmt.Errorf("读取 SQLite foreign_keys 设置失败: %w", err)
	}
	if foreignKeys == 1 {
		if err := DB.Exec("PRAGMA foreign_keys = OFF").Error; err != nil {
			return fmt.Errorf("关闭 SQLite foreign_keys 失败: %w", err)
		}
		defer func() {
			if err := DB.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
				log.Printf("警告: 恢复 SQLite foreign_keys 失败: %v", err)
			}
		}()
	}

	err := DB.Transaction(func(tx *gorm.DB) error {
		statements := []string{
			"DROP TABLE IF EXISTS `promotion_participations__repair`",
			`CREATE TABLE ` + "`promotion_participations__repair`" + ` (
				` + "`id`" + ` integer PRIMARY KEY AUTOINCREMENT,
				` + "`promotion_id`" + ` integer NOT NULL,
				` + "`user_id`" + ` integer NOT NULL,
				` + "`order_id`" + ` integer,
				` + "`reward_type`" + ` varchar(50) NOT NULL,
				` + "`reward_value`" + ` decimal(10,2) NOT NULL,
				` + "`status`" + ` varchar(20) NOT NULL DEFAULT "pending",
				` + "`applied_at`" + ` datetime,
				` + "`expire_at`" + ` datetime,
				` + "`created_at`" + ` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
				` + "`updated_at`" + ` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
				CONSTRAINT ` + "`fk_promotion_participations_promotion`" + ` FOREIGN KEY (` + "`promotion_id`" + `) REFERENCES ` + "`promotions`" + `(` + "`id`" + `) ON DELETE CASCADE,
				CONSTRAINT ` + "`fk_promotion_participations_user`" + ` FOREIGN KEY (` + "`user_id`" + `) REFERENCES ` + "`users`" + `(` + "`id`" + `) ON DELETE CASCADE,
				CONSTRAINT ` + "`fk_promotion_participations_order`" + ` FOREIGN KEY (` + "`order_id`" + `) REFERENCES ` + "`orders`" + `(` + "`id`" + `) ON DELETE SET NULL
			)`,
			`INSERT INTO ` + "`promotion_participations__repair`" + ` (
				` + "`id`" + `, ` + "`promotion_id`" + `, ` + "`user_id`" + `, ` + "`order_id`" + `,
				` + "`reward_type`" + `, ` + "`reward_value`" + `, ` + "`status`" + `,
				` + "`applied_at`" + `, ` + "`expire_at`" + `, ` + "`created_at`" + `, ` + "`updated_at`" + `
			)
			SELECT
				` + "`id`" + `, ` + "`promotion_id`" + `, ` + "`user_id`" + `, ` + "`order_id`" + `,
				COALESCE(` + "`reward_type`" + `, ''), COALESCE(` + "`reward_value`" + `, 0), COALESCE(` + "`status`" + `, 'pending'),
				` + "`applied_at`" + `, ` + "`expire_at`" + `, COALESCE(` + "`created_at`" + `, CURRENT_TIMESTAMP), COALESCE(` + "`updated_at`" + `, CURRENT_TIMESTAMP)
			FROM ` + "`promotion_participations`",
			"DROP TABLE `promotion_participations`",
			"ALTER TABLE `promotion_participations__repair` RENAME TO `promotion_participations`",
			"DELETE FROM sqlite_sequence WHERE name='promotion_participations'",
			"INSERT INTO sqlite_sequence(name, seq) SELECT 'promotion_participations', COALESCE(MAX(`id`), 0) FROM `promotion_participations`",
			"CREATE INDEX IF NOT EXISTS `idx_promotion_participations_promotion_id` ON `promotion_participations`(`promotion_id`)",
			"CREATE INDEX IF NOT EXISTS `idx_promotion_participations_user_id` ON `promotion_participations`(`user_id`)",
			"CREATE INDEX IF NOT EXISTS `idx_promotion_participations_order_id` ON `promotion_participations`(`order_id`)",
			"CREATE INDEX IF NOT EXISTS `idx_promotion_participations_status` ON `promotion_participations`(`status`)",
		}
		for _, statement := range statements {
			if err := tx.Exec(statement).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("修复 promotion_participations 表结构失败: %w", err)
	}

	log.Println("promotion_participations 表结构修复完成")
	return nil
}

func GetDB() *gorm.DB {
	return DB
}

func CloseDatabase() error {
	if DB == nil {
		return nil
	}
	DB.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取底层数据库连接失败: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("关闭数据库失败: %w", err)
	}
	DB = nil
	return nil
}

func ReopenDatabase() error {
	return InitDatabase()
}

// ==========================================
// 数据库辅助函数
// ==========================================

func NullString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}

func NullInt64(i int64) sql.NullInt64 {
	return sql.NullInt64{
		Int64: i,
		Valid: true,
	}
}

func NullFloat64(f float64) sql.NullFloat64 {
	return sql.NullFloat64{
		Float64: f,
		Valid:   true,
	}
}

func NullTime(t time.Time) sql.NullTime {
	return sql.NullTime{
		Time:  t,
		Valid: !t.IsZero(),
	}
}

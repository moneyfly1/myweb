package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
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
		sqlDB.SetMaxOpenConns(3)
		sqlDB.SetMaxIdleConns(2)
		sqlDB.SetConnMaxLifetime(time.Hour)
	} else {
		sqlDB.SetMaxOpenConns(25)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(5 * time.Minute)
	}
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	log.Println("数据库连接成功")
	return nil
}

func AutoMigrate() error {
	if DB == nil {
		return fmt.Errorf("数据库未初始化")
	}
	if strings.Contains(DB.Dialector.Name(), "sqlite") {
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
						log.Printf("添加 display_name 列失败（可能已存在）: %v", err)
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
		&models.Coupon{},
		&models.CouponUsage{},
		&models.RechargeRecord{},
		&models.LoginAttempt{},
		&models.VerificationAttempt{},
		&models.VerificationCode{},
		&models.UserActivity{},
		&models.AuditLog{},
		&models.TokenBlacklist{},
	)

	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			log.Printf("警告: 迁移过程中检测到已存在的索引，这通常不是问题: %v", err)
		} else {
			return fmt.Errorf("数据库迁移失败: %w", err)
		}
	}

	log.Println("数据库迁移成功")
	return nil
}

func GetDB() *gorm.DB {
	return DB
}

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

// InitDatabase 初始化数据库
func InitDatabase() error {
	cfg := config.AppConfig
	if cfg == nil {
		return fmt.Errorf("配置未初始化")
	}

	var dialector gorm.Dialector
	var err error

	// 根据数据库类型选择驱动
	if strings.Contains(cfg.DatabaseURL, "sqlite") {
		// SQLite
		dbPath := strings.Replace(cfg.DatabaseURL, "sqlite:///./", "", 1)
		dbPath = strings.Replace(dbPath, "sqlite:///", "", 1)

		// 转换为绝对路径
		if !filepath.IsAbs(dbPath) {
			dbPath = filepath.Join(".", dbPath)
		}

		dialector = sqlite.Open(dbPath)
	} else if strings.Contains(cfg.DatabaseURL, "mysql") ||
		os.Getenv("USE_MYSQL") == "true" {
		// MySQL
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
		// PostgreSQL
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=5432 sslmode=disable TimeZone=Asia/Shanghai",
			cfg.PostgresServer,
			cfg.PostgresUser,
			cfg.PostgresPass,
			cfg.PostgresDB,
		)
		dialector = postgres.Open(dsn)
	} else {
		// 默认 SQLite
		dbPath := "cboard.db"
		if !filepath.IsAbs(dbPath) {
			dbPath = filepath.Join(".", dbPath)
		}
		dialector = sqlite.Open(dbPath)
	}

	// 配置 GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	if cfg.Debug {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	}

	// 连接数据库
	DB, err = gorm.Open(dialector, gormConfig)
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 配置连接池
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}

	if strings.Contains(cfg.DatabaseURL, "sqlite") {
		// SQLite 优化配置（低配置VPS优化）
		sqlDB.SetMaxOpenConns(3)
		sqlDB.SetMaxIdleConns(2)
		sqlDB.SetConnMaxLifetime(time.Hour)
	} else {
		// MySQL/PostgreSQL 配置
		sqlDB.SetMaxOpenConns(25)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(5 * time.Minute)
	}

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	log.Println("数据库连接成功")
	return nil
}

// AutoMigrate 自动迁移所有模型
func AutoMigrate() error {
	if DB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	// 导入所有模型
	err := DB.AutoMigrate(
		// 用户相关
		&models.User{},
		&models.UserLevel{},
		&models.InviteCode{},
		&models.InviteRelation{},

		// 订阅相关
		&models.Subscription{},
		&models.Device{},
		&models.SubscriptionReset{},

		// 订单和套餐
		&models.Order{},
		&models.Package{},

		// 支付相关
		&models.PaymentTransaction{},
		&models.PaymentConfig{},
		&models.PaymentCallback{},

		// 节点和配置
		&models.Node{},
		&models.SystemConfig{},

		// 通知和邮件
		&models.Notification{},
		&models.EmailQueue{},
		&models.EmailTemplate{},

		// 其他
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
	)

	if err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	log.Println("数据库迁移成功")
	return nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}

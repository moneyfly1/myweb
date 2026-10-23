package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/timeutil"
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
		dbPath := resolveSQLitePath(cfg.DatabaseURL)
		dialector = sqlite.Open(sqliteDSN(dbPath))
		log.Printf("SQLite 数据库路径: %s（%s）", dbPath, dbFileState(dbPath))
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
		dialector = sqlite.Open(sqliteDSN(dbPath))
	}
	// 非 debug 时用 Warn（不是 Silent）：普通 SQL 依旧不打，但超过 SlowThreshold 的慢查询
	// 与 SQL 错误会被记录 —— 线上"某个接口突然变慢"时这是唯一的第一手线索。
	// 历史教训：这里曾是 Silent，慢查询在线上完全不可见，只能靠翻 nginx 日志猜。
	customLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn,
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

	// NowFunc 决定所有 autoCreateTime/autoUpdateTime 字段以及
	// CreatedAt/UpdatedAt 约定字段的时间来源。
	// 历史教训：这里曾把 NowFunc 删掉（以为"VPS 已设北京时间就不需要"），
	// 导致 GORM 回落到 time.Now().Local() —— 随宿主机时区漂移，
	// 在 UTC / -05:00 的机器上写出一批 +00:00 / -05:00 的记录，
	// 与库中多数 +08:00 记录混存（SQLite 按文本比较，直接导致范围查询与排序出错）。
	// 因此必须显式锚定北京时间。
	gormConfig := &gorm.Config{
		Logger:  customLogger,
		NowFunc: timeutil.NowForDB,
	}
	DB, err = gorm.Open(dialector, gormConfig)
	if err != nil {
		// SQLite 打开失败（最常见：文件损坏）：先尝试自动修复（从最近备份恢复），成功则继续启动
		if strings.Contains(cfg.DatabaseURL, "sqlite") {
			curPath := resolveSQLitePath(cfg.DatabaseURL)
			log.Printf("⚠️ SQLite 打开失败（疑似损坏），尝试自动修复: %v", err)
			if sqliteSelfCheckAndRecover(curPath) {
				err = nil // 已从备份恢复并重新打开
			}
		}
		if err != nil {
			return fmt.Errorf("数据库连接失败: %w", err)
		}
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}
	if strings.Contains(cfg.DatabaseURL, "sqlite") {
		// 打开成功后仍做廉价完整性自检：发现损坏自动从最近备份恢复，而不是让服务直接起不来
		curPath := resolveSQLitePath(cfg.DatabaseURL)
		if !sqliteSelfCheckAndRecover(curPath) {
			return fmt.Errorf("数据库完整性自检失败且无可恢复备份，启动中止（详见上方日志）")
		}
		sqlDB, _ = DB.DB()
		// journal_mode / busy_timeout / synchronous / cache_size / mmap_size / temp_store
		// 全部写进 DSN（见 sqliteDSN）：PRAGMA 是**连接级**设置，用 DB.Exec 只会作用于
		// 连接池里当时那一条连接，池扩容后的新连接仍是默认值（busy_timeout=0 → 一有并发写
		// 就立刻 SQLITE_BUSY 报错）。放 DSN 里才能保证每条新连接都带上。
		// SQLite 只有单写者：连接越多，写事务的锁竞争越重（而读在 WAL 下本来就能并发）。
		// 10 条连接换来的是更多的 SQLITE_BUSY，6 条足够覆盖 HTTP 并发读。
		sqlDB.SetMaxOpenConns(6)
		sqlDB.SetMaxIdleConns(6)
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

	// 历史数据去重：invite_relations 新增 InviteeID 唯一索引前，清理重复关系（保留最早一条）
	deduplicateInviteRelations()

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
	// 先确保 system_configs 表存在：金额单位迁移的完成标记依赖它，
	// 且迁移必须在本批 AutoMigrate（会变更 payment_transactions 列类型）之前执行。
	if err := DB.AutoMigrate(&models.SystemConfig{}); err != nil {
		return fmt.Errorf("初始化 system_configs 表失败: %w", err)
	}
	// 历史数据迁移：PaymentTransaction.Amount 曾以「分」存储（int），现统一为「元」。
	// 检测到旧整型列时先 ÷100 转换存量数据，再由下方 AutoMigrate 完成列类型变更（幂等，不会重复执行）。
	if err := migratePaymentTransactionAmountToYuan(); err != nil {
		return err
	}
	err := DB.AutoMigrate(
		&models.User{},
		&models.UserLevel{},
		&models.InviteCode{},
		&models.InviteRelation{},
		&models.Subscription{},
		&models.Device{},
		&models.UserUsageState{}, // 用户使用状态快照（订阅拉取 / 节点按用户上报）
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
		&models.NodeValidationLog{},
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

	// 恢复被重建的 custom_nodes 数据：旧版结构（缺 domain / 含旧字段 / protocol NOT NULL）
	// 触发重建分支时，数据先备份到 custom_nodes_backup / custom_nodes_protocol_backup，
	// 再由本函数在 AutoMigrate 建好新表后按共有列恢复回 custom_nodes，防止升级丢数据。
	if err := restoreCustomNodesFromBackup(); err != nil {
		log.Printf("警告: 恢复 custom_nodes 备份数据失败: %v", err)
	}

	// 自建节点迁移：旧版自建节点存在 nodes 表，现统一挂载到 custom_nodes（专线节点）体系。
	if err := migrateSelfHostNodesToCustom(); err != nil {
		log.Printf("警告: 自建节点迁移失败: %v", err)
	}

	return nil
}

// restoreCustomNodesFromBackup 把重建分支备份的 custom_nodes 数据恢复回新表。
// 兼容两种备份表（旧版结构重建 → custom_nodes_backup；protocol NOT NULL 重建 → custom_nodes_protocol_backup）。
// 按共有列恢复（旧表可能缺少新列，新表可能有旧表没有的默认列）；恢复后删除备份表。
// 幂等：备份表不存在或为空时直接跳过；新表已有数据时不覆盖。
func restoreCustomNodesFromBackup() error {
	if DB == nil || !strings.Contains(DB.Dialector.Name(), "sqlite") {
		return nil
	}
	backupTables := []string{"custom_nodes_backup", "custom_nodes_protocol_backup"}
	for _, backupTable := range backupTables {
		var exists int64
		DB.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", backupTable).Scan(&exists)
		if exists == 0 {
			continue
		}
		// 备份表有数据且新表为空 → 恢复
		var backupCount, newCount int64
		DB.Raw("SELECT COUNT(*) FROM " + backupTable).Scan(&backupCount)
		DB.Raw("SELECT COUNT(*) FROM custom_nodes").Scan(&newCount)
		if backupCount == 0 {
			DB.Exec("DROP TABLE " + backupTable)
			log.Printf("备份表 %s 为空，已清理", backupTable)
			continue
		}
		if newCount > 0 {
			log.Printf("custom_nodes 已有 %d 条数据，跳过从 %s 恢复（保留备份表供人工核对）", newCount, backupTable)
			continue
		}
		// 取两表共有列
		common := commonColumnsOf("custom_nodes", backupTable)
		if len(common) == 0 {
			log.Printf("警告: %s 与 custom_nodes 无共有列，无法自动恢复（保留备份表供人工处理）", backupTable)
			continue
		}
		colList := "`" + strings.Join(common, "`,`") + "`"
		if err := DB.Exec("INSERT INTO custom_nodes (" + colList + ") SELECT " + colList + " FROM " + backupTable).Error; err != nil {
			log.Printf("警告: 从 %s 恢复 custom_nodes 失败: %v（保留备份表）", backupTable, err)
			continue
		}
		log.Printf("✅ 已从备份表 %s 恢复 %d 条 custom_nodes 数据", backupTable, backupCount)
		if err := DB.Exec("DROP TABLE " + backupTable).Error; err != nil {
			log.Printf("警告: 删除备份表 %s 失败: %v", backupTable, err)
		}
	}

	ensurePerformanceIndexes()
	return nil
}

// OptimizeDB 让 SQLite 收集/更新查询计划统计信息（PRAGMA optimize，3.46+ 会按需执行 ANALYZE）。
//
// 为什么需要：本项目从未跑过 ANALYZE（sqlite_stat1 不存在），规划器只能按默认猜测选索引，
// 实测出现过"明明有 (action_type, created_at) 复合索引却选了单列索引再落地临时 B 树排序"的误判。
// PRAGMA optimize 比自己跑 ANALYZE 便宜得多，适合每天低峰期跑一次。
func OptimizeDB() {
	if DB == nil {
		return
	}
	if !strings.Contains(DB.Dialector.Name(), "sqlite") {
		return
	}
	if err := DB.Exec("PRAGMA optimize").Error; err != nil {
		log.Printf("PRAGMA optimize 失败（忽略）: %v", err)
	}
}

// ensurePerformanceIndexes 建 GORM 模型声明不了、但热点查询必需的索引（幂等）。
//
// 这些都是实测出来的：在 30 万行 audit_logs / 15 万行 devices 的等价数据集上，
// 缺索引时的执行计划是 SCAN（全表）或 MATERIALIZE + TEMP B-TREE（落地排序），
// 补上后变成 SEARCH（索引查找），单条查询从 82ms→4ms、76ms→12ms、2ms→0.012ms 量级。
//
//   - users(lower(email)) / users(lower(username))：登录/注册/找回密码全部走 LOWER(email)=?，
//     而 idx_users_email 被函数包住用不上 → 每次登录全表扫 users。
//   - devices(is_active, subscription_id)：订阅管理按「在线设备数」排序时 GROUP BY 需要这个顺序，
//     否则 MATERIALIZE + TEMP B-TREE。
//   - user_custom_nodes(custom_node_id, user_id)：idx_user_node 的前导列是 user_id，
//     按节点反查用户（删节点/改节点时的级联）只能全扫。
//   - login_attempts(created_at)：登录失败列表无条件 ORDER BY created_at。
//   - email_queue(to_email, email_type, created_at)：发信前去重查询（当前全表扫）。
//   - subscription_resets / custom_nodes / devices 的 created_at 排序。
//
// 表达式索引只有 SQLite 才能这样建（MySQL 8 需要函数索引语法且 lower() 依赖 collation），
// 因此仅在 SQLite 上执行；非 SQLite 退化为普通索引里的等价项。
func ensurePerformanceIndexes() {
	if DB == nil {
		return
	}
	isSQLite := strings.Contains(DB.Dialector.Name(), "sqlite")
	var stmts []string
	if isSQLite {
		stmts = append(stmts,
			"CREATE INDEX IF NOT EXISTS idx_users_email_lower ON users(lower(email))",
			"CREATE INDEX IF NOT EXISTS idx_users_username_lower ON users(lower(username))",
		)
	}
	stmts = append(stmts,
		"CREATE INDEX IF NOT EXISTS idx_devices_active_sub ON devices(is_active, subscription_id)",
		"CREATE INDEX IF NOT EXISTS idx_ucn_custom_node ON user_custom_nodes(custom_node_id, user_id)",
		"CREATE INDEX IF NOT EXISTS idx_login_attempts_created_at ON login_attempts(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_email_queue_to_type_created ON email_queue(to_email, email_type, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_email_queue_created_at ON email_queue(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_subreset_created_at ON subscription_resets(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_custom_nodes_created_at ON custom_nodes(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_devices_active_lastaccess ON devices(is_active, kicked_at, last_access)",
		"CREATE INDEX IF NOT EXISTS idx_node_validation_created_at ON node_validation_logs(created_at)",
	)
	for _, stmt := range stmts {
		// 表可能还不存在（首次启动顺序/精简部署），跳过而不是让启动失败
		if err := DB.Exec(stmt).Error; err != nil {
			if !strings.Contains(strings.ToLower(err.Error()), "no such table") {
				log.Printf("警告: 创建索引失败（%s）: %v", stmt, err)
			}
		}
	}
}

// commonColumnsOf 返回两个表共有的列名列表（按 t1 顺序）。
func commonColumnsOf(t1, t2 string) []string {
	cols1 := tableColumnNames(t1)
	cols2 := tableColumnNames(t2)
	set2 := make(map[string]bool, len(cols2))
	for _, c := range cols2 {
		set2[c] = true
	}
	var common []string
	for _, c := range cols1 {
		if set2[c] {
			common = append(common, c)
		}
	}
	return common
}

// tableColumnNames 返回表的列名列表（SQLite pragma）。
func tableColumnNames(table string) []string {
	var cols []string
	rows, err := DB.Raw("PRAGMA table_info(" + table + ")").Rows()
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			continue
		}
		cols = append(cols, name)
	}
	return cols
}

// migrateSelfHostNodesToCustom 把 nodes 表中 self_hosted=true 的记录迁移到 custom_nodes 表。
// 幂等：custom_nodes 已有 install_id 的记录跳过；迁移完成后删除 nodes 表中的自建节点。
func migrateSelfHostNodesToCustom() error {
	var legacyNodes []models.Node
	if err := DB.Where("self_hosted = ?", true).Find(&legacyNodes).Error; err != nil {
		return err
	}
	if len(legacyNodes) == 0 {
		return nil
	}

	migrated := 0
	for _, n := range legacyNodes {
		// 跳过已迁移的（custom_nodes 中存在相同 install_id）
		var count int64
		DB.Model(&models.CustomNode{}).Where("install_id = ?", n.InstallID).Count(&count)
		if count > 0 {
			continue
		}

		cn := models.CustomNode{
			Name:             n.Name,
			DisplayName:      n.Name,
			Protocol:         n.SelfHostProtocol,
			Domain:           extractNodeServerFromConfig(n.Config),
			Port:             extractNodePortFromConfig(n.Config),
			Config:           derefString(n.Config),
			Status:           n.Status,
			IsActive:         n.IsActive,
			Latency:          n.Latency,
			LastTest:         n.LastTest,
			Source:           "selfhost",
			SelfHosted:       true,
			SelfHostProtocol: n.SelfHostProtocol,
			InstallID:        n.InstallID,
			InstallToken:     n.InstallToken,
			InstallExpiresAt: n.InstallExpiresAt,
			LastHeartbeatAt:  n.LastHeartbeatAt,
			InstallCmd:       n.InstallCmd,
			TrafficUp:        n.TrafficUp,
			TrafficDown:      n.TrafficDown,
			TrafficUpdatedAt: n.TrafficUpdatedAt,
			CreatedAt:        n.CreatedAt,
			UpdatedAt:        n.UpdatedAt,
		}
		if err := DB.Create(&cn).Error; err != nil {
			log.Printf("迁移自建节点 %s 失败: %v", n.Name, err)
			continue
		}
		migrated++
	}

	if migrated > 0 {
		// 迁移成功后删除 nodes 表中的自建节点（避免重复出现在普通节点列表）
		if err := DB.Where("self_hosted = ?", true).Delete(&models.Node{}).Error; err != nil {
			return fmt.Errorf("清理 nodes 表自建节点失败: %w", err)
		}
		log.Printf("自建节点迁移完成: %d 个节点已迁移到专线节点体系", migrated)
	}
	return nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// extractNodeServerFromConfig 从节点 Config JSON 中提取 server 字段。
func extractNodeServerFromConfig(config *string) string {
	if config == nil || *config == "" {
		return ""
	}
	var p struct {
		Server string `json:"Server"`
	}
	_ = json.Unmarshal([]byte(*config), &p)
	return p.Server
}

// extractNodePortFromConfig 从节点 Config JSON 中提取 port 字段。
func extractNodePortFromConfig(config *string) int {
	if config == nil || *config == "" {
		return 443
	}
	var p struct {
		Port int `json:"Port"`
	}
	_ = json.Unmarshal([]byte(*config), &p)
	if p.Port <= 0 {
		return 443
	}
	return p.Port
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

// migratePaymentTransactionAmountToYuan 历史数据迁移：
// PaymentTransaction.Amount 曾以「分」（int 列）存储，现统一为「元」。
// 检测旧整型列时把存量数据 ÷100；随后 AutoMigrate 将列改为 decimal(10,2)。
// 幂等：列已是非整型（decimal/numeric/real）时跳过，不会二次换算。
func migratePaymentTransactionAmountToYuan() error {
	if DB == nil || !DB.Migrator().HasTable(&models.PaymentTransaction{}) {
		return nil
	}

	// 迁移完成标记（system_configs）：防止「UPDATE 成功但 AutoMigrate 表重建失败」的中间态
	// 导致下次启动把已是「元」的数据再 ÷100。
	migrationDone := false
	var mark models.SystemConfig
	if err := DB.Where("key = ? AND category = ?", paymentAmountUnitMarkKey, paymentAmountUnitMarkCategory).First(&mark).Error; err == nil && mark.Value == "yuan" {
		migrationDone = true
	}

	dialect := strings.ToLower(DB.Dialector.Name())
	var colType string
	switch {
	case strings.Contains(dialect, "sqlite"):
		DB.Raw("SELECT type FROM pragma_table_info('payment_transactions') WHERE name='amount'").Scan(&colType)
	case strings.Contains(dialect, "mysql"):
		DB.Raw("SELECT DATA_TYPE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'payment_transactions' AND COLUMN_NAME = 'amount'").Scan(&colType)
	case strings.Contains(dialect, "postgres"):
		DB.Raw("SELECT data_type FROM information_schema.columns WHERE table_name = 'payment_transactions' AND column_name = 'amount'").Scan(&colType)
	default:
		return nil
	}

	colType = strings.ToUpper(strings.TrimSpace(colType))
	isLegacyInt := strings.Contains(colType, "INT")

	if migrationDone && isLegacyInt {
		// 数据已换算为元、标记已存在，但列类型仍是整型（上次 AutoMigrate 中途失败）。
		// 只允许 AutoMigrate 改列类型，绝不再 ÷100。
		log.Println("payment_transactions.amount 已完成分→元换算（标记存在），等待 AutoMigrate 变更列类型")
		return nil
	}

	if !isLegacyInt {
		// 列已是非整型（新结构或已由 AutoMigrate 变更）→ 补写完成标记，无需转换
		if !migrationDone {
			_ = setPaymentAmountUnitMark()
		}
		return nil
	}

	log.Printf("检测到 payment_transactions.amount 为整型（历史「分」存储），正在转换为「元」...")
	if err := DB.Exec("UPDATE payment_transactions SET amount = amount / 100.0 WHERE amount IS NOT NULL").Error; err != nil {
		return fmt.Errorf("迁移 payment_transactions.amount 分→元失败: %w", err)
	}
	if err := setPaymentAmountUnitMark(); err != nil {
		// 标记写入失败则拒绝启动：宁可失败也不让下次启动重复 ÷100 造成金额错误
		return fmt.Errorf("写入金额单位迁移标记失败: %w", err)
	}
	log.Println("payment_transactions.amount 分→元 迁移完成（随后 AutoMigrate 变更列类型）")
	return nil
}

const (
	paymentAmountUnitMarkKey      = "payment_amount_unit"
	paymentAmountUnitMarkCategory = "migration"
)

// setPaymentAmountUnitMark 写入「金额单位已统一为元」的迁移完成标记。
func setPaymentAmountUnitMark() error {
	var mark models.SystemConfig
	err := DB.Where("key = ? AND category = ?", paymentAmountUnitMarkKey, paymentAmountUnitMarkCategory).First(&mark).Error
	if err == nil {
		if mark.Value == "yuan" {
			return nil
		}
		mark.Value = "yuan"
		return DB.Save(&mark).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	mark = models.SystemConfig{
		Key:      paymentAmountUnitMarkKey,
		Category: paymentAmountUnitMarkCategory,
		Value:    "yuan",
	}
	return DB.Create(&mark).Error
}

// deduplicateInviteRelations 在 AutoMigrate 前清理 invite_relations 中重复的 InviteeID，
// 保留最早一条记录，否则新增唯一索引会因历史脏数据失败。
func deduplicateInviteRelations() {
	if DB == nil {
		return
	}
	if !DB.Migrator().HasTable(&models.InviteRelation{}) {
		return
	}
	// 找出重复的 invitee_id（出现次数 > 1）
	var dupInvitees []struct {
		InviteeID uint
		Cnt       int64
	}
	if err := DB.Table("invite_relations").
		Select("invitee_id, COUNT(*) as cnt").
		Group("invitee_id").
		Having("COUNT(*) > 1").
		Scan(&dupInvitees).Error; err != nil {
		log.Printf("警告: 检查 invite_relations 重复数据失败: %v", err)
		return
	}
	removed := int64(0)
	for _, dup := range dupInvitees {
		// 保留最早一条（id 最小），删除其余
		res := DB.Table("invite_relations").
			Where("invitee_id = ? AND id NOT IN (SELECT MIN(id) FROM invite_relations WHERE invitee_id = ?)", dup.InviteeID, dup.InviteeID).
			Delete(&models.InviteRelation{})
		if res.Error != nil {
			log.Printf("警告: 清理 invite_relations 重复数据失败 (invitee_id=%d): %v", dup.InviteeID, res.Error)
			continue
		}
		removed += res.RowsAffected
	}
	if removed > 0 {
		log.Printf("invite_relations 历史重复数据已清理: %d 条 (为唯一索引做准备)", removed)
	}
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

// dbFileState 返回 SQLite 数据库文件的状态描述：存在或将被新建。
// resolveSQLitePath 解析 sqlite 数据库文件绝对路径。
// 相对路径统一锚定到可执行文件所在目录，避免因启动目录不同而静默新建数据库
// （例如从其它目录手动启动 ./server 会在该目录生成一个全新的空库，造成"数据丢失"假象）。
// ResolveSQLitePath 把 DATABASE_URL 解析成本地 SQLite 文件路径
// （去前缀 + 相对路径锚定到可执行文件目录）。供需要"落盘前检查库文件"的命令复用。
func ResolveSQLitePath(databaseURL string) string { return resolveSQLitePath(databaseURL) }

func resolveSQLitePath(databaseURL string) string {
	dbPath := strings.Replace(databaseURL, "sqlite:///./", "", 1)
	dbPath = strings.Replace(dbPath, "sqlite:///", "", 1)
	if !filepath.IsAbs(dbPath) {
		if exePath, exeErr := os.Executable(); exeErr == nil {
			dbPath = filepath.Join(filepath.Dir(exePath), dbPath)
		} else {
			dbPath = filepath.Join(".", dbPath)
		}
	}
	return dbPath
}

func dbFileState(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "文件不存在，将创建全新数据库"
	}
	if info.IsDir() {
		return "路径是目录（配置异常）"
	}
	return "已存在，使用现有数据库"
}

// IsSQLiteFreshDB 判断 SQLite 数据库文件是否不存在（即将新建）。
// 供启动流程检测"将创建全新数据库"的场景并告警。
func IsSQLiteFreshDB() bool {
	if DB == nil {
		return false
	}
	cfg := config.AppConfig
	if cfg == nil || !strings.Contains(cfg.DatabaseURL, "sqlite") {
		return false
	}
	_, err := os.Stat(resolveSQLitePath(cfg.DatabaseURL))
	return os.IsNotExist(err)
}

// sqliteSelfCheckAndRecover 对 SQLite 数据库做廉价完整性自检；
// 发现损坏时依次尝试：① 从 uploads/backups/upgrade_pre_*.db 最近备份恢复，
// ② 从 uploads/backups/backup_*.zip 中解压恢复，③ 从同目录 .db.backup* 恢复。
// 恢复成功返回 true（已重新打开 DB）；失败返回 false（调用方中止启动）。
// sqliteQuickCheckOK 对已打开的 SQLite 连接做廉价完整性自检。
// gorm.Open 失败时可能返回部分初始化的非 nil DB 对象，因此用 recover 兜底。
func sqliteQuickCheckOK() (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("SQLite 自检异常（panic 已捕获）: %v", r)
			ok = false
		}
	}()
	if DB == nil {
		return false
	}
	var n int
	err := DB.Raw("SELECT COUNT(*) FROM sqlite_master").Scan(&n).Error
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "malformed") || strings.Contains(msg, "corrupt") || strings.Contains(msg, "disk i/o error") {
			log.Printf("⚠️ 检测到数据库损坏: %v", err)
			return false
		}
		// 非损坏类错误（如锁/瞬时），按正常处理
		log.Printf("SQLite 自检异常（非损坏类）: %v", err)
		return true
	}
	return true
}

func sqliteSelfCheckAndRecover(dbPath string) bool {
	// 情况 A：自检通过（DB 正常）→ 直接返回
	if sqliteQuickCheckOK() {
		return true
	}
	// 情况 B：DB 无效或打开失败/损坏 → 尝试从最近备份自动恢复
	log.Println("======================================================")
	log.Printf("⚠️  数据库打开失败或完整性自检未通过: %s", dbPath)
	log.Println("    正在尝试从最近备份自动恢复...")
	log.Println("======================================================")

	// 候选备份：upgrade_pre_*.db（升级自动备份）→ 同目录 *.db 备份 → zip 备份
	var candidates []string
	backupDirs := []string{}
	cfg := config.AppConfig
	if cfg != nil && cfg.UploadDir != "" {
		backupDirs = append(backupDirs, filepath.Join(cfg.UploadDir, "backups"))
	}
	backupDirs = append(backupDirs, filepath.Dir(dbPath))

	for _, dir := range backupDirs {
		matches, _ := filepath.Glob(filepath.Join(dir, "upgrade_pre_*.db"))
		for _, m := range matches {
			candidates = append(candidates, m)
		}
	}
	// 同目录历史备份（形如 cboard.db.backup* / cboard.db.corrupt.bak* 不取，取 .backup 与手工备份）
	for _, dir := range backupDirs {
		matches, _ := filepath.Glob(filepath.Join(dir, "*.db.backup*"))
		for _, m := range matches {
			candidates = append(candidates, m)
		}
	}

	if len(candidates) == 0 {
		log.Println("❌ 未找到任何可用备份，无法自动恢复。请人工处理：")
		log.Printf("   1) 检查 %s 是否有手动备份", filepath.Dir(dbPath))
		log.Printf("   2) 或使用 install.sh 选项13 回滚到升级前版本")
		return false
	}
	// 取最新的
	latest := candidates[0]
	for _, c := range candidates[1:] {
		info1, e1 := os.Stat(latest)
		info2, e2 := os.Stat(c)
		if e1 == nil && e2 == nil && info2.ModTime().After(info1.ModTime()) {
			latest = c
		}
	}

	// 先把损坏文件留底
	corruptBak := dbPath + ".corrupt." + time.Now().Format("20060102_150405")
	if err := os.Rename(dbPath, corruptBak); err != nil {
		log.Printf("⚠️ 备份损坏文件失败: %v", err)
	}

	log.Printf("正在用备份恢复: %s", latest)
	if err := copyFile(latest, dbPath); err != nil {
		log.Printf("❌ 从备份恢复失败: %v", err)
		return false
	}

	// 重新打开
	sqlDB, err := DB.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
	DB, err = gorm.Open(sqlite.Open(sqliteDSN(dbPath)), &gorm.Config{
		Logger:  logger.Discard,
		NowFunc: timeutil.NowForDB,
	})
	if err != nil {
		log.Printf("❌ 恢复后重新打开失败: %v", err)
		return false
	}
	var n2 int
	if err := DB.Raw("SELECT COUNT(*) FROM sqlite_master").Scan(&n2).Error; err != nil {
		log.Printf("❌ 恢复后的库仍不可用: %v", err)
		return false
	}
	log.Println("✅ 数据库已从备份自动恢复，服务继续启动")
	return true
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// sqliteDSN 给 SQLite 连接串补上时区参数。
//
// 不加 _loc 时：驱动会把「不带时区偏移」的时间串按 UTC 解析读回，
// 之后再被 db.Save() 全字段重写时就固化成 +00:00，与库中 +08:00 的记录混存
// （SQLite 的 datetime 是文本，跨偏移比较等同于按墙钟字符串比较 → 查询结果错误）。
// 加上 _loc=Asia/Shanghai 后，无偏移值按北京时间解释、写回统一带 +08:00。
func sqliteDSN(dbPath string) string {
	base, rawQuery := dbPath, ""
	if i := strings.Index(dbPath, "?"); i >= 0 {
		base, rawQuery = dbPath[:i], dbPath[i+1:]
	}
	params := url.Values{}
	if rawQuery != "" {
		if parsed, err := url.ParseQuery(rawQuery); err == nil {
			params = parsed
		}
	}
	setDefault := func(key, value string) {
		if params.Get(key) == "" {
			params.Set(key, value)
		}
	}
	// 时区：不加 _loc 时驱动会把「不带偏移」的时间串按 UTC 读回，写回时固化成 +00:00，
	// 与库中 +08:00 的记录混存（SQLite 时间是文本，跨偏移比较等于按墙钟字符串比较）。
	setDefault("_loc", "Asia/Shanghai")
	// WAL：读写不互斥，单进程多 goroutine 下并发读显著变快（库级持久设置，写一次即生效）。
	setDefault("_journal_mode", "WAL")
	// busy_timeout：并发写时等待而不是立刻报 database is locked。
	setDefault("_busy_timeout", "5000")
	// synchronous=NORMAL：WAL 下的推荐值，掉电最多丢最近若干事务，不必每次 fsync。
	setDefault("_synchronous", "NORMAL")
	// cache_size 负值 = KB：-32000 ≈ 32MB 页缓存，大幅减少热点表（设备/订阅/配置）的读盘。
	// 注意：mattn/go-sqlite3 的 DSN 只支持上表这些 _pragma 参数（已实测生效），
	// _temp_store / _mmap_size 它不是 DSN 参数、传了会被静默忽略，所以不要加进来假装优化。
	setDefault("_cache_size", "-32000")
	return base + "?" + params.Encode()
}

// SQLite → MySQL 数据迁移脚本
//
// 用法（在生产服务器上，项目根目录执行）：
//
//	go run ./cmd/migrate -sqlite ./cboard.db -mysql "cboard:密码@tcp(127.0.0.1:3306)/cboard?charset=utf8mb4&parseTime=True&loc=Local"
//
// 只读源库（SQLite），只写目标库（MySQL）。迁移前请确保已备份 SQLite 文件。
package main

import (
	"flag"
	"log"
	"reflect"

	"cboard-go/internal/models"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	sqlitePath := flag.String("sqlite", "./cboard.db", "SQLite 数据库文件路径")
	mysqlDSN := flag.String("mysql", "", "MySQL DSN")
	flag.Parse()

	if *mysqlDSN == "" {
		log.Fatal("必须提供 -mysql DSN")
	}

	// 连接源库（SQLite，只读）
	sqliteDB, err := gorm.Open(sqlite.Open(*sqlitePath), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接 SQLite 失败: %v", err)
	}

	// 连接目标库（MySQL）
	// DisableForeignKeyConstraintWhenMigrating: 建表时不创建外键约束，
	// 避免多表互相引用导致的建表顺序问题（数据完整性由应用层保证，与 SQLite 行为一致）。
	mysqlDB, err := gorm.Open(mysql.Open(*mysqlDSN), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		log.Fatalf("连接 MySQL 失败: %v", err)
	}

	// 强制连接使用 utf8mb4 字符集。生产库的 system_configs.config_update_logs 含 4 字节
	// emoji（📊🌐📋 等），若连接回落到 utf8mb3 会在插入时报 Error 1366 "Incorrect string value"。
	if err := mysqlDB.Exec("SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci").Error; err != nil {
		log.Fatalf("设置 MySQL 连接字符集失败: %v", err)
	}

	// 1. MySQL 建表（AutoMigrate，与 database.AutoMigrate 保持一致的模型集合）
	log.Println("=== 步骤1：MySQL 建表 ===")
	modelsList := allModels()
	if err := mysqlDB.AutoMigrate(modelsList...); err != nil {
		log.Fatalf("MySQL AutoMigrate 失败: %v", err)
	}
	log.Printf("✅ 已在 MySQL 建表 %d 个模型", len(modelsList))

	// 1.5. 排序规则对齐：SQLite 唯一索引区分大小写（BINARY），MySQL 默认 utf8mb4_unicode_ci
	// 不区分大小写。生产库存在 'Robert'/'robert'、'Alex'/'alex' 等 30 对大小写不同但都合法的
	// 用户名，直接把 username/email 改为 utf8mb4_bin（区分大小写），与 SQLite 行为一致，
	// 避免插入时 Error 1062 "Duplicate entry"。
	log.Println("=== 步骤1.5：调整 users 排序规则为区分大小写（对齐 SQLite） ===")
	alterStmts := []string{
		"ALTER TABLE users MODIFY username varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL",
		"ALTER TABLE users MODIFY email varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL",
	}
	for _, stmt := range alterStmts {
		if err := mysqlDB.Exec(stmt).Error; err != nil {
			log.Fatalf("调整排序规则失败: %v", err)
		}
	}
	log.Println("✅ 已调整 users.username/email 为 utf8mb4_bin")

	// 1.6. 诊断：打印连接池实际字符集 + emoji 插入测试（排查 Error 1366）
	var csClient, csConn, csColl string
	if err := mysqlDB.Raw("SELECT @@character_set_client, @@character_set_connection, @@collation_connection").Row().Scan(&csClient, &csConn, &csColl); err != nil {
		log.Printf("  ⚠️ 读取连接字符集失败: %v", err)
	} else {
		log.Printf("  [诊断] character_set_client=%s connection=%s collation=%s", csClient, csConn, csColl)
	}
	if err := mysqlDB.Exec("INSERT INTO system_configs (`key`, value, type, category, display_name, description, is_public, sort_order, created_at, updated_at) VALUES ('__diag_emoji__', '测试😀emoji', 'text', 'diag', 'd', '', 0, 0, NOW(), NOW())").Error; err != nil {
		log.Printf("  [诊断] emoji 插入失败: %v", err)
	} else {
		log.Println("  [诊断] emoji 插入成功（连接为 utf8mb4）")
	}
	mysqlDB.Exec("DELETE FROM system_configs WHERE `key` = '__diag_emoji__'")

	// 2. 逐表迁移数据
	log.Println("=== 步骤2：迁移数据 ===")
	results := migrateList(sqliteDB, mysqlDB)
	var total int64
	for _, r := range results {
		log.Printf("  ✓ %s: %d 行", r.name, r.count)
		total += r.count
	}
	log.Printf("✅ 迁移完成，共 %d 行", total)

	// 3. 行数核对（只读，双库各查一遍，不一致大声报错）
	log.Println("=== 步骤3：行数核对（SQLite vs MySQL） ===")
	verifyCounts(sqliteDB, mysqlDB)
}

// allModels 返回与 database.AutoMigrate 一致的模型集合（去重）
func allModels() []interface{} {
	return []interface{}{
		&models.SystemConfig{},
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
	}
}

type migrateResult struct {
	name  string
	count int64
}

// migrateList 逐表迁移数据，返回每表结果。
// 使用泛型 + GORM 的 Find/CreateInBatches，类型映射与主键保留由 GORM 自动处理。
func migrateList(sqliteDB, mysqlDB *gorm.DB) []migrateResult {
	var results []migrateResult

	results = append(results, migrate[models.SystemConfig](sqliteDB, mysqlDB))
	results = append(results, migrate[models.User](sqliteDB, mysqlDB))
	results = append(results, migrate[models.UserLevel](sqliteDB, mysqlDB))
	results = append(results, migrate[models.InviteCode](sqliteDB, mysqlDB))
	results = append(results, migrate[models.InviteRelation](sqliteDB, mysqlDB))
	results = append(results, migrate[models.Subscription](sqliteDB, mysqlDB))
	results = append(results, migrate[models.Device](sqliteDB, mysqlDB))
	results = append(results, migrate[models.SubscriptionReset](sqliteDB, mysqlDB))
	results = append(results, migrate[models.Order](sqliteDB, mysqlDB))
	results = append(results, migrate[models.Package](sqliteDB, mysqlDB))
	results = append(results, migrate[models.PaymentTransaction](sqliteDB, mysqlDB))
	results = append(results, migrate[models.PaymentConfig](sqliteDB, mysqlDB))
	results = append(results, migrate[models.PaymentCallback](sqliteDB, mysqlDB))
	results = append(results, migrate[models.RegistrationLog](sqliteDB, mysqlDB))
	results = append(results, migrate[models.SubscriptionLog](sqliteDB, mysqlDB))
	results = append(results, migrate[models.BalanceLog](sqliteDB, mysqlDB))
	results = append(results, migrate[models.CommissionLog](sqliteDB, mysqlDB))
	results = append(results, migrate[models.Node](sqliteDB, mysqlDB))
	results = append(results, migrate[models.CustomNode](sqliteDB, mysqlDB))
	results = append(results, migrate[models.UserCustomNode](sqliteDB, mysqlDB))
	results = append(results, migrate[models.Notification](sqliteDB, mysqlDB))
	results = append(results, migrate[models.EmailQueue](sqliteDB, mysqlDB))
	results = append(results, migrate[models.EmailTemplate](sqliteDB, mysqlDB))
	results = append(results, migrate[models.Announcement](sqliteDB, mysqlDB))
	results = append(results, migrate[models.Ticket](sqliteDB, mysqlDB))
	results = append(results, migrate[models.TicketReply](sqliteDB, mysqlDB))
	results = append(results, migrate[models.TicketAttachment](sqliteDB, mysqlDB))
	results = append(results, migrate[models.TicketRead](sqliteDB, mysqlDB))
	results = append(results, migrate[models.Coupon](sqliteDB, mysqlDB))
	results = append(results, migrate[models.CouponUsage](sqliteDB, mysqlDB))
	results = append(results, migrate[models.RechargeRecord](sqliteDB, mysqlDB))
	results = append(results, migrate[models.LoginAttempt](sqliteDB, mysqlDB))
	results = append(results, migrate[models.VerificationAttempt](sqliteDB, mysqlDB))
	results = append(results, migrate[models.VerificationCode](sqliteDB, mysqlDB))
	results = append(results, migrate[models.UserActivity](sqliteDB, mysqlDB))
	results = append(results, migrate[models.LoginHistory](sqliteDB, mysqlDB))
	results = append(results, migrate[models.AuditLog](sqliteDB, mysqlDB))
	results = append(results, migrate[models.TokenBlacklist](sqliteDB, mysqlDB))
	results = append(results, migrate[models.CheckinRecord](sqliteDB, mysqlDB))
	results = append(results, migrate[models.KnowledgeCategory](sqliteDB, mysqlDB))
	results = append(results, migrate[models.KnowledgeArticle](sqliteDB, mysqlDB))
	results = append(results, migrate[models.Promotion](sqliteDB, mysqlDB))
	results = append(results, migrate[models.PromotionParticipation](sqliteDB, mysqlDB))

	return results
}

// migrate 迁移单个模型：从 SQLite 读取全部记录，批量插入 MySQL（保留主键）。
func migrate[T any](sqliteDB, mysqlDB *gorm.DB) migrateResult {
	var records []T
	name := realTableName(sqliteDB, new(T))

	if err := sqliteDB.Find(&records).Error; err != nil {
		log.Printf("  ❌ %s 读取失败: %v", name, err)
		return migrateResult{name: name, count: 0}
	}
	if len(records) == 0 {
		return migrateResult{name: name, count: 0}
	}

	// CreateInBatches 批量插入（保留主键，因为记录里主键有值）
	if err := mysqlDB.CreateInBatches(records, 500).Error; err != nil {
		log.Printf("  ❌ %s 插入失败: %v", name, err)
		return migrateResult{name: name, count: 0}
	}
	return migrateResult{name: name, count: int64(len(records))}
}

// realTableName 通过 GORM 解析模型得到真实表名（尊重 TableName() 覆盖，如 LoginHistory→login_history）
func realTableName(db *gorm.DB, model interface{}) string {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		return reflect.TypeOf(model).Elem().Name()
	}
	return stmt.Schema.Table
}

// verifyCounts 只读核对 SQLite 与 MySQL 各表行数，不一致或计数失败时大声报错退出。
func verifyCounts(sqliteDB, mysqlDB *gorm.DB) {
	models := allModels()
	mismatch := 0
	for _, m := range models {
		table := realTableName(sqliteDB, m)
		var sc, mc int64
		if err := sqliteDB.Table(table).Count(&sc).Error; err != nil {
			log.Printf("  ⚠️ %s SQLite 计数失败: %v", table, err)
			mismatch++
			continue
		}
		if err := mysqlDB.Table(table).Count(&mc).Error; err != nil {
			log.Printf("  ⚠️ %s MySQL 计数失败: %v", table, err)
			mismatch++
			continue
		}
		status := "✓"
		if sc != mc {
			status = "❌ 不一致"
			mismatch++
		}
		log.Printf("  %s %s: sqlite=%d mysql=%d", status, table, sc, mc)
	}
	if mismatch > 0 {
		log.Fatalf("❌ 有 %d 张表行数不一致或计数失败，请检查", mismatch)
	}
	log.Println("✅ 全部表行数一致")
}

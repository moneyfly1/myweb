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
	"gorm.io/gorm/schema"
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
	mysqlDB, err := gorm.Open(mysql.Open(*mysqlDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接 MySQL 失败: %v", err)
	}

	// 1. MySQL 建表（AutoMigrate，与 database.AutoMigrate 保持一致的模型集合）
	log.Println("=== 步骤1：MySQL 建表 ===")
	modelsList := allModels()
	if err := mysqlDB.AutoMigrate(modelsList...); err != nil {
		log.Fatalf("MySQL AutoMigrate 失败: %v", err)
	}
	log.Printf("✅ 已在 MySQL 建表 %d 个模型", len(modelsList))

	// 2. 逐表迁移数据
	log.Println("=== 步骤2：迁移数据 ===")
	var total int64
	for _, r := range migrateList(sqliteDB, mysqlDB) {
		log.Printf("  ✓ %s: %d 行", r.name, r.count)
		total += r.count
	}
	log.Printf("✅ 迁移完成，共 %d 行", total)
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
	name := tableNameOf[T]()

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

// tableNameOf 通过 GORM 命名策略获取模型的表名（正确处理复数化，如 LoginHistory→login_histories）
func tableNameOf[T any]() string {
	var m T
	t := reflect.TypeOf(m)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	ns := schema.NamingStrategy{}
	return ns.TableName(t.Name())
}

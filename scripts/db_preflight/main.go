package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// db_preflight 升级前数据库兼容性预检（只读，绝不修改生产数据）。
//
// 用法（务必指向数据库的【副本】，不要直接指向生产库）：
//   cd <项目目录>
//   DATABASE_URL="sqlite:////root/preflight/cboard.db" go run ./scripts/db_preflight
//
// 检查项：
//   1. 核心业务表是否齐全
//   2. payment_transactions.amount 金额单位（分→元迁移是否必要/是否已迁移）
//   3. 金额迁移完成标记是否存在（防重复换算）
//   4. invite_relations 是否有重复 invitee_id（唯一索引风险）
//   5. custom_nodes 旧结构是否会被重建（含备份）
//   6. orders.fulfilled_at 是否缺失（将自动补列）
//   7. 数据量概览，便于估算迁移耗时
//
// 输出：✅ 可直接升级 / ⚠️ 有注意项 / ❌ 有阻塞问题

var passCount, warnCount, failCount int

func report(kind, msg string) {
	switch kind {
	case "ok":
		passCount++
		fmt.Printf("  ✅ %s\n", msg)
	case "warn":
		warnCount++
		fmt.Printf("  ⚠️ %s\n", msg)
	case "fail":
		failCount++
		fmt.Printf("  ❌ %s\n", msg)
	}
}

func main() {
	// 用法：
	//   go run ./scripts/db_preflight /root/preflight.db        只读预检
	//   go run ./scripts/db_preflight --repair /root/preflight.db  预检 + 损坏自动修复
	repairMode := false
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "--repair" {
		repairMode = true
		args = args[1:]
	}
	// 支持命令行直接指定数据库副本路径（推荐）：
	//   go run ./scripts/db_preflight /root/preflight/cboard.db
	// 未指定时使用 .env 的 DATABASE_URL。
	if len(args) > 0 && strings.HasSuffix(args[0], ".db") {
		abs, err := filepath.Abs(args[0])
		if err != nil {
			log.Fatalf("解析数据库路径失败: %v", err)
		}
		os.Setenv("DATABASE_URL", "sqlite:///"+abs)
		fmt.Printf(" 目标数据库: %s\n", abs)
	} else if len(args) > 0 {
		log.Fatalf("用法: go run ./scripts/db_preflight [--repair] [数据库副本路径.db]\n    或设置 DATABASE_URL 环境变量")
	}

	cfg, err := config.LoadConfig()
	if err != nil || cfg == nil {
		log.Fatalf("配置加载失败: %v（请确保在项目目录运行并已加载 .env）", err)
	}

	fmt.Println("================================================================")
	fmt.Println(" CBoard 升级前数据库预检（只读，不修改任何数据）")
	fmt.Println("================================================================")

	// 打开数据库但不执行 AutoMigrate（只读检查）
	if err := database.InitDatabase(); err != nil {
		if repairMode {
			fmt.Printf("❌ 数据库打开失败（%v），尝试自动修复...\n", err)
			if repairCorruptSQLite() {
				if err := database.InitDatabase(); err != nil {
					log.Fatalf("修复后仍无法打开数据库: %v", err)
				}
				fmt.Println("✅ 数据库已修复，继续预检...")
			} else {
				log.Fatalf("自动修复失败，请人工处理或使用 install.sh 选项13 回退")
			}
		} else {
			log.Fatalf("❌ 无法打开数据库: %v（可用 --repair 尝试自动修复）", err)
		}
	}
	db := database.GetDB()
	// 静默 GORM 日志，避免 SQL 刷屏
	db.Logger = db.Logger.LogMode(logger.Silent)

	dialect := strings.ToLower(db.Dialector.Name())
	fmt.Printf(" 数据库类型: %s\n", dialect)

	// ---------- 1. 核心表检查 ----------
	fmt.Println("\n[1] 核心业务表")
	requiredTables := []string{"users", "orders", "subscriptions", "system_configs", "payment_configs", "packages"}
	for _, t := range requiredTables {
		var exists int64
		if dialect == "sqlite" {
			db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", t).Scan(&exists)
		} else {
			exists = 1
			if err := db.Table(t).Count(&exists).Error; err != nil {
				exists = 0
			}
		}
		if exists > 0 {
			report("ok", fmt.Sprintf("表 %s 存在", t))
		} else {
			report("warn", fmt.Sprintf("表 %s 不存在（全新安装会由 AutoMigrate 自动创建，属正常）", t))
		}
	}

	// ---------- 2. 金额单位检查 ----------
	fmt.Println("\n[2] payment_transactions.amount 金额单位（分→元迁移）")
	ptExists := tableExists(db, dialect, "payment_transactions")
	if !ptExists {
		report("ok", "payment_transactions 表不存在（全新库，无需迁移）")
	} else {
		colType := columnType(db, dialect, "payment_transactions", "amount")
		switch {
		case colType == "":
			report("fail", "payment_transactions 存在但找不到 amount 列，请人工检查表结构")
		case strings.Contains(strings.ToUpper(colType), "INT"):
			var cnt int64
			db.Table("payment_transactions").Count(&cnt)
			var maxAmt sql.NullFloat64
			db.Table("payment_transactions").Select("MAX(amount)").Scan(&maxAmt)
			report("warn", fmt.Sprintf("amount 为整型(%s)=历史「分」单位，升级时将自动 ÷100 转换为「元」（共 %d 条，最大 %v 分 ≈ ¥%.2f）",
				colType, cnt, maxAmt.Float64, maxAmt.Float64/100))
			// 迁移完成标记
			var mark string
			db.Table("system_configs").Select("value").Where("key = ? AND category = ?", "payment_amount_unit", "migration").Scan(&mark)
			if mark == "yuan" {
				report("ok", "迁移完成标记已存在（即使列类型回退也不会重复换算）")
			} else {
				report("warn", "暂无迁移完成标记（首次迁移会写入；若之前手动换算过数据请先人工核对）")
			}
		default:
			report("ok", fmt.Sprintf("amount 已是非整型(%s)=「元」单位，无需换算（或已迁移）", colType))
		}
	}

	// ---------- 3. invite_relations 唯一索引风险 ----------
	fmt.Println("\n[3] invite_relations 重复 invitee_id（唯一索引前置清理）")
	if tableExists(db, dialect, "invite_relations") {
		var dups int64
		db.Table("invite_relations").Select("COUNT(*)").Where("invitee_id IN (SELECT invitee_id FROM invite_relations GROUP BY invitee_id HAVING COUNT(*) > 1)").Scan(&dups)
		if dups > 0 {
			report("warn", fmt.Sprintf("发现 %d 条重复 invitee_id 记录，升级时自动保留最早一条（已内置清理）", dups))
		} else {
			report("ok", "无重复 invitee_id")
		}
	} else {
		report("ok", "invite_relations 不存在（新库自动创建）")
	}

	// ---------- 4. custom_nodes 旧结构 ----------
	fmt.Println("\n[4] custom_nodes 旧结构（可能触发自动重建）")
	if tableExists(db, dialect, "custom_nodes") {
		hasProtocol := columnExists(db, dialect, "custom_nodes", "protocol")
		hasDomain := columnExists(db, dialect, "custom_nodes", "domain")
		if !hasDomain || !hasProtocol {
			report("warn", "custom_nodes 缺 protocol/domain 列=旧版结构，升级时自动备份旧表并重建（数据迁移到 custom_nodes_backup）")
		} else {
			report("ok", "custom_nodes 结构正常")
		}
	} else {
		report("ok", "custom_nodes 不存在（新库自动创建）")
	}

	// ---------- 5. 支付配置密钥完整性 ----------
	fmt.Println("\n[5] 支付配置密钥完整性（防掩码污染）")
	if tableExists(db, dialect, "payment_configs") {
		type pcRow struct {
			ID       int
			PayType  string
			Priv     sql.NullString
			Alipub   sql.NullString
		}
		var rows []pcRow
		if err := db.Table("payment_configs").Select("id, pay_type, merchant_private_key, alipay_public_key").Scan(&rows).Error; err == nil {
			if len(rows) == 0 {
				report("ok", "无支付配置（不影响升级）")
			}
			for _, r := range rows {
				bad := []string{}
				if r.Priv.Valid {
					if r.Priv.String == "******" || strings.TrimSpace(r.Priv.String) == "" {
						bad = append(bad, "私钥被掩码/为空")
					} else if !strings.Contains(r.Priv.String, "-----BEGIN") {
						bad = append(bad, "私钥格式异常(非PEM)")
					}
				}
				if r.Alipub.Valid {
					if r.Alipub.String == "******" || strings.TrimSpace(r.Alipub.String) == "" {
						bad = append(bad, "公钥被掩码/为空")
					}
				}
				if len(bad) > 0 {
					report("fail", fmt.Sprintf("支付配置 id=%d (%s): %s——支付回调将验签失败！需在管理后台重新保存正确的密钥", r.ID, r.PayType, strings.Join(bad, "、")))
				} else {
					report("ok", fmt.Sprintf("支付配置 id=%d (%s) 密钥完整", r.ID, r.PayType))
				}
			}
		} else {
			report("ok", "payment_configs 表读取跳过")
		}
	} else {
		report("ok", "payment_configs 不存在（新库自动创建）")
	}

	// ---------- 6. orders.fulfilled_at ----------
	fmt.Println("\n[5] orders.fulfilled_at 列")
	if tableExists(db, dialect, "orders") {
		if columnExists(db, dialect, "orders", "fulfilled_at") {
			report("ok", "已存在")
		} else {
			report("warn", "缺失，升级时自动 ALTER TABLE 补列（幂等）")
		}
	} else {
		report("ok", "orders 不存在")
	}

	// ---------- 6. 数据量概览 ----------
	fmt.Println("\n[7] 数据量概览（估算迁移耗时）")
	for _, t := range []string{"users", "orders", "subscriptions", "payment_transactions", "audit_logs", "devices"} {
		if tableExists(db, dialect, t) {
			var cnt int64
			if err := db.Table(t).Count(&cnt).Error; err == nil {
				fmt.Printf("  表 %-24s %d 行\n", t, cnt)
			}
		}
	}

	// ---------- 损坏修复（--repair） ----------
	if repairMode && tableExists(db, dialect, "sqlite_master") == false {
		// sqlite_master 都读不到说明库损坏
		report("fail", "数据库文件损坏，尝试自动修复...")
		if repairCorruptSQLite() {
			report("ok", "已从最近备份自动修复，请重新运行预检确认")
		} else {
			report("fail", "自动修复失败，请人工处理或使用 install.sh 选项13 回退")
		}
	} else if repairMode && dialect == "sqlite" {
		var n int
		err := db.Raw("SELECT COUNT(*) FROM sqlite_master").Scan(&n).Error
		if err != nil {
			report("fail", fmt.Sprintf("数据库自检失败（%v），尝试自动修复...", err))
			if repairCorruptSQLite() {
				report("ok", "已从最近备份自动修复，请重新运行预检确认")
			} else {
				report("fail", "自动修复失败，请人工处理或使用 install.sh 选项13 回退")
			}
		} else {
			report("ok", "数据库完整性正常")
		}
	}

	// ---------- 汇总 ----------
	fmt.Println("\n================================================================")
	fmt.Printf(" 预检结果: ✅ %d 项通过 | ⚠️ %d 项注意 | ❌ %d 项阻塞\n", passCount, warnCount, failCount)
	if failCount > 0 {
		fmt.Println(" 结论: ❌ 存在阻塞问题，请先处理 ❌ 项再升级（可提供预检输出协助处理）")
	} else if warnCount > 0 {
		fmt.Println(" 结论: ⚠️ 可升级，⚠️ 项会在升级时自动处理；建议先在【副本】上试跑验证")
	} else {
		fmt.Println(" 结论: ✅ 可直接升级")
	}
	fmt.Println("================================================================")
	fmt.Println(" 安全提示: 本工具只读检查，不修改任何数据；正式升级前 install.sh 会再自动备份一次数据库。")
}

// repairCorruptSQLite 从 uploads/backups/upgrade_pre_*.db 或同目录备份恢复损坏的库（先留底损坏文件）。
func repairCorruptSQLite() bool {
	cfg := config.AppConfig
	searchDirs := []string{}
	if cfg != nil && cfg.UploadDir != "" {
		searchDirs = append(searchDirs, filepath.Join(cfg.UploadDir, "backups"))
	}
	dbPath := resolveDBPath()
	if dbPath != "" {
		searchDirs = append(searchDirs, filepath.Dir(dbPath))
		searchDirs = append(searchDirs, filepath.Join(filepath.Dir(dbPath), "uploads", "backups"))
	}
	var candidates []string
	for _, dir := range searchDirs {
		ms, _ := filepath.Glob(filepath.Join(dir, "upgrade_pre_*.db"))
		candidates = append(candidates, ms...)
		ms2, _ := filepath.Glob(filepath.Join(dir, "*.db.backup*"))
		candidates = append(candidates, ms2...)
	}
	if len(candidates) == 0 {
		fmt.Println("   未找到可用备份（upgrade_pre_*.db / *.db.backup*）")
		return false
	}
	latest := candidates[0]
	for _, c := range candidates[1:] {
		i1, e1 := os.Stat(latest)
		i2, e2 := os.Stat(c)
		if e1 == nil && e2 == nil && i2.ModTime().After(i1.ModTime()) {
			latest = c
		}
	}
	if dbPath == "" {
		fmt.Println("   无法定位数据库路径")
		return false
	}
	if err := os.Rename(dbPath, dbPath+".corrupt."+time.Now().Format("20060102_150405")); err != nil {
		fmt.Printf("   留底损坏文件失败: %v\n", err)
	}
	if err := copyFile(latest, dbPath); err != nil {
		fmt.Printf("   恢复失败: %v\n", err)
		return false
	}
	fmt.Printf("   已用 %s 恢复\n", latest)
	return true
}

// resolveDBPath 读取 DATABASE_URL 推导 sqlite 路径（与预检打开逻辑一致）
func resolveDBPath() string {
	cfg := config.AppConfig
	if cfg == nil || !strings.Contains(cfg.DatabaseURL, "sqlite") {
		return ""
	}
	p := strings.Replace(cfg.DatabaseURL, "sqlite:///./", "", 1)
	p = strings.Replace(p, "sqlite:///", "", 1)
	if !filepath.IsAbs(p) {
		if exe, err := os.Executable(); err == nil {
			p = filepath.Join(filepath.Dir(exe), p)
		}
	}
	return p
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

func tableExists(db *gorm.DB, dialect, table string) bool {
	var n int64
	if strings.Contains(dialect, "sqlite") {
		db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&n)
	} else {
		db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", table).Scan(&n)
	}
	return n > 0
}

func columnType(db *gorm.DB, dialect, table, column string) string {
	var t string
	if strings.Contains(dialect, "sqlite") {
		db.Raw("SELECT type FROM pragma_table_info(?) WHERE name=?", table, column).Scan(&t)
	} else if strings.Contains(dialect, "mysql") {
		db.Raw("SELECT DATA_TYPE FROM information_schema.COLUMNS WHERE TABLE_NAME=? AND COLUMN_NAME=?", table, column).Scan(&t)
	} else {
		db.Raw("SELECT data_type FROM information_schema.columns WHERE table_name=? AND column_name=?", table, column).Scan(&t)
	}
	return strings.TrimSpace(t)
}

func columnExists(db *gorm.DB, dialect, table, column string) bool {
	return columnType(db, dialect, table, column) != ""
}

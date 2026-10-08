package handlers

import (
	"testing"

	"cboard-go/internal/core/database"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupRestoreIdentityDB(t *testing.T, rows map[string]string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.Exec(`CREATE TABLE system_configs (id INTEGER PRIMARY KEY AUTOINCREMENT, key TEXT, value TEXT, category TEXT)`).Error; err != nil {
		t.Fatalf("建表失败: %v", err)
	}
	for k, v := range rows {
		if err := db.Exec(`INSERT INTO system_configs (key, value, category) VALUES (?, ?, 'general')`, k, v).Error; err != nil {
			t.Fatalf("插入 %s 失败: %v", k, err)
		}
	}
	prev := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = prev
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

// 跨站点恢复：备份里带的是来源站(b 站)的域名设置，恢复后必须写回本机(a 站)的值，
// 否则新站会把官网/订阅地址/域名池都指回来源站（站点"认错家门"）。
func TestApplySiteIdentityKeepsLocalDomain(t *testing.T) {
	db := setupRestoreIdentityDB(t, map[string]string{
		// 恢复进来的库：来源站的值
		"domain_name":                 "old-site.example.com",
		"subscription_domain":         "https://old-site.example.com",
		"subscription_backup_domains": "b1.example.com,b2.example.com",
	})

	saved := map[string]string{
		"domain_name":                 "speedora.top",
		"subscription_domain":         "https://speedora.top",
		"subscription_backup_domains": "",
	}
	applied := applySiteIdentity(saved)

	if len(applied) != 3 {
		t.Fatalf("期望写回 3 个键，实际 %v", applied)
	}
	var got string
	db.Raw("SELECT value FROM system_configs WHERE key = 'domain_name'").Scan(&got)
	if got != "speedora.top" {
		t.Fatalf("domain_name 应保留本机值 speedora.top，实际 %q", got)
	}
	db.Raw("SELECT value FROM system_configs WHERE key = 'subscription_backup_domains'").Scan(&got)
	if got != "" {
		t.Fatalf("本机备用订阅域名为空，写回后应仍为空（不该留着来源站的域名），实际 %q", got)
	}
}

// 同站点恢复自己的备份：写回的值与库内一致，属于幂等操作，不应报错也不应产生副作用。
func TestApplySiteIdentityIsIdempotentForSameSite(t *testing.T) {
	db := setupRestoreIdentityDB(t, map[string]string{
		"domain_name":         "speedora.top",
		"subscription_domain": "https://speedora.top",
	})

	applied := applySiteIdentity(map[string]string{
		"domain_name":         "speedora.top",
		"subscription_domain": "https://speedora.top",
	})

	if len(applied) != 2 {
		t.Fatalf("期望 2 个键都写回，实际 %v", applied)
	}
	var got string
	db.Raw("SELECT value FROM system_configs WHERE key = 'subscription_domain'").Scan(&got)
	if got != "https://speedora.top" {
		t.Fatalf("值被改坏了: %q", got)
	}
}

// 恢复进来的库里没有这些键（例如老版本备份）时要静默跳过，不能凭空插入。
func TestApplySiteIdentitySkipsMissingKeys(t *testing.T) {
	db := setupRestoreIdentityDB(t, map[string]string{"site_name": "CBoard"})

	if applied := applySiteIdentity(map[string]string{"domain_name": "speedora.top"}); len(applied) != 0 {
		t.Fatalf("库里没有 domain_name 时不应写回，实际 %v", applied)
	}
	var cnt int64
	db.Raw("SELECT COUNT(*) FROM system_configs WHERE key = 'domain_name'").Scan(&cnt)
	if cnt != 0 {
		t.Fatalf("不应凭空插入配置项，实际插入 %d 条", cnt)
	}
}

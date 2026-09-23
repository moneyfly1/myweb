package utils

import (
	"net/http/httptest"
	"strings"
	"testing"

	"cboard-go/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupSubDomainDB(t *testing.T) *gorm.DB {
	return setupSubDomainDBNamed(t, t.Name())
}

// setupSubDomainDBNamed 允许同一测试内开多个互不干扰的内存库。
func setupSubDomainDBNamed(t *testing.T, name string) *gorm.DB {
	t.Helper()
	dsn := "file:" + name + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.SystemConfig{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

func setCfg(t *testing.T, db *gorm.DB, key, value string) {
	t.Helper()
	if err := db.Create(&models.SystemConfig{Key: key, Value: value, Category: "general"}).Error; err != nil {
		t.Fatalf("写入配置 %s 失败: %v", key, err)
	}
}

// 未配置订阅域名时回退到网站域名（旧行为不变）。
func TestSubscriptionBaseURLFallsBackToSiteDomain(t *testing.T) {
	db := setupSubDomainDB(t)
	setCfg(t, db, "domain_name", "dy.moneyfly.top")
	req := httptest.NewRequest("GET", "https://dy.moneyfly.top/api/v1/x", nil)

	if got := SubscriptionBaseURL(req, db); got != "https://dy.moneyfly.top" {
		t.Errorf("SubscriptionBaseURL = %q, want https://dy.moneyfly.top", got)
	}
}

// 配置订阅域名后，订阅链接走订阅域名；网站域名不受影响（GetBuildBaseURL 仍返回网站域名）。
func TestSubscriptionBaseURLPrefersSubscriptionDomain(t *testing.T) {
	db := setupSubDomainDB(t)
	setCfg(t, db, "domain_name", "dy.moneyfly.top")
	setCfg(t, db, SubscriptionDomainKey, "https://sub.moneyfly.dpdns.org")
	req := httptest.NewRequest("GET", "https://dy.moneyfly.top/api/v1/x", nil)

	if got := SubscriptionBaseURL(req, db); got != "https://sub.moneyfly.dpdns.org" {
		t.Errorf("SubscriptionBaseURL = %q, want https://sub.moneyfly.dpdns.org", got)
	}
	if got := GetBuildBaseURL(req, db); got != "https://dy.moneyfly.top" {
		t.Errorf("网站域名链路不应被订阅域名影响: GetBuildBaseURL = %q", got)
	}
}

// 不带 scheme 的配置自动补 https；结尾斜杠被去掉。
func TestSubscriptionBaseURLNormalizesValue(t *testing.T) {
	db := setupSubDomainDB(t)
	setCfg(t, db, SubscriptionDomainKey, "sub.example.com/")
	req := httptest.NewRequest("GET", "https://dy.moneyfly.top/x", nil)

	if got := SubscriptionBaseURL(req, db); got != "https://sub.example.com" {
		t.Errorf("SubscriptionBaseURL = %q, want https://sub.example.com", got)
	}
}

// 域名池：主域名在前 → 备用（逗号/换行分隔）→ 网站域名兜底；去重、封顶 5 个。
func TestSubscriptionBaseURLsPool(t *testing.T) {
	db := setupSubDomainDB(t)
	setCfg(t, db, "domain_name", "dy.moneyfly.top")
	setCfg(t, db, SubscriptionDomainKey, "https://sub.moneyfly.dpdns.org")
	setCfg(t, db, SubscriptionBackupDomainsKey, "moneyfly.dpdns.org,\n https://sub.moneyfly.dpdns.org ;\nicandoit.eu.org")
	req := httptest.NewRequest("GET", "https://dy.moneyfly.top/x", nil)

	got := SubscriptionBaseURLs(req, db)
	want := []string{
		"https://sub.moneyfly.dpdns.org",
		"https://moneyfly.dpdns.org",
		"https://icandoit.eu.org",
		"https://dy.moneyfly.top",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("SubscriptionBaseURLs = %v, want %v（主域名在前、去重、网站域名兜底）", got, want)
	}
}

// 无请求上下文（异步/邮件场景）：只用配置值。
func TestSubscriptionBaseURLWithoutRequest(t *testing.T) {
	db := setupSubDomainDB(t)
	setCfg(t, db, SubscriptionDomainKey, "sub.example.com")
	if got := SubscriptionBaseURL(nil, db); got != "https://sub.example.com" {
		t.Errorf("SubscriptionBaseURL(nil) = %q, want https://sub.example.com", got)
	}

	db2 := setupSubDomainDBNamed(t, t.Name()+"-fallback")
	setCfg(t, db2, "domain_name", "site.example.com")
	if got := SubscriptionBaseURL(nil, db2); got != "https://site.example.com" {
		t.Errorf("未配置订阅域名时应回退网站域名，got %q", got)
	}
}

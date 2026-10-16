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
	// 用「不在配置内」的请求域名，钉住"没有提升时的基础顺序"。
	// （请求域名命中配置内域名时会被提到第 2 位，见
	//   TestSubscriptionBaseURLsPromotesRequestHost）
	req := httptest.NewRequest("GET", "https://unknown.example/x", nil)
	req.Host = "unknown.example"

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

// 用户「服务器线路」里能选到的域名必须**全部**下发，不能被上限截断：
// 线上配置是 主域名 + 5 个备用 + 网站域名兜底，旧上限 5 会把最后一条（例如
// fastora.top）悄悄丢掉 —— 用户在设置里正好切到那条线路时，后端下发的订阅地址
// 里根本没有它，客户端只能继续撞旧域名（「换了线路还是拉取失败」）。
func TestSubscriptionBaseURLsKeepsEverySelectableLine(t *testing.T) {
	db := setupSubDomainDB(t)
	setCfg(t, db, "domain_name", "dy.moneyfly.top")
	setCfg(t, db, SubscriptionDomainKey, "https://dy.moneyfly.top")
	setCfg(t, db, SubscriptionBackupDomainsKey,
		"sub.moneyfly.dpdns.org, moneyfly.dpdns.org, new.moneyfly.dpdns.org, sub.fastora.top, fastora.top")
	req := httptest.NewRequest("GET", "https://dy.moneyfly.top/x", nil)

	got := SubscriptionBaseURLs(req, db)
	for _, host := range []string{
		"dy.moneyfly.top", "sub.moneyfly.dpdns.org", "moneyfly.dpdns.org",
		"new.moneyfly.dpdns.org", "sub.fastora.top", "fastora.top",
	} {
		found := false
		for _, base := range got {
			if strings.Contains(base, host) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("域名 %s 应出现在订阅域名池里（用户能在设置里选到它），实际 %v", host, got)
		}
	}
	if len(got) > subDomainPoolCap {
		t.Errorf("域名数量 %d 超过上限 %d", len(got), subDomainPoolCap)
	}
}

// 客户端这次请求用的是哪个域名，就把那个域名提到第 2 位（主域名仍排第一）：
// 第三方客户端只会用列表里的第一个地址，把"刚被证明可达"的域名提前能减少
// 「订阅地址打不开 → 更新失败」。
func TestSubscriptionBaseURLsPromotesRequestHost(t *testing.T) {
	db := setupSubDomainDB(t)
	setCfg(t, db, "domain_name", "dy.moneyfly.top")
	setCfg(t, db, SubscriptionDomainKey, "https://dy.moneyfly.top")
	setCfg(t, db, SubscriptionBackupDomainsKey, "moneyfly.dpdns.org, sub.moneyfly.dpdns.org, fastora.top")
	// 客户端是通过 fastora.top 访问的（说明它在当前网络可达）
	req := httptest.NewRequest("GET", "https://fastora.top/api/v1/user/subscribe", nil)
	req.Host = "fastora.top"

	got := SubscriptionBaseURLs(req, db)
	if len(got) < 3 {
		t.Fatalf("前置条件：至少有 3 个域名，实际 %v", got)
	}
	if got[0] != "https://dy.moneyfly.top" {
		t.Errorf("主订阅域名必须仍排第一，实际 %v", got)
	}
	if !strings.Contains(got[1], "fastora.top") {
		t.Errorf("请求所用域名应被提到第 2 位，实际 %v", got)
	}
	// 其余域名一个都不能丢
	for _, want := range []string{"moneyfly.dpdns.org", "sub.moneyfly.dpdns.org", "fastora.top"} {
		found := false
		for _, b := range got {
			if strings.Contains(b, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("域名 %s 不该被丢掉，实际 %v", want, got)
		}
	}
}

// 安全：请求头里的**任意域名**绝不能被插进列表（防 Host 注入 —— 否则任何人都能
// 让别人客户端去拉自己服务器的内容）。只有配置过的域名才会被提前。
func TestSubscriptionBaseURLsNeverInjectsUnknownHost(t *testing.T) {
	db := setupSubDomainDB(t)
	setCfg(t, db, "domain_name", "dy.moneyfly.top")
	setCfg(t, db, SubscriptionDomainKey, "https://dy.moneyfly.top")
	setCfg(t, db, SubscriptionBackupDomainsKey, "moneyfly.dpdns.org, fastora.top")
	req := httptest.NewRequest("GET", "https://evil.example/x", nil)
	req.Host = "evil.example"
	req.Header.Set("X-Forwarded-Host", "evil.example")

	got := SubscriptionBaseURLs(req, db)
	for _, b := range got {
		if strings.Contains(b, "evil.example") {
			t.Fatalf("未知域名绝不能进列表（Host 注入），实际 %v", got)
		}
	}
	if got[0] != "https://dy.moneyfly.top" {
		t.Errorf("主域名应仍排第一，实际 %v", got)
	}
}

// 请求用的就是主域名时顺序不变（不需要提升）
func TestSubscriptionBaseURLsKeepOrderWhenRequestUsesPrimary(t *testing.T) {
	db := setupSubDomainDB(t)
	setCfg(t, db, "domain_name", "dy.moneyfly.top")
	setCfg(t, db, SubscriptionDomainKey, "https://dy.moneyfly.top")
	setCfg(t, db, SubscriptionBackupDomainsKey, "moneyfly.dpdns.org, fastora.top")
	req := httptest.NewRequest("GET", "https://dy.moneyfly.top/x", nil)
	req.Host = "dy.moneyfly.top"

	got := SubscriptionBaseURLs(req, db)
	want := []string{"https://dy.moneyfly.top", "https://moneyfly.dpdns.org", "https://fastora.top"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("主域名请求时顺序不应变化，got %v want %v", got, want)
	}
}

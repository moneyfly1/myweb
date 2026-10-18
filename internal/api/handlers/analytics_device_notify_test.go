package handlers

import (
	"strconv"
	"testing"

	"cboard-go/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var emailCfgDBCounter int

func setupEmailCfgDB(t *testing.T, rows [][2]string) *gorm.DB {
	t.Helper()
	emailCfgDBCounter++
	// 每次调用都开一个独立的库（同一个测试里要用多个互不干扰的库）
	dsn := "file:" + t.Name() + "-" + strconv.Itoa(emailCfgDBCounter) + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.SystemConfig{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	t.Cleanup(func() {
		if s, err := db.DB(); err == nil {
			s.Close()
		}
	})
	for _, kv := range rows {
		if err := db.Create(&models.SystemConfig{Key: kv[0], Value: kv[1], Category: "email"}).Error; err != nil {
			t.Fatalf("写入配置失败: %v", err)
		}
	}
	return db
}

// 线上实际用的是 email_username / email_password（2026-10-18 误报"未配置"的根因）：
// 两套键名都必须认，否则管理员会以为 SMTP 配置丢了。
func TestSMTPStatusAcceptsBothKeyStyles(t *testing.T) {
	// ① 后台「系统设置 → 邮件」实际写入的键名
	db := setupEmailCfgDB(t, [][2]string{
		{"smtp_host", "smtp.gmail.com"},
		{"smtp_port", "465"},
		{"email_username", "moneyflysubssr@gmail.com"},
		{"email_password", "abcdefghijklmnop"},
		{"smtp_encryption", "ssl"},
	})
	if ok, host, user := smtpStatus(db); !ok || host != "smtp.gmail.com" || user == "" {
		t.Fatalf("email_username/email_password 组合必须判定为已配置，得到 ok=%v host=%q user=%q", ok, host, user)
	}

	// ② 另一套键名（smtp_username / smtp_password）同样要认
	db2 := setupEmailCfgDB(t, [][2]string{
		{"smtp_host", "smtp.qq.com"},
		{"smtp_username", "ops@example.com"},
		{"smtp_password", "secret-pass"},
	})
	if ok, _, _ := smtpStatus(db2); !ok {
		t.Fatal("smtp_username/smtp_password 组合也必须判定为已配置")
	}

	// ③ 只有主机、没有账号密码 → 未配置
	db3 := setupEmailCfgDB(t, [][2]string{{"smtp_host", "smtp.gmail.com"}})
	if ok, _, _ := smtpStatus(db3); ok {
		t.Fatal("缺账号密码时必须判定为未配置")
	}

	// ④ .env 模板占位值 → 未配置（看起来配了其实发不出去）
	db4 := setupEmailCfgDB(t, [][2]string{
		{"smtp_host", "smtp.qq.com"},
		{"email_username", "your-email@qq.com"},
		{"email_password", "your-smtp-password"},
	})
	if ok, _, _ := smtpStatus(db4); ok {
		t.Fatal("占位值必须判定为未配置")
	}
}

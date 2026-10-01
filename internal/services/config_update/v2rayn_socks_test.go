package config_update

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"cboard-go/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 回归：v2rayN **支持** socks/socks5 节点，不能再按 UA 丢弃它们。
//
// 旧实现在 generateClientConfig 与 GenerateUniversalConfig 里各硬编码了一段
// 「v2rayN 不支持 socks 节点，自动过滤」，会让 v2rayN 用户平白少掉 socks 节点。
// 该假设错误：socks outbound 是 Xray/v2ray-core 的标准协议，v2rayN 也支持导入
// socks:// 分享链接。本用例直接走生产入口 generateClientConfig 锁住这个行为。
func TestV2rayNKeepsSocksNodes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Subscription{}, &models.Node{}, &models.SystemConfig{}, &models.Device{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})

	now := time.Now()
	user := models.User{Username: "u1", Email: "u1@example.com", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("建用户失败: %v", err)
	}
	sub := models.Subscription{UserID: user.ID, SubscriptionURL: "tok-v2rayn-socks",
		Status: "active", IsActive: true, DeviceLimit: 1000, ExpireTime: now.Add(24 * time.Hour)}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("建订阅失败: %v", err)
	}

	seed := []struct {
		name, typ string
		node      ProxyNode
	}{
		{"好节点-ss", "ss", ProxyNode{Name: "好节点-ss", Type: "ss", Server: "1.1.1.1", Port: 8388,
			Cipher: "aes-128-gcm", Password: "pw", Options: map[string]any{}}},
		{"socks5节点", "socks5", ProxyNode{Name: "socks5节点", Type: "socks5", Server: "2.2.2.2", Port: 1080,
			UUID: "user", Password: "pass", Options: map[string]any{}}},
		{"socks节点", "socks", ProxyNode{Name: "socks节点", Type: "socks", Server: "3.3.3.3", Port: 1080,
			UUID: "user", Password: "pass", Options: map[string]any{}}},
	}
	for _, s := range seed {
		b := mustJSON(s.node)
		cfg := string(b)
		if err := db.Create(&models.Node{Name: s.name, Type: s.typ, Region: "test", Status: "online",
			IsActive: true, Config: &cfg}).Error; err != nil {
			t.Fatalf("建节点失败: %v", err)
		}
	}

	svc := &ConfigUpdateService{db: db}
	for _, ua := range []string{"v2rayN/6.23", "v2rayNG/1.8.5", "clash-verge/v1.7.7", "Shadowrocket/1744"} {
		out, _, _ := svc.generateClientConfig(sub.SubscriptionURL, "127.0.0.1", ua, "universal", nil)
		dec, err := base64.StdEncoding.DecodeString(strings.TrimSpace(out))
		if err != nil {
			t.Fatalf("UA=%s 输出不是合法 base64: %v", ua, err)
		}
		text := string(dec)
		if !strings.Contains(text, "socks5://") {
			t.Errorf("UA=%s: socks5 节点被丢弃了（v2rayN 支持 socks，不应按 UA 过滤）\n实际:\n%s", ua, text)
		}
		if !strings.Contains(text, "socks://") {
			t.Errorf("UA=%s: socks 节点被丢弃了", ua)
		}
		if !strings.Contains(text, "ss://") {
			t.Errorf("UA=%s: ss 节点丢失", ua)
		}
		t.Logf("UA=%-22s 链接数=%d 含 socks5=%v 含 socks=%v",
			ua, len(strings.Split(strings.TrimSpace(text), "\n")),
			strings.Contains(text, "socks5://"), strings.Contains(text, "socks://"))
	}
}

// GenerateUniversalConfig 里的同名过滤也必须已移除
func TestGenerateUniversalConfigKeepsSocks(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Subscription{}, &models.Node{}, &models.SystemConfig{}, &models.Device{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	user := models.User{Username: "u2", Email: "u2@example.com", IsActive: true}
	_ = db.Create(&user).Error
	sub := models.Subscription{UserID: user.ID, SubscriptionURL: "tok-univ-socks",
		Status: "active", IsActive: true, DeviceLimit: 1000, ExpireTime: time.Now().Add(24 * time.Hour)}
	_ = db.Create(&sub).Error
	cfg := string(mustJSON(ProxyNode{Name: "socks5节点", Type: "socks5", Server: "2.2.2.2", Port: 1080,
		UUID: "user", Password: "pass", Options: map[string]any{}}))
	_ = db.Create(&models.Node{Name: "socks5节点", Type: "socks5", Region: "test", Status: "online",
		IsActive: true, Config: &cfg}).Error

	svc := &ConfigUpdateService{db: db}
	out, err := svc.GenerateUniversalConfig(sub.SubscriptionURL, "127.0.0.1", "v2rayN/6.23", "base64")
	if err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	dec, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(out))
	if !strings.Contains(string(dec), "socks5://") {
		t.Errorf("GenerateUniversalConfig 仍按 v2rayN 丢弃 socks 节点:\n%s", string(dec))
	}
	t.Logf("GenerateUniversalConfig(v2rayN) 链接=%v", strings.Split(strings.TrimSpace(string(dec)), "\n"))
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

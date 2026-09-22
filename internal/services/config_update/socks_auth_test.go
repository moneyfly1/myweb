package config_update

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"cboard-go/internal/models"
)

// SOCKS 节点凭据拆分：面板里 socks 节点的凭据有多种存量形态，
// Clash 需要 username/password 两个独立字段 —— 旧实现把 UUID 整串当 username、
// password 留空，认证必然失败，线上表现为「节点一直超时」。
func TestSocksAuthFromNode(t *testing.T) {
	cases := []struct {
		name     string
		uuid     string
		password string
		wantUser string
		wantPass string
	}{
		{"正常链接形态：UUID=用户名 + Password=密码", "demo-user", "demo-pass", "demo-user", "demo-pass"},
		{"存量形态：UUID=base64(user:pass)", "demo-secret-285", "", "demo-user", "demo-pass"},
		{"存量形态：base64(user:pass)", "demo-secret-325", "", "demo-user", "demo-pass"},
		{"GOST 形态：base64(user:pass@host:port)", "dXNlcjpwYXNzQDEuMi4zLjQ6ODAwMQ==", "", "user", "pass"},
		{"UUID 里直接写 user:pass", "user:pass", "", "user", "pass"},
		{"只有用户名（无密码认证）", "useronly", "", "useronly", ""},
		{"UUID 空、密码里有值（少见）", "", "onlypass", "", "onlypass"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			user, pass := socksAuthFromNode(&ProxyNode{UUID: tc.uuid, Password: tc.password})
			if user != tc.wantUser || pass != tc.wantPass {
				t.Errorf("socksAuthFromNode(%q, %q) = (%q, %q), want (%q, %q)",
					tc.uuid, tc.password, user, pass, tc.wantUser, tc.wantPass)
			}
		})
	}
}

// 节点映射里必须是拆分后的 username/password，不能把 base64 整串当用户名。
func TestNodeToMapSocksCredentials(t *testing.T) {
	svc := &ConfigUpdateService{}

	// 线上 id1277 的真实 config（base64 形态）
	var p ProxyNode
	raw := `{"Name":"example-node-0004","Type":"socks","Server":"node65.example.com","Port":8001,"UUID":"demo-secret-285","UDP":true}`
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("解析配置失败: %v", err)
	}

	m := svc.nodeToMap(&p)
	if m["username"] != "demo-user" {
		t.Errorf("username = %v, want demo-user", m["username"])
	}
	if m["password"] != "demo-pass" {
		t.Errorf("password = %v, want demo-pass", m["password"])
	}
	if s, _ := m["username"].(string); strings.Contains(s, "Y2Rk") {
		t.Error("username 里不应出现 base64 整串")
	}
}

// 订阅级回归：socks 节点在 Clash 配置里必须是 socks5 + username/password。
func TestClashConfigContainsSocksCredentials(t *testing.T) {
	// generateClashYAML 会读配置表（模板/站点信息），给一个空的内存库即可走默认值
	db := setupNodeSyncTestDB(t)
	if err := db.AutoMigrate(&models.SystemConfig{}); err != nil {
		t.Fatalf("迁移 system_configs 失败: %v", err)
	}
	svc := &ConfigUpdateService{db: db}

	socks := &ProxyNode{
		Name: "example-node-0004", Type: "socks",
		Server: "node65.example.com", Port: 8001,
		UUID: "demo-secret-285", UDP: true,
	}
	ctx := &SubscriptionContext{
		Status:       StatusNormal,
		Subscription: models.Subscription{ExpireTime: time.Now().Add(30 * 24 * time.Hour)},
	}
	yaml := svc.generateClashYAML([]*ProxyNode{socks}, ctx)

	if !strings.Contains(yaml, "type: socks5") {
		t.Errorf("Clash 配置里 socks 节点应为 socks5，实际:\n%s", headLines(yaml, 12))
	}
	if !strings.Contains(yaml, "username: demo-user") || !strings.Contains(yaml, "password: demo-pass") {
		t.Errorf("Clash 配置里缺少正确的 username/password:\n%s", headLines(yaml, 12))
	}
}

// headLines 取前 n 行，便于断言失败时定位。
func headLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

package handlers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 订阅失效占位配置测试（2026-10-28）。
//
// 用户反馈的核心故障：重置订阅地址后，旧地址在 Clash 端点拿到 base64(vmess…)，
// Clash 解析失败 → 客户端不更新 → 旧真实节点残留。这里锁住三件事：
//  1. Clash 端点对失效/不存在的地址返回**合法 YAML**（含提示节点）；
//  2. 通用端点返回可解析的 base64(vmess)；
//  3. 旧地址（subscription_resets.old_subscription_url）能识别并给出更准确的提示。

func setupPlaceholderEnv(t *testing.T) *gorm.DB {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.SubscriptionReset{}, &models.Subscription{}, &models.User{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	old := database.DB
	database.DB = db
	t.Cleanup(func() {
		database.DB = old
		if s, err := db.DB(); err == nil {
			s.Close()
		}
	})
	return db
}

func doSubscribe(t *testing.T, handler gin.HandlerFunc, path, ua string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("User-Agent", ua)
	c.Request = req
	c.Params = gin.Params{{Key: "url", Value: lastSegment(path)}}
	handler(c)
	return w
}

func lastSegment(path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	return parts[len(parts)-1]
}

// Clash 占位配置必须是能被 mihomo 加载的 YAML，且节点不可用
func TestPlaceholderClashConfigIsValidYAML(t *testing.T) {
	raw := GeneratePlaceholderClashConfig("https://cards.example.com", "该订阅地址已失效（订阅地址被重置），请到官网复制新地址")

	var cfg map[string]interface{}
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatalf("占位配置不是合法 YAML: %v\n%s", err, raw)
	}

	proxies, ok := cfg["proxies"].([]interface{})
	if !ok || len(proxies) == 0 {
		t.Fatalf("proxies 为空或类型错误: %v", cfg["proxies"])
	}
	first, _ := proxies[0].(map[string]interface{})
	if first["type"] != "vmess" || first["server"] != placeholderServer {
		t.Fatalf("占位节点字段异常: %v", first)
	}
	name, _ := first["name"].(string)
	if !strings.Contains(name, "失效") {
		t.Fatalf("节点名应带失效提示: %q", name)
	}

	// proxy-group 必须引用这些节点名，否则 mihomo 加载报 "proxies missing"
	groups, ok := cfg["proxy-groups"].([]interface{})
	if !ok || len(groups) == 0 {
		t.Fatalf("proxy-groups 缺失")
	}
	g0, _ := groups[0].(map[string]interface{})
	members, _ := g0["proxies"].([]interface{})
	if len(members) < 2 {
		t.Fatalf("proxy-group 成员过少: %v", members)
	}
}

// 通用订阅占位：base64 解码后是若干 vmess:// 链接，且地址不可解析
func TestPlaceholderBase64ConfigDecodable(t *testing.T) {
	raw := GeneratePlaceholderBase64Config("cards.example.com", "订阅已过期，请续费")

	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("占位配置不是合法 base64: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(decoded)), "\n")
	if len(lines) == 0 {
		t.Fatal("占位配置为空")
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "vmess://") {
			t.Fatalf("应为 vmess 链接: %q", line)
		}
		payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(line, "vmess://"))
		if err != nil {
			t.Fatalf("vmess 内容不是合法 base64: %v", err)
		}
		var node map[string]interface{}
		if err := json.Unmarshal(payload, &node); err != nil {
			t.Fatalf("vmess 内容不是合法 JSON: %v\n%s", err, payload)
		}
		if node["add"] != placeholderServer {
			t.Fatalf("占位节点地址应为不可解析域名，实际 %v", node["add"])
		}
		if ps, _ := node["ps"].(string); !strings.Contains(ps, "请到") && !strings.Contains(ps, "订阅") {
			t.Fatalf("节点名缺少提示: %v", node["ps"])
		}
	}
}

// sing-box 占位：outbounds 可解析
func TestPlaceholderSingboxConfigParseable(t *testing.T) {
	raw := GeneratePlaceholderSingboxConfig("cards.example.com", "订阅已失效")
	var cfg struct {
		Outbounds []map[string]interface{} `json:"outbounds"`
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatalf("sing-box 占位不是合法 JSON: %v", err)
	}
	if len(cfg.Outbounds) < 2 {
		t.Fatalf("outbounds 数量不足: %d", len(cfg.Outbounds))
	}
	if cfg.Outbounds[0]["server"] != placeholderServer {
		t.Fatalf("占位节点地址异常: %v", cfg.Outbounds[0])
	}
}

// Clash 端点：不存在的地址必须回 YAML（不能回 base64）
func TestGetSubscriptionConfig_unknownTokenReturnsYAML(t *testing.T) {
	setupPlaceholderEnv(t)

	w := doSubscribe(t, GetSubscriptionConfig, "/api/v1/subscriptions/clash/unknown-token-xyz", "clash-verge/v2.0.0")
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "proxies:") || !strings.Contains(body, placeholderServer) {
		t.Fatalf("Clash 端点应返回占位 YAML，实际:\n%s", body)
	}
	var cfg map[string]interface{}
	if err := yaml.Unmarshal([]byte(body), &cfg); err != nil {
		t.Fatalf("返回内容不是合法 YAML: %v", err)
	}
}

// 通用端点：不存在的地址回 base64 vmess 占位
func TestGetUniversalSubscription_unknownTokenReturnsBase64(t *testing.T) {
	setupPlaceholderEnv(t)

	w := doSubscribe(t, GetUniversalSubscription, "/api/v1/subscriptions/universal/unknown-token-xyz", "v2rayN/6.0")
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", w.Code)
	}
	if _, err := base64.StdEncoding.DecodeString(strings.TrimSpace(w.Body.String())); err != nil {
		t.Fatalf("通用端点应返回 base64 占位，实际:\n%s", w.Body.String())
	}
}

// 旧地址识别：被重置过的 token 给出"地址已失效"的准确提示
func TestRevokedSubscriptionMessage(t *testing.T) {
	db := setupPlaceholderEnv(t)
	oldURL, newURL := "old-token-123", "new-token-456"
	db.Create(&models.SubscriptionReset{
		UserID: 1, SubscriptionID: 1, ResetType: "user_reset",
		OldSubscriptionURL: &oldURL, NewSubscriptionURL: &newURL,
	})

	if msg, ok := revokedSubscriptionMessage(db, oldURL); !ok || !strings.Contains(msg, "失效") {
		t.Fatalf("应识别出被重置的旧地址，得到 ok=%v msg=%q", ok, msg)
	}
	if _, ok := revokedSubscriptionMessage(db, "never-existed"); ok {
		t.Fatal("随机 token 不应被判定为重置旧地址")
	}
}

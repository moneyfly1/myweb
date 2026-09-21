package config_update

import (
	"encoding/json"
	"testing"
	"time"

	"cboard-go/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupNodeSyncTestDB 创建内存 SQLite 测试库（含 nodes 表）。
func setupNodeSyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Node{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

// 采集同步只更新采集侧字段，绝不能覆盖健康检查维护的 status/is_active/latency/last_test。
// 回归：旧实现用 db.Save(exist) 整行回写并把 status 硬编码回 "online"、is_active 置回 true，
// 导致每次节点更新（默认每小时）都会抹掉健康检查结论，"自动屏蔽失效节点"形同虚设。
func TestImportNodesPreservesHealthStatus(t *testing.T) {
	db := setupNodeSyncTestDB(t)
	svc := &ConfigUpdateService{db: db}

	// 已有采集节点：被健康检查判定为离线并自动屏蔽
	oldNode := ProxyNode{Name: "香港-01", Type: "vless", Server: "1.2.3.4", Port: 443, UUID: "uuid-1"}
	oldCfg, _ := json.Marshal(oldNode)
	oldCfgStr := string(oldCfg)
	testedAt := time.Now().Add(-time.Minute).Truncate(time.Second)
	seed := models.Node{
		Name:          oldNode.Name,
		Type:          oldNode.Type,
		Region:        "香港",
		Config:        &oldCfgStr,
		Status:        "offline",
		IsActive:      false,
		Latency:       -1,
		LastTest:      &testedAt,
		IsManual:      false,
		IsRecommended: false,
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("创建已有节点失败: %v", err)
	}
	// models.Node.IsActive 带 default:true，Create 会忽略零值 false，
	// 这里显式落库为"已屏蔽"，还原 health check auto_disable 之后的真实状态。
	if err := db.Model(&models.Node{}).Where("id = ?", seed.ID).Update("is_active", false).Error; err != nil {
		t.Fatalf("设置 is_active=false 失败: %v", err)
	}

	// 订阅采集到同一个节点（配置一致、名称一致 → 命中更新分支）
	incoming := oldNode
	stats := svc.importNodesToDatabaseWithOrderTx(db, []nodeWithOrder{
		{node: &incoming, orderIndex: 7, sourceIndex: 1},
	})

	if stats.Updated != 1 {
		t.Fatalf("stats.Updated = %d, want 1（应命中更新分支）", stats.Updated)
	}

	var got models.Node
	if err := db.First(&got, seed.ID).Error; err != nil {
		t.Fatalf("读取节点失败: %v", err)
	}

	if got.Status != "offline" {
		t.Errorf("status = %q, want %q（同步不得把健康检查结论改回 online）", got.Status, "offline")
	}
	if got.IsActive {
		t.Error("is_active = true，同步把自动屏蔽的节点重新启用了")
	}
	if got.Latency != -1 {
		t.Errorf("latency = %d, want -1（同步不得回写旧延迟）", got.Latency)
	}
	if got.LastTest == nil || !got.LastTest.Equal(testedAt) {
		t.Errorf("last_test 被同步覆盖（gone）")
	}
	// 采集侧字段应照常更新
	if got.OrderIndex != 7 || got.SourceIndex != 1 {
		t.Errorf("order_index/source_index = %d/%d, want 7/1", got.OrderIndex, got.SourceIndex)
	}
}

// 新采集节点的默认状态保持原行为（online + 启用），避免改变用户可见的订阅内容。
func TestImportNewNodeDefaultsUnchanged(t *testing.T) {
	db := setupNodeSyncTestDB(t)
	svc := &ConfigUpdateService{db: db}

	incoming := ProxyNode{Name: "东京-02", Type: "vmess", Server: "5.6.7.8", Port: 8443, UUID: "uuid-2"}
	stats := svc.importNodesToDatabaseWithOrderTx(db, []nodeWithOrder{
		{node: &incoming, orderIndex: 1, sourceIndex: 0},
	})
	if stats.Created != 1 {
		t.Fatalf("stats.Created = %d, want 1", stats.Created)
	}

	var got models.Node
	if err := db.Where("name = ?", incoming.Name).First(&got).Error; err != nil {
		t.Fatalf("读取新节点失败: %v", err)
	}
	if got.Status != "online" || !got.IsActive {
		t.Errorf("新节点 status/is_active = %q/%v, want online/true", got.Status, got.IsActive)
	}
	if got.IsManual {
		t.Error("采集节点 is_manual 应为 false")
	}
}

// 同步必须保持节点 ID 稳定（不再整表删除重建）：ID 变了，健康检查写入的结果
// 就会落到已删除的行上，节点列表永远显示"在线 0ms"。
func TestImportNodesPreservesIDAndRemovesStale(t *testing.T) {
	db := setupNodeSyncTestDB(t)
	svc := &ConfigUpdateService{db: db}

	keep := ProxyNode{Name: "香港-01", Type: "vless", Server: "1.2.3.4", Port: 443, UUID: "uuid-keep"}
	drop := ProxyNode{Name: "香港-02", Type: "vless", Server: "1.2.3.5", Port: 443, UUID: "uuid-drop"}
	cfgKeep, _ := json.Marshal(keep)
	cfgDrop, _ := json.Marshal(drop)
	keepStr, dropStr := string(cfgKeep), string(cfgDrop)

	seedKeep := models.Node{Name: keep.Name, Type: keep.Type, Config: &keepStr, Status: "online", IsActive: true}
	seedDrop := models.Node{Name: drop.Name, Type: drop.Type, Config: &dropStr, Status: "online", IsActive: true}
	if err := db.Create(&seedKeep).Error; err != nil {
		t.Fatalf("创建节点失败: %v", err)
	}
	if err := db.Create(&seedDrop).Error; err != nil {
		t.Fatalf("创建节点失败: %v", err)
	}

	// 上游只剩 keep 一个节点
	incoming := keep
	stats := svc.importNodesToDatabaseWithOrderTx(db, []nodeWithOrder{{node: &incoming, orderIndex: 1}})

	if stats.Updated != 1 {
		t.Errorf("stats.Updated = %d, want 1（同 key 应更新而不是重建）", stats.Updated)
	}
	if stats.Created != 0 {
		t.Errorf("stats.Created = %d, want 0（不得重建已有节点）", stats.Created)
	}
	if stats.Removed != 1 {
		t.Errorf("stats.Removed = %d, want 1（上游消失的节点应清理）", stats.Removed)
	}

	var got models.Node
	if err := db.Where("name = ?", keep.Name).First(&got).Error; err != nil {
		t.Fatalf("读取保留节点失败: %v", err)
	}
	if got.ID != seedKeep.ID {
		t.Errorf("节点 ID = %d, want %d（同步不得换 ID）", got.ID, seedKeep.ID)
	}
	if got.Status != "online" {
		t.Errorf("status = %q, want online（同步不得重置健康状态）", got.Status)
	}

	var count int64
	db.Model(&models.Node{}).Where("name = ?", drop.Name).Count(&count)
	if count != 0 {
		t.Errorf("上游已消失的节点仍存在 %d 条，want 0", count)
	}
}

// 手动节点不属于采集管理范围：同步（含空同步）绝不能删除它们。
func TestImportNeverRemovesManualNodes(t *testing.T) {
	db := setupNodeSyncTestDB(t)
	svc := &ConfigUpdateService{db: db}

	manualCfg := `{"Name":"自建-东京","Type":"vless","Server":"9.9.9.9","Port":443,"UUID":"manual-1"}`
	manual := models.Node{Name: "自建-东京", Type: "vless", Config: &manualCfg, Status: "online", IsActive: true}
	if err := db.Create(&manual).Error; err != nil {
		t.Fatalf("创建手动节点失败: %v", err)
	}
	if err := db.Model(&models.Node{}).Where("id = ?", manual.ID).Update("is_manual", true).Error; err != nil {
		t.Fatalf("设置 is_manual=true 失败: %v", err)
	}

	stats := svc.importNodesToDatabaseWithOrderTx(db, nil)

	if stats.Removed != 0 {
		t.Errorf("stats.Removed = %d, want 0（手动节点不能被采集同步删除）", stats.Removed)
	}
	var count int64
	db.Model(&models.Node{}).Where("is_manual = ?", true).Count(&count)
	if count != 1 {
		t.Errorf("手动节点数量 = %d, want 1", count)
	}
}

// 「自动屏蔽失效节点」开关关闭时，status=timeout 的节点也必须下发给用户
// （2026-09-21 业务决定：服务端 TCP 探测会误判，探测结果不用于隐藏节点）。
func TestAppendSystemNodesRespectsAutoDisableSwitch(t *testing.T) {
	cases := []struct {
		name      string
		switchVal string
		wantCount int
	}{
		{"开关关闭：timeout 节点照常下发", "false", 2},
		{"开关开启：排除 timeout 节点", "true", 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := setupNodeSyncTestDB(t)
			if err := db.AutoMigrate(&models.SystemConfig{}); err != nil {
				t.Fatalf("迁移 system_configs 失败: %v", err)
			}
			if err := db.Create(&models.SystemConfig{
				Key: "auto_disable_timeout", Value: tc.switchVal, Category: "node_health",
			}).Error; err != nil {
				t.Fatalf("写入开关失败: %v", err)
			}

			mk := func(name, status string) {
				cfg := `{"Type":"vless","Server":"1.2.3.4","Port":443,"Name":"` + name + `"}`
				if err := db.Create(&models.Node{Name: name, Type: "vless", Config: &cfg, Status: status, IsActive: true}).Error; err != nil {
					t.Fatalf("创建节点失败: %v", err)
				}
			}
			mk("在线节点", "online")
			mk("超时节点", "timeout")

			svc := &ConfigUpdateService{db: db}
			var proxies []*ProxyNode
			svc.appendSystemNodes(&proxies, map[string]bool{})

			if len(proxies) != tc.wantCount {
				t.Errorf("下发节点数 = %d, want %d", len(proxies), tc.wantCount)
			}
		})
	}
}

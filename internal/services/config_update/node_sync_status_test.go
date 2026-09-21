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

package config_update

import (
	"testing"
	"time"

	"cboard-go/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupAppendCustomNodesDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.CustomNode{}, &models.UserCustomNode{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

// 端到端钉住「专线节点改了不生效」：订阅下发只取 custom_nodes.config，
// 管理员在「配置(JSON)」里改好的 server 必须原样出现在下发的节点里。
func TestAppendCustomNodesUsesConfigServer(t *testing.T) {
	db := setupAppendCustomNodesDB(t)

	const wantServer = "basic-vip.miyavip.vip"
	node := models.CustomNode{
		Name:        "美国家庭专线",
		DisplayName: "美国家庭专线",
		Protocol:    "socks",
		Domain:      wantServer,
		Port:        8001,
		IsActive:    true,
		Status:      "active",
		Config:      `{"Name":"美国家庭专线","Type":"socks","Server":"` + wantServer + `","Port":8001,"UUID":"dXNlcjpwYXNz","UDP":true}`,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("创建节点失败: %v", err)
	}
	if err := db.Create(&models.UserCustomNode{UserID: 42, CustomNodeID: node.ID}).Error; err != nil {
		t.Fatalf("分配节点失败: %v", err)
	}

	svc := &ConfigUpdateService{db: db}
	proxies := make([]*ProxyNode, 0, 1)
	svc.appendCustomNodes(42, time.Now(), false, &proxies, map[string]bool{})

	if len(proxies) != 1 {
		t.Fatalf("期望下发 1 个专线节点，实际 %d", len(proxies))
	}
	if proxies[0].Server != wantServer {
		t.Fatalf("下发节点地址错误：期望 %s，实际 %s", wantServer, proxies[0].Server)
	}
	if proxies[0].Port != 8001 {
		t.Fatalf("下发节点端口错误：期望 8001，实际 %d", proxies[0].Port)
	}
	if proxies[0].Name != "美国家庭专线" {
		t.Fatalf("下发节点名称错误：%s", proxies[0].Name)
	}
}

// 未分配该节点的用户不应拿到它（保持分配隔离）。
func TestAppendCustomNodesSkipsUnassignedUser(t *testing.T) {
	db := setupAppendCustomNodesDB(t)

	node := models.CustomNode{
		Name: "专线A", Protocol: "socks", Domain: "a.example.com", Port: 8001,
		IsActive: true, Status: "active",
		Config: `{"Type":"socks","Server":"a.example.com","Port":8001,"UUID":"dXNlcjpwYXNz"}`,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("创建节点失败: %v", err)
	}
	if err := db.Create(&models.UserCustomNode{UserID: 7, CustomNodeID: node.ID}).Error; err != nil {
		t.Fatalf("分配节点失败: %v", err)
	}

	svc := &ConfigUpdateService{db: db}
	proxies := make([]*ProxyNode, 0, 1)
	svc.appendCustomNodes(99, time.Now(), false, &proxies, map[string]bool{})

	if len(proxies) != 0 {
		t.Fatalf("未分配用户不应收到节点，实际 %d 个", len(proxies))
	}
}

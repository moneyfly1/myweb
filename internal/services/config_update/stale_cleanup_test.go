package config_update

import (
	"encoding/json"
	"testing"

	"cboard-go/internal/models"
)

// staleDeletionPlan：源本轮**确有产出**时，其上游已消失的节点照常清理。
func TestStaleDeletionPlanDeletesWhenSourceAlive(t *testing.T) {
	gone := &models.Node{Name: "香港-02", Type: "vless", SourceIndex: 1}
	gone.ID = 22
	existing := map[string]*models.Node{"k-gone": gone}
	del, preserved := staleDeletionPlan(existing, map[string]bool{}, map[int]bool{1: true}, 3)
	if len(del) != 1 || del[0] != 22 {
		t.Fatalf("del = %v, want [22]（源存活时上游消失的节点必须清理）", del)
	}
	if preserved != 0 {
		t.Errorf("preserved = %d, want 0", preserved)
	}
}

// staleDeletionPlan：源本轮**零产出**（下载失败/404/限流）时，其节点必须保守保留。
//
// 回归：旧实现只按 seenKeys 判定"上游已消失"，而下载失败的源本轮不贡献任何节点，
// 于是"我们没拿到数据"被当成"上游删了节点"——一次源故障 = 该源节点被整批删光。
// 现网源 14（repo-sync/zibvpn_nodes.txt）长期 404，其节点在旧逻辑下每轮都被删除。
func TestStaleDeletionPlanPreservesWhenSourceSilent(t *testing.T) {
	a := &models.Node{Name: "荷兰-01", Type: "ss", SourceIndex: 1}
	a.ID = 11
	b := &models.Node{Name: "荷兰-02", Type: "ss", SourceIndex: 1}
	b.ID = 12
	existing := map[string]*models.Node{"k-a": a, "k-b": b}
	// 源 1 本轮无产出：aliveSources 里有源 2，没有源 1
	del, preserved := staleDeletionPlan(existing, map[string]bool{}, map[int]bool{2: true}, 3)
	if len(del) != 0 {
		t.Fatalf("del = %v, want 空（源无产出时不得删除其节点）", del)
	}
	if preserved != 2 {
		t.Errorf("preserved = %d, want 2", preserved)
	}
}

// staleDeletionPlan：源已被管理员从配置中移除（SourceIndex 超出当前源数量）→ 允许清理。
func TestStaleDeletionPlanDeletesRemovedSource(t *testing.T) {
	n := &models.Node{Name: "旧源节点", Type: "ss", SourceIndex: 17}
	n.ID = 99
	del, preserved := staleDeletionPlan(map[string]*models.Node{"k": n}, map[string]bool{}, map[int]bool{1: true}, 16)
	if len(del) != 1 || del[0] != 99 || preserved != 0 {
		t.Fatalf("del=%v preserved=%d, want del=[99] preserved=0（源已从配置移除，允许清理）", del, preserved)
	}
}

// staleDeletionPlan：本轮仍在上游出现的节点绝不进入删除计划。
func TestStaleDeletionPlanSkipsSeenNodes(t *testing.T) {
	n := &models.Node{Name: "还在", Type: "ss", SourceIndex: 1}
	n.ID = 5
	del, preserved := staleDeletionPlan(map[string]*models.Node{"k": n}, map[string]bool{"k": true}, map[int]bool{1: true}, 3)
	if len(del) != 0 || preserved != 0 {
		t.Fatalf("del=%v preserved=%d, want 0/0", del, preserved)
	}
}

// 端到端：源下载失败时，库里该源的原节点必须原样保留（Removed=0 且 Preserved>0）。
func TestImportPreservesNodesWhenSourceFetchFailed(t *testing.T) {
	db := setupNodeSyncTestDB(t)
	svc := &ConfigUpdateService{db: db}

	seed := func(name string, srcIdx int) models.Node {
		n := ProxyNode{Name: name, Type: "ss", Server: "9.9.9.9", Port: 443, Password: "pw-" + name, Cipher: "aes-256-gcm"}
		cfg, _ := json.Marshal(n)
		cfgStr := string(cfg)
		row := models.Node{Name: name, Type: n.Type, Config: &cfgStr, Status: "online", IsActive: true, SourceIndex: srcIdx, IsManual: false}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("建种子节点失败: %v", err)
		}
		return row
	}
	seed("源1-节点A", 1)
	seed("源1-节点B", 1)

	// 本轮：源 1 下载失败（aliveSources 不含 1），只有源 2 有产出
	stats := svc.importNodesToDatabaseWithOrderTx(db, nil, map[int]bool{2: true}, 3)
	if stats.Removed != 0 {
		t.Errorf("Removed = %d, want 0（源下载失败不得删除其节点）", stats.Removed)
	}
	if stats.Preserved != 2 {
		t.Errorf("Preserved = %d, want 2", stats.Preserved)
	}
	var count int64
	db.Model(&models.Node{}).Count(&count)
	if count != 2 {
		t.Errorf("库里剩余节点 = %d, want 2（节点被误删）", count)
	}
}

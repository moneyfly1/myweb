package handlers

import (
	"encoding/json"
	"testing"

	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupCustomNodeMergeDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.CustomNode{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

func seedCustomNode(t *testing.T, db *gorm.DB, node models.CustomNode) models.CustomNode {
	t.Helper()
	if node.Config == "" {
		node.Config = `{"Type":"` + node.Protocol + `","Server":"` + node.Domain + `","Port":1}`
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("创建节点 %s 失败: %v", node.Name, err)
	}
	return node
}

// 候选集必须包含「订阅来源 + source 为空的历史遗留节点」，但排除自建与手动链接导入。
func TestLoadCustomNodeMatchCandidates(t *testing.T) {
	db := setupCustomNodeMergeDB(t)

	legacy := seedCustomNode(t, db, models.CustomNode{Name: "L1", Protocol: "vless", Domain: "1.1.1.1", Port: 443, Source: "", SourceURL: ""})
	subA := seedCustomNode(t, db, models.CustomNode{Name: "S1", Protocol: "vmess", Domain: "2.2.2.2", Port: 80, Source: "subscription", SourceURL: "https://sub/a"})
	subB := seedCustomNode(t, db, models.CustomNode{Name: "S2", Protocol: "trojan", Domain: "3.3.3.3", Port: 8443, Source: "subscription", SourceURL: "https://sub/b"})
	selfHosted := seedCustomNode(t, db, models.CustomNode{Name: "H1", Protocol: "vless", Domain: "4.4.4.4", Port: 443, Source: "selfhost"})
	link := seedCustomNode(t, db, models.CustomNode{Name: "K1", Protocol: "ss", Domain: "5.5.5.5", Port: 8388, Source: "link"})
	if err := db.Model(&models.CustomNode{}).Where("id = ?", selfHosted.ID).Update("self_hosted", true).Error; err != nil {
		t.Fatalf("设置 self_hosted 失败: %v", err)
	}

	ids := func(nodes []models.CustomNode) map[uint]bool {
		out := make(map[uint]bool, len(nodes))
		for _, n := range nodes {
			out[n.ID] = true
		}
		return out
	}

	// replaceAll=true：全部订阅来源 + 历史遗留
	got := ids(loadCustomNodeMatchCandidates(db, "https://sub/a", true))
	if !got[legacy.ID] || !got[subA.ID] || !got[subB.ID] {
		t.Errorf("replaceAll 候选集缺少订阅/遗留节点: %v", got)
	}
	if got[selfHosted.ID] {
		t.Error("replaceAll 候选集不应包含自建节点（会被订阅更新误改）")
	}
	if got[link.ID] {
		t.Error("replaceAll 候选集不应包含手动链接导入节点")
	}

	// replaceAll=false：仅同一 source_url 的订阅节点 + 历史遗留
	got = ids(loadCustomNodeMatchCandidates(db, "https://sub/a", false))
	if !got[legacy.ID] || !got[subA.ID] {
		t.Errorf("同 URL 候选集缺少订阅/遗留节点: %v", got)
	}
	if got[subB.ID] {
		t.Error("同 URL 模式不应包含其它订阅地址的节点")
	}
	if got[selfHosted.ID] || got[link.ID] {
		t.Error("同 URL 模式不应包含自建/手动链接节点")
	}
}

// 核心回归：历史遗留节点与传入链接是**同一个节点**（协议+地址+凭据+传输参数全同）时，
// 就地更新，不再重复插入。
func TestMergeUpdatesLegacyNodeByIdentity(t *testing.T) {
	cfg := storedConfigJSON(t, "vless://uuid-1@th.example.org:37579?encryption=none&security=tls&type=ws#老节点")
	legacy := models.CustomNode{ID: 109, Name: "老节点", Protocol: "vless", Domain: "th.example.org", Port: 37579, Source: "", Config: cfg}
	links := []string{"vless://uuid-1@th.example.org:37579?encryption=none&security=tls&type=ws#新名字"}

	res := mergeCustomNodesFromLinks([]models.CustomNode{legacy}, links, "https://sub/x")

	if len(res.newNodes) != 0 {
		t.Fatalf("新增 %d 个节点, want 0（同一节点应就地更新而不是重复插入）", len(res.newNodes))
	}
	if len(res.updates) != 1 {
		t.Fatalf("更新 %d 个节点, want 1", len(res.updates))
	}
	upd := res.updates[0]
	if upd.ID != legacy.ID {
		t.Errorf("更新的是 id=%d, want %d", upd.ID, legacy.ID)
	}
	if upd.Fields["domain"] != "th.example.org" || upd.Fields["port"] != 37579 {
		t.Errorf("更新字段异常: %v", upd.Fields)
	}
	if upd.Fields["source"] != "subscription" {
		t.Errorf("source = %v, want subscription（被订阅认领后应归入订阅来源）", upd.Fields["source"])
	}
	if upd.Fields["source_url"] != "https://sub/x" {
		t.Errorf("source_url = %v, want https://sub/x", upd.Fields["source_url"])
	}
	if res.kept != 0 {
		t.Errorf("kept = %d, want 0", res.kept)
	}
}

// 用户要求（2026-09-22）：服务器 IP + 端口相同，但其他配置不同（凭据/传输参数不同）
// 时必须能导入为**新节点**，不能被当成"已存在"拒收或覆盖老节点。
func TestMergeAllowsSameAddressDifferentConfig(t *testing.T) {
	cases := []struct {
		name    string
		oldLink string
		newLink string
	}{
		{"同地址不同 UUID", "vless://uuid-old@1.2.3.4:443?encryption=none&security=tls#A", "vless://uuid-new@1.2.3.4:443?encryption=none&security=tls#B"},
		{"同地址不同传输参数(path)", "vless://uuid-1@1.2.3.4:443?encryption=none&type=ws&path=%2Fold#A", "vless://uuid-1@1.2.3.4:443?encryption=none&type=ws&path=%2Fnew#B"},
		{"同地址不同协议", "vless://uuid-1@1.2.3.4:443?encryption=none#A", "trojan://pw-1@1.2.3.4:443#B"},
		{"同地址不同密码", "trojan://pw-old@1.2.3.4:443#A", "trojan://pw-new@1.2.3.4:443#B"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := storedConfigJSON(t, tc.oldLink)
			old := models.CustomNode{ID: 500, Name: "老节点", Protocol: "vless", Domain: "1.2.3.4", Port: 443, Source: "", Config: cfg}

			res := mergeCustomNodesFromLinks([]models.CustomNode{old}, []string{tc.newLink}, "https://sub/x")

			if len(res.updates) != 0 {
				t.Errorf("地址相同但配置不同，不应覆盖老节点: %+v", res.updates)
			}
			if len(res.newNodes) != 1 {
				t.Fatalf("新增 %d 个节点, want 1（必须允许导入）", len(res.newNodes))
			}
		})
	}
}

// 存量行 config 为空/损坏时算不出身份 → 按"不重复"处理，宁可允许导入也不误拒。
func TestMergeAllowsImportWhenStoredConfigUnreadable(t *testing.T) {
	old := models.CustomNode{ID: 600, Name: "老节点", Protocol: "vless", Domain: "1.2.3.4", Port: 443, Source: "", Config: "{不是合法 json"}
	res := mergeCustomNodesFromLinks([]models.CustomNode{old}, []string{"vless://uuid-1@1.2.3.4:443?encryption=none#新"}, "https://sub/x")
	if len(res.newNodes) != 1 {
		t.Fatalf("新增 %d 个节点, want 1", len(res.newNodes))
	}
	if len(res.updates) != 0 {
		t.Errorf("不应覆盖无法解析的存量行: %+v", res.updates)
	}
}

// 历史遗留节点不参与「同名即同节点」匹配：名称相同但地址不同时，
// 不能把来源不明的节点静默改地址（宁可新增）。
func TestMergeDoesNotMatchLegacyByName(t *testing.T) {
	legacy := models.CustomNode{ID: 200, Name: "香港01", Protocol: "vless", Domain: "1.1.1.1", Port: 443, Source: ""}
	links := []string{"vless://uuid-2@9.9.9.9:443?encryption=none#香港01"}

	res := mergeCustomNodesFromLinks([]models.CustomNode{legacy}, links, "https://sub/y")

	if len(res.updates) != 0 {
		t.Errorf("按名称命中了历史遗留节点（地址不同），不应更新: %v", res.updates)
	}
	if len(res.newNodes) != 1 {
		t.Fatalf("新增 %d 个节点, want 1", len(res.newNodes))
	}
	if res.newNodes[0].Domain != "9.9.9.9" {
		t.Errorf("新节点域名为 %s, want 9.9.9.9", res.newNodes[0].Domain)
	}
}

// 订阅来源节点保持原有行为：同名即同节点（即使地址变了，视为上游换地址）。
func TestMergeMatchesSubscriptionNodeByName(t *testing.T) {
	sub := models.CustomNode{ID: 300, Name: "东京01", Protocol: "vmess", Domain: "1.1.1.1", Port: 80, Source: "subscription", SourceURL: "https://sub/z"}
	links := []string{"vmess://eyJhZGQiOiIyLjIuMi4yIiwicG9ydCI6ODAsImlkIjoidXVpZC0zIiwicHMiOiLkuJzkuqwwMSJ9"}

	res := mergeCustomNodesFromLinks([]models.CustomNode{sub}, links, "https://sub/z")

	if len(res.newNodes) != 0 {
		t.Fatalf("新增 %d 个节点, want 0（同名订阅节点应被更新）", len(res.newNodes))
	}
	if len(res.updates) != 1 || res.updates[0].ID != sub.ID {
		t.Fatalf("未按名称更新订阅节点: %+v", res.updates)
	}
	if res.updates[0].Fields["domain"] != "2.2.2.2" {
		t.Errorf("domain = %v, want 2.2.2.2（上游地址变化应刷新）", res.updates[0].Fields["domain"])
	}
}

// 订阅里消失的旧节点保留不动（分配保护），只计入 kept。
func TestMergeKeepsDisappearedNodes(t *testing.T) {
	gone := models.CustomNode{ID: 400, Name: "已下架", Protocol: "trojan", Domain: "1.1.1.1", Port: 443, Source: "subscription", SourceURL: "https://sub/w"}
	other := models.CustomNode{ID: 401, Name: "新增", Protocol: "vless", Domain: "2.2.2.2", Port: 8080, Source: "subscription", SourceURL: "https://sub/w"}
	links := []string{"vless://uuid-4@2.2.2.2:8080?encryption=none#新增"}

	res := mergeCustomNodesFromLinks([]models.CustomNode{gone, other}, links, "https://sub/w")

	if len(res.updates) != 1 || res.updates[0].ID != other.ID {
		t.Fatalf("未更新已存在的订阅节点: %+v", res.updates)
	}
	if len(res.newNodes) != 0 {
		t.Errorf("新增 %d 个节点, want 0", len(res.newNodes))
	}
	if res.kept != 1 {
		t.Errorf("kept = %d, want 1（订阅里消失的旧节点应保留）", res.kept)
	}
}

// 无可用链接时报错而不是静默清空。
func TestMergeReportsUnparsableLinks(t *testing.T) {
	res := mergeCustomNodesFromLinks(nil, []string{"这不是节点链接"}, "https://sub/q")
	if len(res.errs) == 0 {
		t.Error("无法解析的链接必须报错，不能静默跳过")
	}
	if len(res.newNodes) != 0 || len(res.updates) != 0 {
		t.Errorf("非法链接不应产生写入: new=%d upd=%d", len(res.newNodes), len(res.updates))
	}
}

// storedConfigJSON 把节点链接解析成 ProxyNode 并序列化，模拟库里存的 config 字段。
func storedConfigJSON(t *testing.T, link string) string {
	t.Helper()
	p, err := config_update.ParseNodeLink(link)
	if err != nil {
		t.Fatalf("解析链接失败 %q: %v", link, err)
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	return string(b)
}

// 用户场景（2026-09-22）：手动导入链接时，服务器 IP + 端口相同但配置不同的节点
// 必须能导入；只有**同一个节点**（连凭据/传输参数都一样）才按"已存在"跳过。
func TestImportLinksAllowsSameAddressDifferentConfig(t *testing.T) {
	db := setupCustomNodeMergeDB(t)

	sameNode := "vless://uuid-a@1.2.3.4:443?encryption=none&security=tls#节点A"
	otherOnSameAddr := "vless://uuid-b@1.2.3.4:443?encryption=none&security=tls#节点B"

	// 先导入一次
	imported, skipped, errCount, errs := importCustomNodesFromLinks(db, []string{sameNode}, "link", "")
	if imported != 1 || skipped != 0 || errCount != 0 {
		t.Fatalf("首次导入 imported=%d skipped=%d err=%d errs=%v", imported, skipped, errCount, errs)
	}

	// 再次导入：同一个节点被跳过，同地址不同 UUID 的必须导入成功
	imported, skipped, errCount, errs = importCustomNodesFromLinks(db, []string{sameNode, otherOnSameAddr}, "link", "")
	if errCount != 0 {
		t.Fatalf("导入失败: %v", errs)
	}
	if imported != 1 {
		t.Errorf("imported = %d, want 1（同 IP:端口但配置不同的节点必须能导入）", imported)
	}
	if skipped != 1 {
		t.Errorf("skipped = %d, want 1（同一个节点才跳过）", skipped)
	}

	var count int64
	db.Model(&models.CustomNode{}).Where("domain = ? AND port = ?", "1.2.3.4", 443).Count(&count)
	if count != 2 {
		t.Errorf("同地址节点数 = %d, want 2（两个不同配置的节点应同时存在）", count)
	}
}

// 同一批次里粘贴两条"同 IP:端口、不同凭据"的链接，两条都要导入。
func TestImportLinksKeepsBothInSameBatch(t *testing.T) {
	db := setupCustomNodeMergeDB(t)

	links := []string{
		"trojan://pw-one@5.6.7.8:8443#入口一",
		"trojan://pw-two@5.6.7.8:8443#入口二",
	}
	imported, skipped, errCount, errs := importCustomNodesFromLinks(db, links, "link", "")
	if errCount != 0 {
		t.Fatalf("导入失败: %v", errs)
	}
	if imported != 2 || skipped != 0 {
		t.Errorf("imported=%d skipped=%d, want 2/0（同地址不同凭据都要导入）", imported, skipped)
	}

	// 完全相同的一条重复出现时才跳过
	imported, skipped, _, _ = importCustomNodesFromLinks(db, []string{links[0]}, "link", "")
	if imported != 0 || skipped != 1 {
		t.Errorf("重复导入 imported=%d skipped=%d, want 0/1", imported, skipped)
	}
}

package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"
	"cboard-go/internal/services/selfhost"

	"github.com/gin-gonic/gin"
)

// 复用节点重装后的回传必须覆盖占位记录（2026-10-23 线上问题）
//
// 现场：08:58 管理员「复用节点重装」，prepareSelfHostNodeForDeploy 把 15 个节点
// 全部置 pending 并清空 config；随后 agent 回传 /agent/report-batch 返回 **400**。
// 原因是回传的去重逻辑把「同 install_id + 同协议」一律当重复跳过，
// 于是一个都没写入 → 节点靠心跳变成 online、config 永远为空 →
// 面板点「查看链接」提示「无法生成链接: 配置格式错误或协议不支持」，点编辑空白。
//
// 这里锁住两条语义：
//  1. 占位记录（pending / config 空）→ 用本次回传覆盖；
//  2. 已有完整配置的记录 → 仍然跳过，不重复创建。
func TestBatchReportOverwritesPendingPlaceholder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupRedeployTestDB(t)
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })

	// 主节点 + 一个重装留下的占位子节点（config 空、pending）
	main := models.CustomNode{
		Name: "美国自建01", DisplayName: "美国自建01", Protocol: "vless-ws", Port: 443,
		Status: selfhost.StatusPending, IsActive: false, Source: "selfhost", SelfHosted: true,
		SelfHostProtocol: "vless-ws", InstallID: "inst-1", DeployMode: "multi",
		ProtocolList: "vless-ws,vless-tcp-tls", SSHHost: "1.2.3.4", SSHPort: 22,
		InstallToken: "tok-1",
	}
	if err := db.Create(&main).Error; err != nil {
		t.Fatalf("插入主节点失败: %v", err)
	}
	placeholder := models.CustomNode{
		Name: "美国自建01-VLESS-TCP-TLS", DisplayName: "美国自建01-VLESS-TCP-TLS",
		Protocol: "vless", Port: 2056, Status: selfhost.StatusPending, IsActive: false,
		Source: "selfhost", SelfHosted: true, SelfHostProtocol: "vless-tcp-tls",
		InstallID: "inst-1", SSHHost: "1.2.3.4", SSHPort: 22, Config: "",
	}
	if err := db.Create(&placeholder).Error; err != nil {
		t.Fatalf("插入占位子节点失败: %v", err)
	}

	// 构造两条回传链接：主节点（vless-ws）+ 子节点（vless-tcp-tls）
	mainLink := "vless://11111111-1111-1111-1111-111111111111@us.example.com:443?security=tls&type=ws&path=%2Fws&sni=us.example.com#main"
	subLink := "vless://22222222-2222-2222-2222-222222222222@us.example.com:2056?security=tls&type=tcp&flow=xtls-rprx-vision&sni=us.example.com#sub"
	body, _ := json.Marshal(map[string]interface{}{
		"install_id": "inst-1",
		"token":      main.InstallToken,
		"links":      []string{mainLink, subLink},
	})

	r := gin.New()
	r.POST("/agent/report-batch", AgentReportBatch)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/agent/report-batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("回传应成功，实际 %d: %s", w.Code, w.Body.String())
	}

	// 占位子节点必须被回填 config（而不是被当重复跳过）
	var got models.CustomNode
	if err := db.First(&got, placeholder.ID).Error; err != nil {
		t.Fatalf("查询子节点失败: %v", err)
	}
	if got.Config == "" {
		t.Fatal("重装后的占位节点应被本次回传覆盖，config 不能为空")
	}
	if got.Status != selfhost.StatusOnline || !got.IsActive {
		t.Errorf("回传后节点应为 online 且启用，实际 status=%s is_active=%v", got.Status, got.IsActive)
	}
	if got.Port != 2056 {
		t.Errorf("端口应更新为链接里的 2056，实际 %d", got.Port)
	}
	// config 必须能被解析回 ProxyNode，并能生成链接（面板「查看链接」依赖它）
	var pn config_update.ProxyNode
	if err := json.Unmarshal([]byte(got.Config), &pn); err != nil {
		t.Fatalf("回填的 config 不是合法 ProxyNode JSON: %v", err)
	}
	if link := config_update.NewConfigUpdateService().NodeToLink(&pn); link == "" {
		t.Fatal("回填的 config 应能生成节点链接")
	}

	// 重复回传（此时已有完整配置）→ 跳过，不重复创建
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/agent/report-batch", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	var count int64
	db.Model(&models.CustomNode{}).Where("install_id = ?", "inst-1").Count(&count)
	if count != 2 {
		t.Fatalf("重复回传不应新建节点，实际节点数 %d", count)
	}
	if !strings.Contains(w2.Body.String(), "已存在") && w2.Code != http.StatusBadRequest {
		t.Logf("重复回传响应: %d %s", w2.Code, w2.Body.String())
	}
}

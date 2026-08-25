package handlers

import (
	"os"
	"testing"

	"cboard-go/internal/models"
	"cboard-go/internal/services/selfhost"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/driver/sqlite"
)

// setupRedeployTestDB 创建内存 SQLite 测试库，并插入一个"已部署"的自建节点。
// setupRedeployTestDB 创建独立 SQLite 测试库（每测试唯一内存库），并返回 db。
func setupRedeployTestDB(t *testing.T) *gorm.DB {
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

// TestFindSelfHostNodeBySSHHost 验证占用检测：按 ssh_host+ssh_port 命中已有节点。
func TestFindSelfHostNodeBySSHHost(t *testing.T) {
	db := setupRedeployTestDB(t)

	// 插入已部署的主节点（deploy_mode=multi）
	main := models.CustomNode{
		Name: "东京VPS", DisplayName: "东京VPS", Protocol: "vless-ws", Port: 443,
		Status: selfhost.StatusOnline, IsActive: true, Source: "selfhost", SelfHosted: true,
		SelfHostProtocol: "vless-ws", InstallID: "abc123", DeployMode: "multi", SSHHost: "1.2.3.4", SSHPort: 22,
	}
	if err := db.Create(&main).Error; err != nil {
		t.Fatalf("插入主节点失败: %v", err)
	}
	// 子节点（同 VPS）
	child := models.CustomNode{
		Name: "东京VPS-SS", DisplayName: "东京VPS-SS", Protocol: "ss", Port: 8388,
		Status: selfhost.StatusOnline, IsActive: true, Source: "selfhost", SelfHosted: true,
		SelfHostProtocol: "ss", InstallID: "abc123", DeployMode: "", SSHHost: "1.2.3.4", SSHPort: 22,
	}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("插入子节点失败: %v", err)
	}

	// 命中主节点（deploy_mode 优先）
	found := findSelfHostNodeBySSHHost(db, "1.2.3.4", 22)
	if found == nil {
		t.Fatal("应该检测到已占用 VPS")
	}
	if found.ID != main.ID || found.DeployMode != "multi" {
		t.Fatalf("应命中主节点 #%d，实际 #%d (deploy_mode=%s)", main.ID, found.ID, found.DeployMode)
	}

	// 不存在的 VPS → nil
	if found := findSelfHostNodeBySSHHost(db, "9.9.9.9", 22); found != nil {
		t.Fatalf("不应命中不存在的 VPS")
	}

	// 已取消的节点不算占用（主+子都取消后）
	db.Model(&main).Update("status", selfhost.StatusCanceled)
	db.Model(&child).Update("status", selfhost.StatusCanceled)
	if found := findSelfHostNodeBySSHHost(db, "1.2.3.4", 22); found != nil {
		t.Fatalf("已取消节点不应算占用")
	}
}

// TestPrepareSelfHostNodeForDeploy_Reuse 验证复用重装：
// 复用已有节点 → 换新 install_id、状态重置 pending、子节点同步新 install_id。
func TestPrepareSelfHostNodeForDeploy_Reuse(t *testing.T) {
	db := setupRedeployTestDB(t)

	main := models.CustomNode{
		Name: "旧东京", DisplayName: "旧东京", Protocol: "vless-ws", Port: 443,
		Status: selfhost.StatusOnline, IsActive: true, Source: "selfhost", SelfHosted: true,
		SelfHostProtocol: "vless-ws", InstallID: "old-install", DeployMode: "multi", SSHHost: "5.6.7.8", SSHPort: 22,
	}
	if err := db.Create(&main).Error; err != nil {
		t.Fatal(err)
	}
	child := models.CustomNode{
		Name: "旧东京-SS", DisplayName: "旧东京-SS", Protocol: "ss", Port: 8388,
		Status: selfhost.StatusOnline, IsActive: true, Source: "selfhost", SelfHosted: true,
		SelfHostProtocol: "ss", InstallID: "old-install", DeployMode: "", SSHHost: "5.6.7.8", SSHPort: 22,
	}
	if err := db.Create(&child).Error; err != nil {
		t.Fatal(err)
	}

	node, installID, token, reused, err := prepareSelfHostNodeForDeploy(db, "新东京", "vless-ws", "5.6.7.8", 22, "root", main.ID)
	if err != nil {
		t.Fatalf("复用重装失败: %v", err)
	}
	if !reused {
		t.Fatal("应标记为复用")
	}
	if node.ID != main.ID {
		t.Fatalf("应复用原节点 #%d，实际 #%d", main.ID, node.ID)
	}
	if node.Status != selfhost.StatusPending || node.IsActive {
		t.Fatalf("复用后应重置为 pending+未激活，实际 status=%s is_active=%v", node.Status, node.IsActive)
	}
	if node.InstallID == "old-install" || node.InstallID != installID {
		t.Fatalf("install_id 应更换为新值 %s，实际 %s", installID, node.InstallID)
	}
	if token == "" {
		t.Fatal("token 不应为空")
	}
	// 子节点 install_id 同步
	var childAfter models.CustomNode
	db.First(&childAfter, child.ID)
	if childAfter.InstallID != installID {
		t.Fatalf("子节点 install_id 未同步：期望 %s 实际 %s", installID, childAfter.InstallID)
	}
	if childAfter.Status != selfhost.StatusPending {
		t.Fatalf("子节点状态应重置为 pending，实际 %s", childAfter.Status)
	}

	// 复用不存在的节点 → 报错
	if _, _, _, _, err := prepareSelfHostNodeForDeploy(db, "x", "vless-ws", "5.6.7.8", 22, "root", 99999); err == nil {
		t.Fatal("复用不存在的节点应报错")
	}
	// 复用其它 VPS 的节点 → 报错
	other := models.CustomNode{
		Name: "别的VPS", DisplayName: "别的VPS", Protocol: "ss", Status: selfhost.StatusPending,
		SelfHosted: true, SelfHostProtocol: "ss", SSHHost: "8.8.8.8", SSHPort: 22,
	}
	db.Create(&other)
	if _, _, _, _, err := prepareSelfHostNodeForDeploy(db, "x", "ss", "5.6.7.8", 22, "root", other.ID); err == nil {
		t.Fatal("复用其它 VPS 的节点应报错")
	}
}

// TestPrepareSelfHostNodeForDeploy_Create 验证非复用路径创建全新记录。
func TestPrepareSelfHostNodeForDeploy_Create(t *testing.T) {
	db := setupRedeployTestDB(t)
	node, installID, token, reused, err := prepareSelfHostNodeForDeploy(db, "新节点", "vless-ws", "1.1.1.1", 22, "root", 0)
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	if reused {
		t.Fatal("新建不应标记复用")
	}
	if node.ID == 0 || node.InstallID != installID || token == "" {
		t.Fatal("新记录字段不完整")
	}
	if node.SSHHost != "1.1.1.1" {
		t.Fatalf("SSHHost 未记录: %s", node.SSHHost)
	}
	var count int64
	db.Model(&models.CustomNode{}).Count(&count)
	if count != 1 {
		t.Fatalf("应只有 1 条记录，实际 %d", count)
	}
}

// TestParseProtocolList 验证多协议列表重建。
func TestParseProtocolList(t *testing.T) {
	protos := parseProtocolList("vless-ws,vless-reality,trojan-ws,ss", "node.example.com")
	if len(protos) != 4 {
		t.Fatalf("应解析出 4 个协议，实际 %d", len(protos))
	}
	if protos[0].Key != "vless-ws" || protos[0].Port != 443 || protos[0].Domain != "node.example.com" {
		t.Fatalf("vless-ws 解析错误: %+v", protos[0])
	}
	if protos[1].Port != 8444 {
		t.Fatalf("vless-reality 默认端口应为 8444，实际 %d", protos[1].Port)
	}
	if protos[3].Port != 8388 {
		t.Fatalf("ss 默认端口应为 8388，实际 %d", protos[3].Port)
	}
	// 空/逗号容忍
	if protos := parseProtocolList(" ,,", ""); len(protos) != 0 {
		t.Fatal("空列表应返回空")
	}
	// 未知协议仍按默认 443
	if protos := parseProtocolList("unknown-proto", ""); len(protos) != 1 || protos[0].Port != 443 {
		t.Fatal("未知协议端口应回落 443")
	}
}

// TestBuildScriptBackupSnippet 验证安装脚本包含旧配置备份逻辑（防二次部署无回退）。
func TestBuildScriptBackupSnippet(t *testing.T) {
	cfg := selfhost.ScriptConfig{
		PanelBaseURL: "http://127.0.0.1:18080",
		InstallID:    "testid",
		Token:        "testtoken",
		Protocol:     "vless-ws",
		NodeName:     "test",
		MirrorURLs:   selfhost.DefaultMirrorURLs(),
	}
	script, err := selfhost.BuildInstallScript(cfg)
	if err != nil {
		t.Fatalf("生成脚本失败: %v", err)
	}
	if !contains(script, "config.json.bak.") {
		t.Fatal("单协议脚本应包含旧配置备份")
	}
	if !contains(script, "已备份旧配置") {
		t.Fatal("单协议脚本应提示备份")
	}

	xcfg := selfhost.XrayScriptConfig{
		PanelBaseURL: "http://127.0.0.1:18080",
		InstallID:    "testid",
		Token:        "testtoken",
		Protocols:    selfhost.DefaultXrayProtocols("node.example.com"),
		MirrorURLs:   selfhost.DefaultMirrorURLs(),
	}
	xscript, err := selfhost.BuildXrayInstallScript(xcfg)
	if err != nil {
		t.Fatalf("生成多协议脚本失败: %v", err)
	}
	if !contains(xscript, "config.json.bak.") {
		t.Fatal("多协议脚本应包含旧配置备份")
	}
	if !contains(xscript, "已备份旧配置") {
		t.Fatal("多协议脚本应提示备份")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

// TestMain 确保测试不依赖真实 cboard.db（防止污染）。
func TestMain(m *testing.M) {
	// 保存并清空可能影响路径解析的环境变量
	origEnv := os.Getenv("DATABASE_URL")
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("SECRET_KEY")
	code := m.Run()
	if origEnv != "" {
		os.Setenv("DATABASE_URL", origEnv)
	}
	os.Exit(code)
}


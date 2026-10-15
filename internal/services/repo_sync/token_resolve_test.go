package repo_sync

import (
	"strings"
	"testing"

	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

// 库里 Token 是「解不开的加密串」时：LoadConfig 必须给出 TokenError，
// 同步 / 测试连接都必须**直接报可读原因**，绝不能拿密文去请求 GitHub
// （线上老站就是这么表现为 401 Bad credentials 的）。
func TestLoadConfigReportsUnresolvableToken(t *testing.T) {
	db := setupRepoSyncTestDB(t)
	enc := utils.EncryptedSecretPrefix + "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo="
	rows := []models.SystemConfig{
		{Key: KeyEnabled, Value: "true", Category: Category},
		{Key: KeyToken, Value: enc, Category: Category},
		{Key: KeyOwner, Value: "moneyfly006", Category: Category},
		{Key: KeyRepo, Value: "nodes", Category: Category},
		{Key: KeyPath, Value: "nodes", Category: Category},
	}
	for _, r := range rows {
		if err := db.Create(&r).Error; err != nil {
			t.Fatalf("写入配置失败: %v", err)
		}
	}

	svc := &Service{db: db}
	cfg := svc.LoadConfig()
	if cfg.TokenError == "" {
		t.Fatal("无法解密的 Token 应写入 TokenError")
	}
	if cfg.Token != "" {
		t.Fatalf("TokenError 非空时 Token 必须为空（不能把密文当明文），实际 %q", cfg.Token)
	}

	// 同步入口：报可读原因 + 状态写 failed（而不是去撞 GitHub 拿 401）
	if _, err := svc.SyncNow(); err == nil {
		t.Fatal("Token 不可用时同步必须失败")
	} else if !strings.Contains(err.Error(), "重新填写") {
		t.Fatalf("错误信息应提示重新填写 Token，实际: %v", err)
	}
	var status models.SystemConfig
	if err := db.Where("key = ? AND category = ?", KeyLastStatus, Category).First(&status).Error; err != nil {
		t.Fatalf("读同步状态失败: %v", err)
	}
	if status.Value != "failed" {
		t.Fatalf("同步状态应为 failed，实际 %q", status.Value)
	}
}

// 管理端刚填的 Token（明文）仍然正常工作路径：TestConnectionWith 拿到明文后
// 不应该因为「还原」逻辑把它弄坏（这里只验证不会返回“Token 无法使用”类错误）。
func TestTestConnectionWithRejectsEncryptedBlob(t *testing.T) {
	db := setupRepoSyncTestDB(t)
	seedRepoSyncConfig(t, db, true)
	svc := &Service{db: db}

	enc := utils.EncryptedSecretPrefix + "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo="
	_, err := svc.TestConnectionWith(enc, "moneyfly006", "nodes", "nodes")
	if err == nil {
		t.Fatal("把加密串当 Token 传进来必须报错")
	}
	if !strings.Contains(err.Error(), "Token 无法使用") {
		t.Fatalf("错误信息应说明 Token 无法使用，实际: %v", err)
	}
}

var _ = gorm.DB{}

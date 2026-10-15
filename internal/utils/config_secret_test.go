package utils

import (
	"strings"
	"testing"
)

// 敏感配置值的读取兼容（线上事故复现）：
// 老站 dy.moneyfly.top 的 repo_sync_token 等 4 个 Token 被外部系统写成了
// "enc:v1:" + AES-GCM 密文（密钥还不是当前 SECRET_KEY）。面板读取时不解密，
// 把密文当 Token 发给 GitHub，得到 `401 Bad credentials` —— 看起来像 Token 失效，
// 实际是存进去的值根本没被还原。这几条用例把读取侧的行为钉住。
func TestResolveConfigSecret(t *testing.T) {
	t.Run("空值原样返回", func(t *testing.T) {
		got, err := ResolveConfigSecret("")
		if err != nil || got != "" {
			t.Fatalf("空值应返回空串且无错误，实际 %q / %v", got, err)
		}
	})

	t.Run("普通明文原样返回（绝大多数情况）", func(t *testing.T) {
		got, err := ResolveConfigSecret("  ghp_plaintextToken  ")
		if err != nil {
			t.Fatalf("明文不该报错: %v", err)
		}
		if got != "ghp_plaintextToken" {
			t.Fatalf("明文应原样返回（仅去空白），实际 %q", got)
		}
	})

	t.Run("enc:v1 前缀可解密回明文", func(t *testing.T) {
		enc, err := EncryptSecret("ghp_roundTripToken")
		if err != nil {
			t.Fatalf("加密失败: %v", err)
		}
		got, err := ResolveConfigSecret(EncryptedSecretPrefix + enc)
		if err != nil {
			t.Fatalf("可解密的密文不该报错: %v", err)
		}
		if got != "ghp_roundTripToken" {
			t.Fatalf("解密结果不符: %q", got)
		}
	})

	t.Run("解不开的密文明确报错，绝不返回密文本体", func(t *testing.T) {
		// 用一个「合法 base64 但不是本机密钥加密的」密文模拟线上情况
		bogus := EncryptedSecretPrefix + "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo="
		got, err := ResolveConfigSecret(bogus)
		if err == nil {
			t.Fatal("解不开的密文必须报错")
		}
		if got != "" {
			t.Fatalf("报错时必须返回空值（不能把密文当明文用），实际 %q", got)
		}
		if !strings.Contains(err.Error(), "无法解密") {
			t.Fatalf("错误信息应说明无法解密，实际: %v", err)
		}
		if !LooksEncryptedSecret(bogus) {
			t.Fatal("LooksEncryptedSecret 应识别 enc:v1 前缀")
		}
	})
}

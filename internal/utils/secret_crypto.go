package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"cboard-go/internal/core/config"
)

// EncryptedSecretPrefix 是**历史遗留**的加密前缀：线上出现过把敏感配置值写成
// "enc:v1:" + base64(AES-GCM) 的情况（2026-10-10 同一时刻 backup/smtp/github
// 等 4 个 Token 被同批加密，密钥还不是当前 SECRET_KEY）。
//
// Go 面板自己从不写这个前缀，但读取侧必须兼容：否则会把密文当 Token 原样发给
// GitHub，得到 `401 Bad credentials`，表面上像"Token 失效"，实际是存进去的值
// 根本没被解密 —— 排查成本极高。这里统一做一次还原 + 明确报错。
const EncryptedSecretPrefix = "enc:v1:"

// ResolveConfigSecret 把配置里的敏感值还原成可直接使用的明文。
//   - 空值 → 空串；
//   - 普通明文 → 原样返回（绝大多数情况）；
//   - "enc:v1:xxx" → 用 SECRET_KEY 解密；解不开时返回可读错误，
//     由调用方提示"请重新填写"，绝不把密文当明文用。
func ResolveConfigSecret(value string) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", nil
	}
	if !strings.HasPrefix(v, EncryptedSecretPrefix) {
		return v, nil
	}
	plain, err := DecryptSecret(strings.TrimPrefix(v, EncryptedSecretPrefix))
	if err != nil {
		return "", fmt.Errorf("该配置项以 %s 加密存储，但用当前 SECRET_KEY 无法解密（可能由其它系统或旧密钥写入）：%w",
			EncryptedSecretPrefix, err)
	}
	return plain, nil
}

// LooksEncryptedSecret 判断配置值是否为加密形态，供管理端提示"请重新填写"。
func LooksEncryptedSecret(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), EncryptedSecretPrefix)
}

// EncryptSecret 用 SECRET_KEY 派生密钥对敏感数据（如 SSH 密码）做 AES-GCM 加密。
// 返回 base64(随机nonce + 密文)。
func EncryptSecret(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	key := deriveSecretKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptSecret 解密 EncryptSecret 的产物。
func DecryptSecret(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	key := deriveSecretKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", errors.New("密文长度不足")
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("解密失败（SECRET_KEY 可能已变更）")
	}
	return string(plaintext), nil
}

// deriveSecretKey 从 SECRET_KEY 派生 32 字节 AES 密钥。
//
// 注意 AppConfig 可能为 nil（单测、或配置尚未加载时调用）：直接取字段会 panic
// （用 nil 指针解引用崩在加解密路径上）。这里做 nil 防护。
//
// 未配置 SECRET_KEY 时**不再使用写死的兜底字符串**：那等于全网面板共用一把密钥，
// 任何人拿到密文都能离线解开。改为进程内随机密钥（至少要求攻击者拿到进程内存），
// 并大声告警——随机密钥重启即变，历史密文会解不开，所以这只是"不要用弱密钥"的兜底，
// 生产环境必须配置 SECRET_KEY。
var (
	fallbackSecretOnce sync.Once
	fallbackSecret     []byte
)

func deriveSecretKey() []byte {
	key := ""
	if config.AppConfig != nil {
		key = config.AppConfig.SecretKey
	}
	if key == "" {
		fallbackSecretOnce.Do(func() {
			buf := make([]byte, 32)
			if _, err := rand.Read(buf); err != nil {
				sum := sha256.Sum256([]byte("cboard-fallback-secret-key"))
				fallbackSecret = sum[:]
			} else {
				fallbackSecret = buf
			}
			LogErrorMsg("[安全告警] 未配置 SECRET_KEY：敏感配置使用进程内随机密钥，重启后历史密文将无法解密，请尽快在 .env 配置 SECRET_KEY")
		})
		return fallbackSecret
	}
	sum := sha256.Sum256([]byte(key))
	return sum[:]
}

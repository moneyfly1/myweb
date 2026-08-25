package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"

	"cboard-go/internal/core/config"
)

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
func deriveSecretKey() []byte {
	key := config.AppConfig.SecretKey
	if key == "" {
		key = "cboard-default-fallback-key-please-set-secret"
	}
	sum := sha256.Sum256([]byte(key))
	return sum[:]
}

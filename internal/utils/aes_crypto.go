package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

var aesKey = []byte("cboard-secret-key-32-bytes!!") // 32字节密钥

func EncryptAES(plaintext string) (string, error) {
	key := make([]byte, 32)
	copy(key, aesKey)
	if len(aesKey) < 32 {
		for i := len(aesKey); i < 32; i++ {
			key[i] = 0
		}
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("创建AES cipher失败: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM失败: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成nonce失败: %w", err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptAES(ciphertext string) (string, error) {
	key := make([]byte, 32)
	copy(key, aesKey)
	if len(aesKey) < 32 {
		for i := len(aesKey); i < 32; i++ {
			key[i] = 0
		}
	}

	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("解码base64失败: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("创建AES cipher失败: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM失败: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertextBytes) < nonceSize {
		return "", fmt.Errorf("密文太短")
	}

	nonce, ciphertextBytes := ciphertextBytes[:nonceSize], ciphertextBytes[nonceSize:]

	plaintext, err := aesGCM.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}

	return string(plaintext), nil
}

func IsEncrypted(text string) bool {
	if len(text) < 20 {
		return false
	}
	_, err := base64.StdEncoding.DecodeString(text)
	return err == nil
}

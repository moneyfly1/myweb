package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// 使用固定的密钥（从环境变量或配置中获取，这里简化处理）
// 生产环境应该从配置文件或环境变量中读取
var aesKey = []byte("cboard-secret-key-32-bytes!!") // 32字节密钥

// EncryptAES 使用AES加密字符串
func EncryptAES(plaintext string) (string, error) {
	// 确保密钥长度为32字节
	key := make([]byte, 32)
	copy(key, aesKey)
	if len(aesKey) < 32 {
		// 如果密钥太短，用0填充
		for i := len(aesKey); i < 32; i++ {
			key[i] = 0
		}
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("创建AES cipher失败: %w", err)
	}

	// 使用GCM模式
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM失败: %w", err)
	}

	// 生成随机nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成nonce失败: %w", err)
	}

	// 加密
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptAES 使用AES解密字符串
func DecryptAES(ciphertext string) (string, error) {
	// 确保密钥长度为32字节
	key := make([]byte, 32)
	copy(key, aesKey)
	if len(aesKey) < 32 {
		for i := len(aesKey); i < 32; i++ {
			key[i] = 0
		}
	}

	// 解码base64
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

	// 提取nonce和密文
	nonce, ciphertextBytes := ciphertextBytes[:nonceSize], ciphertextBytes[nonceSize:]

	// 解密
	plaintext, err := aesGCM.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}

	return string(plaintext), nil
}

// IsEncrypted 检查字符串是否是AES加密的
func IsEncrypted(text string) bool {
	// 简单检查：如果是base64编码且长度足够，可能是加密的
	if len(text) < 20 {
		return false
	}
	_, err := base64.StdEncoding.DecodeString(text)
	return err == nil
}



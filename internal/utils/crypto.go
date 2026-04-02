package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func NormalizePrivateKey(privateKey string) string {
	privateKey = strings.TrimSpace(privateKey)
	if privateKey == "" {
		return ""
	}

	if strings.Contains(privateKey, "BEGIN") {
		privateKey = strings.ReplaceAll(privateKey, "\r\n", "\n")
		privateKey = strings.ReplaceAll(privateKey, "\r", "\n")
		return privateKey
	}

	cleanKey := strings.ReplaceAll(privateKey, "\n", "")
	cleanKey = strings.ReplaceAll(cleanKey, "\r", "")
	cleanKey = strings.ReplaceAll(cleanKey, " ", "")
	cleanKey = strings.ReplaceAll(cleanKey, "\t", "")

	if strings.HasPrefix(cleanKey, "MII") || strings.HasPrefix(cleanKey, "MIIC") {
		privateKey = cleanKey
		if !strings.HasPrefix(privateKey, "-----BEGIN RSA PRIVATE KEY-----") {
			privateKey = "-----BEGIN RSA PRIVATE KEY-----\n" + privateKey
		}
		if !strings.HasSuffix(strings.TrimSpace(privateKey), "-----END RSA PRIVATE KEY-----") {
			privateKey = privateKey + "\n-----END RSA PRIVATE KEY-----"
		}
		privateKey = FormatPEMKey(privateKey, "RSA PRIVATE KEY")
		return privateKey
	}

	if strings.HasPrefix(cleanKey, "MIIE") || strings.HasPrefix(cleanKey, "MIIEv") {
		privateKey = cleanKey
		if !strings.HasPrefix(privateKey, "-----BEGIN PRIVATE KEY-----") {
			privateKey = "-----BEGIN PRIVATE KEY-----\n" + privateKey
		}
		if !strings.HasSuffix(strings.TrimSpace(privateKey), "-----END PRIVATE KEY-----") {
			privateKey = privateKey + "\n-----END PRIVATE KEY-----"
		}
		privateKey = FormatPEMKey(privateKey, "PRIVATE KEY")
		return privateKey
	}

	if len(cleanKey) > 100 {
		privateKey = cleanKey
		privateKey = "-----BEGIN RSA PRIVATE KEY-----\n" + privateKey + "\n-----END RSA PRIVATE KEY-----"
		privateKey = FormatPEMKey(privateKey, "RSA PRIVATE KEY")
		return privateKey
	}

	return ""
}

func NormalizePublicKey(publicKey string) string {
	publicKey = strings.TrimSpace(publicKey)
	if publicKey == "" {
		return ""
	}

	if strings.Contains(publicKey, "BEGIN") {
		publicKey = strings.ReplaceAll(publicKey, "\r\n", "\n")
		publicKey = strings.ReplaceAll(publicKey, "\r", "\n")
		return publicKey
	}

	cleanKey := strings.ReplaceAll(publicKey, "\n", "")
	cleanKey = strings.ReplaceAll(cleanKey, "\r", "")
	cleanKey = strings.ReplaceAll(cleanKey, " ", "")
	cleanKey = strings.ReplaceAll(cleanKey, "\t", "")

	if strings.HasPrefix(cleanKey, "MIGf") || strings.HasPrefix(cleanKey, "MIIBIjAN") || strings.HasPrefix(cleanKey, "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A") {
		publicKey = cleanKey
		if !strings.HasPrefix(publicKey, "-----BEGIN PUBLIC KEY-----") {
			publicKey = "-----BEGIN PUBLIC KEY-----\n" + publicKey
		}
		if !strings.HasSuffix(strings.TrimSpace(publicKey), "-----END PUBLIC KEY-----") {
			publicKey = publicKey + "\n-----END PUBLIC KEY-----"
		}
		return FormatPEMPublicKey(publicKey)
	}

	if len(cleanKey) > 50 {
		publicKey = cleanKey
		publicKey = "-----BEGIN PUBLIC KEY-----\n" + publicKey + "\n-----END PUBLIC KEY-----"
		return FormatPEMPublicKey(publicKey)
	}

	return ""
}

func FormatPEMPublicKey(key string) string {
	beginMarker := "-----BEGIN PUBLIC KEY-----"
	endMarker := "-----END PUBLIC KEY-----"

	key = strings.TrimPrefix(key, beginMarker)
	key = strings.TrimSuffix(key, endMarker)
	key = strings.TrimSpace(key)

	key = strings.ReplaceAll(key, "\n", "")
	key = strings.ReplaceAll(key, "\r", "")
	key = strings.ReplaceAll(key, " ", "")
	key = strings.ReplaceAll(key, "\t", "")

	var formatted strings.Builder
	formatted.WriteString(beginMarker)
	formatted.WriteString("\n")
	for i := 0; i < len(key); i += 64 {
		end := i + 64
		if end > len(key) {
			end = len(key)
		}
		formatted.WriteString(key[i:end])
		if end < len(key) {
			formatted.WriteString("\n")
		}
	}
	formatted.WriteString("\n")
	formatted.WriteString(endMarker)

	return formatted.String()
}

func FormatPEMKey(key, keyType string) string {
	beginMarker := fmt.Sprintf("-----BEGIN %s-----", keyType)
	endMarker := fmt.Sprintf("-----END %s-----", keyType)

	key = strings.TrimPrefix(key, beginMarker)
	key = strings.TrimSuffix(key, endMarker)
	key = strings.TrimSpace(key)

	key = strings.ReplaceAll(key, "\n", "")
	key = strings.ReplaceAll(key, "\r", "")
	key = strings.ReplaceAll(key, " ", "")
	key = strings.ReplaceAll(key, "\t", "")

	var formatted strings.Builder
	formatted.WriteString(beginMarker)
	formatted.WriteString("\n")
	for i := 0; i < len(key); i += 64 {
		end := i + 64
		if end > len(key) {
			end = len(key)
		}
		formatted.WriteString(key[i:end])
		if end < len(key) {
			formatted.WriteString("\n")
		}
	}
	formatted.WriteString("\n")
	formatted.WriteString(endMarker)

	return formatted.String()
}

// ========== AES加密相关 ==========

var aesKey []byte

func init() {
	// 从环境变量读取AES密钥，生产环境必须设置
	key := os.Getenv("AES_ENCRYPTION_KEY")
	if key == "" {
		// 开发环境使用默认密钥（仅用于开发测试）
		if os.Getenv("ENV") != "production" {
			key = "cboard-dev-secret-key-32-bytes!!"
			log.Println("警告: 使用开发环境默认AES密钥，生产环境请设置 AES_ENCRYPTION_KEY 环境变量")
		} else {
			log.Fatal("生产环境必须设置 AES_ENCRYPTION_KEY 环境变量")
		}
	}
	
	if len(key) < 32 {
		log.Fatalf("AES_ENCRYPTION_KEY 必须至少 32 字节，当前为 %d 字节", len(key))
	}
	
	aesKey = []byte(key)[:32] // 确保只取前32字节
}

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

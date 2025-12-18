package utils

import (
	"fmt"
	"strings"
)

// NormalizePrivateKey 规范化私钥格式
// 支持多种输入格式，自动转换为标准的PEM格式
func NormalizePrivateKey(privateKey string) string {
	privateKey = strings.TrimSpace(privateKey)
	if privateKey == "" {
		return ""
	}

	// 如果已经包含 BEGIN 标记，说明已经是PEM格式，直接返回
	if strings.Contains(privateKey, "BEGIN") {
		// 确保有正确的换行符
		privateKey = strings.ReplaceAll(privateKey, "\r\n", "\n")
		privateKey = strings.ReplaceAll(privateKey, "\r", "\n")
		return privateKey
	}

	// 如果没有 BEGIN 标记，尝试自动添加
	// 移除所有空白字符以便识别
	cleanKey := strings.ReplaceAll(privateKey, "\n", "")
	cleanKey = strings.ReplaceAll(cleanKey, "\r", "")
	cleanKey = strings.ReplaceAll(cleanKey, " ", "")
	cleanKey = strings.ReplaceAll(cleanKey, "\t", "")

	// 检查是否是PKCS1格式（RSA PRIVATE KEY）
	// PKCS1格式通常以 MII 或 MIIC 开头
	if strings.HasPrefix(cleanKey, "MII") || strings.HasPrefix(cleanKey, "MIIC") {
		// 可能是PKCS1格式，添加PKCS1标记
		privateKey = cleanKey
		if !strings.HasPrefix(privateKey, "-----BEGIN RSA PRIVATE KEY-----") {
			privateKey = "-----BEGIN RSA PRIVATE KEY-----\n" + privateKey
		}
		if !strings.HasSuffix(strings.TrimSpace(privateKey), "-----END RSA PRIVATE KEY-----") {
			privateKey = privateKey + "\n-----END RSA PRIVATE KEY-----"
		}
		// 确保每64个字符换行（PEM格式标准）
		privateKey = FormatPEMKey(privateKey, "RSA PRIVATE KEY")
		return privateKey
	}

	// 检查是否是PKCS8格式（PRIVATE KEY）
	// PKCS8格式通常以 MIIE 或 MIIEv 开头
	if strings.HasPrefix(cleanKey, "MIIE") || strings.HasPrefix(cleanKey, "MIIEv") {
		// 可能是PKCS8格式，添加PKCS8标记
		privateKey = cleanKey
		if !strings.HasPrefix(privateKey, "-----BEGIN PRIVATE KEY-----") {
			privateKey = "-----BEGIN PRIVATE KEY-----\n" + privateKey
		}
		if !strings.HasSuffix(strings.TrimSpace(privateKey), "-----END PRIVATE KEY-----") {
			privateKey = privateKey + "\n-----END PRIVATE KEY-----"
		}
		// 确保每64个字符换行（PEM格式标准）
		privateKey = FormatPEMKey(privateKey, "PRIVATE KEY")
		return privateKey
	}

	// 如果无法识别格式，尝试作为PKCS1格式处理（最常见）
	if len(cleanKey) > 100 {
		privateKey = cleanKey
		privateKey = "-----BEGIN RSA PRIVATE KEY-----\n" + privateKey + "\n-----END RSA PRIVATE KEY-----"
		privateKey = FormatPEMKey(privateKey, "RSA PRIVATE KEY")
		return privateKey
	}

	return ""
}

// NormalizePublicKey 规范化公钥格式
// 支持多种输入格式，自动转换为标准的PEM格式
func NormalizePublicKey(publicKey string) string {
	publicKey = strings.TrimSpace(publicKey)
	if publicKey == "" {
		return ""
	}

	// 如果已经包含 BEGIN 标记，说明已经是PEM格式
	if strings.Contains(publicKey, "BEGIN") {
		// 确保有正确的换行符
		publicKey = strings.ReplaceAll(publicKey, "\r\n", "\n")
		publicKey = strings.ReplaceAll(publicKey, "\r", "\n")
		return publicKey
	}

	// 如果没有 BEGIN 标记，尝试自动添加
	// 移除所有空白字符以便识别
	cleanKey := strings.ReplaceAll(publicKey, "\n", "")
	cleanKey = strings.ReplaceAll(cleanKey, "\r", "")
	cleanKey = strings.ReplaceAll(cleanKey, " ", "")
	cleanKey = strings.ReplaceAll(cleanKey, "\t", "")

	// 公钥通常以 MIGf 或 MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A 开头
	if strings.HasPrefix(cleanKey, "MIGf") || strings.HasPrefix(cleanKey, "MIIBIjAN") || strings.HasPrefix(cleanKey, "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A") {
		// 添加 PUBLIC KEY 标记
		publicKey = cleanKey
		if !strings.HasPrefix(publicKey, "-----BEGIN PUBLIC KEY-----") {
			publicKey = "-----BEGIN PUBLIC KEY-----\n" + publicKey
		}
		if !strings.HasSuffix(strings.TrimSpace(publicKey), "-----END PUBLIC KEY-----") {
			publicKey = publicKey + "\n-----END PUBLIC KEY-----"
		}
		// 格式化公钥
		return FormatPEMPublicKey(publicKey)
	}

	// 如果无法识别格式，尝试作为标准公钥处理
	if len(cleanKey) > 50 {
		publicKey = cleanKey
		publicKey = "-----BEGIN PUBLIC KEY-----\n" + publicKey + "\n-----END PUBLIC KEY-----"
		return FormatPEMPublicKey(publicKey)
	}

	return ""
}

// FormatPEMPublicKey 格式化PEM公钥，确保每64个字符换行
func FormatPEMPublicKey(key string) string {
	beginMarker := "-----BEGIN PUBLIC KEY-----"
	endMarker := "-----END PUBLIC KEY-----"

	// 移除已有的标记
	key = strings.TrimPrefix(key, beginMarker)
	key = strings.TrimSuffix(key, endMarker)
	key = strings.TrimSpace(key)

	// 移除所有空白字符（包括换行符和空格）
	key = strings.ReplaceAll(key, "\n", "")
	key = strings.ReplaceAll(key, "\r", "")
	key = strings.ReplaceAll(key, " ", "")
	key = strings.ReplaceAll(key, "\t", "")

	// 每64个字符换行
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

// FormatPEMKey 格式化PEM密钥，确保每64个字符换行
func FormatPEMKey(key, keyType string) string {
	// 提取BEGIN和END标记之间的内容
	beginMarker := fmt.Sprintf("-----BEGIN %s-----", keyType)
	endMarker := fmt.Sprintf("-----END %s-----", keyType)

	// 移除已有的标记
	key = strings.TrimPrefix(key, beginMarker)
	key = strings.TrimSuffix(key, endMarker)
	key = strings.TrimSpace(key)

	// 移除所有空白字符（包括换行符和空格）
	key = strings.ReplaceAll(key, "\n", "")
	key = strings.ReplaceAll(key, "\r", "")
	key = strings.ReplaceAll(key, " ", "")
	key = strings.ReplaceAll(key, "\t", "")

	// 每64个字符换行
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


package utils

import (
	"fmt"
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

	// 注意顺序：PKCS#8 私钥 base64 以 "MIIEv" 开头，也以 "MII" 开头，
	// 必须先判更具体的 MIIEv，否则 PKCS#8 会被误包成 RSA 标签
	if strings.HasPrefix(cleanKey, "MIIEv") {
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

	if strings.HasPrefix(cleanKey, "MII") {
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
		return FormatPEMKey(publicKey, "PUBLIC KEY")
	}

	if len(cleanKey) > 50 {
		publicKey = cleanKey
		publicKey = "-----BEGIN PUBLIC KEY-----\n" + publicKey + "\n-----END PUBLIC KEY-----"
		return FormatPEMKey(publicKey, "PUBLIC KEY")
	}

	return ""
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

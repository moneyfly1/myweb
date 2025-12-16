package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashToken 计算Token的哈希值（用于黑名单）
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}


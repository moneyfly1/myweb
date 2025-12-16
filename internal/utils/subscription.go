package utils

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateSubscriptionURL 生成订阅 URL
func GenerateSubscriptionURL() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}


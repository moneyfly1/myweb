package utils

import (
	"encoding/base64"
	"github.com/google/uuid"
)

// Base64Encode 对字符串进行Base64编码
func Base64Encode(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

// GenerateUUID 生成UUID
func GenerateUUID() string {
	return uuid.New().String()
}


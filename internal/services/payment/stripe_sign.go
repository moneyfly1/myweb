package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

// verifyStripeSignature 验证 Stripe Webhook 签名。
// 签名格式: t=<timestamp>,v1=<hmac_hex>
// HMAC 计算: HMAC-SHA256(secret, t.timestamp + "." + payload)
func verifyStripeSignature(payload []byte, signatureHeader, webhookSecret string) bool {
	if signatureHeader == "" {
		return false
	}
	parts := strings.Split(signatureHeader, ",")
	var timestamp string
	var signature string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "t=") {
			timestamp = p[2:]
		} else if strings.HasPrefix(p, "v1=") {
			signature = p[3:]
		}
	}
	if timestamp == "" || signature == "" {
		return false
	}

	// 时间戳防重放（5 分钟窗口）
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	if time.Since(time.Unix(ts, 0)) > 5*time.Minute || time.Since(time.Unix(ts, 0)) < -5*time.Minute {
		return false
	}

	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write([]byte(timestamp + "." + string(payload)))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

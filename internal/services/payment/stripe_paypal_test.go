package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"testing"
	"time"
)

func TestVerifyStripeSignature(t *testing.T) {
	secret := "whsec_test_secret"
	payload := []byte(`{"id":"evt_test","type":"checkout.session.completed","data":{"object":{"id":"cs_test","client_reference_id":"ORD123"}}}`)

	// 构造合法签名
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "." + string(payload)))
	sig := hex.EncodeToString(mac.Sum(nil))
	header := "t=" + ts + ",v1=" + sig

	if !verifyStripeSignature(payload, header, secret) {
		t.Error("合法签名应验证通过")
	}

	// 篡改 payload
	if verifyStripeSignature([]byte(`{"id":"hacked"}`), header, secret) {
		t.Error("篡改 payload 应验证失败")
	}

	// 错误 secret
	if verifyStripeSignature(payload, header, "wrong_secret") {
		t.Error("错误 secret 应验证失败")
	}

	// 空头
	if verifyStripeSignature(payload, "", secret) {
		t.Error("空签名头应验证失败")
	}

	// 过期时间戳（重放）
	oldTs := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
	mac2 := hmac.New(sha256.New, []byte(secret))
	mac2.Write([]byte(oldTs + "." + string(payload)))
	sig2 := hex.EncodeToString(mac2.Sum(nil))
	if verifyStripeSignature(payload, "t="+oldTs+",v1="+sig2, secret) {
		t.Error("过期时间戳应验证失败")
	}
}

func TestStripeEventExtractOrderNo(t *testing.T) {
	evt := &StripeEvent{}
	evt.Data.Object.Metadata = map[string]string{"order_no": "ORD123"}
	if got := evt.ExtractOrderNo(); got != "ORD123" {
		t.Errorf("metadata 提取失败: %s", got)
	}

	evt2 := &StripeEvent{}
	evt2.Data.Object.ClientReferenceID = "ORD456"
	if got := evt2.ExtractOrderNo(); got != "ORD456" {
		t.Errorf("client_reference_id 提取失败: %s", got)
	}
}

func TestStripeEventIsPaid(t *testing.T) {
	paid := &StripeEvent{}
	paid.Type = "checkout.session.completed"
	paid.Data.Object.PaymentStatus = "paid"
	if !paid.IsPaid() {
		t.Error("paid 事件应判定为已支付")
	}

	unpaid := &StripeEvent{}
	unpaid.Type = "payment_intent.created"
	if unpaid.IsPaid() {
		t.Error("created 事件不应判定为已支付")
	}
}

func TestPayPalWebhookExtract(t *testing.T) {
	payload := &PayPalWebhookPayload{}
	payload.Resource.CustomID = "ORD789"
	if got := payload.ExtractOrderNo(); got != "ORD789" {
		t.Errorf("paypal custom_id 提取失败: %s", got)
	}
	if payload.IsPaid() {
		t.Error("空事件不应判定已支付")
	}
	payload.EventType = "PAYMENT.CAPTURE.COMPLETED"
	if !payload.IsPaid() {
		t.Error("PAYMENT.CAPTURE.COMPLETED 应判定已支付")
	}
}

package utils

import (
	"testing"
)

func TestGenerateCouponCode(t *testing.T) {
	code := GenerateCouponCode()

	if len(code) != 8 {
		t.Errorf("优惠券码长度应为 8，实际为 %d", len(code))
	}

	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for _, char := range code {
		found := false
		for _, validChar := range charset {
			if char == validChar {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("优惠券码包含无效字符: %c", char)
		}
	}

	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code := GenerateCouponCode()
		if codes[code] {
			t.Errorf("生成了重复的优惠券码: %s", code)
		}
		codes[code] = true
	}
}

func TestGenerateOrderNo(t *testing.T) {
	orderNo, err := GenerateOrderNo(nil)
	if err != nil {
		t.Errorf("生成订单号失败: %v", err)
		return
	}

	if len(orderNo) != 14 {
		t.Errorf("订单号长度应为 14，实际为 %d", len(orderNo))
	}

	if orderNo[:3] != "ORD" {
		t.Errorf("订单号应以 'ORD' 开头，实际为 %s", orderNo[:3])
	}

	datePart := orderNo[3:11]
	if len(datePart) != 8 {
		t.Errorf("订单号日期部分应为 8 位，实际为 %s", datePart)
	}

	seqPart := orderNo[11:]
	if len(seqPart) != 3 {
		t.Errorf("订单号序号部分应为 3 位，实际为 %s", seqPart)
	}
}

func TestGenerateRechargeOrderNo(t *testing.T) {
	userID := uint(456)
	orderNo, err := GenerateRechargeOrderNo(userID, nil)
	if err != nil {
		t.Errorf("生成充值订单号失败: %v", err)
		return
	}

	if orderNo[:3] != "RCH" {
		t.Errorf("充值订单号应以 'RCH' 开头，实际为 %s", orderNo[:3])
	}

	if len(orderNo) != 14 {
		t.Errorf("充值订单号长度应为 14，实际为 %d", len(orderNo))
	}
}

func TestGenerateTicketNo(t *testing.T) {
	userID := uint(789)
	ticketNo := GenerateTicketNo(userID)

	if ticketNo[:3] != "TKT" {
		t.Errorf("工单号应以 'TKT' 开头，实际为 %s", ticketNo[:3])
	}
}

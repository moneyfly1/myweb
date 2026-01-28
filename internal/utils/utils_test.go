package utils

import (
	"testing"
)

// TestGenerateCouponCode 测试优惠券码生成
func TestGenerateCouponCode(t *testing.T) {
	code := GenerateCouponCode()

	// 测试长度
	if len(code) != 8 {
		t.Errorf("优惠券码长度应为 8，实际为 %d", len(code))
	}

	// 测试字符集
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

	// 测试唯一性（生成多个，应该不同）
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code := GenerateCouponCode()
		if codes[code] {
			t.Errorf("生成了重复的优惠券码: %s", code)
		}
		codes[code] = true
	}
}

// TestGenerateOrderNo 测试订单号生成
func TestGenerateOrderNo(t *testing.T) {
	orderNo, err := GenerateOrderNo(nil)
	if err != nil {
		t.Errorf("生成订单号失败: %v", err)
		return
	}

	// 测试格式：ORD + 日期(YYYYMMDD) + 序号(001-999)
	if len(orderNo) != 14 {
		t.Errorf("订单号长度应为 14，实际为 %d", len(orderNo))
	}

	// 测试前缀
	if orderNo[:3] != "ORD" {
		t.Errorf("订单号应以 'ORD' 开头，实际为 %s", orderNo[:3])
	}

	// 测试日期格式（应该是8位数字）
	datePart := orderNo[3:11]
	if len(datePart) != 8 {
		t.Errorf("订单号日期部分应为 8 位，实际为 %s", datePart)
	}

	// 测试序号格式（应该是3位数字）
	seqPart := orderNo[11:]
	if len(seqPart) != 3 {
		t.Errorf("订单号序号部分应为 3 位，实际为 %s", seqPart)
	}
}

// TestGenerateRechargeOrderNo 测试充值订单号生成
func TestGenerateRechargeOrderNo(t *testing.T) {
	userID := uint(456)
	orderNo, err := GenerateRechargeOrderNo(userID, nil)
	if err != nil {
		t.Errorf("生成充值订单号失败: %v", err)
		return
	}

	// 测试前缀
	if orderNo[:3] != "RCH" {
		t.Errorf("充值订单号应以 'RCH' 开头，实际为 %s", orderNo[:3])
	}

	// 测试格式：RCH + 日期(YYYYMMDD) + 序号(001-999)
	if len(orderNo) != 14 {
		t.Errorf("充值订单号长度应为 14，实际为 %d", len(orderNo))
	}
}

// TestGenerateTicketNo 测试工单号生成
func TestGenerateTicketNo(t *testing.T) {
	userID := uint(789)
	ticketNo := GenerateTicketNo(userID)

	// 测试前缀
	if ticketNo[:3] != "TKT" {
		t.Errorf("工单号应以 'TKT' 开头，实际为 %s", ticketNo[:3])
	}
}

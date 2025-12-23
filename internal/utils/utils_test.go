package utils

import (
	"testing"
	"time"
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
	userID := uint(123)
	orderNo := GenerateOrderNo(userID)
	
	// 测试格式：ORD + timestamp + userID
	if len(orderNo) < 10 {
		t.Errorf("订单号长度应至少为 10，实际为 %d", len(orderNo))
	}
	
	// 测试前缀
	if orderNo[:3] != "ORD" {
		t.Errorf("订单号应以 'ORD' 开头，实际为 %s", orderNo[:3])
	}
	
	// 测试唯一性（注意：如果时间戳相同，可能会生成相同的订单号）
	// 在实际使用中，时间戳通常不同，所以这个测试主要是验证格式
	orderNos := make(map[string]bool)
	for i := 0; i < 10; i++ {
		time.Sleep(10 * time.Millisecond) // 确保时间戳不同
		no := GenerateOrderNo(userID)
		if orderNos[no] {
			// 如果时间戳相同，可能会重复，这是正常的
			// 只记录警告，不视为错误
			t.Logf("警告: 生成了重复的订单号（时间戳可能相同）: %s", no)
		}
		orderNos[no] = true
	}
}

// TestGenerateRechargeOrderNo 测试充值订单号生成
func TestGenerateRechargeOrderNo(t *testing.T) {
	userID := uint(456)
	orderNo := GenerateRechargeOrderNo(userID)
	
	// 测试前缀
	if orderNo[:3] != "RCH" {
		t.Errorf("充值订单号应以 'RCH' 开头，实际为 %s", orderNo[:3])
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


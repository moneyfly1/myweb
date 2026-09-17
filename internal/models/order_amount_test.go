package models

import (
	"database/sql"
	"testing"
)

// 订单金额口径回归测试。
//
// 背景（线上真实事故）：余额支付订单创建时余额即被抵扣，FinalAmount 记为 0
// （其语义是"还需在线支付"），但仪表盘「最近订单」直接读 FinalAmount，
// 导致客户支付 200 元、后台显示 ¥0。这里锁定口径，防止再次回归。

func nullFloat(v float64) sql.NullFloat64 { return sql.NullFloat64{Float64: v, Valid: true} }
func nullStr(v string) sql.NullString     { return sql.NullString{String: v, Valid: true} }

// TestOrderPaidAmount_BalancePaid 线上问题订单：余额全额支付 200 元
func TestOrderPaidAmount_BalancePaid(t *testing.T) {
	order := Order{
		OrderNo:        "ORD202609160926117760",
		Amount:         200,
		DiscountAmount: nullFloat(0),
		FinalAmount:    nullFloat(0), // 余额已抵扣 → 还需在线支付 0
		ExtraData:      nullStr(`{"balance_used":200,"balance_deducted":true}`),
		Status:         "paid",
	}
	if got := order.PaidAmount(); got != 200 {
		t.Errorf("余额支付订单订单金额应为 200，实际 %.2f", got)
	}
	if got := order.AmountStillDueOnline(); got != 0 {
		t.Errorf("还需在线支付应为 0，实际 %.2f", got)
	}
	if !order.BalanceDeductedAtCreation() {
		t.Error("应识别为创建时已抵扣余额")
	}
}

// TestOrderPaidAmount_OnlinePaid 纯在线支付
func TestOrderPaidAmount_OnlinePaid(t *testing.T) {
	order := Order{Amount: 200, DiscountAmount: nullFloat(0), FinalAmount: nullFloat(200)}
	if got := order.PaidAmount(); got != 200 {
		t.Errorf("在线支付订单金额应为 200，实际 %.2f", got)
	}
}

// TestOrderPaidAmount_CouponDiscount 优惠券折扣
func TestOrderPaidAmount_CouponDiscount(t *testing.T) {
	order := Order{Amount: 200, DiscountAmount: nullFloat(20), FinalAmount: nullFloat(180)}
	if got := order.PaidAmount(); got != 180 {
		t.Errorf("折后订单金额应为 180，实际 %.2f", got)
	}
}

// TestOrderPaidAmount_PartialBalanceAtCreation 创建时部分余额抵扣
func TestOrderPaidAmount_PartialBalanceAtCreation(t *testing.T) {
	order := Order{
		Amount:         200,
		DiscountAmount: nullFloat(0),
		FinalAmount:    nullFloat(199.9),
		ExtraData:      nullStr(`{"balance_used":0.1,"balance_deducted":true}`),
	}
	if got := order.PaidAmount(); got != 200 {
		t.Errorf("订单金额应为 200，实际 %.2f", got)
	}
}

// TestOrderPaidAmount_BalanceAtPaymentTime 支付时才用余额（final 未减少）：
// 不得把余额重复加回，否则订单金额会翻倍（历史实现曾有此风险）
func TestOrderPaidAmount_BalanceAtPaymentTime(t *testing.T) {
	order := Order{
		Amount:         200,
		DiscountAmount: nullFloat(0),
		FinalAmount:    nullFloat(199.9), // 支付时 0.1 用余额、199.9 在线支付
		ExtraData:      nullStr(`{"balance_used":0.1,"balance_deducted":false}`),
	}
	if got := order.PaidAmount(); got != 200 {
		t.Errorf("订单金额应为 200（折后价，与资金来源无关），实际 %.2f", got)
	}
	if order.BalanceDeductedAtCreation() {
		t.Error("支付时抵扣不应被识别为创建时抵扣")
	}
}

// TestOrderPaidAmount_DiscountWithBalance 折扣 + 余额
func TestOrderPaidAmount_DiscountWithBalance(t *testing.T) {
	order := Order{
		Amount:         200,
		DiscountAmount: nullFloat(50),
		FinalAmount:    nullFloat(0),
		ExtraData:      nullStr(`{"balance_used":150,"balance_deducted":true}`),
	}
	if got := order.PaidAmount(); got != 150 {
		t.Errorf("折后余额支付订单金额应为 150，实际 %.2f", got)
	}
}

// TestOrderPaidAmount_FullCoupon 100% 优惠券：客户实付 0
func TestOrderPaidAmount_FullCoupon(t *testing.T) {
	order := Order{
		Amount:         200,
		DiscountAmount: nullFloat(200),
		FinalAmount:    nullFloat(0),
		ExtraData:      nullStr(`{}`),
	}
	if got := order.PaidAmount(); got != 0 {
		t.Errorf("全额优惠订单金额应为 0，实际 %.2f", got)
	}
}

// TestOrderPaidAmount_NoFinalAmount 历史订单 FinalAmount 为空 → 回退到折后价
func TestOrderPaidAmount_NoFinalAmount(t *testing.T) {
	order := Order{Amount: 66.6, DiscountAmount: sql.NullFloat64{}}
	if got := order.PaidAmount(); got != 66.6 {
		t.Errorf("FinalAmount 为空时订单金额应为 66.6，实际 %.2f", got)
	}
	if got := order.AmountStillDueOnline(); got != 66.6 {
		t.Errorf("FinalAmount 为空时还需在线支付应回退为 66.6，实际 %.2f", got)
	}
}

// TestOrderPaidAmount_DirtyData 脏数据不得导致 panic 或负值
func TestOrderPaidAmount_DirtyData(t *testing.T) {
	cases := []struct {
		name   string
		order  Order
		expect float64
	}{
		{"坏 JSON", Order{Amount: 200, FinalAmount: nullFloat(0), ExtraData: nullStr(`{bad json`)}, 200},
		{"余额为字符串", Order{Amount: 200, FinalAmount: nullFloat(0), ExtraData: nullStr(`{"balance_used":"200"}`)}, 200},
		{"负余额", Order{Amount: 200, FinalAmount: nullFloat(0), ExtraData: nullStr(`{"balance_used":-5}`)}, 200},
		{"优惠大于原价", Order{Amount: 100, DiscountAmount: nullFloat(150), FinalAmount: nullFloat(0)}, 0},
		{"负原价", Order{Amount: -10, FinalAmount: nullFloat(0)}, 0},
	}
	for _, c := range cases {
		if got := c.order.PaidAmount(); got != c.expect {
			t.Errorf("%s: 期望 %.2f，实际 %.2f", c.name, c.expect, got)
		}
		if got := c.order.PaidAmount(); got < 0 {
			t.Errorf("%s: 订单金额不应为负（%.2f）", c.name, got)
		}
	}
}

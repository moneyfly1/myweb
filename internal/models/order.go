package models

import (
	"database/sql"
	"encoding/json"
	"math"
	"time"
)

type Order struct {
	ID                   uint            `gorm:"primaryKey" json:"id"`
	OrderNo              string          `gorm:"type:varchar(50);uniqueIndex;not null" json:"order_no"`
	UserID               uint            `gorm:"index;index:idx_orders_user_status,priority:1;not null" json:"user_id"`
	PackageID            uint            `gorm:"index;not null" json:"package_id"`
	Amount               float64         `gorm:"type:decimal(10,2);not null" json:"amount"`
	Status               string          `gorm:"type:varchar(20);default:pending;index;index:idx_orders_status_created_at,priority:1;index:idx_orders_user_status,priority:2" json:"status"`
	PaymentMethodID      sql.NullInt64   `json:"payment_method_id,omitempty"`
	PaymentMethodName    sql.NullString  `gorm:"type:varchar(100)" json:"payment_method_name,omitempty"`
	PaymentTime          sql.NullTime    `json:"payment_time,omitempty"`
	PaymentTransactionID sql.NullString  `gorm:"type:varchar(100)" json:"payment_transaction_id,omitempty"`
	FulfilledAt          sql.NullTime    `gorm:"index" json:"fulfilled_at,omitempty"`
	ExpireTime           sql.NullTime    `json:"expire_time,omitempty"`
	CouponID             sql.NullInt64   `gorm:"index" json:"coupon_id,omitempty"`
	DiscountAmount       sql.NullFloat64 `gorm:"type:decimal(10,2);default:0" json:"discount_amount,omitempty"`
	AmountDueOnline      sql.NullFloat64 `gorm:"column:final_amount;type:decimal(10,2)" json:"final_amount,omitempty"`
	ExtraData            sql.NullString  `gorm:"type:text" json:"extra_data,omitempty"`
	CreatedAt            time.Time       `gorm:"autoCreateTime;index;index:idx_orders_status_created_at,priority:2" json:"created_at"`
	UpdatedAt            time.Time       `gorm:"autoUpdateTime" json:"updated_at"`

	User    User    `gorm:"foreignKey:UserID" json:"-"`
	Package Package `gorm:"foreignKey:PackageID" json:"-"`
	Coupon  Coupon  `gorm:"foreignKey:CouponID" json:"-"`
}

func (Order) TableName() string {
	return "orders"
}

// ===== 订单金额口径（务必区分，历史 bug 就出在这里）=====
//
// 字段语义：
//   Amount         —— 订单原价（套餐价 × 时长，未扣任何优惠）
//   DiscountAmount —— 优惠合计（等级折扣 + 优惠券 + 营销活动 / 后台自定义优惠）
//   AmountDueOnline    —— **还需在线支付的金额**，不是订单金额！
//                     （创建订单时若用余额抵扣，余额已从这里扣掉，余额支付订单恒为 0）
//   ExtraData.balance_used / balance_deducted —— 余额抵扣信息，且有两种语义：
//                     balance_deducted=true  → 创建订单时已抵扣（AmountDueOnline 已减少）
//                     未标记 / false          → 支付时抵扣（AmountDueOnline 未减少）
//
// 结论：
//   - 支付/网关场景（下单应付额、回调金额校验、向支付渠道退款）→ 必须用 AmountDueOnline
//   - 展示订单金额、统计用户消费、审计日志等"订单价值"场景 → 必须用 PaidAmount()
//
// 曾出现的问题：余额支付订单 AmountDueOnline=0，展示层直接读 AmountDueOnline，
// 导致「最近订单金额 ¥0」「审计日志 订单金额 ¥0.00」「用户消费统计少计」。

// BalanceUsed 返回订单记录在案的余额抵扣总额（无则 0）。
// 注意：包含"创建时抵扣"与"支付时抵扣"两种来源，展示金额时不要直接相加，
// 否则支付时抵扣的订单会与 AmountDueOnline 重复计数。
func (o Order) BalanceUsed() float64 {
	if !o.ExtraData.Valid || o.ExtraData.String == "" {
		return 0
	}
	var extra map[string]interface{}
	if err := json.Unmarshal([]byte(o.ExtraData.String), &extra); err != nil {
		return 0
	}
	if v, ok := extra["balance_used"].(float64); ok && v > 0 {
		return v
	}
	return 0
}

// BalanceDeductedAtCreation 余额是否在创建订单时就已抵扣（此时 AmountDueOnline 已减少）
func (o Order) BalanceDeductedAtCreation() bool {
	if !o.ExtraData.Valid || o.ExtraData.String == "" {
		return false
	}
	var extra map[string]interface{}
	if err := json.Unmarshal([]byte(o.ExtraData.String), &extra); err != nil {
		return false
	}
	v, ok := extra["balance_deducted"].(bool)
	return ok && v
}

// OrderValue 返回订单折后价（原价 - 优惠），即客户为该订单付出的总金额，
// 与资金来源（余额 / 在线支付）无关。
func (o Order) OrderValue() float64 {
	value := o.Amount
	if o.DiscountAmount.Valid && o.DiscountAmount.Float64 > 0 {
		value = o.Amount - o.DiscountAmount.Float64
	}
	if value < 0 {
		value = 0
	}
	return math.Round(value*100) / 100
}

// PaidAmount 返回订单成交金额（展示/统计用），等价于订单折后价。
//
// 为什么不用 AmountDueOnline + balance_used：
//   - balance_used 有两种语义（见上），一律相加会让"支付时抵扣"的订单金额翻倍；
//   - 折后价本身已包含所有优惠，直接用它既准确又无需解析 ExtraData。
func (o Order) PaidAmount() float64 {
	return o.OrderValue()
}

// AmountStillDueOnline 返回还需通过在线支付渠道支付的金额（余额抵扣后）。
// 仅用于支付/对账场景，不要用于展示订单金额。
func (o Order) AmountStillDueOnline() float64 {
	if o.AmountDueOnline.Valid {
		if o.AmountDueOnline.Float64 < 0 {
			return 0
		}
		return math.Round(o.AmountDueOnline.Float64*100) / 100
	}
	return o.OrderValue()
}

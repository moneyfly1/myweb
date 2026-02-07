package models

import (
	"database/sql"
	"time"
)

// RegistrationLog 注册日志
type RegistrationLog struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	UserID         uint           `gorm:"index;not null" json:"user_id"`
	Username       string         `gorm:"type:varchar(50);not null" json:"username"`
	Email          string         `gorm:"type:varchar(100);not null" json:"email"`
	IPAddress      sql.NullString `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent      sql.NullString `gorm:"type:text" json:"user_agent,omitempty"`
	Location       sql.NullString `gorm:"type:varchar(255)" json:"location,omitempty"`
	RegisterSource sql.NullString `gorm:"type:varchar(50)" json:"register_source,omitempty"` // invite_code, direct, etc.
	InviteCode     sql.NullString `gorm:"type:varchar(20)" json:"invite_code,omitempty"`
	InviterID      sql.NullInt64  `gorm:"index" json:"inviter_id,omitempty"`
	Status         string         `gorm:"type:varchar(20);default:success" json:"status"` // success, failed
	FailureReason  sql.NullString `gorm:"type:text" json:"failure_reason,omitempty"`
	CreatedAt      time.Time      `gorm:"autoCreateTime;index" json:"created_at"`

	User    User  `gorm:"foreignKey:UserID" json:"-"`
	Inviter *User `gorm:"foreignKey:InviterID" json:"-"`
}

func (RegistrationLog) TableName() string {
	return "registration_logs"
}

// SubscriptionLog 订阅日志
type SubscriptionLog struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	SubscriptionID uint           `gorm:"index;not null" json:"subscription_id"`
	UserID         uint           `gorm:"index;not null" json:"user_id"`
	ActionType     string         `gorm:"type:varchar(50);not null;index" json:"action_type"` // create, update, delete, activate, deactivate, reset
	ActionBy       sql.NullString `gorm:"type:varchar(50)" json:"action_by,omitempty"`        // user, admin, system
	ActionByUserID sql.NullInt64  `gorm:"index" json:"action_by_user_id,omitempty"`
	BeforeData     sql.NullString `gorm:"type:json" json:"before_data,omitempty"`
	AfterData      sql.NullString `gorm:"type:json" json:"after_data,omitempty"`
	Description    sql.NullString `gorm:"type:text" json:"description,omitempty"`
	IPAddress      sql.NullString `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	Location       sql.NullString `gorm:"type:varchar(255)" json:"location,omitempty"` // 地理位置信息（JSON格式）
	CreatedAt      time.Time      `gorm:"autoCreateTime;index" json:"created_at"`

	Subscription Subscription `gorm:"foreignKey:SubscriptionID" json:"-"`
	User         User         `gorm:"foreignKey:UserID" json:"-"`
	ActionByUser *User        `gorm:"foreignKey:ActionByUserID" json:"-"`
}

func (SubscriptionLog) TableName() string {
	return "subscription_logs"
}

// BalanceLog 余额日志
type BalanceLog struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	UserID          uint           `gorm:"index;not null" json:"user_id"`
	ChangeType      string         `gorm:"type:varchar(50);not null;index" json:"change_type"` // recharge, consume, refund, commission, gift, admin_adjust
	Amount          float64        `gorm:"type:decimal(10,2);not null" json:"amount"`          // 变更金额（正数为增加，负数为减少）
	BalanceBefore   float64        `gorm:"type:decimal(10,2);not null" json:"balance_before"`  // 变更前余额
	BalanceAfter    float64        `gorm:"type:decimal(10,2);not null" json:"balance_after"`   // 变更后余额
	RelatedOrderID  sql.NullInt64  `gorm:"index" json:"related_order_id,omitempty"`
	RelatedRecordID sql.NullInt64  `gorm:"index" json:"related_record_id,omitempty"` // 关联的充值记录ID或订单ID
	Description     sql.NullString `gorm:"type:text" json:"description,omitempty"`
	Operator        sql.NullString `gorm:"type:varchar(50)" json:"operator,omitempty"` // 操作人（admin username 或 system）
	OperatorUserID  sql.NullInt64  `gorm:"index" json:"operator_user_id,omitempty"`
	IPAddress       sql.NullString `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	Location        sql.NullString `gorm:"type:varchar(255)" json:"location,omitempty"` // 地理位置信息（JSON格式）
	CreatedAt       time.Time      `gorm:"autoCreateTime;index" json:"created_at"`

	User         User   `gorm:"foreignKey:UserID" json:"-"`
	RelatedOrder *Order `gorm:"foreignKey:RelatedOrderID" json:"-"`
	OperatorUser *User  `gorm:"foreignKey:OperatorUserID" json:"-"`
}

func (BalanceLog) TableName() string {
	return "balance_logs"
}

// CommissionLog 佣金日志
type CommissionLog struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	InviterID        uint           `gorm:"index;not null" json:"inviter_id"`                       // 邀请人ID
	InviteeID        uint           `gorm:"index;not null" json:"invitee_id"`                       // 被邀请人ID
	InviteRelationID sql.NullInt64  `gorm:"index" json:"invite_relation_id,omitempty"`              // 邀请关系ID
	CommissionType   string         `gorm:"type:varchar(50);not null;index" json:"commission_type"` // register_reward, order_commission, etc.
	Amount           float64        `gorm:"type:decimal(10,2);not null" json:"amount"`              // 佣金金额
	RelatedOrderID   sql.NullInt64  `gorm:"index" json:"related_order_id,omitempty"`                // 关联订单ID
	Status           string         `gorm:"type:varchar(20);default:pending;index" json:"status"`   // pending, paid, cancelled
	SettledAt        sql.NullTime   `json:"settled_at,omitempty"`                                   // 结算时间
	Description      sql.NullString `gorm:"type:text" json:"description,omitempty"`
	CreatedAt        time.Time      `gorm:"autoCreateTime;index" json:"created_at"`

	Inviter      User   `gorm:"foreignKey:InviterID" json:"-"`
	Invitee      User   `gorm:"foreignKey:InviteeID" json:"-"`
	RelatedOrder *Order `gorm:"foreignKey:RelatedOrderID" json:"-"`
}

func (CommissionLog) TableName() string {
	return "commission_logs"
}

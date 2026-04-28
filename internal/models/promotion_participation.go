package models

import (
	"database/sql"
	"time"
)

// PromotionParticipation 营销活动参与记录
type PromotionParticipation struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	PromotionID uint           `gorm:"not null;index" json:"promotion_id"`
	UserID      uint           `gorm:"not null;index" json:"user_id"`
	OrderID     sql.NullInt64  `gorm:"index" json:"order_id,omitempty"`
	RewardType  string         `gorm:"not null" json:"reward_type"` // discount, free_days, balance
	RewardValue float64        `gorm:"not null" json:"reward_value"`
	Status      string         `gorm:"not null;default:pending" json:"status"` // pending, completed, expired
	AppliedAt   sql.NullTime   `json:"applied_at,omitempty"`
	ExpireAt    sql.NullTime   `json:"expire_at,omitempty"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	Promotion Promotion `gorm:"foreignKey:PromotionID" json:"promotion,omitempty"`
	User      User      `gorm:"foreignKey:UserID" json:"-"`
	Order     Order     `gorm:"foreignKey:OrderID" json:"-"`
}

func (PromotionParticipation) TableName() string {
	return "promotion_participations"
}

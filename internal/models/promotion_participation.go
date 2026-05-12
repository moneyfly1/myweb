package models

import (
	"database/sql"
	"time"
)

// PromotionParticipation 营销活动参与记录
type PromotionParticipation struct {
	ID          uint          `gorm:"primaryKey" json:"id"`
	PromotionID uint          `gorm:"not null;index;index:idx_promotion_participation_unique,priority:1" json:"promotion_id"`
	UserID      uint          `gorm:"not null;index;index:idx_promotion_participation_lookup,priority:1;index:idx_promotion_participation_unique,priority:2" json:"user_id"`
	OrderID     sql.NullInt64 `gorm:"index" json:"order_id,omitempty"`
	RewardType  string        `gorm:"type:varchar(50);not null;index:idx_promotion_participation_lookup,priority:3" json:"reward_type"` // discount, free_days, balance
	RewardValue float64       `gorm:"type:decimal(10,2);not null" json:"reward_value"`                                                  // reward amount or percentage
	Status      string        `gorm:"type:varchar(20);not null;default:pending;index;index:idx_promotion_participation_lookup,priority:2" json:"status"`
	AppliedAt   sql.NullTime  `json:"applied_at,omitempty"`
	ExpireAt    sql.NullTime  `gorm:"index;index:idx_promotion_participation_lookup,priority:4" json:"expire_at,omitempty"`
	CreatedAt   time.Time     `gorm:"autoCreateTime;not null;default:CURRENT_TIMESTAMP;index" json:"created_at"`
	UpdatedAt   time.Time     `gorm:"autoUpdateTime;not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	Promotion Promotion `gorm:"foreignKey:PromotionID;constraint:OnDelete:CASCADE" json:"promotion,omitempty"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Order     Order     `gorm:"foreignKey:OrderID;constraint:OnDelete:SET NULL" json:"-"`
}

func (PromotionParticipation) TableName() string {
	return "promotion_participations"
}

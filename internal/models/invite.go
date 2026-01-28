package models

import (
	"database/sql"
	"time"
)

type InviteCode struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Code           string         `gorm:"type:varchar(20);uniqueIndex;not null" json:"code"`
	UserID         uint           `gorm:"index;not null" json:"user_id"`
	UsedCount      int            `gorm:"default:0" json:"used_count"`
	MaxUses        sql.NullInt64  `json:"max_uses,omitempty"`
	ExpiresAt      sql.NullTime   `json:"expires_at,omitempty"`
	RewardType     string         `gorm:"type:varchar(20);default:balance" json:"reward_type"`
	InviterReward  float64        `gorm:"type:decimal(10,2);default:0" json:"inviter_reward"`
	InviteeReward  float64        `gorm:"type:decimal(10,2);default:0" json:"invitee_reward"`
	PackageIDs     sql.NullString `gorm:"type:text" json:"package_ids,omitempty"`
	MinOrderAmount float64        `gorm:"type:decimal(10,2);default:0" json:"min_order_amount"`
	NewUserOnly    bool           `gorm:"default:true" json:"new_user_only"`
	IsActive       bool           `gorm:"default:true" json:"is_active"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	User            User             `gorm:"foreignKey:UserID" json:"-"`
	InviteRelations []InviteRelation `gorm:"foreignKey:InviteCodeID" json:"-"`
}

func (InviteCode) TableName() string {
	return "invite_codes"
}

type InviteRelation struct {
	ID                      uint          `gorm:"primaryKey" json:"id"`
	InviteCodeID            uint          `gorm:"index;not null" json:"invite_code_id"`
	InviterID               uint          `gorm:"index;not null" json:"inviter_id"`
	InviteeID               uint          `gorm:"index;not null" json:"invitee_id"`
	InviterRewardGiven      bool          `gorm:"default:false" json:"inviter_reward_given"`
	InviteeRewardGiven      bool          `gorm:"default:false" json:"invitee_reward_given"`
	InviterRewardAmount     float64       `gorm:"type:decimal(10,2);default:0" json:"inviter_reward_amount"`
	InviteeRewardAmount     float64       `gorm:"type:decimal(10,2);default:0" json:"invitee_reward_amount"`
	InviteeFirstOrderID     sql.NullInt64 `gorm:"index" json:"invitee_first_order_id,omitempty"`
	InviteeTotalConsumption float64       `gorm:"type:decimal(10,2);default:0" json:"invitee_total_consumption"`
	CreatedAt               time.Time     `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt               time.Time     `gorm:"autoUpdateTime" json:"updated_at"`

	InviteCode InviteCode `gorm:"foreignKey:InviteCodeID" json:"-"`
	Inviter    User       `gorm:"foreignKey:InviterID" json:"-"`
	Invitee    User       `gorm:"foreignKey:InviteeID" json:"-"`
	FirstOrder Order      `gorm:"foreignKey:InviteeFirstOrderID" json:"-"`
}

func (InviteRelation) TableName() string {
	return "invite_relations"
}

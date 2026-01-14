package models

import (
	"database/sql"
	"time"
)

// User 用户模型
type User struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	Username   string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	Email      string         `gorm:"type:varchar(100);uniqueIndex;not null" json:"email"`
	Password   string         `gorm:"type:varchar(255);not null" json:"-"`
	IsActive   bool           `gorm:"default:true" json:"is_active"`
	IsVerified bool           `gorm:"default:false" json:"is_verified"`
	IsAdmin    bool           `gorm:"default:false" json:"is_admin"`
	Nickname   sql.NullString `gorm:"type:varchar(50)" json:"nickname,omitempty"`
	Avatar     sql.NullString `gorm:"type:varchar(255)" json:"avatar,omitempty"`
	CreatedAt  time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	LastLogin  sql.NullTime   `json:"last_login,omitempty"`

	VerificationToken   sql.NullString `gorm:"type:varchar(255)" json:"-"`
	VerificationExpires sql.NullTime   `json:"-"`
	ResetToken          sql.NullString `gorm:"type:varchar(255)" json:"-"`
	ResetExpires        sql.NullTime   `json:"-"`

	Theme    string `gorm:"type:varchar(20);default:light" json:"theme"`
	Language string `gorm:"type:varchar(10);default:zh-CN" json:"language"`
	Timezone string `gorm:"type:varchar(50);default:Asia/Shanghai" json:"timezone"`

	EmailNotifications bool   `gorm:"default:true" json:"email_notifications"`
	NotificationTypes  string `gorm:"type:text" json:"notification_types"`
	SMSNotifications   bool   `gorm:"default:false" json:"sms_notifications"`
	PushNotifications  bool   `gorm:"default:true" json:"push_notifications"`

	DataSharing bool `gorm:"default:true" json:"data_sharing"`
	Analytics   bool `gorm:"default:true" json:"analytics"`

	Balance float64 `gorm:"type:decimal(10,2);default:0;not null" json:"balance"`

	InvitedBy         sql.NullInt64  `gorm:"index" json:"invited_by,omitempty"`
	InviteCodeUsed    sql.NullString `gorm:"type:varchar(20)" json:"invite_code_used,omitempty"`
	TotalInviteCount  int            `gorm:"default:0" json:"total_invite_count"`
	TotalInviteReward float64        `gorm:"type:decimal(10,2);default:0" json:"total_invite_reward"`

	UserLevelID      sql.NullInt64 `gorm:"index" json:"user_level_id,omitempty"`
	TotalConsumption float64       `gorm:"type:decimal(10,2);default:0;not null" json:"total_consumption"`
	LevelExpiresAt   sql.NullTime  `json:"level_expires_at,omitempty"`

	// 专线节点相关配置
	SpecialNodeSubscriptionType string       `gorm:"type:varchar(20);default:both" json:"special_node_subscription_type"` // both, special_only
	SpecialNodeExpiresAt        sql.NullTime `json:"special_node_expires_at,omitempty"`

	// 关系
	Subscriptions            []Subscription       `gorm:"foreignKey:UserID" json:"-"`
	Orders                   []Order              `gorm:"foreignKey:UserID" json:"-"`
	Devices                  []Device             `gorm:"foreignKey:UserID" json:"-"`
	Notifications            []Notification       `gorm:"foreignKey:UserID" json:"-"`
	Payments                 []PaymentTransaction `gorm:"foreignKey:UserID" json:"-"`
	RechargeRecords          []RechargeRecord     `gorm:"foreignKey:UserID" json:"-"`
	Activities               []UserActivity       `gorm:"foreignKey:UserID" json:"-"`
	SubscriptionResets       []SubscriptionReset  `gorm:"foreignKey:UserID" json:"-"`
	LoginHistory             []LoginHistory       `gorm:"foreignKey:UserID" json:"-"`
	Tickets                  []Ticket             `gorm:"foreignKey:UserID" json:"-"`
	InviteCodes              []InviteCode         `gorm:"foreignKey:UserID" json:"-"`
	InviteRelationsAsInviter []InviteRelation     `gorm:"foreignKey:InviterID" json:"-"`
	InviteRelationsAsInvitee []InviteRelation     `gorm:"foreignKey:InviteeID" json:"-"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

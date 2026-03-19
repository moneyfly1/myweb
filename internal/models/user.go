package models

import (
	"database/sql"
	"time"
)

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

	EmailNotifications       bool   `gorm:"default:true" json:"email_notifications"`
	AbnormalLoginAlertEnabled bool   `gorm:"default:false" json:"abnormal_login_alert_enabled"` // 是否接收异常登录/设备告警（邮件+站内通知），默认关闭，用户自行开启
	NotificationTypes        string `gorm:"type:text" json:"notification_types"`
	SMSNotifications         bool   `gorm:"default:false" json:"sms_notifications"`
	PushNotifications        bool   `gorm:"default:true" json:"push_notifications"`

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

	SpecialNodeSubscriptionType string       `gorm:"type:varchar(20);default:both" json:"special_node_subscription_type"` // both, special_only
	SpecialNodeExpiresAt        sql.NullTime `json:"special_node_expires_at,omitempty"`

	TelegramID       sql.NullInt64  `gorm:"index" json:"telegram_id,omitempty"`
	TelegramUsername sql.NullString `gorm:"type:varchar(100)" json:"telegram_username,omitempty"`

	Notes sql.NullString `gorm:"type:text" json:"notes,omitempty"` // 备注字段

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

func (User) TableName() string {
	return "users"
}

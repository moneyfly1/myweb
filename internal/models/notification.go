package models

import (
	"database/sql"
	"time"
)

type Notification struct {
	ID        uint          `gorm:"primaryKey" json:"id"`
	UserID    sql.NullInt64 `gorm:"index" json:"user_id,omitempty"`
	Title     string        `gorm:"type:varchar(255);not null" json:"title"`
	Content   string        `gorm:"type:text;not null" json:"content"`
	Type      string        `gorm:"type:varchar(50);default:system" json:"type"`
	IsRead    bool          `gorm:"default:false" json:"is_read"`
	IsActive  bool          `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time     `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time     `gorm:"autoUpdateTime" json:"updated_at"`
	ReadAt    sql.NullTime  `json:"read_at,omitempty"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (Notification) TableName() string {
	return "notifications"
}

type EmailTemplate struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Subject   string    `gorm:"type:varchar(200);not null" json:"subject"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	Variables string    `gorm:"type:text" json:"variables"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (EmailTemplate) TableName() string {
	return "email_templates"
}

type EmailQueue struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	ToEmail      string         `gorm:"type:varchar(100);not null" json:"to_email"`
	Subject      string         `gorm:"type:varchar(200);not null" json:"subject"`
	Content      string         `gorm:"type:text;not null" json:"content"`
	ContentType  string         `gorm:"type:varchar(20);default:plain" json:"content_type"`
	EmailType    string         `gorm:"type:varchar(50)" json:"email_type"`
	Attachments  string         `gorm:"type:text" json:"attachments"`
	Status       string         `gorm:"type:varchar(20);default:pending" json:"status"`
	RetryCount   int            `gorm:"default:0" json:"retry_count"`
	MaxRetries   int            `gorm:"default:3" json:"max_retries"`
	SentAt       sql.NullTime   `json:"sent_at,omitempty"`
	ErrorMessage sql.NullString `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (EmailQueue) TableName() string {
	return "email_queue"
}

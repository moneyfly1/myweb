package models

import "time"

type TicketStatus string

const (
	TicketStatusPending    TicketStatus = "pending"
	TicketStatusProcessing TicketStatus = "processing"
	TicketStatusResolved   TicketStatus = "resolved"
	TicketStatusClosed     TicketStatus = "closed"
	TicketStatusCancelled  TicketStatus = "cancelled"
)

type TicketType string

const (
	TicketTypeTechnical TicketType = "technical"
	TicketTypeBilling   TicketType = "billing"
	TicketTypeAccount   TicketType = "account"
	TicketTypeOther     TicketType = "other"
)

type TicketPriority string

const (
	TicketPriorityLow    TicketPriority = "low"
	TicketPriorityNormal TicketPriority = "normal"
	TicketPriorityHigh   TicketPriority = "high"
	TicketPriorityUrgent TicketPriority = "urgent"
)

type Ticket struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	TicketNo      string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"ticket_no"`
	UserID        uint       `gorm:"index;not null" json:"user_id"`
	Title         string     `gorm:"type:varchar(200);not null" json:"title"`
	Content       string     `gorm:"type:text;not null" json:"content"`
	Type          string     `gorm:"type:varchar(20);default:other" json:"type"`
	Status        string     `gorm:"type:varchar(20);default:pending" json:"status"`
	Priority      string     `gorm:"type:varchar(20);default:normal" json:"priority"`
	AssignedTo    *int64     `gorm:"index" json:"assigned_to,omitempty"`
	AdminNotes    *string    `gorm:"type:text" json:"admin_notes,omitempty"`
	Rating        *int64     `json:"rating,omitempty"`
	RatingComment *string    `gorm:"type:text" json:"rating_comment,omitempty"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	ResolvedAt    *time.Time `json:"resolved_at,omitempty"`
	ClosedAt      *time.Time `json:"closed_at,omitempty"`

	User        User               `gorm:"foreignKey:UserID" json:"-"`
	Assignee    User               `gorm:"foreignKey:AssignedTo" json:"-"`
	Replies     []TicketReply      `gorm:"foreignKey:TicketID" json:"-"`
	Attachments []TicketAttachment `gorm:"foreignKey:TicketID" json:"-"`
}

func (Ticket) TableName() string {
	return "tickets"
}

type TicketReply struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	TicketID  uint       `gorm:"index;not null" json:"ticket_id"`
	UserID    uint       `gorm:"index;not null" json:"user_id"`
	Content   string     `gorm:"type:text;not null" json:"content"`
	IsAdmin   string     `gorm:"type:varchar(10);default:false" json:"is_admin"`
	IsRead    bool       `gorm:"default:false" json:"is_read"`   // 是否已读（对应用户或管理员）
	ReadBy    *uint      `gorm:"index" json:"read_by,omitempty"` // 被谁已读（用户ID或管理员ID）
	ReadAt    *time.Time `json:"read_at,omitempty"`              // 已读时间
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`

	Ticket Ticket `gorm:"foreignKey:TicketID" json:"-"`
	User   User   `gorm:"foreignKey:UserID" json:"-"`
}

func (TicketReply) TableName() string {
	return "ticket_replies"
}

type TicketAttachment struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TicketID   uint      `gorm:"index;not null" json:"ticket_id"`
	ReplyID    *int64    `gorm:"index" json:"reply_id,omitempty"`
	FileName   string    `gorm:"type:varchar(255);not null" json:"file_name"`
	FilePath   string    `gorm:"type:varchar(500);not null" json:"file_path"`
	FileSize   *int64    `json:"file_size,omitempty"`
	FileType   *string   `gorm:"type:varchar(50)" json:"file_type,omitempty"`
	UploadedBy uint      `gorm:"index;not null" json:"uploaded_by"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`

	Ticket   Ticket      `gorm:"foreignKey:TicketID" json:"-"`
	Reply    TicketReply `gorm:"foreignKey:ReplyID" json:"-"`
	Uploader User        `gorm:"foreignKey:UploadedBy" json:"-"`
}

func (TicketAttachment) TableName() string {
	return "ticket_attachments"
}

type TicketRead struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TicketID  uint      `gorm:"index;not null" json:"ticket_id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	ReadAt    time.Time `gorm:"autoCreateTime;autoUpdateTime" json:"read_at"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Ticket Ticket `gorm:"foreignKey:TicketID" json:"-"`
	User   User   `gorm:"foreignKey:UserID" json:"-"`
}

func (TicketRead) TableName() string {
	return "ticket_reads"
}

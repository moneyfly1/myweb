package models

import (
	"database/sql"
	"time"
)

type AuditLog struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	UserID            sql.NullInt64  `gorm:"index" json:"user_id,omitempty"`
	ActionType        string         `gorm:"type:varchar(50);index;not null" json:"action_type"`
	ResourceType      sql.NullString `gorm:"type:varchar(50);index" json:"resource_type,omitempty"`
	ResourceID        sql.NullInt64  `gorm:"index" json:"resource_id,omitempty"`
	ActionDescription sql.NullString `gorm:"type:text" json:"action_description,omitempty"`
	IPAddress         sql.NullString `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent         sql.NullString `gorm:"type:text" json:"user_agent,omitempty"`
	Location          sql.NullString `gorm:"type:varchar(255)" json:"location,omitempty"`
	RequestMethod     sql.NullString `gorm:"type:varchar(10)" json:"request_method,omitempty"`
	RequestPath       sql.NullString `gorm:"type:varchar(255)" json:"request_path,omitempty"`
	RequestParams     sql.NullString `gorm:"type:json" json:"request_params,omitempty"`
	ResponseStatus    sql.NullInt64  `json:"response_status,omitempty"`
	BeforeData        sql.NullString `gorm:"type:json" json:"before_data,omitempty"`
	AfterData         sql.NullString `gorm:"type:json" json:"after_data,omitempty"`
	CreatedAt         time.Time      `gorm:"autoCreateTime;index" json:"created_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

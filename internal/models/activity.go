package models

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

type UserActivity struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	UserID           uint           `gorm:"index;not null" json:"user_id"`
	ActivityType     string         `gorm:"type:varchar(50);not null" json:"activity_type"`
	Description      sql.NullString `gorm:"type:text" json:"description,omitempty"`
	IPAddress        sql.NullString `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent        sql.NullString `gorm:"type:text" json:"user_agent,omitempty"`
	Location         sql.NullString `gorm:"type:varchar(100)" json:"location,omitempty"`
	ActivityMetadata sql.NullString `gorm:"type:json" json:"activity_metadata,omitempty"`
	CreatedAt        time.Time      `gorm:"autoCreateTime" json:"created_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (UserActivity) TableName() string {
	return "user_activities"
}

type LoginHistory struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	UserID            uint           `gorm:"index;not null" json:"user_id"`
	LoginTime         time.Time      `gorm:"autoCreateTime" json:"login_time"`
	LogoutTime        sql.NullTime   `json:"logout_time,omitempty"`
	IPAddress         sql.NullString `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent         sql.NullString `gorm:"type:text" json:"user_agent,omitempty"`
	Location          sql.NullString `gorm:"type:varchar(100)" json:"location,omitempty"`
	DeviceFingerprint sql.NullString `gorm:"type:varchar(255)" json:"device_fingerprint,omitempty"`
	LoginStatus       string         `gorm:"type:varchar(20);default:success" json:"login_status"`
	FailureReason     sql.NullString `gorm:"type:text" json:"failure_reason,omitempty"`
	SessionDuration   sql.NullInt64  `json:"session_duration,omitempty"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (LoginHistory) TableName() string {
	return "login_history"
}

func (h *LoginHistory) GetLocationInfo() (country, city string) {
	if !h.Location.Valid || h.Location.String == "" {
		return "", ""
	}
	locationStr := h.Location.String
	if strings.Contains(locationStr, ",") {
		parts := strings.Split(locationStr, ",")
		if len(parts) >= 1 {
			country = strings.TrimSpace(parts[0])
		}
		if len(parts) >= 2 {
			city = strings.TrimSpace(parts[1])
		}
	} else {
		var locationData map[string]interface{}
		if err := json.Unmarshal([]byte(locationStr), &locationData); err == nil {
			if c, ok := locationData["country"].(string); ok {
				country = c
			}
			if c, ok := locationData["city"].(string); ok {
				city = c
			}
		}
	}
	return
}

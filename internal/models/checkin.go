package models

import "time"

type CheckinRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Amount    float64   `gorm:"not null" json:"amount"`
	CreatedAt time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

func (CheckinRecord) TableName() string {
	return "checkin_records"
}

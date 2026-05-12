package models

import "time"

type CheckinRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;index:idx_checkin_user_created_at;not null" json:"user_id"`
	Amount    float64   `gorm:"not null" json:"amount"`
	CreatedAt time.Time `gorm:"autoCreateTime;index;index:idx_checkin_user_created_at" json:"created_at"`
}

func (CheckinRecord) TableName() string {
	return "checkin_records"
}

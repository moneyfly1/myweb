package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

type UserLevel struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	LevelName      string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"level_name"`
	LevelOrder     int            `gorm:"uniqueIndex;not null" json:"level_order"`
	MinConsumption float64        `gorm:"type:decimal(10,2);default:0" json:"min_consumption"`
	DiscountRate   float64        `gorm:"type:decimal(5,2);default:1.0" json:"discount_rate"`
	DeviceLimit    int            `gorm:"default:3" json:"device_limit"`
	Benefits       sql.NullString `gorm:"type:text" json:"benefits,omitempty"`
	IconURL        sql.NullString `gorm:"type:varchar(255)" json:"icon_url,omitempty"`
	Color          string         `gorm:"type:varchar(20);default:#909399" json:"color"`
	IsActive       bool           `gorm:"default:true" json:"is_active"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	Users []User `gorm:"foreignKey:UserLevelID" json:"-"`
}

func (UserLevel) TableName() string {
	return "user_levels"
}

type UserLevelResponse struct {
	ID             uint      `json:"id"`
	LevelName      string    `json:"level_name"`
	LevelOrder     int       `json:"level_order"`
	MinConsumption float64   `json:"min_consumption"`
	DiscountRate   float64   `json:"discount_rate"`
	DeviceLimit    int       `json:"device_limit"`
	Benefits       *string   `json:"benefits,omitempty"`
	IconURL        *string   `json:"icon_url,omitempty"`
	Color          string    `json:"color"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (ul *UserLevel) ToUserLevelResponse() UserLevelResponse {
	resp := UserLevelResponse{
		ID:             ul.ID,
		LevelName:      ul.LevelName,
		LevelOrder:     ul.LevelOrder,
		MinConsumption: ul.MinConsumption,
		DiscountRate:   ul.DiscountRate,
		DeviceLimit:    ul.DeviceLimit,
		Color:          ul.Color,
		IsActive:       ul.IsActive,
		CreatedAt:      ul.CreatedAt,
		UpdatedAt:      ul.UpdatedAt,
	}

	if ul.Benefits.Valid {
		resp.Benefits = &ul.Benefits.String
	}
	if ul.IconURL.Valid {
		resp.IconURL = &ul.IconURL.String
	}

	return resp
}

func (ul *UserLevel) MarshalJSON() ([]byte, error) {
	return json.Marshal(ul.ToUserLevelResponse())
}

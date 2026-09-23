package models

import (
	"time"
)

type KnowledgeCategory struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Icon      string    `json:"icon"`
	SortOrder int       `json:"sort_order"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (KnowledgeCategory) TableName() string {
	return "knowledge_categories"
}

type KnowledgeArticle struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	CategoryID uint       `gorm:"index;not null" json:"category_id"`
	Title      string     `gorm:"not null" json:"title"`
	Content    string     `gorm:"not null" json:"content"`
	Summary    JSONString `json:"summary"` // 可空但 JSON 必须是普通字符串（见 json_string.go）
	ViewCount  int        `json:"view_count"`
	SortOrder  int        `json:"sort_order"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	Category KnowledgeCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
}

func (KnowledgeArticle) TableName() string {
	return "knowledge_articles"
}

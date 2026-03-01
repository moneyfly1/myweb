package models

import (
	"database/sql"
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
	ID         uint           `gorm:"primaryKey" json:"id"`
	CategoryID uint           `gorm:"index;not null" json:"category_id"`
	Title      string         `gorm:"not null" json:"title"`
	Content    string         `gorm:"not null" json:"content"`
	Summary    sql.NullString `json:"summary"`
	ViewCount  int            `json:"view_count"`
	SortOrder  int            `json:"sort_order"`
	IsActive   bool           `json:"is_active"`
	CreatedAt  time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	Category KnowledgeCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
}

func (KnowledgeArticle) TableName() string {
	return "knowledge_articles"
}

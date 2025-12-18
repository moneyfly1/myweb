package models

import (
	"time"
)

// Node 节点模型
type Node struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	Name          string     `gorm:"type:varchar(100);not null" json:"name"`
	Region        string     `gorm:"type:varchar(50);not null" json:"region"`
	Type          string     `gorm:"type:varchar(20);not null" json:"type"`
	Status        string     `gorm:"type:varchar(20);default:offline" json:"status"`
	Load          float64    `gorm:"default:0.0" json:"load"`
	Speed         float64    `gorm:"default:0.0" json:"speed"`
	Uptime        int        `gorm:"default:0" json:"uptime"`
	Latency       int        `gorm:"default:0" json:"latency"`
	Description   *string    `gorm:"type:text" json:"description,omitempty"`
	Config        *string    `gorm:"type:text" json:"config,omitempty"`
	IsRecommended bool       `gorm:"default:false" json:"is_recommended"`
	IsActive      bool       `gorm:"default:true" json:"is_active"`
	IsManual      bool       `gorm:"default:false" json:"is_manual"` // 是否为手动添加的节点
	LastTest      *time.Time `json:"last_test,omitempty"`
	LastUpdate    time.Time  `gorm:"autoCreateTime;autoUpdateTime" json:"last_update"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (Node) TableName() string {
	return "nodes"
}

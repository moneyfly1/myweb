package models

import (
	"time"
)

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
	IsManual      bool       `gorm:"default:false" json:"is_manual"`     // 是否为手动添加的节点
	SourceIndex   int        `gorm:"default:0" json:"source_index"`      // 来源订阅编号（1开始），0表示手动添加
	OrderIndex    int        `gorm:"default:0;index" json:"order_index"` // 节点顺序索引，用于排序
	LastTest      *time.Time `json:"last_test,omitempty"`
	LastUpdate    time.Time  `gorm:"autoCreateTime;autoUpdateTime" json:"last_update"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// 自建节点（一键部署）字段
	SelfHosted       bool       `gorm:"default:false" json:"self_hosted"`                      // 是否为自建节点
	SelfHostProtocol string     `gorm:"type:varchar(30);default:''" json:"self_host_protocol"` // 自建协议：vless-ws / vmess-ws / vless-reality / trojan-ws / ss
	InstallID        string     `gorm:"type:varchar(64);index" json:"install_id,omitempty"`    // 一次性安装标识
	InstallToken     string     `gorm:"type:varchar(128)" json:"-"`                            // 回传/心跳令牌（不暴露给前端）
	InstallExpiresAt *time.Time `json:"install_expires_at,omitempty"`                          // 安装令牌过期时间
	LastHeartbeatAt  *time.Time `json:"last_heartbeat_at,omitempty"`                           // 最近一次心跳时间
	InstallCmd       string     `gorm:"type:text" json:"-"`                                    // 生成的安装命令（不暴露给前端）
	TrafficUp        int64      `gorm:"default:0" json:"traffic_up"`                           // 自建节点上行流量（字节）
	TrafficDown      int64      `gorm:"default:0" json:"traffic_down"`                         // 自建节点下行流量（字节）
	TrafficUpdatedAt *time.Time `json:"traffic_updated_at,omitempty"`                          // 流量统计时间
}

func (Node) TableName() string {
	return "nodes"
}

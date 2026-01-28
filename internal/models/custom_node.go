package models

import (
	"time"
)

type CustomNode struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	Name             string     `gorm:"type:varchar(100);not null" json:"name"`
	DisplayName      string     `gorm:"type:varchar(100)" json:"display_name"`
	Protocol         string     `gorm:"type:varchar(20);default:''" json:"protocol"`
	Domain           string     `gorm:"type:varchar(255)" json:"domain"`
	Port             int        `gorm:"default:443" json:"port"`
	Config           string     `gorm:"type:text" json:"config"`
	Status           string     `gorm:"type:varchar(20);default:inactive" json:"status"`
	IsActive         bool       `gorm:"default:true" json:"is_active"`
	Latency          int        `gorm:"default:0" json:"latency"` // 延迟（毫秒）
	LastTest         *time.Time `json:"last_test,omitempty"`      // 最后测试时间
	ExpireTime       *time.Time `json:"expire_time,omitempty"`
	FollowUserExpire bool       `gorm:"default:false" json:"follow_user_expire"`
	CreatedAt        time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

type NodeConfig struct {
	Type              string `json:"type"`
	Server            string `json:"server"`
	Port              int    `json:"port"`
	UUID              string `json:"uuid,omitempty"`
	Password          string `json:"password,omitempty"`
	Encryption        string `json:"encryption,omitempty"`
	Network           string `json:"network,omitempty"`
	Security          string `json:"security,omitempty"`
	SNI               string `json:"sni,omitempty"`
	Fingerprint       string `json:"fingerprint,omitempty"`
	Flow              string `json:"flow,omitempty"`
	PublicKey         string `json:"public_key,omitempty"`
	ShortID           string `json:"short_id,omitempty"`
	ALPN              string `json:"alpn,omitempty"`
	Host              string `json:"host,omitempty"`
	Path              string `json:"path,omitempty"`
	ServiceName       string `json:"service_name,omitempty"`
	Padding           bool   `json:"padding,omitempty"`
	CongestionControl string `json:"congestion_control,omitempty"`
	UDPRelayMode      string `json:"udp_relay_mode,omitempty"`
	SkipCertVerify    bool   `json:"skip_cert_verify,omitempty"`
}

func (CustomNode) TableName() string {
	return "custom_nodes"
}

type UserCustomNode struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	UserID       uint       `gorm:"index:idx_user_node;not null" json:"user_id"`
	CustomNodeID uint       `gorm:"index:idx_user_node;not null" json:"custom_node_id"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	User         User       `gorm:"foreignKey:UserID" json:"-"`
	CustomNode   CustomNode `gorm:"foreignKey:CustomNodeID" json:"custom_node,omitempty"`
}

func (UserCustomNode) TableName() string {
	return "user_custom_nodes"
}

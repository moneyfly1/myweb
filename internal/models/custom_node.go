package models

import (
	"time"
)

type CustomNode struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	Name             string     `gorm:"type:varchar(100);not null" json:"name"`
	DisplayName      string     `gorm:"type:varchar(100)" json:"display_name"`
	Protocol         string     `gorm:"type:varchar(20);default:''" json:"protocol"`
	Port             int        `gorm:"default:443" json:"port"`
	Config           string     `gorm:"type:text" json:"config"`
	Status           string     `gorm:"type:varchar(20);default:inactive" json:"status"`
	IsActive         bool       `gorm:"default:true" json:"is_active"`
	Latency          int        `gorm:"default:0" json:"latency"` // 延迟（毫秒）
	LastTest         *time.Time `json:"last_test,omitempty"`      // 最后测试时间
	ExpireTime       *time.Time `json:"expire_time,omitempty"`
	FollowUserExpire bool       `gorm:"default:false" json:"follow_user_expire"`
	Source           string     `gorm:"type:varchar(20);default:''" json:"source"` // 节点来源: manual / link / subscription / selfhost

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

	// VPS 自动搭建（SSH）字段
	SSHHost        string `gorm:"type:varchar(255)" json:"ssh_host,omitempty"`  // VPS IP/域名
	SSHPort        int    `gorm:"default:22" json:"ssh_port,omitempty"`         // SSH 端口
	SSHUser        string `gorm:"type:varchar(64);default:root" json:"ssh_user"` // SSH 用户名
	SSHPasswordEnc string `gorm:"type:text" json:"-"`                           // SSH 密码（AES 加密存储，不暴露给前端）

	// 域名多协议部署（Xray）字段
	CoreType       string `gorm:"type:varchar(20);default:''" json:"core_type"` // 核心类型: xray / sing-box
	Domain         string `gorm:"type:varchar(255)" json:"domain,omitempty"`    // 证书域名（TLS）
	DeployMode     string `gorm:"type:varchar(20);default:''" json:"deploy_mode"` // 部署模式: single(单协议) / multi(多协议)
	ProtocolList   string `gorm:"type:text" json:"-"`                           // 多协议部署时该节点所属协议列表（JSON，不直接暴露）

	// 流量配额
	TrafficLimitEnabled bool  `gorm:"default:false" json:"traffic_limit_enabled"` // 是否启用流量配额
	TrafficLimitBytes   int64 `gorm:"default:0" json:"traffic_limit_bytes"`      // 流量配额（字节）
	TrafficLimitResetAt *time.Time `json:"traffic_limit_reset_at,omitempty"`     // 配额重置时间（周期重置用）

	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
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

	// 分配级流量配额：客户独享节点时，按分配设置配额（不污染节点全局配额）
	TrafficLimitEnabled bool `gorm:"default:false" json:"traffic_limit_enabled"`
	TrafficLimitBytes   int64 `gorm:"default:0" json:"traffic_limit_bytes"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	User      User      `gorm:"foreignKey:UserID" json:"-"`
	CustomNode CustomNode `gorm:"foreignKey:CustomNodeID" json:"custom_node,omitempty"`
}

func (UserCustomNode) TableName() string {
	return "user_custom_nodes"
}

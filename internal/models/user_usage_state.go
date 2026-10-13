package models

import "time"

// UserUsageState 记录「这个用户到底有没有在用」的**最终快照**（一个用户一行）。
//
// 与 models.UserActivity 的区别：UserActivity 是「活动事件日志」（每次操作一行），
// 本表是「每个用户一行的使用状态」，用于后台列表直接展示，避免每次拉订阅都往日志表写一行。
//
// 设计背景：第三方客户端（v2rayN / Clash / Shadowrocket / sing-box…）不会给面板发心跳，
// 协议上也没有这个能力，所以只能用**被动信号**推断，共两级：
//
//	① 订阅拉取（LastSubAt / LastSubUA / LastSubIP / LastSubFormat）
//	   —— 说明客户端还活着、还在按 interval 更新订阅；粒度取决于客户端更新间隔
//	     （面板下发了 Profile-Update-Interval，默认 60 分钟）。
//	② 节点侧按用户上报（OnlineConns / LastNodeReportAt / NodeTraffic*）
//	   —— 说明此刻真的在连节点，是唯一的「秒级在线」信号；需要节点侧能区分用户。
//	     自建节点当前是**单凭据**设计（安装时生成一套 UUID/密码，所有用户共用），
//	     因此该字段在节点升级为多用户前一直是 0；面板侧已就绪，届时无需再改。
//
// 实时新鲜度放在 Redis（见 internal/services/activity），本表只落**节流后的**快照，
// 作为 Redis 重启/换机后的兜底。
type UserUsageState struct {
	UserID uint `gorm:"primaryKey" json:"user_id"`

	// 订阅侧信号
	LastSubAt       *time.Time `json:"last_sub_at,omitempty"`
	LastSubUA       *string    `gorm:"type:varchar(255)" json:"last_sub_ua,omitempty"`
	LastSubIP       *string    `gorm:"type:varchar(45)" json:"last_sub_ip,omitempty"`
	LastSubLocation *string    `gorm:"type:varchar(255)" json:"last_sub_location,omitempty"`
	LastSubFormat   *string    `gorm:"type:varchar(20)" json:"last_sub_format,omitempty"` // clash / universal / xboard
	LastSubSoftware *string    `gorm:"type:varchar(100)" json:"last_sub_software,omitempty"`
	SubFetchCount   int64      `gorm:"default:0" json:"sub_fetch_count"`

	// 节点侧信号（需要节点支持按用户区分）
	OnlineConns      int        `gorm:"default:0" json:"online_conns"`
	LastNodeReportAt *time.Time `json:"last_node_report_at,omitempty"`
	NodeTrafficUp    int64      `gorm:"default:0" json:"node_traffic_up"`
	NodeTrafficDown  int64      `gorm:"default:0" json:"node_traffic_down"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (UserUsageState) TableName() string {
	return "user_usage_states"
}

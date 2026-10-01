package models

import "time"

// 节点校验日志事件类型（两层防御的可观测面）
const (
	// NodeValidationDroppedAtIngest 采集/解析阶段丢弃（静态白名单校验不通过）
	NodeValidationDroppedAtIngest = "dropped_at_ingest"
	// NodeValidationCorrectedAtIngest 采集/解析阶段修正（如 ss 2022 密钥的 URL 编码污染）
	NodeValidationCorrectedAtIngest = "corrected_at_ingest"
	// NodeValidationPrunedAtGenerate 生成配置时被内核自检剔除
	NodeValidationPrunedAtGenerate = "pruned_at_generate"
	// NodeValidationRenamedAtIngest 采集阶段因重名被自动改名（避免内核静默覆盖）
	NodeValidationRenamedAtIngest = "renamed_at_ingest"
)

// NodeValidationLog 节点校验日志
//
// 记录两类事件，供后台展示「已丢弃 / 已剔除」列表：
//   - dropped_at_ingest / corrected_at_ingest / renamed_at_ingest：第一层（静态白名单校验）
//   - pruned_at_generate：第二层（mihomo -t 内核自检 + 二分剔除）
//
// 只记录判定结论，不记录节点凭据（password/uuid 不入库），便于脱敏展示。
type NodeValidationLog struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	Event  string `gorm:"type:varchar(32);not null;index:idx_nvl_event_created,priority:1" json:"event"`
	Source string `gorm:"type:varchar(255);not null;default:''" json:"source"`
	// Format 输出格式（clash-yaml / base64-links / surge / singbox-json / quantumultx / loon）
	Format    string    `gorm:"type:varchar(32);not null;default:'';index" json:"format"`
	NodeName  string    `gorm:"type:varchar(255);not null;default:''" json:"node_name"`
	NodeType  string    `gorm:"type:varchar(32);not null;default:''" json:"node_type"`
	Server    string    `gorm:"type:varchar(255);not null;default:''" json:"server"`
	Port      int       `gorm:"not null;default:0" json:"port"`
	Reason    string    `gorm:"type:text;not null;default:''" json:"reason"`
	CreatedAt time.Time `gorm:"autoCreateTime;index;index:idx_nvl_event_created,priority:2" json:"created_at"`
}

// TableName 指定表名
func (NodeValidationLog) TableName() string {
	return "node_validation_logs"
}

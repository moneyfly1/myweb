// Package selfhost 提供"自建节点"能力（挂载在专线节点 custom_nodes 体系下）：
// 1. 手动搭建：管理员生成一次性安装令牌，用户到 VPS 执行面板下发的安装脚本，
//    脚本自动部署 sing-box、生成协议凭据并回传节点链接，面板据此激活节点。
// 2. VPS 自动搭建：管理员填 VPS IP/SSH端口/root密码，面板 SSH 全自动部署（见 sshdeploy）。
// 节点通过心跳维护在线状态，并上报流量统计。
package selfhost

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"

	"gorm.io/gorm"
)

// 自建节点状态
const (
	StatusPending  = "pending"  // 等待安装
	StatusOnline   = "online"   // 已回传且心跳正常
	StatusOffline  = "offline"  // 心跳超时
	StatusExpired  = "expired"  // 安装令牌过期且未回传
	StatusCanceled = "canceled" // 管理员取消/删除
)

// 支持的协议（与前端下拉一致）
var SupportedProtocols = []string{"vless-ws", "vmess-ws", "vless-reality", "trojan-ws", "ss"}

// 配置常量
const (
	installTokenTTL   = 30 * time.Minute // 安装令牌有效期
	heartbeatTimeout  = 3 * time.Minute  // 心跳超时阈值
	heartbeatInterval = 30 * time.Second // 脚本心跳间隔（须小于 heartbeatTimeout）
)

var (
	ErrNotFound      = errors.New("自建节点不存在")
	ErrTokenInvalid  = errors.New("安装令牌无效或已过期")
	ErrAlreadyReport = errors.New("该节点已完成安装回传")
	ErrBadProtocol   = errors.New("不支持的协议")
)

// ProtocolDisplay 返回协议的中文展示名（前端直接用，后端校验时用英文值）。
func ProtocolDisplay(p string) string {
	switch p {
	case "vless-ws":
		return "VLESS + WebSocket"
	case "vmess-ws":
		return "VMess + WebSocket"
	case "vless-reality":
		return "VLESS + Reality"
	case "trojan-ws":
		return "Trojan + WebSocket"
	case "ss":
		return "Shadowsocks"
	default:
		return p
	}
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GenerateUUID 生成一个 v4 格式 UUID（重置/改密码用）。
func GenerateUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16])), nil
}

// InstallToken 承载新生成的安装标识与令牌。
type InstallToken struct {
	InstallID string
	Token     string
}

// NewInstallToken 生成一对新的安装标识+令牌（重装/换绑用）。
func NewInstallToken() (*InstallToken, error) {
	installID, err := randomHex(8)
	if err != nil {
		return nil, err
	}
	token, err := randomHex(32)
	if err != nil {
		return nil, err
	}
	return &InstallToken{InstallID: installID, Token: token}, nil
}

// CreateRecord 创建一个自建节点占位记录（custom_nodes 表），返回 install_id 与安装令牌。
// 占位记录 IsActive=false 且 Config 为空，不会进入订阅；回传成功后才会生效。
// vpsHost 非空时同时记录 SSH 主机信息（VPS 自动搭建场景）。
func CreateRecord(db *gorm.DB, name, protocol string, vpsHost string, sshPort int, sshUser string) (*models.CustomNode, string, string, error) {
	protocol = strings.TrimSpace(protocol)
	if protocol == "" {
		return nil, "", "", ErrBadProtocol
	}
	valid := false
	for _, p := range SupportedProtocols {
		if p == protocol {
			valid = true
			break
		}
	}
	if !valid {
		return nil, "", "", ErrBadProtocol
	}

	installID, err := randomHex(8)
	if err != nil {
		return nil, "", "", err
	}
	token, err := randomHex(32)
	if err != nil {
		return nil, "", "", err
	}
	now := time.Now()
	expiresAt := now.Add(installTokenTTL)

	node := models.CustomNode{
		Name:              name,
		DisplayName:       name,
		Protocol:          protocol,
		Status:            StatusPending,
		IsActive:          false,
		Source:            "selfhost",
		SelfHosted:        true,
		SelfHostProtocol:  protocol,
		InstallID:         installID,
		InstallToken:      token,
		InstallExpiresAt:  &expiresAt,
		LastHeartbeatAt:   nil,
		SSHHost:           vpsHost,
		SSHPort:           sshPort,
		SSHUser:           sshUser,
	}
	// 显式 Select 强制写入零值字段（IsActive=false），
	// 否则 GORM 会跳过零值导致落库为模型默认值 true，占位节点会被误认为激活。
	if err := db.Select("Name", "DisplayName", "Protocol", "Status", "IsActive", "Source", "SelfHosted", "SelfHostProtocol", "InstallID", "InstallToken", "InstallExpiresAt", "SSHHost", "SSHPort", "SSHUser").Create(&node).Error; err != nil {
		return nil, "", "", err
	}
	// GORM 对带 default 标签的 bool 零值字段仍会回填 DB 默认值（true），
	// 这里再显式 UPDATE 一次确保占位节点 is_active=false（防误入订阅）。
	if err := db.Model(&node).UpdateColumn("is_active", false).Error; err != nil {
		return nil, "", "", err
	}
	return &node, installID, token, nil
}

// GetByInstallID 通过安装标识查找自建节点（agent 回传/心跳用）。
func GetByInstallID(db *gorm.DB, installID string) (*models.CustomNode, error) {
	var node models.CustomNode
	// 多协议模式下主节点标记 deploy_mode != ''，优先返回主节点（心跳/回传等按 install_id 定位时
	// 必须命中主节点；子节点只是展示记录，不承载心跳）。
	mainNode := db.Where("install_id = ? AND deploy_mode != ?", installID, "").Order("id ASC").First(&node)
	if mainNode.Error == nil {
		if !node.SelfHosted {
			return nil, ErrNotFound
		}
		return &node, nil
	}
	if err := db.Where("install_id = ?", installID).Order("id ASC").First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !node.SelfHosted {
		return nil, ErrNotFound
	}
	return &node, nil
}

// VerifyToken 校验安装令牌有效性（回传/心跳前调用）。
func VerifyToken(node *models.CustomNode, token string) error {
	if node.InstallToken == "" || node.InstallToken != token {
		return ErrTokenInvalid
	}
	if node.InstallExpiresAt != nil && time.Now().After(*node.InstallExpiresAt) {
		return ErrTokenInvalid
	}
	return nil
}

// VerifyTokenAllowExpired 校验令牌，但不检查过期时间。
// 用途：心跳上报。安装令牌过期只应阻止「安装脚本回传」，不应阻止已上线节点的持续心跳——
// 否则令牌过期后节点心跳被永久拒绝，节点被误判离线并自动禁用，用户订阅中的专线节点消失。
func VerifyTokenAllowExpired(node *models.CustomNode, token string) error {
	if node.InstallToken == "" || node.InstallToken != token {
		return ErrTokenInvalid
	}
	return nil
}

// ReportNode 处理 agent 安装完成后的回传：用解析出的节点链接填充占位记录并激活。
func ReportNode(db *gorm.DB, node *models.CustomNode, link, serverIP string) (*models.CustomNode, error) {
	if node.Status == StatusOnline && node.Config != "" {
		return nil, ErrAlreadyReport
	}
	if node.Status == StatusCanceled {
		return nil, errors.New("该节点已被取消")
	}

	link = strings.TrimSpace(link)
	if link == "" {
		return nil, errors.New("回传的节点链接为空")
	}

	parsed, err := config_update.ParseNodeLink(link)
	if err != nil {
		return nil, fmt.Errorf("回传的节点链接解析失败: %w", err)
	}

	// 若 agent 探测到公网 IP，且链接中未携带时用公网 IP 兜底
	if serverIP != "" && (parsed.Server == "" || parsed.Server == "127.0.0.1" || parsed.Server == "0.0.0.0") {
		parsed.Server = strings.TrimSpace(serverIP)
	}
	if parsed.Server == "" {
		return nil, errors.New("无法确定节点服务器地址")
	}

	cfgJSON, err := json.Marshal(parsed) // #nosec G117 - 代理节点凭据，非用户口令
	if err != nil {
		return nil, err
	}
	now := time.Now()
	port := parsed.Port
	if port <= 0 {
		port = 443
	}
	updates := map[string]interface{}{
		"config":            string(cfgJSON),
		"protocol":          parsed.Type,
		"domain":            parsed.Server,
		"port":              port,
		"status":            StatusOnline,
		"is_active":         true,
		"last_heartbeat_at": &now,
	}
	if node.Name == "" || node.Name == "自建节点" {
		updates["name"] = parsed.Name
		updates["display_name"] = parsed.Name
	}
	if err := db.Model(node).Updates(updates).Error; err != nil {
		return nil, err
	}
	var fresh models.CustomNode
	if err := db.First(&fresh, node.ID).Error; err != nil {
		return nil, err
	}
	return &fresh, nil
}

// Heartbeat 更新自建节点心跳时间并保持在线状态。
// trafficUp / trafficDown 为自上次上报以来的增量。
// 节点曾因心跳超时被屏蔽（is_active=false）时，心跳恢复会自动重新启用。
// 多协议部署：同 install_id 的协议子节点共享心跳（同步更新状态/流量）。
func Heartbeat(db *gorm.DB, node *models.CustomNode, trafficUp, trafficDown int64) error {
	now := time.Now()
	if node.Status == StatusCanceled {
		return errors.New("该节点已被取消")
	}
	updates := map[string]interface{}{
		"status":            StatusOnline,
		"last_heartbeat_at": &now,
		"is_active":         true, // 心跳恢复即自动启用
	}
	if trafficUp > 0 || trafficDown > 0 {
		updates["traffic_up"] = gorm.Expr("traffic_up + ?", trafficUp)
		updates["traffic_down"] = gorm.Expr("traffic_down + ?", trafficDown)
		updates["traffic_updated_at"] = &now
	}
	if err := db.Model(node).Updates(updates).Error; err != nil {
		return err
	}
	// 同步更新同 install_id 的协议子节点（多协议模式）
	if node.InstallID != "" {
		db.Model(&models.CustomNode{}).
			Where("install_id = ? AND id != ?", node.InstallID, node.ID).
			Updates(map[string]interface{}{
				"status":            StatusOnline,
				"last_heartbeat_at": &now,
				"is_active":         true,
				"traffic_up":        gorm.Expr("traffic_up + ?", trafficUp),
				"traffic_down":      gorm.Expr("traffic_down + ?", trafficDown),
				"traffic_updated_at": &now,
			})
	}
	return nil
}

// MarkOffline 将心跳超时的自建节点标记为离线（调度器定期调用）。
func MarkOffline(db *gorm.DB, timeout time.Duration) (int, error) {
	threshold := time.Now().Add(-timeout)
	res := db.Model(&models.CustomNode{}).
		Where("self_hosted = ? AND status = ? AND (last_heartbeat_at IS NULL OR last_heartbeat_at < ?)",
			true, StatusOnline, threshold).
		Update("status", StatusOffline)
	return int(res.RowsAffected), res.Error
}

// ExpirePending 将超过令牌有效期仍未回传的自建节点标记为过期（调度器定期调用）。
func ExpirePending(db *gorm.DB) (int, error) {
	now := time.Now()
	res := db.Model(&models.CustomNode{}).
		Where("self_hosted = ? AND status = ? AND install_expires_at IS NOT NULL AND install_expires_at < ?",
			true, StatusPending, now).
		Update("status", StatusExpired)
	return int(res.RowsAffected), res.Error
}

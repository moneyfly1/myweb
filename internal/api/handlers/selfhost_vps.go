package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/selfhost"
	"cboard-go/internal/services/sshdeploy"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ============================================================
// 自建节点 VPS 自动搭建 / 批量管理（SSH）
// ============================================================

// findSelfHostNodeBySSHHost 查找某 VPS（ssh_host + ssh_port）上已部署的自建节点。
// 只匹配未取消的自建节点；多协议模式返回主节点（deploy_mode 非空优先）。
func findSelfHostNodeBySSHHost(db *gorm.DB, host string, port int) *models.CustomNode {
	if host == "" {
		return nil
	}
	var node models.CustomNode
	// 主节点优先（多协议部署时子节点共享同一 VPS，只提示一次）
	mainErr := db.Where("self_hosted = ? AND ssh_host = ? AND ssh_port = ? AND deploy_mode != ? AND status != ?",
		true, host, port, "", selfhost.StatusCanceled).Order("id ASC").First(&node).Error
	if mainErr == nil {
		return &node
	}
	if err := db.Where("self_hosted = ? AND ssh_host = ? AND ssh_port = ? AND status != ?",
		true, host, port, selfhost.StatusCanceled).Order("id ASC").First(&node).Error; err == nil {
		return &node
	}
	return nil
}

// prepareSelfHostNodeForDeploy 准备待部署的自建节点记录：
//   - reuseNodeID > 0：复用该节点记录（校验属于该 VPS），换新安装令牌、重置为 pending、
//     同步同 install_id 的子节点 → 返回 (node, installID, token)
//   - reuseNodeID == 0：创建全新占位记录
//
// 返回是否复用，供调用方决定审计文案。
func prepareSelfHostNodeForDeploy(db *gorm.DB, name, protocol, host string, sshPort int, sshUser string, reuseNodeID uint) (*models.CustomNode, string, string, bool, error) {
	if reuseNodeID > 0 {
		var node models.CustomNode
		if err := db.First(&node, reuseNodeID).Error; err != nil {
			return nil, "", "", false, fmt.Errorf("复用节点不存在（ID %d）", reuseNodeID)
		}
		if !node.SelfHosted {
			return nil, "", "", false, fmt.Errorf("节点 #%d 不是自建节点，无法复用", reuseNodeID)
		}
		if node.SSHHost != "" && node.SSHHost != host {
			return nil, "", "", false, fmt.Errorf("节点 #%d 属于另一台 VPS（%s），无法复用", reuseNodeID, node.SSHHost)
		}
		tok, err := selfhost.NewInstallToken()
		if err != nil {
			return nil, "", "", false, err
		}
		expiresAt := time.Now().Add(30 * time.Minute)
		// 保存重装前 install_id（GORM Updates 回写结构体导致 Where 失配的坑，见 reinstall）
		oldInstallID := node.InstallID
		updates := map[string]interface{}{
			"name":               name,
			"display_name":       name,
			"status":             selfhost.StatusPending,
			"is_active":          false,
			"config":             "",
			"install_id":         tok.InstallID,
			"install_token":      tok.Token,
			"install_expires_at": &expiresAt,
			"last_heartbeat_at":  nil,
			"ssh_host":           host,
			"ssh_port":           sshPort,
			"ssh_user":           sshUser,
		}
		if protocol != "" {
			updates["self_host_protocol"] = protocol
			updates["protocol"] = protocol
		}
		if err := db.Model(&node).Updates(updates).Error; err != nil {
			return nil, "", "", false, fmt.Errorf("重置节点状态失败: %w", err)
		}
		// 同步更新同 install_id 的子节点（否则子节点心跳同步失效）
		db.Model(&models.CustomNode{}).
			Where("install_id = ? AND id != ?", oldInstallID, node.ID).
			Updates(map[string]interface{}{
				"install_id":        tok.InstallID,
				"install_token":     tok.Token,
				"status":            selfhost.StatusPending,
				"is_active":         false,
				"config":            "",
				"last_heartbeat_at": nil,
			})
		return &node, tok.InstallID, tok.Token, true, nil
	}
	node, installID, token, err := selfhost.CreateRecord(db, name, protocol, host, sshPort, sshUser)
	if err != nil {
		return nil, "", "", false, err
	}
	return node, installID, token, false, nil
}

// resolveSSHPassword 解析部署用的 SSH 密码：
//   - 优先用请求里明文密码；
//   - 请求没传密码但给了 savedNodeID（已保存的 VPS）时，从该节点解密存储的密码；
//   - 都拿不到返回错误。
func resolveSSHPassword(db *gorm.DB, reqPass string, savedNodeID uint) (string, error) {
	if strings.TrimSpace(reqPass) != "" {
		return strings.TrimSpace(reqPass), nil
	}
	if savedNodeID > 0 {
		var node models.CustomNode
		if err := db.First(&node, savedNodeID).Error; err == nil && node.SSHPasswordEnc != "" {
			dec, err := utils.DecryptSecret(node.SSHPasswordEnc)
			if err == nil && dec != "" {
				return dec, nil
			}
		}
		return "", fmt.Errorf("无法从已保存节点 #%d 读取 SSH 密码，请手动输入", savedNodeID)
	}
	return "", fmt.Errorf("root 密码不能为空")
}

// DeploySelfHostVPS 填 VPS IP/SSH端口/root密码 → SSH 全自动部署 sing-box 节点。
// POST /admin/custom-nodes/selfhost/deploy
func DeploySelfHostVPS(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Protocol    string `json:"protocol" binding:"required"`
		SSHHost     string `json:"ssh_host" binding:"required"`
		SSHPort     int    `json:"ssh_port"`
		SSHUser     string `json:"ssh_user"`
		SSHPass     string `json:"ssh_pass"`
		ReuseNodeID uint   `json:"reuse_node_id"` // 复用已有节点记录重装（覆盖同一 VPS 旧节点）
		SavedSSHID  uint   `json:"saved_ssh_id"`  // 引用已保存 VPS 的加密密码（不传明文密码时使用）
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请填写节点名称、协议、VPS 地址与 root 密码", err)
		return
	}

	name := strings.TrimSpace(req.Name)
	host := strings.TrimSpace(req.SSHHost)
	if name == "" || host == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "节点名称与 VPS 地址不能为空", nil)
		return
	}
	sshPort := req.SSHPort
	if sshPort <= 0 {
		sshPort = 22
	}
	sshUser := strings.TrimSpace(req.SSHUser)
	if sshUser == "" {
		sshUser = "root"
	}

	db := database.GetDB()

	// ===== 防二次部署覆盖：该 VPS 已有自建节点时必须显式复用 =====
	if req.ReuseNodeID == 0 {
		if existing := findSelfHostNodeBySSHHost(db, host, sshPort); existing != nil {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"code":    "vps_occupied",
				"message": fmt.Sprintf("该 VPS 已部署自建节点「%s」（#%d），直接再次搭建会覆盖它并使旧节点失效。如需重装请确认复用该节点。", existing.Name, existing.ID),
				"data": gin.H{
					"existing_node_id":     existing.ID,
					"existing_node_name":   existing.Name,
					"existing_node_status": existing.Status,
				},
				"timestamp": time.Now().Unix(),
			})
			return
		}
	}

	// 解析 SSH 密码：明文优先，否则用已保存 VPS 的加密密码
	pass, err := resolveSSHPassword(db, req.SSHPass, req.SavedSSHID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), err)
		return
	}

	// 先验证 SSH 凭据可用再重置节点（避免密码错误/SSH 不可达时破坏现有节点，
	// 见 DeploySelfHostVPSDomain 同注释）
	client, err := sshdeploy.Dial(sshdeploy.Credentials{
		Host:     host,
		Port:     sshPort,
		User:     sshUser,
		Password: pass,
	}, 15*time.Second)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadGateway, "SSH 连接失败: "+err.Error(), err)
		return
	}
	defer client.Close()

	// 1. 创建占位记录（含 SSH 信息）或复用已有记录，获得 install_id/token
	node, installID, token, reused, err := prepareSelfHostNodeForDeploy(db, name, req.Protocol, host, sshPort, sshUser, req.ReuseNodeID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "准备自建节点失败: "+err.Error(), err)
		return
	}

	// 2. 加密存储 SSH 密码（供后续远程管理）
	encPass, err := utils.EncryptSecret(pass)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "加密 SSH 密码失败", err)
		return
	}
	if err := db.Model(node).Update("ssh_password_enc", encPass).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "保存 SSH 凭据失败", err)
		return
	}

	// 3. SSH 连接 VPS 部署
	// 回传地址：优先用 PANEL_PUBLIC_URL 环境变量（VPS 可达的面板公网地址，生产必配）；
	// 未配置时回退到请求视角地址（本地开发场景需自行用隧道/公网映射）。
	panelBase := os.Getenv("PANEL_PUBLIC_URL")
	if panelBase == "" {
		panelBase = resolvePanelBaseURL(c)
	}
	script, err := selfhost.BuildInstallScript(selfhost.ScriptConfig{
		PanelBaseURL: panelBase,
		InstallID:    installID,
		Token:        token,
		Protocol:     req.Protocol,
		NodeName:     name,
		MirrorURLs:   selfhost.DefaultMirrorURLs(),
		GeneratedAt:  time.Now(),
	})
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成安装脚本失败: "+err.Error(), err)
		return
	}

	// 脚本里的回传地址需为 VPS 可访问地址：把脚本中的面板地址替换为 SSH 目标主机视角不可行，
	// 直接复用面板地址（若面板在公网可达则正常；本地开发场景由反向隧道转发）。
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	out, err := client.UploadAndRun(ctx, "/tmp/cboard-install.sh", script, 3*time.Minute)
	if err != nil {
		// 部署失败：把本次创建的占位记录清理掉（复用场景保留原记录，避免误删旧节点）
		if !reused {
			db.Model(&models.CustomNode{}).Where("id = ? AND status = ?", node.ID, selfhost.StatusPending).Update("status", selfhost.StatusCanceled)
		}
		utils.ErrorResponse(c, http.StatusBadGateway, "VPS 部署失败: "+err.Error(), err)
		return
	}

	action := "deploy_selfhost_vps"
	actionDesc := fmt.Sprintf("管理员操作: VPS 自动搭建自建节点 %s (%s:%d 协议 %s)", name, host, sshPort, selfhost.ProtocolDisplay(req.Protocol))
	if reused {
		action = "redeploy_selfhost_vps"
		actionDesc = fmt.Sprintf("管理员操作: 复用节点重装 %s (%s:%d 协议 %s)", name, host, sshPort, selfhost.ProtocolDisplay(req.Protocol))
	}
	utils.CreateAuditLogSimple(c, action, "custom_node", node.ID, actionDesc)

	utils.SuccessResponse(c, http.StatusOK, "VPS 自动搭建完成", gin.H{
		"node_id":    node.ID,
		"install_id": installID,
		"output":     out,
		"ssh_host":   host,
		"ssh_port":   sshPort,
		"reused":     reused,
	})
}

// DeploySelfHostVPSDomain 域名多协议全自动搭建：
// 填 VPS IP/SSH端口/root密码 + 域名 + 邮箱 → SSH 部署 sing-box 多协议（含 TLS 证书）→ 批量回传。
// POST /admin/custom-nodes/selfhost/deploy-domain
func DeploySelfHostVPSDomain(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Domain      string   `json:"domain"`
		Email       string   `json:"email"`
		Protocols   []string `json:"protocols"` // 为空则用默认多协议组合
		SSHHost     string   `json:"ssh_host" binding:"required"`
		SSHPort     int      `json:"ssh_port"`
		SSHUser     string   `json:"ssh_user"`
		SSHPass     string   `json:"ssh_pass"`
		ReuseNodeID uint     `json:"reuse_node_id"` // 复用已有节点记录重装（覆盖同一 VPS 旧节点）
		SavedSSHID  uint     `json:"saved_ssh_id"`  // 引用已保存 VPS 的加密密码（不传明文密码时使用）
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请填写节点名称、VPS 地址与 root 密码", err)
		return
	}
	name := strings.TrimSpace(req.Name)
	host := strings.TrimSpace(req.SSHHost)
	domain := strings.TrimSpace(req.Domain)
	email := strings.TrimSpace(req.Email)
	if name == "" || host == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "节点名称与 VPS 地址不能为空", nil)
		return
	}
	sshPort := req.SSHPort
	if sshPort <= 0 {
		sshPort = 22
	}
	sshUser := strings.TrimSpace(req.SSHUser)
	if sshUser == "" {
		sshUser = "root"
	}

	// 协议组合：未指定时用默认多协议（VLESS+WS+TLS / VLESS+Reality / Trojan+WS+TLS / SS）
	protocols := req.Protocols
	if len(protocols) == 0 {
		protocols = []string{"vless-ws", "vless-reality", "trojan-ws", "ss"}
	}
	// 无域名时：TLS 协议需要证书（acme 无法申请），自动过滤仅保留无需证书的协议（Reality 系列 + SS）
	if domain == "" {
		filtered := make([]string, 0, len(protocols))
		for _, p := range protocols {
			switch p {
			case "vless-reality", "vless-reality-grpc", "vless-reality-xhttp", "ss":
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			utils.ErrorResponse(c, http.StatusBadRequest, "无域名时只能选择 Reality 系列或 Shadowsocks 协议", nil)
			return
		}
		protocols = filtered
	}
	protoList := make([]selfhost.XrayProtocol, 0, len(protocols))
	for _, p := range protocols {
		switch p {
		case "vless-ws":
			protoList = append(protoList, selfhost.XrayProtocol{Key: "vless-ws", Port: 443, Domain: domain})
		case "vmess-ws":
			protoList = append(protoList, selfhost.XrayProtocol{Key: "vmess-ws", Port: 8443, Domain: domain})
		case "vless-reality":
			protoList = append(protoList, selfhost.XrayProtocol{Key: "vless-reality", Port: 8444})
		case "vless-reality-grpc":
			protoList = append(protoList, selfhost.XrayProtocol{Key: "vless-reality-grpc", Port: 8445})
		case "vless-reality-xhttp":
			protoList = append(protoList, selfhost.XrayProtocol{Key: "vless-reality-xhttp", Port: 8446})
		case "vless-grpc-tls":
			protoList = append(protoList, selfhost.XrayProtocol{Key: "vless-grpc-tls", Port: 2053, Domain: domain})
		case "vless-tcp-tls":
			protoList = append(protoList, selfhost.XrayProtocol{Key: "vless-tcp-tls", Port: 2056, Domain: domain})
		case "trojan-tcp-tls":
			protoList = append(protoList, selfhost.XrayProtocol{Key: "trojan-tcp-tls", Port: 2083, Domain: domain})
		case "trojan-ws":
			protoList = append(protoList, selfhost.XrayProtocol{Key: "trojan-ws", Port: 2055, Domain: domain})
		case "trojan-grpc-tls":
			protoList = append(protoList, selfhost.XrayProtocol{Key: "trojan-grpc-tls", Port: 2087, Domain: domain})
		case "anytls":
			protoList = append(protoList, selfhost.XrayProtocol{Key: "anytls", Port: 2089, Domain: domain})
		case "vmess-httpupgrade":
			protoList = append(protoList, selfhost.XrayProtocol{Key: "vmess-httpupgrade", Port: 2091, Domain: domain})
		case "ss":
			protoList = append(protoList, selfhost.XrayProtocol{Key: "ss", Port: 8388})
		case "hysteria2":
			protoList = append(protoList, selfhost.XrayProtocol{Key: "hysteria2", Port: 36712, Domain: domain})
		case "tuic":
			protoList = append(protoList, selfhost.XrayProtocol{Key: "tuic", Port: 36713, Domain: domain})
		}
	}
	if len(protoList) == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "没有有效的协议选择", nil)
		return
	}

	db := database.GetDB()

	// ===== 防二次部署覆盖：该 VPS 已有自建节点时必须显式复用 =====
	if req.ReuseNodeID == 0 {
		if existing := findSelfHostNodeBySSHHost(db, host, sshPort); existing != nil {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"code":    "vps_occupied",
				"message": fmt.Sprintf("该 VPS 已部署自建节点「%s」（#%d），直接再次搭建会覆盖它并使旧节点失效。如需重装请确认复用该节点。", existing.Name, existing.ID),
				"data": gin.H{
					"existing_node_id":     existing.ID,
					"existing_node_name":   existing.Name,
					"existing_node_status": existing.Status,
				},
				"timestamp": time.Now().Unix(),
			})
			return
		}
	}

	// 解析 SSH 密码：明文优先，否则用已保存 VPS 的加密密码
	pass, err := resolveSSHPassword(db, req.SSHPass, req.SavedSSHID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), err)
		return
	}

	// 先验证 SSH 凭据可用再重置节点（避免密码错误/SSH 不可达时破坏现有节点：
	// prepareSelfHostNodeForDeploy 会把节点置为 pending+禁用+换 install_id，
	// 若此时 SSH 连接失败，节点将无法恢复。提前 Dial 可让失败发生在任何 DB 变更之前）
	client, err := sshdeploy.Dial(sshdeploy.Credentials{Host: host, Port: sshPort, User: sshUser, Password: pass}, 15*time.Second)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadGateway, "SSH 连接失败: "+err.Error(), err)
		return
	}
	defer client.Close()

	node, installID, token, reused, err := prepareSelfHostNodeForDeploy(db, name, protoList[0].Key, host, sshPort, sshUser, req.ReuseNodeID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "准备自建节点失败: "+err.Error(), err)
		return
	}
	encPass, err := utils.EncryptSecret(pass)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "加密 SSH 密码失败", err)
		return
	}
	if err := db.Model(node).Updates(map[string]interface{}{
		"ssh_password_enc": encPass,
		"domain":           domain,
		"core_type":        "sing-box",
		"deploy_mode":      "multi",
		"protocol_list":    joinStrings(selfhost.XrayProtocolNames(protoList)),
	}).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "保存节点信息失败", err)
		return
	}

	panelBase := os.Getenv("PANEL_PUBLIC_URL")
	if panelBase == "" {
		panelBase = resolvePanelBaseURL(c)
	}
	script, err := selfhost.BuildXrayInstallScript(selfhost.XrayScriptConfig{
		PanelBaseURL: panelBase,
		InstallID:    installID,
		Token:        token,
		Domain:       domain,
		Email:        email,
		Protocols:    protoList,
		MirrorURLs:   selfhost.DefaultMirrorURLs(),
		GeneratedAt:  time.Now(),
	})
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成部署脚本失败: "+err.Error(), err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	out, err := client.UploadAndRun(ctx, "/tmp/cboard-domain-install.sh", script, 5*time.Minute)
	if err != nil {
		// 部署失败：把本次创建的占位记录清理掉（复用场景保留原记录，避免误删旧节点）
		if !reused {
			db.Model(&models.CustomNode{}).Where("id = ? AND status = ?", node.ID, selfhost.StatusPending).Update("status", selfhost.StatusCanceled)
		}
		utils.ErrorResponse(c, http.StatusBadGateway, "域名部署失败: "+err.Error(), err)
		return
	}

	action := "deploy_selfhost_vps_domain"
	actionDesc := fmt.Sprintf("管理员操作: 域名多协议搭建 %s (%s 域名 %s 协议 %d 个)", name, host, domain, len(protoList))
	if reused {
		action = "redeploy_selfhost_vps_domain"
		actionDesc = fmt.Sprintf("管理员操作: 复用节点重装 %s (%s 域名 %s 协议 %d 个)", name, host, domain, len(protoList))
	}
	utils.CreateAuditLogSimple(c, action, "custom_node", node.ID, actionDesc)

	utils.SuccessResponse(c, http.StatusOK, "域名多协议搭建完成", gin.H{
		"node_id":    node.ID,
		"install_id": installID,
		"domain":     domain,
		"protocols":  selfhost.XrayProtocolNames(protoList),
		"output":     out,
		"ssh_host":   host,
		"ssh_port":   sshPort,
	})
}

func joinStrings(s []string) string {
	return strings.Join(s, ",")
}

// defaultProtocolPort 返回各协议的重装默认端口（与 DeploySelfHostVPSDomain 分配一致）。
func defaultProtocolPort(key string) int {
	switch key {
	case "vless-ws":
		return 443
	case "vmess-ws":
		return 8443
	case "vless-reality":
		return 8444
	case "vless-reality-grpc":
		return 8445
	case "vless-reality-xhttp":
		return 8446
	case "vless-grpc-tls":
		return 2053
	case "vless-tcp-tls":
		return 2056
	case "trojan-tcp-tls":
		return 2083
	case "trojan-ws":
		return 2055
	case "trojan-grpc-tls":
		return 2087
	case "anytls":
		return 2089
	case "vmess-httpupgrade":
		return 2091
	case "hysteria2":
		return 36712
	case "tuic":
		return 36713
	case "ss":
		return 8388
	default:
		return 443
	}
}

// parseProtocolList 把节点 DB 里的 protocol_list（逗号分隔协议 key）重建为 XrayProtocol 列表。
// 重装多协议节点用（端口按默认分配恢复；若某协议无默认端口则跳过）。
func parseProtocolList(protocolList, domain string) []selfhost.XrayProtocol {
	keys := strings.Split(protocolList, ",")
	protos := make([]selfhost.XrayProtocol, 0, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		protos = append(protos, selfhost.XrayProtocol{Key: k, Port: defaultProtocolPort(k), Domain: domain})
	}
	return protos
}

// SelfHostUpdateSSH 更新自建节点的 SSH 凭据（VPS 密码变更后使用）。
// POST /admin/custom-nodes/selfhost/:id/update-ssh  body: {"ssh_host":"","ssh_port":22,"ssh_user":"root","ssh_pass":"newpassword"}
func SelfHostUpdateSSH(c *gin.Context) {
	nodeID := c.Param("id")
	var req struct {
		SSHHost string `json:"ssh_host"`
		SSHPort int    `json:"ssh_port"`
		SSHUser string `json:"ssh_user"`
		SSHPass string `json:"ssh_pass" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请提供新的 SSH 密码", err)
		return
	}
	pass := strings.TrimSpace(req.SSHPass)
	if pass == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "SSH 密码不能为空", nil)
		return
	}

	db := database.GetDB()
	var node models.CustomNode
	if err := db.First(&node, nodeID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "自建节点不存在", err)
		return
	}
	if !node.SelfHosted {
		utils.ErrorResponse(c, http.StatusBadRequest, "该节点不是自建节点", nil)
		return
	}

	encPass, err := utils.EncryptSecret(pass)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "加密 SSH 密码失败", err)
		return
	}
	updates := map[string]interface{}{"ssh_password_enc": encPass}
	if req.SSHHost != "" {
		updates["ssh_host"] = strings.TrimSpace(req.SSHHost)
	}
	if req.SSHPort > 0 {
		updates["ssh_port"] = req.SSHPort
	}
	if strings.TrimSpace(req.SSHUser) != "" {
		updates["ssh_user"] = strings.TrimSpace(req.SSHUser)
	}
	if err := db.Model(&node).Updates(updates).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新 SSH 凭据失败", err)
		return
	}

	utils.CreateAuditLogSimple(c, "update_selfhost_ssh", "custom_node", node.ID,
		fmt.Sprintf("管理员操作: 更新自建节点 %s 的 SSH 凭据", node.Name))

	utils.SuccessResponse(c, http.StatusOK, "SSH 凭据已更新", nil)
}

// SelfHostTrafficLimit 设置/更新自建节点流量配额。
// POST /admin/custom-nodes/selfhost/:id/traffic-limit  body: {"enabled":true,"limit_bytes":107374182400}
func SelfHostTrafficLimit(c *gin.Context) {
	nodeID := c.Param("id")
	var req struct {
		Enabled    bool  `json:"enabled"`
		LimitBytes int64 `json:"limit_bytes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}
	if req.Enabled && req.LimitBytes <= 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "配额必须大于 0", nil)
		return
	}

	db := database.GetDB()
	var node models.CustomNode
	if err := db.First(&node, nodeID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "自建节点不存在", err)
		return
	}
	if !node.SelfHosted {
		utils.ErrorResponse(c, http.StatusBadRequest, "该节点不是自建节点", nil)
		return
	}

	updates := map[string]interface{}{
		"traffic_limit_enabled": req.Enabled,
		"traffic_limit_bytes":   req.LimitBytes,
	}
	if !req.Enabled {
		updates["traffic_limit_reset_at"] = nil
	}
	if err := db.Model(&node).Updates(updates).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "设置配额失败", err)
		return
	}

	utils.CreateAuditLogSimple(c, "selfhost_traffic_limit", "custom_node", node.ID,
		fmt.Sprintf("管理员操作: 设置自建节点 %s 流量配额 %s", node.Name, formatBytesHuman(req.LimitBytes)))

	utils.SuccessResponse(c, http.StatusOK, "流量配额已设置", gin.H{
		"enabled":     req.Enabled,
		"limit_bytes": req.LimitBytes,
		"used":        node.TrafficUp + node.TrafficDown,
	})
}

// SelfHostResetTraffic 清零自建节点已用流量。
// POST /admin/custom-nodes/selfhost/:id/reset-traffic
func SelfHostResetTraffic(c *gin.Context) {
	nodeID := c.Param("id")
	db := database.GetDB()
	var node models.CustomNode
	if err := db.First(&node, nodeID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "自建节点不存在", err)
		return
	}
	if !node.SelfHosted {
		utils.ErrorResponse(c, http.StatusBadRequest, "该节点不是自建节点", nil)
		return
	}
	if err := db.Model(&node).Updates(map[string]interface{}{
		"traffic_up":             0,
		"traffic_down":           0,
		"traffic_limit_reset_at": nil,
	}).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "清零流量失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "流量已清零", nil)
}

func formatBytesHuman(b int64) string {
	if b <= 0 {
		return "0"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	i := 0
	f := float64(b)
	for f >= 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	return fmt.Sprintf("%.2f%s", f, units[i])
}

// SelfHostBatchManage 自建节点批量管理入口：
// POST /admin/custom-nodes/selfhost/:id/manage  body: {"action":"reset|change-password|change-port|reinstall|status", ...}
func SelfHostBatchManage(c *gin.Context) {
	nodeID := c.Param("id")
	var req struct {
		Action  string `json:"action" binding:"required"` // reset / change-password / change-port / reinstall / status
		NewPass string `json:"new_pass"`
		NewPort int    `json:"new_port"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Action == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "请提供管理动作", err)
		return
	}

	db := database.GetDB()
	var node models.CustomNode
	if err := db.First(&node, nodeID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "自建节点不存在", err)
		return
	}
	if !node.SelfHosted {
		utils.ErrorResponse(c, http.StatusBadRequest, "该节点不是自建节点", nil)
		return
	}
	if node.SSHHost == "" || node.SSHPasswordEnc == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "该节点无 SSH 凭据（非 VPS 自动搭建节点）", nil)
		return
	}

	pass, err := utils.DecryptSecret(node.SSHPasswordEnc)
	if err != nil || pass == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "SSH 凭据解密失败: "+err.Error(), err)
		return
	}

	client, err := sshdeploy.Dial(sshdeploy.Credentials{
		Host:     node.SSHHost,
		Port:     node.SSHPort,
		User:     node.SSHUser,
		Password: pass,
	}, 15*time.Second)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadGateway, "SSH 连接失败: "+err.Error(), err)
		return
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	switch req.Action {
	case "status":
		out, err := client.RunWithTimeout("systemctl is-active sing-box 2>/dev/null; echo '---'; ss -tlnp 2>/dev/null | grep -E ':443 |:8443 ' || echo 'no listener'; echo '---'; /usr/local/bin/sing-box version 2>/dev/null | head -1 || echo 'sing-box not installed'", 20*time.Second)
		if err != nil {
			out = out + "\n(err: " + err.Error() + ")"
		}
		utils.SuccessResponse(c, http.StatusOK, "", gin.H{"status": out})
		return

	case "reset":
		// 重新生成 UUID 并重启（重置节点凭据）
		newUUID, _ := selfhost.GenerateUUID()
		conf, err := client.ReadFile("/etc/sing-box/config.json")
		if err != nil || conf == "" {
			utils.ErrorResponse(c, http.StatusBadGateway, "读取远程配置失败", err)
			return
		}
		updated := replaceUUIDInConfig(conf, newUUID)
		if err := client.WriteFile("/etc/sing-box/config.json", updated); err != nil {
			utils.ErrorResponse(c, http.StatusBadGateway, "更新远程配置失败", err)
			return
		}
		if _, err := client.RunWithTimeout("systemctl restart sing-box", 30*time.Second); err != nil {
			utils.ErrorResponse(c, http.StatusBadGateway, "重启 sing-box 失败: "+err.Error(), err)
			return
		}
		syncNodeConfigField(db, &node, "UUID", newUUID)
		utils.CreateAuditLogSimple(c, "reset_selfhost_node", "custom_node", node.ID,
			fmt.Sprintf("管理员操作: 重置自建节点 %s 凭据（新UUID已生成）", node.Name))
		utils.SuccessResponse(c, http.StatusOK, "节点已重置，凭据已更新", gin.H{"new_uuid": newUUID})
		return

	case "change-password":
		if strings.TrimSpace(req.NewPass) == "" {
			utils.ErrorResponse(c, http.StatusBadRequest, "请提供新密码（UUID）", nil)
			return
		}
		newUUID := strings.TrimSpace(req.NewPass)
		conf, err := client.ReadFile("/etc/sing-box/config.json")
		if err != nil || conf == "" {
			utils.ErrorResponse(c, http.StatusBadGateway, "读取远程配置失败", err)
			return
		}
		updated := replaceUUIDInConfig(conf, newUUID)
		if err := client.WriteFile("/etc/sing-box/config.json", updated); err != nil {
			utils.ErrorResponse(c, http.StatusBadGateway, "更新远程配置失败", err)
			return
		}
		if _, err := client.RunWithTimeout("systemctl restart sing-box", 30*time.Second); err != nil {
			utils.ErrorResponse(c, http.StatusBadGateway, "重启 sing-box 失败: "+err.Error(), err)
			return
		}
		syncNodeConfigField(db, &node, "UUID", newUUID)
		utils.CreateAuditLogSimple(c, "change_selfhost_password", "custom_node", node.ID,
			fmt.Sprintf("管理员操作: 更改自建节点 %s 密码", node.Name))
		utils.SuccessResponse(c, http.StatusOK, "节点密码已更新", gin.H{"new_uuid": newUUID})
		return

	case "change-port":
		conf, err := client.ReadFile("/etc/sing-box/config.json")
		if err != nil || conf == "" {
			utils.ErrorResponse(c, http.StatusBadGateway, "读取远程配置失败", err)
			return
		}

		var newPorts []int
		var updated string
		if node.DeployMode == "multi" {
			// 多协议节点：为所有 inbound 随机生成互不重复的新端口
			count := countListenPorts(conf)
			if count == 0 {
				count = 1
			}
			// 获取系统当前监听端口，随机时避开（防止撞上 sshd 等已有服务）
			occupied := fetchSystemListenPorts(client)
			newPorts = randomDistinctPorts(count, occupied)
			updated = replaceAllPortsRandom(conf, newPorts)
			// 主节点端口取第一个（VLESS+WS/TLS 或主协议）
			req.NewPort = newPorts[0]
		} else {
			if req.NewPort <= 0 || req.NewPort > 65535 {
				utils.ErrorResponse(c, http.StatusBadRequest, "请提供有效端口(1-65535)", nil)
				return
			}
			// 单协议手动指定端口：若被系统占用则拒绝
			occupied := fetchSystemListenPorts(client)
			if occupied[req.NewPort] {
				utils.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("端口 %d 已被系统服务占用，请换一个", req.NewPort), nil)
				return
			}
			newPorts = []int{req.NewPort}
			updated = replacePortInConfig(conf, req.NewPort)
		}

		if err := client.WriteFile("/etc/sing-box/config.json", updated); err != nil {
			utils.ErrorResponse(c, http.StatusBadGateway, "更新远程配置失败", err)
			return
		}
		if _, err := client.RunWithTimeout("systemctl restart sing-box", 30*time.Second); err != nil {
			utils.ErrorResponse(c, http.StatusBadGateway, "重启 sing-box 失败: "+err.Error(), err)
			return
		}

		// 更新 DB：主节点 + 同 install_id 的多协议子节点端口
		updates := map[string]interface{}{"port": req.NewPort}
		db.Model(&node).Updates(updates)
		// 同步 config 中的 Port（订阅链接用），否则订阅仍是旧端口
		syncNodeConfigField(db, &node, "Port", req.NewPort)
		if node.DeployMode == "multi" && len(newPorts) > 1 {
			siblings := []models.CustomNode{}
			db.Where("install_id = ? AND id != ?", node.InstallID, node.ID).Find(&siblings)
			for i, sib := range siblings {
				if i+1 < len(newPorts) {
					db.Model(&sib).Update("port", newPorts[i+1])
					syncNodeConfigField(db, &sib, "Port", newPorts[i+1])
				}
			}
		}
		utils.CreateAuditLogSimple(c, "change_selfhost_port", "custom_node", node.ID,
			fmt.Sprintf("管理员操作: 更改自建节点 %s 端口（多协议随机分配 %v）", node.Name, newPorts))
		utils.SuccessResponse(c, http.StatusOK, "节点端口已更新", gin.H{"new_port": req.NewPort, "all_ports": newPorts})
		return

	case "reinstall":
		// 重新搭建：重置节点为待安装状态 + 生成新令牌，远程重装 sing-box 后自动回传。
		// （必须换新 install_id/token 并把 status 置回 pending，
		//   否则 agent/install.sh 会因节点已安装返回 410，回传被拒绝。）
		newInstallID, err := selfhost.NewInstallToken()
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "生成安装令牌失败", err)
			return
		}
		newToken := newInstallID.Token
		newID := newInstallID.InstallID
		expiresAt := time.Now().Add(30 * time.Minute)
		// 保存重装前的 install_id（GORM Updates 可能回写结构体，导致后续同步 Where 失配）
		oldInstallID := node.InstallID
		if err := db.Model(&node).Updates(map[string]interface{}{
			"status":             selfhost.StatusPending,
			"is_active":          false,
			"config":             "",
			"install_id":         newID,
			"install_token":      newToken,
			"install_expires_at": &expiresAt,
		}).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "重置节点状态失败", err)
			return
		}
		// 同步更新同 install_id 的多协议子节点（否则子节点心跳同步失效）
		db.Model(&models.CustomNode{}).
			Where("install_id = ? AND id != ?", oldInstallID, node.ID).
			Updates(map[string]interface{}{
				"install_id":    newID,
				"install_token": newToken,
				"status":        selfhost.StatusPending,
				"is_active":     false,
				"config":        "",
			})

		panelBase := os.Getenv("PANEL_PUBLIC_URL")
		if panelBase == "" {
			panelBase = resolvePanelBaseURL(c)
		}
		// 多协议节点必须用多协议脚本重装（单协议脚本会把 12 协议降级成 1 个）
		if node.DeployMode == "multi" {
			protoList := parseProtocolList(node.ProtocolList, node.Domain)
			if len(protoList) == 0 {
				utils.ErrorResponse(c, http.StatusBadRequest, "该多协议节点的协议列表为空，无法重装", nil)
				return
			}
			script, err := selfhost.BuildXrayInstallScript(selfhost.XrayScriptConfig{
				PanelBaseURL: panelBase,
				InstallID:    newID,
				Token:        newToken,
				Domain:       node.Domain,
				Email:        "",
				Protocols:    protoList,
				MirrorURLs:   selfhost.DefaultMirrorURLs(),
				GeneratedAt:  time.Now(),
			})
			if err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, "生成多协议部署脚本失败: "+err.Error(), err)
				return
			}
			out, err := client.UploadAndRun(ctx, "/tmp/cboard-reinstall.sh", script, 5*time.Minute)
			if err != nil {
				utils.ErrorResponse(c, http.StatusBadGateway, "重新搭建失败: "+err.Error(), err)
				return
			}
			utils.CreateAuditLogSimple(c, "reinstall_selfhost_node", "custom_node", node.ID,
				fmt.Sprintf("管理员操作: 重新搭建多协议自建节点 %s（%d 个协议）", node.Name, len(protoList)))
			utils.SuccessResponse(c, http.StatusOK, "节点已重新搭建", gin.H{"output": out})
			return
		}
		script, err := selfhost.BuildInstallScript(selfhost.ScriptConfig{
			PanelBaseURL: panelBase,
			InstallID:    newID,
			Token:        newToken,
			Protocol:     node.SelfHostProtocol,
			NodeName:     node.Name,
			MirrorURLs:   selfhost.DefaultMirrorURLs(),
			GeneratedAt:  time.Now(),
		})
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "生成安装脚本失败: "+err.Error(), err)
			return
		}
		out, err := client.UploadAndRun(ctx, "/tmp/cboard-reinstall.sh", script, 3*time.Minute)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadGateway, "重新搭建失败: "+err.Error(), err)
			return
		}
		utils.CreateAuditLogSimple(c, "reinstall_selfhost_node", "custom_node", node.ID,
			fmt.Sprintf("管理员操作: 重新搭建自建节点 %s", node.Name))
		utils.SuccessResponse(c, http.StatusOK, "节点已重新搭建", gin.H{"output": out})
		return
	}

	utils.ErrorResponse(c, http.StatusBadRequest, "未知的管理动作: "+req.Action, nil)
}

// replaceUUIDInConfig 替换 sing-box 配置中的 UUID（vless/vmess 用户字段）。
// syncNodeConfigField 更新节点 DB 中 config JSON 的指定字段（Port/UUID/Password），
// 确保订阅生成的链接与实际生效的节点配置一致（改端口/重置/改密码后必须调用）。
func syncNodeConfigField(db *gorm.DB, node *models.CustomNode, field string, value interface{}) {
	if node.Config == "" {
		return
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(node.Config), &cfg); err != nil {
		return
	}
	cfg[field] = value
	updated, err := json.Marshal(cfg)
	if err != nil {
		return
	}
	db.Model(node).Update("config", string(updated))
}

func replaceUUIDInConfig(conf, newUUID string) string {
	// 匹配 "uuid": "xxx-xxx..." （JSON 引号形式）
	re := regexp.MustCompile(`"uuid"\s*:\s*"[0-9a-fA-F-]{36}"`)
	return re.ReplaceAllString(conf, `"uuid": "`+newUUID+`"`)
}

// replacePortInConfig 替换 sing-box 配置中的 listen_port（仅第一个，即主协议）。
// 多协议配置下不能全量替换（会导致端口冲突），只改主节点端口。
func replacePortInConfig(conf string, newPort int) string {
	re := regexp.MustCompile(`"listen_port"\s*:\s*\d+`)
	loc := re.FindStringIndex(conf)
	if loc == nil {
		return conf
	}
	return conf[:loc[0]] + fmt.Sprintf(`"listen_port": %d`, newPort) + conf[loc[1]:]
}

// countListenPorts 统计配置中的 listen_port 数量。
func countListenPorts(conf string) int {
	re := regexp.MustCompile(`"listen_port"\s*:\s*\d+`)
	return len(re.FindAllString(conf, -1))
}

// replaceAllPortsRandom 将配置中所有 listen_port 依次替换为 newPorts 中的端口。
// newPorts 长度需等于配置中的端口数量，否则按可用数量截断/循环。
func replaceAllPortsRandom(conf string, newPorts []int) string {
	re := regexp.MustCompile(`"listen_port"\s*:\s*\d+`)
	var sb strings.Builder
	last := 0
	idx := 0
	for _, loc := range re.FindAllStringIndex(conf, -1) {
		sb.WriteString(conf[last:loc[0]])
		port := newPorts[idx%len(newPorts)]
		sb.WriteString(fmt.Sprintf(`"listen_port": %d`, port))
		last = loc[1]
		idx++
	}
	sb.WriteString(conf[last:])
	return sb.String()
}

// fetchSystemListenPorts 通过 SSH 获取系统当前监听的 TCP 端口集合。
func fetchSystemListenPorts(client *sshdeploy.Client) map[int]bool {
	ports := make(map[int]bool)
	if client == nil {
		return ports
	}
	out, err := client.RunWithTimeout("ss -tln 2>/dev/null | awk '{print $4}' | grep -oE ':[0-9]+$' | grep -oE '[0-9]+' | sort -u", 10*time.Second)
	if err != nil {
		return ports
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if p, err := strconv.Atoi(line); err == nil {
			ports[p] = true
		}
	}
	return ports
}

// randomDistinctPorts 生成 n 个互不重复的随机端口（1024-60000，避开常用端口与系统已占用端口）。
func randomDistinctPorts(n int, occupied map[int]bool) []int {
	// 避开的常用端口（合并系统已占用端口）
	reserved := map[int]bool{
		22: true, 25: true, 53: true, 80: true, 443: true, 465: true,
		853: true, 2053: true, 2083: true, 2087: true, 3306: true,
		5432: true, 8080: true, 8388: true, 8443: true, 8888: true, 10080: true,
	}
	for p := range occupied {
		reserved[p] = true
	}
	seen := make(map[int]bool)
	ports := make([]int, 0, n)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for len(ports) < n {
		p := 1024 + r.Intn(60000-1024)
		if reserved[p] || seen[p] {
			continue
		}
		seen[p] = true
		ports = append(ports, p)
	}
	sort.Ints(ports) // 排序保证稳定，主协议端口优先小值
	return ports
}

// getSelfHostNodeByID 按 ID 查询自建节点（供管理操作复用）。
func getSelfHostNodeByID(db *gorm.DB, id string) (*models.CustomNode, error) {
	var node models.CustomNode
	if err := db.First(&node, id).Error; err != nil {
		return nil, err
	}
	if !node.SelfHosted {
		return nil, fmt.Errorf("该节点不是自建节点")
	}
	return &node, nil
}

// ============================================================
// 自建节点批量管理
// ============================================================

// SelfHostBatchManageMany 批量管理自建节点。
// POST /admin/custom-nodes/selfhost/batch-manage
// body: {"node_ids":[1,2,3], "action":"reset|change-password|change-port|traffic-limit|reset-traffic|enable|disable|delete",
//
//	"new_pass":"", "new_port":0, "enabled":true, "limit_bytes":0}
//
// SSH 类操作（reset/change-password/change-port）按 ssh_host 去重：同一 VPS 只执行一次，避免端口冲突。
func SelfHostBatchManageMany(c *gin.Context) {
	var req struct {
		NodeIDs    []uint `json:"node_ids" binding:"required"`
		Action     string `json:"action" binding:"required"`
		NewPass    string `json:"new_pass"`
		NewPort    int    `json:"new_port"`
		Enabled    bool   `json:"enabled"`
		LimitBytes int64  `json:"limit_bytes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.NodeIDs) == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "请选择节点并提供管理动作", err)
		return
	}

	db := database.GetDB()
	results := make([]gin.H, 0, len(req.NodeIDs))

	switch req.Action {
	case "reset", "change-password", "change-port":
		// SSH 类操作：按 ssh_host 去重，每台 VPS 只执行一次
		// 先收集节点
		var nodes []models.CustomNode
		db.Where("id IN ? AND self_hosted = ?", req.NodeIDs, true).Find(&nodes)
		byHost := make(map[string][]*models.CustomNode)
		for i := range nodes {
			n := nodes[i]
			if n.SSHHost == "" {
				results = append(results, gin.H{"node_id": n.ID, "name": n.Name, "success": false, "message": "无 SSH 凭据（手动节点）"})
				continue
			}
			key := fmt.Sprintf("%s:%d", n.SSHHost, n.SSHPort)
			byHost[key] = append(byHost[key], &nodes[i])
		}
		for _, hostNodes := range byHost {
			primary := hostNodes[0]
			pass, err := utils.DecryptSecret(primary.SSHPasswordEnc)
			if err != nil || pass == "" {
				for _, n := range hostNodes {
					results = append(results, gin.H{"node_id": n.ID, "name": n.Name, "success": false, "message": "SSH 凭据解密失败"})
				}
				continue
			}
			client, err := sshdeploy.Dial(sshdeploy.Credentials{
				Host: primary.SSHHost, Port: primary.SSHPort, User: primary.SSHUser, Password: pass,
			}, 15*time.Second)
			if err != nil {
				for _, n := range hostNodes {
					results = append(results, gin.H{"node_id": n.ID, "name": n.Name, "success": false, "message": "SSH 连接失败: " + err.Error()})
				}
				continue
			}
			conf, err := client.ReadFile("/etc/sing-box/config.json")
			client.Close()
			if err != nil || conf == "" {
				for _, n := range hostNodes {
					results = append(results, gin.H{"node_id": n.ID, "name": n.Name, "success": false, "message": "读取远程配置失败"})
				}
				continue
			}

			var updated string
			var newPorts []int
			switch req.Action {
			case "reset":
				newUUID, _ := selfhost.GenerateUUID()
				updated = replaceUUIDInConfig(conf, newUUID)
				for _, n := range hostNodes {
					syncNodeConfigField(db, n, "UUID", newUUID)
					results = append(results, gin.H{"node_id": n.ID, "name": n.Name, "success": true, "message": "已重置 UUID", "new_uuid": newUUID})
				}
			case "change-password":
				if strings.TrimSpace(req.NewPass) == "" {
					for _, n := range hostNodes {
						results = append(results, gin.H{"node_id": n.ID, "name": n.Name, "success": false, "message": "请提供新密码"})
					}
					continue
				}
				updated = replaceUUIDInConfig(conf, strings.TrimSpace(req.NewPass))
				for _, n := range hostNodes {
					syncNodeConfigField(db, n, "UUID", strings.TrimSpace(req.NewPass))
					results = append(results, gin.H{"node_id": n.ID, "name": n.Name, "success": true, "message": "已更新密码"})
				}
			case "change-port":
				count := countListenPorts(conf)
				if count == 0 {
					count = 1
				}
				occupied := fetchSystemListenPortsByHost(primary.SSHHost, primary.SSHPort, pass, primary.SSHUser)
				newPorts = randomDistinctPorts(count, occupied)
				updated = replaceAllPortsRandom(conf, newPorts)
				for i, n := range hostNodes {
					if i < len(newPorts) {
						db.Model(n).Update("port", newPorts[i])
						syncNodeConfigField(db, n, "Port", newPorts[i])
						results = append(results, gin.H{"node_id": n.ID, "name": n.Name, "success": true, "message": fmt.Sprintf("端口已更新为 %d", newPorts[i]), "new_port": newPorts[i]})
					} else {
						results = append(results, gin.H{"node_id": n.ID, "name": n.Name, "success": true, "message": "端口已更新"})
					}
				}
			}

			// 写回配置并重启
			client2, err := sshdeploy.Dial(sshdeploy.Credentials{
				Host: primary.SSHHost, Port: primary.SSHPort, User: primary.SSHUser, Password: pass,
			}, 15*time.Second)
			if err != nil {
				continue
			}
			if err := client2.WriteFile("/etc/sing-box/config.json", updated); err == nil {
				client2.RunWithTimeout("systemctl restart sing-box", 30*time.Second)
			}
			client2.Close()
		}

	case "traffic-limit", "reset-traffic", "enable", "disable":
		// DB 类操作：逐节点
		for _, id := range req.NodeIDs {
			var n models.CustomNode
			if err := db.First(&n, id).Error; err != nil {
				results = append(results, gin.H{"node_id": id, "success": false, "message": "节点不存在"})
				continue
			}
			if !n.SelfHosted {
				results = append(results, gin.H{"node_id": id, "name": n.Name, "success": false, "message": "非自建节点"})
				continue
			}
			var err error
			switch req.Action {
			case "traffic-limit":
				updates := map[string]interface{}{"traffic_limit_enabled": req.Enabled, "traffic_limit_bytes": req.LimitBytes}
				if !req.Enabled {
					updates["traffic_limit_reset_at"] = nil
				}
				err = db.Model(&n).Updates(updates).Error
				results = append(results, gin.H{"node_id": id, "name": n.Name, "success": err == nil, "message": "配额已设置"})
			case "reset-traffic":
				err = db.Model(&n).Updates(map[string]interface{}{"traffic_up": 0, "traffic_down": 0, "traffic_limit_reset_at": nil}).Error
				results = append(results, gin.H{"node_id": id, "name": n.Name, "success": err == nil, "message": "流量已清零"})
			case "enable":
				err = db.Model(&n).Update("is_active", true).Error
				results = append(results, gin.H{"node_id": id, "name": n.Name, "success": err == nil, "message": "已启用"})
			case "disable":
				err = db.Model(&n).Update("is_active", false).Error
				results = append(results, gin.H{"node_id": id, "name": n.Name, "success": err == nil, "message": "已禁用"})
			}
		}

	case "delete":
		for _, id := range req.NodeIDs {
			var n models.CustomNode
			if err := db.First(&n, id).Error; err != nil {
				results = append(results, gin.H{"node_id": id, "success": false, "message": "节点不存在"})
				continue
			}
			db.Where("custom_node_id = ?", id).Delete(&models.UserCustomNode{})
			if err := db.Delete(&n).Error; err == nil {
				results = append(results, gin.H{"node_id": id, "name": n.Name, "success": true, "message": "已删除"})
			} else {
				results = append(results, gin.H{"node_id": id, "name": n.Name, "success": false, "message": "删除失败"})
			}
		}

	default:
		utils.ErrorResponse(c, http.StatusBadRequest, "未知的管理动作: "+req.Action, nil)
		return
	}

	clearNodeCaches()
	successCount := 0
	for _, r := range results {
		if s, ok := r["success"].(bool); ok && s {
			successCount++
		}
	}
	utils.CreateAuditLogSimple(c, "batch_manage_selfhost_nodes", "custom_node", 0,
		fmt.Sprintf("管理员操作: 批量管理自建节点 %d 个（%s）成功 %d", len(req.NodeIDs), req.Action, successCount))

	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("批量%s成功 %d/%d", req.Action, successCount, len(req.NodeIDs)), gin.H{
		"results": results,
		"success": successCount,
		"total":   len(req.NodeIDs),
	})
}

// fetchSystemListenPortsByHost 通过 SSH 获取指定主机当前监听端口集合。
func fetchSystemListenPortsByHost(host string, port int, pass, user string) map[int]bool {
	ports := make(map[int]bool)
	client, err := sshdeploy.Dial(sshdeploy.Credentials{Host: host, Port: port, User: user, Password: pass}, 15*time.Second)
	if err != nil {
		return ports
	}
	defer client.Close()
	out, err := client.RunWithTimeout("ss -tln 2>/dev/null | awk '{print $4}' | grep -oE ':[0-9]+$' | grep -oE '[0-9]+' | sort -u", 10*time.Second)
	if err != nil {
		return ports
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if p, err := strconv.Atoi(line); err == nil {
			ports[p] = true
		}
	}
	return ports
}

package handlers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"
	"cboard-go/internal/services/selfhost"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// ============================================================
// 自建节点（一键部署）相关 API
// ============================================================

// CreateSelfHostNode 管理员创建一个自建节点占位记录，返回安装命令。
// POST /admin/nodes/selfhost
func CreateSelfHostNode(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Protocol string `json:"protocol" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请填写节点名称和协议", err)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "节点名称不能为空", nil)
		return
	}
	if len([]rune(name)) > 100 {
		utils.ErrorResponse(c, http.StatusBadRequest, "节点名称过长", nil)
		return
	}

	db := database.GetDB()
	node, installID, _, err := selfhost.CreateRecord(db, name, req.Protocol, "", 0, "")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "创建自建节点失败: "+err.Error(), err)
		return
	}

	// 安装命令：一行 curl 管道到 bash（脚本由 /api/v1/agent/install.sh 动态生成）
	panelBase := resolvePanelBaseURL(c)
	installScriptURL := panelBase + "/api/v1/agent/install.sh?install_id=" + installID
	installCmd := "bash <(curl -fsSL '" + installScriptURL + "')"

	// 保存安装命令到节点记录（供管理端查看）
	db.Model(node).Update("install_cmd", installCmd)

	utils.CreateAuditLogSimple(c, "create_selfhost_node", "node", node.ID,
		"管理员操作: 创建自建节点 "+name+"（协议 "+selfhost.ProtocolDisplay(req.Protocol)+"）")

	utils.SuccessResponse(c, http.StatusCreated, "自建节点已创建，请复制安装命令到您的服务器执行", gin.H{
		"node":               node,
		"install_id":         installID,
		"install_cmd":        installCmd,
		"install_script_url": installScriptURL,
		"protocol_display":   selfhost.ProtocolDisplay(req.Protocol),
		"expires_at":         node.InstallExpiresAt,
	})
}

// GetSelfHostNodeStatus 查询自建节点的安装/在线状态。
// GET /admin/nodes/selfhost/:id
func GetSelfHostNodeStatus(c *gin.Context) {
	db := database.GetDB()
	var node models.CustomNode
	if err := db.First(&node, c.Param("id")).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "节点不存在", err)
		return
	}
	if !node.SelfHosted {
		utils.ErrorResponse(c, http.StatusBadRequest, "该节点不是自建节点", nil)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"id":                 node.ID,
		"name":               node.Name,
		"status":             node.Status,
		"self_hosted":        true,
		"protocol":           node.SelfHostProtocol,
		"protocol_display":   selfhost.ProtocolDisplay(node.SelfHostProtocol),
		"install_id":         node.InstallID,
		"install_cmd":        node.InstallCmd,
		"install_expires_at": node.InstallExpiresAt,
		"last_heartbeat_at":  node.LastHeartbeatAt,
		"is_active":          node.IsActive,
		"traffic_up":         node.TrafficUp,
		"traffic_down":       node.TrafficDown,
		"traffic_updated_at": node.TrafficUpdatedAt,
		"link":               extractNodeLink(node),
	})
}

// GetSelfHostNodes 管理端自建节点视图列表。
// GET /admin/nodes/selfhost
func GetSelfHostNodes(c *gin.Context) {
	db := database.GetDB()
	var nodes []models.CustomNode
	if err := db.Where("self_hosted = ?", true).Order("created_at DESC").Find(&nodes).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取自建节点列表失败", err)
		return
	}

	list := make([]gin.H, 0, len(nodes))
	for i := range nodes {
		n := nodes[i]
		list = append(list, gin.H{
			"id":                    n.ID,
			"name":                  n.Name,
			"status":                n.Status,
			"protocol":              n.SelfHostProtocol,
			"protocol_display":      selfhost.ProtocolDisplay(n.SelfHostProtocol),
			"install_id":            n.InstallID,
			"install_cmd":           n.InstallCmd,
			"install_expires_at":    n.InstallExpiresAt,
			"last_heartbeat_at":     n.LastHeartbeatAt,
			"is_active":             n.IsActive,
			"traffic_up":            n.TrafficUp,
			"traffic_down":          n.TrafficDown,
			"traffic_updated_at":    n.TrafficUpdatedAt,
			"traffic_limit_enabled": n.TrafficLimitEnabled,
			"traffic_limit_bytes":   n.TrafficLimitBytes,
			"domain":                n.Domain,
			"port":                  n.Port,
			"deploy_mode":           n.DeployMode,
			"core_type":             n.CoreType,
			"ssh_host":              n.SSHHost,
			"ssh_port":              n.SSHPort,
			"ssh_user":              n.SSHUser,
			"link":                  extractNodeLink(n),
			"created_at":            n.CreatedAt,
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"list":  list,
		"total": len(list),
	})
}

// GetSavedSelfHostVPS 管理端"已保存的 VPS"档案列表：按 ssh_host+ssh_port 去重，
// 聚合每台 VPS 的上次部署信息（域名、协议列表、节点数、密码状态），供二次搭建一键调用。
// GET /admin/custom-nodes/selfhost/saved-vps
func GetSavedSelfHostVPS(c *gin.Context) {
	db := database.GetDB()
	var nodes []models.CustomNode
	if err := db.Where("self_hosted = ? AND ssh_host != ? AND status != ?", true, "", selfhost.StatusCanceled).
		Order("created_at DESC").Find(&nodes).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取已保存VPS失败", err)
		return
	}

	type vpsProfile struct {
		Key          string   `json:"key"`
		SSHHost      string   `json:"ssh_host"`
		SSHPort      int      `json:"ssh_port"`
		SSHUser      string   `json:"ssh_user"`
		Domain       string   `json:"domain"`
		DeployMode   string   `json:"deploy_mode"`
		ProtocolList []string `json:"protocol_list"` // 上次部署的协议（多协议）
		Protocol     string   `json:"protocol"`      // 单协议时的协议
		NodeName     string   `json:"node_name"`     // 上次节点名（主节点）
		NodeID       uint     `json:"node_id"`       // 引用节点（有加密密码，供 saved_ssh_id 用）
		HasPassword  bool     `json:"has_password"`
		NodeCount    int      `json:"node_count"`  // 该 VPS 上的自建节点数
		MainNodeID   uint     `json:"main_node_id"` // 主节点 id（multi 模式）
		CreatedAt    *time.Time `json:"created_at,omitempty"`
	}

	profiles := make(map[string]*vpsProfile)
	order := make([]string, 0, len(nodes))
	for i := range nodes {
		n := nodes[i]
		key := fmt.Sprintf("%s:%d", n.SSHHost, n.SSHPort)
		p, exists := profiles[key]
		if !exists {
			p = &vpsProfile{
				Key:        key,
				SSHHost:    n.SSHHost,
				SSHPort:    n.SSHPort,
				SSHUser:    n.SSHUser,
				HasPassword: n.SSHPasswordEnc != "",
			}
			profiles[key] = p
			order = append(order, key)
		}
		p.NodeCount++
		// 优先记录主节点/最新节点的域名与协议信息
		if n.DeployMode != "" || (p.Domain == "" && n.Domain != "") {
			p.Domain = n.Domain
			p.DeployMode = n.DeployMode
			if n.ProtocolList != "" {
				p.ProtocolList = strings.Split(n.ProtocolList, ",")
			}
			p.Protocol = n.SelfHostProtocol
			p.NodeName = n.Name
			p.NodeID = n.ID
			if n.DeployMode != "" {
				p.MainNodeID = n.ID
			}
			p.CreatedAt = &n.CreatedAt
		}
		if p.NodeID == 0 {
			p.NodeID = n.ID
		}
		if n.SSHPasswordEnc != "" {
			p.HasPassword = true
		}
	}

	list := make([]*vpsProfile, 0, len(order))
	for _, key := range order {
		list = append(list, profiles[key])
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"list":  list,
		"total": len(list),
	})
}
// GET /api/v1/agent/install.sh?install_id=xxx
func AgentInstallScript(c *gin.Context) {
	installID := strings.TrimSpace(c.Query("install_id"))
	if installID == "" {
		c.String(http.StatusBadRequest, "missing install_id")
		return
	}

	db := database.GetDB()
	node, err := selfhost.GetByInstallID(db, installID)
	if err != nil {
		c.String(http.StatusNotFound, "install_id not found")
		return
	}
	if node.Status == selfhost.StatusCanceled || node.Status == selfhost.StatusExpired {
		c.String(http.StatusGone, "install token expired or canceled")
		return
	}
	if node.InstallExpiresAt != nil && time.Now().After(*node.InstallExpiresAt) {
		c.String(http.StatusGone, "install token expired")
		return
	}
	if node.Status == selfhost.StatusOnline && node.Config != "" {
		c.String(http.StatusGone, "node already installed")
		return
	}

	panelBase := resolvePanelBaseURL(c)
	script, err := selfhost.BuildInstallScript(selfhost.ScriptConfig{
		PanelBaseURL: panelBase,
		InstallID:    node.InstallID,
		Token:        node.InstallToken,
		Protocol:     node.SelfHostProtocol,
		NodeName:     node.Name,
		MirrorURLs:   selfhost.DefaultMirrorURLs(),
		GeneratedAt:  time.Now(),
	})
	if err != nil {
		c.String(http.StatusInternalServerError, "script generation failed: "+err.Error())
		return
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "no-store")
	c.String(http.StatusOK, "%s", script)
}

// AgentReport 处理 agent 安装完成后的回传（无鉴权，凭 install_id+token）。
// POST /api/v1/agent/report
func AgentReport(c *gin.Context) {
	var req struct {
		InstallID string `json:"install_id"`
		Token     string `json:"token"`
		Link      string `json:"link"`
		ServerIP  string `json:"server_ip"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}
	if req.InstallID == "" || req.Token == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "install_id 和 token 不能为空", nil)
		return
	}

	db := database.GetDB()
	node, err := selfhost.GetByInstallID(db, req.InstallID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "自建节点不存在", err)
		return
	}
	if err := selfhost.VerifyToken(node, req.Token); err != nil {
		utils.ErrorResponse(c, http.StatusForbidden, "令牌无效或已过期", err)
		return
	}

	fresh, err := selfhost.ReportNode(db, node, req.Link, req.ServerIP)
	if err != nil {
		if err == selfhost.ErrAlreadyReport {
			utils.ErrorResponse(c, http.StatusConflict, "该节点已完成安装回传", err)
			return
		}
		utils.ErrorResponse(c, http.StatusBadRequest, "节点回传失败: "+err.Error(), err)
		return
	}

	utils.CreateAuditLogSimple(c, "report_selfhost_node", "node", fresh.ID,
		"自建节点回传成功: "+fresh.Name+"（"+fresh.Protocol+"）")

	utils.SuccessResponse(c, http.StatusOK, "节点回传成功", gin.H{
		"node_id":   fresh.ID,
		"name":      fresh.Name,
		"type":      fresh.Protocol,
		"is_active": fresh.IsActive,
	})
}

// AgentReportBatch 处理多协议部署的批量回传：一次上报多个协议链接，
// 第一个协议更新主记录，其余协议各自新建节点记录（共享 SSH 信息与流量视图）。
// POST /api/v1/agent/report-batch
func AgentReportBatch(c *gin.Context) {
	var req struct {
		InstallID string   `json:"install_id"`
		Token     string   `json:"token"`
		Links     []string `json:"links"`
		ServerIP  string   `json:"server_ip"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}
	if req.InstallID == "" || req.Token == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "install_id 和 token 不能为空", nil)
		return
	}
	if len(req.Links) == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "links 不能为空", nil)
		return
	}

	db := database.GetDB()
	mainNode, err := selfhost.GetByInstallID(db, req.InstallID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "自建节点不存在", err)
		return
	}
	if err := selfhost.VerifyToken(mainNode, req.Token); err != nil {
		utils.ErrorResponse(c, http.StatusForbidden, "令牌无效或已过期", err)
		return
	}

	created := make([]gin.H, 0, len(req.Links))
	var firstErr error

	for i, link := range req.Links {
		link = strings.TrimSpace(link)
		if link == "" {
			continue
		}
		// links[0] 是主节点自己的链接：主节点未回传时激活它，已回传则跳过（避免重复）
		if i == 0 {
			if mainNode.Status != selfhost.StatusOnline {
				fresh, err := selfhost.ReportNode(db, mainNode, link, req.ServerIP)
				if err != nil {
					firstErr = err
					continue
				}
				created = append(created, gin.H{"node_id": fresh.ID, "name": fresh.Name, "type": fresh.Protocol})
			}
			continue
		}

		parsed, err := config_update.ParseNodeLink(link)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("链接 %d 解析失败: %w", i+1, err)
			}
			continue
		}
		// 按链接顺序匹配主节点 protocol_list，取完整协议标识（区分 vless-ws / vless-reality 等同类型协议）。
		// links[0] 是主节点自己的链接（协议=plist[0]），子节点 links[i] 对应 plist[i]。
		protoKey := ""
		if mainNode.ProtocolList != "" {
			plist := strings.Split(mainNode.ProtocolList, ",")
			if i >= 0 && i < len(plist) {
				protoKey = strings.TrimSpace(plist[i])
			}
		}
		if protoKey == "" {
			protoKey = parsed.Type
		}
		// 去重：同 install_id + 完整协议标识 已存在则跳过（避免重复回传创建重复子节点）
		var dupCount int64
		db.Model(&models.CustomNode{}).Where("install_id = ? AND self_host_protocol = ? AND self_hosted = ?", req.InstallID, protoKey, true).Count(&dupCount)
		if dupCount > 0 {
			if firstErr == nil {
				firstErr = fmt.Errorf("协议 %s 已存在（跳过重复回传）", protoKey)
			}
			continue
		}
		if req.ServerIP != "" && (parsed.Server == "" || parsed.Server == "127.0.0.1" || parsed.Server == "0.0.0.0") {
			parsed.Server = strings.TrimSpace(req.ServerIP)
		}
		cfgJSON, _ := json.Marshal(parsed)
		protoLabel := strings.ToUpper(protoKey)
		sub := models.CustomNode{
			Name:             fmt.Sprintf("%s-%s", mainNode.Name, protoLabel),
			DisplayName:      fmt.Sprintf("%s-%s", mainNode.Name, protoLabel),
			Protocol:         parsed.Type,
			Domain:           parsed.Server,
			Port:             parsed.Port,
			Config:           string(cfgJSON),
			Status:           selfhost.StatusOnline,
			IsActive:         true,
			Source:           "selfhost",
			SelfHosted:       true,
			SelfHostProtocol: protoKey,
			InstallID:        req.InstallID,
			SSHHost:          mainNode.SSHHost,
			SSHPort:          mainNode.SSHPort,
			SSHUser:          mainNode.SSHUser,
			SSHPasswordEnc:   mainNode.SSHPasswordEnc,
			// 子节点不标记 multi（主节点才标记），GetByInstallID 据此区分主/子
		}
		if err := db.Create(&sub).Error; err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		created = append(created, gin.H{"node_id": sub.ID, "name": sub.Name, "type": sub.Protocol})
	}

	if len(created) > 0 {
		db.Model(mainNode).Update("deploy_mode", "multi")
	}

	if len(created) == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "没有成功回传任何节点: "+errString(firstErr), firstErr)
		return
	}

	utils.CreateAuditLogSimple(c, "report_selfhost_nodes_batch", "custom_node", mainNode.ID,
		fmt.Sprintf("多协议节点回传成功: 共 %d 个", len(created)))

	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("回传成功 %d 个节点", len(created)), gin.H{
		"created": created,
		"total":   len(created),
	})
}

// AgentHeartbeat 处理 agent 心跳（无鉴权，凭 install_id+token）。
// POST /api/v1/agent/heartbeat
func AgentHeartbeat(c *gin.Context) {
	var req struct {
		InstallID  string `json:"install_id"`
		Token      string `json:"token"`
		TrafficUp  int64  `json:"traffic_up"`  // 上行流量增量（字节），可选
		TrafficDown int64 `json:"traffic_down"` // 下行流量增量（字节），可选
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}
	if req.InstallID == "" || req.Token == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "install_id 和 token 不能为空", nil)
		return
	}

	db := database.GetDB()
	node, err := selfhost.GetByInstallID(db, req.InstallID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "自建节点不存在", err)
		return
	}
	if err := selfhost.VerifyToken(node, req.Token); err != nil {
		utils.ErrorResponse(c, http.StatusForbidden, "令牌无效或已过期", err)
		return
	}
	if err := selfhost.Heartbeat(db, node, req.TrafficUp, req.TrafficDown); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "心跳更新失败: "+err.Error(), err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "ok", gin.H{"status": node.Status})
}

// resolvePanelBaseURL 从请求推导面板基础地址（安装脚本回传用）。
// 优先级：X-Forwarded-Proto/Host 头 > 请求自身。
func resolvePanelBaseURL(c *gin.Context) string {
	scheme := "https"
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		if strings.Contains(proto, "http") {
			scheme = strings.Split(proto, ",")[0]
		}
	} else if c.Request.TLS == nil {
		scheme = "http"
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return scheme + "://" + host
}

// extractNodeLink 从节点 Config 反解出节点链接（前端展示用），解析失败返回空串。
func extractNodeLink(node models.CustomNode) string {
	if node.Config == "" {
		return ""
	}
	link, err := proxyNodeToLink(node.Config)
	if err != nil {
		return ""
	}
	return link
}

// proxyNodeToLink 尝试把存储的 ProxyNode JSON 还原为可分享链接。
func proxyNodeToLink(cfgJSON string) (string, error) {
	var p struct {
		Name     string `json:"Name"`
		Type     string `json:"Type"`
		Server   string `json:"Server"`
		Port     int    `json:"Port"`
		UUID     string `json:"UUID,omitempty"`
		Password string `json:"Password,omitempty"`
		Cipher   string `json:"Cipher,omitempty"`
		Network  string `json:"Network,omitempty"`
		TLS      bool   `json:"TLS,omitempty"`
	}
	if err := json.Unmarshal([]byte(cfgJSON), &p); err != nil {
		return "", err
	}
	switch p.Type {
	case "vless":
		q := url.Values{}
		q.Set("type", p.Network)
		if p.Network == "ws" {
			if opts, ok := extractOptions(cfgJSON); ok == nil {
				if path := opts["path"]; path != "" {
					q.Set("path", path)
				}
				if host := opts["host"]; host != "" {
					q.Set("host", host)
				}
			}
		}
		if p.TLS {
			q.Set("security", "tls")
		} else {
			q.Set("security", "none")
		}
		return "vless://" + p.UUID + "@" + p.Server + ":" + itoa(p.Port) + "?" + q.Encode() + "#" + p.Name, nil
	case "vmess":
		// vmess 链接为 base64 JSON，从 Options 还原
		m := map[string]any{
			"v": 2, "ps": p.Name, "add": p.Server, "port": p.Port,
			"id": p.UUID, "aid": 0, "net": p.Network, "type": "none",
		}
		if opts, ok := extractOptions(cfgJSON); ok == nil {
			if path := opts["path"]; path != "" {
				m["path"] = path
			}
			if host := opts["host"]; host != "" {
				m["host"] = host
			}
		}
		if p.TLS {
			m["tls"] = "tls"
		}
		b, _ := json.Marshal(m)
		return "vmess://" + base64.StdEncoding.EncodeToString(b), nil
	case "trojan":
		q := url.Values{}
		if p.Network == "ws" {
			q.Set("type", "ws")
			if opts, ok := extractOptions(cfgJSON); ok == nil {
				if path := opts["path"]; path != "" {
					q.Set("path", path)
				}
			}
		}
		if p.TLS {
			q.Set("security", "tls")
		}
		return "trojan://" + p.Password + "@" + p.Server + ":" + itoa(p.Port) + "?" + q.Encode() + "#" + p.Name, nil
	case "ss":
		userinfo := p.Cipher + ":" + p.Password
		return "ss://" + base64.StdEncoding.EncodeToString([]byte(userinfo)) + "@" + p.Server + ":" + itoa(p.Port) + "#" + p.Name, nil
	default:
		return "", errUnsupported
	}
}

var errUnsupported = errors.New("unsupported node type")

func itoa(i int) string {
	return strconv.Itoa(i)
}

// extractOptions 从 ProxyNode JSON 中取出 Options 内联字段。
func extractOptions(cfgJSON string) (map[string]string, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(cfgJSON), &raw); err != nil {
		return nil, err
	}
	out := map[string]string{}
	if opts, ok := raw["Options"].(map[string]any); ok {
		for k, v := range opts {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
	}
	return out, nil
}

// errString 将 error 安全转为字符串（nil → 空串）。
func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

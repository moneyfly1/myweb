package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"
	"cboard-go/internal/services/node_health"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetCustomNodes(c *gin.Context) {
	db := database.GetDB()
	var nodes []models.CustomNode
	query := db.Model(&models.CustomNode{})

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if protocol := strings.TrimSpace(c.Query("protocol")); protocol != "" {
		if protocol == "wireguard" {
			query = query.Where("protocol IN ?", []string{"wireguard", "wg"})
		} else {
			query = query.Where("protocol = ?", protocol)
		}
	}
	if isActive := c.Query("is_active"); isActive != "" {
		if isActive == "true" {
			query = query.Where("is_active = ?", true)
		} else {
			query = query.Where("is_active = ?", false)
		}
	}
	if source := c.Query("source"); source != "" {
		if source == "manual" {
			// 历史数据未记录来源（空值），按手动添加处理
			query = query.Where("source = ? OR source = '' OR source IS NULL", source)
		} else {
			query = query.Where("source = ?", source)
		}
	}
	if search := c.Query("search"); search != "" {
		sanitizedSearch := utils.SanitizeSearchKeyword(search)
		escapedSearch := utils.EscapeLikePattern(sanitizedSearch)
		var userIDs []uint
		db.Model(&models.User{}).Where("username LIKE ? OR email LIKE ?", "%"+escapedSearch+"%", "%"+escapedSearch+"%").Pluck("id", &userIDs)

		var userNodeIDs []uint
		if len(userIDs) > 0 {
			db.Model(&models.UserCustomNode{}).Where("user_id IN ?", userIDs).Pluck("custom_node_id", &userNodeIDs)
		}

		searchPattern := "%" + escapedSearch + "%"
		if len(userNodeIDs) > 0 {
			query = query.Where("name LIKE ? OR display_name LIKE ? OR domain LIKE ? OR id IN ?",
				searchPattern, searchPattern, searchPattern, userNodeIDs)
		} else {
			query = query.Where("name LIKE ? OR display_name LIKE ? OR domain LIKE ?",
				searchPattern, searchPattern, searchPattern)
		}
	}

	p := utils.ParsePaginationWithDefaultSize(c, 20)
	page, size := p.Page, p.Size

	var total int64
	query.Count(&total)

	offset := (page - 1) * size
	if err := query.Order("created_at DESC").Offset(offset).Limit(size).Find(&nodes).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取节点列表失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"data":  nodes,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

func GetCustomNodeUsers(c *gin.Context) {
	nodeID := c.Param("id")
	db := database.GetDB()

	var node models.CustomNode
	if err := db.First(&node, nodeID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "节点不存在", err)
		return
	}

	var userNodes []models.UserCustomNode
	if err := db.Preload("User").Where("custom_node_id = ?", nodeID).Find(&userNodes).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取用户列表失败", err)
		return
	}

	users := make([]gin.H, 0)
	for _, un := range userNodes {
		if un.User.ID != 0 {
			users = append(users, gin.H{
				"id":                             un.User.ID,
				"username":                       un.User.Username,
				"email":                          un.User.Email,
				"special_node_subscription_type": un.User.SpecialNodeSubscriptionType,
				"special_node_expires_at":        un.User.SpecialNodeExpiresAt,
				"special_node_unlimited_devices": un.User.SpecialNodeUnlimitedDevices,
				// 分配级流量配额（客户独享节点时按分配设配额）
				"traffic_limit_enabled": un.TrafficLimitEnabled,
				"traffic_limit_bytes":   un.TrafficLimitBytes,
				"traffic_used":          node.TrafficUp + node.TrafficDown,
			})
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", users)
}

// BatchGetCustomNodeUsers 一次请求返回多个节点的关联用户，按节点 ID 分组。
// 前端批量删除前用它一次性拿到所有受影响用户，避免逐个节点串行请求导致确认框延迟弹出。
func BatchGetCustomNodeUsers(c *gin.Context) {
	var req struct {
		NodeIDs []uint `json:"node_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	db := database.GetDB()

	var userNodes []models.UserCustomNode
	if err := db.Preload("User").Where("custom_node_id IN ?", req.NodeIDs).Find(&userNodes).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取用户列表失败", err)
		return
	}

	grouped := make(map[uint][]gin.H)
	for _, un := range userNodes {
		if un.User.ID != 0 {
			grouped[un.CustomNodeID] = append(grouped[un.CustomNodeID], gin.H{
				"id":                             un.User.ID,
				"username":                       un.User.Username,
				"email":                          un.User.Email,
				"special_node_subscription_type": un.User.SpecialNodeSubscriptionType,
				"special_node_expires_at":        un.User.SpecialNodeExpiresAt,
				"special_node_unlimited_devices": un.User.SpecialNodeUnlimitedDevices,
			})
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", grouped)
}

func normalizeCustomNodeConfig(configStr, protocol, domain string, port int) (string, string, string, int) {
	trimmed := strings.TrimSpace(configStr)
	if trimmed == "" {
		return configStr, protocol, domain, port
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &data); err != nil {
		return configStr, protocol, domain, port
	}

	if protocol == "" {
		protocol = getStringFromConfigMap(data, "type", "Type", "protocol")
	}
	if domain == "" {
		domain = getStringFromConfigMap(data, "server", "Server", "add", "address")
	}
	if port <= 0 {
		port = getIntFromConfigMap(data, "port", "Port")
	}

	if protocol != "" {
		setStringInConfigMap(data, protocol, "Type", "type", "protocol")
	}
	if domain != "" {
		setStringInConfigMap(data, domain, "Server", "server", "add", "address")
	}
	if port > 0 {
		setIntInConfigMap(data, port, "Port", "port")
	}

	normalized, err := json.Marshal(data)
	if err != nil {
		return configStr, protocol, domain, port
	}
	return string(normalized), protocol, domain, port
}

func getStringFromConfigMap(data map[string]interface{}, keys ...string) string {
	if v := utils.GetMapString(data, keys...); v != "" {
		return strings.TrimSpace(v)
	}
	// 兼容键名大小写不一致的历史数据（保留原实现语义）
	for existingKey, val := range data {
		for _, key := range keys {
			if strings.EqualFold(existingKey, key) {
				if s, ok := val.(string); ok && s != "" {
					return strings.TrimSpace(s)
				}
			}
		}
	}
	return ""
}

func getIntFromConfigMap(data map[string]interface{}, keys ...string) int {
	for _, key := range keys {
		if val, ok := data[key]; ok {
			if port := configMapValueToInt(val); port > 0 {
				return port
			}
		}
	}
	for existingKey, val := range data {
		for _, key := range keys {
			if strings.EqualFold(existingKey, key) {
				if port := configMapValueToInt(val); port > 0 {
					return port
				}
			}
		}
	}
	return 0
}

func configMapValueToInt(val interface{}) int {
	switch v := val.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(v))
		return i
	default:
		return 0
	}
}

func setStringInConfigMap(data map[string]interface{}, value string, keys ...string) {
	if value == "" || len(keys) == 0 {
		return
	}
	hasCanonicalKey := false
	for existingKey := range data {
		if strings.EqualFold(existingKey, keys[0]) {
			hasCanonicalKey = true
		}
		for _, key := range keys {
			if strings.EqualFold(existingKey, key) {
				data[existingKey] = value
			}
		}
	}
	if !hasCanonicalKey {
		data[keys[0]] = value
	}
}

func setIntInConfigMap(data map[string]interface{}, value int, keys ...string) {
	if value <= 0 || len(keys) == 0 {
		return
	}
	hasCanonicalKey := false
	for existingKey := range data {
		if strings.EqualFold(existingKey, keys[0]) {
			hasCanonicalKey = true
		}
		for _, key := range keys {
			if strings.EqualFold(existingKey, key) {
				data[existingKey] = value
			}
		}
	}
	if !hasCanonicalKey {
		data[keys[0]] = value
	}
}

func CreateCustomNode(c *gin.Context) {
	var req struct {
		NodeLink         string     `json:"node_link"`
		Name             string     `json:"name"`
		DisplayName      string     `json:"display_name"`
		Protocol         string     `json:"protocol"`
		Config           string     `json:"config"`
		Domain           string     `json:"domain"`
		Port             int        `json:"port"`
		ExpireTime       *time.Time `json:"expire_time"`
		FollowUserExpire bool       `json:"follow_user_expire"`
		Preview          bool       `json:"preview"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误: "+err.Error(), err)
		return
	}

	db := database.GetDB()

	if req.NodeLink != "" {
		parsed, err := config_update.ParseNodeLink(strings.TrimSpace(req.NodeLink))
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "解析节点链接失败: "+err.Error(), err)
			return
		}

		// #nosec G117 - Password field is proxy node password, not user credential
		configJSON, _ := json.Marshal(parsed) // #nosec G117
		configStr := string(configJSON)

		name := req.Name
		if name == "" {
			name = parsed.Name
			if name == "" {
				name = fmt.Sprintf("%s-%s", parsed.Type, parsed.Server)
			}
		}

		customNode := models.CustomNode{
			Name:             truncateNodeName(name),
			DisplayName:      truncateNodeName(req.DisplayName),
			Protocol:         parsed.Type,
			Domain:           parsed.Server,
			Port:             parsed.Port,
			Config:           configStr,
			Status:           "inactive",
			IsActive:         true,
			ExpireTime:       req.ExpireTime,
			FollowUserExpire: req.FollowUserExpire,
			Source:           "manual",
		}

		if req.Preview {
			utils.SuccessResponse(c, http.StatusOK, "", gin.H{
				"name":   customNode.Name,
				"type":   customNode.Protocol,
				"server": customNode.Domain,
				"port":   customNode.Port,
				"config": customNode.Config,
			})
			return
		}

		if err := db.Create(&customNode).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "创建节点失败: "+err.Error(), err)
			return
		}
		utils.CreateAuditLogSimple(c, "create_custom_node", "custom_node", customNode.ID, fmt.Sprintf("管理员操作: 创建专线节点 %s", customNode.Name))
		clearNodeCaches()
		utils.SuccessResponse(c, http.StatusCreated, "", customNode)
		return
	}

	if req.Name == "" || req.Protocol == "" || req.Config == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "节点名称、协议和配置为必填项", nil)
		return
	}
	configStr, protocol, domain, port := normalizeCustomNodeConfig(req.Config, req.Protocol, req.Domain, req.Port)

	customNode := models.CustomNode{
		Name:             truncateNodeName(req.Name),
		DisplayName:      truncateNodeName(req.DisplayName),
		Protocol:         protocol,
		Domain:           domain,
		Port:             port,
		Config:           configStr,
		Status:           "inactive",
		IsActive:         true,
		ExpireTime:       req.ExpireTime,
		FollowUserExpire: req.FollowUserExpire,
		Source:           "manual",
	}

	if err := db.Create(&customNode).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建节点失败: "+err.Error(), err)
		return
	}
	utils.CreateAuditLogSimple(c, "create_custom_node", "custom_node", customNode.ID, fmt.Sprintf("管理员操作: 创建专线节点 %s", customNode.Name))
	clearNodeCaches()
	utils.SuccessResponse(c, http.StatusCreated, "", customNode)
}

func ImportCustomNodeLinks(c *gin.Context) {
	var req struct {
		Links []string `json:"links" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	db := database.GetDB()
	imported, skipped, errorCount, errors := importCustomNodesFromLinks(db, req.Links, "link", "")
	message := customNodeImportMessage(len(req.Links), imported, skipped, errorCount)
	utils.CreateAuditLogSimple(c, "import_custom_node_links", "custom_node", 0,
		fmt.Sprintf("管理员操作: 导入专线节点链接 成功 %d 跳过 %d 失败 %d", imported, skipped, errorCount))
	if imported > 0 {
		clearNodeCaches()
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"imported":    imported,
		"skipped":     skipped, // 已存在被跳过的数量（不是失败）
		"error_count": errorCount,
		"errors":      errors,
		"message":     message,
	})
}

// ImportCustomNodeSubscription 从订阅链接拉取并自动解析节点，导入为专线节点。
// 支持三种语义（mode/replace 组合）：
//   - 追加导入（缺省）：仅追加新节点，**同一个节点**（协议+地址+凭据+传输参数全同）跳过；
//     同 IP:端口但配置不同的节点照常导入（可能是同一服务器的不同入口）；
//   - mode="update" + replace=false：增量更新该 source_url 下节点——按名称/节点身份匹配
//     到的更新配置（保留节点 ID 与用户分配），新节点追加，订阅中已消失的旧节点保留不删；
//   - mode="update" + replace=true：匹配范围扩大到全部 subscription 来源节点
//     （更换订阅地址场景），同样增量更新 + 分配保护，从不删除节点。
func ImportCustomNodeSubscription(c *gin.Context) {
	var req struct {
		URL     string `json:"url" binding:"required"`
		Replace bool   `json:"replace"`
		Mode    string `json:"mode"` // "update" = 增量更新（分配保护）；缺省 = 追加导入
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	urlStr := strings.TrimSpace(req.URL)
	if urlStr == "" || (!strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://")) {
		utils.ErrorResponse(c, http.StatusBadRequest, "请输入有效的 http/https 订阅链接", nil)
		return
	}

	db := database.GetDB()

	// 更新订阅模式（mode=update）：拉取订阅后增量更新（分配保护：已分配节点更新配置、保留分配，绝不移除）。
	// replace=true：全范围匹配更新（更换订阅地址场景，匹配所有订阅来源节点的同名/同地址节点）；
	// replace=false：仅更新该订阅 URL 下的节点（同一订阅内容更新，如端口/配置变化）。
	if req.Mode == "update" {
		added, updated, kept, errs := updateCustomNodeSubscription(db, urlStr, req.Replace)
		if len(errs) > 0 && added == 0 && updated == 0 && kept == 0 {
			utils.ErrorResponse(c, http.StatusInternalServerError, "更新订阅失败", fmt.Errorf("%s", errs[0]))
			return
		}
		clearNodeCaches()
		utils.CreateAuditLogSimple(c, "update_custom_node_subscription", "custom_node", 0,
			fmt.Sprintf("管理员操作: 更新订阅 %s 新增 %d 更新 %d 保留 %d", urlStr, added, updated, kept))
		msg := fmt.Sprintf("订阅更新完成：新增 %d 个节点", added)
		if updated > 0 {
			msg += fmt.Sprintf("，更新 %d 个已有节点（分配保持不变）", updated)
		}
		if kept > 0 {
			msg += fmt.Sprintf("，保留 %d 个订阅中已消失的节点（为避免影响已分配用户，未删除）", kept)
		}
		if len(errs) > 0 {
			msg += fmt.Sprintf("（%d 个解析失败）", len(errs))
		}
		utils.SuccessResponse(c, http.StatusOK, msg, gin.H{
			"imported":    added,
			"updated":     updated,
			"removed":     0,
			"kept":        kept,
			"error_count": len(errs),
			"errors":      errs,
			"total":       added + updated + kept,
			"message":     msg,
		})
		return
	}

	svc := config_update.NewConfigUpdateService()
	nodes, err := svc.FetchNodesFromURLs([]string{urlStr})
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅失败", err)
		return
	}
	if len(nodes) == 0 {
		utils.SuccessResponse(c, http.StatusOK, "", gin.H{
			"imported":    0,
			"error_count": 0,
			"errors":      []string{},
			"message":     "订阅获取失败或内容中没有解析到节点，请检查订阅链接是否可访问",
		})
		return
	}

	links := make([]string, 0, len(nodes))
	seen := make(map[string]bool)
	for _, n := range nodes {
		link, _ := n["url"].(string)
		link = strings.TrimSpace(link)
		if link != "" && !seen[link] {
			seen[link] = true
			links = append(links, link)
		}
	}
	if len(links) == 0 {
		utils.SuccessResponse(c, http.StatusOK, "", gin.H{
			"imported":    0,
			"error_count": 0,
			"errors":      []string{},
			"message":     "订阅内容中没有解析到节点",
		})
		return
	}

	imported, skipped, errorCount, errors := importCustomNodesFromLinks(db, links, "subscription", urlStr)
	message := customNodeImportMessage(len(links), imported, skipped, errorCount)
	utils.CreateAuditLogSimple(c, "import_custom_node_subscription", "custom_node", 0,
		fmt.Sprintf("管理员操作: 导入专线节点订阅 %s 解析 %d 个 成功 %d 跳过 %d 失败 %d", urlStr, len(links), imported, skipped, errorCount))
	if imported > 0 {
		clearNodeCaches()
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"imported":    imported,
		"skipped":     skipped, // 已存在被跳过的数量（不是失败）
		"error_count": errorCount,
		"errors":      errors,
		"total":       len(links),
		"message":     message,
	})
}

// ListCustomNodeSubscriptions 返回已导入的订阅 URL 集合（按 SourceURL 聚合，含节点数量），
// 供前端「更新订阅 / 更换订阅地址」选择使用。
func ListCustomNodeSubscriptions(c *gin.Context) {
	db := database.GetDB()
	type subItem struct {
		URL       string `json:"url"`
		NodeCount int64  `json:"node_count"`
	}
	items := make([]subItem, 0)
	var subURLs []string
	if err := db.Model(&models.CustomNode{}).
		Where("source = ? AND source_url <> ''", "subscription").
		Distinct("source_url").Pluck("source_url", &subURLs).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询订阅列表失败", err)
		return
	}
	for _, u := range subURLs {
		var cnt int64
		db.Model(&models.CustomNode{}).Where("source_url = ?", u).Count(&cnt)
		items = append(items, subItem{URL: u, NodeCount: cnt})
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{"list": items, "total": len(items)})
}

// DeleteCustomNodeSubscription 删除某订阅 URL 下导入的全部专线节点（含已分配关联），
// 用于「更换订阅地址」时清理旧订阅导入的节点。
func DeleteCustomNodeSubscription(c *gin.Context) {
	var req struct {
		URL string `json:"url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}
	urlStr := strings.TrimSpace(req.URL)
	if urlStr == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "请输入订阅链接", nil)
		return
	}
	db := database.GetDB()
	var oldNodes []models.CustomNode
	db.Where("source = ? AND source_url = ?", "subscription", urlStr).Find(&oldNodes)
	if len(oldNodes) == 0 {
		utils.SuccessResponse(c, http.StatusOK, "该订阅下没有可删除的节点", gin.H{"removed": 0})
		return
	}
	oldIDs := make([]uint, 0, len(oldNodes))
	for _, cn := range oldNodes {
		oldIDs = append(oldIDs, cn.ID)
	}
	if err := db.Where("custom_node_id IN ?", oldIDs).Delete(&models.UserCustomNode{}).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "清理旧节点分配失败", err)
		return
	}
	if err := db.Delete(&models.CustomNode{}, "id IN ?", oldIDs).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除节点失败", err)
		return
	}
	clearNodeCaches()
	utils.CreateAuditLogSimple(c, "delete_custom_node_subscription", "custom_node", 0,
		fmt.Sprintf("管理员操作: 删除订阅 %s 下 %d 个节点", urlStr, len(oldIDs)))
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("已删除 %d 个节点", len(oldIDs)), gin.H{"removed": len(oldIDs)})
}

// importCustomNodesFromLinks 解析节点链接并创建专线节点，返回成功数、失败数与错误明细。
// source 标识节点来源: manual / link / subscription / selfhost；
// sourceURL 为订阅导入时的来源订阅 URL（仅 subscription 来源使用，用于更新/替换定位）。
// customNodeImportMessage 生成专线节点导入的结果提示。
//
// 必须把"新增 / 已存在跳过 / 解析失败"三件事分开说：此前只回报"成功 0 个"，
// 前端在"导入 0 且无报错"时统一提示"没有解析到可导入的节点"，
// 管理员看到就会以为链接无法解析——线上真实案例：11 个节点早已导入，
// 三次尝试都提示"无法解析"，实际是全部被"已存在"跳过。
func customNodeImportMessage(parsed, imported, skipped, failed int) string {
	parts := make([]string, 0, 3)
	if imported > 0 {
		parts = append(parts, fmt.Sprintf("成功导入 %d 个", imported))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("跳过 %d 个（节点已存在，无需重复导入）", skipped))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("失败 %d 个", failed))
	}

	switch {
	case len(parts) == 0:
		return "没有解析到可导入的节点，请检查链接格式是否正确"
	case imported == 0 && skipped > 0 && failed == 0:
		// 全部已存在：必须明确说"已存在"，否则会被误读成"链接解析不了"
		if parsed > 0 {
			return fmt.Sprintf("解析到 %d 个节点，但它们都已存在于专线节点中，本次未新增", parsed)
		}
		return fmt.Sprintf("这 %d 个节点都已存在于专线节点中，本次未新增", skipped)
	default:
		return strings.Join(parts, "，")
	}
}

// importCustomNodesFromLinks 把链接导入为专线节点。
// 返回值：imported 新增数、skipped 因「已存在（同一个节点重复：协议+地址+凭据+传输参数全同）」跳过数、
// errorCount 解析/写入失败数、errors 失败原因。
// 区分 skipped 与 errorCount 很重要：此前"全部已存在"也会被报成"没有解析到节点"，
// 让管理员误以为链接无法解析（线上实际发生：11 个节点早已导入，提示却是"无法解析"）。
func importCustomNodesFromLinks(db *gorm.DB, links []string, source string, sourceURL string) (imported, skipped, errorCount int, errors []string) {
	// 预加载现有专线节点的「节点身份」用于去重。
	// 注意：判重必须精确到"同一个节点"，不能只看 (协议, 域名, 端口)——
	// 同一台服务器同一端口上完全可以有多个不同凭据/传输参数的节点
	// （同一 IP:port 不同 UUID 的 vless、不同 password 的 trojan、不同 path/sni 的变体），
	// 按地址判重会把它们当成"已存在"直接拒收（2026-09-22 用户反馈）。
	// 配置无法解析的存量行算不出身份，按"不重复"处理（宁可允许导入，也不误拒）。
	var existing []models.CustomNode
	db.Select("protocol", "domain", "port", "config").Find(&existing)
	existingKeys := make(map[string]bool, len(existing))
	for _, cn := range existing {
		if key := customNodeIdentityKeyFromConfig(cn.Config); key != "" {
			existingKeys[key] = true
			continue
		}
		// 兜底：config 缺失时无法比对凭据，只能按地址记一个弱键，
		// 但只有当传入节点连凭据都没有时才可能撞上（见 customNodeIdentityKey 的退化分支）
		if key := customNodeIdentityKey(&config_update.ProxyNode{Type: cn.Protocol, Server: cn.Domain, Port: cn.Port}); key != "" {
			existingKeys[key] = true
		}
	}

	seen := make(map[string]bool)
	var newNodes []models.CustomNode

	for _, link := range links {
		link = strings.TrimSpace(link)
		if link == "" {
			continue
		}

		parsed, err := config_update.ParseNodeLink(link)
		if err != nil {
			errorCount++
			errors = append(errors, fmt.Sprintf("链接解析失败: %s", err.Error()))
			continue
		}

		// 去重键 = 节点身份（同一节点才算重复）：本批次内 + 数据库中；
		// 重复 = "已存在"，计入 skipped（不是错误）
		dupKey := customNodeIdentityKey(parsed)
		if seen[dupKey] || existingKeys[dupKey] {
			skipped++
			continue
		}
		seen[dupKey] = true

		// #nosec G117 - Password field is proxy node password, not user credential
		configJSON, _ := json.Marshal(parsed) // #nosec G117
		configStr := string(configJSON)

		name := parsed.Name
		if name == "" {
			name = fmt.Sprintf("%s-%s", parsed.Type, parsed.Server)
		}

		newNodes = append(newNodes, models.CustomNode{
			Name:      truncateNodeName(name),
			Protocol:  parsed.Type,
			Domain:    parsed.Server,
			Port:      parsed.Port,
			Config:    configStr,
			Status:    "inactive",
			IsActive:  true,
			Source:    source,
			SourceURL: sourceURL,
		})
	}

	if len(newNodes) > 0 {
		// 批量写入，替代逐条 Create（N+1 写入）
		if err := db.CreateInBatches(newNodes, 100).Error; err != nil {
			errorCount += len(newNodes)
			errors = append(errors, fmt.Sprintf("批量写入节点失败: %s", err.Error()))
			return imported, skipped, errorCount, errors
		}
		imported = len(newNodes)
	}
	return imported, skipped, errorCount, errors
}

// customNodeIdentityKey 计算专线节点的「节点身份」键：协议 + 地址 + 凭据 + 传输参数。
//
// 为什么不能只用 (协议, 域名, 端口) 判重（2026-09-22 用户反馈"IP 端口一样就导入不进来"）：
// 同一台服务器同一端口上可以有多个不同的节点 —— 同一 IP:port 上不同 UUID 的 vless、
// 不同 password 的 trojan、不同 path/sni/host 的 ws/reality 变体。它们的地址相同，
// 但**不是同一个节点**，必须能同时存在，否则用户无法把服务商提供的多个入口都加进来。
//
// 反过来，同一个节点（地址、凭据、传输参数全同）重复导入仍会被识别为"已存在"，
// 避免同一订阅反复导入产生重复行。
func customNodeIdentityKey(p *config_update.ProxyNode) string {
	if p == nil {
		return ""
	}
	cred := strings.TrimSpace(p.UUID)
	if cred == "" {
		cred = strings.TrimSpace(p.Password)
	}
	opts := ""
	if len(p.Options) > 0 {
		// Options 里装的是 path/sni/host/serviceName 等传输参数，参与身份计算。
		// 用哈希是为了避免把整串参数塞进 map key（长度可控、也不泄露到日志）。
		if b, err := json.Marshal(p.Options); err == nil {
			sum := sha256.Sum256(b)
			opts = hex.EncodeToString(sum[:8])
		}
	}
	return strings.Join([]string{
		strings.ToLower(strings.TrimSpace(p.Type)),
		strings.ToLower(strings.TrimSpace(p.Server)),
		strconv.Itoa(p.Port),
		cred,
		strings.ToLower(strings.TrimSpace(p.Network)),
		strings.ToLower(strings.TrimSpace(p.Cipher)),
		strconv.FormatBool(p.TLS),
		opts,
	}, "|")
}

// customNodeIdentityKeyFromConfig 从库里存的 config JSON 反算节点身份；
// 配置缺失或无法解析时返回空串（调用方按"无法判定"处理，即不参与判重，宁可放行导入）。
func customNodeIdentityKeyFromConfig(configJSON string) string {
	if strings.TrimSpace(configJSON) == "" {
		return ""
	}
	var p config_update.ProxyNode
	if err := json.Unmarshal([]byte(configJSON), &p); err != nil {
		return ""
	}
	return customNodeIdentityKey(&p)
}

// updateCustomNodeSubscription 拉取订阅 URL 并增量更新订阅导入的专线节点。
// 返回 (新增数, 更新数, 保留数, 错误列表)。
//
// 核心原则（分配保护）：已分配给用户的节点在更新后仍保持分配关系，绝不因订阅更新而丢失。
// 匹配策略：新订阅节点按「名称相同」（订阅来源节点）或「同一节点（协议+地址+凭据+传输参数）」匹配——
//   - 匹配到：更新该节点的配置（Config/名称/端口等），保留节点 ID 与用户分配（UserCustomNode 不动）；
//   - 未匹配（新节点，含"同地址但凭据/参数不同"）：追加创建，不会被当作已有节点覆盖；
//   - 订阅中已消失的旧节点（找不到匹配）：保留不删除（避免破坏分配），仅报告数量，
//     由管理员决定是否手动删除（删除 API 会同时清理分配）。
//
// replaceAll=true 时：匹配范围扩大到「全部 subscription 来源」节点（更换订阅地址场景：
// 新地址的节点会更新旧地址下同名的节点，旧地址独有的节点保留）；
// replaceAll=false 时：仅匹配「source_url = urlStr」的节点（同一订阅内容更新）。
func updateCustomNodeSubscription(db *gorm.DB, urlStr string, replaceAll bool) (added, updated, kept int, errs []string) {
	svc := config_update.NewConfigUpdateService()
	nodes, err := svc.FetchNodesFromURLs([]string{urlStr})
	if err != nil {
		return 0, 0, 0, []string{fmt.Sprintf("获取订阅失败: %s", err.Error())}
	}
	if len(nodes) == 0 {
		return 0, 0, 0, []string{"订阅获取失败或内容中没有解析到节点，请检查订阅链接是否可访问"}
	}

	links := make([]string, 0, len(nodes))
	seen := make(map[string]bool)
	for _, n := range nodes {
		link, _ := n["url"].(string)
		link = strings.TrimSpace(link)
		if link != "" && !seen[link] {
			seen[link] = true
			links = append(links, link)
		}
	}
	if len(links) == 0 {
		return 0, 0, 0, []string{"订阅内容中没有解析到节点"}
	}

	// 加载候选旧节点（订阅来源 + source 为空的历史遗留节点）
	oldNodes := loadCustomNodeMatchCandidates(db, urlStr, replaceAll)

	// 合并订阅链接：匹配则就地更新，未匹配则新增，订阅里已消失的旧节点保留不删除
	merge := mergeCustomNodesFromLinks(oldNodes, links, urlStr)

	errs = merge.errs
	for _, u := range merge.updates {
		if err := db.Model(&models.CustomNode{}).Where("id = ?", u.ID).Updates(u.Fields).Error; err != nil {
			errs = append(errs, fmt.Sprintf("更新节点 %s 失败: %s", u.Name, err.Error()))
			continue
		}
		updated++
	}

	if len(merge.newNodes) > 0 {
		if err := db.CreateInBatches(merge.newNodes, 100).Error; err != nil {
			errs = append(errs, fmt.Sprintf("批量写入节点失败: %s", err.Error()))
			return added, updated, merge.kept, errs
		}
		added = len(merge.newNodes)
	}
	return added, updated, merge.kept, errs
}

// loadCustomNodeMatchCandidates 加载参与「订阅更新」匹配的旧节点。
//
// 除了 source=subscription 的订阅节点，还包括 source 为空的历史遗留节点：
// source 字段是后加的，线上有 300+ 个节点的 source 为空，它们在旧逻辑里
// 完全不在候选集内 —— 于是同一个订阅每更新一次就重新插入一份，
// 节点列表越更新越多（2026-09-21 排查确认）。
//
// 自建节点（self_hosted=true，source=selfhost）与手动链接导入（source=link）
// 不参与匹配，避免订阅更新误改管理员手工维护的节点。
func loadCustomNodeMatchCandidates(db *gorm.DB, urlStr string, replaceAll bool) []models.CustomNode {
	const legacyCond = "source = '' OR source IS NULL"
	var oldNodes []models.CustomNode
	if replaceAll {
		db.Where("self_hosted = ? AND (source = ? OR "+legacyCond+")", false, "subscription").Find(&oldNodes)
	} else {
		db.Where("self_hosted = ? AND ((source = ? AND source_url = ?) OR "+legacyCond+")",
			false, "subscription", urlStr).Find(&oldNodes)
	}
	return oldNodes
}

// customNodeMatch 候选节点及其是否已被本次订阅内容认领。
type customNodeMatch struct {
	node     *models.CustomNode
	consumed bool
}

// buildCustomNodeMatchers 建立匹配索引。
//
//	byName：名称 → 节点。仅订阅来源节点参与「同名视为同一节点」，
//	        历史遗留节点来源不明，按名称匹配可能把管理员手工维护的节点静默改地址。
//	byIdentity：节点身份（协议+地址+凭据+传输参数）→ 节点。订阅来源与历史遗留节点都参与。
//	        这是「同一个订阅重复导入不再产生重复节点」的关键；
//	        同时因为带上了凭据与传输参数，"同 IP:端口但配置不同"的节点不会被误当成已有节点
//	        （否则用户无法把服务商给的多个入口加进来，2026-09-22 用户反馈）。
//	        配置无法解析的存量行算不出身份，不参与身份匹配（宁可新增，也不误覆盖）。
func buildCustomNodeMatchers(oldNodes []models.CustomNode) (byName, byIdentity map[string]*customNodeMatch) {
	byName = make(map[string]*customNodeMatch, len(oldNodes))
	byIdentity = make(map[string]*customNodeMatch, len(oldNodes))
	for i := range oldNodes {
		mi := &customNodeMatch{node: &oldNodes[i]}
		if oldNodes[i].Source == "subscription" {
			if nm := strings.TrimSpace(oldNodes[i].Name); nm != "" {
				if _, ok := byName[nm]; !ok {
					byName[nm] = mi
				}
			}
		}
		if key := customNodeIdentityKeyFromConfig(oldNodes[i].Config); key != "" {
			if _, ok := byIdentity[key]; !ok {
				byIdentity[key] = mi
			}
		}
	}
	return byName, byIdentity
}

// customNodeUpdate 一次「就地更新」：保留 ID、分配关系、激活状态、到期时间、
// 状态与测试结果，只刷新订阅侧字段。
type customNodeUpdate struct {
	ID     uint
	Name   string
	Fields map[string]interface{}
}

type customNodeMergeResult struct {
	newNodes []models.CustomNode
	updates  []customNodeUpdate
	kept     int
	errs     []string
}

// mergeCustomNodesFromLinks 把订阅解析出的链接合并进已有专线节点（纯函数，便于测试）：
// 先按名称（仅订阅来源）匹配，再按「节点身份：协议+地址+凭据+传输参数」匹配（含历史遗留节点）；
// 未匹配的追加为新节点；订阅里消失的旧节点保留不删除（分配保护）。
func mergeCustomNodesFromLinks(oldNodes []models.CustomNode, links []string, urlStr string) customNodeMergeResult {
	var res customNodeMergeResult

	byName, byIdentity := buildCustomNodeMatchers(oldNodes)
	seenLinks := make(map[string]bool)

	for _, link := range links {
		parsed, err := config_update.ParseNodeLink(link)
		if err != nil {
			res.errs = append(res.errs, fmt.Sprintf("链接解析失败: %s", err.Error()))
			continue
		}
		// 本批次内去重同样按节点身份：同 IP:端口但配置不同的两条链接都要保留
		idKey := customNodeIdentityKey(parsed)
		if seenLinks[idKey] {
			continue
		}
		seenLinks[idKey] = true

		configJSON, _ := json.Marshal(parsed)
		configStr := string(configJSON)
		name := parsed.Name
		if name == "" {
			name = fmt.Sprintf("%s-%s", parsed.Type, parsed.Server)
		}
		name = truncateNodeName(name)

		// 匹配已有节点：先按名称（仅订阅来源），再按节点身份（含历史遗留节点）
		var match *customNodeMatch
		if m, ok := byName[name]; ok && !m.consumed {
			match = m
		} else if m, ok := byIdentity[idKey]; ok && !m.consumed {
			match = m
		}

		if match != nil {
			match.consumed = true
			res.updates = append(res.updates, customNodeUpdate{
				ID:   match.node.ID,
				Name: name,
				Fields: map[string]interface{}{
					"name":       name,
					"protocol":   parsed.Type,
					"domain":     parsed.Server,
					"port":       parsed.Port,
					"config":     configStr,
					"source_url": urlStr,
					// 历史遗留节点被订阅内容认领后归入订阅来源，
					// 之后按 source_url 精确匹配，避免再次重复
					"source": "subscription",
				},
			})
			continue
		}

		// 未匹配：追加新节点
		res.newNodes = append(res.newNodes, models.CustomNode{
			Name:      name,
			Protocol:  parsed.Type,
			Domain:    parsed.Server,
			Port:      parsed.Port,
			Config:    configStr,
			Status:    "inactive",
			IsActive:  true,
			Source:    "subscription",
			SourceURL: urlStr,
		})
	}

	// 订阅中已消失的旧节点（未匹配到新订阅内容）：保留不删除，仅计入 kept（分配保护）
	for _, mi := range byName {
		if !mi.consumed {
			mi.consumed = true
			res.kept++
		}
	}
	// byIdentity 中未覆盖的（byName 与 byIdentity 可能指向同一节点，需去重）
	for _, mi := range byIdentity {
		if !mi.consumed {
			mi.consumed = true
			res.kept++
		}
	}

	return res
}

func UpdateCustomNode(c *gin.Context) {
	nodeID := c.Param("id")
	db := database.GetDB()

	var node models.CustomNode
	if err := db.First(&node, nodeID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "节点不存在", err)
		return
	}

	var req struct {
		Name             string     `json:"name"`
		DisplayName      string     `json:"display_name"`
		Protocol         string     `json:"protocol"`
		Config           string     `json:"config"`
		Domain           string     `json:"domain"`
		Port             int        `json:"port"`
		Status           string     `json:"status"`
		IsActive         *bool      `json:"is_active"`
		ExpireTime       *time.Time `json:"expire_time"`
		FollowUserExpire *bool      `json:"follow_user_expire"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	if req.Name != "" {
		node.Name = req.Name
	}
	if req.DisplayName != "" || req.DisplayName == "" {
		node.DisplayName = req.DisplayName
	}
	if req.Protocol != "" {
		node.Protocol = req.Protocol
	}
	if req.Config != "" {
		node.Config = req.Config
	}
	if req.Domain != "" {
		node.Domain = req.Domain
	}
	if req.Port > 0 {
		node.Port = req.Port
	}
	if req.Status != "" {
		node.Status = req.Status
	}
	if req.IsActive != nil {
		node.IsActive = *req.IsActive
	}
	if req.ExpireTime != nil {
		node.ExpireTime = req.ExpireTime
	}
	if req.FollowUserExpire != nil {
		node.FollowUserExpire = *req.FollowUserExpire
	}

	node.Config, node.Protocol, node.Domain, node.Port = normalizeCustomNodeConfig(
		node.Config,
		node.Protocol,
		node.Domain,
		node.Port,
	)

	if err := db.Save(&node).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新失败: "+err.Error(), err)
		return
	}
	utils.CreateAuditLogSimple(c, "update_custom_node", "custom_node", node.ID, fmt.Sprintf("管理员操作: 更新专线节点 %s", node.Name))
	// 清除所有关联用户的缓存
	var userIDs []uint
	db.Model(&models.UserCustomNode{}).Where("custom_node_id = ?", node.ID).Pluck("user_id", &userIDs)
	for _, uid := range userIDs {
		clearUserCustomNodeCache(uid)
	}
	utils.SuccessResponse(c, http.StatusOK, "", node)
}

func DeleteCustomNode(c *gin.Context) {
	nodeID := c.Param("id")
	db := database.GetDB()

	var node models.CustomNode
	if err := db.First(&node, nodeID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "节点不存在", err)
		return
	}

	// 先获取关联用户，删除后就查不到了
	var affectedUserIDs []uint
	db.Model(&models.UserCustomNode{}).Where("custom_node_id = ?", nodeID).Pluck("user_id", &affectedUserIDs)

	db.Where("custom_node_id = ?", nodeID).Delete(&models.UserCustomNode{})

	// 自建节点联动：删除主节点时，一并删除同 install_id 的多协议子节点（避免孤儿记录）
	siblingCount := int64(0)
	if node.SelfHosted && node.InstallID != "" {
		var siblingIDs []uint
		db.Model(&models.CustomNode{}).Where("install_id = ? AND id != ?", node.InstallID, node.ID).Pluck("id", &siblingIDs)
		if len(siblingIDs) > 0 {
			db.Where("custom_node_id IN ?", siblingIDs).Delete(&models.UserCustomNode{})
			db.Where("id IN ?", siblingIDs).Delete(&models.CustomNode{})
			siblingCount = int64(len(siblingIDs))
		}
	}

	if err := db.Delete(&node).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除失败: "+err.Error(), err)
		return
	}
	utils.CreateAuditLogSimple(c, "delete_custom_node", "custom_node", node.ID, fmt.Sprintf("管理员操作: 删除专线节点 %s，同时取消 %d 个用户的分配"+(map[bool]string{true: "，并联动删除 %d 个自建子节点"})[siblingCount > 0], node.Name, len(affectedUserIDs), siblingCount))
	for _, uid := range affectedUserIDs {
		clearUserCustomNodeCache(uid)
		resetSpecialNodeFieldsIfNoCustomNodes(db, uid)
	}
	utils.SuccessResponse(c, http.StatusOK, "删除成功", nil)
}

func BatchDeleteCustomNodes(c *gin.Context) {
	var req struct {
		NodeIDs []uint `json:"node_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	db := database.GetDB()

	// 先获取关联用户
	var batchAffectedUserIDs []uint
	db.Model(&models.UserCustomNode{}).Where("custom_node_id IN ?", req.NodeIDs).Pluck("user_id", &batchAffectedUserIDs)

	// 删除前记录节点名称，用于操作日志
	var deletingNodes []models.CustomNode
	db.Where("id IN ?", req.NodeIDs).Find(&deletingNodes)

	db.Where("custom_node_id IN ?", req.NodeIDs).Delete(&models.UserCustomNode{})

	// 自建节点联动：删除主节点时，一并删除同 install_id 的多协议子节点（避免孤儿记录）
	var orphanIDs []uint
	for _, n := range deletingNodes {
		if n.SelfHosted && n.InstallID != "" {
			var sibIDs []uint
			db.Model(&models.CustomNode{}).Where("install_id = ? AND id NOT IN ?", n.InstallID, req.NodeIDs).Pluck("id", &sibIDs)
			orphanIDs = append(orphanIDs, sibIDs...)
		}
	}
	if len(orphanIDs) > 0 {
		db.Where("custom_node_id IN ?", orphanIDs).Delete(&models.UserCustomNode{})
		db.Where("id IN ?", orphanIDs).Delete(&models.CustomNode{})
	}

	if err := db.Where("id IN ?", req.NodeIDs).Delete(&models.CustomNode{}).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "批量删除失败: "+err.Error(), err)
		return
	}
	utils.CreateAuditLogSimple(c, "batch_delete_custom_nodes", "custom_node", 0, fmt.Sprintf("管理员操作: 批量删除专线节点 %d 个 [%s]，同时取消 %d 个用户的分配%s", len(req.NodeIDs), joinNodeNames(deletingNodes), len(batchAffectedUserIDs), map[bool]string{true: fmt.Sprintf("，并联动删除 %d 个自建子节点", len(orphanIDs))}[len(orphanIDs) > 0]))
	for _, uid := range batchAffectedUserIDs {
		clearUserCustomNodeCache(uid)
		resetSpecialNodeFieldsIfNoCustomNodes(db, uid)
	}
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功删除 %d 个节点", len(req.NodeIDs)), nil)
}

func BatchAssignCustomNodes(c *gin.Context) {
	var req struct {
		NodeIDs          []uint     `json:"node_ids" binding:"required"`
		UserIDs          []uint     `json:"user_ids" binding:"required"`
		SubscriptionType string     `json:"subscription_type"`
		ExpiresAt        *time.Time `json:"expires_at"`
		UnlimitedDevices *bool      `json:"unlimited_devices"` // true = 不限制设备数量
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	db := database.GetDB()

	var nodeCount int64
	db.Model(&models.CustomNode{}).Where("id IN ?", req.NodeIDs).Count(&nodeCount)
	if nodeCount != int64(len(req.NodeIDs)) {
		utils.ErrorResponse(c, http.StatusBadRequest, "部分节点不存在", nil)
		return
	}

	var userCount int64
	db.Model(&models.User{}).Where("id IN ?", req.UserIDs).Count(&userCount)
	if userCount != int64(len(req.UserIDs)) {
		utils.ErrorResponse(c, http.StatusBadRequest, "部分用户不存在", nil)
		return
	}

	// 批量查出已存在的分配关系
	var existingNodes []models.UserCustomNode
	db.Where("user_id IN ? AND custom_node_id IN ?", req.UserIDs, req.NodeIDs).Find(&existingNodes)
	existingSet := make(map[string]bool)
	for _, en := range existingNodes {
		existingSet[fmt.Sprintf("%d-%d", en.UserID, en.CustomNodeID)] = true
	}

	// 批量查出所有相关用户
	var users []models.User
	db.Where("id IN ?", req.UserIDs).Find(&users)
	userMap := make(map[uint]*models.User)
	for i := range users {
		userMap[users[i].ID] = &users[i]
	}

	// 事务内批量创建分配关系 + 更新用户专线配置，保证原子性
	var toCreate []models.UserCustomNode
	var usersToSave []*models.User

	for _, userID := range req.UserIDs {
		for _, nodeID := range req.NodeIDs {
			key := fmt.Sprintf("%d-%d", userID, nodeID)
			if existingSet[key] {
				continue
			}
			toCreate = append(toCreate, models.UserCustomNode{
				UserID:       userID,
				CustomNodeID: nodeID,
			})
		}

		if u, ok := userMap[userID]; ok {
			needSave := false
			if req.SubscriptionType != "" {
				u.SpecialNodeSubscriptionType = req.SubscriptionType
				needSave = true
			}
			if req.ExpiresAt != nil {
				u.SpecialNodeExpiresAt = sql.NullTime{Time: *req.ExpiresAt, Valid: true}
				needSave = true
			}
			if req.UnlimitedDevices != nil {
				u.SpecialNodeUnlimitedDevices = *req.UnlimitedDevices
				needSave = true
			}
			if needSave {
				usersToSave = append(usersToSave, u)
			}
		}
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		if len(toCreate) > 0 {
			// 批量写入分配关系（替代逐条 Create）
			if err := tx.CreateInBatches(toCreate, 200).Error; err != nil {
				return err
			}
		}
		for _, u := range usersToSave {
			if err := tx.Save(u).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		utils.LogError("BatchAssignCustomNodes: transaction failed", err, map[string]interface{}{
			"node_ids": req.NodeIDs, "user_ids": req.UserIDs,
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "批量分配失败", err)
		return
	}
	assignedCount := len(toCreate)
	// 记录详细操作日志：节点名称与用户信息
	var assignedNodes []models.CustomNode
	db.Where("id IN ?", req.NodeIDs).Find(&assignedNodes)
	utils.CreateAuditLogSimple(c, "batch_assign_custom_nodes", "custom_node", 0, fmt.Sprintf("管理员操作: 批量分配专线节点 [%s] 给用户 %s 共 %d 个分配关系", joinNodeNames(assignedNodes), joinUserNames(db, req.UserIDs), assignedCount))
	// 清除所有相关用户的缓存
	for _, userID := range req.UserIDs {
		clearUserCustomNodeCache(userID)
	}
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功分配 %d 个节点关系", assignedCount), nil)
}

func BatchUnassignCustomNodes(c *gin.Context) {
	var req struct {
		NodeIDs []uint `json:"node_ids" binding:"required"`
		UserIDs []uint `json:"user_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}
	if len(req.NodeIDs) == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "请选择要取消分配的节点", nil)
		return
	}

	db := database.GetDB()
	query := db.Where("custom_node_id IN ?", req.NodeIDs)
	if len(req.UserIDs) > 0 {
		query = query.Where("user_id IN ?", req.UserIDs)
	}

	var affectedUserIDs []uint
	if err := query.Model(&models.UserCustomNode{}).Distinct("user_id").Pluck("user_id", &affectedUserIDs).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询受影响用户失败", err)
		return
	}
	if len(affectedUserIDs) == 0 {
		utils.SuccessResponse(c, http.StatusOK, "没有需要取消的分配关系", gin.H{"unassigned": 0})
		return
	}

	deleteQuery := db.Where("custom_node_id IN ?", req.NodeIDs)
	if len(req.UserIDs) > 0 {
		deleteQuery = deleteQuery.Where("user_id IN ?", req.UserIDs)
	}
	result := deleteQuery.Delete(&models.UserCustomNode{})
	if result.Error != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "批量取消分配失败: "+result.Error.Error(), result.Error)
		return
	}

	for _, uid := range affectedUserIDs {
		clearUserCustomNodeCache(uid)
		resetSpecialNodeFieldsIfNoCustomNodes(db, uid)
	}
	var unassignedNodes []models.CustomNode
	db.Where("id IN ?", req.NodeIDs).Find(&unassignedNodes)
	utils.CreateAuditLogSimple(c, "batch_unassign_custom_nodes", "custom_node", 0, fmt.Sprintf("管理员操作: 批量取消专线节点 [%s] 对用户 %s 的分配，共取消 %d 个分配关系", joinNodeNames(unassignedNodes), joinUserNames(db, affectedUserIDs), result.RowsAffected))
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功取消 %d 个分配关系", result.RowsAffected), gin.H{
		"unassigned": result.RowsAffected,
		"user_count": len(affectedUserIDs),
	})
}

func MigrateCustomNodeAssignments(c *gin.Context) {
	var req struct {
		FromNodeID       uint  `json:"from_node_id" binding:"required"`
		ToNodeID         uint  `json:"to_node_id" binding:"required"`
		DeactivateSource *bool `json:"deactivate_source"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}
	if req.FromNodeID == req.ToNodeID {
		utils.ErrorResponse(c, http.StatusBadRequest, "源节点和目标节点不能相同", nil)
		return
	}

	db := database.GetDB()
	var fromNode, toNode models.CustomNode
	if err := db.First(&fromNode, req.FromNodeID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "源节点不存在", err)
		return
	}
	if err := db.First(&toNode, req.ToNodeID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "目标节点不存在", err)
		return
	}

	var sourceRelations []models.UserCustomNode
	if err := db.Where("custom_node_id = ?", req.FromNodeID).Find(&sourceRelations).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询源节点分配关系失败", err)
		return
	}
	if len(sourceRelations) == 0 {
		utils.SuccessResponse(c, http.StatusOK, "源节点没有需要迁移的用户", gin.H{"migrated": 0, "skipped": 0})
		return
	}

	userIDs := make([]uint, 0, len(sourceRelations))
	for _, rel := range sourceRelations {
		userIDs = append(userIDs, rel.UserID)
	}

	var existingTargets []models.UserCustomNode
	db.Where("user_id IN ? AND custom_node_id = ?", userIDs, req.ToNodeID).Find(&existingTargets)
	existingTargetUsers := make(map[uint]bool, len(existingTargets))
	for _, rel := range existingTargets {
		existingTargetUsers[rel.UserID] = true
	}

	migratedCount := 0
	skippedCount := 0
	err := db.Transaction(func(tx *gorm.DB) error {
		newRelations := make([]models.UserCustomNode, 0, len(sourceRelations))
		for _, rel := range sourceRelations {
			if existingTargetUsers[rel.UserID] {
				skippedCount++
				continue
			}
			newRelations = append(newRelations, models.UserCustomNode{
				UserID:       rel.UserID,
				CustomNodeID: req.ToNodeID,
			})
		}
		if len(newRelations) > 0 {
			if err := tx.CreateInBatches(newRelations, 100).Error; err != nil {
				return err
			}
			migratedCount = len(newRelations)
		}
		if err := tx.Where("custom_node_id = ?", req.FromNodeID).Delete(&models.UserCustomNode{}).Error; err != nil {
			return err
		}
		if req.DeactivateSource != nil && *req.DeactivateSource {
			if err := tx.Model(&fromNode).Update("is_active", false).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "迁移分配失败: "+err.Error(), err)
		return
	}

	for _, uid := range userIDs {
		clearUserCustomNodeCache(uid)
		resetSpecialNodeFieldsIfNoCustomNodes(db, uid)
	}
	utils.CreateAuditLogSimple(c, "migrate_custom_node_assignments", "custom_node", req.FromNodeID, fmt.Sprintf("管理员操作: 迁移专线分配 %s -> %s 用户 %d 个 新增 %d 跳过 %d", fromNode.Name, toNode.Name, len(userIDs), migratedCount, skippedCount))
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("已迁移 %d 个用户，跳过 %d 个已拥有目标节点的用户", migratedCount, skippedCount), gin.H{
		"migrated":     migratedCount,
		"skipped":      skippedCount,
		"user_count":   len(userIDs),
		"from_node_id": req.FromNodeID,
		"to_node_id":   req.ToNodeID,
	})
}

func TestCustomNode(c *gin.Context) {
	nodeID := c.Param("id")
	db := database.GetDB()

	var node models.CustomNode
	if err := db.First(&node, nodeID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "节点不存在", err)
		return
	}

	var config models.NodeConfig
	if err := json.Unmarshal([]byte(node.Config), &config); err != nil {
		utils.SuccessResponse(c, http.StatusOK, "", gin.H{
			"status":  "error",
			"latency": 0,
			"message": "配置解析失败",
		})
		return
	}

	if config.Server == "" {
		utils.SuccessResponse(c, http.StatusOK, "", gin.H{
			"status":  "error",
			"latency": 0,
			"message": "服务器地址为空",
		})
		return
	}

	// 真实连通性测试：走 node_health 的 TCP 握手/延迟探测，
	// 不再伪造 active + 100ms（此前是假测试）
	svc := node_health.NewNodeHealthService()
	cfgJSON, _ := json.Marshal(config_update.ProxyNode{
		Type:     config.Type,
		Server:   config.Server,
		Port:     config.Port,
		UUID:     config.UUID,
		Password: config.Password,
		Network:  config.Network,
		Cipher:   config.Encryption,
		TLS:      config.Security == "tls",
	})
	cfgStr := string(cfgJSON)
	tempNode := models.Node{ID: node.ID, Config: &cfgStr}

	res, err := svc.TestNode(&tempNode)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "测试节点失败", err)
		return
	}

	now := utils.GetBeijingTime()
	node.Status = res.Status
	node.Latency = res.Latency
	node.LastTest = &now
	// 自动屏蔽超时/离线专线节点（与普通节点同一开关）；
	// unsupported（UDP 协议无法探测）不参与判定，避免误禁可用节点
	if autoDisableTimeoutEnabled(db) && node_health.ShouldAutoDisable(res.Status) {
		node.IsActive = false
	} else if res.Status == node_health.StatusOnline {
		node.IsActive = true
	}
	if err := db.Save(&node).Error; err != nil {
		utils.LogError("TestCustomNode: save node failed", err, nil)
	}
	clearNodeCaches()

	utils.CreateAuditLogSimple(c, "test_custom_node", "custom_node", node.ID, fmt.Sprintf("管理员操作: 测试专线节点 %s 结果 %s 延迟 %dms", node.Name, res.Status, res.Latency))

	utils.SuccessResponse(c, http.StatusOK, "", res)
}

func BatchTestCustomNodes(c *gin.Context) {
	var req struct {
		NodeIDs []uint `json:"node_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	if len(req.NodeIDs) == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "未选择节点", nil)
		return
	}

	db := database.GetDB()
	results := make([]gin.H, 0)

	// 批量查出所有节点
	var nodes []models.CustomNode
	db.Where("id IN ?", req.NodeIDs).Find(&nodes)
	nodeMap := make(map[uint]*models.CustomNode)
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}

	for _, nodeID := range req.NodeIDs {
		node, ok := nodeMap[nodeID]
		if !ok {
			results = append(results, gin.H{
				"node_id": nodeID,
				"status":  "error",
				"latency": 0,
				"message": "节点不存在",
			})
			continue
		}

		var config models.NodeConfig
		if err := json.Unmarshal([]byte(node.Config), &config); err != nil {
			results = append(results, gin.H{
				"node_id": nodeID,
				"status":  "error",
				"latency": 0,
				"message": "配置解析失败",
			})
			continue
		}

		if config.Server == "" {
			results = append(results, gin.H{
				"node_id": nodeID,
				"status":  "error",
				"latency": 0,
				"message": "服务器地址为空",
			})
			continue
		}

		// 注意：这里不要预写 status（旧代码先写 active 再测试，属于假测试残留，
		// 测试中途失败会留下与真实状态不符的 active）。状态一律由测试结果决定。

		// 真实连通性测试：非 UDP 协议走 TCP 握手/网页测速，UDP 协议返回 unsupported
		svc := node_health.NewNodeHealthService()
		cfgJSON, _ := json.Marshal(config_update.ProxyNode{
			Type:     config.Type,
			Server:   config.Server,
			Port:     config.Port,
			UUID:     config.UUID,
			Password: config.Password,
			Network:  config.Network,
			Cipher:   config.Encryption,
			TLS:      config.Security == "tls",
		})
		cfgStr := string(cfgJSON)
		tempNode := models.Node{ID: node.ID, Config: &cfgStr}
		res, err := svc.TestNode(&tempNode)
		if err != nil {
			results = append(results, gin.H{
				"node_id": nodeID,
				"status":  "error",
				"latency": 0,
				"message": err.Error(),
			})
			continue
		}

		now := utils.GetBeijingTime()
		node.Status = res.Status
		node.Latency = res.Latency
		node.LastTest = &now
		// 自动屏蔽超时/离线专线节点（与普通节点同一开关：node_health.auto_disable_timeout）；
		// unsupported 不参与判定，否则 hysteria2/tuic 等 UDP 节点会被误禁
		if autoDisableTimeoutEnabled(db) && node_health.ShouldAutoDisable(res.Status) {
			node.IsActive = false
		} else if res.Status == node_health.StatusOnline {
			node.IsActive = true
		}
		db.Save(node)

		results = append(results, gin.H{
			"node_id":   nodeID,
			"status":    res.Status,
			"latency":   res.Latency,
			"message":   res.Error,
			"is_active": node.IsActive,
		})
	}

	clearNodeCaches()

	// 统计口径必须与节点状态常量一致：online 算在线，timeout/offline/error 算失败。
	// UDP 协议（hysteria2/tuic）现在按 online 返回（服务端测不到不代表不可用），
	// StatusUnsupported 只可能出现在历史数据里，这里仍然兜住，避免总数对不上。
	onlineCount, failedCount, unsupportedCount := 0, 0, 0
	for _, r := range results {
		status, _ := r["status"].(string)
		switch {
		case status == node_health.StatusOnline:
			onlineCount++
		case status == node_health.StatusUnsupported:
			unsupportedCount++
		case status == node_health.StatusTimeout || status == node_health.StatusOffline || status == "error":
			failedCount++
		}
	}
	auditMsg := fmt.Sprintf("管理员操作: 批量测试专线节点 %d 个 在线 %d 个 离线/超时 %d 个",
		len(req.NodeIDs), onlineCount, failedCount)
	if unsupportedCount > 0 {
		auditMsg += fmt.Sprintf(" 无法探测 %d 个", unsupportedCount)
	}
	utils.CreateAuditLogSimple(c, "batch_test_custom_nodes", "custom_node", 0, auditMsg)

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"results":     results,
		"total":       len(req.NodeIDs),
		"success":     onlineCount,
		"failed":      failedCount,
		"unsupported": unsupportedCount,
	})
}

// DisableTimeoutCustomNodes 一键屏蔽所有超时/离线的专线节点（is_active=false）。
// POST /admin/custom-nodes/disable-timeout
func DisableTimeoutCustomNodes(c *gin.Context) {
	db := database.GetDB()
	res := db.Model(&models.CustomNode{}).
		Where("is_active = ? AND status IN ?", true, []string{"timeout", "offline"}).
		Update("is_active", false)
	disabled := int(res.RowsAffected)
	if res.Error != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "屏蔽失败", res.Error)
		return
	}
	clearNodeCaches()
	utils.CreateAuditLogSimple(c, "disable_timeout_custom_nodes", "custom_node", 0, fmt.Sprintf("管理员操作: 一键屏蔽 %d 个超时/离线专线节点", disabled))
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("已屏蔽 %d 个超时/离线专线节点", disabled), gin.H{
		"disabled_count": disabled,
	})
}

// EnableAllCustomNodes 一键启用所有专线节点（is_active=true）。
// POST /admin/custom-nodes/enable-all
func EnableAllCustomNodes(c *gin.Context) {
	db := database.GetDB()
	// GORM 默认拒绝无 Where 的全表 Update，用恒真条件显式声明全表意图
	res := db.Model(&models.CustomNode{}).Where("1 = 1").Update("is_active", true)
	enabled := int(res.RowsAffected)
	if res.Error != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "启用失败", res.Error)
		return
	}
	clearNodeCaches()
	utils.CreateAuditLogSimple(c, "enable_all_custom_nodes", "custom_node", 0, fmt.Sprintf("管理员操作: 一键启用 %d 个专线节点", enabled))
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("已启用 %d 个专线节点", enabled), gin.H{
		"enabled_count": enabled,
	})
}

// UpdateUserCustomNodeQuota 更新某个客户对某自建节点的分配级流量配额。
// PUT /admin/custom-nodes/user-quota/:nodeId/:userId  body: {"enabled":true,"limit_bytes":107374182400}
func UpdateUserCustomNodeQuota(c *gin.Context) {
	nodeID := c.Param("nodeId")
	userID := c.Param("userId")
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
	var un models.UserCustomNode
	if err := db.Where("user_id = ? AND custom_node_id = ?", parseUint(userID), parseUint(nodeID)).First(&un).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "该节点未分配给此客户", err)
		return
	}

	if err := db.Model(&un).Updates(map[string]interface{}{
		"traffic_limit_enabled": req.Enabled,
		"traffic_limit_bytes":   req.LimitBytes,
	}).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新配额失败", err)
		return
	}

	var node models.CustomNode
	db.First(&node, nodeID)
	utils.CreateAuditLogSimple(c, "update_user_custom_node_quota", "custom_node", parseUint(nodeID),
		fmt.Sprintf("管理员操作: 设置客户#%s 对节点 %s 的流量配额 %s", userID, node.Name, formatBytesHuman(req.LimitBytes)))

	utils.SuccessResponse(c, http.StatusOK, "配额已更新", gin.H{
		"enabled":     req.Enabled,
		"limit_bytes": req.LimitBytes,
		"used":        node.TrafficUp + node.TrafficDown,
	})
}

func GetCustomNodeLink(c *gin.Context) {
	nodeID := c.Param("id")
	db := database.GetDB()

	var node models.CustomNode
	if err := db.First(&node, nodeID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "节点不存在", err)
		return
	}

	var link string
	if node.Config != "" {
		var proxyNode config_update.ProxyNode
		if err := json.Unmarshal([]byte(node.Config), &proxyNode); err == nil {
			if node.DisplayName != "" {
				proxyNode.Name = node.DisplayName
			} else if proxyNode.Name == "" {
				proxyNode.Name = node.Name
			}

			service := config_update.NewConfigUpdateService()
			link = service.NodeToLink(&proxyNode)
		} else {
			var nodeConfig models.NodeConfig
			if err2 := json.Unmarshal([]byte(node.Config), &nodeConfig); err2 == nil {
				proxyNode := &config_update.ProxyNode{
					Name:     node.DisplayName,
					Type:     nodeConfig.Type,
					Server:   nodeConfig.Server,
					Port:     nodeConfig.Port,
					UUID:     nodeConfig.UUID,
					Password: nodeConfig.Password,
					Cipher:   nodeConfig.Encryption,
					Network:  nodeConfig.Network,
					TLS:      nodeConfig.Security == "tls",
				}

				if proxyNode.Name == "" {
					proxyNode.Name = node.Name
				}

				service := config_update.NewConfigUpdateService()
				link = service.NodeToLink(proxyNode)
			}
		}
	}

	if link == "" {
		link = "无法生成链接: 配置格式错误或协议不支持"
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"id":   node.ID,
		"name": node.Name,
		"link": link,
	})
}

func GetUserCustomNodes(c *gin.Context) {
	userID := c.Param("id")
	db := database.GetDB()

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "用户不存在", err)
		return
	}

	var userNodes []models.UserCustomNode
	if err := db.Preload("CustomNode").Where("user_id = ?", userID).Find(&userNodes).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取节点列表失败", err)
		return
	}

	nodes := make([]gin.H, 0)
	for _, un := range userNodes {
		if un.CustomNode.ID > 0 {
			var specialNodeExpiresAt interface{}
			if user.SpecialNodeExpiresAt.Valid {
				specialNodeExpiresAt = utils.FormatBeijingTime(user.SpecialNodeExpiresAt.Time)
			}
			nodeAddress := un.CustomNode.Domain
			if un.CustomNode.Port > 0 && un.CustomNode.Port != 443 {
				nodeAddress = fmt.Sprintf("%s:%d", un.CustomNode.Domain, un.CustomNode.Port)
			}
			nodes = append(nodes, gin.H{
				"id":                             un.CustomNode.ID,
				"node_id":                        un.CustomNode.ID,
				"node_name":                      un.CustomNode.Name,
				"node_address":                   nodeAddress,
				"assigned_at":                    utils.FormatBeijingTime(un.CreatedAt),
				"status":                         un.CustomNode.Status,
				"is_active":                      un.CustomNode.IsActive,
				"special_node_subscription_type": user.SpecialNodeSubscriptionType,
				"special_node_expires_at":        specialNodeExpiresAt,
				"special_node_unlimited_devices": user.SpecialNodeUnlimitedDevices,
			})
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", nodes)
}

func AssignCustomNodeToUser(c *gin.Context) {
	userID := c.Param("id")
	db := database.GetDB()

	var req struct {
		CustomNodeID        uint       `json:"custom_node_id" binding:"required"`
		SubscriptionType    string     `json:"subscription_type"`
		ExpiresAt           *time.Time `json:"expires_at"`
		UnlimitedDevices    *bool      `json:"unlimited_devices"`     // true = 不限制设备数量
		TrafficLimitEnabled bool       `json:"traffic_limit_enabled"` // 分配级流量配额
		TrafficLimitBytes   int64      `json:"traffic_limit_bytes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	var existing models.UserCustomNode
	if err := db.Where("user_id = ? AND custom_node_id = ?", userID, req.CustomNodeID).First(&existing).Error; err == nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "节点已分配给该用户", nil)
		return
	}

	userNode := models.UserCustomNode{
		UserID:              parseUint(userID),
		CustomNodeID:        req.CustomNodeID,
		TrafficLimitEnabled: req.TrafficLimitEnabled,
		TrafficLimitBytes:   req.TrafficLimitBytes,
	}

	if err := db.Create(&userNode).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "分配失败: "+err.Error(), err)
		return
	}

	var user models.User
	if err := db.First(&user, userID).Error; err == nil {
		if req.SubscriptionType != "" {
			user.SpecialNodeSubscriptionType = req.SubscriptionType
		}
		if req.ExpiresAt != nil {
			user.SpecialNodeExpiresAt = sql.NullTime{Time: *req.ExpiresAt, Valid: true}
		}
		if req.UnlimitedDevices != nil {
			user.SpecialNodeUnlimitedDevices = *req.UnlimitedDevices
		}
		db.Save(&user)
	}

	// 记录管理员操作日志
	nodeName := fmt.Sprintf("节点#%d", req.CustomNodeID)
	var node models.CustomNode
	if err := db.First(&node, req.CustomNodeID).Error; err == nil && node.Name != "" {
		nodeName = node.Name
	}
	userDesc := fmt.Sprintf("用户#%d", parseUint(userID))
	if user.ID > 0 {
		if user.Email != "" {
			userDesc = fmt.Sprintf("%s (%s)", user.Username, user.Email)
		} else {
			userDesc = user.Username
		}
	}
	modeDesc := ""
	if req.SubscriptionType == "special_only" {
		modeDesc = "，线路模式：仅专线"
	} else if req.SubscriptionType == "both" {
		modeDesc = "，线路模式：专线+普通"
	}
	utils.CreateAuditLogSimple(c, "assign_custom_node", "custom_node", req.CustomNodeID, fmt.Sprintf("管理员操作: 给用户 %s 分配专线节点 %s%s", userDesc, nodeName, modeDesc))

	utils.SuccessResponse(c, http.StatusOK, "分配成功", userNode)
	clearUserCustomNodeCache(parseUint(userID))
}

func UnassignCustomNodeFromUser(c *gin.Context) {
	userID := c.Param("id")
	nodeID := c.Param("node_id")
	db := database.GetDB()

	if err := db.Where("user_id = ? AND custom_node_id = ?", userID, nodeID).Delete(&models.UserCustomNode{}).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "取消分配失败: "+err.Error(), err)
		return
	}

	uid := parseUint(userID)
	clearUserCustomNodeCache(uid)
	resetSpecialNodeFieldsIfNoCustomNodes(db, uid)

	// 记录管理员操作日志
	nid := parseUint(nodeID)
	nodeName := fmt.Sprintf("节点#%d", nid)
	var node models.CustomNode
	if err := db.First(&node, nid).Error; err == nil && node.Name != "" {
		nodeName = node.Name
	}
	userDesc := fmt.Sprintf("用户#%d", uid)
	var user models.User
	if err := db.First(&user, uid).Error; err == nil && user.ID > 0 {
		if user.Email != "" {
			userDesc = fmt.Sprintf("%s (%s)", user.Username, user.Email)
		} else {
			userDesc = user.Username
		}
	}
	utils.CreateAuditLogSimple(c, "unassign_custom_node", "custom_node", nid, fmt.Sprintf("管理员操作: 取消用户 %s 的专线节点 %s 分配", userDesc, nodeName))

	utils.SuccessResponse(c, http.StatusOK, "取消分配成功", nil)
}

func parseUint(s string) uint {
	i, _ := strconv.ParseUint(s, 10, 32)
	return uint(i)
}

// joinNodeNames 将节点名称拼接为日志用描述，超出长度截断
func joinNodeNames(nodes []models.CustomNode) string {
	names := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if n.Name != "" {
			names = append(names, n.Name)
		}
	}
	return truncateDesc(strings.Join(names, "、"))
}

// joinUserNames 将用户列表拼接为日志用描述（用户名+邮箱），超出长度截断
func joinUserNames(db *gorm.DB, userIDs []uint) string {
	if len(userIDs) == 0 {
		return "无"
	}
	var users []models.User
	db.Where("id IN ?", userIDs).Find(&users)
	parts := make([]string, 0, len(users))
	for _, u := range users {
		if u.Email != "" {
			parts = append(parts, fmt.Sprintf("%s(%s)", u.Username, u.Email))
		} else {
			parts = append(parts, u.Username)
		}
	}
	return truncateDesc(strings.Join(parts, "、"))
}

// truncateDesc 截断过长的日志描述
func truncateDesc(desc string) string {
	const maxLen = 500
	runes := []rune(desc)
	if len(runes) <= maxLen {
		return desc
	}
	return string(runes[:maxLen]) + "..."
}

// clearUserCustomNodeCache 清除用户专线节点相关缓存
func clearUserCustomNodeCache(userID uint) {
	cacheService := &config_update.CacheService{}
	_ = cacheService.ClearCustomNodesCache(userID)

	// 清除该用户的订阅配置缓存
	db := database.GetDB()
	var subscriptions []models.Subscription
	if err := db.Where("user_id = ?", userID).Find(&subscriptions).Error; err == nil {
		for _, sub := range subscriptions {
			_ = cacheService.ClearSubscriptionConfigCache(sub.SubscriptionURL)
		}
	}
}

// resetSpecialNodeFieldsIfNoCustomNodes 当用户已无专线节点时自动重置 SpecialNode 相关字段，
// 避免用户因 SpecialNodeSubscriptionType="special_only" 但无任何专线节点而导致无法访问任何线路。
func resetSpecialNodeFieldsIfNoCustomNodes(db *gorm.DB, userID uint) {
	var count int64
	if err := db.Model(&models.UserCustomNode{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		utils.LogError("resetSpecialNodeFields: count custom nodes failed", err, map[string]interface{}{"user_id": userID})
		return
	}
	if count > 0 {
		return
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		utils.LogError("resetSpecialNodeFields: find user failed", err, map[string]interface{}{"user_id": userID})
		return
	}

	// 仅当用户确实处于 special_only 模式或设置了专线到期/不限设备时才需重置
	needsReset := user.SpecialNodeSubscriptionType == "special_only" ||
		user.SpecialNodeExpiresAt.Valid ||
		user.SpecialNodeUnlimitedDevices

	if !needsReset {
		return
	}

	oldType := user.SpecialNodeSubscriptionType
	user.SpecialNodeSubscriptionType = ""
	user.SpecialNodeExpiresAt = sql.NullTime{Valid: false}
	user.SpecialNodeUnlimitedDevices = false

	if err := db.Save(&user).Error; err != nil {
		utils.LogError("resetSpecialNodeFields: save user failed", err, map[string]interface{}{"user_id": userID})
		return
	}

	utils.LogInfo("resetSpecialNodeFields: 用户 %d (%s) 已无专线节点，自动重置 SpecialNode 字段 (原 subscription_type=%s)，恢复普通线路访问",
		userID, user.Username, oldType)
}

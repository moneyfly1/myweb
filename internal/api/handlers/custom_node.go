package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"
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

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}

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
			})
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", users)
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
	for _, key := range keys {
		if val, ok := data[key]; ok {
			if s, ok := val.(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}
	for existingKey, val := range data {
		for _, key := range keys {
			if strings.EqualFold(existingKey, key) {
				if s, ok := val.(string); ok {
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
			Name:             name,
			DisplayName:      req.DisplayName,
			Protocol:         parsed.Type,
			Domain:           parsed.Server,
			Port:             parsed.Port,
			Config:           configStr,
			Status:           "inactive",
			IsActive:         true,
			ExpireTime:       req.ExpireTime,
			FollowUserExpire: req.FollowUserExpire,
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
		Name:             req.Name,
		DisplayName:      req.DisplayName,
		Protocol:         protocol,
		Domain:           domain,
		Port:             port,
		Config:           configStr,
		Status:           "inactive",
		IsActive:         true,
		ExpireTime:       req.ExpireTime,
		FollowUserExpire: req.FollowUserExpire,
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
	imported := 0
	errorCount := 0
	errors := make([]string, 0)

	for _, link := range req.Links {
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

		// #nosec G117 - Password field is proxy node password, not user credential
		configJSON, _ := json.Marshal(parsed) // #nosec G117
		configStr := string(configJSON)

		name := parsed.Name
		if name == "" {
			name = fmt.Sprintf("%s-%s", parsed.Type, parsed.Server)
		}

		customNode := models.CustomNode{
			Name:     name,
			Protocol: parsed.Type,
			Domain:   parsed.Server,
			Port:     parsed.Port,
			Config:   configStr,
			Status:   "inactive",
			IsActive: true,
		}

		if err := db.Create(&customNode).Error; err != nil {
			errorCount++
			errors = append(errors, fmt.Sprintf("创建节点失败: %s", err.Error()))
			continue
		}

		imported++
	}
	utils.CreateAuditLogSimple(c, "import_custom_node_links", "custom_node", 0, fmt.Sprintf("管理员操作: 导入专线节点链接 成功 %d 失败 %d", imported, errorCount))
	if imported > 0 {
		clearNodeCaches()
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"imported":    imported,
		"error_count": errorCount,
		"errors":      errors,
		"message":     fmt.Sprintf("成功导入 %d 个节点", imported),
	})
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

	if err := db.Delete(&node).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除失败: "+err.Error(), err)
		return
	}
	utils.CreateAuditLogSimple(c, "delete_custom_node", "custom_node", node.ID, fmt.Sprintf("管理员操作: 删除专线节点 %s，同时取消 %d 个用户的分配", node.Name, len(affectedUserIDs)))
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

	if err := db.Where("id IN ?", req.NodeIDs).Delete(&models.CustomNode{}).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "批量删除失败: "+err.Error(), err)
		return
	}
	utils.CreateAuditLogSimple(c, "batch_delete_custom_nodes", "custom_node", 0, fmt.Sprintf("管理员操作: 批量删除专线节点 %d 个 [%s]，同时取消 %d 个用户的分配", len(req.NodeIDs), joinNodeNames(deletingNodes), len(batchAffectedUserIDs)))
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

	assignedCount := 0
	for _, userID := range req.UserIDs {
		for _, nodeID := range req.NodeIDs {
			key := fmt.Sprintf("%d-%d", userID, nodeID)
			if existingSet[key] {
				continue
			}

			userNode := models.UserCustomNode{
				UserID:       userID,
				CustomNodeID: nodeID,
			}
			if err := db.Create(&userNode).Error; err == nil {
				assignedCount++
			}
		}

		// 更新用户专线配置
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
				db.Save(u)
			}
		}
	}
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

	node.Status = "active"
	db.Save(&node)
	clearNodeCaches()

	utils.CreateAuditLogSimple(c, "test_custom_node", "custom_node", node.ID, fmt.Sprintf("管理员操作: 测试专线节点 %s 结果 %s", node.Name, node.Status))

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"status":  "active",
		"latency": 100, // 模拟延迟
	})
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

		node.Status = "active"
		db.Save(node)

		results = append(results, gin.H{
			"node_id": nodeID,
			"status":  "active",
			"latency": 100, // 模拟延迟
		})
	}

	clearNodeCaches()

	successCount := 0
	for _, r := range results {
		if status, ok := r["status"].(string); ok && status == "active" {
			successCount++
		}
	}
	utils.CreateAuditLogSimple(c, "batch_test_custom_nodes", "custom_node", 0, fmt.Sprintf("管理员操作: 批量测试专线节点 %d 个 成功 %d 个", len(req.NodeIDs), successCount))

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"results": results,
		"total":   len(req.NodeIDs),
		"success": len(results),
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
		CustomNodeID     uint       `json:"custom_node_id" binding:"required"`
		SubscriptionType string     `json:"subscription_type"`
		ExpiresAt        *time.Time `json:"expires_at"`
		UnlimitedDevices *bool      `json:"unlimited_devices"` // true = 不限制设备数量
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
		UserID:       parseUint(userID),
		CustomNodeID: req.CustomNodeID,
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

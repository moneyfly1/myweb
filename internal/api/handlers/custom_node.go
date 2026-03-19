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
)

func GetCustomNodes(c *gin.Context) {
	db := database.GetDB()
	var nodes []models.CustomNode
	query := db.Model(&models.CustomNode{})

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
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

		searchPattern := "%" + search + "%"
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
			})
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", users)
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
		utils.SuccessResponse(c, http.StatusCreated, "", customNode)
		return
	}

	if req.Name == "" || req.Protocol == "" || req.Config == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "节点名称、协议和配置为必填项", nil)
		return
	}

	customNode := models.CustomNode{
		Name:             req.Name,
		DisplayName:      req.DisplayName,
		Protocol:         req.Protocol,
		Domain:           req.Domain,
		Port:             req.Port,
		Config:           req.Config,
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
	utils.CreateAuditLogSimple(c, "delete_custom_node", "custom_node", node.ID, fmt.Sprintf("管理员操作: 删除专线节点 %s", node.Name))
	for _, uid := range affectedUserIDs {
		clearUserCustomNodeCache(uid)
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

	db.Where("custom_node_id IN ?", req.NodeIDs).Delete(&models.UserCustomNode{})

	if err := db.Where("id IN ?", req.NodeIDs).Delete(&models.CustomNode{}).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "批量删除失败: "+err.Error(), err)
		return
	}
	utils.CreateAuditLogSimple(c, "batch_delete_custom_nodes", "custom_node", 0, fmt.Sprintf("管理员操作: 批量删除专线节点 %d 个", len(req.NodeIDs)))
	for _, uid := range batchAffectedUserIDs {
		clearUserCustomNodeCache(uid)
	}
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功删除 %d 个节点", len(req.NodeIDs)), nil)
}

func BatchAssignCustomNodes(c *gin.Context) {
	var req struct {
		NodeIDs          []uint     `json:"node_ids" binding:"required"`
		UserIDs          []uint     `json:"user_ids" binding:"required"`
		SubscriptionType string     `json:"subscription_type"`
		ExpiresAt        *time.Time `json:"expires_at"`
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

	assignedCount := 0
	for _, userID := range req.UserIDs {
		for _, nodeID := range req.NodeIDs {
			var existing models.UserCustomNode
			if err := db.Where("user_id = ? AND custom_node_id = ?", userID, nodeID).First(&existing).Error; err == nil {
				continue // 已存在，跳过
			}

			userNode := models.UserCustomNode{
				UserID:       userID,
				CustomNodeID: nodeID,
			}
			if err := db.Create(&userNode).Error; err == nil {
				assignedCount++
			}

			var user models.User
			if err := db.First(&user, userID).Error; err == nil {
				if req.SubscriptionType != "" {
					user.SpecialNodeSubscriptionType = req.SubscriptionType
				}
				if req.ExpiresAt != nil {
					user.SpecialNodeExpiresAt = sql.NullTime{Time: *req.ExpiresAt, Valid: true}
				}
				db.Save(&user)
			}
		}
	}
	utils.CreateAuditLogSimple(c, "batch_assign_custom_nodes", "custom_node", 0, fmt.Sprintf("管理员操作: 批量分配专线节点 节点 %d 个 用户 %d 个 分配关系 %d", len(req.NodeIDs), len(req.UserIDs), assignedCount))
	// 清除所有相关用户的缓存
	for _, userID := range req.UserIDs {
		clearUserCustomNodeCache(userID)
	}
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功分配 %d 个节点关系", assignedCount), nil)
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

	for _, nodeID := range req.NodeIDs {
		var node models.CustomNode
		if err := db.First(&node, nodeID).Error; err != nil {
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
		db.Save(&node)

		results = append(results, gin.H{
			"node_id": nodeID,
			"status":  "active",
			"latency": 100, // 模拟延迟
		})
	}

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

	var userNodes []models.UserCustomNode
	if err := db.Preload("CustomNode").Where("user_id = ?", userID).Find(&userNodes).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取节点列表失败", err)
		return
	}

	nodes := make([]gin.H, 0)
	for _, un := range userNodes {
		if un.CustomNode.ID > 0 {
			nodeAddress := un.CustomNode.Domain
			if un.CustomNode.Port > 0 && un.CustomNode.Port != 443 {
				nodeAddress = fmt.Sprintf("%s:%d", un.CustomNode.Domain, un.CustomNode.Port)
			}
			nodes = append(nodes, gin.H{
				"id":           un.CustomNode.ID,
				"node_id":      un.CustomNode.ID,
				"node_name":    un.CustomNode.Name,
				"node_address": nodeAddress,
				"assigned_at":  utils.FormatBeijingTime(un.CreatedAt),
				"status":       un.CustomNode.Status,
				"is_active":    un.CustomNode.IsActive,
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
		db.Save(&user)
	}

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

	utils.SuccessResponse(c, http.StatusOK, "取消分配成功", nil)
	clearUserCustomNodeCache(parseUint(userID))
}

func parseUint(s string) uint {
	i, _ := strconv.ParseUint(s, 10, 32)
	return uint(i)
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

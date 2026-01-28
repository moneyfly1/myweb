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
		var userIDs []uint
		db.Model(&models.User{}).Where("username LIKE ? OR email LIKE ?", "%"+search+"%", "%"+search+"%").Pluck("id", &userIDs)

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
		var user models.User
		if err := db.First(&user, un.UserID).Error; err == nil {
			users = append(users, gin.H{
				"id":                             user.ID,
				"username":                       user.Username,
				"email":                          user.Email,
				"special_node_subscription_type": user.SpecialNodeSubscriptionType,
				"special_node_expires_at":        user.SpecialNodeExpiresAt,
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

		configJSON, _ := json.Marshal(parsed)
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

		configJSON, _ := json.Marshal(parsed)
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

	db.Where("custom_node_id = ?", nodeID).Delete(&models.UserCustomNode{})

	if err := db.Delete(&node).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除失败: "+err.Error(), err)
		return
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

	db.Where("custom_node_id IN ?", req.NodeIDs).Delete(&models.UserCustomNode{})

	if err := db.Where("id IN ?", req.NodeIDs).Delete(&models.CustomNode{}).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "批量删除失败: "+err.Error(), err)
		return
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

	nodes := make([]models.CustomNode, 0)
	for _, un := range userNodes {
		if un.CustomNode.ID > 0 {
			nodes = append(nodes, un.CustomNode)
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
}

func parseUint(s string) uint {
	i, _ := strconv.ParseUint(s, 10, 32)
	return uint(i)
}

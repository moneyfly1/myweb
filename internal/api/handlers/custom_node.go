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

	"github.com/gin-gonic/gin"
)

// GetCustomNodes 获取专线节点列表
func GetCustomNodes(c *gin.Context) {
	db := database.GetDB()
	var nodes []models.CustomNode
	query := db.Model(&models.CustomNode{})

	// 筛选条件
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
		query = query.Where("name LIKE ? OR display_name LIKE ? OR domain LIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// 分页
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取节点列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    nodes,
		"total":   total,
		"page":    page,
		"size":    size,
	})
}

// GetCustomNodeUsers 获取节点的已分配用户列表
func GetCustomNodeUsers(c *gin.Context) {
	nodeID := c.Param("id")
	db := database.GetDB()

	var node models.CustomNode
	if err := db.First(&node, nodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "节点不存在",
		})
		return
	}

	var userNodes []models.UserCustomNode
	if err := db.Preload("User").Where("custom_node_id = ?", nodeID).Find(&userNodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取用户列表失败",
		})
		return
	}

	// 获取用户详细信息
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    users,
	})
}

// CreateCustomNode 创建专线节点
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
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	db := database.GetDB()

	// 如果提供了节点链接，解析链接
	if req.NodeLink != "" {
		parsed, err := config_update.ParseNodeLink(strings.TrimSpace(req.NodeLink))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "解析节点链接失败: " + err.Error(),
			})
			return
		}

		// 构建节点配置
		configJSON, _ := json.Marshal(parsed)
		configStr := string(configJSON)

		// 构建节点名称
		name := req.Name
		if name == "" {
			name = parsed.Name
			if name == "" {
				name = fmt.Sprintf("%s-%s", parsed.Type, parsed.Server)
			}
		}

		// 构建 CustomNode
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
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data": gin.H{
					"name":   customNode.Name,
					"type":   customNode.Protocol,
					"server": customNode.Domain,
					"port":   customNode.Port,
					"config": customNode.Config,
				},
			})
			return
		}

		if err := db.Create(&customNode).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "创建节点失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"data":    customNode,
		})
		return
	}

	// 手动创建模式
	if req.Name == "" || req.Protocol == "" || req.Config == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "节点名称、协议和配置为必填项",
		})
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建节点失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    customNode,
	})
}

// ImportCustomNodeLinks 批量导入节点链接
func ImportCustomNodeLinks(c *gin.Context) {
	var req struct {
		Links []string `json:"links" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误",
		})
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

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"imported":    imported,
		"error_count": errorCount,
		"errors":      errors,
		"message":     fmt.Sprintf("成功导入 %d 个节点", imported),
	})
}

// UpdateCustomNode 更新专线节点
func UpdateCustomNode(c *gin.Context) {
	nodeID := c.Param("id")
	db := database.GetDB()

	var node models.CustomNode
	if err := db.First(&node, nodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "节点不存在",
		})
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
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}

	// 更新字段
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    node,
	})
}

// DeleteCustomNode 删除专线节点
func DeleteCustomNode(c *gin.Context) {
	nodeID := c.Param("id")
	db := database.GetDB()

	var node models.CustomNode
	if err := db.First(&node, nodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "节点不存在",
		})
		return
	}

	// 删除用户关联
	db.Where("custom_node_id = ?", nodeID).Delete(&models.UserCustomNode{})

	// 删除节点
	if err := db.Delete(&node).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "删除成功",
	})
}

// BatchDeleteCustomNodes 批量删除专线节点
func BatchDeleteCustomNodes(c *gin.Context) {
	var req struct {
		NodeIDs []uint `json:"node_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}

	db := database.GetDB()

	// 删除用户关联
	db.Where("custom_node_id IN ?", req.NodeIDs).Delete(&models.UserCustomNode{})

	// 删除节点
	if err := db.Where("id IN ?", req.NodeIDs).Delete(&models.CustomNode{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "批量删除失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("成功删除 %d 个节点", len(req.NodeIDs)),
	})
}

// BatchAssignCustomNodes 批量分配专线节点给用户
func BatchAssignCustomNodes(c *gin.Context) {
	var req struct {
		NodeIDs          []uint     `json:"node_ids" binding:"required"`
		UserIDs          []uint     `json:"user_ids" binding:"required"`
		SubscriptionType string     `json:"subscription_type"`
		ExpiresAt        *time.Time `json:"expires_at"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}

	db := database.GetDB()

	// 验证节点存在
	var nodeCount int64
	db.Model(&models.CustomNode{}).Where("id IN ?", req.NodeIDs).Count(&nodeCount)
	if nodeCount != int64(len(req.NodeIDs)) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "部分节点不存在",
		})
		return
	}

	// 验证用户存在
	var userCount int64
	db.Model(&models.User{}).Where("id IN ?", req.UserIDs).Count(&userCount)
	if userCount != int64(len(req.UserIDs)) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "部分用户不存在",
		})
		return
	}

	// 分配节点
	assignedCount := 0
	for _, userID := range req.UserIDs {
		for _, nodeID := range req.NodeIDs {
			// 检查是否已分配
			var existing models.UserCustomNode
			if err := db.Where("user_id = ? AND custom_node_id = ?", userID, nodeID).First(&existing).Error; err == nil {
				continue // 已存在，跳过
			}

			// 创建分配记录
			userNode := models.UserCustomNode{
				UserID:       userID,
				CustomNodeID: nodeID,
			}
			if err := db.Create(&userNode).Error; err == nil {
				assignedCount++
			}

			// 更新用户订阅类型和到期时间
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("成功分配 %d 个节点关系", assignedCount),
	})
}

// TestCustomNode 测试专线节点
func TestCustomNode(c *gin.Context) {
	nodeID := c.Param("id")
	db := database.GetDB()

	var node models.CustomNode
	if err := db.First(&node, nodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "节点不存在",
		})
		return
	}

	// 简单的测试逻辑：解析配置并检查基本字段
	var config models.NodeConfig
	if err := json.Unmarshal([]byte(node.Config), &config); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"status":  "error",
				"latency": 0,
				"message": "配置解析失败",
			},
		})
		return
	}

	// 检查基本字段
	if config.Server == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"status":  "error",
				"latency": 0,
				"message": "服务器地址为空",
			},
		})
		return
	}

	// 更新节点状态为 active（简单测试通过）
	node.Status = "active"
	db.Save(&node)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"status":  "active",
			"latency": 100, // 模拟延迟
		},
	})
}

// GetCustomNodeLink 获取节点订阅链接
func GetCustomNodeLink(c *gin.Context) {
	nodeID := c.Param("id")
	db := database.GetDB()

	var node models.CustomNode
	if err := db.First(&node, nodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "节点不存在",
		})
		return
	}

	// 构建订阅链接（这里简化处理，实际应该根据配置生成完整的订阅链接）
	link := fmt.Sprintf("专线节点: %s", node.Name)
	if node.Config != "" {
		link = node.Config // 使用配置作为链接
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":   node.ID,
			"name": node.Name,
			"link": link,
		},
	})
}

// GetUserCustomNodes 获取用户的专线节点列表
func GetUserCustomNodes(c *gin.Context) {
	userID := c.Param("id")
	db := database.GetDB()

	var userNodes []models.UserCustomNode
	if err := db.Preload("CustomNode").Where("user_id = ?", userID).Find(&userNodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取节点列表失败",
		})
		return
	}

	nodes := make([]models.CustomNode, 0)
	for _, un := range userNodes {
		if un.CustomNode.ID > 0 {
			nodes = append(nodes, un.CustomNode)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    nodes,
	})
}

// AssignCustomNodeToUser 为用户分配专线节点
func AssignCustomNodeToUser(c *gin.Context) {
	userID := c.Param("id")
	db := database.GetDB()

	var req struct {
		CustomNodeID     uint       `json:"custom_node_id" binding:"required"`
		SubscriptionType string     `json:"subscription_type"`
		ExpiresAt        *time.Time `json:"expires_at"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}

	// 检查是否已分配
	var existing models.UserCustomNode
	if err := db.Where("user_id = ? AND custom_node_id = ?", userID, req.CustomNodeID).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "节点已分配给该用户",
		})
		return
	}

	// 创建分配记录
	userNode := models.UserCustomNode{
		UserID:       parseUint(userID),
		CustomNodeID: req.CustomNodeID,
	}

	if err := db.Create(&userNode).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "分配失败: " + err.Error(),
		})
		return
	}

	// 更新用户订阅类型和到期时间
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    userNode,
		"message": "分配成功",
	})
}

// UnassignCustomNodeFromUser 取消用户的专线节点分配
func UnassignCustomNodeFromUser(c *gin.Context) {
	userID := c.Param("id")
	nodeID := c.Param("node_id")
	db := database.GetDB()

	if err := db.Where("user_id = ? AND custom_node_id = ?", userID, nodeID).Delete(&models.UserCustomNode{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "取消分配失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "取消分配成功",
	})
}

// 辅助函数
func parseUint(s string) uint {
	i, _ := strconv.ParseUint(s, 10, 32)
	return uint(i)
}

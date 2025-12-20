package handlers

import (
	"cboard-go/internal/models"
	"cboard-go/internal/services/custom_node"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func TestCustomNode(c *gin.Context) {
	id := c.Param("id")
	nodeID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的节点ID",
		})
		return
	}
	service := custom_node.NewCustomNodeService()
	result, err := service.TestCustomNode(uint(nodeID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "测试失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"node_id":   result.NodeID,
			"status":    result.Status,
			"latency":   result.Latency,
			"error":     result.Error,
			"tested_at": result.TestedAt.Format("2006-01-02 15:04:05"),
		},
	})
}

func GetCustomNodeLink(c *gin.Context) {
	id := c.Param("id")
	service := custom_node.NewCustomNodeService()
	var customNode models.CustomNode
	if err := service.GetDB().First(&customNode, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "专线节点不存在",
		})
		return
	}
	link, err := service.GenerateNodeLink(&customNode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "生成节点链接失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"link":    link,
			"name":    customNode.Name,
			"id":      customNode.ID,
			"node_id": customNode.ID,
		},
	})
}

func GetCustomNodes(c *gin.Context) {
	service := custom_node.NewCustomNodeService()
	var customNodes []models.CustomNode
	query := service.GetDB().Model(&models.CustomNode{})
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if isActive := c.Query("is_active"); isActive != "" {
		query = query.Where("is_active = ?", isActive == "true")
	}
	if search := c.Query("search"); search != "" {
		query = query.Where("name LIKE ? OR display_name LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if err := query.Order("created_at DESC").Find(&customNodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取专线节点列表失败",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    customNodes,
	})
}

func GetCustomNodeUsers(c *gin.Context) {
	id := c.Param("id")
	nodeID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的节点ID",
		})
		return
	}
	service := custom_node.NewCustomNodeService()
	users, err := service.GetCustomNodeUsers(uint(nodeID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取已分配用户列表失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    users,
	})
}

func CreateCustomNode(c *gin.Context) {
	var req struct {
		NodeLink         string     `json:"node_link"`
		Name             string     `json:"name"`
		DisplayName      string     `json:"display_name"`
		Protocol         string     `json:"protocol"`
		Config           string     `json:"config"`
		ExpireTime       *time.Time `json:"expire_time,omitempty"`
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
	service := custom_node.NewCustomNodeService()
	var customNode *models.CustomNode
	var err error
	if req.NodeLink != "" {
		customNode, err = service.CreateCustomNodeFromLink(req.NodeLink, req.Preview)
	} else {
		if req.Name == "" || req.Protocol == "" || req.Config == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "参数错误: name、protocol、config 为必填项",
			})
			return
		}
		createReq := &custom_node.CreateNodeRequest{
			Name:             req.Name,
			DisplayName:      req.DisplayName,
			Protocol:         req.Protocol,
			Config:           req.Config,
			ExpireTime:       req.ExpireTime,
			FollowUserExpire: req.FollowUserExpire,
		}
		customNode, err = service.CreateCustomNode(createReq)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建专线节点失败: " + err.Error(),
		})
		return
	}
	if req.Preview {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "解析成功",
			"data":    customNode,
		})
	} else {
		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"message": "专线节点创建成功",
			"data":    customNode,
		})
	}
}

func UpdateCustomNode(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name             string     `json:"name"`
		DisplayName      *string    `json:"display_name"` // 改为指针类型，支持清空
		ExpireTime       *time.Time `json:"expire_time"`
		FollowUserExpire *bool      `json:"follow_user_expire"`
		IsActive         *bool      `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}
	service := custom_node.NewCustomNodeService()
	var customNode models.CustomNode
	if err := service.GetDB().First(&customNode, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "专线节点不存在",
		})
		return
	}
	if req.Name != "" {
		customNode.Name = req.Name
	}
	if req.DisplayName != nil {
		customNode.DisplayName = *req.DisplayName
	}
	if req.ExpireTime != nil {
		customNode.ExpireTime = req.ExpireTime
	}
	if req.FollowUserExpire != nil {
		customNode.FollowUserExpire = *req.FollowUserExpire
	}
	if req.IsActive != nil {
		customNode.IsActive = *req.IsActive
	}
	if err := service.GetDB().Save(&customNode).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新专线节点失败",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    customNode,
	})
}

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
	service := custom_node.NewCustomNodeService()
	importedCount := 0
	errors := make([]string, 0)
	for _, link := range req.Links {
		link = strings.TrimSpace(link)
		if link == "" {
			continue
		}
		_, err := service.CreateCustomNodeFromLink(link, false)
		if err != nil {
			errors = append(errors, fmt.Sprintf("链接解析失败: %s", err.Error()))
			continue
		}
		importedCount++
	}
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     fmt.Sprintf("导入完成: 成功 %d 个, 失败 %d 个", importedCount, len(errors)),
		"imported":    importedCount,
		"errors":      errors,
		"error_count": len(errors),
	})
}

func DeleteCustomNode(c *gin.Context) {
	id := c.Param("id")
	service := custom_node.NewCustomNodeService()
	var customNode models.CustomNode
	if err := service.GetDB().First(&customNode, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "专线节点不存在",
		})
		return
	}
	service.GetDB().Where("custom_node_id = ?", id).Delete(&models.UserCustomNode{})
	if err := service.GetDB().Delete(&customNode).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除专线节点失败",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "删除成功",
	})
}

func AssignCustomNodeToUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的用户ID",
		})
		return
	}
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
	service := custom_node.NewCustomNodeService()
	if err := service.AssignNodeToUser(uint(userID), req.CustomNodeID, req.SubscriptionType, req.ExpiresAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "分配节点失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "分配成功",
	})
}

func UnassignCustomNodeFromUser(c *gin.Context) {
	userIDStr := c.Param("id")
	nodeIDStr := c.Param("node_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的用户ID",
		})
		return
	}
	nodeID, err := strconv.ParseUint(nodeIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的节点ID",
		})
		return
	}
	service := custom_node.NewCustomNodeService()
	if err := service.UnassignNodeFromUser(uint(userID), uint(nodeID)); err != nil {
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

func GetUserCustomNodes(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的用户ID",
		})
		return
	}
	service := custom_node.NewCustomNodeService()
	customNodes, err := service.GetUserCustomNodes(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取专线节点列表失败",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    customNodes,
	})
}

func BatchDeleteCustomNodes(c *gin.Context) {
	var req struct {
		NodeIDs []uint `json:"node_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}
	if len(req.NodeIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请选择要删除的节点",
		})
		return
	}
	service := custom_node.NewCustomNodeService()
	if err := service.BatchDeleteCustomNodes(req.NodeIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "批量删除失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "批量删除成功",
	})
}

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
			"message": "请求参数错误",
		})
		return
	}
	if len(req.NodeIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请选择要分配的节点",
		})
		return
	}
	if len(req.UserIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请选择要分配的用户",
		})
		return
	}
	service := custom_node.NewCustomNodeService()
	if err := service.BatchAssignNodesToUsers(req.NodeIDs, req.UserIDs, req.SubscriptionType, req.ExpiresAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "批量分配失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "批量分配成功",
	})
}

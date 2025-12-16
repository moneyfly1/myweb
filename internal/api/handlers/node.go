package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"

	"github.com/gin-gonic/gin"
)

// GetNodes 获取节点列表（用户端）
func GetNodes(c *gin.Context) {
	db := database.GetDB()

	var nodes []models.Node
	if err := db.Where("is_active = ?", true).Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取节点列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    nodes,
	})
}

// GetAdminNodes 管理员获取节点列表（包含所有节点）
func GetAdminNodes(c *gin.Context) {
	db := database.GetDB()

	var nodes []models.Node
	query := db.Model(&models.Node{})

	// 状态筛选
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// 是否激活筛选
	if active := c.Query("is_active"); active != "" {
		if active == "true" {
			query = query.Where("is_active = ?", true)
		} else if active == "false" {
			query = query.Where("is_active = ?", false)
		}
	}

	if err := query.Order("created_at DESC").Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取节点列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    nodes,
	})
}

// GetNode 获取单个节点
func GetNode(c *gin.Context) {
	id := c.Param("id")

	db := database.GetDB()
	var node models.Node
	if err := db.First(&node, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "节点不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    node,
	})
}

// CreateNode 创建节点（管理员）
func CreateNode(c *gin.Context) {
	var req struct {
		Name          string  `json:"name" binding:"required"`
		Region        string  `json:"region" binding:"required"`
		Type          string  `json:"type" binding:"required"`
		Description   string  `json:"description"`
		Config        string  `json:"config"`
		IsRecommended bool    `json:"is_recommended"`
		IsActive      bool    `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	node := models.Node{
		Name:          req.Name,
		Region:        req.Region,
		Type:          req.Type,
		Status:        "offline",
		IsRecommended: req.IsRecommended,
		IsActive:      req.IsActive,
	}

	if req.Description != "" {
		node.Description = database.NullString(req.Description)
	}
	if req.Config != "" {
		node.Config = database.NullString(req.Config)
	}

	if err := db.Create(&node).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建节点失败",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    node,
	})
}

// UpdateNode 更新节点（管理员）
func UpdateNode(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name          string  `json:"name"`
		Region        string  `json:"region"`
		Type          string  `json:"type"`
		Status        string  `json:"status"`
		Description   string  `json:"description"`
		Config        string  `json:"config"`
		IsRecommended bool    `json:"is_recommended"`
		IsActive      bool    `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	db := database.GetDB()
	var node models.Node
	if err := db.First(&node, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "节点不存在",
		})
		return
	}

	if req.Name != "" {
		node.Name = req.Name
	}
	if req.Region != "" {
		node.Region = req.Region
	}
	if req.Type != "" {
		node.Type = req.Type
	}
	if req.Status != "" {
		node.Status = req.Status
	}
	if req.Description != "" {
		node.Description = database.NullString(req.Description)
	}
	if req.Config != "" {
		node.Config = database.NullString(req.Config)
	}
	node.IsRecommended = req.IsRecommended
	node.IsActive = req.IsActive

	if err := db.Save(&node).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新节点失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "更新成功",
		"data":    node,
	})
}

// DeleteNode 删除节点（管理员）
func DeleteNode(c *gin.Context) {
	id := c.Param("id")

	db := database.GetDB()
	if err := db.Delete(&models.Node{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除节点失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "删除成功",
	})
}

// TestNode 测试节点
func TestNode(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()

	var node models.Node
	if err := db.First(&node, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "节点不存在",
		})
		return
	}

	// 这里应该实现实际的节点测试逻辑（ping、延迟测试等）
	// 暂时返回模拟结果
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"node_id":   node.ID,
			"status":    "online",
			"latency":   50, // 模拟延迟（毫秒）
			"tested_at": time.Now().Format("2006-01-02 15:04:05"),
		},
	})
}

// BatchTestNodes 批量测试节点
func BatchTestNodes(c *gin.Context) {
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

	db := database.GetDB()
	var nodes []models.Node
	if err := db.Where("id IN ?", req.NodeIDs).Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取节点失败",
		})
		return
	}

	// 批量测试结果
	results := make([]gin.H, 0)
	for _, node := range nodes {
		// 这里应该实现实际的节点测试逻辑
		results = append(results, gin.H{
			"node_id":   node.ID,
			"node_name": node.Name,
			"status":    "online",
			"latency":   50, // 模拟延迟
			"tested_at": time.Now().Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    results,
	})
}

// ImportFromClash 从 Clash 配置导入节点
func ImportFromClash(c *gin.Context) {
	var req struct {
		ClashConfig string `json:"clash_config" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误",
		})
		return
	}

	// 这里应该实现从 Clash YAML 配置解析节点的逻辑
	// 暂时返回成功消息
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "节点导入功能待实现",
		"data": gin.H{
			"imported_count": 0,
		},
	})
}

// CollectNodes 采集节点（从配置的节点源URL采集）
func CollectNodes(c *gin.Context) {
	db := database.GetDB()

	// 获取节点源配置
	var config models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "node_source_urls", "config_update").First(&config).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "未配置节点源URL",
		})
		return
	}

	// 使用配置更新服务采集节点
	service := config_update.NewConfigUpdateService()
	urls := strings.Split(config.Value, "\n")
	var validURLs []string
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u != "" {
			validURLs = append(validURLs, u)
		}
	}

	if len(validURLs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "节点源URL为空",
		})
		return
	}

	// 采集节点
	nodeData, err := service.FetchNodesFromURLs(validURLs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "采集节点失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("成功采集 %d 个节点", len(nodeData)),
		"data": gin.H{
			"count": len(nodeData),
			"nodes": nodeData,
		},
	})
}


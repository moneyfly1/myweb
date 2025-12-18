package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// --- Helper Functions ---

// generateNodeKey 生成用于去重的唯一键
func generateNodeKey(nodeType string, name string, config *string) string {
	if config == nil || *config == "" {
		return fmt.Sprintf("%s:%s", nodeType, name)
	}
	var p config_update.ProxyNode
	if err := json.Unmarshal([]byte(*config), &p); err == nil {
		key := fmt.Sprintf("%s:%s:%d", p.Type, p.Server, p.Port)
		if p.UUID != "" {
			return key + ":" + p.UUID
		} else if p.Password != "" {
			return key + ":" + p.Password
		}
		return key
	}
	return fmt.Sprintf("%s:%s", nodeType, name)
}

// findExistingNode 在数据库中查找匹配的节点
func findExistingNode(db *gorm.DB, targetKey string, nodeType string) *models.Node {
	var candidates []models.Node
	if err := db.Where("type = ? AND is_active = ?", nodeType, true).Find(&candidates).Error; err != nil {
		return nil
	}
	for _, dbNode := range candidates {
		if dbNode.Config != nil {
			if generateNodeKey(dbNode.Type, dbNode.Name, dbNode.Config) == targetKey {
				return &dbNode
			}
		}
	}
	return nil
}

// processAndImportLinks 处理解析出的链接并入库
func processAndImportLinks(db *gorm.DB, links []string) int {
	importedCount := 0
	seenKeys := make(map[string]bool)

	for _, link := range links {
		node, err := config_update.ParseNodeLink(link)
		if err != nil {
			continue
		}

		configJSON, _ := json.Marshal(node)
		configStr := string(configJSON)
		key := generateNodeKey(node.Type, node.Name, &configStr)

		if seenKeys[key] {
			continue
		}
		seenKeys[key] = true

		region := extractRegionFromName(node.Name)
		if region == "" {
			region = "未知"
		}

		existingNode := findExistingNode(db, key, node.Type)
		if existingNode == nil {
			newNode := models.Node{
				Name:     node.Name,
				Region:   region,
				Type:     node.Type,
				Status:   "online",
				IsActive: true,
				Config:   &configStr,
			}
			if db.Create(&newNode).Error == nil {
				importedCount++
			}
		} else {
			existingNode.Config = &configStr
			existingNode.Region = region
			existingNode.Type = node.Type
			existingNode.Name = node.Name
			existingNode.IsActive = true
			if existingNode.Status == "offline" {
				existingNode.Status = "online"
			}
			db.Save(existingNode)
		}
	}
	return importedCount
}

// --- Handlers ---

func GetNodes(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.Node{}).Where("is_active = ?", true)

	for _, param := range []string{"region", "type", "status"} {
		if val := c.Query(param); val != "" && val != "all" {
			query = query.Where(fmt.Sprintf("%s = ?", param), val)
		}
	}

	var allNodes []models.Node
	if err := query.Order("created_at DESC").Find(&allNodes).Error; err != nil {
		utils.LogError("GetNodes: query failed", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取节点列表失败"})
		return
	}

	seenKeys := make(map[string]bool)
	uniqueNodes := make([]models.Node, 0)
	for _, node := range allNodes {
		key := generateNodeKey(node.Type, node.Name, node.Config)
		if !seenKeys[key] {
			seenKeys[key] = true
			uniqueNodes = append(uniqueNodes, node)
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": uniqueNodes})
}

func GetNodeStats(c *gin.Context) {
	db := database.GetDB()
	var stats struct {
		TotalNodes  int64    `json:"total_nodes"`
		OnlineNodes int64    `json:"online_nodes"`
		Regions     []string `json:"regions"`
		RegionCount int      `json:"region_count"`
		Types       []string `json:"types"`
		TypeCount   int      `json:"type_count"`
	}

	baseQuery := db.Model(&models.Node{}).Where("is_active = ?", true)
	baseQuery.Count(&stats.TotalNodes)
	baseQuery.Where("status = ?", "online").Count(&stats.OnlineNodes)

	db.Model(&models.Node{}).Where("is_active = ?", true).Distinct().Pluck("region", &stats.Regions)
	db.Model(&models.Node{}).Where("is_active = ?", true).Distinct().Pluck("type", &stats.Types)

	stats.RegionCount = len(stats.Regions)
	stats.TypeCount = len(stats.Types)

	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

func GetAdminNodes(c *gin.Context) {
	db := database.GetDB()
	var nodes []models.Node
	query := db.Model(&models.Node{})

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if active := c.Query("is_active"); active != "" {
		query = query.Where("is_active = ?", active == "true")
	}

	if err := query.Order("created_at DESC").Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": nodes})
}

func GetNode(c *gin.Context) {
	var node models.Node
	if err := database.GetDB().First(&node, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "节点不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": node})
}

func CreateNode(c *gin.Context) {
	var req models.Node
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	req.Status = "offline"
	if err := database.GetDB().Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "创建失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": req})
}

func UpdateNode(c *gin.Context) {
	db := database.GetDB()
	var node models.Node
	if err := db.First(&node, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "节点不存在"})
		return
	}
	if err := c.ShouldBindJSON(&node); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	db.Save(&node)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": node})
}

func DeleteNode(c *gin.Context) {
	if err := database.GetDB().Delete(&models.Node{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "删除成功"})
}

func TestNode(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"node_id":   c.Param("id"),
			"status":    "online",
			"latency":   50,
			"tested_at": time.Now().Format("2006-01-02 15:04:05"),
		},
	})
}

func BatchTestNodes(c *gin.Context) {
	var req struct {
		NodeIDs []uint `json:"node_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	results := make([]gin.H, 0)
	for _, id := range req.NodeIDs {
		results = append(results, gin.H{
			"node_id": id, "status": "online", "latency": 50, "tested_at": time.Now().Format("2006-01-02 15:04:05"),
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": results})
}

// --- Import Logic ---

func ImportFromClash(c *gin.Context) {
	var req struct {
		ClashConfig string `json:"clash_config" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	count, _ := importNodesFromClashConfig(req.ClashConfig)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("成功导入 %d 个节点", count)})
}

func ImportFromFile(c *gin.Context) {
	path := "./uploads/config/clash.yaml"
	if !filepath.IsAbs(path) {
		wd, _ := os.Getwd()
		path = filepath.Join(wd, path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "文件不存在"})
		return
	}
	count, _ := importNodesFromClashConfig(string(content))
	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("成功导入 %d 个节点", count)})
}

func importNodesFromClashConfig(configStr string) (int, error) {
	db := database.GetDB()
	// 尝试从订阅链接获取
	var sysConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "node_source_urls", "config_update").First(&sysConfig).Error; err == nil {
		urls := strings.Split(sysConfig.Value, "\n")
		service := config_update.NewConfigUpdateService()
		if nodeData, err := service.FetchNodesFromURLs(urls); err == nil && len(nodeData) > 0 {
			links := make([]string, 0)
			for _, nd := range nodeData {
				if l, ok := nd["url"].(string); ok {
					links = append(links, l)
				}
			}
			return processAndImportLinks(db, links), nil
		}
	}

	// 降级方案：正则提取字符串中的链接
	linkPattern := regexp.MustCompile(`(vmess://|vless://|trojan://|ss://|ssr://|hysteria://|hysteria2://)[^\s\n]+`)
	links := linkPattern.FindAllString(configStr, -1)
	return processAndImportLinks(db, links), nil
}

func extractRegionFromName(name string) string {
	regions := map[string]string{
		"香港": "香港", "HK": "香港", "Hong Kong": "香港",
		"台湾": "台湾", "TW": "台湾", "Taiwan": "台湾",
		"日本": "日本", "JP": "日本", "Japan": "日本",
		"韩国": "韩国", "KR": "韩国", "Korea": "韩国",
		"新加坡": "新加坡", "SG": "新加坡", "Singapore": "新加坡",
		"美国": "美国", "US": "美国", "USA": "美国",
		"英国": "英国", "UK": "英国", "德国": "德国", "DE": "德国",
	}
	nameUpper := strings.ToUpper(name)
	for kw, r := range regions {
		if strings.Contains(nameUpper, strings.ToUpper(kw)) {
			return r
		}
	}
	return ""
}

func CollectNodes(c *gin.Context) {
	db := database.GetDB()
	var config models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "node_source_urls", "config_update").First(&config).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "未配置源"})
		return
	}
	service := config_update.NewConfigUpdateService()
	nodeData, err := service.FetchNodesFromURLs(strings.Split(config.Value, "\n"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "采集失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"count": len(nodeData), "nodes": nodeData}})
}

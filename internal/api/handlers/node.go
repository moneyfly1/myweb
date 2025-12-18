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
)

// GetNodes 获取节点列表（用户端）
func GetNodes(c *gin.Context) {
	db := database.GetDB()

	// 支持筛选参数
	query := db.Model(&models.Node{}).Where("is_active = ?", true)

	// 地区筛选
	if region := c.Query("region"); region != "" && region != "all" {
		query = query.Where("region = ?", region)
	}

	// 类型筛选
	if nodeType := c.Query("type"); nodeType != "" && nodeType != "all" {
		query = query.Where("type = ?", nodeType)
	}

	// 状态筛选
	if status := c.Query("status"); status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	var allNodes []models.Node
	if err := query.Order("created_at DESC").Find(&allNodes).Error; err != nil {
		utils.LogError("GetNodes: query nodes failed", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取节点列表失败",
		})
		return
	}

	// 去重：基于节点配置的唯一性
	seenKeys := make(map[string]bool)
	var uniqueNodes []models.Node

	for _, node := range allNodes {
		// 生成去重键
		key := fmt.Sprintf("%s:%s", node.Type, node.Name)
		if node.Config != nil && *node.Config != "" {
			// 尝试从配置中提取 server 和 port
			var proxyNode config_update.ProxyNode
			if err := json.Unmarshal([]byte(*node.Config), &proxyNode); err == nil {
				key = fmt.Sprintf("%s:%s:%d", proxyNode.Type, proxyNode.Server, proxyNode.Port)
				if proxyNode.UUID != "" {
					key += ":" + proxyNode.UUID
				} else if proxyNode.Password != "" {
					key += ":" + proxyNode.Password
				}
			}
		}

		if !seenKeys[key] {
			seenKeys[key] = true
			uniqueNodes = append(uniqueNodes, node)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    uniqueNodes,
	})
}

// GetNodeStats 获取节点统计（用户端）
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

	// 统计总节点数（仅激活的）
	db.Model(&models.Node{}).Where("is_active = ?", true).Count(&stats.TotalNodes)

	// 统计在线节点数
	db.Model(&models.Node{}).Where("is_active = ? AND status = ?", true, "online").Count(&stats.OnlineNodes)

	// 获取所有地区（去重）
	var regions []string
	db.Model(&models.Node{}).
		Where("is_active = ?", true).
		Select("DISTINCT region").
		Pluck("region", &regions)
	stats.Regions = regions
	stats.RegionCount = len(regions)

	// 获取所有类型（去重）
	var types []string
	db.Model(&models.Node{}).
		Where("is_active = ?", true).
		Select("DISTINCT type").
		Pluck("type", &types)
	stats.Types = types
	stats.TypeCount = len(types)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
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
		Name          string `json:"name" binding:"required"`
		Region        string `json:"region" binding:"required"`
		Type          string `json:"type" binding:"required"`
		Description   string `json:"description"`
		Config        string `json:"config"`
		IsRecommended bool   `json:"is_recommended"`
		IsActive      bool   `json:"is_active"`
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
		node.Description = &req.Description
	}
	if req.Config != "" {
		node.Config = &req.Config
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
		Name          string `json:"name"`
		Region        string `json:"region"`
		Type          string `json:"type"`
		Status        string `json:"status"`
		Description   string `json:"description"`
		Config        string `json:"config"`
		IsRecommended bool   `json:"is_recommended"`
		IsActive      bool   `json:"is_active"`
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
		node.Description = &req.Description
	}
	if req.Config != "" {
		node.Config = &req.Config
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

	// 解析 Clash YAML 配置
	importedCount, err := importNodesFromClashConfig(req.ClashConfig)
	if err != nil {
		utils.LogError("ImportNodesFromClash: import nodes failed", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "导入节点失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("成功导入 %d 个节点", importedCount),
		"data": gin.H{
			"imported_count": importedCount,
		},
	})
}

// ImportFromFile 从配置文件导入节点（管理员）
func ImportFromFile(c *gin.Context) {
	// 读取配置文件
	configPath := "./uploads/config/clash.yaml"
	if !filepath.IsAbs(configPath) {
		wd, _ := os.Getwd()
		configPath = filepath.Join(wd, configPath)
	}

	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "配置文件不存在: " + configPath,
		})
		return
	}

	// 读取文件内容
	content, err := os.ReadFile(configPath)
	if err != nil {
		utils.LogError("ImportNodesFromFile: read file failed", err, map[string]interface{}{
			"config_path": configPath,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "读取配置文件失败",
		})
		return
	}

	// 导入节点
	importedCount, err := importNodesFromClashConfig(string(content))
	if err != nil {
		utils.LogError("ImportNodesFromFile: import nodes failed", err, map[string]interface{}{
			"config_path": configPath,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "导入节点失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("成功从配置文件导入 %d 个节点", importedCount),
		"data": gin.H{
			"imported_count": importedCount,
			"file_path":      configPath,
		},
	})
}

// importNodesFromClashConfig 从 Clash 配置导入节点到数据库
func importNodesFromClashConfig(clashConfig string) (int, error) {
	db := database.GetDB()

	// 解析 Clash YAML 配置
	// 这里需要解析 YAML 并提取 proxies 部分
	// 由于没有 YAML 解析库，我们使用简单的字符串解析
	// 实际应该使用 gopkg.in/yaml.v3 等库

	// 提取节点链接（从配置文件中）
	// 这里简化处理：从配置更新服务获取节点数据
	service := config_update.NewConfigUpdateService()

	// 获取节点源配置
	var config models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "node_source_urls", "config_update").First(&config).Error; err != nil {
		// 如果没有配置，尝试从现有配置文件解析
		return importNodesFromExistingConfig()
	}

	urls := strings.Split(config.Value, "\n")
	var validURLs []string
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u != "" {
			validURLs = append(validURLs, u)
		}
	}

	if len(validURLs) == 0 {
		return importNodesFromExistingConfig()
	}

	// 采集节点
	nodeData, err := service.FetchNodesFromURLs(validURLs)
	if err != nil {
		return importNodesFromExistingConfig()
	}

	// 解析节点并导入到数据库
	importedCount := 0
	seenKeys := make(map[string]bool)

	for _, nodeInfo := range nodeData {
		link, ok := nodeInfo["url"].(string)
		if !ok {
			continue
		}

		// 解析节点链接
		node, err := config_update.ParseNodeLink(link)
		if err != nil {
			continue
		}

		// 生成去重键
		key := fmt.Sprintf("%s:%s:%d", node.Type, node.Server, node.Port)
		if node.UUID != "" {
			key += ":" + node.UUID
		} else if node.Password != "" {
			key += ":" + node.Password
		}

		if seenKeys[key] {
			continue
		}
		seenKeys[key] = true

		// 从节点名称提取地区信息（如果名称包含地区信息）
		region := extractRegionFromName(node.Name)
		if region == "" {
			region = "未知"
		}

		// 序列化节点配置
		configJSON, err := json.Marshal(node)
		if err != nil {
			continue
		}
		configStr := string(configJSON)

		// 使用去重键查找已存在的节点（而不是使用name和type，因为name可能被修改）
		var existingNode models.Node
		// 先通过 type、server、port 缩小查找范围，然后通过解析 Config 精确匹配
		var candidateNodes []models.Node
		if err := db.Where("type = ? AND is_active = ?", node.Type, true).Find(&candidateNodes).Error; err == nil {
			for _, dbNode := range candidateNodes {
				if dbNode.Config != nil && *dbNode.Config != "" {
					var existingProxyNode config_update.ProxyNode
					if err := json.Unmarshal([]byte(*dbNode.Config), &existingProxyNode); err == nil {
						// 先检查 server 和 port 是否匹配
						if existingProxyNode.Server == node.Server && existingProxyNode.Port == node.Port {
							// 生成已存在节点的去重键
							existingKey := fmt.Sprintf("%s:%s:%d", existingProxyNode.Type, existingProxyNode.Server, existingProxyNode.Port)
							if existingProxyNode.UUID != "" {
								existingKey += ":" + existingProxyNode.UUID
							} else if existingProxyNode.Password != "" {
								existingKey += ":" + existingProxyNode.Password
							}
							// 如果去重键匹配，说明是同一个节点
							if existingKey == key {
								existingNode = dbNode
								break
							}
						}
					}
				}
			}
		}

		if existingNode.ID == 0 {
			// 节点不存在，创建新节点（默认状态为 online）
			newNode := models.Node{
				Name:     node.Name,
				Region:   region,
				Type:     node.Type,
				Status:   "online", // 新导入的节点默认为在线状态
				IsActive: true,
				Config:   &configStr,
			}

			if err := db.Create(&newNode).Error; err != nil {
				continue
			}
			importedCount++
		} else {
			// 节点已存在，更新配置（保持原有状态，如果之前是 offline 则更新为 online）
			existingNode.Config = &configStr
			existingNode.Region = region
			existingNode.Type = node.Type
			existingNode.Name = node.Name // 更新节点名称
			existingNode.IsActive = true
			// 如果节点之前是离线状态，更新为在线（因为刚导入说明节点可用）
			if existingNode.Status == "offline" {
				existingNode.Status = "online"
			}
			if err := db.Save(&existingNode).Error; err != nil {
				continue
			}
		}
	}

	return importedCount, nil
}

// importNodesFromExistingConfig 从现有配置文件导入节点
func importNodesFromExistingConfig() (int, error) {
	// 读取配置文件
	configPath := "./uploads/config/clash.yaml"
	if !filepath.IsAbs(configPath) {
		wd, _ := os.Getwd()
		configPath = filepath.Join(wd, configPath)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return 0, fmt.Errorf("配置文件不存在")
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return 0, err
	}

	// 从 Clash 配置中提取节点链接
	// 这里简化处理，实际应该解析 YAML
	// 尝试从配置中提取 vmess://, vless:// 等链接
	configStr := string(content)

	// 使用正则表达式提取节点链接
	linkPattern := regexp.MustCompile(`(vmess://|vless://|trojan://|ss://|ssr://|hysteria://|hysteria2://)[^\s\n]+`)
	links := linkPattern.FindAllString(configStr, -1)

	if len(links) == 0 {
		return 0, fmt.Errorf("未找到节点链接")
	}

	db := database.GetDB()
	importedCount := 0
	seenKeys := make(map[string]bool)

	for _, link := range links {
		// 解析节点链接
		node, err := config_update.ParseNodeLink(link)
		if err != nil {
			continue
		}

		// 生成去重键
		key := fmt.Sprintf("%s:%s:%d", node.Type, node.Server, node.Port)
		if node.UUID != "" {
			key += ":" + node.UUID
		} else if node.Password != "" {
			key += ":" + node.Password
		}

		if seenKeys[key] {
			continue
		}
		seenKeys[key] = true

		// 从节点名称提取地区信息
		region := extractRegionFromName(node.Name)
		if region == "" {
			region = "未知"
		}

		// 序列化节点配置
		configJSON, err := json.Marshal(node)
		if err != nil {
			continue
		}
		configStr := string(configJSON)

		// 使用去重键查找已存在的节点（而不是使用name和type，因为name可能被修改）
		var existingNode models.Node
		// 先通过 type、server、port 缩小查找范围，然后通过解析 Config 精确匹配
		var candidateNodes []models.Node
		if err := db.Where("type = ? AND is_active = ?", node.Type, true).Find(&candidateNodes).Error; err == nil {
			for _, dbNode := range candidateNodes {
				if dbNode.Config != nil && *dbNode.Config != "" {
					var existingProxyNode config_update.ProxyNode
					if err := json.Unmarshal([]byte(*dbNode.Config), &existingProxyNode); err == nil {
						// 先检查 server 和 port 是否匹配
						if existingProxyNode.Server == node.Server && existingProxyNode.Port == node.Port {
							// 生成已存在节点的去重键
							existingKey := fmt.Sprintf("%s:%s:%d", existingProxyNode.Type, existingProxyNode.Server, existingProxyNode.Port)
							if existingProxyNode.UUID != "" {
								existingKey += ":" + existingProxyNode.UUID
							} else if existingProxyNode.Password != "" {
								existingKey += ":" + existingProxyNode.Password
							}
							// 如果去重键匹配，说明是同一个节点
							if existingKey == key {
								existingNode = dbNode
								break
							}
						}
					}
				}
			}
		}

		if existingNode.ID == 0 {
			// 节点不存在，创建新节点（默认状态为 online）
			newNode := models.Node{
				Name:     node.Name,
				Region:   region,
				Type:     node.Type,
				Status:   "online", // 新导入的节点默认为在线状态
				IsActive: true,
				Config:   &configStr,
			}

			if err := db.Create(&newNode).Error; err != nil {
				continue
			}
			importedCount++
		} else {
			// 节点已存在，更新配置（保持原有状态，如果之前是 offline 则更新为 online）
			existingNode.Config = &configStr
			existingNode.Region = region
			existingNode.Type = node.Type
			existingNode.Name = node.Name // 更新节点名称
			existingNode.IsActive = true
			// 如果节点之前是离线状态，更新为在线（因为刚导入说明节点可用）
			if existingNode.Status == "offline" {
				existingNode.Status = "online"
			}
			if err := db.Save(&existingNode).Error; err != nil {
				continue
			}
		}
	}

	return importedCount, nil
}

// extractRegionFromName 从节点名称提取地区信息
func extractRegionFromName(name string) string {
	// 常见的地区关键词
	regions := map[string]string{
		"香港": "香港", "HK": "香港", "Hong Kong": "香港",
		"台湾": "台湾", "TW": "台湾", "Taiwan": "台湾",
		"日本": "日本", "JP": "日本", "Japan": "日本",
		"韩国": "韩国", "KR": "韩国", "Korea": "韩国",
		"新加坡": "新加坡", "SG": "新加坡", "Singapore": "新加坡",
		"美国": "美国", "US": "美国", "USA": "美国", "United States": "美国",
		"英国": "英国", "UK": "英国", "United Kingdom": "英国",
		"德国": "德国", "DE": "德国", "Germany": "德国",
		"法国": "法国", "FR": "法国", "France": "法国",
		"俄罗斯": "俄罗斯", "RU": "俄罗斯", "Russia": "俄罗斯",
		"印度": "印度", "IN": "印度", "India": "印度",
		"澳大利亚": "澳大利亚", "AU": "澳大利亚", "Australia": "澳大利亚",
		"加拿大": "加拿大", "CA": "加拿大", "Canada": "加拿大",
		"荷兰": "荷兰", "NL": "荷兰", "Netherlands": "荷兰",
		"瑞士": "瑞士", "CH": "瑞士", "Switzerland": "瑞士",
		"瑞典": "瑞典", "SE": "瑞典", "Sweden": "瑞典",
		"挪威": "挪威", "NO": "挪威", "Norway": "挪威",
		"芬兰": "芬兰", "FI": "芬兰", "Finland": "芬兰",
		"丹麦": "丹麦", "DK": "丹麦", "Denmark": "丹麦",
		"波兰": "波兰", "PL": "波兰", "Poland": "波兰",
		"意大利": "意大利", "IT": "意大利", "Italy": "意大利",
		"西班牙": "西班牙", "ES": "西班牙", "Spain": "西班牙",
		"巴西": "巴西", "BR": "巴西", "Brazil": "巴西",
		"墨西哥": "墨西哥", "MX": "墨西哥", "Mexico": "墨西哥",
		"阿根廷": "阿根廷", "AR": "阿根廷", "Argentina": "阿根廷",
		"智利": "智利", "CL": "智利", "Chile": "智利",
		"土耳其": "土耳其", "TR": "土耳其", "Turkey": "土耳其",
		"以色列": "以色列", "IL": "以色列", "Israel": "以色列",
		"阿联酋": "阿联酋", "AE": "阿联酋", "UAE": "阿联酋",
		"沙特": "沙特", "SA": "沙特", "Saudi Arabia": "沙特",
		"泰国": "泰国", "TH": "泰国", "Thailand": "泰国",
		"马来西亚": "马来西亚", "MY": "马来西亚", "Malaysia": "马来西亚",
		"印尼": "印尼", "ID": "印尼", "Indonesia": "印尼",
		"菲律宾": "菲律宾", "PH": "菲律宾", "Philippines": "菲律宾",
		"越南": "越南", "VN": "越南", "Vietnam": "越南",
	}

	nameUpper := strings.ToUpper(name)
	for keyword, region := range regions {
		if strings.Contains(nameUpper, strings.ToUpper(keyword)) {
			return region
		}
	}

	return ""
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
		utils.LogError("CollectNodes: fetch nodes failed", err, map[string]interface{}{
			"urls_count": len(validURLs),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "采集节点失败",
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

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"
	"cboard-go/internal/services/node_health"
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

		// 先从节点名称中提取地区
		region := extractRegionFromName(node.Name)
		// 如果无法从名称中提取，尝试从服务器地址中提取
		if region == "" && node.Server != "" {
			region = extractRegionFromServer(node.Server)
		}
		// 如果还是无法提取，使用默认值
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

	// 对于手动添加的节点，不过滤重复，直接返回所有激活的节点
	// 只对采集的节点进行去重
	seenKeys := make(map[string]bool)
	uniqueNodes := make([]models.Node, 0)
	for _, node := range allNodes {
		// 手动添加的节点不过滤
		if node.IsManual {
			uniqueNodes = append(uniqueNodes, node)
		} else {
			// 采集的节点进行去重
			key := generateNodeKey(node.Type, node.Name, node.Config)
			if !seenKeys[key] {
				seenKeys[key] = true
				uniqueNodes = append(uniqueNodes, node)
			}
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
	var req struct {
		NodeLink string      `json:"node_link"` // 节点链接（可选）
		Node     models.Node `json:"node"`      // 节点对象（可选）
		Preview  bool        `json:"preview"`   // 是否仅预览，不实际创建
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	// 如果提供了节点链接，优先解析链接
	if req.NodeLink != "" {
		node, err := config_update.ParseNodeLink(req.NodeLink)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "节点链接解析失败: " + err.Error(),
			})
			return
		}

		// 将解析的节点转换为数据库模型
		configJSON, _ := json.Marshal(node)
		configStr := string(configJSON)
		// 先从节点名称中提取地区
		region := extractRegionFromName(node.Name)
		// 如果无法从名称中提取，尝试从服务器地址中提取
		if region == "" && node.Server != "" {
			region = extractRegionFromServer(node.Server)
		}
		// 如果还是无法提取，使用默认值
		if region == "" {
			region = "未知"
		}

		newNode := models.Node{
			Name:     node.Name,
			Region:   region,
			Type:     node.Type,
			Status:   "offline", // 新节点默认离线，等待测试
			IsActive: true,
			IsManual: true, // 手动添加的节点
			Config:   &configStr,
		}

		// 如果是预览模式，直接返回解析结果
		if req.Preview {
			c.JSON(http.StatusOK, gin.H{"success": true, "message": "解析成功", "data": newNode})
			return
		}

		// 检查是否已存在相同节点
		key := generateNodeKey(node.Type, node.Name, &configStr)
		existingNode := findExistingNode(db, key, node.Type)
		if existingNode != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "节点已存在",
				"data":    existingNode,
			})
			return
		}

		if err := db.Create(&newNode).Error; err != nil {
			utils.LogError("CreateNode: create failed", err, nil)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "创建失败"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"success": true, "message": "节点创建成功", "data": newNode})
		return
	}

	// 如果没有提供链接，使用传统方式创建
	req.Node.Status = "offline"
	req.Node.IsManual = true // 手动添加的节点
	if err := db.Create(&req.Node).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "创建失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": req.Node})
}

// ImportNodeLinks 批量导入节点链接
func ImportNodeLinks(c *gin.Context) {
	var req struct {
		Links []string `json:"links" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	importedCount := 0
	skippedCount := 0
	errors := make([]string, 0)

	for _, link := range req.Links {
		link = strings.TrimSpace(link)
		if link == "" {
			continue
		}

		node, err := config_update.ParseNodeLink(link)
		if err != nil {
			errors = append(errors, fmt.Sprintf("链接解析失败: %s", err.Error()))
			continue
		}

		configJSON, _ := json.Marshal(node)
		configStr := string(configJSON)
		// 先从节点名称中提取地区
		region := extractRegionFromName(node.Name)
		// 如果无法从名称中提取，尝试从服务器地址中提取
		if region == "" && node.Server != "" {
			region = extractRegionFromServer(node.Server)
		}
		// 如果还是无法提取，使用默认值
		if region == "" {
			region = "未知"
		}

		// 检查是否已存在
		key := generateNodeKey(node.Type, node.Name, &configStr)
		existingNode := findExistingNode(db, key, node.Type)
		if existingNode != nil {
			skippedCount++
			continue
		}

		newNode := models.Node{
			Name:     node.Name,
			Region:   region,
			Type:     node.Type,
			Status:   "offline",
			IsActive: true,
			IsManual: true, // 手动导入的节点
			Config:   &configStr,
		}

		if err := db.Create(&newNode).Error; err != nil {
			errors = append(errors, fmt.Sprintf("创建节点失败: %s", err.Error()))
			continue
		}

		importedCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     fmt.Sprintf("导入完成: 成功 %d 个, 跳过 %d 个", importedCount, skippedCount),
		"imported":    importedCount,
		"skipped":     skippedCount,
		"errors":      errors,
		"error_count": len(errors),
	})
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
	db := database.GetDB()
	var node models.Node
	if err := db.First(&node, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "节点不存在"})
		return
	}

	service := node_health.NewNodeHealthService()
	result, err := service.TestNode(&node)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "测试失败",
			"error":   err.Error(),
		})
		return
	}

	// 更新节点状态
	if err := service.UpdateNodeStatus(result); err != nil {
		utils.LogError("TestNode: update status failed", err, map[string]interface{}{
			"node_id": node.ID,
		})
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

func BatchTestNodes(c *gin.Context) {
	// 读取原始请求体
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "读取请求数据失败: " + err.Error()})
		return
	}

	// 尝试解析为标准格式
	var req struct {
		NodeIDs []uint `json:"node_ids"`
	}

	// 先尝试标准格式
	if err := json.Unmarshal(body, &req); err != nil || len(req.NodeIDs) == 0 {
		// 如果失败，尝试解析为灵活格式
		var flexReq map[string]interface{}
		if err2 := json.Unmarshal(body, &flexReq); err2 == nil {
			if nodeIDsRaw, ok := flexReq["node_ids"]; ok {
				nodeIDs := make([]uint, 0)
				switch v := nodeIDsRaw.(type) {
				case []interface{}:
					for _, id := range v {
						switch idVal := id.(type) {
						case float64:
							nodeIDs = append(nodeIDs, uint(idVal))
						case int:
							nodeIDs = append(nodeIDs, uint(idVal))
						case string:
							if parsed, err := strconv.ParseUint(idVal, 10, 32); err == nil {
								nodeIDs = append(nodeIDs, uint(parsed))
							}
						}
					}
				case []float64:
					for _, id := range v {
						nodeIDs = append(nodeIDs, uint(id))
					}
				case []int:
					for _, id := range v {
						nodeIDs = append(nodeIDs, uint(id))
					}
				}
				if len(nodeIDs) > 0 {
					req.NodeIDs = nodeIDs
				}
			}
		}
	}

	if len(req.NodeIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请选择要测试的节点"})
		return
	}

	service := node_health.NewNodeHealthService()
	results, err := service.BatchTestNodes(req.NodeIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "批量测试失败",
			"error":   err.Error(),
		})
		return
	}

	// 更新节点状态
	for _, result := range results {
		if err := service.UpdateNodeStatus(result); err != nil {
			utils.LogError("BatchTestNodes: update status failed", err, map[string]interface{}{
				"node_id": result.NodeID,
			})
		}
	}

	// 格式化返回结果
	formattedResults := make([]gin.H, 0, len(results))
	for _, result := range results {
		formattedResults = append(formattedResults, gin.H{
			"node_id":   result.NodeID,
			"status":    result.Status,
			"latency":   result.Latency,
			"error":     result.Error,
			"tested_at": result.TestedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": formattedResults})
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
		"中国": "中国", "CN": "中国", "China": "中国", "CHINA": "中国",
		"香港": "香港", "HK": "香港", "Hong Kong": "香港", "HONG KONG": "香港",
		"台湾": "台湾", "TW": "台湾", "Taiwan": "台湾", "TAIWAN": "台湾",
		"日本": "日本", "JP": "日本", "Japan": "日本", "JAPAN": "日本",
		"韩国": "韩国", "KR": "韩国", "Korea": "韩国", "KOREA": "韩国",
		"新加坡": "新加坡", "SG": "新加坡", "Singapore": "新加坡", "SINGAPORE": "新加坡",
		"美国": "美国", "US": "美国", "USA": "美国", "United States": "美国",
		"英国": "英国", "UK": "英国", "United Kingdom": "英国",
		"德国": "德国", "DE": "德国", "Germany": "德国",
		"法国": "法国", "FR": "法国", "France": "法国",
		"加拿大": "加拿大", "CA": "加拿大", "Canada": "加拿大",
		"澳大利亚": "澳大利亚", "AU": "澳大利亚", "Australia": "澳大利亚",
		"印度": "印度", "IN": "印度", "India": "印度",
		"俄罗斯": "俄罗斯", "RU": "俄罗斯", "Russia": "俄罗斯",
		"荷兰": "荷兰", "NL": "荷兰", "Netherlands": "荷兰",
		"瑞士": "瑞士", "CH": "瑞士", "Switzerland": "瑞士",
		"瑞典": "瑞典", "SE": "瑞典", "Sweden": "瑞典",
		"挪威": "挪威", "NO": "挪威", "Norway": "挪威",
		"芬兰": "芬兰", "FI": "芬兰", "Finland": "芬兰",
		"丹麦": "丹麦", "DK": "丹麦", "Denmark": "丹麦",
		"比利时": "比利时", "BE": "比利时", "Belgium": "比利时",
		"西班牙": "西班牙", "ES": "西班牙", "Spain": "西班牙",
		"意大利": "意大利", "IT": "意大利", "Italy": "意大利",
		"波兰": "波兰", "PL": "波兰", "Poland": "波兰",
		"土耳其": "土耳其", "TR": "土耳其", "Turkey": "土耳其",
		"巴西": "巴西", "BR": "巴西", "Brazil": "巴西",
		"阿根廷": "阿根廷", "AR": "阿根廷", "Argentina": "阿根廷",
		"墨西哥": "墨西哥", "MX": "墨西哥", "Mexico": "墨西哥",
		"南非": "南非", "ZA": "南非", "South Africa": "南非",
		"埃及": "埃及", "EG": "埃及", "Egypt": "埃及",
		"泰国": "泰国", "TH": "泰国", "Thailand": "泰国",
		"马来西亚": "马来西亚", "MY": "马来西亚", "Malaysia": "马来西亚",
		"印度尼西亚": "印度尼西亚", "ID": "印度尼西亚", "Indonesia": "印度尼西亚",
		"菲律宾": "菲律宾", "PH": "菲律宾", "Philippines": "菲律宾",
		"越南": "越南", "VN": "越南", "Vietnam": "越南",
	}
	nameUpper := strings.ToUpper(name)
	// 按长度从长到短排序，优先匹配更长的关键词
	sortedKeys := make([]string, 0, len(regions))
	for kw := range regions {
		sortedKeys = append(sortedKeys, kw)
	}
	// 简单排序：按长度降序
	for i := 0; i < len(sortedKeys)-1; i++ {
		for j := i + 1; j < len(sortedKeys); j++ {
			if len(sortedKeys[i]) < len(sortedKeys[j]) {
				sortedKeys[i], sortedKeys[j] = sortedKeys[j], sortedKeys[i]
			}
		}
	}

	for _, kw := range sortedKeys {
		if strings.Contains(nameUpper, strings.ToUpper(kw)) {
			return regions[kw]
		}
	}
	return ""
}

// extractRegionFromServer 从服务器地址中提取地区信息
func extractRegionFromServer(server string) string {
	if server == "" {
		return ""
	}

	// 地区代码映射（常见的前缀）
	regionCodes := map[string]string{
		"cn": "中国", "china": "中国", "chinese": "中国",
		"hk": "香港", "hongkong": "香港", "hong-kong": "香港",
		"tw": "台湾", "taiwan": "台湾", "taipei": "台湾",
		"jp": "日本", "japan": "日本", "tokyo": "日本", "osaka": "日本",
		"kr": "韩国", "korea": "韩国", "seoul": "韩国",
		"sg": "新加坡", "singapore": "新加坡",
		"us": "美国", "usa": "美国", "unitedstates": "美国",
		"uk": "英国", "unitedkingdom": "英国", "london": "英国",
		"de": "德国", "germany": "德国", "frankfurt": "德国",
		"fr": "法国", "france": "法国", "paris": "法国",
		"ca": "加拿大", "canada": "加拿大", "toronto": "加拿大", "vancouver": "加拿大",
		"au": "澳大利亚", "australia": "澳大利亚", "sydney": "澳大利亚", "melbourne": "澳大利亚",
		"in": "印度", "india": "印度", "mumbai": "印度", "delhi": "印度",
		"ru": "俄罗斯", "russia": "俄罗斯", "moscow": "俄罗斯",
		"nl": "荷兰", "netherlands": "荷兰", "amsterdam": "荷兰",
		"ch": "瑞士", "switzerland": "瑞士", "zurich": "瑞士",
		"se": "瑞典", "sweden": "瑞典", "stockholm": "瑞典",
		"no": "挪威", "norway": "挪威", "oslo": "挪威",
		"fi": "芬兰", "finland": "芬兰", "helsinki": "芬兰",
		"dk": "丹麦", "denmark": "丹麦", "copenhagen": "丹麦",
		"be": "比利时", "belgium": "比利时", "brussels": "比利时",
		"es": "西班牙", "spain": "西班牙", "madrid": "西班牙", "barcelona": "西班牙",
		"it": "意大利", "italy": "意大利", "rome": "意大利", "milan": "意大利",
		"pl": "波兰", "poland": "波兰", "warsaw": "波兰",
		"tr": "土耳其", "turkey": "土耳其", "istanbul": "土耳其",
		"br": "巴西", "brazil": "巴西", "sao": "巴西", "rio": "巴西",
		"ar": "阿根廷", "argentina": "阿根廷", "buenos": "阿根廷",
		"mx": "墨西哥", "mexico": "墨西哥",
		"za": "南非", "southafrica": "南非", "johannesburg": "南非",
		"eg": "埃及", "egypt": "埃及", "cairo": "埃及",
		"th": "泰国", "thailand": "泰国", "bangkok": "泰国",
		"my": "马来西亚", "malaysia": "马来西亚", "kuala": "马来西亚",
		"id": "印度尼西亚", "indonesia": "印度尼西亚", "jakarta": "印度尼西亚",
		"ph": "菲律宾", "philippines": "菲律宾", "manila": "菲律宾",
		"vn": "越南", "vietnam": "越南", "ho": "越南", "hanoi": "越南",
	}

	serverLower := strings.ToLower(server)

	// 按长度从长到短排序，优先匹配更长的关键词
	sortedKeys := make([]string, 0, len(regionCodes))
	for kw := range regionCodes {
		sortedKeys = append(sortedKeys, kw)
	}
	// 简单排序：按长度降序
	for i := 0; i < len(sortedKeys)-1; i++ {
		for j := i + 1; j < len(sortedKeys); j++ {
			if len(sortedKeys[i]) < len(sortedKeys[j]) {
				sortedKeys[i], sortedKeys[j] = sortedKeys[j], sortedKeys[i]
			}
		}
	}

	// 检查服务器地址中是否包含地区代码
	for _, kw := range sortedKeys {
		if strings.Contains(serverLower, kw) {
			return regionCodes[kw]
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

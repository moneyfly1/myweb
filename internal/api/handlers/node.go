package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"
	"cboard-go/internal/services/node_health"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	regionMap = map[string]string{
		"中国": "中国", "CN": "中国", "China": "中国", "香港": "香港", "HK": "香港", "Hong Kong": "香港",
		"台湾": "台湾", "TW": "台湾", "Taiwan": "台湾", "日本": "日本", "JP": "日本", "Japan": "日本",
		"韩国": "韩国", "KR": "韩国", "Korea": "韩国", "新加坡": "新加坡", "SG": "新加坡", "Singapore": "新加坡",
		"美国": "美国", "US": "美国", "USA": "美国", "英国": "英国", "UK": "英国", "德国": "德国", "DE": "德国",
		"法国": "法国", "FR": "法国", "加拿大": "加拿大", "CA": "加拿大", "澳洲": "澳大利亚", "AU": "澳大利亚",
		"印度": "印度", "IN": "印度", "俄罗斯": "俄罗斯", "RU": "俄罗斯", "荷兰": "荷兰", "NL": "荷兰",
		"泰国": "泰国", "TH": "泰国", "马来西亚": "马来西亚", "MY": "马来西亚", "越南": "越南", "VN": "越南",
		"菲律宾": "菲律宾", "PH": "菲律宾",
	}
	serverCodeMap = map[string]string{
		"tokyo": "日本", "osaka": "日本", "seoul": "韩国", "london": "英国", "frankfurt": "德国",
		"paris": "法国", "toronto": "加拿大", "sydney": "澳大利亚", "mumbai": "印度", "moscow": "俄罗斯",
		"amsterdam": "荷兰", "taipei": "台湾", "bangkok": "泰国", "hanoi": "越南",
	}
	sortedRegionKeys = func() []string {
		keys := make([]string, 0, len(regionMap))
		for k := range regionMap {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
		return keys
	}()
)

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

func resolveRegion(name, server string) string {
	nameUpper := strings.ToUpper(name)
	for _, kw := range sortedRegionKeys {
		if strings.Contains(nameUpper, strings.ToUpper(kw)) {
			return regionMap[kw]
		}
	}
	serverLower := strings.ToLower(server)
	for kw, region := range serverCodeMap {
		if strings.Contains(serverLower, kw) {
			return region
		}
	}
	return "未知"
}

func buildNodeModel(node *config_update.ProxyNode, isManual bool) models.Node {
	configJSON, _ := json.Marshal(node)
	configStr := string(configJSON)
	return models.Node{
		Name:     node.Name,
		Region:   resolveRegion(node.Name, node.Server),
		Type:     node.Type,
		Status:   "offline",
		IsActive: true,
		IsManual: isManual,
		Config:   &configStr,
	}
}

func findExistingNode(db *gorm.DB, targetKey string, nodeType string) *models.Node {
	var candidates []models.Node
	if err := db.Where("type = ? AND is_active = ?", nodeType, true).Find(&candidates).Error; err != nil {
		return nil
	}
	for _, dbNode := range candidates {
		if dbNode.Config != nil && generateNodeKey(dbNode.Type, dbNode.Name, dbNode.Config) == targetKey {
			return &dbNode
		}
	}
	return nil
}

func processAndImportLinks(db *gorm.DB, links []string) int {
	importedCount := 0
	seenKeys := make(map[string]bool)
	for _, link := range links {
		parsed, err := config_update.ParseNodeLink(link)
		if err != nil {
			continue
		}
		newNode := buildNodeModel(parsed, false)
		key := generateNodeKey(newNode.Type, newNode.Name, newNode.Config)
		if seenKeys[key] {
			continue
		}
		seenKeys[key] = true
		if existing := findExistingNode(db, key, newNode.Type); existing == nil {
			newNode.Status = "online"
			if db.Create(&newNode).Error == nil {
				importedCount++
			}
		} else {
			existing.Config, existing.Region, existing.Type, existing.Name = newNode.Config, newNode.Region, newNode.Type, newNode.Name
			existing.IsActive = true
			if existing.Status == "offline" {
				existing.Status = "online"
			}
			db.Save(existing)
		}
	}
	return importedCount
}

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
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取失败"})
		return
	}
	seenKeys := make(map[string]bool)
	uniqueNodes := make([]models.Node, 0)
	for _, node := range allNodes {
		if node.IsManual {
			uniqueNodes = append(uniqueNodes, node)
		} else {
			key := generateNodeKey(node.Type, node.Name, node.Config)
			if !seenKeys[key] {
				seenKeys[key] = true
				uniqueNodes = append(uniqueNodes, node)
			}
		}
	}
	if user, ok := middleware.GetCurrentUser(c); ok && user != nil {
		var sub models.Subscription
		isOrdExpired := true
		if err := db.Where("user_id = ? AND status = ?", user.ID, "active").First(&sub).Error; err == nil {
			isOrdExpired = sub.ExpireTime.Before(utils.GetBeijingTime())
		}
		if user.SpecialNodeSubscriptionType == "special_only" || isOrdExpired {
			uniqueNodes = make([]models.Node, 0)
		}
		var nodeIDs []uint
		db.Model(&models.UserCustomNode{}).Where("user_id = ?", user.ID).Pluck("custom_node_id", &nodeIDs)
		if len(nodeIDs) > 0 {
			var customNodes []models.CustomNode
			if err := db.Where("id IN ? AND is_active = ?", nodeIDs, true).Find(&customNodes).Error; err == nil {
				now := utils.GetBeijingTime()
				for _, cn := range customNodes {
					isSpecExpired := false
					if cn.FollowUserExpire {
						if user.SpecialNodeExpiresAt.Valid {
							isSpecExpired = user.SpecialNodeExpiresAt.Time.Before(now)
						} else {
							isSpecExpired = isOrdExpired
						}
					} else if cn.ExpireTime != nil {
						isSpecExpired = cn.ExpireTime.Before(now)
					}
					if isSpecExpired {
						continue
					}
					var nc models.NodeConfig
					if err := json.Unmarshal([]byte(cn.Config), &nc); err == nil {
						pn := config_update.ProxyNode{
							Type:     nc.Type,
							Server:   nc.Server,
							Port:     nc.Port,
							UUID:     nc.UUID,
							Password: nc.Password,
							Network:  nc.Network,
							Cipher:   nc.Encryption,
							TLS:      nc.Security == "tls",
						}
						cfgJSON, _ := json.Marshal(pn)
						cfgStr := string(cfgJSON)
						name := cn.DisplayName
						if name == "" {
							name = "专线定制-" + cn.Name
						}
					// 设置最后测试时间
					var lastTest *time.Time
					if cn.LastTest != nil {
						lastTest = cn.LastTest
					}
					
					uniqueNodes = append(uniqueNodes, models.Node{
						ID:       cn.ID + 1000000,
						Name:     name,
						Type:     cn.Protocol,
						Region:   cn.Domain,
						Status:   cn.Status, // 使用 CustomNode 自身的 status
						Latency:  cn.Latency, // 使用 CustomNode 的延迟
						LastTest: lastTest,    // 使用 CustomNode 的最后测试时间
						IsActive: true,
						IsManual: true,
						Config:   &cfgStr,
					})
					}
				}
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
	base := db.Model(&models.Node{}).Where("is_active = ?", true)
	base.Count(&stats.TotalNodes)
	base.Where("status = ?", "online").Count(&stats.OnlineNodes)
	db.Model(&models.Node{}).Where("is_active = ?", true).Distinct().Pluck("region", &stats.Regions)
	db.Model(&models.Node{}).Where("is_active = ?", true).Distinct().Pluck("type", &stats.Types)
	if user, ok := middleware.GetCurrentUser(c); ok && user != nil {
		var nodeIDs []uint
		db.Model(&models.UserCustomNode{}).Where("user_id = ?", user.ID).Pluck("custom_node_id", &nodeIDs)
		if len(nodeIDs) > 0 {
			var cns []models.CustomNode
			db.Where("id IN ? AND is_active = ?", nodeIDs, true).Find(&cns)
			for _, n := range cns {
				stats.TotalNodes++
				stats.OnlineNodes++
				reg := n.Domain
				if reg == "" {
					reg = "专线"
				}
				foundR := false
				for _, r := range stats.Regions {
					if r == reg {
						foundR = true
						break
					}
				}
				if !foundR {
					stats.Regions = append(stats.Regions, reg)
				}
			}
		}
	}
	stats.RegionCount, stats.TypeCount = len(stats.Regions), len(stats.Types)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
}

func GetAdminNodes(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.Node{})
	
	// 状态筛选
	if s := c.Query("status"); s != "" {
		query = query.Where("status = ?", s)
	}
	
	// 激活状态筛选（默认只显示激活的节点，与用户端保持一致）
	// 如果前端明确传递 is_active=false，则显示未激活的节点
	if a := c.Query("is_active"); a != "" {
		query = query.Where("is_active = ?", a == "true")
	} else {
		// 默认只显示激活的节点
		query = query.Where("is_active = ?", true)
	}
	
	// 地区筛选
	if r := c.Query("region"); r != "" {
		query = query.Where("region = ?", r)
	}
	
	// 类型筛选
	if t := c.Query("type"); t != "" {
		query = query.Where("type = ?", t)
	}
	
	// 搜索关键词
	if search := c.Query("search"); search != "" {
		search = utils.SanitizeSearchKeyword(search)
		if search != "" {
			query = query.Where("name LIKE ?", "%"+search+"%")
		}
	}
	
	// 先查询所有符合条件的节点（不分页，用于去重）
	var allNodes []models.Node
	if err := query.Order("created_at DESC").Find(&allNodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取节点列表失败"})
		return
	}
	
	// 去重处理（与用户端逻辑保持一致）
	seenKeys := make(map[string]bool)
	uniqueNodes := make([]models.Node, 0)
	for _, node := range allNodes {
		if node.IsManual {
			// 手动添加的节点直接添加，不去重
			uniqueNodes = append(uniqueNodes, node)
		} else {
			// 非手动节点使用 generateNodeKey 去重
			key := generateNodeKey(node.Type, node.Name, node.Config)
			if !seenKeys[key] {
				seenKeys[key] = true
				uniqueNodes = append(uniqueNodes, node)
			}
		}
	}
	
	// 计算去重后的总数
	total := int64(len(uniqueNodes))
	
	// 分页参数
	page := 1
	size := 20
	if pageStr := c.Query("page"); pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if sizeStr := c.Query("size"); sizeStr != "" {
		fmt.Sscanf(sizeStr, "%d", &size)
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100 // 限制最大每页数量
	}
	
	// 分页处理
	offset := (page - 1) * size
	end := offset + size
	if end > len(uniqueNodes) {
		end = len(uniqueNodes)
	}
	if offset >= len(uniqueNodes) {
		uniqueNodes = []models.Node{}
	} else {
		uniqueNodes = uniqueNodes[offset:end]
	}
	
	c.JSON(http.StatusOK, gin.H{"success": true, "data": uniqueNodes, "total": total, "page": page, "size": size})
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
		NodeLink string      `json:"node_link"`
		Node     models.Node `json:"node"`
		Preview  bool        `json:"preview"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	db := database.GetDB()
	if req.NodeLink != "" {
		parsed, err := config_update.ParseNodeLink(req.NodeLink)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "解析失败"})
			return
		}
		newNode := buildNodeModel(parsed, true)
		if req.Preview {
			c.JSON(http.StatusOK, gin.H{"success": true, "data": newNode})
			return
		}
		key := generateNodeKey(newNode.Type, newNode.Name, newNode.Config)
		if existing := findExistingNode(db, key, newNode.Type); existing != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "节点已存在"})
			return
		}
		db.Create(&newNode)
		c.JSON(http.StatusCreated, gin.H{"success": true, "data": newNode})
		return
	}
	req.Node.Status, req.Node.IsManual = "offline", true
	db.Create(&req.Node)
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": req.Node})
}

func ImportNodeLinks(c *gin.Context) {
	var req struct {
		Links []string `json:"links" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	db := database.GetDB()
	imp, skp := 0, 0
	for _, link := range req.Links {
		if parsed, err := config_update.ParseNodeLink(strings.TrimSpace(link)); err == nil {
			node := buildNodeModel(parsed, true)
			if findExistingNode(db, generateNodeKey(node.Type, node.Name, node.Config), node.Type) == nil {
				if db.Create(&node).Error == nil {
					imp++
					continue
				}
			}
			skp++
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("成功 %d, 跳过 %d", imp, skp)})
}

func UpdateNode(c *gin.Context) {
	db := database.GetDB()
	var node models.Node
	if db.First(&node, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "不存在"})
		return
	}
	c.ShouldBindJSON(&node)
	db.Save(&node)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": node})
}

func DeleteNode(c *gin.Context) {
	database.GetDB().Delete(&models.Node{}, c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "删除成功"})
}

func TestNode(c *gin.Context) {
	nodeIDStr := c.Param("id")
	nodeID, err := strconv.ParseUint(nodeIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的节点ID"})
		return
	}

	db := database.GetDB()
	svc := node_health.NewNodeHealthService()

	// 判断是否为专线节点（ID > 1000000）
	if nodeID > 1000000 {
		// 专线节点：从 custom_nodes 表查询
		customNodeID := uint(nodeID - 1000000)
		var customNode models.CustomNode
		if db.First(&customNode, customNodeID).Error != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "专线节点不存在"})
			return
		}

		// 构建临时 Node 对象用于测试
		var nc models.NodeConfig
		if err := json.Unmarshal([]byte(customNode.Config), &nc); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "解析节点配置失败"})
			return
		}

		cfgJSON, _ := json.Marshal(config_update.ProxyNode{
			Type:     nc.Type,
			Server:   nc.Server,
			Port:     nc.Port,
			UUID:     nc.UUID,
			Password: nc.Password,
			Network:  nc.Network,
			Cipher:   nc.Encryption,
			TLS:      nc.Security == "tls",
		})
		cfgStr := string(cfgJSON)

		tempNode := models.Node{
			ID:     uint(nodeID),
			Config: &cfgStr,
		}

		// 测试节点
		res, err := svc.TestNode(&tempNode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}

		// 更新专线节点的状态和延迟
		now := utils.GetBeijingTime()
		customNode.Status = res.Status
		customNode.Latency = res.Latency
		customNode.LastTest = &now
		db.Save(&customNode)

		c.JSON(http.StatusOK, gin.H{"success": true, "data": res})
		return
	}

	// 普通节点：从 nodes 表查询
	var node models.Node
	if db.First(&node, nodeID).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "节点不存在"})
		return
	}
	
	res, err := svc.TestNode(&node)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	svc.UpdateNodeStatus(res)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": res})
}

func BatchTestNodes(c *gin.Context) {
	var req struct {
		NodeIDs []uint `json:"node_ids"`
	}
	body, _ := c.GetRawData()
	if err := json.Unmarshal(body, &req); err != nil || len(req.NodeIDs) == 0 {
		var flex map[string]interface{}
		if err2 := json.Unmarshal(body, &flex); err2 == nil {
			if idsRaw, ok := flex["node_ids"]; ok {
				if ids, ok := idsRaw.([]interface{}); ok {
					for _, id := range ids {
						if val, err := strconv.Atoi(fmt.Sprint(id)); err == nil {
							req.NodeIDs = append(req.NodeIDs, uint(val))
						}
					}
				}
			}
		}
	}
	if len(req.NodeIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "未选择节点"})
		return
	}
	svc := node_health.NewNodeHealthService()
	results, _ := svc.BatchTestNodes(req.NodeIDs)
	for _, res := range results {
		svc.UpdateNodeStatus(res)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": results})
}

func ImportFromClash(c *gin.Context) {
	var req struct {
		ClashConfig string `json:"clash_config" binding:"required"`
	}
	c.ShouldBindJSON(&req)
	count, _ := importNodesFromClashConfig(req.ClashConfig)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("导入 %d 个", count)})
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
	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("导入 %d 个", count)})
}

func importNodesFromClashConfig(configStr string) (int, error) {
	db := database.GetDB()
	var sysConfig models.SystemConfig
	if db.Where("key = ? AND category = ?", "node_source_urls", "config_update").First(&sysConfig).Error == nil {
		svc := config_update.NewConfigUpdateService()
		if nodeData, err := svc.FetchNodesFromURLs(strings.Split(sysConfig.Value, "\n")); err == nil {
			links := make([]string, 0)
			for _, nd := range nodeData {
				if l, ok := nd["url"].(string); ok {
					links = append(links, l)
				}
			}
			return processAndImportLinks(db, links), nil
		}
	}
	linkPattern := regexp.MustCompile(`(vmess|vless|trojan|ss|ssr|hysteria2?)://[^\s\n]+`)
	return processAndImportLinks(db, linkPattern.FindAllString(configStr, -1)), nil
}

func CollectNodes(c *gin.Context) {
	var config models.SystemConfig
	db := database.GetDB()
	if db.Where("key = ? AND category = ?", "node_source_urls", "config_update").First(&config).Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "未配置源"})
		return
	}
	svc := config_update.NewConfigUpdateService()
	data, err := svc.FetchNodesFromURLs(strings.Split(config.Value, "\n"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "采集失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"count": len(data), "nodes": data}})
}

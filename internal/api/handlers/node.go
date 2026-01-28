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
	if err := query.Order("order_index ASC, created_at ASC").Find(&allNodes).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取失败", err)
		return
	}
	seenKeys := make(map[string]bool)
	uniqueNodes := make([]models.Node, 0)
	var customNodesList []models.Node // 专线节点列表

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
		hasOrdSubscription := false
		if err := db.Where("user_id = ? AND status = ?", user.ID, "active").First(&sub).Error; err == nil {
			hasOrdSubscription = true
			isOrdExpired = sub.ExpireTime.Before(utils.GetBeijingTime())
		}

		now := utils.GetBeijingTime()

		var specialExpireTime time.Time
		hasSpecialExpireTime := false
		if user.SpecialNodeExpiresAt.Valid {
			specialExpireTime = user.SpecialNodeExpiresAt.Time
			hasSpecialExpireTime = true
		} else if hasOrdSubscription {
			specialExpireTime = sub.ExpireTime
			hasSpecialExpireTime = true
		}
		isSpecialExpired := hasSpecialExpireTime && specialExpireTime.Before(now)

		if user.SpecialNodeSubscriptionType == "special_only" {
			uniqueNodes = make([]models.Node, 0)
			utils.LogInfo("GetNodes: 用户 %s (ID: %d) 订阅类型为 special_only，只显示专线节点", user.Username, user.ID)
		} else if user.SpecialNodeSubscriptionType == "both" {
			if isOrdExpired {
				uniqueNodes = make([]models.Node, 0)
				utils.LogInfo("GetNodes: 用户 %s (ID: %d) 订阅类型为 both，但普通订阅已过期，只显示专线节点", user.Username, user.ID)
			} else {
				utils.LogInfo("GetNodes: 用户 %s (ID: %d) 订阅类型为 both，显示普通节点+专线节点", user.Username, user.ID)
			}
		} else {
			if isOrdExpired {
				uniqueNodes = make([]models.Node, 0)
			}
		}

		var nodeIDs []uint
		db.Model(&models.UserCustomNode{}).Where("user_id = ?", user.ID).Pluck("custom_node_id", &nodeIDs)
		if len(nodeIDs) > 0 {
			var customNodes []models.CustomNode
			if err := db.Where("id IN ? AND is_active = ?", nodeIDs, true).Find(&customNodes).Error; err == nil {
				for _, cn := range customNodes {
					isSpecNodeExpired := false
					if cn.FollowUserExpire {
						isSpecNodeExpired = isSpecialExpired
					} else if cn.ExpireTime != nil {
						isSpecNodeExpired = cn.ExpireTime.Before(now)
					} else {
						isSpecNodeExpired = isSpecialExpired
					}

					if isSpecNodeExpired {
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
						var lastTest *time.Time
						if cn.LastTest != nil {
							lastTest = cn.LastTest
						}

						customNodesList = append(customNodesList, models.Node{
							ID:         cn.ID + 1000000,
							Name:       name,
							Type:       cn.Protocol,
							Region:     cn.Domain,
							Status:     cn.Status,  // 使用 CustomNode 自身的 status
							Latency:    cn.Latency, // 使用 CustomNode 的延迟
							LastTest:   lastTest,   // 使用 CustomNode 的最后测试时间
							IsActive:   true,
							IsManual:   true,
							Config:     &cfgStr,
							OrderIndex: -1, // 专线节点使用 -1，确保显示在最前面
						})
					}
				}
			}
		}
	}

	finalNodes := append(customNodesList, uniqueNodes...)

	if user, ok := middleware.GetCurrentUser(c); ok && user != nil {
		utils.LogInfo("GetNodes: 用户 %s (ID: %d) 订阅类型=%s, 返回节点数: 专线=%d, 普通=%d, 总计=%d",
			user.Username, user.ID, user.SpecialNodeSubscriptionType,
			len(customNodesList), len(uniqueNodes), len(finalNodes))
	}

	utils.SuccessResponse(c, http.StatusOK, "", finalNodes)
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
	utils.SuccessResponse(c, http.StatusOK, "", stats)
}

func GetAdminNodes(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.Node{})

	if s := c.Query("status"); s != "" {
		query = query.Where("status = ?", s)
	}

	if a := c.Query("is_active"); a != "" {
		query = query.Where("is_active = ?", a == "true")
	} else {
		query = query.Where("is_active = ?", true)
	}

	if r := c.Query("region"); r != "" {
		query = query.Where("region = ?", r)
	}

	if t := c.Query("type"); t != "" {
		query = query.Where("type = ?", t)
	}

	if search := c.Query("search"); search != "" {
		search = utils.SanitizeSearchKeyword(search)
		if search != "" {
			query = query.Where("name LIKE ?", "%"+search+"%")
		}
	}

	var allNodes []models.Node
	if err := query.Order("order_index ASC, created_at ASC").Find(&allNodes).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取节点列表失败", err)
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

	total := int64(len(uniqueNodes))

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

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"data":  uniqueNodes,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

func GetNode(c *gin.Context) {
	var node models.Node
	if err := database.GetDB().First(&node, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "节点不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取节点失败", err)
		}
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", node)
}

func CreateNode(c *gin.Context) {
	var req struct {
		NodeLink string      `json:"node_link"`
		Node     models.Node `json:"node"`
		Preview  bool        `json:"preview"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}
	db := database.GetDB()
	if req.NodeLink != "" {
		parsed, err := config_update.ParseNodeLink(req.NodeLink)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "解析失败", err)
			return
		}
		newNode := buildNodeModel(parsed, true)
		if req.Preview {
			utils.SuccessResponse(c, http.StatusOK, "", newNode)
			return
		}
		key := generateNodeKey(newNode.Type, newNode.Name, newNode.Config)
		if existing := findExistingNode(db, key, newNode.Type); existing != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "节点已存在", nil)
			return
		}
		if err := db.Create(&newNode).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "创建节点失败", err)
			return
		}
		utils.SuccessResponse(c, http.StatusCreated, "", newNode)
		return
	}
	req.Node.Status, req.Node.IsManual = "offline", true
	if err := db.Create(&req.Node).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建节点失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusCreated, "", req.Node)
}

func ImportNodeLinks(c *gin.Context) {
	var req struct {
		Links []string `json:"links" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
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
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功 %d, 跳过 %d", imp, skp), gin.H{
		"imported": imp,
		"skipped":  skp,
	})
}

func UpdateNode(c *gin.Context) {
	db := database.GetDB()
	var node models.Node
	if err := db.First(&node, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "节点不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取节点失败", err)
		}
		return
	}
	if err := c.ShouldBindJSON(&node); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}
	if err := db.Save(&node).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "更新节点失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "更新成功", node)
}

func DeleteNode(c *gin.Context) {
	if err := database.GetDB().Delete(&models.Node{}, c.Param("id")).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除节点失败", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "删除成功", nil)
}

func TestNode(c *gin.Context) {
	nodeIDStr := c.Param("id")
	nodeID, err := strconv.ParseUint(nodeIDStr, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的节点ID", err)
		return
	}

	db := database.GetDB()
	svc := node_health.NewNodeHealthService()

	if nodeID > 1000000 {
		customNodeID := uint(nodeID - 1000000)
		var customNode models.CustomNode
		if err := db.First(&customNode, customNodeID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				utils.ErrorResponse(c, http.StatusNotFound, "专线节点不存在", err)
			} else {
				utils.ErrorResponse(c, http.StatusInternalServerError, "获取专线节点失败", err)
			}
			return
		}

		var nc models.NodeConfig
		if err := json.Unmarshal([]byte(customNode.Config), &nc); err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "解析节点配置失败", err)
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

		res, err := svc.TestNode(&tempNode)
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "测试节点失败", err)
			return
		}

		now := utils.GetBeijingTime()
		customNode.Status = res.Status
		customNode.Latency = res.Latency
		customNode.LastTest = &now
		if err := db.Save(&customNode).Error; err != nil {
			utils.LogError("TestNode: save custom node failed", err, nil)
		}

		utils.SuccessResponse(c, http.StatusOK, "", res)
		return
	}

	var node models.Node
	if err := db.First(&node, nodeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "节点不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取节点失败", err)
		}
		return
	}

	res, err := svc.TestNode(&node)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "测试节点失败", err)
		return
	}
	svc.UpdateNodeStatus(res)
	utils.SuccessResponse(c, http.StatusOK, "", res)
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
		utils.ErrorResponse(c, http.StatusBadRequest, "未选择节点", nil)
		return
	}
	svc := node_health.NewNodeHealthService()
	results, _ := svc.BatchTestNodes(req.NodeIDs)
	for _, res := range results {
		svc.UpdateNodeStatus(res)
	}
	utils.SuccessResponse(c, http.StatusOK, "", results)
}

func BatchDeleteNodes(c *gin.Context) {
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
		utils.ErrorResponse(c, http.StatusBadRequest, "未选择节点", nil)
		return
	}

	db := database.GetDB()

	var normalNodeIDs []uint
	var customNodeIDs []uint

	for _, nodeID := range req.NodeIDs {
		if nodeID > 1000000 {
			customNodeIDs = append(customNodeIDs, nodeID-1000000)
		} else {
			normalNodeIDs = append(normalNodeIDs, nodeID)
		}
	}

	deletedCount := 0

	if len(normalNodeIDs) > 0 {
		result := db.Where("id IN ?", normalNodeIDs).Delete(&models.Node{})
		if result.Error != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "删除节点失败", result.Error)
			return
		}
		deletedCount += int(result.RowsAffected)
	}

	if len(customNodeIDs) > 0 {
		result := db.Where("id IN ?", customNodeIDs).Delete(&models.CustomNode{})
		if result.Error != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "删除专线节点失败", result.Error)
			return
		}
		deletedCount += int(result.RowsAffected)
	}

	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功删除 %d 个节点", deletedCount), gin.H{"deleted_count": deletedCount})
}

func ImportFromClash(c *gin.Context) {
	var req struct {
		ClashConfig string `json:"clash_config" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}
	count, _ := importNodesFromClashConfig(req.ClashConfig)
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("导入 %d 个", count), gin.H{"count": count})
}

func ImportFromFile(c *gin.Context) {
	path := "./uploads/config/clash.yaml"
	if !filepath.IsAbs(path) {
		wd, _ := os.Getwd()
		path = filepath.Join(wd, path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "文件不存在", err)
		return
	}
	count, _ := importNodesFromClashConfig(string(content))
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("导入 %d 个", count), gin.H{"count": count})
}

func importNodesFromClashConfig(configStr string) (int, error) {
	db := database.GetDB()
	var sysConfig models.SystemConfig
	if db.Where("key = ? AND category = ?", "urls", "config_update").First(&sysConfig).Error == nil {
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

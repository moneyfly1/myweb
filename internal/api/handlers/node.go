package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	"cboard-go/internal/services/cache_service"
	"cboard-go/internal/services/config_update"
	"cboard-go/internal/services/node_health"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// clearNodeCaches 同步清除所有节点相关缓存，确保管理员操作后客户立即获取最新数据
func clearNodeCaches() {
	cs := cache_service.NewCacheService()
	if err := cs.ClearNodesCache(); err != nil {
		log.Printf("failed to clear nodes cache: %v", err)
	}
	cacheService := &config_update.CacheService{}
	if err := cacheService.ClearSystemNodesCache(); err != nil {
		log.Printf("failed to clear system nodes cache: %v", err)
	}
	if err := cacheService.ClearAllSubscriptionCache(); err != nil {
		log.Printf("failed to clear all subscription cache: %v", err)
	}
}

var (
	regionMatcherOnce sync.Once
	regionMatcher     *config_update.RegionMatcher
)

func getRegionMatcher() *config_update.RegionMatcher {
	regionMatcherOnce.Do(func() {
		cfg, err := config_update.LoadRegionConfig()
		if err != nil {
			utils.LogWarn("node handlers 地区配置加载失败: %v", err)
		}
		if cfg == nil {
			regionMatcher = config_update.NewRegionMatcher(map[string]string{}, map[string]string{})
			return
		}
		regionMatcher = config_update.NewRegionMatcher(cfg.RegionMap, cfg.ServerMap)
	})
	return regionMatcher
}

func generateNodeKey(nodeType string, name string, config *string) string {
	if config == nil || *config == "" {
		return fmt.Sprintf("%s:%s", nodeType, name)
	}
	var p config_update.ProxyNode
	if err := json.Unmarshal([]byte(*config), &p); err == nil {
		key := fmt.Sprintf("%s:%s:%d:%s", p.Type, p.Server, p.Port, p.Name)
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
	matcher := getRegionMatcher()
	if matcher == nil {
		return "未知"
	}
	return matcher.MatchRegion(name, server)
}

// truncateNodeName 将节点名称截断到 varchar(100) 长度以内（按 rune 计数）。
// MySQL 严格模式下超长名称会导致写入失败；SQLite 不校验长度，本地不会复现此问题。
func truncateNodeName(name string) string {
	const maxLen = 100
	runes := []rune(name)
	if len(runes) <= maxLen {
		return name
	}
	return string(runes[:maxLen])
}

func buildNodeModel(node *config_update.ProxyNode, isManual bool) models.Node {
	node.Name = truncateNodeName(node.Name)
	// #nosec G117 - Password field is proxy node password, not user credential
	configJSON, _ := json.Marshal(node) // #nosec G117
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

func findNodeIDsByKey(db *gorm.DB, key string) []uint {
	var nodes []models.Node
	if err := db.Find(&nodes).Error; err != nil {
		return nil
	}

	ids := make([]uint, 0)
	for _, node := range nodes {
		if generateNodeKey(node.Type, node.Name, node.Config) == key {
			ids = append(ids, node.ID)
		}
	}
	return ids
}

func collectEquivalentNodeIDs(db *gorm.DB, selectedIDs []uint) ([]uint, error) {
	if len(selectedIDs) == 0 {
		return nil, nil
	}

	var selected []models.Node
	if err := db.Where("id IN ?", selectedIDs).Find(&selected).Error; err != nil {
		return nil, err
	}

	// 一次性加载全部节点并建立 key -> ids 映射，
	// 避免对每个选中节点都 findNodeIDsByKey 全表扫描（原实现为 O(选中数 × 全表)）
	var allNodes []models.Node
	if err := db.Find(&allNodes).Error; err != nil {
		return nil, err
	}
	keyToIDs := make(map[string][]uint, len(allNodes))
	for _, n := range allNodes {
		k := generateNodeKey(n.Type, n.Name, n.Config)
		keyToIDs[k] = append(keyToIDs[k], n.ID)
	}

	idSet := make(map[uint]bool)
	for _, node := range selected {
		idSet[node.ID] = true
		for _, id := range keyToIDs[generateNodeKey(node.Type, node.Name, node.Config)] {
			idSet[id] = true
		}
	}

	ids := make([]uint, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	return ids, nil
}

func processAndImportLinks(db *gorm.DB, links []string) int {
	// 预加载所有活跃节点到 map，避免循环内 N+1 查询
	var allActive []models.Node
	db.Where("is_active = ?", true).Find(&allActive)
	existingMap := make(map[string]*models.Node)
	for i := range allActive {
		key := generateNodeKey(allActive[i].Type, allActive[i].Name, allActive[i].Config)
		existingMap[key] = &allActive[i]
	}

	importedCount := 0
	seenKeys := make(map[string]bool)
	var newNodes []models.Node

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
		if existing := existingMap[key]; existing == nil {
			newNode.Status = "online"
			newNodes = append(newNodes, newNode)
		} else {
			existing.Config, existing.Region, existing.Type, existing.Name = newNode.Config, newNode.Region, newNode.Type, newNode.Name
			existing.IsActive = true
			if existing.Status == "offline" {
				existing.Status = "online"
			}
			db.Save(existing)
		}
	}

	if len(newNodes) > 0 {
		if err := db.CreateInBatches(newNodes, 100).Error; err == nil {
			importedCount = len(newNodes)
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
		if user.SpecialNodeSubscriptionType == "special_only" || user.SpecialNodeSubscriptionType == "both" {
			db.Model(&models.UserCustomNode{}).Where("user_id = ?", user.ID).Pluck("custom_node_id", &nodeIDs)
		}
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
						// #nosec G117 - Password field is proxy node password, not user credential
						cfgJSON, _ := json.Marshal(pn) // #nosec G117
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

	// 对普通用户进行配置脱敏
	isAdmin := false
	if user, ok := middleware.GetCurrentUser(c); ok && user != nil {
		isAdmin = user.IsAdmin
	}

	if !isAdmin {
		for i := range finalNodes {
			finalNodes[i].Config = nil
		}
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
	if user, ok := middleware.GetCurrentUser(c); ok && user != nil && (user.SpecialNodeSubscriptionType == "special_only" || user.SpecialNodeSubscriptionType == "both") {
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
	}

	if r := c.Query("region"); r != "" {
		query = query.Where("region = ?", r)
	}

	if t := c.Query("type"); t != "" {
		query = query.Where("type = ?", t)
	}

	if m := c.Query("is_manual"); m != "" {
		query = query.Where("is_manual = ?", m == "true")
	}

	if search := c.Query("search"); search != "" {
		search = utils.SanitizeSearchKeyword(search)
		if search != "" {
			escapedSearch := utils.EscapeLikePattern(search)
			searchPattern := "%" + escapedSearch + "%"
			query = query.Where("name LIKE ? OR config LIKE ?", searchPattern, searchPattern)
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
		_, _ = fmt.Sscanf(pageStr, "%d", &page) // Ignore error, use default value
	}
	if sizeStr := c.Query("size"); sizeStr != "" {
		_, _ = fmt.Sscanf(sizeStr, "%d", &size) // Ignore error, use default value
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

	// 对非管理员用户进行脱敏处理
	isAdmin := false
	if user, ok := middleware.GetCurrentUser(c); ok && user != nil {
		isAdmin = user.IsAdmin
	}

	if !isAdmin {
		node.Config = nil
	}

	utils.SuccessResponse(c, http.StatusOK, "", node)
}

func CreateNode(c *gin.Context) {
	var req struct {
		NodeLink string      `json:"node_link"`
		Node     models.Node `json:"node"`
		Preview  bool        `json:"preview"`
	}
	body, _ := c.GetRawData()
	if err := json.Unmarshal(body, &req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}
	if req.NodeLink == "" && req.Node.Name == "" {
		var node models.Node
		if err := json.Unmarshal(body, &node); err == nil {
			req.Node = node
		}
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
		utils.CreateAuditLogSimple(c, "create_node", "node", newNode.ID, fmt.Sprintf("管理员操作: 创建节点 %s", newNode.Name))

		// 清除节点相关缓存
		clearNodeCaches()

		utils.SuccessResponse(c, http.StatusCreated, "", newNode)
		return
	}
	if strings.TrimSpace(req.Node.Name) == "" || strings.TrimSpace(req.Node.Region) == "" || strings.TrimSpace(req.Node.Type) == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "请填写节点名称、地区和类型", nil)
		return
	}
	req.Node.Name = truncateNodeName(req.Node.Name)
	req.Node.Status, req.Node.IsManual, req.Node.IsActive = "offline", true, true

	// 读取 manual_node_position 配置，设置手动节点的 order_index
	var posConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "manual_node_position", "config_update").First(&posConfig).Error; err == nil {
		if pos, err := strconv.Atoi(posConfig.Value); err == nil && pos >= 0 {
			orderIndex := pos*10000 - 5000
			if pos == 0 {
				orderIndex = -500
			}
			req.Node.OrderIndex = orderIndex
		}
	}

	if err := db.Create(&req.Node).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建节点失败", err)
		return
	}
	utils.CreateAuditLogSimple(c, "create_node", "node", req.Node.ID, fmt.Sprintf("管理员操作: 创建节点 %s", req.Node.Name))

	// 清除节点相关缓存
	clearNodeCaches()

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
	imp, skp, fail, failReasons := importNodeLinks(db, req.Links)
	utils.CreateAuditLogSimple(c, "import_node_links", "node", 0, fmt.Sprintf("管理员操作: 导入节点链接 成功 %d 跳过 %d 失败 %d", imp, skp, fail))

	// 清除节点相关缓存
	clearNodeCaches()

	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("成功 %d, 跳过 %d, 失败 %d", imp, skp, fail), gin.H{
		"imported": imp,
		"skipped":  skp,
		"failed":   fail,
		"errors":   failReasons,
	})
}

// importNodeLinks 批量解析并创建普通节点，返回成功/跳过/失败数与失败原因。
func importNodeLinks(db *gorm.DB, links []string) (imp, skp, fail int, failReasons []string) {
	for _, link := range links {
		parsed, err := config_update.ParseNodeLink(strings.TrimSpace(link))
		if err != nil {
			// 解析失败单独统计并返回原因，避免静默失败让用户以为导入成功
			fail++
			if len(failReasons) < 3 {
				failReasons = append(failReasons, fmt.Sprintf("解析失败: %v", err))
			}
			continue
		}
		node := buildNodeModel(parsed, true)
		if findExistingNode(db, generateNodeKey(node.Type, node.Name, node.Config), node.Type) != nil {
			skp++
			continue
		}
		if err := db.Create(&node).Error; err != nil {
			fail++
			if len(failReasons) < 3 {
				failReasons = append(failReasons, fmt.Sprintf("写入失败: %v", err))
			}
			continue
		}
		imp++
	}
	return
}

// ImportNodeSubscription 通过订阅地址导入普通节点（替代手动填写）。
func ImportNodeSubscription(c *gin.Context) {
	var req struct {
		URL string `json:"url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	urlStr := strings.TrimSpace(req.URL)
	if urlStr == "" || (!strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://")) {
		utils.ErrorResponse(c, http.StatusBadRequest, "请输入有效的 http/https 订阅链接", nil)
		return
	}

	svc := config_update.NewConfigUpdateService()
	nodes, err := svc.FetchNodesFromURLs([]string{urlStr})
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取订阅失败", err)
		return
	}
	if len(nodes) == 0 {
		utils.SuccessResponse(c, http.StatusOK, "", gin.H{
			"imported": 0, "error_count": 0, "errors": []string{},
			"message": "订阅获取失败或内容中没有解析到节点，请检查订阅链接是否可访问",
		})
		return
	}

	links := make([]string, 0, len(nodes))
	seen := make(map[string]bool)
	for _, n := range nodes {
		link, _ := n["url"].(string)
		link = strings.TrimSpace(link)
		if link != "" && !seen[link] {
			seen[link] = true
			links = append(links, link)
		}
	}
	if len(links) == 0 {
		utils.SuccessResponse(c, http.StatusOK, "", gin.H{
			"imported": 0, "error_count": 0, "errors": []string{},
			"message": "订阅内容中没有解析到节点",
		})
		return
	}

	db := database.GetDB()
	imp, skp, fail, failReasons := importNodeLinks(db, links)
	utils.CreateAuditLogSimple(c, "import_node_subscription", "node", 0,
		fmt.Sprintf("管理员操作: 导入节点订阅 %s 解析 %d 个 成功 %d 跳过 %d 失败 %d", urlStr, len(links), imp, skp, fail))
	if imp > 0 {
		clearNodeCaches()
	}
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("订阅解析出 %d 个节点，成功导入 %d 个", len(links), imp), gin.H{
		"imported": imp, "skipped": skp, "failed": fail, "errors": failReasons, "total": len(links),
		"message": fmt.Sprintf("订阅解析出 %d 个节点，成功导入 %d 个", len(links), imp),
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
	utils.CreateAuditLogSimple(c, "update_node", "node", node.ID, fmt.Sprintf("管理员操作: 更新节点 %s", node.Name))

	// 清除节点和订阅配置缓存
	clearNodeCaches()

	utils.SuccessResponse(c, http.StatusOK, "更新成功", node)
}

func GetNodeLink(c *gin.Context) {
	db := database.GetDB()
	var node models.Node
	if err := db.First(&node, c.Param("id")).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "节点不存在", err)
		return
	}

	if node.Config == nil || *node.Config == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "节点配置为空", nil)
		return
	}

	var proxyNode config_update.ProxyNode
	if err := json.Unmarshal([]byte(*node.Config), &proxyNode); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "解析节点配置失败", err)
		return
	}

	// 如果直接解析失败（字段为空），尝试用通用 map 解析
	if proxyNode.Server == "" || proxyNode.Type == "" {
		var rawConfig map[string]interface{}
		if err := json.Unmarshal([]byte(*node.Config), &rawConfig); err == nil {
			if s, ok := rawConfig["Server"].(string); ok && s != "" {
				proxyNode.Server = s
			}
			if t, ok := rawConfig["Type"].(string); ok && t != "" {
				proxyNode.Type = t
			}
			if p, ok := rawConfig["Port"].(float64); ok {
				proxyNode.Port = int(p)
			}
			if u, ok := rawConfig["UUID"].(string); ok {
				proxyNode.UUID = u
			}
			if pw, ok := rawConfig["Password"].(string); ok {
				proxyNode.Password = pw
			}
			if ci, ok := rawConfig["Cipher"].(string); ok {
				proxyNode.Cipher = ci
			}
			if nw, ok := rawConfig["Network"].(string); ok {
				proxyNode.Network = nw
			}
			if tls, ok := rawConfig["TLS"].(bool); ok {
				proxyNode.TLS = tls
			}
			if opts, ok := rawConfig["Options"].(map[string]interface{}); ok && proxyNode.Options == nil {
				proxyNode.Options = opts
			}
		}
	}

	if proxyNode.Name == "" {
		proxyNode.Name = node.Name
	}

	service := config_update.NewConfigUpdateService()
	link := service.NodeToLink(&proxyNode)
	if link == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "不支持的节点类型", nil)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", map[string]string{"link": link})
}

func DeleteNode(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var node models.Node
	if err := db.First(&node, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "节点不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取节点失败", err)
		}
		return
	}

	key := generateNodeKey(node.Type, node.Name, node.Config)
	nodeIDs := findNodeIDsByKey(db, key)
	if len(nodeIDs) == 0 {
		nodeIDs = []uint{node.ID}
	}

	result := db.Where("id IN ?", nodeIDs).Delete(&models.Node{})
	if result.Error != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除节点失败", result.Error)
		return
	}

	// 清除节点和订阅配置缓存
	clearNodeCaches()

	utils.CreateAuditLogSimple(c, "delete_node", "node", node.ID, fmt.Sprintf("管理员操作: 删除节点 %s，实际删除 %d 条", node.Name, result.RowsAffected))
	utils.SuccessResponse(c, http.StatusOK, "删除成功", gin.H{"deleted_count": result.RowsAffected})
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

		// #nosec G117 - Password field is proxy node password, not user credential
		cfgJSON, _ := json.Marshal(config_update.ProxyNode{ // #nosec G117
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
		clearNodeCaches()

		utils.CreateAuditLogSimple(c, "test_custom_node", "custom_node", customNode.ID, fmt.Sprintf("管理员操作: 测试专线节点 %s 结果 %s 延迟 %dms", customNode.Name, res.Status, res.Latency))

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
	if err := svc.UpdateNodeStatus(res); err != nil {
		log.Printf("failed to update node status: %v", err)
	}
	clearNodeCaches()
	utils.CreateAuditLogSimple(c, "test_node", "node", node.ID, fmt.Sprintf("管理员操作: 测试节点 %s 结果 %s 延迟 %dms", node.Name, res.Status, res.Latency))
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
							req.NodeIDs = append(req.NodeIDs, utils.MustSafeIntToUint(val))
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
	svc := node_health.NewNodeHealthService()

	// 分离专线虚拟 ID（>1000000）：service 的 BatchTestNodes 只查普通节点表，
	// 专线节点逐个走 TestNode（其内部已处理虚拟 ID 分支，见 TestNode 实现）
	var normalIDs, customIDs []uint
	for _, id := range req.NodeIDs {
		if id > 1000000 {
			customIDs = append(customIDs, id)
		} else {
			normalIDs = append(normalIDs, id)
		}
	}

	results := make([]*node_health.TestResult, 0, len(req.NodeIDs))
	if len(normalIDs) > 0 {
		normalResults, err := svc.BatchTestNodes(normalIDs)
		if err != nil {
			utils.LogError("BatchTestNodes: batch test normal nodes failed", err, nil)
		} else {
			results = append(results, normalResults...)
		}
	}
	for _, virtualID := range customIDs {
		res := testNodeByVirtualID(c, db, virtualID)
		if res != nil {
			results = append(results, res)
		}
	}

	for _, res := range results {
		if res == nil {
			continue
		}
		if err := svc.UpdateNodeStatus(res); err != nil {
			log.Printf("failed to update node status: %v", err)
		}
	}
	clearNodeCaches()
	successCount := 0
	for _, res := range results {
		if res != nil && res.Status == "online" {
			successCount++
		}
	}
	utils.CreateAuditLogSimple(c, "batch_test_nodes", "node", 0, fmt.Sprintf("管理员操作: 批量测试节点 %d 个 在线 %d 个", len(req.NodeIDs), successCount))
	utils.SuccessResponse(c, http.StatusOK, "", results)
}

// testNodeByVirtualID 测试单个专线虚拟 ID（>1000000）对应的专线节点，返回测试结果
func testNodeByVirtualID(c *gin.Context, db *gorm.DB, virtualID uint) *node_health.TestResult {
	customNodeID := virtualID - 1000000
	var customNode models.CustomNode
	if err := db.First(&customNode, customNodeID).Error; err != nil {
		return nil
	}
	var nc models.NodeConfig
	if err := json.Unmarshal([]byte(customNode.Config), &nc); err != nil {
		return nil
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
	tempNode := models.Node{ID: customNode.ID, Config: &cfgStr}

	svc := node_health.NewNodeHealthService()
	res, err := svc.TestNode(&tempNode)
	if err != nil {
		return &node_health.TestResult{
			NodeID:   virtualID,
			Status:   "offline",
			Error:    err.Error(),
			TestedAt: utils.GetBeijingTime(),
		}
	}
	res.NodeID = virtualID
	// 同步更新专线节点状态
	now := utils.GetBeijingTime()
	customNode.Status = res.Status
	customNode.Latency = res.Latency
	customNode.LastTest = &now
	if saveErr := db.Save(&customNode).Error; saveErr != nil {
		utils.LogError("testNodeByVirtualID: save custom node failed", saveErr, map[string]interface{}{
			"custom_node_id": customNode.ID,
		})
	}
	return res
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
							req.NodeIDs = append(req.NodeIDs, utils.MustSafeIntToUint(val))
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

	// 虚拟 ID 约定：>1000000 的 ID 表示专线节点 ID + 1000000（见 GetUserNodes 合并列表）。
	// 先验证这些 ID 在 custom_nodes 中确实存在；不存在的按普通节点 ID 处理，
	// 避免普通节点 ID 超过 1000000 时被误判为专线节点，导致批量删除 0 行。
	var maybeCustomIDs []uint
	for _, nodeID := range req.NodeIDs {
		if nodeID > 1000000 {
			maybeCustomIDs = append(maybeCustomIDs, nodeID-1000000)
		} else {
			normalNodeIDs = append(normalNodeIDs, nodeID)
		}
	}
	if len(maybeCustomIDs) > 0 {
		var existingCustomIDs []uint
		db.Model(&models.CustomNode{}).Where("id IN ?", maybeCustomIDs).Pluck("id", &existingCustomIDs)
		existingSet := make(map[uint]bool, len(existingCustomIDs))
		for _, id := range existingCustomIDs {
			existingSet[id] = true
		}
		for _, baseID := range maybeCustomIDs {
			if existingSet[baseID] {
				customNodeIDs = append(customNodeIDs, baseID)
			} else {
				// custom_nodes 中不存在 → 实际是 ID 超过 1000000 的普通节点
				normalNodeIDs = append(normalNodeIDs, baseID+1000000)
			}
		}
	}

	// 删除前记录节点名称，用于操作日志
	var deletingNodeNames []string
	if len(normalNodeIDs) > 0 {
		var normalNodes []models.Node
		db.Where("id IN ?", normalNodeIDs).Find(&normalNodes)
		for _, n := range normalNodes {
			if n.Name != "" {
				deletingNodeNames = append(deletingNodeNames, n.Name)
			}
		}
	}
	if len(customNodeIDs) > 0 {
		var customNodes []models.CustomNode
		db.Where("id IN ?", customNodeIDs).Find(&customNodes)
		for _, n := range customNodes {
			if n.Name != "" {
				deletingNodeNames = append(deletingNodeNames, n.Name+"(专线)")
			}
		}
	}

	deletedCount := 0

	if len(normalNodeIDs) > 0 {
		equivalentIDs, err := collectEquivalentNodeIDs(db, normalNodeIDs)
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取节点失败", err)
			return
		}
		if len(equivalentIDs) == 0 {
			// 所选节点已不存在（如页面展示旧数据时重复删除），返回成功并提示刷新
			utils.SuccessResponse(c, http.StatusOK, "未删除任何节点：所选节点已不存在，请刷新列表", gin.H{"deleted_count": 0})
			return
		}
		result := db.Where("id IN ?", equivalentIDs).Delete(&models.Node{})
		if result.Error != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "删除节点失败", result.Error)
			return
		}
		deletedCount += int(result.RowsAffected)
	}

	if len(customNodeIDs) > 0 {
		// 先记录关联用户，删除分配关系后同步清理其缓存，避免用户订阅残留已删除专线节点
		var affectedUserIDs []uint
		db.Model(&models.UserCustomNode{}).Where("custom_node_id IN ?", customNodeIDs).Distinct("user_id").Pluck("user_id", &affectedUserIDs)

		db.Where("custom_node_id IN ?", customNodeIDs).Delete(&models.UserCustomNode{})

		result := db.Where("id IN ?", customNodeIDs).Delete(&models.CustomNode{})
		if result.Error != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "删除专线节点失败", result.Error)
			return
		}
		deletedCount += int(result.RowsAffected)

		for _, uid := range affectedUserIDs {
			clearUserCustomNodeCache(uid)
			resetSpecialNodeFieldsIfNoCustomNodes(db, uid)
		}
	}

	if deletedCount == 0 {
		// 所选节点已不存在（如页面展示旧数据时重复删除），返回成功并提示刷新，
		// 避免前端把"无节点可删"当成删除失败
		utils.SuccessResponse(c, http.StatusOK, "未删除任何节点：所选节点已不存在，请刷新列表", gin.H{"deleted_count": 0})
		return
	}

	// 清理所有节点和订阅缓存
	clearNodeCaches()

	utils.CreateAuditLogSimple(c, "batch_delete_nodes", "node", 0, fmt.Sprintf("管理员操作: 批量删除节点 %d 个 [%s]", deletedCount, truncateDesc(strings.Join(deletingNodeNames, "、"))))
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

	// 清理所有节点和订阅缓存
	clearNodeCaches()

	utils.CreateAuditLogSimple(c, "import_from_clash", "node", 0, fmt.Sprintf("管理员操作: 从 Clash 配置导入节点 %d 个", count))
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

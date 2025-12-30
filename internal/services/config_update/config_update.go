package config_update

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

// ==========================================
// Constants & Variables
// ==========================================

// SubscriptionStatus 订阅状态枚举
type SubscriptionStatus int

const (
	StatusNormal          SubscriptionStatus = iota
	StatusExpired                            // 过期
	StatusInactive                           // 失效（被禁用）
	StatusAccountAbnormal                    // 账户异常（被禁用）
	StatusDeviceOverLimit                    // 设备超限
	StatusOldAddress                         // 旧订阅地址
	StatusNotFound                           // 订阅不存在
)

// 预编译正则表达式以提升性能
// 注意：需要匹配完整的链接，包括参数部分（?和#之后的内容）
// 重要：使用 (^|\s) 确保前面是行首或空白字符，避免被其他协议包含（如vmess://包含ss://）
var nodeLinkPatterns = []*regexp.Regexp{
	// VMess/VLESS: Base64编码的JSON，可能包含参数
	// 使用 (^|\s) 确保前面是行首或空白字符，避免被其他协议包含
	regexp.MustCompile(`(?:^|\s)(vmess://[^\s]+)`),
	regexp.MustCompile(`(?:^|\s)(vless://[^\s]+)`),
	// Trojan: UUID@服务器:端口?参数#名称
	regexp.MustCompile(`(?:^|\s)(trojan://[^\s]+)`),
	// SS: 加密方法:密码@服务器:端口#名称 或 Base64编码格式
	// 特别注意：vmess://包含ss://，需要通过位置跟踪避免误匹配
	regexp.MustCompile(`(?:^|\s)(ss://[^\s]+)`),
	// SSR: Base64编码
	regexp.MustCompile(`(?:^|\s)(ssr://[^\s]+)`),
	// Hysteria: 可能包含参数
	regexp.MustCompile(`(?:^|\s)(hysteria://[^\s]+)`),
	regexp.MustCompile(`(?:^|\s)(hysteria2://[^\s]+)`),
	// TUIC: 可能包含参数
	regexp.MustCompile(`(?:^|\s)(tuic://[^\s]+)`),
	// Naive: 可能包含参数
	regexp.MustCompile(`(?:^|\s)(naive\+https://[^\s]+)`),
	regexp.MustCompile(`(?:^|\s)(naive://[^\s]+)`),
	// Anytls: 可能包含参数
	regexp.MustCompile(`(?:^|\s)(anytls://[^\s]+)`),
}

// Clash 支持的节点类型
var supportedClashTypes = map[string]bool{
	"vmess":     true,
	"vless":     true,
	"trojan":    true,
	"ss":        true,
	"ssr":       true, // Clash Verge/Meta 支持 SSR
	"hysteria":  true,
	"hysteria2": true,
	"tuic":      true,
	"direct":    true, // 信息节点
}

// 注意：regionMap、serverCodeMap 和 regionKeys 已移至 region_config.json 和 region_loader.go
// 这些变量保留作为向后兼容的降级方案（当 JSON 文件不存在时使用）
// 实际运行时优先使用从 JSON 文件加载的配置

// ==========================================
// Types
// ==========================================

// SubscriptionContext 订阅上下文，包含生成配置所需的所有信息
type SubscriptionContext struct {
	User           models.User
	Subscription   models.Subscription
	Proxies        []*ProxyNode
	Status         SubscriptionStatus
	ResetRecord    *models.SubscriptionReset // 如果是旧订阅地址，这里会有记录
	CurrentDevices int
	DeviceLimit    int
}

// ConfigUpdateService 配置更新服务
type ConfigUpdateService struct {
	db            *gorm.DB
	isRunning     bool
	runningMutex  sync.Mutex
	siteURL       string         // 缓存站点URL，避免频繁查询
	supportQQ     string         // 缓存客服QQ
	regionMatcher *RegionMatcher // 地区匹配器（优化版）
	parserPool    *ParserPool    // 解析器池（并发处理）
}

// nodeWithOrder 用于排序导入
type nodeWithOrder struct {
	node       *ProxyNode
	orderIndex int
}

// ==========================================
// Service Lifecycle
// ==========================================

// NewConfigUpdateService 创建配置更新服务
func NewConfigUpdateService() *ConfigUpdateService {
	service := &ConfigUpdateService{
		db:         database.GetDB(),
		parserPool: NewParserPool(10), // 默认10个worker
	}

	// 加载地区配置
	regionConfig, err := LoadRegionConfig()
	if err != nil {
		// 记录警告，但不阻止服务启动
		if utils.AppLogger != nil {
			utils.AppLogger.Warn("地区配置加载失败: %v，将使用空配置", err)
		}
	}

	if regionConfig != nil && (len(regionConfig.RegionMap) > 0 || len(regionConfig.ServerMap) > 0) {
		service.regionMatcher = NewRegionMatcher(regionConfig.RegionMap, regionConfig.ServerMap)
		if utils.AppLogger != nil {
			utils.AppLogger.Info("地区配置加载成功: region_map=%d, server_map=%d",
				len(regionConfig.RegionMap), len(regionConfig.ServerMap))
		}
	} else {
		// 如果加载失败或配置为空，使用空的匹配器
		service.regionMatcher = NewRegionMatcher(make(map[string]string), make(map[string]string))
		if utils.AppLogger != nil {
			utils.AppLogger.Warn("使用空的地区匹配器（所有节点将显示为'未知'地区）")
		}
	}

	// 初始化缓存配置
	service.refreshSystemConfig()
	return service
}

// loadLegacyRegionMaps 从旧代码加载地区映射（向后兼容）
// 注意：此方法已不再需要，因为配置现在从 JSON 文件加载
// 如果 JSON 文件不存在，RegionMatcher 会使用空映射
func (s *ConfigUpdateService) loadLegacyRegionMaps() {
	// 不再需要，配置从 region_config.json 加载
}

// refreshSystemConfig 刷新系统配置缓存
func (s *ConfigUpdateService) refreshSystemConfig() {
	// 获取网站域名（使用公共函数）
	domain := utils.GetDomainFromDB(s.db)
	if domain != "" {
		s.siteURL = utils.FormatDomainURL(domain)
	} else {
		s.siteURL = "请在系统设置中配置域名"
	}

	// 获取客服QQ（只从 category = "general" 获取）
	var supportQQConfig models.SystemConfig
	if err := s.db.Where("key = ? AND category = ?", "support_qq", "general").First(&supportQQConfig).Error; err == nil && supportQQConfig.Value != "" {
		s.supportQQ = strings.TrimSpace(supportQQConfig.Value)
	} else {
		s.supportQQ = "" // 不设置默认值，如果未配置则为空
	}
}

// ==========================================
// Public API
// ==========================================

// IsRunning 检查是否正在运行
func (s *ConfigUpdateService) IsRunning() bool {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()
	return s.isRunning
}

// GetStatus 获取状态
func (s *ConfigUpdateService) GetStatus() map[string]interface{} {
	var lastUpdate string
	var config models.SystemConfig
	if err := s.db.Where("key = ?", "config_update_last_update").First(&config).Error; err == nil {
		lastUpdate = config.Value
	}

	return map[string]interface{}{
		"is_running":  s.IsRunning(),
		"last_update": lastUpdate,
		"next_update": "",
	}
}

// GetLogs 获取日志
func (s *ConfigUpdateService) GetLogs(limit int) []map[string]interface{} {
	var config models.SystemConfig
	if err := s.db.Where("key = ?", "config_update_logs").First(&config).Error; err != nil {
		return []map[string]interface{}{}
	}

	var logs []map[string]interface{}
	if err := json.Unmarshal([]byte(config.Value), &logs); err != nil {
		return []map[string]interface{}{}
	}

	if len(logs) > limit {
		return logs[len(logs)-limit:]
	}
	return logs
}

// ClearLogs 清理日志
func (s *ConfigUpdateService) ClearLogs() error {
	var config models.SystemConfig
	err := s.db.Where("key = ?", "config_update_logs").First(&config).Error

	if err != nil {
		config = models.SystemConfig{
			Key:         "config_update_logs",
			Value:       "[]",
			Type:        "json",
			Category:    "general",
			DisplayName: "配置更新日志",
			Description: "配置更新任务日志",
		}
		return s.db.Create(&config).Error
	}
	config.Value = "[]"
	return s.db.Save(&config).Error
}

// GetConfig 获取配置（公开方法）
func (s *ConfigUpdateService) GetConfig() (map[string]interface{}, error) {
	return s.getConfig()
}

// RunUpdateTask 执行配置更新任务
func (s *ConfigUpdateService) RunUpdateTask() error {
	s.runningMutex.Lock()
	if s.isRunning {
		s.runningMutex.Unlock()
		return fmt.Errorf("任务已在运行中")
	}
	s.isRunning = true
	s.runningMutex.Unlock()

	defer func() {
		s.runningMutex.Lock()
		s.isRunning = false
		s.runningMutex.Unlock()
	}()

	s.log("INFO", "开始执行配置更新任务")

	// 获取配置
	config, err := s.getConfig()
	if err != nil {
		s.log("ERROR", fmt.Sprintf("获取配置失败: %v", err))
		return err
	}

	urls := config["urls"].([]string)
	if len(urls) == 0 {
		msg := "未配置节点源URL"
		s.log("ERROR", msg)
		return fmt.Errorf("%s", msg)
	}

	s.log("INFO", fmt.Sprintf("获取到 %d 个节点源URL", len(urls)))

	// 1. 获取节点
	nodes, err := s.FetchNodesFromURLs(urls)
	if err != nil {
		s.log("ERROR", fmt.Sprintf("获取节点失败: %v", err))
		return err
	}

	if len(nodes) == 0 {
		msg := "未获取到有效节点"
		s.log("WARN", msg)
		return fmt.Errorf("%s", msg)
	}

	s.log("INFO", fmt.Sprintf("共获取到 %d 个有效节点链接，准备入库", len(nodes)))

	// 2. 解析节点并整理准备入库（包含关键词过滤）
	filterKeywords := []string{}
	if keywords, ok := config["filter_keywords"].([]string); ok {
		filterKeywords = keywords
	} else if keywordsStr, ok := config["filter_keywords"].(string); ok && keywordsStr != "" {
		// 处理字符串格式的关键词（用换行符分隔）- 向后兼容
		for _, kw := range strings.Split(keywordsStr, "\n") {
			if kw = strings.TrimSpace(kw); kw != "" {
				filterKeywords = append(filterKeywords, kw)
			}
		}
	}

	if len(filterKeywords) > 0 {
		s.log("INFO", fmt.Sprintf("已配置 %d 个过滤关键词: %v，将过滤包含这些关键词的节点", len(filterKeywords), filterKeywords))
	} else {
		s.log("DEBUG", "未配置过滤关键词，将不过滤任何节点")
	}

	nodesWithOrder, stats := s.processFetchedNodes(urls, nodes, filterKeywords)

	// 输出统计信息
	if stats.parseFailed > 0 {
		s.log("WARN", fmt.Sprintf("解析失败的节点: %d 个", stats.parseFailed))
	}
	if stats.filtered > 0 {
		s.log("INFO", fmt.Sprintf("被关键词过滤的节点: %d 个", stats.filtered))
	}
	if stats.duplicates > 0 {
		s.log("INFO", fmt.Sprintf("去重跳过的节点: %d 个", stats.duplicates))
	}
	if stats.invalidLinks > 0 {
		s.log("WARN", fmt.Sprintf("无效链接的节点: %d 个", stats.invalidLinks))
	}
	s.log("INFO", fmt.Sprintf("成功解析并准备入库的节点: %d 个", len(nodesWithOrder)))

	// 3. 入库
	importedCount := s.importNodesToDatabaseWithOrder(nodesWithOrder)
	s.updateLastUpdateTime()

	s.log("SUCCESS", fmt.Sprintf("任务完成: 解析出 %d 个节点，成功入库/更新 %d 个", len(nodesWithOrder), importedCount))
	return nil
}

// ==========================================
// Internal Logic
// ==========================================

// updateStats 统计信息结构
type updateStats struct {
	parseFailed   int
	duplicates    int
	invalidLinks  int
	missingSource int
	filtered      int // 被关键词过滤的节点数量
}

// processFetchedNodes 处理获取到的节点：分组、去重、排序、关键词过滤
func (s *ConfigUpdateService) processFetchedNodes(urls []string, nodes []map[string]interface{}, filterKeywords []string) ([]nodeWithOrder, updateStats) {
	var nodesWithOrder []nodeWithOrder
	stats := updateStats{}
	seenKeys := make(map[string]bool)
	usedNames := make(map[string]bool)

	// 按订阅地址分组节点
	nodesByURL := make(map[string][]map[string]interface{})
	for _, nodeInfo := range nodes {
		sourceURL, _ := nodeInfo["source_url"].(string)
		if sourceURL == "" {
			stats.missingSource++
			continue
		}
		nodesByURL[sourceURL] = append(nodesByURL[sourceURL], nodeInfo)
	}

	// 按照订阅地址的顺序处理节点
	for urlIndex, url := range urls {
		urlNodes := nodesByURL[url]
		if len(urlNodes) == 0 {
			continue
		}

		s.log("INFO", fmt.Sprintf("开始处理订阅地址 [%d/%d] 的节点，共 %d 个链接", urlIndex+1, len(urls), len(urlNodes)))

		// 提取所有链接
		links := make([]string, 0, len(urlNodes))
		linkToNodeInfo := make(map[string]map[string]interface{})
		for _, nodeInfo := range urlNodes {
			link, ok := nodeInfo["url"].(string)
			if !ok {
				stats.invalidLinks++
				s.log("WARN", fmt.Sprintf("订阅地址 [%d/%d] 中发现无效链接（缺少url字段）", urlIndex+1, len(urls)))
				continue
			}
			links = append(links, link)
			linkToNodeInfo[link] = nodeInfo
		}

		// 使用 ParserPool 并发解析
		results := s.parserPool.ParseLinks(links)

		nodeIndexInURL := 0
		counts := struct{ Processed, Failed, Filtered, Duplicate int }{}

		for _, result := range results {
			link := result.Link

			// 链接去重
			if seenKeys[link] {
				stats.duplicates++
				counts.Duplicate++
				continue
			}
			seenKeys[link] = true

			// 检查解析错误
			if result.Err != nil {
				stats.parseFailed++
				counts.Failed++
				// 增强错误日志：记录更多上下文信息
				if counts.Failed <= 10 { // 增加到10条，提供更多调试信息
					s.log("WARN", fmt.Sprintf("解析失败 [订阅地址 %d/%d, 链接索引 %d]: %v, 链接片段: %s",
						urlIndex+1, len(urls), nodeIndexInURL, result.Err, truncateString(link, 50)))
				}
				continue
			}

			if result.Node == nil {
				stats.parseFailed++
				counts.Failed++
				s.log("WARN", fmt.Sprintf("解析返回空节点 [订阅地址 %d/%d, 链接索引 %d]: %s",
					urlIndex+1, len(urls), nodeIndexInURL, truncateString(link, 50)))
				continue
			}

			node := result.Node

			// 关键词过滤
			if filtered, keyword := s.isNodeFiltered(node, filterKeywords); filtered {
				stats.filtered++
				counts.Filtered++
				s.log("DEBUG", fmt.Sprintf("节点被过滤 [订阅地址 %d/%d]: %s (关键词: %s)",
					urlIndex+1, len(urls), node.Name, keyword))
				continue
			}

			counts.Processed++

			// 名称去重和重命名
			node.Name = s.ensureUniqueName(node.Name, usedNames)
			usedNames[node.Name] = true

			// 添加到结果列表
			nodesWithOrder = append(nodesWithOrder, nodeWithOrder{
				node:       node,
				orderIndex: urlIndex*10000 + nodeIndexInURL,
			})
			nodeIndexInURL++
		}

		s.log("INFO", fmt.Sprintf("订阅地址 [%d/%d] 完成: 成功=%d, 失败=%d, 过滤=%d, 重复=%d",
			urlIndex+1, len(urls), counts.Processed, counts.Failed, counts.Filtered, counts.Duplicate))
	}
	return nodesWithOrder, stats
}

// isNodeFiltered 检查节点是否应被过滤
func (s *ConfigUpdateService) isNodeFiltered(node *ProxyNode, keywords []string) (bool, string) {
	if len(keywords) == 0 {
		return false, ""
	}
	nameLower := strings.ToLower(node.Name)
	serverLower := strings.ToLower(node.Server)

	for _, kw := range keywords {
		kwLower := strings.ToLower(strings.TrimSpace(kw))
		if kwLower == "" {
			continue
		}
		if strings.Contains(nameLower, kwLower) || strings.Contains(serverLower, kwLower) {
			return true, kw
		}
	}
	return false, ""
}

// ensureUniqueName 确保节点名称唯一
func (s *ConfigUpdateService) ensureUniqueName(name string, usedNames map[string]bool) string {
	if !usedNames[name] {
		return name
	}
	counter := 1
	for {
		newName := fmt.Sprintf("%s-%d", name, counter)
		if !usedNames[newName] {
			return newName
		}
		counter++
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

// getConfig 获取配置
func (s *ConfigUpdateService) getConfig() (map[string]interface{}, error) {
	var configs []models.SystemConfig
	s.db.Where("category = ?", "config_update").Find(&configs)

	result := map[string]interface{}{
		"urls":              []string{},
		"target_dir":        "./uploads/config",
		"v2ray_file":        "xr",
		"clash_file":        "clash.yaml",
		"filter_keywords":   []string{},
		"enable_schedule":   false,
		"schedule_interval": 3600,
	}

	var urlsConfig *models.SystemConfig
	var filterKeywordsConfig *models.SystemConfig

	for _, config := range configs {
		if config.Key == "urls" {
			urlsConfig = &config
		} else if config.Key == "filter_keywords" {
			filterKeywordsConfig = &config
		} else if config.Key == "enable_schedule" {
			result[config.Key] = config.Value == "true" || config.Value == "1"
		} else if config.Key == "schedule_interval" {
			var interval int
			fmt.Sscanf(config.Value, "%d", &interval)
			if interval == 0 {
				interval = 3600
			}
			result[config.Key] = interval
		} else {
			result[config.Key] = config.Value
		}
	}

	// 处理 URLs
	if urlsConfig != nil && strings.TrimSpace(urlsConfig.Value) != "" {
		var filtered []string
		for _, u := range strings.Split(urlsConfig.Value, "\n") {
			if u = strings.TrimSpace(u); u != "" {
				filtered = append(filtered, u)
			}
		}
		result["urls"] = filtered
	}

	// 处理 filter_keywords（按换行符分割）
	if filterKeywordsConfig != nil && strings.TrimSpace(filterKeywordsConfig.Value) != "" {
		var filtered []string
		for _, kw := range strings.Split(filterKeywordsConfig.Value, "\n") {
			if kw = strings.TrimSpace(kw); kw != "" {
				filtered = append(filtered, kw)
			}
		}
		result["filter_keywords"] = filtered
	}

	return result, nil
}

// updateLastUpdateTime 更新最后更新时间
func (s *ConfigUpdateService) updateLastUpdateTime() {
	now := utils.GetBeijingTime().Format("2006-01-02T15:04:05")
	var config models.SystemConfig
	err := s.db.Where("key = ?", "config_update_last_update").First(&config).Error

	if err != nil {
		config = models.SystemConfig{
			Key:         "config_update_last_update",
			Value:       now,
			Type:        "string",
			Category:    "config_update",
			DisplayName: "最后更新时间",
			Description: "配置更新任务的最后执行时间",
		}
		s.db.Create(&config)
	} else {
		config.Value = now
		s.db.Save(&config)
	}
}

// log 记录日志
func (s *ConfigUpdateService) log(level, message string) {
	now := utils.GetBeijingTime().Format("2006-01-02 15:04:05")
	logEntry := map[string]interface{}{
		"time":    now,
		"level":   level,
		"message": message,
	}

	go s.saveLogToDB(logEntry)

	// 同时打印到系统日志
	if utils.AppLogger != nil {
		if level == "ERROR" {
			utils.AppLogger.Error("%s", message)
		} else {
			utils.AppLogger.Info("%s", message)
		}
	}
}

// saveLogToDB 保存日志到数据库 (异步执行)
func (s *ConfigUpdateService) saveLogToDB(logEntry map[string]interface{}) {
	var config models.SystemConfig
	err := s.db.Where("key = ?", "config_update_logs").First(&config).Error

	if err != nil {
		initialLogs := []map[string]interface{}{logEntry}
		logsJSON, _ := json.Marshal(initialLogs)
		config = models.SystemConfig{
			Key:         "config_update_logs",
			Value:       string(logsJSON),
			Type:        "json",
			Category:    "config_update",
			DisplayName: "配置更新日志",
			Description: "配置更新任务日志",
		}
		s.db.Create(&config)
	} else {
		var logs []map[string]interface{}
		json.Unmarshal([]byte(config.Value), &logs)
		logs = append(logs, logEntry)

		// 限制日志数量，保留最近 100 条
		if len(logs) > 100 {
			logs = logs[len(logs)-100:]
		}

		logsJSON, _ := json.Marshal(logs)
		config.Value = string(logsJSON)
		s.db.Save(&config)
	}
}

// ==========================================
// Node Processing
// ==========================================

// FetchNodesFromURLs 从URL列表获取节点
func (s *ConfigUpdateService) FetchNodesFromURLs(urls []string) ([]map[string]interface{}, error) {
	var allNodes []map[string]interface{}
	// 增加超时时间，特别是对于 GitHub Gist 等可能较慢的服务
	client := &http.Client{
		Timeout: 60 * time.Second, // 增加到 60 秒
		Transport: &http.Transport{
			DisableKeepAlives: false,
			MaxIdleConns:      10,
			IdleConnTimeout:   30 * time.Second,
		},
	}

	for i, url := range urls {
		s.log("INFO", fmt.Sprintf("正在下载节点源 [%d/%d]: %s", i+1, len(urls), url))

		content, err := s.fetchURLContent(client, url)
		if err != nil {
			s.log("ERROR", fmt.Sprintf("获取节点源失败: %v", err))
			continue
		}

		// 使用 node_parser.go 中的统一解码函数
		decoded := TryDecodeNodeList(string(content))

		// 调试日志
		decodedPreview := decoded
		if len(decodedPreview) > 200 {
			decodedPreview = decodedPreview[:200] + "..."
		}
		s.log("DEBUG", fmt.Sprintf("处理后内容长度: %d, 预览: %s", len(decoded), decodedPreview))

		nodeLinks := s.extractNodeLinks(decoded)
		s.logNodeTypeStats(url, nodeLinks)

		for _, link := range nodeLinks {
			allNodes = append(allNodes, map[string]interface{}{
				"url":        link,
				"source_url": url,
			})
		}
	}

	return allNodes, nil
}

// fetchURLContent 下载单个 URL 内容（带重试）
func (s *ConfigUpdateService) fetchURLContent(client *http.Client, url string) ([]byte, error) {
	maxRetries := 3
	retryDelay := 2 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("创建请求失败: %v", err)
		}

		// 设置请求头（针对 GitHub Gist 等服务的优化）
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
		// 对于 GitHub Gist，使用 close 而不是 keep-alive，避免连接问题
		if strings.Contains(url, "gist.githubusercontent.com") {
			req.Header.Set("Connection", "close")
		} else {
			req.Header.Set("Connection", "keep-alive")
		}
		// 不设置 Accept-Encoding，让服务器决定是否压缩，避免解压问题

		resp, err := client.Do(req)
		if err != nil {
			if attempt < maxRetries {
				s.log("WARN", fmt.Sprintf("下载失败 (尝试 %d/%d): %v，%v 后重试", attempt, maxRetries, err, retryDelay))
				time.Sleep(retryDelay)
				retryDelay *= 2
				continue
			}
			return nil, fmt.Errorf("下载失败: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			if attempt < maxRetries {
				s.log("WARN", fmt.Sprintf("状态码 %d (尝试 %d/%d)，%v 后重试", resp.StatusCode, attempt, maxRetries, retryDelay))
				time.Sleep(retryDelay)
				retryDelay *= 2
				continue
			}
			return nil, fmt.Errorf("状态码错误: %d", resp.StatusCode)
		}

		// 使用 LimitReader 防止读取过大内容，同时设置合理的限制
		// GitHub Gist raw 文件通常不会超过 10MB
		limitedReader := io.LimitReader(resp.Body, 10*1024*1024) // 10MB 限制
		content, err := io.ReadAll(limitedReader)
		if err != nil {
			resp.Body.Close()
			if attempt < maxRetries {
				s.log("WARN", fmt.Sprintf("读取内容失败 (尝试 %d/%d): %v，%v 后重试", attempt, maxRetries, err, retryDelay))
				time.Sleep(retryDelay)
				retryDelay *= 2
				continue
			}
			return nil, fmt.Errorf("读取内容失败: %v", err)
		}

		if len(content) > 0 {
			return content, nil
		}

		// 如果内容为空，也进行重试
		if attempt < maxRetries {
			s.log("WARN", fmt.Sprintf("内容为空 (尝试 %d/%d)，%v 后重试", attempt, maxRetries, retryDelay))
			time.Sleep(retryDelay)
			retryDelay *= 2
			continue
		}
	}
	return nil, fmt.Errorf("内容为空或获取失败")
}

// logNodeTypeStats 记录节点类型统计
func (s *ConfigUpdateService) logNodeTypeStats(url string, nodeLinks []string) {
	typeCount := make(map[string]int)
	for _, link := range nodeLinks {
		found := false
		for t := range supportedClashTypes {
			if strings.HasPrefix(link, t+"://") {
				typeCount[t]++
				found = true
				break
			}
		}
		if !found {
			// 简单检查其他协议
			if strings.HasPrefix(link, "hysteria2://") {
				typeCount["hysteria2"]++
			} else if strings.HasPrefix(link, "naive://") || strings.HasPrefix(link, "naive+https://") {
				typeCount["naive"]++
			} else if strings.HasPrefix(link, "anytls://") {
				typeCount["anytls"]++
			} else {
				typeCount["other"]++
			}
		}
	}

	var parts []string
	for t, c := range typeCount {
		parts = append(parts, fmt.Sprintf("%s:%d", t, c))
	}
	s.log("INFO", fmt.Sprintf("从 %s 提取到 %d 个节点链接 (%s)", url, len(nodeLinks), strings.Join(parts, ", ")))
}

// extractNodeLinks 提取节点链接
// 注意：需要按优先级顺序提取，避免误匹配
// 例如：vmess://xxx 的Base64部分可能包含 "ss://" 字符串，需要先匹配vmess
func (s *ConfigUpdateService) extractNodeLinks(content string) []string {
	var links []string
	var invalidLinks []string
	// 用于记录已经匹配的位置，避免重复匹配
	matchedPositions := make(map[int]bool)

	// 按优先级顺序匹配：先匹配vmess/vless/trojan，再匹配ss/ssr
	// 这样可以避免vmess链接的Base64部分被误识别为ss链接
	for _, re := range nodeLinkPatterns {
		matches := re.FindAllStringSubmatchIndex(content, -1)
		for _, match := range matches {
			// FindAllStringSubmatchIndex 返回 [完整匹配开始, 完整匹配结束, 子组1开始, 子组1结束, ...]
			// 我们需要获取第一个捕获组（实际链接）的位置
			var start, end int
			var matchStr string

			if len(match) >= 4 {
				// 有捕获组：使用第一个捕获组（实际链接）
				start = match[2]
				end = match[3]
				matchStr = content[start:end]
			} else if len(match) >= 2 {
				// 没有捕获组：使用完整匹配（向后兼容）
				start = match[0]
				end = match[1]
				matchStr = content[start:end]
				// 移除可能的前导空白字符
				matchStr = strings.TrimSpace(matchStr)
			} else {
				continue
			}

			// 额外检查：如果匹配到ss://，确保前面不是vme（避免匹配到vmess://中的ss://）
			if strings.HasPrefix(matchStr, "ss://") && start >= 3 {
				prefix := content[start-3 : start]
				if prefix == "vme" {
					// 这是vmess://中的ss://，应该跳过
					continue
				}
			}

			// 检查这个位置是否已经被其他模式匹配过
			alreadyMatched := false
			for pos := start; pos < end; pos++ {
				if matchedPositions[pos] {
					alreadyMatched = true
					break
				}
			}

			if alreadyMatched {
				continue
			}

			// 标记这个位置已被匹配
			for pos := start; pos < end; pos++ {
				matchedPositions[pos] = true
			}

			if s.isValidNodeLink(matchStr) {
				links = append(links, matchStr)
			} else {
				invalidLinks = append(invalidLinks, matchStr)
			}
		}
	}

	if len(invalidLinks) > 0 {
		limit := 3
		if len(invalidLinks) < limit {
			limit = len(invalidLinks)
		}
		s.log("DEBUG", fmt.Sprintf("发现 %d 个无效链接，示例: %v", len(invalidLinks), invalidLinks[:limit]))
	}

	// 去重
	uniqueLinks := make(map[string]bool)
	var result []string
	for _, link := range links {
		if !uniqueLinks[link] {
			uniqueLinks[link] = true
			result = append(result, link)
		}
	}

	return result
}

// isValidNodeLink 验证节点链接是否完整有效
func (s *ConfigUpdateService) isValidNodeLink(link string) bool {
	link = strings.TrimSpace(link)
	if link == "" {
		return false
	}

	linkWithoutFragment := link
	if idx := strings.Index(link, "#"); idx != -1 {
		linkWithoutFragment = link[:idx]
	}

	if strings.HasPrefix(link, "ss://") {
		// SS 链接标准格式: ss://base64(method:password)@server:port#name
		// 必须包含 @ 符号和服务器地址
		if !strings.Contains(linkWithoutFragment, "@") {
			// 没有 @ 符号，可能是格式错误的链接或被截断的链接
			return false
		}
		parts := strings.Split(linkWithoutFragment, "@")
		if len(parts) < 2 {
			return false
		}
		serverPart := parts[1]
		if idx := strings.Index(serverPart, "?"); idx != -1 {
			serverPart = serverPart[:idx]
		}
		if !strings.Contains(serverPart, ":") {
			return false
		}
	} else if strings.HasPrefix(link, "vmess://") || strings.HasPrefix(link, "vless://") {
		encoded := strings.TrimPrefix(linkWithoutFragment, "vmess://")
		encoded = strings.TrimPrefix(encoded, "vless://")
		if idx := strings.Index(encoded, "?"); idx != -1 {
			encoded = encoded[:idx]
		}
		if len(encoded) < 10 {
			return false
		}
	} else if strings.HasPrefix(link, "trojan://") {
		if !strings.Contains(linkWithoutFragment, "@") {
			return false
		}
		parts := strings.Split(linkWithoutFragment, "@")
		if len(parts) < 2 {
			return false
		}
		serverPart := parts[1]
		if idx := strings.Index(serverPart, "?"); idx != -1 {
			serverPart = serverPart[:idx]
		}
		if !strings.Contains(serverPart, ":") {
			return false
		}
	} else if strings.HasPrefix(link, "ssr://") {
		encoded := strings.TrimPrefix(linkWithoutFragment, "ssr://")
		if len(encoded) < 10 {
			return false
		}
	} else if strings.HasPrefix(link, "hysteria://") || strings.HasPrefix(link, "hysteria2://") {
		if !strings.Contains(linkWithoutFragment, "@") && !strings.Contains(linkWithoutFragment, ":") {
			return false
		}
	} else if strings.HasPrefix(link, "tuic://") {
		if !strings.Contains(linkWithoutFragment, "@") {
			return false
		}
	}

	return true
}

// resolveRegion 从节点名称和服务器地址中解析地区信息（使用优化的匹配器）
func (s *ConfigUpdateService) resolveRegion(name, server string) string {
	if s.regionMatcher != nil {
		return s.regionMatcher.MatchRegion(name, server)
	}
	// 如果匹配器未初始化，返回"未知"
	return "未知"
}

// generateNodeDedupKey 生成节点去重键（统一格式：Type:Server:Port）
func (s *ConfigUpdateService) generateNodeDedupKey(nodeType, server string, port int) string {
	return fmt.Sprintf("%s:%s:%d", nodeType, server, port)
}

// ==========================================
// Database Operations
// ==========================================

// importNodesToDatabaseWithOrder 将节点导入到数据库的 nodes 表，并保存顺序索引
func (s *ConfigUpdateService) importNodesToDatabaseWithOrder(nodesWithOrder []nodeWithOrder) int {
	importedCount := 0

	for _, item := range nodesWithOrder {
		node := item.node
		orderIndex := item.orderIndex

		configJSON, _ := json.Marshal(node)
		configStr := string(configJSON)

		region := s.resolveRegion(node.Name, node.Server)

		var existingNode models.Node
		err := s.db.Where("type = ? AND name = ?", node.Type, node.Name).First(&existingNode).Error

		if err == nil {
			existingNode.Config = &configStr
			existingNode.Status = "online"
			existingNode.IsActive = true
			existingNode.OrderIndex = orderIndex
			existingNode.Region = region

			if err := s.db.Save(&existingNode).Error; err == nil {
				importedCount++
			} else {
				s.log("ERROR", fmt.Sprintf("更新节点失败: %s (%s), 错误: %v", node.Name, node.Type, err))
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			newNode := models.Node{
				Name:       node.Name,
				Type:       node.Type,
				Status:     "online",
				IsActive:   true,
				IsManual:   false,
				Config:     &configStr,
				Region:     region,
				OrderIndex: orderIndex,
			}
			if err := s.db.Create(&newNode).Error; err == nil {
				importedCount++
			} else {
				s.log("ERROR", fmt.Sprintf("创建节点失败: %s (%s), 错误: %v", node.Name, node.Type, err))
			}
		} else {
			s.log("ERROR", fmt.Sprintf("查询节点失败: %s (%s), 错误: %v", node.Name, node.Type, err))
		}
	}
	return importedCount
}

// fetchProxiesForUser 获取用户的可用节点
func (s *ConfigUpdateService) fetchProxiesForUser(user models.User, sub models.Subscription) ([]*ProxyNode, error) {
	var proxies []*ProxyNode
	processedNodes := make(map[string]bool)

	now := utils.GetBeijingTime()

	// 检查普通订阅是否过期
	isOrdExpired := !sub.ExpireTime.IsZero() && sub.ExpireTime.Before(now)

	// 计算专线到期时间
	// 如果设置了专线到期时间，以专线到期时间为准
	// 如果没设置专线到期时间，以普通线路到期时间为准
	var specialExpireTime time.Time
	hasSpecialExpireTime := false
	if user.SpecialNodeExpiresAt.Valid {
		specialExpireTime = utils.ToBeijingTime(user.SpecialNodeExpiresAt.Time)
		hasSpecialExpireTime = true
	} else if !sub.ExpireTime.IsZero() {
		specialExpireTime = utils.ToBeijingTime(sub.ExpireTime)
		hasSpecialExpireTime = true
	}
	isSpecialExpired := hasSpecialExpireTime && specialExpireTime.Before(now)

	// 根据用户的订阅类型决定是否包含普通节点
	// special_only: 只包含专线节点，不包含普通节点
	// both: 包含普通节点+专线节点，专线节点在最前面
	// 如果普通订阅过期，客户无法订阅普通线路（但可以订阅专线，如果专线未过期）
	includeNormalNodes := false
	if user.SpecialNodeSubscriptionType == "both" {
		// 全部订阅：只有普通订阅未过期时才包含普通节点
		includeNormalNodes = !isOrdExpired
	} else if user.SpecialNodeSubscriptionType == "special_only" {
		// 仅专线：不包含普通节点
		includeNormalNodes = false
	} else {
		// 默认情况：如果普通订阅未过期，包含普通节点
		includeNormalNodes = !isOrdExpired
	}

	if includeNormalNodes {
		// 获取普通节点
		var nodes []models.Node
		query := s.db.Model(&models.Node{}).Where("is_active = ?", true).Where("status != ?", "timeout")

		if err := query.Find(&nodes).Error; err != nil {
			return nil, err
		}

		for _, node := range nodes {
			proxyNodes, err := s.parseNodeToProxies(&node)
			if err != nil {
				continue
			}

			for _, proxy := range proxyNodes {
				// 使用统一的去重键生成函数
				key := s.generateNodeDedupKey(proxy.Type, proxy.Server, proxy.Port)
				if processedNodes[key] {
					continue
				}
				processedNodes[key] = true
				proxies = append(proxies, proxy)
			}
		}
	}

	// 获取专属节点（专线节点始终在最前面）
	var customNodes []models.CustomNode
	if err := s.db.Joins("JOIN user_custom_nodes ON user_custom_nodes.custom_node_id = custom_nodes.id").
		Where("user_custom_nodes.user_id = ? AND custom_nodes.is_active = ?", user.ID, true).
		Find(&customNodes).Error; err == nil {

		var customProxies []*ProxyNode
		for _, cn := range customNodes {
			// 判断专线节点是否过期
			// 1. 如果节点设置了 FollowUserExpire，使用用户的专线到期时间（或普通到期时间）
			// 2. 如果节点设置了 ExpireTime，使用节点的到期时间
			// 3. 如果都没设置，使用用户的专线到期时间（或普通到期时间）
			isSpecNodeExpired := false
			if cn.FollowUserExpire {
				// 跟随用户到期时间
				isSpecNodeExpired = isSpecialExpired
			} else if cn.ExpireTime != nil {
				// 使用节点自己的到期时间
				expireTimeBeijing := utils.ToBeijingTime(*cn.ExpireTime)
				isSpecNodeExpired = expireTimeBeijing.Before(now)
			} else {
				// 默认使用用户的专线到期时间（或普通到期时间）
				isSpecNodeExpired = isSpecialExpired
			}

			if isSpecNodeExpired || cn.Status == "timeout" {
				continue
			}

			displayName := cn.DisplayName
			if displayName == "" {
				displayName = "专线-" + cn.Name
			}

			if cn.Config != "" {
				var proxyNode ProxyNode
				if err := json.Unmarshal([]byte(cn.Config), &proxyNode); err == nil {
					proxyNode.Name = displayName
					customProxies = append(customProxies, &proxyNode)
				}
			}
		}

		// 将专线节点放在最前面
		proxies = append(customProxies, proxies...)
	}

	return proxies, nil
}

// parseNodeToProxies 解析数据库节点模型为代理节点对象
func (s *ConfigUpdateService) parseNodeToProxies(node *models.Node) ([]*ProxyNode, error) {
	if node.Config != nil && *node.Config != "" {
		var configProxy ProxyNode
		if err := json.Unmarshal([]byte(*node.Config), &configProxy); err == nil {
			configProxy.Name = node.Name
			return []*ProxyNode{&configProxy}, nil
		}
	}
	return nil, fmt.Errorf("节点配置为空")
}

// getSubscriptionContext 获取订阅上下文
func (s *ConfigUpdateService) getSubscriptionContext(token string, clientIP string, userAgent string) *SubscriptionContext {
	ctx := &SubscriptionContext{
		Status: StatusNotFound,
	}

	// 1. 查找订阅
	var sub models.Subscription
	if err := s.db.Where("subscription_url = ?", token).First(&sub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			var reset models.SubscriptionReset
			if err := s.db.Where("old_subscription_url = ?", token).First(&reset).Error; err == nil {
				ctx.Status = StatusOldAddress
				ctx.ResetRecord = &reset
				return ctx
			}
		}
		return ctx
	}
	ctx.Subscription = sub

	// 2. 查找用户
	var user models.User
	if err := s.db.First(&user, sub.UserID).Error; err != nil {
		return ctx
	}
	ctx.User = user

	// 3. 检查状态
	if !user.IsActive {
		ctx.Status = StatusAccountAbnormal
		return ctx
	}
	if !sub.IsActive || sub.Status != "active" {
		ctx.Status = StatusInactive
		return ctx
	}
	// 检查订阅是否过期
	// SQLite 存储的时间格式是 UTC (如: 2027-01-22 00:00:00+00:00)
	// 我们需要统一使用 UTC 时间进行比较，避免时区问题
	if !sub.ExpireTime.IsZero() {
		// 将 ExpireTime 转换为 UTC（如果它还不是 UTC）
		expireTimeUTC := sub.ExpireTime.UTC()
		// 获取当前 UTC 时间
		nowUTC := time.Now().UTC()

		// 调试日志：记录时间比较信息
		if utils.AppLogger != nil {
			utils.AppLogger.Info("订阅过期检查 - SubscriptionID=%d, UserID=%d, ExpireTime(原始)=%s, ExpireTime(UTC)=%s, Now(UTC)=%s, ExpireTime.Unix=%d, Now.Unix=%d, Before=%v",
				sub.ID, sub.UserID,
				sub.ExpireTime.Format("2006-01-02 15:04:05 MST"),
				expireTimeUTC.Format("2006-01-02 15:04:05 MST"),
				nowUTC.Format("2006-01-02 15:04:05 MST"),
				expireTimeUTC.Unix(),
				nowUTC.Unix(),
				expireTimeUTC.Before(nowUTC))
		}

		// 使用 UTC 时间进行比较
		if expireTimeUTC.Before(nowUTC) {
			ctx.Status = StatusExpired
			return ctx
		}
	}

	// 4. 检查设备
	var currentDevices int64
	s.db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", sub.ID, true).Count(&currentDevices)
	ctx.CurrentDevices = int(currentDevices)
	ctx.DeviceLimit = sub.DeviceLimit

	// 设备限制检查：如果限制为0，不允许使用
	if sub.DeviceLimit == 0 {
		ctx.Status = StatusDeviceOverLimit
		return ctx
	}

	// 如果设备数量达到或超过限制，检查当前设备是否已存在
	if sub.DeviceLimit > 0 && int(currentDevices) >= sub.DeviceLimit {
		var device models.Device
		isKnownDevice := false
		if err := s.db.Where("subscription_id = ? AND ip_address = ? AND user_agent = ?", sub.ID, clientIP, userAgent).First(&device).Error; err == nil {
			isKnownDevice = true
		}
		if !isKnownDevice {
			ctx.Status = StatusDeviceOverLimit
			return ctx
		}
	}

	// 5. 获取节点
	proxies, err := s.fetchProxiesForUser(user, sub)
	if err != nil {
		ctx.Proxies = []*ProxyNode{}
	} else {
		ctx.Proxies = proxies
	}

	ctx.Status = StatusNormal
	return ctx
}

// UpdateSubscriptionConfig 更新订阅配置
func (s *ConfigUpdateService) UpdateSubscriptionConfig(subscriptionURL string) error {
	var count int64
	s.db.Model(&models.Subscription{}).Where("subscription_url = ?", subscriptionURL).Count(&count)
	if count == 0 {
		return fmt.Errorf("订阅不存在")
	}
	return nil
}

// ==========================================
// Config Generation
// ==========================================

// GenerateClashConfig 生成 Clash 配置
func (s *ConfigUpdateService) GenerateClashConfig(token string, clientIP string, userAgent string) (string, error) {
	nodes, err := s.prepareExportNodes(token, clientIP, userAgent)
	if err != nil {
		return "", err
	}
	return s.generateClashYAML(nodes), nil
}

// GenerateUniversalConfig 生成通用订阅配置
func (s *ConfigUpdateService) GenerateUniversalConfig(token string, clientIP string, userAgent string, format string) (string, error) {
	nodes, err := s.prepareExportNodes(token, clientIP, userAgent)
	if err != nil {
		return "", err
	}

	var links []string
	for _, node := range nodes {
		var link string
		if format == "ssr" && node.Type == "ssr" {
			link = s.nodeToSSRLink(node)
		} else {
			link = s.nodeToLink(node)
		}
		if link != "" {
			links = append(links, link)
		}
	}

	return base64.StdEncoding.EncodeToString([]byte(strings.Join(links, "\n"))), nil
}

// prepareExportNodes 准备导出的节点列表（包含信息节点或错误节点）
func (s *ConfigUpdateService) prepareExportNodes(token, clientIP, userAgent string) ([]*ProxyNode, error) {
	// 每次生成配置前都刷新系统配置，确保使用最新的域名设置
	s.refreshSystemConfig()

	ctx := s.getSubscriptionContext(token, clientIP, userAgent)

	if ctx.Status != StatusNormal {
		return s.generateErrorNodes(ctx.Status, ctx), nil
	}

	return s.addInfoNodes(ctx.Proxies, ctx), nil
}

// generateClashYAML 生成 Clash YAML 配置
func (s *ConfigUpdateService) generateClashYAML(proxies []*ProxyNode) string {
	var builder strings.Builder

	// 过滤支持的节点
	filteredProxies := make([]*ProxyNode, 0)
	for _, proxy := range proxies {
		if supportedClashTypes[proxy.Type] {
			filteredProxies = append(filteredProxies, proxy)
		}
	}

	// 基础配置
	builder.WriteString("port: 7890\n")
	builder.WriteString("socks-port: 7891\n")
	builder.WriteString("allow-lan: true\n")
	builder.WriteString("mode: Rule\n")
	builder.WriteString("log-level: info\n")
	builder.WriteString("external-controller: 127.0.0.1:9090\n\n")

	builder.WriteString("proxies:\n")

	// 确保节点名称唯一
	usedNames := make(map[string]bool)
	var proxyNames []string

	for _, proxy := range filteredProxies {
		originalName := proxy.Name
		newName := originalName
		counter := 1
		for usedNames[newName] {
			newName = fmt.Sprintf("%s_%d", originalName, counter)
			counter++
		}
		proxy.Name = newName
		usedNames[newName] = true

		builder.WriteString(s.nodeToYAML(proxy, 2))
		proxyNames = append(proxyNames, s.escapeYAMLString(proxy.Name))
	}

	// 代理组
	builder.WriteString("\nproxy-groups:\n")

	// 节点选择
	builder.WriteString("  - name: \"🚀 节点选择\"\n")
	builder.WriteString("    type: select\n")
	builder.WriteString("    proxies:\n")
	builder.WriteString("      - \"♻️ 自动选择\"\n")
	for _, name := range proxyNames {
		builder.WriteString(fmt.Sprintf("      - %s\n", name))
	}

	// 自动选择
	builder.WriteString("  - name: \"♻️ 自动选择\"\n")
	builder.WriteString("    type: url-test\n")
	builder.WriteString("    url: http://www.gstatic.com/generate_204\n")
	builder.WriteString("    interval: 300\n")
	builder.WriteString("    tolerance: 50\n")
	builder.WriteString("    proxies:\n")
	for _, name := range proxyNames {
		builder.WriteString(fmt.Sprintf("      - %s\n", name))
	}

	// 规则
	builder.WriteString("\nrules:\n")
	builder.WriteString("  - DOMAIN-SUFFIX,local,DIRECT\n")
	builder.WriteString("  - IP-CIDR,127.0.0.0/8,DIRECT\n")
	builder.WriteString("  - IP-CIDR,172.16.0.0/12,DIRECT\n")
	builder.WriteString("  - IP-CIDR,192.168.0.0/16,DIRECT\n")
	builder.WriteString("  - GEOIP,CN,DIRECT\n")
	builder.WriteString("  - MATCH,🚀 节点选择\n")

	return builder.String()
}

// addInfoNodes 添加信息节点
func (s *ConfigUpdateService) addInfoNodes(proxies []*ProxyNode, ctx *SubscriptionContext) []*ProxyNode {
	expireTimeStr := "无限期"
	if !ctx.Subscription.ExpireTime.IsZero() {
		expireTimeStr = ctx.Subscription.ExpireTime.Format("2006-01-02")
	}

	infoNodes := []*ProxyNode{
		s.createMessageNode(fmt.Sprintf("📢 官网: %s", s.siteURL)),
		s.createMessageNode(fmt.Sprintf("⏰ 到期: %s", expireTimeStr)),
		s.createMessageNode(fmt.Sprintf("📱 设备: %d/%d", ctx.CurrentDevices, ctx.DeviceLimit)),
	}

	if s.supportQQ != "" {
		infoNodes = append(infoNodes, s.createMessageNode(fmt.Sprintf("💬 客服QQ: %s", s.supportQQ)))
	}

	return append(infoNodes, proxies...)
}

// generateErrorNodes 生成错误提示节点
func (s *ConfigUpdateService) generateErrorNodes(status SubscriptionStatus, ctx *SubscriptionContext) []*ProxyNode {
	var reason, solution string

	switch status {
	case StatusExpired:
		reason = "订阅已过期"
		solution = fmt.Sprintf("请前往官网续费 (过期时间: %s)", ctx.Subscription.ExpireTime.Format("2006-01-02"))
	case StatusInactive:
		reason = "订阅已失效"
		solution = "请联系管理员检查订阅状态"
	case StatusAccountAbnormal:
		reason = "账户异常"
		solution = "您的账户状态异常或已被禁用，请联系客服"
	case StatusDeviceOverLimit:
		reason = "设备数量超限"
		solution = fmt.Sprintf("当前设备 %d/%d，请在官网删除不使用的设备", ctx.CurrentDevices, ctx.DeviceLimit)
	case StatusOldAddress:
		reason = "订阅地址已变更"
		solution = "请登录官网获取最新的订阅地址"
	case StatusNotFound:
		reason = "订阅不存在"
		solution = "请检查订阅链接是否正确，或重新复制"
	default:
		reason = "账户异常"
		solution = "检测到账户异常，请联系管理员"
	}

	infoNodes := []*ProxyNode{
		s.createMessageNode(fmt.Sprintf("📢 官网: %s", s.siteURL)),
		s.createMessageNode(fmt.Sprintf("❌ 原因: %s", reason), "error"),
		s.createMessageNode(fmt.Sprintf("💡 解决: %s", solution), "error"),
	}

	qqMsg := "💬 客服QQ: 请在系统设置中配置"
	if s.supportQQ != "" {
		qqMsg = fmt.Sprintf("💬 客服QQ: %s", s.supportQQ)
	}
	infoNodes = append(infoNodes, s.createMessageNode(qqMsg, "error"))

	return infoNodes
}

// createMessageNode 创建消息提示节点 (SS类型)
func (s *ConfigUpdateService) createMessageNode(name string, password ...string) *ProxyNode {
	pwd := "info"
	if len(password) > 0 {
		pwd = password[0]
	}
	return &ProxyNode{
		Name:     name,
		Type:     "ss",
		Server:   "baidu.com",
		Port:     1234,
		Cipher:   "aes-128-gcm",
		Password: pwd,
	}
}

// nodeToYAML 将节点转换为 YAML 格式
func (s *ConfigUpdateService) nodeToYAML(node *ProxyNode, indent int) string {
	indentStr := strings.Repeat(" ", indent)
	var builder strings.Builder

	escapedName := s.escapeYAMLString(node.Name)

	builder.WriteString(fmt.Sprintf("%s- name: %s\n", indentStr, escapedName))
	builder.WriteString(fmt.Sprintf("%s  type: %s\n", indentStr, node.Type))
	builder.WriteString(fmt.Sprintf("%s  server: %s\n", indentStr, node.Server))
	builder.WriteString(fmt.Sprintf("%s  port: %d\n", indentStr, node.Port))

	switch node.Type {
	case "ss":
		if node.Cipher != "" {
			builder.WriteString(fmt.Sprintf("%s  cipher: %s\n", indentStr, node.Cipher))
		}
		if node.Password != "" {
			builder.WriteString(fmt.Sprintf("%s  password: %s\n", indentStr, node.Password))
		}
	case "vmess":
		if node.UUID != "" {
			builder.WriteString(fmt.Sprintf("%s  uuid: %s\n", indentStr, node.UUID))
		}
		if alterId, ok := node.Options["alterId"]; !ok {
			builder.WriteString(fmt.Sprintf("%s  alterId: 0\n", indentStr))
		} else {
			builder.WriteString(fmt.Sprintf("%s  alterId: %v\n", indentStr, alterId))
		}
		if node.Cipher == "" {
			node.Cipher = "auto"
		}
		builder.WriteString(fmt.Sprintf("%s  cipher: %s\n", indentStr, node.Cipher))
	case "vless":
		if node.UUID != "" {
			builder.WriteString(fmt.Sprintf("%s  uuid: %s\n", indentStr, node.UUID))
		}
	case "trojan":
		if node.Password != "" {
			builder.WriteString(fmt.Sprintf("%s  password: %s\n", indentStr, node.Password))
		}
	case "ssr":
		if node.Cipher != "" {
			builder.WriteString(fmt.Sprintf("%s  cipher: %s\n", indentStr, node.Cipher))
		}
		if node.Password != "" {
			builder.WriteString(fmt.Sprintf("%s  password: %s\n", indentStr, node.Password))
		}
	}

	if node.TLS {
		builder.WriteString(fmt.Sprintf("%s  tls: true\n", indentStr))
	}
	if node.Network != "" && node.Network != "tcp" {
		builder.WriteString(fmt.Sprintf("%s  network: %s\n", indentStr, node.Network))
	}
	if node.UDP {
		builder.WriteString(fmt.Sprintf("%s  udp: true\n", indentStr))
	}

	// 写入 Options
	optionsIndentStr := indentStr + "  "

	// 对 Options key 进行排序以保证输出稳定
	var keys []string
	for k := range node.Options {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := node.Options[key]
		if key == "alterId" && node.Type == "vmess" {
			continue
		}
		s.writeYAMLValue(&builder, optionsIndentStr, key, value, 2)
	}

	return builder.String()
}

// writeYAMLValue 递归写入 YAML 值
func (s *ConfigUpdateService) writeYAMLValue(builder *strings.Builder, indentStr, key string, value interface{}, indentLevel int) {
	escapedKey := s.escapeYAMLString(key)

	switch v := value.(type) {
	case map[string]interface{}:
		builder.WriteString(fmt.Sprintf("%s%s:\n", indentStr, escapedKey))
		subIndentStr := indentStr + "  "

		// 特殊处理 http-opts
		if key == "http-opts" {
			s.writeHTTPOpts(builder, subIndentStr, v)
			return
		}

		for k, val := range v {
			if strMap, ok := val.(map[string]string); ok {
				escapedK := s.escapeYAMLString(k)
				builder.WriteString(fmt.Sprintf("%s%s:\n", subIndentStr, escapedK))
				subSubIndentStr := subIndentStr + "  "
				for k2, v2 := range strMap {
					escapedK2 := s.escapeYAMLString(k2)
					escapedV2 := s.escapeYAMLString(v2)
					builder.WriteString(fmt.Sprintf("%s%s: %s\n", subSubIndentStr, escapedK2, escapedV2))
				}
			} else {
				s.writeYAMLValue(builder, subIndentStr, k, val, indentLevel+1)
			}
		}
	case []interface{}:
		builder.WriteString(fmt.Sprintf("%s%s:\n", indentStr, escapedKey))
		subIndentStr := indentStr + "  "
		for _, item := range v {
			builder.WriteString(fmt.Sprintf("%s- ", subIndentStr))
			s.writeYAMLValueInline(builder, item)
			builder.WriteString("\n")
		}
	case []string:
		builder.WriteString(fmt.Sprintf("%s%s:\n", indentStr, escapedKey))
		subIndentStr := indentStr + "  "
		for _, item := range v {
			escapedItem := s.escapeYAMLString(item)
			builder.WriteString(fmt.Sprintf("%s- %s\n", subIndentStr, escapedItem))
		}
	default:
		escapedVal := s.escapeYAMLString(fmt.Sprintf("%v", v))
		builder.WriteString(fmt.Sprintf("%s%s: %s\n", indentStr, escapedKey, escapedVal))
	}
}

// writeHTTPOpts 辅助写入 http-opts
func (s *ConfigUpdateService) writeHTTPOpts(builder *strings.Builder, indentStr string, v map[string]interface{}) {
	for k, val := range v {
		if k == "path" {
			s.writeYAMLList(builder, indentStr, k, val)
		} else if k == "headers" {
			escapedK := s.escapeYAMLString(k)
			builder.WriteString(fmt.Sprintf("%s%s:\n", indentStr, escapedK))
			subIndentStr := indentStr + "  "
			if headersMap, ok := val.(map[string]interface{}); ok {
				for hk, hv := range headersMap {
					s.writeYAMLList(builder, subIndentStr, hk, hv)
				}
			}
		}
	}
}

// writeYAMLList 辅助写入列表配置
func (s *ConfigUpdateService) writeYAMLList(builder *strings.Builder, indentStr, key string, val interface{}) {
	escapedK := s.escapeYAMLString(key)
	builder.WriteString(fmt.Sprintf("%s%s:\n", indentStr, escapedK))
	subIndentStr := indentStr + "  "

	writeItem := func(item interface{}) {
		escapedItem := s.escapeYAMLString(fmt.Sprintf("%v", item))
		builder.WriteString(fmt.Sprintf("%s- %s\n", subIndentStr, escapedItem))
	}

	if str, ok := val.(string); ok {
		writeItem(str)
	} else if slice, ok := val.([]string); ok {
		for _, item := range slice {
			writeItem(item)
		}
	} else if slice, ok := val.([]interface{}); ok {
		for _, item := range slice {
			writeItem(item)
		}
	}
}

// writeYAMLValueInline 内联写入 YAML 值
func (s *ConfigUpdateService) writeYAMLValueInline(builder *strings.Builder, value interface{}) {
	switch v := value.(type) {
	case string:
		builder.WriteString(s.escapeYAMLString(v))
	case int, int64, float64, bool:
		builder.WriteString(fmt.Sprintf("%v", v))
	default:
		builder.WriteString(s.escapeYAMLString(fmt.Sprintf("%v", v)))
	}
}

// escapeYAMLString 转义 YAML 字符串
func (s *ConfigUpdateService) escapeYAMLString(str string) string {
	if str == "" {
		return "\"\""
	}
	needsQuotes := false
	specialChars := []string{":", "\"", "'", "\n", "\r", "\t", "#", "@", "&", "*", "?", "|", ">", "!", "%", "`", "[", "]", "{", "}", ","}
	for _, char := range specialChars {
		if strings.Contains(str, char) {
			needsQuotes = true
			break
		}
	}
	if strings.HasPrefix(str, " ") || strings.HasSuffix(str, " ") {
		needsQuotes = true
	}
	if needsQuotes {
		escaped := strings.ReplaceAll(str, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		escaped = strings.ReplaceAll(escaped, "\n", "\\n")
		return fmt.Sprintf("\"%s\"", escaped)
	}
	return str
}

// ==========================================
// Utils & Helpers
// ==========================================

// NodeToLink 将节点转换为通用链接（公开方法）
func (s *ConfigUpdateService) NodeToLink(node *ProxyNode) string {
	return s.nodeToLink(node)
}

// nodeToLink 将节点转换为通用链接
func (s *ConfigUpdateService) nodeToLink(node *ProxyNode) string {
	switch node.Type {
	case "vmess":
		return s.vmessToLink(node)
	case "vless":
		return s.vlessToLink(node)
	case "trojan":
		return s.trojanToLink(node)
	case "ss":
		return s.shadowsocksToLink(node)
	case "ssr":
		return s.nodeToSSRLink(node)
	case "hysteria":
		return s.hysteriaToLink(node)
	case "hysteria2":
		return s.hysteria2ToLink(node)
	case "tuic":
		return s.tuicToLink(node)
	case "naive":
		return s.naiveToLink(node)
	case "anytls":
		return s.anytlsToLink(node)
	default:
		return ""
	}
}

func (s *ConfigUpdateService) vmessToLink(proxy *ProxyNode) string {
	data := map[string]interface{}{
		"v":    "2",
		"ps":   proxy.Name,
		"add":  proxy.Server,
		"port": proxy.Port,
		"id":   proxy.UUID,
		"net":  proxy.Network,
		"type": "none",
	}

	if proxy.TLS {
		data["tls"] = "tls"
	}

	if proxy.Options != nil {
		if wsOpts, ok := proxy.Options["ws-opts"].(map[string]interface{}); ok {
			if path, ok := wsOpts["path"].(string); ok {
				data["path"] = path
			}
			if headers, ok := wsOpts["headers"].(map[string]interface{}); ok {
				if host, ok := headers["Host"].(string); ok {
					data["host"] = host
				}
			}
		}
	}

	jsonData, _ := json.Marshal(data)
	encoded := base64.StdEncoding.EncodeToString(jsonData)
	return "vmess://" + encoded
}

func (s *ConfigUpdateService) vlessToLink(proxy *ProxyNode) string {
	u := &url.URL{
		Scheme:   "vless",
		User:     url.User(proxy.UUID),
		Host:     fmt.Sprintf("%s:%d", proxy.Server, proxy.Port),
		Fragment: proxy.Name,
	}

	q := url.Values{}
	if proxy.Network != "" {
		q.Set("type", proxy.Network)
	}
	if proxy.TLS {
		q.Set("security", "tls")
	}

	u.RawQuery = q.Encode()
	return u.String()
}

func (s *ConfigUpdateService) trojanToLink(proxy *ProxyNode) string {
	u := &url.URL{
		Scheme:   "trojan",
		User:     url.User(proxy.Password),
		Host:     fmt.Sprintf("%s:%d", proxy.Server, proxy.Port),
		Fragment: proxy.Name,
	}
	return u.String()
}

func (s *ConfigUpdateService) shadowsocksToLink(proxy *ProxyNode) string {
	auth := fmt.Sprintf("%s:%s", proxy.Cipher, proxy.Password)
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	u := &url.URL{
		Scheme:   "ss",
		User:     url.User(encoded),
		Host:     fmt.Sprintf("%s:%d", proxy.Server, proxy.Port),
		Fragment: proxy.Name,
	}
	return u.String()
}

func (s *ConfigUpdateService) nodeToSSRLink(node *ProxyNode) string {
	if node.Type != "ssr" && node.Type != "ss" {
		return ""
	}

	getString := func(opts map[string]interface{}, key, defaultValue string) string {
		if v, ok := opts[key].(string); ok {
			return v
		}
		return defaultValue
	}

	server := node.Server
	port := node.Port
	protocol := getString(node.Options, "protocol", "origin")
	method := node.Cipher
	obfs := getString(node.Options, "obfs", "plain")
	password := base64.RawURLEncoding.EncodeToString([]byte(node.Password))

	obfsparam := base64.RawURLEncoding.EncodeToString([]byte(getString(node.Options, "obfs-param", "")))
	protoparam := base64.RawURLEncoding.EncodeToString([]byte(getString(node.Options, "protocol-param", "")))
	remarks := base64.RawURLEncoding.EncodeToString([]byte(node.Name))
	group := base64.RawURLEncoding.EncodeToString([]byte("GoWeb"))

	ssrStr := fmt.Sprintf("%s:%d:%s:%s:%s:%s/?obfsparam=%s&protoparam=%s&remarks=%s&group=%s",
		server, port, protocol, method, obfs, password,
		obfsparam, protoparam, remarks, group)

	return "ssr://" + base64.RawURLEncoding.EncodeToString([]byte(ssrStr))
}

func (s *ConfigUpdateService) hysteriaToLink(proxy *ProxyNode) string {
	u := &url.URL{
		Scheme:   "hysteria",
		Host:     fmt.Sprintf("%s:%d", proxy.Server, proxy.Port),
		Fragment: proxy.Name,
	}

	q := url.Values{}
	if proxy.Options != nil {
		if auth, ok := proxy.Options["auth"].(string); ok && auth != "" {
			q.Set("auth", auth)
		}
		if up, ok := proxy.Options["up"].(string); ok && up != "" {
			// 移除 " mbps" 后缀
			up = strings.TrimSuffix(up, " mbps")
			q.Set("upmbps", up)
		}
		if down, ok := proxy.Options["down"].(string); ok && down != "" {
			// 移除 " mbps" 后缀
			down = strings.TrimSuffix(down, " mbps")
			q.Set("downmbps", down)
		}
		if skipCert, ok := proxy.Options["skip-cert-verify"].(bool); ok && skipCert {
			q.Set("insecure", "1")
		}
	}

	u.RawQuery = q.Encode()
	return u.String()
}

func (s *ConfigUpdateService) hysteria2ToLink(proxy *ProxyNode) string {
	u := &url.URL{
		Scheme:   "hysteria2",
		User:     url.User(proxy.Password),
		Host:     fmt.Sprintf("%s:%d", proxy.Server, proxy.Port),
		Fragment: proxy.Name,
	}

	q := url.Values{}
	if proxy.Options != nil {
		if up, ok := proxy.Options["up"].(string); ok && up != "" {
			// 移除 " mbps" 后缀
			up = strings.TrimSuffix(up, " mbps")
			q.Set("mbpsUp", up)
		}
		if down, ok := proxy.Options["down"].(string); ok && down != "" {
			// 移除 " mbps" 后缀
			down = strings.TrimSuffix(down, " mbps")
			q.Set("mbpsDown", down)
		}
		if skipCert, ok := proxy.Options["skip-cert-verify"].(bool); ok && skipCert {
			q.Set("insecure", "1")
		}
		if sni, ok := proxy.Options["servername"].(string); ok && sni != "" {
			q.Set("sni", sni)
		} else if peer, ok := proxy.Options["peer"].(string); ok && peer != "" {
			q.Set("peer", peer)
		}
		if alpn, ok := proxy.Options["alpn"].([]string); ok && len(alpn) > 0 {
			q.Set("alpn", strings.Join(alpn, ","))
		} else if alpn, ok := proxy.Options["alpn"].([]interface{}); ok && len(alpn) > 0 {
			alpnStrs := make([]string, 0, len(alpn))
			for _, v := range alpn {
				if str, ok := v.(string); ok {
					alpnStrs = append(alpnStrs, str)
				}
			}
			if len(alpnStrs) > 0 {
				q.Set("alpn", strings.Join(alpnStrs, ","))
			}
		}
	}

	u.RawQuery = q.Encode()
	return u.String()
}

func (s *ConfigUpdateService) tuicToLink(proxy *ProxyNode) string {
	userInfo := url.UserPassword(proxy.UUID, proxy.Password)
	u := &url.URL{
		Scheme:   "tuic",
		User:     userInfo,
		Host:     fmt.Sprintf("%s:%d", proxy.Server, proxy.Port),
		Fragment: proxy.Name,
	}

	q := url.Values{}
	if proxy.Options != nil {
		if sni, ok := proxy.Options["servername"].(string); ok && sni != "" {
			q.Set("sni", sni)
		}
		if alpn, ok := proxy.Options["alpn"].([]string); ok && len(alpn) > 0 {
			q.Set("alpn", alpn[0]) // TUIC 通常只支持单个 ALPN
		} else if alpn, ok := proxy.Options["alpn"].([]interface{}); ok && len(alpn) > 0 {
			if str, ok := alpn[0].(string); ok {
				q.Set("alpn", str)
			}
		}
		if cc, ok := proxy.Options["congestion_control"].(string); ok && cc != "" {
			q.Set("congestion_control", cc)
		}
		if udpRelayMode, ok := proxy.Options["udp_relay_mode"].(string); ok && udpRelayMode != "" {
			q.Set("udp_relay_mode", udpRelayMode)
		}
		if skipCert, ok := proxy.Options["skip-cert-verify"].(bool); ok && skipCert {
			q.Set("allow_insecure", "1")
		}
	}

	u.RawQuery = q.Encode()
	return u.String()
}

func (s *ConfigUpdateService) naiveToLink(proxy *ProxyNode) string {
	// Naive 使用 UUID 作为 username，Password 作为 password
	userInfo := url.UserPassword(proxy.UUID, proxy.Password)
	u := &url.URL{
		Scheme:   "naive+https",
		User:     userInfo,
		Host:     fmt.Sprintf("%s:%d", proxy.Server, proxy.Port),
		Fragment: proxy.Name,
	}

	q := url.Values{}
	if proxy.Options != nil {
		if sni, ok := proxy.Options["servername"].(string); ok && sni != "" {
			q.Set("sni", sni)
		}
		if padding, ok := proxy.Options["padding"].(bool); ok && padding {
			q.Set("padding", "true")
		}
		if skipCert, ok := proxy.Options["skip-cert-verify"].(bool); ok && skipCert {
			q.Set("insecure", "1")
		}
	}

	u.RawQuery = q.Encode()
	return u.String()
}

func (s *ConfigUpdateService) anytlsToLink(proxy *ProxyNode) string {
	u := &url.URL{
		Scheme:   "anytls",
		User:     url.User(proxy.UUID),
		Host:     fmt.Sprintf("%s:%d", proxy.Server, proxy.Port),
		Fragment: proxy.Name,
	}

	q := url.Values{}
	if proxy.Options != nil {
		if peer, ok := proxy.Options["peer"].(string); ok && peer != "" {
			q.Set("peer", peer)
		} else if sni, ok := proxy.Options["servername"].(string); ok && sni != "" {
			q.Set("sni", sni)
		}
		if skipCert, ok := proxy.Options["skip-cert-verify"].(bool); ok && skipCert {
			q.Set("insecure", "1")
		}
	}

	u.RawQuery = q.Encode()
	return u.String()
}

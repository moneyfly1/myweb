package config_update

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"cboard-go/internal/services/cache_service"
)

// ==========================================
// 常量与类型定义
// ==========================================

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

var nodeLinkPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?:^|\s)(vmess://[^\s]+)`),
	regexp.MustCompile(`(?:^|\s)(vless://[^\s]+)`),
	regexp.MustCompile(`(?:^|\s)(trojan://[^\s]+)`),
	regexp.MustCompile(`(?:^|\s)(ss://[^\s]+)`),
	regexp.MustCompile(`(?:^|\s)(ssr://[^\s]+)`),
	regexp.MustCompile(`(?:^|\s)(hysteria://[^\s]+)`),
	regexp.MustCompile(`(?:^|\s)(hysteria2://[^\s]+)`),
	regexp.MustCompile(`(?:^|\s)(tuic://[^\s]+)`),
	regexp.MustCompile(`(?:^|\s)(naive\+https://[^\s]+)`),
	regexp.MustCompile(`(?:^|\s)(naive://[^\s]+)`),
	regexp.MustCompile(`(?:^|\s)(anytls://[^\s]+)`),
	regexp.MustCompile(`(?:^|\s)(socks5://[^\s]+)`),
	regexp.MustCompile(`(?:^|\s)(socks://[^\s]+)`),
	regexp.MustCompile(`(?:^|\s)(http://[^\s]+)`),
	regexp.MustCompile(`(?:^|\s)(https://[^\s]+)`),
}

var supportedClashTypes = map[string]bool{
	"vmess":     true,
	"vless":     true,
	"trojan":    true,
	"ss":        true,
	"ssr":       true,
	"hysteria":  true,
	"hysteria2": true,
	"tuic":      true,
	"anytls":    true,
	"socks":     true,
	"socks5":    true,
	"http":      true,
	"wireguard": true,
	"direct":    true,
}

type SubscriptionContext struct {
	User           models.User
	Subscription   models.Subscription
	Proxies        []*ProxyNode
	Status         SubscriptionStatus
	ResetRecord    *models.SubscriptionReset
	CurrentDevices int
	DeviceLimit    int
}

type ConfigUpdateService struct {
	db            *gorm.DB
	isRunning     bool
	runningMutex  sync.Mutex
	siteURL       string
	supportQQ     string
	regionMatcher *RegionMatcher
	parserPool    *ParserPool
	sseManager    *SSEManager                 // SSE 管理器
	logBuffer     []map[string]interface{}    // 内存日志缓冲
	logMutex      sync.RWMutex                // 日志缓冲锁
}

type nodeWithOrder struct {
	node        *ProxyNode
	orderIndex  int
	sourceIndex int // 来源订阅编号（1开始）
}

type updateStats struct {
	parseFailed   int
	duplicates    int
	invalidLinks  int
	missingSource int
	filtered      int
}

// ==========================================
// 初始化与生命周期
// ==========================================

func NewConfigUpdateService() *ConfigUpdateService {
	service := &ConfigUpdateService{
		db:         database.GetDB(),
		parserPool: NewParserPool(10),
		sseManager: NewSSEManager(),
		logBuffer:  make([]map[string]interface{}, 0, 500),
	}

	regionConfig, err := LoadRegionConfig()
	if err != nil {
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
		service.regionMatcher = NewRegionMatcher(make(map[string]string), make(map[string]string))
		if utils.AppLogger != nil {
			utils.AppLogger.Warn("使用空的地区匹配器（所有节点将显示为'未知'地区）")
		}
	}

	service.refreshSystemConfig()
	return service
}

func (s *ConfigUpdateService) loadLegacyRegionMaps() {
	// 占位符，保持原样
}

func (s *ConfigUpdateService) refreshSystemConfig() {
	domain := utils.GetDomainFromDB(s.db)
	if domain != "" {
		s.siteURL = utils.FormatDomainURL(domain)
	} else {
		s.siteURL = "请在系统设置中配置域名"
	}

	var supportQQConfig models.SystemConfig
	if err := s.db.Where("key = ? AND category = ?", "support_qq", "general").First(&supportQQConfig).Error; err == nil && supportQQConfig.Value != "" {
		s.supportQQ = strings.TrimSpace(supportQQConfig.Value)
	} else {
		s.supportQQ = ""
	}
}

func (s *ConfigUpdateService) IsRunning() bool {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()
	return s.isRunning
}

// ==========================================
// 状态与日志管理
// ==========================================

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

func (s *ConfigUpdateService) GetLogs(limit int) []map[string]interface{} {
	// 优先从内存缓冲读取（实时日志）
	s.logMutex.RLock()
	if len(s.logBuffer) > 0 {
		logs := make([]map[string]interface{}, len(s.logBuffer))
		copy(logs, s.logBuffer)
		s.logMutex.RUnlock()
		return s.limitLogs(logs, limit)
	}
	s.logMutex.RUnlock()

	// 如果内存中没有，从数据库读取（历史日志）
	return s.getLogsFromDB(limit)
}

// getLogsFromDB 从数据库读取日志
func (s *ConfigUpdateService) getLogsFromDB(limit int) []map[string]interface{} {
	var config models.SystemConfig
	if err := s.db.Where("key = ?", "config_update_logs").First(&config).Error; err != nil {
		return []map[string]interface{}{}
	}

	var logs []map[string]interface{}
	if err := json.Unmarshal([]byte(config.Value), &logs); err != nil {
		return []map[string]interface{}{}
	}

	return s.limitLogs(logs, limit)
}

// limitLogs 限制日志数量
func (s *ConfigUpdateService) limitLogs(logs []map[string]interface{}, limit int) []map[string]interface{} {
	if len(logs) > limit {
		return logs[len(logs)-limit:]
	}
	return logs
}

func (s *ConfigUpdateService) ClearLogs() error {
	// 清空内存缓冲
	s.clearLogBuffer()

	// 清空 SSE 历史
	if s.sseManager != nil {
		s.sseManager.ClearHistory()
	}

	// 清空数据库
	return s.clearLogsInDB()
}

// clearLogBuffer 清空内存日志缓冲
func (s *ConfigUpdateService) clearLogBuffer() {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()
	s.logBuffer = make([]map[string]interface{}, 0, 500)
}

// clearLogsInDB 清空数据库中的日志
func (s *ConfigUpdateService) clearLogsInDB() error {
	var config models.SystemConfig
	err := s.db.Where("key = ?", "config_update_logs").First(&config).Error
	if err != nil {
		return s.saveLogConfig("[]")
	}
	config.Value = "[]"
	return s.db.Save(&config).Error
}

// GetSSEManager 获取 SSE 管理器
func (s *ConfigUpdateService) GetSSEManager() *SSEManager {
	return s.sseManager
}

func (s *ConfigUpdateService) log(level, message string) {
	logEntry := s.createLogEntry(level, message)

	// 写入内存缓冲
	s.addLogToBuffer(logEntry)

	// 通过 SSE 实时广播
	if s.sseManager != nil {
		s.sseManager.Broadcast(logEntry)
	}

	// 写入应用日志
	s.writeToAppLogger(level, message)
}

// createLogEntry 创建日志条目
func (s *ConfigUpdateService) createLogEntry(level, message string) map[string]interface{} {
	now := utils.FormatBeijingTime(utils.GetBeijingTime())
	return map[string]interface{}{
		"timestamp": now,
		"time":      now,
		"level":     level,
		"message":   message,
	}
}

// addLogToBuffer 添加日志到内存缓冲
func (s *ConfigUpdateService) addLogToBuffer(logEntry map[string]interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.logBuffer = append(s.logBuffer, logEntry)
	if len(s.logBuffer) > 500 {
		s.logBuffer = s.logBuffer[len(s.logBuffer)-500:]
	}
}

// writeToAppLogger 写入应用日志
func (s *ConfigUpdateService) writeToAppLogger(level, message string) {
	if utils.AppLogger != nil {
		if level == "ERROR" {
			utils.AppLogger.Error("%s", message)
		} else {
			utils.AppLogger.Info("%s", message)
		}
	}
}

// flushLogsToDB 批量保存内存中的日志到数据库
func (s *ConfigUpdateService) flushLogsToDB() error {
	s.logMutex.RLock()
	if len(s.logBuffer) == 0 {
		s.logMutex.RUnlock()
		return nil
	}

	// 复制日志缓冲
	logs := make([]map[string]interface{}, len(s.logBuffer))
	copy(logs, s.logBuffer)
	s.logMutex.RUnlock()

	// 批量写入数据库
	logsJSON, err := json.Marshal(logs)
	if err != nil {
		return err
	}

	var config models.SystemConfig
	err = s.db.Where("key = ?", "config_update_logs").First(&config).Error
	if err != nil {
		// 不存在则创建
		return s.saveLogConfig(string(logsJSON))
	}

	config.Value = string(logsJSON)
	return s.db.Save(&config).Error
}

func (s *ConfigUpdateService) saveLogConfig(value string) error {
	config := models.SystemConfig{
		Key:         "config_update_logs",
		Value:       value,
		Type:        "json",
		Category:    "config_update",
		DisplayName: "配置更新日志",
		Description: "配置更新任务日志",
	}
	return s.db.Create(&config).Error
}

// ==========================================
// 任务执行逻辑
// ==========================================

func (s *ConfigUpdateService) GetConfig() (map[string]interface{}, error) {
	return s.getConfig()
}

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

	s.logSection("🚀", "开始执行配置更新任务")

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

	s.log("INFO", "📋 配置信息")
	s.logItem("└─", fmt.Sprintf("节点源数量: %d 个", len(urls)))

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

	filterKeywords := s.extractFilterKeywords(config)
	if len(filterKeywords) > 0 {
		s.log("INFO", fmt.Sprintf("已配置 %d 个过滤关键词，将过滤包含这些关键词的节点", len(filterKeywords)))
	} else {
		s.log("DEBUG", "未配置过滤关键词，将不过滤任何节点")
	}

	nodesWithOrder, stats := s.processFetchedNodes(urls, nodes, filterKeywords)

	s.logUpdateStats(stats, len(nodesWithOrder))

	// 在导入新节点之前，删除所有自动导入的节点（IsManual = false），保留手动添加的节点（IsManual = true，包括专线节点）
	deletedCount := s.deleteAutoImportedNodes()
	s.logSeparator()
	s.log("INFO", "💾 数据库操作")
	s.logSeparator()
	s.log("INFO", fmt.Sprintf("🗑️  删除旧节点: %d 个", deletedCount))

	s.log("INFO", "⠼ 正在导入节点到数据库...")
	importStats := s.importNodesToDatabaseWithOrder(nodesWithOrder)
	s.updateLastUpdateTime()

	// 输出详细的统计信息
	s.log("INFO", fmt.Sprintf("➕ 新增节点: %d 个", importStats.Created))
	s.log("INFO", fmt.Sprintf("🔄 更新节点: %d 个", importStats.Updated))
	s.log("INFO", fmt.Sprintf("⏭️  跳过节点: %d 个 (手动添加)", importStats.Skipped))

	// 等待日志保存完成，确保最终日志能够保存
	time.Sleep(100 * time.Millisecond)

	s.logSection("✅", "任务完成")
	s.logItem("└─", fmt.Sprintf("最终结果: 成功导入 %d 个节点", importStats.Created))


	// 清除节点和订阅配置缓存
	cs := cache_service.NewCacheService()
	if err := cs.ClearNodesCache(); err != nil {
		log.Printf("failed to clear nodes cache: %v", err)
	}
	if err := (&CacheService{}).ClearSystemNodesCache(); err != nil {
		log.Printf("failed to clear system nodes cache: %v", err)
	}
	if err := (&CacheService{}).ClearAllSubscriptionCache(); err != nil {
		log.Printf("failed to clear all subscription cache: %v", err)
	}
	s.logSeparator()
	s.log("INFO", "🧹 清理缓存")
	s.logSeparator()
	s.log("INFO", "✓ 节点列表缓存已清除")
	s.log("INFO", "✓ 系统节点缓存已清除")
	s.log("INFO", "✓ 订阅配置缓存已清除")

	// 任务完成，批量保存日志到数据库
	s.log("INFO", "💾 保存日志到数据库...")
	if err := s.flushLogsToDB(); err != nil {
		s.log("ERROR", fmt.Sprintf("保存日志失败: %v", err))
	} else {
		s.log("INFO", "✓ 日志已保存")
	}

	// 等待最后的日志广播完成
	time.Sleep(100 * time.Millisecond)

	return nil
}

func (s *ConfigUpdateService) extractFilterKeywords(config map[string]interface{}) []string {
	var filterKeywords []string
	if keywords, ok := config["filter_keywords"].([]string); ok {
		filterKeywords = keywords
	} else if keywordsStr, ok := config["filter_keywords"].(string); ok && keywordsStr != "" {
		for _, kw := range strings.Split(keywordsStr, "\n") {
			if kw = strings.TrimSpace(kw); kw != "" {
				filterKeywords = append(filterKeywords, kw)
			}
		}
	}
	return filterKeywords
}

func (s *ConfigUpdateService) logUpdateStats(stats updateStats, successCount int) {
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
	s.log("INFO", fmt.Sprintf("成功解析并准备入库的节点: %d 个", successCount))
}

func (s *ConfigUpdateService) processFetchedNodes(urls []string, nodes []map[string]interface{}, filterKeywords []string) ([]nodeWithOrder, updateStats) {
	var nodesWithOrder []nodeWithOrder
	stats := updateStats{}
	seenKeys := make(map[string]bool)
	usedNames := make(map[string]bool)

	nodesByURL := make(map[string][]map[string]interface{})
	for _, nodeInfo := range nodes {
		sourceURL, _ := nodeInfo["source_url"].(string)
		if sourceURL == "" {
			stats.missingSource++
			continue
		}
		nodesByURL[sourceURL] = append(nodesByURL[sourceURL], nodeInfo)
	}

	for urlIndex, url := range urls {
		urlNodes := nodesByURL[url]
		if len(urlNodes) == 0 {
			continue
		}

		s.log("INFO", fmt.Sprintf("⠸ 开始处理订阅地址 [%d/%d] 的节点，共 %d 个链接", urlIndex+1, len(urls), len(urlNodes)))

		links := make([]string, 0, len(urlNodes))
		for _, nodeInfo := range urlNodes {
			link, ok := nodeInfo["url"].(string)
			if !ok {
				stats.invalidLinks++
				continue
			}
			links = append(links, link)
		}

		if len(links) == 0 {
			continue
		}

		results := s.parserPool.ParseLinks(links)
		nodeIndexInURL := 0
		counts := struct{ Processed, Failed, Filtered, Duplicate int }{}

		for _, result := range results {
			link := result.Link
			if seenKeys[link] {
				stats.duplicates++
				counts.Duplicate++
				continue
			}
			seenKeys[link] = true

			if result.Err != nil || result.Node == nil {
				stats.parseFailed++
				counts.Failed++
				if counts.Failed <= 10 && result.Err != nil {
					s.log("WARN", fmt.Sprintf("解析失败 [订阅地址 %d/%d]: %v, 链接: %s",
						urlIndex+1, len(urls), result.Err, truncateString(link, 50)))
				}
				continue
			}

			node := result.Node
			if filtered, keyword := s.isNodeFiltered(node, filterKeywords); filtered {
				stats.filtered++
				counts.Filtered++
				s.log("DEBUG", fmt.Sprintf("节点被过滤: %s (关键词: %s)", node.Name, keyword))
				continue
			}

			counts.Processed++
			node.Name = s.ensureUniqueName(node.Name, usedNames)
			usedNames[node.Name] = true

			nodesWithOrder = append(nodesWithOrder, nodeWithOrder{
				node:        node,
				orderIndex:  urlIndex*10000 + nodeIndexInURL,
				sourceIndex: urlIndex + 1,
			})
			nodeIndexInURL++
		}

		s.log("INFO", fmt.Sprintf("✓ 订阅地址 [%d/%d] 完成: 成功=%d, 失败=%d, 过滤=%d, 重复=%d",
			urlIndex+1, len(urls), counts.Processed, counts.Failed, counts.Filtered, counts.Duplicate))
	}
	return nodesWithOrder, stats
}

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

func (s *ConfigUpdateService) getConfig() (map[string]interface{}, error) {
	var configs []models.SystemConfig
	s.db.Where("category = ?", "config_update").Find(&configs)

	result := map[string]interface{}{
		"urls":              []string{},
		"filter_keywords":   []string{},
		"enable_schedule":   false,
		"schedule_interval": 3600,
	}

	for _, config := range configs {
		switch config.Key {
		case "urls":
			// 内联替代 utils.SplitLinesFilterEmpty
			var urls []string
			for _, line := range strings.Split(config.Value, "\n") {
				if trimmed := strings.TrimSpace(line); trimmed != "" {
					urls = append(urls, trimmed)
				}
			}
			result["urls"] = urls
		case "filter_keywords":
			// 内联替代 utils.SplitLinesFilterEmpty
			var kws []string
			for _, line := range strings.Split(config.Value, "\n") {
				if trimmed := strings.TrimSpace(line); trimmed != "" {
					kws = append(kws, trimmed)
				}
			}
			result["filter_keywords"] = kws
		case "enable_schedule":
			result[config.Key] = config.Value == "true" || config.Value == "1"
		case "schedule_interval":
			interval, _ := strconv.Atoi(config.Value)
			if interval == 0 {
				interval = 3600
			}
			result[config.Key] = interval
		default:
			result[config.Key] = config.Value
		}
	}

	return result, nil
}

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

// ==========================================
// 节点获取与解析
// ==========================================

func (s *ConfigUpdateService) FetchNodesFromURLs(urls []string) ([]map[string]interface{}, error) {
	var allNodes []map[string]interface{}
	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: false,
			MaxIdleConns:      10,
			IdleConnTimeout:   30 * time.Second,
		},
	}

	for i, url := range urls {
		s.logSeparator()
		s.log("INFO", fmt.Sprintf("📥 下载节点源 [%d/%d]", i+1, len(urls)))
		s.logSeparator()
		s.logItem("└─", fmt.Sprintf("URL: %s", url))
		s.log("INFO", "⠋ 正在连接...")

		content, err := s.fetchURLContent(client, url)
		if err != nil {
			s.log("ERROR", fmt.Sprintf("获取节点源失败: %v", err))
			continue
		}

		decoded := TryDecodeNodeList(string(content))

		s.log("INFO", "⠹ 正在解析节点...")
		nodeLinks := s.extractNodeLinks(decoded)
		s.logNodeTypeStats(url, nodeLinks)
		s.logNodeNames(nodeLinks)

		for _, link := range nodeLinks {
			allNodes = append(allNodes, map[string]interface{}{
				"url":        link,
				"source_url": url,
			})
		}
	}

	return allNodes, nil
}

func (s *ConfigUpdateService) fetchURLContent(client *http.Client, url string) ([]byte, error) {
	maxRetries := 3
	retryDelay := 2 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		content, err := s.doFetch(client, url)
		if err == nil {
			return content, nil
		}

		if attempt < maxRetries {
			s.log("WARN", fmt.Sprintf("下载失败 (尝试 %d/%d): %v，%v 后重试", attempt, maxRetries, err, retryDelay))
			time.Sleep(retryDelay)
			retryDelay *= 2
		} else {
			return nil, err
		}
	}
	return nil, fmt.Errorf("多次重试后失败")
}

func (s *ConfigUpdateService) doFetch(client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	if strings.Contains(url, "gist.githubusercontent.com") {
		req.Header.Set("Connection", "close")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("状态码错误: %d", resp.StatusCode)
	}

	limitedReader := io.LimitReader(resp.Body, 10*1024*1024) // 10MB Limit
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("内容为空")
	}
	return content, nil
}

func (s *ConfigUpdateService) logNodeTypeStats(url string, nodeLinks []string) {
	typeCount := make(map[string]int)
	for _, link := range nodeLinks {
		scheme := strings.Split(link, ":")[0]
		typeCount[scheme]++
	}

	var parts []string
	for t, c := range typeCount {
		parts = append(parts, fmt.Sprintf("%s:%d", t, c))
	}
	s.log("INFO", fmt.Sprintf("✓ 提取到 %d 个节点 (%s)", len(nodeLinks), strings.Join(parts, ", ")))
}

func (s *ConfigUpdateService) logNodeNames(nodeLinks []string) {
	if len(nodeLinks) == 0 {
		return
	}

	// 从节点链接中提取节点名称
	var nodeNames []string
	for _, link := range nodeLinks {
		name := s.extractNodeName(link)
		if name != "" {
			nodeNames = append(nodeNames, name)
		}
	}

	if len(nodeNames) > 0 {
		s.logSeparator()
		s.log("INFO", "📋 采集到的节点:")
		for i, name := range nodeNames {
			s.log("INFO", fmt.Sprintf("  %d. %s", i+1, name))
		}
		s.logSeparator()
	}
}

func (s *ConfigUpdateService) extractNodeName(link string) string {
	// 节点名称通常在 # 后面
	if idx := strings.Index(link, "#"); idx != -1 {
		name := link[idx+1:]
		// URL decode
		if decoded, err := url.QueryUnescape(name); err == nil {
			return decoded
		}
		return name
	}
	return ""
}

func (s *ConfigUpdateService) extractNodeLinks(content string) []string {
	// 首先尝试解析 Clash YAML 格式
	if yamlLinks := s.parseClashYAML(content); len(yamlLinks) > 0 {
		return yamlLinks
	}

	// 如果不是 YAML 格式，则使用原有的正则提取逻辑
	s.log("INFO", "✓ 检测到节点链接格式（非 YAML）")
	var links []string
	var invalidLinks []string
	// 使用 bitset 或区间树会更高效，但这里用 map[int]bool 简单处理
	matchedPositions := make(map[int]bool)

	for _, re := range nodeLinkPatterns {
		matches := re.FindAllStringSubmatchIndex(content, -1)
		for _, match := range matches {
			// match[0], match[1] 是整个匹配的起止（包括前缀空格）
			// match[2], match[3] 是捕获组的起止（真正的链接）
			if len(match) < 4 {
				continue
			}
			start, end := match[2], match[3]
			matchStr := content[start:end]

			// VME 过滤逻辑
			if strings.HasPrefix(matchStr, "ss://") && start >= 3 {
				if content[start-3:start] == "vme" {
					continue
				}
			}

			// 重叠检查
			isOverlapped := false
			for pos := start; pos < end; pos++ {
				if matchedPositions[pos] {
					isOverlapped = true
					break
				}
			}
			if isOverlapped {
				continue
			}
			// 标记位置
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

	// 记录无效链接样本
	if len(invalidLinks) > 0 {
		limit := 3
		if len(invalidLinks) < limit {
			limit = len(invalidLinks)
		}
		s.log("DEBUG", fmt.Sprintf("发现 %d 个无效链接，示例: %v", len(invalidLinks), invalidLinks[:limit]))
	}

	// 内联替代 utils.RemoveDuplicates
	uniqueMap := make(map[string]bool)
	var uniqueLinks []string
	for _, link := range links {
		if !uniqueMap[link] {
			uniqueMap[link] = true
			uniqueLinks = append(uniqueLinks, link)
		}
	}
	return uniqueLinks
}

func (s *ConfigUpdateService) isValidNodeLink(link string) bool {
	link = strings.TrimSpace(link)
	if link == "" {
		return false
	}

	parts := strings.SplitN(link, ":", 2)
	if len(parts) != 2 {
		return false
	}
	scheme := parts[0]
	// 移除fragment
	body := strings.Split(link, "#")[0]

	switch scheme {
	case "ss":
		// ss://method:pass@server:port or ss://base64@server:port
		if !strings.Contains(body, "@") {
			return false
		}
		serverPart := strings.Split(body, "@")[1]
		serverPart = strings.Split(serverPart, "?")[0]
		return strings.Contains(serverPart, ":")
	case "vmess", "vless", "ssr":
		// 需要一定长度
		encoded := strings.TrimPrefix(body, scheme+"://")
		return len(strings.Split(encoded, "?")[0]) >= 10
	case "trojan", "tuic", "naive+https", "socks", "socks5", "http", "https":
		// user:pass@host:port
		return strings.Contains(body, "@")
	case "hysteria", "hysteria2":
		// host:port?params or user@host:port
		return strings.Contains(body, ":")
	default:
		// 默认放行其他类型
		return true
	}
}

func (s *ConfigUpdateService) resolveRegion(name, server string) string {
	if s.regionMatcher != nil {
		return s.regionMatcher.MatchRegion(name, server)
	}
	return "未知"
}

// ==========================================
// 数据库存储与节点导入
// ==========================================

// deleteAutoImportedNodes 删除所有自动导入的节点（IsManual = false），保留手动添加的节点（IsManual = true）
func (s *ConfigUpdateService) deleteAutoImportedNodes() int64 {
	result := s.db.Where("is_manual = ?", false).Delete(&models.Node{})
	if result.Error != nil {
		s.log("ERROR", fmt.Sprintf("删除自动导入节点失败: %v", result.Error))
		return 0
	}
	return result.RowsAffected
}

type importStats struct {
	Created int // 新增节点数
	Updated int // 更新节点数
	Skipped int // 跳过节点数（与手动节点同名）
}

// generateNodeKey 生成节点的唯一键，与节点管理页面的去重逻辑保持一致
func (s *ConfigUpdateService) generateNodeKey(nodeType string, name string, config *string) string {
	if config == nil || *config == "" {
		return fmt.Sprintf("%s:%s", nodeType, name)
	}
	var p ProxyNode
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

func (s *ConfigUpdateService) importNodesToDatabaseWithOrder(nodesWithOrder []nodeWithOrder) importStats {
	stats := importStats{}
	seenKeys := make(map[string]bool) // 用于在本次导入中检测重复节点

	for _, item := range nodesWithOrder {
		node := item.node
		// #nosec G117 - Password field is proxy node password, not user credential
		configJSON, _ := json.Marshal(node) // #nosec G117
		configStr := string(configJSON)
		region := s.resolveRegion(node.Name, node.Server)

		// 使用与节点管理页面相同的去重逻辑：基于 config 生成唯一键
		nodeKey := s.generateNodeKey(node.Type, node.Name, &configStr)

		// 检查本次导入中是否已经有相同的节点（基于 config 的唯一键）
		if seenKeys[nodeKey] {
			stats.Skipped++
			s.log("DEBUG", fmt.Sprintf("跳过重复节点 %s（本次导入中已存在相同配置的节点）", node.Name))
			continue
		}
		seenKeys[nodeKey] = true

		// 查找数据库中是否已存在相同配置的节点（基于 config 的唯一键）
		// 注意：由于我们已经删除了所有自动导入的节点，这里主要是处理本次导入中的重复节点
		// 但为了安全起见，仍然检查数据库中是否有相同配置的节点
		var allNodes []models.Node
		s.db.Where("is_manual = ?", false).Find(&allNodes)

		var existingNode *models.Node
		for i := range allNodes {
			existingKey := s.generateNodeKey(allNodes[i].Type, allNodes[i].Name, allNodes[i].Config)
			if existingKey == nodeKey {
				existingNode = &allNodes[i]
				break
			}
		}

		if existingNode != nil {
			// 如果找到相同配置的节点，更新它
			existingNode.Config = &configStr
			existingNode.Status = "online"
			existingNode.IsActive = true
			existingNode.IsManual = false
			existingNode.OrderIndex = item.orderIndex
			existingNode.SourceIndex = item.sourceIndex
			existingNode.Region = region
			existingNode.Name = node.Name // 更新名称（可能不同）
			if s.db.Save(existingNode).Error == nil {
				stats.Updated++
			} else {
				s.log("ERROR", fmt.Sprintf("更新节点失败 (%s): %v", node.Name, s.db.Error))
			}
		} else {
			// 检查是否存在同名的手动节点（IsManual = true），如果存在则跳过
			var manualNode models.Node
			err := s.db.Where("type = ? AND name = ? AND is_manual = ?", node.Type, node.Name, true).First(&manualNode).Error
			if err == nil {
				stats.Skipped++
				s.log("DEBUG", fmt.Sprintf("跳过节点 %s（手动添加的节点，不覆盖）", node.Name))
				continue
			}

			// Create 新节点
			newNode := models.Node{
				Name:        node.Name,
				Type:        node.Type,
				Status:      "online",
				IsActive:    true,
				IsManual:    false,
				Config:      &configStr,
				Region:      region,
				OrderIndex:  item.orderIndex,
				SourceIndex: item.sourceIndex,
			}
			if s.db.Create(&newNode).Error == nil {
				stats.Created++
			} else {
				s.log("ERROR", fmt.Sprintf("创建节点失败 (%s): %v", node.Name, s.db.Error))
			}
		}
	}
	return stats
}

// ==========================================
// 订阅内容生成
// ==========================================

func (s *ConfigUpdateService) GetSubscriptionContext(token string, clientIP string, userAgent string) *SubscriptionContext {
	ctx := &SubscriptionContext{Status: StatusNotFound}
	var sub models.Subscription

	if err := s.db.Where("subscription_url = ?", token).First(&sub).Error; err != nil {
		// 检查是否为旧订阅
		var reset models.SubscriptionReset
		if s.db.Where("old_subscription_url = ?", token).First(&reset).Error == nil {
			ctx.Status = StatusOldAddress
			ctx.ResetRecord = &reset
		}
		return ctx
	}

	ctx.Subscription = sub
	var user models.User
	if err := s.db.First(&user, sub.UserID).Error; err != nil {
		return ctx
	}
	ctx.User = user

	// 状态检查
	if !user.IsActive {
		ctx.Status = StatusAccountAbnormal
		return ctx
	}
	if !sub.IsActive || sub.Status != "active" {
		ctx.Status = StatusInactive
		return ctx
	}

	// 检查过期
	if !sub.ExpireTime.IsZero() && sub.ExpireTime.Before(utils.GetBeijingTime()) {
		// 即便过期也可能需要显示节点（视业务逻辑而定），但这里标记为过期
		// 原始代码逻辑：如果没有节点且过期，才标记过期。如果有节点，会先获取节点。
		// 但 fetchProxiesForUser 内部也会判断过期。
	}

	proxies, _ := s.fetchProxiesForUser(user, sub)
	ctx.Proxies = proxies

	if len(ctx.Proxies) == 0 {
		if !sub.ExpireTime.IsZero() && sub.ExpireTime.Before(utils.GetBeijingTime()) {
			ctx.Status = StatusExpired
			return ctx
		}
	}

	// 设备限制检查
	var currentDevices int64
	s.db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", sub.ID, true).Count(&currentDevices)
	ctx.CurrentDevices = int(currentDevices)
	ctx.DeviceLimit = sub.DeviceLimit

	if sub.DeviceLimit == 0 {
		ctx.Status = StatusDeviceOverLimit
		return ctx
	}
	if sub.DeviceLimit > 0 && int(currentDevices) >= sub.DeviceLimit {
		// 检查当前设备是否在允许列表中
		var device models.Device
		if err := s.db.Where("subscription_id = ? AND ip_address = ? AND user_agent = ?", sub.ID, clientIP, userAgent).First(&device).Error; err != nil {
			ctx.Status = StatusDeviceOverLimit
			return ctx
		}
	}

	ctx.Status = StatusNormal
	return ctx
}

func (s *ConfigUpdateService) fetchProxiesForUser(user models.User, sub models.Subscription) ([]*ProxyNode, error) {
	proxies := make([]*ProxyNode, 0)
	processedNodes := make(map[string]bool)
	now := utils.GetBeijingTime()

	// 确定过期时间
	isSubExpired := !sub.ExpireTime.IsZero() && sub.ExpireTime.Before(now)
	isSpecialExpired := false
	if user.SpecialNodeExpiresAt.Valid {
		isSpecialExpired = utils.ToBeijingTime(user.SpecialNodeExpiresAt.Time).Before(now)
	} else if user.SpecialNodeSubscriptionType != "special_only" && isSubExpired {
		isSpecialExpired = true
	}

	// 1. 获取自定义节点
	s.appendCustomNodes(user.ID, now, isSpecialExpired, &proxies, processedNodes)

	// 2. 获取普通节点
	if user.SpecialNodeSubscriptionType != "special_only" && !isSubExpired {
		s.appendSystemNodes(&proxies, processedNodes)
	}

	return proxies, nil
}

func (s *ConfigUpdateService) appendCustomNodes(userID uint, now time.Time, isGlobalExpired bool, proxies *[]*ProxyNode, processed map[string]bool) {
	var customNodes []models.CustomNode
	s.db.Joins("JOIN user_custom_nodes ON user_custom_nodes.custom_node_id = custom_nodes.id").
		Where("user_custom_nodes.user_id = ? AND custom_nodes.is_active = ?", userID, true).
		Find(&customNodes)

	for _, cn := range customNodes {
		isNodeExpired := false
		if cn.ExpireTime != nil {
			isNodeExpired = utils.ToBeijingTime(*cn.ExpireTime).Before(now)
		} else if cn.FollowUserExpire {
			isNodeExpired = isGlobalExpired
		}

		if isNodeExpired || cn.Status == "timeout" {
			continue
		}

		// 专线节点配置应该直接存储为 ProxyNode JSON（与普通节点相同）
		// 这样可以保留所有解析的配置字段
		var proxyNode ProxyNode
		if err := json.Unmarshal([]byte(cn.Config), &proxyNode); err != nil {
			// 如果解析失败，记录错误并跳过
			s.log("ERROR", fmt.Sprintf("专线节点 %s 配置解析失败: %v", cn.Name, err))
			continue
		}

		// 设置节点名称
		proxyNode.Name = cn.DisplayName
		if proxyNode.Name == "" {
			proxyNode.Name = "专线-" + cn.Name
		}

		key := s.generateNodeDedupKey(proxyNode.Type, proxyNode.Server, proxyNode.Port)
		if !processed[key] {
			processed[key] = true
			*proxies = append(*proxies, &proxyNode)
		}
	}
}

func (s *ConfigUpdateService) appendSystemNodes(proxies *[]*ProxyNode, processed map[string]bool) {
	// 尝试从缓存获取系统节点
	cacheService := &CacheService{}
	if cachedNodes, ok := cacheService.GetSystemNodesCache(); ok {
		for _, node := range cachedNodes {
			key := s.generateNodeDedupKey(node.Type, node.Server, node.Port)
			if !processed[key] {
				processed[key] = true
				*proxies = append(*proxies, node)
			}
		}
		return
	}

	// 缓存未命中，从数据库查询
	var nodes []models.Node
	s.db.Model(&models.Node{}).Where("is_active = ? AND status != ?", true, "timeout").Find(&nodes)

	var systemNodes []*ProxyNode
	for _, node := range nodes {
		if node.Config == nil || *node.Config == "" {
			continue
		}
		var proxy ProxyNode
		if err := json.Unmarshal([]byte(*node.Config), &proxy); err == nil {
			proxy.Name = node.Name
			key := s.generateNodeDedupKey(proxy.Type, proxy.Server, proxy.Port)
			if !processed[key] {
				processed[key] = true
				*proxies = append(*proxies, &proxy)
				systemNodes = append(systemNodes, &proxy)
			}
		}
	}

	// 异步写入缓存
	if len(systemNodes) > 0 {
		go func(nodes []*ProxyNode) {
			if err := cacheService.SetSystemNodesCache(nodes); err != nil {
				log.Printf("failed to set system nodes cache: %v", err)
			}
		}(systemNodes)
	}
}

func (s *ConfigUpdateService) generateNodeDedupKey(nodeType, server string, port int) string {
	return fmt.Sprintf("%s:%s:%d", nodeType, server, port)
}

// calculateCacheTTL 智能计算缓存TTL
func (s *ConfigUpdateService) calculateCacheTTL(sub *models.Subscription, user *models.User) time.Duration {
	now := utils.GetBeijingTime()

	// 已到期，不缓存
	if !sub.ExpireTime.IsZero() && sub.ExpireTime.Before(now) {
		return 0
	}

	// 即将到期（24小时内），短缓存
	if !sub.ExpireTime.IsZero() {
		timeUntilExpire := sub.ExpireTime.Sub(now)
		if timeUntilExpire < 24*time.Hour {
			return 1 * time.Minute
		}
	}

	// 正常情况，长缓存
	return 10 * time.Minute
}

func (s *ConfigUpdateService) UpdateSubscriptionConfig(subscriptionURL string) error {
	var count int64
	s.db.Model(&models.Subscription{}).Where("subscription_url = ?", subscriptionURL).Count(&count)
	if count == 0 {
		return fmt.Errorf("订阅不存在")
	}
	return nil
}

func (s *ConfigUpdateService) GenerateClashConfig(token string, clientIP string, userAgent string) (string, error) {
	// 尝试从缓存获取
	cacheService := &CacheService{}
	if cached, ok := cacheService.GetSubscriptionConfigCache(token, "clash"); ok {
		return cached, nil
	}

	// 缓存未命中，生成配置
	nodes, err := s.prepareExportNodes(token, clientIP, userAgent)
	if err != nil {
		return "", err
	}
	ctx := s.GetSubscriptionContext(token, clientIP, userAgent)
	config := s.generateClashYAML(nodes, ctx)

	// 计算智能TTL并异步写入缓存
	if ctx.Status == StatusNormal {
		ttl := s.calculateCacheTTL(&ctx.Subscription, &ctx.User)
		if ttl > 0 {
			go func(token, cfg string, cacheTTL time.Duration) {
				if err := cacheService.SetSubscriptionConfigCache(token, "clash", cfg, cacheTTL); err != nil {
					log.Printf("failed to set subscription config cache: %v", err)
				}
			}(token, config, ttl)
		}
	}

	return config, nil
}

func (s *ConfigUpdateService) GenerateUniversalConfig(token string, clientIP string, userAgent string, format string) (string, error) {
	// 尝试从缓存获取
	cacheService := &CacheService{}
	cacheFormat := "base64"
	if format == "ssr" {
		cacheFormat = "ssr"
	}
	if cached, ok := cacheService.GetSubscriptionConfigCache(token, cacheFormat); ok {
		return cached, nil
	}

	// 缓存未命中，生成配置
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

	config := base64.StdEncoding.EncodeToString([]byte(strings.Join(links, "\n")))

	// 计算智能TTL并异步写入缓存
	ctx := s.GetSubscriptionContext(token, clientIP, userAgent)
	if ctx.Status == StatusNormal {
		ttl := s.calculateCacheTTL(&ctx.Subscription, &ctx.User)
		if ttl > 0 {
			go func(token, fmt, cfg string, cacheTTL time.Duration) {
				if err := cacheService.SetSubscriptionConfigCache(token, fmt, cfg, cacheTTL); err != nil {
					log.Printf("failed to set subscription config cache: %v", err)
				}
			}(token, cacheFormat, config, ttl)
		}
	}

	return config, nil
}

func (s *ConfigUpdateService) prepareExportNodes(token, clientIP, userAgent string) ([]*ProxyNode, error) {
	s.refreshSystemConfig()
	ctx := s.GetSubscriptionContext(token, clientIP, userAgent)

	if ctx.Status != StatusNormal {
		return s.generateErrorNodes(ctx.Status, ctx), nil
	}
	return s.addInfoNodes(ctx.Proxies, ctx), nil
}

// ==========================================
// Clash YAML 生成逻辑
// ==========================================

func (s *ConfigUpdateService) generateClashYAML(proxies []*ProxyNode, ctx *SubscriptionContext) string {
	filteredProxies := make([]*ProxyNode, 0)
	for _, proxy := range proxies {
		if supportedClashTypes[proxy.Type] {
			filteredProxies = append(filteredProxies, proxy)
		}
	}

	// 节点命名去重
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
		proxyNames = append(proxyNames, proxy.Name)
	}

	// 生成订阅名称（用于 Clash Verge、Flash 等所有 Clash 客户端显示）
	subscriptionName := s.GenerateSubscriptionName(ctx)

	// 尝试加载模板
	templatePath := filepath.Join("uploads", "config", "temp.yaml")
	// 清理并验证模板路径
	cleanTemplatePath := filepath.Clean(templatePath)
	if strings.Contains(cleanTemplatePath, "..") {
		return s.generateDefaultClashYAML(filteredProxies, proxyNames, subscriptionName)
	}

	templateData, err := os.ReadFile(cleanTemplatePath)
	if err != nil {
		return s.generateDefaultClashYAML(filteredProxies, proxyNames, subscriptionName)
	}

	var templateConfig map[string]interface{}
	if err := yaml.Unmarshal(templateData, &templateConfig); err != nil {
		return s.generateDefaultClashYAML(filteredProxies, proxyNames, subscriptionName)
	}

	// 设置订阅名称（Clash Verge、Flash 等所有 Clash 客户端会使用这个字段作为订阅显示名称）
	templateConfig["name"] = subscriptionName

	// 注入节点
	proxyList := make([]map[string]interface{}, 0)
	for _, proxy := range filteredProxies {
		proxyList = append(proxyList, s.nodeToMap(proxy))
	}
	templateConfig["proxies"] = proxyList

	// 更新 Proxy Groups
	if proxyGroups, ok := templateConfig["proxy-groups"].([]interface{}); ok {
		s.updateProxyGroups(proxyGroups, proxyNames)
		templateConfig["proxy-groups"] = proxyGroups
	}

	output, err := yaml.Marshal(templateConfig)
	if err != nil {
		return s.generateDefaultClashYAML(filteredProxies, proxyNames, subscriptionName)
	}

	return unescapeUnicode(string(output))
}

func (s *ConfigUpdateService) updateProxyGroups(groups []interface{}, proxyNames []string) {
	groupNames := make(map[string]bool)
	for _, g := range groups {
		if m, ok := g.(map[string]interface{}); ok {
			if name, ok := m["name"].(string); ok {
				groupNames[name] = true
			}
		}
	}

	for _, g := range groups {
		group, ok := g.(map[string]interface{})
		if !ok {
			continue
		}
		gType, _ := group["type"].(string)

		// 需要注入节点的组类型
		if gType == "select" || gType == "url-test" || gType == "fallback" || gType == "load-balance" {
			existingProxies := make([]string, 0)
			if oldProxies, ok := group["proxies"].([]interface{}); ok {
				for _, p := range oldProxies {
					if pStr, ok := p.(string); ok {
						// 保留特殊策略和组名
						if pStr == "DIRECT" || pStr == "REJECT" || groupNames[pStr] {
							existingProxies = append(existingProxies, pStr)
						}
					}
				}
			}

			if gType == "select" {
				group["proxies"] = append(existingProxies, proxyNames...)
			} else {
				// 自动测速类通常只包含具体节点
				group["proxies"] = proxyNames
			}
		}
	}
}

// GenerateSubscriptionName 生成订阅名称，用于 Clash Verge、Flash 等所有 Clash 客户端显示
func (s *ConfigUpdateService) GenerateSubscriptionName(ctx *SubscriptionContext) string {
	if ctx.Status != StatusNormal {
		switch ctx.Status {
		case StatusExpired:
			return "订阅已过期"
		case StatusInactive:
			return "订阅已失效"
		case StatusAccountAbnormal:
			return "账户异常"
		case StatusDeviceOverLimit:
			return "设备超限"
		default:
			return "订阅异常"
		}
	}

	expireTimeStr := "无限期"
	if !ctx.Subscription.ExpireTime.IsZero() {
		expireTimeStr = utils.FormatBeijingDate(ctx.Subscription.ExpireTime)
	}
	return fmt.Sprintf("到期: %s", expireTimeStr)
}

func (s *ConfigUpdateService) generateDefaultClashYAML(proxies []*ProxyNode, proxyNames []string, subscriptionName string) string {
	var builder strings.Builder

	// 添加订阅名称（Clash Verge、Flash 等所有 Clash 客户端会使用这个字段作为订阅显示名称）
	builder.WriteString(fmt.Sprintf("name: %s\n", s.escapeYAMLString(subscriptionName)))
	builder.WriteString("port: 7890\n")
	builder.WriteString("socks-port: 7891\n")
	builder.WriteString("allow-lan: true\n")
	builder.WriteString("mode: Rule\n")
	builder.WriteString("log-level: info\n")
	builder.WriteString("external-controller: 127.0.0.1:9090\n\n")

	builder.WriteString("proxies:\n")
	for _, proxy := range proxies {
		builder.WriteString(s.nodeToYAML(proxy, 2))
	}

	builder.WriteString("\nproxy-groups:\n")
	// 节点选择组
	builder.WriteString("  - name: \"🚀 节点选择\"\n")
	builder.WriteString("    type: select\n")
	builder.WriteString("    proxies:\n")
	builder.WriteString("      - \"♻️ 自动选择\"\n")
	for _, name := range proxyNames {
		builder.WriteString(fmt.Sprintf("      - %s\n", s.escapeYAMLString(name)))
	}

	// 自动选择组
	builder.WriteString("  - name: \"♻️ 自动选择\"\n")
	builder.WriteString("    type: url-test\n")
	builder.WriteString("    url: http://www.gstatic.com/generate_204\n")
	builder.WriteString("    interval: 300\n")
	builder.WriteString("    tolerance: 50\n")
	builder.WriteString("    proxies:\n")
	for _, name := range proxyNames {
		builder.WriteString(fmt.Sprintf("      - %s\n", s.escapeYAMLString(name)))
	}

	builder.WriteString("\nrules:\n")
	builder.WriteString("  - DOMAIN-SUFFIX,local,DIRECT\n")
	builder.WriteString("  - IP-CIDR,127.0.0.0/8,DIRECT\n")
	builder.WriteString("  - IP-CIDR,172.16.0.0/12,DIRECT\n")
	builder.WriteString("  - IP-CIDR,192.168.0.0/16,DIRECT\n")
	builder.WriteString("  - GEOIP,CN,DIRECT\n")
	builder.WriteString("  - MATCH,🚀 节点选择\n")

	return builder.String()
}

func (s *ConfigUpdateService) addInfoNodes(proxies []*ProxyNode, ctx *SubscriptionContext) []*ProxyNode {
	expireTimeStr := "无限期"
	if !ctx.Subscription.ExpireTime.IsZero() {
		expireTimeStr = utils.FormatBeijingDate(ctx.Subscription.ExpireTime)
	}

	infoNodes := []*ProxyNode{
		s.createMessageNode(fmt.Sprintf("📢 官网: %s", s.siteURL)),
		s.createMessageNode(fmt.Sprintf("⏰ 到期: %s", expireTimeStr)),
		s.createMessageNode(fmt.Sprintf("📱 设备: %d/%d", ctx.CurrentDevices, ctx.DeviceLimit)),
	}

	if s.supportQQ != "" {
		infoNodes = append(infoNodes, s.createMessageNode(fmt.Sprintf("💬 客服: %s", s.supportQQ)))
	}

	return append(infoNodes, proxies...)
}

func (s *ConfigUpdateService) generateErrorNodes(status SubscriptionStatus, ctx *SubscriptionContext) []*ProxyNode {
	var reason, solution string

	switch status {
	case StatusExpired:
		reason = "订阅已过期"
		solution = fmt.Sprintf("请前往官网续费 (过期时间: %s)", utils.FormatBeijingDate(ctx.Subscription.ExpireTime))
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

	qqMsg := "💬 客服: 请在系统设置中配置"
	if s.supportQQ != "" {
		qqMsg = fmt.Sprintf("💬 客服: %s", s.supportQQ)
	}
	infoNodes = append(infoNodes, s.createMessageNode(qqMsg, "error"))

	return infoNodes
}

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

// ==========================================
// 节点对象转 YAML/Map
// ==========================================

func (s *ConfigUpdateService) nodeToYAML(node *ProxyNode, indent int) string {
	indentStr := strings.Repeat(" ", indent)
	var builder strings.Builder

	// 基础字段
	builder.WriteString(fmt.Sprintf("%s- name: %s\n", indentStr, s.escapeYAMLString(node.Name)))
	builder.WriteString(fmt.Sprintf("%s  type: %s\n", indentStr, node.Type))
	builder.WriteString(fmt.Sprintf("%s  server: %s\n", indentStr, node.Server))
	builder.WriteString(fmt.Sprintf("%s  port: %d\n", indentStr, node.Port))

	// 特有字段映射
	m := s.nodeToMap(node)
	keys := make([]string, 0, len(m))
	for k := range m {
		// 跳过已处理的基础字段
		if k == "name" || k == "type" || k == "server" || k == "port" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		s.writeYAMLValue(&builder, indentStr+"  ", k, m[k], indent+2)
	}

	return builder.String()
}

func (s *ConfigUpdateService) nodeToMap(node *ProxyNode) map[string]interface{} {
	result := make(map[string]interface{})
	result["name"] = node.Name
	result["type"] = node.Type
	result["server"] = node.Server
	result["port"] = node.Port

	// 通用字段处理 helper
	setIfNotEmpty := func(key, val string) {
		if val != "" {
			result[key] = val
		}
	}

	switch node.Type {
	case "ss":
		setIfNotEmpty("cipher", node.Cipher)
		// SS 协议必须要有 password 字段
		if node.Password != "" {
			result["password"] = node.Password
		} else {
			result["password"] = "" // 即使为空也要设置
		}
		// 处理 plugin 配置
		if pluginName, ok := node.Options["plugin"].(string); ok && pluginName != "" {
			result["plugin"] = pluginName
			if pluginOpts, ok := node.Options["plugin-opts"].(map[string]interface{}); ok {
				result["plugin-opts"] = pluginOpts
			}
		}
	case "vmess":
		setIfNotEmpty("uuid", node.UUID)
		result["alterId"] = 0
		if val, ok := node.Options["alterId"]; ok {
			result["alterId"] = val
		}
		result["cipher"] = "auto"
		if node.Cipher != "" {
			result["cipher"] = node.Cipher
		}
	case "vless":
		setIfNotEmpty("uuid", node.UUID)
	case "trojan":
		// Trojan 协议必须要有 password 字段
		if node.Password != "" {
			result["password"] = node.Password
		} else {
			result["password"] = "" // 即使为空也要设置
		}
	case "ssr":
		setIfNotEmpty("cipher", node.Cipher)
		// SSR 协议必须要有 password 字段
		if node.Password != "" {
			result["password"] = node.Password
		} else {
			result["password"] = "" // 即使为空也要设置
		}
	case "tuic":
		setIfNotEmpty("uuid", node.UUID)
		// TUIC 协议必须要有 password 字段
		if node.Password != "" {
			result["password"] = node.Password
		} else {
			result["password"] = "" // 即使为空也要设置
		}
		// TUIC 特殊字段处理
		if node.TLS {
			result["tls"] = true
		}
		// 设置默认值
		if _, ok := node.Options["disable-sni"]; !ok {
			result["disable-sni"] = false
		}
		if _, ok := node.Options["reduce-rtt"]; !ok {
			result["reduce-rtt"] = false
		}
		if _, ok := node.Options["request-timeout"]; !ok {
			result["request-timeout"] = 15000
		}
		if _, ok := node.Options["udp-relay-mode"]; !ok {
			result["udp-relay-mode"] = "native"
		}
		// 字段名映射：congestion_control → congestion-controller
		if cc, ok := node.Options["congestion_control"].(string); ok && cc != "" {
			result["congestion-controller"] = cc
			delete(node.Options, "congestion_control") // 避免重复
		} else if cc, ok := node.Options["congestion-controller"].(string); ok && cc != "" {
			result["congestion-controller"] = cc
		}
		// sni 字段映射
		if sni, ok := node.Options["servername"].(string); ok && sni != "" {
			result["sni"] = sni
			delete(node.Options, "servername") // TUIC 使用 sni 而不是 servername
		}
	case "anytls":
		// Anytls 协议没有 UUID 字段，只有 password
		// Anytls 协议必须要有 password 字段
		if node.Password != "" {
			result["password"] = node.Password
		} else {
			result["password"] = "" // 即使为空也要设置
		}
		// Anytls 需要 TLS 和 sni
		if node.TLS {
			result["tls"] = true
		}
		// sni 字段映射
		if sni, ok := node.Options["servername"].(string); ok && sni != "" {
			result["sni"] = sni
			delete(node.Options, "servername") // Anytls 使用 sni 而不是 servername
		}
		// udp 字段：Anytls 通常设置为 false
		result["udp"] = false
		if node.UDP {
			result["udp"] = true
		}
	case "hysteria", "hysteria2":
		// Hysteria 和 Hysteria2 协议必须要有 password 字段（auth）
		if node.Password != "" {
			result["password"] = node.Password
			// Hysteria2 的 auth 和 password 是同一个值
			if node.Type == "hysteria2" {
				result["auth"] = node.Password
			}
		} else if auth, ok := node.Options["auth"].(string); ok && auth != "" {
			result["password"] = auth
			if node.Type == "hysteria2" {
				result["auth"] = auth
			}
		} else {
			result["password"] = "" // 即使为空也要设置
		}
	case "socks", "socks5", "http":
		setIfNotEmpty("username", node.UUID) // 借用 UUID 字段存 user
		// SOCKS 和 HTTP 协议必须要有 password 字段
		if node.Password != "" {
			result["password"] = node.Password
		} else {
			result["password"] = "" // 即使为空也要设置
		}
	case "wireguard":
		// WireGuard 协议特殊处理
		// 必需字段已经在 Options 中，这里不需要额外处理
		// public-key, private-key, ip 等都会通过 Options 复制到 result
	}

	if node.TLS {
		result["tls"] = true
	}
	if node.Network != "" && node.Network != "tcp" {
		result["network"] = node.Network
	}
	if node.UDP {
		result["udp"] = true
	}

	for key, value := range node.Options {
		if key == "alterId" && node.Type == "vmess" {
			continue
		}
		result[key] = value
	}

	return result
}

func (s *ConfigUpdateService) writeYAMLValue(builder *strings.Builder, indentStr, key string, value interface{}, indentLevel int) {
	escapedKey := s.escapeYAMLString(key)

	switch v := value.(type) {
	case map[string]interface{}:
		builder.WriteString(fmt.Sprintf("%s%s:\n", indentStr, escapedKey))
		s.writeMapContent(builder, indentStr+"  ", v, key, indentLevel+1)
	case []interface{}:
		builder.WriteString(fmt.Sprintf("%s%s:\n", indentStr, escapedKey))
		subIndent := indentStr + "  "
		for _, item := range v {
			builder.WriteString(fmt.Sprintf("%s- ", subIndent))
			s.writeYAMLValueInline(builder, item)
			builder.WriteString("\n")
		}
	case []string:
		builder.WriteString(fmt.Sprintf("%s%s:\n", indentStr, escapedKey))
		for _, item := range v {
			builder.WriteString(fmt.Sprintf("%s  - %s\n", indentStr, s.escapeYAMLString(item)))
		}
	default:
		builder.WriteString(fmt.Sprintf("%s%s: %s\n", indentStr, escapedKey, s.formatYAMLInline(v)))
	}
}

func (s *ConfigUpdateService) writeMapContent(builder *strings.Builder, indentStr string, v map[string]interface{}, parentKey string, level int) {
	if parentKey == "http-opts" {
		// 特殊处理 http-opts
		if path, ok := v["path"]; ok {
			s.writeYAMLList(builder, indentStr, "path", path)
		}
		if headers, ok := v["headers"].(map[string]interface{}); ok {
			builder.WriteString(fmt.Sprintf("%sheaders:\n", indentStr))
			for hk, hv := range headers {
				s.writeYAMLList(builder, indentStr+"  ", hk, hv)
			}
		}
		return
	}

	// 通用 Map 递归
	// 为了稳定排序
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		val := v[k]
		if strMap, ok := val.(map[string]string); ok {
			// 转换 map[string]string 到 map[string]interface{} 以复用逻辑
			newMap := make(map[string]interface{})
			for mk, mv := range strMap {
				newMap[mk] = mv
			}
			s.writeYAMLValue(builder, indentStr, k, newMap, level+1)
		} else {
			s.writeYAMLValue(builder, indentStr, k, val, level+1)
		}
	}
}

func (s *ConfigUpdateService) writeYAMLList(builder *strings.Builder, indentStr, key string, val interface{}) {
	builder.WriteString(fmt.Sprintf("%s%s:\n", indentStr, s.escapeYAMLString(key)))
	subIndent := indentStr + "  "

	writeItem := func(item interface{}) {
		builder.WriteString(fmt.Sprintf("%s- %s\n", subIndent, s.formatYAMLInline(item)))
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

func (s *ConfigUpdateService) formatYAMLInline(v interface{}) string {
	switch val := v.(type) {
	case string:
		return s.escapeYAMLString(val)
	case int, int64, float64, bool:
		return fmt.Sprintf("%v", val)
	default:
		return s.escapeYAMLString(fmt.Sprintf("%v", val))
	}
}

func (s *ConfigUpdateService) writeYAMLValueInline(builder *strings.Builder, v interface{}) {
	builder.WriteString(s.formatYAMLInline(v))
}

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
// 链接生成 (ToLink)
// ==========================================

func (s *ConfigUpdateService) NodeToLink(node *ProxyNode) string {
	return s.nodeToLink(node)
}

func (s *ConfigUpdateService) nodeToLink(node *ProxyNode) string {
	switch node.Type {
	case "vmess":
		return s.vmessToLink(node)
	case "ss":
		return s.shadowsocksToLink(node)
	case "ssr":
		return s.nodeToSSRLink(node)
	// 以下类型均符合标准URL格式，使用统一函数处理
	case "vless":
		return s.buildStandardNodeURL("vless", node.UUID, "", node.Server, node.Port, node.Name, s.getQueryFromOptions(node))
	case "trojan":
		return s.buildStandardNodeURL("trojan", node.Password, "", node.Server, node.Port, node.Name, s.getQueryFromOptions(node))
	case "hysteria":
		return s.buildStandardNodeURL("hysteria", "", "", node.Server, node.Port, node.Name, s.getQueryFromOptions(node))
	case "hysteria2":
		return s.buildStandardNodeURL("hysteria2", node.Password, "", node.Server, node.Port, node.Name, s.getQueryFromOptions(node))
	case "tuic":
		return s.buildStandardNodeURL("tuic", node.UUID, node.Password, node.Server, node.Port, node.Name, s.getQueryFromOptions(node))
	case "naive":
		return s.buildStandardNodeURL("naive+https", node.UUID, node.Password, node.Server, node.Port, node.Name, s.getQueryFromOptions(node))
	case "anytls":
		return s.buildStandardNodeURL("anytls", node.Password, "", node.Server, node.Port, node.Name, s.getQueryFromOptions(node))
	case "socks", "socks5":
		scheme := "socks5"
		if node.Type == "socks" {
			scheme = "socks"
		}
		return s.buildStandardNodeURL(scheme, node.UUID, node.Password, node.Server, node.Port, node.Name, nil)
	case "http":
		scheme := "http"
		if node.TLS {
			scheme = "https"
		}
		return s.buildStandardNodeURL(scheme, node.UUID, node.Password, node.Server, node.Port, node.Name, s.getQueryFromOptions(node))
	default:
		return ""
	}
}

// 通用链接构建函数
func (s *ConfigUpdateService) buildStandardNodeURL(scheme, user, password, host string, port int, fragment string, query url.Values) string {
	u := &url.URL{
		Scheme:   scheme,
		Host:     fmt.Sprintf("%s:%d", host, port),
		Fragment: fragment,
	}

	if user != "" {
		if password != "" {
			u.User = url.UserPassword(user, password)
		} else {
			u.User = url.User(user)
		}
	} else if password != "" {
		u.User = url.User(password)
	}

	if query != nil && len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	return u.String()
}

// 提取通用 Query 参数
func (s *ConfigUpdateService) getQueryFromOptions(node *ProxyNode) url.Values {
	q := url.Values{}
	if node.Options == nil {
		return q
	}

	// 辅助函数：简化获取 string/bool 选项
	optStr := func(k string) string {
		if v, ok := node.Options[k].(string); ok {
			return v
		}
		return ""
	}
	optBool := func(k string) bool {
		if v, ok := node.Options[k].(bool); ok {
			return v
		}
		return false
	}

	// 1. 通用 TLS 参数
	if sni := optStr("servername"); sni != "" {
		q.Set("sni", sni)
	}
	if peer := optStr("peer"); peer != "" {
		if q.Get("sni") == "" {
			q.Set("peer", peer)
		} else if node.Type == "anytls" {
			q.Set("peer", peer)
		}
	}
	if optBool("skip-cert-verify") {
		q.Set("insecure", "1")
		q.Set("allow_insecure", "1")
	}

	// client-fingerprint (fp)
	if fp := optStr("client-fingerprint"); fp != "" {
		q.Set("fp", fp)
	}

	// ALPN 处理
	if alpnVal, ok := node.Options["alpn"]; ok {
		var alpnStr string
		if strs, ok := alpnVal.([]string); ok && len(strs) > 0 {
			alpnStr = strings.Join(strs, ",")
		} else if infs, ok := alpnVal.([]interface{}); ok {
			var tmp []string
			for _, v := range infs {
				if s, ok := v.(string); ok {
					tmp = append(tmp, s)
				}
			}
			if len(tmp) > 0 {
				alpnStr = strings.Join(tmp, ",")
			}
		}
		if alpnStr != "" {
			q.Set("alpn", alpnStr)
		}
	}

	// 2. 传输层参数（vless/trojan 等）
	if node.Type == "vless" || node.Type == "trojan" {
		if node.Network != "" {
			q.Set("type", node.Network)
		}

		// ws-opts
		if wsOpts, ok := node.Options["ws-opts"].(map[string]interface{}); ok {
			if path, ok := wsOpts["path"].(string); ok && path != "" {
				q.Set("path", path)
			}
			if headers, ok := wsOpts["headers"].(map[string]interface{}); ok {
				if host, ok := headers["Host"].(string); ok && host != "" {
					q.Set("host", host)
				}
			}
			// map[string]string 类型的 headers
			if headers, ok := wsOpts["headers"].(map[string]string); ok {
				if host, ok := headers["Host"]; ok && host != "" {
					q.Set("host", host)
				}
			}
		}
		// grpc-opts
		if grpcOpts, ok := node.Options["grpc-opts"].(map[string]interface{}); ok {
			if sn, ok := grpcOpts["grpc-service-name"].(string); ok && sn != "" {
				q.Set("serviceName", sn)
			}
		}
		// h2-opts
		if h2Opts, ok := node.Options["h2-opts"].(map[string]interface{}); ok {
			if path, ok := h2Opts["path"].(string); ok && path != "" {
				q.Set("path", path)
			}
			if hosts, ok := h2Opts["host"].([]string); ok && len(hosts) > 0 {
				q.Set("host", hosts[0])
			}
		}
		// tcp header-type
		if ht := optStr("header-type"); ht != "" {
			q.Set("headerType", ht)
		}
	}

	// 3. VLESS 特有参数
	if node.Type == "vless" {
		if node.TLS {
			// 检查是否是 reality
			if realityOpts, ok := node.Options["reality-opts"].(map[string]interface{}); ok {
				q.Set("security", "reality")
				if pbk, ok := realityOpts["public-key"].(string); ok && pbk != "" {
					q.Set("pbk", pbk)
				}
				if sid, ok := realityOpts["short-id"].(string); ok && sid != "" {
					q.Set("sid", sid)
				}
				if pqv, ok := realityOpts["pqv"].(string); ok && pqv != "" {
					q.Set("pqv", pqv)
				}
			} else {
				q.Set("security", "tls")
			}
		}
		if flow := optStr("flow"); flow != "" {
			q.Set("flow", flow)
		}
		if enc := optStr("encryption"); enc != "" {
			q.Set("encryption", enc)
		}
	}

	// 4. Hysteria / Hysteria2 参数
	if node.Type == "hysteria" || node.Type == "hysteria2" {
		if auth := optStr("auth"); auth != "" {
			q.Set("auth", auth)
		}
		if up := optStr("up"); up != "" {
			trimmed := strings.TrimSuffix(up, " mbps")
			q.Set("upmbps", trimmed)
			q.Set("mbpsUp", trimmed)
		}
		if down := optStr("down"); down != "" {
			trimmed := strings.TrimSuffix(down, " mbps")
			q.Set("downmbps", trimmed)
			q.Set("mbpsDown", trimmed)
		}
	}

	// 5. TUIC 参数
	if node.Type == "tuic" {
		if cc := optStr("congestion_control"); cc != "" {
			q.Set("congestion_control", cc)
		}
		if mode := optStr("udp_relay_mode"); mode != "" {
			q.Set("udp_relay_mode", mode)
		}
	}

	// 6. Naive 参数
	if node.Type == "naive" {
		if optBool("padding") {
			q.Set("padding", "true")
		}
	}

	return q
}

func (s *ConfigUpdateService) vmessToLink(proxy *ProxyNode) string {
	network := proxy.Network
	obfsType := "none"

	// http 网络类型需要转回 tcp + http 伪装
	if network == "http" {
		network = "tcp"
		obfsType = "http"
	}

	data := map[string]interface{}{
		"v":    "2",
		"ps":   proxy.Name,
		"add":  proxy.Server,
		"port": proxy.Port,
		"id":   proxy.UUID,
		"net":  network,
		"type": obfsType,
		"tls":  "",
		"sni":  "",
		"host": "",
		"path": "",
		"aid":  0,
		"scy":  "auto",
	}

	if proxy.TLS {
		data["tls"] = "tls"
	}

	if proxy.Options != nil {
		// alterId
		if aid, ok := proxy.Options["alterId"]; ok {
			data["aid"] = aid
		}
		// cipher
		if cipher, ok := proxy.Options["cipher"].(string); ok && cipher != "" {
			data["scy"] = cipher
		}
		// sni / servername
		if sni, ok := proxy.Options["servername"].(string); ok && sni != "" {
			data["sni"] = sni
		}
		// skip-cert-verify (insecure)
		if skip, ok := proxy.Options["skip-cert-verify"].(bool); ok && skip {
			data["insecure"] = "1"
		}

		// ws-opts
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
		// http-opts (TCP+HTTP 伪装)
		if httpOpts, ok := proxy.Options["http-opts"].(map[string]interface{}); ok {
			if paths, ok := httpOpts["path"].([]string); ok && len(paths) > 0 {
				data["path"] = paths[0]
			}
			if headers, ok := httpOpts["headers"].(map[string]interface{}); ok {
				if hosts, ok := headers["Host"].([]string); ok && len(hosts) > 0 {
					data["host"] = hosts[0]
				}
			}
		}
		// h2-opts
		if h2Opts, ok := proxy.Options["h2-opts"].(map[string]interface{}); ok {
			if path, ok := h2Opts["path"].(string); ok {
				data["path"] = path
			}
			if hosts, ok := h2Opts["host"].([]string); ok && len(hosts) > 0 {
				data["host"] = hosts[0]
			}
		}
		// grpc-opts
		if grpcOpts, ok := proxy.Options["grpc-opts"].(map[string]interface{}); ok {
			if sn, ok := grpcOpts["grpc-service-name"].(string); ok {
				data["path"] = sn
			}
		}
	}

	jsonData, _ := json.Marshal(data)
	encoded := base64.StdEncoding.EncodeToString(jsonData)
	return "vmess://" + encoded
}

func (s *ConfigUpdateService) shadowsocksToLink(proxy *ProxyNode) string {
	auth := fmt.Sprintf("%s:%s", proxy.Cipher, proxy.Password)
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))

	// 构建 plugin 查询参数
	var query url.Values
	if pluginName, ok := proxy.Options["plugin"].(string); ok && pluginName != "" {
		query = url.Values{}
		// 还原插件名称
		linkPluginName := pluginName
		switch pluginName {
		case "obfs":
			linkPluginName = "obfs-local"
		}
		pluginStr := linkPluginName
		if pluginOpts, ok := proxy.Options["plugin-opts"].(map[string]interface{}); ok {
			if mode, ok := pluginOpts["mode"].(string); ok && mode != "" {
				pluginStr += ";obfs=" + mode
			}
			if host, ok := pluginOpts["host"].(string); ok && host != "" {
				pluginStr += ";obfs-host=" + host
			}
			if path, ok := pluginOpts["path"].(string); ok && path != "" {
				pluginStr += ";obfs-uri=" + path
			}
			if tls, ok := pluginOpts["tls"].(bool); ok && tls {
				pluginStr += ";tls"
			}
		}
		query.Set("plugin", pluginStr)
	}

	return s.buildStandardNodeURL("ss", encoded, "", proxy.Server, proxy.Port, proxy.Name, query)
}

func (s *ConfigUpdateService) nodeToSSRLink(node *ProxyNode) string {
	getString := func(key, def string) string {
		if v, ok := node.Options[key].(string); ok {
			return v
		}
		return def
	}

	password := base64.RawURLEncoding.EncodeToString([]byte(node.Password))
	obfsparam := base64.RawURLEncoding.EncodeToString([]byte(getString("obfs-param", "")))
	protoparam := base64.RawURLEncoding.EncodeToString([]byte(getString("protocol-param", "")))
	remarks := base64.RawURLEncoding.EncodeToString([]byte(node.Name))
	group := base64.RawURLEncoding.EncodeToString([]byte("GoWeb"))

	ssrStr := fmt.Sprintf("%s:%d:%s:%s:%s:%s/?obfsparam=%s&protoparam=%s&remarks=%s&group=%s",
		node.Server, node.Port,
		getString("protocol", "origin"),
		node.Cipher,
		getString("obfs", "plain"),
		password,
		obfsparam, protoparam, remarks, group)

	return "ssr://" + base64.RawURLEncoding.EncodeToString([]byte(ssrStr))
}

func unescapeUnicode(s string) string {
	re := regexp.MustCompile(`\\U([0-9A-Fa-f]{8})`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		hexStr := match[2:]
		codePoint, err := strconv.ParseInt(hexStr, 16, 64)
		if err != nil {
			return match
		}
		return string(utils.MustSafeInt64ToRune(codePoint))
	})
}

// ==========================================
// 格式化日志辅助函数
// ==========================================

// logSeparator 输出分隔线
func (s *ConfigUpdateService) logSeparator() {
	s.log("INFO", "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// logSection 输出章节标题
func (s *ConfigUpdateService) logSection(icon, title string) {
	s.logSeparator()
	s.log("INFO", fmt.Sprintf("%s %s", icon, title))
	s.logSeparator()
}

// logItem 输出带缩进的项目
func (s *ConfigUpdateService) logItem(prefix, content string) {
	s.log("INFO", fmt.Sprintf("           %s %s", prefix, content))
}

// formatBytes 格式化字节大小
func formatBytes(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
}

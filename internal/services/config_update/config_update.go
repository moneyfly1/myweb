package config_update

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

	"cboard-go/internal/services/cache_service"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

// ==========================================
// 常量与类型定义
// ==========================================

type SubscriptionStatus int

const (
	StatusNormal SubscriptionStatus = iota
	StatusExpired
	StatusInactive
	StatusAccountAbnormal
	StatusDeviceOverLimit
	StatusOldAddress
	StatusNotFound
	StatusSystemError
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
	"vmess": true, "vless": true, "trojan": true, "ss": true, "ssr": true,
	"hysteria": true, "hysteria2": true, "tuic": true, "anytls": true,
	"socks": true, "socks5": true, "http": true, "wireguard": true, "direct": true,
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
	sseManager    *SSEManager
	logBuffer     []map[string]interface{}
	logMutex      sync.RWMutex
}

type nodeWithOrder struct {
	node        *ProxyNode
	orderIndex  int
	sourceIndex int
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
	if err != nil && utils.AppLogger != nil {
		utils.AppLogger.Warn("地区配置加载失败: %v，将使用空配置", err)
	}

	if regionConfig != nil && (len(regionConfig.RegionMap) > 0 || len(regionConfig.ServerMap) > 0) {
		service.regionMatcher = NewRegionMatcher(regionConfig.RegionMap, regionConfig.ServerMap)
		if utils.AppLogger != nil {
			utils.AppLogger.Info("地区配置加载成功: region_map=%d, server_map=%d", len(regionConfig.RegionMap), len(regionConfig.ServerMap))
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

func (s *ConfigUpdateService) loadLegacyRegionMaps() {}

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
	s.logMutex.RLock()
	if len(s.logBuffer) > 0 {
		logs := make([]map[string]interface{}, len(s.logBuffer))
		copy(logs, s.logBuffer)
		s.logMutex.RUnlock()
		return s.limitLogs(logs, limit)
	}
	s.logMutex.RUnlock()

	return s.getLogsFromDB(limit)
}

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

func (s *ConfigUpdateService) limitLogs(logs []map[string]interface{}, limit int) []map[string]interface{} {
	if len(logs) > limit {
		return logs[len(logs)-limit:]
	}
	return logs
}

func (s *ConfigUpdateService) ClearLogs() error {
	s.clearLogBuffer()
	if s.sseManager != nil {
		s.sseManager.ClearHistory()
	}
	return s.clearLogsInDB()
}

func (s *ConfigUpdateService) clearLogBuffer() {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()
	s.logBuffer = make([]map[string]interface{}, 0, 500)
}

func (s *ConfigUpdateService) clearLogsInDB() error {
	var config models.SystemConfig
	err := s.db.Where("key = ?", "config_update_logs").First(&config).Error
	if err != nil {
		return s.saveLogConfig("[]")
	}
	config.Value = "[]"
	return s.db.Save(&config).Error
}

func (s *ConfigUpdateService) GetSSEManager() *SSEManager {
	return s.sseManager
}

func (s *ConfigUpdateService) log(level, message string) {
	logEntry := s.createLogEntry(level, message)
	s.addLogToBuffer(logEntry)

	if s.sseManager != nil {
		s.sseManager.Broadcast(logEntry)
	}

	if utils.AppLogger != nil {
		if level == "ERROR" {
			utils.AppLogger.Error("%s", message)
		} else {
			utils.AppLogger.Info("%s", message)
		}
	}
}

func (s *ConfigUpdateService) createLogEntry(level, message string) map[string]interface{} {
	now := utils.FormatBeijingTime(utils.GetBeijingTime())
	return map[string]interface{}{
		"timestamp": now, "time": now, "level": level, "message": message,
	}
}

func (s *ConfigUpdateService) addLogToBuffer(logEntry map[string]interface{}) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.logBuffer = append(s.logBuffer, logEntry)
	if len(s.logBuffer) > 500 {
		s.logBuffer = s.logBuffer[len(s.logBuffer)-500:]
	}
}

func (s *ConfigUpdateService) flushLogsToDB() error {
	s.logMutex.RLock()
	if len(s.logBuffer) == 0 {
		s.logMutex.RUnlock()
		return nil
	}
	logs := make([]map[string]interface{}, len(s.logBuffer))
	copy(logs, s.logBuffer)
	s.logMutex.RUnlock()

	logsJSON, err := json.Marshal(logs)
	if err != nil {
		return err
	}

	var config models.SystemConfig
	if err = s.db.Where("key = ?", "config_update_logs").First(&config).Error; err != nil {
		return s.saveLogConfig(string(logsJSON))
	}

	config.Value = string(logsJSON)
	return s.db.Save(&config).Error
}

func (s *ConfigUpdateService) saveLogConfig(value string) error {
	return s.db.Create(&models.SystemConfig{
		Key: "config_update_logs", Value: value, Type: "json",
		Category: "config_update", DisplayName: "配置更新日志", Description: "配置更新任务日志",
	}).Error
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
		s.log("ERROR", "未配置节点源URL")
		return fmt.Errorf("未配置节点源URL")
	}

	s.log("INFO", "📋 配置信息")
	s.logItem("└─", fmt.Sprintf("节点源数量: %d 个", len(urls)))

	nodes, err := s.FetchNodesFromURLs(urls)
	if err != nil {
		s.log("ERROR", fmt.Sprintf("获取节点失败: %v", err))
		return err
	}

	if len(nodes) == 0 {
		s.log("WARN", "未获取到有效节点")
		return fmt.Errorf("未获取到有效节点")
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

	deletedCount := s.deleteAutoImportedNodes()
	s.logSection("💾", "数据库操作")
	s.log("INFO", fmt.Sprintf("🗑️  删除旧节点: %d 个", deletedCount))
	s.log("INFO", "⠼ 正在导入节点到数据库...")

	importStats := s.importNodesToDatabaseWithOrder(nodesWithOrder)
	s.updateLastUpdateTime()

	s.log("INFO", fmt.Sprintf("➕ 新增节点: %d 个", importStats.Created))
	s.log("INFO", fmt.Sprintf("🔄 更新节点: %d 个", importStats.Updated))
	s.log("INFO", fmt.Sprintf("⏭️  跳过节点: %d 个 (手动添加)", importStats.Skipped))

	time.Sleep(100 * time.Millisecond)
	s.logSection("✅", "任务完成")
	s.logItem("└─", fmt.Sprintf("最终结果: 成功导入 %d 个节点", importStats.Created))

	s.clearAllCaches()

	s.log("INFO", "💾 保存日志到数据库...")
	if err := s.flushLogsToDB(); err != nil {
		s.log("ERROR", fmt.Sprintf("保存日志失败: %v", err))
	} else {
		s.log("INFO", "✓ 日志已保存")
	}

	time.Sleep(100 * time.Millisecond)
	return nil
}

func (s *ConfigUpdateService) clearAllCaches() {
	cs := cache_service.NewCacheService()
	_ = cs.ClearNodesCache()
	_ = (&CacheService{}).ClearSystemNodesCache()
	_ = (&CacheService{}).ClearAllSubscriptionCache()

	s.logSection("🧹", "清理缓存")
	s.log("INFO", "✓ 节点列表缓存已清除")
	s.log("INFO", "✓ 系统节点缓存已清除")
	s.log("INFO", "✓ 订阅配置缓存已清除")
}

func (s *ConfigUpdateService) extractFilterKeywords(config map[string]interface{}) []string {
	if keywords, ok := config["filter_keywords"].([]string); ok {
		return keywords
	}
	if keywordsStr, ok := config["filter_keywords"].(string); ok && keywordsStr != "" {
		return s.splitAndTrim(keywordsStr)
	}
	return nil
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
		if sourceURL, _ := nodeInfo["source_url"].(string); sourceURL != "" {
			nodesByURL[sourceURL] = append(nodesByURL[sourceURL], nodeInfo)
		} else {
			stats.missingSource++
		}
	}

	for urlIndex, url := range urls {
		urlNodes := nodesByURL[url]
		if len(urlNodes) == 0 {
			continue
		}

		s.log("INFO", fmt.Sprintf("⠸ 开始处理订阅地址 [%d/%d] 的节点，共 %d 个链接", urlIndex+1, len(urls), len(urlNodes)))

		var links []string
		for _, nodeInfo := range urlNodes {
			if link, ok := nodeInfo["url"].(string); ok {
				links = append(links, link)
			} else {
				stats.invalidLinks++
			}
		}

		if len(links) == 0 {
			continue
		}

		results := s.parserPool.ParseLinks(links)
		nodeIndexInURL := 0
		counts := struct{ Processed, Failed, Filtered, Duplicate int }{}

		for _, result := range results {
			if seenKeys[result.Link] {
				stats.duplicates++
				counts.Duplicate++
				continue
			}
			seenKeys[result.Link] = true

			if result.Err != nil || result.Node == nil {
				stats.parseFailed++
				counts.Failed++
				if counts.Failed <= 10 && result.Err != nil {
					s.log("WARN", fmt.Sprintf("解析失败 [订阅地址 %d/%d]: %v, 链接: %s", urlIndex+1, len(urls), result.Err, truncateString(result.Link, 50)))
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
	nameLower, serverLower := strings.ToLower(node.Name), strings.ToLower(node.Server)
	for _, kw := range keywords {
		if kwLower := strings.ToLower(strings.TrimSpace(kw)); kwLower != "" && (strings.Contains(nameLower, kwLower) || strings.Contains(serverLower, kwLower)) {
			return true, kw
		}
	}
	return false, ""
}

func (s *ConfigUpdateService) ensureUniqueName(name string, usedNames map[string]bool) string {
	if !usedNames[name] {
		return name
	}
	for counter := 1; ; counter++ {
		if newName := fmt.Sprintf("%s-%d", name, counter); !usedNames[newName] {
			return newName
		}
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
	if err := s.db.Where("category = ?", "config_update").Find(&configs).Error; err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"urls":              []string{},
		"filter_keywords":   []string{},
		"enable_schedule":   false,
		"schedule_interval": 3600,
	}

	for _, config := range configs {
		switch config.Key {
		case "urls", "filter_keywords":
			result[config.Key] = s.splitAndTrim(config.Value)
		case "enable_schedule":
			result[config.Key] = config.Value == "true" || config.Value == "1"
		case "schedule_interval":
			if interval, _ := strconv.Atoi(config.Value); interval != 0 {
				result[config.Key] = interval
			}
		default:
			result[config.Key] = config.Value
		}
	}
	return result, nil
}

func (s *ConfigUpdateService) splitAndTrim(value string) []string {
	var res []string
	for _, line := range strings.Split(value, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			res = append(res, trimmed)
		}
	}
	return res
}

func (s *ConfigUpdateService) updateLastUpdateTime() {
	now := utils.GetBeijingTime().Format("2006-01-02T15:04:05")
	var config models.SystemConfig
	if err := s.db.Where("key = ?", "config_update_last_update").First(&config).Error; err != nil {
		if createErr := s.db.Create(&models.SystemConfig{
			Key: "config_update_last_update", Value: now, Type: "string",
			Category: "config_update", DisplayName: "最后更新时间", Description: "配置更新任务的最后执行时间",
		}).Error; createErr != nil {
			s.log("ERROR", fmt.Sprintf("写入更新时间失败: %v", createErr))
		}
	} else {
		config.Value = now
		if saveErr := s.db.Save(&config).Error; saveErr != nil {
			s.log("ERROR", fmt.Sprintf("保存更新时间失败: %v", saveErr))
		}
	}
}

// ==========================================
// 节点获取与解析
// ==========================================

func (s *ConfigUpdateService) FetchNodesFromURLs(urls []string) ([]map[string]interface{}, error) {
	var allNodes []map[string]interface{}
	client := &http.Client{
		Timeout:   60 * time.Second,
		Transport: &http.Transport{DisableKeepAlives: false, MaxIdleConns: 10, IdleConnTimeout: 30 * time.Second},
	}

	for i, urlStr := range urls {
		s.logSection("📥", fmt.Sprintf("下载节点源 [%d/%d]", i+1, len(urls)))
		s.logItem("└─", fmt.Sprintf("URL: %s", urlStr))
		s.log("INFO", "⠋ 正在连接...")

		content, err := s.fetchURLContent(client, urlStr)
		if err != nil {
			s.log("ERROR", fmt.Sprintf("获取节点源失败: %v", err))
			continue
		}

		decoded := TryDecodeNodeList(string(content))
		s.log("INFO", "⠹ 正在解析节点...")
		nodeLinks := s.extractNodeLinks(decoded)
		s.logNodeTypeStats(urlStr, nodeLinks)
		s.logNodeNames(nodeLinks)

		for _, link := range nodeLinks {
			allNodes = append(allNodes, map[string]interface{}{"url": link, "source_url": urlStr})
		}
	}
	return allNodes, nil
}

func (s *ConfigUpdateService) fetchURLContent(client *http.Client, url string) ([]byte, error) {
	maxRetries, retryDelay := 3, 2*time.Second

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

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
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

	content, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
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
		typeCount[strings.Split(link, ":")[0]]++
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
	var nodeNames []string
	for _, link := range nodeLinks {
		if name := s.extractNodeName(link); name != "" {
			nodeNames = append(nodeNames, name)
		}
	}
	if len(nodeNames) > 0 {
		s.logSection("📋", "采集到的节点:")
		for i, name := range nodeNames {
			s.log("INFO", fmt.Sprintf("  %d. %s", i+1, name))
		}
		s.logSeparator()
	}
}

func (s *ConfigUpdateService) extractNodeName(link string) string {
	if idx := strings.Index(link, "#"); idx != -1 {
		name := link[idx+1:]
		if decoded, err := url.QueryUnescape(name); err == nil {
			return decoded
		}
		return name
	}
	return ""
}

func (s *ConfigUpdateService) extractNodeLinks(content string) []string {
	if yamlLinks := s.parseClashYAML(content); len(yamlLinks) > 0 {
		return yamlLinks
	}

	s.log("INFO", "✓ 检测到节点链接格式（非 YAML）")
	var links, invalidLinks []string
	matchedPositions := make(map[int]bool)

	for _, re := range nodeLinkPatterns {
		for _, match := range re.FindAllStringSubmatchIndex(content, -1) {
			if len(match) < 4 {
				continue
			}
			start, end := match[2], match[3]
			matchStr := content[start:end]

			if strings.HasPrefix(matchStr, "ss://") && start >= 3 && content[start-3:start] == "vme" {
				continue
			}

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
	if link = strings.TrimSpace(link); link == "" {
		return false
	}
	parts := strings.SplitN(link, ":", 2)
	if len(parts) != 2 {
		return false
	}
	scheme := parts[0]
	body := strings.Split(link, "#")[0]

	switch scheme {
	case "ss":
		if !strings.Contains(body, "@") {
			return false
		}
		return strings.Contains(strings.Split(strings.Split(body, "@")[1], "?")[0], ":")
	case "vmess", "vless", "ssr":
		return len(strings.Split(strings.TrimPrefix(body, scheme+"://"), "?")[0]) >= 10
	case "trojan", "tuic", "naive+https", "socks", "socks5", "http", "https":
		return strings.Contains(body, "@")
	case "hysteria", "hysteria2":
		return strings.Contains(body, ":")
	default:
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

func (s *ConfigUpdateService) deleteAutoImportedNodes() int64 {
	result := s.db.Where("is_manual = ?", false).Delete(&models.Node{})
	if result.Error != nil {
		s.log("ERROR", fmt.Sprintf("删除自动导入节点失败: %v", result.Error))
		return 0
	}
	return result.RowsAffected
}

type importStats struct {
	Created, Updated, Skipped int
}

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
	seenKeys := make(map[string]bool)

	var allAutoNodes []models.Node
	s.db.Where("is_manual = ?", false).Find(&allAutoNodes)
	existingNodeMap := make(map[string]*models.Node)
	for i := range allAutoNodes {
		existingNodeMap[s.generateNodeKey(allAutoNodes[i].Type, allAutoNodes[i].Name, allAutoNodes[i].Config)] = &allAutoNodes[i]
	}

	for _, item := range nodesWithOrder {
		configJSON, _ := json.Marshal(item.node)
		configStr := string(configJSON)
		nodeKey := s.generateNodeKey(item.node.Type, item.node.Name, &configStr)

		if seenKeys[nodeKey] {
			stats.Skipped++
			continue
		}
		seenKeys[nodeKey] = true
		region := s.resolveRegion(item.node.Name, item.node.Server)

		if existingNode := existingNodeMap[nodeKey]; existingNode != nil {
			existingNode.Config, existingNode.Status, existingNode.IsActive, existingNode.IsManual = &configStr, "online", true, false
			existingNode.OrderIndex, existingNode.SourceIndex, existingNode.Region, existingNode.Name = item.orderIndex, item.sourceIndex, region, item.node.Name
			if s.db.Save(existingNode).Error == nil {
				stats.Updated++
			}
		} else {
			if s.db.Where("type = ? AND name = ? AND is_manual = ?", item.node.Type, item.node.Name, true).First(&models.Node{}).Error == nil {
				stats.Skipped++
				continue
			}
			if s.db.Create(&models.Node{
				Name: item.node.Name, Type: item.node.Type, Status: "online", IsActive: true, IsManual: false,
				Config: &configStr, Region: region, OrderIndex: item.orderIndex, SourceIndex: item.sourceIndex,
			}).Error == nil {
				stats.Created++
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

	if !user.IsActive {
		ctx.Status = StatusAccountAbnormal
		return ctx
	}
	if !sub.IsActive || sub.Status != "active" {
		ctx.Status = StatusInactive
		return ctx
	}

	proxies, err := s.fetchProxiesForUser(user, sub)
	if err != nil {
		if utils.AppLogger != nil {
			utils.AppLogger.Error("获取订阅节点失败: user_id=%d subscription_id=%d err=%v", user.ID, sub.ID, err)
		}
		ctx.Status = StatusSystemError
		return ctx
	}
	ctx.Proxies = proxies

	if len(ctx.Proxies) == 0 && !sub.ExpireTime.IsZero() && sub.ExpireTime.Before(utils.GetBeijingTime()) {
		ctx.Status = StatusExpired
		return ctx
	}

	var currentDevices int64
	s.db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", sub.ID, true).Count(&currentDevices)
	ctx.CurrentDevices, ctx.DeviceLimit = int(currentDevices), sub.DeviceLimit

	if sub.DeviceLimit == 0 {
		ctx.Status = StatusDeviceOverLimit
		return ctx
	}
	if sub.DeviceLimit > 0 && int(currentDevices) >= sub.DeviceLimit {
		if s.db.Where("subscription_id = ? AND ip_address = ? AND user_agent = ?", sub.ID, clientIP, userAgent).First(&models.Device{}).Error != nil {
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

	isSubExpired := !sub.ExpireTime.IsZero() && sub.ExpireTime.Before(now)
	isSpecialExpired := (user.SpecialNodeExpiresAt.Valid && utils.ToBeijingTime(user.SpecialNodeExpiresAt.Time).Before(now)) ||
		(user.SpecialNodeSubscriptionType != "special_only" && isSubExpired)

	s.appendCustomNodes(user.ID, now, isSpecialExpired, &proxies, processedNodes)
	if user.SpecialNodeSubscriptionType != "special_only" && !isSubExpired {
		s.appendSystemNodes(&proxies, processedNodes)
	}

	return proxies, nil
}

func (s *ConfigUpdateService) appendCustomNodes(userID uint, now time.Time, isGlobalExpired bool, proxies *[]*ProxyNode, processed map[string]bool) {
	var customNodes []models.CustomNode
	s.db.Joins("JOIN user_custom_nodes ON user_custom_nodes.custom_node_id = custom_nodes.id").
		Where("user_custom_nodes.user_id = ? AND custom_nodes.is_active = ?", userID, true).Find(&customNodes)

	for _, cn := range customNodes {
		isNodeExpired := (cn.ExpireTime != nil && utils.ToBeijingTime(*cn.ExpireTime).Before(now)) || (cn.FollowUserExpire && isGlobalExpired)
		if isNodeExpired || cn.Status == "timeout" {
			continue
		}

		var proxyNode ProxyNode
		if err := json.Unmarshal([]byte(cn.Config), &proxyNode); err != nil {
			continue
		}

		proxyNode.Name = cn.DisplayName
		if proxyNode.Name == "" {
			proxyNode.Name = "专线-" + cn.Name
		}

		if key := s.generateNodeDedupKey(proxyNode.Type, proxyNode.Server, proxyNode.Port); !processed[key] {
			processed[key] = true
			*proxies = append(*proxies, &proxyNode)
		}
	}
}

func (s *ConfigUpdateService) appendSystemNodes(proxies *[]*ProxyNode, processed map[string]bool) {
	cacheService := &CacheService{}
	if cachedNodes, ok := cacheService.GetSystemNodesCache(); ok {
		for _, node := range cachedNodes {
			if key := s.generateNodeDedupKey(node.Type, node.Server, node.Port); !processed[key] {
				processed[key] = true
				*proxies = append(*proxies, node)
			}
		}
		return
	}

	var nodes []models.Node
	s.db.Where("is_active = ? AND status != ?", true, "timeout").Find(&nodes)

	var systemNodes []*ProxyNode
	for _, node := range nodes {
		if node.Config == nil || *node.Config == "" {
			continue
		}
		var proxy ProxyNode
		if json.Unmarshal([]byte(*node.Config), &proxy) == nil {
			proxy.Name = node.Name
			if key := s.generateNodeDedupKey(proxy.Type, proxy.Server, proxy.Port); !processed[key] {
				processed[key] = true
				*proxies = append(*proxies, &proxy)
				systemNodes = append(systemNodes, &proxy)
			}
		}
	}

	if len(systemNodes) > 0 {
		go cacheService.SetSystemNodesCache(systemNodes)
	}
}

func (s *ConfigUpdateService) generateNodeDedupKey(nodeType, server string, port int) string {
	return fmt.Sprintf("%s:%s:%d", nodeType, server, port)
}

func (s *ConfigUpdateService) calculateCacheTTL(sub *models.Subscription, user *models.User) time.Duration {
	now := utils.GetBeijingTime()
	if !sub.ExpireTime.IsZero() {
		if sub.ExpireTime.Before(now) {
			return 0
		}
		if sub.ExpireTime.Sub(now) < 24*time.Hour {
			return 1 * time.Minute
		}
	}
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
	cacheService := &CacheService{}
	if cached, ok := cacheService.GetSubscriptionConfigCache(token, "clash"); ok {
		return cached, nil
	}

	nodes, err := s.prepareExportNodes(token, clientIP, userAgent)
	if err != nil {
		return "", err
	}
	ctx := s.GetSubscriptionContext(token, clientIP, userAgent)
	config := s.generateClashYAML(nodes, ctx)

	if ctx.Status == StatusNormal {
		if ttl := s.calculateCacheTTL(&ctx.Subscription, &ctx.User); ttl > 0 {
			go cacheService.SetSubscriptionConfigCache(token, "clash", config, ttl)
		}
	}
	return config, nil
}

func (s *ConfigUpdateService) GenerateUniversalConfig(token string, clientIP string, userAgent string, format string) (string, error) {
	cacheService := &CacheService{}
	cacheFormat := "base64"
	if format == "ssr" {
		cacheFormat = "ssr"
	}
	if cached, ok := cacheService.GetSubscriptionConfigCache(token, cacheFormat); ok {
		return cached, nil
	}

	nodes, err := s.prepareExportNodes(token, clientIP, userAgent)
	if err != nil {
		return "", err
	}

	var links []string
	for _, node := range nodes {
		link := s.nodeToLink(node)
		if format == "ssr" && node.Type == "ssr" {
			link = s.nodeToSSRLink(node)
		}
		if link != "" {
			links = append(links, link)
		}
	}

	config := base64.StdEncoding.EncodeToString([]byte(strings.Join(links, "\n")))
	ctx := s.GetSubscriptionContext(token, clientIP, userAgent)
	if ctx.Status == StatusNormal {
		if ttl := s.calculateCacheTTL(&ctx.Subscription, &ctx.User); ttl > 0 {
			go cacheService.SetSubscriptionConfigCache(token, cacheFormat, config, ttl)
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

	usedNames := make(map[string]bool)
	var proxyNames []string
	for _, proxy := range filteredProxies {
		originalName, newName, counter := proxy.Name, proxy.Name, 1
		for usedNames[newName] {
			newName = fmt.Sprintf("%s_%d", originalName, counter)
			counter++
		}
		proxy.Name = newName
		usedNames[newName] = true
		proxyNames = append(proxyNames, proxy.Name)
	}

	subscriptionName := s.GenerateSubscriptionName(ctx)
	templatePath := filepath.Clean(filepath.Join("uploads", "config", "temp.yaml"))
	if cfg, err := s.getConfig(); err == nil {
		templatePath = ResolveTemplatePath(cfg)
	}

	if utils.IsWithinBaseDir(".", templatePath) {
		if templateData, err := os.ReadFile(templatePath); err == nil {
			var templateConfig map[string]interface{}
			if yaml.Unmarshal(templateData, &templateConfig) == nil {
				templateConfig["name"] = subscriptionName
				proxyList := make([]map[string]interface{}, 0, len(filteredProxies))
				for _, proxy := range filteredProxies {
					proxyList = append(proxyList, s.nodeToMap(proxy))
				}
				templateConfig["proxies"] = proxyList

				if proxyGroups, ok := templateConfig["proxy-groups"].([]interface{}); ok {
					s.updateProxyGroups(proxyGroups, proxyNames)
					templateConfig["proxy-groups"] = proxyGroups
				}
				if output, err := yaml.Marshal(templateConfig); err == nil {
					return unescapeUnicode(string(output))
				}
			}
		}
	}
	return s.generateDefaultClashYAML(filteredProxies, proxyNames, subscriptionName)
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

		if gType == "select" || gType == "url-test" || gType == "fallback" || gType == "load-balance" {
			existingProxies := make([]string, 0)
			if oldProxies, ok := group["proxies"].([]interface{}); ok {
				for _, p := range oldProxies {
					if pStr, ok := p.(string); ok && (pStr == "DIRECT" || pStr == "REJECT" || groupNames[pStr]) {
						existingProxies = append(existingProxies, pStr)
					}
				}
			}
			if gType == "select" {
				group["proxies"] = append(existingProxies, proxyNames...)
			} else {
				group["proxies"] = proxyNames
			}
		}
	}
}

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
		case StatusSystemError:
			return "系统繁忙"
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

	builder.WriteString(fmt.Sprintf(`name: %s
port: 7890
socks-port: 7891
allow-lan: true
mode: Rule
log-level: info
external-controller: 127.0.0.1:9090

proxies:
`, s.escapeYAMLString(subscriptionName)))

	for _, proxy := range proxies {
		builder.WriteString(s.nodeToYAML(proxy, 2))
	}

	builder.WriteString(`
proxy-groups:
  - name: "🚀 节点选择"
    type: select
    proxies:
      - "♻️ 自动选择"
`)
	for _, name := range proxyNames {
		builder.WriteString(fmt.Sprintf("      - %s\n", s.escapeYAMLString(name)))
	}

	builder.WriteString(`  - name: "♻️ 自动选择"
    type: url-test
    url: http://www.gstatic.com/generate_204
    interval: 300
    tolerance: 50
    proxies:
`)
	for _, name := range proxyNames {
		builder.WriteString(fmt.Sprintf("      - %s\n", s.escapeYAMLString(name)))
	}

	builder.WriteString(`
rules:
  - DOMAIN-SUFFIX,local,DIRECT
  - IP-CIDR,127.0.0.0/8,DIRECT
  - IP-CIDR,172.16.0.0/12,DIRECT
  - IP-CIDR,192.168.0.0/16,DIRECT
  - GEOIP,CN,DIRECT
  - MATCH,🚀 节点选择
`)

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
		reason, solution = "订阅已过期", fmt.Sprintf("请前往官网续费 (过期时间: %s)", utils.FormatBeijingDate(ctx.Subscription.ExpireTime))
	case StatusInactive:
		reason, solution = "订阅已失效", "请联系管理员检查订阅状态"
	case StatusAccountAbnormal:
		reason, solution = "账户异常", "您的账户状态异常或已被禁用，请联系客服"
	case StatusDeviceOverLimit:
		reason, solution = "设备数量超限", fmt.Sprintf("当前设备 %d/%d，请在官网删除不使用的设备", ctx.CurrentDevices, ctx.DeviceLimit)
	case StatusOldAddress:
		reason, solution = "订阅地址已变更", "请登录官网获取最新的订阅地址"
	case StatusNotFound:
		reason, solution = "订阅不存在", "请检查订阅链接是否正确，或重新复制"
	case StatusSystemError:
		reason, solution = "系统异常", "节点加载失败，请稍后重试或联系管理员"
	default:
		reason, solution = "账户异常", "检测到账户异常，请联系管理员"
	}

	qqMsg := "💬 客服: 请在系统设置中配置"
	if s.supportQQ != "" {
		qqMsg = fmt.Sprintf("💬 客服: %s", s.supportQQ)
	}

	return []*ProxyNode{
		s.createMessageNode(fmt.Sprintf("📢 官网: %s", s.siteURL)),
		s.createMessageNode(fmt.Sprintf("❌ 原因: %s", reason), "error"),
		s.createMessageNode(fmt.Sprintf("💡 解决: %s", solution), "error"),
		s.createMessageNode(qqMsg, "error"),
	}
}

func (s *ConfigUpdateService) createMessageNode(name string, password ...string) *ProxyNode {
	pwd := "info"
	if len(password) > 0 {
		pwd = password[0]
	}
	return &ProxyNode{
		Name: name, Type: "ss", Server: "baidu.com", Port: 1234, Cipher: "aes-128-gcm", Password: pwd,
	}
}

// ==========================================
// Map 数据提取 Helpers
// ==========================================

func optStr(opts map[string]interface{}, key string) string {
	if opts != nil {
		if v, ok := opts[key].(string); ok {
			return v
		}
	}
	return ""
}

func optBool(opts map[string]interface{}, key string) bool {
	if opts != nil {
		if v, ok := opts[key].(bool); ok {
			return v
		}
	}
	return false
}

func optMap(opts map[string]interface{}, key string) map[string]interface{} {
	if opts != nil {
		if v, ok := opts[key].(map[string]interface{}); ok {
			return v
		}
	}
	return nil
}

// ==========================================
// 节点对象转 YAML/Map
// ==========================================

func (s *ConfigUpdateService) nodeToYAML(node *ProxyNode, indent int) string {
	indentStr := strings.Repeat(" ", indent)
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("%s- name: %s\n", indentStr, s.escapeYAMLString(node.Name)))
	builder.WriteString(fmt.Sprintf("%s  type: %s\n", indentStr, node.Type))
	builder.WriteString(fmt.Sprintf("%s  server: %s\n", indentStr, node.Server))
	builder.WriteString(fmt.Sprintf("%s  port: %d\n", indentStr, node.Port))

	m := s.nodeToMap(node)
	keys := make([]string, 0, len(m))
	for k := range m {
		if k != "name" && k != "type" && k != "server" && k != "port" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for _, k := range keys {
		s.writeYAMLValue(&builder, indentStr+"  ", k, m[k], indent+2)
	}
	return builder.String()
}

func (s *ConfigUpdateService) nodeToMap(node *ProxyNode) map[string]interface{} {
	result := map[string]interface{}{
		"name": node.Name, "type": node.Type, "server": node.Server, "port": node.Port,
	}

	setIfNotEmpty := func(k, v string) {
		if v != "" {
			result[k] = v
		}
	}

	if s.getRequirePassword(node.Type) {
		result["password"] = node.Password
	}

	switch node.Type {
	case "ss":
		setIfNotEmpty("cipher", node.Cipher)
		if pluginName := optStr(node.Options, "plugin"); pluginName != "" {
			result["plugin"] = pluginName
			if pluginOpts := optMap(node.Options, "plugin-opts"); pluginOpts != nil {
				result["plugin-opts"] = pluginOpts
			}
		}
	case "vmess":
		setIfNotEmpty("uuid", node.UUID)
		result["alterId"], result["cipher"] = 0, "auto"
		if val, ok := node.Options["alterId"]; ok {
			result["alterId"] = val
		}
		setIfNotEmpty("cipher", node.Cipher)
	case "vless", "tuic":
		setIfNotEmpty("uuid", node.UUID)
		if node.Type == "tuic" {
			s.buildTuicMap(node, result)
		}
	case "ssr":
		setIfNotEmpty("cipher", node.Cipher)
	case "anytls":
		result["udp"] = node.UDP
		if sni := optStr(node.Options, "servername"); sni != "" {
			result["sni"] = sni
			delete(node.Options, "servername")
		}
	case "hysteria", "hysteria2":
		if result["password"] == "" {
			result["password"] = optStr(node.Options, "auth")
		}
		if node.Type == "hysteria2" {
			result["auth"] = result["password"]
		}
	case "socks", "socks5", "http":
		setIfNotEmpty("username", node.UUID)
	}

	if node.TLS || node.Type == "tuic" || node.Type == "anytls" {
		result["tls"] = true
	}
	if node.Network != "" && node.Network != "tcp" {
		result["network"] = node.Network
	}
	if node.UDP {
		result["udp"] = true
	}

	for key, value := range node.Options {
		if key != "alterId" || node.Type != "vmess" {
			result[key] = value
		}
	}

	return result
}

func (s *ConfigUpdateService) getRequirePassword(nodeType string) bool {
	switch nodeType {
	case "ss", "trojan", "ssr", "tuic", "anytls", "hysteria", "hysteria2", "socks", "socks5", "http":
		return true
	}
	return false
}

func (s *ConfigUpdateService) buildTuicMap(node *ProxyNode, result map[string]interface{}) {
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

	if cc := optStr(node.Options, "congestion_control"); cc != "" {
		result["congestion-controller"] = cc
		delete(node.Options, "congestion_control")
	} else if cc := optStr(node.Options, "congestion-controller"); cc != "" {
		result["congestion-controller"] = cc
	}

	if sni := optStr(node.Options, "servername"); sni != "" {
		result["sni"] = sni
		delete(node.Options, "servername")
	}
}

// 保持 YAML 写入辅助函数的原样（避免影响兼容性）
func (s *ConfigUpdateService) writeYAMLValue(builder *strings.Builder, indentStr, key string, value interface{}, indentLevel int) {
	escapedKey := s.escapeYAMLString(key)
	switch v := value.(type) {
	case map[string]interface{}:
		builder.WriteString(fmt.Sprintf("%s%s:\n", indentStr, escapedKey))
		s.writeMapContent(builder, indentStr+"  ", v, key, indentLevel+1)
	case []interface{}:
		builder.WriteString(fmt.Sprintf("%s%s:\n", indentStr, escapedKey))
		for _, item := range v {
			builder.WriteString(fmt.Sprintf("%s  - %s\n", indentStr, s.formatYAMLInline(item)))
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

	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		val := v[k]
		if strMap, ok := val.(map[string]string); ok {
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
	writeItem := func(item interface{}) {
		builder.WriteString(fmt.Sprintf("%s  - %s\n", indentStr, s.formatYAMLInline(item)))
	}
	switch v := val.(type) {
	case string:
		writeItem(v)
	case []string:
		for _, item := range v {
			writeItem(item)
		}
	case []interface{}:
		for _, item := range v {
			writeItem(item)
		}
	}
}

func (s *ConfigUpdateService) formatYAMLInline(v interface{}) string {
	switch val := v.(type) {
	case string:
		return s.escapeYAMLString(val)
	default:
		return s.escapeYAMLString(fmt.Sprintf("%v", val))
	}
}

func (s *ConfigUpdateService) escapeYAMLString(str string) string {
	if str == "" {
		return `""`
	}
	specialChars := ":\"'\n\r\t#@&*?|>!%`[]{},\x00"
	if strings.ContainsAny(str, specialChars) || strings.HasPrefix(str, " ") || strings.HasSuffix(str, " ") {
		escaped := strings.ReplaceAll(str, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		escaped = strings.ReplaceAll(escaped, "\n", "\\n")
		return fmt.Sprintf(`"%s"`, escaped)
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

func (s *ConfigUpdateService) getQueryFromOptions(node *ProxyNode) url.Values {
	q := url.Values{}
	if node.Options == nil {
		return q
	}

	if sni := optStr(node.Options, "servername"); sni != "" {
		q.Set("sni", sni)
	}
	if peer := optStr(node.Options, "peer"); peer != "" && (q.Get("sni") == "" || node.Type == "anytls") {
		q.Set("peer", peer)
	}
	if optBool(node.Options, "skip-cert-verify") {
		q.Set("insecure", "1")
		q.Set("allow_insecure", "1")
	}
	if fp := optStr(node.Options, "client-fingerprint"); fp != "" {
		q.Set("fp", fp)
	}

	if alpnVal, ok := node.Options["alpn"]; ok {
		if strs, ok := alpnVal.([]string); ok && len(strs) > 0 {
			q.Set("alpn", strings.Join(strs, ","))
		} else if infs, ok := alpnVal.([]interface{}); ok {
			var tmp []string
			for _, v := range infs {
				if str, ok := v.(string); ok {
					tmp = append(tmp, str)
				}
			}
			if len(tmp) > 0 {
				q.Set("alpn", strings.Join(tmp, ","))
			}
		}
	}

	switch node.Type {
	case "vless", "trojan":
		s.buildTransportQuery(node, q)
	case "hysteria", "hysteria2":
		if auth := optStr(node.Options, "auth"); auth != "" {
			q.Set("auth", auth)
		}
		if up := optStr(node.Options, "up"); up != "" {
			trimmed := strings.TrimSuffix(up, " mbps")
			q.Set("upmbps", trimmed)
			q.Set("mbpsUp", trimmed)
		}
		if down := optStr(node.Options, "down"); down != "" {
			trimmed := strings.TrimSuffix(down, " mbps")
			q.Set("downmbps", trimmed)
			q.Set("mbpsDown", trimmed)
		}
	case "tuic":
		if cc := optStr(node.Options, "congestion_control"); cc != "" {
			q.Set("congestion_control", cc)
		}
		if mode := optStr(node.Options, "udp_relay_mode"); mode != "" {
			q.Set("udp_relay_mode", mode)
		}
	case "naive":
		if optBool(node.Options, "padding") {
			q.Set("padding", "true")
		}
	}

	return q
}

func (s *ConfigUpdateService) buildTransportQuery(node *ProxyNode, q url.Values) {
	if node.Network != "" {
		q.Set("type", node.Network)
	}

	if wsOpts := optMap(node.Options, "ws-opts"); wsOpts != nil {
		if path := optStr(wsOpts, "path"); path != "" {
			q.Set("path", path)
		}
		if headers := optMap(wsOpts, "headers"); headers != nil {
			if host := optStr(headers, "Host"); host != "" {
				q.Set("host", host)
			}
		} else if headers, ok := wsOpts["headers"].(map[string]string); ok {
			if host, ok := headers["Host"]; ok && host != "" {
				q.Set("host", host)
			}
		}
	}

	if grpcOpts := optMap(node.Options, "grpc-opts"); grpcOpts != nil {
		if sn := optStr(grpcOpts, "grpc-service-name"); sn != "" {
			q.Set("serviceName", sn)
		}
	}

	if h2Opts := optMap(node.Options, "h2-opts"); h2Opts != nil {
		if path := optStr(h2Opts, "path"); path != "" {
			q.Set("path", path)
		}
		if hosts, ok := h2Opts["host"].([]string); ok && len(hosts) > 0 {
			q.Set("host", hosts[0])
		}
	}

	if ht := optStr(node.Options, "header-type"); ht != "" {
		q.Set("headerType", ht)
	}

	if node.Type == "vless" {
		if node.TLS {
			if realityOpts := optMap(node.Options, "reality-opts"); realityOpts != nil {
				q.Set("security", "reality")
				if pbk := optStr(realityOpts, "public-key"); pbk != "" {
					q.Set("pbk", pbk)
				}
				if sid := optStr(realityOpts, "short-id"); sid != "" {
					q.Set("sid", sid)
				}
				if pqv := optStr(realityOpts, "pqv"); pqv != "" {
					q.Set("pqv", pqv)
				}
			} else {
				q.Set("security", "tls")
			}
		}
		if flow := optStr(node.Options, "flow"); flow != "" {
			q.Set("flow", flow)
		}
		if enc := optStr(node.Options, "encryption"); enc != "" {
			q.Set("encryption", enc)
		}
	}
}

func (s *ConfigUpdateService) vmessToLink(proxy *ProxyNode) string {
	network, obfsType := proxy.Network, "none"
	if network == "http" {
		network, obfsType = "tcp", "http"
	}

	data := map[string]interface{}{
		"v": "2", "ps": proxy.Name, "add": proxy.Server, "port": proxy.Port,
		"id": proxy.UUID, "net": network, "type": obfsType,
		"tls": "", "sni": "", "host": "", "path": "", "aid": 0, "scy": "auto",
	}

	if proxy.TLS {
		data["tls"] = "tls"
	}

	if proxy.Options != nil {
		if aid, ok := proxy.Options["alterId"]; ok {
			data["aid"] = aid
		}
		if cipher := optStr(proxy.Options, "cipher"); cipher != "" {
			data["scy"] = cipher
		}
		if sni := optStr(proxy.Options, "servername"); sni != "" {
			data["sni"] = sni
		}
		if optBool(proxy.Options, "skip-cert-verify") {
			data["insecure"] = "1"
		}

		if wsOpts := optMap(proxy.Options, "ws-opts"); wsOpts != nil {
			if path := optStr(wsOpts, "path"); path != "" {
				data["path"] = path
			}
			if headers := optMap(wsOpts, "headers"); headers != nil {
				if host := optStr(headers, "Host"); host != "" {
					data["host"] = host
				}
			}
		}
		if httpOpts := optMap(proxy.Options, "http-opts"); httpOpts != nil {
			if paths, ok := httpOpts["path"].([]string); ok && len(paths) > 0 {
				data["path"] = paths[0]
			}
			if headers := optMap(httpOpts, "headers"); headers != nil {
				if hosts, ok := headers["Host"].([]string); ok && len(hosts) > 0 {
					data["host"] = hosts[0]
				}
			}
		}
		if h2Opts := optMap(proxy.Options, "h2-opts"); h2Opts != nil {
			if path := optStr(h2Opts, "path"); path != "" {
				data["path"] = path
			}
			if hosts, ok := h2Opts["host"].([]string); ok && len(hosts) > 0 {
				data["host"] = hosts[0]
			}
		}
		if grpcOpts := optMap(proxy.Options, "grpc-opts"); grpcOpts != nil {
			if sn := optStr(grpcOpts, "grpc-service-name"); sn != "" {
				data["path"] = sn
			}
		}
	}

	jsonData, _ := json.Marshal(data)
	return "vmess://" + base64.StdEncoding.EncodeToString(jsonData)
}

func (s *ConfigUpdateService) shadowsocksToLink(proxy *ProxyNode) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", proxy.Cipher, proxy.Password)))
	var query url.Values

	if pluginName := optStr(proxy.Options, "plugin"); pluginName != "" {
		query = url.Values{}
		pluginStr := pluginName
		if pluginName == "obfs" {
			pluginStr = "obfs-local"
		}
		if pluginOpts := optMap(proxy.Options, "plugin-opts"); pluginOpts != nil {
			if mode := optStr(pluginOpts, "mode"); mode != "" {
				pluginStr += ";obfs=" + mode
			}
			if host := optStr(pluginOpts, "host"); host != "" {
				pluginStr += ";obfs-host=" + host
			}
			if path := optStr(pluginOpts, "path"); path != "" {
				pluginStr += ";obfs-uri=" + path
			}
			if optBool(pluginOpts, "tls") {
				pluginStr += ";tls"
			}
		}
		query.Set("plugin", pluginStr)
	}
	return s.buildStandardNodeURL("ss", encoded, "", proxy.Server, proxy.Port, proxy.Name, query)
}

func (s *ConfigUpdateService) nodeToSSRLink(node *ProxyNode) string {
	getString := func(key, def string) string {
		if v := optStr(node.Options, key); v != "" {
			return v
		}
		return def
	}

	ssrStr := fmt.Sprintf("%s:%d:%s:%s:%s:%s/?obfsparam=%s&protoparam=%s&remarks=%s&group=%s",
		node.Server, node.Port, getString("protocol", "origin"), node.Cipher, getString("obfs", "plain"),
		base64.RawURLEncoding.EncodeToString([]byte(node.Password)),
		base64.RawURLEncoding.EncodeToString([]byte(getString("obfs-param", ""))),
		base64.RawURLEncoding.EncodeToString([]byte(getString("protocol-param", ""))),
		base64.RawURLEncoding.EncodeToString([]byte(node.Name)),
		base64.RawURLEncoding.EncodeToString([]byte("GoWeb")))

	return "ssr://" + base64.RawURLEncoding.EncodeToString([]byte(ssrStr))
}

func unescapeUnicode(s string) string {
	return regexp.MustCompile(`\\U([0-9A-Fa-f]{8})`).ReplaceAllStringFunc(s, func(match string) string {
		if codePoint, err := strconv.ParseInt(match[2:], 16, 64); err == nil {
			return string(utils.MustSafeInt64ToRune(codePoint))
		}
		return match
	})
}

// ==========================================
// 格式化日志辅助函数
// ==========================================

func (s *ConfigUpdateService) logSeparator() {
	s.log("INFO", "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func (s *ConfigUpdateService) logSection(icon, title string) {
	s.logSeparator()
	s.log("INFO", fmt.Sprintf("%s %s", icon, title))
	s.logSeparator()
}

func (s *ConfigUpdateService) logItem(prefix, content string) {
	s.log("INFO", fmt.Sprintf("           %s %s", prefix, content))
}

func formatBytes(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}

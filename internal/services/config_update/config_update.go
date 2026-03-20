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
	"cboard-go/internal/services/cache_service"
	"cboard-go/internal/utils"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

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

// 优化：将15个单独的正则合并为一个，极大提高匹配性能
var nodeLinkPattern = regexp.MustCompile(`(?i)(?:^|\s)((?:vmess|vless|trojan|ssr?|hysteria2?|tuic|naive(?:\+https)?|anytls|socks5?|https?|wg)://[^\s]+)`)

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

type importStats struct {
	Created int // 新增节点数
	Updated int // 更新节点数
	Skipped int // 跳过节点数（与手动节点同名）
}

type updateStats struct {
	parseFailed   int
	duplicates    int
	invalidLinks  int
	missingSource int
	filtered      int
}

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
	}

	service.refreshSystemConfig()
	return service
}

func (s *ConfigUpdateService) refreshSystemConfig() {
	if domain := utils.GetDomainFromDB(s.db); domain != "" {
		s.siteURL = utils.FormatDomainURL(domain)
	} else {
		s.siteURL = "请在系统设置中配置域名"
	}

	var qqConfig models.SystemConfig
	if err := s.db.Where("key = ? AND category = ?", "support_qq", "general").First(&qqConfig).Error; err == nil && qqConfig.Value != "" {
		s.supportQQ = strings.TrimSpace(qqConfig.Value)
	} else {
		s.supportQQ = ""
	}
}

func (s *ConfigUpdateService) IsRunning() bool {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()
	return s.isRunning
}

func (s *ConfigUpdateService) GetStatus() map[string]interface{} {
	var lastUpdate string
	var config models.SystemConfig
	if err := s.db.Where("key = ?", "config_update_last_update").First(&config).Error; err == nil {
		lastUpdate = config.Value
	}
	return map[string]interface{}{"is_running": s.IsRunning(), "last_update": lastUpdate, "next_update": ""}
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
	var config models.SystemConfig
	if err := s.db.Where("key = ?", "config_update_logs").First(&config).Error; err != nil {
		return s.saveLogConfig("[]")
	}
	config.Value = "[]"
	return s.db.Save(&config).Error
}

func (s *ConfigUpdateService) clearLogBuffer() {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()
	s.logBuffer = make([]map[string]interface{}, 0, 500)
}

func (s *ConfigUpdateService) GetSSEManager() *SSEManager { return s.sseManager }

// --- 日志辅助函数 (避免散落的 fmt.Sprintf) ---
func (s *ConfigUpdateService) infof(format string, args ...any) {
	s.log("INFO", fmt.Sprintf(format, args...))
}
func (s *ConfigUpdateService) errorf(format string, args ...any) {
	s.log("ERROR", fmt.Sprintf(format, args...))
}
func (s *ConfigUpdateService) warnf(format string, args ...any) {
	s.log("WARN", fmt.Sprintf(format, args...))
}
func (s *ConfigUpdateService) debugf(format string, args ...any) {
	s.log("DEBUG", fmt.Sprintf(format, args...))
}

func (s *ConfigUpdateService) log(level, message string) {
	now := utils.FormatBeijingTime(utils.GetBeijingTime())
	entry := map[string]interface{}{"timestamp": now, "time": now, "level": level, "message": message}

	s.logMutex.Lock()
	s.logBuffer = append(s.logBuffer, entry)
	if len(s.logBuffer) > 500 {
		s.logBuffer = s.logBuffer[len(s.logBuffer)-500:]
	}
	s.logMutex.Unlock()

	if s.sseManager != nil {
		s.sseManager.Broadcast(entry)
	}
	if utils.AppLogger != nil {
		if level == "ERROR" {
			utils.AppLogger.Error("%s", message)
		} else {
			utils.AppLogger.Info("%s", message)
		}
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
		Key: "config_update_logs", Value: value, Type: "json", Category: "config_update",
		DisplayName: "配置更新日志", Description: "配置更新任务日志",
	}).Error
}

func (s *ConfigUpdateService) GetConfig() (map[string]interface{}, error) { return s.getConfig() }

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
		s.errorf("获取配置失败: %v", err)
		return err
	}

	urls := config["urls"].([]string)
	if len(urls) == 0 {
		s.errorf("未配置节点源URL")
		return fmt.Errorf("未配置节点源URL")
	}

	s.infof("📋 配置信息\n           └─ 节点源数量: %d 个", len(urls))

	nodes, err := s.FetchNodesFromURLs(urls)
	if err != nil {
		s.errorf("获取节点失败: %v", err)
		return err
	}
	if len(nodes) == 0 {
		s.warnf("未获取到有效节点")
		return fmt.Errorf("未获取到有效节点")
	}

	s.infof("共获取到 %d 个有效节点链接，准备入库", len(nodes))
	filterKeywords := s.extractFilterKeywords(config)

	nodesWithOrder, stats := s.processFetchedNodes(urls, nodes, filterKeywords)
	s.logUpdateStats(stats, len(nodesWithOrder))

	deletedCount := s.deleteAutoImportedNodes()
	s.logSection("💾", "数据库操作")
	s.infof("🗑️  删除旧节点: %d 个\n           ⠼ 正在导入节点到数据库...", deletedCount)

	importStats := s.importNodesToDatabaseWithOrder(nodesWithOrder)
	s.updateLastUpdateTime()

	s.infof("➕ 新增节点: %d 个\n🔄 更新节点: %d 个\n⏭️  跳过节点: %d 个 (手动添加)", importStats.Created, importStats.Updated, importStats.Skipped)
	s.logSection("✅", "任务完成")
	s.infof("           └─ 最终结果: 成功导入 %d 个节点", importStats.Created)

	s.clearAllCaches()
	s.infof("💾 保存日志到数据库...")
	if err := s.flushLogsToDB(); err != nil {
		s.errorf("保存日志失败: %v", err)
	} else {
		s.infof("✓ 日志已保存")
	}

	return nil
}

func (s *ConfigUpdateService) clearAllCaches() {
	cs, cache := cache_service.NewCacheService(), &CacheService{}
	_ = cs.ClearNodesCache()
	_ = cache.ClearSystemNodesCache()
	_ = cache.ClearAllSubscriptionCache()
	s.logSection("🧹", "清理缓存")
	s.infof("✓ 节点列表缓存已清除\n✓ 系统节点缓存已清除\n✓ 订阅配置缓存已清除")
}

func (s *ConfigUpdateService) extractFilterKeywords(config map[string]interface{}) []string {
	if kw, ok := config["filter_keywords"].([]string); ok {
		return kw
	}
	if kwStr, ok := config["filter_keywords"].(string); ok && kwStr != "" {
		return s.splitAndTrim(kwStr)
	}
	return nil
}

func (s *ConfigUpdateService) logUpdateStats(stats updateStats, success int) {
	if stats.parseFailed > 0 {
		s.warnf("解析失败的节点: %d 个", stats.parseFailed)
	}
	if stats.filtered > 0 {
		s.infof("被关键词过滤的节点: %d 个", stats.filtered)
	}
	if stats.duplicates > 0 {
		s.infof("去重跳过的节点: %d 个", stats.duplicates)
	}
	if stats.invalidLinks > 0 {
		s.warnf("无效链接的节点: %d 个", stats.invalidLinks)
	}
	s.infof("成功解析并准备入库的节点: %d 个", success)
}

func (s *ConfigUpdateService) processFetchedNodes(urls []string, nodes []map[string]interface{}, filterKeywords []string) ([]nodeWithOrder, updateStats) {
	var nodesWithOrder []nodeWithOrder
	stats := updateStats{}
	seenKeys, usedNames := make(map[string]bool), make(map[string]bool)
	nodesByURL := make(map[string][]map[string]interface{})

	for _, n := range nodes {
		if u, _ := n["source_url"].(string); u != "" {
			nodesByURL[u] = append(nodesByURL[u], n)
		} else {
			stats.missingSource++
		}
	}

	for urlIndex, url := range urls {
		urlNodes := nodesByURL[url]
		if len(urlNodes) == 0 {
			continue
		}

		s.infof("⠸ 开始处理订阅地址 [%d/%d] 的节点，共 %d 个链接", urlIndex+1, len(urls), len(urlNodes))
		var links []string
		for _, n := range urlNodes {
			if link, ok := n["url"].(string); ok {
				links = append(links, link)
			} else {
				stats.invalidLinks++
			}
		}

		results := s.parserPool.ParseLinks(links)
		counts := struct{ Processed, Failed, Filtered, Duplicate int }{}

		for idx, result := range results {
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
					s.warnf("解析失败 [订阅地址 %d/%d]: %v, 链接: %s", urlIndex+1, len(urls), result.Err, truncateString(result.Link, 50))
				}
				continue
			}

			if filtered, kw := s.isNodeFiltered(result.Node, filterKeywords); filtered {
				stats.filtered++
				counts.Filtered++
				s.debugf("节点被过滤: %s (关键词: %s)", result.Node.Name, kw)
				continue
			}

			counts.Processed++
			result.Node.Name = s.ensureUniqueName(result.Node.Name, usedNames)
			usedNames[result.Node.Name] = true
			nodesWithOrder = append(nodesWithOrder, nodeWithOrder{node: result.Node, orderIndex: urlIndex*10000 + idx, sourceIndex: urlIndex + 1})
		}
		s.infof("✓ 订阅地址 [%d/%d] 完成: 成功=%d, 失败=%d, 过滤=%d, 重复=%d", urlIndex+1, len(urls), counts.Processed, counts.Failed, counts.Filtered, counts.Duplicate)
	}
	return nodesWithOrder, stats
}

func (s *ConfigUpdateService) isNodeFiltered(node *ProxyNode, keywords []string) (bool, string) {
	nameL, serverL := strings.ToLower(node.Name), strings.ToLower(node.Server)
	for _, kw := range keywords {
		if kwL := strings.ToLower(strings.TrimSpace(kw)); kwL != "" && (strings.Contains(nameL, kwL) || strings.Contains(serverL, kwL)) {
			return true, kw
		}
	}
	return false, ""
}

func (s *ConfigUpdateService) ensureUniqueName(name string, used map[string]bool) string {
	if !used[name] {
		return name
	}
	for i := 1; ; i++ {
		if newName := fmt.Sprintf("%s-%d", name, i); !used[newName] {
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

	res := map[string]interface{}{
		"urls": []string{}, "filter_keywords": []string{},
		"enable_schedule": false, "schedule_interval": 3600,
	}
	for _, c := range configs {
		switch c.Key {
		case "urls", "filter_keywords":
			res[c.Key] = s.splitAndTrim(c.Value)
		case "enable_schedule":
			res[c.Key] = c.Value == "true" || c.Value == "1"
		case "schedule_interval":
			if val, _ := strconv.Atoi(c.Value); val > 0 {
				res[c.Key] = val
			}
		default:
			res[c.Key] = c.Value
		}
	}
	return res, nil
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
	var cfg models.SystemConfig
	if err := s.db.Where("key = ?", "config_update_last_update").First(&cfg).Error; err != nil {
		s.db.Create(&models.SystemConfig{Key: "config_update_last_update", Value: now, Type: "string", Category: "config_update", DisplayName: "最后更新", Description: ""})
	} else {
		cfg.Value = now
		s.db.Save(&cfg)
	}
}

func (s *ConfigUpdateService) FetchNodesFromURLs(urls []string) ([]map[string]interface{}, error) {
	var allNodes []map[string]interface{}
	client := &http.Client{Timeout: 60 * time.Second, Transport: &http.Transport{MaxIdleConns: 10, IdleConnTimeout: 30 * time.Second}}

	for i, urlStr := range urls {
		s.logSection("📥", fmt.Sprintf("下载节点源 [%d/%d]", i+1, len(urls)))
		s.infof("           └─ URL: %s\n⠋ 正在连接...", urlStr)

		content, err := s.fetchURLContent(client, urlStr)
		if err != nil {
			s.errorf("获取节点源失败: %v", err)
			continue
		}

		decoded := TryDecodeNodeList(string(content))
		s.infof("⠹ 正在解析节点...")
		nodeLinks := s.extractNodeLinks(decoded)
		s.logNodeTypeStats(urlStr, nodeLinks)

		for _, link := range nodeLinks {
			allNodes = append(allNodes, map[string]interface{}{"url": link, "source_url": urlStr})
		}
	}
	return allNodes, nil
}

func (s *ConfigUpdateService) fetchURLContent(client *http.Client, url string) ([]byte, error) {
	maxRetries, delay := 3, 2*time.Second
	for i := 1; i <= maxRetries; i++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		if strings.Contains(url, "gist.githubusercontent.com") {
			req.Header.Set("Connection", "close")
		}

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			return io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
		}
		if resp != nil {
			resp.Body.Close()
		}

		if i < maxRetries {
			s.warnf("下载失败 (尝试 %d/%d): %v，%v 后重试", i, maxRetries, err, delay)
			time.Sleep(delay)
			delay *= 2
		}
	}
	return nil, fmt.Errorf("多次重试后失败")
}

func (s *ConfigUpdateService) logNodeTypeStats(url string, nodeLinks []string) {
	tc := make(map[string]int)
	for _, l := range nodeLinks {
		tc[strings.Split(l, ":")[0]]++
	}
	var parts []string
	for t, c := range tc {
		parts = append(parts, fmt.Sprintf("%s:%d", t, c))
	}
	s.infof("✓ 提取到 %d 个节点 (%s)", len(nodeLinks), strings.Join(parts, ", "))
}

func (s *ConfigUpdateService) extractNodeLinks(content string) []string {
	if yamlLinks := s.parseClashYAML(content); len(yamlLinks) > 0 {
		return yamlLinks
	}

	var links, invalidLinks []string
	// 优化：将原先 map[int]bool 的按字节哈希查询，替换为按索引的布尔数组 $O(1)$ 直接寻址，解决内存和性能瓶颈
	matched := make([]bool, len(content))

	// 1. VMess 轻量级宽容扫描 (允许base64内部有空格等)
	prefixes := []string{"vmess://", "vless://", "trojan://", "ss://", "ssr://", "hysteria://", "hysteria2://", "tuic://", "naive+https://", "naive://", "anytls://", "socks5://", "socks://", "http://", "https://", "wg://"}
	start := 0
	for {
		idx := strings.Index(content[start:], "vmess://")
		if idx == -1 {
			break
		}
		start += idx
		if matched[start] {
			start++
			continue
		}

		end := start + 8
		seenPadding := false
	scanLoop:
		for end < len(content) {
			for _, p := range prefixes {
				if strings.HasPrefix(content[end:], p) {
					break scanLoop
				}
			}
			ch := content[end]
			if ch == '#' {
				for end++; end < len(content) && content[end] > ' '; end++ {
				}
				break
			}
			if ch <= ' ' {
				if seenPadding {
					break
				}
				end++
				continue
			}
			if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '+' || ch == '/' || ch == '-' || ch == '_' || ch == '=' {
				if ch == '=' {
					seenPadding = true
				}
				end++
				continue
			}
			break
		}

		if end > start+8 {
			isOverlapped := false
			for i := start; i < end; i++ {
				if matched[i] {
					isOverlapped = true
					break
				}
			}
			if !isOverlapped {
				for i := start; i < end; i++ {
					matched[i] = true
				}
				matchStr := content[start:end]
				if s.isValidNodeLink(matchStr) {
					links = append(links, matchStr)
				}
			}
		}
		start = end
	}

	// 2. 统一正则扫描其他标准链接
	for _, match := range nodeLinkPattern.FindAllStringSubmatchIndex(content, -1) {
		start, end := match[2], match[3]
		if matched[start] || (strings.HasPrefix(content[start:end], "ss://") && start >= 3 && content[start-3:start] == "vme") {
			continue
		}
		isOverlapped := false
		for i := start; i < end; i++ {
			if matched[i] {
				isOverlapped = true
				break
			}
		}
		if isOverlapped {
			continue
		}
		for i := start; i < end; i++ {
			matched[i] = true
		}
		matchStr := content[start:end]
		if s.isValidNodeLink(matchStr) {
			links = append(links, matchStr)
		} else {
			invalidLinks = append(invalidLinks, matchStr)
		}
	}

	if len(invalidLinks) > 0 {
		limit := len(invalidLinks)
		if limit > 3 {
			limit = 3
		}
		s.debugf("发现 %d 个无效链接，示例: %v", len(invalidLinks), invalidLinks[:limit])
	}

	return s.uniqueLinks(links)
}

func (s *ConfigUpdateService) uniqueLinks(links []string) []string {
	m := make(map[string]bool)
	var res []string
	for _, l := range links {
		if !m[l] {
			m[l] = true
			res = append(res, l)
		}
	}
	return res
}

func (s *ConfigUpdateService) isValidNodeLink(link string) bool {
	link = strings.TrimSpace(link)
	parts := strings.SplitN(link, ":", 2)
	if len(parts) != 2 {
		return false
	}
	scheme, body := parts[0], strings.Split(link, "#")[0]

	switch scheme {
	case "ss":
		return strings.Contains(body, "@") && strings.Contains(strings.Split(strings.Split(body, "@")[1], "?")[0], ":")
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

func (s *ConfigUpdateService) deleteAutoImportedNodes() int64 {
	res := s.db.Where("is_manual = ?", false).Delete(&models.Node{})
	if res.Error != nil {
		s.errorf("删除自动导入节点失败: %v", res.Error)
		return 0
	}
	return res.RowsAffected
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

func (s *ConfigUpdateService) importNodesToDatabaseWithOrder(nodes []nodeWithOrder) importStats {
	var stats importStats
	seenKeys := make(map[string]bool)

	var existing []models.Node
	s.db.Where("is_manual = ?", false).Find(&existing)
	existingMap := make(map[string]*models.Node)
	for i := range existing {
		existingMap[s.generateNodeKey(existing[i].Type, existing[i].Name, existing[i].Config)] = &existing[i]
	}

	for _, item := range nodes {
		cfgJSON, _ := json.Marshal(item.node)
		cfgStr := string(cfgJSON)
		key := s.generateNodeKey(item.node.Type, item.node.Name, &cfgStr)

		if seenKeys[key] {
			stats.Skipped++
			continue
		}
		seenKeys[key] = true
		region := s.resolveRegion(item.node.Name, item.node.Server)

		if exist := existingMap[key]; exist != nil {
			exist.Config, exist.Status, exist.IsActive, exist.IsManual = &cfgStr, "online", true, false
			exist.OrderIndex, exist.SourceIndex, exist.Region, exist.Name = item.orderIndex, item.sourceIndex, region, item.node.Name
			if s.db.Save(exist).Error == nil {
				stats.Updated++
			}
		} else {
			if s.db.Where("type = ? AND name = ? AND is_manual = ?", item.node.Type, item.node.Name, true).First(&models.Node{}).Error == nil {
				stats.Skipped++
				continue
			}
			if s.db.Create(&models.Node{
				Name: item.node.Name, Type: item.node.Type, Status: "online", IsActive: true, IsManual: false,
				Config: &cfgStr, Region: region, OrderIndex: item.orderIndex, SourceIndex: item.sourceIndex,
			}).Error == nil {
				stats.Created++
			}
		}
	}
	return stats
}

func (s *ConfigUpdateService) GetSubscriptionContext(token, clientIP, userAgent string) *SubscriptionContext {
	ctx := &SubscriptionContext{Status: StatusNotFound}
	var sub models.Subscription

	if err := s.db.Where("subscription_url = ?", token).First(&sub).Error; err != nil {
		var reset models.SubscriptionReset
		if s.db.Where("old_subscription_url = ?", token).First(&reset).Error == nil {
			ctx.Status, ctx.ResetRecord = StatusOldAddress, &reset
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

	proxies, _ := s.fetchProxiesForUser(user, sub)
	ctx.Proxies = proxies

	if len(ctx.Proxies) == 0 && !sub.ExpireTime.IsZero() && sub.ExpireTime.Before(utils.GetBeijingTime()) {
		ctx.Status = StatusExpired
		return ctx
	}

	var devices int64
	s.db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", sub.ID, true).Count(&devices)
	ctx.CurrentDevices, ctx.DeviceLimit = int(devices), sub.DeviceLimit

	if sub.DeviceLimit == 0 || (sub.DeviceLimit > 0 && ctx.CurrentDevices >= sub.DeviceLimit && s.db.Where("subscription_id = ? AND ip_address = ? AND user_agent = ?", sub.ID, clientIP, userAgent).First(&models.Device{}).Error != nil) {
		ctx.Status = StatusDeviceOverLimit
		return ctx
	}

	ctx.Status = StatusNormal
	return ctx
}

func (s *ConfigUpdateService) fetchProxiesForUser(user models.User, sub models.Subscription) ([]*ProxyNode, error) {
	var proxies []*ProxyNode
	processed := make(map[string]bool)
	now := utils.GetBeijingTime()

	subExpired := !sub.ExpireTime.IsZero() && sub.ExpireTime.Before(now)
	specialExpired := (user.SpecialNodeExpiresAt.Valid && utils.ToBeijingTime(user.SpecialNodeExpiresAt.Time).Before(now)) || (user.SpecialNodeSubscriptionType != "special_only" && subExpired)

	s.appendCustomNodes(user.ID, now, specialExpired, &proxies, processed)
	if user.SpecialNodeSubscriptionType != "special_only" && !subExpired {
		s.appendSystemNodes(&proxies, processed)
	}
	return proxies, nil
}

func (s *ConfigUpdateService) appendCustomNodes(userID uint, now time.Time, isGlobalExpired bool, proxies *[]*ProxyNode, processed map[string]bool) {
	var nodes []models.CustomNode
	s.db.Joins("JOIN user_custom_nodes ON user_custom_nodes.custom_node_id = custom_nodes.id").
		Where("user_custom_nodes.user_id = ? AND custom_nodes.is_active = ?", userID, true).Find(&nodes)

	for _, cn := range nodes {
		if cn.Status == "timeout" || (cn.ExpireTime != nil && utils.ToBeijingTime(*cn.ExpireTime).Before(now)) || (cn.FollowUserExpire && isGlobalExpired) {
			continue
		}
		var proxy ProxyNode
		if json.Unmarshal([]byte(cn.Config), &proxy) == nil {
			proxy.Name = cn.DisplayName
			if proxy.Name == "" {
				proxy.Name = "专线-" + cn.Name
			}
			key := fmt.Sprintf("%s:%s:%d", proxy.Type, proxy.Server, proxy.Port)
			if !processed[key] {
				processed[key] = true
				*proxies = append(*proxies, &proxy)
			}
		}
	}
}

func (s *ConfigUpdateService) appendSystemNodes(proxies *[]*ProxyNode, processed map[string]bool) {
	cache := &CacheService{}
	if cached, ok := cache.GetSystemNodesCache(); ok {
		for _, n := range cached {
			if key := fmt.Sprintf("%s:%s:%d", n.Type, n.Server, n.Port); !processed[key] {
				processed[key] = true
				*proxies = append(*proxies, n)
			}
		}
		return
	}

	var nodes []models.Node
	s.db.Where("is_active = ? AND status != ?", true, "timeout").Find(&nodes)
	var sysNodes []*ProxyNode

	for _, n := range nodes {
		if n.Config == nil || *n.Config == "" {
			continue
		}
		var p ProxyNode
		if json.Unmarshal([]byte(*n.Config), &p) == nil {
			p.Name = n.Name
			if key := fmt.Sprintf("%s:%s:%d", p.Type, p.Server, p.Port); !processed[key] {
				processed[key] = true
				*proxies = append(*proxies, &p)
				sysNodes = append(sysNodes, &p)
			}
		}
	}
	if len(sysNodes) > 0 {
		go cache.SetSystemNodesCache(sysNodes)
	}
}

func (s *ConfigUpdateService) calculateCacheTTL(sub *models.Subscription) time.Duration {
	if !sub.ExpireTime.IsZero() {
		if utils.GetBeijingTime().After(sub.ExpireTime) {
			return 0
		}
		if sub.ExpireTime.Sub(utils.GetBeijingTime()) < 24*time.Hour {
			return time.Minute
		}
	}
	return 10 * time.Minute
}

func (s *ConfigUpdateService) GenerateClashConfig(token, clientIP, userAgent string) (string, error) {
	cache := &CacheService{}
	if cached, ok := cache.GetSubscriptionConfigCache(token, "clash"); ok {
		return cached, nil
	}

	ctx := s.GetSubscriptionContext(token, clientIP, userAgent)
	var nodes []*ProxyNode
	s.refreshSystemConfig()
	if ctx.Status != StatusNormal {
		nodes = s.generateErrorNodes(ctx.Status, ctx)
	} else {
		nodes = s.addInfoNodes(ctx.Proxies, ctx)
	}

	config := s.generateClashYAML(nodes, ctx)
	if ctx.Status == StatusNormal {
		if ttl := s.calculateCacheTTL(&ctx.Subscription); ttl > 0 {
			go cache.SetSubscriptionConfigCache(token, "clash", config, ttl)
		}
	}
	return config, nil
}

func (s *ConfigUpdateService) GenerateUniversalConfig(token, clientIP, userAgent, format string) (string, error) {
	cache, cacheFormat := &CacheService{}, "base64"
	if format == "ssr" {
		cacheFormat = "ssr"
	}
	if cached, ok := cache.GetSubscriptionConfigCache(token, cacheFormat); ok {
		return cached, nil
	}

	ctx := s.GetSubscriptionContext(token, clientIP, userAgent)
	var nodes []*ProxyNode
	s.refreshSystemConfig()
	if ctx.Status != StatusNormal {
		nodes = s.generateErrorNodes(ctx.Status, ctx)
	} else {
		nodes = s.addInfoNodes(ctx.Proxies, ctx)
	}

	var links []string
	for _, n := range nodes {
		link := s.nodeToLink(n)
		if format == "ssr" && n.Type == "ssr" {
			link = s.nodeToSSRLink(n)
		}
		if link != "" {
			links = append(links, link)
		}
	}

	config := base64.StdEncoding.EncodeToString([]byte(strings.Join(links, "\n")))
	if ctx.Status == StatusNormal {
		if ttl := s.calculateCacheTTL(&ctx.Subscription); ttl > 0 {
			go cache.SetSubscriptionConfigCache(token, cacheFormat, config, ttl)
		}
	}
	return config, nil
}

func (s *ConfigUpdateService) generateClashYAML(proxies []*ProxyNode, ctx *SubscriptionContext) string {
	var filtered []*ProxyNode
	var proxyNames []string
	used := make(map[string]bool)

	for _, p := range proxies {
		// 跳过不兼容的节点：SOCKS/SOCKS5 with WebSocket (Clash 不支持)
		if (p.Type == "socks" || p.Type == "socks5") && p.Network == "ws" {
			continue
		}

		if supportedClashTypes[p.Type] {
			// 统一 socks 类型为 socks5 (Clash 标准)
			if p.Type == "socks" {
				p.Type = "socks5"
			}

			orig, name, c := p.Name, p.Name, 1
			for used[name] {
				name = fmt.Sprintf("%s_%d", orig, c)
				c++
			}
			p.Name = name
			used[name] = true
			filtered = append(filtered, p)
			proxyNames = append(proxyNames, name)
		}
	}

	subName := "订阅异常"
	if ctx.Status == StatusNormal {
		exp := "无限期"
		if !ctx.Subscription.ExpireTime.IsZero() {
			exp = utils.FormatBeijingDate(ctx.Subscription.ExpireTime)
		}
		subName = "到期: " + exp
	}

	tplPath := filepath.Clean(filepath.Join("uploads", "config", "temp.yaml"))
	if cfg, err := s.getConfig(); err == nil {
		if targetDir, ok := cfg["target_dir"].(string); ok && targetDir != "" {
			tplPath = filepath.Clean(filepath.Join(targetDir, "temp.yaml"))
		}
	}

	if utils.IsWithinBaseDir(".", tplPath) {
		if data, err := os.ReadFile(tplPath); err == nil {
			var tpl map[string]interface{}
			if yaml.Unmarshal(data, &tpl) == nil {
				tpl["name"] = subName
				var plist []map[string]interface{}
				for _, p := range filtered {
					plist = append(plist, s.nodeToMap(p))
				}
				tpl["proxies"] = plist

				if grps, ok := tpl["proxy-groups"].([]interface{}); ok {
					s.updateProxyGroups(grps, proxyNames)
					tpl["proxy-groups"] = grps
				}
				if out, err := yaml.Marshal(tpl); err == nil {
					return unescapeUnicode(string(out))
				}
			}
		}
	}
	return s.generateDefaultClashYAML(filtered, proxyNames, subName)
}

func (s *ConfigUpdateService) updateProxyGroups(groups []interface{}, proxyNames []string) {
	groupNames := make(map[string]bool)
	for _, g := range groups {
		if m, ok := g.(map[string]interface{}); ok {
			if n, ok := m["name"].(string); ok {
				groupNames[n] = true
			}
		}
	}
	for _, g := range groups {
		m, ok := g.(map[string]interface{})
		if !ok {
			continue
		}
		t, _ := m["type"].(string)
		if t == "select" || t == "url-test" || t == "fallback" || t == "load-balance" {
			var exist []string
			if old, ok := m["proxies"].([]interface{}); ok {
				for _, p := range old {
					if ps, ok := p.(string); ok && (ps == "DIRECT" || ps == "REJECT" || groupNames[ps]) {
						exist = append(exist, ps)
					}
				}
			}
			if t == "select" {
				m["proxies"] = append(exist, proxyNames...)
			} else {
				m["proxies"] = proxyNames
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

func (s *ConfigUpdateService) generateDefaultClashYAML(proxies []*ProxyNode, proxyNames []string, subName string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("name: %s\nport: 7890\nsocks-port: 7891\nallow-lan: true\nmode: Rule\nlog-level: info\nexternal-controller: 127.0.0.1:9090\n\nproxies:\n", s.escapeYAMLString(subName)))

	for _, p := range proxies {
		b.WriteString(s.nodeToYAML(p, 2))
	}

	b.WriteString("\nproxy-groups:\n  - name: \"🚀 节点选择\"\n    type: select\n    proxies:\n      - \"♻️ 自动选择\"\n")
	for _, n := range proxyNames {
		b.WriteString(fmt.Sprintf("      - %s\n", s.escapeYAMLString(n)))
	}

	b.WriteString("  - name: \"♻️ 自动选择\"\n    type: url-test\n    url: http://www.gstatic.com/generate_204\n    interval: 300\n    tolerance: 50\n    proxies:\n")
	for _, n := range proxyNames {
		b.WriteString(fmt.Sprintf("      - %s\n", s.escapeYAMLString(n)))
	}

	b.WriteString("\nrules:\n  - DOMAIN-SUFFIX,local,DIRECT\n  - IP-CIDR,127.0.0.0/8,DIRECT\n  - IP-CIDR,172.16.0.0/12,DIRECT\n  - IP-CIDR,192.168.0.0/16,DIRECT\n  - GEOIP,CN,DIRECT\n  - MATCH,🚀 节点选择\n")
	return b.String()
}

func (s *ConfigUpdateService) addInfoNodes(proxies []*ProxyNode, ctx *SubscriptionContext) []*ProxyNode {
	exp := "无限期"
	if !ctx.Subscription.ExpireTime.IsZero() {
		exp = utils.FormatBeijingDate(ctx.Subscription.ExpireTime)
	}

	info := []*ProxyNode{
		s.createMessageNode("📢 官网: " + s.siteURL),
		s.createMessageNode("⏰ 到期: " + exp),
		s.createMessageNode(fmt.Sprintf("📱 设备: %d/%d", ctx.CurrentDevices, ctx.DeviceLimit)),
	}
	if s.supportQQ != "" {
		info = append(info, s.createMessageNode("💬 客服: "+s.supportQQ))
	}
	return append(info, proxies...)
}

func (s *ConfigUpdateService) generateErrorNodes(status SubscriptionStatus, ctx *SubscriptionContext) []*ProxyNode {
	reason, solution := "账户异常", "检测到账户异常，请联系管理员"
	switch status {
	case StatusExpired:
		reason, solution = "订阅已过期", "请前往官网续费 (过期时间: "+utils.FormatBeijingDate(ctx.Subscription.ExpireTime)+")"
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
	}

	qqMsg := "💬 客服: 请在系统设置中配置"
	if s.supportQQ != "" {
		qqMsg = "💬 客服: " + s.supportQQ
	}
	return []*ProxyNode{
		s.createMessageNode("📢 官网: " + s.siteURL),
		s.createMessageNode("❌ 原因: "+reason, "error"),
		s.createMessageNode("💡 解决: "+solution, "error"),
		s.createMessageNode(qqMsg, "error"),
	}
}

func (s *ConfigUpdateService) createMessageNode(name string, pwd ...string) *ProxyNode {
	p := "info"
	if len(pwd) > 0 {
		p = pwd[0]
	}
	return &ProxyNode{Name: name, Type: "ss", Server: "baidu.com", Port: 1234, Cipher: "aes-128-gcm", Password: p}
}

// 优化：泛型提取字典中的值
func optVal[T any](opts map[string]interface{}, key string) T {
	var zero T
	if opts == nil {
		return zero
	}
	if v, ok := opts[key].(T); ok {
		return v
	}
	return zero
}

func (s *ConfigUpdateService) nodeToYAML(node *ProxyNode, indent int) string {
	ind := strings.Repeat(" ", indent)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s- name: %s\n%s  type: %s\n%s  server: %s\n%s  port: %d\n", ind, s.escapeYAMLString(node.Name), ind, node.Type, ind, node.Server, ind, node.Port))

	m := s.nodeToMap(node)
	var keys []string
	for k := range m {
		if k != "name" && k != "type" && k != "server" && k != "port" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		s.writeYAMLValue(&b, ind+"  ", k, m[k], indent+2)
	}
	return b.String()
}

func (s *ConfigUpdateService) nodeToMap(n *ProxyNode) map[string]interface{} {
	res := map[string]interface{}{"name": n.Name, "type": n.Type, "server": n.Server, "port": n.Port}

	if n.Type == "ss" || n.Type == "trojan" || n.Type == "ssr" || n.Type == "tuic" || n.Type == "anytls" || strings.HasPrefix(n.Type, "hysteria") || strings.HasPrefix(n.Type, "socks") || n.Type == "http" {
		res["password"] = n.Password
	}

	switch n.Type {
	case "ss":
		if n.Cipher != "" {
			res["cipher"] = n.Cipher
		}
		if p := optVal[string](n.Options, "plugin"); p != "" {
			res["plugin"] = p
			if popt := optVal[map[string]interface{}](n.Options, "plugin-opts"); popt != nil {
				res["plugin-opts"] = popt
			}
		}
	case "vmess":
		if n.UUID != "" {
			res["uuid"] = n.UUID
		}
		res["alterId"], res["cipher"] = optVal[int](n.Options, "alterId"), "auto"
		if n.Cipher != "" {
			res["cipher"] = n.Cipher
		}
	case "vless", "tuic":
		if n.UUID != "" {
			res["uuid"] = n.UUID
		}
		if n.Type == "tuic" {
			if _, ok := n.Options["disable-sni"]; !ok {
				res["disable-sni"] = false
			}
			if _, ok := n.Options["reduce-rtt"]; !ok {
				res["reduce-rtt"] = false
			}
			res["request-timeout"] = 15000
			res["udp-relay-mode"] = "native"
			if cc := optVal[string](n.Options, "congestion_control"); cc != "" {
				res["congestion-controller"] = cc
				delete(n.Options, "congestion_control")
			} else if cc := optVal[string](n.Options, "congestion-controller"); cc != "" {
				res["congestion-controller"] = cc
			}
			if sni := optVal[string](n.Options, "servername"); sni != "" {
				res["sni"] = sni
				delete(n.Options, "servername")
			}
		}
	case "ssr":
		if n.Cipher != "" {
			res["cipher"] = n.Cipher
		}
	case "anytls":
		res["udp"] = n.UDP
		if sni := optVal[string](n.Options, "servername"); sni != "" {
			res["sni"] = sni
			delete(n.Options, "servername")
		}
	case "hysteria", "hysteria2":
		if res["password"] == "" {
			res["password"] = optVal[string](n.Options, "auth")
		}
		if n.Type == "hysteria2" {
			res["auth"] = res["password"]
		}
	case "socks", "socks5", "http":
		if n.UUID != "" {
			res["username"] = n.UUID
		}
	}

	if n.TLS || n.Type == "tuic" || n.Type == "anytls" {
		res["tls"] = true
	}
	if n.Network != "" && n.Network != "tcp" {
		res["network"] = n.Network
	}
	if n.UDP {
		res["udp"] = true
	}

	for k, v := range n.Options {
		if k != "alterId" || n.Type != "vmess" {
			res[k] = v
		}
	}
	return res
}

func (s *ConfigUpdateService) writeYAMLValue(b *strings.Builder, ind, key string, val interface{}, lvl int) {
	ek := s.escapeYAMLString(key)
	switch v := val.(type) {
	case map[string]interface{}:
		b.WriteString(fmt.Sprintf("%s%s:\n", ind, ek))
		s.writeMapContent(b, ind+"  ", v, key, lvl+1)
	case []interface{}:
		b.WriteString(fmt.Sprintf("%s%s:\n", ind, ek))
		for _, item := range v {
			b.WriteString(fmt.Sprintf("%s  - %s\n", ind, s.formatYAMLInline(item)))
		}
	case []string:
		b.WriteString(fmt.Sprintf("%s%s:\n", ind, ek))
		for _, item := range v {
			b.WriteString(fmt.Sprintf("%s  - %s\n", ind, s.escapeYAMLString(item)))
		}
	default:
		b.WriteString(fmt.Sprintf("%s%s: %s\n", ind, ek, s.formatYAMLInline(v)))
	}
}

func (s *ConfigUpdateService) writeMapContent(b *strings.Builder, ind string, v map[string]interface{}, pk string, lvl int) {
	if pk == "http-opts" {
		if m, ok := v["method"].(string); ok {
			b.WriteString(fmt.Sprintf("%smethod: %s\n", ind, m))
		}
		if p, ok := v["path"]; ok {
			s.writeYAMLList(b, ind, "path", p)
		}
		if h, ok := v["headers"].(map[string]interface{}); ok {
			b.WriteString(ind + "headers:\n")
			for hk, hv := range h {
				s.writeYAMLList(b, ind+"  ", hk, hv)
			}
		}
		return
	}
	var keys []string
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if sm, ok := v[k].(map[string]string); ok {
			nm := make(map[string]interface{})
			for mk, mv := range sm {
				nm[mk] = mv
			}
			s.writeYAMLValue(b, ind, k, nm, lvl+1)
		} else {
			s.writeYAMLValue(b, ind, k, v[k], lvl+1)
		}
	}
}

func (s *ConfigUpdateService) writeYAMLList(b *strings.Builder, ind, key string, val interface{}) {
	b.WriteString(fmt.Sprintf("%s%s:\n", ind, s.escapeYAMLString(key)))
	w := func(i interface{}) { b.WriteString(fmt.Sprintf("%s  - %s\n", ind, s.formatYAMLInline(i))) }
	switch v := val.(type) {
	case string:
		w(v)
	case []string:
		for _, i := range v {
			w(i)
		}
	case []interface{}:
		for _, i := range v {
			w(i)
		}
	}
}

func (s *ConfigUpdateService) formatYAMLInline(v interface{}) string {
	if str, ok := v.(string); ok {
		return s.escapeYAMLString(str)
	}
	return s.escapeYAMLString(fmt.Sprintf("%v", v))
}

func (s *ConfigUpdateService) escapeYAMLString(str string) string {
	if str == "" {
		return `""`
	}
	if strings.ContainsAny(str, ":\"'\n\r\t#@&*?|>!%`[]{},\x00") || strings.HasPrefix(str, " ") || strings.HasSuffix(str, " ") {
		return fmt.Sprintf(`"%s"`, strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(str, "\\", "\\\\"), "\"", "\\\""), "\n", "\\n"))
	}
	return str
}

func (s *ConfigUpdateService) NodeToLink(node *ProxyNode) string { return s.nodeToLink(node) }

func (s *ConfigUpdateService) nodeToLink(n *ProxyNode) string {
	switch n.Type {
	case "vmess":
		return s.vmessToLink(n)
	case "ss":
		return s.shadowsocksToLink(n)
	case "ssr":
		return s.nodeToSSRLink(n)
	case "vless", "trojan", "hysteria", "hysteria2", "tuic", "naive", "anytls":
		scheme, user, pwd := n.Type, n.UUID, n.Password
		if n.Type == "trojan" || n.Type == "hysteria2" || n.Type == "anytls" {
			user, pwd = pwd, ""
		} else if n.Type == "naive" {
			scheme = "naive+https"
		} else if n.Type == "hysteria" {
			user = ""
		}
		return s.buildStandardNodeURL(scheme, user, pwd, n.Server, n.Port, n.Name, s.getQueryFromOptions(n))
	case "socks", "socks5":
		sc := "socks5"
		if n.Type == "socks" {
			sc = "socks"
		}
		return s.buildStandardNodeURL(sc, n.UUID, n.Password, n.Server, n.Port, n.Name, nil)
	case "http":
		sc := "http"
		if n.TLS {
			sc = "https"
		}
		return s.buildStandardNodeURL(sc, n.UUID, n.Password, n.Server, n.Port, n.Name, s.getQueryFromOptions(n))
	}
	return ""
}

func (s *ConfigUpdateService) buildStandardNodeURL(sch, usr, pwd, hst string, prt int, frag string, q url.Values) string {
	u := &url.URL{Scheme: sch, Host: fmt.Sprintf("%s:%d", hst, prt), Fragment: frag}
	if usr != "" {
		if pwd != "" {
			u.User = url.UserPassword(usr, pwd)
		} else {
			u.User = url.User(usr)
		}
	} else if pwd != "" {
		u.User = url.User(pwd)
	}
	if q != nil && len(q) > 0 {
		u.RawQuery = q.Encode()
	}
	return u.String()
}

func (s *ConfigUpdateService) getQueryFromOptions(n *ProxyNode) url.Values {
	q := url.Values{}
	if n.Options == nil {
		return q
	}

	if sni := optVal[string](n.Options, "servername"); sni != "" {
		q.Set("sni", sni)
	}
	if peer := optVal[string](n.Options, "peer"); peer != "" && (q.Get("sni") == "" || n.Type == "anytls") {
		q.Set("peer", peer)
	}
	if optVal[bool](n.Options, "skip-cert-verify") {
		q.Set("insecure", "1")
		q.Set("allow_insecure", "1")
	}
	if fp := optVal[string](n.Options, "client-fingerprint"); fp != "" {
		q.Set("fp", fp)
	}
	if alpn := n.Options["alpn"]; alpn != nil {
		if strs, ok := alpn.([]string); ok && len(strs) > 0 {
			q.Set("alpn", strings.Join(strs, ","))
		} else if infs, ok := alpn.([]interface{}); ok {
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

	switch n.Type {
	case "vless", "trojan":
		if n.Network != "" {
			q.Set("type", n.Network)
		}
		if ws := optVal[map[string]interface{}](n.Options, "ws-opts"); ws != nil {
			if path := optVal[string](ws, "path"); path != "" {
				q.Set("path", path)
			}
			if hdrs := optVal[map[string]interface{}](ws, "headers"); hdrs != nil {
				if h := optVal[string](hdrs, "Host"); h != "" {
					q.Set("host", h)
				}
			} else if hm, ok := ws["headers"].(map[string]string); ok && hm["Host"] != "" {
				q.Set("host", hm["Host"])
			}
		}
		if grpc := optVal[map[string]interface{}](n.Options, "grpc-opts"); grpc != nil {
			if sn := optVal[string](grpc, "grpc-service-name"); sn != "" {
				q.Set("serviceName", sn)
			}
		}
		if h2 := optVal[map[string]interface{}](n.Options, "h2-opts"); h2 != nil {
			if p := optVal[string](h2, "path"); p != "" {
				q.Set("path", p)
			}
			if hs, ok := h2["host"].([]string); ok && len(hs) > 0 {
				q.Set("host", hs[0])
			}
		}
		if ht := optVal[string](n.Options, "header-type"); ht != "" {
			q.Set("headerType", ht)
		}
		if n.Type == "vless" {
			if n.TLS {
				if real := optVal[map[string]interface{}](n.Options, "reality-opts"); real != nil {
					q.Set("security", "reality")
					if pbk := optVal[string](real, "public-key"); pbk != "" {
						q.Set("pbk", pbk)
					}
					if sid := optVal[string](real, "short-id"); sid != "" {
						q.Set("sid", sid)
					}
					if pqv := optVal[string](real, "pqv"); pqv != "" {
						q.Set("pqv", pqv)
					}
				} else {
					q.Set("security", "tls")
				}
			}
			if f := optVal[string](n.Options, "flow"); f != "" {
				q.Set("flow", f)
			}
			if e := optVal[string](n.Options, "encryption"); e != "" {
				q.Set("encryption", e)
			}
		}
	case "hysteria", "hysteria2":
		if a := optVal[string](n.Options, "auth"); a != "" {
			q.Set("auth", a)
		}
		if up := optVal[string](n.Options, "up"); up != "" {
			t := strings.TrimSuffix(up, " mbps")
			q.Set("upmbps", t)
			q.Set("mbpsUp", t)
		}
		if dn := optVal[string](n.Options, "down"); dn != "" {
			t := strings.TrimSuffix(dn, " mbps")
			q.Set("downmbps", t)
			q.Set("mbpsDown", t)
		}
	case "tuic":
		if cc := optVal[string](n.Options, "congestion_control"); cc != "" {
			q.Set("congestion_control", cc)
		}
		if mode := optVal[string](n.Options, "udp_relay_mode"); mode != "" {
			q.Set("udp_relay_mode", mode)
		}
	case "naive":
		if optVal[bool](n.Options, "padding") {
			q.Set("padding", "true")
		}
	}
	return q
}

func (s *ConfigUpdateService) vmessToLink(p *ProxyNode) string {
	net, obfs := p.Network, "none"
	if net == "http" {
		net, obfs = "tcp", "http"
	}
	d := map[string]interface{}{
		"v": "2", "ps": p.Name, "add": p.Server, "port": p.Port, "id": p.UUID, "net": net, "type": obfs,
		"tls": "", "sni": "", "host": "", "path": "", "aid": 0, "scy": "auto",
	}
	if p.TLS {
		d["tls"] = "tls"
	}
	if p.Options != nil {
		if aid, ok := p.Options["alterId"]; ok {
			d["aid"] = aid
		}
		if c := optVal[string](p.Options, "cipher"); c != "" {
			d["scy"] = c
		}
		if sni := optVal[string](p.Options, "servername"); sni != "" {
			d["sni"] = sni
		}
		if optVal[bool](p.Options, "skip-cert-verify") {
			d["insecure"] = "1"
		}
		if ws := optVal[map[string]interface{}](p.Options, "ws-opts"); ws != nil {
			if path := optVal[string](ws, "path"); path != "" {
				d["path"] = path
			}
			if hdrs := optVal[map[string]interface{}](ws, "headers"); hdrs != nil {
				if h := optVal[string](hdrs, "Host"); h != "" {
					d["host"] = h
				}
			}
		}
		if http := optVal[map[string]interface{}](p.Options, "http-opts"); http != nil {
			if ps, ok := http["path"].([]string); ok && len(ps) > 0 {
				d["path"] = ps[0]
			}
			if hdrs := optVal[map[string]interface{}](http, "headers"); hdrs != nil {
				if hs, ok := hdrs["Host"].([]string); ok && len(hs) > 0 {
					d["host"] = hs[0]
				}
			}
		}
		if h2 := optVal[map[string]interface{}](p.Options, "h2-opts"); h2 != nil {
			if path := optVal[string](h2, "path"); path != "" {
				d["path"] = path
			}
			if hs, ok := h2["host"].([]string); ok && len(hs) > 0 {
				d["host"] = hs[0]
			}
		}
		if grpc := optVal[map[string]interface{}](p.Options, "grpc-opts"); grpc != nil {
			if sn := optVal[string](grpc, "grpc-service-name"); sn != "" {
				d["path"] = sn
			}
		}
	}
	jd, _ := json.Marshal(d)
	return "vmess://" + base64.StdEncoding.EncodeToString(jd)
}

func (s *ConfigUpdateService) shadowsocksToLink(p *ProxyNode) string {
	enc := base64.StdEncoding.EncodeToString([]byte(p.Cipher + ":" + p.Password))
	var q url.Values
	if pn := optVal[string](p.Options, "plugin"); pn != "" {
		q, ps := url.Values{}, pn
		if pn == "obfs" {
			ps = "obfs-local"
		}
		if po := optVal[map[string]interface{}](p.Options, "plugin-opts"); po != nil {
			if m := optVal[string](po, "mode"); m != "" {
				ps += ";obfs=" + m
			}
			if h := optVal[string](po, "host"); h != "" {
				ps += ";obfs-host=" + h
			}
			if path := optVal[string](po, "path"); path != "" {
				ps += ";obfs-uri=" + path
			}
			if optVal[bool](po, "tls") {
				ps += ";tls"
			}
		}
		q.Set("plugin", ps)
	}
	return s.buildStandardNodeURL("ss", enc, "", p.Server, p.Port, p.Name, q)
}

func (s *ConfigUpdateService) nodeToSSRLink(n *ProxyNode) string {
	gs := func(k, d string) string {
		if v := optVal[string](n.Options, k); v != "" {
			return v
		}
		return d
	}
	str := fmt.Sprintf("%s:%d:%s:%s:%s:%s/?obfsparam=%s&protoparam=%s&remarks=%s&group=%s",
		n.Server, n.Port, gs("protocol", "origin"), n.Cipher, gs("obfs", "plain"),
		base64.RawURLEncoding.EncodeToString([]byte(n.Password)),
		base64.RawURLEncoding.EncodeToString([]byte(gs("obfs-param", ""))),
		base64.RawURLEncoding.EncodeToString([]byte(gs("protocol-param", ""))),
		base64.RawURLEncoding.EncodeToString([]byte(n.Name)),
		base64.RawURLEncoding.EncodeToString([]byte("GoWeb")))
	return "ssr://" + base64.RawURLEncoding.EncodeToString([]byte(str))
}

func unescapeUnicode(s string) string {
	return regexp.MustCompile(`\\U([0-9A-Fa-f]{8})`).ReplaceAllStringFunc(s, func(m string) string {
		if cp, err := strconv.ParseInt(m[2:], 16, 64); err == nil {
			return string(utils.MustSafeInt64ToRune(cp))
		}
		return m
	})
}

func (s *ConfigUpdateService) logSeparator() {
	s.log("INFO", "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func (s *ConfigUpdateService) logSection(icon, title string) {
	s.logSeparator()
	s.log("INFO", fmt.Sprintf("%s %s", icon, title))
	s.logSeparator()
}

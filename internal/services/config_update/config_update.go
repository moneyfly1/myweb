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
	"strings"
	"sync"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

// ConfigUpdateService 配置更新服务
type ConfigUpdateService struct {
	db           *gorm.DB
	isRunning    bool
	runningMutex sync.Mutex
}

// NewConfigUpdateService 创建配置更新服务
func NewConfigUpdateService() *ConfigUpdateService {
	return &ConfigUpdateService{
		db: database.GetDB(),
	}
}

// FetchNodesFromURLs 从URL列表获取节点
func (s *ConfigUpdateService) FetchNodesFromURLs(urls []string) ([]map[string]interface{}, error) {
	var allNodes []map[string]interface{}

	for i, url := range urls {
		if utils.AppLogger != nil {
			utils.AppLogger.Info("正在下载节点源 [%d/%d]: %s", i+1, len(urls), url)
		}

		// 下载内容
		resp, err := http.Get(url)
		if err != nil {
			if utils.AppLogger != nil {
				utils.AppLogger.Error("下载失败: %v", err)
			}
			continue
		}
		defer resp.Body.Close()

		content, err := io.ReadAll(resp.Body)
		if err != nil {
			if utils.AppLogger != nil {
				utils.AppLogger.Error("读取内容失败: %v", err)
			}
			continue
		}

		// 尝试 Base64 解码
		decoded := s.tryBase64Decode(string(content))

		// 提取节点链接
		nodeLinks := s.extractNodeLinks(decoded)
		if utils.AppLogger != nil {
			utils.AppLogger.Info("从 %s 提取到 %d 个节点链接", url, len(nodeLinks))
		}

		for _, link := range nodeLinks {
			allNodes = append(allNodes, map[string]interface{}{
				"url":        link,
				"source_url": url,
			})
		}
	}

	return allNodes, nil
}

// tryBase64Decode 尝试 Base64 解码
func (s *ConfigUpdateService) tryBase64Decode(text string) string {
	// 清理文本
	cleanText := strings.ReplaceAll(text, " ", "")
	cleanText = strings.ReplaceAll(cleanText, "\n", "")
	cleanText = strings.ReplaceAll(cleanText, "\r", "")
	cleanText = strings.ReplaceAll(cleanText, "-", "+")
	cleanText = strings.ReplaceAll(cleanText, "_", "/")

	// 补全 padding
	if len(cleanText)%4 != 0 {
		cleanText += strings.Repeat("=", 4-len(cleanText)%4)
	}

	decoded, err := base64.StdEncoding.DecodeString(cleanText)
	if err != nil {
		return text
	}

	return string(decoded)
}

// 预编译正则表达式以提升性能
var nodeLinkPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(vmess://[^\s]+)`),
	regexp.MustCompile(`(vless://[^\s]+)`),
	regexp.MustCompile(`(trojan://[^\s]+)`),
	regexp.MustCompile(`(ss://[^\s]+)`),
	regexp.MustCompile(`(ssr://[^\s]+)`),
	regexp.MustCompile(`(hysteria://[^\s]+)`),
	regexp.MustCompile(`(hysteria2://[^\s]+)`),
	regexp.MustCompile(`(tuic://[^\s]+)`),
	regexp.MustCompile(`(wireguard://[^\s]+)`),
	regexp.MustCompile(`(http://[^\s]+)`),
	regexp.MustCompile(`(https://[^\s]+)`),
}

// extractNodeLinks 提取节点链接
func (s *ConfigUpdateService) extractNodeLinks(content string) []string {
	var links []string

	for _, re := range nodeLinkPatterns {
		matches := re.FindAllString(content, -1)
		links = append(links, matches...)
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

// GenerateClashConfig 生成 Clash 配置
func (s *ConfigUpdateService) GenerateClashConfig(userID uint, subscriptionURL string) (string, error) {
	// 获取节点
	proxies, subscription, user, isExpired, isInactive, isDeviceOverLimit, currentDevices, deviceLimit, err := s.getNodesForSubscription(userID, subscriptionURL)
	if err != nil {
		// 如果是“没有可用节点”错误，我们仍然尝试生成基础配置（带提醒）
		if err.Error() == "没有可用的节点" {
			// 获取订阅信息以便生成提醒节点
			var sub models.Subscription
			s.db.Where("subscription_url = ?", subscriptionURL).First(&sub)
			var u models.User
			s.db.First(&u, userID)

			proxies = s.addInfoAndReminderNodes([]*ProxyNode{}, sub, u, isExpired, isInactive, isDeviceOverLimit, currentDevices, deviceLimit)
			return s.generateClashYAML(proxies), nil
		}
		return "", err
	}

	// 添加信息节点和提醒节点
	proxies = s.addInfoAndReminderNodes(proxies, subscription, user, isExpired, isInactive, isDeviceOverLimit, currentDevices, deviceLimit)

	// 生成 Clash YAML 配置
	return s.generateClashYAML(proxies), nil
}

// generateClashYAML 生成 Clash YAML 配置
func (s *ConfigUpdateService) generateClashYAML(proxies []*ProxyNode) string {
	var builder strings.Builder

	// 写入基础配置
	builder.WriteString("port: 7890\n")
	builder.WriteString("socks-port: 7891\n")
	builder.WriteString("allow-lan: true\n")
	builder.WriteString("mode: Rule\n")
	builder.WriteString("log-level: info\n")
	builder.WriteString("external-controller: 127.0.0.1:9090\n\n")

	// 写入代理节点
	builder.WriteString("proxies:\n")
	for _, proxy := range proxies {
		builder.WriteString(s.nodeToYAML(proxy, 2))
	}

	// 生成代理名称列表
	var proxyNames []string
	for _, proxy := range proxies {
		proxyNames = append(proxyNames, proxy.Name)
	}

	// 写入代理组
	builder.WriteString("\nproxy-groups:\n")
	builder.WriteString("  - name: 🚀 节点选择\n")
	builder.WriteString("    type: select\n")
	builder.WriteString("    proxies:\n")
	builder.WriteString("      - ♻️ 自动选择\n")
	builder.WriteString("      - DIRECT\n")
	for _, name := range proxyNames {
		builder.WriteString(fmt.Sprintf("      - %s\n", name))
	}

	builder.WriteString("  - name: ♻️ 自动选择\n")
	builder.WriteString("    type: url-test\n")
	builder.WriteString("    url: http://www.gstatic.com/generate_204\n")
	builder.WriteString("    interval: 300\n")
	builder.WriteString("    tolerance: 50\n")
	builder.WriteString("    proxies:\n")
	for _, name := range proxyNames {
		builder.WriteString(fmt.Sprintf("      - %s\n", name))
	}

	builder.WriteString("  - name: 📢 失败切换\n")
	builder.WriteString("    type: fallback\n")
	builder.WriteString("    url: http://www.gstatic.com/generate_204\n")
	builder.WriteString("    interval: 300\n")
	builder.WriteString("    proxies:\n")
	for _, name := range proxyNames {
		builder.WriteString(fmt.Sprintf("      - %s\n", name))
	}

	// 写入规则
	builder.WriteString("\nrules:\n")
	builder.WriteString("  - DOMAIN-SUFFIX,local,DIRECT\n")
	builder.WriteString("  - IP-CIDR,127.0.0.0/8,DIRECT\n")
	builder.WriteString("  - IP-CIDR,172.16.0.0/12,DIRECT\n")
	builder.WriteString("  - IP-CIDR,192.168.0.0/16,DIRECT\n")
	builder.WriteString("  - IP-CIDR,10.0.0.0/8,DIRECT\n")
	builder.WriteString("  - GEOIP,CN,DIRECT\n")
	builder.WriteString("  - MATCH,🚀 节点选择\n")

	return builder.String()
}

// nodeToYAML 将节点转换为 YAML 格式
func (s *ConfigUpdateService) nodeToYAML(node *ProxyNode, indent int) string {
	indentStr := strings.Repeat(" ", indent)
	var builder strings.Builder

	// 信息节点（direct 类型）特殊处理
	if node.Type == "direct" && node.Server == "127.0.0.1" {
		// 对于信息节点，创建一个不可用的节点，但名称会显示信息
		builder.WriteString(fmt.Sprintf("%s- name: %s\n", indentStr, node.Name))
		builder.WriteString(fmt.Sprintf("%s  type: direct\n", indentStr))
		return builder.String()
	}

	builder.WriteString(fmt.Sprintf("%s- name: %s\n", indentStr, node.Name))
	builder.WriteString(fmt.Sprintf("%s  type: %s\n", indentStr, node.Type))
	builder.WriteString(fmt.Sprintf("%s  server: %s\n", indentStr, node.Server))
	builder.WriteString(fmt.Sprintf("%s  port: %d\n", indentStr, node.Port))

	if node.UUID != "" {
		builder.WriteString(fmt.Sprintf("%s  uuid: %s\n", indentStr, node.UUID))
	}
	if node.Password != "" {
		builder.WriteString(fmt.Sprintf("%s  password: %s\n", indentStr, node.Password))
	}
	if node.Cipher != "" {
		builder.WriteString(fmt.Sprintf("%s  cipher: %s\n", indentStr, node.Cipher))
	}
	if node.Network != "" && node.Network != "tcp" {
		builder.WriteString(fmt.Sprintf("%s  network: %s\n", indentStr, node.Network))
	}
	if node.TLS {
		builder.WriteString(fmt.Sprintf("%s  tls: true\n", indentStr))
	}
	if node.UDP {
		builder.WriteString(fmt.Sprintf("%s  udp: true\n", indentStr))
	}

	// 写入额外选项
	for key, value := range node.Options {
		builder.WriteString(fmt.Sprintf("%s  %s: %v\n", indentStr, key, value))
	}

	return builder.String()
}

// UpdateSubscriptionConfig 更新订阅配置
func (s *ConfigUpdateService) UpdateSubscriptionConfig(subscriptionURL string) error {
	// 获取订阅信息
	var subscription models.Subscription
	if err := s.db.Where("subscription_url = ?", subscriptionURL).First(&subscription).Error; err != nil {
		return fmt.Errorf("订阅不存在: %v", err)
	}

	// 生成新配置
	config, err := s.GenerateClashConfig(subscription.UserID, subscriptionURL)
	if err != nil {
		return fmt.Errorf("生成配置失败: %v", err)
	}

	// 配置是实时生成的，这里主要是验证配置生成是否成功
	if utils.AppLogger != nil {
		utils.AppLogger.Info("订阅配置已更新: %s, 配置长度: %d 字符", subscriptionURL, len(config))
	}

	return nil
}

// RunUpdateTask 执行配置更新任务
func (s *ConfigUpdateService) RunUpdateTask() error {
	s.runningMutex.Lock()
	if s.isRunning {
		s.runningMutex.Unlock()
		s.addLog("任务已在运行中", "warning")
		return fmt.Errorf("任务已在运行中")
	}
	s.isRunning = true
	s.runningMutex.Unlock()

	defer func() {
		s.runningMutex.Lock()
		s.isRunning = false
		s.runningMutex.Unlock()
	}()

	s.addLog("开始执行配置更新任务", "info")

	// 获取配置
	config, err := s.getConfig()
	if err != nil {
		s.addLog(fmt.Sprintf("获取配置失败: %v", err), "error")
		return err
	}

	urls := config["urls"].([]string)
	if len(urls) == 0 {
		s.addLog("未配置节点源URL", "error")
		return fmt.Errorf("未配置节点源URL")
	}

	// 1. 获取节点
	s.addLog(fmt.Sprintf("开始下载节点，共 %d 个源", len(urls)), "info")
	nodes, err := s.FetchNodesFromURLs(urls)
	if err != nil {
		s.addLog(fmt.Sprintf("获取节点失败: %v", err), "error")
		return err
	}

	if len(nodes) == 0 {
		s.addLog("未获取到有效节点", "error")
		return fmt.Errorf("未获取到有效节点")
	}

	s.addLog(fmt.Sprintf("成功获取 %d 个节点", len(nodes)), "success")

	// 2. 生成配置
	targetDir := config["target_dir"].(string)
	if !filepath.IsAbs(targetDir) {
		// 相对路径，转换为绝对路径
		wd, _ := os.Getwd()
		targetDir = filepath.Join(wd, strings.TrimPrefix(targetDir, "./"))
	}

	// 确保目录存在
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		s.addLog(fmt.Sprintf("创建目录失败: %v", err), "error")
		return err
	}

	filterKeywords := []string{}
	if keywords, ok := config["filter_keywords"].([]string); ok {
		filterKeywords = keywords
	}

	// 解析节点为代理节点
	var proxies []*ProxyNode
	seenKeys := make(map[string]bool)
	nameCounter := make(map[string]int)

	for _, nodeInfo := range nodes {
		link, ok := nodeInfo["url"].(string)
		if !ok {
			continue
		}

		node, err := ParseNodeLink(link)
		if err != nil {
			continue
		}

		// 过滤关键词
		if len(filterKeywords) > 0 {
			shouldSkip := false
			for _, keyword := range filterKeywords {
				if strings.Contains(node.Name, keyword) {
					shouldSkip = true
					break
				}
			}
			if shouldSkip {
				continue
			}
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

		// 处理名称重复
		if count, exists := nameCounter[node.Name]; exists {
			nameCounter[node.Name] = count + 1
			node.Name = fmt.Sprintf("%s-%d", node.Name, count+1)
		} else {
			nameCounter[node.Name] = 0
		}

		proxies = append(proxies, node)
	}

	s.addLog(fmt.Sprintf("解析后有效节点数: %d", len(proxies)), "info")

	// 生成 V2Ray 配置（Base64）
	v2rayFileName := config["v2ray_file"].(string)
	v2rayFilePath := filepath.Join(targetDir, v2rayFileName)
	v2rayContent := s.generateV2RayConfig(nodes)
	if err := os.WriteFile(v2rayFilePath, []byte(v2rayContent), 0644); err != nil {
		s.addLog(fmt.Sprintf("保存V2Ray配置失败: %v", err), "error")
	} else {
		s.addLog(fmt.Sprintf("V2Ray配置已保存: %s", v2rayFilePath), "success")
		s.saveConfigToDB("v2ray_config", "v2ray", v2rayContent)
	}

	// 生成 Clash 配置
	clashFileName := config["clash_file"].(string)
	clashFilePath := filepath.Join(targetDir, clashFileName)
	clashContent := s.generateClashYAML(proxies)
	if err := os.WriteFile(clashFilePath, []byte(clashContent), 0644); err != nil {
		s.addLog(fmt.Sprintf("保存Clash配置失败: %v", err), "error")
	} else {
		s.addLog(fmt.Sprintf("Clash配置已保存: %s", clashFilePath), "success")
		s.saveConfigToDB("clash_config", "clash", clashContent)
	}

	// 导入节点到数据库的 nodes 表
	s.addLog(fmt.Sprintf("准备导入 %d 个节点到数据库", len(proxies)), "info")
	importedCount := s.importNodesToDatabase(proxies)
	s.addLog(fmt.Sprintf("成功导入节点到数据库: %d 个", importedCount), "success")
	
	// 验证数据库中的节点数量
	var totalNodes int64
	s.db.Model(&models.Node{}).Where("is_manual = ?", false).Count(&totalNodes)
	s.addLog(fmt.Sprintf("数据库中非手动节点总数: %d", totalNodes), "info")

	// 更新最后更新时间
	s.updateLastUpdateTime()

	s.addLog(fmt.Sprintf("✅ 配置更新任务完成！下载节点数: %d, 最终节点数: %d, 数据库节点数: %d", len(nodes), len(proxies), importedCount), "success")

	return nil
}

// generateV2RayConfig 生成 V2Ray 配置（Base64编码）
func (s *ConfigUpdateService) generateV2RayConfig(nodes []map[string]interface{}) string {
	var links []string
	for _, nodeInfo := range nodes {
		if link, ok := nodeInfo["url"].(string); ok {
			links = append(links, link)
		}
	}
	content := strings.Join(links, "\n")
	return base64.StdEncoding.EncodeToString([]byte(content))
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

	for _, config := range configs {
		key := config.Key
		value := config.Value

		switch key {
		case "urls", "node_source_urls":
			urls := strings.Split(value, "\n")
			filtered := []string{}
			for _, url := range urls {
				url = strings.TrimSpace(url)
				if url != "" {
					filtered = append(filtered, url)
				}
			}
			result["urls"] = filtered
		case "filter_keywords":
			keywords := strings.Split(value, "\n")
			filtered := []string{}
			for _, keyword := range keywords {
				keyword = strings.TrimSpace(keyword)
				if keyword != "" {
					filtered = append(filtered, keyword)
				}
			}
			result["filter_keywords"] = filtered
		case "enable_schedule":
			result[key] = value == "true" || value == "1"
		case "schedule_interval":
			var interval int
			fmt.Sscanf(value, "%d", &interval)
			if interval == 0 {
				interval = 3600
			}
			result[key] = interval
		default:
			result[key] = value
		}
	}

	return result, nil
}

// addLog 添加日志
func (s *ConfigUpdateService) addLog(message string, level string) {
	logEntry := map[string]interface{}{
		"timestamp": utils.GetBeijingTime().Format("2006-01-02T15:04:05"),
		"level":     level,
		"message":   message,
	}

	var config models.SystemConfig
	err := s.db.Where("key = ?", "config_update_logs").First(&config).Error

	var logs []map[string]interface{}
	if err == nil && config.Value != "" {
		json.Unmarshal([]byte(config.Value), &logs)
	}

	logs = append(logs, logEntry)
	// 只保留最近100条
	if len(logs) > 100 {
		logs = logs[len(logs)-100:]
	}

	logsJSON, _ := json.Marshal(logs)

	if err != nil {
		// 创建新记录
		config = models.SystemConfig{
			Key:         "config_update_logs",
			Value:       string(logsJSON),
			Type:        "json",
			Category:    "general",
			DisplayName: "配置更新日志",
			Description: "配置更新任务日志",
		}
		s.db.Create(&config)
	} else {
		// 更新现有记录
		config.Value = string(logsJSON)
		s.db.Save(&config)
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
		// 如果记录不存在，创建空记录
		config = models.SystemConfig{
			Key:         "config_update_logs",
			Value:       "[]",
			Type:        "json",
			Category:    "general",
			DisplayName: "配置更新日志",
			Description: "配置更新任务日志",
		}
		return s.db.Create(&config).Error
	} else {
		// 清空日志
		config.Value = "[]"
		return s.db.Save(&config).Error
	}
}

// saveConfigToDB 保存配置到数据库
func (s *ConfigUpdateService) saveConfigToDB(key, configType, value string) {
	var config models.SystemConfig
	err := s.db.Where("key = ? AND type = ?", key, configType).First(&config).Error

	if err != nil {
		config = models.SystemConfig{
			Key:         key,
			Value:       value,
			Type:        configType,
			Category:    "proxy",
			DisplayName: fmt.Sprintf("%s配置", configType),
			Description: "自动生成的配置",
		}
		s.db.Create(&config)
	} else {
		config.Value = value
		s.db.Save(&config)
	}
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

// GetConfig 获取配置（公开方法）
func (s *ConfigUpdateService) GetConfig() (map[string]interface{}, error) {
	return s.getConfig()
}

// importNodesToDatabase 将节点导入到数据库的 nodes 表
func (s *ConfigUpdateService) importNodesToDatabase(proxies []*ProxyNode) int {
	// 不清空所有节点，只更新或创建新节点
	// 手动添加的节点（is_manual=true）不会被删除
	s.addLog(fmt.Sprintf("开始导入节点到数据库，共 %d 个节点", len(proxies)), "info")

	importedCount := 0
	updatedCount := 0
	createdCount := 0
	skippedCount := 0
	seenKeys := make(map[string]bool)
	errorCount := 0

	for _, node := range proxies {
		// 验证必要字段
		if node.Server == "" || node.Port == 0 || node.Type == "" {
			errorCount++
			s.addLog(fmt.Sprintf("跳过无效节点: Server=%s, Port=%d, Type=%s", node.Server, node.Port, node.Type), "warning")
			continue
		}

		// 生成去重键（用于内存去重）
		key := fmt.Sprintf("%s:%s:%d", node.Type, node.Server, node.Port)
		if node.UUID != "" {
			key += ":" + node.UUID
		} else if node.Password != "" {
			key += ":" + node.Password
		}

		// 内存去重
		if seenKeys[key] {
			continue
		}
		seenKeys[key] = true

		// 确保节点名称不为空
		if node.Name == "" {
			node.Name = fmt.Sprintf("%s-%s:%d", node.Type, node.Server, node.Port)
		}

		// 从节点名称提取地区信息
		region := s.extractRegionFromName(node.Name)
		if region == "" {
			region = "未知"
		}

		// 序列化节点配置（确保包含所有字段）
		configJSON, err := json.Marshal(node)
		if err != nil {
			errorCount++
			s.addLog(fmt.Sprintf("序列化节点配置失败: %v", err), "error")
			continue
		}
		configStr := string(configJSON)

		// 使用更精确的方式检查是否已存在相同节点
		// 查询所有相同类型的节点，然后通过解析 JSON 配置精确匹配
		var existingNodes []models.Node
		s.db.Where("type = ? AND is_manual = ?", node.Type, false).Find(&existingNodes)
		
		var existingNode *models.Node
		for i := range existingNodes {
			if existingNodes[i].Config == nil {
				continue
			}
			var existingProxy ProxyNode
			if err := json.Unmarshal([]byte(*existingNodes[i].Config), &existingProxy); err != nil {
				continue
			}
			
			// 精确匹配：server、port、uuid/password 必须完全相同
			if existingProxy.Server == node.Server && existingProxy.Port == node.Port {
				// 对于有 UUID 的节点（vmess, vless），必须 UUID 相同
				if node.UUID != "" {
					if existingProxy.UUID == node.UUID {
						existingNode = &existingNodes[i]
						break
					}
				} else if node.Password != "" {
					// 对于有 Password 的节点（trojan, ss），必须 Password 相同
					if existingProxy.Password == node.Password {
						existingNode = &existingNodes[i]
						break
					}
				} else {
					// 对于没有 UUID 和 Password 的节点，server 和 port 相同即认为是同一个节点
					existingNode = &existingNodes[i]
					break
				}
			}
		}

		if existingNode != nil {
			// 如果节点已存在且是手动添加的，跳过更新
			if existingNode.IsManual {
				skippedCount++
				continue
			}
			// 更新现有节点（非手动添加的）
			existingNode.Name = node.Name
			existingNode.Region = region
			existingNode.Status = "online"
			existingNode.IsActive = true
			existingNode.Config = &configStr
			if err := s.db.Save(existingNode).Error; err != nil {
				errorCount++
				s.addLog(fmt.Sprintf("更新节点失败 [%s]: %v", node.Name, err), "error")
				continue
			}
			updatedCount++
			importedCount++
		} else {
			// 创建新节点（标记为非手动添加）
			newNode := models.Node{
				Name:     node.Name,
				Region:   region,
				Type:     node.Type,
				Status:   "online", // 新采集的节点默认为在线状态
				IsActive:  true,
				IsManual: false, // 采集的节点标记为非手动
				Config:   &configStr,
			}

			if err := s.db.Create(&newNode).Error; err != nil {
				errorCount++
				s.addLog(fmt.Sprintf("创建节点失败 [%s]: %v", node.Name, err), "error")
				continue
			}
			createdCount++
			importedCount++
		}
	}

	if errorCount > 0 {
		s.addLog(fmt.Sprintf("导入过程中有 %d 个节点失败", errorCount), "warning")
	}
	
	s.addLog(fmt.Sprintf("导入完成：新建 %d 个，更新 %d 个，跳过 %d 个（手动节点），失败 %d 个，总计 %d 个", 
		createdCount, updatedCount, skippedCount, errorCount, importedCount), "info")

	return importedCount
}

// extractRegionFromName 从节点名称提取地区信息
func (s *ConfigUpdateService) extractRegionFromName(name string) string {
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

// addInfoAndReminderNodes 添加信息节点和提醒节点到配置前
// 信息节点使用特殊的节点名称，在 Clash 中会显示在节点列表中
// 对于 V2Ray/SSR 格式，这些信息节点会被转换为特殊的 VMess 链接，在客户端中显示
func (s *ConfigUpdateService) addInfoAndReminderNodes(proxies []*ProxyNode, subscription models.Subscription, user models.User, isExpired, isInactive, isDeviceOverLimit bool, currentDevices, deviceLimit int) []*ProxyNode {
	// 获取网站域名（自动识别）
	siteURL := s.getSiteURL()
	// 如果找不到域名，使用默认提示
	if siteURL == "" {
		siteURL = "请在系统设置中配置域名"
	}

	// 格式化到期时间
	expireTimeStr := subscription.ExpireTime.Format("2006-01-02 15:04:05")

	// 售后QQ
	supportQQ := "3219904322"

	// 创建信息节点列表（使用 DIRECT 类型的特殊节点，在 Clash 中会显示但不可用）
	// 在 V2Ray/SSR 格式中，这些节点会被转换为特殊的 VMess 链接
	infoNodes := make([]*ProxyNode, 0)

	// 1. 网站域名信息节点
	infoNode1 := &ProxyNode{
		Name:   fmt.Sprintf("📢 网站域名: %s", siteURL),
		Type:   "direct",
		Server: "127.0.0.1",
		Port:   0,
		Options: map[string]interface{}{
			"info": fmt.Sprintf("网站域名: %s", siteURL),
		},
	}
	infoNodes = append(infoNodes, infoNode1)

	// 2. 到期时间信息节点
	infoNode2 := &ProxyNode{
		Name:   fmt.Sprintf("⏰ 到期时间: %s", expireTimeStr),
		Type:   "direct",
		Server: "127.0.0.1",
		Port:   0,
		Options: map[string]interface{}{
			"info": fmt.Sprintf("到期时间: %s", expireTimeStr),
		},
	}
	infoNodes = append(infoNodes, infoNode2)

	// 3. 售后QQ信息节点
	infoNode3 := &ProxyNode{
		Name:   fmt.Sprintf("💬 售后QQ: %s", supportQQ),
		Type:   "direct",
		Server: "127.0.0.1",
		Port:   0,
		Options: map[string]interface{}{
			"info": fmt.Sprintf("售后QQ: %s", supportQQ),
		},
	}
	infoNodes = append(infoNodes, infoNode3)

	// 4. 到期提醒节点（如果已过期）
	if isExpired {
		reminderNode := &ProxyNode{
			Name:   "⚠️ 订阅已过期，请及时续费！",
			Type:   "direct",
			Server: "127.0.0.1",
			Port:   0,
			Options: map[string]interface{}{
				"info": "订阅已过期，请及时续费！",
			},
		}
		infoNodes = append(infoNodes, reminderNode)
	}

	// 5. 设备超限提醒节点（如果设备超限）
	if isDeviceOverLimit {
		reminderNode := &ProxyNode{
			Name:   fmt.Sprintf("⚠️ 设备超限！当前 %d/%d，请删除多余设备", currentDevices, deviceLimit),
			Type:   "direct",
			Server: "127.0.0.1",
			Port:   0,
			Options: map[string]interface{}{
				"info": fmt.Sprintf("设备超限！当前 %d/%d，请删除多余设备", currentDevices, deviceLimit),
			},
		}
		infoNodes = append(infoNodes, reminderNode)
	}

	// 6. 订阅失效提醒节点（如果订阅未激活）
	if isInactive {
		reminderNode := &ProxyNode{
			Name:   "⚠️ 订阅已失效，请联系客服！",
			Type:   "direct",
			Server: "127.0.0.1",
			Port:   0,
			Options: map[string]interface{}{
				"info": "订阅已失效，请联系客服！",
			},
		}
		infoNodes = append(infoNodes, reminderNode)
	}

	// 将信息节点插入到最前面
	return append(infoNodes, proxies...)
}

// getSiteURL 获取网站域名
func (s *ConfigUpdateService) getSiteURL() string {
	// 优先从系统配置获取 domain_name（general 类别）
	var config models.SystemConfig
	if err := s.db.Where("key = ? AND category = ?", "domain_name", "general").First(&config).Error; err == nil && config.Value != "" {
		domain := strings.TrimSpace(config.Value)
		// 如果配置的域名包含协议，直接使用
		if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
			return strings.TrimSuffix(domain, "/")
		}
		// 否则默认使用 https
		return "https://" + domain
	}

	// 其次查找 domain_name（不限制 category，兼容旧配置）
	if err := s.db.Where("key = ?", "domain_name").First(&config).Error; err == nil && config.Value != "" {
		domain := strings.TrimSpace(config.Value)
		if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
			return strings.TrimSuffix(domain, "/")
		}
		return "https://" + domain
	}

	// 再次查找 site_url 或 base_url（不限制 category，兼容旧配置）
	if err := s.db.Where("key = ?", "site_url").Or("key = ?", "base_url").First(&config).Error; err == nil && config.Value != "" {
		return strings.TrimSpace(config.Value)
	}

	// 从环境变量获取
	if baseURL := os.Getenv("BASE_URL"); baseURL != "" {
		return baseURL
	}

	// 如果都找不到，返回空字符串（由调用方处理，或使用默认值）
	// 这不应该发生，应该在系统设置中配置 domain_name
	return ""
}

// GenerateV2RayConfig 生成 V2Ray 格式订阅配置
func (s *ConfigUpdateService) GenerateV2RayConfig(userID uint, subscriptionURL string) (string, error) {
	// 获取节点（复用 Clash 的逻辑）
	proxies, subscription, user, isExpired, isInactive, isDeviceOverLimit, currentDevices, deviceLimit, err := s.getNodesForSubscription(userID, subscriptionURL)
	if err != nil {
		return "", err
	}

	// 添加信息节点（信息节点会转换为 VMess 链接，在客户端中显示）
	proxies = s.addInfoAndReminderNodes(proxies, subscription, user, isExpired, isInactive, isDeviceOverLimit, currentDevices, deviceLimit)

	// 生成 V2Ray 格式的节点链接列表
	var links []string

	// 添加所有节点链接（包括信息节点，它们会被转换为特殊的 VMess 链接）
	for _, proxy := range proxies {
		link := s.ProxyNodeToLink(proxy)
		if link != "" {
			links = append(links, link)
		}
	}

	return strings.Join(links, "\n"), nil
}

// GenerateSSRConfig 生成 SSR 格式订阅配置
func (s *ConfigUpdateService) GenerateSSRConfig(userID uint, subscriptionURL string) (string, error) {
	// 获取节点（复用 Clash 的逻辑）
	proxies, subscription, user, isExpired, isInactive, isDeviceOverLimit, currentDevices, deviceLimit, err := s.getNodesForSubscription(userID, subscriptionURL)
	if err != nil {
		return "", err
	}

	// 添加信息节点（信息节点会转换为 VMess 链接，在客户端中显示）
	proxies = s.addInfoAndReminderNodes(proxies, subscription, user, isExpired, isInactive, isDeviceOverLimit, currentDevices, deviceLimit)

	// SSR 格式也是节点链接列表
	var links []string

	// 添加所有节点链接（包括信息节点，它们会被转换为特殊的 VMess 链接）
	for _, proxy := range proxies {
		link := s.ProxyNodeToLink(proxy)
		if link != "" {
			links = append(links, link)
		}
	}

	return strings.Join(links, "\n"), nil
}

// getNodesForSubscription 获取订阅节点（公共逻辑）
func (s *ConfigUpdateService) getNodesForSubscription(userID uint, subscriptionURL string) ([]*ProxyNode, models.Subscription, models.User, bool, bool, bool, int, int, error) {
	// 获取用户订阅
	var subscription models.Subscription
	if err := s.db.Where("subscription_url = ?", subscriptionURL).First(&subscription).Error; err != nil {
		return nil, subscription, models.User{}, false, false, false, 0, 0, fmt.Errorf("订阅不存在")
	}

	// 获取用户信息
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, subscription, user, false, false, false, 0, 0, fmt.Errorf("用户不存在")
	}

	// 检查订阅状态
	now := time.Now()
	isExpired := subscription.ExpireTime.Before(now)
	isInactive := !subscription.IsActive || subscription.Status != "active"

	// 检查设备数量
	var deviceCount int64
	s.db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", subscription.ID, true).Count(&deviceCount)
	isDeviceOverLimit := int(deviceCount) > subscription.DeviceLimit

	// 获取节点
	// 只获取激活的节点
	// 正常显示所有节点信息，但订阅时排除超时节点
	var proxies []*ProxyNode
	var dbNodes []models.Node

	// 根据用户设置决定是否包含普通节点
	if user.SpecialNodeSubscriptionType != "special_only" {
		// 检查普通订阅是否到期
		if !isExpired {
			// 获取激活的节点，过滤掉 status 为 timeout 的节点
			if err := s.db.Where("is_active = ? AND status != ?", true, "timeout").Find(&dbNodes).Error; err == nil && len(dbNodes) > 0 {
				// 从数据库获取节点
				for _, dbNode := range dbNodes {
					// 双重验证：确保节点配置存在且有效
					if dbNode.Config != nil && *dbNode.Config != "" {
						var proxyNode ProxyNode
						if err := json.Unmarshal([]byte(*dbNode.Config), &proxyNode); err == nil {
							// 验证节点配置的基本字段
							if proxyNode.Server != "" && proxyNode.Port > 0 && proxyNode.Type != "" {
								proxyNode.Name = dbNode.Name
								proxies = append(proxies, &proxyNode)
							}
						}
					}
				}
			}
		}
	}

	// 获取用户的专线节点
	var customNodes []models.CustomNode
	if err := s.db.Joins("JOIN user_custom_nodes ON user_custom_nodes.custom_node_id = custom_nodes.id").
		Where("user_custom_nodes.user_id = ? AND custom_nodes.is_active = ?", userID, true).
		Find(&customNodes).Error; err == nil && len(customNodes) > 0 {

		// 检查每个专线节点的到期时间
		for _, customNode := range customNodes {
			isSpecialExpired := false
			if customNode.FollowUserExpire {
				// 如果跟随用户到期时间，优先使用 SpecialNodeExpiresAt
				if user.SpecialNodeExpiresAt.Valid {
					isSpecialExpired = user.SpecialNodeExpiresAt.Time.Before(now)
				} else {
					isSpecialExpired = subscription.ExpireTime.Before(now)
				}
			} else if customNode.ExpireTime != nil {
				isSpecialExpired = customNode.ExpireTime.Before(now)
			}

			// 如果专线已到期或节点状态为 timeout，则不显示在订阅中
			if isSpecialExpired || customNode.Status == "timeout" {
				continue
			}

			// 解析节点配置
			if customNode.Config != "" {
				var proxyNode ProxyNode
				if err := json.Unmarshal([]byte(customNode.Config), &proxyNode); err == nil {
					if proxyNode.Server != "" && proxyNode.Port > 0 && proxyNode.Type != "" {
						displayName := customNode.DisplayName
						if displayName == "" {
							displayName = "专线定制-" + customNode.Name
						}
						proxyNode.Name = displayName
						proxies = append(proxies, &proxyNode)
					}
				}
			}
		}
	}

	// 如果没有任何节点，返回错误
	if len(proxies) == 0 {
		return nil, subscription, user, isExpired, isInactive, isDeviceOverLimit, int(deviceCount), subscription.DeviceLimit, fmt.Errorf("没有可用的节点")
	}

	return proxies, subscription, user, isExpired, isInactive, isDeviceOverLimit, int(deviceCount), subscription.DeviceLimit, nil
}

// ProxyNodeToLink 将 ProxyNode 转换为节点链接（公开方法）
func (s *ConfigUpdateService) ProxyNodeToLink(proxy *ProxyNode) string {
	// 信息节点（direct 类型且 server 为 127.0.0.1）转换为特殊的 VMess 链接
	// 这样在 V2Ray/SSR 格式中也能显示信息
	if proxy.Type == "direct" && proxy.Server == "127.0.0.1" {
		return s.infoNodeToLink(proxy)
	}

	switch proxy.Type {
	case "vmess":
		return s.vmessToLink(proxy)
	case "vless":
		return s.vlessToLink(proxy)
	case "trojan":
		return s.trojanToLink(proxy)
	case "ss":
		return s.shadowsocksToLink(proxy)
	case "ssr":
		return s.ssrToLink(proxy)
	default:
		return ""
	}
}

// infoNodeToLink 将信息节点转换为 VMess 链接（用于在 V2Ray/SSR 格式中显示信息）
func (s *ConfigUpdateService) infoNodeToLink(proxy *ProxyNode) string {
	// 创建一个特殊的 VMess 节点，将信息编码在节点名称中
	// 使用一个无效的服务器地址，这样客户端会显示节点但无法连接
	data := map[string]interface{}{
		"v":    "2",
		"ps":   proxy.Name,                             // 节点名称包含信息
		"add":  "127.0.0.1",                            // 无效地址，防止实际连接
		"port": 0,                                      // 无效端口
		"id":   "00000000-0000-0000-0000-000000000000", // 无效 UUID
		"net":  "tcp",
		"type": "none",
	}

	jsonData, _ := json.Marshal(data)
	encoded := base64.StdEncoding.EncodeToString(jsonData)
	return "vmess://" + encoded
}

// vmessToLink 将 VMess 节点转换为链接
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
		if grpcOpts, ok := proxy.Options["grpc-opts"].(map[string]interface{}); ok {
			if serviceName, ok := grpcOpts["grpc-service-name"].(string); ok {
				data["path"] = serviceName
			}
		}
	}

	jsonData, _ := json.Marshal(data)
	encoded := base64.StdEncoding.EncodeToString(jsonData)
	return "vmess://" + encoded
}

// vlessToLink 将 VLESS 节点转换为链接
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

	if proxy.Options != nil {
		if wsOpts, ok := proxy.Options["ws-opts"].(map[string]interface{}); ok {
			if path, ok := wsOpts["path"].(string); ok {
				q.Set("path", path)
			}
			if headers, ok := wsOpts["headers"].(map[string]interface{}); ok {
				if host, ok := headers["Host"].(string); ok {
					q.Set("host", host)
				}
			}
		}
	}

	u.RawQuery = q.Encode()
	return u.String()
}

// trojanToLink 将 Trojan 节点转换为链接
func (s *ConfigUpdateService) trojanToLink(proxy *ProxyNode) string {
	u := &url.URL{
		Scheme:   "trojan",
		User:     url.User(proxy.Password),
		Host:     fmt.Sprintf("%s:%d", proxy.Server, proxy.Port),
		Fragment: proxy.Name,
	}

	q := url.Values{}
	if proxy.Network != "" {
		q.Set("type", proxy.Network)
	}

	if proxy.Options != nil {
		if wsOpts, ok := proxy.Options["ws-opts"].(map[string]interface{}); ok {
			if path, ok := wsOpts["path"].(string); ok {
				q.Set("path", path)
			}
			if headers, ok := wsOpts["headers"].(map[string]interface{}); ok {
				if host, ok := headers["Host"].(string); ok {
					q.Set("host", host)
				}
			}
		}
	}

	u.RawQuery = q.Encode()
	return u.String()
}

// shadowsocksToLink 将 Shadowsocks 节点转换为链接
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

// ssrToLink 将 SSR 节点转换为链接
func (s *ConfigUpdateService) ssrToLink(proxy *ProxyNode) string {
	// SSR 链接格式较复杂，这里简化处理
	// 实际应该根据 SSR 协议规范生成
	return ""
}

// GenerateClashConfigWithReminder 生成带提醒的 Clash 配置（用于设备超限等情况）
func (s *ConfigUpdateService) GenerateClashConfigWithReminder(userID uint, subscriptionURL string, isDeviceOverLimit, isExpired bool, currentDevices, deviceLimit int) (string, error) {
	// 获取用户订阅
	var subscription models.Subscription
	if err := s.db.Where("subscription_url = ?", subscriptionURL).First(&subscription).Error; err != nil {
		return "", fmt.Errorf("订阅不存在")
	}

	// 获取用户信息
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return "", fmt.Errorf("用户不存在")
	}

	// 获取节点（即使超限也要生成配置，只是添加提醒）
	var proxies []*ProxyNode
	var dbNodes []models.Node
	if err := s.db.Where("is_active = ?", true).Find(&dbNodes).Error; err == nil && len(dbNodes) > 0 {
		for _, dbNode := range dbNodes {
			if dbNode.Config != nil && *dbNode.Config != "" {
				var proxyNode ProxyNode
				if err := json.Unmarshal([]byte(*dbNode.Config), &proxyNode); err == nil {
					proxyNode.Name = dbNode.Name
					proxies = append(proxies, &proxyNode)
				}
			}
		}
	}

	if len(proxies) == 0 {
		return "", fmt.Errorf("没有可用的节点")
	}

	// 添加信息和提醒节点
	isInactive := !subscription.IsActive || subscription.Status != "active"
	proxies = s.addInfoAndReminderNodes(proxies, subscription, user, isExpired, isInactive, isDeviceOverLimit, currentDevices, deviceLimit)

	return s.generateClashYAML(proxies), nil
}

// GenerateV2RayConfigWithReminder 生成带提醒的 V2Ray 配置
func (s *ConfigUpdateService) GenerateV2RayConfigWithReminder(userID uint, subscriptionURL string, isDeviceOverLimit, isExpired bool, currentDevices, deviceLimit int) (string, error) {
	// 获取用户订阅
	var subscription models.Subscription
	if err := s.db.Where("subscription_url = ?", subscriptionURL).First(&subscription).Error; err != nil {
		return "", fmt.Errorf("订阅不存在")
	}

	// 获取用户信息
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return "", fmt.Errorf("用户不存在")
	}

	// 获取节点
	var proxies []*ProxyNode
	var dbNodes []models.Node
	if err := s.db.Where("is_active = ?", true).Find(&dbNodes).Error; err == nil && len(dbNodes) > 0 {
		for _, dbNode := range dbNodes {
			if dbNode.Config != nil && *dbNode.Config != "" {
				var proxyNode ProxyNode
				if err := json.Unmarshal([]byte(*dbNode.Config), &proxyNode); err == nil {
					proxyNode.Name = dbNode.Name
					proxies = append(proxies, &proxyNode)
				}
			}
		}
	}

	if len(proxies) == 0 {
		return "", fmt.Errorf("没有可用的节点")
	}

	// 添加信息和提醒节点
	isInactive := !subscription.IsActive || subscription.Status != "active"
	proxies = s.addInfoAndReminderNodes(proxies, subscription, user, isExpired, isInactive, isDeviceOverLimit, currentDevices, deviceLimit)

	// 生成 V2Ray 格式的节点链接列表
	var links []string

	// 添加信息注释
	siteURL := s.getSiteURL()
	expireTimeStr := subscription.ExpireTime.Format("2006-01-02 15:04:05")
	supportQQ := "3219904322"

	infoText := fmt.Sprintf("网站域名: %s | 到期时间: %s | 售后QQ: %s", siteURL, expireTimeStr, supportQQ)
	if isExpired {
		infoText += " | ⚠️ 订阅已过期，请及时续费！"
	}
	if isDeviceOverLimit {
		infoText += fmt.Sprintf(" | ⚠️ 设备超限！当前 %d/%d，请删除多余设备", currentDevices, deviceLimit)
	}
	if isInactive {
		infoText += " | ⚠️ 订阅已失效，请联系客服！"
	}

	infoEncoded := base64.StdEncoding.EncodeToString([]byte(infoText))
	links = append(links, "# "+infoEncoded)

	// 添加实际节点链接
	for _, proxy := range proxies {
		link := s.ProxyNodeToLink(proxy)
		if link != "" {
			links = append(links, link)
		}
	}

	return strings.Join(links, "\n"), nil
}

// GenerateSSRConfigWithReminder 生成带提醒的 SSR 配置
func (s *ConfigUpdateService) GenerateSSRConfigWithReminder(userID uint, subscriptionURL string, isDeviceOverLimit, isExpired bool, currentDevices, deviceLimit int) (string, error) {
	// 获取用户订阅
	var subscription models.Subscription
	if err := s.db.Where("subscription_url = ?", subscriptionURL).First(&subscription).Error; err != nil {
		return "", fmt.Errorf("订阅不存在")
	}

	// 获取用户信息
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return "", fmt.Errorf("用户不存在")
	}

	// 获取节点
	var proxies []*ProxyNode
	var dbNodes []models.Node
	if err := s.db.Where("is_active = ?", true).Find(&dbNodes).Error; err == nil && len(dbNodes) > 0 {
		for _, dbNode := range dbNodes {
			if dbNode.Config != nil && *dbNode.Config != "" {
				var proxyNode ProxyNode
				if err := json.Unmarshal([]byte(*dbNode.Config), &proxyNode); err == nil {
					proxyNode.Name = dbNode.Name
					proxies = append(proxies, &proxyNode)
				}
			}
		}
	}

	if len(proxies) == 0 {
		return "", fmt.Errorf("没有可用的节点")
	}

	// 添加信息和提醒节点
	isInactive := !subscription.IsActive || subscription.Status != "active"
	proxies = s.addInfoAndReminderNodes(proxies, subscription, user, isExpired, isInactive, isDeviceOverLimit, currentDevices, deviceLimit)

	// SSR 格式也是节点链接列表
	var links []string

	// 添加信息注释
	siteURL := s.getSiteURL()
	expireTimeStr := subscription.ExpireTime.Format("2006-01-02 15:04:05")
	supportQQ := "3219904322"

	infoText := fmt.Sprintf("网站域名: %s | 到期时间: %s | 售后QQ: %s", siteURL, expireTimeStr, supportQQ)
	if isExpired {
		infoText += " | ⚠️ 订阅已过期，请及时续费！"
	}
	if isDeviceOverLimit {
		infoText += fmt.Sprintf(" | ⚠️ 设备超限！当前 %d/%d，请删除多余设备", currentDevices, deviceLimit)
	}
	if isInactive {
		infoText += " | ⚠️ 订阅已失效，请联系客服！"
	}

	infoEncoded := base64.StdEncoding.EncodeToString([]byte(infoText))
	links = append(links, "# "+infoEncoded)

	// 添加实际节点链接
	for _, proxy := range proxies {
		link := s.ProxyNodeToLink(proxy)
		if link != "" {
			links = append(links, link)
		}
	}

	return strings.Join(links, "\n"), nil
}

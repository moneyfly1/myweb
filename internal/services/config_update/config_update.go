package config_update

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		fmt.Printf("正在下载节点源 [%d/%d]: %s\n", i+1, len(urls), url)

		// 下载内容
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("下载失败: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		content, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("读取内容失败: %v\n", err)
			continue
		}

		// 尝试 Base64 解码
		decoded := s.tryBase64Decode(string(content))

		// 提取节点链接
		nodeLinks := s.extractNodeLinks(decoded)
		fmt.Printf("从 %s 提取到 %d 个节点链接\n", url, len(nodeLinks))

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

// extractNodeLinks 提取节点链接
func (s *ConfigUpdateService) extractNodeLinks(content string) []string {
	var links []string

	// 匹配各种协议链接
	patterns := []string{
		`(vmess://[^\s]+)`,
		`(vless://[^\s]+)`,
		`(trojan://[^\s]+)`,
		`(ss://[^\s]+)`,
		`(ssr://[^\s]+)`,
		`(hysteria://[^\s]+)`,
		`(hysteria2://[^\s]+)`,
		`(tuic://[^\s]+)`,
		`(wireguard://[^\s]+)`,
		`(http://[^\s]+)`,
		`(https://[^\s]+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
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
	// 获取用户订阅
	var subscription models.Subscription
	if err := s.db.Where("subscription_url = ?", subscriptionURL).First(&subscription).Error; err != nil {
		return "", fmt.Errorf("订阅不存在")
	}

	// 检查订阅是否有效
	if !subscription.IsActive || subscription.Status != "active" {
		return "", fmt.Errorf("订阅已失效")
	}

	now := time.Now()
	if subscription.ExpireTime.Before(now) {
		return "", fmt.Errorf("订阅已过期")
	}

	// 获取节点配置
	var systemConfig models.SystemConfig
	if err := s.db.Where("key = ?", "node_source_urls").First(&systemConfig).Error; err != nil {
		return "", fmt.Errorf("未配置节点源")
	}

	// 解析节点源URL列表
	urls := strings.Split(systemConfig.Value, "\n")
	var validURLs []string
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u != "" {
			validURLs = append(validURLs, u)
		}
	}

	// 获取节点链接
	nodeData, err := s.FetchNodesFromURLs(validURLs)
	if err != nil {
		return "", err
	}

	// 解析节点链接为代理节点
	var proxies []*ProxyNode
	seenKeys := make(map[string]bool)
	nameCounter := make(map[string]int)

	for _, nodeInfo := range nodeData {
		link, ok := nodeInfo["url"].(string)
		if !ok {
			continue
		}

		node, err := ParseNodeLink(link)
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

		// 处理名称重复
		if count, exists := nameCounter[node.Name]; exists {
			nameCounter[node.Name] = count + 1
			node.Name = fmt.Sprintf("%s-%d", node.Name, count+1)
		} else {
			nameCounter[node.Name] = 0
		}

		proxies = append(proxies, node)
	}

	if len(proxies) == 0 {
		return "", fmt.Errorf("没有可用的节点")
	}

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

	// 这里可以选择保存到文件系统或更新数据库记录
	// 目前配置是实时生成的，所以这里主要是验证配置生成是否成功
	fmt.Printf("订阅配置已更新: %s, 配置长度: %d 字符\n", subscriptionURL, len(config))

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

	// 更新最后更新时间
	s.updateLastUpdateTime()

	s.addLog(fmt.Sprintf("✅ 配置更新任务完成！下载节点数: %d, 最终节点数: %d", len(nodes), len(proxies)), "success")

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

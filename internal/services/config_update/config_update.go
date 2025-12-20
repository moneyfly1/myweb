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
	"strings"
	"sync"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

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
	db           *gorm.DB
	isRunning    bool
	runningMutex sync.Mutex
	// 缓存站点URL，避免频繁查询
	siteURL string
	// 缓存客服QQ
	supportQQ string
}

// NewConfigUpdateService 创建配置更新服务
func NewConfigUpdateService() *ConfigUpdateService {
	service := &ConfigUpdateService{
		db: database.GetDB(),
	}
	// 初始化缓存配置
	service.refreshSystemConfig()
	return service
}

// refreshSystemConfig 刷新系统配置缓存
func (s *ConfigUpdateService) refreshSystemConfig() {
	// 获取网站域名
	var config models.SystemConfig
	if err := s.db.Where("key = ? AND category = ?", "domain_name", "general").First(&config).Error; err == nil && config.Value != "" {
		s.siteURL = strings.TrimSpace(config.Value)
		// 确保没有协议前缀
		s.siteURL = strings.TrimPrefix(s.siteURL, "http://")
		s.siteURL = strings.TrimPrefix(s.siteURL, "https://")
		s.siteURL = strings.TrimRight(s.siteURL, "/")
	} else {
		// 尝试其他配置项
		if err := s.db.Where("key = ?", "site_url").Or("key = ?", "base_url").First(&config).Error; err == nil && config.Value != "" {
			s.siteURL = strings.TrimSpace(config.Value)
		} else {
			s.siteURL = "请在系统设置中配置域名"
		}
	}

	// 获取客服QQ
	s.supportQQ = "3219904322"
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

	// 去重并验证链接完整性
	uniqueLinks := make(map[string]bool)
	var result []string
	for _, link := range links {
		// 跳过重复的链接
		if uniqueLinks[link] {
			continue
		}

		// 验证链接完整性
		if !s.isValidNodeLink(link) {
			continue
		}

		uniqueLinks[link] = true
		result = append(result, link)
	}

	return result
}

// isValidNodeLink 验证节点链接是否完整有效
func (s *ConfigUpdateService) isValidNodeLink(link string) bool {
	link = strings.TrimSpace(link)
	if link == "" {
		return false
	}

	// 检查基本格式：必须有协议前缀和至少一个 @ 或 : 符号
	if strings.HasPrefix(link, "ss://") {
		// SS 链接必须包含 @ 符号（认证信息@服务器:端口）
		if !strings.Contains(link, "@") {
			return false
		}
		// 检查是否有服务器地址和端口
		parts := strings.Split(link, "@")
		if len(parts) < 2 {
			return false
		}
		serverPart := parts[1]
		if !strings.Contains(serverPart, ":") {
			return false
		}
	} else if strings.HasPrefix(link, "vmess://") || strings.HasPrefix(link, "vless://") {
		// VMess/VLESS 链接必须包含 Base64 编码的内容
		encoded := strings.TrimPrefix(link, "vmess://")
		encoded = strings.TrimPrefix(encoded, "vless://")
		if len(encoded) < 10 {
			return false
		}
	} else if strings.HasPrefix(link, "trojan://") {
		// Trojan 链接必须包含 @ 符号
		if !strings.Contains(link, "@") {
			return false
		}
	} else if strings.HasPrefix(link, "ssr://") {
		// SSR 链接必须包含 Base64 编码的内容
		encoded := strings.TrimPrefix(link, "ssr://")
		if len(encoded) < 10 {
			return false
		}
	}

	return true
}

// GenerateClashConfig 生成 Clash 配置
func (s *ConfigUpdateService) GenerateClashConfig(token string, clientIP string, userAgent string) (string, error) {
	// 1. 获取订阅上下文（统一入口）
	ctx := s.getSubscriptionContext(token, clientIP, userAgent)

	// 2. 如果状态不正常，返回错误节点配置
	if ctx.Status != StatusNormal {
		errorNodes := s.generateErrorNodes(ctx.Status, ctx)
		return s.generateClashYAML(errorNodes), nil
	}

	// 3. 正常状态：添加信息节点到真实节点前
	finalNodes := s.addInfoNodes(ctx.Proxies, ctx)

	// 4. 生成 YAML
	return s.generateClashYAML(finalNodes), nil
}

// GenerateUniversalConfig 生成通用订阅配置 (V2Ray/SSR/Base64)
// format: "base64" (普通通用订阅), "ssr" (SSR订阅)
func (s *ConfigUpdateService) GenerateUniversalConfig(token string, clientIP string, userAgent string, format string) (string, error) {
	// 1. 获取订阅上下文
	ctx := s.getSubscriptionContext(token, clientIP, userAgent)

	var nodesToExport []*ProxyNode

	// 2. 根据状态决定使用真实节点还是错误节点
	if ctx.Status != StatusNormal {
		nodesToExport = s.generateErrorNodes(ctx.Status, ctx)
	} else {
		nodesToExport = s.addInfoNodes(ctx.Proxies, ctx)
	}

	// 3. 生成链接列表
	var links []string
	for _, node := range nodesToExport {
		var link string
		switch format {
		case "ssr":
			if node.Type == "ssr" {
				link = s.nodeToSSRLink(node)
			} else {
				// 非 SSR 节点在 SSR 订阅中尽量转换，或者忽略
				// 为了兼容性，我们尝试转换为通用链接
				link = s.nodeToLink(node)
			}
		default:
			link = s.nodeToLink(node)
		}

		if link != "" {
			links = append(links, link)
		}
	}

	// 4. Base64 编码
	return base64.StdEncoding.EncodeToString([]byte(strings.Join(links, "\n"))), nil
}

// getSubscriptionContext 获取订阅上下文，统一处理所有查询和逻辑
func (s *ConfigUpdateService) getSubscriptionContext(token string, clientIP string, userAgent string) *SubscriptionContext {
	ctx := &SubscriptionContext{
		Status: StatusNotFound,
	}

	// 1. 查找订阅
	var sub models.Subscription
	if err := s.db.Where("subscription_url = ?", token).First(&sub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 检查是否是旧订阅地址
			var reset models.SubscriptionReset
			if err := s.db.Where("old_subscription_url = ?", token).First(&reset).Error; err == nil {
				ctx.Status = StatusOldAddress
				ctx.ResetRecord = &reset
				return ctx
			}
		}
		return ctx // StatusNotFound
	}
	ctx.Subscription = sub

	// 2. 查找用户
	var user models.User
	if err := s.db.First(&user, sub.UserID).Error; err != nil {
		return ctx // StatusNotFound (User missing)
	}
	ctx.User = user

	// 3. 检查用户状态
	if !user.IsActive {
		ctx.Status = StatusAccountAbnormal
		return ctx
	}

	// 4. 检查订阅状态
	if !sub.IsActive || sub.Status != "active" {
		ctx.Status = StatusInactive
		return ctx
	}

	// 5. 检查过期时间
	now := time.Now()
	if !sub.ExpireTime.IsZero() && sub.ExpireTime.Before(now) {
		ctx.Status = StatusExpired
		return ctx
	}

	// 6. 检查设备限制
	var currentDevices int64
	s.db.Model(&models.Device{}).Where("subscription_id = ? AND is_active = ?", sub.ID, true).Count(&currentDevices)
	ctx.CurrentDevices = int(currentDevices)
	ctx.DeviceLimit = sub.DeviceLimit

	// 检查是否为新设备且超限
	// 只有当设备限制 > 0 时才检查
	if sub.DeviceLimit > 0 && int(currentDevices) >= sub.DeviceLimit {
		// 检查当前设备是否已在列表中
		var device models.Device
		isKnownDevice := false
		// 尝试根据IP和UA查找设备（这只是一个近似检查）
		// 实际生产环境应该有设备指纹
		if err := s.db.Where("subscription_id = ? AND ip_address = ? AND user_agent = ?", sub.ID, clientIP, userAgent).First(&device).Error; err == nil {
			isKnownDevice = true
		}

		if !isKnownDevice {
			ctx.Status = StatusDeviceOverLimit
			return ctx
		}
	} else if sub.DeviceLimit == 0 {
		// 设备限制为 0，禁止任何设备连接
		ctx.Status = StatusDeviceOverLimit
		return ctx
	}

	// 7. 获取真实节点
	proxies, err := s.fetchProxiesForUser(user, sub)
	if err != nil {
		// 获取节点失败，暂且认为无节点
		ctx.Proxies = []*ProxyNode{}
	} else {
		ctx.Proxies = proxies
	}

	ctx.Status = StatusNormal
	return ctx
}

// fetchProxiesForUser 获取用户的可用节点
func (s *ConfigUpdateService) fetchProxiesForUser(user models.User, sub models.Subscription) ([]*ProxyNode, error) {
	// 查询用户可用的节点
	var nodes []models.Node

	query := s.db.Model(&models.Node{}).Where("is_active = ?", true)

	// 获取用户等级
	// userLevel := user.UserLevelID

	// 过滤掉超时节点
	query = query.Where("status != ?", "timeout")

	if err := query.Find(&nodes).Error; err != nil {
		return nil, err
	}

	var proxies []*ProxyNode

	// 处理节点去重和转换
	processedNodes := make(map[string]bool)

	for _, node := range nodes {
		// 权限检查
		// if node.Level > int(userLevel.Int64) {
		// 	continue
		// }

		// 转换节点
		proxyNodes, err := s.parseNodeToProxies(&node)
		if err != nil {
			continue
		}

		for _, proxy := range proxyNodes {
			// 简单的去重 key
			key := fmt.Sprintf("%s|%s|%d", proxy.Type, proxy.Server, proxy.Port)
			if processedNodes[key] {
				continue
			}
			processedNodes[key] = true
			proxies = append(proxies, proxy)
		}
	}

	// 获取专属节点
	var customNodes []models.CustomNode
	if err := s.db.Joins("JOIN user_custom_nodes ON user_custom_nodes.custom_node_id = custom_nodes.id").
		Where("user_custom_nodes.user_id = ? AND custom_nodes.is_active = ?", user.ID, true).
		Find(&customNodes).Error; err == nil {
		for _, cn := range customNodes {
			// 检查过期时间
			now := time.Now()
			isExpired := false
			if cn.FollowUserExpire {
				if user.SpecialNodeExpiresAt.Valid {
					isExpired = user.SpecialNodeExpiresAt.Time.Before(now)
				} else {
					isExpired = sub.ExpireTime.Before(now)
				}
			} else if cn.ExpireTime != nil {
				isExpired = cn.ExpireTime.Before(now)
			}

			if isExpired || cn.Status == "timeout" {
				continue
			}

			// 构造临时 Node 对象进行解析
			displayName := cn.DisplayName
			if displayName == "" {
				displayName = "专线-" + cn.Name
			}

			// 直接解析配置
			if cn.Config != "" {
				var proxyNode ProxyNode
				if err := json.Unmarshal([]byte(cn.Config), &proxyNode); err == nil {
					proxyNode.Name = displayName
					proxies = append(proxies, &proxyNode)
				}
			}
		}
	}

	return proxies, nil
}

// parseNodeToProxies 解析数据库节点模型为代理节点对象
func (s *ConfigUpdateService) parseNodeToProxies(node *models.Node) ([]*ProxyNode, error) {
	// 如果配置中包含详细信息，优先使用
	if node.Config != nil && *node.Config != "" {
		var configProxy ProxyNode
		if err := json.Unmarshal([]byte(*node.Config), &configProxy); err == nil {
			// 保留数据库中的名称
			configProxy.Name = node.Name
			return []*ProxyNode{&configProxy}, nil
		}
	}

	// 尝试解析 Link (如果存在且不在 Config 中)
	// 注意：models.Node 目前没有 Link 字段，所以这里主要依赖 Config
	// 如果需要支持 Link，需要在 models.Node 中添加 Link 字段或从其他地方获取

	return nil, fmt.Errorf("节点配置为空")
}

// generateErrorNodes 根据状态生成 4 个错误提示节点
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

	// 确保 siteURL 不为空
	if s.siteURL == "" {
		s.refreshSystemConfig()
	}

	// 创建4个提示节点
	// 使用 ss 类型，配置无效信息，确保在所有客户端都能显示且无法连接
	return []*ProxyNode{
		{
			Name:     fmt.Sprintf("📢 官网: %s", s.siteURL),
			Type:     "ss",
			Server:   "127.0.0.1",
			Port:     1234,
			Cipher:   "aes-128-gcm",
			Password: "error",
		},
		{
			Name:     fmt.Sprintf("❌ 原因: %s", reason),
			Type:     "ss",
			Server:   "127.0.0.1",
			Port:     1234,
			Cipher:   "aes-128-gcm",
			Password: "error",
		},
		{
			Name:     fmt.Sprintf("💡 解决: %s", solution),
			Type:     "ss",
			Server:   "127.0.0.1",
			Port:     1234,
			Cipher:   "aes-128-gcm",
			Password: "error",
		},
		{
			Name:     fmt.Sprintf("💬 客服QQ: %s", s.supportQQ),
			Type:     "ss",
			Server:   "127.0.0.1",
			Port:     1234,
			Cipher:   "aes-128-gcm",
			Password: "error",
		},
	}
}

// addInfoNodes 添加信息节点（流量、到期时间等）到真实节点列表前
func (s *ConfigUpdateService) addInfoNodes(proxies []*ProxyNode, ctx *SubscriptionContext) []*ProxyNode {
	// 刷新配置确保是最新的
	if s.siteURL == "" {
		s.refreshSystemConfig()
	}

	expireTimeStr := "无限期"
	if !ctx.Subscription.ExpireTime.IsZero() {
		expireTimeStr = ctx.Subscription.ExpireTime.Format("2006-01-02")
	}

	infoNodes := []*ProxyNode{
		{
			Name:     fmt.Sprintf("📢 官网: %s", s.siteURL),
			Type:     "ss",
			Server:   "127.0.0.1",
			Port:     1234,
			Cipher:   "aes-128-gcm",
			Password: "info",
		},
		{
			Name:     fmt.Sprintf("⏰ 到期: %s", expireTimeStr),
			Type:     "ss",
			Server:   "127.0.0.1",
			Port:     1234,
			Cipher:   "aes-128-gcm",
			Password: "info",
		},
		{
			Name:     fmt.Sprintf("📱 设备: %d/%d", ctx.CurrentDevices, ctx.DeviceLimit),
			Type:     "ss",
			Server:   "127.0.0.1",
			Port:     1234,
			Cipher:   "aes-128-gcm",
			Password: "info",
		},
	}

	// 将信息节点插入到最前面
	return append(infoNodes, proxies...)
}

// generateClashYAML 生成 Clash YAML 配置
func (s *ConfigUpdateService) generateClashYAML(proxies []*ProxyNode) string {
	var builder strings.Builder

	// 过滤掉 Clash 不支持的节点类型
	supportedTypes := map[string]bool{
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

	filteredProxies := make([]*ProxyNode, 0)
	for _, proxy := range proxies {
		if supportedTypes[proxy.Type] {
			filteredProxies = append(filteredProxies, proxy)
		}
	}

	// 写入基础配置
	builder.WriteString("port: 7890\n")
	builder.WriteString("socks-port: 7891\n")
	builder.WriteString("allow-lan: true\n")
	builder.WriteString("mode: Rule\n")
	builder.WriteString("log-level: info\n")
	builder.WriteString("external-controller: 127.0.0.1:9090\n\n")

	// 写入代理节点
	builder.WriteString("proxies:\n")
	for _, proxy := range filteredProxies {
		builder.WriteString(s.nodeToYAML(proxy, 2))
	}

	// 生成代理名称列表
	var proxyNames []string
	for _, proxy := range filteredProxies {
		escapedName := s.escapeYAMLString(proxy.Name)
		proxyNames = append(proxyNames, escapedName)
	}

	// 写入代理组
	builder.WriteString("\nproxy-groups:\n")
	builder.WriteString("  - name: \"🚀 节点选择\"\n")
	builder.WriteString("    type: select\n")
	builder.WriteString("    proxies:\n")
	builder.WriteString("      - \"♻️ 自动选择\"\n")
	for _, name := range proxyNames {
		builder.WriteString(fmt.Sprintf("      - %s\n", name))
	}

	builder.WriteString("  - name: \"♻️ 自动选择\"\n")
	builder.WriteString("    type: url-test\n")
	builder.WriteString("    url: http://www.gstatic.com/generate_204\n")
	builder.WriteString("    interval: 300\n")
	builder.WriteString("    tolerance: 50\n")
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
	builder.WriteString("  - GEOIP,CN,DIRECT\n")
	builder.WriteString("  - MATCH,🚀 节点选择\n")

	return builder.String()
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

// nodeToYAML 将节点转换为 YAML 格式
func (s *ConfigUpdateService) nodeToYAML(node *ProxyNode, indent int) string {
	indentStr := strings.Repeat(" ", indent)
	var builder strings.Builder

	escapedName := s.escapeYAMLString(node.Name)

	builder.WriteString(fmt.Sprintf("%s- name: %s\n", indentStr, escapedName))
	builder.WriteString(fmt.Sprintf("%s  type: %s\n", indentStr, node.Type))
	builder.WriteString(fmt.Sprintf("%s  server: %s\n", indentStr, node.Server))
	builder.WriteString(fmt.Sprintf("%s  port: %d\n", indentStr, node.Port))

	// 根据类型处理字段
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

	// 额外选项
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
	for key, value := range node.Options {
		// 跳过已处理字段
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
		for k, val := range v {
			// 特殊处理 http-opts：path 和 headers[Host] 必须是数组
			if key == "http-opts" {
				if k == "path" {
					escapedK := s.escapeYAMLString(k)
					builder.WriteString(fmt.Sprintf("%s%s:\n", subIndentStr, escapedK))
					subSubIndentStr := subIndentStr + "  "
					if pathStr, ok := val.(string); ok {
						escapedPath := s.escapeYAMLString(pathStr)
						builder.WriteString(fmt.Sprintf("%s- %s\n", subSubIndentStr, escapedPath))
					} else if pathSlice, ok := val.([]string); ok {
						for _, p := range pathSlice {
							escapedPath := s.escapeYAMLString(p)
							builder.WriteString(fmt.Sprintf("%s- %s\n", subSubIndentStr, escapedPath))
						}
					} else if pathSlice, ok := val.([]interface{}); ok {
						for _, p := range pathSlice {
							escapedPath := s.escapeYAMLString(fmt.Sprintf("%v", p))
							builder.WriteString(fmt.Sprintf("%s- %s\n", subSubIndentStr, escapedPath))
						}
					}
					continue
				} else if k == "headers" {
					escapedK := s.escapeYAMLString(k)
					builder.WriteString(fmt.Sprintf("%s%s:\n", subIndentStr, escapedK))
					subSubIndentStr := subIndentStr + "  "
					if headersMap, ok := val.(map[string]interface{}); ok {
						for hk, hv := range headersMap {
							escapedHK := s.escapeYAMLString(hk)
							builder.WriteString(fmt.Sprintf("%s%s:\n", subSubIndentStr, escapedHK))
							subSubSubIndentStr := subSubIndentStr + "  "
							if hostStr, ok := hv.(string); ok {
								escapedHost := s.escapeYAMLString(hostStr)
								builder.WriteString(fmt.Sprintf("%s- %s\n", subSubSubIndentStr, escapedHost))
							} else if hostSlice, ok := hv.([]string); ok {
								for _, h := range hostSlice {
									escapedHost := s.escapeYAMLString(h)
									builder.WriteString(fmt.Sprintf("%s- %s\n", subSubSubIndentStr, escapedHost))
								}
							} else if hostSlice, ok := hv.([]interface{}); ok {
								for _, h := range hostSlice {
									escapedHost := s.escapeYAMLString(fmt.Sprintf("%v", h))
									builder.WriteString(fmt.Sprintf("%s- %s\n", subSubSubIndentStr, escapedHost))
								}
							}
						}
					}
					continue
				}
			}

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
	default:
		return ""
	}
}

// nodeToSSRLink 将节点转换为 SSR 链接
func (s *ConfigUpdateService) nodeToSSRLink(node *ProxyNode) string {
	if node.Type != "ssr" && node.Type != "ss" {
		return ""
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

// UpdateSubscriptionConfig 更新订阅配置 (保留用于兼容性，但逻辑简化)
func (s *ConfigUpdateService) UpdateSubscriptionConfig(subscriptionURL string) error {
	// 简单的验证存在性
	var count int64
	s.db.Model(&models.Subscription{}).Where("subscription_url = ?", subscriptionURL).Count(&count)
	if count == 0 {
		return fmt.Errorf("订阅不存在")
	}
	return nil
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

	// 获取配置
	config, err := s.getConfig()
	if err != nil {
		return err
	}

	urls := config["urls"].([]string)
	if len(urls) == 0 {
		return fmt.Errorf("未配置节点源URL")
	}

	// 1. 获取节点
	nodes, err := s.FetchNodesFromURLs(urls)
	if err != nil {
		return err
	}

	if len(nodes) == 0 {
		return fmt.Errorf("未获取到有效节点")
	}

	// 2. 解析节点并导入数据库
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

		// 生成去重键
		key := fmt.Sprintf("%s:%s:%d", node.Type, node.Server, node.Port)
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

	s.importNodesToDatabase(proxies)
	s.updateLastUpdateTime()
	return nil
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
		default:
			result[key] = value
		}
	}
	return result, nil
}

// importNodesToDatabase 将节点导入到数据库的 nodes 表
func (s *ConfigUpdateService) importNodesToDatabase(proxies []*ProxyNode) int {
	importedCount := 0

	for _, node := range proxies {
		configJSON, _ := json.Marshal(node)
		configStr := string(configJSON)

		// 检查是否已存在
		var count int64
		s.db.Model(&models.Node{}).Where("type = ? AND name = ?", node.Type, node.Name).Count(&count)

		if count == 0 {
			newNode := models.Node{
				Name:     node.Name,
				Type:     node.Type,
				Status:   "online",
				IsActive: true,
				IsManual: false,
				Config:   &configStr,
				// Link:     s.nodeToLink(node), // 模型中没有 Link 字段
			}
			s.db.Create(&newNode)
			importedCount++
		}
	}
	return importedCount
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
	} else {
		config.Value = "[]"
		return s.db.Save(&config).Error
	}
}

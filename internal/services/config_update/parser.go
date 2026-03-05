package config_update

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// ==========================================
// 解析器池
// ==========================================

type ParseResult struct {
	Node *ProxyNode
	Err  error
	Link string
}

type ParserPool struct {
	workers int
	cache   *ParseCache
}

func NewParserPool(workers int) *ParserPool {
	if workers <= 0 {
		workers = 10 // 默认10个worker
	}
	return &ParserPool{
		workers: workers,
		cache:   NewParseCache(),
	}
}

func (p *ParserPool) ParseLinks(links []string) []ParseResult {
	if len(links) == 0 {
		return []ParseResult{}
	}

	taskChan := make(chan string, len(links))
	resultChan := make(chan ParseResult, len(links))
	var wg sync.WaitGroup

	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for link := range taskChan {
				if cached, ok := p.cache.Get(link); ok {
					resultChan <- ParseResult{
						Node: cached,
						Err:  nil,
						Link: link,
					}
					continue
				}

				node, err := ParseNodeLink(link)
				if err != nil {
					resultChan <- ParseResult{
						Node: nil,
						Err:  fmt.Errorf("解析失败 [链接: %s...]: %w", truncateLink(link, 50), err),
						Link: link,
					}
					continue
				}

				p.cache.Set(link, node)

				resultChan <- ParseResult{
					Node: node,
					Err:  nil,
					Link: link,
				}
			}
		}()
	}

	go func() {
		defer close(taskChan)
		for _, link := range links {
			taskChan <- link
		}
	}()

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	results := make([]ParseResult, 0, len(links))
	for result := range resultChan {
		results = append(results, result)
	}

	return results
}

func truncateLink(link string, maxLen int) string {
	if len(link) > maxLen {
		return link[:maxLen] + "..."
	}
	return link
}

// ==========================================
// 解析缓存
// ==========================================

type ParseCache struct {
	cache map[string]*ProxyNode
	mu    sync.RWMutex
	ttl   time.Duration
	times map[string]time.Time
}

func NewParseCache() *ParseCache {
	return &ParseCache{
		cache: make(map[string]*ProxyNode),
		times: make(map[string]time.Time),
		ttl:   5 * time.Minute, // 5分钟TTL
	}
}

func (c *ParseCache) Get(key string) (*ProxyNode, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	node, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	if t, ok := c.times[key]; ok {
		if time.Since(t) > c.ttl {
			go c.delete(key)
			return nil, false
		}
	}

	return node, true
}

func (c *ParseCache) Set(key string, node *ProxyNode) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = node
	c.times[key] = time.Now()

	if len(c.cache) > 1000 {
		c.cleanup()
	}
}

func (c *ParseCache) delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, key)
	delete(c.times, key)
}

func (c *ParseCache) cleanup() {
	now := time.Now()
	for key, t := range c.times {
		if now.Sub(t) > c.ttl {
			delete(c.cache, key)
			delete(c.times, key)
		}
	}
}

// ==========================================
// Clash YAML 解析
// ==========================================

// parseClashYAML 尝试解析 Clash YAML 格式的配置文件
func (s *ConfigUpdateService) parseClashYAML(content string) []string {
	var clashConfig struct {
		Proxies []map[string]interface{} `yaml:"proxies"`
	}

	if err := yaml.Unmarshal([]byte(content), &clashConfig); err != nil {
		// YAML 解析失败是正常的，不是所有内容都是 YAML 格式
		return nil
	}

	if len(clashConfig.Proxies) == 0 {
		// YAML 解析成功但无 proxies 字段，继续用正则提取
		return nil
	}

	s.log("INFO", fmt.Sprintf("✓ 检测到 Clash YAML 格式 (%d 个代理)", len(clashConfig.Proxies)))

	var links []string
	var nodeNames []string
	var failedCount int
	for i, proxy := range clashConfig.Proxies {
		// 将 Clash YAML 格式的代理转换为节点链接
		if link := s.convertClashProxyToLink(proxy); link != "" {
			links = append(links, link)
			if name, ok := proxy["name"].(string); ok {
				nodeNames = append(nodeNames, name)
			}
		} else {
			failedCount++
			if failedCount <= 3 {
				s.log("DEBUG", fmt.Sprintf("节点 %d 转换失败: %v", i+1, proxy))
			}
		}
	}

	if failedCount > 0 {
		s.log("WARN", fmt.Sprintf("有 %d 个节点转换失败", failedCount))
	}

	s.log("INFO", fmt.Sprintf("✓ 成功转换 %d 个节点", len(links)))

	// 显示采集到的节点名称
	if len(nodeNames) > 0 {
		s.logSeparator()
		s.log("INFO", "📋 采集到的节点:")
		for i, name := range nodeNames {
			s.log("INFO", fmt.Sprintf("  %d. %s", i+1, name))
		}
		s.logSeparator()
	}

	return links
}

// convertClashProxyToLink 将 Clash 代理配置转换为节点链接
func (s *ConfigUpdateService) convertClashProxyToLink(proxy map[string]interface{}) string {
	proxyType, _ := proxy["type"].(string)
	name, _ := proxy["name"].(string)
	server, _ := proxy["server"].(string)
	port := int(getFloat(proxy, "port"))

	if server == "" || port == 0 {
		return ""
	}

	switch proxyType {
	case "vmess":
		return s.buildVMessLink(proxy, name, server, port)
	case "vless":
		return s.buildVLESSLink(proxy, name, server, port)
	case "trojan":
		return s.buildTrojanLink(proxy, name, server, port)
	case "ss", "shadowsocks":
		return s.buildSSLink(proxy, name, server, port)
	case "hysteria2":
		return s.buildHysteria2Link(proxy, name, server, port)
	default:
		s.log("DEBUG", fmt.Sprintf("不支持的代理类型: %s", proxyType))
		return ""
	}
}

// buildVMessLink 构建 VMess 链接
func (s *ConfigUpdateService) buildVMessLink(proxy map[string]interface{}, name, server string, port int) string {
	uuid, _ := proxy["uuid"].(string)
	if uuid == "" {
		return ""
	}

	vmessData := map[string]interface{}{
		"v":    "2",
		"ps":   name,
		"add":  server,
		"port": port,
		"id":   uuid,
		"aid":  int(getFloat(proxy, "alterId")),
		"net":  getString(proxy, "network", "tcp"),
		"type": "none",
		"host": "",
		"path": "",
		"tls":  "",
	}

	if tls, _ := proxy["tls"].(bool); tls {
		vmessData["tls"] = "tls"
		if sni, ok := proxy["servername"].(string); ok {
			vmessData["sni"] = sni
		}
	}

	// 处理传输层配置
	if wsOpts, ok := proxy["ws-opts"].(map[string]interface{}); ok {
		if path, ok := wsOpts["path"].(string); ok {
			vmessData["path"] = path
		}
		if headers, ok := wsOpts["headers"].(map[string]interface{}); ok {
			if host, ok := headers["Host"].(string); ok {
				vmessData["host"] = host
			}
		}
	}

	jsonData, _ := json.Marshal(vmessData)
	encoded := base64.StdEncoding.EncodeToString(jsonData)
	return "vmess://" + encoded
}

// buildVLESSLink 构建 VLESS 链接
func (s *ConfigUpdateService) buildVLESSLink(proxy map[string]interface{}, name, server string, port int) string {
	uuid, _ := proxy["uuid"].(string)
	if uuid == "" {
		return ""
	}

	params := url.Values{}
	params.Set("type", getString(proxy, "network", "tcp"))
	params.Set("security", "none")

	if tls, _ := proxy["tls"].(bool); tls {
		params.Set("security", "tls")
		if sni, ok := proxy["servername"].(string); ok {
			params.Set("sni", sni)
		}
	}

	// 处理传输层配置
	if wsOpts, ok := proxy["ws-opts"].(map[string]interface{}); ok {
		if path, ok := wsOpts["path"].(string); ok {
			params.Set("path", path)
		}
		if headers, ok := wsOpts["headers"].(map[string]interface{}); ok {
			if host, ok := headers["Host"].(string); ok {
				params.Set("host", host)
			}
		}
	}

	link := fmt.Sprintf("vless://%s@%s:%d?%s#%s", uuid, server, port, params.Encode(), url.QueryEscape(name))
	return link
}

// buildTrojanLink 构建 Trojan 链接
func (s *ConfigUpdateService) buildTrojanLink(proxy map[string]interface{}, name, server string, port int) string {
	password, _ := proxy["password"].(string)
	if password == "" {
		return ""
	}

	params := url.Values{}
	params.Set("type", getString(proxy, "network", "tcp"))

	if sni, ok := proxy["sni"].(string); ok {
		params.Set("sni", sni)
	} else if sni, ok := proxy["servername"].(string); ok {
		params.Set("sni", sni)
	}

	// 处理传输层配置
	if wsOpts, ok := proxy["ws-opts"].(map[string]interface{}); ok {
		if path, ok := wsOpts["path"].(string); ok {
			params.Set("path", path)
		}
		if headers, ok := wsOpts["headers"].(map[string]interface{}); ok {
			if host, ok := headers["Host"].(string); ok {
				params.Set("host", host)
			}
		}
	}

	link := fmt.Sprintf("trojan://%s@%s:%d?%s#%s", password, server, port, params.Encode(), url.QueryEscape(name))
	return link
}

// buildSSLink 构建 Shadowsocks 链接
func (s *ConfigUpdateService) buildSSLink(proxy map[string]interface{}, name, server string, port int) string {
	password, _ := proxy["password"].(string)
	cipher, _ := proxy["cipher"].(string)

	if password == "" || cipher == "" {
		return ""
	}

	auth := fmt.Sprintf("%s:%s", cipher, password)
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	link := fmt.Sprintf("ss://%s@%s:%d#%s", encoded, server, port, url.QueryEscape(name))

	return link
}

// buildHysteria2Link 构建 Hysteria2 链接
func (s *ConfigUpdateService) buildHysteria2Link(proxy map[string]interface{}, name, server string, port int) string {
	password, _ := proxy["password"].(string)
	if password == "" {
		return ""
	}

	params := url.Values{}
	if sni, ok := proxy["sni"].(string); ok {
		params.Set("sni", sni)
	}

	link := fmt.Sprintf("hysteria2://%s@%s:%d?%s#%s", password, server, port, params.Encode(), url.QueryEscape(name))
	return link
}

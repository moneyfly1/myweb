package config_update

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"

	"gopkg.in/yaml.v3"
)

// parseClashYAML 尝试解析 Clash YAML 格式的配置文件
func (s *ConfigUpdateService) parseClashYAML(content string) []string {
	var clashConfig struct {
		Proxies []map[string]interface{} `yaml:"proxies"`
	}

	if err := yaml.Unmarshal([]byte(content), &clashConfig); err != nil {
		s.log("DEBUG", fmt.Sprintf("YAML 解析失败: %v", err))
		return nil
	}

	if len(clashConfig.Proxies) == 0 {
		s.log("DEBUG", "YAML 解析成功但未找到 proxies 字段")
		return nil
	}

	s.log("INFO", fmt.Sprintf("检测到 Clash YAML 格式，包含 %d 个代理节点", len(clashConfig.Proxies)))

	var links []string
	var failedCount int
	for i, proxy := range clashConfig.Proxies {
		// 将 Clash YAML 格式的代理转换为节点链接
		if link := s.convertClashProxyToLink(proxy); link != "" {
			links = append(links, link)
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

	s.log("INFO", fmt.Sprintf("成功转换 %d 个节点链接", len(links)))
	return links
}

// convertClashProxyToLink 将 Clash 代理配置转换为节点链接
func (s *ConfigUpdateService) convertClashProxyToLink(proxy map[string]interface{}) string {
	proxyType, _ := proxy["type"].(string)
	name, _ := proxy["name"].(string)
	server, _ := proxy["server"].(string)
	port := int(getFloat(proxy, "port"))

	s.log("DEBUG", fmt.Sprintf("convertClashProxyToLink: type=%s, name=%s, server=%s, port=%d", 
		proxyType, name, server, port))

	if server == "" || port == 0 {
		s.log("DEBUG", fmt.Sprintf("convertClashProxyToLink 失败: server=%q, port=%d", server, port))
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
	
	s.log("DEBUG", fmt.Sprintf("buildSSLink: name=%s, server=%s, port=%d, cipher=%s, password=%s", 
		name, server, port, cipher, password))
	
	if password == "" || cipher == "" {
		s.log("DEBUG", fmt.Sprintf("buildSSLink 失败: password=%q, cipher=%q", password, cipher))
		return ""
	}

	auth := fmt.Sprintf("%s:%s", cipher, password)
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	link := fmt.Sprintf("ss://%s@%s:%d#%s", encoded, server, port, url.QueryEscape(name))
	
	s.log("DEBUG", fmt.Sprintf("buildSSLink 成功: %s", link[:50]+"..."))
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

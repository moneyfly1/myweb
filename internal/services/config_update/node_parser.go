package config_update

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ProxyNode 代理节点结构
type ProxyNode struct {
	Name     string                 `yaml:"name"`
	Type     string                 `yaml:"type"`
	Server   string                 `yaml:"server"`
	Port     int                    `yaml:"port"`
	UUID     string                 `yaml:"uuid,omitempty"`
	Password string                 `yaml:"password,omitempty"`
	Cipher   string                 `yaml:"cipher,omitempty"`
	Network  string                 `yaml:"network,omitempty"`
	TLS      bool                   `yaml:"tls,omitempty"`
	UDP      bool                   `yaml:"udp,omitempty"`
	Options  map[string]interface{} `yaml:",inline"`
}

// ParseNodeLink 解析节点链接
func ParseNodeLink(link string) (*ProxyNode, error) {
	link = strings.TrimSpace(link)

	if strings.HasPrefix(link, "vmess://") {
		return parseVMess(link)
	} else if strings.HasPrefix(link, "vless://") {
		return parseVLESS(link)
	} else if strings.HasPrefix(link, "trojan://") {
		return parseTrojan(link)
	} else if strings.HasPrefix(link, "ss://") {
		return parseShadowsocks(link)
	} else if strings.HasPrefix(link, "ssr://") {
		return parseSSR(link)
	} else if strings.HasPrefix(link, "hysteria://") {
		return parseHysteria(link)
	} else if strings.HasPrefix(link, "hysteria2://") {
		return parseHysteria2(link)
	}

	return nil, fmt.Errorf("不支持的协议: %s", link[:10])
}

// parseVMess 解析 VMess 链接
func parseVMess(link string) (*ProxyNode, error) {
	encoded := strings.TrimPrefix(link, "vmess://")

	// 尝试 Base64 解码
	decoded, err := safeBase64Decode(encoded)
	if err != nil {
		return nil, fmt.Errorf("Base64 解码失败: %v", err)
	}

	// 解析 JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(decoded), &data); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %v", err)
	}

	// 提取基本信息
	server, _ := data["add"].(string)
	port, _ := data["port"].(float64)
	uuid, _ := data["id"].(string)
	alterID, _ := data["aid"].(float64)
	network, _ := data["net"].(string)
	if network == "" {
		network = "tcp"
	}

	// 构建节点
	node := &ProxyNode{
		Name:    getString(data, "ps", fmt.Sprintf("VMess-%s:%d", server, int(port))),
		Type:    "vmess",
		Server:  server,
		Port:    int(port),
		UUID:    uuid,
		Network: network,
		UDP:     true,
		Options: make(map[string]interface{}),
	}

	// TLS 配置
	if tls, ok := data["tls"].(string); ok && tls == "tls" {
		node.TLS = true
		node.Options["skip-cert-verify"] = getBool(data, "allowInsecure", false)
		if sni, ok := data["sni"].(string); ok && sni != "" {
			node.Options["servername"] = sni
		}
	}

	// AlterID
	if alterID > 0 {
		node.Options["alterId"] = int(alterID)
	}

	// 网络配置
	if network == "ws" {
		node.Options["ws-opts"] = map[string]interface{}{
			"path": getString(data, "path", "/"),
			"headers": map[string]string{
				"Host": getString(data, "host", server),
			},
		}
	} else if network == "grpc" {
		node.Options["grpc-opts"] = map[string]interface{}{
			"grpc-service-name": getString(data, "path", ""),
		}
	} else if network == "h2" {
		node.Options["h2-opts"] = map[string]interface{}{
			"path": getString(data, "path", "/"),
			"host": []string{getString(data, "host", server)},
		}
	}

	return node, nil
}

// parseVLESS 解析 VLESS 链接
func parseVLESS(link string) (*ProxyNode, error) {
	parsed, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	uuid := parsed.User.Username()
	if uuid == "" {
		return nil, fmt.Errorf("缺少 UUID")
	}

	query := parsed.Query()
	network := query.Get("type")
	if network == "" {
		network = "tcp"
	}

	security := query.Get("security")
	if security == "" {
		security = "none"
	}

	node := &ProxyNode{
		Name:    getFragment(parsed, fmt.Sprintf("VLESS-%s:%s", parsed.Hostname(), parsed.Port())),
		Type:    "vless",
		Server:  parsed.Hostname(),
		Port:    getPort(parsed),
		UUID:    uuid,
		Network: network,
		UDP:     true,
		Options: make(map[string]interface{}),
	}

	// TLS 配置
	if security == "tls" || security == "xtls" || security == "reality" {
		node.TLS = true
		node.Options["skip-cert-verify"] = query.Get("allowInsecure") == "1"
		if sni := query.Get("sni"); sni != "" {
			node.Options["servername"] = sni
		} else {
			node.Options["servername"] = parsed.Hostname()
		}

		// Reality 配置
		if security == "reality" || query.Get("pbk") != "" {
			if pbk := query.Get("pbk"); pbk != "" {
				node.Options["reality-opts"] = map[string]interface{}{
					"public-key": pbk,
					"short-id":   query.Get("sid"),
				}
			}
		}

		if flow := query.Get("flow"); flow != "" {
			node.Options["flow"] = flow
		}
	}

	// 网络配置
	if network == "ws" {
		node.Options["ws-opts"] = map[string]interface{}{
			"path": query.Get("path"),
			"headers": map[string]string{
				"Host": query.Get("host"),
			},
		}
	} else if network == "grpc" {
		node.Options["grpc-opts"] = map[string]interface{}{
			"grpc-service-name": query.Get("serviceName"),
		}
	}

	return node, nil
}

// parseTrojan 解析 Trojan 链接
func parseTrojan(link string) (*ProxyNode, error) {
	parsed, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	password := parsed.User.Username()
	if password == "" {
		return nil, fmt.Errorf("缺少密码")
	}

	query := parsed.Query()
	network := query.Get("type")
	if network == "" {
		network = "tcp"
	}

	node := &ProxyNode{
		Name:     getFragment(parsed, fmt.Sprintf("Trojan-%s:%s", parsed.Hostname(), parsed.Port())),
		Type:     "trojan",
		Server:   parsed.Hostname(),
		Port:     getPort(parsed),
		Password: password,
		Network:  network,
		UDP:      true,
		TLS:      true,
		Options:  make(map[string]interface{}),
	}

	// TLS 配置
	node.Options["skip-cert-verify"] = query.Get("allowInsecure") == "1"
	if sni := query.Get("sni"); sni != "" {
		node.Options["servername"] = sni
	} else {
		node.Options["servername"] = parsed.Hostname()
	}

	// 网络配置
	if network == "ws" {
		node.Options["ws-opts"] = map[string]interface{}{
			"path": query.Get("path"),
			"headers": map[string]string{
				"Host": query.Get("host"),
			},
		}
	} else if network == "grpc" {
		node.Options["grpc-opts"] = map[string]interface{}{
			"grpc-service-name": query.Get("serviceName"),
		}
	}

	return node, nil
}

// parseShadowsocks 解析 Shadowsocks 链接
func parseShadowsocks(link string) (*ProxyNode, error) {
	parsed, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	// 解析认证信息
	var method, password string
	if parsed.User != nil {
		authInfo := parsed.User.String()
		if strings.Contains(authInfo, ":") {
			parts := strings.SplitN(authInfo, ":", 2)
			method = parts[0]
			password = parts[1]
		} else {
			// 可能是 Base64 编码的 method:password
			decoded, err := safeBase64Decode(authInfo)
			if err == nil && strings.Contains(decoded, ":") {
				parts := strings.SplitN(decoded, ":", 2)
				method = parts[0]
				password = parts[1]
			} else {
				method = authInfo
			}
		}
	}

	if method == "" || password == "" {
		return nil, fmt.Errorf("缺少认证信息")
	}

	node := &ProxyNode{
		Name:     getFragment(parsed, fmt.Sprintf("SS-%s:%s", parsed.Hostname(), parsed.Port())),
		Type:     "ss",
		Server:   parsed.Hostname(),
		Port:     getPort(parsed),
		Cipher:   method,
		Password: password,
		Options:  make(map[string]interface{}),
	}

	return node, nil
}

// parseSSR 解析 SSR 链接
func parseSSR(link string) (*ProxyNode, error) {
	encoded := strings.TrimPrefix(link, "ssr://")
	decoded, err := safeBase64Decode(encoded)
	if err != nil {
		return nil, err
	}

	// 格式: server:port:protocol:method:obfs:password_base64/?params_base64#name_base64
	parts := strings.SplitN(decoded, "/?", 2)
	if len(parts) < 1 {
		return nil, fmt.Errorf("SSR 格式错误")
	}

	mainPart := parts[0]
	mainParts := strings.Split(mainPart, ":")
	if len(mainParts) < 6 {
		return nil, fmt.Errorf("SSR 格式错误")
	}

	server := mainParts[0]
	port, _ := strconv.Atoi(mainParts[1])
	protocol := mainParts[2]
	method := mainParts[3]
	obfs := mainParts[4]
	passwordB64 := strings.Join(mainParts[5:], ":")

	password, err := safeBase64Decode(passwordB64)
	if err != nil {
		return nil, fmt.Errorf("密码解码失败: %v", err)
	}

	node := &ProxyNode{
		Name:     fmt.Sprintf("SSR-%s:%d", server, port),
		Type:     "ssr",
		Server:   server,
		Port:     port,
		Password: password,
		Cipher:   method,
		Options: map[string]interface{}{
			"protocol":       protocol,
			"obfs":           obfs,
			"protocol-param": "",
			"obfs-param":     "",
		},
	}

	return node, nil
}

// parseHysteria 解析 Hysteria v1 链接
func parseHysteria(link string) (*ProxyNode, error) {
	parsed, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	query := parsed.Query()

	node := &ProxyNode{
		Name:    getFragment(parsed, fmt.Sprintf("Hysteria-%s:%s", parsed.Hostname(), parsed.Port())),
		Type:    "hysteria",
		Server:  parsed.Hostname(),
		Port:    getPort(parsed),
		Options: make(map[string]interface{}),
	}

	if auth := query.Get("auth"); auth != "" {
		node.Options["auth"] = auth
	}

	if up := query.Get("upmbps"); up != "" {
		node.Options["up"] = up + " mbps"
	}
	if down := query.Get("downmbps"); down != "" {
		node.Options["down"] = down + " mbps"
	}

	node.Options["skip-cert-verify"] = query.Get("insecure") == "1"

	return node, nil
}

// parseHysteria2 解析 Hysteria2 链接
func parseHysteria2(link string) (*ProxyNode, error) {
	parsed, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	password := parsed.User.Username()
	query := parsed.Query()

	node := &ProxyNode{
		Name:     getFragment(parsed, fmt.Sprintf("Hysteria2-%s:%s", parsed.Hostname(), parsed.Port())),
		Type:     "hysteria2",
		Server:   parsed.Hostname(),
		Port:     getPort(parsed),
		Password: password,
		Options:  make(map[string]interface{}),
	}

	if up := query.Get("mbpsUp"); up != "" {
		node.Options["up"] = up + " mbps"
	}
	if down := query.Get("mbpsDown"); down != "" {
		node.Options["down"] = down + " mbps"
	}

	node.Options["skip-cert-verify"] = query.Get("insecure") == "1"

	return node, nil
}

// 辅助函数
func safeBase64Decode(s string) (string, error) {
	// 清理文本
	clean := strings.ReplaceAll(s, " ", "")
	clean = strings.ReplaceAll(clean, "\n", "")
	clean = strings.ReplaceAll(clean, "\r", "")
	clean = strings.ReplaceAll(clean, "-", "+")
	clean = strings.ReplaceAll(clean, "_", "/")

	// 补全 padding
	if len(clean)%4 != 0 {
		clean += strings.Repeat("=", 4-len(clean)%4)
	}

	decoded, err := base64.StdEncoding.DecodeString(clean)
	if err != nil {
		return "", err
	}

	return string(decoded), nil
}

func getString(m map[string]interface{}, key, defaultValue string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultValue
}

func getBool(m map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
		if s, ok := v.(string); ok {
			return s == "1" || s == "true"
		}
	}
	return defaultValue
}

func getFragment(parsed *url.URL, defaultValue string) string {
	if parsed.Fragment != "" {
		decoded, err := url.QueryUnescape(parsed.Fragment)
		if err == nil {
			return decoded
		}
		return parsed.Fragment
	}
	return defaultValue
}

func getPort(parsed *url.URL) int {
	portStr := parsed.Port()
	if portStr == "" {
		// 根据协议推断默认端口
		switch parsed.Scheme {
		case "vmess", "vless", "trojan":
			return 443
		case "ss", "ssr":
			return 8388
		case "hysteria", "hysteria2":
			return 443
		default:
			return 443
		}
	}
	port, _ := strconv.Atoi(portStr)
	return port
}

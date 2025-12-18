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
	} else if strings.HasPrefix(link, "tuic://") {
		return parseTUIC(link)
	} else if strings.HasPrefix(link, "naive+https://") || strings.HasPrefix(link, "naive://") {
		return parseNaive(link)
	} else if strings.HasPrefix(link, "anytls://") {
		return parseAnytls(link)
	}

	if len(link) > 10 {
		return nil, fmt.Errorf("不支持的协议: %s", link[:10])
	}
	return nil, fmt.Errorf("不支持的协议")
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

	// 处理端口（可能是数字或字符串）
	var port int
	if portFloat, ok := data["port"].(float64); ok {
		port = int(portFloat)
	} else if portStr, ok := data["port"].(string); ok {
		if parsedPort, err := strconv.Atoi(portStr); err == nil {
			port = parsedPort
		}
	}

	// 验证端口
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("无效的端口: %v", data["port"])
	}

	uuid, _ := data["id"].(string)
	if uuid == "" {
		return nil, fmt.Errorf("缺少 UUID (id)")
	}

	// 验证服务器地址
	if server == "" {
		return nil, fmt.Errorf("缺少服务器地址 (add)")
	}

	// 处理 alterID（可能是数字或字符串）
	var alterID float64
	if aidFloat, ok := data["aid"].(float64); ok {
		alterID = aidFloat
	} else if aidStr, ok := data["aid"].(string); ok {
		if parsedAid, err := strconv.ParseFloat(aidStr, 64); err == nil {
			alterID = parsedAid
		}
	}

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
	} else if network == "httpupgrade" {
		// HTTP Upgrade 配置
		node.Options["http-opts"] = map[string]interface{}{
			"path": getString(data, "path", "/"),
			"headers": map[string]string{
				"Host": getString(data, "host", server),
			},
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
			realityOpts := make(map[string]interface{})
			if pbk := query.Get("pbk"); pbk != "" {
				realityOpts["public-key"] = pbk
			}
			if sid := query.Get("sid"); sid != "" {
				realityOpts["short-id"] = sid
			}
			if len(realityOpts) > 0 {
				node.Options["reality-opts"] = realityOpts
			}
		}

		// Flow 配置（用于 XTLS）
		if flow := query.Get("flow"); flow != "" {
			node.Options["flow"] = flow
		}
	}

	// 网络配置
	if network == "ws" {
		wsOpts := make(map[string]interface{})
		if path := query.Get("path"); path != "" {
			wsOpts["path"] = path
		}
		headers := make(map[string]string)
		if host := query.Get("host"); host != "" {
			headers["Host"] = host
		}
		if len(headers) > 0 {
			wsOpts["headers"] = headers
		}
		if len(wsOpts) > 0 {
			node.Options["ws-opts"] = wsOpts
		}
	} else if network == "grpc" {
		grpcOpts := make(map[string]interface{})
		if serviceName := query.Get("serviceName"); serviceName != "" {
			grpcOpts["grpc-service-name"] = serviceName
		} else if path := query.Get("path"); path != "" {
			// 如果没有 serviceName，使用 path 作为 serviceName
			grpcOpts["grpc-service-name"] = path
		}
		if len(grpcOpts) > 0 {
			node.Options["grpc-opts"] = grpcOpts
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

	// 解析参数部分
	var nodeName string
	protocolParam := ""
	obfsParam := ""

	if len(parts) > 1 {
		// 解析 URL 参数
		paramsPart := parts[1]
		// 移除可能的 #name 部分
		if hashIdx := strings.Index(paramsPart, "#"); hashIdx != -1 {
			nameB64 := paramsPart[hashIdx+1:]
			paramsPart = paramsPart[:hashIdx]
			if decodedName, err := safeBase64Decode(nameB64); err == nil {
				nodeName = decodedName
			}
		}

		// 解析 URL 参数
		params, err := url.ParseQuery(paramsPart)
		if err == nil {
			// 解析 remarks（节点名称）
			if remarks := params.Get("remarks"); remarks != "" {
				if decodedRemarks, err := safeBase64Decode(remarks); err == nil {
					nodeName = decodedRemarks
				} else {
					// 如果解码失败，尝试直接使用
					nodeName = remarks
				}
			}

			// 解析 protoparam
			if protoparam := params.Get("protoparam"); protoparam != "" {
				if decoded, err := safeBase64Decode(protoparam); err == nil {
					protocolParam = decoded
				} else {
					protocolParam = protoparam
				}
			}

			// 解析 obfsparam
			if obfsparam := params.Get("obfsparam"); obfsparam != "" {
				if decoded, err := safeBase64Decode(obfsparam); err == nil {
					obfsParam = decoded
				} else {
					obfsParam = obfsparam
				}
			}
		}
	}

	// 如果没有节点名称，使用默认格式
	if nodeName == "" {
		nodeName = fmt.Sprintf("SSR-%s:%d", server, port)
	}

	node := &ProxyNode{
		Name:     nodeName,
		Type:     "ssr",
		Server:   server,
		Port:     port,
		Password: password,
		Cipher:   method,
		Options: map[string]interface{}{
			"protocol":       protocol,
			"obfs":           obfs,
			"protocol-param": protocolParam,
			"obfs-param":     obfsParam,
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
			// 如果 fragment 包含有意义的信息，使用它；否则使用默认值
			if decoded != "" && decoded != parsed.Fragment {
				return decoded
			}
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
		case "hysteria", "hysteria2", "tuic", "anytls":
			return 443
		case "https", "naive":
			return 443
		default:
			return 443
		}
	}
	port, _ := strconv.Atoi(portStr)
	return port
}

// parseTUIC 解析 TUIC 链接
func parseTUIC(link string) (*ProxyNode, error) {
	parsed, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	// TUIC 格式: tuic://uuid:password@host:port?params#name
	userInfo := parsed.User
	if userInfo == nil {
		return nil, fmt.Errorf("缺少认证信息")
	}

	uuid := userInfo.Username()
	password, _ := userInfo.Password()
	if uuid == "" {
		return nil, fmt.Errorf("缺少 UUID")
	}

	query := parsed.Query()

	node := &ProxyNode{
		Name:     getFragment(parsed, fmt.Sprintf("TUIC-%s:%s", parsed.Hostname(), parsed.Port())),
		Type:     "tuic",
		Server:   parsed.Hostname(),
		Port:     getPort(parsed),
		UUID:     uuid,
		Password: password,
		UDP:      true,
		TLS:      true,
		Options:  make(map[string]interface{}),
	}

	// TLS 配置
	if sni := query.Get("sni"); sni != "" {
		node.Options["servername"] = sni
	} else {
		node.Options["servername"] = parsed.Hostname()
	}

	// 其他配置
	if alpn := query.Get("alpn"); alpn != "" {
		node.Options["alpn"] = []string{alpn}
	}
	if cc := query.Get("congestion_control"); cc != "" {
		node.Options["congestion_control"] = cc
	}
	if udpRelayMode := query.Get("udp_relay_mode"); udpRelayMode != "" {
		node.Options["udp_relay_mode"] = udpRelayMode
	}
	node.Options["skip-cert-verify"] = query.Get("allow_insecure") == "1" || query.Get("allow_insecure") == "true"

	return node, nil
}

// parseNaive 解析 Naive 链接
func parseNaive(link string) (*ProxyNode, error) {
	// 处理 naive+https:// 和 naive:// 格式
	link = strings.Replace(link, "naive+https://", "https://", 1)
	link = strings.Replace(link, "naive://", "https://", 1)

	parsed, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	userInfo := parsed.User
	if userInfo == nil {
		return nil, fmt.Errorf("缺少认证信息")
	}

	username := userInfo.Username()
	password, _ := userInfo.Password()
	if username == "" || password == "" {
		return nil, fmt.Errorf("缺少用户名或密码")
	}

	query := parsed.Query()

	node := &ProxyNode{
		Name:     getFragment(parsed, fmt.Sprintf("Naive-%s:%s", parsed.Hostname(), parsed.Port())),
		Type:     "naive",
		Server:   parsed.Hostname(),
		Port:     getPort(parsed),
		UUID:     username, // 使用 username 作为标识
		Password: password,
		TLS:      true,
		Options:  make(map[string]interface{}),
	}

	// TLS 配置
	if sni := query.Get("sni"); sni != "" {
		node.Options["servername"] = sni
	} else {
		node.Options["servername"] = parsed.Hostname()
	}

	// Padding 配置
	if padding := query.Get("padding"); padding != "" {
		node.Options["padding"] = padding == "true" || padding == "1"
	}

	node.Options["skip-cert-verify"] = query.Get("insecure") == "1" || query.Get("insecure") == "true"

	return node, nil
}

// parseAnytls 解析 Anytls 链接
func parseAnytls(link string) (*ProxyNode, error) {
	parsed, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	uuid := parsed.User.Username()
	if uuid == "" {
		return nil, fmt.Errorf("缺少 UUID")
	}

	query := parsed.Query()

	node := &ProxyNode{
		Name:    getFragment(parsed, fmt.Sprintf("Anytls-%s:%s", parsed.Hostname(), parsed.Port())),
		Type:    "anytls",
		Server:  parsed.Hostname(),
		Port:    getPort(parsed),
		UUID:    uuid,
		UDP:     true,
		TLS:     true,
		Options: make(map[string]interface{}),
	}

	// TLS 配置
	if peer := query.Get("peer"); peer != "" {
		node.Options["servername"] = peer
	} else if sni := query.Get("sni"); sni != "" {
		node.Options["servername"] = sni
	} else {
		node.Options["servername"] = parsed.Hostname()
	}

	node.Options["skip-cert-verify"] = query.Get("insecure") == "1" || query.Get("insecure") == "true" || query.Get("insecure") == "0"

	return node, nil
}

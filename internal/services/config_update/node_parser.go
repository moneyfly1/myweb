package config_update

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ProxyNode 代理节点结构体
type ProxyNode struct {
	Name     string                 `yaml:"name" json:"Name"`
	Type     string                 `yaml:"type" json:"Type"`
	Server   string                 `yaml:"server" json:"Server"`
	Port     int                    `yaml:"port" json:"Port"`
	UUID     string                 `yaml:"uuid,omitempty" json:"UUID,omitempty"`
	Password string                 `yaml:"password,omitempty" json:"Password,omitempty"`
	Cipher   string                 `yaml:"cipher,omitempty" json:"Cipher,omitempty"`
	Network  string                 `yaml:"network,omitempty" json:"Network,omitempty"`
	TLS      bool                   `yaml:"tls,omitempty" json:"TLS,omitempty"`
	UDP      bool                   `yaml:"udp,omitempty" json:"UDP,omitempty"`
	Options  map[string]interface{} `yaml:",inline" json:"Options,omitempty"`
}

type nodeParser func(string) (*ProxyNode, error)

var protocolParsers = map[string]nodeParser{
	"vmess://":       parseVMess,
	"vless://":       parseVLESS,
	"trojan://":      parseTrojan,
	"ss://":          parseShadowsocks,
	"ssr://":         parseSSR,
	"hysteria://":    parseHysteria,
	"hysteria2://":   parseHysteria2,
	"tuic://":        parseTUIC,
	"naive+https://": parseNaive,
	"naive://":       parseNaive,
	"anytls://":      parseAnytls,
	"socks5://":      parseSOCKS,
	"socks://":       parseSOCKS,
	"http://":        parseHTTP,
	"https://":       parseHTTP,
	"wg://":          parseWireGuard,
}

// ==========================================
// 核心入口
// ==========================================

// ParseNodeLink 解析节点链接的主入口
func ParseNodeLink(link string) (*ProxyNode, error) {
	link = strings.TrimSpace(link)
	for prefix, parser := range protocolParsers {
		if strings.HasPrefix(link, prefix) {
			return parser(link)
		}
	}

	if len(link) > 10 {
		return nil, fmt.Errorf("不支持的协议: %s", link[:10])
	}
	return nil, fmt.Errorf("不支持的协议")
}

// ==========================================
// 协议解析函数 (Protocol Parsers)
// ==========================================

func parseVMess(link string) (*ProxyNode, error) {
	encoded := strings.TrimPrefix(link, "vmess://")
	decoded, err := DecodeBase64(encoded)
	if err != nil {
		return nil, fmt.Errorf("VMess Base64 解码失败: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(decoded), &data); err != nil {
		return nil, fmt.Errorf("VMess JSON 解析失败: %v", err)
	}

	server := getString(data, "add", "")
	port := getInt(data, "port")
	uuid := getString(data, "id", "")

	if port <= 0 || port > 65535 || uuid == "" || server == "" {
		return nil, fmt.Errorf("VMess 配置信息不完整")
	}

	network := getString(data, "net", "tcp")
	node := &ProxyNode{
		Name:    getString(data, "ps", fmt.Sprintf("VMess-%s:%d", server, port)),
		Type:    "vmess",
		Server:  server,
		Port:    port,
		UUID:    uuid,
		Network: network,
		UDP:     true,
		Options: make(map[string]interface{}),
	}

	if getString(data, "tls", "") == "tls" {
		node.TLS = true
		node.Options["skip-cert-verify"] = getBool(data, "allowInsecure", false)
		if sni := getString(data, "sni", ""); sni != "" {
			node.Options["servername"] = sni
		}
	}

	node.Options["alterId"] = int(getFloat(data, "aid"))

	applyTransportMapping(node, network, getString(data, "path", "/"), getString(data, "host", server), getString(data, "type", ""))
	return node, nil
}

func parseVLESS(link string) (*ProxyNode, error) {
	return parseGenericNode(link, "vless", func(n *ProxyNode, q url.Values, p *url.URL) {
		n.UUID = p.User.Username()
		n.Network = firstNotEmpty(q.Get("type"), "tcp")
		n.UDP = true

		security := q.Get("security")
		if security == "tls" || security == "xtls" || security == "reality" {
			n.TLS = true
			applyTLSOptions(n, q, p.Hostname())
			if security == "reality" || q.Get("pbk") != "" {
				applyRealityOptions(n, q)
			}
			if flow := q.Get("flow"); flow != "" {
				n.Options["flow"] = flow
			}
			if enc := q.Get("encryption"); enc != "" {
				n.Options["encryption"] = enc
			}
		}
		applyTransportOptions(n, q)
	})
}

func parseTrojan(link string) (*ProxyNode, error) {
	return parseGenericNode(link, "trojan", func(n *ProxyNode, q url.Values, p *url.URL) {
		n.Password = p.User.Username()
		n.Network = firstNotEmpty(q.Get("type"), "tcp")
		n.UDP = true
		n.TLS = true
		applyTLSOptions(n, q, p.Hostname())
		applyTransportOptions(n, q)
	})
}

func parseShadowsocks(link string) (*ProxyNode, error) {
	// 处理 SS 特殊的 Base64 复合格式 (未包含 @ 时尝试解密前半部分)
	if !strings.Contains(link, "@") {
		encoded := strings.TrimPrefix(link, "ss://")
		if idx := strings.Index(encoded, "#"); idx != -1 {
			encoded = encoded[:idx]
		}
		if decoded, err := DecodeBase64(encoded); err == nil && !strings.HasPrefix(decoded, "{") {
			if parts := strings.Split(decoded, "@"); len(parts) == 2 {
				return parseSSParts(parts[0], parts[1], link)
			}
		}
	}

	// 标准 URI 解析
	parsed, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	method, password := extractSSAuth(parsed)
	if method == "" || password == "" {
		return nil, fmt.Errorf("SS 缺少认证信息")
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

	parseSSPlugin(node, parsed.Query())
	return node, nil
}

func parseSSR(link string) (*ProxyNode, error) {
	encoded := strings.TrimPrefix(link, "ssr://")
	decoded, err := DecodeBase64(encoded)
	if err != nil {
		return nil, err
	}

	parts := strings.SplitN(decoded, "/?", 2)
	mainParts := strings.Split(parts[0], ":")
	if len(mainParts) < 6 {
		return nil, fmt.Errorf("SSR 格式错误")
	}

	port, _ := strconv.Atoi(mainParts[1])
	password, _ := DecodeBase64(strings.Join(mainParts[5:], ":"))

	node := &ProxyNode{
		Name:     fmt.Sprintf("SSR-%s:%d", mainParts[0], port),
		Type:     "ssr",
		Server:   mainParts[0],
		Port:     port,
		Password: password,
		Cipher:   mainParts[3],
		Options: map[string]interface{}{
			"protocol": mainParts[2],
			"obfs":     mainParts[4],
		},
	}

	if len(parts) > 1 {
		paramsPart := strings.SplitN(parts[1], "#", 2)[0]
		if params, err := url.ParseQuery(paramsPart); err == nil {
			if d, err := DecodeBase64(params.Get("remarks")); err == nil && d != "" {
				node.Name = d
			}
			if d, err := DecodeBase64(params.Get("protoparam")); err == nil && d != "" {
				node.Options["protocol-param"] = d
			}
			if d, err := DecodeBase64(params.Get("obfsparam")); err == nil && d != "" {
				node.Options["obfs-param"] = d
			}
		}
	}
	return node, nil
}

func parseHysteria(link string) (*ProxyNode, error) {
	return parseGenericNode(link, "hysteria", func(n *ProxyNode, q url.Values, p *url.URL) {
		if auth := q.Get("auth"); auth != "" {
			n.Options["auth"] = auth
		}
		applyHysteriaBandwidth(n, q, "upmbps", "downmbps")
		n.Options["skip-cert-verify"] = isTrue(q.Get("insecure"))
	})
}

func parseHysteria2(link string) (*ProxyNode, error) {
	return parseGenericNode(link, "hysteria2", func(n *ProxyNode, q url.Values, p *url.URL) {
		if p.User != nil {
			n.Password = p.User.Username()
			if password, ok := p.User.Password(); ok && password != "" {
				n.Password = password
			}
		}
		n.TLS = true
		applyHysteriaBandwidth(n, q, "mbpsUp", "mbpsDown")
		applyTLSOptions(n, q, n.Server)
	})
}

func parseTUIC(link string) (*ProxyNode, error) {
	return parseGenericNode(link, "tuic", func(n *ProxyNode, q url.Values, p *url.URL) {
		user, _ := url.QueryUnescape(p.User.Username())
		if strings.Contains(user, ":") {
			parts := strings.SplitN(user, ":", 2)
			n.UUID, n.Password = parts[0], parts[1]
		} else {
			n.UUID = user
			n.Password, _ = p.User.Password()
		}
		n.UDP, n.TLS = true, true
		applyTLSOptions(n, q, n.Server)
		if cc := q.Get("congestion_control"); cc != "" {
			n.Options["congestion_control"] = cc
		}
		if m := q.Get("udp_relay_mode"); m != "" {
			n.Options["udp_relay_mode"] = m
		}
	})
}

func parseNaive(link string) (*ProxyNode, error) {
	normalized := link
	for _, old := range []string{"naive+https://", "naive://"} {
		if strings.HasPrefix(normalized, old) {
			normalized = "https://" + strings.TrimPrefix(normalized, old)
			break
		}
	}
	return parseGenericNode(normalized, "naive", func(n *ProxyNode, q url.Values, p *url.URL) {
		n.UUID = p.User.Username()
		n.Password, _ = p.User.Password()
		n.TLS = true
		applyTLSOptions(n, q, n.Server)
		if pad := q.Get("padding"); pad != "" {
			n.Options["padding"] = isTrue(pad)
		}
	})
}

func parseAnytls(link string) (*ProxyNode, error) {
	return parseGenericNode(link, "anytls", func(n *ProxyNode, q url.Values, p *url.URL) {
		if p.User != nil {
			n.Password = p.User.Username()
			if password, ok := p.User.Password(); ok && password != "" {
				n.Password = password
			}
		}
		n.UDP, n.TLS = true, true
		applyTLSOptions(n, q, n.Server)
	})
}

func parseSOCKS(link string) (*ProxyNode, error) {
	scheme := "socks5"
	if strings.HasPrefix(link, "socks://") {
		scheme = "socks"
	}

	linkWithoutScheme := strings.TrimPrefix(link, scheme+"://")
	baseLink, queryString, fragment := splitURLComponents(linkWithoutScheme)

	var server, username, password string
	var port int

	// 尝试 Base64 解码 (处理 GOST 格式)
	if decoded, err := DecodeBase64(baseLink); err == nil && strings.Contains(decoded, "@") {
		server, port, username, password = parseSOCKSBase64Auth(decoded)
	} else {
		// 标准 URL 解析
		parsed, err := url.Parse(scheme + "://" + linkWithoutScheme)
		if err != nil {
			return nil, fmt.Errorf("SOCKS 链接解析失败: %v", err)
		}
		server = parsed.Hostname()
		port = getPort(parsed)
		if parsed.User != nil {
			username = parsed.User.Username()
			password, _ = parsed.User.Password()
		}
	}

	if server == "" {
		return nil, fmt.Errorf("SOCKS 链接缺少服务器地址")
	}
	if port == 0 {
		port = 1080
	}

	query, _ := url.ParseQuery(queryString)
	nodeName := extractSOCKSNodeName(server, port, fragment, query)

	node := &ProxyNode{
		Name:     nodeName,
		Type:     scheme,
		Server:   server,
		Port:     port,
		UUID:     username,
		Password: password,
		UDP:      true,
		Options:  make(map[string]interface{}),
	}

	applySOCKSGOSTOptions(node, query.Get("gost"))
	return node, nil
}

func parseHTTP(link string) (*ProxyNode, error) {
	isTLS := strings.HasPrefix(link, "https://")
	return parseGenericNode(link, "http", func(n *ProxyNode, q url.Values, p *url.URL) {
		n.UUID = p.User.Username()
		n.Password, _ = p.User.Password()
		n.TLS = isTLS
		if isTLS {
			applyTLSOptions(n, q, n.Server)
		}
	})
}

func parseWireGuard(link string) (*ProxyNode, error) {
	parsed, err := url.Parse(link)
	if err != nil {
		return nil, fmt.Errorf("WireGuard 链接解析失败: %v", err)
	}

	query := parsed.Query()
	publicKey := query.Get("publicKey")
	privateKey := query.Get("privateKey")
	ipAddr := query.Get("ip")

	if publicKey == "" || privateKey == "" {
		return nil, fmt.Errorf("WireGuard 缺少必需参数")
	}

	node := &ProxyNode{
		Name:    getFragment(parsed, fmt.Sprintf("WG-%s:%s", parsed.Hostname(), parsed.Port())),
		Type:    "wireguard",
		Server:  parsed.Hostname(),
		Port:    getPort(parsed),
		UDP:     true, // WG 默认开启 UDP
		Options: make(map[string]interface{}),
	}

	node.Options["public-key"] = publicKey
	node.Options["private-key"] = privateKey

	if ipAddr != "" {
		ips := strings.Split(ipAddr, ",")
		node.Options["ip"] = strings.TrimSpace(ips[0])
		if len(ips) > 1 {
			node.Options["ipv6"] = strings.TrimSpace(ips[1])
		}
	}

	if mtu := query.Get("mtu"); mtu != "" {
		if mtuInt, err := strconv.Atoi(mtu); err == nil {
			node.Options["mtu"] = mtuInt
		}
	}

	if udp := query.Get("udp"); udp != "" {
		node.UDP = (udp == "1" || udp == "true")
	}
	node.Options["udp"] = node.UDP

	if psk := query.Get("presharedKey"); psk != "" {
		node.Options["preshared-key"] = psk
	}
	if reserved := query.Get("reserved"); reserved != "" {
		node.Options["reserved"] = reserved
	}
	if keepalive := query.Get("keepalive"); keepalive != "" {
		if k, err := strconv.Atoi(keepalive); err == nil {
			node.Options["keepalive"] = k
		}
	}

	return node, nil
}

// ==========================================
// 协议专属解析辅助函数 (Protocol Helpers)
// ==========================================

func extractSSAuth(parsed *url.URL) (method, password string) {
	if parsed.User != nil {
		authInfo := parsed.User.String()
		if parts := strings.SplitN(authInfo, ":", 2); len(parts) == 2 {
			return parts[0], parts[1]
		} else if decoded, err := DecodeBase64(authInfo); err == nil {
			if dParts := strings.SplitN(decoded, ":", 2); len(dParts) == 2 {
				return dParts[0], dParts[1]
			}
		}
	}
	return "", ""
}

func parseSSPlugin(node *ProxyNode, query url.Values) {
	pluginStr := query.Get("plugin")
	if pluginStr == "" {
		return
	}

	parts := strings.Split(pluginStr, ";")
	pluginName := strings.TrimSpace(parts[0])
	switch pluginName {
	case "simple-obfs", "obfs-local":
		pluginName = "obfs"
	case "v2ray-plugin":
		pluginName = "v2ray-plugin"
	}

	node.Options["plugin"] = pluginName
	pluginOpts := make(map[string]interface{})

	for _, part := range parts[1:] {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 {
			key, val := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
			switch key {
			case "obfs":
				pluginOpts["mode"] = val
			case "obfs-host":
				pluginOpts["host"] = val
			case "obfs-uri", "path":
				pluginOpts["path"] = val
			case "tls":
				pluginOpts["tls"] = true
			default:
				pluginOpts[key] = val
			}
		}
	}

	if len(pluginOpts) > 0 {
		node.Options["plugin-opts"] = pluginOpts
	}
}

func parseSSParts(authPart, serverPart, originalLink string) (*ProxyNode, error) {
	auth := strings.SplitN(authPart, ":", 2)
	server := strings.SplitN(serverPart, ":", 2)
	if len(auth) != 2 || len(server) != 2 {
		return nil, fmt.Errorf("SS 格式解析失败")
	}

	hostPort := server[1]
	if idx := strings.Index(hostPort, "?"); idx != -1 {
		hostPort = hostPort[:idx]
	}
	if idx := strings.Index(hostPort, "#"); idx != -1 {
		hostPort = hostPort[:idx]
	}

	port, _ := strconv.Atoi(hostPort)
	parsed, _ := url.Parse(originalLink)

	node := &ProxyNode{
		Name:     getFragment(parsed, fmt.Sprintf("SS-%s:%d", server[0], port)),
		Type:     "ss",
		Server:   server[0],
		Port:     port,
		Cipher:   auth[0],
		Password: auth[1],
		Options:  make(map[string]interface{}),
	}

	if parsed != nil {
		parseSSPlugin(node, parsed.Query())
	}
	return node, nil
}

func splitURLComponents(raw string) (base, query, fragment string) {
	base = raw
	if idx := strings.Index(base, "?"); idx != -1 {
		query = base[idx+1:]
		base = base[:idx]
		if idx2 := strings.Index(query, "#"); idx2 != -1 {
			fragment = query[idx2+1:]
			query = query[:idx2]
		}
	} else if idx := strings.Index(base, "#"); idx != -1 {
		fragment = base[idx+1:]
		base = base[:idx]
	}
	return
}

func parseSOCKSBase64Auth(decoded string) (server string, port int, username, password string) {
	decodedURL, _ := url.QueryUnescape(decoded)
	parts := strings.SplitN(decodedURL, "@", 2)
	if len(parts) == 2 {
		authInfo := parts[0]
		if lastColonIdx := strings.LastIndex(authInfo, ":"); lastColonIdx != -1 {
			username, password = authInfo[:lastColonIdx], authInfo[lastColonIdx+1:]
		} else {
			username = authInfo
		}

		serverParts := strings.SplitN(parts[1], ":", 2)
		server = serverParts[0]
		if len(serverParts) == 2 {
			port, _ = strconv.Atoi(serverParts[1])
		}
	}
	return
}

func extractSOCKSNodeName(server string, port int, fragment string, query url.Values) string {
	if remarks := query.Get("remarks"); remarks != "" {
		if dec, err := url.QueryUnescape(remarks); err == nil {
			return dec
		}
		return remarks
	}
	if fragment != "" {
		if dec, err := url.QueryUnescape(fragment); err == nil {
			return dec
		}
		return fragment
	}
	return fmt.Sprintf("SOCKS-%s:%d", server, port)
}

func applySOCKSGOSTOptions(node *ProxyNode, gostParam string) {
	if gostParam == "" {
		return
	}
	gostJSON, err := DecodeBase64(gostParam)
	if err != nil {
		return
	}

	var gostConfig map[string]interface{}
	if err := json.Unmarshal([]byte(gostJSON), &gostConfig); err == nil {
		if route, ok := gostConfig["route"].(string); ok && route == "ws" {
			node.Network = "ws"
			wsOpts := make(map[string]interface{})
			if path, ok := gostConfig["path"].(string); ok && path != "" {
				wsOpts["path"] = path
			}
			if host, ok := gostConfig["host"].(string); ok && host != "" {
				wsOpts["headers"] = map[string]string{"Host": host}
			}
			if len(wsOpts) > 0 {
				node.Options["ws-opts"] = wsOpts
			}
		}
	}
}

// ==========================================
// 节点公共修饰函数 (Modifiers)
// ==========================================

func parseGenericNode(link, nodeType string, modifier func(*ProxyNode, url.Values, *url.URL)) (*ProxyNode, error) {
	parsed, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	node := &ProxyNode{
		Name:    getFragment(parsed, fmt.Sprintf("%s-%s:%s", strings.ToUpper(nodeType), parsed.Hostname(), parsed.Port())),
		Type:    nodeType,
		Server:  parsed.Hostname(),
		Port:    getPort(parsed),
		Options: make(map[string]interface{}),
	}

	modifier(node, parsed.Query(), parsed)
	return node, nil
}

func applyTLSOptions(node *ProxyNode, query url.Values, defaultSNI string) {
	node.Options["skip-cert-verify"] = isTrue(query.Get("allowInsecure")) || isTrue(query.Get("insecure")) || isTrue(query.Get("allow_insecure"))
	node.Options["servername"] = firstNotEmpty(query.Get("sni"), query.Get("peer"), defaultSNI)

	if fp := query.Get("fp"); fp != "" {
		node.Options["client-fingerprint"] = fp
	}
	if alpn := query.Get("alpn"); alpn != "" {
		node.Options["alpn"] = strings.Split(alpn, ",")
	}
}

func applyTransportOptions(node *ProxyNode, query url.Values) {
	path, host := query.Get("path"), query.Get("host")

	switch node.Network {
	case "ws":
		wsOpts := make(map[string]interface{})
		if path != "" {
			wsOpts["path"] = path
		}
		if host != "" {
			wsOpts["headers"] = map[string]string{"Host": host}
		}
		if len(wsOpts) > 0 {
			node.Options["ws-opts"] = wsOpts
		}
	case "grpc":
		if serviceName := firstNotEmpty(query.Get("serviceName"), path); serviceName != "" {
			node.Options["grpc-opts"] = map[string]interface{}{"grpc-service-name": serviceName}
		}
	case "tcp":
		if hType := query.Get("headerType"); hType != "" {
			node.Options["header-type"] = hType
		}
	}
}

func applyTransportMapping(node *ProxyNode, network, path, host, obfsType string) {
	switch network {
	case "tcp":
		if obfsType == "http" {
			node.Network = "http"
			httpOpts := map[string]interface{}{"method": "GET", "path": []string{path}}
			if host != "" {
				httpOpts["headers"] = map[string]interface{}{"Host": []string{host}}
			}
			node.Options["http-opts"] = httpOpts
		}
	case "ws":
		node.Options["ws-opts"] = map[string]interface{}{
			"path":    path,
			"headers": map[string]string{"Host": host},
		}
	case "grpc":
		node.Options["grpc-opts"] = map[string]interface{}{"grpc-service-name": path}
	case "h2":
		node.Options["h2-opts"] = map[string]interface{}{"path": path, "host": []string{host}}
	case "httpupgrade":
		node.Network = "ws"
		node.Options["ws-opts"] = map[string]interface{}{
			"path":               path,
			"headers":            map[string]string{"Host": host},
			"v2ray-http-upgrade": true,
		}
	}
}

func applyRealityOptions(node *ProxyNode, query url.Values) {
	reality := make(map[string]interface{})
	if pbk := query.Get("pbk"); pbk != "" {
		reality["public-key"] = pbk
	}
	if sid := query.Get("sid"); sid != "" {
		reality["short-id"] = sid
	}
	if pqv := query.Get("pqv"); pqv != "" {
		reality["pqv"] = pqv
	}
	if len(reality) > 0 {
		node.Options["reality-opts"] = reality
	}
}

func applyHysteriaBandwidth(node *ProxyNode, query url.Values, upKey, downKey string) {
	if up := query.Get(upKey); up != "" {
		node.Options["up"] = strings.TrimSuffix(up, " mbps") + " mbps"
	}
	if down := query.Get(downKey); down != "" {
		node.Options["down"] = strings.TrimSuffix(down, " mbps") + " mbps"
	}
}

// ==========================================
// 基础工具函数 (Base Utilities)
// ==========================================

func isTrue(s string) bool {
	return s == "1" || s == "true"
}

// DecodeBase64 经过优化的 Base64 解码器，自动处理 URL 编码变体和缺失的 Padding
func DecodeBase64(s string) (string, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\n", ""), "\r", "")
	if s == "" {
		return "", nil
	}

	// 支持的解码器
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.RawURLEncoding,
	}

	// 1. 尝试直接解码
	for _, enc := range encodings {
		if b, err := enc.DecodeString(s); err == nil {
			return string(b), nil
		}
	}

	// 2. 尝试修复字符并补充 Padding 后解码
	clean := strings.ReplaceAll(strings.ReplaceAll(s, "-", "+"), "_", "/")
	if m := len(clean) % 4; m != 0 {
		clean += strings.Repeat("=", 4-m)
	}

	if b, err := base64.StdEncoding.DecodeString(clean); err == nil {
		return string(b), nil
	}

	return "", fmt.Errorf("Base64 解码失败")
}

func TryDecodeNodeList(content string) string {
	if containsNodeLinks(content) {
		return content
	}
	if decoded, err := DecodeBase64(content); err == nil && containsNodeLinks(decoded) {
		return decoded
	}

	lines := strings.Split(content, "\n")
	var result []string
	changed := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if decoded, err := DecodeBase64(line); err == nil && (containsNodeLinks(decoded) || strings.Contains(decoded, "://")) {
			result = append(result, decoded)
			changed = true
		} else {
			result = append(result, line)
		}
	}

	if changed {
		return strings.Join(result, "\n")
	}
	return content
}

func containsNodeLinks(s string) bool {
	protocols := []string{"vmess://", "vless://", "trojan://", "ss://", "ssr://"}
	for _, p := range protocols {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

func getString(m map[string]interface{}, key, def string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return def
}

func getInt(m map[string]interface{}, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case string:
		i, _ := strconv.Atoi(v)
		return i
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	}
	return 0
}

func getBool(m map[string]interface{}, key string, def bool) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	if v, ok := m[key].(string); ok {
		return isTrue(v)
	}
	return def
}

func getFragment(parsed *url.URL, def string) string {
	if parsed.Fragment != "" {
		if d, err := url.QueryUnescape(parsed.Fragment); err == nil {
			return d
		}
		return parsed.Fragment
	}
	return def
}

func getPort(parsed *url.URL) int {
	if p := parsed.Port(); p != "" {
		i, _ := strconv.Atoi(p)
		return i
	}
	if parsed.Scheme == "ss" || parsed.Scheme == "ssr" {
		return 8388
	}
	return 443
}

func firstNotEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

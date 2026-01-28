package config_update

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

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

func parseVMess(link string) (*ProxyNode, error) {
	encoded := strings.TrimPrefix(link, "vmess://")
	decoded, err := DecodeBase64(encoded)
	if err != nil {
		return nil, fmt.Errorf("Base64 解码失败: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(decoded), &data); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %v", err)
	}

	server := getString(data, "add", "")
	port := getInt(data, "port")
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("无效的端口: %v", data["port"])
	}

	uuid := getString(data, "id", "")
	if uuid == "" {
		return nil, fmt.Errorf("缺少 UUID (id)")
	}
	if server == "" {
		return nil, fmt.Errorf("缺少服务器地址 (add)")
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

	if tls := getString(data, "tls", ""); tls == "tls" {
		node.TLS = true
		node.Options["skip-cert-verify"] = getBool(data, "allowInsecure", false)
		if sni := getString(data, "sni", ""); sni != "" {
			node.Options["servername"] = sni
		}
	}

	alterID := getFloat(data, "aid")
	node.Options["alterId"] = int(alterID)

	switch network {
	case "ws":
		node.Options["ws-opts"] = map[string]interface{}{
			"path": getString(data, "path", "/"),
			"headers": map[string]string{
				"Host": getString(data, "host", server),
			},
		}
	case "grpc":
		node.Options["grpc-opts"] = map[string]interface{}{
			"grpc-service-name": getString(data, "path", ""),
		}
	case "h2":
		node.Options["h2-opts"] = map[string]interface{}{
			"path": getString(data, "path", "/"),
			"host": []string{getString(data, "host", server)},
		}
	case "httpupgrade":
		node.Network = "ws"
		node.Options["ws-opts"] = map[string]interface{}{
			"path": getString(data, "path", "/"),
			"headers": map[string]string{
				"Host": getString(data, "host", server),
			},
			"v2ray-http-upgrade": true,
		}
	}

	return node, nil
}

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

	if security == "tls" || security == "xtls" || security == "reality" {
		node.TLS = true
		node.Options["skip-cert-verify"] = query.Get("allowInsecure") == "1" || query.Get("allowInsecure") == "true"
		node.Options["servername"] = firstNotEmpty(query.Get("sni"), parsed.Hostname())

		if fp := query.Get("fp"); fp != "" {
			node.Options["client-fingerprint"] = fp
		}

		if security == "reality" || query.Get("pbk") != "" {
			realityOpts := make(map[string]interface{})
			if pbk := query.Get("pbk"); pbk != "" {
				realityOpts["public-key"] = pbk
			}
			if sid := query.Get("sid"); sid != "" {
				realityOpts["short-id"] = sid
			}
			if pqv := query.Get("pqv"); pqv != "" {
				realityOpts["pqv"] = pqv
			}
			if len(realityOpts) > 0 {
				node.Options["reality-opts"] = realityOpts
			}
		}

		if flow := query.Get("flow"); flow != "" {
			node.Options["flow"] = flow
		}
		if encryption := query.Get("encryption"); encryption != "" {
			node.Options["encryption"] = encryption
		}
	}

	switch network {
	case "ws":
		wsOpts := make(map[string]interface{})
		if path := query.Get("path"); path != "" {
			wsOpts["path"] = path
		}
		if host := query.Get("host"); host != "" {
			wsOpts["headers"] = map[string]string{"Host": host}
		}
		if len(wsOpts) > 0 {
			node.Options["ws-opts"] = wsOpts
		}
	case "grpc":
		grpcOpts := make(map[string]interface{})
		serviceName := firstNotEmpty(query.Get("serviceName"), query.Get("path"))
		if serviceName != "" {
			grpcOpts["grpc-service-name"] = serviceName
			node.Options["grpc-opts"] = grpcOpts
		}
	case "tcp":
		if headerType := query.Get("headerType"); headerType != "" {
			node.Options["header-type"] = headerType
		}
	}

	return node, nil
}

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
	network := firstNotEmpty(query.Get("type"), "tcp")

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

	node.Options["skip-cert-verify"] = query.Get("allowInsecure") == "1" || query.Get("allowInsecure") == "true"
	node.Options["servername"] = firstNotEmpty(query.Get("sni"), query.Get("peer"), parsed.Hostname())

	if fp := query.Get("fp"); fp != "" {
		node.Options["client-fingerprint"] = fp
	}
	if alpn := query.Get("alpn"); alpn != "" {
		node.Options["alpn"] = strings.Split(alpn, ",")
	}

	if network == "ws" {
		wsOpts := make(map[string]interface{})
		if path := query.Get("path"); path != "" {
			wsOpts["path"] = path
		}
		if host := query.Get("host"); host != "" {
			wsOpts["headers"] = map[string]string{"Host": host}
		}
		if len(wsOpts) > 0 {
			node.Options["ws-opts"] = wsOpts
		}
	} else if network == "grpc" {
		if serviceName := query.Get("serviceName"); serviceName != "" {
			node.Options["grpc-opts"] = map[string]interface{}{"grpc-service-name": serviceName}
		}
	}

	return node, nil
}

func parseShadowsocks(link string) (*ProxyNode, error) {
	if !strings.Contains(link, "@") {
		encoded := strings.TrimPrefix(link, "ss://")
		if idx := strings.Index(encoded, "#"); idx != -1 {
			encoded = encoded[:idx]
		}

		decoded, err := DecodeBase64(encoded)
		if err == nil && !strings.HasPrefix(decoded, "{") { // 排除 VMess 误判
			if parts := strings.Split(decoded, "@"); len(parts) == 2 {
				return parseSSParts(parts[0], parts[1], link, decoded)
			}
		}
	}

	parsed, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	var method, password string
	if parsed.User != nil {
		authInfo := parsed.User.String()
		if parts := strings.SplitN(authInfo, ":", 2); len(parts) == 2 {
			method, password = parts[0], parts[1]
		} else {
			if decoded, err := DecodeBase64(authInfo); err == nil {
				if parts := strings.SplitN(decoded, ":", 2); len(parts) == 2 {
					method, password = parts[0], parts[1]
				}
			}
		}
	}

	if method == "" || password == "" {
		return nil, fmt.Errorf("缺少认证信息")
	}

	return &ProxyNode{
		Name:     getFragment(parsed, fmt.Sprintf("SS-%s:%s", parsed.Hostname(), parsed.Port())),
		Type:     "ss",
		Server:   parsed.Hostname(),
		Port:     getPort(parsed),
		Cipher:   method,
		Password: password,
		Options:  make(map[string]interface{}),
	}, nil
}

func parseSSParts(authPart, serverPart, originalLink, _ string) (*ProxyNode, error) {
	authParts := strings.SplitN(authPart, ":", 2)
	if len(authParts) != 2 {
		return nil, fmt.Errorf("SS认证格式错误")
	}
	method, password := authParts[0], authParts[1]

	serverParts := strings.SplitN(serverPart, ":", 2)
	if len(serverParts) != 2 {
		return nil, fmt.Errorf("SS服务器格式错误")
	}
	server := serverParts[0]
	port, err := strconv.Atoi(serverParts[1])
	if err != nil {
		port = 8388
	}

	parsed, _ := url.Parse(originalLink)
	name := getFragment(parsed, fmt.Sprintf("SS-%s:%d", server, port))

	return &ProxyNode{
		Name:     name,
		Type:     "ss",
		Server:   server,
		Port:     port,
		Cipher:   method,
		Password: password,
		Options:  make(map[string]interface{}),
	}, nil
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
		paramsPart := parts[1]
		if idx := strings.Index(paramsPart, "#"); idx != -1 {
			paramsPart = paramsPart[:idx]
		}

		if params, err := url.ParseQuery(paramsPart); err == nil {
			if remarks := params.Get("remarks"); remarks != "" {
				if d, err := DecodeBase64(remarks); err == nil {
					node.Name = d
				}
			}
			if p := params.Get("protoparam"); p != "" {
				if d, err := DecodeBase64(p); err == nil {
					node.Options["protocol-param"] = d
				}
			}
			if o := params.Get("obfsparam"); o != "" {
				if d, err := DecodeBase64(o); err == nil {
					node.Options["obfs-param"] = d
				}
			}
		}
	}
	return node, nil
}

func parseHysteria(link string) (*ProxyNode, error) {
	return parseGenericNode(link, "hysteria", func(n *ProxyNode, q url.Values) {
		if auth := q.Get("auth"); auth != "" {
			n.Options["auth"] = auth
		}
		if up := q.Get("upmbps"); up != "" {
			n.Options["up"] = up + " mbps"
		}
		if down := q.Get("downmbps"); down != "" {
			n.Options["down"] = down + " mbps"
		}
		n.Options["skip-cert-verify"] = q.Get("insecure") == "1"
	})
}

func parseHysteria2(link string) (*ProxyNode, error) {
	return parseGenericNode(link, "hysteria2", func(n *ProxyNode, q url.Values) {
		if p, _ := url.Parse(link); p.User != nil {
			n.Password, _ = p.User.Password()
		}
		n.TLS = true
		if up := q.Get("mbpsUp"); up != "" {
			n.Options["up"] = strings.TrimSuffix(up, " mbps") + " mbps"
		}
		if down := q.Get("mbpsDown"); down != "" {
			n.Options["down"] = strings.TrimSuffix(down, " mbps") + " mbps"
		}
		n.Options["skip-cert-verify"] = q.Get("insecure") == "1" || q.Get("insecure") == "true"
		n.Options["servername"] = firstNotEmpty(q.Get("sni"), q.Get("peer"), n.Server)
		if alpn := q.Get("alpn"); alpn != "" {
			n.Options["alpn"] = strings.Split(alpn, ",")
		}
	})
}

func parseTUIC(link string) (*ProxyNode, error) {
	return parseGenericNode(link, "tuic", func(n *ProxyNode, q url.Values) {
		if p, _ := url.Parse(link); p.User != nil {
			n.UUID = p.User.Username()
			n.Password, _ = p.User.Password()
		}
		n.UDP = true
		n.TLS = true
		n.Options["servername"] = firstNotEmpty(q.Get("sni"), n.Server)
		if alpn := q.Get("alpn"); alpn != "" {
			n.Options["alpn"] = []string{alpn}
		}
		if cc := q.Get("congestion_control"); cc != "" {
			n.Options["congestion_control"] = cc
		}
		if m := q.Get("udp_relay_mode"); m != "" {
			n.Options["udp_relay_mode"] = m
		}
		n.Options["skip-cert-verify"] = q.Get("allow_insecure") == "1" || q.Get("allow_insecure") == "true"
	})
}

func parseNaive(link string) (*ProxyNode, error) {
	link = strings.Replace(link, "naive+https://", "https://", 1)
	link = strings.Replace(link, "naive://", "https://", 1)
	return parseGenericNode(link, "naive", func(n *ProxyNode, q url.Values) {
		if p, _ := url.Parse(link); p.User != nil {
			n.UUID = p.User.Username()
			n.Password, _ = p.User.Password()
		}
		n.TLS = true
		n.Options["servername"] = firstNotEmpty(q.Get("sni"), n.Server)
		if pad := q.Get("padding"); pad != "" {
			n.Options["padding"] = pad == "true" || pad == "1"
		}
		n.Options["skip-cert-verify"] = q.Get("insecure") == "1" || q.Get("insecure") == "true"
	})
}

func parseAnytls(link string) (*ProxyNode, error) {
	return parseGenericNode(link, "anytls", func(n *ProxyNode, q url.Values) {
		if p, _ := url.Parse(link); p.User != nil {
			n.UUID = p.User.Username()
		}
		n.UDP = true
		n.TLS = true
		n.Options["servername"] = firstNotEmpty(q.Get("peer"), q.Get("sni"), n.Server)
		n.Options["skip-cert-verify"] = q.Get("insecure") == "1" || q.Get("insecure") == "true" || q.Get("insecure") == "0"
	})
}

func parseGenericNode(link, nodeType string, modifier func(*ProxyNode, url.Values)) (*ProxyNode, error) {
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

	modifier(node, parsed.Query())
	return node, nil
}

func DecodeBase64(s string) (string, error) {
	if s == "" {
		return "", nil
	}

	s = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\t' || r == '\r' {
			return -1
		}
		return r
	}, s)

	if len(s) == 0 {
		return "", nil
	}

	decoded, err := tryDecodeBase64(s, base64.StdEncoding)
	if err == nil {
		return decoded, nil
	}

	decoded, err = tryDecodeBase64(s, base64.URLEncoding)
	if err == nil {
		return decoded, nil
	}

	decoded, err = tryDecodeBase64(s, base64.RawStdEncoding)
	if err == nil {
		return decoded, nil
	}

	decoded, err = tryDecodeBase64(s, base64.RawURLEncoding)
	if err == nil {
		return decoded, nil
	}

	return "", fmt.Errorf("Base64解码失败: 所有编码方式都失败")
}

func tryDecodeBase64(s string, encoding *base64.Encoding) (string, error) {
	clean := s
	if encoding == base64.StdEncoding || encoding == base64.RawStdEncoding {
		clean = strings.ReplaceAll(clean, "-", "+")
		clean = strings.ReplaceAll(clean, "_", "/")
	}

	if encoding == base64.StdEncoding || encoding == base64.URLEncoding {
		if m := len(clean) % 4; m != 0 {
			clean += strings.Repeat("=", 4-m)
		}
	}

	b, err := encoding.DecodeString(clean)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func TryDecodeNodeList(content string) string {
	if containsNodeLinks(content) {
		return content
	}

	if decoded, err := DecodeBase64(content); err == nil {
		if containsNodeLinks(decoded) {
			return decoded
		}
	}

	lines := strings.Split(content, "\n")
	var decodedLines []string
	hasDecoded := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if decoded, err := DecodeBase64(line); err == nil && decoded != "" {
			if containsNodeLinks(decoded) {
				decodedLines = append(decodedLines, decoded)
				hasDecoded = true
				continue
			}
			if strings.Contains(decoded, "://") {
				decodedLines = append(decodedLines, decoded)
				hasDecoded = true
				continue
			}
		}
		decodedLines = append(decodedLines, line)
	}

	if hasDecoded {
		return strings.Join(decodedLines, "\n")
	}

	return content
}

func containsNodeLinks(s string) bool {
	return strings.Contains(s, "vmess://") ||
		strings.Contains(s, "vless://") ||
		strings.Contains(s, "trojan://") ||
		strings.Contains(s, "ss://") ||
		strings.Contains(s, "ssr://")
}

func getString(m map[string]interface{}, key, defaultValue string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultValue
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return int(f)
		}
		if s, ok := v.(string); ok {
			if i, err := strconv.Atoi(s); err == nil {
				return i
			}
		}
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
		if s, ok := v.(string); ok {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return f
			}
		}
	}
	return 0
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
		if decoded, err := url.QueryUnescape(parsed.Fragment); err == nil {
			return decoded
		}
		return parsed.Fragment
	}
	return defaultValue
}

func getPort(parsed *url.URL) int {
	if p := parsed.Port(); p != "" {
		if i, err := strconv.Atoi(p); err == nil {
			return i
		}
	}
	switch parsed.Scheme {
	case "ss", "ssr":
		return 8388
	default:
		return 443
	}
}

func firstNotEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

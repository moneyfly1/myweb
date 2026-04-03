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
	Name     string         `yaml:"name" json:"Name"`
	Type     string         `yaml:"type" json:"Type"`
	Server   string         `yaml:"server" json:"Server"`
	Port     int            `yaml:"port" json:"Port"`
	UUID     string         `yaml:"uuid,omitempty" json:"UUID,omitempty"`
	Password string         `yaml:"password,omitempty" json:"Password,omitempty"`
	Cipher   string         `yaml:"cipher,omitempty" json:"Cipher,omitempty"`
	Network  string         `yaml:"network,omitempty" json:"Network,omitempty"`
	TLS      bool           `yaml:"tls,omitempty" json:"TLS,omitempty"`
	UDP      bool           `yaml:"udp,omitempty" json:"UDP,omitempty"`
	Options  map[string]any `yaml:",inline" json:"Options,omitempty"`
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

func ParseNodeLink(link string) (*ProxyNode, error) {
	link = strings.TrimSpace(link)
	for prefix, parser := range protocolParsers {
		if strings.HasPrefix(link, prefix) {
			return parser(link)
		}
	}
	if len(link) > 10 {
		return nil, fmt.Errorf("不支持的协议: %s...", link[:10])
	}
	return nil, fmt.Errorf("不支持的协议")
}

func parseVMess(link string) (*ProxyNode, error) {
	decoded, err := DecodeBase64(strings.TrimPrefix(link, "vmess://"))
	if err != nil {
		return nil, fmt.Errorf("VMess 解码失败: %v", err)
	}

	// 检查是否是非标准格式：auto:UUID@Server:Port
	if strings.Contains(decoded, "@") && !strings.HasPrefix(decoded, "{") {
		parts := strings.Split(decoded, "@")
		if len(parts) == 2 {
			uuid := parts[0]
			if strings.Contains(uuid, ":") {
				uuid = strings.Split(uuid, ":")[1]
			}

			serverPort := parts[1]
			var server string
			var port int
			if strings.Contains(serverPort, ":") {
				sp := strings.Split(serverPort, ":")
				server = sp[0]
				port, _ = strconv.Atoi(sp[1])
			}

			if server != "" && port > 0 && uuid != "" {
				return &ProxyNode{
					Name:    fmt.Sprintf("VMess-%s:%d", server, port),
					Type:    "vmess",
					Server:  server,
					Port:    port,
					UUID:    uuid,
					Network: "tcp",
					UDP:     true,
					Options: map[string]any{"alterId": 0},
				}, nil
			}
		}
	}

	// 标准 JSON 格式
	var data map[string]any
	if err := json.Unmarshal([]byte(decoded), &data); err != nil {
		return nil, fmt.Errorf("VMess 解析失败: %v", err)
	}

	server, port, uuid := getString(data, "add", ""), getInt(data, "port"), getString(data, "id", "")
	if port <= 0 || port > 65535 || uuid == "" || server == "" {
		return nil, fmt.Errorf("VMess 配置不完整")
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
		Options: map[string]any{"alterId": int(getFloat(data, "aid"))},
	}

	if getString(data, "tls", "") == "tls" {
		node.TLS = true
		node.Options["skip-cert-verify"] = getBool(data, "allowInsecure", false)
		if sni := getString(data, "sni", ""); sni != "" {
			node.Options["servername"] = sni
		}
	}

	applyTransportMapping(node, network, getString(data, "path", "/"), getString(data, "host", server), getString(data, "type", ""))
	return node, nil
}

func parseVLESS(link string) (*ProxyNode, error) {
	// 预处理：检查是否是 Base64 编码的格式
	// 格式：vless://Base64(auto:UUID@Server:Port)?params
	if strings.HasPrefix(link, "vless://") {
		parts := strings.SplitN(link[8:], "?", 2)
		hostPart := parts[0]
		if idx := strings.Index(hostPart, "#"); idx != -1 {
			hostPart = hostPart[:idx]
		}

		// 尝试解码，如果成功且包含 @，则重构 URL
		if decoded, err := DecodeBase64(hostPart); err == nil && strings.Contains(decoded, "@") {
			uuidAndServer := strings.Split(decoded, "@")
			if len(uuidAndServer) == 2 {
				uuid := uuidAndServer[0]
				if strings.Contains(uuid, ":") {
					uuid = strings.Split(uuid, ":")[1]
				}
				serverPort := uuidAndServer[1]

				// 重构为标准格式
				link = "vless://" + uuid + "@" + serverPort
				if len(parts) > 1 {
					link += "?" + parts[1]
				}
			}
		}
	}

	return parseGenericNode(link, "vless", func(n *ProxyNode, q url.Values, p *url.URL) {
		n.UUID = p.User.Username()
		n.Network = firstNotEmpty(q.Get("type"), "tcp")
		n.UDP = true

		sec := q.Get("security")
		if sec == "tls" || sec == "xtls" || sec == "reality" {
			n.TLS = true
			applyTLSOptions(n, q, p.Hostname())
			if sec == "reality" || q.Get("pbk") != "" {
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
		n.Network, n.UDP, n.TLS = firstNotEmpty(q.Get("type"), "tcp"), true, true
		applyTLSOptions(n, q, p.Hostname())
		applyTransportOptions(n, q)
	})
}

func parseShadowsocks(link string) (*ProxyNode, error) {
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

	return parseGenericNode(link, "ss", func(n *ProxyNode, q url.Values, p *url.URL) {
		n.Cipher, n.Password = extractSSAuth(p)
		parseSSPlugin(n, q)
	})
}

func parseSSR(link string) (*ProxyNode, error) {
	decoded, err := DecodeBase64(strings.TrimPrefix(link, "ssr://"))
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
		Options:  map[string]any{"protocol": mainParts[2], "obfs": mainParts[4]},
	}

	if len(parts) > 1 {
		if params, err := url.ParseQuery(strings.SplitN(parts[1], "#", 2)[0]); err == nil {
			if d, _ := DecodeBase64(params.Get("remarks")); d != "" {
				node.Name = d
			}
			if d, _ := DecodeBase64(params.Get("protoparam")); d != "" {
				node.Options["protocol-param"] = d
			}
			if d, _ := DecodeBase64(params.Get("obfsparam")); d != "" {
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
		extractAuthToNode(n, p, true)
		n.TLS = true
		applyHysteriaBandwidth(n, q, "mbpsUp", "mbpsDown")
		applyTLSOptions(n, q, n.Server)
	})
}

func parseTUIC(link string) (*ProxyNode, error) {
	return parseGenericNode(link, "tuic", func(n *ProxyNode, q url.Values, p *url.URL) {
		user, _ := url.QueryUnescape(p.User.Username())
		if parts := strings.SplitN(user, ":", 2); len(parts) == 2 {
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
	link = "https://" + strings.TrimPrefix(strings.TrimPrefix(link, "naive+https://"), "naive://")
	return parseGenericNode(link, "naive", func(n *ProxyNode, q url.Values, p *url.URL) {
		extractAuthToNode(n, p, false)
		n.TLS = true
		applyTLSOptions(n, q, n.Server)
		if pad := q.Get("padding"); pad != "" {
			n.Options["padding"] = isTrue(pad)
		}
	})
}

func parseAnytls(link string) (*ProxyNode, error) {
	return parseGenericNode(link, "anytls", func(n *ProxyNode, q url.Values, p *url.URL) {
		extractAuthToNode(n, p, true)
		n.UDP, n.TLS = true, true
		applyTLSOptions(n, q, n.Server)
	})
}

func parseSOCKS(link string) (*ProxyNode, error) {
	scheme := "socks5"
	if strings.HasPrefix(link, "socks://") {
		scheme = "socks"
	}

	baseLink, queryString, fragment := splitURLComponents(strings.TrimPrefix(link, scheme+"://"))

	// 处理 GOST 格式的 Base64 编码
	if decoded, err := DecodeBase64(baseLink); err == nil && strings.Contains(decoded, "@") {
		server, port, username, password := parseSOCKSBase64Auth(decoded)
		query, _ := url.ParseQuery(queryString)
		node := &ProxyNode{
			Name: extractSOCKSNodeName(server, port, fragment, query),
			Type: scheme, Server: server, Port: port, UUID: username, Password: password,
			UDP: true, Options: make(map[string]any),
		}
		applySOCKSGOSTOptions(node, query.Get("gost"))
		return node, nil
	}

	// 走标准 URL 解析逻辑
	return parseGenericNode(link, scheme, func(n *ProxyNode, q url.Values, p *url.URL) {
		extractAuthToNode(n, p, false)
		n.UDP = true
		n.Name = extractSOCKSNodeName(n.Server, n.Port, getFragment(p, ""), q)
	})
}

func parseHTTP(link string) (*ProxyNode, error) {
	return parseGenericNode(link, "http", func(n *ProxyNode, q url.Values, p *url.URL) {
		extractAuthToNode(n, p, false)
		if n.TLS = strings.HasPrefix(link, "https://"); n.TLS {
			applyTLSOptions(n, q, n.Server)
		}
	})
}

func parseWireGuard(link string) (*ProxyNode, error) {
	return parseGenericNode(link, "wireguard", func(n *ProxyNode, q url.Values, p *url.URL) {
		n.UDP = q.Get("udp") == "1" || q.Get("udp") == "true" || q.Get("udp") == ""

		// 手动解析原始查询字符串以保留 Base64 中的 + 字符
		publicKey := extractRawQueryParam(p.RawQuery, "publicKey")
		privateKey := extractRawQueryParam(p.RawQuery, "privateKey")

		// 清理 Base64 字符串中的空格（可能由 URL 解码错误导致）
		publicKey = strings.ReplaceAll(publicKey, " ", "")
		privateKey = strings.ReplaceAll(privateKey, " ", "")

		n.Options["public-key"] = publicKey
		n.Options["private-key"] = privateKey
		n.Options["udp"] = n.UDP

		if ipAddr := q.Get("ip"); ipAddr != "" {
			ips := strings.Split(ipAddr, ",")
			n.Options["ip"] = strings.TrimSpace(ips[0])
			if len(ips) > 1 {
				n.Options["ipv6"] = strings.TrimSpace(ips[1])
			}
		}

		if mtu, err := strconv.Atoi(q.Get("mtu")); err == nil {
			n.Options["mtu"] = mtu
		}
		if psk := q.Get("presharedKey"); psk != "" {
			n.Options["preshared-key"] = psk
		}
		if reserved := q.Get("reserved"); reserved != "" {
			n.Options["reserved"] = reserved
		}
		if k, err := strconv.Atoi(q.Get("keepalive")); err == nil {
			n.Options["keepalive"] = k
		}
	})
}

// extractRawQueryParam 从原始查询字符串中提取参数值，保留 + 字符
func extractRawQueryParam(rawQuery, key string) string {
	params := strings.Split(rawQuery, "&")
	prefix := key + "="
	for _, param := range params {
		if strings.HasPrefix(param, prefix) {
			return strings.TrimPrefix(param, prefix)
		}
	}
	return ""
}

func extractAuthToNode(n *ProxyNode, p *url.URL, preferPwd bool) {
	if p.User == nil {
		return
	}
	if preferPwd {
		n.Password = p.User.Username()
		if pwd, ok := p.User.Password(); ok && pwd != "" {
			n.Password = pwd
		}
	} else {
		n.UUID = p.User.Username()
		n.Password, _ = p.User.Password()
	}
}

func extractSSAuth(parsed *url.URL) (string, string) {
	if parsed.User == nil {
		return "", ""
	}
	if parts := strings.SplitN(parsed.User.String(), ":", 2); len(parts) == 2 {
		return parts[0], parts[1]
	}
	if decoded, err := DecodeBase64(parsed.User.String()); err == nil {
		if dParts := strings.SplitN(decoded, ":", 2); len(dParts) == 2 {
			return dParts[0], dParts[1]
		}
	}
	return "", ""
}

func parseSSPlugin(node *ProxyNode, query url.Values) {
	if pluginStr := query.Get("plugin"); pluginStr != "" {
		parts := strings.Split(pluginStr, ";")
		name := strings.TrimSpace(parts[0])
		if name == "simple-obfs" || name == "obfs-local" {
			name = "obfs"
		}

		node.Options["plugin"] = name
		opts := make(map[string]any)
		for _, part := range parts[1:] {
			if kv := strings.SplitN(strings.TrimSpace(part), "=", 2); len(kv) == 2 {
				k, v := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
				switch k {
				case "obfs":
					opts["mode"] = v
				case "obfs-host":
					opts["host"] = v
				case "obfs-uri", "path":
					opts["path"] = v
				case "tls":
					opts["tls"] = true
				default:
					opts[k] = v
				}
			}
		}
		if len(opts) > 0 {
			node.Options["plugin-opts"] = opts
		}
	}
}

func parseSSParts(authPart, serverPart, originalLink string) (*ProxyNode, error) {
	auth, server := strings.SplitN(authPart, ":", 2), strings.SplitN(serverPart, ":", 2)
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
		Name: getFragment(parsed, fmt.Sprintf("SS-%s:%d", server[0], port)),
		Type: "ss", Server: server[0], Port: port, Cipher: auth[0], Password: auth[1],
		Options: make(map[string]any),
	}
	if parsed != nil {
		parseSSPlugin(node, parsed.Query())
	}
	return node, nil
}

func splitURLComponents(raw string) (base, query, fragment string) {
	base = raw
	if idx := strings.Index(base, "?"); idx != -1 {
		query, base = base[idx+1:], base[:idx]
		if i2 := strings.Index(query, "#"); i2 != -1 {
			fragment, query = query[i2+1:], query[:i2]
		}
	} else if idx := strings.Index(base, "#"); idx != -1 {
		fragment, base = base[idx+1:], base[:idx]
	}
	return
}

func parseSOCKSBase64Auth(decoded string) (server string, port int, username, password string) {
	dec, _ := url.QueryUnescape(decoded)
	if parts := strings.SplitN(dec, "@", 2); len(parts) == 2 {
		// 使用 LastIndex 从右往左找最后一个冒号
		// 这样可以正确处理用户名中包含冒号的情况（如 pro:u2025887）
		if idx := strings.LastIndex(parts[0], ":"); idx != -1 {
			username, password = parts[0][:idx], parts[0][idx+1:]
		} else {
			username = parts[0]
		}

		sp := strings.SplitN(parts[1], ":", 2)
		server = sp[0]
		if len(sp) == 2 {
			port, _ = strconv.Atoi(sp[1])
		}
	}
	return
}

func extractSOCKSNodeName(server string, port int, fragment string, q url.Values) string {
	if rm := q.Get("remarks"); rm != "" {
		if d, err := url.QueryUnescape(rm); err == nil {
			return d
		}
		return rm
	}
	if fragment != "" {
		if d, err := url.QueryUnescape(fragment); err == nil {
			return d
		}
		return fragment
	}
	return fmt.Sprintf("SOCKS-%s:%d", server, port)
}

func applySOCKSGOSTOptions(node *ProxyNode, gostParam string) {
	if gostParam == "" {
		return
	}
	if gj, err := DecodeBase64(gostParam); err == nil {
		var cfg map[string]any
		if json.Unmarshal([]byte(gj), &cfg) == nil && cfg["route"] == "ws" {
			node.Network = "ws"
			wsOpts := make(map[string]any)
			if p, ok := cfg["path"].(string); ok && p != "" {
				wsOpts["path"] = p
			}
			if h, ok := cfg["host"].(string); ok && h != "" {
				wsOpts["headers"] = map[string]any{"Host": h}
			}
			if len(wsOpts) > 0 {
				node.Options["ws-opts"] = wsOpts
			}
		}
	}
}

func parseGenericNode(link, nodeType string, modifier func(*ProxyNode, url.Values, *url.URL)) (*ProxyNode, error) {
	parsed, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	node := &ProxyNode{
		Name: getFragment(parsed, fmt.Sprintf("%s-%s:%s", strings.ToUpper(nodeType), parsed.Hostname(), parsed.Port())),
		Type: nodeType, Server: parsed.Hostname(), Port: getPort(parsed),
		Options: make(map[string]any),
	}
	modifier(node, parsed.Query(), parsed)
	return node, nil
}

func applyTLSOptions(node *ProxyNode, q url.Values, defSNI string) {
	node.Options["skip-cert-verify"] = isTrue(q.Get("allowInsecure")) || isTrue(q.Get("insecure")) || isTrue(q.Get("allow_insecure"))
	node.Options["servername"] = firstNotEmpty(q.Get("sni"), q.Get("peer"), defSNI)
	if fp := q.Get("fp"); fp != "" {
		node.Options["client-fingerprint"] = fp
	}
	if alpn := q.Get("alpn"); alpn != "" {
		node.Options["alpn"] = strings.Split(alpn, ",")
	}
}

func applyTransportOptions(node *ProxyNode, q url.Values) {
	path, host := q.Get("path"), q.Get("host")
	switch node.Network {
	case "ws":
		opts := make(map[string]any)
		if path != "" {
			opts["path"] = path
		}
		if host != "" {
			opts["headers"] = map[string]any{"Host": host}
		}
		if len(opts) > 0 {
			node.Options["ws-opts"] = opts
		}
	case "grpc":
		if srv := firstNotEmpty(q.Get("serviceName"), path); srv != "" {
			node.Options["grpc-opts"] = map[string]any{"grpc-service-name": srv}
		}
	case "tcp":
		if hType := q.Get("headerType"); hType != "" {
			node.Options["header-type"] = hType
		}
	}
}

func applyTransportMapping(node *ProxyNode, network, path, host, obfs string) {
	switch network {
	case "tcp":
		if obfs == "http" {
			node.Network = "http"
			opts := map[string]any{"method": "GET", "path": []string{path}}
			if host != "" {
				opts["headers"] = map[string]any{"Host": []string{host}}
			}
			node.Options["http-opts"] = opts
		}
	case "ws":
		node.Options["ws-opts"] = map[string]any{"path": path, "headers": map[string]any{"Host": host}}
	case "grpc":
		node.Options["grpc-opts"] = map[string]any{"grpc-service-name": path}
	case "h2":
		node.Options["h2-opts"] = map[string]any{"path": path, "host": []string{host}}
	case "httpupgrade":
		node.Network = "ws"
		node.Options["ws-opts"] = map[string]any{"path": path, "headers": map[string]any{"Host": host}, "v2ray-http-upgrade": true}
	}
}

func applyRealityOptions(node *ProxyNode, q url.Values) {
	reality := make(map[string]any)
	if pbk := q.Get("pbk"); pbk != "" {
		reality["public-key"] = pbk
	}
	if sid := q.Get("sid"); sid != "" {
		reality["short-id"] = sid
	}
	if pqv := q.Get("pqv"); pqv != "" {
		reality["pqv"] = pqv
	}
	if len(reality) > 0 {
		node.Options["reality-opts"] = reality
	}
}

func applyHysteriaBandwidth(node *ProxyNode, q url.Values, upK, downK string) {
	if up := q.Get(upK); up != "" {
		node.Options["up"] = strings.TrimSuffix(up, " mbps") + " mbps"
	}
	if down := q.Get(downK); down != "" {
		node.Options["down"] = strings.TrimSuffix(down, " mbps") + " mbps"
	}
}

func isTrue(s string) bool { return s == "1" || s == "true" }

func DecodeBase64(s string) (string, error) {
	s = strings.TrimSpace(s)
	clean := strings.Map(func(r rune) rune {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '+' || r == '/' || r == '-' || r == '_' || r == '=' {
			return r
		}
		return -1
	}, s)

	if clean == "" {
		return "", nil
	}

	// 统一转为标准 Base64 字典并补全 Padding，避免多次使用不同 Encoder 重试
	clean = strings.ReplaceAll(strings.ReplaceAll(clean, "-", "+"), "_", "/")
	if pad := len(clean) % 4; pad != 0 {
		clean += strings.Repeat("=", 4-pad)
	}

	b, err := base64.StdEncoding.DecodeString(clean)
	if err != nil {
		return "", fmt.Errorf("Base64 解析失败")
	}
	return string(b), nil
}

func TryDecodeNodeList(content string) string {
	if containsNodeLinks(content) {
		return content
	}
	if d, err := DecodeBase64(content); err == nil && containsNodeLinks(d) {
		return d
	}

	var result []string
	changed := false
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if d, err := DecodeBase64(line); err == nil && (containsNodeLinks(d) || strings.Contains(d, "://")) {
			result, changed = append(result, d), true
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
	for _, p := range []string{"vmess://", "vless://", "trojan://", "ss://", "ssr://"} {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

func getString(m map[string]any, key, def string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return def
}

func getInt(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case string:
		i, _ := strconv.Atoi(v)
		return i
	}
	return 0
}

func getFloat(m map[string]any, key string) float64 {
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

func getBool(m map[string]any, key string, def bool) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	if v, ok := m[key].(string); ok {
		return isTrue(v)
	}
	return def
}

func getFragment(p *url.URL, def string) string {
	if p.Fragment != "" {
		if d, err := url.QueryUnescape(p.Fragment); err == nil {
			return d
		}
		return p.Fragment
	}
	return def
}

func getPort(p *url.URL) int {
	if port := p.Port(); port != "" {
		i, _ := strconv.Atoi(port)
		return i
	}
	if p.Scheme == "ss" || p.Scheme == "ssr" {
		return 8388
	}
	return 443
}

func firstNotEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

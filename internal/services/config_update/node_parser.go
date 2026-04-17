package config_update

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ProxyNode 统一代理节点结构体
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

// ParseNodeLink 解析任意支持的代理链接
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

	// 检查是否是非标准格式：auto:UUID@Server:Port (支持 IPv6)
	if strings.Contains(decoded, "@") && !strings.HasPrefix(decoded, "{") {
		parts := strings.SplitN(decoded, "@", 2)
		if len(parts) == 2 {
			uuidParts := strings.Split(parts[0], ":")
			uuid := uuidParts[len(uuidParts)-1]

			server, port := parseHostPort(parts[1])
			if server != "" && port > 0 && uuid != "" {
				return &ProxyNode{
					Name: fmt.Sprintf("VMess-%s:%d", server, port),
					Type: "vmess", Server: server, Port: port, UUID: uuid,
					Network: "tcp", UDP: true,
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
		Name: getString(data, "ps", fmt.Sprintf("VMess-%s:%d", server, port)),
		Type: "vmess", Server: server, Port: port, UUID: uuid, Network: network, UDP: true,
		Cipher:  getString(data, "scy", "auto"),
		Options: map[string]any{"alterId": int(getFloat(data, "aid"))},
	}

	if getString(data, "tls", "") == "tls" {
		node.TLS = true
		node.Options["skip-cert-verify"] = getBool(data, "allowInsecure", false)
		if sni := firstNotEmpty(getString(data, "sni", ""), getString(data, "host", "")); sni != "" {
			node.Options["servername"] = sni
		}
		if alpn := getString(data, "alpn", ""); alpn != "" {
			node.Options["alpn"] = strings.Split(alpn, ",")
		}
	}

	applyTransportMapping(node, network, getString(data, "path", "/"), getString(data, "host", server), getString(data, "type", ""))
	return node, nil
}

func parseVLESS(link string) (*ProxyNode, error) {
	// 偶尔会有 vless://[base64]?params... 的包装格式，预先做个提取容错
	if strings.HasPrefix(link, "vless://") {
		parts := strings.SplitN(link[8:], "?", 2)
		hostPart := strings.Split(parts[0], "#")[0]

		if decoded, err := DecodeBase64(hostPart); err == nil && strings.Contains(decoded, "@") {
			link = "vless://" + decoded
			if len(parts) > 1 {
				link += "?" + parts[1]
			}
		}
	}

	return parseGenericNode(link, "vless", func(n *ProxyNode, q url.Values, p *url.URL) {
		n.UUID = p.User.Username()

		// 兼容类似 auto:UUID 的异常格式，此时真正的 UUID 会被 url.Parse 识别为 Password
		if pwd, ok := p.User.Password(); ok && pwd != "" && n.UUID == "auto" {
			n.UUID = pwd
		} else if strings.Contains(n.UUID, ":") {
			parts := strings.Split(n.UUID, ":")
			n.UUID = parts[len(parts)-1]
		}

		n.Network = firstNotEmpty(q.Get("type"), "tcp")
		n.UDP = true

		sec := q.Get("security")
		if sec == "tls" || sec == "xtls" || sec == "reality" || isTrue(q.Get("tls")) {
			n.TLS = true
			applyTLSOptions(n, q, p.Hostname())

			// 指纹默认 fallback
			if fp := q.Get("fp"); fp != "" {
				n.Options["client-fingerprint"] = fp
			} else {
				n.Options["client-fingerprint"] = "chrome"
			}

			// Reality 处理
			if sec == "reality" || q.Get("pbk") != "" {
				applyRealityOptions(n, q)
			}

			// XTLS/流控处理
			flow := q.Get("flow")
			if flow == "" && q.Get("xtls") != "" {
				if q.Get("xtls") == "2" {
					flow = "xtls-rprx-vision"
				} else if q.Get("xtls") == "1" {
					flow = "xtls-rprx-direct"
				}
			}
			if flow != "" {
				n.Options["flow"] = flow
			}
		}

		n.Options["encryption"] = firstNotEmpty(q.Get("encryption"), "none")
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
	// 处理不包含 @ 的纯 Base64 的旧版 SS 链接
	if !strings.Contains(link, "@") {
		encoded := strings.TrimPrefix(link, "ss://")
		encoded = strings.SplitN(encoded, "#", 2)[0]
		if decoded, err := DecodeBase64(encoded); err == nil && !strings.HasPrefix(decoded, "{") {
			if parts := strings.SplitN(decoded, "@", 2); len(parts) == 2 {
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
	l := len(mainParts)
	// 逆向取值，防止 IPv6 地址内的冒号破坏解析
	if l < 6 {
		return nil, fmt.Errorf("SSR 格式错误")
	}

	port, _ := strconv.Atoi(mainParts[l-5])
	password, _ := DecodeBase64(mainParts[l-1])
	host := strings.Join(mainParts[:l-5], ":") // 如果是 IPv6，组合回原来的样子

	node := &ProxyNode{
		Name: fmt.Sprintf("SSR-%s:%d", host, port),
		Type: "ssr", Server: host, Port: port, Password: password, Cipher: mainParts[l-3],
		Options: map[string]any{"protocol": mainParts[l-4], "obfs": mainParts[l-2]},
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
		applyTLSOptions(n, q, n.Server) // Hysteria v1 也有可能含 SNI 和 Insecure

		if obfs := firstNotEmpty(q.Get("obfsParam"), q.Get("obfs")); obfs != "" {
			n.Options["obfs"] = obfs
		}
	})
}

func parseHysteria2(link string) (*ProxyNode, error) {
	return parseGenericNode(link, "hysteria2", func(n *ProxyNode, q url.Values, p *url.URL) {
		extractAuthToNode(n, p, true)
		n.TLS = true
		applyHysteriaBandwidth(n, q, "mbpsUp", "mbpsDown")
		applyTLSOptions(n, q, n.Server)

		// Hysteria2 特定混淆参数
		if obfs := q.Get("obfs"); obfs != "" {
			n.Options["obfs"] = obfs
		}
		if obfsPwd := q.Get("obfs-password"); obfsPwd != "" {
			n.Options["obfs-password"] = obfsPwd
		}
		if pin := q.Get("pinSHA256"); pin != "" {
			n.Options["pinSHA256"] = pin
		}
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
		n.UDP = q.Get("udp") == "1" || isTrue(q.Get("udp")) || q.Get("udp") == ""

		// URL Query 会将 Base64 里的 '+' 变成 ' ' (空格)，这里直接替换回去即可
		n.Options["public-key"] = getBase64Query(q, "publicKey")
		n.Options["private-key"] = getBase64Query(q, "privateKey")
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

// ---------------------- 辅助工具函数 ----------------------

// parseHostPort 手动分割 Host 和 Port 以完美兼容 IPv6 的中括号模式
func parseHostPort(s string) (host string, port int) {
	if strings.HasPrefix(s, "[") {
		if end := strings.LastIndex(s, "]"); end != -1 {
			host = s[1:end]
			if end+2 < len(s) && s[end+1] == ':' {
				port, _ = strconv.Atoi(s[end+2:])
			}
			return
		}
	}
	if idx := strings.LastIndex(s, ":"); idx != -1 {
		host = s[:idx]
		port, _ = strconv.Atoi(s[idx+1:])
		return
	}
	return s, 0
}

func getBase64Query(q url.Values, key string) string {
	return strings.ReplaceAll(q.Get(key), " ", "+")
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
	auth := strings.SplitN(authPart, ":", 2)
	if len(auth) != 2 {
		return nil, fmt.Errorf("SS 格式解析失败")
	}

	hostPort := strings.SplitN(serverPart, "?", 2)[0]
	hostPort = strings.SplitN(hostPort, "#", 2)[0]
	serverHost, port := parseHostPort(hostPort)

	parsed, _ := url.Parse(originalLink)

	node := &ProxyNode{
		Name: getFragment(parsed, fmt.Sprintf("SS-%s:%d", serverHost, port)),
		Type: "ss", Server: serverHost, Port: port, Cipher: auth[0], Password: auth[1],
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
		if idx := strings.LastIndex(parts[0], ":"); idx != -1 {
			username, password = parts[0][:idx], parts[0][idx+1:]
		} else {
			username = parts[0]
		}
		server, port = parseHostPort(parts[1])
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
	if sni := firstNotEmpty(q.Get("sni"), q.Get("peer"), defSNI); sni != "" {
		node.Options["servername"] = sni
	}
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
		if srv := firstNotEmpty(q.Get("serviceName"), q.Get("authority"), path); srv != "" {
			node.Options["grpc-opts"] = map[string]any{"grpc-service-name": srv}
		}
	case "tcp":
		if hType := q.Get("headerType"); hType != "" {
			if hType == "http" {
				node.Network = "http"
				opts := map[string]any{"method": "GET"}
				if path != "" {
					opts["path"] = strings.Split(path, ",")
				}
				if host != "" {
					opts["headers"] = map[string]any{"Host": strings.Split(host, ",")}
				}
				node.Options["http-opts"] = opts
			} else {
				node.Options["header-type"] = hType
			}
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
	if spx := q.Get("spx"); spx != "" {
		reality["spider-x"] = spx
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

func isTrue(s string) bool {
	s = strings.ToLower(s)
	return s == "1" || s == "true" || s == "yes"
}

// DecodeBase64 智能且健壮的 Base64 解码器
func DecodeBase64(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}

	// 优先尝试各种标准/原始/URL编码组合，防止破坏 JSON 内的特殊符号
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.RawURLEncoding,
	}
	for _, enc := range encodings {
		if b, err := enc.DecodeString(s); err == nil {
			return string(b), nil
		}
	}

	// 失败后尝试手动修复非标编码（常用于不严谨的订阅环境）
	alt := strings.ReplaceAll(strings.ReplaceAll(s, "-", "+"), "_", "/")
	if pad := len(alt) % 4; pad != 0 {
		alt += strings.Repeat("=", 4-pad)
	}
	if b, err := base64.StdEncoding.DecodeString(alt); err == nil {
		return string(b), nil
	}

	return "", fmt.Errorf("Base64 解析失败")
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
	for _, p := range []string{"vmess://", "vless://", "trojan://", "ss://", "ssr://", "hysteria2://", "tuic://", "socks://", "socks5://", "hysteria://", "naive+https://", "naive://", "anytls://", "wg://"} {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

func getString(m map[string]any, key, def string) string {
	if v, ok := m[key].(string); ok && v != "" {
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

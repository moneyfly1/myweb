package config_update

import (
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNodeToMapServernameToSNI(t *testing.T) {
	s := &ConfigUpdateService{}
	n := &ProxyNode{
		Name: "Test-Trojan", Type: "trojan",
		Server: "example.com", Port: 443, Password: "test123", TLS: true,
		Options: map[string]interface{}{"servername": "sni.example.com"},
	}
	m := s.nodeToMap(n)
	if _, ok := m["servername"]; ok {
		t.Error("servername should be converted to sni")
	}
	if sn, ok := m["sni"]; !ok || sn != "sni.example.com" {
		t.Errorf("sni field wrong: %v", m["sni"])
	}
}

func TestNodeToYAMLFlowStyle(t *testing.T) {
	s := &ConfigUpdateService{}
	n := &ProxyNode{
		Name: "Test-Trojan", Type: "trojan",
		Server: "example.com", Port: 443, Password: "test123", TLS: true,
		Options: map[string]interface{}{"servername": "sni.example.com"},
	}
	out := s.nodeToYAML(n, 0)
	t.Log(out)
	if strings.Count(out, "\n") > 2 {
		t.Errorf("Expected compact (1-2 lines), got multi-line: %s", out)
	}
	if !strings.Contains(out, "- {") {
		t.Errorf("Expected flow-style '- {', got: %s", out)
	}
	if strings.Contains(out, "servername") {
		t.Error("Should not contain servername")
	}
	if !strings.Contains(out, "sni:") {
		t.Error("Should contain sni:")
	}
}

func TestTemplateFlowStyleConversion(t *testing.T) {
	s := &ConfigUpdateService{}
	n := &ProxyNode{
		Name: "FastStunnel-Test", Type: "trojan",
		Server: "104.248.167.79", Port: 5624, Password: "fg20mtCFHy", TLS: true,
		UDP: true, Network: "tcp",
		Options: map[string]interface{}{
			"servername":         "dsgvc.southbyte.xyz",
			"client-fingerprint": "firefox",
			"flow":               "xtls-rprx-vision",
		},
	}
	// Simulate template path: nodeToMap -> yaml.Marshal -> yaml.Unmarshal -> FlowStyle -> yaml.Marshal
	m := s.nodeToMap(n)
	data, _ := yaml.Marshal(m)
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Content) == 0 {
		t.Fatal("no content in doc")
	}
	flowNode := *doc.Content[0]
	flowNode.Style = yaml.FlowStyle
	out, err := yaml.Marshal(&flowNode)
	if err != nil {
		t.Fatal(err)
	}
	result := string(out)
	t.Log(result)
	if !strings.Contains(result, "name:") {
		t.Errorf("missing name field")
	}
	if strings.Contains(result, "servername") {
		t.Errorf("should be sni, not servername")
	}
	if !strings.Contains(result, "sni:") {
		t.Errorf("missing sni field")
	}
}

func TestFormatYAMLFlow(t *testing.T) {
	s := &ConfigUpdateService{}
	if got := s.formatYAMLFlow("hello world"); got != `"hello world"` {
		t.Errorf("want quoted: got %s", got)
	}
	if got := s.formatYAMLFlow(true); got != "true" {
		t.Errorf("bool: got %s", got)
	}
	if got := s.formatYAMLFlow(42); got != "42" {
		t.Errorf("int: got %s", got)
	}
	if got := s.formatYAMLFlow(""); got != `""` {
		t.Errorf("empty: got %s", got)
	}
}

func TestNodeToYAMLNestedOptions(t *testing.T) {
	s := &ConfigUpdateService{}
	n := &ProxyNode{
		Name: "Test-WS", Type: "vmess", Server: "example.com", Port: 443,
		UUID: "b831381d-6324-4d53-ad4f-8cda48b30836", Network: "ws",
		Options: map[string]interface{}{
			"servername": "sni.example.com",
			"ws-opts": map[string]interface{}{
				"path":    "/ws-path",
				"headers": map[string]interface{}{"Host": "ws.example.com"},
			},
		},
	}
	out := s.nodeToYAML(n, 0)
	t.Log(out)
	if !strings.Contains(out, "ws-opts: {") {
		t.Errorf("Expected inline ws-opts: %s", out)
	}
}

func TestAlpnList(t *testing.T) {
	s := &ConfigUpdateService{}
	result := s.formatYAMLFlow([]string{"h2", "http/1.1"})
	t.Log(result)
	if !strings.Contains(result, "[") || !strings.Contains(result, "]") {
		t.Errorf("Expected inline list: %s", result)
	}
}

func TestClashURLSamplesUseNativeOptions(t *testing.T) {
	s := &ConfigUpdateService{}

	vlessWS := "vless://33c41229-3e5a-456f-bf62-e050d2b84d81@usbwg.icandoit.eu.org:26823?encryption=none&security=tls&sni=usbwg.icandoit.eu.org&fp=chrome&insecure=0&allowInsecure=0&type=ws&host=usbwg.icandoit.eu.org&path=%2Fzcxgws#%E4%B8%93%E7%BA%BF-%E7%BE%8E%E5%9B%BD%E6%90%AC%E7%93%A6%E5%B7%A5VLESS_WS"
	node, err := ParseNodeLink(vlessWS)
	if err != nil {
		t.Fatal(err)
	}
	m := s.nodeToMap(node)
	if m["network"] != "ws" {
		t.Fatalf("network = %v, want ws", m["network"])
	}
	ws, ok := m["ws-opts"].(map[string]any)
	if !ok {
		t.Fatalf("missing ws-opts: %#v", m)
	}
	if ws["path"] != "/zcxgws" {
		t.Fatalf("ws path = %v", ws["path"])
	}
	headers, ok := ws["headers"].(map[string]any)
	if !ok || headers["Host"] != "usbwg.icandoit.eu.org" {
		t.Fatalf("ws headers = %#v", ws["headers"])
	}

	vlessGRPC := "vless://33c41229-3e5a-456f-bf62-e050d2b84d81@usbwg.icandoit.eu.org:23435?encryption=none&security=reality&sni=usbwg.icandoit.eu.org&fp=chrome&pbk=lf2FVJzxSafTmEvbgJdGwc9-dAR_5OGP20JxDuimbgc&sid=6ba85179e30d4fc2&type=grpc&authority=&serviceName=grpc&mode=gun#%E4%B8%93%E7%BA%BF-%E7%BE%8E%E5%9B%BD%E6%90%AC%E7%93%A6%E5%B7%A5VLESS_Reality_gPRC"
	node, err = ParseNodeLink(vlessGRPC)
	if err != nil {
		t.Fatal(err)
	}
	m = s.nodeToMap(node)
	reality, ok := m["reality-opts"].(map[string]any)
	if !ok || reality["public-key"] != "lf2FVJzxSafTmEvbgJdGwc9-dAR_5OGP20JxDuimbgc" || reality["short-id"] != "6ba85179e30d4fc2" {
		t.Fatalf("reality-opts = %#v", m["reality-opts"])
	}
	grpc, ok := m["grpc-opts"].(map[string]any)
	if !ok || grpc["grpc-service-name"] != "grpc" || grpc["grpc-mode"] != "gun" {
		t.Fatalf("grpc-opts = %#v", m["grpc-opts"])
	}

	tuic := "tuic://33c41229-3e5a-456f-bf62-e050d2b84d81%3A33c41229-3e5a-456f-bf62-e050d2b84d81@usbwg.icandoit.eu.org:15074?sni=usbwg.icandoit.eu.org&alpn=h3&insecure=0&allowInsecure=0&congestion_control=bbr#%E4%B8%93%E7%BA%BF-%E7%BE%8E%E5%9B%BD%E6%90%AC%E7%93%A6%E5%B7%A5singbox_tuic"
	node, err = ParseNodeLink(tuic)
	if err != nil {
		t.Fatal(err)
	}
	m = s.nodeToMap(node)
	if m["server"] != "usbwg.icandoit.eu.org" || m["port"] != 15074 {
		t.Fatalf("tuic endpoint = %v:%v", m["server"], m["port"])
	}
	if m["uuid"] != "33c41229-3e5a-456f-bf62-e050d2b84d81" || m["password"] != "33c41229-3e5a-456f-bf62-e050d2b84d81" {
		t.Fatalf("tuic auth uuid=%v password=%v", m["uuid"], m["password"])
	}
	if m["congestion-controller"] != "bbr" {
		t.Fatalf("tuic congestion-controller = %v", m["congestion-controller"])
	}
}

func TestClashVLESSWSSampleKeepsTLSAndWSOptions(t *testing.T) {
	s := &ConfigUpdateService{}

	raw := "vless://15884975-9b74-4502-b6a3-f69df7ec93d7@n1743918709.4u9ma.icu:443?encryption=none&security=tls&sni=n1743918709.4u9ma.icu&fp=chrome&insecure=0&allowInsecure=0&type=ws&host=n1743918709.4u9ma.icu&path=%2F15884975-9b74-4502-b6a3-f69df7ec93d7#%E6%97%A5%E6%9C%AC01%E5%BF%AB%E6%A9%99"
	node, err := ParseNodeLink(raw)
	if err != nil {
		t.Fatal(err)
	}
	m := s.nodeToMap(node)

	if m["type"] != "vless" || m["server"] != "n1743918709.4u9ma.icu" || m["port"] != 443 {
		t.Fatalf("endpoint = type:%v server:%v port:%v", m["type"], m["server"], m["port"])
	}
	if m["uuid"] != "15884975-9b74-4502-b6a3-f69df7ec93d7" || m["tls"] != true || m["network"] != "ws" {
		t.Fatalf("core fields uuid=%v tls=%v network=%v", m["uuid"], m["tls"], m["network"])
	}
	if _, ok := m["udp"]; ok {
		t.Fatalf("vless ws should not emit udp by default: %#v", m["udp"])
	}
	if m["tfo"] != false {
		t.Fatalf("tfo = %v, want false", m["tfo"])
	}
	if m["servername"] != "n1743918709.4u9ma.icu" || m["client-fingerprint"] != "chrome" || m["skip-cert-verify"] != true {
		t.Fatalf("tls opts servername=%v fp=%v skip=%v", m["servername"], m["client-fingerprint"], m["skip-cert-verify"])
	}
	ws, ok := m["ws-opts"].(map[string]any)
	if !ok {
		t.Fatalf("missing ws-opts: %#v", m)
	}
	if ws["path"] != "/15884975-9b74-4502-b6a3-f69df7ec93d7" {
		t.Fatalf("path = %v", ws["path"])
	}
	headers, ok := ws["headers"].(map[string]any)
	if !ok || headers["Host"] != "n1743918709.4u9ma.icu" {
		t.Fatalf("headers = %#v", ws["headers"])
	}

	out := s.nodeToYAML(node, 0)
	wantPrefix := "- {name: 日本01快橙, server: n1743918709.4u9ma.icu, port: 443, type: vless, uuid: 15884975-9b74-4502-b6a3-f69df7ec93d7, tls: true, tfo: false, skip-cert-verify: true, servername: n1743918709.4u9ma.icu, client-fingerprint: chrome, network: ws, ws-opts:"
	if !strings.HasPrefix(out, wantPrefix) {
		t.Fatalf("unexpected yaml order:\n%s", out)
	}
	if strings.Contains(out, "udp: true") {
		t.Fatalf("vless yaml should not contain udp: %s", out)
	}
}

func TestClashTemplateFlowNodeKeepsConverterLikeVLESSOrder(t *testing.T) {
	s := &ConfigUpdateService{}

	raw := "vless://15884975-9b74-4502-b6a3-f69df7ec93d7@n1743918709.4u9ma.icu:443?encryption=none&security=tls&sni=n1743918709.4u9ma.icu&fp=chrome&insecure=0&allowInsecure=0&type=ws&host=n1743918709.4u9ma.icu&path=%2F15884975-9b74-4502-b6a3-f69df7ec93d7#%E6%97%A5%E6%9C%AC01%E5%BF%AB%E6%A9%99"
	node, err := ParseNodeLink(raw)
	if err != nil {
		t.Fatal(err)
	}

	flowNode, err := s.nodeToYAMLFlowNode(node)
	if err != nil {
		t.Fatal(err)
	}
	data, err := yaml.Marshal(&flowNode)
	if err != nil {
		t.Fatal(err)
	}
	out := string(data)
	want := "{name: 日本01快橙, server: n1743918709.4u9ma.icu, port: 443, type: vless, uuid: 15884975-9b74-4502-b6a3-f69df7ec93d7, tls: true, tfo: false, skip-cert-verify: true, servername: n1743918709.4u9ma.icu, client-fingerprint: chrome, network: ws"
	if !strings.Contains(out, want) {
		t.Fatalf("expected converter-like VLESS order in template flow node:\nwant: %s\noutput:\n%s", want, out)
	}
}

func TestGenerateSingBoxConfigUsesValidOutboundFields(t *testing.T) {
	s := &ConfigUpdateService{}
	proxies := []*ProxyNode{
		{
			Name:    "VMess WS",
			Type:    "vmess",
			Server:  "vmess.example.com",
			Port:    443,
			UUID:    "b831381d-6324-4d53-ad4f-8cda48b30836",
			Cipher:  "auto",
			Network: "ws",
			Options: map[string]any{
				"tls": true,
				"ws-opts": map[string]any{
					"path":    "/ws",
					"headers": map[string]any{"Host": "host.example.com"},
				},
			},
		},
		{
			Name:   "VLESS Reality",
			Type:   "vless",
			Server: "203.0.113.10",
			Port:   443,
			UUID:   "33c41229-3e5a-456f-bf62-e050d2b84d81",
			TLS:    true,
			Options: map[string]any{
				"flow":               "xtls-rprx-vision",
				"servername":         "apple.com",
				"client-fingerprint": "chrome",
				"reality-opts": map[string]any{
					"public-key": "public-key-value",
					"short-id":   "ef0cb521",
				},
			},
		},
	}

	var config struct {
		Outbounds []map[string]any `json:"outbounds"`
	}
	if err := json.Unmarshal([]byte(s.generateSingBoxConfig(proxies)), &config); err != nil {
		t.Fatal(err)
	}
	if len(config.Outbounds) != 3 {
		t.Fatalf("outbounds len = %d, want 3", len(config.Outbounds))
	}

	direct := config.Outbounds[0]
	if direct["type"] != "direct" || direct["tag"] != "DIRECT" {
		t.Fatalf("unexpected direct outbound: %#v", direct)
	}
	if _, ok := direct["server"]; ok {
		t.Fatalf("direct outbound must not contain server: %#v", direct)
	}
	if _, ok := direct["server_port"]; ok {
		t.Fatalf("direct outbound must not contain server_port: %#v", direct)
	}

	vmess := config.Outbounds[1]
	if vmess["security"] != "auto" {
		t.Fatalf("vmess security = %v, want auto", vmess["security"])
	}
	if _, ok := vmess["method"]; ok {
		t.Fatalf("vmess outbound must not contain method: %#v", vmess)
	}
	transport, ok := vmess["transport"].(map[string]any)
	if !ok {
		t.Fatalf("missing vmess transport: %#v", vmess)
	}
	if _, ok := transport["host"]; ok {
		t.Fatalf("sing-box ws transport must not contain host: %#v", transport)
	}
	headers, ok := transport["headers"].(map[string]any)
	if !ok || headers["Host"] != "host.example.com" {
		t.Fatalf("ws headers = %#v", transport["headers"])
	}

	vless := config.Outbounds[2]
	if vless["flow"] != "xtls-rprx-vision" {
		t.Fatalf("vless flow = %v", vless["flow"])
	}
	tls, ok := vless["tls"].(map[string]any)
	if !ok {
		t.Fatalf("missing vless tls: %#v", vless)
	}
	utls, ok := tls["utls"].(map[string]any)
	if !ok || utls["fingerprint"] != "chrome" {
		t.Fatalf("utls = %#v", tls["utls"])
	}
	reality, ok := tls["reality"].(map[string]any)
	if !ok || reality["public_key"] != "public-key-value" || reality["short_id"] != "ef0cb521" {
		t.Fatalf("reality = %#v", tls["reality"])
	}
}

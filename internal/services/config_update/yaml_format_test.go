package config_update

import (
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
			"servername": "dsgvc.southbyte.xyz",
			"client-fingerprint": "firefox",
			"flow": "xtls-rprx-vision",
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
				"path": "/ws-path",
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

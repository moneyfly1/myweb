package config_update

import (
	"net/url"
	"strings"
	"testing"
)

func TestEscapeYAMLString(t *testing.T) {
	var s ConfigUpdateService

	// 会被 YAML 1.1 解析器（js-yaml 3 等）解析为非字符串标量的值必须加引号
	quoted := []string{
		"953e8078", // 本 bug 事故值：浮点形式（数字+e+数字），js-yaml 3 解析成 Infinity
		"76827754", // 纯数字（int 形式）
		"953e807",  // 无小数点的浮点形式（溢出）
		"0123", "007531", "0x1a", "0b101", "0o17", "1e3", "-1.5", "+42",
		".inf", "-.inf", ".nan", "1_000",
		"123456789012345678901234567890", // int64 溢出
		"yes", "YES", "No", "on", "OFF", "true", "false", "null", "NULL", "~",
		"2026-08-18", // YAML 1.1 timestamp 纯日期形式
		"1:30",       // 含冒号，走特殊字符分支
		"",
	}
	for _, in := range quoted {
		want := `"` + in + `"`
		if got := s.escapeYAMLString(in); got != want {
			t.Errorf("escapeYAMLString(%q) = %q, want %q", in, got, want)
		}
	}

	// 普通字符串保持裸值
	bare := []string{
		"953e", "e8078", "953e807z", "abc123", "abc_123",
		"www.apple.com", "45.12.254.86",
		"550e8400-e29b-41d4-a716-446655440000",
		"deadbeef", "00a735b91f0e", "xtls-rprx-vision", "GotoVPN-德国",
	}
	for _, in := range bare {
		if got := s.escapeYAMLString(in); got != in {
			t.Errorf("escapeYAMLString(%q) = %q, want bare %q", in, got, in)
		}
	}
}

func TestNodeToYAMLQuotesNumericShortID(t *testing.T) {
	var s ConfigUpdateService

	for _, sid := range []string{"953e8078", "76827754"} {
		n := &ProxyNode{
			Name:   "GotoVPN-德国",
			Type:   "vless",
			Server: "45.12.254.86",
			Port:   443,
			UUID:   "c6bc8627-ab12-457b-b143-0ea0de51b1ed",
			TLS:    true,
			Options: map[string]any{
				"flow": "xtls-rprx-vision",
				"reality-opts": map[string]any{
					"public-key": "4Al0xAzgmDer_tEDUj7kx5SJ6A2a4FRi1S5oa0WBMR8",
					"short-id":   sid,
				},
			},
		}
		out := s.nodeToYAML(n, 0)
		quoted := `short-id: "` + sid + `"`
		bare := "short-id: " + sid
		if !strings.Contains(out, quoted) {
			t.Errorf("sid %q: nodeToYAML 输出缺少 %q:\n%s", sid, quoted, out)
		}
		if strings.Contains(out, bare) {
			t.Errorf("sid %q: nodeToYAML 输出了未加引号的 %q:\n%s", sid, bare, out)
		}
	}
}

func TestApplyRealityOptionsShortID(t *testing.T) {
	cases := []struct {
		name    string
		sid     string // 空字符串表示不带 sid 参数
		wantSid string // 期望写入的 short-id，空表示不应存在
	}{
		{"valid-8-hex", "deadbeef", "deadbeef"},
		{"valid-12-hex", "00a735b91f0e", "00a735b91f0e"},
		{"valid-accident-value", "953e8078", "953e8078"},
		{"invalid-non-hex", "xyz123", ""},
		{"invalid-odd-length", "123", ""},
		{"invalid-too-long", "12345678901234567", ""},
		{"missing-sid", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			node := &ProxyNode{Options: map[string]any{}}
			q := url.Values{}
			q.Set("pbk", "pubkey")
			if tc.sid != "" {
				q.Set("sid", tc.sid)
			}
			applyRealityOptions(node, q)

			real, ok := node.Options["reality-opts"]
			if !ok {
				if tc.wantSid != "" {
					t.Fatalf("期望 reality-opts 存在（short-id=%q），实际不存在", tc.wantSid)
				}
				return
			}
			m, ok := real.(map[string]any)
			if !ok {
				t.Fatalf("reality-opts 类型错误: %T", real)
			}
			got, exists := m["short-id"]
			if tc.wantSid == "" {
				if exists {
					t.Errorf("short-id 应被丢弃，实际为 %v", got)
				}
			} else if !exists || got != tc.wantSid {
				t.Errorf("short-id = %v (exists=%v), want %q", got, exists, tc.wantSid)
			}
			if m["public-key"] != "pubkey" {
				t.Errorf("public-key 被意外改动: %v", m["public-key"])
			}
		})
	}
}

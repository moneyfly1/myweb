package handlers

import (
	"encoding/json"
	"testing"
)

func parseNodeConfigForTest(t *testing.T, config string) map[string]interface{} {
	t.Helper()
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(config), &data); err != nil {
		t.Fatalf("config 不是合法 JSON: %v\n%s", err, config)
	}
	return data
}

// 回归：编辑专线节点时 config 是权威来源。
// 更新接口传入的 domain/port 来自数据库旧列值（编辑表单不提交这两个字段），
// 旧实现会用旧列值覆盖 config，导致「改了服务器地址保存后不生效」。
func TestNormalizeCustomNodeConfigConfigWinsOverStaleColumns(t *testing.T) {
	config := `{"Name":"美国家庭专线","Type":"socks","Server":"basic-vip.miyavip.vip","Port":8001,"UUID":"dXNlcjpwYXNz","UDP":true}`

	gotConfig, protocol, domain, port := normalizeCustomNodeConfig(config, "socks", "old.example.com", 8001)

	data := parseNodeConfigForTest(t, gotConfig)
	if data["Server"] != "basic-vip.miyavip.vip" {
		t.Fatalf("config.Server 被旧列值覆盖：期望 basic-vip.miyavip.vip，实际 %v", data["Server"])
	}
	if domain != "basic-vip.miyavip.vip" {
		t.Fatalf("返回的 domain 未跟随 config：期望 basic-vip.miyavip.vip，实际 %s", domain)
	}
	if protocol != "socks" {
		t.Fatalf("protocol 期望 socks，实际 %s", protocol)
	}
	if port != 8001 {
		t.Fatalf("port 期望 8001，实际 %d", port)
	}
}

// 回归：config 里改过的端口同样不能被旧端口列覆盖。
func TestNormalizeCustomNodeConfigPortChangeIsKept(t *testing.T) {
	config := `{"Type":"socks","Server":"basic-vip.miyavip.vip","Port":18001,"UUID":"dXNlcjpwYXNz"}`

	gotConfig, _, _, port := normalizeCustomNodeConfig(config, "socks", "basic-vip.miyavip.vip", 8001)

	if port != 18001 {
		t.Fatalf("config 端口改动被旧列值覆盖：期望 18001，实际 %d", port)
	}
	if v := parseNodeConfigForTest(t, gotConfig)["Port"]; v != float64(18001) {
		t.Fatalf("config.Port 期望 18001，实际 %v", v)
	}
}

// config 缺少 server/port/type 时，仍要用传入值补全（并规范到标准键名）。
func TestNormalizeCustomNodeConfigFillsMissingKeys(t *testing.T) {
	config := `{"Name":"只有名字"}`

	gotConfig, protocol, domain, port := normalizeCustomNodeConfig(config, "vless", "fill.example.com", 443)

	data := parseNodeConfigForTest(t, gotConfig)
	if data["Server"] != "fill.example.com" {
		t.Fatalf("缺失的 Server 未被补全：%v", data["Server"])
	}
	if data["Type"] != "vless" {
		t.Fatalf("缺失的 Type 未被补全：%v", data["Type"])
	}
	if data["Port"] != float64(443) {
		t.Fatalf("缺失的 Port 未被补全：%v", data["Port"])
	}
	if protocol != "vless" || domain != "fill.example.com" || port != 443 {
		t.Fatalf("返回值未补全：protocol=%s domain=%s port=%d", protocol, domain, port)
	}
}

// 历史数据常用 add/address 等别名键：规范化后值不变，且补上标准键。
func TestNormalizeCustomNodeConfigCanonicalizesAliasKeys(t *testing.T) {
	config := `{"Type":"vmess","add":"alias.example.com","port":"8443"}`

	gotConfig, _, domain, port := normalizeCustomNodeConfig(config, "", "", 0)

	data := parseNodeConfigForTest(t, gotConfig)
	if data["Server"] != "alias.example.com" {
		t.Fatalf("别名键 add 的值被改写：%v", data["Server"])
	}
	if data["add"] != "alias.example.com" {
		t.Fatalf("别名键 add 被改动：%v", data["add"])
	}
	if domain != "alias.example.com" {
		t.Fatalf("domain 期望 alias.example.com，实际 %s", domain)
	}
	if port != 8443 {
		t.Fatalf("字符串端口 8443 未识别：%d", port)
	}
}

// 表单**显式提交**的字段（「节点类型」下拉框等）优先级最高：
// 先写进 config，再走规范化，最终 config 与三列都应是显式值。
// 这条与「旧列值不得覆盖 config」互为补充，避免修 A 引入 B。
func TestExplicitNodeConfigOverridesBeatConfig(t *testing.T) {
	config := `{"Type":"socks","Server":"a.example.com","Port":8001,"UUID":"dXNlcjpwYXNz"}`

	gotConfig, protocol, domain, port := normalizeCustomNodeConfig(
		applyExplicitNodeConfigOverrides(config, "http", "b.example.com", 9000),
		"http",
		"b.example.com",
		9000,
	)

	data := parseNodeConfigForTest(t, gotConfig)
	if data["Type"] != "http" {
		t.Fatalf("显式提交的协议未写进 config：%v", data["Type"])
	}
	if data["Server"] != "b.example.com" {
		t.Fatalf("显式提交的地址未写进 config：%v", data["Server"])
	}
	if data["Port"] != float64(9000) {
		t.Fatalf("显式提交的端口未写进 config：%v", data["Port"])
	}
	if protocol != "http" || domain != "b.example.com" || port != 9000 {
		t.Fatalf("列值未跟随显式提交：protocol=%s domain=%s port=%d", protocol, domain, port)
	}
}

// 只提交 config（编辑表单的真实行为：不含 domain/port）时，
// 显式覆盖层对旧列值不做任何事，config 里的新地址必须保留。
func TestUpdatePathKeepsEditedConfigWhenNoExplicitFields(t *testing.T) {
	edited := `{"Type":"socks","Server":"basic-vip.miyavip.vip","Port":8001,"UUID":"dXNlcjpwYXNz"}`

	gotConfig, protocol, domain, port := normalizeCustomNodeConfig(
		applyExplicitNodeConfigOverrides(edited, "socks", "", 0),
		"socks",
		"old.example.com",
		8001,
	)

	data := parseNodeConfigForTest(t, gotConfig)
	if data["Server"] != "basic-vip.miyavip.vip" {
		t.Fatalf("编辑后的地址被旧列值覆盖：%v", data["Server"])
	}
	if domain != "basic-vip.miyavip.vip" || protocol != "socks" || port != 8001 {
		t.Fatalf("列值未跟随 config：protocol=%s domain=%s port=%d", protocol, domain, port)
	}
}

// 空 config / 非 JSON config 必须原样返回，不能被改写。
func TestNormalizeCustomNodeConfigPassesThroughUnparsable(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config string
	}{
		{"空配置", ""},
		{"仅空白", "   "},
		{"非 JSON（历史脏数据）", "cdn.example.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotConfig, protocol, domain, port := normalizeCustomNodeConfig(tc.config, "socks", "keep.example.com", 8001)
			if gotConfig != tc.config {
				t.Fatalf("config 被改写：期望 %q，实际 %q", tc.config, gotConfig)
			}
			if protocol != "socks" || domain != "keep.example.com" || port != 8001 {
				t.Fatalf("传入值被改动：protocol=%s domain=%s port=%d", protocol, domain, port)
			}
		})
	}
}

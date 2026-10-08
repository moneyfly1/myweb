package selfhost

import (
	"strings"
	"testing"
	"time"
)

// 默认配置下替换必须是恒等操作：老节点重装时目录不能变
func TestApplyTargetPathsDefaultsAreIdentity(t *testing.T) {
	script := "SINGBOX_DIR=\"/usr/local/bin\"\nCONF_DIR=\"/etc/sing-box\"\nCERT_DIR=\"/etc/sing-box/cert\"\n  local script=\"/usr/local/bin/cboard-agent-heartbeat.sh\"\n"
	if got := applyTargetPaths(script); got != script {
		t.Fatalf("默认配置下不应发生替换：\n%s", got)
	}
}

// 用环境变量覆盖后，脚本里的路径必须跟着变（这就是「不要写死路径」的诉求）
func TestApplyTargetPathsOverrides(t *testing.T) {
	t.Setenv(envBinDir, "/opt/singbox/bin")
	t.Setenv(envConfDir, "/opt/singbox/conf")
	t.Setenv(envCertDir, "/opt/singbox/conf/cert")
	t.Setenv(envAcmeHome, "/opt/acme")

	script := "SINGBOX_DIR=\"/usr/local/bin\"\nCONF_DIR=\"/etc/sing-box\"\nCERT_DIR=\"/etc/sing-box/cert\"\n" +
		"  local script=\"/usr/local/bin/cboard-agent-heartbeat.sh\"\n" +
		"  if [ ! -f /root/.acme.sh/acme.sh ]; then\n" +
		"    cp \"/root/.acme.sh/${DOMAIN}_ecc/fullchain.cer\" \"$CERT_DIR/fullchain.pem\"\n"
	got := applyTargetPaths(script)

	for _, want := range []string{
		`SINGBOX_DIR="/opt/singbox/bin"`,
		`CONF_DIR="/opt/singbox/conf"`,
		`CERT_DIR="/opt/singbox/conf/cert"`,
		`script="/opt/singbox/bin/cboard-agent-heartbeat.sh"`,
		`/opt/acme/acme.sh`,
		`"/opt/acme/${DOMAIN}_ecc/fullchain.cer"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("脚本里缺少覆盖后的路径 %q：\n%s", want, got)
		}
	}
	if strings.Contains(got, "/usr/local/bin") || strings.Contains(got, "/etc/sing-box") || strings.Contains(got, "/root/.acme.sh") {
		t.Fatalf("仍残留默认路径：\n%s", got)
	}
}

// 目标机器路径的访问器要能拿到覆盖值，并容忍结尾斜杠
func TestTargetPathAccessors(t *testing.T) {
	t.Setenv(envConfDir, "/opt/sing-box/")
	if got := ConfDir(); got != "/opt/sing-box" {
		t.Fatalf("结尾斜杠未清理：%q", got)
	}
	if got := SingBoxConfigPath(); got != "/opt/sing-box/config.json" {
		t.Fatalf("SingBoxConfigPath 不符：%q", got)
	}
	t.Setenv(envBinDir, "/opt/bin")
	if got := HeartbeatScriptPath(); got != "/opt/bin/cboard-agent-heartbeat.sh" {
		t.Fatalf("HeartbeatScriptPath 不符：%q", got)
	}
}

// 端到端：生成的安装脚本里必须体现配置的目标路径（否则改配置也没用）
func TestBuildInstallScriptUsesConfiguredPaths(t *testing.T) {
	t.Setenv(envConfDir, "/opt/singbox/conf")
	script, err := BuildInstallScript(ScriptConfig{
		PanelBaseURL: "https://panel.example.com",
		InstallID:    "test-install",
		Token:        "test-token",
		Protocol:     "vless-ws",
		NodeName:     "测试节点",
		GeneratedAt:  time.Now(),
	})
	if err != nil {
		t.Fatalf("生成脚本失败: %v", err)
	}
	if !strings.Contains(script, `CONF_DIR="/opt/singbox/conf"`) {
		t.Fatalf("生成的脚本未使用配置的 CONF_DIR")
	}
	if strings.Contains(script, `CONF_DIR="/etc/sing-box"`) {
		t.Fatalf("生成的脚本仍写死默认 CONF_DIR")
	}
}

// xray/sing-box 全套安装脚本同样要跟随配置：证书目录与 acme.sh 目录都在脚本内部
// 的 heredoc/JSON 里被引用（22 处），必须整体替换而不是只改头部的变量定义。
func TestBuildXrayInstallScriptUsesConfiguredPaths(t *testing.T) {
	t.Setenv(envConfDir, "/opt/singbox/conf")
	t.Setenv(envCertDir, "/opt/singbox/conf/certs")
	t.Setenv(envAcmeHome, "/opt/acme")

	script, err := BuildXrayInstallScript(XrayScriptConfig{
		PanelBaseURL: "https://panel.example.com",
		InstallID:    "test-xray",
		Token:        "test-token",
		Domain:       "node.example.com",
		Email:        "admin@example.com",
		Protocols:    []XrayProtocol{{Key: "vless-ws", Port: 443, Domain: "node.example.com", Password: "uuid-test", WS: "/ws-test"}},
		GeneratedAt:  time.Now(),
	})
	if err != nil {
		t.Fatalf("生成脚本失败: %v", err)
	}
	for _, want := range []string{`CONF_DIR="/opt/singbox/conf"`, "/opt/singbox/conf/certs", "/opt/acme"} {
		if !strings.Contains(script, want) {
			t.Fatalf("生成的脚本里缺少 %q", want)
		}
	}
	for _, bad := range []string{"/etc/sing-box", "/root/.acme.sh"} {
		if strings.Contains(script, bad) {
			t.Fatalf("生成的脚本仍写死默认路径 %q", bad)
		}
	}
}

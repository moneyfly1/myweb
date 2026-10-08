package selfhost

import (
	"os"
	"strings"
)

// 自建节点部署脚本里会用到的**目标机器**路径。
//
// 这些路径属于「下发到用户 VPS 上的部署约定」（不是面板本机的目录）：
// 默认值与历史行为完全一致，保证既有部署不受影响；如果某台机器上 sing-box 装在别处
// （例如 /opt/sing-box），可以在面板的 .env 里覆盖，不必改代码重新编译。
//
// 变量名统一加 SELFHOST_ 前缀，避免与面板自身的路径配置混淆。
const (
	envBinDir   = "SELFHOST_BIN_DIR"   // 二进制安装目录（sing-box / 心跳脚本）
	envConfDir  = "SELFHOST_CONF_DIR"  // sing-box 配置目录
	envCertDir  = "SELFHOST_CERT_DIR"  // 证书目录
	envAcmeHome = "SELFHOST_ACME_HOME" // acme.sh 安装目录

	// 历史默认值（不要改动，否则老节点重装时会换目录）
	defaultBinDir   = "/usr/local/bin"
	defaultConfDir  = "/etc/sing-box"
	defaultCertDir  = "/etc/sing-box/cert"
	defaultAcmeHome = "/root/.acme.sh"
)

func envOrDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return strings.TrimSuffix(v, "/")
	}
	return def
}

// BinDir 目标机器上的二进制目录
func BinDir() string { return envOrDefault(envBinDir, defaultBinDir) }

// ConfDir 目标机器上的 sing-box 配置目录
func ConfDir() string { return envOrDefault(envConfDir, defaultConfDir) }

// CertDir 目标机器上的证书目录
func CertDir() string { return envOrDefault(envCertDir, defaultCertDir) }

// AcmeHome 目标机器上的 acme.sh 目录
func AcmeHome() string { return envOrDefault(envAcmeHome, defaultAcmeHome) }

// SingBoxConfigPath 目标机器上的 sing-box 配置文件路径
func SingBoxConfigPath() string { return ConfDir() + "/config.json" }

// HeartbeatScriptPath 目标机器上的心跳脚本路径
func HeartbeatScriptPath() string { return BinDir() + "/cboard-agent-heartbeat.sh" }

// applyTargetPaths 把渲染好的脚本文本里的默认路径替换成当前配置的路径。
//
// 采用「渲染后替换」而不是改 fmt 模板的参数顺序：脚本模板非常长、占位符众多，
// 调整参数顺序极易出错；替换只针对几条固定的变量定义行与引用，风险可控且易于测试。
// 默认配置下这些替换是恒等操作（旧行为不变）。
func applyTargetPaths(script string) string {
	// 按「默认路径 -> 配置路径」替换，长路径优先：
	// 这样 /etc/sing-box/cert 先被替换掉，再做 /etc/sing-box 的替换时就不会二次命中，
	// 避免出现 /opt/sing-box/cert 被拼成 /opt/sing-box/opt/sing-box/cert 之类的错。
	pairs := [][2]string{
		{defaultCertDir, CertDir()},
		{defaultConfDir, ConfDir()},
		{defaultBinDir, BinDir()},
		{defaultAcmeHome, AcmeHome()},
	}
	// 长 -> 短
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			if len(pairs[j][0]) > len(pairs[i][0]) {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}
	for _, r := range pairs {
		if r[0] == r[1] || r[0] == "" {
			continue
		}
		script = strings.ReplaceAll(script, r[0], r[1])
	}
	return script
}

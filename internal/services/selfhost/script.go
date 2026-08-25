package selfhost

import (
	"fmt"
	"strings"
	"time"
)

// BuildInstallScript 生成 sing-box 一键安装脚本内容。
// 脚本职责：检测系统 → 下载 sing-box（多镜像源 fallback）→ 生成协议配置 →
// 启动服务 → 探测公网 IP → 构造节点链接并回传面板 → 后台心跳。
// 所有变量由面板动态注入，避免脚本里写死任何凭据。
func BuildInstallScript(cfg ScriptConfig) (string, error) {
	if cfg.PanelBaseURL == "" {
		return "", fmt.Errorf("面板地址不能为空")
	}
	if cfg.InstallID == "" || cfg.Token == "" {
		return "", fmt.Errorf("安装标识与令牌不能为空")
	}
	if cfg.Protocol == "" {
		return "", fmt.Errorf("协议不能为空")
	}

	// 回传与心跳地址
	reportURL := strings.TrimRight(cfg.PanelBaseURL, "/") + "/api/v1/agent/report"
	heartbeatURL := strings.TrimRight(cfg.PanelBaseURL, "/") + "/api/v1/agent/heartbeat"
	downloadPrefixes := strings.Join(cfg.MirrorURLs, " ")

	// 协议内嵌配置（占位符将在脚本中替换为运行时生成的值）
	protoConfig := protocolScriptBlock(cfg.Protocol)

	script := fmt.Sprintf(`#!/bin/bash
# ============================================================
#  CBoard 一键自建节点安装脚本（自动生成，请勿手动编辑）
#  协议: %s
#  安装标识: %s
#  生成时间: %s
# ============================================================
set -e

INSTALL_ID="%s"
TOKEN="%s"
PROTOCOL="%s"
REPORT_URL="%s"
HEARTBEAT_URL="%s"
PANEL_BASE="%s"
NAME="%s"
DOWNLOAD_PREFIXES="%s"

RED='\033[1;31m'; GREEN='\033[1;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()  { printf "${GREEN}[INFO]${NC} %%s\n" "$*"; }
warn()  { printf "${YELLOW}[WARN]${NC} %%s\n" "$*"; }
err()   { printf "${RED}[ERR]${NC} %%s\n" "$*" >&2; }

# ---------- 前置检查 ----------
if [ "$(id -u)" != "0" ]; then
    err "请使用 root 权限运行: sudo bash <(curl -fsSL ...)"
    exit 1
fi
command -v curl >/dev/null 2>&1 || { err "未检测到 curl，请先安装: apt-get install -y curl / apk add curl"; exit 1; }
command -v tar >/dev/null 2>&1 || { err "未检测到 tar"; exit 1; }

# ---------- 系统/架构检测 ----------
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        case "$ID$ID_LIKE" in
            *alpine*) OS="alpine" ;;
            *debian*|*ubuntu*) OS="debian" ;;
            *centos*|*rhel*|*fedora*|*rocky*|*almalinux*) OS="centos" ;;
            *) OS="unknown" ;;
        esac
    else
        OS="unknown"
    fi
}
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        armv7l|armv7) ARCH="armv7" ;;
        *) err "不支持的架构: $(uname -m)"; exit 1 ;;
    esac
}
detect_init() {
    if command -v systemctl >/dev/null 2>&1; then INIT="systemd"
    elif command -v rc-service >/dev/null 2>&1; then INIT="openrc"
    else INIT="none"; fi
}
detect_os; detect_arch; detect_init
info "系统: $OS | 架构: $ARCH | init: $INIT"

# ---------- sing-box 下载（多镜像源 fallback） ----------
SINGBOX_DIR="/usr/local/bin"
SINGBOX_BIN="${SINGBOX_DIR}/sing-box"
SINGBOX_VER="1.11.3"

# 镜像前缀列表（面板下发，运行时按架构拼装完整 URL）
MIRROR_PREFIXES="${DOWNLOAD_PREFIXES}"

download_singbox() {
    local prefixes="$1" tmp="$2" ver="v${SINGBOX_VER}" fname="sing-box-${SINGBOX_VER}-linux-${ARCH}.tar.gz"
    for p in $prefixes; do
        local u="${p}${ver}/${fname}"
        info "尝试下载 sing-box: $u"
        if curl -fsSL --connect-timeout 15 -o "$tmp" "$u"; then
            return 0
        fi
        warn "下载失败，尝试下一个镜像源: $u"
    done
    return 1
}

install_singbox() {
    local tmpdir="/tmp/singbox-install-$$"
    mkdir -p "$tmpdir"
    local tmpfile="$tmpdir/singbox.tar.gz"
    if ! download_singbox "$MIRROR_PREFIXES" "$tmpfile"; then
        err "sing-box 下载失败，请检查网络或手动安装后重试"
        rm -rf "$tmpdir"
        exit 1
    fi
    tar -xzf "$tmpfile" -C "$tmpdir"
    local bin="$(find "$tmpdir" -type f -name 'sing-box' | head -1)"
    if [ -z "$bin" ]; then
        err "sing-box 解压失败（未找到二进制）"
        rm -rf "$tmpdir"
        exit 1
    fi
    chmod +x "$bin"
    cp "$bin" "$SINGBOX_BIN"
    rm -rf "$tmpdir"
    info "sing-box 安装完成: $($SINGBOX_BIN version 2>/dev/null | head -1 || echo "OK")"
}

if [ ! -x "$SINGBOX_BIN" ]; then
    install_singbox
else
    info "检测到已安装 sing-box，跳过下载"
fi

# ---------- 生成随机凭据 ----------
gen_uuid() {
    if command -v uuidgen >/dev/null 2>&1; then uuidgen
    else cat /proc/sys/kernel/random/uuid 2>/dev/null || echo "$(date +%%s%%N | md5sum | sed 's/..\\(.\\)/\\1/g' | fold -w4 | paste -sd'-' | sed 's/\\(.\\{8\\}\\)-\\(.\\{4\\}\\)-\\(.\\{4\\}\\)-\\(.\\{4\\}\\)-\\(.\\{12\\}\\)/\\1-\\2-3\\3-9\\4-\\5/')"
    fi
}
gen_password() { openssl rand -base64 16 2>/dev/null | tr -d '=+/' | head -c 24 || head -c 24 /dev/urandom | base64 | tr -d '=+/' | head -c 24; }
gen_ws_path() { echo "cboard$(head -c 8 /dev/urandom | base64 | tr -d '=+/' | tr '/' '_')"; }

UUID="$(gen_uuid)"
PASSWORD="$(gen_password)"
WS_PATH="$(gen_ws_path)"
LISTEN_PORT="${LISTEN_PORT:-443}"
if [ "$LISTEN_PORT" = "443" ] && command -v ss >/dev/null 2>&1 && ss -tln | grep -q ':443 '; then
    warn "443 端口被占用，改用 8443"
    LISTEN_PORT="8443"
fi

# ---------- 生成 sing-box 配置 ----------
CONF_DIR="/etc/sing-box"
mkdir -p "$CONF_DIR"

# 二次部署保护：先备份旧配置（重装失败时可回退）
if [ -f "$CONF_DIR/config.json" ]; then
    OLD_BAK="$CONF_DIR/config.json.bak.$(date +%%Y%%m%%d_%%H%%M%%S)"
    cp "$CONF_DIR/config.json" "$OLD_BAK"
    info "已备份旧配置: $OLD_BAK"
fi

%s

# ---------- 探测公网 IP ----------
detect_public_ip() {
    for api in "https://api.ipify.org" "https://ifconfig.me/ip" "https://ip.sb" "https://api-ipv4.ip.sb/ip"; do
        IP="$(curl -fsSL --connect-timeout 8 "$api" 2>/dev/null | tr -d '[:space:]' | grep -E '^[0-9.]+$' | head -1)"
        [ -n "$IP" ] && return 0
    done
    return 1
}
IP=""
detect_public_ip || warn "无法自动探测公网 IP，回传时将使用空地址"

# ---------- 写入节点链接（由协议块填充） ----------
%s

# ---------- 防火墙放行端口 ----------
open_firewall() {
    local ports="${1:-443}"
    if command -v ufw >/dev/null 2>&1; then
        for p in $ports; do ufw allow "$p/tcp" >/dev/null 2>&1 || true; done
        ufw allow "${LISTEN_PORT:-443}/tcp" >/dev/null 2>&1 || true
        echo "ufw 已放行: $ports"
    elif command -v firewall-cmd >/dev/null 2>&1; then
        for p in $ports; do firewall-cmd --permanent --add-port="$p/tcp" >/dev/null 2>&1 || true; done
        firewall-cmd --reload >/dev/null 2>&1 || true
        echo "firewalld 已放行: $ports"
    else
        echo "未检测到 ufw/firewalld，跳过防火墙配置"
    fi
}

# ---------- 启动服务 ----------
start_service() {
    local name="sing-box"
    case "$INIT" in
        systemd)
            cat > /etc/systemd/system/sing-box.service <<EOF
[Unit]
Description=Sing-box Service
After=network.target

[Service]
Type=simple
ExecStart=${SINGBOX_BIN} run -c ${CONF_DIR}/config.json
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF
            systemctl daemon-reload
            systemctl enable sing-box >/dev/null 2>&1 || true
            systemctl restart sing-box
            ;;
        openrc)
            cat > /etc/init.d/sing-box <<EOF
#!/sbin/openrc-run
command="${SINGBOX_BIN}"
command_args="run -c ${CONF_DIR}/config.json"
command_background="yes"
pidfile="/run/sing-box.pid"
EOF
            chmod +x /etc/init.d/sing-box
            rc-update add sing-box default >/dev/null 2>&1 || true
            rc-service sing-box restart
            ;;
        none)
            nohup "$SINGBOX_BIN" run -c "$CONF_DIR/config.json" >/var/log/sing-box.log 2>&1 &
            ;;
    esac
}

# ---------- 启动心跳（后台守护，含流量上报） ----------
HEARTBEAT_INTERVAL=30
start_heartbeat() {
    local script="/usr/local/bin/cboard-agent-heartbeat.sh"
    # sing-box 提供 external-controller 用于查询实时流量（上行/下行字节累计值）
    cat > "$script" <<EOF
#!/bin/bash
# CBoard 自建节点心跳（含流量上报）
API_PORT="\${API_PORT:-127.0.0.1:19090}"
LAST_UP=0
LAST_DOWN=0
while true; do
    # sing-box clash_api /connections 提供累计流量（downloadTotal/uploadTotal），
    # /traffic 为实时速率（服务端模式常为 0），故取累计值做增量上报。
    TRAFFIC_JSON="\$(curl -fsSL --connect-timeout 5 "http://\${API_PORT}/connections" 2>/dev/null || echo '')"
    UP=""
    DOWN=""
    if [ -n "\$TRAFFIC_JSON" ]; then
        UP="\$(echo "\$TRAFFIC_JSON" | sed -n 's/.*"uploadTotal":\([0-9]*\).*/\1/p' | head -1)"
        DOWN="\$(echo "\$TRAFFIC_JSON" | sed -n 's/.*"downloadTotal":\([0-9]*\).*/\1/p' | head -1)"
    fi
    # 计算增量（首次无基线则上报当前值）
    UP_DELTA=""
    DOWN_DELTA=""
    if [ -n "\$UP" ] && [ "\$UP" -ge "\$LAST_UP" ] 2>/dev/null; then
        UP_DELTA=\$((UP - LAST_UP))
        LAST_UP=\$UP
    elif [ -n "\$UP" ]; then
        UP_DELTA=\$UP
        LAST_UP=\$UP
    fi
    if [ -n "\$DOWN" ] && [ "\$DOWN" -ge "\$LAST_DOWN" ] 2>/dev/null; then
        DOWN_DELTA=\$((DOWN - LAST_DOWN))
        LAST_DOWN=\$DOWN
    elif [ -n "\$DOWN" ]; then
        DOWN_DELTA=\$DOWN
        LAST_DOWN=\$DOWN
    fi
    PAYLOAD="{\"install_id\":\"${INSTALL_ID}\",\"token\":\"${TOKEN}\""
    if [ -n "\$UP_DELTA" ]; then PAYLOAD="\${PAYLOAD},\"traffic_up\":\$UP_DELTA"; fi
    if [ -n "\$DOWN_DELTA" ]; then PAYLOAD="\${PAYLOAD},\"traffic_down\":\$DOWN_DELTA"; fi
    PAYLOAD="\${PAYLOAD}}"
    # 单行 curl（不使用反斜杠续行，避免在 bash <(curl) 等嵌套执行环境中续行失效）
    curl -fsSL --connect-timeout 10 -X POST "${HEARTBEAT_URL}" -H "Content-Type: application/json" -d "\$PAYLOAD" >/dev/null 2>&1 || true
    sleep ${HEARTBEAT_INTERVAL}
done
EOF
    chmod +x "$script"
    case "$INIT" in
        systemd)
            cat > /etc/systemd/system/cboard-heartbeat.service <<EOF
[Unit]
Description=CBoard Node Heartbeat
After=network.target

[Service]
Type=simple
ExecStart=${script}
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
            systemctl daemon-reload
            systemctl enable cboard-heartbeat >/dev/null 2>&1 || true
            systemctl restart cboard-heartbeat
            ;;
        openrc)
            nohup "$script" >/dev/null 2>&1 &
            ;;
        none)
            nohup "$script" >/dev/null 2>&1 &
            ;;
    esac
}

open_firewall "$LISTEN_PORT"
start_service
info "sing-box 服务已启动"
start_heartbeat
info "心跳守护已启动"

# ---------- 回传节点链接到面板 ----------
if [ -z "$IP" ]; then
    warn "未探测到公网 IP，跳过回传（可在面板手动编辑节点）"
    exit 0
fi
PAYLOAD="$(python3 -c 'import json,sys;print(json.dumps({"install_id":sys.argv[1],"token":sys.argv[2],"link":sys.argv[3]}))' "$INSTALL_ID" "$TOKEN" "$LINK" 2>/dev/null || echo '')"
if [ -z "$PAYLOAD" ]; then
    PAYLOAD="{\"install_id\":\"$INSTALL_ID\",\"token\":\"$TOKEN\",\"link\":\"$LINK\"}"
fi
info "正在回传节点链接到面板..."
# 单行 curl（不使用反斜杠续行，避免在 bash <(curl) 等嵌套执行环境中续行失效）
if curl -fsSL --connect-timeout 15 -X POST "$REPORT_URL" -H "Content-Type: application/json" -d "$PAYLOAD"; then
    info "回传成功！节点已添加，可到面板查看"
else
    warn "回传失败，可手动复制链接在面板添加: $LINK"
    exit 0
fi

echo ""
echo "========================================================"
echo "  节点安装完成！"
echo "  协议: $PROTOCOL | 地址: $IP:$LISTEN_PORT"
echo "  节点链接已自动回传面板"
echo "========================================================"
`,
		cfg.Protocol,
		cfg.InstallID,
		cfg.GeneratedAt.Format("2006-01-02 15:04:05"),
		cfg.InstallID,
		cfg.Token,
		cfg.Protocol,
		reportURL,
		heartbeatURL,
		strings.TrimRight(cfg.PanelBaseURL, "/"),
		shellQuote(cfg.NodeName),
		downloadPrefixes,
		protoConfig,
		linkBuildBlock(cfg.Protocol),
	)
	return script, nil
}

// ScriptConfig 安装脚本的动态注入参数。
type ScriptConfig struct {
	PanelBaseURL string // 面板访问地址（含协议与端口，如 https://panel.example.com）
	InstallID    string // 安装标识
	Token        string // 安装令牌
	Protocol     string // 协议
	NodeName     string // 节点名称
	MirrorURLs   []string
	GeneratedAt  time.Time
}

func shellQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "`", "\\`")
	s = strings.ReplaceAll(s, "$", "\\$")
	return s
}

// protocolScriptBlock 返回写入 sing-box config.json 的协议配置片段。
func protocolScriptBlock(protocol string) string {
	switch protocol {
	case "vless-ws":
		return `cat > "$CONF_DIR/config.json" <<EOF
{
  "log": { "level": "info" },
  "experimental": {
    "clash_api": {
      "external_controller": "127.0.0.1:19090"
    }
  },
  "inbounds": [
    {
      "type": "vless",
      "tag": "vless-in",
      "listen": "0.0.0.0",
      "listen_port": ${LISTEN_PORT},
      "users": [ { "uuid": "${UUID}" } ],
      "transport": { "type": "ws", "path": "/${WS_PATH}" }
    }
  ],
  "outbounds": [ { "type": "direct" } ]
}
EOF`
	case "vmess-ws":
		return `cat > "$CONF_DIR/config.json" <<EOF
{
  "log": { "level": "info" },
  "experimental": {
    "clash_api": {
      "external_controller": "127.0.0.1:19090"
    }
  },
  "inbounds": [
    {
      "type": "vmess",
      "tag": "vmess-in",
      "listen": "0.0.0.0",
      "listen_port": ${LISTEN_PORT},
      "users": [ { "uuid": "${UUID}" } ],
      "transport": { "type": "ws", "path": "/${WS_PATH}" }
    }
  ],
  "outbounds": [ { "type": "direct" } ]
}
EOF`
	case "vless-reality":
		// reality 需要 x25519 密钥对与短 ID，运行时用 sing-box 生成
		return `# 一次生成 keypair 同时解析两把密钥（多次调用生成不匹配密钥对，Reality 不可用）
REALITY_KEYPAIR="$($SINGBOX_BIN generate reality-keypair || true)"
REALITY_PRIVATE_KEY="$(echo "$REALITY_KEYPAIR" | grep -oP 'PrivateKey:\s*\K.*' || true)"
REALITY_PUBLIC_KEY="$(echo "$REALITY_KEYPAIR" | grep -oP 'PublicKey:\s*\K.*' || true)"
if [ -z "$REALITY_PRIVATE_KEY" ] || [ -z "$REALITY_PUBLIC_KEY" ]; then
    err "sing-box generate reality-keypair 失败，请检查 sing-box 版本"
    exit 1
fi
REALITY_SHORT_ID="$(head -c 4 /dev/urandom | xxd -p || echo 'abcdef')"
cat > "$CONF_DIR/config.json" <<EOF
{
  "log": { "level": "info" },
  "experimental": {
    "clash_api": {
      "external_controller": "127.0.0.1:19090"
    }
  },
  "inbounds": [
    {
      "type": "vless",
      "tag": "vless-in",
      "listen": "0.0.0.0",
      "listen_port": ${LISTEN_PORT},
      "users": [ { "uuid": "${UUID}", "flow": "xtls-rprx-vision" } ],
      "tls": {
        "enabled": true,
        "server_name": "www.microsoft.com",
        "reality": {
          "enabled": true,
          "handshake": { "server": "www.microsoft.com", "server_port": 443 },
          "private_key": "${REALITY_PRIVATE_KEY}",
          "short_id": [ "${REALITY_SHORT_ID}" ]
        }
      }
    }
  ],
  "outbounds": [ { "type": "direct" } ]
}
EOF`
	case "trojan-ws":
		return `cat > "$CONF_DIR/config.json" <<EOF
{
  "log": { "level": "info" },
  "experimental": {
    "clash_api": {
      "external_controller": "127.0.0.1:19090"
    }
  },
  "inbounds": [
    {
      "type": "trojan",
      "tag": "trojan-in",
      "listen": "0.0.0.0",
      "listen_port": ${LISTEN_PORT},
      "users": [ { "password": "${PASSWORD}" } ],
      "transport": { "type": "ws", "path": "/${WS_PATH}" }
    }
  ],
  "outbounds": [ { "type": "direct" } ]
}
EOF`
	case "ss":
		return `cat > "$CONF_DIR/config.json" <<EOF
{
  "log": { "level": "info" },
  "experimental": {
    "clash_api": {
      "external_controller": "127.0.0.1:19090"
    }
  },
  "inbounds": [
    {
      "type": "shadowsocks",
      "tag": "ss-in",
      "listen": "0.0.0.0",
      "listen_port": ${LISTEN_PORT},
      "method": "aes-128-gcm",
      "password": "${PASSWORD}"
    }
  ],
  "outbounds": [ { "type": "direct" } ]
}
EOF`
	default:
		return ""
	}
}

// linkBuildBlock 返回构造节点链接（LINK 变量）的脚本片段。
func linkBuildBlock(protocol string) string {
	switch protocol {
	case "vless-ws":
		return `LINK="vless://${UUID}@${IP}:${LISTEN_PORT}?type=ws&path=%2F${WS_PATH}&security=none#${NAME}"
`
	case "vmess-ws":
		return `VMESS_JSON="{\"v\":2,\"ps\":\"${NAME}\",\"add\":\"${IP}\",\"port\":\"${LISTEN_PORT}\",\"id\":\"${UUID}\",\"aid\":\"0\",\"net\":\"ws\",\"type\":\"none\",\"host\":\"\",\"path\":\"/${WS_PATH}\",\"tls\":\"\"}"
LINK="vmess://$(echo -n "$VMESS_JSON" | base64 -w0 | tr -d '=')"
`
	case "vless-reality":
		return `LINK="vless://${UUID}@${IP}:${LISTEN_PORT}?encryption=none&flow=xtls-rprx-vision&security=reality&sni=www.microsoft.com&fp=chrome&pbk=${REALITY_PUBLIC_KEY}&sid=${REALITY_SHORT_ID}&type=tcp#${NAME}"
`
	case "trojan-ws":
		return `LINK="trojan://${PASSWORD}@${IP}:${LISTEN_PORT}?type=ws&path=%2F${WS_PATH}&security=none#${NAME}"
`
	case "ss":
		return `SS_B64="$(echo -n "aes-128-gcm:${PASSWORD}" | base64 -w0 | tr -d '=')"
LINK="ss://${SS_B64}@${IP}:${LISTEN_PORT}#${NAME}"
`
	default:
		return `LINK=""`
	}
}

// DefaultMirrorURLs 返回默认 sing-box 下载镜像列表（多架构）。
func DefaultMirrorURLs() []string {
	// 下载地址在脚本内按运行时架构拼装，这里返回 URL 模板前缀列表
	return []string{
		"https://github.com/SagerNet/sing-box/releases/download/",
		"https://gh-proxy.com/https://github.com/SagerNet/sing-box/releases/download/",
		"https://mirror.ghproxy.com/https://github.com/SagerNet/sing-box/releases/download/",
	}
}

// buildMirrorURLs 返回 sing-box 下载镜像列表。
func buildMirrorURLs(version, arch string) []string {
	return []string{
		fmt.Sprintf("https://github.com/SagerNet/sing-box/releases/download/v%s/sing-box-%s-linux-%s.tar.gz", version, version, arch),
		fmt.Sprintf("https://gh-proxy.com/https://github.com/SagerNet/sing-box/releases/download/v%s/sing-box-%s-linux-%s.tar.gz", version, version, arch),
		fmt.Sprintf("https://mirror.ghproxy.com/https://github.com/SagerNet/sing-box/releases/download/v%s/sing-box-%s-linux-%s.tar.gz", version, version, arch),
	}
}

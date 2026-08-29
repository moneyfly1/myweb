package selfhost

import (
	"fmt"
	"strings"
	"time"
)

// XrayProtocol 域名多协议部署的单个协议配置。
type XrayProtocol struct {
	Key        string // 协议标识（见 SupportedXrayProtocols）
	Port       int    // 监听端口
	Domain     string // TLS 域名
	Password   string // UUID / 密码（面板生成后注入）
	ServerIP   string // 服务器公网 IP（回传链接用）
	WS         string // ws path（ws 协议）
	GRPCPath   string // grpc 的 serviceName（grpc 协议）
	RealitySNI string // reality 的 sni（如 www.microsoft.com）
	HyUp       int    // hysteria2 上行带宽（Mbps）
	HyDown     int    // hysteria2 下行带宽（Mbps）
}

// SupportedXrayProtocols 支持的全部协议（对照 v2ray-agent 八合一）。
var SupportedXrayProtocols = []string{
	"vless-ws",            // VLESS + WS + TLS
	"vmess-ws",            // VMess + WS + TLS
	"vless-reality",       // VLESS + Reality + Vision
	"vless-reality-grpc",  // VLESS + Reality + gRPC
	"vless-reality-xhttp", // VLESS + Reality + XHTTP
	"vless-grpc-tls",      // VLESS + gRPC + TLS
	"vless-tcp-tls",       // VLESS + TCP + TLS + Vision
	"trojan-tcp-tls",      // Trojan + TCP + TLS
	"trojan-ws",           // Trojan + WS + TLS
	"trojan-grpc-tls",     // Trojan + gRPC + TLS
	"hysteria2",           // Hysteria2
	"tuic",                // TUIC
	"ss",                  // Shadowsocks
	"anytls",              // AnyTLS + TLS
	"vmess-httpupgrade",   // VMess + HTTPUpgrade + TLS
}

// XrayProtocolNames 供批量回传 handler 记录协议列表。
func XrayProtocolNames(protocols []XrayProtocol) []string {
	names := make([]string, 0, len(protocols))
	for _, p := range protocols {
		names = append(names, p.Key)
	}
	return names
}

// XrayScriptConfig Xray 部署脚本配置。
type XrayScriptConfig struct {
	PanelBaseURL string
	InstallID    string
	Token        string
	Domain       string // 证书域名
	Email        string // acme 邮箱
	Protocols    []XrayProtocol
	MirrorURLs   []string
	GeneratedAt  time.Time
}

// DefaultXrayProtocols 返回默认多协议组合（端口分配，对照 v2ray-agent 八合一）。
func DefaultXrayProtocols(domain string) []XrayProtocol {
	return []XrayProtocol{
		{Key: "vless-ws", Port: 443, Domain: domain},
		{Key: "vless-reality", Port: 8443},
		{Key: "vless-grpc-tls", Port: 2053, Domain: domain},
		{Key: "trojan-tcp-tls", Port: 2083, Domain: domain},
		{Key: "ss", Port: 8388},
	}
}

// BuildXrayInstallScript 生成 Xray 多协议一键安装脚本。
// 流程：检测系统 → 下载 Xray → acme 申请证书（有域名时）→ 生成多协议配置 →
// 启动服务 → 探测公网IP → 构造所有协议链接 → 批量回传 → 心跳。
func BuildXrayInstallScript(cfg XrayScriptConfig) (string, error) {
	if cfg.PanelBaseURL == "" || cfg.InstallID == "" || cfg.Token == "" {
		return "", fmt.Errorf("面板地址/安装标识/令牌不能为空")
	}
	if len(cfg.Protocols) == 0 {
		return "", fmt.Errorf("至少需要一个协议")
	}

	reportURL := strings.TrimRight(cfg.PanelBaseURL, "/") + "/api/v1/agent/report-batch"
	heartbeatURL := strings.TrimRight(cfg.PanelBaseURL, "/") + "/api/v1/agent/heartbeat"
	downloadPrefixes := strings.Join(cfg.MirrorURLs, " ")

	script := fmt.Sprintf(`#!/bin/bash
# ============================================================
#  CBoard 域名多协议自建节点安装脚本（自动生成）
#  域名: %s | 协议数: %d
# ============================================================
set -e

INSTALL_ID="%s"
TOKEN="%s"
REPORT_URL="%s"
HEARTBEAT_URL="%s"
DOWNLOAD_PREFIXES="%s"
DOMAIN="%s"
EMAIL="%s"

RED='\033[1;31m'; GREEN='\033[1;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()  { printf "${GREEN}[INFO]${NC} %%s\n" "$*"; }
warn()  { printf "${YELLOW}[WARN]${NC} %%s\n" "$*"; }
err()   { printf "${RED}[ERR]${NC} %%s\n" "$*" >&2; }

# ---------- 前置检查 ----------
if [ "$(id -u)" != "0" ]; then err "请使用 root 权限运行"; exit 1; fi
command -v curl >/dev/null 2>&1 || { err "未安装 curl"; exit 1; }
command -v tar >/dev/null 2>&1 || { err "未安装 tar"; exit 1; }

# ---------- 系统/架构检测 ----------
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        case "$ID$ID_LIKE" in
            *alpine*) OS="alpine" ;;
            *debian*|*ubuntu*) OS="debian" ;;
            *centos*|*rhel*|*fedora*) OS="centos" ;;
            *) OS="unknown" ;;
        esac
    else OS="unknown"; fi
}
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        armv7l) ARCH="armv7" ;;
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

# ---------- 下载 sing-box ----------
SB_DIR="/usr/local/bin"
SB_BIN="${SB_DIR}/sing-box"
SB_VER="1.12.2"
install_singbox() {
    local tmpdir="/tmp/sb-install-$$"
    mkdir -p "$tmpdir"
    local tmpfile="$tmpdir/sb.tar.gz"
    for p in $DOWNLOAD_PREFIXES; do
        local u="${p}v${SB_VER}/sing-box-${SB_VER}-linux-${ARCH}.tar.gz"
        info "下载 sing-box: $u"
        if curl -fsSL --connect-timeout 15 -o "$tmpfile" "$u"; then break; fi
        warn "下载失败: $u"
    done
    tar -xzf "$tmpfile" -C "$tmpdir"
    local bin="$(find "$tmpdir" -type f -name 'sing-box' | head -1)"
    [ -n "$bin" ] || { err "sing-box 解压失败"; exit 1; }
    chmod +x "$bin" && cp "$bin" "$SB_BIN"
    rm -rf "$tmpdir"
    info "sing-box 安装完成: $($SB_BIN version 2>/dev/null | head -1)"
}
if [ ! -x "$SB_BIN" ]; then install_singbox; else info "sing-box 已存在，跳过下载"; fi

# ---------- acme 申请证书（有域名时） ----------
install_acme() {
    if [ ! -f /root/.acme.sh/acme.sh ]; then
        info "安装 acme.sh..."
        curl -fsSL https://get.acme.sh | sh -s email="${EMAIL}" >/dev/null 2>&1 || \
        curl -fsSL https://raw.githubusercontent.com/acmesh-official/acme.sh/master/acme.sh | sh -s email="${EMAIL}" >/dev/null 2>&1 || \
        warn "acme.sh 安装失败（证书将由客户端信任跳过）"
    fi
}
apply_cert() {
    if [ -z "$DOMAIN" ]; then info "无域名，跳过证书"; return; fi
    install_acme
    if [ -f "/root/.acme.sh/${DOMAIN}_ecc/fullchain.cer" ]; then
        info "证书已存在: ${DOMAIN}"
        CERT_DIR="/etc/sing-box/cert"
        mkdir -p "$CERT_DIR"
        cp "/root/.acme.sh/${DOMAIN}_ecc/fullchain.cer" "$CERT_DIR/fullchain.pem"
        cp "/root/.acme.sh/${DOMAIN}_ecc/${DOMAIN}.key" "$CERT_DIR/privkey.pem"
        return
    fi
    info "申请证书: ${DOMAIN}..."
    /root/.acme.sh/acme.sh --issue -d "${DOMAIN}" --standalone --keylength ec-256 --force 2>/dev/null || \
    warn "证书申请失败（可能端口被占或 DNS 未解析），TLS 协议可能不可用"
    if [ -f "/root/.acme.sh/${DOMAIN}_ecc/fullchain.cer" ]; then
        CERT_DIR="/etc/sing-box/cert"
        mkdir -p "$CERT_DIR"
        cp "/root/.acme.sh/${DOMAIN}_ecc/fullchain.cer" "$CERT_DIR/fullchain.pem"
        cp "/root/.acme.sh/${DOMAIN}_ecc/${DOMAIN}.key" "$CERT_DIR/privkey.pem"
        # 安装续期 hook：acme 更新证书后自动复制到 sing-box 目录并重载服务
        cat > "/root/.acme.sh/${DOMAIN}_ecc/renew_hook.sh" <<EOF
#!/bin/bash
CERT_DIR="/etc/sing-box/cert"
mkdir -p "\$CERT_DIR"
cp "/root/.acme.sh/${DOMAIN}_ecc/fullchain.cer" "\$CERT_DIR/fullchain.pem"
cp "/root/.acme.sh/${DOMAIN}_ecc/${DOMAIN}.key" "\$CERT_DIR/privkey.pem"
systemctl restart sing-box 2>/dev/null || true
EOF
        chmod +x "/root/.acme.sh/${DOMAIN}_ecc/renew_hook.sh"
        /root/.acme.sh/acme.sh --install-cert -d "${DOMAIN}" --ecc             --reloadcmd "bash /root/.acme.sh/${DOMAIN}_ecc/renew_hook.sh" >/dev/null 2>&1 || true
        info "证书已申请（含自动续期）"
    fi
}
apply_cert

# ---------- 生成随机凭据 ----------
gen_uuid() {
    if command -v uuidgen >/dev/null 2>&1; then uuidgen
    else cat /proc/sys/kernel/random/uuid 2>/dev/null || echo "$(date +%%s%%N | md5sum | sed 's/..\\(.\\)/\\1/g' | fold -w4 | paste -sd'-' | sed 's/\\(.\\{8\\}\\)-\\(.\\{4\\}\\)-\\(.\\{4\\}\\)-\\(.\\{4\\}\\)-\\(.\\{12\\}\\)/\\1-\\2-3\\3-9\\4-\\5/')"
    fi
}
gen_password() { openssl rand -base64 16 2>/dev/null | tr -d '=+/' | head -c 24 || head -c 24 /dev/urandom | base64 | tr -d '=+/' | head -c 24; }
gen_ws_path() { echo "cboard$(head -c 8 /dev/urandom | base64 | tr -d '=+/' | tr '/' '_')"; }

# ---------- 生成凭据（每协议） ----------
declare -A UUIDS PASSWORDS WSPATHS
%s

# ---------- 生成 sing-box 多协议配置 ----------
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
    for api in "https://api.ipify.org" "https://ifconfig.me/ip" "https://ip.sb"; do
        IP="$(curl -fsSL --connect-timeout 8 "$api" 2>/dev/null | tr -d '[:space:]' | grep -E '^[0-9.]+$' | head -1)"
        [ -n "$IP" ] && return 0
    done
    return 1
}
IP=""
detect_public_ip || warn "无法探测公网 IP"

# ---------- 节点地址：有域名用域名（TLS 证书校验/用户可读），无域名回退公网 IP ----------
if [ -n "$DOMAIN" ]; then
    SERVER_ADDR="$DOMAIN"
    info "节点地址使用域名: $SERVER_ADDR"
elif [ -n "$IP" ]; then
    SERVER_ADDR="$IP"
else
    SERVER_ADDR="127.0.0.1"
    warn "既无域名也未探测到公网 IP，节点地址暂用 127.0.0.1（回传后可手动修正）"
fi

# ---------- 构造节点链接 ----------
%s

# ---------- 防火墙放行端口 ----------
open_firewall() {
    if command -v ufw >/dev/null 2>&1; then
        for p in $ALL_PORTS; do ufw allow "$p/tcp" >/dev/null 2>&1 || true; done
        ufw allow "${HY2_PORT}/udp" >/dev/null 2>&1 || true
        echo "ufw 已放行: $ALL_PORTS"
    elif command -v firewall-cmd >/dev/null 2>&1; then
        for p in $ALL_PORTS; do firewall-cmd --permanent --add-port="$p/tcp" >/dev/null 2>&1 || true; done
        firewall-cmd --reload >/dev/null 2>&1 || true
        echo "firewalld 已放行: $ALL_PORTS"
    else
        echo "未检测到 ufw/firewalld，跳过防火墙配置"
    fi
}

# ---------- 启动服务 ----------
start_service() {
    case "$INIT" in
        systemd)
            cat > /etc/systemd/system/sing-box.service <<EOF
[Unit]
Description=Sing-box Service
After=network.target

[Service]
Type=simple
ExecStart=${SB_BIN} run -c ${CONF_DIR}/config.json
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
command="${SB_BIN}"
command_args="run -c ${CONF_DIR}/config.json"
command_background="yes"
pidfile="/run/sing-box.pid"
EOF
            chmod +x /etc/init.d/sing-box
            rc-update add sing-box default >/dev/null 2>&1 || true
            rc-service sing-box restart
            ;;
        none)
            nohup "$SB_BIN" run -c "$CONF_DIR/config.json" >/var/log/sing-box.log 2>&1 &
            ;;
    esac
}

# ---------- 启动心跳（含流量上报） ----------
HEARTBEAT_INTERVAL=30
start_heartbeat() {
    local script="/usr/local/bin/cboard-agent-heartbeat.sh"
    cat > "$script" <<EOF
#!/bin/bash
# CBoard 域名多协议节点心跳（含流量上报）
API_PORT="\${API_PORT:-127.0.0.1:19090}"
LAST_UP=0
LAST_DOWN=0
while true; do
    TRAFFIC_JSON="\$(curl -fsSL --connect-timeout 5 "http://\${API_PORT}/connections" 2>/dev/null || echo '')"
    UP=""; DOWN=""
    if [ -n "\$TRAFFIC_JSON" ]; then
        UP="\$(echo "\$TRAFFIC_JSON" | sed -n 's/.*"uploadTotal":\([0-9]*\).*/\1/p' | head -1)"
        DOWN="\$(echo "\$TRAFFIC_JSON" | sed -n 's/.*"downloadTotal":\([0-9]*\).*/\1/p' | head -1)"
    fi
    UP_DELTA=""; DOWN_DELTA=""
    if [ -n "\$UP" ] && [ "\$UP" -ge "\$LAST_UP" ] 2>/dev/null; then
        UP_DELTA=\$((UP - LAST_UP)); LAST_UP=\$UP
    elif [ -n "\$UP" ]; then UP_DELTA=\$UP; LAST_UP=\$UP; fi
    if [ -n "\$DOWN" ] && [ "\$DOWN" -ge "\$LAST_DOWN" ] 2>/dev/null; then
        DOWN_DELTA=\$((DOWN - LAST_DOWN)); LAST_DOWN=\$DOWN
    elif [ -n "\$DOWN" ]; then DOWN_DELTA=\$DOWN; LAST_DOWN=\$DOWN; fi
    PAYLOAD="{\"install_id\":\"${INSTALL_ID}\",\"token\":\"${TOKEN}\""
    if [ -n "\$UP_DELTA" ]; then PAYLOAD="\${PAYLOAD},\"traffic_up\":\$UP_DELTA"; fi
    if [ -n "\$DOWN_DELTA" ]; then PAYLOAD="\${PAYLOAD},\"traffic_down\":\$DOWN_DELTA"; fi
    PAYLOAD="\${PAYLOAD}}"
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
        *) nohup "$script" >/dev/null 2>&1 & ;;
    esac
}

# 收集所有监听端口用于防火墙放行
ALL_PORTS="$(grep -o '"listen_port": [0-9]*' "$CONF_DIR/config.json" | grep -o '[0-9]*' | tr '\n' ' ')"
HY2_PORT="$(echo "$ALL_PORTS" | awk '{print $NF}')"
open_firewall
start_service
info "sing-box 服务已启动"
start_heartbeat
info "心跳守护已启动"

# ---------- 批量回传所有节点 ----------
if [ -z "$IP" ]; then warn "未探测到公网 IP，跳过回传"; exit 0; fi
BATCH_JSON="{\"install_id\":\"$INSTALL_ID\",\"token\":\"$TOKEN\",\"links\":["
FIRST=1
%s
BATCH_JSON="${BATCH_JSON%%,}"
BATCH_JSON="${BATCH_JSON}]}"
info "正在回传 0 个节点链接..."
if curl -fsSL --connect-timeout 15 -X POST "$REPORT_URL" -H "Content-Type: application/json" -d "$BATCH_JSON"; then
    info "回传成功！多协议节点已添加到面板"
else
    warn "回传失败，可手动添加"
fi

echo ""
echo "========================================================"
echo "  域名多协议节点安装完成！"
echo "  域名: ${DOMAIN:-无} | 公网IP: $IP"
echo "========================================================"
`,
		cfg.Domain,
		len(cfg.Protocols),
		cfg.InstallID,
		cfg.Token,
		reportURL,
		heartbeatURL,
		downloadPrefixes,
		cfg.Domain,
		cfg.Email,
		buildXrayCreds(cfg.Protocols),
		buildXrayConfig(cfg.Protocols),
		buildXrayLinks(cfg.Protocols),
		buildXrayBatchPayload(cfg.Protocols),
	)
	return script, nil
}

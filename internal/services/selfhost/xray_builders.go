package selfhost

import (
	"encoding/json"
	"fmt"
	"strings"
)

// buildXrayCreds 生成每个协议的随机凭据脚本（UUID/密码/ws path）。
// 输出形如:
//   UUID_vless_ws="xxxx"   PASS_ss="yyy"   WS_vless_ws="/cboardxxx"
func buildXrayCreds(protocols []XrayProtocol) string {
	var sb strings.Builder
	for _, p := range protocols {
		key := sanitizeKey(p.Key)
		sb.WriteString(fmt.Sprintf("UUID_%s=\"$(gen_uuid)\"\n", key))
		sb.WriteString(fmt.Sprintf("PASS_%s=\"$(gen_password)\"\n", key))
		sb.WriteString(fmt.Sprintf("WS_%s=\"$(gen_ws_path)\"\n", key))
	}
	// reality 需要密钥对
	for _, p := range protocols {
		if p.Key == "vless-reality" {
			sb.WriteString(`REALITY_KEYPAIR="$($SB_BIN generate reality-keypair || true)"` + "\n")
			sb.WriteString(`REALITY_PRIVATE_KEY="$(echo "$REALITY_KEYPAIR" | grep -oP 'PrivateKey:\s*\K.*' || true)"` + "\n")
			sb.WriteString(`REALITY_PUBLIC_KEY="$(echo "$REALITY_KEYPAIR" | grep -oP 'PublicKey:\s*\K.*' || true)"` + "\n")
			sb.WriteString(`REALITY_SHORT_ID="$(head -c 4 /dev/urandom | xxd -p || echo 'abcdef')"` + "\n")
			break
		}
	}
	return sb.String()
}

// buildXrayConfig 生成 sing-box 多协议 config.json 写入脚本。
func buildXrayConfig(protocols []XrayProtocol) string {
	var sb strings.Builder
	sb.WriteString("cat > \"$CONF_DIR/config.json\" <<EOF\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"log\": { \"level\": \"info\" },\n")
	sb.WriteString("  \"experimental\": { \"clash_api\": { \"external_controller\": \"127.0.0.1:19090\" } },\n")
	sb.WriteString("  \"inbounds\": [\n")

	for i, p := range protocols {
		key := sanitizeKey(p.Key)
		comma := ","
		if i == len(protocols)-1 {
			comma = ""
		}
		switch p.Key {
		case "vless-ws":
			// WS 类协议在域名模式下走 TLS（acme 证书），与客户端链接 security=tls 一致；
			// 无域名时后端已过滤掉 WS 类协议，不会走到这里。
			sb.WriteString(fmt.Sprintf(`    { "type": "vless", "tag": "in-%s", "listen": "0.0.0.0", "listen_port": %d, "users": [ { "uuid": "${UUID_%s}" } ], "tls": { "enabled": true, "certificate_path": "/etc/sing-box/cert/fullchain.pem", "key_path": "/etc/sing-box/cert/privkey.pem", "server_name": "%s" }, "transport": { "type": "ws", "path": "/${WS_%s}" } }%s`+"\n", key, p.Port, key, tlsServerName(p), key, comma))
		case "vmess-ws":
			// 修复：之前链接声称 tls:tls 但服务端没有 TLS 块 → 握手失败连不上。
			// 现在域名模式下服务端补 TLS（acme 证书），与链接一致。
			sb.WriteString(fmt.Sprintf(`    { "type": "vmess", "tag": "in-%s", "listen": "0.0.0.0", "listen_port": %d, "users": [ { "uuid": "${UUID_%s}" } ], "tls": { "enabled": true, "certificate_path": "/etc/sing-box/cert/fullchain.pem", "key_path": "/etc/sing-box/cert/privkey.pem", "server_name": "%s" }, "transport": { "type": "ws", "path": "/${WS_%s}" } }%s`+"\n", key, p.Port, key, tlsServerName(p), key, comma))
		case "vless-reality", "vless-reality-grpc", "vless-reality-xhttp":
			flow := "xtls-rprx-vision"
			transport := ""
			switch p.Key {
			case "vless-reality-grpc":
				transport = `, "transport": { "type": "grpc", "service_name": "/${WS_%s}" }`
			case "vless-reality-xhttp":
				transport = `, "transport": { "type": "httpupgrade", "path": "/${WS_%s}" }`
			}
			transportStr := ""
			if transport != "" {
				transportStr = fmt.Sprintf(transport, key)
			}
			sb.WriteString(fmt.Sprintf(`    { "type": "vless", "tag": "in-%s", "listen": "0.0.0.0", "listen_port": %d, "users": [ { "uuid": "${UUID_%s}", "flow": "%s" } ], "tls": { "enabled": true, "server_name": "%s", "reality": { "enabled": true, "handshake": { "server": "%s", "server_port": 443 }, "private_key": "${REALITY_PRIVATE_KEY}", "short_id": [ "${REALITY_SHORT_ID}" ] } }%s }%s`+"\n", key, p.Port, key, flow, defaultRealitySNI(p), defaultRealitySNI(p), transportStr, comma))
		case "vless-grpc-tls":
			sb.WriteString(fmt.Sprintf(`    { "type": "vless", "tag": "in-%s", "listen": "0.0.0.0", "listen_port": %d, "users": [ { "uuid": "${UUID_%s}" } ], "tls": { "enabled": true, "certificate_path": "/etc/sing-box/cert/fullchain.pem", "key_path": "/etc/sing-box/cert/privkey.pem", "server_name": "%s" }, "transport": { "type": "grpc", "service_name": "/${WS_%s}" } }%s`+"\n", key, p.Port, key, tlsServerName(p), key, comma))
		case "vless-tcp-tls":
			// VLESS + TCP + TLS + Vision（对齐 v2ray-agent VLESS_TCP/TLS_Vision）
			sb.WriteString(fmt.Sprintf(`    { "type": "vless", "tag": "in-%s", "listen": "0.0.0.0", "listen_port": %d, "users": [ { "uuid": "${UUID_%s}", "flow": "xtls-rprx-vision" } ], "tls": { "enabled": true, "certificate_path": "/etc/sing-box/cert/fullchain.pem", "key_path": "/etc/sing-box/cert/privkey.pem", "server_name": "%s" } }%s`+"\n", key, p.Port, key, tlsServerName(p), comma))
		case "anytls":
			// AnyTLS + TLS（sing-box 1.10+；users 用 password 字段，对齐 v2ray-agent）
			sb.WriteString(fmt.Sprintf(`    { "type": "anytls", "tag": "in-%s", "listen": "0.0.0.0", "listen_port": %d, "users": [ { "password": "${PASS_%s}" } ], "tls": { "enabled": true, "certificate_path": "/etc/sing-box/cert/fullchain.pem", "key_path": "/etc/sing-box/cert/privkey.pem", "server_name": "%s" } }%s`+"\n", key, p.Port, key, tlsServerName(p), comma))
		case "vmess-httpupgrade":
			// VMess + HTTPUpgrade + TLS（对齐 v2ray-agent VMess_HTTPUpgrade）
			sb.WriteString(fmt.Sprintf(`    { "type": "vmess", "tag": "in-%s", "listen": "0.0.0.0", "listen_port": %d, "users": [ { "uuid": "${UUID_%s}" } ], "tls": { "enabled": true, "certificate_path": "/etc/sing-box/cert/fullchain.pem", "key_path": "/etc/sing-box/cert/privkey.pem", "server_name": "%s" }, "transport": { "type": "httpupgrade", "path": "/${WS_%s}" } }%s`+"\n", key, p.Port, key, tlsServerName(p), key, comma))
		case "trojan-tcp-tls":
			sb.WriteString(fmt.Sprintf(`    { "type": "trojan", "tag": "in-%s", "listen": "0.0.0.0", "listen_port": %d, "users": [ { "password": "${PASS_%s}" } ], "tls": { "enabled": true, "certificate_path": "/etc/sing-box/cert/fullchain.pem", "key_path": "/etc/sing-box/cert/privkey.pem", "server_name": "%s" } }%s`+"\n", key, p.Port, key, tlsServerName(p), comma))
		case "trojan-ws":
			sb.WriteString(fmt.Sprintf(`    { "type": "trojan", "tag": "in-%s", "listen": "0.0.0.0", "listen_port": %d, "users": [ { "password": "${PASS_%s}" } ], "tls": { "enabled": true, "certificate_path": "/etc/sing-box/cert/fullchain.pem", "key_path": "/etc/sing-box/cert/privkey.pem", "server_name": "%s" }, "transport": { "type": "ws", "path": "/${WS_%s}" } }%s`+"\n", key, p.Port, key, tlsServerName(p), key, comma))
		case "trojan-grpc-tls":
			sb.WriteString(fmt.Sprintf(`    { "type": "trojan", "tag": "in-%s", "listen": "0.0.0.0", "listen_port": %d, "users": [ { "password": "${PASS_%s}" } ], "tls": { "enabled": true, "certificate_path": "/etc/sing-box/cert/fullchain.pem", "key_path": "/etc/sing-box/cert/privkey.pem", "server_name": "%s" }, "transport": { "type": "grpc", "service_name": "/${WS_%s}" } }%s`+"\n", key, p.Port, key, tlsServerName(p), key, comma))
		case "ss":
			sb.WriteString(fmt.Sprintf(`    { "type": "shadowsocks", "tag": "in-%s", "listen": "0.0.0.0", "listen_port": %d, "method": "aes-128-gcm", "password": "${PASS_%s}" }%s`+"\n", key, p.Port, key, comma))
		case "hysteria2":
			// sing-box 1.11 hysteria2 使用 users 数组（顶层 password 字段不存在）
			sb.WriteString(fmt.Sprintf(`    { "type": "hysteria2", "tag": "in-%s", "listen": "0.0.0.0", "listen_port": %d, "users": [ { "password": "${PASS_%s}" } ], "tls": { "enabled": true, "certificate_path": "/etc/sing-box/cert/fullchain.pem", "key_path": "/etc/sing-box/cert/privkey.pem" } }%s`+"\n", key, p.Port, key, comma))
		case "tuic":
			sb.WriteString(fmt.Sprintf(`    { "type": "tuic", "tag": "in-%s", "listen": "0.0.0.0", "listen_port": %d, "users": [ { "uuid": "${UUID_%s}", "password": "${PASS_%s}" } ], "congestion_control": "bbr", "tls": { "enabled": true, "certificate_path": "/etc/sing-box/cert/fullchain.pem", "key_path": "/etc/sing-box/cert/privkey.pem" } }%s`+"\n", key, p.Port, key, key, comma))
		}
	}

	sb.WriteString("  ],\n")
	sb.WriteString("  \"outbounds\": [ { \"type\": \"direct\" } ]\n")
	sb.WriteString("}\nEOF\n")
	return sb.String()
}

// buildXrayLinks 生成构造各协议 LINK 变量的脚本，并定义 LINKS 数组。
func buildXrayLinks(protocols []XrayProtocol) string {
	var sb strings.Builder
	sb.WriteString("LINKS=()\n")
	for _, p := range protocols {
		key := sanitizeKey(p.Key)
		switch p.Key {
		case "vless-ws":
			// 域名模式下 WS 走 TLS：security=tls + sni + fp（与 v2ray-agent VLESS_WS 对齐）
			sb.WriteString(fmt.Sprintf(`LINK_%s="vless://${UUID_%s}@${SERVER_ADDR}:%d?encryption=none&security=tls&sni=${DOMAIN}&fp=chrome&type=ws&host=${DOMAIN}&path=%%2F${WS_%s}#${SERVER_ADDR}-%s"
LINKS+=("$LINK_%s")
`+"\n", key, key, p.Port, key, protoLabel(p.Key), key))
		case "vmess-ws":
			// 修复：链接 tls:tls 与服务端 TLS 一致；补 scy/sni/fp 字段（对齐 v2ray-agent VMess_WS）
			sb.WriteString(fmt.Sprintf(`VJSON_%s="{\"v\":2,\"ps\":\"${SERVER_ADDR}-%s\",\"add\":\"${SERVER_ADDR}\",\"port\":\"%d\",\"id\":\"${UUID_%s}\",\"aid\":\"0\",\"scy\":\"auto\",\"net\":\"ws\",\"host\":\"${DOMAIN}\",\"path\":\"/${WS_%s}\",\"tls\":\"tls\",\"sni\":\"${DOMAIN}\",\"fp\":\"chrome\",\"alpn\":\"\"}"
LINK_%s="vmess://$(echo -n "$VJSON_%s" | base64 -w0 | tr -d '=')"
LINKS+=("$LINK_%s")
`+"\n", key, protoLabel(p.Key), p.Port, key, key, key, key, key))
		case "vless-reality":
			sb.WriteString(fmt.Sprintf(`LINK_%s="vless://${UUID_%s}@${SERVER_ADDR}:%d?encryption=none&flow=xtls-rprx-vision&security=reality&sni=%s&fp=chrome&pbk=${REALITY_PUBLIC_KEY}&sid=${REALITY_SHORT_ID}&type=tcp#${SERVER_ADDR}-%s"
LINKS+=("$LINK_%s")
`+"\n", key, key, p.Port, defaultRealitySNI(p), protoLabel(p.Key), key))
		case "vless-reality-grpc":
			sb.WriteString(fmt.Sprintf(`LINK_%s="vless://${UUID_%s}@${SERVER_ADDR}:%d?encryption=none&flow=xtls-rprx-vision&security=reality&sni=%s&fp=chrome&pbk=${REALITY_PUBLIC_KEY}&sid=${REALITY_SHORT_ID}&type=grpc&serviceName=%%2F${WS_%s}#${SERVER_ADDR}-%s"
LINKS+=("$LINK_%s")
`+"\n", key, key, p.Port, defaultRealitySNI(p), key, protoLabel(p.Key), key))
		case "vless-reality-xhttp":
			sb.WriteString(fmt.Sprintf(`LINK_%s="vless://${UUID_%s}@${SERVER_ADDR}:%d?encryption=none&flow=xtls-rprx-vision&security=reality&sni=%s&fp=chrome&pbk=${REALITY_PUBLIC_KEY}&sid=${REALITY_SHORT_ID}&type=httpupgrade&path=%%2F${WS_%s}#${SERVER_ADDR}-%s"
LINKS+=("$LINK_%s")
`+"\n", key, key, p.Port, defaultRealitySNI(p), key, protoLabel(p.Key), key))
		case "vless-grpc-tls":
			sb.WriteString(fmt.Sprintf(`LINK_%s="vless://${UUID_%s}@${SERVER_ADDR}:%d?encryption=none&security=tls&sni=${DOMAIN}&fp=chrome&type=grpc&serviceName=%%2F${WS_%s}&host=${DOMAIN}#${SERVER_ADDR}-%s"
LINKS+=("$LINK_%s")
`+"\n", key, key, p.Port, key, protoLabel(p.Key), key))
		case "trojan-tcp-tls":
			// 补 fp=chrome&alpn=http/1.1（对齐 v2ray-agent Trojan_TCP，提升握手兼容性）
			sb.WriteString(fmt.Sprintf(`LINK_%s="trojan://${PASS_%s}@${SERVER_ADDR}:%d?security=tls&sni=${DOMAIN}&fp=chrome&alpn=http%%2F1.1&type=tcp#${SERVER_ADDR}-%s"
LINKS+=("$LINK_%s")
`+"\n", key, key, p.Port, protoLabel(p.Key), key))
		case "trojan-ws":
			sb.WriteString(fmt.Sprintf(`LINK_%s="trojan://${PASS_%s}@${SERVER_ADDR}:%d?type=ws&path=%%2F${WS_%s}&security=tls&sni=${DOMAIN}&fp=chrome&host=${DOMAIN}#${SERVER_ADDR}-%s"
LINKS+=("$LINK_%s")
`+"\n", key, key, p.Port, key, protoLabel(p.Key), key))
		case "trojan-grpc-tls":
			sb.WriteString(fmt.Sprintf(`LINK_%s="trojan://${PASS_%s}@${SERVER_ADDR}:%d?security=tls&sni=${DOMAIN}&fp=chrome&type=grpc&serviceName=%%2F${WS_%s}&host=${DOMAIN}#${SERVER_ADDR}-%s"
LINKS+=("$LINK_%s")
`+"\n", key, key, p.Port, key, protoLabel(p.Key), key))
		case "vless-tcp-tls":
			// VLESS + TCP + TLS + Vision（对齐 v2ray-agent VLESS_TCP/TLS_Vision）
			sb.WriteString(fmt.Sprintf(`LINK_%s="vless://${UUID_%s}@${SERVER_ADDR}:%d?encryption=none&flow=xtls-rprx-vision&security=tls&sni=${DOMAIN}&fp=chrome&type=tcp&headerType=none&host=${DOMAIN}#${SERVER_ADDR}-%s"
LINKS+=("$LINK_%s")
`+"\n", key, key, p.Port, protoLabel(p.Key), key))
		case "anytls":
			// AnyTLS + TLS（password 即 UUID，对齐 v2ray-agent anytls 链接格式）
			sb.WriteString(fmt.Sprintf(`LINK_%s="anytls://${PASS_%s}@${SERVER_ADDR}:%d?security=tls&sni=${DOMAIN}&insecure=0&type=tcp&headerType=none#${SERVER_ADDR}-%s"
LINKS+=("$LINK_%s")
`+"\n", key, key, p.Port, protoLabel(p.Key), key))
		case "vmess-httpupgrade":
			// VMess + HTTPUpgrade + TLS（对齐 v2ray-agent VMess_HTTPUpgrade）
			sb.WriteString(fmt.Sprintf(`VJSON_%s="{\"v\":2,\"ps\":\"${SERVER_ADDR}-%s\",\"add\":\"${SERVER_ADDR}\",\"port\":\"%d\",\"id\":\"${UUID_%s}\",\"aid\":\"0\",\"scy\":\"auto\",\"net\":\"httpupgrade\",\"host\":\"${DOMAIN}\",\"path\":\"/${WS_%s}\",\"tls\":\"tls\",\"sni\":\"${DOMAIN}\",\"fp\":\"chrome\",\"alpn\":\"\"}"
LINK_%s="vmess://$(echo -n "$VJSON_%s" | base64 -w0 | tr -d '=')"
LINKS+=("$LINK_%s")
`+"\n", key, protoLabel(p.Key), p.Port, key, key, key, key, key))
		case "ss":
			sb.WriteString(fmt.Sprintf(`SSB64_%s="$(echo -n "aes-128-gcm:${PASS_%s}" | base64 -w0 | tr -d '=')"
LINK_%s="ss://${SSB64_%s}@${SERVER_ADDR}:%d#${SERVER_ADDR}-%s"
LINKS+=("$LINK_%s")
`+"\n", key, key, key, key, p.Port, protoLabel(p.Key), key))
		case "hysteria2":
			// 有 acme 真证书，insecure=0 校验证书（对齐 v2ray-agent）
			sb.WriteString(fmt.Sprintf(`LINK_%s="hysteria2://${PASS_%s}@${SERVER_ADDR}:%d?sni=${DOMAIN}&insecure=0&alpn=h3#${SERVER_ADDR}-%s"
LINKS+=("$LINK_%s")
`+"\n", key, key, p.Port, protoLabel(p.Key), key))
		case "tuic":
			sb.WriteString(fmt.Sprintf(`LINK_%s="tuic://${UUID_%s}:${PASS_%s}@${SERVER_ADDR}:%d?sni=${DOMAIN}&alpn=h3&congestion_control=bbr#${SERVER_ADDR}-%s"
LINKS+=("$LINK_%s")
`+"\n", key, key, key, p.Port, protoLabel(p.Key), key))
		}
	}
	return sb.String()
}

// buildXrayBatchPayload 生成批量回传 JSON 的 links 数组拼接脚本。
// 输出形如: BATCH_JSON="${BATCH_JSON}\"${LINK_vless_ws}\"," （每个协议一行）
func buildXrayBatchPayload(protocols []XrayProtocol) string {
	var sb strings.Builder
	for _, p := range protocols {
		key := sanitizeKey(p.Key)
		sb.WriteString(fmt.Sprintf(`if [ "$FIRST" = "1" ]; then FIRST=0; else BATCH_JSON="${BATCH_JSON},"; fi
BATCH_JSON="${BATCH_JSON}\"${LINK_%s}\""
`+"\n", key))
	}
	return sb.String()
}

func sanitizeKey(k string) string {
	return strings.NewReplacer("-", "_", ".", "_").Replace(k)
}

// tlsServerName 返回 TLS 协议的 server_name（有域名用域名，否则用 IP）。
func tlsServerName(p XrayProtocol) string {
	if p.Domain != "" {
		return p.Domain
	}
	if p.ServerIP != "" {
		return p.ServerIP
	}
	return "${IP}"
}

func defaultRealitySNI(p XrayProtocol) string {
	if p.RealitySNI != "" {
		return p.RealitySNI
	}
	return "www.microsoft.com"
}

func protoLabel(k string) string {
	switch k {
	case "vless-ws":
		return "VLESS-WS"
	case "vmess-ws":
		return "VMess-WS"
	case "vless-reality":
		return "VLESS-Reality"
	case "vless-reality-grpc":
		return "VLESS-Reality-gRPC"
	case "vless-reality-xhttp":
		return "VLESS-Reality-XHTTP"
	case "vless-grpc-tls":
		return "VLESS-gRPC-TLS"
	case "trojan-tcp-tls":
		return "Trojan-TCP-TLS"
	case "trojan-ws":
		return "Trojan-WS"
	case "trojan-grpc-tls":
		return "Trojan-gRPC-TLS"
	case "vless-tcp-tls":
		return "VLESS-TCP-TLS-Vision"
	case "anytls":
		return "AnyTLS"
	case "vmess-httpupgrade":
		return "VMess-HTTPUpgrade"
	case "ss":
		return "SS"
	case "hysteria2":
		return "Hysteria2"
	case "tuic":
		return "TUIC"
	default:
		return strings.ToUpper(k)
	}
}

var _ = json.Marshal

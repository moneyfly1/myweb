// 从 VPS 上已部署的 sing-box 配置回填「自建节点」的 config 字段。
//
// 背景（2026-10-23 用户反馈）：VPS 自动搭建出来的节点在专线节点列表里显示 online，
// 但点「查看链接」提示「无法生成链接: 配置格式错误或协议不支持」、点「编辑」没有任何信息。
// 根因是这些节点的 config 字段为空 —— agent 只跑了心跳（心跳会把 status 置 online），
// 回传节点链接那一步没成功 / 又被重装流程清空，而面板的查看/编辑都依赖 config。
//
// 本工具直接 SSH 到那台 VPS 读 /etc/sing-box/config.json，把每个 inbound 的
// 服务器/端口/凭据/传输层解析成面板的 ProxyNode 结构写回 custom_nodes.config。
// 默认只预演（dry-run），加 -apply 才写库。
//
// 用法（项目目录下）：
//
//	go run ./cmd/sync-selfhost-config -install-id 9edbdc07ead9be58
//	go run ./cmd/sync-selfhost-config -install-id 9edbdc07ead9be58 -apply
//	go run ./cmd/sync-selfhost-config -config-json /tmp/sb_config.json -install-id xxx   # 离线调试
package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"
	"cboard-go/internal/services/sshdeploy"
	"cboard-go/internal/utils"

	"golang.org/x/crypto/curve25519"
)

// singBoxInbound sing-box config.json 里的一个监听项（只取回填需要的字段）
type singBoxInbound struct {
	Type       string          `json:"type"`
	Tag        string          `json:"tag"`
	ListenPort int             `json:"listen_port"`
	Method     string          `json:"method"`
	Password   string          `json:"password"`
	Users      []singBoxUser   `json:"users"`
	TLS        *singBoxTLS     `json:"tls"`
	Transport  *singBoxTranspt `json:"transport"`
}

type singBoxUser struct {
	UUID     string `json:"uuid"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type singBoxTLS struct {
	Enabled    bool            `json:"enabled"`
	ServerName string          `json:"server_name"`
	Reality    *singBoxReality `json:"reality"`
}

type singBoxReality struct {
	Enabled   bool           `json:"enabled"`
	PrivateKV string         `json:"private_key"`
	ShortID   []string       `json:"short_id"`
	// handshake 里既有字符串（server）也有数字（server_port），用 any 接收
	Handshake map[string]any `json:"handshake"`
}

type singBoxTranspt struct {
	Type        string `json:"type"`
	Path        string `json:"path"`
	ServiceName string `json:"service_name"`
}

type singBoxConfig struct {
	Inbounds []singBoxInbound `json:"inbounds"`
}

func main() {
	installID := flag.String("install-id", "", "自建节点的 install_id（同一次部署的节点共用）")
	nodeID := flag.Uint("node-id", 0, "只处理指定节点 id")
	configJSONPath := flag.String("config-json", "", "本地已有的 sing-box config.json（跳过 SSH，调试用）")
	apply := flag.Bool("apply", false, "真正写库（缺省只预演）")
	flag.Parse()

	if *installID == "" && *nodeID == 0 {
		log.Fatal("必须指定 -install-id 或 -node-id")
	}
	if _, err := config.LoadConfig(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if err := database.InitDatabase(); err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	db := database.GetDB()

	q := db.Where("self_hosted = ?", true)
	if *nodeID > 0 {
		q = q.Where("id = ?", *nodeID)
	} else {
		q = q.Where("install_id = ?", *installID)
	}
	var nodes []models.CustomNode
	if err := q.Order("id ASC").Find(&nodes).Error; err != nil {
		log.Fatalf("查询节点失败: %v", err)
	}
	if len(nodes) == 0 {
		log.Fatal("没有找到匹配的自建节点")
	}
	main := nodes[0]

	mode := "预演（不写库）"
	if *apply {
		mode = "执行（写库）"
	}
	fmt.Printf("=== 自建节点配置回填 · %s ===\n", mode)
	fmt.Printf("install_id=%s 节点数=%d VPS=%s\n\n", main.InstallID, len(nodes), main.SSHHost)

	raw, err := loadSingBoxConfig(main, *configJSONPath)
	if err != nil {
		log.Fatalf("读取 sing-box 配置失败: %v", err)
	}
	var cfg singBoxConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		log.Fatalf("解析 sing-box 配置失败: %v", err)
	}
	fmt.Printf("sing-box inbounds: %d\n", len(cfg.Inbounds))

	// 解析每个 inbound → ProxyNode；按「协议 key + 端口」建索引
	byKeyPort := make(map[string]*config_update.ProxyNode, len(cfg.Inbounds))
	for _, in := range cfg.Inbounds {
		pn, key, err := inboundToProxyNode(in, main)
		if err != nil {
			fmt.Printf("  [跳过] %s: %v\n", in.Tag, err)
			continue
		}
		byKeyPort[fmt.Sprintf("%s|%d", key, in.ListenPort)] = pn
	}

	svc := config_update.NewConfigUpdateService()
	updated, skipped := 0, 0
	for i := range nodes {
		n := &nodes[i]
		key := strings.TrimSpace(n.SelfHostProtocol)
		lookup := fmt.Sprintf("%s|%d", key, n.Port)
		pn, ok := byKeyPort[lookup]
		if !ok {
			// 协议 key 对不上时退化为「同端口」匹配（老数据可能只有 protocol 没有 key）
			for k, v := range byKeyPort {
				if strings.HasSuffix(k, fmt.Sprintf("|%d", n.Port)) {
					pn, ok = v, true
					lookup = k
					break
				}
			}
		}
		if !ok {
			fmt.Printf("  [未匹配] #%d %s（协议 %s 端口 %d）\n", n.ID, n.Name, key, n.Port)
			skipped++
			continue
		}
		// 用面板里的展示名覆盖节点名，让生成的链接跟随面板改名
		pn.Name = n.DisplayName
		if pn.Name == "" {
			pn.Name = n.Name
		}
		link := svc.NodeToLink(pn)
		if link == "" {
			fmt.Printf("  [生成失败] #%d %s（协议 %s）\n", n.ID, n.Name, key)
			skipped++
			continue
		}
		cfgJSON, err := json.Marshal(pn)
		if err != nil {
			fmt.Printf("  [序列化失败] #%d %s: %v\n", n.ID, n.Name, err)
			skipped++
			continue
		}
		fmt.Printf("  [ok] #%d %-28s ← %s\n       %s\n", n.ID, n.Name, lookup, truncate(link, 110))
		if *apply {
			if err := db.Model(&models.CustomNode{}).Where("id = ?", n.ID).Updates(map[string]interface{}{
				"config":   string(cfgJSON),
				"domain":   pn.Server,
				"port":     pn.Port,
				"protocol": pn.Type,
			}).Error; err != nil {
				fmt.Printf("  [写库失败] #%d: %v\n", n.ID, err)
				skipped++
				continue
			}
			updated++
		}
	}

	fmt.Printf("\n完成：匹配 %d 个，未匹配/失败 %d 个\n", len(nodes)-skipped, skipped)
	if !*apply {
		fmt.Println("\n以上为预演结果，确认无误后加 -apply 写库：")
		fmt.Printf("  go run ./cmd/sync-selfhost-config -install-id %s -apply\n", main.InstallID)
	}
}

func readLocal(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// loadSingBoxConfig 优先读本地文件，否则 SSH 到 VPS cat /etc/sing-box/config.json
func loadSingBoxConfig(node models.CustomNode, localPath string) ([]byte, error) {
	if localPath != "" {
		return readLocal(localPath)
	}
	pass, err := utils.DecryptSecret(node.SSHPasswordEnc)
	if err != nil || pass == "" {
		return nil, fmt.Errorf("解密 SSH 密码失败: %v", err)
	}
	client, err := sshdeploy.Dial(sshdeploy.Credentials{
		Host: node.SSHHost, Port: node.SSHPort, User: node.SSHUser, Password: pass,
	}, 20*time.Second)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败: %w", err)
	}
	defer client.Close()
	out, err := client.RunWithTimeout("cat /etc/sing-box/config.json", 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("读取 config.json 失败: %w", err)
	}
	return []byte(out), nil
}

// inboundToProxyNode 把 sing-box 的 inbound 还原成面板的 ProxyNode。
// 返回 (节点, 协议key, 错误)。协议 key 与面板 self_host_protocol 一一对应，
// 用于和 custom_nodes 精确匹配。
func inboundToProxyNode(in singBoxInbound, main models.CustomNode) (*config_update.ProxyNode, string, error) {
	server := strings.TrimSpace(main.Domain)
	if server == "" {
		server = strings.TrimSpace(main.SSHHost)
	}
	if server == "" {
		return nil, "", fmt.Errorf("节点没有域名/IP，无法确定服务器地址")
	}
	pn := &config_update.ProxyNode{
		Type:   in.Type,
		Server: server,
		Port:   in.ListenPort,
	}
	opts := map[string]any{}
	sni := ""
	if in.TLS != nil {
		sni = strings.TrimSpace(in.TLS.ServerName)
		pn.TLS = in.TLS.Enabled
	}
	if sni != "" {
		opts["servername"] = sni
	}
	if in.TLS != nil && in.TLS.Reality != nil && in.TLS.Reality.Enabled {
		pub, err := realityPublicKey(in.TLS.Reality.PrivateKV)
		if err != nil {
			return nil, "", fmt.Errorf("reality 公钥推导失败: %w", err)
		}
		ro := map[string]any{"public-key": pub}
		if len(in.TLS.Reality.ShortID) > 0 {
			ro["short-id"] = in.TLS.Reality.ShortID[0]
		}
		opts["reality-opts"] = ro
		opts["client-fingerprint"] = "chrome"
	}

	tag := strings.TrimPrefix(in.Tag, "in-")
	key := strings.ReplaceAll(tag, "_", "-")

	switch {
	case in.Type == "shadowsocks":
		if in.Password == "" || in.Method == "" {
			return nil, "", fmt.Errorf("shadowsocks 缺少 method/password（旧脚本未写入，需要重新搭建该协议）")
		}
		pn.Cipher = in.Method
		pn.Password = in.Password
		key = "ss"
	case in.Type == "hysteria2":
		if len(in.Users) == 0 || in.Users[0].Password == "" {
			return nil, "", fmt.Errorf("hysteria2 缺少用户密码")
		}
		pn.Password = in.Users[0].Password
		opts["alpn"] = []string{"h3"}
		key = "hysteria2"
	case in.Type == "tuic":
		if len(in.Users) == 0 {
			return nil, "", fmt.Errorf("tuic 缺少用户凭据")
		}
		pn.UUID = in.Users[0].UUID
		pn.Password = in.Users[0].Password
		opts["alpn"] = []string{"h3"}
		opts["congestion_control"] = "bbr"
		key = "tuic"
	case in.Type == "anytls":
		if len(in.Users) == 0 || in.Users[0].Password == "" {
			return nil, "", fmt.Errorf("anytls 缺少用户密码")
		}
		pn.Password = in.Users[0].Password
		key = "anytls"
	case in.Type == "vless" || in.Type == "vmess":
		if len(in.Users) == 0 || in.Users[0].UUID == "" {
			return nil, "", fmt.Errorf("%s 缺少 uuid", in.Type)
		}
		pn.UUID = in.Users[0].UUID
		if in.Type == "vmess" {
			opts["cipher"] = "auto"
		}
	case in.Type == "trojan":
		if len(in.Users) == 0 || in.Users[0].Password == "" {
			return nil, "", fmt.Errorf("trojan 缺少密码")
		}
		pn.Password = in.Users[0].Password
	default:
		return nil, "", fmt.Errorf("不支持的协议类型 %s", in.Type)
	}

	// 传输层
	if in.Transport != nil {
		switch in.Transport.Type {
		case "ws":
			pn.Network = "ws"
			path := in.Transport.Path
			opts["ws-opts"] = map[string]any{"path": path, "headers": map[string]string{"Host": sni}}
		case "grpc":
			pn.Network = "grpc"
			opts["grpc-opts"] = map[string]any{"grpc-service-name": strings.TrimPrefix(in.Transport.ServiceName, "/")}
		case "httpupgrade":
			pn.Network = "httpupgrade"
			opts["ws-opts"] = map[string]any{"path": in.Transport.Path}
		}
	}
	// VLESS + TCP + TLS 用 Vision（与脚本生成的链接一致）
	if key == "vless-tcp-tls" {
		opts["flow"] = "xtls-rprx-vision"
		opts["client-fingerprint"] = "chrome"
	}
	if key == "trojan-tcp-tls" {
		opts["alpn"] = []string{"http/1.1"}
	}
	if in.Type == "vless" || in.Type == "vmess" || in.Type == "trojan" {
		if _, ok := opts["client-fingerprint"]; !ok && pn.TLS {
			opts["client-fingerprint"] = "chrome"
		}
	}
	pn.Options = opts
	return pn, key, nil
}

// realityPublicKey 由 sing-box 的 reality 私钥（base64url，无 padding）推导公钥。
func realityPublicKey(priv string) (string, error) {
	priv = strings.TrimSpace(priv)
	if priv == "" {
		return "", fmt.Errorf("私钥为空")
	}
	raw, err := base64.RawURLEncoding.DecodeString(priv)
	if err != nil {
		if raw, err = base64.RawStdEncoding.DecodeString(priv); err != nil {
			return "", fmt.Errorf("私钥不是合法的 base64url")
		}
	}
	if len(raw) != 32 {
		return "", fmt.Errorf("私钥长度应为 32 字节，实际 %d", len(raw))
	}
	pub, err := curve25519.X25519(raw, curve25519.Basepoint)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(pub), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

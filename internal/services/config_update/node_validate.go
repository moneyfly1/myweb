package config_update

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"

	"cboard-go/internal/models"
	"gorm.io/gorm"
)

// ============================================================================
// 第一层防御：采集/解析阶段静态校验（白名单 + 必填字段 + 取值白名单）
//
// 目标：坏节点在「入库之前」就被丢弃，永远不会参与配置生成。
//
// 所有白名单均以真内核实测为准，实测命令（mihomo Meta v1.19.32 linux/arm64）：
//
//	mihomo -t -d <workdir> -f <cfg.yaml>
//
// 每种取值生成一份最小 ss/ssr/vmess/vless 配置逐条跑 -t，PASS/FAIL 清单见
// 下方各白名单的注释，完整实测输出见上线报告「ss/ssr 专项」一节。
// ============================================================================

// 校验原因码（后台可按 reason 一眼看出根因）
const (
	ReasonUnsupportedType      = "unsupported-type"           // 内核不支持的协议类型（如 naive/naive+https）
	ReasonEmptyName            = "empty-name"                 // 节点名为空
	ReasonInvalidServer        = "invalid-server"             // server 非空但非合法域名/IP
	ReasonInvalidPort          = "invalid-port"               // port 不在 1..65535
	ReasonMissingField         = "missing-field"              // 必填字段缺失（detail 里带字段名）
	ReasonUnsupportedCipher    = "cipher-unsupported"         // ss/ssr cipher 不在内核白名单
	ReasonCipherKeyNotB64      = "cipher-key-not-base64"      // ss 2022-blake3 密钥不是合法 base64
	ReasonCipherKeyMismatch    = "cipher-key-length-mismatch" // ss 2022-blake3 密钥长度不等于内核要求
	ReasonPluginInvalid        = "plugin-invalid"             // ss plugin 名未知或必需参数缺失
	ReasonUnsupportedProtocol  = "ssr-protocol-unsupported"   // ssr protocol 不在内核枚举内
	ReasonUnsupportedObfs      = "ssr-obfs-unsupported"       // ssr obfs 不在内核枚举内
	ReasonVMessCipher          = "vmess-cipher-unsupported"   // vmess cipher 不在内核白名单
	ReasonVLESSFlow            = "vless-flow-unsupported"     // vless flow 被内核拒绝
	ReasonVLESSEncryption      = "vless-encryption-unsupported"
	ReasonKernelInvalid        = "kernel-config-invalid"   // 第二层：内核 -t 判定该节点使配置失效
	ReasonWireGuardKey         = "wireguard-key-invalid"   // wireguard 密钥非法 base64
	ReasonObfsInvalid          = "obfs-unsupported"        // hysteria2 等 obfs 取值不被内核支持
	ReasonRealityInvalid       = "reality-opts-incomplete" // reality-opts 缺 public-key
	ReasonReservedName         = "reserved-name"           // 节点名与内核内置代理名冲突（DIRECT 等）
	ReasonCredentialURLDecoded = "credential-url-decoded"  // 修正：凭据含 %XX 转义，已做一次 URL 解码还原
	// ReasonCipherKeyInvalid 是"密钥类校验不通过"的总类码：node_validation_logs.reason 以精确
	// 子类开头并在正文带上本总类码，因此
	//   WHERE reason LIKE '%cipher-key-invalid%'  → 命中全部密钥类丢弃
	// 既能按总类检索，也保留 not-base64 / length-mismatch 的精确聚合。
	ReasonCipherKeyInvalid = "cipher-key-invalid"
)

// mihomoSupportedNodeTypes mihomo 实际支持的代理协议白名单。
// 注意：naive / naive+https 不在其中——内核会直接报
// "unsupport proxy type: naive" 并让「整份配置」失效（实测见报告）。
var mihomoSupportedNodeTypes = map[string]bool{
	"ss": true, "ssr": true, "vmess": true, "vless": true, "trojan": true,
	"hysteria": true, "hysteria2": true, "tuic": true, "wireguard": true,
	"anytls": true, "socks5": true, "http": true, "snell": true, "ssh": true,
	"mieru": true,
}

// nodeTypeAliases 面板内部/链接里的别名 → 内核类型名
var nodeTypeAliases = map[string]string{
	"socks": "socks5",
	"wg":    "wireguard",
	"hy2":   "hysteria2",
}

// mihomoReservedProxyNames 内核内置代理名，节点重名会让整份配置失效。
// 内核实测（大小写敏感，仅全大写触发 "proxy X is the duplicate name"）：
//
//	DIRECT / REJECT / REJECT-DROP / PASS / COMPATIBLE → FAIL
//	direct / Direct / reject / GLOBAL / PROXY / Proxy  → PASS（不拦，避免误杀）
var mihomoReservedProxyNames = map[string]bool{
	"DIRECT": true, "REJECT": true, "REJECT-DROP": true, "PASS": true, "COMPATIBLE": true,
}

// ssCipherWhitelist ss cipher 白名单（真内核实测 PASS 的取值，逐条验证命令见文件头）。
//
// 实测 PASS：
//
//	AEAD      : aes-128-gcm aes-192-gcm aes-256-gcm chacha20-ietf-poly1305
//	            xchacha20-ietf-poly1305 chacha8-ietf-poly1305 xchacha8-ietf-poly1305
//	            aes-128-gcm-siv aes-256-gcm-siv aes-128-ccm aes-256-ccm
//	2022      : 2022-blake3-aes-128-gcm 2022-blake3-aes-256-gcm 2022-blake3-chacha20-poly1305
//	流密码    : aes-128-cfb aes-192-cfb aes-256-cfb aes-128-ctr aes-192-ctr aes-256-ctr
//	            chacha20-ietf chacha20 xchacha20 rc4-md5
//	特殊      : none（内核实测接受=明文，保留以免误杀现网节点，但会在自检日志中可见）
//
// 实测 FAIL（因此必须拒绝，否则整份配置失效）：
//
//	salsa20 rc4-md5-6 aes-128-ocb aes-192-ocb aes-256-ocb
//	aes-128-cfb1/8 aes-192-cfb1/8 aes-256-cfb1/8 aes-128-cfb128 aes-256-cfb128
//	AEAD_AES_128_GCM dummy "" bogus-xyz aes-256-gcm2
var ssCipherWhitelist = map[string]bool{
	"aes-128-gcm": true, "aes-192-gcm": true, "aes-256-gcm": true,
	"chacha20-ietf-poly1305": true, "xchacha20-ietf-poly1305": true,
	"chacha8-ietf-poly1305": true, "xchacha8-ietf-poly1305": true,
	"aes-128-gcm-siv": true, "aes-256-gcm-siv": true,
	"aes-128-ccm": true, "aes-256-ccm": true,
	"2022-blake3-aes-128-gcm": true, "2022-blake3-aes-256-gcm": true,
	"2022-blake3-chacha20-poly1305": true,
	"aes-128-cfb":                   true, "aes-192-cfb": true, "aes-256-cfb": true,
	"aes-128-ctr": true, "aes-192-ctr": true, "aes-256-ctr": true,
	"chacha20-ietf": true, "chacha20": true, "xchacha20": true, "rc4-md5": true,
	"none": true,
}

// ss2022RequiredKeyLen 2022-blake3-* 的密钥字节长度（实测：长度不符 → "bad key length"，
// 整份配置失效）。password 形如 "k" 或 "serverKey:userKey1[:userKey2...]"，
// 每一段都必须是合法 base64 且解码后长度精确等于该值（实测 16:16 PASS / 32:16 FAIL）。
var ss2022RequiredKeyLen = map[string]int{
	"2022-blake3-aes-128-gcm":       16,
	"2022-blake3-aes-256-gcm":       32,
	"2022-blake3-chacha20-poly1305": 32,
}

// ssrCipherWhitelist ssr 的 cipher 白名单与 ss **不同**：ssr 只接受流密码或 none。
// 实测：ssr + aes-128-gcm → "aes-128-gcm is not none or a supported stream cipher in ssr"（FAIL）。
var ssrCipherWhitelist = map[string]bool{
	"aes-128-cfb": true, "aes-192-cfb": true, "aes-256-cfb": true,
	"aes-128-ctr": true, "aes-192-ctr": true, "aes-256-ctr": true,
	"chacha20-ietf": true, "chacha20": true, "xchacha20": true,
	"rc4-md5": true, "none": true,
}

// ssrProtocolWhitelist 实测 PASS：origin auth_sha1_v4 auth_aes128_md5 auth_aes128_sha1
// auth_chain_a auth_chain_b。
// 实测 FAIL：auth_chain_c/d/e/f auth_sha1_v4_compatible verify_sha1 verify_simple ""。
var ssrProtocolWhitelist = map[string]bool{
	"origin": true, "auth_sha1_v4": true, "auth_aes128_md5": true,
	"auth_aes128_sha1": true, "auth_chain_a": true, "auth_chain_b": true,
}

// ssrObfsWhitelist 实测 PASS：plain http_simple http_post random_head
// tls1.2_ticket_auth tls1.2_ticket_fastauth。
// 实测 FAIL：tls1.0_session_auth "" bogus。
var ssrObfsWhitelist = map[string]bool{
	"plain": true, "http_simple": true, "http_post": true,
	"random_head": true, "tls1.2_ticket_auth": true, "tls1.2_ticket_fastauth": true,
}

// vmessCipherWhitelist 实测 PASS：auto aes-128-gcm chacha20-poly1305 none zero。
// 实测 FAIL："" bogus → "vmess: unsupported security type"。
var vmessCipherWhitelist = map[string]bool{
	"auto": true, "aes-128-gcm": true, "chacha20-poly1305": true,
	"none": true, "zero": true,
}

// vlessFlowBlocklist 实测被内核拒绝的 xtls flow（会让整份配置失效）。
// 注意解析器在 xtls=1 时会生成 xtls-rprx-direct，必须在此拦下。
// 实测：xtls-rprx-vision PASS，"" PASS，未知值（bogus-flow）内核容忍 PASS。
var vlessFlowBlocklist = map[string]bool{
	"xtls-rprx-direct": true, "xtls-rprx-origin": true,
}

// vlessEncryptionWhitelist 实测 PASS：none / ""（空）。实测 FAIL：bogus
// → "invaild vless encryption value"。
var vlessEncryptionWhitelist = map[string]bool{
	"none": true, "": true,
}

// ssPluginSpec ss 插件必需参数（内核实测：obfs 缺 mode → "obfs mode error"，
// v2ray-plugin 缺 mode → "” has unset fields: mode"，都会让整份配置失效）。
type ssPluginSpec struct {
	Aliases  []string
	Required []string
	// Enum 限定某个参数的合法取值（内核实测）。缺省表示不校验取值。
	Enum map[string][]string
}

// ssPluginSpecs 内核实测结论：
//   - obfs         : 必需 mode，且 mode 只能是 tls / http（mode=bogus/空 → "obfs mode error"）
//   - v2ray-plugin : 必需 mode，且 mode 只能是 websocket（mode=quic → 同样报 obfs mode error）
//   - shadow-tls   : 必需 host + password（version 可缺省）
//   - restls       : 必需 host + password
var ssPluginSpecs = []ssPluginSpec{
	{Aliases: []string{"obfs", "simple-obfs", "obfs-local"}, Required: []string{"mode"},
		Enum: map[string][]string{"mode": {"tls", "http"}}},
	{Aliases: []string{"v2ray-plugin"}, Required: []string{"mode"},
		Enum: map[string][]string{"mode": {"websocket"}}},
	{Aliases: []string{"shadow-tls", "shadowtls"}, Required: []string{"host", "password"}},
	{Aliases: []string{"restls"}, Required: []string{"host", "password", "version"}},
}

// hysteria2ObfsWhitelist 内核实测：obfs 只接受 salamander（空=不启用）。
// 其它取值 → "unknown obfs type" → 整份配置失效。
var hysteria2ObfsWhitelist = map[string]bool{"": true, "salamander": true}

var (
	domainRe = regexp.MustCompile(`^(?i)[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$`)
)

// NodeValidationError 节点校验失败原因
type NodeValidationError struct {
	Code   string
	Detail string
}

func (e *NodeValidationError) Error() string {
	return e.Code + ": " + e.Detail
}

func newValidationError(code, format string, args ...any) *NodeValidationError {
	return &NodeValidationError{Code: code, Detail: fmt.Sprintf(format, args...)}
}

// NormalizeNodeType 归一化协议类型（别名 → 内核类型名）。返回空串表示内核不支持。
func NormalizeNodeType(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	if alias, ok := nodeTypeAliases[t]; ok {
		t = alias
	}
	if !mihomoSupportedNodeTypes[t] {
		return ""
	}
	return t
}

// IsMihomoSupportedType 该类型是否在内核协议白名单内
func IsMihomoSupportedType(t string) bool {
	return NormalizeNodeType(t) != ""
}

// IsClashInfoNode 判定是否为面板注入的占位提示节点（不参与校验）
func isPlaceholderInfoNode(n *ProxyNode) bool {
	return n != nil && n.Type == "ss" && n.Server == "baidu.com" && n.Port == 1234
}

// ValidateProxyNode 第一层校验：单个节点是否可安全下发给内核。
// 返回 nil 表示合法；否则返回 *NodeValidationError，调用方据此丢弃并记录原因。
func ValidateProxyNode(n *ProxyNode) error {
	if n == nil {
		return newValidationError(ReasonUnsupportedType, "空节点")
	}
	if isPlaceholderInfoNode(n) {
		return nil // 面板自己的提示节点，固定 ss/baidu.com，永远是安全的
	}

	nt := NormalizeNodeType(n.Type)
	if nt == "" {
		return newValidationError(ReasonUnsupportedType, "mihomo 不支持的协议类型 %q", n.Type)
	}

	if strings.TrimSpace(n.Name) == "" {
		return newValidationError(ReasonEmptyName, "节点名为空（内核同名会静默覆盖）")
	}
	if mihomoReservedProxyNames[n.Name] {
		return newValidationError(ReasonReservedName,
			"节点名 %q 与内核内置代理重名（内核实测报 is the duplicate name），会让整份配置失效", n.Name)
	}

	if err := validateServerHost(n.Server); err != nil {
		return err
	}
	if n.Port < 1 || n.Port > 65535 {
		return newValidationError(ReasonInvalidPort, "端口 %d 不在 1..65535", n.Port)
	}

	switch nt {
	case "ss":
		return validateSSNode(n)
	case "ssr":
		return validateSSRNode(n)
	case "vmess":
		return validateVMessNode(n)
	case "vless":
		return validateVLESSNode(n)
	case "trojan":
		if strings.TrimSpace(n.Password) == "" {
			return newValidationError(ReasonMissingField, "trojan 缺少 password")
		}
	case "tuic":
		if strings.TrimSpace(n.UUID) == "" && strings.TrimSpace(n.Password) == "" {
			return newValidationError(ReasonMissingField, "tuic 缺少 uuid/password")
		}
	case "anytls":
		if strings.TrimSpace(n.Password) == "" {
			return newValidationError(ReasonMissingField, "anytls 缺少 password")
		}
	case "hysteria2":
		// 内核实测：obfs 只接受 salamander（或留空），其它取值 → "unknown obfs type"
		if obfs := strings.ToLower(strings.TrimSpace(optString(n.Options, "obfs"))); !hysteria2ObfsWhitelist[obfs] {
			return newValidationError(ReasonObfsInvalid, "hysteria2 obfs %q 不被内核支持（只接受 salamander）", obfs)
		}
	case "wireguard":
		// 内核实测：private-key 非空但不是合法 base64 → "decode private key"
		if pk := strings.TrimSpace(optString(n.Options, "private-key")); pk != "" {
			if _, err := base64.StdEncoding.DecodeString(pk); err != nil {
				return newValidationError(ReasonWireGuardKey, "wireguard private-key 不是合法 base64（内核报 decode private key）")
			}
		}
	case "hysteria", "snell", "ssh", "mieru", "socks5", "http":
		// 内核实测这些类型在缺省参数下仍能通过 -t（运行期才失败）；
		// 仅拦截必然导致配置失效的取值，不做过度校验以免误杀。
	}
	return nil
}

func validateServerHost(server string) error {
	s := strings.TrimSpace(server)
	if s == "" {
		return newValidationError(ReasonInvalidServer, "server 为空")
	}
	// 去掉 IPv6 字面量的中括号
	host := strings.Trim(s, "[]")
	if net.ParseIP(host) != nil {
		return nil
	}
	if len(host) > 253 || strings.ContainsAny(host, " \t/\\") {
		return newValidationError(ReasonInvalidServer, "server %q 不是合法域名或 IP", server)
	}
	if !strings.Contains(host, ".") {
		return newValidationError(ReasonInvalidServer, "server %q 不是合法域名或 IP", server)
	}
	if !domainRe.MatchString(host) {
		return newValidationError(ReasonInvalidServer, "server %q 不是合法域名或 IP", server)
	}
	return nil
}

// decodeSS2022KeyPart 校验 2022-blake3 密钥片段：必须是合法 base64 且解码长度精确匹配
func decodeSS2022KeyPart(part string, wantLen int) (int, error) {
	decoded, err := base64.StdEncoding.DecodeString(part)
	if err != nil {
		return 0, fmt.Errorf("非法 base64")
	}
	if len(decoded) != wantLen {
		return len(decoded), fmt.Errorf("长度 %d != 要求的 %d", len(decoded), wantLen)
	}
	return len(decoded), nil
}

// ssKeyValid 判断 2022-blake3 密钥是否完全合法
func ssKeyValid(password string, wantLen int) bool {
	code, _ := ssKeyCheck(password, wantLen)
	return code == ""
}

// ssKeyCheck 校验 2022-blake3 密钥，返回精确原因码与说明（"" 表示合法）。
//
// 形态以真内核实测为准（mihomo Meta v1.19.32）：
//   - 单段：password 为一段 base64，解码长度必须精确等于 wantLen
//   - 两段（多用户）：password 形如 "serverKey:userKey[:userKey2...]"，**每一段**都必须
//     合法 base64 且解码长度精确等于 wantLen
//
// 实测矩阵（cipher=2022-blake3-aes-128-gcm，wantLen=16）：
//
//	"k16"            PASS
//	"k16:k16"        PASS   ← 两段式内核接受
//	"k32"            FAIL   bad key length, required 16
//	"k32:k16"        FAIL   bad key length
//	非 base64        FAIL   decode key: illegal base64 data at input byte N
func ssKeyCheck(password string, wantLen int) (code, detail string) {
	parts := strings.Split(password, ":")
	if len(parts) == 0 || strings.TrimSpace(password) == "" {
		return ReasonCipherKeyNotB64, "密钥为空"
	}
	for i, p := range parts {
		decoded, err := base64.StdEncoding.DecodeString(p)
		if err != nil {
			return ReasonCipherKeyNotB64, fmt.Sprintf("第 %d 段不是合法 base64（内核报 decode key: illegal base64 data）", i+1)
		}
		if len(decoded) != wantLen {
			return ReasonCipherKeyMismatch, fmt.Sprintf("第 %d 段解码后 %d 字节，内核要求 %d 字节（bad key length）", i+1, len(decoded), wantLen)
		}
	}
	return "", ""
}

// NormalizeCredentials 凭据规范化统一入口：在【写库前】与【出配置前】各跑一次，
// 因此旧库里未解码的历史数据无需回填迁移，出配置时即可自动恢复可用。
//
// 覆盖范围与"为什么只对这些字段做"：
//   - ss 且 cipher 为 2022-blake3-*：password 被内核当"密钥"用，有明确合法性判据
//     （base64 + 精确长度），因此可以**证明**解码后更优 → 安全地自动修正。
//   - ss / ssr 其它 cipher：password 在内核眼里是**任意字符串**（实测
//     aes-128-gcm + "pw" PASS、chacha20-ietf-poly1305 + 面板口令 PASS），既不会让配置失效，
//     也没有任何判据能区分"字面 % 号"与"被 URL 编码的 %"——盲解码会把本可用的口令改坏
//     （例如 password 本身就是 "a%20b"）。这类凭据只在**解析阶段**（值确凿来自 URL userinfo）
//     做一次百分号解码，见 node_parser.go 的 extractSSAuth / parseSSR。
//
// 返回是否发生了修正。
func NormalizeCredentials(n *ProxyNode) (bool, string) {
	if n == nil {
		return false, ""
	}
	switch NormalizeNodeType(n.Type) {
	case "ss":
		return normalizeSS2022Key(n)
	case "ssr":
		// ssr 密码没有长度判据，不做盲解码；其 base64 主体已在解析阶段处理
		return false, ""
	}
	return false, ""
}

// NormalizeSS2022Key 兼容入口：等价于对单节点的凭据规范化
func NormalizeSS2022Key(n *ProxyNode) (bool, string) { return NormalizeCredentials(n) }

// normalizeSS2022Key 修正 2022-blake3 密钥的 URL 编码污染。
//
// 现网真实故障：订阅里的 ss 节点 password 被 URL 编码成
// "XD8...%2FQ=%3Alll..."（'/'→%2F，':'→%3A），内核报
// "decode key: illegal base64 data at input byte 41" 并让**整份 583 节点的配置失效**。
// 实测 URL 解码后密钥合法、`mihomo -t` 通过，所以这里做「修正」而不是「丢弃」。
//
// 返回 (是否修正, 说明)。
func normalizeSS2022Key(n *ProxyNode) (bool, string) {
	if n == nil {
		return false, ""
	}
	cipher := strings.ToLower(strings.TrimSpace(n.Cipher))
	wantLen, is2022 := ss2022RequiredKeyLen[cipher]
	if !is2022 {
		return false, ""
	}
	pw := n.Password
	if strings.TrimSpace(pw) == "" || ssKeyValid(pw, wantLen) {
		return false, ""
	}
	// 只有真的含百分号转义才需要还原，避免误改正常密钥
	if !strings.Contains(pw, "%") {
		return false, ""
	}
	// 必须用 PathUnescape（纯百分号解码）：QueryUnescape 会把 base64 中合法的 '+' 解码成空格，
	// 反而把密钥改坏——这正是本函数要在"还原"这一步避免的坑。
	// 仅当 PathUnescape 仍不合法时，才回退尝试 QueryUnescape 语义。
	candidates := []string{}
	if dec, err := url.PathUnescape(pw); err == nil && dec != pw {
		candidates = append(candidates, dec)
	}
	if dec, err := url.QueryUnescape(pw); err == nil && dec != pw {
		candidates = append(candidates, dec)
	}
	for _, decoded := range candidates {
		if ssKeyValid(decoded, wantLen) {
			n.Password = decoded
			return true, fmt.Sprintf("%s 密钥含 URL 编码（%%2F/%%3A 等），已还原为合法 base64（%d 字节）", cipher, wantLen)
		}
	}
	return false, ""
}

func validateSSNode(n *ProxyNode) error {
	cipher := strings.ToLower(strings.TrimSpace(n.Cipher))
	if cipher == "" {
		return newValidationError(ReasonMissingField, "ss 缺少 cipher")
	}
	if !ssCipherWhitelist[cipher] {
		return newValidationError(ReasonUnsupportedCipher, "ss cipher %q 不在内核白名单内（内核会拒绝整份配置）", n.Cipher)
	}
	if strings.TrimSpace(n.Password) == "" {
		return newValidationError(ReasonMissingField, "ss 缺少 password（内核报 missing password）")
	}
	// 密钥形态校验只对 2022-blake3-* 生效。
	// 依据（内核实测）：其它 cipher 的 password 是任意字符串——例如
	//   {type: ss, cipher: aes-128-gcm, password: "pw"} → mihomo -t PASS
	// 若对所有 ss 密码都强制 base64，会把现网绝大多数正常节点（如
	// chacha20-ietf-poly1305 + "panel-pass-example!"）全部误杀。
	// 只有 2022-blake3-* 把 password 当"密钥"用，内核才会做 base64 + 长度校验，
	// 不合法时直接让**整份配置**失效（不是少一个节点），所以必须在这里拦死。
	if wantLen, is2022 := ss2022RequiredKeyLen[cipher]; is2022 {
		if code, detail := ssKeyCheck(n.Password, wantLen); code != "" {
			return newValidationError(code,
				"%s password 非法（%s）：%s —— 内核报 decode key / bad key length，会让整份配置 -t 失败、客户端起不来",
				cipher, ReasonCipherKeyInvalid, detail)
		}
	}
	return validateSSPlugin(n)
}

// validateSSPlugin 校验 ss plugin 名与必需参数
func validateSSPlugin(n *ProxyNode) error {
	raw, ok := n.Options["plugin"]
	if !ok || raw == nil {
		return nil
	}
	name := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", raw)))
	if name == "" {
		return nil
	}
	// 归一化别名（解析器可能保留 simple-obfs / obfs-local）
	normalized := name
	if name == "simple-obfs" || name == "obfs-local" {
		normalized = "obfs"
	}
	var spec *ssPluginSpec
	for i := range ssPluginSpecs {
		for _, alias := range ssPluginSpecs[i].Aliases {
			if alias == normalized {
				spec = &ssPluginSpecs[i]
				break
			}
		}
		if spec != nil {
			break
		}
	}
	if spec == nil {
		return newValidationError(ReasonPluginInvalid, "ss plugin %q 不是内核实现的插件（内核会忽略它，节点必然连不上）", name)
	}
	opts, _ := n.Options["plugin-opts"].(map[string]any)
	for _, req := range spec.Required {
		v, exists := opts[req]
		if !exists || strings.TrimSpace(fmt.Sprintf("%v", v)) == "" {
			return newValidationError(ReasonPluginInvalid, "ss plugin %q 缺少必需参数 %q（内核报 unset fields / mode error）", name, req)
		}
		if allowed, hasEnum := spec.Enum[req]; hasEnum {
			got := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", v)))
			valid := false
			for _, a := range allowed {
				if a == got {
					valid = true
					break
				}
			}
			if !valid {
				return newValidationError(ReasonPluginInvalid,
					"ss plugin %q 的 %s=%q 不是内核接受的取值（只接受 %s），内核会报 obfs mode error",
					name, req, got, strings.Join(allowed, "/"))
			}
		}
	}
	return nil
}

func validateSSRNode(n *ProxyNode) error {
	cipher := strings.ToLower(strings.TrimSpace(n.Cipher))
	if cipher == "" {
		return newValidationError(ReasonMissingField, "ssr 缺少 cipher")
	}
	if !ssrCipherWhitelist[cipher] {
		return newValidationError(ReasonUnsupportedCipher, "ssr cipher %q 不是内核支持的流密码（ssr 不接受 AEAD）", n.Cipher)
	}
	if strings.TrimSpace(n.Password) == "" {
		return newValidationError(ReasonMissingField, "ssr 缺少 password")
	}
	protocol := strings.ToLower(strings.TrimSpace(optString(n.Options, "protocol")))
	if protocol == "" {
		return newValidationError(ReasonMissingField, "ssr 缺少 protocol")
	}
	if !ssrProtocolWhitelist[protocol] {
		return newValidationError(ReasonUnsupportedProtocol, "ssr protocol %q 不在内核枚举内", protocol)
	}
	obfs := strings.ToLower(strings.TrimSpace(optString(n.Options, "obfs")))
	if obfs == "" {
		return newValidationError(ReasonMissingField, "ssr 缺少 obfs")
	}
	if !ssrObfsWhitelist[obfs] {
		return newValidationError(ReasonUnsupportedObfs, "ssr obfs %q 不在内核枚举内", obfs)
	}
	return nil
}

func validateVMessNode(n *ProxyNode) error {
	if strings.TrimSpace(n.UUID) == "" {
		return newValidationError(ReasonMissingField, "vmess 缺少 uuid")
	}
	cipher := strings.ToLower(strings.TrimSpace(n.Cipher))
	if cipher == "" {
		cipher = "auto" // 内核拒绝空值，按默认值兜底（与解析器 getString(data,"scy","auto") 一致）
	}
	if !vmessCipherWhitelist[cipher] {
		return newValidationError(ReasonVMessCipher, "vmess cipher %q 不在内核白名单内（unsupported security type）", n.Cipher)
	}
	return nil
}

func validateVLESSNode(n *ProxyNode) error {
	// 内核实测 vless 的 uuid 不做格式校验（空/非 UUID 都能通过 -t），故只做非空检查以免误杀
	if strings.TrimSpace(n.UUID) == "" {
		return newValidationError(ReasonMissingField, "vless 缺少 uuid")
	}
	flow := strings.ToLower(strings.TrimSpace(optString(n.Options, "flow")))
	if flow != "" && vlessFlowBlocklist[flow] {
		return newValidationError(ReasonVLESSFlow, "vless flow %q 被内核拒绝（unsupported xtls flow type），会让整份配置失效", flow)
	}
	enc, hasEnc := n.Options["encryption"]
	if hasEnc {
		encStr := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", enc)))
		if !vlessEncryptionWhitelist[encStr] {
			return newValidationError(ReasonVLESSEncryption, "vless encryption %q 不被内核接受（invaild vless encryption value）", encStr)
		}
	}
	// reality-opts 出现时 public-key 必填。内核实测缺它 →
	// "'reality-opts' has unset fields: public-key" → 整份配置失效。
	// 解析器在 security=reality 但无 pbk、却带 sid/pqv/spx 时会生成这种残缺 reality-opts，
	// 是真实可触发路径（applyRealityOptions 只在 pbk 非空时才写 public-key）。
	if ro, ok := n.Options["reality-opts"]; ok && ro != nil {
		m, isMap := ro.(map[string]any)
		if !isMap {
			return newValidationError(ReasonRealityInvalid, "reality-opts 结构异常（应为键值对）")
		}
		if v, exists := m["public-key"]; !exists || v == nil || strings.TrimSpace(fmt.Sprintf("%v", v)) == "" {
			return newValidationError(ReasonRealityInvalid,
				"reality-opts 缺少 public-key（内核报 unset fields: public-key），会让整份配置失效")
		}
	}
	return nil
}

// optString 安全读取 Options 里的字符串值
func optString(opts map[string]any, key string) string {
	if opts == nil {
		return ""
	}
	v, ok := opts[key]
	if !ok || v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// nodeIdentity 节点身份标识（用于去重与坏节点缓存），不含凭据明文以外的敏感信息
func nodeIdentity(n *ProxyNode) string {
	if n == nil {
		return ""
	}
	return fmt.Sprintf("%s|%s|%s|%d", NormalizeNodeType(n.Type), n.Name, n.Server, n.Port)
}

// NodeValidationEvent 一条待落库的校验事件
type NodeValidationEvent struct {
	Event    string
	Source   string
	NodeName string
	NodeType string
	Server   string
	Port     int
	Reason   string
}

// NewNodeValidationEvent 由节点构造事件
func NewNodeValidationEvent(event, source string, n *ProxyNode, code, detail string) NodeValidationEvent {
	ev := NodeValidationEvent{Event: event, Source: source, Reason: code}
	if detail != "" {
		ev.Reason = code + ": " + detail
	}
	if n != nil {
		ev.NodeName, ev.NodeType = n.Name, NormalizeNodeType(n.Type)
		if ev.NodeType == "" {
			ev.NodeType = n.Type
		}
		ev.Server, ev.Port = n.Server, n.Port
	}
	return ev
}

// RecordNodeValidationEvents 批量落库（失败只打日志，绝不影响主流程）
func RecordNodeValidationEvents(db *gorm.DB, events []NodeValidationEvent) {
	if db == nil || len(events) == 0 {
		return
	}
	rows := make([]models.NodeValidationLog, 0, len(events))
	for _, e := range events {
		if len(e.NodeName) > 200 {
			e.NodeName = e.NodeName[:200]
		}
		if len(e.Server) > 200 {
			e.Server = e.Server[:200]
		}
		rows = append(rows, models.NodeValidationLog{
			Event: e.Event, Source: e.Source, NodeName: e.NodeName,
			NodeType: e.NodeType, Server: e.Server, Port: e.Port, Reason: e.Reason,
		})
	}
	if err := db.CreateInBatches(rows, 100).Error; err != nil {
		logf("节点校验日志写入失败: %v", err)
	}
}

// StaticValidateNodes 对一组节点跑第一层静态校验。
// 返回保留的节点与丢弃事件；同时会把可安全修正的节点就地修正（返回修正事件）。
//
// 说明：重名（会触发内核 "is the duplicate name" 而让整份配置失效）在本函数内统一
// 通过「重命名」处理而不是丢弃——重命名不会让用户少一个可用节点，且生成层
// （generateClashYAML）还有一层兜底去重。
func StaticValidateNodes(nodes []*ProxyNode, source string) (kept []*ProxyNode, events []NodeValidationEvent) {
	used := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		if n == nil {
			continue
		}
		if corrected, detail := NormalizeCredentials(n); corrected {
			events = append(events, NewNodeValidationEvent(models.NodeValidationCorrectedAtIngest, source, n, ReasonCredentialURLDecoded, detail))
		}
		if err := ValidateProxyNode(n); err != nil {
			ve, ok := err.(*NodeValidationError)
			if !ok {
				ve = &NodeValidationError{Code: ReasonKernelInvalid, Detail: err.Error()}
			}
			events = append(events, NewNodeValidationEvent(models.NodeValidationDroppedAtIngest, source, n, ve.Code, ve.Detail))
			continue
		}
		if used[n.Name] {
			orig := n.Name
			for i := 1; ; i++ {
				cand := fmt.Sprintf("%s-%d", orig, i)
				if !used[cand] {
					n.Name = cand
					break
				}
			}
			events = append(events, NewNodeValidationEvent(models.NodeValidationRenamedAtIngest, source, n, "duplicate-name",
				fmt.Sprintf("原始名 %q 与本次采集的其它节点重名，已重命名为 %q（内核同名会静默覆盖）", orig, n.Name)))
		}
		used[n.Name] = true
		kept = append(kept, n)
	}
	return kept, events
}

// logf 统一日志前缀（与本包其它日志一致打到 server.log）
func logf(format string, args ...any) {
	fmt.Printf("[node-validate] "+format+"\n", args...)
}

// knownUnsupportedSchemes 已知但 mihomo 不支持的链接协议。
// 这些链接会被 extractNodeLinks 提取出来（便于可见性），但解析阶段直接丢弃，
// 并记录 dropped_at_ingest 事件——不能让它们静默消失，否则运维无从发现订阅源退化。
var knownUnsupportedSchemes = []string{
	"naive+https://", "naive+quic://", "naive://",
	"juicity://", "brook://", "h2://", "http2://", "kcp://", "quic://", "mtproto://",
}

// knownUnsupportedScheme 返回该链接的已知不支持 scheme（不含 "://"，无则返回空串）
func knownUnsupportedScheme(link string) string {
	l := strings.ToLower(strings.TrimSpace(link))
	for _, s := range knownUnsupportedSchemes {
		if strings.HasPrefix(l, s) {
			return strings.TrimSuffix(s, "://")
		}
	}
	return ""
}

// linkDisplayName 从链接里安全提取节点名（#fragment），拿不到则返回占位串。
// 重要：绝不返回链接原文——链接里含 password/uuid，落库即等于凭据泄露。
func linkDisplayName(link string) string {
	idx := strings.Index(link, "#")
	if idx < 0 || idx+1 >= len(link) {
		return "(未解析节点)"
	}
	name := strings.TrimSpace(link[idx+1:])
	if decoded, err := url.QueryUnescape(name); err == nil {
		name = decoded
	}
	if name == "" {
		return "(未解析节点)"
	}
	if len(name) > 200 {
		name = name[:200]
	}
	return name
}

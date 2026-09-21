package node_health

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

type NodeHealthService struct {
	db          *gorm.DB
	httpClient  *http.Client
	testTimeout time.Duration
	maxLatency  int    // 最大允许延迟（毫秒），超过此值视为超时
	testURL     string // 测速URL，用于HTTP延迟测试（如 ping.pe）
}

// 节点探测结果状态。全站（后端判定、审计文案、前端映射）都以这组常量为准，
// 不要再散落字符串字面量，避免"数 active 而实际状态是 online"这类判定与统计错位。
const (
	StatusOnline  = "online"  // 握手成功且延迟在阈值内
	StatusTimeout = "timeout" // 握手成功但延迟超过阈值
	StatusOffline = "offline" // 握手失败（含配置错误、端口不可达）
	// StatusUnsupported 表示该协议无法用本服务的探测方式判断（只监听 UDP 的 QUIC 系协议）。
	// 既不能判在线也不能判离线，自动屏蔽逻辑必须跳过它，否则会误禁真实可用的节点。
	StatusUnsupported = "unsupported"
)

// udpOnlyProtocols 只监听 UDP/QUIC 的协议（键为小写）。
// 这些协议用 TCP 握手探测必然失败（实测 hysteria2/tuic 端口直接 Connection refused），
// 不能据此判定节点离线。
var udpOnlyProtocols = map[string]bool{
	"hysteria":  true,
	"hysteria2": true,
	"hy2":       true,
	"tuic":      true,
	"wireguard": true,
	"wg":        true,
}

// IsUDPOnlyProtocol 判断协议是否只监听 UDP（无法用 TCP 探测）。
func IsUDPOnlyProtocol(protocol string) bool {
	return udpOnlyProtocols[strings.ToLower(strings.TrimSpace(protocol))]
}

// ShouldAutoDisable 判断探测结果是否应触发"自动屏蔽失效节点"。
// 只有 timeout/offline 才代表节点确实不可用；unsupported 是"探测方式不适用"，必须放行。
func ShouldAutoDisable(status string) bool {
	return status == StatusTimeout || status == StatusOffline
}

// errNoLatencyInPage 表示测速页能打开但里面没有可解析的延迟数据
// （ping.pe 的延迟由前端 JS 渲染，服务端拿到的 HTML 中不含毫秒数）。
var errNoLatencyInPage = errors.New("网页未包含可解析的延迟数据")

// 网页测速熔断：一旦确认测速页解析不出延迟，就在冷却期内直接走 TCP，
// 避免每个节点都白等一次无意义的网页请求；管理员更换测速页后到期自动恢复。
const webTestCooldown = 30 * time.Minute

var webTestState struct {
	sync.Mutex
	disabledUntil time.Time
	reason        string
}

func webTestDisabled() (bool, string) {
	webTestState.Lock()
	defer webTestState.Unlock()
	if webTestState.disabledUntil.IsZero() || time.Now().After(webTestState.disabledUntil) {
		return false, ""
	}
	return true, webTestState.reason
}

func disableWebTest(reason string) {
	webTestState.Lock()
	defer webTestState.Unlock()
	webTestState.disabledUntil = time.Now().Add(webTestCooldown)
	webTestState.reason = reason
}

func enableWebTest() {
	webTestState.Lock()
	defer webTestState.Unlock()
	webTestState.disabledUntil = time.Time{}
	webTestState.reason = ""
}

func NewNodeHealthService() *NodeHealthService {
	service := &NodeHealthService{
		db:          database.GetDB(),
		httpClient:  &http.Client{Timeout: 30 * time.Second}, // 增加超时时间，因为需要等待网页响应
		testTimeout: 5 * time.Second,
		maxLatency:  3000,              // 默认3秒超时
		testURL:     "https://ping.pe", // 默认使用ping.pe
	}
	service.loadConfig()
	return service
}

func (s *NodeHealthService) loadConfig() {
	var configs []models.SystemConfig
	s.db.Where("category = ?", "node_health").Find(&configs)

	configMap := make(map[string]string)
	for _, config := range configs {
		configMap[config.Key] = config.Value
	}

	if testURL, ok := configMap["test_url"]; ok {
		// 允许显式保存空值以禁用网页测试（仅 TCP 探测）；
		// 只有未配置该项时才使用默认值。
		s.testURL = testURL
	}

	if maxLatencyStr, ok := configMap["node_max_latency"]; ok {
		if latency, err := strconv.Atoi(maxLatencyStr); err == nil {
			s.maxLatency = latency
		}
	}
	if testTimeoutStr, ok := configMap["node_test_timeout"]; ok {
		if timeout, err := strconv.Atoi(testTimeoutStr); err == nil {
			s.testTimeout = time.Duration(timeout) * time.Second
		}
	}
}

type TestResult struct {
	NodeID   uint      `json:"node_id"`
	Status   string    `json:"status"`  // StatusOnline / StatusTimeout / StatusOffline / StatusUnsupported
	Latency  int       `json:"latency"` // 延迟（毫秒）；offline 为 -1，unsupported 为 0（表示"未测到"，不是"很快"）
	Error    string    `json:"error,omitempty"`
	TestedAt time.Time `json:"tested_at"`
}

func (s *NodeHealthService) TestNode(node *models.Node) (*TestResult, error) {
	result := &TestResult{
		NodeID:   node.ID,
		TestedAt: utils.GetBeijingTime(),
	}

	if node.Config == nil || *node.Config == "" {
		result.Status = StatusOffline
		result.Error = "节点配置为空"
		return result, nil
	}

	var proxyNode config_update.ProxyNode
	if err := json.Unmarshal([]byte(*node.Config), &proxyNode); err != nil {
		result.Status = StatusOffline
		result.Error = "解析节点配置失败"
		return result, nil
	}

	// UDP-only 协议（hysteria2/tuic 等）只监听 UDP，没有 TCP 监听端口，TCP 握手必然失败。
	// 本探测方式对它们不适用，因此**按在线处理**（2026-09-21 业务决定：这类节点
	// 在客户端实测可用，服务端测不到不等于不可用，不应显示异常、更不能被自动屏蔽）。
	// 延迟保持 0，表示"未测到"而不是"很快"。
	if IsUDPOnlyProtocol(proxyNode.Type) {
		result.Status = StatusOnline
		result.Latency = 0
		return result, nil
	}

	latency, err := s.testConnection(&proxyNode)
	if err != nil {
		result.Status = StatusOffline
		result.Error = err.Error()
		result.Latency = -1
	} else if latency > s.maxLatency {
		result.Status = StatusTimeout
		result.Latency = latency
		result.Error = fmt.Sprintf("延迟超过限制: %dms", latency)
	} else {
		result.Status = StatusOnline
		result.Latency = latency
	}

	return result, nil
}

func (s *NodeHealthService) testConnection(node *config_update.ProxyNode) (int, error) {
	if s.testURL != "" {
		if disabled, _ := webTestDisabled(); disabled {
			// 冷却期内已知测速页拿不到延迟数据，直接 TCP，避免每个节点重复白等。
			return s.testTCPConnection(node.Server, node.Port)
		}
		latency, err := s.testViaWebPage(node)
		if err == nil {
			return latency, nil
		}
		// 网页测速不可用（页面结构变化、被墙、超时）时回退 TCP 握手，
		// 绝不再把"网页加载耗时"当作节点延迟上报。
		utils.LogWarn("节点网页测速失败，回退 TCP 探测: node=%s:%d err=%v", node.Server, node.Port, err)
		if errors.Is(err, errNoLatencyInPage) {
			disableWebTest(err.Error())
			utils.LogWarn("测速页解析不出延迟数据，%s 内直接改用 TCP 探测（更换或清空测速URL可立即生效）", webTestCooldown)
		}
	}

	return s.testTCPConnection(node.Server, node.Port)
}

func (s *NodeHealthService) testViaWebPage(node *config_update.ProxyNode) (int, error) {
	testAddress := fmt.Sprintf("%s:%d", node.Server, node.Port)

	// 只有 ping.pe 系页面支持把节点地址拼进 URL（https://ping.pe/<host:port>）；
	// 其它自定义测速页按配置原样请求并从响应里解析毫秒数，
	// 解析不到就返回错误交由 TCP 兜底（此前会静默改成请求 ping.pe）。
	target := s.testURL
	if strings.Contains(s.testURL, "ping.pe") {
		target = fmt.Sprintf("https://ping.pe/%s", url.QueryEscape(testAddress))
	}

	return s.fetchPageLatency(target)
}

// fetchPageLatency 请求测速页并解析其中的延迟；拿不到延迟即返回错误，
// 由调用方回退 TCP，绝不把请求耗时冒充成节点延迟。
func (s *NodeHealthService) fetchPageLatency(testURL string) (int, error) {
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return -1, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return -1, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return -1, fmt.Errorf("测速页返回状态码 %d", resp.StatusCode)
	}

	// 测速页通常只有几十 KB；限制读取上限，避免异常页面拖垮健康检查。
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return -1, fmt.Errorf("读取响应失败: %v", err)
	}

	latency, err := s.parsePingPeResponse(string(body))
	if err != nil {
		return -1, err
	}

	// 能解析出延迟说明测速页可用，解除熔断。
	enableWebTest()
	return latency, nil
}

func (s *NodeHealthService) parsePingPeResponse(html string) (int, error) {

	latencyPattern := regexp.MustCompile(`(\d+)\s*(?:ms|毫秒)`)
	matches := latencyPattern.FindAllStringSubmatch(html, -1)

	if len(matches) > 0 {
		chinaPattern := regexp.MustCompile(`(?i)(?:china|中国|cn|beijing|shanghai|guangzhou|shenzhen).*?(\d+)\s*(?:ms|毫秒)`)
		chinaMatches := chinaPattern.FindAllStringSubmatch(html, -1)

		if len(chinaMatches) > 0 {
			if latency, err := strconv.Atoi(chinaMatches[0][1]); err == nil {
				return latency, nil
			}
		}

		var latencies []int
		for _, match := range matches {
			if latency, err := strconv.Atoi(match[1]); err == nil && latency > 0 && latency < 10000 {
				latencies = append(latencies, latency)
			}
		}

		if len(latencies) > 0 {
			sum := 0
			for _, l := range latencies {
				sum += l
			}
			return sum / len(latencies), nil
		}

		if latency, err := strconv.Atoi(matches[0][1]); err == nil {
			return latency, nil
		}
	}

	jsonPattern := regexp.MustCompile(`"latency"\s*:\s*(\d+)`)
	jsonMatches := jsonPattern.FindStringSubmatch(html)
	if len(jsonMatches) > 1 {
		if latency, err := strconv.Atoi(jsonMatches[1]); err == nil {
			return latency, nil
		}
	}

	return -1, errNoLatencyInPage
}

func (s *NodeHealthService) testTCPConnection(host string, port int) (int, error) {
	address := net.JoinHostPort(host, strconv.Itoa(port))

	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, s.testTimeout)
	if err != nil {
		return -1, fmt.Errorf("连接失败: %v", err)
	}
	defer conn.Close()

	latency := int(time.Since(start).Milliseconds())
	return latency, nil
}

func (s *NodeHealthService) BatchTestNodes(nodeIDs []uint) ([]*TestResult, error) {
	var nodes []models.Node
	if err := s.db.Where("id IN ?", nodeIDs).Find(&nodes).Error; err != nil {
		return nil, err
	}

	results := make([]*TestResult, 0, len(nodes))
	var wg sync.WaitGroup
	var mu sync.Mutex

	semaphore := make(chan struct{}, 10) // 最多10个并发测试

	for _, node := range nodes {
		wg.Add(1)
		semaphore <- struct{}{} // 获取信号量

		go func(n models.Node) {
			defer wg.Done()
			defer func() { <-semaphore }() // 释放信号量

			result, err := s.TestNode(&n)
			if err != nil {
				result = &TestResult{
					NodeID:   n.ID,
					Status:   StatusOffline,
					Error:    err.Error(),
					TestedAt: utils.GetBeijingTime(),
				}
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(node)
	}

	wg.Wait()
	return results, nil
}

func (s *NodeHealthService) UpdateNodeStatus(result *TestResult) error {
	now := utils.GetBeijingTime()
	updates := map[string]interface{}{
		"status":     result.Status,
		"latency":    result.Latency,
		"last_test":  now,
		"updated_at": now,
	}

	// 自动屏蔽超时/离线节点开关（system_configs category=node_health key=auto_disable_timeout）
	// 开启时：检测到 timeout/offline 的节点自动置 is_active=false，用户订阅将不再包含失效节点。
	// unsupported（UDP-only 协议无法探测）不参与判定：既不禁用也不启用。
	if s.autoDisableTimeoutEnabled() {
		if ShouldAutoDisable(result.Status) {
			updates["is_active"] = false
		} else if result.Status == StatusOnline {
			updates["is_active"] = true
		}
	}

	return s.db.Model(&models.Node{}).Where("id = ?", result.NodeID).Updates(updates).Error
}

// autoDisableTimeoutEnabled 读取自动屏蔽超时节点开关（默认开启）。
func (s *NodeHealthService) autoDisableTimeoutEnabled() bool {
	var cfg models.SystemConfig
	if err := s.db.Where("key = ? AND category = ?", "auto_disable_timeout", "node_health").First(&cfg).Error; err == nil {
		return cfg.Value != "false" && cfg.Value != "0"
	}
	return true // 默认开启：保证用户订阅不到失效节点
}

// DisableTimeoutNodes 一键屏蔽所有状态为 timeout/offline 的节点（is_active=false）。
// 返回被屏蔽的节点数。selfHosted 为 true 时同时处理自建节点（自建节点状态由心跳维护，
// 若心跳超时已 offline，同样应屏蔽）。
func (s *NodeHealthService) DisableTimeoutNodes() (int, error) {
	res := s.db.Model(&models.Node{}).
		Where("is_active = ? AND status IN ?", true, []string{"timeout", "offline"}).
		Update("is_active", false)
	return int(res.RowsAffected), res.Error
}

// EnableAllNodes 一键启用所有节点（is_active=true，管理员手动恢复用）。
func (s *NodeHealthService) EnableAllNodes() (int, error) {
	// GORM 默认拒绝无 Where 的全表 Update（ErrMissingWhereClause），
	// 这里用恒真条件显式声明"全表"意图。
	res := s.db.Model(&models.Node{}).Where("1 = 1").Update("is_active", true)
	return int(res.RowsAffected), res.Error
}

func (s *NodeHealthService) CheckAllNodes() error {
	// 检查全部节点（含 is_active=false 的离线节点），
	// 否则节点一旦被置为离线就永远不会再被健康检查覆盖，无法自动恢复
	// 注意：自建节点（self_hosted=true）的在线状态由心跳机制独占维护，
	// 不参与自动健康检查，避免 TCP 探测覆盖心跳判定的 online/offline 状态。
	var nodes []models.Node
	if err := s.db.Where("self_hosted = ?", false).Find(&nodes).Error; err != nil {
		return err
	}

	batchSize := 50
	for i := 0; i < len(nodes); i += batchSize {
		end := i + batchSize
		if end > len(nodes) {
			end = len(nodes)
		}

		batch := nodes[i:end]
		nodeIDs := make([]uint, len(batch))
		for j, node := range batch {
			nodeIDs[j] = node.ID
		}

		results, err := s.BatchTestNodes(nodeIDs)
		if err != nil {
			utils.LogError("CheckAllNodes: batch test failed", err, map[string]interface{}{
				"batch_start": i,
				"batch_end":   end,
			})
			continue
		}

		for _, result := range results {
			if err := s.UpdateNodeStatus(result); err != nil {
				utils.LogError("CheckAllNodes: update node status failed", err, map[string]interface{}{
					"node_id": result.NodeID,
				})
			}
		}
	}

	return nil
}

func (s *NodeHealthService) SetMaxLatency(latency int) {
	s.maxLatency = latency
}

func (s *NodeHealthService) SetTestTimeout(timeout time.Duration) {
	s.testTimeout = timeout
}

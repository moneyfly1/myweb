package node_health

import (
	"context"
	"encoding/json"
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

// NodeHealthService 节点健康检查服务
type NodeHealthService struct {
	db          *gorm.DB
	httpClient  *http.Client
	testTimeout time.Duration
	maxLatency  int    // 最大允许延迟（毫秒），超过此值视为超时
	testURL     string // 测速URL，用于HTTP延迟测试（如 ping.pe）
}

// NewNodeHealthService 创建节点健康检查服务
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

// loadConfig 从数据库加载配置
func (s *NodeHealthService) loadConfig() {
	var configs []models.SystemConfig
	s.db.Where("category = ?", "node_health").Find(&configs)

	configMap := make(map[string]string)
	for _, config := range configs {
		configMap[config.Key] = config.Value
	}

	// 加载测试URL
	if testURL, ok := configMap["test_url"]; ok && testURL != "" {
		s.testURL = testURL
	}

	// 加载其他配置
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

// TestResult 测试结果
type TestResult struct {
	NodeID   uint      `json:"node_id"`
	Status   string    `json:"status"`  // online, offline, timeout
	Latency  int       `json:"latency"` // 延迟（毫秒）
	Error    string    `json:"error,omitempty"`
	TestedAt time.Time `json:"tested_at"`
}

// TestNode 测试单个节点
func (s *NodeHealthService) TestNode(node *models.Node) (*TestResult, error) {
	result := &TestResult{
		NodeID:   node.ID,
		TestedAt: utils.GetBeijingTime(),
	}

	// 解析节点配置
	if node.Config == nil || *node.Config == "" {
		result.Status = "offline"
		result.Error = "节点配置为空"
		return result, nil
	}

	var proxyNode config_update.ProxyNode
	if err := json.Unmarshal([]byte(*node.Config), &proxyNode); err != nil {
		result.Status = "offline"
		result.Error = "解析节点配置失败"
		return result, nil
	}

	// 测试节点连接
	latency, err := s.testConnection(&proxyNode)
	if err != nil {
		result.Status = "offline"
		result.Error = err.Error()
		result.Latency = -1
	} else if latency > s.maxLatency {
		result.Status = "timeout"
		result.Latency = latency
		result.Error = fmt.Sprintf("延迟超过限制: %dms", latency)
	} else {
		result.Status = "online"
		result.Latency = latency
	}

	return result, nil
}

// testConnection 测试节点连接
func (s *NodeHealthService) testConnection(node *config_update.ProxyNode) (int, error) {
	// 如果配置了测试URL（如ping.pe），使用网页测试
	if s.testURL != "" {
		latency, err := s.testViaWebPage(node)
		if err == nil {
			return latency, nil
		}
		// 如果网页测试失败，回退到TCP测试
		utils.LogError("网页测试失败，回退到TCP测试", err, map[string]interface{}{
			"node_server": node.Server,
			"node_port":   node.Port,
		})
	}

	// 默认使用TCP连接测试
	return s.testTCPConnection(node.Server, node.Port)
}

// testViaWebPage 通过网页（如ping.pe）测试节点延迟
func (s *NodeHealthService) testViaWebPage(node *config_update.ProxyNode) (int, error) {
	testAddress := fmt.Sprintf("%s:%d", node.Server, node.Port)

	// 根据测试URL选择不同的测试方式
	if strings.Contains(s.testURL, "ping.pe") {
		return s.testViaPingPe(testAddress)
	}

	// 默认尝试ping.pe格式
	return s.testViaPingPe(testAddress)
}

// testViaPingPe 通过ping.pe网页测试延迟
func (s *NodeHealthService) testViaPingPe(address string) (int, error) {
	// ping.pe的测试URL格式: https://ping.pe/{address}
	testURL := fmt.Sprintf("https://ping.pe/%s", url.QueryEscape(address))

	// 创建请求
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return -1, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置User-Agent，模拟浏览器
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	start := time.Now()
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return -1, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return -1, fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析HTML，查找延迟数据
	latency, err := s.parsePingPeResponse(string(body))
	if err != nil {
		// 如果解析失败，使用总耗时作为估算
		totalLatency := int(time.Since(start).Milliseconds())
		return totalLatency, nil
	}

	return latency, nil
}

// parsePingPeResponse 解析ping.pe网页响应，提取延迟数据
func (s *NodeHealthService) parsePingPeResponse(html string) (int, error) {
	// ping.pe页面中，延迟数据通常在特定的HTML元素中
	// 尝试多种方式解析：

	// 方式1: 查找包含"ms"或"毫秒"的数字
	// 正则表达式匹配延迟数据（例如: "50ms", "100 毫秒"等）
	latencyPattern := regexp.MustCompile(`(\d+)\s*(?:ms|毫秒)`)
	matches := latencyPattern.FindAllStringSubmatch(html, -1)

	if len(matches) > 0 {
		// 尝试找到中国的节点延迟（查找包含"China"、"中国"、"CN"等关键词附近的延迟）
		chinaPattern := regexp.MustCompile(`(?i)(?:china|中国|cn|beijing|shanghai|guangzhou|shenzhen).*?(\d+)\s*(?:ms|毫秒)`)
		chinaMatches := chinaPattern.FindAllStringSubmatch(html, -1)

		if len(chinaMatches) > 0 {
			// 使用第一个中国节点的延迟
			if latency, err := strconv.Atoi(chinaMatches[0][1]); err == nil {
				return latency, nil
			}
		}

		// 如果没有找到中国节点，使用所有延迟的平均值
		var latencies []int
		for _, match := range matches {
			if latency, err := strconv.Atoi(match[1]); err == nil && latency > 0 && latency < 10000 {
				latencies = append(latencies, latency)
			}
		}

		if len(latencies) > 0 {
			// 计算平均值
			sum := 0
			for _, l := range latencies {
				sum += l
			}
			return sum / len(latencies), nil
		}

		// 如果都没有，使用第一个匹配的延迟
		if latency, err := strconv.Atoi(matches[0][1]); err == nil {
			return latency, nil
		}
	}

	// 方式2: 查找JSON数据（如果页面包含JSON）
	jsonPattern := regexp.MustCompile(`"latency"\s*:\s*(\d+)`)
	jsonMatches := jsonPattern.FindStringSubmatch(html)
	if len(jsonMatches) > 1 {
		if latency, err := strconv.Atoi(jsonMatches[1]); err == nil {
			return latency, nil
		}
	}

	return -1, fmt.Errorf("无法从网页中解析延迟数据")
}

// testTCPConnection 测试TCP连接
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

// BatchTestNodes 批量测试节点
func (s *NodeHealthService) BatchTestNodes(nodeIDs []uint) ([]*TestResult, error) {
	var nodes []models.Node
	if err := s.db.Where("id IN ?", nodeIDs).Find(&nodes).Error; err != nil {
		return nil, err
	}

	results := make([]*TestResult, 0, len(nodes))
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 限制并发数，避免过多连接
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
					Status:   "offline",
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

// UpdateNodeStatus 更新节点状态
func (s *NodeHealthService) UpdateNodeStatus(result *TestResult) error {
	now := utils.GetBeijingTime()
	updates := map[string]interface{}{
		"status":     result.Status,
		"latency":    result.Latency,
		"last_test":  now,
		"updated_at": now,
	}

	// 如果节点超时或离线，自动禁用
	if result.Status == "timeout" || result.Status == "offline" {
		updates["is_active"] = false
	} else if result.Status == "online" {
		// 如果节点在线，确保启用
		updates["is_active"] = true
	}

	return s.db.Model(&models.Node{}).Where("id = ?", result.NodeID).Updates(updates).Error
}

// CheckAllNodes 检查所有节点
func (s *NodeHealthService) CheckAllNodes() error {
	var nodes []models.Node
	if err := s.db.Where("is_active = ?", true).Find(&nodes).Error; err != nil {
		return err
	}

	// 分批处理，避免一次性处理太多节点
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

		// 更新节点状态
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

// StartPeriodicCheck 启动定期检查
func (s *NodeHealthService) StartPeriodicCheck(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			utils.LogError("节点健康检查", nil, map[string]interface{}{
				"message": "开始执行节点健康检查",
			})
			if err := s.CheckAllNodes(); err != nil {
				utils.LogError("节点健康检查失败", err, nil)
			} else {
				utils.LogError("节点健康检查", nil, map[string]interface{}{
					"message": "节点健康检查完成",
				})
			}
		}
	}()
}

// GetMaxLatency 获取最大允许延迟
func (s *NodeHealthService) GetMaxLatency() int {
	return s.maxLatency
}

// SetMaxLatency 设置最大允许延迟
func (s *NodeHealthService) SetMaxLatency(latency int) {
	s.maxLatency = latency
}

// SetTestTimeout 设置测试超时时间
func (s *NodeHealthService) SetTestTimeout(timeout time.Duration) {
	s.testTimeout = timeout
}

// TestNodeWithContext 带上下文的节点测试（用于取消操作）
func (s *NodeHealthService) TestNodeWithContext(ctx context.Context, node *models.Node) (*TestResult, error) {
	resultChan := make(chan *TestResult, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := s.TestNode(node)
		if err != nil {
			errChan <- err
		} else {
			resultChan <- result
		}
	}()

	select {
	case <-ctx.Done():
		return &TestResult{
			NodeID:   node.ID,
			Status:   "offline",
			Error:    "测试超时",
			TestedAt: utils.GetBeijingTime(),
		}, ctx.Err()
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return &TestResult{
			NodeID:   node.ID,
			Status:   "offline",
			Error:    err.Error(),
			TestedAt: utils.GetBeijingTime(),
		}, err
	}
}

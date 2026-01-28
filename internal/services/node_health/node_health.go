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

type NodeHealthService struct {
	db          *gorm.DB
	httpClient  *http.Client
	testTimeout time.Duration
	maxLatency  int    // 最大允许延迟（毫秒），超过此值视为超时
	testURL     string // 测速URL，用于HTTP延迟测试（如 ping.pe）
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

	if testURL, ok := configMap["test_url"]; ok && testURL != "" {
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
	Status   string    `json:"status"`  // online, offline, timeout
	Latency  int       `json:"latency"` // 延迟（毫秒）
	Error    string    `json:"error,omitempty"`
	TestedAt time.Time `json:"tested_at"`
}

func (s *NodeHealthService) TestNode(node *models.Node) (*TestResult, error) {
	result := &TestResult{
		NodeID:   node.ID,
		TestedAt: utils.GetBeijingTime(),
	}

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

func (s *NodeHealthService) testConnection(node *config_update.ProxyNode) (int, error) {
	if s.testURL != "" {
		latency, err := s.testViaWebPage(node)
		if err == nil {
			return latency, nil
		}
		utils.LogError("网页测试失败，回退到TCP测试", err, map[string]interface{}{
			"node_server": node.Server,
			"node_port":   node.Port,
		})
	}

	return s.testTCPConnection(node.Server, node.Port)
}

func (s *NodeHealthService) testViaWebPage(node *config_update.ProxyNode) (int, error) {
	testAddress := fmt.Sprintf("%s:%d", node.Server, node.Port)

	if strings.Contains(s.testURL, "ping.pe") {
		return s.testViaPingPe(testAddress)
	}

	return s.testViaPingPe(testAddress)
}

func (s *NodeHealthService) testViaPingPe(address string) (int, error) {
	testURL := fmt.Sprintf("https://ping.pe/%s", url.QueryEscape(address))

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return -1, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	start := time.Now()
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return -1, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return -1, fmt.Errorf("读取响应失败: %v", err)
	}

	latency, err := s.parsePingPeResponse(string(body))
	if err != nil {
		totalLatency := int(time.Since(start).Milliseconds())
		return totalLatency, nil
	}

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

	return -1, fmt.Errorf("无法从网页中解析延迟数据")
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

func (s *NodeHealthService) UpdateNodeStatus(result *TestResult) error {
	now := utils.GetBeijingTime()
	updates := map[string]interface{}{
		"status":     result.Status,
		"latency":    result.Latency,
		"last_test":  now,
		"updated_at": now,
	}

	if result.Status == "timeout" || result.Status == "offline" {
		updates["is_active"] = false
	} else if result.Status == "online" {
		updates["is_active"] = true
	}

	return s.db.Model(&models.Node{}).Where("id = ?", result.NodeID).Updates(updates).Error
}

func (s *NodeHealthService) CheckAllNodes() error {
	var nodes []models.Node
	if err := s.db.Where("is_active = ?", true).Find(&nodes).Error; err != nil {
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

func (s *NodeHealthService) GetMaxLatency() int {
	return s.maxLatency
}

func (s *NodeHealthService) SetMaxLatency(latency int) {
	s.maxLatency = latency
}

func (s *NodeHealthService) SetTestTimeout(timeout time.Duration) {
	s.testTimeout = timeout
}

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

package node_health

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"
)

func newTestService(testURL string) *NodeHealthService {
	return &NodeHealthService{
		httpClient:  &http.Client{Timeout: 3 * time.Second},
		testTimeout: time.Second,
		maxLatency:  3000,
		testURL:     testURL,
	}
}

func testNode(configJSON string) *models.Node {
	cfg := configJSON
	return &models.Node{ID: 1, Config: &cfg}
}

func nodeConfigJSON(protocol string, port int) string {
	return `{"Type":"` + protocol + `","Server":"127.0.0.1","Port":` + strconv.Itoa(port) + `}`
}

// listenTCP 起一个真实可连的本地 TCP 端口，返回端口号。
func listenTCP(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port
}

// 只监听 UDP 的协议必须被识别，否则会被 TCP 探测误判为离线。
func TestIsUDPOnlyProtocol(t *testing.T) {
	udpOnly := []string{"hysteria", "hysteria2", "HY2", " tuic ", "wireguard", "wg"}
	for _, p := range udpOnly {
		if !IsUDPOnlyProtocol(p) {
			t.Errorf("IsUDPOnlyProtocol(%q) = false, want true", p)
		}
	}

	tcpBased := []string{"vless", "vmess", "trojan", "ss", "anytls", "naive", "", "VLESS"}
	for _, p := range tcpBased {
		if IsUDPOnlyProtocol(p) {
			t.Errorf("IsUDPOnlyProtocol(%q) = true, want false", p)
		}
	}
}

// unsupported 不能触发自动屏蔽（否则真实可用的 UDP 节点会被禁用）。
func TestShouldAutoDisable(t *testing.T) {
	cases := map[string]bool{
		StatusTimeout:     true,
		StatusOffline:     true,
		StatusOnline:      false,
		StatusUnsupported: false,
		"error":           false,
		"":                false,
	}
	for status, want := range cases {
		if got := ShouldAutoDisable(status); got != want {
			t.Errorf("ShouldAutoDisable(%q) = %v, want %v", status, got, want)
		}
	}
}

// UDP 协议（hysteria2/tuic）按在线处理：TCP 探测不适用于它们，
// 不能因此报离线或"无法探测"，更不能被自动屏蔽（2026-09-21 业务决定）。
func TestTestNodeUDPProtocolTreatedAsOnline(t *testing.T) {
	port := listenTCP(t)
	svc := newTestService("")

	for _, protocol := range []string{"hysteria2", "tuic", "wireguard"} {
		res, err := svc.TestNode(testNode(nodeConfigJSON(protocol, port)))
		if err != nil {
			t.Fatalf("TestNode(%s): %v", protocol, err)
		}
		if res.Status != StatusOnline {
			t.Fatalf("%s status = %q, want %q", protocol, res.Status, StatusOnline)
		}
		if res.Latency != 0 {
			t.Errorf("%s latency = %d, want 0（未测到不能伪装成很快）", protocol, res.Latency)
		}
		if res.Error != "" {
			t.Errorf("%s error = %q, 不应显示异常提示", protocol, res.Error)
		}
		if ShouldAutoDisable(res.Status) {
			t.Errorf("%s 被判为可自动屏蔽，UDP 节点会被误禁", protocol)
		}
	}
}

func TestTestNodeInvalidConfigIsOffline(t *testing.T) {
	svc := newTestService("")

	res, err := svc.TestNode(&models.Node{ID: 1})
	if err != nil {
		t.Fatalf("TestNode(nil config): %v", err)
	}
	if res.Status != StatusOffline {
		t.Errorf("空配置 status = %q, want %q", res.Status, StatusOffline)
	}

	res, err = svc.TestNode(testNode("{not json"))
	if err != nil {
		t.Fatalf("TestNode(bad json): %v", err)
	}
	if res.Status != StatusOffline {
		t.Errorf("非法配置 status = %q, want %q", res.Status, StatusOffline)
	}
}

// TCP 可达时判定在线，延迟来自真实握手。
func TestTestNodeTCPReachableIsOnline(t *testing.T) {
	port := listenTCP(t)
	svc := newTestService("")

	res, err := svc.TestNode(testNode(nodeConfigJSON("vless", port)))
	if err != nil {
		t.Fatalf("TestNode: %v", err)
	}
	if res.Status != StatusOnline {
		t.Fatalf("status = %q (err=%q), want %q", res.Status, res.Error, StatusOnline)
	}
	if res.Latency < 0 || res.Latency > 3000 {
		t.Errorf("latency = %d, 不在合理范围", res.Latency)
	}
}

// 延迟超阈值应为 timeout（而不是离线）。
func TestTestNodeLatencyOverMaxIsTimeout(t *testing.T) {
	port := listenTCP(t)
	svc := newTestService("")
	svc.maxLatency = -1 // 任何非负延迟都视为超限

	res, err := svc.TestNode(testNode(nodeConfigJSON("vless", port)))
	if err != nil {
		t.Fatalf("TestNode: %v", err)
	}
	if res.Status != StatusTimeout {
		t.Fatalf("status = %q, want %q", res.Status, StatusTimeout)
	}
	if !ShouldAutoDisable(res.Status) {
		t.Error("timeout 应触发自动屏蔽")
	}
}

func TestParsePingPeResponse(t *testing.T) {
	svc := newTestService("")

	// 中国节点优先取中国地区的延迟
	if got, err := svc.parsePingPeResponse(`<tr><td>China</td><td>23 ms</td></tr><tr><td>USA</td><td>180 ms</td></tr>`); err != nil || got != 23 {
		t.Errorf("china match = (%d, %v), want (23, nil)", got, err)
	}

	// 没有中国行时取有效延迟的平均值
	if got, err := svc.parsePingPeResponse(`20ms 40ms`); err != nil || got != 30 {
		t.Errorf("average = (%d, %v), want (30, nil)", got, err)
	}

	// JSON 形式
	if got, err := svc.parsePingPeResponse(`{"latency": 55}`); err != nil || got != 55 {
		t.Errorf("json = (%d, %v), want (55, nil)", got, err)
	}

	// 没有延迟数据时必须报错（不能把页面耗时当延迟）
	for _, html := range []string{"", "<html><table class=pingtable></table></html>", "no numbers here"} {
		if _, err := svc.parsePingPeResponse(html); err == nil {
			t.Errorf("parsePingPeResponse(%q) 未报错，会把页面耗时当作节点延迟", html)
		} else if !errors.Is(err, errNoLatencyInPage) {
			t.Errorf("错误未包裹 errNoLatencyInPage: %v", err)
		}
	}
}

// 关键回归：测速页打开但无延迟数据时，必须返回错误（调用方回退 TCP），
// 绝不能把网页加载耗时（这里故意 sleep 400ms）当作节点延迟。
func TestFetchPageLatencyNoDataDoesNotReportPageTime(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(400 * time.Millisecond)
		_, _ = w.Write([]byte(`<html><table class="pingtable"></table><script src="/x.js"></script></html>`))
	}))
	defer srv.Close()

	svc := newTestService(srv.URL)
	latency, err := svc.fetchPageLatency(srv.URL)
	if err == nil {
		t.Fatalf("latency = %d, want error（页面无延迟数据）", latency)
	}
	if !errors.Is(err, errNoLatencyInPage) {
		t.Errorf("err = %v, want errNoLatencyInPage", err)
	}
}

func TestFetchPageLatencyParsesAndResetsBreaker(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<tr><td>China</td><td>18 ms</td></tr>`))
	}))
	defer srv.Close()

	disableWebTest("测试前置状态")
	defer enableWebTest()

	svc := newTestService(srv.URL)
	got, err := svc.fetchPageLatency(srv.URL)
	if err != nil {
		t.Fatalf("fetchPageLatency: %v", err)
	}
	if got != 18 {
		t.Errorf("latency = %d, want 18", got)
	}
	if disabled, _ := webTestDisabled(); disabled {
		t.Error("解析成功后应解除熔断")
	}
}

// 熔断生效后不再请求测速页（否则每个节点都要白等一次），到期/成功后恢复。
func TestWebTestBreakerSkipsPageRequests(t *testing.T) {
	enableWebTest()
	defer enableWebTest()

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(`<html></html>`)) // 无延迟数据
	}))
	defer srv.Close()

	svc := newTestService(srv.URL)
	node := &config_update.ProxyNode{Type: "vless", Server: "127.0.0.1", Port: listenTCP(t)}

	// 第一次：请求测速页 → 解析失败 → 触发熔断 → 回退 TCP（本地端口可达）
	if _, err := svc.testConnection(node); err != nil {
		t.Fatalf("testConnection: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("测速页请求次数 = %d, want 1", got)
	}
	if disabled, _ := webTestDisabled(); !disabled {
		t.Fatal("解析失败后应进入熔断状态")
	}

	// 熔断期内：不再请求测速页
	if _, err := svc.testConnection(node); err != nil {
		t.Fatalf("testConnection(冷却期): %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("冷却期内仍在请求测速页（hits=%d），每个节点都会白等一次", got)
	}

	// 解除后恢复请求
	enableWebTest()
	if _, err := svc.testConnection(node); err != nil {
		t.Fatalf("testConnection(恢复后): %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Errorf("恢复后测速页请求次数 = %d, want 2", got)
	}
}

// 自定义测速页（非 ping.pe）必须请求配置的地址，而不是被静默改成 ping.pe。
func TestTestViaWebPageUsesConfiguredURL(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`China 33 ms`))
	}))
	defer srv.Close()

	enableWebTest()
	defer enableWebTest()

	svc := newTestService(srv.URL + "/latency")
	got, err := svc.testViaWebPage(&config_update.ProxyNode{Type: "vless", Server: "example.com", Port: 443})
	if err != nil {
		t.Fatalf("testViaWebPage: %v", err)
	}
	if got != 33 {
		t.Errorf("latency = %d, want 33", got)
	}
	if path != "/latency" {
		t.Errorf("请求路径 = %q, want /latency（应使用配置的测速URL）", path)
	}
}

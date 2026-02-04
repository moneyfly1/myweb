package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type TestResult struct {
	TotalRequests    int64
	SuccessRequests  int64
	FailedRequests   int64
	TotalTime        time.Duration
	MinResponseTime  time.Duration
	MaxResponseTime  time.Duration
	AvgResponseTime  time.Duration
	ErrorMessages    []string
}

var (
	successCount int64
	failCount    int64
	totalTime    int64 // 总响应时间（纳秒）
	minTime      int64 = 999999999999
	maxTime      int64
	errors       []string
	mu           sync.Mutex
)

func main() {
	baseURL := "https://dy.moneyfly.top"
	
	fmt.Println("=========================================")
	fmt.Println("网站压力测试工具")
	fmt.Println("目标网站:", baseURL)
	fmt.Println("=========================================\n")

	// 测试配置
	configs := []struct {
		name        string
		concurrency int
		requests    int
		endpoint    string
		method      string
		body        map[string]interface{}
	}{
		{
			name:        "首页加载测试",
			concurrency: 10,
			requests:    100,
			endpoint:    "/",
			method:      "GET",
		},
		{
			name:        "登录页面测试",
			concurrency: 10,
			requests:    100,
			endpoint:    "/login",
			method:      "GET",
		},
		{
			name:        "注册页面测试",
			concurrency: 10,
			requests:    100,
			endpoint:    "/register",
			method:      "GET",
		},
		{
			name:        "公共设置API测试",
			concurrency: 20,
			requests:    200,
			endpoint:    "/api/v1/settings/public-settings",
			method:      "GET",
		},
		{
			name:        "登录API压力测试（错误请求）",
			concurrency: 50,
			requests:    500,
			endpoint:    "/api/v1/auth/login-json",
			method:      "POST",
			body: map[string]interface{}{
				"username": "test_user",
				"password": "wrong_password",
			},
		},
	}

	// 执行所有测试
	for _, config := range configs {
		fmt.Printf("\n【测试 %s】\n", config.name)
		fmt.Printf("并发数: %d, 总请求数: %d\n", config.concurrency, config.requests)
		fmt.Println("----------------------------------------")
		
		result := runStressTest(baseURL, config.endpoint, config.method, config.body, config.concurrency, config.requests)
		printResult(result)
		
		// 重置计数器
		atomic.StoreInt64(&successCount, 0)
		atomic.StoreInt64(&failCount, 0)
		atomic.StoreInt64(&totalTime, 0)
		atomic.StoreInt64(&minTime, 999999999999)
		atomic.StoreInt64(&maxTime, 0)
		mu.Lock()
		errors = []string{}
		mu.Unlock()
		
		// 测试间隔
		time.Sleep(2 * time.Second)
	}

	fmt.Println("\n=========================================")
	fmt.Println("所有测试完成！")
	fmt.Println("=========================================")
}

func runStressTest(baseURL, endpoint, method string, body map[string]interface{}, concurrency, totalRequests int) TestResult {
	startTime := time.Now()
	
	var wg sync.WaitGroup
	requestChan := make(chan int, totalRequests)
	
	// 填充请求通道
	for i := 0; i < totalRequests; i++ {
		requestChan <- i
	}
	close(requestChan)
	
	// 启动并发请求
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range requestChan {
				makeRequest(baseURL, endpoint, method, body)
			}
		}()
	}
	
	wg.Wait()
	
	endTime := time.Now()
	totalDuration := endTime.Sub(startTime)
	
	success := atomic.LoadInt64(&successCount)
	failed := atomic.LoadInt64(&failCount)
	totalRespTime := atomic.LoadInt64(&totalTime)
	
	var avgTime time.Duration
	if success > 0 {
		avgTime = time.Duration(totalRespTime / success)
	}
	
	minRespTime := time.Duration(atomic.LoadInt64(&minTime))
	maxRespTime := time.Duration(atomic.LoadInt64(&maxTime))
	
	mu.Lock()
	errorList := make([]string, len(errors))
	copy(errorList, errors)
	mu.Unlock()
	
	return TestResult{
		TotalRequests:   int64(totalRequests),
		SuccessRequests: success,
		FailedRequests:  failed,
		TotalTime:       totalDuration,
		MinResponseTime: minRespTime,
		MaxResponseTime: maxRespTime,
		AvgResponseTime: avgTime,
		ErrorMessages:   errorList,
	}
}

func makeRequest(baseURL, endpoint, method string, body map[string]interface{}) {
	url := baseURL + endpoint
	
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			recordError(fmt.Sprintf("JSON编码错误: %v", err))
			return
		}
		reqBody = bytes.NewBuffer(jsonData)
	}
	
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		recordError(fmt.Sprintf("创建请求失败: %v", err))
		atomic.AddInt64(&failCount, 1)
		return
	}
	
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)
	
	if err != nil {
		recordError(fmt.Sprintf("请求失败: %v", err))
		atomic.AddInt64(&failCount, 1)
		return
	}
	defer resp.Body.Close()
	
	// 读取响应体（至少读取一部分）
	io.CopyN(io.Discard, resp.Body, 1024)
	
	// 记录响应时间
	nanos := duration.Nanoseconds()
	atomic.AddInt64(&totalTime, nanos)
	
	// 更新最小/最大响应时间
	for {
		oldMin := atomic.LoadInt64(&minTime)
		if nanos >= oldMin {
			break
		}
		if atomic.CompareAndSwapInt64(&minTime, oldMin, nanos) {
			break
		}
	}
	
	for {
		oldMax := atomic.LoadInt64(&maxTime)
		if nanos <= oldMax {
			break
		}
		if atomic.CompareAndSwapInt64(&maxTime, oldMax, nanos) {
			break
		}
	}
	
	// 判断成功或失败
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		atomic.AddInt64(&successCount, 1)
	} else {
		atomic.AddInt64(&failCount, 1)
		recordError(fmt.Sprintf("HTTP %d: %s", resp.StatusCode, url))
	}
}

func recordError(msg string) {
	mu.Lock()
	defer mu.Unlock()
	if len(errors) < 10 { // 只记录前10个错误
		errors = append(errors, msg)
	}
}

func printResult(result TestResult) {
	fmt.Printf("总请求数: %d\n", result.TotalRequests)
	fmt.Printf("成功请求: %d (%.2f%%)\n", result.SuccessRequests, 
		float64(result.SuccessRequests)/float64(result.TotalRequests)*100)
	fmt.Printf("失败请求: %d (%.2f%%)\n", result.FailedRequests,
		float64(result.FailedRequests)/float64(result.TotalRequests)*100)
	fmt.Printf("总耗时: %v\n", result.TotalTime)
	fmt.Printf("QPS (每秒请求数): %.2f\n", 
		float64(result.TotalRequests)/result.TotalTime.Seconds())
	fmt.Printf("平均响应时间: %v\n", result.AvgResponseTime)
	fmt.Printf("最小响应时间: %v\n", result.MinResponseTime)
	fmt.Printf("最大响应时间: %v\n", result.MaxResponseTime)
	
	if len(result.ErrorMessages) > 0 {
		fmt.Printf("\n错误信息（前10条）:\n")
		for i, err := range result.ErrorMessages {
			fmt.Printf("  %d. %s\n", i+1, err)
		}
	}
}

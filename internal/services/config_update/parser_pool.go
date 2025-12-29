package config_update

import (
	"fmt"
	"sync"
	"time"
)

// ParseResult 解析结果
type ParseResult struct {
	Node *ProxyNode
	Err  error
	Link string
}

// ParserPool 解析器池（Worker Pool）
type ParserPool struct {
	workers int
	cache   *ParseCache
}

// NewParserPool 创建解析器池
func NewParserPool(workers int) *ParserPool {
	if workers <= 0 {
		workers = 10 // 默认10个worker
	}
	return &ParserPool{
		workers: workers,
		cache:   NewParseCache(),
	}
}

// ParseLinks 并发解析链接列表
func (p *ParserPool) ParseLinks(links []string) []ParseResult {
	if len(links) == 0 {
		return []ParseResult{}
	}

	// 为每次调用创建新的 channels（避免并发冲突）
	taskChan := make(chan string, len(links))
	resultChan := make(chan ParseResult, len(links))
	var wg sync.WaitGroup

	// 启动workers
	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for link := range taskChan {
				// 检查缓存
				if cached, ok := p.cache.Get(link); ok {
					resultChan <- ParseResult{
						Node: cached,
						Err:  nil,
						Link: link,
					}
					continue
				}

				// 解析节点
				node, err := ParseNodeLink(link)
				if err != nil {
					resultChan <- ParseResult{
						Node: nil,
						Err:  fmt.Errorf("解析失败 [链接: %s...]: %w", truncateLink(link, 50), err),
						Link: link,
					}
					continue
				}

				// 缓存结果
				p.cache.Set(link, node)

				resultChan <- ParseResult{
					Node: node,
					Err:  nil,
					Link: link,
				}
			}
		}()
	}

	// 发送任务
	go func() {
		defer close(taskChan)
		for _, link := range links {
			taskChan <- link
		}
	}()

	// 等待所有workers完成并关闭结果channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	results := make([]ParseResult, 0, len(links))
	for result := range resultChan {
		results = append(results, result)
	}

	return results
}

func truncateLink(link string, maxLen int) string {
	if len(link) > maxLen {
		return link[:maxLen] + "..."
	}
	return link
}

// ParseCache 解析结果缓存
type ParseCache struct {
	cache map[string]*ProxyNode
	mu    sync.RWMutex
	ttl   time.Duration
	times map[string]time.Time
}

// NewParseCache 创建解析缓存
func NewParseCache() *ParseCache {
	return &ParseCache{
		cache: make(map[string]*ProxyNode),
		times: make(map[string]time.Time),
		ttl:   5 * time.Minute, // 5分钟TTL
	}
}

// Get 获取缓存
func (c *ParseCache) Get(key string) (*ProxyNode, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	node, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	// 检查TTL
	if t, ok := c.times[key]; ok {
		if time.Since(t) > c.ttl {
			// 过期，异步清理
			go c.delete(key)
			return nil, false
		}
	}

	return node, true
}

// Set 设置缓存
func (c *ParseCache) Set(key string, node *ProxyNode) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = node
	c.times[key] = time.Now()

	// 定期清理过期项（简单实现）
	if len(c.cache) > 1000 {
		c.cleanup()
	}
}

// delete 删除缓存项
func (c *ParseCache) delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, key)
	delete(c.times, key)
}

// cleanup 清理过期项
func (c *ParseCache) cleanup() {
	now := time.Now()
	for key, t := range c.times {
		if now.Sub(t) > c.ttl {
			delete(c.cache, key)
			delete(c.times, key)
		}
	}
}

package config_update

import (
	"fmt"
	"sync"
	"time"
)

type ParseResult struct {
	Node *ProxyNode
	Err  error
	Link string
}

type ParserPool struct {
	workers int
	cache   *ParseCache
}

func NewParserPool(workers int) *ParserPool {
	if workers <= 0 {
		workers = 10 // 默认10个worker
	}
	return &ParserPool{
		workers: workers,
		cache:   NewParseCache(),
	}
}

func (p *ParserPool) ParseLinks(links []string) []ParseResult {
	if len(links) == 0 {
		return []ParseResult{}
	}

	taskChan := make(chan string, len(links))
	resultChan := make(chan ParseResult, len(links))
	var wg sync.WaitGroup

	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for link := range taskChan {
				if cached, ok := p.cache.Get(link); ok {
					resultChan <- ParseResult{
						Node: cached,
						Err:  nil,
						Link: link,
					}
					continue
				}

				node, err := ParseNodeLink(link)
				if err != nil {
					resultChan <- ParseResult{
						Node: nil,
						Err:  fmt.Errorf("解析失败 [链接: %s...]: %w", truncateLink(link, 50), err),
						Link: link,
					}
					continue
				}

				p.cache.Set(link, node)

				resultChan <- ParseResult{
					Node: node,
					Err:  nil,
					Link: link,
				}
			}
		}()
	}

	go func() {
		defer close(taskChan)
		for _, link := range links {
			taskChan <- link
		}
	}()

	go func() {
		wg.Wait()
		close(resultChan)
	}()

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

type ParseCache struct {
	cache map[string]*ProxyNode
	mu    sync.RWMutex
	ttl   time.Duration
	times map[string]time.Time
}

func NewParseCache() *ParseCache {
	return &ParseCache{
		cache: make(map[string]*ProxyNode),
		times: make(map[string]time.Time),
		ttl:   5 * time.Minute, // 5分钟TTL
	}
}

func (c *ParseCache) Get(key string) (*ProxyNode, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	node, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	if t, ok := c.times[key]; ok {
		if time.Since(t) > c.ttl {
			go c.delete(key)
			return nil, false
		}
	}

	return node, true
}

func (c *ParseCache) Set(key string, node *ProxyNode) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = node
	c.times[key] = time.Now()

	if len(c.cache) > 1000 {
		c.cleanup()
	}
}

func (c *ParseCache) delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, key)
	delete(c.times, key)
}

func (c *ParseCache) cleanup() {
	now := time.Now()
	for key, t := range c.times {
		if now.Sub(t) > c.ttl {
			delete(c.cache, key)
			delete(c.times, key)
		}
	}
}

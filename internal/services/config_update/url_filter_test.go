package config_update

import (
	"testing"
)

// TestExtractURLFilterFlags 验证过滤开关解析与缺省补齐。
func TestExtractURLFilterFlags(t *testing.T) {
	var s ConfigUpdateService

	// 1. []string 形态
	flags := s.extractURLFilterFlags(map[string]interface{}{
		"url_filter_flags": []string{"1", "0", "1"},
	}, 3)
	if len(flags) != 3 || !flags[0] || flags[1] || !flags[2] {
		t.Fatalf("[]string 解析错误: %v", flags)
	}

	// 2. string 换行形态
	flags = s.extractURLFilterFlags(map[string]interface{}{
		"url_filter_flags": "1\n0\n1",
	}, 3)
	if len(flags) != 3 || !flags[0] || flags[1] || !flags[2] {
		t.Fatalf("string 解析错误: %v", flags)
	}

	// 3. 缺省（旧配置无 flags）→ 全部启用过滤
	flags = s.extractURLFilterFlags(map[string]interface{}{}, 4)
	for i, f := range flags {
		if !f {
			t.Fatalf("缺省应全部启用过滤，index %d 为 false", i)
		}
	}

	// 4. flags 短于 urls → 缺失部分默认启用
	flags = s.extractURLFilterFlags(map[string]interface{}{
		"url_filter_flags": []string{"0"},
	}, 3)
	if flags[0] || !flags[1] || !flags[2] {
		t.Fatalf("flags 短于 urls 补齐错误: %v", flags)
	}

	// 5. flags 长于 urls → 截断
	flags = s.extractURLFilterFlags(map[string]interface{}{
		"url_filter_flags": []string{"1", "0"},
	}, 1)
	if len(flags) != 1 || !flags[0] {
		t.Fatalf("flags 长于 urls 应截断: %v", flags)
	}
}

// TestProcessFetchedNodes_PerSourceFilter 验证按订阅源过滤：
// 源 A 启用过滤 → 命中关键词的节点被丢弃；源 B 不启用 → 全部保留。
func TestProcessFetchedNodes_PerSourceFilter(t *testing.T) {
	var s ConfigUpdateService
	s.parserPool = NewParserPool(2)

	urls := []string{"https://src-a.example.com/sub", "https://src-b.example.com/sub"}
	// 源 A：两个节点（广告节点A + 正常节点A）
	// 源 B：两个节点（广告节点B + 正常节点B）
	nodes := []map[string]interface{}{
		{"url": "vmess://eyJ2IjoiMiIsInBzIjoi5bm/5ZGK6IqC54K5QSIsImFkZCI6IjEuMi4zLjQiLCJwb3J0IjoiNDQzIiwiaWQiOiJhYWEtYmJiLWNjYyIsImFpZCI6IjAiLCJuZXQiOiJ3cyIsInR5cGUiOiJub25lIiwiaG9zdCI6IiIsInBhdGgiOiIvIiwidGxzIjoiIn0=", "source_url": urls[0]},
		{"url": "vmess://eyJ2IjoiMiIsInBzIjoi5q2j5bi46IqC54K5QSIsImFkZCI6IjEuMi4zLjUiLCJwb3J0IjoiNDQzIiwiaWQiOiJhYWEtYmJiLWNjYyIsImFpZCI6IjAiLCJuZXQiOiJ3cyIsInR5cGUiOiJub25lIiwiaG9zdCI6IiIsInBhdGgiOiIvIiwidGxzIjoiIn0=", "source_url": urls[0]},
		{"url": "vmess://eyJ2IjoiMiIsInBzIjoi5bm/5ZGK6IqC54K5QiIsImFkZCI6IjUuNi43LjgiLCJwb3J0IjoiNDQzIiwiaWQiOiJhYWEtYmJiLWNjYyIsImFpZCI6IjAiLCJuZXQiOiJ3cyIsInR5cGUiOiJub25lIiwiaG9zdCI6IiIsInBhdGgiOiIvIiwidGxzIjoiIn0=", "source_url": urls[1]},
		{"url": "vmess://eyJ2IjoiMiIsInBzIjoi5q2j5bi46IqC54K5QiIsImFkZCI6IjUuNi43LjkiLCJwb3J0IjoiNDQzIiwiaWQiOiJhYWEtYmJiLWNjYyIsImFpZCI6IjAiLCJuZXQiOiJ3cyIsInR5cGUiOiJub25lIiwiaG9zdCI6IiIsInBhdGgiOiIvIiwidGxzIjoiIn0=", "source_url": urls[1]},
	}

	// 关键词"广告"：源 A 启用过滤，源 B 不启用
	filterKeywords := []string{"广告"}
	urlFilterFlags := []bool{true, false}

	result, stats := s.processFetchedNodes(urls, nodes, filterKeywords, urlFilterFlags)

	// 源 A：2 个节点中 1 个被过滤 → 保留 1
	// 源 B：2 个节点都保留（不过滤）
	if len(result) != 3 {
		t.Fatalf("应保留 3 个节点（A 1 个 + B 2 个），实际 %d", len(result))
	}
	if stats.filtered != 1 {
		t.Fatalf("应过滤 1 个节点（仅源 A 的广告节点），实际 %d", stats.filtered)
	}

	// 验证保留的节点归属
	sourceAKept, sourceBKept := 0, 0
	for _, n := range result {
		if n.sourceIndex == 1 {
			sourceAKept++
		} else {
			sourceBKept++
		}
	}
	if sourceAKept != 1 || sourceBKept != 2 {
		t.Fatalf("源 A 应保留 1、源 B 应保留 2，实际 A=%d B=%d", sourceAKept, sourceBKept)
	}
}

// TestProcessFetchedNodes_AllFilter 验证全部源启用过滤时行为与旧版一致。
func TestProcessFetchedNodes_AllFilter(t *testing.T) {
	var s ConfigUpdateService
	s.parserPool = NewParserPool(2)

	urls := []string{"https://src-a.example.com/sub"}
	nodes := []map[string]interface{}{
		{"url": "vmess://eyJ2IjoiMiIsInBzIjoi5bm/5ZGK6IqC54K5QSIsImFkZCI6IjEuMi4zLjQiLCJwb3J0IjoiNDQzIiwiaWQiOiJhYWEtYmJiLWNjYyIsImFpZCI6IjAiLCJuZXQiOiJ3cyIsInR5cGUiOiJub25lIiwiaG9zdCI6IiIsInBhdGgiOiIvIiwidGxzIjoiIn0=", "source_url": urls[0]},
		{"url": "vmess://eyJ2IjoiMiIsInBzIjoi5q2j5bi46IqC54K5QSIsImFkZCI6IjEuMi4zLjUiLCJwb3J0IjoiNDQzIiwiaWQiOiJhYWEtYmJiLWNjYyIsImFpZCI6IjAiLCJuZXQiOiJ3cyIsInR5cGUiOiJub25lIiwiaG9zdCI6IiIsInBhdGgiOiIvIiwidGxzIjoiIn0=", "source_url": urls[0]},
	}
	// 默认 flags（全启用）→ 广告节点被过滤
	result, stats := s.processFetchedNodes(urls, nodes, []string{"广告"}, nil)
	if len(result) != 1 {
		t.Fatalf("全启用过滤应保留 1 个节点，实际 %d", len(result))
	}
	if stats.filtered != 1 {
		t.Fatalf("应过滤 1 个，实际 %d", stats.filtered)
	}

	// 无关键词 → 全部保留
	result, stats = s.processFetchedNodes(urls, nodes, nil, []bool{true, true})
	if len(result) != 2 {
		t.Fatalf("无关键词应保留全部 2 个，实际 %d", len(result))
	}
	if stats.filtered != 0 {
		t.Fatalf("无关键词不应过滤，实际 %d", stats.filtered)
	}
}

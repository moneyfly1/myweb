package config_update

import (
	"sort"
	"strings"
	"sync"
)

// RegionMatcher 地区匹配器（优化版本：使用排序的关键词列表）
type RegionMatcher struct {
	// 按长度排序的关键词列表（长关键词优先匹配）
	regionKeywords []keywordEntry
	serverMap      map[string]string
	mu             sync.RWMutex
}

type keywordEntry struct {
	keyword string
	region  string
	length  int
}

// NewRegionMatcher 创建地区匹配器
func NewRegionMatcher(regionMap map[string]string, serverMap map[string]string) *RegionMatcher {
	rm := &RegionMatcher{
		regionKeywords: make([]keywordEntry, 0, len(regionMap)),
		serverMap:      make(map[string]string, len(serverMap)),
	}

	// 构建排序的关键词列表（按长度降序，优先匹配长关键词）
	for keyword, region := range regionMap {
		rm.regionKeywords = append(rm.regionKeywords, keywordEntry{
			keyword: strings.ToUpper(keyword),
			region:  region,
			length:  len(keyword),
		})
	}
	
	// 按长度降序排序
	sort.Slice(rm.regionKeywords, func(i, j int) bool {
		return rm.regionKeywords[i].length > rm.regionKeywords[j].length
	})

	// 复制服务器映射（小写化）
	for kw, region := range serverMap {
		rm.serverMap[strings.ToLower(kw)] = region
	}

	return rm
}

// MatchRegion 匹配地区（优化版本：使用排序的关键词列表）
func (rm *RegionMatcher) MatchRegion(name, server string) string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	nameUpper := strings.ToUpper(name)

	// 使用排序的关键词列表（长关键词优先）
	for _, entry := range rm.regionKeywords {
		if strings.Contains(nameUpper, entry.keyword) {
			return entry.region
		}
	}

	// 服务器地址匹配（使用map，已经是小写）
	serverLower := strings.ToLower(server)
	for kw, region := range rm.serverMap {
		if strings.Contains(serverLower, kw) {
			return region
		}
	}

	return "未知"
}

// UpdateMaps 更新映射表（线程安全）
func (rm *RegionMatcher) UpdateMaps(regionMap, serverMap map[string]string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.regionKeywords = make([]keywordEntry, 0, len(regionMap))
	for keyword, region := range regionMap {
		rm.regionKeywords = append(rm.regionKeywords, keywordEntry{
			keyword: strings.ToUpper(keyword),
			region:  region,
			length:  len(keyword),
		})
	}

	sort.Slice(rm.regionKeywords, func(i, j int) bool {
		return rm.regionKeywords[i].length > rm.regionKeywords[j].length
	})

	rm.serverMap = make(map[string]string, len(serverMap))
	for kw, region := range serverMap {
		rm.serverMap[strings.ToLower(kw)] = region
	}
}


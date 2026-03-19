package config_update

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// ==========================================
// 地区配置加载
// ==========================================

var (
	regionConfigOnce sync.Once
	regionConfig     *RegionConfig
	regionConfigErr  error
)

type RegionConfig struct {
	RegionMap map[string]string `json:"region_map"`
	ServerMap map[string]string `json:"server_map"`
}

func LoadRegionConfig() (*RegionConfig, error) {
	regionConfigOnce.Do(func() {
		if len(builtInRegionMap) == 0 && len(builtInServerMap) == 0 {
			regionConfigErr = fmt.Errorf("地区配置资源为空")
			regionConfig = getDefaultRegionConfig()
			return
		}

		regionConfig = &RegionConfig{
			RegionMap: cloneStringMap(builtInRegionMap),
			ServerMap: cloneStringMap(builtInServerMap),
		}
	})

	if regionConfig == nil {
		return nil, fmt.Errorf("无法加载地区配置")
	}

	return regionConfig, regionConfigErr
}

func getDefaultRegionConfig() *RegionConfig {
	return &RegionConfig{
		RegionMap: make(map[string]string),
		ServerMap: make(map[string]string),
	}
}

func cloneStringMap(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// ==========================================
// 地区匹配器
// ==========================================

type RegionMatcher struct {
	regionKeywords []keywordEntry
	serverMap      map[string]string
	mu             sync.RWMutex
}

type keywordEntry struct {
	keyword string
	region  string
	length  int
}

func NewRegionMatcher(regionMap map[string]string, serverMap map[string]string) *RegionMatcher {
	rm := &RegionMatcher{
		regionKeywords: make([]keywordEntry, 0, len(regionMap)),
		serverMap:      make(map[string]string, len(serverMap)),
	}

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

	for kw, region := range serverMap {
		rm.serverMap[strings.ToLower(kw)] = region
	}

	return rm
}

func (rm *RegionMatcher) MatchRegion(name, server string) string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	nameUpper := strings.ToUpper(name)

	for _, entry := range rm.regionKeywords {
		if strings.Contains(nameUpper, entry.keyword) {
			return entry.region
		}
	}

	serverLower := strings.ToLower(server)
	for kw, region := range rm.serverMap {
		if strings.Contains(serverLower, kw) {
			return region
		}
	}

	return "未知"
}

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

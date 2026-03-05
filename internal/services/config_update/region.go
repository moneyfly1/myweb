package config_update

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
		wd, _ := os.Getwd()

		paths := []string{
			"./internal/services/config_update/region_config.json",
			"./region_config.json",
			filepath.Join(wd, "internal/services/config_update/region_config.json"),
			filepath.Join(wd, "region_config.json"),
			filepath.Join(filepath.Dir(os.Args[0]), "region_config.json"),
			filepath.Join(filepath.Dir(os.Args[0]), "internal/services/config_update/region_config.json"),
		}

		var lastErr error
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err == nil {
				var config RegionConfig
				if err := json.Unmarshal(data, &config); err == nil {
					if len(config.RegionMap) > 0 || len(config.ServerMap) > 0 {
						regionConfig = &config
						return
					}
					lastErr = fmt.Errorf("配置文件为空: %s", path)
				} else {
					lastErr = fmt.Errorf("JSON解析失败 %s: %v", path, err)
				}
			} else {
				lastErr = fmt.Errorf("文件读取失败 %s: %v", path, err)
			}
		}

		if lastErr != nil {
			regionConfigErr = fmt.Errorf("无法加载地区配置文件，尝试的路径都失败，最后错误: %v", lastErr)
		}
		regionConfig = getDefaultRegionConfig()
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

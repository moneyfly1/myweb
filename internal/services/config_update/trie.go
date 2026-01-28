package config_update

import (
	"sort"
	"strings"
	"sync"
)

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

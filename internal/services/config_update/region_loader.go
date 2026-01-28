package config_update

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

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

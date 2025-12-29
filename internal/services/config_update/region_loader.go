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

// RegionConfig 地区配置结构
type RegionConfig struct {
	RegionMap map[string]string `json:"region_map"`
	ServerMap map[string]string `json:"server_map"`
}

// LoadRegionConfig 加载地区配置（单例模式）
func LoadRegionConfig() (*RegionConfig, error) {
	regionConfigOnce.Do(func() {
		// 获取当前工作目录
		wd, _ := os.Getwd()

		// 尝试多个可能的路径
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
					// 验证配置不为空
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

		// 如果所有路径都失败，记录错误并使用空配置
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

// getDefaultRegionConfig 获取默认配置（向后兼容：返回空配置，实际使用时会从 region_config.json 加载）
func getDefaultRegionConfig() *RegionConfig {
	// 返回空配置，实际配置应该从 region_config.json 文件加载
	// 如果文件不存在，RegionMatcher 会使用空映射，返回"未知"地区
	return &RegionConfig{
		RegionMap: make(map[string]string),
		ServerMap: make(map[string]string),
	}
}

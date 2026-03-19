package config_update

import (
	"path/filepath"
)

const (
	defaultTargetDir = "./uploads/config"
	defaultV2RayFile = "xr"
	defaultClashFile = "clash.yaml"
	defaultTempFile  = "temp.yaml"
)

func ResolveOutputPaths(cfg map[string]interface{}) (targetDir, v2rayFile, clashFile string) {
	targetDir = defaultTargetDir
	if v, ok := cfg["target_dir"].(string); ok && v != "" {
		targetDir = v
	}
	targetDir = filepath.Clean(targetDir)

	v2rayFile = defaultV2RayFile
	if v, ok := cfg["v2ray_file"].(string); ok && v != "" {
		v2rayFile = filepath.Base(v)
	}

	clashFile = defaultClashFile
	if v, ok := cfg["clash_file"].(string); ok && v != "" {
		clashFile = filepath.Base(v)
	}

	return targetDir, v2rayFile, clashFile
}

func ResolveTemplatePath(cfg map[string]interface{}) string {
	targetDir, _, _ := ResolveOutputPaths(cfg)
	return filepath.Clean(filepath.Join(targetDir, defaultTempFile))
}

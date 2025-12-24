package handlers

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cboard-go/internal/core/config"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// CreateBackup 创建备份
func CreateBackup(c *gin.Context) {
	cfg := config.AppConfig

	// 创建备份目录（使用绝对路径，防止路径遍历）
	wd, err := os.Getwd()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取工作目录失败", err)
		return
	}

	// 构建备份目录路径
	backupDir := filepath.Join(wd, cfg.UploadDir, "backups")
	// 清理路径，防止路径遍历攻击
	backupDir = filepath.Clean(backupDir)

	// 验证路径是否在允许的目录内（防止路径遍历）
	if !strings.HasPrefix(backupDir, wd) {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的备份路径", nil)
		return
	}

	// 检查是否包含危险字符
	if strings.Contains(backupDir, "..") || strings.Contains(backupDir, "~") {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的备份路径", nil)
		return
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建备份目录失败", err)
		return
	}

	// 生成备份文件名（使用白名单验证，只允许字母、数字、下划线和连字符）
	backupFileName := fmt.Sprintf("backup_%s.zip", time.Now().Format("20060102_150405"))

	// 验证文件名（防止路径遍历）
	if strings.Contains(backupFileName, "..") || strings.Contains(backupFileName, "/") ||
		strings.Contains(backupFileName, "\\") || strings.Contains(backupFileName, "~") {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的文件名", nil)
		return
	}

	backupPath := filepath.Join(backupDir, backupFileName)
	// 再次清理和验证最终路径
	backupPath = filepath.Clean(backupPath)
	if !strings.HasPrefix(backupPath, backupDir) {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的备份路径", nil)
		return
	}

	// 创建 ZIP 文件
	zipFile, err := os.Create(backupPath)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建备份文件失败", err)
		return
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 备份数据库文件（使用绝对路径，防止路径遍历）
	dbPath := filepath.Join(wd, "cboard.db")
	dbPath = filepath.Clean(dbPath)
	// 验证路径在允许的目录内
	if strings.HasPrefix(dbPath, wd) && !strings.Contains(dbPath, "..") {
		if _, err := os.Stat(dbPath); err == nil {
			dbFile, err := os.Open(dbPath)
			if err == nil {
				defer dbFile.Close()

				// 使用filepath.Base确保只使用文件名，防止路径遍历
				writer, err := zipWriter.Create("cboard.db")
				if err == nil {
					io.Copy(writer, dbFile)
				}
			}
		}
	}

	// 备份配置文件（使用白名单，只允许特定文件）
	configFiles := []string{".env", "config.yaml"}
	for _, configFile := range configFiles {
		// 验证文件名（防止路径遍历）
		if strings.Contains(configFile, "..") || strings.Contains(configFile, "/") ||
			strings.Contains(configFile, "\\") || strings.Contains(configFile, "~") {
			continue // 跳过无效的文件名
		}

		configPath := filepath.Join(wd, configFile)
		configPath = filepath.Clean(configPath)
		// 验证路径在允许的目录内
		if strings.HasPrefix(configPath, wd) && !strings.Contains(configPath, "..") {
			if _, err := os.Stat(configPath); err == nil {
				file, err := os.Open(configPath)
				if err == nil {
					defer file.Close()

					// 使用filepath.Base确保只使用文件名
					writer, err := zipWriter.Create(filepath.Base(configFile))
					if err == nil {
						io.Copy(writer, file)
					}
				}
			}
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "备份创建成功", gin.H{
		"filename": backupFileName,
		"path":     backupPath,
		"size":     getFileSize(backupPath),
	})
}

// ListBackups 列出备份文件
func ListBackups(c *gin.Context) {
	cfg := config.AppConfig

	// 使用绝对路径，防止路径遍历
	wd, err := os.Getwd()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取工作目录失败", err)
		return
	}

	backupDir := filepath.Join(wd, cfg.UploadDir, "backups")
	// 清理路径，防止路径遍历攻击
	backupDir = filepath.Clean(backupDir)

	// 验证路径是否在允许的目录内
	if !strings.HasPrefix(backupDir, wd) || strings.Contains(backupDir, "..") || strings.Contains(backupDir, "~") {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的备份路径", nil)
		return
	}

	files, err := os.ReadDir(backupDir)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "读取备份目录失败", err)
		return
	}

	var backups []map[string]interface{}
	for _, file := range files {
		// 验证文件名（防止路径遍历）
		fileName := file.Name()
		if !file.IsDir() && filepath.Ext(fileName) == ".zip" {
			// 检查文件名是否包含危险字符
			if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") ||
				strings.Contains(fileName, "\\") || strings.Contains(fileName, "~") {
				continue // 跳过无效的文件名
			}

			info, err := file.Info()
			if err == nil {
				backups = append(backups, map[string]interface{}{
					"filename":   fileName, // 只返回文件名，不返回完整路径
					"size":       info.Size(),
					"created_at": info.ModTime().Format("2006-01-02 15:04:05"),
				})
			}
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", backups)
}

// getFileSize 获取文件大小
func getFileSize(filePath string) int64 {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0
	}
	return info.Size()
}

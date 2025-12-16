package handlers

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"cboard-go/internal/core/config"

	"github.com/gin-gonic/gin"
)

// CreateBackup 创建备份
func CreateBackup(c *gin.Context) {
	cfg := config.AppConfig

	// 创建备份目录
	backupDir := filepath.Join(cfg.UploadDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建备份目录失败",
		})
		return
	}

	// 生成备份文件名
	backupFileName := fmt.Sprintf("backup_%s.zip", time.Now().Format("20060102_150405"))
	backupPath := filepath.Join(backupDir, backupFileName)

	// 创建 ZIP 文件
	zipFile, err := os.Create(backupPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建备份文件失败",
		})
		return
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 备份数据库文件
	dbPath := "cboard.db"
	if _, err := os.Stat(dbPath); err == nil {
		dbFile, err := os.Open(dbPath)
		if err == nil {
			defer dbFile.Close()

			writer, err := zipWriter.Create("cboard.db")
			if err == nil {
				io.Copy(writer, dbFile)
			}
		}
	}

	// 备份配置文件
	configFiles := []string{".env", "config.yaml"}
	for _, configFile := range configFiles {
		if _, err := os.Stat(configFile); err == nil {
			file, err := os.Open(configFile)
			if err == nil {
				defer file.Close()

				writer, err := zipWriter.Create(configFile)
				if err == nil {
					io.Copy(writer, file)
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "备份创建成功",
		"data": gin.H{
			"filename": backupFileName,
			"path":     backupPath,
			"size":     getFileSize(backupPath),
		},
	})
}

// ListBackups 列出备份文件
func ListBackups(c *gin.Context) {
	cfg := config.AppConfig
	backupDir := filepath.Join(cfg.UploadDir, "backups")

	files, err := os.ReadDir(backupDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "读取备份目录失败",
		})
		return
	}

	var backups []map[string]interface{}
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".zip" {
			info, err := file.Info()
			if err == nil {
				backups = append(backups, map[string]interface{}{
					"filename":   file.Name(),
					"size":       info.Size(),
					"created_at": info.ModTime().Format("2006-01-02 15:04:05"),
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    backups,
	})
}

// getFileSize 获取文件大小
func getFileSize(filePath string) int64 {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0
	}
	return info.Size()
}

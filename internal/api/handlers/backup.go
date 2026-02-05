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
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/gitee"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

func CreateBackup(c *gin.Context) {
	cfg := config.AppConfig

	wd, err := os.Getwd()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取工作目录失败", err)
		return
	}

	backupDir := filepath.Join(wd, cfg.UploadDir, "backups")
	backupDir = filepath.Clean(backupDir)

	if !strings.HasPrefix(backupDir, wd) {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的备份路径", nil)
		return
	}

	if strings.Contains(backupDir, "..") || strings.Contains(backupDir, "~") {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的备份路径", nil)
		return
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建备份目录失败", err)
		return
	}

	isAutoBackup := c.Query("auto") == "true" || c.GetHeader("X-Auto-Backup") == "true"
	var backupFileName string
	if isAutoBackup {
		backupFileName = "backup_auto.zip"
	} else {
		backupFileName = fmt.Sprintf("backup_%s.zip", time.Now().Format("20060102_150405"))
	}

	if strings.Contains(backupFileName, "..") || strings.Contains(backupFileName, "/") ||
		strings.Contains(backupFileName, "\\") || strings.Contains(backupFileName, "~") {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的文件名", nil)
		return
	}

	backupPath := filepath.Join(backupDir, backupFileName)
	backupPath = filepath.Clean(backupPath)
	if !strings.HasPrefix(backupPath, backupDir) {
		utils.ErrorResponse(c, http.StatusBadRequest, "无效的备份路径", nil)
		return
	}

	if isAutoBackup {
		if _, err := os.Stat(backupPath); err == nil {
			if err := os.Remove(backupPath); err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, "删除旧备份文件失败", err)
				return
			}
		}
	}

	zipFile, err := os.Create(backupPath)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建备份文件失败", err)
		return
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	dbPath := filepath.Join(wd, "cboard.db")
	dbPath = filepath.Clean(dbPath)
	if strings.HasPrefix(dbPath, wd) && !strings.Contains(dbPath, "..") {
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
	}

	configFiles := []string{".env", "config.yaml"}
	for _, configFile := range configFiles {
		if strings.Contains(configFile, "..") || strings.Contains(configFile, "/") ||
			strings.Contains(configFile, "\\") || strings.Contains(configFile, "~") {
			continue
		}

		configPath := filepath.Join(wd, configFile)
		configPath = filepath.Clean(configPath)
		if strings.HasPrefix(configPath, wd) && !strings.Contains(configPath, "..") {
			if _, err := os.Stat(configPath); err == nil {
				file, err := os.Open(configPath)
				if err == nil {
					defer file.Close()

					writer, err := zipWriter.Create(filepath.Base(configFile))
					if err == nil {
						io.Copy(writer, file)
					}
				}
			}
		}
	}

	uploadResult := gin.H{
		"uploaded": false,
	}

	db := database.GetDB()
	var backupConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", "backup_gitee_enabled", "backup").First(&backupConfig).Error; err == nil {
		if backupConfig.Value == "true" {
			var tokenConfig models.SystemConfig
			if err := db.Where("key = ? AND category = ?", "backup_gitee_token", "backup").First(&tokenConfig).Error; err == nil && tokenConfig.Value != "" {
				var ownerConfig models.SystemConfig
				var repoConfig models.SystemConfig
				owner := "moneyfly"
				repo := "backup"
				if err := db.Where("key = ? AND category = ?", "backup_gitee_owner", "backup").First(&ownerConfig).Error; err == nil {
					owner = ownerConfig.Value
				}
				if err := db.Where("key = ? AND category = ?", "backup_gitee_repo", "backup").First(&repoConfig).Error; err == nil {
					repo = repoConfig.Value
				}

				// 创建只包含数据库的临时备份文件用于上传到Gitee
				giteeBackupFileName := fmt.Sprintf("backup_db_%s.zip", time.Now().Format("20060102_150405"))
				giteeBackupPath := filepath.Join(backupDir, giteeBackupFileName)
				giteeBackupPath = filepath.Clean(giteeBackupPath)

				if strings.HasPrefix(giteeBackupPath, backupDir) {
					giteeZipFile, err := os.Create(giteeBackupPath)
					if err == nil {
						giteeZipWriter := zip.NewWriter(giteeZipFile)

						// 只添加数据库文件到Gitee备份
						dbPath := filepath.Join(wd, "cboard.db")
						dbPath = filepath.Clean(dbPath)
						if strings.HasPrefix(dbPath, wd) && !strings.Contains(dbPath, "..") {
							if _, err := os.Stat(dbPath); err == nil {
								dbFile, err := os.Open(dbPath)
								if err == nil {
									writer, err := giteeZipWriter.Create("cboard.db")
									if err == nil {
										io.Copy(writer, dbFile)
									}
									dbFile.Close()
								}
							}
						}

						giteeZipWriter.Close()
						giteeZipFile.Close()

						// 上传只包含数据库的备份到Gitee
						client := gitee.NewGiteeClient(tokenConfig.Value, owner, repo)
						if err := client.UploadBackup(giteeBackupPath); err != nil {
							utils.LogError("上传备份到 Gitee 失败", err, nil)
							uploadResult["error"] = err.Error()
						} else {
							uploadResult["uploaded"] = true
							uploadResult["message"] = "已成功上传数据库备份到 Gitee（仅数据库文件）"
						}

						// 删除临时的Gitee备份文件
						os.Remove(giteeBackupPath)
					}
				}
			}
		}
	}

	var fileSize int64
	if _, err := os.Stat(backupPath); err == nil {
		fileSize = getFileSize(backupPath)
	}

	utils.SuccessResponse(c, http.StatusOK, "备份创建成功", gin.H{
		"filename": backupFileName,
		"path":     backupPath,
		"size":     fileSize,
		"gitee":    uploadResult,
	})
}

func ListBackups(c *gin.Context) {
	cfg := config.AppConfig

	wd, err := os.Getwd()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取工作目录失败", err)
		return
	}

	backupDir := filepath.Join(wd, cfg.UploadDir, "backups")
	backupDir = filepath.Clean(backupDir)

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
		fileName := file.Name()
		if !file.IsDir() && filepath.Ext(fileName) == ".zip" {
			if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") ||
				strings.Contains(fileName, "\\") || strings.Contains(fileName, "~") {
				continue
			}

			info, err := file.Info()
			if err == nil {
				backups = append(backups, map[string]interface{}{
					"filename":   fileName,
					"size":       info.Size(),
					"created_at": info.ModTime().Format("2006-01-02 15:04:05"),
				})
			}
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", backups)
}

func getFileSize(filePath string) int64 {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0
	}
	return info.Size()
}

func TestGiteeConnection(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
		Owner string `json:"owner"`
		Repo  string `json:"repo"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		db := database.GetDB()
		var tokenConfig models.SystemConfig
		if err := db.Where("key = ? AND category = ?", "backup_gitee_token", "backup").First(&tokenConfig).Error; err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "未配置 Gitee Token，请在请求中提供token或先保存设置", nil)
			return
		}
		req.Token = tokenConfig.Value

		var ownerConfig models.SystemConfig
		if err := db.Where("key = ? AND category = ?", "backup_gitee_owner", "backup").First(&ownerConfig).Error; err == nil {
			req.Owner = ownerConfig.Value
		} else {
			req.Owner = "moneyfly"
		}

		var repoConfig models.SystemConfig
		if err := db.Where("key = ? AND category = ?", "backup_gitee_repo", "backup").First(&repoConfig).Error; err == nil {
			req.Repo = repoConfig.Value
		} else {
			req.Repo = "backup"
		}
	}

	if req.Token == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Token不能为空", nil)
		return
	}

	if req.Owner == "" {
		req.Owner = "moneyfly"
	}
	if req.Repo == "" {
		req.Repo = "backup"
	}

	client := gitee.NewGiteeClient(req.Token, req.Owner, req.Repo)
	if err := client.TestConnection(); err != nil {
		utils.LogError("测试 Gitee 连接失败", err, nil)
		utils.ErrorResponse(c, http.StatusBadRequest, "Gitee 连接测试失败: "+err.Error(), err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Gitee 连接测试成功", gin.H{
		"owner":   req.Owner,
		"repo":    req.Repo,
		"message": "连接正常，可以正常上传文件",
	})
}

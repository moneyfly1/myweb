package handlers

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/git"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	if err := os.MkdirAll(backupDir, 0750); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建备份目录失败", err)
		return
	}

	isAutoBackup := c.Query("auto") == "true" || c.GetHeader("X-Auto-Backup") == "true"
	var backupFileName string
	if isAutoBackup {
		backupFileName = "backup_auto.zip"
	} else {
		backupFileName = fmt.Sprintf("backup_%s.zip", utils.GetBeijingTime().Format("20060102_150405"))
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
					_, _ = io.Copy(writer, dbFile) // Ignore copy error, best effort
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
						_, _ = io.Copy(writer, file) // Ignore copy error, best effort
					}
				}
			}
		}
	}

	uploadResult := gin.H{
		"uploaded": false,
		"async":    false,
	}

	db := database.GetDB()

	// 获取备份目标配置（gitee 或 github）
	var targetConfig models.SystemConfig
	backupTarget := "gitee" // 默认使用gitee
	if err := db.Where("key = ? AND category = ?", "backup_target", "backup").First(&targetConfig).Error; err == nil {
		if targetConfig.Value == "github" {
			backupTarget = "github"
		}
	}

	// 检查是否启用了远程备份
	var enabledKey string
	if backupTarget == "github" {
		enabledKey = "backup_github_enabled"
	} else {
		enabledKey = "backup_gitee_enabled"
	}

	var backupConfig models.SystemConfig
	if err := db.Where("key = ? AND category = ?", enabledKey, "backup").First(&backupConfig).Error; err == nil {
		if backupConfig.Value == "true" {
			var tokenConfig models.SystemConfig
			var tokenKey string
			if backupTarget == "github" {
				tokenKey = "backup_github_token"
			} else {
				tokenKey = "backup_gitee_token" // #nosec G101 - Config key name, not credential
			}

			if err := db.Where("key = ? AND category = ?", tokenKey, "backup").First(&tokenConfig).Error; err == nil && tokenConfig.Value != "" {
				var ownerConfig models.SystemConfig
				var repoConfig models.SystemConfig
				var ownerKey, repoKey string

				if backupTarget == "github" {
					ownerKey = "backup_github_owner"
					repoKey = "backup_github_repo"
				} else {
					ownerKey = "backup_gitee_owner"
					repoKey = "backup_gitee_repo"
				}

				owner := "moneyfly1"
				repo := "backup"
				if backupTarget == "github" {
					owner = "moneyfly1"
					repo = "backup"
				} else {
					owner = "moneyfly"
					repo = "backup"
				}

				if err := db.Where("key = ? AND category = ?", ownerKey, "backup").First(&ownerConfig).Error; err == nil {
					owner = ownerConfig.Value
				}
				if err := db.Where("key = ? AND category = ?", repoKey, "backup").First(&repoConfig).Error; err == nil {
					repo = repoConfig.Value
				}

				// 创建只包含数据库的临时备份文件
				backupFileName := fmt.Sprintf("backup_db_%s.zip", utils.GetBeijingTime().Format("20060102_150405"))
				backupFilePath := filepath.Join(backupDir, backupFileName)
				backupFilePath = filepath.Clean(backupFilePath)

				if strings.HasPrefix(backupFilePath, backupDir) {
					zipFile, err := os.Create(backupFilePath)
					if err == nil {
						zipWriter := zip.NewWriter(zipFile)

						// 只添加数据库文件
						dbPath := filepath.Join(wd, "cboard.db")
						dbPath = filepath.Clean(dbPath)
						if strings.HasPrefix(dbPath, wd) && !strings.Contains(dbPath, "..") {
							if _, err := os.Stat(dbPath); err == nil {
								dbFile, err := os.Open(dbPath)
								if err == nil {
									writer, err := zipWriter.Create("cboard.db")
									if err == nil {
										_, _ = io.Copy(writer, dbFile) // Ignore copy error, best effort
									}
									if err := dbFile.Close(); err != nil {
										log.Printf("failed to close db file: %v", err)
									}
								}
							}
						}

						if err := zipWriter.Close(); err != nil {
							log.Printf("failed to close zip writer: %v", err)
						}
						if err := zipFile.Close(); err != nil {
							log.Printf("failed to close zip file: %v", err)
						}

						// 获取文件大小
						var fileSize int64
						if fileInfo, err := os.Stat(backupFilePath); err == nil {
							fileSize = fileInfo.Size()
						}

						// 生成任务ID
						taskID := uuid.New().String()

						// 确定平台类型
						var platformType git.PlatformType
						var platformName string
						if backupTarget == "github" {
							platformType = git.PlatformGitHub
							platformName = "GitHub"
						} else {
							platformType = git.PlatformGitee
							platformName = "Gitee"
						}

						// 使用统一的状态管理器
						statusManager := git.GetUploadStatusManager()
						status := &git.UploadStatus{
							Status:    "uploading",
							Progress:  0,
							Message:   "正在准备上传...",
							StartTime: utils.GetBeijingTime(),
							FileName:  backupFileName,
							FileSize:  fileSize,
						}
						statusManager.SetStatus(taskID, status)

						// 异步上传（使用统一的Git客户端）
						go func() {
							client := git.NewClient(platformType, tokenConfig.Value, owner, repo)

							progressCallback := func(progress int, message string) {
								statusManager.UpdateStatus(taskID, "uploading", message, progress)
							}

							if err := client.UploadBackupWithProgress(backupFilePath, progressCallback); err != nil {
								utils.LogError(fmt.Sprintf("上传备份到 %s 失败", platformName), err, nil)
								statusManager.UpdateError(taskID, err)
							} else {
								statusManager.UpdateStatus(taskID, "success", fmt.Sprintf("已成功上传数据库备份到 %s（仅数据库文件）", platformName), 100)
								utils.LogInfo("数据库备份文件已成功上传到 %s（仅数据库文件）", platformName)
							}

							if err := os.Remove(backupFilePath); err != nil {
							log.Printf("failed to remove backup file: %v", err)
						}
						}()

						uploadResult["async"] = true
						uploadResult["task_id"] = taskID
						uploadResult["target"] = backupTarget
						uploadResult["message"] = fmt.Sprintf("备份文件已创建，正在后台上传到%s...", platformName)
					}
				}
			}
		}
	}

	var fileSize int64
	if _, err := os.Stat(backupPath); err == nil {
		fileSize = getFileSize(backupPath)
	}

	response := gin.H{
		"filename": backupFileName,
		"path":     backupPath,
		"size":     fileSize,
	}

	// 根据目标平台设置响应字段
	if backupTarget == "github" {
		response["github"] = uploadResult
	} else {
		response["gitee"] = uploadResult
	}
	utils.CreateAuditLogSimple(c, "create_backup", "backup", 0, fmt.Sprintf("管理员操作: 创建备份 %s", backupFileName))
	utils.SuccessResponse(c, http.StatusOK, "备份创建成功", response)
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
					"created_at": utils.FormatBeijingTime(info.ModTime()),
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

	client := git.NewClient(git.PlatformGitee, req.Token, req.Owner, req.Repo)
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

func TestGitHubConnection(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
		Owner string `json:"owner"`
		Repo  string `json:"repo"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		db := database.GetDB()
		var tokenConfig models.SystemConfig
		if err := db.Where("key = ? AND category = ?", "backup_github_token", "backup").First(&tokenConfig).Error; err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "未配置 GitHub Token，请在请求中提供token或先保存设置", nil)
			return
		}
		req.Token = tokenConfig.Value

		var ownerConfig models.SystemConfig
		if err := db.Where("key = ? AND category = ?", "backup_github_owner", "backup").First(&ownerConfig).Error; err == nil {
			req.Owner = ownerConfig.Value
		} else {
			req.Owner = "moneyfly1"
		}

		var repoConfig models.SystemConfig
		if err := db.Where("key = ? AND category = ?", "backup_github_repo", "backup").First(&repoConfig).Error; err == nil {
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
		req.Owner = "moneyfly1"
	}
	if req.Repo == "" {
		req.Repo = "backup"
	}

	client := git.NewClient(git.PlatformGitHub, req.Token, req.Owner, req.Repo)
	if err := client.TestConnection(); err != nil {
		utils.LogError("测试 GitHub 连接失败", err, nil)
		utils.ErrorResponse(c, http.StatusBadRequest, "GitHub 连接测试失败: "+err.Error(), err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "GitHub 连接测试成功", gin.H{
		"owner":   req.Owner,
		"repo":    req.Repo,
		"message": "连接正常，可以正常上传文件",
	})
}

// GetUploadStatus 获取上传状态
func GetUploadStatus(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "任务ID不能为空", nil)
		return
	}

	// 使用统一的状态管理器
	statusManager := git.GetUploadStatusManager()
	status, exists := statusManager.GetStatus(taskID)

	if !exists {
		utils.ErrorResponse(c, http.StatusNotFound, "未找到该上传任务", nil)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", status)
}

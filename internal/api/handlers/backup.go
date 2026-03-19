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
	"cboard-go/internal/services/backup_service"
	"cboard-go/internal/services/git"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreateBackup(c *gin.Context) {
	cfg := config.AppConfig

	// WAL checkpoint: 将 WAL 文件内容刷入主数据库文件
	if strings.Contains(cfg.DatabaseURL, "sqlite") {
		db := database.GetDB()
		db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	}

	wd, err := os.Getwd()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取工作目录失败", err)
		return
	}

	backupDir := filepath.Join(wd, cfg.UploadDir, "backups")
	backupDir = filepath.Clean(backupDir)
	if !utils.IsWithinBaseDir(wd, backupDir) {
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

	backupPath, ok := utils.JoinWithinBaseDir(backupDir, backupFileName)
	if !ok {
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
	defer func() {
		if closeErr := zipFile.Close(); closeErr != nil {
			log.Printf("failed to close backup zip file: %v", closeErr)
		}
	}()

	zipWriter := zip.NewWriter(zipFile)
	defer func() {
		if closeErr := zipWriter.Close(); closeErr != nil {
			log.Printf("failed to close zip writer: %v", closeErr)
		}
	}()

	dbPath, ok := utils.JoinWithinBaseDir(wd, "cboard.db")
	if ok {
		if _, err := os.Stat(dbPath); err == nil {
			dbFile, err := os.Open(dbPath)
			if err == nil {
				defer func() {
					if closeErr := dbFile.Close(); closeErr != nil {
						log.Printf("failed to close db file: %v", closeErr)
					}
				}()

				writer, err := zipWriter.Create("cboard.db")
				if err == nil {
					if _, copyErr := io.Copy(writer, dbFile); copyErr != nil {
						utils.ErrorResponse(c, http.StatusInternalServerError, "写入数据库备份失败", copyErr)
						return
					}
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

		configPath, inBase := utils.JoinWithinBaseDir(wd, configFile)
		if inBase {
			if _, err := os.Stat(configPath); err == nil {
				file, err := os.Open(configPath)
				if err == nil {
					defer func() {
						if closeErr := file.Close(); closeErr != nil {
							log.Printf("failed to close config file %s: %v", configFile, closeErr)
						}
					}()

					writer, err := zipWriter.Create(filepath.Base(configFile))
					if err == nil {
						if _, copyErr := io.Copy(writer, file); copyErr != nil {
							log.Printf("failed to copy config file %s: %v", configFile, copyErr)
						}
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
	remoteCfg := backup_service.LoadRemoteBackupConfig(db)
	backupTarget := remoteCfg.Target
	if remoteCfg.Enabled && remoteCfg.Token != "" {
		tmpFileName, tmpFilePath, tmpFileSize, err := backup_service.BuildDBOnlyBackupZip(wd, backupDir, utils.GetBeijingTime())
		if err != nil {
			utils.LogWarn("创建数据库备份文件用于远程上传失败: %v", err)
		} else {
			taskID := uuid.New().String()
			statusManager := git.GetUploadStatusManager()
			statusManager.SetStatus(taskID, &git.UploadStatus{
				Status:    "uploading",
				Progress:  0,
				Message:   "正在准备上传...",
				StartTime: utils.GetBeijingTime(),
				FileName:  tmpFileName,
				FileSize:  tmpFileSize,
			})

			go func() {
				client := git.NewClient(remoteCfg.PlatformType, remoteCfg.Token, remoteCfg.Owner, remoteCfg.Repo)
				progressCallback := func(progress int, message string) {
					statusManager.UpdateStatus(taskID, "uploading", message, progress)
				}

				if upErr := client.UploadBackupWithProgress(tmpFilePath, progressCallback); upErr != nil {
					utils.LogError(fmt.Sprintf("上传备份到 %s 失败", remoteCfg.PlatformName), upErr, nil)
					statusManager.UpdateError(taskID, upErr)
				} else {
					statusManager.UpdateStatus(taskID, "success", fmt.Sprintf("已成功上传数据库备份到 %s（仅数据库文件）", remoteCfg.PlatformName), 100)
					utils.LogInfo("数据库备份文件已成功上传到 %s（仅数据库文件）", remoteCfg.PlatformName)
				}

				if rmErr := os.Remove(tmpFilePath); rmErr != nil {
					log.Printf("failed to remove backup file: %v", rmErr)
				}
			}()

			uploadResult["async"] = true
			uploadResult["task_id"] = taskID
			uploadResult["target"] = backupTarget
			uploadResult["message"] = fmt.Sprintf("备份文件已创建，正在后台上传到%s...", remoteCfg.PlatformName)
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
	if !utils.IsWithinBaseDir(wd, backupDir) {
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
	testBackupConnection(c, "gitee")
}

func TestGitHubConnection(c *gin.Context) {
	testBackupConnection(c, "github")
}

func testBackupConnection(c *gin.Context, target string) {
	var req struct {
		Token string `json:"token"`
		Owner string `json:"owner"`
		Repo  string `json:"repo"`
	}
	_ = c.ShouldBindJSON(&req) // 请求体可为空，允许使用已保存配置

	platformCfg := backup_service.LoadPlatformConfig(database.GetDB(), target)
	if req.Token == "" {
		req.Token = platformCfg.Token
	}
	if req.Owner == "" {
		req.Owner = platformCfg.Owner
	}
	if req.Repo == "" {
		req.Repo = platformCfg.Repo
	}
	if req.Token == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Token不能为空", nil)
		return
	}

	client := git.NewClient(platformCfg.PlatformType, req.Token, req.Owner, req.Repo)
	if err := client.TestConnection(); err != nil {
		utils.LogError(fmt.Sprintf("测试 %s 连接失败", platformCfg.PlatformName), err, nil)
		utils.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("%s 连接测试失败: %s", platformCfg.PlatformName, err.Error()), err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("%s 连接测试成功", platformCfg.PlatformName), gin.H{
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

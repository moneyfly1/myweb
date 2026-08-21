package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"cboard-go/internal/services/repo_sync"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// UpdateRepoSyncSettings 保存节点同步设置
func UpdateRepoSyncSettings(c *gin.Context) {
	updateSettingsCommon(c, repo_sync.Category)
}

// GetRepoSyncStatus 获取同步状态与本地文件列表
func GetRepoSyncStatus(c *gin.Context) {
	svc := repo_sync.NewService()
	utils.SuccessResponse(c, http.StatusOK, "", svc.GetStatus())
}

// TestRepoSyncConnection 测试 GitHub 连接并列出远程目录文件
func TestRepoSyncConnection(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
		Owner string `json:"owner"`
		Repo  string `json:"repo"`
		Path  string `json:"path"`
	}
	_ = c.ShouldBindJSON(&req)

	svc := repo_sync.NewService()
	files, err := svc.TestConnectionWith(req.Token, req.Owner, req.Repo, req.Path)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "连接失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("连接成功，远程目录共 %d 个文件", len(files)), gin.H{
		"file_count": len(files),
		"entries":    files,
	})
}

// RunRepoSync 立即执行一次同步
func RunRepoSync(c *gin.Context) {
	svc := repo_sync.NewService()
	result, err := svc.SyncNow()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "同步失败", err)
		return
	}

	msg := fmt.Sprintf("同步完成: 下载 %d 个文件, 清理 %d 个文件", result.FilesDownloaded, result.FilesRemoved)
	if len(result.Errors) > 0 {
		msg += fmt.Sprintf(", %d 个文件失败", len(result.Errors))
	}
	utils.SuccessResponse(c, http.StatusOK, msg, result)
}

// ServeRepoSyncFile 公开访问同步目录中的文件（GET /repo-sync/*filepath）
func ServeRepoSyncFile(c *gin.Context) {
	svc := repo_sync.NewService()
	localDir := svc.LocalDirPath()

	reqPath := strings.TrimPrefix(c.Param("filepath"), "/")
	if reqPath == "" {
		c.Status(http.StatusNotFound)
		return
	}

	fullPath, ok := utils.JoinWithinBaseDir(localDir, filepath.FromSlash(reqPath))
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		c.Status(http.StatusNotFound)
		return
	}

	c.File(fullPath)
}

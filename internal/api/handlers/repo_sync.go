package handlers

import (
	"fmt"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"net/http"
	"net/url"
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

	// 请求体 token 为空或为脱敏掩码时，使用数据库已保存的真实 token（双保险，避免 401）
	if req.Token == "" || req.Token == maskedSecretValue {
		var saved models.SystemConfig
		if err := database.GetDB().Where("key = ? AND category = ?", "repo_sync_token", "repo_sync").First(&saved).Error; err == nil {
			req.Token = saved.Value
		}
	}

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

// ServeRepoSyncFile 公开访问同步目录（GET /repo-sync/*filepath）
// 访问目录时返回文件列表页，访问文件时直接返回文件内容
func ServeRepoSyncFile(c *gin.Context) {
	svc := repo_sync.NewService()
	localDir := svc.LocalDirPath()

	reqPath := strings.TrimPrefix(c.Param("filepath"), "/")

	dirPath, ok := utils.JoinWithinBaseDir(localDir, filepath.FromSlash(reqPath))
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}

	info, err := os.Stat(dirPath)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	if info.IsDir() {
		serveRepoSyncDirListing(c, dirPath, reqPath)
		return
	}

	// 拒绝公开点开头的隐藏/敏感文件（.env、.git 等），防止误同步的密钥文件被公开读取
	if strings.HasPrefix(filepath.Base(dirPath), ".") {
		c.Status(http.StatusNotFound)
		return
	}

	c.File(dirPath)
}

// serveRepoSyncDirListing 返回同步目录的文件列表 HTML 页
func serveRepoSyncDirListing(c *gin.Context, dirPath, relPath string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	type item struct {
		Href string
		Name string
		Size string
	}

	items := make([]item, 0, len(entries)+1)

	// 上级目录链接（根目录不显示）
	if relPath != "" {
		trimmed := strings.TrimSuffix(relPath, "/")
		idx := strings.LastIndex(trimmed, "/")
		parentRel := ""
		if idx >= 0 {
			parentRel = trimmed[:idx]
		}
		items = append(items, item{Href: "/repo-sync/" + escapeURLPath(parentRel), Name: ".. 返回上级", Size: ""})
	}

	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			continue // 跳过符号链接
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".tmp") {
			continue // 跳过下载中的临时文件
		}
		childRel := strings.TrimSuffix(relPath, "/")
		if childRel != "" {
			childRel += "/"
		}
		childRel += name

		size := ""
		if info, err := entry.Info(); err == nil && info.Mode().IsRegular() {
			size = formatListingSize(info.Size())
		} else if entry.IsDir() {
			size = "-"
		}
		displayName := name
		if entry.IsDir() {
			displayName += "/"
		}
		items = append(items, item{
			Href: "/repo-sync/" + escapeURLPath(childRel),
			Name: displayName,
			Size: size,
		})
	}

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>节点文件目录</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Microsoft YaHei', sans-serif; background: #f7f8fa; margin: 0; padding: 24px; }
.container { max-width: 900px; margin: 0 auto; background: #fff; border-radius: 12px; box-shadow: 0 2px 12px rgba(0,0,0,.06); padding: 24px 28px; }
h1 { font-size: 20px; margin: 0 0 4px; }
.path { color: #909399; font-size: 13px; margin-bottom: 16px; word-break: break-all; }
table { width: 100%%; border-collapse: collapse; }
th, td { text-align: left; padding: 10px 12px; border-bottom: 1px solid #ebeef5; font-size: 14px; }
th { color: #909399; font-weight: 500; }
a { color: #409eff; text-decoration: none; }
a:hover { text-decoration: underline; }
.size { color: #909399; text-align: right; white-space: nowrap; }
.empty { color: #909399; padding: 24px 0; text-align: center; }
</style>
</head>
<body>
<div class="container">
<h1>节点文件目录</h1>
<div class="path">/repo-sync/` + htmlEscape(relPath) + `</div>
<table>
<tr><th>名称</th><th class="size">大小</th></tr>
`)
	if len(items) == 0 {
		b.WriteString(`<tr><td colspan="2" class="empty">暂无文件</td></tr>`)
	}
	for _, it := range items {
		b.WriteString(`<tr><td><a href="` + htmlEscape(it.Href) + `">` + htmlEscape(it.Name) + `</a></td><td class="size">` + it.Size + `</td></tr>`)
	}
	b.WriteString(`</table>
</div>
</body>
</html>`)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(b.String()))
}

// escapeURLPath 对 URL 路径的每一段做百分号编码
func escapeURLPath(relPath string) string {
	if relPath == "" {
		return ""
	}
	parts := strings.Split(relPath, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

// htmlEscape 转义 HTML 特殊字符
func htmlEscape(s string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;")
	return replacer.Replace(s)
}

// formatListingSize 格式化文件大小
func formatListingSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

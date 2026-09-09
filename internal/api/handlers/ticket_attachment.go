package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cboard-go/internal/core/config"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// maxTicketAttachmentSize 单文件上限（30MB，允许上传视频）
const maxTicketAttachmentSize = 30 * 1024 * 1024

// ticketAllowedExts 允许的工单附件扩展名（图片/视频/常见文档/压缩包）
var ticketAllowedExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".bmp": true, ".svg": true,
	".mp4": true, ".mov": true, ".avi": true, ".mkv": true, ".webm": true, ".m4v": true, ".mp3": true, ".wav": true,
	".pdf": true, ".txt": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true, ".zip": true, ".rar": true, ".7z": true, ".csv": true, ".log": true,
}

// UploadTicketAttachment 工单附件上传接口（用户端 + 管理员端共用）。
// 仅做文件保存与元数据返回，不直接落库；由创建工单/回复接口通过
// attachments 数组把已上传文件绑定到对应 ticket/reply。
// 返回: { url, filename, file_name, file_size, file_type }
func UploadTicketAttachment(c *gin.Context) {
	// 登录校验（管理员与普通用户均可）
	if _, ok := getCurrentUserOrError(c); !ok {
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "文件上传失败", err)
		return
	}

	if file.Size > maxTicketAttachmentSize {
		utils.ErrorResponse(c, http.StatusBadRequest,
			fmt.Sprintf("文件超限（最大 %d MB）", maxTicketAttachmentSize>>20), nil)
		return
	}

	// 防路径穿越
	baseName := filepath.Base(file.Filename)
	if strings.Contains(baseName, "..") || strings.ContainsAny(baseName, "/\\") {
		utils.ErrorResponse(c, http.StatusBadRequest, "文件名包含非法字符", nil)
		return
	}

	ext := strings.ToLower(filepath.Ext(baseName))
	if !ticketAllowedExts[ext] {
		utils.ErrorResponse(c, http.StatusBadRequest, "不支持的文件类型", nil)
		return
	}

	// 读取头字节做 MIME 探测（加固：防伪造扩展名上传）
	f, err := file.Open()
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "无法读取文件", err)
		return
	}
	defer f.Close()
	buffer := make([]byte, 512)
	n, _ := f.Read(buffer)
	contentType := http.DetectContentType(buffer[:n])
	// 允许图片/视频/文档等常见类型；octet-stream 放行（压缩包等）
	if !strings.HasPrefix(contentType, "image/") &&
		!strings.HasPrefix(contentType, "video/") &&
		!strings.HasPrefix(contentType, "audio/") &&
		!strings.HasPrefix(contentType, "text/") &&
		contentType != "application/octet-stream" &&
		!strings.Contains(contentType, "pdf") &&
		!strings.Contains(contentType, "word") &&
		!strings.Contains(contentType, "excel") &&
		!strings.Contains(contentType, "powerpoint") &&
		!strings.Contains(contentType, "zip") &&
		!strings.Contains(contentType, "rar") {
		utils.ErrorResponse(c, http.StatusBadRequest, "文件内容类型验证失败", nil)
		return
	}
	_, _ = f.Seek(0, 0) // 重置指针

	// 目录：uploads/tickets/YYYYMM/
	cfg := config.AppConfig
	uploadBase := "uploads"
	if cfg != nil && cfg.UploadDir != "" {
		uploadBase = cfg.UploadDir
	}
	subDir := time.Now().Format("tickets/200601")
	dir := filepath.Join(uploadBase, filepath.FromSlash(subDir))
	if err := os.MkdirAll(dir, 0750); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "系统错误", err)
		return
	}

	// 安全文件名：时间戳 + 随机段 + 清理后的原名（保留中文等，仅去非法字符）
	cleanName := utils.SanitizeInput(baseName)
	if cleanName == "" {
		cleanName = fmt.Sprintf("file%d%s", time.Now().UnixNano(), ext)
	}
	safeName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), cleanName)
	fullPath := filepath.Join(dir, safeName)
	if err := c.SaveUploadedFile(file, fullPath); err != nil {
		utils.LogError("UploadTicketAttachment", err, nil)
		utils.ErrorResponse(c, http.StatusInternalServerError, "保存失败", err)
		return
	}
	_ = os.Chmod(fullPath, 0640) // 服务进程可读即可（nginx 代理场景 www 同进程）

	// URL：/uploads/tickets/YYYYMM/safeName（相对根）
	rel := filepath.ToSlash(filepath.Join(subDir, safeName))
	url := "/uploads/" + rel

	fileType := contentType
	if strings.HasPrefix(fileType, "image/") {
		fileType = "image"
	} else if strings.HasPrefix(fileType, "video/") {
		fileType = "video"
	} else if strings.HasPrefix(fileType, "audio/") {
		fileType = "audio"
	} else if strings.Contains(fileType, "pdf") {
		fileType = "pdf"
	} else if strings.Contains(contentType, "zip") || strings.Contains(contentType, "rar") || ext == ".zip" || ext == ".rar" || ext == ".7z" {
		fileType = "archive"
	} else {
		fileType = "file"
	}

	utils.SuccessResponse(c, http.StatusOK, "上传成功", gin.H{
		"url":       url,
		"filename":  safeName,
		"file_name": baseName,
		"file_size": file.Size,
		"file_type": fileType,
		"mime":      contentType,
	})
}

// ticketAttachmentInput 创建工单/回复时传入的附件信息
type ticketAttachmentInput struct {
	FileName string `json:"file_name"`
	FilePath string `json:"file_path"` // /uploads/... 相对路径
	FileSize int64  `json:"file_size"`
	FileType string `json:"file_type"`
}

// bindTicketAttachments 把已上传附件绑定到 ticket（replyID 可为 nil）。
func bindTicketAttachments(db *gorm.DB, ticketID uint, replyID *int64, uploadedBy uint, files []ticketAttachmentInput) error {
	for _, f := range files {
		if f.FilePath == "" || f.FileName == "" {
			continue
		}
		att := models.TicketAttachment{
			TicketID:   ticketID,
			ReplyID:    replyID,
			FileName:   truncateString(f.FileName, 255),
			FilePath:   truncateString(f.FilePath, 500),
			UploadedBy: uploadedBy,
		}
		if f.FileSize > 0 {
			size := f.FileSize
			att.FileSize = &size
		}
		if f.FileType != "" {
			ft := truncateString(f.FileType, 50)
			att.FileType = &ft
		}
		if err := db.Create(&att).Error; err != nil {
			return err
		}
	}
	return nil
}

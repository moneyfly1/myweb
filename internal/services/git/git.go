package git

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cboard-go/internal/utils"
)

// PlatformType 平台类型
type PlatformType string

const (
	PlatformGitee  PlatformType = "gitee"
	PlatformGitHub PlatformType = "github"
)

// GitClient 统一的Git客户端
type GitClient struct {
	Platform   PlatformType
	Token      string
	Owner      string
	Repo       string
	BaseURL    string
	APIPath    string
	AuthType   string // "token" 或 "Bearer"
	HTTPMethod string // "POST" 或 "PUT"
}

// NewClient 创建Git客户端
func NewClient(platform PlatformType, token, owner, repo string) *GitClient {
	client := &GitClient{
		Platform: platform,
		Token:    token,
		Owner:    owner,
		Repo:     repo,
	}

	// 根据平台设置不同的配置
	if platform == PlatformGitHub {
		client.BaseURL = "https://api.github.com"
		client.APIPath = "/repos/%s/%s"
		client.AuthType = "Bearer"
		client.HTTPMethod = "PUT"
	} else {
		// 默认Gitee
		client.BaseURL = "https://gitee.com"
		client.APIPath = "/api/v5/repos/%s/%s"
		client.AuthType = "token"
		client.HTTPMethod = "POST"
	}

	return client
}

// NewGiteeClient 创建Gitee客户端（兼容旧代码）
func NewGiteeClient(token, owner, repo string) *GitClient {
	return NewClient(PlatformGitee, token, owner, repo)
}

// NewGitHubClient 创建GitHub客户端（兼容旧代码）
func NewGitHubClient(token, owner, repo string) *GitClient {
	return NewClient(PlatformGitHub, token, owner, repo)
}

// ProgressCallback 进度回调函数类型
type ProgressCallback func(progress int, message string)

// getPlatformName 获取平台名称
func (c *GitClient) getPlatformName() string {
	if c.Platform == PlatformGitHub {
		return "GitHub"
	}
	return "Gitee"
}

// getAuthHeader 获取认证头
func (c *GitClient) getAuthHeader() string {
	if c.AuthType == "Bearer" {
		return "Bearer " + c.Token
	}
	return "token " + c.Token
}

// getAPIURL 获取API URL
func (c *GitClient) getAPIURL(path string) string {
	basePath := fmt.Sprintf(c.APIPath, c.Owner, c.Repo)
	return fmt.Sprintf("%s%s%s", c.BaseURL, basePath, path)
}

// UploadFile 上传文件（不带进度）
func (c *GitClient) UploadFile(filePath, remotePath string) error {
	return c.UploadFileWithProgress(filePath, remotePath, nil)
}

// UploadFileWithProgress 上传文件（带进度回调）
func (c *GitClient) UploadFileWithProgress(filePath, remotePath string, progressCallback ProgressCallback) error {
	if progressCallback != nil {
		progressCallback(5, "正在打开文件...")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	if progressCallback != nil {
		progressCallback(10, "正在读取文件...")
	}

	fileContent, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	if progressCallback != nil {
		progressCallback(30, "正在编码文件...")
	}

	contentBase64 := base64.StdEncoding.EncodeToString(fileContent)

	if progressCallback != nil {
		progressCallback(50, "正在检查远程文件...")
	}

	existingSHA, err := c.getFileSHA(remotePath)
	if err != nil {
		existingSHA = ""
	}

	apiURL := c.getAPIURL("/contents/" + remotePath)

	// GitHub需要message在前，Gitee需要content在前
	var payload map[string]interface{}
	if c.Platform == PlatformGitHub {
		payload = map[string]interface{}{
			"message": fmt.Sprintf("备份文件: %s", filepath.Base(remotePath)),
			"content": contentBase64,
		}
	} else {
		payload = map[string]interface{}{
			"content": contentBase64,
			"message": fmt.Sprintf("备份文件: %s", filepath.Base(remotePath)),
		}
	}

	if existingSHA != "" {
		payload["sha"] = existingSHA
	}

	if progressCallback != nil {
		progressCallback(60, "正在准备上传数据...")
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequest(c.HTTPMethod, apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.getAuthHeader())

	// GitHub需要Accept头
	if c.Platform == PlatformGitHub {
		req.Header.Set("Accept", "application/vnd.github.v3+json")
	}

	platformName := c.getPlatformName()
	if progressCallback != nil {
		progressCallback(70, fmt.Sprintf("正在上传到%s...", platformName))
	}

	// 创建带超时的context，支持大文件上传（15分钟）
	ctx, cancel := context.WithTimeout(context.Background(), 15*60*time.Second)
	defer cancel()

	// 创建自定义的HTTP客户端，优化大文件上传
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   15 * 60 * time.Second, // 15分钟超时
	}

	// 将请求绑定到context
	req = req.WithContext(ctx)

	// 使用goroutine模拟上传进度（但不要阻塞实际上传）
	progressDone := make(chan bool)
	if progressCallback != nil {
		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			progress := 70
			for {
				select {
				case <-ticker.C:
					if progress < 95 {
						progress += 2
						progressCallback(progress, fmt.Sprintf("正在上传到%s...", platformName))
					}
				case <-progressDone:
					return
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// 执行上传请求，带重试机制
	var resp *http.Response
	maxRetries := 3
	var uploadErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			if progressCallback != nil {
				progressCallback(70, fmt.Sprintf("重试上传 (第%d次)...", attempt))
			}
			// 等待一段时间后重试
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
			// 重新创建请求（因为请求体已经被读取）
			var newReqErr error
			req, newReqErr = http.NewRequestWithContext(ctx, c.HTTPMethod, apiURL, strings.NewReader(string(jsonData)))
			if newReqErr != nil {
				close(progressDone)
				return fmt.Errorf("重新创建请求失败: %w", newReqErr)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", c.getAuthHeader())
			if c.Platform == PlatformGitHub {
				req.Header.Set("Accept", "application/vnd.github.v3+json")
			}
		}

		resp, uploadErr = client.Do(req)
		if uploadErr == nil {
			break
		}

		// 检查是否是网络连接错误，如果是则重试
		errMsg := uploadErr.Error()
		if strings.Contains(errMsg, "closed network connection") ||
			strings.Contains(errMsg, "connection reset") ||
			strings.Contains(errMsg, "timeout") ||
			strings.Contains(errMsg, "broken pipe") {
			if attempt < maxRetries {
				log.Printf("[INFO] 上传失败，准备重试 (第%d次): %v", attempt, uploadErr)
				continue
			}
		}

		// 如果不是可重试的错误，直接返回
		if attempt == maxRetries {
			close(progressDone)
			return fmt.Errorf("请求失败 (已重试%d次): %w", maxRetries, uploadErr)
		}
	}

	close(progressDone)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("上传文件失败: %s, 响应: %s", resp.Status, string(body))
	}

	if progressCallback != nil {
		progressCallback(100, "上传完成")
	}

	return nil
}

// getFileSHA 获取文件的SHA值
func (c *GitClient) getFileSHA(filePath string) (string, error) {
	apiURL := c.getAPIURL("/contents/" + filePath)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", c.getAuthHeader())
	if c.Platform == PlatformGitHub {
		req.Header.Set("Accept", "application/vnd.github.v3+json")
	}

	// 增加超时时间到2分钟
	httpClient := &http.Client{Timeout: 120 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}

	if resp.StatusCode != http.StatusOK {
		return "", nil
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", nil
	}

	if sha, ok := result["sha"].(string); ok {
		return sha, nil
	}

	return "", nil
}

// UploadBackup 上传备份文件（不带进度）
func (c *GitClient) UploadBackup(filePath string) error {
	return c.UploadBackupWithProgress(filePath, nil)
}

// UploadBackupWithProgress 上传备份文件（带进度回调）
// 按照年/月/日的文件夹结构组织: YYYY/MM/DD/filename.zip
func (c *GitClient) UploadBackupWithProgress(filePath string, progressCallback ProgressCallback) error {
	now := utils.GetBeijingTime()
	year := now.Format("2006")
	month := now.Format("01")
	day := now.Format("02")
	fileName := filepath.Base(filePath)

	// 构建远程路径: 年/月/日/文件名
	remotePath := fmt.Sprintf("%s/%s/%s/%s", year, month, day, fileName)

	return c.UploadFileWithProgress(filePath, remotePath, progressCallback)
}

// TestConnection 测试连接
func (c *GitClient) TestConnection() error {
	apiURL := c.getAPIURL("")

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", c.getAuthHeader())
	if c.Platform == PlatformGitHub {
		req.Header.Set("Accept", "application/vnd.github.v3+json")
	}

	httpClient := &http.Client{Timeout: 60 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("测试连接失败: %s, 响应: %s", resp.Status, string(body))
	}

	return nil
}

// UploadStatus 上传状态
type UploadStatus struct {
	Status     string    `json:"status"`      // uploading, success, failed
	Progress   int       `json:"progress"`    // 0-100
	Message    string    `json:"message"`     // 状态消息
	Error      string    `json:"error"`       // 错误信息
	StartTime  time.Time `json:"start_time"`  // 开始时间
	FinishTime time.Time `json:"finish_time"` // 完成时间
	FileName   string    `json:"file_name"`   // 文件名
	FileSize   int64     `json:"file_size"`   // 文件大小
}

// UploadStatusManager 上传状态管理器
type UploadStatusManager struct {
	statuses map[string]*UploadStatus
	mu       sync.RWMutex
}

var globalUploadStatusManager = &UploadStatusManager{
	statuses: make(map[string]*UploadStatus),
}

// GetUploadStatusManager 获取全局上传状态管理器
func GetUploadStatusManager() *UploadStatusManager {
	return globalUploadStatusManager
}

// SetStatus 设置上传状态
func (m *UploadStatusManager) SetStatus(taskID string, status *UploadStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statuses[taskID] = status
}

// GetStatus 获取上传状态
func (m *UploadStatusManager) GetStatus(taskID string) (*UploadStatus, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	status, exists := m.statuses[taskID]
	return status, exists
}

// UpdateStatus 更新上传状态
func (m *UploadStatusManager) UpdateStatus(taskID string, status string, message string, progress int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, exists := m.statuses[taskID]; exists {
		s.Status = status
		s.Message = message
		s.Progress = progress
		if status == "success" || status == "failed" {
			s.FinishTime = utils.GetBeijingTime()
		}
	}
}

// UpdateError 更新错误信息
func (m *UploadStatusManager) UpdateError(taskID string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, exists := m.statuses[taskID]; exists {
		s.Status = "failed"
		s.Error = err.Error()
		s.Message = "上传失败: " + err.Error()
		s.FinishTime = utils.GetBeijingTime()
	}
}

// CleanOldStatuses 清理超过1小时的状态记录
func (m *UploadStatusManager) CleanOldStatuses() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := utils.GetBeijingTime()
	for taskID, status := range m.statuses {
		if status.FinishTime.IsZero() {
			// 未完成的任务，如果超过2小时也清理
			if now.Sub(status.StartTime) > 2*time.Hour {
				delete(m.statuses, taskID)
			}
		} else {
			// 已完成的任务，超过1小时清理
			if now.Sub(status.FinishTime) > time.Hour {
				delete(m.statuses, taskID)
			}
		}
	}
}

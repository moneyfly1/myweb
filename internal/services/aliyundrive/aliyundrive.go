package aliyundrive

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cboard-go/internal/utils"
)

// AliyunDriveClient 阿里云盘客户端
type AliyunDriveClient struct {
	RefreshToken string
	AccessToken  string
	ExpiresAt    time.Time
	DriveID      string // 缓存 drive_id
}

// TokenResponse 阿里云盘Token响应
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// FileInfo 文件信息
type FileInfo struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
	Size     int64  `json:"size"`
}

// UploadResponse 上传响应
type UploadResponse struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
	Size     int64  `json:"size"`
}

// NewAliyunDriveClient 创建新的阿里云盘客户端
func NewAliyunDriveClient(refreshToken string) *AliyunDriveClient {
	return &AliyunDriveClient{
		RefreshToken: refreshToken,
	}
}

// GetAccessToken 获取访问令牌
func (c *AliyunDriveClient) GetAccessToken() error {
	if c.AccessToken != "" && time.Now().Before(c.ExpiresAt) {
		return nil // Token仍然有效
	}

	url := "https://auth.aliyundrive.com/v2/account/token"
	payload := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": c.RefreshToken,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// 尝试解析错误响应
		var errorResp map[string]interface{}
		if err := json.Unmarshal([]byte(bodyStr), &errorResp); err == nil {
			if msg, ok := errorResp["message"].(string); ok && msg != "" {
				return fmt.Errorf("获取token失败: %s (状态码: %d)", msg, resp.StatusCode)
			}
			if msg, ok := errorResp["msg"].(string); ok && msg != "" {
				return fmt.Errorf("获取token失败: %s (状态码: %d)", msg, resp.StatusCode)
			}
			if code, ok := errorResp["code"].(string); ok && code != "" {
				return fmt.Errorf("获取token失败: 错误代码 %s (状态码: %d)", code, resp.StatusCode)
			}
			if code, ok := errorResp["code"].(float64); ok {
				return fmt.Errorf("获取token失败: 错误代码 %.0f (状态码: %d)", code, resp.StatusCode)
			}
		}

		// 如果响应体太长，截断
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500] + "..."
		}

		return fmt.Errorf("获取token失败: HTTP %d, 响应: %s", resp.StatusCode, bodyStr)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	c.AccessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		c.RefreshToken = tokenResp.RefreshToken // 更新refresh_token
	}
	c.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// 尝试从 token 响应中获取 drive_id（如果有的话）
	// 如果没有，稍后通过其他API获取

	return nil
}

// GetDriveID 获取Drive ID
func (c *AliyunDriveClient) GetDriveID() (string, error) {
	if err := c.GetAccessToken(); err != nil {
		return "", err
	}

	// 如果已经缓存了 drive_id，直接返回
	if c.DriveID != "" {
		return c.DriveID, nil
	}

	// 尝试通过文件列表接口获取 drive_id
	// 使用一个简单的文件列表请求来获取 drive_id
	url := "https://api.aliyundrive.com/v2/file/list"
	payload := map[string]interface{}{
		"parent_file_id": "root",
		"limit":          1,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			// 尝试从响应中获取 drive_id
			if driveID, ok := result["drive_id"].(string); ok && driveID != "" {
				c.DriveID = driveID
				return driveID, nil
			}
			// 尝试从 items 中的第一个文件获取 drive_id
			if items, ok := result["items"].([]interface{}); ok && len(items) > 0 {
				if firstItem, ok := items[0].(map[string]interface{}); ok {
					if driveID, ok := firstItem["drive_id"].(string); ok && driveID != "" {
						c.DriveID = driveID
						return driveID, nil
					}
				}
			}
		}
	} else {
		// 如果请求失败，读取错误信息
		body, _ := io.ReadAll(resp.Body)
		// 如果是 404 或其他错误，可能是 drive_id 的问题
		// 尝试从错误响应中获取信息
		var errorResp map[string]interface{}
		if err := json.Unmarshal([]byte(body), &errorResp); err == nil {
			// 检查是否是 drive_id 相关错误
			if code, ok := errorResp["code"].(string); ok {
				if strings.Contains(code, "Drive") {
					// 可能需要使用不同的方法获取 drive_id
					// 暂时返回空，让调用者处理
				}
			}
		}
	}

	// 如果无法从文件列表获取，尝试使用创建文件夹时让API自动处理
	// 或者，我们可以尝试使用一个已知的API来获取
	// 对于移动端token，可能需要使用不同的API端点
	return "", fmt.Errorf("无法获取Drive ID。响应状态: %d", resp.StatusCode)
}

// CreateFolder 创建文件夹（如果不存在）
func (c *AliyunDriveClient) CreateFolder(parentFileID, folderName string) (string, error) {
	if err := c.GetAccessToken(); err != nil {
		return "", err
	}

	// 先尝试获取 drive_id，如果失败，尝试在创建文件夹时让API自动处理
	driveID, err := c.GetDriveID()
	if err != nil {
		// 如果获取 drive_id 失败，尝试使用空字符串，让API自动处理
		driveID = ""
	}

	// 先检查文件夹是否存在（如果drive_id可用）
	if driveID != "" {
		folderID, err := c.FindFolder(parentFileID, folderName)
		if err == nil && folderID != "" {
			return folderID, nil // 文件夹已存在
		}
	}

	url := "https://api.aliyundrive.com/v2/file/create"
	payload := map[string]interface{}{
		"parent_file_id": parentFileID,
		"name":           folderName,
		"type":           "folder",
	}
	// 只有在有 drive_id 时才添加
	if driveID != "" {
		payload["drive_id"] = driveID
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("创建文件夹失败: %s, 响应: %s", resp.Status, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	fileID, ok := result["file_id"].(string)
	if !ok {
		return "", fmt.Errorf("无法获取文件夹ID")
	}

	return fileID, nil
}

// FindFolder 查找文件夹
func (c *AliyunDriveClient) FindFolder(parentFileID, folderName string) (string, error) {
	if err := c.GetAccessToken(); err != nil {
		return "", err
	}

	driveID, err := c.GetDriveID()
	if err != nil {
		return "", err
	}

	url := "https://api.aliyundrive.com/v2/file/list"
	payload := map[string]interface{}{
		"drive_id":       driveID,
		"parent_file_id": parentFileID,
		"limit":          100,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("查找文件夹失败: %s, 响应: %s", resp.Status, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	items, ok := result["items"].([]interface{})
	if !ok {
		return "", fmt.Errorf("文件夹不存在")
	}

	for _, item := range items {
		file, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if name, ok := file["name"].(string); ok && name == folderName {
			if fileID, ok := file["file_id"].(string); ok {
				return fileID, nil
			}
		}
	}

	return "", fmt.Errorf("文件夹不存在")
}

// UploadFile 上传文件到阿里云盘
func (c *AliyunDriveClient) UploadFile(filePath, parentFileID string) (*UploadResponse, error) {
	if err := c.GetAccessToken(); err != nil {
		return nil, err
	}

	driveID, err := c.GetDriveID()
	if err != nil {
		return nil, err
	}

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	fileName := filepath.Base(filePath)
	fileSize := fileInfo.Size()

	// 1. 创建上传会话
	uploadURL := "https://api.aliyundrive.com/v2/file/create_with_proof"
	createPayload := map[string]interface{}{
		"drive_id":        driveID,
		"parent_file_id":  parentFileID,
		"name":            fileName,
		"type":            "file",
		"check_name_mode": "auto_rename",
		"size":            fileSize,
	}

	jsonData, err := json.Marshal(createPayload)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequest("POST", uploadURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("创建上传会话失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("创建上传会话失败: %s, 响应: %s", resp.Status, string(body))
	}

	var createResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&createResult); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	fileID, ok := createResult["file_id"].(string)
	if !ok {
		return nil, fmt.Errorf("无法获取文件ID")
	}

	uploadURLStr, ok := createResult["upload_url"].(string)
	if !ok {
		// 如果文件已存在或不需要上传，直接返回
		return &UploadResponse{
			FileID:   fileID,
			FileName: fileName,
			Size:     fileSize,
		}, nil
	}

	// 2. 上传文件内容
	file.Seek(0, 0) // 重置文件指针
	uploadReq, err := http.NewRequest("PUT", uploadURLStr, file)
	if err != nil {
		return nil, fmt.Errorf("创建上传请求失败: %w", err)
	}

	uploadReq.ContentLength = fileSize
	uploadReq.Header.Set("Content-Type", "application/octet-stream")

	uploadClient := &http.Client{Timeout: 300 * time.Second} // 大文件需要更长时间
	uploadResp, err := uploadClient.Do(uploadReq)
	if err != nil {
		return nil, fmt.Errorf("上传文件失败: %w", err)
	}
	defer uploadResp.Body.Close()

	if uploadResp.StatusCode != http.StatusOK && uploadResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(uploadResp.Body)
		return nil, fmt.Errorf("上传文件失败: %s, 响应: %s", uploadResp.Status, string(body))
	}

	// 3. 完成上传
	completeURL := "https://api.aliyundrive.com/v2/file/complete"
	completePayload := map[string]interface{}{
		"drive_id": driveID,
		"file_id":  fileID,
	}

	completeData, err := json.Marshal(completePayload)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	completeReq, err := http.NewRequest("POST", completeURL, bytes.NewBuffer(completeData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	completeReq.Header.Set("Authorization", "Bearer "+c.AccessToken)
	completeReq.Header.Set("Content-Type", "application/json")

	completeResp, err := client.Do(completeReq)
	if err != nil {
		return nil, fmt.Errorf("完成上传失败: %w", err)
	}
	defer completeResp.Body.Close()

	if completeResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(completeResp.Body)
		utils.LogError("完成上传失败", fmt.Errorf("状态码: %s, 响应: %s", completeResp.Status, string(body)), nil)
		// 即使完成失败，文件可能已经上传成功
	}

	return &UploadResponse{
		FileID:   fileID,
		FileName: fileName,
		Size:     fileSize,
	}, nil
}

// UploadBackup 上传备份文件到阿里云盘
// folderID: 如果提供，则使用指定的文件夹ID；如果为空，则自动创建"CBoard备份"文件夹
func (c *AliyunDriveClient) UploadBackup(filePath string, folderID string) error {
	var backupFolderID string
	var err error

	if folderID != "" {
		// 使用用户指定的文件夹ID
		backupFolderID = folderID
	} else {
		// 自动创建备份文件夹
		// 先尝试获取 drive_id，如果失败则使用 "root"
		driveID, driveErr := c.GetDriveID()
		parentFileID := "root"
		if driveErr == nil && driveID != "" {
			// 如果成功获取 drive_id，尝试使用它
			// 但创建文件夹时仍然使用 root 作为 parent
			parentFileID = "root"
		}

		backupFolderID, err = c.CreateFolder(parentFileID, "CBoard备份")
		if err != nil {
			return fmt.Errorf("创建备份文件夹失败: %w", err)
		}
	}

	// 上传文件
	_, err = c.UploadFile(filePath, backupFolderID)
	if err != nil {
		return fmt.Errorf("上传备份文件失败: %w", err)
	}

	return nil
}

// Package aliyundrive 封装阿里云盘客户端。
//
// 使用阿里云盘「官方 App/网页 内部 API」（与 tickstep/aliyunpan CLI 同款方案）：
//   - 认证：auth.aliyundrive.com/v2/account/token + 官方 Web 客户端公开 api_id，
//     无需注册开放平台开发者账号；不绑定 IP、无境外风控。
//   - 列表/上传/删除/取链：api.aliyundrive.com（内部接口，未文档化，与 aliyunpan CLI 一致）。
//
// 已知要点（实测验证 2026-08）：
//   - drive_id 必须是真实 drive_id（刷新响应 default_drive_id），不能用 "root"。
//   - 下载直链与上传分片 PUT 都必须带 Referer: https://www.aliyundrive.com/，
//     否则 CDN 返回 400/403。
//   - 上传分片 PUT 不能带 Content-Type 头（OSS 预签名按无 CT 计算签名）。
//   - refresh_token 为一次性轮换制：每次刷新旧 token 立即作废，需及时回存（OnRotate）。
package aliyundrive

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// authURL 刷新令牌地址（官方 Web 客户端公开 api_id）
	authURL = "https://auth.aliyundrive.com/v2/account/token"
	// apiBase 内部 API 地址
	apiBase = "https://api.aliyundrive.com"
	// webAPIID 官方 Web 客户端公开 api_id（aliyunpan CLI 同款，无需开发者账号）
	webAPIID = "pJZInNHN2dZWk8qg"
	// referer 下载/上传 CDN 校验来源
	referer = "https://www.aliyundrive.com/"
	// userAgent 与 aliyunpan CLI 一致的浏览器 UA
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/63.0.3239.132 Safari/537.36"

	partSize = 20 << 20 // 20MB 分片
)

// File 阿里云盘文件信息
type File struct {
	FileID       string `json:"file_id"`
	Name         string `json:"name"`
	Type         string `json:"type"` // folder / file
	Size         int64  `json:"size"`
	ParentFileID string `json:"parent_file_id"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// DisplaySize 人类可读大小
func (f File) DisplaySize() string {
	const unit = 1024
	if f.Size < unit {
		return fmt.Sprintf("%d B", f.Size)
	}
	div, exp := int64(unit), 0
	for n := f.Size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(f.Size)/float64(div), "KMGTPE"[exp])
}

// Client 阿里云盘客户端
type Client struct {
	RefreshToken string
	AccessToken  string
	DriveID      string // 真实 drive_id（刷新响应 default_drive_id）
	http         *http.Client
	// OnRotate 刷新成功后回调（传入新的 refresh_token），
	// 由调用方持久化到数据库。阿里云盘 refresh_token 为一次性轮换制：
	// 每次刷新旧 token 立即作废，若不及时回存，下一次请求必然失败。
	OnRotate func(newRefreshToken string)
}

// New 创建客户端
func New(refreshToken string) *Client {
	return &Client{
		RefreshToken: strings.TrimSpace(refreshToken),
		http:         &http.Client{Timeout: 30 * time.Second},
	}
}

// Refresh 用 refresh_token 换取 access_token（并拿到新的 refresh_token，需回存），
// 同时记录真实 drive_id。该接口不校验请求 IP，海外服务器可用。
func (c *Client) Refresh() (newRefreshToken string, err error) {
	body := map[string]string{
		"refresh_token": c.RefreshToken,
		"api_id":        webAPIID,
		"grant_type":    "refresh_token",
	}
	payload, err := postJSON(authURL, body, nil)
	if err != nil {
		return "", err
	}
	var out struct {
		AccessToken      string `json:"access_token"`
		RefreshToken     string `json:"refresh_token"`
		DefaultDriveID   string `json:"default_drive_id"`
		DefaultSboxDrive string `json:"default_sbox_drive_id"`
		Error            string `json:"error"`
		ErrorDesc        string `json:"error_description"`
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return "", fmt.Errorf("token 刷新响应解析失败: %w", err)
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("token 刷新失败: %s %s", out.Error, out.ErrorDesc)
	}
	c.AccessToken = out.AccessToken
	if out.DefaultDriveID != "" {
		c.DriveID = out.DefaultDriveID
	}
	if out.RefreshToken != "" {
		c.RefreshToken = out.RefreshToken
		if c.OnRotate != nil {
			c.OnRotate(out.RefreshToken)
		}
		return out.RefreshToken, nil
	}
	return "", nil
}

// ensureToken 确保有 access_token（并已拿到真实 drive_id）
func (c *Client) ensureToken() error {
	if c.AccessToken != "" {
		return nil
	}
	_, err := c.Refresh()
	return err
}

// List 列出目录文件（分页，limit<=200）
func (c *Client) List(parentFileID string, limit int) ([]File, error) {
	items, _, err := c.listPage(parentFileID, limit, "")
	return items, err
}

func (c *Client) listPage(parentFileID string, limit int, marker string) ([]File, string, error) {
	if err := c.ensureToken(); err != nil {
		return nil, "", err
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	if parentFileID == "" {
		parentFileID = "root"
	}
	body := map[string]interface{}{
		"drive_id":        c.DriveID,
		"parent_file_id":  parentFileID,
		"limit":           limit,
		"order_by":        "updated_at",
		"order_direction": "DESC",
	}
	if marker != "" {
		body["marker"] = marker
	}
	payload, err := c.request("/adrive/v3/file/list", body)
	if err != nil {
		return nil, "", err
	}
	var out struct {
		Items      []File `json:"items"`
		NextMarker string `json:"next_marker"`
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, "", fmt.Errorf("列表响应解析失败: %w", err)
	}
	return out.Items, out.NextMarker, nil
}

// Search 搜索文件：内部 API 无搜索端点，退化为「根目录 + 一级子目录」列表按名称过滤
// （供管理端调试使用；生产路径依赖 file_id_map，不需要搜索）
func (c *Client) Search(keyword string, limit int) ([]File, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	kw := strings.ToLower(strings.TrimSpace(keyword))
	if kw == "" {
		return nil, nil
	}
	out := make([]File, 0, limit)
	roots, _, err := c.listPage("root", 200, "")
	if err != nil {
		return nil, err
	}
	dirs := make([]File, 0)
	for _, f := range roots {
		if strings.Contains(strings.ToLower(f.Name), kw) && f.Type == "file" {
			out = append(out, f)
			if len(out) >= limit {
				return out, nil
			}
		}
		if f.Type == "folder" {
			dirs = append(dirs, f)
		}
	}
	for _, d := range dirs {
		sub, _, serr := c.listPage(d.FileID, 100, "")
		if serr != nil {
			continue
		}
		for _, f := range sub {
			if strings.Contains(strings.ToLower(f.Name), kw) && f.Type == "file" {
				out = append(out, f)
				if len(out) >= limit {
					return out, nil
				}
			}
		}
	}
	return out, nil
}

// GetFile 校验文件是否仍存在：通过取直链接口探测（文件不存在时返回错误）
func (c *Client) GetFile(fileID string) (*File, error) {
	if err := c.ensureToken(); err != nil {
		return nil, err
	}
	body := map[string]interface{}{
		"drive_id":   c.DriveID,
		"file_id":    fileID,
		"expire_sec": 60,
	}
	payload, err := c.request("/v2/file/get_download_url", body)
	if err != nil {
		return nil, err
	}
	var out struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, fmt.Errorf("文件信息响应解析失败: %w", err)
	}
	if out.URL == "" {
		return nil, errors.New("文件不存在")
	}
	return &File{FileID: fileID}, nil
}

// IsFileNotFound 判断错误是否为"文件不存在"（用户手动删除等场景）
func IsFileNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "notfound") ||
		strings.Contains(msg, "not_found") ||
		strings.Contains(msg, "not exist") ||
		strings.Contains(msg, "不存在")
}

// DownloadURL 获取文件直链（最长 4 小时有效；获取后需带 Referer 才能下载，见 Referer()）
func (c *Client) DownloadURL(fileID string) (string, error) {
	if err := c.ensureToken(); err != nil {
		return "", err
	}
	body := map[string]interface{}{
		"drive_id":   c.DriveID,
		"file_id":    fileID,
		"expire_sec": 14400,
	}
	payload, err := c.request("/v2/file/get_download_url", body)
	if err != nil {
		return "", err
	}
	var out struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return "", fmt.Errorf("直链响应解析失败: %w", err)
	}
	if out.URL == "" {
		return "", errors.New("直链为空")
	}
	return out.URL, nil
}

// Referer 下载直链必需的来源头
func Referer() string { return referer }

// Upload 上传本地文件到指定目录，返回 file_id。
// 流程：createWithFolders → 分片 PUT（带 Referer、不带 Content-Type）→ complete
func (c *Client) Upload(localPath, fileName, parentFileID string) (string, error) {
	if err := c.ensureToken(); err != nil {
		return "", err
	}
	fi, err := os.Stat(localPath)
	if err != nil {
		return "", fmt.Errorf("读取本地文件失败: %w", err)
	}
	size := fi.Size()
	if size == 0 {
		return "", errors.New("文件大小为 0")
	}
	if parentFileID == "" {
		parentFileID = "root"
	}

	partCount := int((size + partSize - 1) / partSize)
	partInfoList := make([]map[string]interface{}, 0, partCount)
	for i := 1; i <= partCount; i++ {
		partInfoList = append(partInfoList, map[string]interface{}{"part_number": i})
	}

	// 1. createWithFolders
	createBody := map[string]interface{}{
		"drive_id":        c.DriveID,
		"parent_file_id":  parentFileID,
		"name":            fileName,
		"type":            "file",
		"check_name_mode": "ignore",
		"size":            size,
		"part_info_list":  partInfoList,
	}
	payload, err := c.request("/adrive/v2/file/createWithFolders", createBody)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	var created struct {
		FileID       string `json:"file_id"`
		UploadID     string `json:"upload_id"`
		RapidUpload  bool   `json:"rapid_upload"`
		PartInfoList []struct {
			PartNumber int    `json:"part_number"`
			UploadURL  string `json:"upload_url"`
		} `json:"part_info_list"`
	}
	if err := json.Unmarshal(payload, &created); err != nil {
		return "", fmt.Errorf("创建文件响应解析失败: %w", err)
	}
	if created.FileID == "" {
		return "", errors.New("创建文件失败：未返回 file_id")
	}
	if created.RapidUpload {
		return created.FileID, nil // 秒传
	}
	if created.UploadID == "" {
		return "", errors.New("创建文件失败：未返回 upload_id")
	}

	// 2. 组装分片上传地址（缺失时 getUploadUrl 补取）
	uploadURLs := map[int]string{}
	for _, p := range created.PartInfoList {
		uploadURLs[p.PartNumber] = p.UploadURL
	}
	if len(uploadURLs) < partCount {
		upPayload, uerr := c.request("/v2/file/get_upload_url", map[string]interface{}{
			"drive_id":       c.DriveID,
			"file_id":        created.FileID,
			"part_info_list": partInfoList,
		})
		if uerr != nil {
			return "", fmt.Errorf("获取分片上传地址失败: %w", uerr)
		}
		var up struct {
			PartInfoList []struct {
				PartNumber int    `json:"part_number"`
				UploadURL  string `json:"upload_url"`
			} `json:"part_info_list"`
		}
		if err := json.Unmarshal(upPayload, &up); err == nil {
			for _, p := range up.PartInfoList {
				uploadURLs[p.PartNumber] = p.UploadURL
			}
		}
	}

	// 3. 分片 PUT：必须带 Referer，且【不能带 Content-Type】（OSS 预签名按无 CT 计算签名）
	f, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer f.Close()

	for part := 1; part <= partCount; part++ {
		uploadURL := uploadURLs[part]
		if uploadURL == "" {
			return "", fmt.Errorf("第 %d 片上传地址为空", part)
		}
		offset := int64(part-1) * partSize
		curSize := int64(partSize)
		if part == partCount {
			curSize = size - offset
		}
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return "", err
		}
		req, err := http.NewRequest(http.MethodPut, uploadURL, io.LimitReader(f, curSize))
		if err != nil {
			return "", err
		}
		req.ContentLength = curSize
		req.Header.Set("Referer", referer)
		req.Header.Set("User-Agent", userAgent)
		// 注意：不设置 Content-Type（io.LimitReader 不会被 Go 嗅探，
		// 因此请求不会带 Content-Type 头，符合 OSS 预签名要求）
		resp, err := c.http.Do(req)
		if err != nil {
			return "", fmt.Errorf("第 %d 片上传失败: %w", part, err)
		}
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("第 %d 片上传失败: HTTP %d", part, resp.StatusCode)
		}
	}

	// 4. complete
	completeBody := map[string]interface{}{
		"drive_id":  c.DriveID,
		"file_id":   created.FileID,
		"upload_id": created.UploadID,
	}
	if _, err := c.request("/v2/file/complete", completeBody); err != nil {
		return "", fmt.Errorf("完成上传失败: %w", err)
	}
	return created.FileID, nil
}

// EnsureFolder 确保父目录下存在指定名称的文件夹，不存在则创建，返回其 file_id
func (c *Client) EnsureFolder(parentFileID, name string) (string, error) {
	if err := c.ensureToken(); err != nil {
		return "", err
	}
	name = strings.TrimSpace(name)
	if name == "" || name == "/" {
		return "root", nil
	}
	if parentFileID == "" {
		parentFileID = "root"
	}
	marker := ""
	for page := 0; page < 10; page++ {
		files, next, err := c.listPage(parentFileID, 200, marker)
		if err != nil {
			return "", fmt.Errorf("列出目录失败: %w", err)
		}
		for _, f := range files {
			if f.Type == "folder" && f.Name == name {
				return f.FileID, nil
			}
		}
		if next == "" {
			break
		}
		marker = next
	}
	body := map[string]interface{}{
		"drive_id":        c.DriveID,
		"parent_file_id":  parentFileID,
		"name":            name,
		"type":            "folder",
		"check_name_mode": "refuse",
	}
	payload, err := c.request("/adrive/v2/file/createWithFolders", body)
	if err != nil {
		return "", fmt.Errorf("创建文件夹失败: %w", err)
	}
	var out struct {
		FileID string `json:"file_id"`
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return "", fmt.Errorf("创建文件夹响应解析失败: %w", err)
	}
	if out.FileID == "" {
		return "", errors.New("创建文件夹失败：未返回 file_id")
	}
	return out.FileID, nil
}

// EnsureDir 按路径（/ 分隔，支持多级）确保目录存在，返回最终目录 file_id；空路径返回 "root"
func (c *Client) EnsureDir(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return "root", nil
	}
	parent := "root"
	for _, seg := range strings.Split(path, "/") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		id, err := c.EnsureFolder(parent, seg)
		if err != nil {
			return "", err
		}
		parent = id
	}
	return parent, nil
}

// Trash 删除文件（移入回收站），用于替换旧版本时清理
func (c *Client) Trash(fileID string) error {
	if err := c.ensureToken(); err != nil {
		return err
	}
	body := map[string]interface{}{
		"requests": []map[string]interface{}{
			{
				"headers": map[string]string{"Content-Type": "application/json"},
				"method":  "POST",
				"url":     "/recyclebin/trash",
				"body": map[string]string{
					"drive_id": c.DriveID,
					"file_id":  fileID,
				},
			},
		},
	}
	payload, err := c.request("/adrive/v4/batch", body)
	if err != nil {
		return fmt.Errorf("删除文件失败: %w", err)
	}
	var out struct {
		Responses []struct {
			Status int `json:"status"`
		} `json:"responses"`
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return fmt.Errorf("删除文件响应解析失败: %w", err)
	}
	for _, r := range out.Responses {
		if r.Status != 200 {
			return fmt.Errorf("删除文件失败: 批量响应状态 %d", r.Status)
		}
	}
	return nil
}

// request 发起带鉴权的 API 请求；401 令牌失效时自动刷新重试一次
func (c *Client) request(uri string, body interface{}) ([]byte, error) {
	if err := c.ensureToken(); err != nil {
		return nil, err
	}
	payload, err := c.do(uri, body)
	if err != nil {
		if isTokenInvalidErr(err) {
			if _, rerr := c.Refresh(); rerr != nil {
				return nil, rerr
			}
			return c.do(uri, body)
		}
		return nil, err
	}
	return payload, nil
}

// isTokenInvalidErr 判断错误是否为 access_token 失效/过期（HTTP 401 + 响应体含 AccessToken* 错误码）
func isTokenInvalidErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "accesstoken") && strings.Contains(msg, "401")
}

func (c *Client) do(uri string, body interface{}) ([]byte, error) {
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, apiBase+uri, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("authorization", "Bearer "+c.AccessToken)
	req.Header.Set("content-type", "application/json")
	req.Header.Set("user-agent", userAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("阿里云盘 API HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	return payload, nil
}

func postJSON(rawURL string, body interface{}, headers map[string]string) ([]byte, error) {
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, rawURL, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("user-agent", userAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	return payload, nil
}

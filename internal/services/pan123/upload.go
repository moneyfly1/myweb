package pan123

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// 上传相关响应结构（对齐 123 云盘网页版 API / OpenList 驱动）

type uploadResp struct {
	Data struct {
		AccessKeyId     string `json:"AccessKeyId"`
		Bucket          string `json:"Bucket"`
		Key             string `json:"Key"`
		SecretAccessKey string `json:"SecretAccessKey"`
		SessionToken    string `json:"SessionToken"`
		FileId          int64  `json:"FileId"`
		Reuse           bool   `json:"Reuse"`
		EndPoint        string `json:"EndPoint"`
		StorageNode     string `json:"StorageNode"`
		UploadId        string `json:"UploadId"`
	} `json:"data"`
}

type uploadMeta struct {
	Bucket      string
	Key         string
	FileId      int64
	StorageNode string
	UploadId    string
}

type s3PreSignedURLs struct {
	Data struct {
		PreSignedUrls map[string]string `json:"presignedUrls"`
	} `json:"data"`
}

const uploadChunkSize = 16 << 20 // 16MB 分片，与 OpenList 一致

// CreateFolder 在网盘创建文件夹（type=1 的 upload_request），返回文件夹 fileId。
// 文件夹已存在时由调用方自行判断。
func (c *Client) CreateFolder(name string, parentFileID int64) (int64, error) {
	body := map[string]interface{}{
		"driveId":      0,
		"etag":         "",
		"fileName":     name,
		"parentFileId": parentFileID,
		"size":         0,
		"type":         1,
	}
	payload, err := c.request(http.MethodPost, "/file/upload_request", nil, body)
	if err != nil {
		return 0, fmt.Errorf("创建文件夹失败: %w", err)
	}
	var out uploadResp
	if err := json.Unmarshal(payload, &out); err != nil {
		return 0, fmt.Errorf("创建文件夹响应解析失败: %w", err)
	}
	if out.Data.FileId == 0 {
		return 0, errors.New("创建文件夹失败：未返回 FileId")
	}
	return out.Data.FileId, nil
}

// UploadFile 将本地文件上传到网盘（父目录 parentFileID，0=根目录），返回 fileId。
// 相同内容（md5 etag 命中）时接口会返回 Reuse=true，直接复用已有文件。
func (c *Client) UploadFile(localPath, fileName string, parentFileID int64) (int64, error) {
	fi, err := os.Stat(localPath)
	if err != nil {
		return 0, fmt.Errorf("读取本地文件失败: %w", err)
	}
	size := fi.Size()
	if size == 0 {
		return 0, errors.New("文件大小为 0，拒绝上传")
	}

	etag, err := fileMD5(localPath)
	if err != nil {
		return 0, err
	}

	// 1. 上传请求
	body := map[string]interface{}{
		"driveId":      0,
		"duplicate":    2, // 2=覆盖 1=重命名 0=默认
		"etag":         etag,
		"fileName":     fileName,
		"parentFileId": parentFileID,
		"size":         size,
		"type":         0,
	}
	payload, err := c.request(http.MethodPost, "/file/upload_request", nil, body)
	if err != nil {
		return 0, fmt.Errorf("上传请求失败: %w", err)
	}
	var up uploadResp
	if err := json.Unmarshal(payload, &up); err != nil {
		return 0, fmt.Errorf("上传响应解析失败: %w", err)
	}
	if up.Data.Reuse || (up.Data.FileId > 0 && up.Data.Key == "") {
		// 内容已存在，直接复用
		return up.Data.FileId, nil
	}
	if up.Data.FileId == 0 {
		return 0, errors.New("上传请求未返回 FileId")
	}

	meta := uploadMeta{
		Bucket:      up.Data.Bucket,
		Key:         up.Data.Key,
		FileId:      up.Data.FileId,
		StorageNode: up.Data.StorageNode,
		UploadId:    up.Data.UploadId,
	}

	// 2. S3 预签名分片上传（无需 AWS SDK，预签名 URL 直接 PUT）
	chunkCount := int((size + uploadChunkSize - 1) / uploadChunkSize)
	if err := c.uploadParts(meta, localPath, size, chunkCount); err != nil {
		return 0, err
	}

	// 3. 完成上传
	completeBody := map[string]interface{}{
		"StorageNode": meta.StorageNode,
		"bucket":      meta.Bucket,
		"fileId":      meta.FileId,
		"fileSize":    size,
		"isMultipart": chunkCount > 1,
		"key":         meta.Key,
		"uploadId":    meta.UploadId,
	}
	if _, err := c.request(http.MethodPost, "/file/upload_complete/v2", nil, completeBody); err != nil {
		return 0, fmt.Errorf("完成上传失败: %w", err)
	}
	return meta.FileId, nil
}

// uploadParts 分片上传：单片走 s3_upload_object/auth，多片走 s3_repare_upload_parts_batch（每批 10 片）
func (c *Client) uploadParts(meta uploadMeta, localPath string, size int64, chunkCount int) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer f.Close()

	batchSize := 1
	if chunkCount > 1 {
		batchSize = 10
	}

	for start := 1; start <= chunkCount; start += batchSize {
		end := start + batchSize
		if end > chunkCount+1 {
			end = chunkCount + 1
		}
		urls, err := c.fetchPartURLs(meta, start, end, chunkCount)
		if err != nil {
			return err
		}
		for part := start; part < end; part++ {
			offset := int64(part-1) * uploadChunkSize
			curSize := uploadChunkSize
			if part == chunkCount {
				curSize = int(size - offset)
			}
			status, err := c.putPart(f, urls[strconv.Itoa(part)], offset, curSize)
			if err != nil {
				return fmt.Errorf("第 %d 片上传失败: %w", part, err)
			}
			if status == http.StatusForbidden {
				// 预签名 URL 过期，刷新整批后重试一次
				urls2, err2 := c.fetchPartURLs(meta, start, end, chunkCount)
				if err2 != nil {
					return err2
				}
				status, err = c.putPart(f, urls2[strconv.Itoa(part)], offset, curSize)
				if err != nil {
					return fmt.Errorf("第 %d 片重试失败: %w", part, err)
				}
				if status != http.StatusOK {
					return fmt.Errorf("第 %d 片重试仍失败: HTTP %d", part, status)
				}
			} else if status != http.StatusOK {
				return fmt.Errorf("第 %d 片上传失败: HTTP %d", part, status)
			}
			// 分片上传也保持限频节奏
			c.mu.Lock()
			gap := time.Until(c.lastCall.Add(minRequestGap))
			c.mu.Unlock()
			if gap > 0 {
				time.Sleep(gap)
			}
		}
	}
	return nil
}

// putPart 上传单个分片，返回 HTTP 状态码
func (c *Client) putPart(f *os.File, presigned string, offset int64, curSize int) (int, error) {
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return 0, err
	}
	return c.doPut(presigned, io.LimitReader(f, int64(curSize)), int64(curSize))
}

// fetchPartURLs 获取分片预签名 URL；单片走 auth，多片走 presigned batch
func (c *Client) fetchPartURLs(meta uploadMeta, start, end, chunkCount int) (map[string]string, error) {
	body := map[string]interface{}{
		"StorageNode":     meta.StorageNode,
		"bucket":          meta.Bucket,
		"key":             meta.Key,
		"partNumberEnd":   end,
		"partNumberStart": start,
		"uploadId":        meta.UploadId,
	}
	endpoint := "/file/s3_upload_object/auth"
	if chunkCount > 1 {
		endpoint = "/file/s3_repare_upload_parts_batch"
	}
	payload, err := c.request(http.MethodPost, endpoint, nil, body)
	if err != nil {
		return nil, fmt.Errorf("获取分片预签名 URL 失败: %w", err)
	}
	var out s3PreSignedURLs
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, fmt.Errorf("分片 URL 响应解析失败: %w", err)
	}
	return out.Data.PreSignedUrls, nil
}

func (c *Client) doPut(url string, body io.Reader, contentLength int64) (int, error) {
	req, err := http.NewRequest(http.MethodPut, url, body)
	if err != nil {
		return 0, err
	}
	req.ContentLength = contentLength
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	resp.Body.Close()
	return resp.StatusCode, nil
}

func fileMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("打开文件计算 MD5 失败: %w", err)
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("计算 MD5 失败: %w", err)
	}
	return strings.ToLower(hex.EncodeToString(h.Sum(nil))), nil
}

// TrashFile 将文件移入回收站（用于清理）
func (c *Client) TrashFile(fileID int64) error {
	body := map[string]interface{}{
		"driveId":            0,
		"operation":          true,
		"fileTrashInfoList":  []map[string]interface{}{{"FileId": fileID}},
	}
	if _, err := c.request(http.MethodPost, "/file/trash", nil, body); err != nil {
		return fmt.Errorf("删除文件失败: %w", err)
	}
	return nil
}

package gitee

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type GiteeClient struct {
	Token      string
	Owner      string
	Repo       string
	BaseURL    string
	APIVersion string
}

func NewGiteeClient(token, owner, repo string) *GiteeClient {
	return &GiteeClient{
		Token:      token,
		Owner:      owner,
		Repo:       repo,
		BaseURL:    "https://gitee.com",
		APIVersion: "v5",
	}
}

func (c *GiteeClient) UploadFile(filePath, remotePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	contentBase64 := base64.StdEncoding.EncodeToString(fileContent)

	existingSHA, err := c.getFileSHA(remotePath)
	if err != nil {
		existingSHA = ""
	}

	apiURL := fmt.Sprintf("%s/api/%s/repos/%s/%s/contents/%s", c.BaseURL, c.APIVersion, c.Owner, c.Repo, remotePath)

	payload := map[string]interface{}{
		"content": contentBase64,
		"message": fmt.Sprintf("备份文件: %s", filepath.Base(remotePath)),
	}

	if existingSHA != "" {
		payload["sha"] = existingSHA
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+c.Token)

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("上传文件失败: %s, 响应: %s", resp.Status, string(body))
	}

	return nil
}

func (c *GiteeClient) getFileSHA(filePath string) (string, error) {
	apiURL := fmt.Sprintf("%s/api/%s/repos/%s/%s/contents/%s", c.BaseURL, c.APIVersion, c.Owner, c.Repo, filePath)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "token "+c.Token)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
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
		return "", err
	}

	if sha, ok := result["sha"].(string); ok {
		return sha, nil
	}

	return "", nil
}

func (c *GiteeClient) UploadBackup(filePath string) error {
	now := time.Now()
	dateFolder := now.Format("2006-01-02")
	fileName := filepath.Base(filePath)
	remotePath := fmt.Sprintf("%s/%s", dateFolder, fileName)

	return c.UploadFile(filePath, remotePath)
}

func (c *GiteeClient) TestConnection() error {
	apiURL := fmt.Sprintf("%s/api/%s/repos/%s/%s", c.BaseURL, c.APIVersion, c.Owner, c.Repo)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.Token)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
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

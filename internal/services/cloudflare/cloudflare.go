package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CloudflareService Cloudflare API服务
type CloudflareService struct {
	APIKey   string
	Email    string
	APIToken string // 优先使用API Token，如果没有则使用API Key + Email
	BaseURL  string
	Client   *http.Client
}

// NewCloudflareService 创建Cloudflare服务实例
func NewCloudflareService(apiKey, email, apiToken string) *CloudflareService {
	baseURL := "https://api.cloudflare.com/client/v4"
	if apiToken == "" {
		apiToken = "" // 如果没有token，使用key+email方式
	}
	
	return &CloudflareService{
		APIKey:   apiKey,
		Email:    email,
		APIToken: apiToken,
		BaseURL:  baseURL,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// makeRequest 发起API请求
func (s *CloudflareService) makeRequest(method, endpoint string, body interface{}) ([]byte, error) {
	url := s.BaseURL + endpoint
	
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}
	
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	
	// 设置认证头
	if s.APIToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.APIToken)
	} else {
		req.Header.Set("X-Auth-Key", s.APIKey)
		req.Header.Set("X-Auth-Email", s.Email)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	
	if resp.StatusCode >= 400 {
		var errorResp struct {
			Errors []struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(respBody, &errorResp); err == nil && len(errorResp.Errors) > 0 {
			return nil, fmt.Errorf("cloudflare API error: %s", errorResp.Errors[0].Message)
		}
		return nil, fmt.Errorf("cloudflare API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}
	
	return respBody, nil
}

// GetZoneID 获取域名Zone ID
func (s *CloudflareService) GetZoneID(domain string) (string, error) {
	endpoint := fmt.Sprintf("/zones?name=%s", domain)
	respBody, err := s.makeRequest("GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	
	var result struct {
		Result []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"result"`
	}
	
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	
	if len(result.Result) == 0 {
		return "", fmt.Errorf("domain %s not found", domain)
	}
	
	return result.Result[0].ID, nil
}

// CreateDNSRecord 创建DNS记录
func (s *CloudflareService) CreateDNSRecord(zoneID, recordType, name, content string, ttl int, proxied bool) (string, error) {
	endpoint := fmt.Sprintf("/zones/%s/dns_records", zoneID)
	
	record := map[string]interface{}{
		"type":    recordType,
		"name":    name,
		"content": content,
		"ttl":     ttl,
		"proxied": proxied,
	}
	
	respBody, err := s.makeRequest("POST", endpoint, record)
	if err != nil {
		return "", err
	}
	
	var result struct {
		Result struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	
	return result.Result.ID, nil
}

// UpdateDNSRecord 更新DNS记录
func (s *CloudflareService) UpdateDNSRecord(zoneID, recordID, recordType, name, content string, ttl int, proxied bool) error {
	endpoint := fmt.Sprintf("/zones/%s/dns_records/%s", zoneID, recordID)
	
	record := map[string]interface{}{
		"type":    recordType,
		"name":    name,
		"content": content,
		"ttl":     ttl,
		"proxied": proxied,
	}
	
	_, err := s.makeRequest("PUT", endpoint, record)
	return err
}

// DeleteDNSRecord 删除DNS记录
func (s *CloudflareService) DeleteDNSRecord(zoneID, recordID string) error {
	endpoint := fmt.Sprintf("/zones/%s/dns_records/%s", zoneID, recordID)
	_, err := s.makeRequest("DELETE", endpoint, nil)
	return err
}

// GetDNSRecords 获取DNS记录列表
func (s *CloudflareService) GetDNSRecords(zoneID, recordType, name string) ([]map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/zones/%s/dns_records", zoneID)
	if recordType != "" {
		endpoint += "?type=" + recordType
	}
	if name != "" {
		if recordType != "" {
			endpoint += "&name=" + name
		} else {
			endpoint += "?name=" + name
		}
	}
	
	respBody, err := s.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	
	var result struct {
		Result []map[string]interface{} `json:"result"`
	}
	
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	
	return result.Result, nil
}

// VerifyDNSRecord 验证DNS记录是否存在
func (s *CloudflareService) VerifyDNSRecord(zoneID, recordType, name string) (bool, string, error) {
	records, err := s.GetDNSRecords(zoneID, recordType, name)
	if err != nil {
		return false, "", err
	}
	
	for _, record := range records {
		if rName, ok := record["name"].(string); ok && rName == name {
			if rID, ok := record["id"].(string); ok {
				return true, rID, nil
			}
		}
	}
	
	return false, "", nil
}



package sms

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"cboard-go/internal/core/config"
)

// AliyunSMSService 阿里云短信服务
type AliyunSMSService struct {
	accessKeyID     string
	accessKeySecret string
	signName        string
	templateCode    string
	endpoint        string
}

// NewAliyunSMSService 创建阿里云短信服务
func NewAliyunSMSService() *AliyunSMSService {
	cfg := config.AppConfig
	return &AliyunSMSService{
		accessKeyID:     cfg.AliyunAccessKeyID,
		accessKeySecret: cfg.AliyunAccessKeySecret,
		signName:        cfg.AliyunSMSSignName,
		templateCode:    cfg.AliyunSMSTemplateCode,
		endpoint:        "dysmsapi.aliyuncs.com",
	}
}

// SendSMS 发送短信
func (s *AliyunSMSService) SendSMS(phone, code string) error {
	if s.accessKeyID == "" || s.accessKeySecret == "" {
		return fmt.Errorf("阿里云短信配置未完成")
	}

	params := map[string]string{
		"PhoneNumbers":  phone,
		"SignName":      s.signName,
		"TemplateCode":  s.templateCode,
		"TemplateParam": fmt.Sprintf(`{"code":"%s"}`, code),
	}

	resp, err := s.MakeRequest(params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("短信发送失败，状态码: %d", resp.StatusCode)
	}

	return nil
}

// SendVerificationCode 发送验证码
func (s *AliyunSMSService) SendVerificationCode(phone, code string) error {
	// 调用阿里云 API
	return s.SendSMS(phone, code)
}

// Sign 签名
func (s *AliyunSMSService) Sign(params map[string]string) string {
	// 排序参数
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 构建待签名字符串
	var signStr strings.Builder
	for i, k := range keys {
		if i > 0 {
			signStr.WriteString("&")
		}
		signStr.WriteString(specialURLEncode(k))
		signStr.WriteString("=")
		signStr.WriteString(specialURLEncode(params[k]))
	}

	// HMAC-SHA1 签名
	mac := hmac.New(sha1.New, []byte(s.accessKeySecret+"&"))
	mac.Write([]byte("GET&%2F&" + specialURLEncode(signStr.String())))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return signature
}

// specialURLEncode 特殊URL编码
func specialURLEncode(str string) string {
	encoded := url.QueryEscape(str)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}

// MakeRequest 发送请求
func (s *AliyunSMSService) MakeRequest(params map[string]string) (*http.Response, error) {
	if s.accessKeyID == "" || s.accessKeySecret == "" {
		return nil, fmt.Errorf("阿里云短信配置未完成")
	}
	// 添加公共参数
	params["AccessKeyId"] = s.accessKeyID
	params["Format"] = "JSON"
	params["SignatureMethod"] = "HMAC-SHA1"
	params["SignatureVersion"] = "1.0"
	params["SignatureNonce"] = fmt.Sprintf("%d", time.Now().UnixNano())
	params["Timestamp"] = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	params["Version"] = "2017-05-25"
	params["Action"] = "SendSms"

	// 签名
	params["Signature"] = s.Sign(params)

	// 构建请求URL
	query := url.Values{}
	for k, v := range params {
		query.Set(k, v)
	}

	requestURL := fmt.Sprintf("https://%s/?%s", s.endpoint, query.Encode())

	// 发送请求
	resp, err := http.Get(requestURL)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// Package pan123 封装 123 云盘（123pan / yun.123pan.com）网页版 API，
// 提供账号登录、文件名搜索、直链获取与分享链接生成能力。
//
// 接口与签名算法参考开源实现 alist（github.com/alist-org/alist，MIT License）
// 与 OpenList（github.com/OpenListTeam/OpenList），已按本项目结构重写。
package pan123

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// MainAPIBases 网页版 API 基地址；.cn 为部分用户可达的备用域名
	MainAPIBases = "https://yun.123pan.com/b/api|https://yun.123pan.cn/b/api"

	defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)"
	minRequestGap    = 700 * time.Millisecond // 与 alist/OpenList 一致的限频节奏，避免触发风控
)

// File 云盘文件信息（仅保留需要用到的字段）
type File struct {
	FileID    int64  `json:"fileId"`
	FileName  string `json:"fileName"`
	Size      int64  `json:"size"`
	Type      int    `json:"type"` // 0=文件 1=目录
	Etag      string `json:"etag"`
	S3KeyFlag string `json:"S3KeyFlag"`
	UpdateAt  string `json:"updateAt"`
}

// DisplaySize 人类可读的文件大小
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

// Client 123 云盘客户端。凭证只保存在调用方（本服务）内存中，不落盘。
// 认证优先级：Token（Bearer）> Cookies（Cookie 头）> 账号密码自动登录。
type Client struct {
	Username string
	Password string
	SharePwd string
	// Token 直接使用 Bearer token（可从浏览器 localStorage 获取）
	Token string
	// Cookies 直接使用浏览器 Cookie 头（可从 DevTools 复制）
	Cookies string

	mu       sync.Mutex
	token    string
	lastCall time.Time
	apiBases []string
	http     *http.Client
}

// New 创建一个新的 123 云盘客户端（账号密码模式）
func New(username, password, sharePwd string) *Client {
	return &Client{
		Username: strings.TrimSpace(username),
		Password: password,
		SharePwd: sharePwd,
		apiBases: strings.Split(MainAPIBases, "|"),
		http: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // 不自动跟随，由业务层处理 302
			},
		},
	}
}

// NewWithCookies 使用浏览器 Cookie 直接认证（跳过登录）。
// 若 Cookie 中包含 jwt=...（HttpOnly 登录 token），会自动提取作为 Bearer 使用。
func NewWithCookies(cookies, sharePwd string) *Client {
	c := New("", "", sharePwd)
	cookies = strings.TrimSpace(cookies)
	c.Cookies = cookies
	if jwt := extractCookieValue(cookies, "jwt"); jwt != "" {
		c.Token = jwt
	}
	return c
}

// extractCookieValue 从 Cookie 头字符串中提取指定 key 的值
func extractCookieValue(cookieHeader, key string) string {
	for _, part := range strings.Split(cookieHeader, ";") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && strings.EqualFold(strings.TrimSpace(kv[0]), key) {
			return strings.TrimSpace(kv[1])
		}
	}
	return ""
}

// NewWithToken 使用 Bearer token 直接认证（跳过登录）
func NewWithToken(token, sharePwd string) *Client {
	c := New("", "", sharePwd)
	c.Token = strings.TrimSpace(token)
	return c
}

// ---------------------------------------------------------------------------
// 签名算法（复刻自 alist drivers/123/util.go）
// ---------------------------------------------------------------------------

func signPath(path, os, version string) (string, string) {
	table := []byte{'a', 'd', 'e', 'f', 'g', 'h', 'l', 'm', 'y', 'i', 'j', 'n', 'o', 'p', 'k', 'q', 'r', 's', 't', 'u', 'b', 'c', 'v', 'w', 's', 'z'}
	random := fmt.Sprintf("%.f", math.Round(1e7*rand.Float64()))
	now := time.Now().In(time.FixedZone("CST", 8*3600))
	timestamp := fmt.Sprint(now.Unix())
	nowStr := []byte(now.Format("200601021504"))
	for i := 0; i < len(nowStr); i++ {
		nowStr[i] = table[nowStr[i]-48]
	}
	timeSign := fmt.Sprint(crc32.ChecksumIEEE(nowStr))
	data := strings.Join([]string{timestamp, random, path, os, version, timeSign}, "|")
	dataSign := fmt.Sprint(crc32.ChecksumIEEE([]byte(data)))
	return timeSign, strings.Join([]string{timestamp, random, dataSign}, "-")
}

func getSignedURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	k, v := signPath(u.Path, "web", "3")
	q.Add(k, v)
	u.RawQuery = q.Encode()
	return u.String()
}

// ---------------------------------------------------------------------------
// 底层请求
// ---------------------------------------------------------------------------

// request 发送带签名与鉴权的请求；401 时密码模式自动重新登录重试一次。
// 网络层失败或 cookie/token 模式鉴权失败时自动在备用域名间切换（.com → .cn）。
func (c *Client) request(method, rawURL string, query url.Values, body interface{}) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	usePassword := c.Username != "" && c.Password != "" && c.Cookies == "" && c.Token == ""
	if usePassword && c.token == "" {
		if err := c.loginLocked(); err != nil {
			return nil, err
		}
	}

	// 限频：同一客户端请求间隔不低于 minRequestGap
	if gap := time.Until(c.lastCall.Add(minRequestGap)); gap > 0 {
		time.Sleep(gap)
	}

	lastErr := error(nil)
	for _, base := range c.apiBases {
		full := base + rawURL
		if len(query) > 0 {
			full += "?" + query.Encode()
		}
		full = getSignedURL(full)

		resp, err := c.do(method, full, body, c.token, c.Cookies)
		if err != nil {
			lastErr = err
			continue // 网络层失败，尝试下一个域名
		}
		payload, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		var envelope struct {
			Code    int             `json:"code"`
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data"`
		}
		_ = json.Unmarshal(payload, &envelope)

		// 401：密码模式重新登录后重试一次；cookie/token 模式尝试下一个域名
		if resp.StatusCode == http.StatusUnauthorized || envelope.Code == 401 {
			if usePassword {
				if err := c.loginLocked(); err != nil {
					return nil, fmt.Errorf("登录失败: %w", err)
				}
				resp2, err2 := c.do(method, full, body, c.token, "")
				if err2 != nil {
					lastErr = err2
					continue
				}
				payload2, err2 := io.ReadAll(resp2.Body)
				resp2.Body.Close()
				if err2 != nil {
					return nil, err2
				}
				var env2 struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				}
				_ = json.Unmarshal(payload2, &env2)
				if env2.Code != 0 {
					return nil, fmt.Errorf("123云盘接口错误: %s", firstNonEmpty(env2.Message, "code="+strconv.Itoa(env2.Code)))
				}
				c.lastCall = time.Now()
				return payload2, nil
			}
			lastErr = fmt.Errorf("鉴权失败（401），请检查登录凭证（Cookie/Token）是否有效")
			continue
		}

		if envelope.Code != 0 {
			return nil, fmt.Errorf("123云盘接口错误: %s", firstNonEmpty(envelope.Message, "code="+strconv.Itoa(envelope.Code)))
		}
		c.lastCall = time.Now()
		return payload, nil
	}
	if lastErr == nil {
		lastErr = errors.New("所有 API 域名均不可达")
	}
	return nil, lastErr
}

func (c *Client) do(method, fullURL string, body interface{}, token, cookies string) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = strings.NewReader(string(b))
	}
	req, err := http.NewRequest(method, fullURL, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("origin", "https://yun.123pan.com")
	req.Header.Set("referer", "https://yun.123pan.com/")
	req.Header.Set("user-agent", defaultUserAgent)
	req.Header.Set("platform", "web")
	req.Header.Set("app-version", "3")
	if token != "" {
		req.Header.Set("authorization", "Bearer "+token)
	}
	if cookies != "" {
		req.Header.Set("cookie", cookies)
	}
	if body != nil {
		req.Header.Set("content-type", "application/json")
	}
	return c.http.Do(req)
}

// loginLocked 登录并保存 token（调用方需持有 c.mu）
func (c *Client) loginLocked() error {
	body := map[string]interface{}{}
	if strings.Contains(c.Username, "@") {
		body = map[string]interface{}{"mail": c.Username, "password": c.Password, "type": 2}
	} else {
		body = map[string]interface{}{"passport": c.Username, "password": c.Password, "remember": true}
	}
	loginBases := []string{"https://login.123pan.com", "https://login.123pan.cn"}
	var lastErr error
	for _, base := range loginBases {
		req, err := http.NewRequest(http.MethodPost, base+"/api/user/sign_in", strings.NewReader(mustJSON(body)))
		if err != nil {
			return err
		}
		req.Header.Set("origin", "https://yun.123pan.com")
		req.Header.Set("referer", "https://yun.123pan.com/")
		req.Header.Set("user-agent", defaultUserAgent)
		req.Header.Set("platform", "web")
		req.Header.Set("app-version", "3")
		req.Header.Set("content-type", "application/json")

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			continue // 网络层失败，尝试另一个登录域名
		}
		payload, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}
		var out struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    struct {
				Token string `json:"token"`
			} `json:"data"`
		}
		if err := json.Unmarshal(payload, &out); err != nil {
			return fmt.Errorf("登录响应解析失败: %w", err)
		}
		if out.Code != 200 || out.Data.Token == "" {
			return fmt.Errorf("登录失败: %s", firstNonEmpty(out.Message, "请检查账号密码"))
		}
		c.token = out.Data.Token
		c.lastCall = time.Now()
		return nil
	}
	if lastErr == nil {
		lastErr = errors.New("登录域名不可达")
	}
	return fmt.Errorf("登录失败: %w", lastErr)
}

func mustJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func firstNonEmpty(items ...string) string {
	for _, s := range items {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// 业务接口
// ---------------------------------------------------------------------------

// Test 测试登录是否可用，返回账号信息
func (c *Client) Test() (map[string]interface{}, error) {
	raw, err := c.request(http.MethodGet, "/user/info", nil, nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		Data struct {
			Nickname string `json:"nickname"`
			VipType  int    `json:"vipType"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &out)
	return map[string]interface{}{"nickname": out.Data.Nickname, "vipType": out.Data.VipType}, nil
}

// ShareGet 获取分享的文件列表（公开分享免登录，也用于验证签名算法）
func (c *Client) ShareGet(shareKey, sharePwd string) ([]File, error) {
	query := url.Values{}
	query.Set("limit", "100")
	query.Set("next", "0")
	query.Set("orderBy", "file_id")
	query.Set("orderDirection", "desc")
	query.Set("parentFileId", "0")
	query.Set("Page", "1")
	query.Set("shareKey", shareKey)
	query.Set("SharePwd", sharePwd)

	raw, err := c.request(http.MethodGet, "/share/get", query, nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		Data struct {
			InfoList []File `json:"InfoList"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("分享列表解析失败: %w\nraw: %s", err, truncateBytes(raw, 4000))
	}
	return out.Data.InfoList, nil
}

func truncateBytes(b []byte, n int) string {
	if len(b) > n {
		return string(b[:n]) + "..."
	}
	return string(b)
}

// SearchFiles 按文件名关键词搜索文件，返回按更新时间倒序的文件列表
func (c *Client) SearchFiles(keyword string, limit int) ([]File, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	query := url.Values{}
	query.Set("driveId", "0")
	query.Set("limit", strconv.Itoa(limit))
	query.Set("next", "0")
	query.Set("orderBy", "file_id")
	query.Set("orderDirection", "desc")
	query.Set("parentFileId", "0")
	query.Set("trashed", "false")
	query.Set("SearchData", keyword)
	query.Set("Page", "1")
	query.Set("OnlyLookAbnormalFile", "0")
	query.Set("event", "searchFile")
	query.Set("operateType", "4")
	query.Set("inDirectSpace", "false")

	raw, err := c.request(http.MethodGet, "/file/list/new", query, nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		Data struct {
			InfoList []File `json:"InfoList"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("搜索响应解析失败: %w", err)
	}
	return out.Data.InfoList, nil
}

// SearchFilesByExt 按关键词搜索并按扩展名过滤（大小写不敏感），
// 结果按更新时间倒序（同时间按 fileId 倒序），用于同一软件多平台/多版本的场景。
// ext 为空表示不过滤。
func (c *Client) SearchFilesByExt(keyword, ext string, limit int) ([]File, error) {
	files, err := c.SearchFiles(keyword, limit)
	if err != nil {
		return nil, err
	}
	ext = strings.TrimSpace(strings.ToLower(ext))
	out := make([]File, 0, len(files))
	for _, f := range files {
		if ext != "" && !strings.HasSuffix(strings.ToLower(f.FileName), "."+ext) {
			continue
		}
		out = append(out, f)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].UpdateAt != out[j].UpdateAt {
			return out[i].UpdateAt > out[j].UpdateAt
		}
		return out[i].FileID > out[j].FileID
	})
	return out, nil
}

// GetDirectLink 获取文件的直链（resolve 到最终可下载地址）。
// 逻辑复刻 OpenList 123 驱动：download_info → 可能 base64 的 params → 带 Referer 跟随 302。
func (c *Client) GetDirectLink(f File) (string, error) {
	payload, err := c.request(http.MethodPost, "/file/download_info", nil, map[string]interface{}{
		"driveId":   0,
		"etag":      f.Etag,
		"fileId":    f.FileID,
		"fileName":  f.FileName,
		"s3keyFlag": f.S3KeyFlag,
		"size":      f.Size,
		"type":      f.Type,
	})
	if err != nil {
		return "", err
	}
	var out struct {
		Data struct {
			DownloadURL string `json:"DownloadUrl"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return "", fmt.Errorf("直链响应解析失败: %w", err)
	}
	rawURL := strings.TrimSpace(out.Data.DownloadURL)
	if rawURL == "" {
		return "", errors.New("直链为空")
	}

	// 旧协议：DownloadUrl 的 params 参数为 base64 编码的真实地址
	if u, err := url.Parse(rawURL); err == nil {
		if encoded := u.Query().Get("params"); encoded != "" {
			if decoded, derr := base64.StdEncoding.DecodeString(encoded); derr == nil {
				if du, perr := url.Parse(string(decoded)); perr == nil {
					rawURL = du.String()
				}
			}
		}
	}

	return c.resolveFinalURL(rawURL)
}

// shareExpirationValue 分享有效期：默认永久（0）。
func shareExpirationValue() interface{} {
	return 0
}

// GetShareLink 创建分享并返回分享下载链接（实验性，未完整验证）。
// 注意：当前以直链为主要取链方式；分享创建的完整参数（expiration 格式等）依赖官方网页端行为，可能需按版本调整。
func (c *Client) GetShareLink(f File) (string, error) {
	sharePwd := strings.TrimSpace(c.SharePwd)
	usePwd := 0
	if sharePwd != "" {
		usePwd = 1
	}
	expiration := shareExpirationValue()
	createBody := map[string]interface{}{
		"driveId":        0,
		"fileIdList":     []int64{f.FileID},
		"sharePwd":       sharePwd,
		"shareDelayTime": 0,
		"shareName":      f.FileName,
		"expiration":     expiration,
		"useSharePwd":    usePwd,
	}
	payload, err := c.request(http.MethodPost, "/share/create", nil, createBody)
	if err != nil {
		return "", fmt.Errorf("创建分享失败: %w", err)
	}
	var created struct {
		Data struct {
			ShareKey  string `json:"ShareKey"`
			SharePwd  string `json:"SharePwd"`
			ShareLink string `json:"ShareLink"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &created); err != nil {
		return "", fmt.Errorf("分享响应解析失败: %w", err)
	}
	shareKey := strings.TrimSpace(created.Data.ShareKey)
	if shareKey == "" {
		return "", errors.New("创建分享失败：未返回 ShareKey")
	}
	if created.Data.SharePwd != "" {
		sharePwd = created.Data.SharePwd
	}

	infoBody := map[string]interface{}{
		"shareKey": shareKey,
		"sharePwd": sharePwd,
		"fileId":   f.FileID,
	}
	infoPayload, err := c.request(http.MethodPost, "/share/download/info", nil, infoBody)
	if err != nil {
		return "", fmt.Errorf("获取分享下载链接失败: %w", err)
	}
	var info struct {
		Data struct {
			DownloadURL string `json:"DownloadUrl"`
		} `json:"data"`
	}
	if err := json.Unmarshal(infoPayload, &info); err != nil {
		return "", fmt.Errorf("分享下载链接解析失败: %w", err)
	}
	dl := strings.TrimSpace(info.Data.DownloadURL)
	if dl == "" {
		return "", errors.New("分享下载链接为空")
	}
	return dl, nil
}

// GetDownloadLink 按模式获取最终可下载链接
func (c *Client) GetDownloadLink(f File, mode string) (string, error) {
	if mode == "direct" {
		return c.GetDirectLink(f)
	}
	return c.GetShareLink(f)
}

// resolveFinalURL 带 Referer 跟随一次 302/JSON 跳转，拿到最终 CDN 地址。
// 返回的地址即浏览器可直接下载的 URL。
func (c *Client) resolveFinalURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("直链地址非法: %w", err)
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("referer", "https://yun.123pan.com/")
	req.Header.Set("user-agent", defaultUserAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		// 网络不可达时退回原直链，交给前端浏览器尝试
		return rawURL, nil
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusTemporaryRedirect:
		if loc := resp.Header.Get("location"); loc != "" {
			return loc, nil
		}
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		var j struct {
			Data struct {
				RedirectURL string `json:"redirect_url"`
			} `json:"data"`
		}
		if json.Unmarshal(body, &j) == nil && j.Data.RedirectURL != "" {
			return j.Data.RedirectURL, nil
		}
	}
	return rawURL, nil
}

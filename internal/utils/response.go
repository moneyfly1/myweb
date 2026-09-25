package utils

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// APIResponse 统一API响应格式
type APIResponse struct {
	Success   bool        `json:"success"`
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp int64       `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
	// Reason 机器可读的失败原因（如 "vps_occupied"），供前端做分支处理。
	// code 恒为整数（与 HTTP 状态语义一致），业务原因不再挤进 code 字段，
	// 避免同一字段有时是整数、有时是字符串。
	Reason string `json:"reason,omitempty"`
}

// Standard error codes
const (
	ErrCodeSuccess      = 0
	ErrCodeBadRequest   = 400
	ErrCodeUnauthorized = 401
	ErrCodeForbidden    = 403
	ErrCodeNotFound     = 404
	ErrCodeInternal     = 500
)

// GetRequestID 从上下文中获取请求ID
func GetRequestID(c *gin.Context) string {
	if rid, exists := c.Get("request_id"); exists {
		if s, ok := rid.(string); ok {
			return s
		}
	}
	return ""
}

// IsProduction 检查是否为生产环境
func IsProduction() bool {
	env := os.Getenv("ENV")
	return env == "production" || env == "prod"
}

func SuccessResponse(c *gin.Context, code int, message string, data interface{}) {
	resp := APIResponse{
		Success:   true,
		Code:      ErrCodeSuccess,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Unix(),
		RequestID: GetRequestID(c),
	}
	c.JSON(code, resp)
}

func ErrorResponse(c *gin.Context, code int, message string, err error) {
	// 确定错误码
	errCode := getErrorCode(code)

	// 记录详细错误到日志
	if err != nil {
		LogErrorWithStatus(code, message, err, map[string]interface{}{
			"path":     c.Request.URL.Path,
			"method":   c.Request.Method,
			"code":     code,
			"err_code": errCode,
		})
	}

	// 系统错误记录到审计日志
	if code >= http.StatusInternalServerError {
		alreadyLogged := false
		if logged, exists := c.Get("system_error_logged"); exists {
			if v, ok := logged.(bool); ok && v {
				alreadyLogged = true
			}
		}
		if !alreadyLogged {
			CreateSystemErrorLog(c, code, message, err)
			c.Set("system_error_logged", true)
		}
	}

	// 生产环境：隐藏详细错误信息，使用通用错误消息
	userMessage := message
	if IsProduction() && err != nil {
		// 生产环境不返回详细错误信息
		if code >= http.StatusInternalServerError {
			userMessage = "服务器内部错误，请稍后重试"
		} else if strings.Contains(strings.ToLower(message), "error") ||
			strings.Contains(strings.ToLower(err.Error()), "path") ||
			strings.Contains(strings.ToLower(err.Error()), "file") {
			// 如果错误信息包含路径、文件等敏感信息，使用通用消息
			userMessage = "操作失败，请稍后重试"
		}
	}

	resp := APIResponse{
		Success:   false,
		Code:      errCode,
		Message:   userMessage,
		Timestamp: time.Now().Unix(),
		RequestID: GetRequestID(c),
	}
	c.JSON(code, resp)
}

// ErrorResponseWithData 与 ErrorResponse 相同的统一格式，但携带业务数据与机器可读原因。
// 用于"HTTP 状态码 + 业务原因 + 附加数据"这类需要前端分支处理的失败响应
// （例：409 + reason=vps_occupied + data.existing_node_id），
// 替代此前手写 c.JSON 并把 code 写成字符串 "vps_occupied" 的做法。
func ErrorResponseWithData(c *gin.Context, code int, reason, message string, data interface{}) {
	resp := APIResponse{
		Success:   false,
		Code:      getErrorCode(code),
		Message:   message,
		Data:      data,
		Reason:    reason,
		Timestamp: time.Now().Unix(),
		RequestID: GetRequestID(c),
	}
	c.JSON(code, resp)
}

// getErrorCode 根据HTTP状态码返回对应的业务错误码
func getErrorCode(statusCode int) int {
	switch statusCode {
	case http.StatusBadRequest:
		return ErrCodeBadRequest
	case http.StatusUnauthorized:
		return ErrCodeUnauthorized
	case http.StatusForbidden:
		return ErrCodeForbidden
	case http.StatusNotFound:
		return ErrCodeNotFound
	case http.StatusInternalServerError:
		return ErrCodeInternal
	default:
		return statusCode
	}
}

// ========== 分页相关 ==========

type PaginationParams struct {
	Page int
	Size int
}

func ParsePagination(c *gin.Context) PaginationParams {
	page := 1
	size := 10

	if pageStr := c.Query("page"); pageStr != "" {
		if _, err := fmt.Sscanf(pageStr, "%d", &page); err != nil {
			page = 1
		}
	}
	if sizeStr := c.Query("size"); sizeStr != "" {
		if _, err := fmt.Sscanf(sizeStr, "%d", &size); err != nil {
			size = 10
		}
	}
	// 兼容 page_size 参数
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if _, err := fmt.Sscanf(pageSizeStr, "%d", &size); err == nil {
			// page_size 优先
		}
	}

	// 先解析 limit（它是 size 的别名），再据此换算 skip → page。
	// 历史缺陷：原先先算 page 再覆盖 size，且换算是 (skip/size)+1 用的还是**旧 size**，
	// 于是 pageSize=20/50/100 时 offset 被算成 40/250/1000（应为 20/50/100），
	// 管理端切换每页条数后订单行会整段漏显。
	if limitStr := c.Query("limit"); limitStr != "" {
		var limit int
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil {
			limit = 10
		}
		if limit > 0 {
			size = limit
		}
	}
	if skipStr := c.Query("skip"); skipStr != "" {
		var skip int
		if _, err := fmt.Sscanf(skipStr, "%d", &skip); err != nil {
			skip = 0
		}
		if skip < 0 {
			skip = 0
		}
		if skip > 100000 {
			skip = 100000
		}
		page = (skip / size) + 1
	}

	if page < 1 {
		page = 1
	}
	if page > 10000 {
		page = 10000
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}

	return PaginationParams{Page: page, Size: size}
}

func (p PaginationParams) GetOffset() int {
	return (p.Page - 1) * p.Size
}

// ParsePaginationWithDefaultSize 解析分页参数；当请求未携带任何分页参数
// （page/size/page_size/limit/skip 均未提供）时，使用 defaultSize 作为每页条数，
// 用于保持各接口原有的默认 size（如 20/100/12）语义不变。
func ParsePaginationWithDefaultSize(c *gin.Context, defaultSize int) PaginationParams {
	p := ParsePagination(c)
	if c.Query("size") == "" && c.Query("page_size") == "" && c.Query("limit") == "" && c.Query("skip") == "" {
		p.Size = defaultSize
	}
	return p
}

// 敏感字段列表，这些字段在日志中会被隐藏
var sensitiveFields = map[string]bool{
	"password":         true,
	"token":            true,
	"secret":           true,
	"api_key":          true,
	"api_key_id":       true,
	"access_token":     true,
	"refresh_token":    true,
	"csrf_token":       true,
	"session_id":       true,
	"private_key":      true,
	"public_key":       true,
	"secret_key":       true,
	"encryption_key":   true,
	"decryption_key":   true,
	"auth_token":       true,
	"bearer_token":     true,
	"jwt_token":        true,
	"session_token":    true,
	"api_secret":       true,
	"client_secret":    true,
	"app_secret":       true,
	"webhook_secret":   true,
	"signature":        true,
	"hmac":             true,
	"credential":       true,
	"credentials":      true,
	"subscription_url": true, // 订阅URL也是敏感信息
	"subscription_key": true,
}

func LogError(operation string, err error, context map[string]interface{}) {
	logErrorAt(logLevelError, operation, err, context)
}

// 日志级别：ErrorResponse 依据 HTTP 状态码选级，避免 4xx 噪声淹没 5xx 真故障。
const (
	logLevelWarn  = 1
	logLevelError = 2
)

// 4xx 去重窗口（秒）：同一 (状态码|路径|消息) 在窗口内只记一条。
// 审计证据：POST /api/v1/agent/heartbeat 曾刷出 58,178 条 404 ERROR，占全部 ERROR 的 79%。
const logSuppressWindowSec = 60

// 去重表上限，超过则整体清空，保证内存有界（宁可多记几条也不无限增长）。
const logSuppressMaxEntries = 4096

var (
	logSuppressMu   sync.Mutex
	logSuppressLast = make(map[string]int64)
)

// shouldLogOnce 返回该日志键是否应当写入（窗口内去重）。
func shouldLogOnce(key string) bool {
	nowUnix := time.Now().Unix()
	logSuppressMu.Lock()
	defer logSuppressMu.Unlock()
	if last, ok := logSuppressLast[key]; ok && nowUnix-last < logSuppressWindowSec {
		return false
	}
	if len(logSuppressLast) >= logSuppressMaxEntries {
		logSuppressLast = make(map[string]int64)
	}
	logSuppressLast[key] = nowUnix
	return true
}

// LogErrorWithStatus 按状态码分级：>=500 → ERROR；其余（4xx，即客户端可预期结果：
// 令牌过期、资源不存在、参数错误）→ WARN，并对同一 path 做窗口内去重。
func LogErrorWithStatus(code int, operation string, err error, context map[string]interface{}) {
	if code >= http.StatusInternalServerError {
		logErrorAt(logLevelError, operation, err, context)
		return
	}
	path := ""
	if context != nil {
		if v, ok := context["path"].(string); ok {
			path = v
		}
	}
	if !shouldLogOnce(fmt.Sprintf("%d|%s|%s", code, path, operation)) {
		return
	}
	logErrorAt(logLevelWarn, operation, err, context)
}

func logErrorAt(level int, operation string, err error, context map[string]interface{}) {
	if err == nil {
		return
	}

	// 过滤错误信息中的敏感路径
	errMsg := err.Error()
	if errMsg != "" {
		// 移除文件路径中的敏感信息
		errMsg = sanitizeErrorPath(errMsg)
	}

	msg := fmt.Sprintf("Operation: %s, Error: %v", operation, errMsg)
	if context != nil {
		safeContext := make(map[string]interface{})
		for k, v := range context {
			// 检查字段名（不区分大小写）
			keyLower := strings.ToLower(k)
			if sensitiveFields[keyLower] {
				safeContext[k] = "***REDACTED***"
			} else {
				// 检查值中是否包含敏感信息
				if strVal, ok := v.(string); ok {
					safeContext[k] = sanitizeSensitiveValue(strVal)
				} else {
					safeContext[k] = v
				}
			}
		}
		msg += fmt.Sprintf(", Context: %+v", safeContext)
	}

	if AppLogger != nil {
		if level == logLevelError {
			AppLogger.Error("%s", msg)
		} else {
			AppLogger.Warn("%s", msg)
		}
		return
	}
	if level == logLevelError {
		log.Printf("[ERROR] %s", msg)
	} else {
		log.Printf("[WARN] %s", msg)
	}
}

// sanitizeErrorPath 清理错误信息中的文件路径
func sanitizeErrorPath(errMsg string) string {
	// 移除绝对路径，只保留文件名
	// 例如: /Users/apple/Downloads/goweb/file.go -> file.go
	parts := strings.Split(errMsg, "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		// 如果包含文件名，尝试提取
		if strings.Contains(lastPart, ".") {
			// 保留最后两个部分（目录名和文件名），只有一段时直接返回该段
			if len(parts) >= 2 {
				return strings.Join(parts[len(parts)-2:], "/")
			}
			return lastPart
		}
	}
	return errMsg
}

// sanitizeSensitiveValue 清理值中的敏感信息
func sanitizeSensitiveValue(value string) string {
	valueLower := strings.ToLower(value)
	// 检查是否包含敏感关键词
	for field := range sensitiveFields {
		if strings.Contains(valueLower, field) {
			return "***REDACTED***"
		}
	}
	// 检查是否看起来像token或密钥（长字符串）
	if len(value) > 20 && (strings.Contains(value, "-") || strings.Contains(value, "_")) {
		// 可能是token或密钥，部分隐藏
		if len(value) > 40 {
			return value[:10] + "..." + value[len(value)-10:]
		}
	}
	return value
}

// PaginatedList 构造统一的分页响应体。
//
// 为什么需要它：历史上各接口的列表字段名多达 15 种（list/items/logs/attempts/
// subscriptions/records/orders/users/emails/tickets/coupons/relations/invite_codes/
// recharges…），分页元数据也有 size/page_size、total_pages/pages 两套命名，
// 前端被迫写 100 多处 `data.logs || data.list || data.items` 之类的兜底解包。
//
// 现在的约定：
//   - 标准字段：list（前端统一按 list 解包，见 frontend unwrapList）
//   - 兼容字段：legacyKey（旧字段名继续返回，老前端/老书签不受影响）
//   - 分页元数据同时给出两套命名，避免调用方各写一遍
//
// 用法：
//
//	payload := utils.PaginatedList(items, "orders", total, page, size)
//	payload["summary"] = extra
//	utils.SuccessResponse(c, http.StatusOK, "", payload)
func PaginatedList(list any, legacyKey string, total int64, page, size int) gin.H {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(size) - 1) / int64(size))
	}
	if totalPages < 1 {
		totalPages = 1
	}
	if list == nil {
		list = []any{}
	}

	payload := gin.H{
		"list":        list,
		"total":       total,
		"page":        page,
		"size":        size, // 兼容旧命名
		"page_size":   size, // 兼容旧命名
		"total_pages": totalPages,
		"pages":       totalPages, // 兼容旧命名
	}
	if legacyKey != "" && legacyKey != "list" {
		payload[legacyKey] = list
	}
	return payload
}

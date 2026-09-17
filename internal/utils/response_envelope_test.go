package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestErrorResponseWithDataKeepsCodeInt 锁定响应契约：
// code 必须是整数（与 HTTP 状态语义一致），业务原因放在 reason 字段，
// 附加数据放 data。历史上 selfhost 的 409 直接把 code 写成字符串 "vps_occupied"，
// 导致同一字段时而整数时而字符串，前端只能按字符串比较。
func TestErrorResponseWithDataKeepsCodeInt(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	ErrorResponseWithData(c, http.StatusConflict, "vps_occupied", "该 VPS 已部署自建节点", gin.H{
		"existing_node_id": 7,
	})

	if recorder.Code != http.StatusConflict {
		t.Errorf("HTTP 状态码应为 409，实际 %d", recorder.Code)
	}

	var payload struct {
		Success bool                   `json:"success"`
		Code    json.RawMessage        `json:"code"`
		Reason  string                 `json:"reason"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("响应不是合法 JSON: %v (%s)", err, recorder.Body.String())
	}

	// code 必须是数字，不能是字符串
	var codeInt int
	if err := json.Unmarshal(payload.Code, &codeInt); err != nil {
		t.Fatalf("code 必须是整数，实际收到 %s", payload.Code)
	}
	if codeInt != http.StatusConflict {
		t.Errorf("未映射的业务码沿用 HTTP 状态码，409 时应为 %d，实际 %d", http.StatusConflict, codeInt)
	}
	if payload.Reason != "vps_occupied" {
		t.Errorf("reason 应为 vps_occupied，实际 %q", payload.Reason)
	}
	if payload.Success {
		t.Error("失败响应 success 应为 false")
	}
	if payload.Data["existing_node_id"] == nil {
		t.Error("data 必须保留 existing_node_id，前端依赖它提示复用")
	}
}

// TestErrorResponseOmitsReasonWhenEmpty 普通错误响应不应凭空多出 reason 字段
func TestErrorResponseOmitsReasonWhenEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	ErrorResponse(c, http.StatusBadRequest, "参数错误", nil)

	if body := recorder.Body.String(); json.Valid(recorder.Body.Bytes()) {
		var raw map[string]json.RawMessage
		_ = json.Unmarshal(recorder.Body.Bytes(), &raw)
		if _, exists := raw["reason"]; exists {
			t.Errorf("reason 为空时不应出现在响应里: %s", body)
		}
	} else {
		t.Fatalf("响应不是合法 JSON: %s", body)
	}
}

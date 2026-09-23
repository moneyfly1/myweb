package models

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestJSONStringMarshalsAsPlainString 回归：可空文本字段绝不能序列化成
// {"String":"…","Valid":true} —— 线上教程摘要曾整串花括号渲染到页面上。
func TestJSONStringMarshalsAsPlainString(t *testing.T) {
	cases := []struct {
		name  string
		value JSONString
		want  string
	}{
		{"有值", NewJSONString("界面简洁的 Clash 客户端"), `"界面简洁的 Clash 客户端"`},
		{"NULL 输出空串", JSONString{}, `""`},
		{"显式空串", NullableJSONString(""), `""`},
	}
	for _, c := range cases {
		got, err := json.Marshal(c.value)
		if err != nil {
			t.Fatalf("%s: 序列化失败 %v", c.name, err)
		}
		if string(got) != c.want {
			t.Errorf("%s: got %s want %s", c.name, got, c.want)
		}
		if strings.Contains(string(got), "Valid") || strings.Contains(string(got), "String") {
			t.Errorf("%s: 仍出现 NullString 结构体痕迹: %s", c.name, got)
		}
	}
}

// TestKnowledgeArticleSummaryJSON 文章对象里 summary 必须是字符串
func TestKnowledgeArticleSummaryJSON(t *testing.T) {
	article := KnowledgeArticle{
		ID:      1,
		Title:   "Clash Verge 使用教程",
		Content: "<p>…</p>",
		Summary: NewJSONString("界面简洁的 Clash 客户端，支持 Windows 与 macOS。"),
	}
	raw, err := json.Marshal(article)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}
	if _, isString := decoded["summary"].(string); !isString {
		t.Fatalf("summary 应为字符串，实际 %T: %v", decoded["summary"], decoded["summary"])
	}
	if !strings.Contains(string(raw), `"summary":"界面简洁的 Clash 客户端`) {
		t.Errorf("summary 文本不正确: %s", string(raw))
	}
}

// TestJSONStringScanAndValue 数据库读写：NULL ↔ 空值、字符串 ↔ 字符串
func TestJSONStringScanAndValue(t *testing.T) {
	var s JSONString
	if err := s.Scan(nil); err != nil || s.Valid {
		t.Fatalf("Scan(nil) 应得到无效值: %+v err=%v", s, err)
	}
	if v, err := s.Value(); err != nil || v != nil {
		t.Fatalf("NULL 应写入 nil: %v err=%v", v, err)
	}
	if err := s.Scan([]byte("正文摘要")); err != nil || s.String != "正文摘要" || !s.Valid {
		t.Fatalf("Scan([]byte) 结果错误: %+v err=%v", s, err)
	}
	if v, err := s.Value(); err != nil || v != "正文摘要" {
		t.Fatalf("Value() 结果错误: %v err=%v", v, err)
	}
}

// TestJSONStringFromRequest 请求体绑定：字符串与 null 都能接受
func TestJSONStringFromRequest(t *testing.T) {
	var s JSONString
	if err := json.Unmarshal([]byte(`"用户提交的摘要"`), &s); err != nil || s.String != "用户提交的摘要" {
		t.Fatalf("字符串绑定失败: %+v err=%v", s, err)
	}
	if err := json.Unmarshal([]byte(`null`), &s); err != nil || s.Valid {
		t.Fatalf("null 绑定应得到无效值: %+v err=%v", s, err)
	}
}

// TestPromotionNullableFields 营销活动同样对外展示，这两个字段也必须是字符串
func TestPromotionNullableFields(t *testing.T) {
	p := Promotion{Name: "限时活动", PackageIDs: NullableJSONString("1,2,3"), Description: NewJSONString("新用户专享")}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, `"package_ids":"1,2,3"`) {
		t.Errorf("package_ids 不是字符串: %s", text)
	}
	if !strings.Contains(text, `"description":"新用户专享"`) {
		t.Errorf("description 不是字符串: %s", text)
	}
	for _, bad := range []string{"Valid", `"String"`} {
		if strings.Contains(text, bad) {
			t.Errorf("出现 NullString 痕迹 %q: %s", bad, text)
		}
	}
}

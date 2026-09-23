package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

// JSONString 是「数据库里可为 NULL、但 JSON 必须是普通字符串」的文本字段。
//
// 为什么不能用 sql.NullString：它的 JSON 编码是 {"String":"…","Valid":true}，
// 序列化进接口后前端会把这串花括号当正文渲染出来。线上就出现过：
// 客户端中心的教程摘要显示成
//
//	{ "String": "界面简洁的 Clash 客户端…", "Valid": true }
//
// 凡是「对外返回文本」的可空字段都该用本类型；只在内部判断是否存在的字段
// （如各种 *Exists 标记）不必替换。
type JSONString struct {
	String string
	Valid  bool
}

// NewJSONString 由普通字符串构造；空串视为 NULL
func NewJSONString(s string) JSONString {
	if s == "" {
		return JSONString{}
	}
	return JSONString{String: s, Valid: true}
}

// NullableJSONString 保留"空串也是有效值"的语义（需要区分 ” 与 NULL 时使用）
func NullableJSONString(s string) JSONString {
	return JSONString{String: s, Valid: true}
}

// MarshalJSON 输出普通字符串；NULL 输出空串（而不是 null，前端可直接当文本用）
func (s JSONString) MarshalJSON() ([]byte, error) {
	if !s.Valid {
		return []byte(`""`), nil
	}
	return json.Marshal(s.String)
}

// UnmarshalJSON 兼容字符串与 null（便于请求体绑定与测试构造）
func (s *JSONString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		s.String, s.Valid = "", false
		return nil
	}
	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	s.String, s.Valid = v, true
	return nil
}

// Scan 实现 sql.Scanner（读取数据库）
func (s *JSONString) Scan(value interface{}) error {
	if value == nil {
		s.String, s.Valid = "", false
		return nil
	}
	switch v := value.(type) {
	case string:
		s.String, s.Valid = v, true
	case []byte:
		s.String, s.Valid = string(v), true
	default:
		return fmt.Errorf("JSONString: 不支持的类型 %T", value)
	}
	return nil
}

// Value 实现 driver.Valuer（写入数据库）
func (s JSONString) Value() (driver.Value, error) {
	if !s.Valid {
		return nil, nil
	}
	return s.String, nil
}

// IsZero 便于在需要时判断
func (s JSONString) IsZero() bool { return !s.Valid || s.String == "" }

// 确保实现了标准接口
var (
	_ driver.Valuer = JSONString{}
	_ error         = errors.New("")
)

package utils

import (
	"testing"
)

func TestValidateEmail(t *testing.T) {
	testCases := []struct {
		name      string
		email     string
		wantValid bool
	}{
		{"有效邮箱1", "test@example.com", true},
		{"有效邮箱2", "user.name@example.co.uk", true},
		{"有效邮箱3", "user+tag@example.com", true},
		{"无效邮箱1", "invalid", false},
		{"无效邮箱2", "invalid@", false},
		{"无效邮箱3", "@example.com", false},
		{"无效邮箱4", "invalid@.com", false},
		{"空邮箱", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valid := ValidateEmail(tc.email)
			if valid != tc.wantValid {
				t.Errorf("ValidateEmail(%q) = %v, want %v", tc.email, valid, tc.wantValid)
			}
		})
	}
}

func TestSanitizeSearchKeyword(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{"正常关键词", "test keyword", "test keyword"},
		{"包含SQL注入", "test'; DROP TABLE users;--", "test  table users"}, // 实际实现会移除 DROP
		{"包含特殊字符", "test@#$%keyword", "test@keyword"},                  // 实际实现保留 @ 和 .
		{"空字符串", "", ""},
		{"只有空格", "   ", ""},
		{"Unicode字符", "测试关键词", "测试关键词"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeSearchKeyword(tc.input)
			if got != tc.want {
				t.Errorf("SanitizeSearchKeyword(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

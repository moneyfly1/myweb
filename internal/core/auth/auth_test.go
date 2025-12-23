package auth

import (
	"testing"
)

// TestHashPassword 测试密码哈希
func TestHashPassword(t *testing.T) {
	password := "testPassword123"
	
	hashed, err := HashPassword(password)
	if err != nil {
		t.Fatalf("密码哈希失败: %v", err)
	}
	
	// 测试哈希不为空
	if hashed == "" {
		t.Error("密码哈希不应为空")
	}
	
	// 测试哈希长度（bcrypt 哈希通常为 60 字符）
	if len(hashed) < 50 {
		t.Errorf("密码哈希长度应至少为 50，实际为 %d", len(hashed))
	}
	
	// 测试相同密码生成不同哈希（由于 salt）
	hashed2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("第二次密码哈希失败: %v", err)
	}
	if hashed == hashed2 {
		t.Error("相同密码应生成不同的哈希（由于 salt）")
	}
}

// TestVerifyPassword 测试密码验证
func TestVerifyPassword(t *testing.T) {
	password := "testPassword123"
	
	hashed, err := HashPassword(password)
	if err != nil {
		t.Fatalf("密码哈希失败: %v", err)
	}
	
	// 测试正确密码
	if !VerifyPassword(password, hashed) {
		t.Error("正确密码验证失败")
	}
	
	// 测试错误密码
	if VerifyPassword("wrongPassword", hashed) {
		t.Error("错误密码验证应失败")
	}
	
	// 测试空密码
	if VerifyPassword("", hashed) {
		t.Error("空密码验证应失败")
	}
}

// TestPasswordEdgeCases 测试边界条件
func TestPasswordEdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"空密码", "", false}, // HashPassword 可能允许空密码，取决于实现
		{"短密码", "12345", false},
		{"长密码", string(make([]byte, 1000)), false},
		{"特殊字符", "!@#$%^&*()", false},
		{"Unicode字符", "测试密码🔒", false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hashed, err := HashPassword(tc.password)
			if tc.wantErr && err == nil {
				t.Errorf("期望错误但未返回错误")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("不期望错误但返回了错误: %v", err)
			}
			if err == nil && hashed != "" {
				// 验证可以验证
				if !VerifyPassword(tc.password, hashed) {
					t.Errorf("密码验证失败: %s", tc.name)
				}
			}
		})
	}
}


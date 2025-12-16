package auth

import (
	"errors"

	"cboard-go/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// VerifyPassword 验证密码
func VerifyPassword(plainPassword, hashedPassword string) bool {
	if hashedPassword == "" {
		return false
	}
	
	// 检查是否是 bcrypt 哈希
	if len(hashedPassword) < 7 || 
		(hashedPassword[:4] != "$2a$" && hashedPassword[:4] != "$2b$" && hashedPassword[:4] != "$2y$") {
		return false
	}
	
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}

// HashPassword 生成密码哈希
func HashPassword(password string) (string, error) {
	// 限制密码长度（bcrypt 最大 72 字节）
	if len(password) > 72 {
		password = password[:72]
	}
	
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// ValidatePasswordStrength 验证密码强度
func ValidatePasswordStrength(password string, minLength int) (bool, string) {
	if len(password) < minLength {
		return false, "密码长度至少8位"
	}
	
	weakPasswords := []string{
		"password", "123456", "123456789", "qwerty", "abc123",
		"password123", "admin", "root", "user", "test",
	}
	
	for _, weak := range weakPasswords {
		if password == weak {
			return false, "密码过于简单，请使用更复杂的密码"
		}
	}
	
	return true, "密码强度符合要求"
}

// CreateAccessToken 创建访问令牌（已移至 utils/jwt.go）
// CreateRefreshToken 创建刷新令牌（已移至 utils/jwt.go）

// AuthenticateUser 认证用户
func AuthenticateUser(db *gorm.DB, email, password string) (*models.User, error) {
	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, errors.New("用户不存在或密码错误")
	}
	
	if !VerifyPassword(password, user.Password) {
		return nil, errors.New("用户不存在或密码错误")
	}
	
	return &user, nil
}


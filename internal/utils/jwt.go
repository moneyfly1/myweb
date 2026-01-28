package utils

import (
	"errors"
	"time"

	"cboard-go/internal/core/config"
	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	UserID  uint   `json:"sub"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
	Type    string `json:"type"`
	jwt.RegisteredClaims
}

func CreateAccessToken(userID uint, email string, isAdmin bool) (string, error) {
	cfg := config.AppConfig
	if cfg == nil {
		return "", errors.New("配置未初始化")
	}

	expiresAt := time.Now().Add(time.Duration(cfg.AccessTokenExpireMinutes) * time.Minute)

	claims := JWTClaims{
		UserID:  userID,
		Email:   email,
		IsAdmin: isAdmin,
		Type:    "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.SecretKey))
}

func CreateRefreshToken(userID uint, email string) (string, error) {
	cfg := config.AppConfig
	if cfg == nil {
		return "", errors.New("配置未初始化")
	}

	expiresAt := time.Now().Add(time.Duration(cfg.RefreshTokenExpireDays) * 24 * time.Hour)

	claims := JWTClaims{
		UserID: userID,
		Email:  email,
		Type:   "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.SecretKey))
}

func VerifyToken(tokenString string) (*JWTClaims, error) {
	cfg := config.AppConfig
	if cfg == nil {
		return nil, errors.New("配置未初始化")
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("无效的签名方法")
		}
		return []byte(cfg.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("无效的令牌")
}

package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	// 保留给旧方法兼容使用，新项目直接用结构体字段
	// Deprecated: 使用结构体 AccessToken / RefreshToken 替代 MapClaims
	// uidFieldName       string = "uid"
	// emailFieldName     string = "email"
	// tokenTypeFieldName string = "type"
	refreshTokenType string = "refresh"
	accessTokenType  string = "access"
)

// ==================== 结构体定义（推荐方式）====================

// JwtClaims JWT 令牌声明
type JwtClaims struct {
	Uid       json.Number `json:"uid"`     // 用户ID（json.Number 避免精度丢失）
	Version   string      `json:"version"` // 版本号，用于强制刷新
	TokenType string      `json:"type"`    // token类型: access/refresh
	Iat       int64       `json:"iat"`     // 签发时间
	Exp       int64       `json:"exp"`     // 过期时间
}

// Valid 实现 jwt.Claims 接口
func (c JwtClaims) Valid() error {
	if c.Uid == "" {
		return errors.New("uid is required")
	}
	if _, err := c.Uid.Int64(); err != nil {
		return fmt.Errorf("invalid uid format: %w", err)
	}
	if c.TokenType == "" {
		return errors.New("token type is required")
	}

	now := time.Now().Unix()
	if now > c.Exp {
		return errors.New("token is expired")
	}
	if now < c.Iat-60 {
		return errors.New("token used before issued")
	}
	return nil
}

// GetUID 安全获取 int64 类型的 UID
func (c JwtClaims) GetUID() (int64, error) {
	return c.Uid.Int64()
}

// AccessToken 访问令牌
type AccessToken struct {
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	JwtClaims
}

// Valid 实现接口
func (a AccessToken) Valid() error {
	if a.Email == "" {
		return errors.New("email is required in access token")
	}
	return a.JwtClaims.Valid()
}

// RefreshToken 刷新令牌
type RefreshToken struct {
	JwtClaims
}

// Valid 实现接口
func (r RefreshToken) Valid() error {
	if r.TokenType != refreshTokenType {
		return fmt.Errorf("invalid token type for refresh: %s", r.TokenType)
	}
	return r.JwtClaims.Valid()
}

// ==================== Token 生成（推荐）====================

// GenerateAccessToken 生成 Access Token
func GenerateAccessToken(secret string, expireSeconds int64, uid int64, nickname, email string) (string, error) {
	now := time.Now()
	claims := AccessToken{
		Nickname: nickname,
		Email:    email,
		JwtClaims: JwtClaims{
			Uid:       json.Number(strconv.FormatInt(uid, 10)),
			Version:   "1.0",
			TokenType: accessTokenType,
			Iat:       now.Unix(),
			Exp:       now.Add(time.Duration(expireSeconds) * time.Second).Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken 生成 Refresh Token
func GenerateRefreshToken(secret string, expireSeconds int64, uid int64) (string, error) {
	now := time.Now()
	claims := RefreshToken{
		JwtClaims: JwtClaims{
			Uid:       json.Number(strconv.FormatInt(uid, 10)),
			Version:   "1.0",
			TokenType: refreshTokenType,
			Iat:       now.Unix(),
			Exp:       now.Add(time.Duration(expireSeconds) * time.Second).Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ==================== Token 解析（推荐）====================

// ParseAccessToken 解析 Access Token
func ParseAccessToken(tokenString, secret string) (*AccessToken, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessToken{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse access token failed: %w", err)
	}
	claims, ok := token.Claims.(*AccessToken)
	if !ok || !token.Valid {
		return nil, errors.New("invalid access token claims")
	}
	return claims, nil
}

// ParseRefreshToken 解析 Refresh Token
func ParseRefreshToken(tokenString, secret string) (*RefreshToken, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshToken{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse refresh token failed: %w", err)
	}
	claims, ok := token.Claims.(*RefreshToken)
	if !ok || !token.Valid {
		return nil, errors.New("invalid refresh token claims")
	}
	return claims, nil
}

// ==================== Context 操作（推荐）====================

// UIDFromAccessToken 从 context 获取 AccessToken 的 UID
func UIDFromAccessToken(ctx context.Context) (int64, error) {
	claims, ok := ctx.Value("claims").(*AccessToken)
	if !ok {
		return 0, errors.New("access token claims not found in context")
	}
	return claims.GetUID()
}

// UIDFromRefreshToken 从 context 获取 RefreshToken 的 UID
func UIDFromRefreshToken(ctx context.Context) (int64, error) {
	claims, ok := ctx.Value("claims").(*RefreshToken)
	if !ok {
		return 0, errors.New("refresh token claims not found in context")
	}
	return claims.GetUID()
}

// AccessTokenFromContext 获取完整的 AccessToken（需要其他字段时用）
func AccessTokenFromContext(ctx context.Context) (*AccessToken, error) {
	claims, ok := ctx.Value("claims").(*AccessToken)
	if !ok {
		return nil, errors.New("access token claims not found in context")
	}
	return claims, nil
}

// RefreshTokenFromContext 获取完整的 RefreshToken
func RefreshTokenFromContext(ctx context.Context) (*RefreshToken, error) {
	claims, ok := ctx.Value("claims").(*RefreshToken)
	if !ok {
		return nil, errors.New("refresh token claims not found in context")
	}
	return claims, nil
}

func GetEmailByAccessToken(ctx context.Context) (string, error) {
	claims, ok := ctx.Value("claims").(*AccessToken)
	if !ok {
		return "", errors.New("access token claims not found in context")
	}
	return claims.Email, nil
}

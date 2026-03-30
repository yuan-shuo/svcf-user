package utils

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
)

func TestGetUidByJwt_Success_Int64(t *testing.T) {
	ctx := context.WithValue(context.Background(), uidFieldName, int64(12345))

	uid, err := GetUidByJwt(ctx)

	assert.NoError(t, err)
	assert.Equal(t, int64(12345), uid)
}

func TestGetUidByJwt_Success_Int(t *testing.T) {
	ctx := context.WithValue(context.Background(), uidFieldName, 12345)

	uid, err := GetUidByJwt(ctx)

	assert.NoError(t, err)
	assert.Equal(t, int64(12345), uid)
}

func TestGetUidByJwt_Success_JsonNumber(t *testing.T) {
	ctx := context.WithValue(context.Background(), uidFieldName, json.Number("12345"))

	uid, err := GetUidByJwt(ctx)

	assert.NoError(t, err)
	assert.Equal(t, int64(12345), uid)
}

func TestGetUidByJwt_Success_String(t *testing.T) {
	ctx := context.WithValue(context.Background(), uidFieldName, "12345")

	uid, err := GetUidByJwt(ctx)

	assert.NoError(t, err)
	assert.Equal(t, int64(12345), uid)
}

func TestGetUidByJwt_Success_Float64(t *testing.T) {
	ctx := context.WithValue(context.Background(), uidFieldName, float64(12345))

	uid, err := GetUidByJwt(ctx)

	assert.NoError(t, err)
	assert.Equal(t, int64(12345), uid)
}

func TestGetUidByJwt_NotFound(t *testing.T) {
	ctx := context.Background()

	uid, err := GetUidByJwt(ctx)

	assert.Error(t, err)
	assert.Equal(t, int64(0), uid)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetUidByJwt_InvalidString(t *testing.T) {
	ctx := context.WithValue(context.Background(), uidFieldName, "not-a-number")

	uid, err := GetUidByJwt(ctx)

	assert.Error(t, err)
	assert.Equal(t, int64(0), uid)
}

func TestGetUidByJwt_UnsupportedType(t *testing.T) {
	ctx := context.WithValue(context.Background(), uidFieldName, []string{"invalid"})

	uid, err := GetUidByJwt(ctx)

	assert.Error(t, err)
	assert.Equal(t, int64(0), uid)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestGetEmailByJwt_Success(t *testing.T) {
	ctx := context.WithValue(context.Background(), emailFieldName, "test@example.com")

	email, err := GetEmailByJwt(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "test@example.com", email)
}

func TestGetEmailByJwt_NotFound(t *testing.T) {
	ctx := context.Background()

	email, err := GetEmailByJwt(ctx)

	assert.Error(t, err)
	assert.Empty(t, email)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetEmailByJwt_InvalidType(t *testing.T) {
	ctx := context.WithValue(context.Background(), emailFieldName, 12345)

	email, err := GetEmailByJwt(ctx)

	assert.Error(t, err)
	assert.Empty(t, email)
	assert.Contains(t, err.Error(), "type error")
}

func TestGenerateAccessToken_Success(t *testing.T) {
	secret := "test-secret"
	expireSeconds := int64(3600)
	uid := int64(12345)
	nickname := "testuser"
	email := "test@example.com"

	token, err := GenerateAccessToken(secret, expireSeconds, uid, nickname, email)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// 验证 token 可以解析
	claims, err := ParseToken(token, secret)
	assert.NoError(t, err)
	assert.Equal(t, uid, int64(claims["uid"].(float64)))
	assert.Equal(t, nickname, claims["nickname"])
	assert.Equal(t, email, claims["email"])
	assert.Equal(t, "access", claims["type"])
}

func TestGenerateRefreshToken_Success(t *testing.T) {
	secret := "test-secret"
	expireSeconds := int64(7200)
	uid := int64(12345)

	token, err := GenerateRefreshToken(secret, expireSeconds, uid)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// 验证 token 可以解析
	claims, err := ParseToken(token, secret)
	assert.NoError(t, err)
	assert.Equal(t, uid, int64(claims["uid"].(float64)))
	assert.Equal(t, "refresh", claims["type"])
}

func TestParseToken_Success(t *testing.T) {
	secret := "test-secret"
	uid := int64(12345)
	nickname := "testuser"
	email := "test@example.com"

	token, err := GenerateAccessToken(secret, 3600, uid, nickname, email)
	assert.NoError(t, err)

	claims, err := ParseToken(token, secret)

	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, uid, int64(claims["uid"].(float64)))
	assert.Equal(t, nickname, claims["nickname"])
	assert.Equal(t, email, claims["email"])
}

func TestParseToken_InvalidSecret(t *testing.T) {
	secret := "test-secret"
	wrongSecret := "wrong-secret"
	uid := int64(12345)
	nickname := "testuser"
	email := "test@example.com"

	token, err := GenerateAccessToken(secret, 3600, uid, nickname, email)
	assert.NoError(t, err)

	claims, err := ParseToken(token, wrongSecret)

	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestParseToken_InvalidToken(t *testing.T) {
	secret := "test-secret"

	claims, err := ParseToken("invalid.token.here", secret)

	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestParseToken_Expired(t *testing.T) {
	secret := "test-secret"
	uid := int64(12345)
	nickname := "testuser"
	email := "test@example.com"

	// 生成一个已经过期的 token
	now := time.Now()
	claims := jwt.MapClaims{
		uidFieldName:   uid,
		"nickname":     nickname,
		emailFieldName: email,
		"type":         "access",
		"iat":          now.Add(-2 * time.Hour).Unix(),
		"exp":          now.Add(-1 * time.Hour).Unix(), // 已过期
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	assert.NoError(t, err)

	parsedClaims, err := ParseToken(tokenString, secret)

	assert.Error(t, err)
	assert.Nil(t, parsedClaims)
}

func TestTokenRoundTrip(t *testing.T) {
	secret := "my-secret-key"
	expireSeconds := int64(3600)
	uid := int64(12345)
	nickname := "testuser"
	email := "test@example.com"

	// 生成 token
	token, err := GenerateAccessToken(secret, expireSeconds, uid, nickname, email)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// 解析 token
	claims, err := ParseToken(token, secret)
	assert.NoError(t, err)

	// 验证所有字段
	assert.Equal(t, uid, int64(claims["uid"].(float64)))
	assert.Equal(t, nickname, claims["nickname"])
	assert.Equal(t, email, claims["email"])
	assert.Equal(t, "access", claims["type"])

	// 验证时间字段
	iat, ok := claims["iat"].(float64)
	assert.True(t, ok)
	assert.Greater(t, iat, float64(0))

	exp, ok := claims["exp"].(float64)
	assert.True(t, ok)
	assert.Greater(t, exp, iat)
}

func TestGenerateAccessToken_DifferentUsers(t *testing.T) {
	secret := "test-secret"
	expireSeconds := int64(3600)

	testCases := []struct {
		uid      int64
		nickname string
		email    string
	}{
		{1, "user1", "user1@example.com"},
		{2, "user2", "user2@example.com"},
		{999, "testuser", "test@example.com"},
	}

	for _, tc := range testCases {
		token, err := GenerateAccessToken(secret, expireSeconds, tc.uid, tc.nickname, tc.email)
		assert.NoError(t, err)

		claims, err := ParseToken(token, secret)
		assert.NoError(t, err)

		assert.Equal(t, tc.uid, int64(claims["uid"].(float64)))
		assert.Equal(t, tc.nickname, claims["nickname"])
		assert.Equal(t, tc.email, claims["email"])
	}
}

// ==================== GetUidFromClaims 测试 ====================

func TestGetUidFromClaims_Success(t *testing.T) {
	claims := jwt.MapClaims{
		"uid": float64(12345),
	}

	uid, err := GetUidFromClaims(claims)

	assert.NoError(t, err)
	assert.Equal(t, int64(12345), uid)
}

func TestGetUidFromClaims_Int64(t *testing.T) {
	claims := jwt.MapClaims{
		"uid": int64(12345),
	}

	uid, err := GetUidFromClaims(claims)

	assert.NoError(t, err)
	assert.Equal(t, int64(12345), uid)
}

func TestGetUidFromClaims_Int(t *testing.T) {
	claims := jwt.MapClaims{
		"uid": int(12345),
	}

	uid, err := GetUidFromClaims(claims)

	assert.NoError(t, err)
	assert.Equal(t, int64(12345), uid)
}

func TestGetUidFromClaims_String(t *testing.T) {
	claims := jwt.MapClaims{
		"uid": "12345",
	}

	uid, err := GetUidFromClaims(claims)

	assert.NoError(t, err)
	assert.Equal(t, int64(12345), uid)
}

func TestGetUidFromClaims_JsonNumber(t *testing.T) {
	claims := jwt.MapClaims{
		"uid": json.Number("12345"),
	}

	uid, err := GetUidFromClaims(claims)

	assert.NoError(t, err)
	assert.Equal(t, int64(12345), uid)
}

func TestGetUidFromClaims_NotFound(t *testing.T) {
	claims := jwt.MapClaims{}

	uid, err := GetUidFromClaims(claims)

	assert.Error(t, err)
	assert.Equal(t, int64(0), uid)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetUidFromClaims_InvalidString(t *testing.T) {
	claims := jwt.MapClaims{
		"uid": "not-a-number",
	}

	uid, err := GetUidFromClaims(claims)

	assert.Error(t, err)
	assert.Equal(t, int64(0), uid)
}

func TestGetUidFromClaims_UnsupportedType(t *testing.T) {
	claims := jwt.MapClaims{
		"uid": []string{"invalid"},
	}

	uid, err := GetUidFromClaims(claims)

	assert.Error(t, err)
	assert.Equal(t, int64(0), uid)
	assert.Contains(t, err.Error(), "unsupported")
}

// ==================== IsRefreshToken 测试 ====================

func TestIsRefreshToken_Success(t *testing.T) {
	claims := jwt.MapClaims{
		"type": "refresh",
	}

	err := IsRefreshToken(claims)

	assert.NoError(t, err)
}

func TestIsRefreshToken_NotRefreshToken(t *testing.T) {
	claims := jwt.MapClaims{
		"type": "access",
	}

	err := IsRefreshToken(claims)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mismatch")
}

func TestIsRefreshToken_TypeNotFound(t *testing.T) {
	claims := jwt.MapClaims{
		"uid": float64(12345),
	}

	err := IsRefreshToken(claims)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ==================== ValidateClaimString 测试 ====================

func TestValidateClaimString_Success(t *testing.T) {
	claims := jwt.MapClaims{
		"type": "refresh",
	}

	err := ValidateClaimString(claims, "type", "refresh")

	assert.NoError(t, err)
}

func TestValidateClaimString_KeyNotFound(t *testing.T) {
	claims := jwt.MapClaims{}

	err := ValidateClaimString(claims, "type", "refresh")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestValidateClaimString_ValueMismatch(t *testing.T) {
	claims := jwt.MapClaims{
		"type": "access",
	}

	err := ValidateClaimString(claims, "type", "refresh")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mismatch")
}

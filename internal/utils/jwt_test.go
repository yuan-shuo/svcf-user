package utils

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ==================== JwtClaims 测试 ====================

func TestJwtClaims_Valid_Success(t *testing.T) {
	claims := JwtClaims{
		Uid:       json.Number("12345"),
		Version:   "1.0",
		TokenType: accessTokenType,
		Iat:       time.Now().Unix(),
		Exp:       time.Now().Add(time.Hour).Unix(),
	}

	err := claims.Valid()

	assert.NoError(t, err)
}

func TestJwtClaims_Valid_MissingUid(t *testing.T) {
	claims := JwtClaims{
		Uid:       "",
		Version:   "1.0",
		TokenType: accessTokenType,
		Iat:       time.Now().Unix(),
		Exp:       time.Now().Add(time.Hour).Unix(),
	}

	err := claims.Valid()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "uid is required")
}

func TestJwtClaims_Valid_InvalidUidFormat(t *testing.T) {
	claims := JwtClaims{
		Uid:       json.Number("not-a-number"),
		Version:   "1.0",
		TokenType: accessTokenType,
		Iat:       time.Now().Unix(),
		Exp:       time.Now().Add(time.Hour).Unix(),
	}

	err := claims.Valid()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid uid format")
}

func TestJwtClaims_Valid_MissingTokenType(t *testing.T) {
	claims := JwtClaims{
		Uid:       json.Number("12345"),
		Version:   "1.0",
		TokenType: "",
		Iat:       time.Now().Unix(),
		Exp:       time.Now().Add(time.Hour).Unix(),
	}

	err := claims.Valid()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token type is required")
}

// TestJwtClaims_Valid_Expired 已移除，因为 iat/exp 校验由 go-zero 中间件处理
// func TestJwtClaims_Valid_Expired(t *testing.T) {
// 	claims := JwtClaims{
// 		Uid:       json.Number("12345"),
// 		Version:   "1.0",
// 		TokenType: accessTokenType,
// 		Iat:       time.Now().Add(-2 * time.Hour).Unix(),
// 		Exp:       time.Now().Add(-time.Hour).Unix(),
// 	}
//
// 	err := claims.Valid()
//
// 	assert.Error(t, err)
// 	assert.Contains(t, err.Error(), "expired")
// }

func TestJwtClaims_GetUID_Success(t *testing.T) {
	claims := JwtClaims{
		Uid: json.Number("12345"),
	}

	uid, err := claims.GetUID()

	assert.NoError(t, err)
	assert.Equal(t, int64(12345), uid)
}

// ==================== AccessToken 测试 ====================

func TestAccessToken_Valid_Success(t *testing.T) {
	claims := AccessToken{
		Nickname: "testuser",
		Email:    "test@example.com",
		JwtClaims: JwtClaims{
			Uid:       json.Number("12345"),
			Version:   "1.0",
			TokenType: accessTokenType,
			Iat:       time.Now().Unix(),
			Exp:       time.Now().Add(time.Hour).Unix(),
		},
	}

	err := claims.Valid()

	assert.NoError(t, err)
}

func TestAccessToken_Valid_MissingEmail(t *testing.T) {
	claims := AccessToken{
		Nickname: "testuser",
		Email:    "",
		JwtClaims: JwtClaims{
			Uid:       json.Number("12345"),
			Version:   "1.0",
			TokenType: accessTokenType,
			Iat:       time.Now().Unix(),
			Exp:       time.Now().Add(time.Hour).Unix(),
		},
	}

	err := claims.Valid()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email is required")
}

// ==================== RefreshToken 测试 ====================

func TestRefreshToken_Valid_Success(t *testing.T) {
	claims := RefreshToken{
		JwtClaims: JwtClaims{
			Uid:       json.Number("12345"),
			Version:   "1.0",
			TokenType: refreshTokenType,
			Iat:       time.Now().Unix(),
			Exp:       time.Now().Add(time.Hour).Unix(),
		},
	}

	err := claims.Valid()

	assert.NoError(t, err)
}

func TestRefreshToken_Valid_WrongTokenType(t *testing.T) {
	claims := RefreshToken{
		JwtClaims: JwtClaims{
			Uid:       json.Number("12345"),
			Version:   "1.0",
			TokenType: accessTokenType, // 错误的类型
			Iat:       time.Now().Unix(),
			Exp:       time.Now().Add(time.Hour).Unix(),
		},
	}

	err := claims.Valid()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token type")
}

// ==================== GenerateAccessToken 测试 ====================

func TestGenerateAccessToken_Success(t *testing.T) {
	secret := "test-secret"
	expireSeconds := int64(3600)
	uid := int64(12345)
	nickname := "testuser"
	email := "test@example.com"

	token, err := GenerateAccessToken(secret, expireSeconds, uid, nickname, email)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// 解析验证
	parsed, err := ParseAccessToken(token, secret)
	assert.NoError(t, err)
	assert.Equal(t, nickname, parsed.Nickname)
	assert.Equal(t, email, parsed.Email)
	parsedUid, _ := parsed.GetUID()
	assert.Equal(t, uid, parsedUid)
}

func TestGenerateAccessToken_DifferentSecrets(t *testing.T) {
	secret1 := "secret-1"
	secret2 := "secret-2"

	token1, _ := GenerateAccessToken(secret1, 3600, 12345, "user1", "user1@test.com")
	token2, _ := GenerateAccessToken(secret2, 3600, 12345, "user1", "user1@test.com")

	// 相同内容不同密钥应该生成不同token
	assert.NotEqual(t, token1, token2)
}

// ==================== GenerateRefreshToken 测试 ====================

func TestGenerateRefreshToken_Success(t *testing.T) {
	secret := "test-secret"
	expireSeconds := int64(3600)
	uid := int64(12345)

	token, err := GenerateRefreshToken(secret, expireSeconds, uid)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// 解析验证
	parsed, err := ParseRefreshToken(token, secret)
	assert.NoError(t, err)
	parsedUid, _ := parsed.GetUID()
	assert.Equal(t, uid, parsedUid)
	assert.Equal(t, refreshTokenType, parsed.TokenType)
}

// ==================== ParseAccessToken 测试 ====================

func TestParseAccessToken_Success(t *testing.T) {
	secret := "test-secret"
	token, _ := GenerateAccessToken(secret, 3600, 12345, "testuser", "test@example.com")

	parsed, err := ParseAccessToken(token, secret)

	assert.NoError(t, err)
	assert.NotNil(t, parsed)
	assert.Equal(t, "testuser", parsed.Nickname)
	assert.Equal(t, "test@example.com", parsed.Email)
	uid, _ := parsed.GetUID()
	assert.Equal(t, int64(12345), uid)
}

func TestParseAccessToken_InvalidSecret(t *testing.T) {
	secret := "test-secret"
	wrongSecret := "wrong-secret"
	token, _ := GenerateAccessToken(secret, 3600, 12345, "testuser", "test@example.com")

	parsed, err := ParseAccessToken(token, wrongSecret)

	assert.Error(t, err)
	assert.Nil(t, parsed)
}

func TestParseAccessToken_Expired(t *testing.T) {
	secret := "test-secret"
	// 生成已过期的token
	token, _ := GenerateAccessToken(secret, -1, 12345, "testuser", "test@example.com")

	parsed, err := ParseAccessToken(token, secret)

	assert.Error(t, err)
	assert.Nil(t, parsed)
}

func TestParseAccessToken_InvalidFormat(t *testing.T) {
	secret := "test-secret"

	parsed, err := ParseAccessToken("invalid-token", secret)

	assert.Error(t, err)
	assert.Nil(t, parsed)
}

// ==================== ParseRefreshToken 测试 ====================

func TestParseRefreshToken_Success(t *testing.T) {
	secret := "test-secret"
	token, _ := GenerateRefreshToken(secret, 3600, 12345)

	parsed, err := ParseRefreshToken(token, secret)

	assert.NoError(t, err)
	assert.NotNil(t, parsed)
	uid, _ := parsed.GetUID()
	assert.Equal(t, int64(12345), uid)
	assert.Equal(t, refreshTokenType, parsed.TokenType)
}

func TestParseRefreshToken_InvalidSecret(t *testing.T) {
	secret := "test-secret"
	wrongSecret := "wrong-secret"
	token, _ := GenerateRefreshToken(secret, 3600, 12345)

	parsed, err := ParseRefreshToken(token, wrongSecret)

	assert.Error(t, err)
	assert.Nil(t, parsed)
}

func TestParseRefreshToken_WrongTokenType(t *testing.T) {
	// 使用 access token 尝试解析为 refresh token
	secret := "test-secret"
	accessToken, _ := GenerateAccessToken(secret, 3600, 12345, "testuser", "test@example.com")

	parsed, err := ParseRefreshToken(accessToken, secret)

	assert.Error(t, err)
	assert.Nil(t, parsed)
}

// ==================== Context 操作测试 ====================
// 注意：这些测试模拟 go-zero JWT 中间件将 claims 字段存入 context 的行为

func TestUIDFromAccessToken_Success(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "uid", json.Number("12345"))
	ctx = context.WithValue(ctx, "version", "1.0")
	ctx = context.WithValue(ctx, "type", accessTokenType)
	ctx = context.WithValue(ctx, "nickname", "testuser")
	ctx = context.WithValue(ctx, "email", "test@example.com")

	uid, err := UIDFromAccessToken(ctx)

	assert.NoError(t, err)
	assert.Equal(t, int64(12345), uid)
}

func TestUIDFromAccessToken_NotFound(t *testing.T) {
	ctx := context.Background()

	uid, err := UIDFromAccessToken(ctx)

	assert.Error(t, err)
	assert.Equal(t, int64(0), uid)
	assert.Contains(t, err.Error(), "not found")
}

func TestUIDFromAccessToken_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), "uid", "not-a-number")

	uid, err := UIDFromAccessToken(ctx)

	assert.Error(t, err)
	assert.Equal(t, int64(0), uid)
}

func TestUIDFromRefreshToken_Success(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "uid", json.Number("12345"))
	ctx = context.WithValue(ctx, "version", "1.0")
	ctx = context.WithValue(ctx, "type", refreshTokenType)

	uid, err := UIDFromRefreshToken(ctx)

	assert.NoError(t, err)
	assert.Equal(t, int64(12345), uid)
}

func TestUIDFromRefreshToken_NotFound(t *testing.T) {
	ctx := context.Background()

	uid, err := UIDFromRefreshToken(ctx)

	assert.Error(t, err)
	assert.Equal(t, int64(0), uid)
}

func TestAccessTokenFromContext_Success(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "uid", json.Number("12345"))
	ctx = context.WithValue(ctx, "version", "1.0")
	ctx = context.WithValue(ctx, "type", accessTokenType)
	ctx = context.WithValue(ctx, "nickname", "testuser")
	ctx = context.WithValue(ctx, "email", "test@example.com")

	parsed, err := AccessTokenFromContext(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, parsed)
	assert.Equal(t, "testuser", parsed.Nickname)
	assert.Equal(t, "test@example.com", parsed.Email)
	uid, _ := parsed.GetUID()
	assert.Equal(t, int64(12345), uid)
}

func TestAccessTokenFromContext_NotFound(t *testing.T) {
	ctx := context.Background()

	parsed, err := AccessTokenFromContext(ctx)

	assert.Error(t, err)
	assert.Nil(t, parsed)
}

func TestRefreshTokenFromContext_Success(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "uid", json.Number("12345"))
	ctx = context.WithValue(ctx, "version", "1.0")
	ctx = context.WithValue(ctx, "type", refreshTokenType)

	parsed, err := RefreshTokenFromContext(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, parsed)
	uid, _ := parsed.GetUID()
	assert.Equal(t, int64(12345), uid)
}

func TestRefreshTokenFromContext_NotFound(t *testing.T) {
	ctx := context.Background()

	parsed, err := RefreshTokenFromContext(ctx)

	assert.Error(t, err)
	assert.Nil(t, parsed)
}

func TestGetEmailByAccessToken_Success(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "uid", json.Number("12345"))
	ctx = context.WithValue(ctx, "version", "1.0")
	ctx = context.WithValue(ctx, "type", accessTokenType)
	ctx = context.WithValue(ctx, "nickname", "testuser")
	ctx = context.WithValue(ctx, "email", "test@example.com")

	email, err := GetEmailByAccessToken(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "test@example.com", email)
}

func TestGetEmailByAccessToken_NotFound(t *testing.T) {
	ctx := context.Background()

	email, err := GetEmailByAccessToken(ctx)

	assert.Error(t, err)
	assert.Empty(t, email)
}

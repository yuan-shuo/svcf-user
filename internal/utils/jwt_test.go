package utils

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== RS256 Token 生成测试 ====================

func TestGenerateAccessTokenWithRSA(t *testing.T) {
	keyPair, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	expireSeconds := int64(3600)
	uid := int64(123456789)
	nickname := "TestUser"
	email := "test@example.com"

	token, err := GenerateAccessTokenWithRSA(keyPair.PrivateKey, keyPair.KeyID, expireSeconds, uid, nickname, email)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// 使用公钥解析验证
	claims, err := ParseAccessTokenWithRSA(token, keyPair.PublicKey)
	require.NoError(t, err)
	gotUID, err := claims.GetUID()
	require.NoError(t, err)
	assert.Equal(t, uid, gotUID)
	assert.Equal(t, nickname, claims.Nickname)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, accessTokenType, claims.TokenType)
}

func TestGenerateRefreshTokenWithRSA(t *testing.T) {
	keyPair, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	expireSeconds := int64(604800)
	uid := int64(123456789)

	token, err := GenerateRefreshTokenWithRSA(keyPair.PrivateKey, keyPair.KeyID, expireSeconds, uid)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// 使用公钥解析验证
	claims, err := ParseRefreshTokenWithRSA(token, keyPair.PublicKey)
	require.NoError(t, err)
	gotUID, err := claims.GetUID()
	require.NoError(t, err)
	assert.Equal(t, uid, gotUID)
	assert.Equal(t, refreshTokenType, claims.TokenType)
}

// ==================== RS256 Token 解析测试 ====================

func TestParseAccessTokenWithRSA_Invalid(t *testing.T) {
	keyPair, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	// 测试无效 token
	_, err = ParseAccessTokenWithRSA("invalid.token.here", keyPair.PublicKey)
	assert.Error(t, err)

	// 测试过期 token
	token, _ := GenerateAccessTokenWithRSA(keyPair.PrivateKey, keyPair.KeyID, -1, 123, "test", "test@example.com")
	_, err = ParseAccessTokenWithRSA(token, keyPair.PublicKey)
	assert.Error(t, err)
}

func TestParseRefreshTokenWithRSA_Invalid(t *testing.T) {
	keyPair, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	// 测试无效 token
	_, err = ParseRefreshTokenWithRSA("invalid.token.here", keyPair.PublicKey)
	assert.Error(t, err)

	// 测试过期 token
	token, _ := GenerateRefreshTokenWithRSA(keyPair.PrivateKey, keyPair.KeyID, -1, 123)
	_, err = ParseRefreshTokenWithRSA(token, keyPair.PublicKey)
	assert.Error(t, err)
}

func TestParseAccessTokenWithRSA_WrongKey(t *testing.T) {
	keyPair1, _ := GenerateRSAKeyPair(2048)
	keyPair2, _ := GenerateRSAKeyPair(2048)

	token, _ := GenerateAccessTokenWithRSA(keyPair1.PrivateKey, keyPair1.KeyID, 3600, 123, "test", "test@example.com")

	// 使用错误的公钥验证
	_, err := ParseAccessTokenWithRSA(token, keyPair2.PublicKey)
	assert.Error(t, err)
}

func TestParseAccessTokenUnverified(t *testing.T) {
	keyPair, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	token, _ := GenerateAccessTokenWithRSA(keyPair.PrivateKey, keyPair.KeyID, 3600, 123, "test", "test@example.com")

	// 不验证签名解析
	claims, err := ParseAccessTokenUnverified(token)
	require.NoError(t, err)
	gotUID, err := claims.GetUID()
	require.NoError(t, err)
	assert.Equal(t, int64(123), gotUID)
	assert.Equal(t, "test", claims.Nickname)
	assert.Equal(t, "test@example.com", claims.Email)
}

func TestParseRefreshTokenUnverified(t *testing.T) {
	keyPair, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	token, _ := GenerateRefreshTokenWithRSA(keyPair.PrivateKey, keyPair.KeyID, 604800, 123)

	// 不验证签名解析
	claims, err := ParseRefreshTokenUnverified(token)
	require.NoError(t, err)
	gotUID, err := claims.GetUID()
	require.NoError(t, err)
	assert.Equal(t, int64(123), gotUID)
	assert.Equal(t, refreshTokenType, claims.TokenType)
}

func TestParseAccessTokenUnverified_Invalid(t *testing.T) {
	_, err := ParseAccessTokenUnverified("invalid.token")
	assert.Error(t, err)
}

func TestParseRefreshTokenUnverified_Invalid(t *testing.T) {
	_, err := ParseRefreshTokenUnverified("invalid.token")
	assert.Error(t, err)
}

// ==================== JWK/JWKS 测试 ====================

func TestRSAKeyPair_ToJWK(t *testing.T) {
	keyPair, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	jwk := keyPair.ToJWK()
	assert.Equal(t, "RSA", jwk.Kty)
	assert.Equal(t, keyPair.KeyID, jwk.Kid)
	assert.Equal(t, "sig", jwk.Use)
	assert.Equal(t, "RS256", jwk.Alg)
	assert.NotEmpty(t, jwk.N)
	assert.NotEmpty(t, jwk.E)
}

func TestRSAKeyPair_ToJWKS(t *testing.T) {
	keyPair, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	jwks := keyPair.ToJWKS()
	assert.Len(t, jwks.Keys, 1)
	assert.Equal(t, keyPair.KeyID, jwks.Keys[0].Kid)
}

func TestJWKS_ToJSON(t *testing.T) {
	keyPair, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	jwks := keyPair.ToJWKS()
	jsonStr, err := jwks.ToJSON()
	require.NoError(t, err)
	assert.Contains(t, jsonStr, "keys")
	assert.Contains(t, jsonStr, keyPair.KeyID)
	assert.Contains(t, jsonStr, "RSA")
}

func TestRSAKeyPair_ToJWKSJSON(t *testing.T) {
	keyPair, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	jsonStr, err := keyPair.ToJWKSJSON()
	require.NoError(t, err)
	assert.Contains(t, jsonStr, "keys")
	assert.Contains(t, jsonStr, keyPair.KeyID)
}

func TestInitJWKSFromPEM(t *testing.T) {
	keyPair, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	pemStr, err := PublicKeyToPEM(keyPair.PublicKey)
	require.NoError(t, err)

	jwks, err := InitJWKSFromPEM(pemStr, keyPair.KeyID)
	require.NoError(t, err)
	assert.Len(t, jwks.Keys, 1)
	assert.Equal(t, keyPair.KeyID, jwks.Keys[0].Kid)
}

func TestInitJWKSFromPEM_Empty(t *testing.T) {
	jwks, err := InitJWKSFromPEM("", "test-key")
	require.NoError(t, err)
	assert.Len(t, jwks.Keys, 0)
}

func TestInitJWKSFromPEM_Invalid(t *testing.T) {
	_, err := InitJWKSFromPEM("invalid pem", "test-key")
	assert.Error(t, err)
}

// ==================== RSA 密钥对测试 ====================

func TestGenerateRSAKeyPair(t *testing.T) {
	keyPair, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)
	assert.NotNil(t, keyPair.PrivateKey)
	assert.NotNil(t, keyPair.PublicKey)
	assert.NotEmpty(t, keyPair.KeyID)
	assert.Contains(t, keyPair.KeyID, "key-")
}

func TestPrivateKeyToPEM(t *testing.T) {
	keyPair, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	pemStr := PrivateKeyToPEM(keyPair.PrivateKey)
	assert.Contains(t, pemStr, "BEGIN RSA PRIVATE KEY")
	assert.Contains(t, pemStr, "END RSA PRIVATE KEY")

	// 验证可以解析回来
	parsedKey, err := ParseRSAPrivateKeyFromPEM(pemStr)
	require.NoError(t, err)
	assert.NotNil(t, parsedKey)
}

func TestPublicKeyToPEM(t *testing.T) {
	keyPair, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	pemStr, err := PublicKeyToPEM(keyPair.PublicKey)
	require.NoError(t, err)
	assert.Contains(t, pemStr, "BEGIN PUBLIC KEY")
	assert.Contains(t, pemStr, "END PUBLIC KEY")

	// 验证可以解析回来
	parsedKey, err := ParseRSAPublicKeyFromPEM(pemStr)
	require.NoError(t, err)
	assert.NotNil(t, parsedKey)
}

func TestParseRSAPrivateKeyFromPEM_Invalid(t *testing.T) {
	// 测试无效 PEM
	_, err := ParseRSAPrivateKeyFromPEM("invalid pem")
	assert.Error(t, err)

	// 测试空字符串
	_, err = ParseRSAPrivateKeyFromPEM("")
	assert.Error(t, err)
}

func TestParseRSAPublicKeyFromPEM_Invalid(t *testing.T) {
	// 测试无效 PEM
	_, err := ParseRSAPublicKeyFromPEM("invalid pem")
	assert.Error(t, err)

	// 测试空字符串
	_, err = ParseRSAPublicKeyFromPEM("")
	assert.Error(t, err)
}

// ==================== Context 测试 ====================

func TestUIDFromAccessToken(t *testing.T) {
	uid := int64(123456789)
	claims := &JwtClaims{
		Uid:       json.Number("123456789"),
		TokenType: accessTokenType,
	}
	ctx := context.WithValue(context.Background(), uidFieldName, claims.Uid)
	ctx = context.WithValue(ctx, tokenTypeFieldName, claims.TokenType)
	ctx = context.WithValue(ctx, nicknameFieldName, "TestUser")
	ctx = context.WithValue(ctx, emailFieldName, "test@example.com")

	result, err := UIDFromAccessToken(ctx)
	require.NoError(t, err)
	assert.Equal(t, uid, result)
}

func TestUIDFromAccessToken_NotFound(t *testing.T) {
	ctx := context.Background()
	_, err := UIDFromAccessToken(ctx)
	assert.Error(t, err)
}

func TestUIDFromRefreshToken(t *testing.T) {
	uid := int64(123456789)
	claims := &JwtClaims{
		Uid:       json.Number("123456789"),
		TokenType: refreshTokenType,
	}
	ctx := context.WithValue(context.Background(), uidFieldName, claims.Uid)
	ctx = context.WithValue(ctx, tokenTypeFieldName, claims.TokenType)

	result, err := UIDFromRefreshToken(ctx)
	require.NoError(t, err)
	assert.Equal(t, uid, result)
}

func TestUIDFromRefreshToken_NotFound(t *testing.T) {
	ctx := context.Background()
	_, err := UIDFromRefreshToken(ctx)
	assert.Error(t, err)
}

func TestGetJWTClaimsByContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), uidFieldName, json.Number("123456789"))
	ctx = context.WithValue(ctx, tokenTypeFieldName, accessTokenType)

	claims, err := GetJWTClaimsByContext(ctx)
	require.NoError(t, err)
	gotUID, err := claims.GetUID()
	require.NoError(t, err)
	assert.Equal(t, int64(123456789), gotUID)
	assert.Equal(t, accessTokenType, claims.TokenType)
}

func TestGetJWTClaimsByContext_NotFound(t *testing.T) {
	ctx := context.Background()
	_, err := GetJWTClaimsByContext(ctx)
	assert.Error(t, err)
}

func TestAccessTokenFromContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), uidFieldName, json.Number("123456789"))
	ctx = context.WithValue(ctx, tokenTypeFieldName, accessTokenType)
	ctx = context.WithValue(ctx, nicknameFieldName, "TestUser")
	ctx = context.WithValue(ctx, emailFieldName, "test@example.com")

	at, err := AccessTokenFromContext(ctx)
	require.NoError(t, err)
	gotUID, err := at.GetUID()
	require.NoError(t, err)
	assert.Equal(t, int64(123456789), gotUID)
	assert.Equal(t, "TestUser", at.Nickname)
	assert.Equal(t, "test@example.com", at.Email)
}

func TestAccessTokenFromContext_MissingFields(t *testing.T) {
	ctx := context.WithValue(context.Background(), uidFieldName, json.Number("123456789"))
	ctx = context.WithValue(ctx, tokenTypeFieldName, accessTokenType)
	// 缺少 nickname 和 email

	_, err := AccessTokenFromContext(ctx)
	assert.Error(t, err)
}

func TestRefreshTokenFromContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), uidFieldName, json.Number("123456789"))
	ctx = context.WithValue(ctx, tokenTypeFieldName, refreshTokenType)

	rt, err := RefreshTokenFromContext(ctx)
	require.NoError(t, err)
	gotUID, err := rt.GetUID()
	require.NoError(t, err)
	assert.Equal(t, int64(123456789), gotUID)
	assert.Equal(t, refreshTokenType, rt.TokenType)
}

func TestRefreshTokenFromContext_NotFound(t *testing.T) {
	ctx := context.Background()
	_, err := RefreshTokenFromContext(ctx)
	assert.Error(t, err)
}

func TestGetEmailByAccessToken(t *testing.T) {
	ctx := context.WithValue(context.Background(), uidFieldName, json.Number("123456789"))
	ctx = context.WithValue(ctx, tokenTypeFieldName, accessTokenType)
	ctx = context.WithValue(ctx, nicknameFieldName, "TestUser")
	ctx = context.WithValue(ctx, emailFieldName, "test@example.com")

	email, err := GetEmailByAccessToken(ctx)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", email)
}

func TestGetEmailByAccessToken_NotFound(t *testing.T) {
	ctx := context.Background()
	_, err := GetEmailByAccessToken(ctx)
	assert.Error(t, err)
}

// ==================== Claims 验证测试 ====================

func TestJwtClaims_Valid(t *testing.T) {
	tests := []struct {
		name    string
		claims  JwtClaims
		wantErr bool
	}{
		{
			name: "valid claims",
			claims: JwtClaims{
				Uid:       json.Number("123456789"),
				TokenType: accessTokenType,
				Exp:       time.Now().Add(time.Hour).Unix(),
			},
			wantErr: false,
		},
		{
			name: "missing uid",
			claims: JwtClaims{
				TokenType: accessTokenType,
			},
			wantErr: true,
		},
		{
			name: "invalid uid format",
			claims: JwtClaims{
				Uid:       json.Number("not-a-number"),
				TokenType: accessTokenType,
			},
			wantErr: true,
		},
		{
			name: "missing token type",
			claims: JwtClaims{
				Uid: json.Number("123456789"),
			},
			wantErr: true,
		},
		{
			name: "expired token",
			claims: JwtClaims{
				Uid:       json.Number("123456789"),
				TokenType: accessTokenType,
				Exp:       time.Now().Add(-time.Hour).Unix(),
			},
			wantErr: true,
		},
		{
			name: "no expiration",
			claims: JwtClaims{
				Uid:       json.Number("123456789"),
				TokenType: accessTokenType,
				Exp:       0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.claims.Valid()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestJwtClaims_GetUID(t *testing.T) {
	claims := JwtClaims{Uid: json.Number("123456789")}
	uid, err := claims.GetUID()
	require.NoError(t, err)
	assert.Equal(t, int64(123456789), uid)
}

func TestJwtClaims_GetUID_Invalid(t *testing.T) {
	claims := JwtClaims{Uid: json.Number("not-a-number")}
	_, err := claims.GetUID()
	assert.Error(t, err)
}

func TestAccessToken_Valid(t *testing.T) {
	tests := []struct {
		name    string
		claims  AccessToken
		wantErr bool
	}{
		{
			name: "valid access token",
			claims: AccessToken{
				Nickname: "TestUser",
				Email:    "test@example.com",
				JwtClaims: JwtClaims{
					Uid:       json.Number("123456789"),
					TokenType: accessTokenType,
				},
			},
			wantErr: false,
		},
		{
			name: "missing email",
			claims: AccessToken{
				Nickname: "TestUser",
				JwtClaims: JwtClaims{
					Uid:       json.Number("123456789"),
					TokenType: accessTokenType,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.claims.Valid()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRefreshToken_Valid(t *testing.T) {
	tests := []struct {
		name    string
		claims  RefreshToken
		wantErr bool
	}{
		{
			name: "valid refresh token",
			claims: RefreshToken{
				JwtClaims: JwtClaims{
					Uid:       json.Number("123456789"),
					TokenType: refreshTokenType,
				},
			},
			wantErr: false,
		},
		{
			name: "wrong token type",
			claims: RefreshToken{
				JwtClaims: JwtClaims{
					Uid:       json.Number("123456789"),
					TokenType: accessTokenType,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.claims.Valid()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ==================== 边界测试 ====================

func TestGenerateAccessTokenWithRSA_NoKeyID(t *testing.T) {
	keyPair, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	token, err := GenerateAccessTokenWithRSA(keyPair.PrivateKey, "", 3600, 123, "test", "test@example.com")
	require.NoError(t, err)

	// 解析 token 检查 header
	claims, err := ParseAccessTokenWithRSA(token, keyPair.PublicKey)
	require.NoError(t, err)
	gotUID, err := claims.GetUID()
	require.NoError(t, err)
	assert.Equal(t, int64(123), gotUID)
}

func TestGenerateRefreshTokenWithRSA_NoKeyID(t *testing.T) {
	keyPair, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	token, err := GenerateRefreshTokenWithRSA(keyPair.PrivateKey, "", 3600, 123)
	require.NoError(t, err)

	// 解析 token 检查
	claims, err := ParseRefreshTokenWithRSA(token, keyPair.PublicKey)
	require.NoError(t, err)
	gotUID, err := claims.GetUID()
	require.NoError(t, err)
	assert.Equal(t, int64(123), gotUID)
}

func TestGenerateRSAKeyPair_DifferentKeys(t *testing.T) {
	keyPair1, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	keyPair2, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	// 验证生成的密钥对不同
	assert.NotEqual(t, keyPair1.KeyID, keyPair2.KeyID)
	assert.NotEqual(t, keyPair1.PrivateKey.D, keyPair2.PrivateKey.D)
}

func TestPrivateKeyToPEM_RoundTrip(t *testing.T) {
	keyPair, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	pemStr := PrivateKeyToPEM(keyPair.PrivateKey)
	parsedKey, err := ParseRSAPrivateKeyFromPEM(pemStr)
	require.NoError(t, err)

	// 验证解析后的密钥与原密钥相同
	assert.Equal(t, keyPair.PrivateKey.D, parsedKey.D)
	assert.Equal(t, keyPair.PrivateKey.N, parsedKey.N)
}

func TestPublicKeyToPEM_RoundTrip(t *testing.T) {
	keyPair, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	pemStr, err := PublicKeyToPEM(keyPair.PublicKey)
	require.NoError(t, err)

	parsedKey, err := ParseRSAPublicKeyFromPEM(pemStr)
	require.NoError(t, err)

	// 验证解析后的公钥与原公钥相同
	assert.Equal(t, keyPair.PublicKey.N, parsedKey.N)
	assert.Equal(t, keyPair.PublicKey.E, parsedKey.E)
}

func TestRSAKeyPair_ToJWK_Consistency(t *testing.T) {
	keyPair, err := GenerateRSAKeyPair(2048)
	require.NoError(t, err)

	jwk1 := keyPair.ToJWK()
	jwk2 := keyPair.ToJWK()

	// 同一密钥对生成的 JWK 应该相同
	assert.Equal(t, jwk1.Kid, jwk2.Kid)
	assert.Equal(t, jwk1.N, jwk2.N)
	assert.Equal(t, jwk1.E, jwk2.E)
}

func TestJWKS_ToJSON_Empty(t *testing.T) {
	jwks := JWKS{Keys: []JWK{}}
	jsonStr, err := jwks.ToJSON()
	require.NoError(t, err)
	assert.Contains(t, jsonStr, "keys")
	assert.Contains(t, jsonStr, "[]")
}

// BenchmarkGenerateRSAKeyPair 基准测试
func BenchmarkGenerateRSAKeyPair(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GenerateRSAKeyPair(2048)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGenerateAccessTokenWithRSA 基准测试
func BenchmarkGenerateAccessTokenWithRSA(b *testing.B) {
	keyPair, _ := GenerateRSAKeyPair(2048)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := GenerateAccessTokenWithRSA(keyPair.PrivateKey, keyPair.KeyID, 3600, 123, "test", "test@example.com")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseAccessTokenWithRSA 基准测试
func BenchmarkParseAccessTokenWithRSA(b *testing.B) {
	keyPair, _ := GenerateRSAKeyPair(2048)
	token, _ := GenerateAccessTokenWithRSA(keyPair.PrivateKey, keyPair.KeyID, 3600, 123, "test", "test@example.com")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ParseAccessTokenWithRSA(token, keyPair.PublicKey)
		if err != nil {
			b.Fatal(err)
		}
	}
}

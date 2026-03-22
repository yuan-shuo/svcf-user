package utils

import (
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
)

func TestGenerateAccessToken(t *testing.T) {
	t.Run("成功生成 accessToken", func(t *testing.T) {
		secret := "test-secret-key"
		expireSeconds := int64(3600)
		uid := int64(12345)
		nickname := "testuser"
		email := "test@example.com"

		token, err := GenerateAccessToken(secret, expireSeconds, uid, nickname, email)

		assert.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("生成的 token 可以正确解析", func(t *testing.T) {
		secret := "test-secret-key"
		expireSeconds := int64(3600)
		uid := int64(12345)
		nickname := "testuser"
		email := "test@example.com"

		token, err := GenerateAccessToken(secret, expireSeconds, uid, nickname, email)
		assert.NoError(t, err)

		// 解析 token
		claims, err := ParseToken(token, secret)
		assert.NoError(t, err)
		assert.NotNil(t, claims)

		// 验证 claims
		assert.Equal(t, float64(uid), claims["uid"])
		assert.Equal(t, nickname, claims["nickname"])
		assert.Equal(t, email, claims["email"])
		assert.Equal(t, "access", claims["type"])
	})

	t.Run("不同参数生成不同 token", func(t *testing.T) {
		secret := "test-secret-key"
		expireSeconds := int64(3600)

		token1, _ := GenerateAccessToken(secret, expireSeconds, 1, "user1", "user1@example.com")
		token2, _ := GenerateAccessToken(secret, expireSeconds, 2, "user2", "user2@example.com")

		assert.NotEqual(t, token1, token2)
	})
}

func TestGenerateRefreshToken(t *testing.T) {
	t.Run("成功生成 refreshToken", func(t *testing.T) {
		secret := "test-refresh-secret"
		expireSeconds := int64(7200)
		uid := int64(12345)

		token, err := GenerateRefreshToken(secret, expireSeconds, uid)

		assert.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("生成的 token 可以正确解析", func(t *testing.T) {
		secret := "test-refresh-secret"
		expireSeconds := int64(7200)
		uid := int64(12345)

		token, err := GenerateRefreshToken(secret, expireSeconds, uid)
		assert.NoError(t, err)

		// 解析 token
		claims, err := ParseToken(token, secret)
		assert.NoError(t, err)
		assert.NotNil(t, claims)

		// 验证 claims
		assert.Equal(t, float64(uid), claims["uid"])
		assert.Equal(t, "refresh", claims["type"])
	})
}

func TestParseToken(t *testing.T) {
	t.Run("成功解析有效的 accessToken", func(t *testing.T) {
		secret := "test-secret-key"
		token, _ := GenerateAccessToken(secret, 3600, 12345, "testuser", "test@example.com")

		claims, err := ParseToken(token, secret)

		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, float64(12345), claims["uid"])
		assert.Equal(t, "testuser", claims["nickname"])
		assert.Equal(t, "test@example.com", claims["email"])
		assert.Equal(t, "access", claims["type"])
	})

	t.Run("成功解析有效的 refreshToken", func(t *testing.T) {
		secret := "test-refresh-secret"
		token, _ := GenerateRefreshToken(secret, 7200, 12345)

		claims, err := ParseToken(token, secret)

		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, float64(12345), claims["uid"])
		assert.Equal(t, "refresh", claims["type"])
	})

	t.Run("错误的 secret 解析失败", func(t *testing.T) {
		secret := "test-secret-key"
		wrongSecret := "wrong-secret-key"
		token, _ := GenerateAccessToken(secret, 3600, 12345, "testuser", "test@example.com")

		claims, err := ParseToken(token, wrongSecret)

		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("解析无效的 token 字符串", func(t *testing.T) {
		secret := "test-secret-key"
		invalidToken := "invalid.token.string"

		claims, err := ParseToken(invalidToken, secret)

		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("解析空字符串失败", func(t *testing.T) {
		secret := "test-secret-key"

		claims, err := ParseToken("", secret)

		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("过期的 token 解析失败", func(t *testing.T) {
		secret := "test-secret-key"
		// 生成已经过期的 token（过期时间为 -1 秒）
		token, _ := GenerateAccessToken(secret, -1, 12345, "testuser", "test@example.com")

		claims, err := ParseToken(token, secret)

		assert.Error(t, err)
		assert.Nil(t, claims)
	})
}

func TestTokenExpiration(t *testing.T) {
	t.Run("accessToken 过期时间正确", func(t *testing.T) {
		secret := "test-secret-key"
		expireSeconds := int64(3600)
		token, _ := GenerateAccessToken(secret, expireSeconds, 12345, "testuser", "test@example.com")

		claims, err := ParseToken(token, secret)
		assert.NoError(t, err)

		// 验证过期时间
		exp := int64(claims["exp"].(float64))
		iat := int64(claims["iat"].(float64))
		assert.Equal(t, expireSeconds, exp-iat)
	})

	t.Run("refreshToken 过期时间正确", func(t *testing.T) {
		secret := "test-refresh-secret"
		expireSeconds := int64(7200)
		token, _ := GenerateRefreshToken(secret, expireSeconds, 12345)

		claims, err := ParseToken(token, secret)
		assert.NoError(t, err)

		// 验证过期时间
		exp := int64(claims["exp"].(float64))
		iat := int64(claims["iat"].(float64))
		assert.Equal(t, expireSeconds, exp-iat)
	})
}

func TestTokenClaims(t *testing.T) {
	t.Run("accessToken 包含所有必要字段", func(t *testing.T) {
		secret := "test-secret-key"
		token, _ := GenerateAccessToken(secret, 3600, 12345, "testuser", "test@example.com")

		claims, err := ParseToken(token, secret)
		assert.NoError(t, err)

		// 验证所有字段存在
		assert.NotNil(t, claims["uid"])
		assert.NotNil(t, claims["nickname"])
		assert.NotNil(t, claims["email"])
		assert.NotNil(t, claims["type"])
		assert.NotNil(t, claims["iat"])
		assert.NotNil(t, claims["exp"])

		// 验证类型为 access
		assert.Equal(t, "access", claims["type"])
	})

	t.Run("refreshToken 包含所有必要字段", func(t *testing.T) {
		secret := "test-refresh-secret"
		token, _ := GenerateRefreshToken(secret, 7200, 12345)

		claims, err := ParseToken(token, secret)
		assert.NoError(t, err)

		// 验证所有字段存在
		assert.NotNil(t, claims["uid"])
		assert.NotNil(t, claims["type"])
		assert.NotNil(t, claims["iat"])
		assert.NotNil(t, claims["exp"])

		// 验证类型为 refresh
		assert.Equal(t, "refresh", claims["type"])
	})
}

func TestTokenSigningMethod(t *testing.T) {
	t.Run("token 使用 HS256 签名方法", func(t *testing.T) {
		secret := "test-secret-key"
		token, _ := GenerateAccessToken(secret, 3600, 12345, "testuser", "test@example.com")

		// 解析但不验证
		parsedToken, _ := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		assert.Equal(t, jwt.SigningMethodHS256, parsedToken.Method)
	})
}

package userutils

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"user/internal/mock"
	"user/internal/model"
	"user/internal/utils"

	"github.com/stretchr/testify/assert"
	mocklib "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestGenerateAccessTokenWithKeyManager 测试使用 KeyManager 生成 accessToken
func TestGenerateAccessTokenWithKeyManager(t *testing.T) {
	keyManager, err := mock.NewTestKeyManager()
	require.NoError(t, err)
	defer keyManager.Destroy()

	user := &model.Users{
		SnowflakeId: 123456789,
		Nickname:    "TestUser",
		Email:       "test@example.com",
	}

	ctx := context.Background()
	token, err := GenerateAccessTokenWithKeyManager(ctx, keyManager, user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// 验证 token 可以被解析
	publicKey := keyManager.GetCurrentPrivateKey().PublicKey
	claims, err := utils.ParseAccessTokenWithRSA(token, &publicKey)
	require.NoError(t, err)
	gotUID, err := claims.GetUID()
	require.NoError(t, err)
	assert.Equal(t, user.SnowflakeId, gotUID)
	assert.Equal(t, user.Nickname, claims.Nickname)
	assert.Equal(t, user.Email, claims.Email)
}

// TestGenerateAccessTokenWithKeyManager_NilPrivateKey 测试私钥为空的情况
func TestGenerateAccessTokenWithKeyManager_NilPrivateKey(t *testing.T) {
	// 创建一个无效的 keyManager（没有 current key）
	keyManager := &utils.RSAKeyManager{}

	user := &model.Users{
		SnowflakeId: 123456789,
		Nickname:    "TestUser",
		Email:       "test@example.com",
	}

	ctx := context.Background()
	_, err := GenerateAccessTokenWithKeyManager(ctx, keyManager, user)
	assert.Error(t, err)
}

// TestGenerateRefreshTokenWithKeyManager 测试使用 KeyManager 生成 refreshToken
func TestGenerateRefreshTokenWithKeyManager(t *testing.T) {
	keyManager, err := mock.NewTestKeyManager()
	require.NoError(t, err)
	defer keyManager.Destroy()

	user := &model.Users{
		SnowflakeId: 123456789,
		Nickname:    "TestUser",
		Email:       "test@example.com",
	}

	ctx := context.Background()
	expireSeconds := int64(604800)
	token, err := GenerateRefreshTokenWithKeyManager(ctx, keyManager, expireSeconds, user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// 验证 token 可以被解析
	publicKey := keyManager.GetCurrentPrivateKey().PublicKey
	claims, err := utils.ParseRefreshTokenWithRSA(token, &publicKey)
	require.NoError(t, err)
	gotUID, err := claims.GetUID()
	require.NoError(t, err)
	assert.Equal(t, user.SnowflakeId, gotUID)
	assert.Equal(t, "refresh", claims.TokenType)
}

// TestGenerateRefreshTokenWithKeyManager_NilPrivateKey 测试私钥为空的情况
func TestGenerateRefreshTokenWithKeyManager_NilPrivateKey(t *testing.T) {
	// 创建一个无效的 keyManager（没有 current key）
	keyManager := &utils.RSAKeyManager{}

	user := &model.Users{
		SnowflakeId: 123456789,
		Nickname:    "TestUser",
		Email:       "test@example.com",
	}

	ctx := context.Background()
	_, err := GenerateRefreshTokenWithKeyManager(ctx, keyManager, 604800, user)
	assert.Error(t, err)
}

// TestGetUserByRefreshTokenWithKeyManager 测试使用 KeyManager 从 refreshToken 获取用户
func TestGetUserByRefreshTokenWithKeyManager(t *testing.T) {
	keyManager, err := mock.NewTestKeyManager()
	require.NoError(t, err)
	defer keyManager.Destroy()

	// 生成 refreshToken
	user := &model.Users{
		SnowflakeId: 123456789,
		Nickname:    "TestUser",
		Email:       "test@example.com",
	}
	ctx := context.Background()
	expireSeconds := int64(604800)
	rtBase64, err := GenerateRefreshTokenWithKeyManager(ctx, keyManager, expireSeconds, user)
	require.NoError(t, err)

	// 使用 mock 数据库
	usersModel := &mock.UsersModel{}
	usersModel.On("FindOneBySnowflakeId", ctx, user.SnowflakeId).Return(user, nil)

	// 解析获取用户
	resultUser, err := GetUserByRefreshTokenWithKeyManager(ctx, usersModel, rtBase64, keyManager)
	require.NoError(t, err)
	assert.Equal(t, user.SnowflakeId, resultUser.SnowflakeId)
	usersModel.AssertExpectations(t)
}

// TestGetUserByRefreshTokenWithKeyManager_InvalidToken 测试无效 token
func TestGetUserByRefreshTokenWithKeyManager_InvalidToken(t *testing.T) {
	keyManager, err := mock.NewTestKeyManager()
	require.NoError(t, err)
	defer keyManager.Destroy()

	usersModel := &mock.UsersModel{}
	ctx := context.Background()

	// 测试无效 token
	_, err = GetUserByRefreshTokenWithKeyManager(ctx, usersModel, "invalid.token", keyManager)
	assert.Error(t, err)
}

// TestGetUserByRefreshTokenWithKeyManager_MissingKid 测试缺少 kid 的 token
func TestGetUserByRefreshTokenWithKeyManager_MissingKid(t *testing.T) {
	keyManager, err := mock.NewTestKeyManager()
	require.NoError(t, err)
	defer keyManager.Destroy()

	// 生成没有 kid 的 token
	privateKey := keyManager.GetCurrentPrivateKey()
	token, _ := utils.GenerateRefreshTokenWithRSA(privateKey, "", 604800, 123456789)

	usersModel := &mock.UsersModel{}
	ctx := context.Background()

	_, err = GetUserByRefreshTokenWithKeyManager(ctx, usersModel, token, keyManager)
	assert.Error(t, err)
}

// TestGetUserByRefreshTokenWithKeyManager_WrongKey 测试使用错误密钥签发的 token
func TestGetUserByRefreshTokenWithKeyManager_WrongKey(t *testing.T) {
	keyManager1, _ := mock.NewTestKeyManager()
	defer keyManager1.Destroy()

	keyManager2, _ := mock.NewTestKeyManager()
	defer keyManager2.Destroy()

	// 使用 keyManager1 签发 token
	privateKey := keyManager1.GetCurrentPrivateKey()
	token, _ := utils.GenerateRefreshTokenWithRSA(privateKey, keyManager1.GetCurrentKeyID(), 604800, 123456789)

	usersModel := &mock.UsersModel{}
	ctx := context.Background()

	// 使用 keyManager2 验证（应该失败，因为 kid 不匹配）
	_, err := GetUserByRefreshTokenWithKeyManager(ctx, usersModel, token, keyManager2)
	assert.Error(t, err)
}

// TestGetUserByRefreshTokenWithKeyManager_UserNotFound 测试用户不存在
func TestGetUserByRefreshTokenWithKeyManager_UserNotFound(t *testing.T) {
	keyManager, err := mock.NewTestKeyManager()
	require.NoError(t, err)
	defer keyManager.Destroy()

	// 生成 refreshToken
	user := &model.Users{
		SnowflakeId: 999999999, // 不存在的用户
		Nickname:    "TestUser",
		Email:       "test@example.com",
	}
	ctx := context.Background()
	expireSeconds := int64(604800)
	rtBase64, err := GenerateRefreshTokenWithKeyManager(ctx, keyManager, expireSeconds, user)
	require.NoError(t, err)

	// 使用 mock 数据库返回错误
	usersModel := &mock.UsersModel{}
	usersModel.On("FindOneBySnowflakeId", ctx, user.SnowflakeId).Return(nil, model.ErrNotFound)

	_, err = GetUserByRefreshTokenWithKeyManager(ctx, usersModel, rtBase64, keyManager)
	assert.Error(t, err)
	usersModel.AssertExpectations(t)
}

// TestGetUserByAccessTokenClaims 测试从 accessToken claims 获取用户
func TestGetUserByAccessTokenClaims(t *testing.T) {
	user := &model.Users{
		SnowflakeId: 123456789,
		Nickname:    "TestUser",
		Email:       "test@example.com",
	}

	usersModel := &mock.UsersModel{}
	usersModel.On("FindOneBySnowflakeId", mocklib.Anything, user.SnowflakeId).Return(user, nil)

	// 创建包含完整 access token claims 的 context
	ctx := context.WithValue(context.Background(), "uid", json.Number("123456789"))
	ctx = context.WithValue(ctx, "type", "access")
	ctx = context.WithValue(ctx, "nickname", "TestUser")
	ctx = context.WithValue(ctx, "email", "test@example.com")

	resultUser, err := GetUserByAccessTokenClaims(ctx, usersModel)
	require.NoError(t, err)
	assert.Equal(t, user.SnowflakeId, resultUser.SnowflakeId)
	usersModel.AssertExpectations(t)
}

// TestGetUserByAccessTokenClaims_UIDNotFound 测试 context 中没有 uid
func TestGetUserByAccessTokenClaims_UIDNotFound(t *testing.T) {
	usersModel := &mock.UsersModel{}
	ctx := context.Background()

	_, err := GetUserByAccessTokenClaims(ctx, usersModel)
	assert.Error(t, err)
}

// TestGetUserByRefreshTokenClaims 测试从 refreshToken claims 获取用户
func TestGetUserByRefreshTokenClaims(t *testing.T) {
	user := &model.Users{
		SnowflakeId: 123456789,
		Nickname:    "TestUser",
		Email:       "test@example.com",
	}

	usersModel := &mock.UsersModel{}
	usersModel.On("FindOneBySnowflakeId", mocklib.Anything, user.SnowflakeId).Return(user, nil)

	// 创建包含完整 refresh token claims 的 context
	ctx := context.WithValue(context.Background(), "uid", json.Number("123456789"))
	ctx = context.WithValue(ctx, "type", "refresh")

	resultUser, err := GetUserByRefreshTokenClaims(ctx, usersModel)
	require.NoError(t, err)
	assert.Equal(t, user.SnowflakeId, resultUser.SnowflakeId)
	usersModel.AssertExpectations(t)
}

// TestGetUserByRefreshTokenClaims_UIDNotFound 测试 context 中没有 uid
func TestGetUserByRefreshTokenClaims_UIDNotFound(t *testing.T) {
	usersModel := &mock.UsersModel{}
	ctx := context.Background()

	_, err := GetUserByRefreshTokenClaims(ctx, usersModel)
	assert.Error(t, err)
}

// TestGetEmailByJwtCtx 测试从 context 获取邮箱
func TestGetEmailByJwtCtx(t *testing.T) {
	ctx := context.WithValue(context.Background(), "uid", json.Number("123456789"))
	ctx = context.WithValue(ctx, "type", "access")
	ctx = context.WithValue(ctx, "nickname", "TestUser")
	ctx = context.WithValue(ctx, "email", "test@example.com")

	email, err := GetEmailByJwtCtx(ctx)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", email)
}

// TestGetEmailByJwtCtx_NotFound 测试 context 中没有 email
func TestGetEmailByJwtCtx_NotFound(t *testing.T) {
	ctx := context.Background()

	_, err := GetEmailByJwtCtx(ctx)
	assert.Error(t, err)
}

// TestGetAccessTokenClaimsByJWT 测试从 JWT 解析 accessToken claims
func TestGetAccessTokenClaimsByJWT(t *testing.T) {
	secret := "test-secret"
	token, _ := utils.GenerateAccessToken(secret, 3600, 123456789, "TestUser", "test@example.com")

	ctx := context.Background()
	claims, err := GetAccessTokenClaimsByJWT(ctx, token, secret)
	require.NoError(t, err)
	gotUID, err := claims.GetUID()
	require.NoError(t, err)
	assert.Equal(t, int64(123456789), gotUID)
	assert.Equal(t, "TestUser", claims.Nickname)
}

// TestGetAccessTokenClaimsByJWT_Invalid 测试无效 JWT
func TestGetAccessTokenClaimsByJWT_Invalid(t *testing.T) {
	ctx := context.Background()
	_, err := GetAccessTokenClaimsByJWT(ctx, "invalid.token", "secret")
	assert.Error(t, err)
}

// TestGetRefreshTokenClaimsByJWT 测试从 JWT 解析 refreshToken claims
func TestGetRefreshTokenClaimsByJWT(t *testing.T) {
	secret := "test-secret"
	token, _ := utils.GenerateRefreshToken(secret, 3600, 123456789)

	ctx := context.Background()
	claims, err := GetRefreshTokenClaimsByJWT(ctx, token, secret)
	require.NoError(t, err)
	gotUID, err := claims.GetUID()
	require.NoError(t, err)
	assert.Equal(t, int64(123456789), gotUID)
	assert.Equal(t, "refresh", claims.TokenType)
}

// TestGetRefreshTokenClaimsByJWT_Invalid 测试无效 JWT
func TestGetRefreshTokenClaimsByJWT_Invalid(t *testing.T) {
	ctx := context.Background()
	_, err := GetRefreshTokenClaimsByJWT(ctx, "invalid.token", "secret")
	assert.Error(t, err)
}

// TestGetUserByRefreshTokenWithKeyManager_ExpiredKey 测试使用过期的密钥
func TestGetUserByRefreshTokenWithKeyManager_ExpiredKey(t *testing.T) {
	// 创建配置，使用很短的 grace period（但必须 >= token_expire_secs）
	cfg := &utils.KeyManagerConfig{
		Bits:             2048,
		RotationInterval: 24 * time.Hour,
		GracePeriod:      2 * time.Second,
		TokenExpireSecs:  1, // 1秒过期时间
	}

	keyManager, err := utils.NewRSAKeyManagerWithConfig(cfg)
	require.NoError(t, err)
	defer keyManager.Destroy()

	// 生成 refreshToken
	user := &model.Users{
		SnowflakeId: 123456789,
		Nickname:    "TestUser",
		Email:       "test@example.com",
	}
	ctx := context.Background()
	expireSeconds := int64(604800)
	rtBase64, err := GenerateRefreshTokenWithKeyManager(ctx, keyManager, expireSeconds, user)
	require.NoError(t, err)

	// 轮换密钥
	err = keyManager.Rotate(ctx)
	require.NoError(t, err)

	// 等待 grace period 过期
	time.Sleep(3 * time.Second)

	// 此时旧密钥应该已过期
	usersModel := &mock.UsersModel{}
	_, err = GetUserByRefreshTokenWithKeyManager(ctx, usersModel, rtBase64, keyManager)
	// 应该失败，因为密钥已过期
	assert.Error(t, err)
}

// TestGetUserByRefreshTokenWithKeyManager_PreviousKey 测试使用 previous 密钥
func TestGetUserByRefreshTokenWithKeyManager_PreviousKey(t *testing.T) {
	cfg := &utils.KeyManagerConfig{
		Bits:             2048,
		RotationInterval: 24 * time.Hour,
		GracePeriod:      1 * time.Hour,
		TokenExpireSecs:  60,
	}

	keyManager, err := utils.NewRSAKeyManagerWithConfig(cfg)
	require.NoError(t, err)
	defer keyManager.Destroy()

	// 生成 refreshToken
	user := &model.Users{
		SnowflakeId: 123456789,
		Nickname:    "TestUser",
		Email:       "test@example.com",
	}
	ctx := context.Background()
	expireSeconds := int64(604800)
	rtBase64, err := GenerateRefreshTokenWithKeyManager(ctx, keyManager, expireSeconds, user)
	require.NoError(t, err)

	oldKeyID := keyManager.GetCurrentKeyID()

	// 轮换密钥
	err = keyManager.Rotate(ctx)
	require.NoError(t, err)

	// 此时旧密钥应该还是 previous，未过期
	usersModel := &mock.UsersModel{}
	usersModel.On("FindOneBySnowflakeId", ctx, user.SnowflakeId).Return(user, nil)
	resultUser, err := GetUserByRefreshTokenWithKeyManager(ctx, usersModel, rtBase64, keyManager)
	require.NoError(t, err)
	assert.Equal(t, user.SnowflakeId, resultUser.SnowflakeId)
	usersModel.AssertExpectations(t)

	// 验证使用的是 previous 密钥
	assert.NotEqual(t, keyManager.GetCurrentKeyID(), oldKeyID)
}

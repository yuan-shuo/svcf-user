package user_noauth

import (
	"context"
	"testing"

	"user/internal/config"
	"user/internal/errs"
	"user/internal/middleware"
	"user/internal/mock"
	"user/internal/model"
	"user/internal/svc"
	"user/internal/types"
	"user/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func TestRefreshTokenLogic_RefreshToken_Success(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(mock.UsersModel)

	// 准备测试数据
	uid := int64(12345)
	email := "test@example.com"
	nickname := "testuser"

	user := &model.Users{
		Id:          1,
		SnowflakeId: uid,
		Email:       email,
		Nickname:    nickname,
	}

	// 创建 RSA KeyManager
	refreshExpire := int64(7200)
	keyManager, err := mock.NewTestKeyManager()
	assert.NoError(t, err)

	// 生成有效的 refresh token（使用 RSA）
	refreshToken, err := utils.GenerateRefreshTokenWithRSA(
		keyManager.GetCurrentPrivateKey(),
		keyManager.GetCurrentKeyID(),
		refreshExpire,
		uid,
	)
	assert.NoError(t, err)

	// 将 refresh token 放入 context（模拟中间件从 Cookie 读取）
	ctx = context.WithValue(ctx, middleware.RefreshTokenKey{}, refreshToken)

	// 设置 mock 期望
	mockUsersModel.On("FindOneBySnowflakeId", ctx, uid).Return(user, nil)

	svcCtx := &svc.ServiceContext{
		UsersModel: mockUsersModel,
		Config: config.Config{
			Auth: config.Auth{
				AccessExpire: 3600,
			},
			RefreshExpire: refreshExpire,
		},
		Metrics:    mock.GetTestMetrics(),
		KeyManager: keyManager,
	}

	logic := NewRefreshTokenLogic(ctx, svcCtx)
	req := &types.RefreshTokenReq{}

	resp, err := logic.RefreshToken(req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	// Token 通过 Cookie 返回，不再在响应体中
	assert.Equal(t, int64(3600), resp.ExpiresIn)
	mockUsersModel.AssertExpectations(t)
}

func TestRefreshTokenLogic_RefreshToken_NoTokenInContext(t *testing.T) {
	ctx := context.Background()

	// 创建 RSA KeyManager
	keyManager, err := mock.NewTestKeyManager()
	assert.NoError(t, err)

	svcCtx := &svc.ServiceContext{
		Config:     config.Config{},
		Metrics:    mock.GetTestMetrics(),
		KeyManager: keyManager,
	}

	logic := NewRefreshTokenLogic(ctx, svcCtx)
	req := &types.RefreshTokenReq{}

	resp, err := logic.RefreshToken(req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeUnauthorized))
}

func TestRefreshTokenLogic_RefreshToken_InvalidToken(t *testing.T) {
	ctx := context.Background()

	// 创建 RSA KeyManager
	keyManager, err := mock.NewTestKeyManager()
	assert.NoError(t, err)

	svcCtx := &svc.ServiceContext{
		Config:     config.Config{},
		Metrics:    mock.GetTestMetrics(),
		KeyManager: keyManager,
	}

	// 将无效 token 放入 context
	ctx = context.WithValue(ctx, middleware.RefreshTokenKey{}, "invalid-token")

	logic := NewRefreshTokenLogic(ctx, svcCtx)
	req := &types.RefreshTokenReq{}

	resp, err := logic.RefreshToken(req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidToken))
}

func TestRefreshTokenLogic_RefreshToken_WrongKey(t *testing.T) {
	ctx := context.Background()

	// 创建两个不同的 KeyManager（模拟密钥轮换后的情况）
	keyManager1, err := mock.NewTestKeyManager()
	assert.NoError(t, err)
	keyManager2, err := mock.NewTestKeyManager()
	assert.NoError(t, err)

	// 用 keyManager1 生成 token
	uid := int64(12345)
	wrongToken, err := utils.GenerateRefreshTokenWithRSA(
		keyManager1.GetCurrentPrivateKey(),
		keyManager1.GetCurrentKeyID(),
		7200,
		uid,
	)
	assert.NoError(t, err)

	// 将 token 放入 context
	ctx = context.WithValue(ctx, middleware.RefreshTokenKey{}, wrongToken)

	// 但验证时使用 keyManager2（不同的密钥）
	svcCtx := &svc.ServiceContext{
		Config:     config.Config{},
		Metrics:    mock.GetTestMetrics(),
		KeyManager: keyManager2,
	}

	logic := NewRefreshTokenLogic(ctx, svcCtx)
	req := &types.RefreshTokenReq{}

	resp, err := logic.RefreshToken(req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidToken))
}

func TestRefreshTokenLogic_RefreshToken_UserNotFound(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(mock.UsersModel)

	// 创建 RSA KeyManager
	keyManager, err := mock.NewTestKeyManager()
	assert.NoError(t, err)

	// 生成有效的 refresh token
	uid := int64(99999)
	refreshExpire := int64(7200)
	refreshToken, err := utils.GenerateRefreshTokenWithRSA(
		keyManager.GetCurrentPrivateKey(),
		keyManager.GetCurrentKeyID(),
		refreshExpire,
		uid,
	)
	assert.NoError(t, err)

	// 将 token 放入 context
	ctx = context.WithValue(ctx, middleware.RefreshTokenKey{}, refreshToken)

	// 设置 mock 期望：用户不存在
	mockUsersModel.On("FindOneBySnowflakeId", ctx, uid).Return(nil, sqlx.ErrNotFound)

	svcCtx := &svc.ServiceContext{
		UsersModel: mockUsersModel,
		Config: config.Config{
			Auth: config.Auth{
				AccessExpire: 3600,
			},
			RefreshExpire: refreshExpire,
		},
		Metrics:    mock.GetTestMetrics(),
		KeyManager: keyManager,
	}

	logic := NewRefreshTokenLogic(ctx, svcCtx)
	req := &types.RefreshTokenReq{}

	resp, err := logic.RefreshToken(req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeUserNotFound))
	mockUsersModel.AssertExpectations(t)
}

func TestRefreshTokenLogic_RefreshToken_WithCookieSetter(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(mock.UsersModel)

	// 准备测试数据
	uid := int64(12345)
	email := "test@example.com"
	nickname := "testuser"

	user := &model.Users{
		Id:          1,
		SnowflakeId: uid,
		Email:       email,
		Nickname:    nickname,
	}

	// 创建 RSA KeyManager
	keyManager, err := mock.NewTestKeyManager()
	assert.NoError(t, err)

	// 生成有效的 refresh token
	refreshExpire := int64(7200)
	refreshToken, err := utils.GenerateRefreshTokenWithRSA(
		keyManager.GetCurrentPrivateKey(),
		keyManager.GetCurrentKeyID(),
		refreshExpire,
		uid,
	)
	assert.NoError(t, err)

	// 将 refresh token 放入 context
	ctx = context.WithValue(ctx, middleware.RefreshTokenKey{}, refreshToken)

	// 创建 CookieSetter 并放入 context
	setter := &middleware.CookieSetter{}
	ctx = context.WithValue(ctx, middleware.CookieSetter{}, setter)

	// 设置 mock 期望
	mockUsersModel.On("FindOneBySnowflakeId", ctx, uid).Return(user, nil)

	svcCtx := &svc.ServiceContext{
		UsersModel: mockUsersModel,
		Config: config.Config{
			Auth: config.Auth{
				AccessExpire: 3600,
			},
			RefreshExpire: refreshExpire,
		},
		Metrics:    mock.GetTestMetrics(),
		KeyManager: keyManager,
	}

	logic := NewRefreshTokenLogic(ctx, svcCtx)
	req := &types.RefreshTokenReq{}

	resp, err := logic.RefreshToken(req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	mockUsersModel.AssertExpectations(t)
}

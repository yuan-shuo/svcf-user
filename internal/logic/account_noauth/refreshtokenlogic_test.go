package account_noauth

import (
	"context"
	"errors"
	"testing"

	"user/internal/config"
	"user/internal/errs"
	"user/internal/mock"
	"user/internal/model"
	"user/internal/svc"
	"user/internal/types"
	"user/internal/utils"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/logx"
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

	// 生成有效的 refresh token
	refreshSecret := "test-refresh-secret"
	refreshExpire := int64(7200)
	refreshToken, err := utils.GenerateRefreshToken(refreshSecret, refreshExpire, uid)
	assert.NoError(t, err)

	// 设置 mock 期望
	mockUsersModel.On("FindOneBySnowflakeId", ctx, uid).Return(user, nil)

	svcCtx := &svc.ServiceContext{
		UsersModel: mockUsersModel,
		Config: config.Config{
			Auth: config.Auth{
				AccessSecret: "test-access-secret",
				AccessExpire: 3600,
			},
			RefreshSecret: refreshSecret,
			RefreshExpire: refreshExpire,
		},
	}

	logic := NewRefreshTokenLogic(ctx, svcCtx)
	req := &types.RefreshTokenReq{
		RefreshToken: refreshToken,
	}

	resp, err := logic.RefreshToken(req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, int64(3600), resp.ExpiresIn)
	mockUsersModel.AssertExpectations(t)
}

func TestRefreshTokenLogic_RefreshToken_InvalidToken(t *testing.T) {
	ctx := context.Background()

	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			RefreshSecret: "test-refresh-secret",
		},
	}

	logic := NewRefreshTokenLogic(ctx, svcCtx)
	req := &types.RefreshTokenReq{
		RefreshToken: "invalid-token",
	}

	resp, err := logic.RefreshToken(req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidToken))
}

func TestRefreshTokenLogic_RefreshToken_WrongSecret(t *testing.T) {
	ctx := context.Background()

	// 用错误的 secret 生成 token
	uid := int64(12345)
	wrongToken, err := utils.GenerateRefreshToken("wrong-secret", 7200, uid)
	assert.NoError(t, err)

	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			RefreshSecret: "correct-secret", // 不同的 secret
		},
	}

	logic := NewRefreshTokenLogic(ctx, svcCtx)
	req := &types.RefreshTokenReq{
		RefreshToken: wrongToken,
	}

	resp, err := logic.RefreshToken(req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidToken))
}

func TestRefreshTokenLogic_RefreshToken_ExpiredToken(t *testing.T) {
	ctx := context.Background()

	// 生成已过期的 token
	refreshSecret := "test-refresh-secret"
	now := jwt.TimeFunc()
	claims := jwt.MapClaims{
		"uid":    int64(12345),
		"type":   "refresh",
		"iat":    now.Add(-2 * 3600).Unix(), // 2小时前签发
		"exp":    now.Add(-3600).Unix(),     // 1小时前过期
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredToken, err := token.SignedString([]byte(refreshSecret))
	assert.NoError(t, err)

	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			RefreshSecret: refreshSecret,
		},
	}

	logic := NewRefreshTokenLogic(ctx, svcCtx)
	req := &types.RefreshTokenReq{
		RefreshToken: expiredToken,
	}

	resp, err := logic.RefreshToken(req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestRefreshTokenLogic_RefreshToken_NotRefreshTokenType(t *testing.T) {
	ctx := context.Background()

	// 生成 access token 而不是 refresh token
	refreshSecret := "test-refresh-secret"
	accessToken, err := utils.GenerateAccessToken(refreshSecret, 7200, 12345, "testuser", "test@example.com")
	assert.NoError(t, err)

	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			RefreshSecret: refreshSecret,
		},
	}

	logic := NewRefreshTokenLogic(ctx, svcCtx)
	req := &types.RefreshTokenReq{
		RefreshToken: accessToken, // 使用 access token
	}

	resp, err := logic.RefreshToken(req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidToken))
}

func TestRefreshTokenLogic_RefreshToken_UserNotFound(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(mock.UsersModel)

	uid := int64(12345)
	refreshSecret := "test-refresh-secret"
	refreshExpire := int64(7200)

	// 生成有效的 refresh token
	refreshToken, err := utils.GenerateRefreshToken(refreshSecret, refreshExpire, uid)
	assert.NoError(t, err)

	// 设置 mock 期望：用户不存在
	mockUsersModel.On("FindOneBySnowflakeId", ctx, uid).Return(nil, sqlx.ErrNotFound)

	svcCtx := &svc.ServiceContext{
		UsersModel: mockUsersModel,
		Config: config.Config{
			Auth: config.Auth{
				AccessSecret: "test-access-secret",
				AccessExpire: 3600,
			},
			RefreshSecret: refreshSecret,
			RefreshExpire: refreshExpire,
		},
	}

	logic := NewRefreshTokenLogic(ctx, svcCtx)
	req := &types.RefreshTokenReq{
		RefreshToken: refreshToken,
	}

	resp, err := logic.RefreshToken(req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeUserNotFound))
	mockUsersModel.AssertExpectations(t)
}

func TestRefreshTokenLogic_RefreshToken_DatabaseError(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(mock.UsersModel)

	uid := int64(12345)
	refreshSecret := "test-refresh-secret"
	refreshExpire := int64(7200)

	// 生成有效的 refresh token
	refreshToken, err := utils.GenerateRefreshToken(refreshSecret, refreshExpire, uid)
	assert.NoError(t, err)

	// 设置 mock 期望：数据库错误
	mockUsersModel.On("FindOneBySnowflakeId", ctx, uid).Return(nil, errors.New("database error"))

	svcCtx := &svc.ServiceContext{
		UsersModel: mockUsersModel,
		Config: config.Config{
			Auth: config.Auth{
				AccessSecret: "test-access-secret",
				AccessExpire: 3600,
			},
			RefreshSecret: refreshSecret,
			RefreshExpire: refreshExpire,
		},
	}

	// 禁用日志输出
	logx.Disable()

	logic := NewRefreshTokenLogic(ctx, svcCtx)
	req := &types.RefreshTokenReq{
		RefreshToken: refreshToken,
	}

	resp, err := logic.RefreshToken(req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError))
	mockUsersModel.AssertExpectations(t)
}

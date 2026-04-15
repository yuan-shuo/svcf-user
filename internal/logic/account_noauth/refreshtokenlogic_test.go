package account_noauth

import (
	"context"
	"testing"

	"user/internal/config"
	"user/internal/errs"
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

	// 鍑嗗娴嬭瘯鏁版嵁
	uid := int64(12345)
	email := "test@example.com"
	nickname := "testuser"

	user := &model.Users{
		Id:          1,
		SnowflakeId: uid,
		Email:       email,
		Nickname:    nickname,
	}

	// 鐢熸垚鏈夋晥鐨?refresh token
	refreshSecret := "test-refresh-secret"
	refreshExpire := int64(7200)
	refreshToken, err := utils.GenerateRefreshToken(refreshSecret, refreshExpire, uid)
	assert.NoError(t, err)

	// 璁剧疆 mock 鏈熸湜
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
		Metrics: mock.GetTestMetrics(),
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
		Metrics: mock.GetTestMetrics(),
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

	// 鐢ㄩ敊璇殑 secret 鐢熸垚 token
	uid := int64(12345)
	wrongToken, err := utils.GenerateRefreshToken("wrong-secret", 7200, uid)
	assert.NoError(t, err)

	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			RefreshSecret: "correct-secret", // 涓嶅悓鐨?secret
		},
		Metrics: mock.GetTestMetrics(),
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

	// 鐢熸垚宸茶繃鏈熺殑 token锛堜娇鐢ㄨ礋鐨勮繃鏈熸椂闂达級
	refreshSecret := "test-refresh-secret"
	expiredToken, err := utils.GenerateRefreshToken(refreshSecret, -1, 12345)
	assert.NoError(t, err)

	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			RefreshSecret: refreshSecret,
		},
		Metrics: mock.GetTestMetrics(),
	}

	logic := NewRefreshTokenLogic(ctx, svcCtx)
	req := &types.RefreshTokenReq{
		RefreshToken: expiredToken,
	}

	resp, err := logic.RefreshToken(req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidToken))
}

func TestRefreshTokenLogic_RefreshToken_NotRefreshTokenType(t *testing.T) {
	ctx := context.Background()

	// 鐢熸垚 access token 鑰屼笉鏄?refresh token
	refreshSecret := "test-refresh-secret"
	accessToken, err := utils.GenerateAccessToken(refreshSecret, 7200, 12345, "testuser", "test@example.com")
	assert.NoError(t, err)

	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			RefreshSecret: refreshSecret,
		},
		Metrics: mock.GetTestMetrics(),
	}

	logic := NewRefreshTokenLogic(ctx, svcCtx)
	req := &types.RefreshTokenReq{
		RefreshToken: accessToken, // 浣跨敤 access token
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

	// 鐢熸垚鏈夋晥鐨?refresh token
	refreshToken, err := utils.GenerateRefreshToken(refreshSecret, refreshExpire, uid)
	assert.NoError(t, err)

	// 璁剧疆 mock 鏈熸湜锛氱敤鎴蜂笉瀛樺湪
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
		Metrics: mock.GetTestMetrics(),
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

func TestRefreshTokenLogic_RefreshToken_DBError(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(mock.UsersModel)

	uid := int64(12345)
	refreshSecret := "test-refresh-secret"
	refreshExpire := int64(7200)

	// 鐢熸垚鏈夋晥鐨?refresh token
	refreshToken, err := utils.GenerateRefreshToken(refreshSecret, refreshExpire, uid)
	assert.NoError(t, err)

	// 璁剧疆 mock 鏈熸湜锛氭暟鎹簱閿欒
	mockUsersModel.On("FindOneBySnowflakeId", ctx, uid).Return(nil, assert.AnError)

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
		Metrics: mock.GetTestMetrics(),
	}

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

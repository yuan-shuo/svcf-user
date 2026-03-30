package accutil

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"user/internal/config"
	"user/internal/errs"
	"user/internal/mock"
	"user/internal/model"
	"user/internal/svc"
	"user/internal/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// setupJwtTest 设置 JWT 测试环境
func setupJwtTest(t *testing.T) (*miniredis.Miniredis, *redis.Redis, *mock.UsersModel, *svc.ServiceContext) {
	// 创建 miniredis
	s := miniredis.RunT(t)

	// 创建 redis 客户端
	rds := redis.New(s.Addr())

	// 创建 mock users model
	mockUsersModel := new(mock.UsersModel)

	// 创建 service context
	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			Auth: config.Auth{
				AccessSecret: "test-access-secret",
				AccessExpire: 3600,
			},
			RefreshSecret: "test-refresh-secret",
			RefreshExpire: 7200,
			VerifyCodeConfig: config.VerifyCodeConfig{
				Type: config.VerifyCodeType{
					Register:      "register",
					ResetPassword: "reset_password",
				},
				Redis: config.VerifyCodeRedisConfig{
					KeyPrefix: "account",
				},
			},
		},
		Redis:      rds,
		UsersModel: mockUsersModel,
	}

	// 初始化雪花算法
	err := utils.InitSonyflake(1, "2024-01-01")
	assert.NoError(t, err)

	return s, rds, mockUsersModel, svcCtx
}

// ==================== GetUserByAccessTokenClaims 测试 ====================

func TestGetUserByAccessTokenClaims_Success(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupJwtTest(t)

	// 创建包含 AccessToken claims 的 context
	claims := &utils.AccessToken{
		Nickname: "testuser",
		Email:    "test@example.com",
		JwtClaims: utils.JwtClaims{
			Uid:       json.Number("12345"),
			Version:   "1.0",
			TokenType: "access",
			Iat:       time.Now().Unix(),
			Exp:       time.Now().Add(time.Hour).Unix(),
		},
	}
	ctx := context.WithValue(context.Background(), "claims", claims)

	expectedUser := &model.Users{
		Id:           1,
		SnowflakeId:  12345,
		Email:        "test@example.com",
		Nickname:     "testuser",
		PasswordHash: "hashedpassword",
	}
	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(expectedUser, nil)

	user, err := GetUserByAccessTokenClaims(ctx, svcCtx)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, int64(12345), user.SnowflakeId)
	mockUsersModel.AssertExpectations(t)
}

func TestGetUserByAccessTokenClaims_ClaimsNotFound(t *testing.T) {
	_, _, _, svcCtx := setupJwtTest(t)

	ctx := context.Background()

	user, err := GetUserByAccessTokenClaims(ctx, svcCtx)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
}

func TestGetUserByAccessTokenClaims_UserNotFound(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupJwtTest(t)

	claims := &utils.AccessToken{
		Nickname: "testuser",
		Email:    "test@example.com",
		JwtClaims: utils.JwtClaims{
			Uid:       json.Number("12345"),
			Version:   "1.0",
			TokenType: "access",
			Iat:       time.Now().Unix(),
			Exp:       time.Now().Add(time.Hour).Unix(),
		},
	}
	ctx := context.WithValue(context.Background(), "claims", claims)

	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(nil, model.ErrNotFound)

	user, err := GetUserByAccessTokenClaims(ctx, svcCtx)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeUserNotFound), "应该是用户不存在错误")
	mockUsersModel.AssertExpectations(t)
}

func TestGetUserByAccessTokenClaims_DBError(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupJwtTest(t)

	claims := &utils.AccessToken{
		Nickname: "testuser",
		Email:    "test@example.com",
		JwtClaims: utils.JwtClaims{
			Uid:       json.Number("12345"),
			Version:   "1.0",
			TokenType: "access",
			Iat:       time.Now().Unix(),
			Exp:       time.Now().Add(time.Hour).Unix(),
		},
	}
	ctx := context.WithValue(context.Background(), "claims", claims)

	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(nil, assert.AnError)

	user, err := GetUserByAccessTokenClaims(ctx, svcCtx)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

// ==================== GetUserByRefreshTokenClaims 测试 ====================

func TestGetUserByRefreshTokenClaims_Success(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupJwtTest(t)

	claims := &utils.RefreshToken{
		JwtClaims: utils.JwtClaims{
			Uid:       json.Number("12345"),
			Version:   "1.0",
			TokenType: "refresh",
			Iat:       time.Now().Unix(),
			Exp:       time.Now().Add(time.Hour).Unix(),
		},
	}
	ctx := context.WithValue(context.Background(), "claims", claims)

	expectedUser := &model.Users{
		Id:           1,
		SnowflakeId:  12345,
		Email:        "test@example.com",
		Nickname:     "testuser",
		PasswordHash: "hashedpassword",
	}
	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(expectedUser, nil)

	user, err := GetUserByRefreshTokenClaims(ctx, svcCtx)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, int64(12345), user.SnowflakeId)
	mockUsersModel.AssertExpectations(t)
}

func TestGetUserByRefreshTokenClaims_ClaimsNotFound(t *testing.T) {
	_, _, _, svcCtx := setupJwtTest(t)

	ctx := context.Background()

	user, err := GetUserByRefreshTokenClaims(ctx, svcCtx)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
}

// ==================== GetUserByRefreshToken 测试 ====================

func TestGetUserByRefreshToken_Success(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupJwtTest(t)

	// 生成有效的 refresh token
	rt, err := utils.GenerateRefreshToken(svcCtx.Config.RefreshSecret, svcCtx.Config.RefreshExpire, 12345)
	assert.NoError(t, err)

	ctx := context.Background()
	expectedUser := &model.Users{
		Id:           1,
		SnowflakeId:  12345,
		Email:        "test@example.com",
		Nickname:     "testuser",
		PasswordHash: "hashedpassword",
	}
	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(expectedUser, nil)

	user, err := GetUserByRefreshToken(ctx, svcCtx, rt)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, int64(12345), user.SnowflakeId)
	mockUsersModel.AssertExpectations(t)
}

func TestGetUserByRefreshToken_InvalidToken(t *testing.T) {
	_, _, _, svcCtx := setupJwtTest(t)

	ctx := context.Background()

	user, err := GetUserByRefreshToken(ctx, svcCtx, "invalid-token")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidToken), "应该是令牌无效错误")
}

func TestGetUserByRefreshToken_WrongSecret(t *testing.T) {
	_, _, _, svcCtx := setupJwtTest(t)

	// 使用错误的密钥生成 token
	rt, err := utils.GenerateRefreshToken("wrong-secret", svcCtx.Config.RefreshExpire, 12345)
	assert.NoError(t, err)

	ctx := context.Background()

	user, err := GetUserByRefreshToken(ctx, svcCtx, rt)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidToken), "应该是令牌无效错误")
}

func TestGetUserByRefreshToken_UserNotFound(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupJwtTest(t)

	rt, err := utils.GenerateRefreshToken(svcCtx.Config.RefreshSecret, svcCtx.Config.RefreshExpire, 12345)
	assert.NoError(t, err)

	ctx := context.Background()
	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(nil, model.ErrNotFound)

	user, err := GetUserByRefreshToken(ctx, svcCtx, rt)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeUserNotFound), "应该是用户不存在错误")
	mockUsersModel.AssertExpectations(t)
}

// ==================== GetAccessTokenClaimsByJWT 测试 ====================

func TestGetAccessTokenClaimsByJWT_Success(t *testing.T) {
	_, _, _, svcCtx := setupJwtTest(t)

	token, err := utils.GenerateAccessToken(
		svcCtx.Config.Auth.AccessSecret,
		svcCtx.Config.Auth.AccessExpire,
		12345,
		"testuser",
		"test@example.com",
	)
	assert.NoError(t, err)

	claims, err := GetAccessTokenClaimsByJWT(token, svcCtx.Config.Auth.AccessSecret)

	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.Equal(t, "testuser", claims.Nickname)
	uid, _ := claims.GetUID()
	assert.Equal(t, int64(12345), uid)
}

func TestGetAccessTokenClaimsByJWT_InvalidToken(t *testing.T) {
	_, _, _, svcCtx := setupJwtTest(t)

	claims, err := GetAccessTokenClaimsByJWT("invalid-token", svcCtx.Config.Auth.AccessSecret)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidToken), "应该是无效token错误")
}

func TestGetAccessTokenClaimsByJWT_WrongSecret(t *testing.T) {
	_, _, _, svcCtx := setupJwtTest(t)

	token, _ := utils.GenerateAccessToken(
		svcCtx.Config.Auth.AccessSecret,
		svcCtx.Config.Auth.AccessExpire,
		12345,
		"testuser",
		"test@example.com",
	)

	claims, err := GetAccessTokenClaimsByJWT(token, "wrong-secret")

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidToken), "应该是无效token错误")
}

func TestGetAccessTokenClaimsByJWT_Expired(t *testing.T) {
	_, _, _, svcCtx := setupJwtTest(t)

	// 生成已过期的 token
	token, _ := utils.GenerateAccessToken(
		svcCtx.Config.Auth.AccessSecret,
		-1, // 已过期
		12345,
		"testuser",
		"test@example.com",
	)

	claims, err := GetAccessTokenClaimsByJWT(token, svcCtx.Config.Auth.AccessSecret)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidToken), "应该是无效token错误")
}

// ==================== GetRefreshTokenClaimsByJWT 测试 ====================

func TestGetRefreshTokenClaimsByJWT_Success(t *testing.T) {
	_, _, _, svcCtx := setupJwtTest(t)

	token, err := utils.GenerateRefreshToken(
		svcCtx.Config.RefreshSecret,
		svcCtx.Config.RefreshExpire,
		12345,
	)
	assert.NoError(t, err)

	claims, err := GetRefreshTokenClaimsByJWT(token, svcCtx.Config.RefreshSecret)

	assert.NoError(t, err)
	assert.NotNil(t, claims)
	uid, _ := claims.GetUID()
	assert.Equal(t, int64(12345), uid)
}

func TestGetRefreshTokenClaimsByJWT_InvalidToken(t *testing.T) {
	_, _, _, svcCtx := setupJwtTest(t)

	claims, err := GetRefreshTokenClaimsByJWT("invalid-token", svcCtx.Config.RefreshSecret)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidToken), "应该是无效token错误")
}

func TestGetRefreshTokenClaimsByJWT_WrongTokenType(t *testing.T) {
	_, _, _, svcCtx := setupJwtTest(t)

	// 使用 access token 尝试解析为 refresh token
	token, _ := utils.GenerateAccessToken(
		svcCtx.Config.RefreshSecret, // 注意：这里使用 RefreshSecret
		svcCtx.Config.RefreshExpire,
		12345,
		"testuser",
		"test@example.com",
	)

	claims, err := GetRefreshTokenClaimsByJWT(token, svcCtx.Config.RefreshSecret)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidToken), "应该是无效token错误")
}

// ==================== GetEmailByJwtCtx 测试 ====================

func TestGetEmailByJwtCtx_Success(t *testing.T) {
	claims := &utils.AccessToken{
		Nickname: "testuser",
		Email:    "test@example.com",
		JwtClaims: utils.JwtClaims{
			Uid:       json.Number("12345"),
			Version:   "1.0",
			TokenType: "access",
			Iat:       time.Now().Unix(),
			Exp:       time.Now().Add(time.Hour).Unix(),
		},
	}
	ctx := context.WithValue(context.Background(), "claims", claims)

	email, err := GetEmailByJwtCtx(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "test@example.com", email)
}

func TestGetEmailByJwtCtx_ClaimsNotFound(t *testing.T) {
	ctx := context.Background()

	email, err := GetEmailByJwtCtx(ctx)

	assert.Error(t, err)
	assert.Empty(t, email)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
}

// ==================== GenerateAccessToken (wrapper) 测试 ====================

func TestGenerateAccessToken_Wrapper_Success(t *testing.T) {
	_, _, _, svcCtx := setupJwtTest(t)

	user := &model.Users{
		Id:           1,
		SnowflakeId:  12345,
		Email:        "test@example.com",
		Nickname:     "testuser",
		PasswordHash: "hashedpassword",
	}

	token, err := GenerateAccessToken(svcCtx.Config, user)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// 验证 token 可以解析
	claims, err := utils.ParseAccessToken(token, svcCtx.Config.Auth.AccessSecret)
	assert.NoError(t, err)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.Equal(t, "testuser", claims.Nickname)
}

// ==================== GenerateRefreshToken (wrapper) 测试 ====================

func TestGenerateRefreshToken_Wrapper_Success(t *testing.T) {
	_, _, _, svcCtx := setupJwtTest(t)

	user := &model.Users{
		Id:           1,
		SnowflakeId:  12345,
		Email:        "test@example.com",
		Nickname:     "testuser",
		PasswordHash: "hashedpassword",
	}

	token, err := GenerateRefreshToken(svcCtx.Config, user)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// 验证 token 可以解析
	claims, err := utils.ParseRefreshToken(token, svcCtx.Config.RefreshSecret)
	assert.NoError(t, err)
	uid, _ := claims.GetUID()
	assert.Equal(t, int64(12345), uid)
}

// ==================== GetUserByAccessJwtCtx 测试 ====================

func TestGetUserByAccessJwtCtx_Success(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupJwtTest(t)

	claims := &utils.AccessToken{
		Nickname: "testuser",
		Email:    "test@example.com",
		JwtClaims: utils.JwtClaims{
			Uid:       json.Number("12345"),
			Version:   "1.0",
			TokenType: "access",
			Iat:       time.Now().Unix(),
			Exp:       time.Now().Add(time.Hour).Unix(),
		},
	}
	ctx := context.WithValue(context.Background(), "claims", claims)

	expectedUser := &model.Users{
		Id:           1,
		SnowflakeId:  12345,
		Email:        "test@example.com",
		Nickname:     "testuser",
		PasswordHash: "hashedpassword",
	}
	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(expectedUser, nil)

	user, err := GetUserByAccessJwtCtx(ctx, svcCtx)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, int64(12345), user.SnowflakeId)
	mockUsersModel.AssertExpectations(t)
}

func TestGetUserByAccessJwtCtx_ClaimsNotFound(t *testing.T) {
	_, _, _, svcCtx := setupJwtTest(t)

	ctx := context.Background()

	user, err := GetUserByAccessJwtCtx(ctx, svcCtx)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
}

// ==================== GetUserByRefreshJwtCtx 测试 ====================

func TestGetUserByRefreshJwtCtx_Success(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupJwtTest(t)

	claims := &utils.RefreshToken{
		JwtClaims: utils.JwtClaims{
			Uid:       json.Number("12345"),
			Version:   "1.0",
			TokenType: "refresh",
			Iat:       time.Now().Unix(),
			Exp:       time.Now().Add(time.Hour).Unix(),
		},
	}
	ctx := context.WithValue(context.Background(), "claims", claims)

	expectedUser := &model.Users{
		Id:           1,
		SnowflakeId:  12345,
		Email:        "test@example.com",
		Nickname:     "testuser",
		PasswordHash: "hashedpassword",
	}
	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(expectedUser, nil)

	user, err := GetUserByRefreshJwtCtx(ctx, svcCtx)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, int64(12345), user.SnowflakeId)
	mockUsersModel.AssertExpectations(t)
}

func TestGetUserByRefreshJwtCtx_ClaimsNotFound(t *testing.T) {
	_, _, _, svcCtx := setupJwtTest(t)

	ctx := context.Background()

	user, err := GetUserByRefreshJwtCtx(ctx, svcCtx)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
}

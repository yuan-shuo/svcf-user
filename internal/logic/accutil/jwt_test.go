package accutil

import (
	"context"
	"testing"
	"time"

	"user/internal/config"
	"user/internal/errs"
	"user/internal/mock"
	"user/internal/model"
	"user/internal/svc"
	"user/internal/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
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

func TestGetEmailByJwtCtx_Success(t *testing.T) {
	ctx := context.WithValue(context.Background(), "email", "test@example.com")

	email, err := GetEmailByJwtCtx(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "test@example.com", email)
}

func TestGetEmailByJwtCtx_EmailNotFound(t *testing.T) {
	ctx := context.Background()

	email, err := GetEmailByJwtCtx(ctx)

	assert.Error(t, err)
	assert.Empty(t, email)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
}

func TestGetEmailByJwtCtx_InvalidType(t *testing.T) {
	ctx := context.WithValue(context.Background(), "email", 12345)

	email, err := GetEmailByJwtCtx(ctx)

	assert.Error(t, err)
	assert.Empty(t, email)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
}

func TestGetUserByJwtCtx_Success(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupJwtTest(t)

	ctx := context.WithValue(context.Background(), "uid", int64(12345))

	expectedUser := &model.Users{
		Id:           1,
		SnowflakeId:  12345,
		Email:        "test@example.com",
		Nickname:     "testuser",
		PasswordHash: "hashedpassword",
	}
	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(expectedUser, nil)

	user, err := GetUserByJwtCtx(ctx, svcCtx)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, int64(12345), user.SnowflakeId)
	mockUsersModel.AssertExpectations(t)
}

func TestGetUserByJwtCtx_UidNotFound(t *testing.T) {
	_, _, _, svcCtx := setupJwtTest(t)

	ctx := context.Background()

	user, err := GetUserByJwtCtx(ctx, svcCtx)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
}

func TestGetUserByJwtCtx_InvalidUidType(t *testing.T) {
	_, _, _, svcCtx := setupJwtTest(t)

	ctx := context.WithValue(context.Background(), "uid", "invalid")

	user, err := GetUserByJwtCtx(ctx, svcCtx)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
}

func TestGetUserByJwtCtx_UserNotFound(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupJwtTest(t)

	ctx := context.WithValue(context.Background(), "uid", int64(12345))

	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(nil, sqlx.ErrNotFound)

	user, err := GetUserByJwtCtx(ctx, svcCtx)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeUserNotFound), "应该是用户不存在错误")
	mockUsersModel.AssertExpectations(t)
}

func TestGetUserByJwtCtx_DatabaseError(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupJwtTest(t)

	ctx := context.WithValue(context.Background(), "uid", int64(12345))

	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(nil, assert.AnError)

	user, err := GetUserByJwtCtx(ctx, svcCtx)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

func TestGetUserByUid_Success(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupJwtTest(t)

	ctx := context.Background()
	uid := int64(12345)

	expectedUser := &model.Users{
		Id:           1,
		SnowflakeId:  uid,
		Email:        "test@example.com",
		Nickname:     "testuser",
		PasswordHash: "hashedpassword",
	}
	mockUsersModel.On("FindOneBySnowflakeId", ctx, uid).Return(expectedUser, nil)

	user, err := GetUserByUid(ctx, svcCtx, uid)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, uid, user.SnowflakeId)
	mockUsersModel.AssertExpectations(t)
}

func TestGetUserByUid_NotFound(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupJwtTest(t)

	ctx := context.Background()
	uid := int64(12345)

	mockUsersModel.On("FindOneBySnowflakeId", ctx, uid).Return(nil, sqlx.ErrNotFound)

	user, err := GetUserByUid(ctx, svcCtx, uid)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeUserNotFound), "应该是用户不存在错误")
	mockUsersModel.AssertExpectations(t)
}

func TestGetUserByUid_DatabaseError(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupJwtTest(t)

	ctx := context.Background()
	uid := int64(12345)

	mockUsersModel.On("FindOneBySnowflakeId", ctx, uid).Return(nil, assert.AnError)

	user, err := GetUserByUid(ctx, svcCtx, uid)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

// ==================== GetUserByClaims 测试 ====================

func TestGetUserByClaims_Success(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupJwtTest(t)

	ctx := context.Background()
	uid := int64(12345)

	claims := jwt.MapClaims{
		"uid": float64(uid),
	}

	expectedUser := &model.Users{
		Id:           1,
		SnowflakeId:  uid,
		Email:        "test@example.com",
		Nickname:     "testuser",
		PasswordHash: "hashedpassword",
	}
	mockUsersModel.On("FindOneBySnowflakeId", ctx, uid).Return(expectedUser, nil)

	user, err := GetUserByClaims(ctx, svcCtx, claims)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, uid, user.SnowflakeId)
	mockUsersModel.AssertExpectations(t)
}

func TestGetUserByClaims_InvalidUid(t *testing.T) {
	_, _, _, svcCtx := setupJwtTest(t)

	ctx := context.Background()

	claims := jwt.MapClaims{
		"uid": "invalid",
	}

	user, err := GetUserByClaims(ctx, svcCtx, claims)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
}

func TestGetUserByClaims_UidNotFound(t *testing.T) {
	_, _, _, svcCtx := setupJwtTest(t)

	ctx := context.Background()

	claims := jwt.MapClaims{}

	user, err := GetUserByClaims(ctx, svcCtx, claims)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
}

func TestGetUserByClaims_UserNotFound(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupJwtTest(t)

	ctx := context.Background()
	uid := int64(12345)

	claims := jwt.MapClaims{
		"uid": float64(uid),
	}

	mockUsersModel.On("FindOneBySnowflakeId", ctx, uid).Return(nil, sqlx.ErrNotFound)

	user, err := GetUserByClaims(ctx, svcCtx, claims)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeUserNotFound), "应该是用户不存在错误")
	mockUsersModel.AssertExpectations(t)
}

// ==================== GetClaimsByJWT 测试 ====================

func TestGetClaimsByJWT_Success(t *testing.T) {
	secret := "test-secret"
	uid := int64(12345)

	// 生成有效的 token
	token, err := utils.GenerateAccessToken(secret, 3600, uid, "testuser", "test@example.com")
	assert.NoError(t, err)

	claims, err := GetClaimsByJWT(token, secret)

	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, float64(uid), claims["uid"])
}

func TestGetClaimsByJWT_InvalidToken(t *testing.T) {
	secret := "test-secret"

	claims, err := GetClaimsByJWT("invalid-token", secret)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidToken), "应该是无效token错误")
}

func TestGetClaimsByJWT_WrongSecret(t *testing.T) {
	correctSecret := "correct-secret"
	wrongSecret := "wrong-secret"
	uid := int64(12345)

	// 用正确的 secret 生成 token
	token, err := utils.GenerateAccessToken(correctSecret, 3600, uid, "testuser", "test@example.com")
	assert.NoError(t, err)

	// 用错误的 secret 解析
	claims, err := GetClaimsByJWT(token, wrongSecret)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidToken), "应该是无效token错误")
}

func TestGetClaimsByJWT_ExpiredToken(t *testing.T) {
	secret := "test-secret"

	// 生成已过期的 token
	now := jwt.TimeFunc()
	expiredClaims := jwt.MapClaims{
		"uid":  float64(12345),
		"exp":  now.Add(-3600 * time.Second).Unix(),
		"iat":  now.Add(-7200 * time.Second).Unix(),
		"type": "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	tokenString, err := token.SignedString([]byte(secret))
	assert.NoError(t, err)

	claims, err := GetClaimsByJWT(tokenString, secret)

	assert.Error(t, err)
	assert.Nil(t, claims)
}

// ==================== IsTokenTypeEqualToRefreshToken 测试 ====================

func TestIsTokenTypeEqualToRefreshToken_Success(t *testing.T) {
	claims := jwt.MapClaims{
		"type": "refresh",
	}

	err := IsTokenTypeEqualToRefreshToken(claims)

	assert.NoError(t, err)
}

func TestIsTokenTypeEqualToRefreshToken_NotRefreshToken(t *testing.T) {
	claims := jwt.MapClaims{
		"type": "access",
	}

	err := IsTokenTypeEqualToRefreshToken(claims)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidToken), "应该是无效token错误")
}

func TestIsTokenTypeEqualToRefreshToken_TypeNotFound(t *testing.T) {
	claims := jwt.MapClaims{
		"uid": float64(12345),
	}

	err := IsTokenTypeEqualToRefreshToken(claims)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidToken), "应该是无效token错误")
}

// ==================== GenerateAccessToken 测试 ====================

func TestGenerateAccessToken_Success(t *testing.T) {
	user := &model.Users{
		SnowflakeId: 12345,
		Nickname:    "testuser",
		Email:       "test@example.com",
	}

	cfg := config.Config{
		Auth: config.Auth{
			AccessSecret: "test-access-secret",
			AccessExpire: 3600,
		},
	}

	token, err := GenerateAccessToken(cfg, user)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// 验证 token 可以解析
	claims, err := utils.ParseToken(token, cfg.Auth.AccessSecret)
	assert.NoError(t, err)
	assert.Equal(t, float64(user.SnowflakeId), claims["uid"])
	assert.Equal(t, user.Nickname, claims["nickname"])
	assert.Equal(t, user.Email, claims["email"])
	assert.Equal(t, "access", claims["type"])
}

// ==================== GenerateRefreshToken 测试 ====================

func TestGenerateRefreshToken_Success(t *testing.T) {
	user := &model.Users{
		SnowflakeId: 12345,
		Email:       "test@example.com",
	}

	cfg := config.Config{
		RefreshSecret: "test-refresh-secret",
		RefreshExpire: 7200,
	}

	token, err := GenerateRefreshToken(cfg, user)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// 验证 token 可以解析
	claims, err := utils.ParseToken(token, cfg.RefreshSecret)
	assert.NoError(t, err)
	assert.Equal(t, float64(user.SnowflakeId), claims["uid"])
	assert.Equal(t, "refresh", claims["type"])
}

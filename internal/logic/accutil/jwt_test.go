package accutil

import (
	"context"
	"testing"

	"user/internal/config"
	"user/internal/errs"
	"user/internal/mock"
	"user/internal/model"
	"user/internal/svc"
	"user/internal/utils"

	"github.com/alicebob/miniredis/v2"
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

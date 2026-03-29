package accutil

import (
	"context"
	"errors"
	"testing"

	"user/internal/config"
	"user/internal/errs"
	"user/internal/mock"
	"user/internal/model"
	"user/internal/svc"
	"user/internal/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	mock2 "github.com/stretchr/testify/mock"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// setupPasswordTest 设置密码测试环境
func setupPasswordTest(t *testing.T) (*miniredis.Miniredis, *redis.Redis, *mock.UsersModel, *svc.ServiceContext) {
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

func TestHashPassword_Success(t *testing.T) {
	email := "test@example.com"
	password := "password123"

	hashed, err := HashPassword(email, password)

	assert.NoError(t, err)
	assert.NotEmpty(t, hashed)
	assert.NotEqual(t, password, hashed) // 哈希后的密码应该与原密码不同
}

func TestHashPassword_EmptyPassword(t *testing.T) {
	email := "test@example.com"
	password := ""

	hashed, err := HashPassword(email, password)

	assert.NoError(t, err)
	assert.NotEmpty(t, hashed) // 空密码也应该能生成哈希
}

func TestResetUserPassword_Success(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	email := "test@example.com"
	newPassword := "newhashedpassword"

	// 设置 mock 期望
	existingUser := &model.Users{
		Id:           1,
		SnowflakeId:  123456789,
		Email:        email,
		Nickname:     "testuser",
		PasswordHash: "oldhashedpassword",
	}
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(existingUser, nil)
	mockUsersModel.On("Update", ctx, mock2.MatchedBy(func(u *model.Users) bool {
		return u.PasswordHash == newPassword
	})).Return(nil)

	err := ResetUserPassword(ctx, svcCtx, email, newPassword)

	assert.NoError(t, err)
	mockUsersModel.AssertExpectations(t)
}

func TestResetUserPassword_UserNotFound(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	email := "test@example.com"
	newPassword := "newhashedpassword"

	// 设置 mock 期望 - 用户不存在
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, sqlx.ErrNotFound)

	err := ResetUserPassword(ctx, svcCtx, email, newPassword)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

func TestResetUserPassword_FindUserError(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	email := "test@example.com"
	newPassword := "newhashedpassword"

	// 设置 mock 期望 - 数据库查询错误
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, errors.New("database connection failed"))

	err := ResetUserPassword(ctx, svcCtx, email, newPassword)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

func TestResetUserPassword_UpdateFailed(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	email := "test@example.com"
	newPassword := "newhashedpassword"

	// 设置 mock 期望
	existingUser := &model.Users{
		Id:           1,
		SnowflakeId:  123456789,
		Email:        email,
		Nickname:     "testuser",
		PasswordHash: "oldhashedpassword",
	}
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(existingUser, nil)
	mockUsersModel.On("Update", ctx, mock2.AnythingOfType("*model.Users")).Return(errors.New("update failed"))

	err := ResetUserPassword(ctx, svcCtx, email, newPassword)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

func TestResetUserPassword_SameAsOldPassword(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	email := "test@example.com"
	oldPassword := "oldhashedpassword"

	// 设置 mock 期望 - 新密码与旧密码相同
	existingUser := &model.Users{
		Id:           1,
		SnowflakeId:  123456789,
		Email:        email,
		Nickname:     "testuser",
		PasswordHash: oldPassword,
	}
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(existingUser, nil)
	// 注意：由于密码相同，不会调用 Update

	err := ResetUserPassword(ctx, svcCtx, email, oldPassword)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodePasswordSameAsOld), "应该是新密码与旧密码相同错误")
	mockUsersModel.AssertExpectations(t)
}

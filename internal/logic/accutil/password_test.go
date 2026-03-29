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
	newPassword := "newpassword123"

	// 设置 mock 期望
	existingUser := &model.Users{
		Id:           1,
		SnowflakeId:  123456789,
		Email:        email,
		Nickname:     "testuser",
		PasswordHash: "oldhashedpassword",
	}

	// 设置 Update mock 期望
	mockUsersModel.On("Update", ctx, mock2.AnythingOfType("*model.Users")).Return(nil)

	// 直接使用 ResetUserPassword，传入 user 对象
	err := ResetUserPassword(ctx, svcCtx, existingUser, newPassword)

	assert.NoError(t, err)
	// 验证密码已被更新
	assert.NotEqual(t, "oldhashedpassword", existingUser.PasswordHash)
	mockUsersModel.AssertExpectations(t)
}

func TestResetUserPassword_SameAsOldPassword(t *testing.T) {
	_, _, _, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	email := "test@example.com"
	oldPassword := "oldpassword123"

	// 创建一个已有密码的用户
	hashedOldPassword, _ := HashPassword(email, oldPassword)
	existingUser := &model.Users{
		Id:           1,
		SnowflakeId:  123456789,
		Email:        email,
		Nickname:     "testuser",
		PasswordHash: hashedOldPassword,
	}

	// 尝试使用相同的密码重置
	err := ResetUserPassword(ctx, svcCtx, existingUser, oldPassword)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodePasswordSameAsOld), "应该是新密码与旧密码相同错误")
}

func TestResetUserPasswordByEmail_Success(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	email := "test@example.com"
	newPassword := "newpassword123"

	// 设置 mock 期望
	existingUser := &model.Users{
		Id:           1,
		SnowflakeId:  123456789,
		Email:        email,
		Nickname:     "testuser",
		PasswordHash: "oldhashedpassword",
	}
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(existingUser, nil)
	mockUsersModel.On("Update", ctx, mock2.AnythingOfType("*model.Users")).Return(nil)

	err := ResetUserPasswordByEmail(ctx, svcCtx, email, newPassword)

	assert.NoError(t, err)
	mockUsersModel.AssertExpectations(t)
}

func TestResetUserPasswordByEmail_UserNotFound(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	email := "test@example.com"
	newPassword := "newpassword123"

	// 设置 mock 期望 - 用户不存在
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, sqlx.ErrNotFound)

	err := ResetUserPasswordByEmail(ctx, svcCtx, email, newPassword)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeUserNotFound), "应该是用户不存在错误")
	mockUsersModel.AssertExpectations(t)
}

func TestGetUserByEmail_Success(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	email := "test@example.com"

	expectedUser := &model.Users{
		Id:           1,
		SnowflakeId:  123456789,
		Email:        email,
		Nickname:     "testuser",
		PasswordHash: "hashedpassword",
	}
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(expectedUser, nil)

	user, err := GetUserByEmail(ctx, svcCtx, email)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, email, user.Email)
	mockUsersModel.AssertExpectations(t)
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	email := "nonexistent@example.com"

	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, sqlx.ErrNotFound)

	user, err := GetUserByEmail(ctx, svcCtx, email)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeUserNotFound), "应该是用户不存在错误")
	mockUsersModel.AssertExpectations(t)
}

func TestGetUserByEmail_DatabaseError(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	email := "test@example.com"

	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, errors.New("database connection failed"))

	user, err := GetUserByEmail(ctx, svcCtx, email)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

func TestVerifyPasswordWithVagueMismatchErrHint_Success(t *testing.T) {
	email := "test@example.com"
	password := "password123"
	hashedPassword, _ := HashPassword(email, password)

	err := VerifyPasswordWithVagueMismatchErrHint(hashedPassword, password, email)

	assert.NoError(t, err)
}

func TestVerifyPasswordWithVagueMismatchErrHint_InvalidPassword(t *testing.T) {
	email := "test@example.com"
	password := "password123"
	wrongPassword := "wrongpassword"
	hashedPassword, _ := HashPassword(email, password)

	err := VerifyPasswordWithVagueMismatchErrHint(hashedPassword, wrongPassword, email)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeUserNotExistOrPasswordIncorrect), "应该是用户不存在或密码错误")
}

func TestVerifyPasswordWithOldPasswordMismatchErrHint_InvalidPassword(t *testing.T) {
	email := "test@example.com"
	password := "password123"
	wrongPassword := "wrongpassword"
	hashedPassword, _ := HashPassword(email, password)

	err := VerifyPasswordWithOldPasswordMismatchErrHint(hashedPassword, wrongPassword, email)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeOldPasswordIncorrect), "应该是旧密码错误")
}

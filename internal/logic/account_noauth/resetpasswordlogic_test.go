package account_noauth

import (
	"context"
	"errors"
	"testing"
	"time"

	"user/internal/config"
	"user/internal/errs"
	"user/internal/mock"
	"user/internal/model"
	"user/internal/svc"
	"user/internal/types"
	"user/internal/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	mock2 "github.com/stretchr/testify/mock"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// setupResetPasswordTest 设置重置密码测试环境
func setupResetPasswordTest(t *testing.T) (*miniredis.Miniredis, *redis.Redis, *mock.UsersModel, *svc.ServiceContext) {
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
					ResetPassword: "reset_password",
				},
				Redis: config.VerifyCodeRedisConfig{
					KeyPrefix: "account",
				},
			},
		},
		Redis:      rds,
		UsersModel: mockUsersModel,
		Metrics:    mock.GetTestMetrics(),
	}

	// 初始化雪花算法
	err := utils.InitSonyflake(1, "2024-01-01")
	assert.NoError(t, err)

	return s, rds, mockUsersModel, svcCtx
}

func TestResetPasswordLogic_ResetPassword_Success(t *testing.T) {
	s, _, mockUsersModel, svcCtx := setupResetPasswordTest(t)
	defer s.Close()

	ctx := context.Background()
	logic := NewResetPasswordLogic(ctx, svcCtx)

	// 准备测试数据
	email := "test@example.com"
	code := "123456"
	newPassword := "newpassword123"

	// 在 redis 中设置验证码
	key := "account:reset_password:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

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

	// 执行测试
	req := &types.ResetPasswordReq{
		Email:    email,
		Password: newPassword,
		Code:     code,
	}

	resp, err := logic.ResetPassword(req)

	// 验证结果
	assert.NoError(t, err)
	_ = resp
	mockUsersModel.AssertExpectations(t)

	// 验证验证码被标记为已使用
	used := s.HGet(key, "used")
	assert.Equal(t, "1", used)
}

func TestResetPasswordLogic_ResetPassword_InvalidCode(t *testing.T) {
	s, _, _, svcCtx := setupResetPasswordTest(t)
	defer s.Close()

	ctx := context.Background()
	logic := NewResetPasswordLogic(ctx, svcCtx)

	// 准备测试数据
	email := "test@example.com"
	code := "wrongcode"

	// 在 redis 中设置正确的验证码
	key := "account:reset_password:verify:" + email
	s.HSet(key, "code", "123456")
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	// 执行测试
	req := &types.ResetPasswordReq{
		Email:    email,
		Password: "newpassword123",
		Code:     code,
	}

	resp, err := logic.ResetPassword(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidCode), "应该是验证码错误")
}

func TestResetPasswordLogic_ResetPassword_CodeExpired(t *testing.T) {
	s, _, _, svcCtx := setupResetPasswordTest(t)
	defer s.Close()

	ctx := context.Background()
	logic := NewResetPasswordLogic(ctx, svcCtx)

	// 准备测试数据
	email := "test@example.com"

	// 执行测试（redis 中没有验证码）
	req := &types.ResetPasswordReq{
		Email:    email,
		Password: "newpassword123",
		Code:     "123456",
	}

	resp, err := logic.ResetPassword(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidCode), "应该是验证码无效错误")
}

func TestResetPasswordLogic_ResetPassword_CodeAlreadyUsed(t *testing.T) {
	s, _, _, svcCtx := setupResetPasswordTest(t)
	defer s.Close()

	ctx := context.Background()
	logic := NewResetPasswordLogic(ctx, svcCtx)

	// 准备测试数据
	email := "test@example.com"
	code := "123456"

	// 在 redis 中设置已使用的验证码
	key := "account:reset_password:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "1")
	s.SetTTL(key, 5*time.Minute)

	// 执行测试
	req := &types.ResetPasswordReq{
		Email:    email,
		Password: "newpassword123",
		Code:     code,
	}

	resp, err := logic.ResetPassword(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeCodeAlreadyUsed), "应该是验证码已使用错误")
}

func TestResetPasswordLogic_ResetPassword_UserNotFound(t *testing.T) {
	s, _, mockUsersModel, svcCtx := setupResetPasswordTest(t)
	defer s.Close()

	ctx := context.Background()
	logic := NewResetPasswordLogic(ctx, svcCtx)

	// 准备测试数据
	email := "test@example.com"
	code := "123456"

	// 在 redis 中设置验证码
	key := "account:reset_password:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	// 设置 mock 期望 - 用户不存在
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, sqlx.ErrNotFound)

	// 执行测试
	req := &types.ResetPasswordReq{
		Email:    email,
		Password: "newpassword123",
		Code:     code,
	}

	resp, err := logic.ResetPassword(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeUserNotFound), "应该是用户不存在错误")
	mockUsersModel.AssertExpectations(t)
}

func TestResetPasswordLogic_ResetPassword_FindUserError(t *testing.T) {
	s, _, mockUsersModel, svcCtx := setupResetPasswordTest(t)
	defer s.Close()

	ctx := context.Background()
	logic := NewResetPasswordLogic(ctx, svcCtx)

	// 准备测试数据
	email := "test@example.com"
	code := "123456"

	// 在 redis 中设置验证码
	key := "account:reset_password:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	// 设置 mock 期望 - 数据库查询错误
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, errors.New("database connection failed"))

	// 执行测试
	req := &types.ResetPasswordReq{
		Email:    email,
		Password: "newpassword123",
		Code:     code,
	}

	resp, err := logic.ResetPassword(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

func TestResetPasswordLogic_ResetPassword_UpdateFailed(t *testing.T) {
	s, _, mockUsersModel, svcCtx := setupResetPasswordTest(t)
	defer s.Close()

	ctx := context.Background()
	logic := NewResetPasswordLogic(ctx, svcCtx)

	// 准备测试数据
	email := "test@example.com"
	code := "123456"
	newPassword := "newpassword123"

	// 在 redis 中设置验证码
	key := "account:reset_password:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

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

	// 执行测试
	req := &types.ResetPasswordReq{
		Email:    email,
		Password: newPassword,
		Code:     code,
	}

	resp, err := logic.ResetPassword(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

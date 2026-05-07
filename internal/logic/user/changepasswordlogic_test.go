package user

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
	"user/internal/types"
	"user/internal/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	mock2 "github.com/stretchr/testify/mock"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// setupChangePasswordTest 设置修改密码测试环境
func setupChangePasswordTest(t *testing.T) (*miniredis.Miniredis, *redis.Redis, *mock.UsersModel, *svc.ServiceContext) {
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
					ChangePassword: "change_password",
				},
				// Redis: config.VerifyCodeRedisConfig{
				// 	KeyPrefix: "account",
				// },
			},
		},
		Redis:      rds,
		UsersModel: mockUsersModel,
		Metrics:    mock.GetTestMetrics(),
	}

	// 初始化雪花算�?
	err := utils.InitSonyflake(1, "2024-01-01")
	assert.NoError(t, err)

	return s, rds, mockUsersModel, svcCtx
}

// createTestContextWithAccessToken 创建包含 AccessToken claims �?context
// 模拟 go-zero JWT 中间件将 claims 字段存入 context 的行�?
func createTestContextWithAccessToken(uid int64, email string) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "uid", json.Number("12345"))
	ctx = context.WithValue(ctx, "version", "1.0")
	ctx = context.WithValue(ctx, "type", "access")
	ctx = context.WithValue(ctx, "nickname", "testuser")
	ctx = context.WithValue(ctx, "email", email)
	return ctx
}

func TestChangePasswordLogic_ChangePassword_Success(t *testing.T) {
	s, _, mockUsersModel, svcCtx := setupChangePasswordTest(t)
	defer s.Close()

	// 准备 JWT 上下�?
	ctx := createTestContextWithAccessToken(12345, "test@example.com")

	logic := NewChangePasswordLogic(ctx, svcCtx)

	// 准备测试数据
	email := "test@example.com"
	code := "123456"
	oldPassword := "OldPassword123!"
	newPassword := "NewPassword123!"

	// 先生成旧密码的哈�?
	hashedOldPassword, _ := utils.HashPassword(oldPassword)

	// �?redis 中设置验证码
	key := "user:change_password:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	// 设置 mock 期望
	expectedUser := &model.Users{
		Id:           1,
		SnowflakeId:  12345,
		Email:        email,
		Nickname:     "testuser",
		PasswordHash: hashedOldPassword,
	}
	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(expectedUser, nil)
	mockUsersModel.On("Update", ctx, mock2.AnythingOfType("*model.Users")).Return(nil)

	// 执行测试
	req := &types.ChangePasswordReq{
		OldPassword: oldPassword,
		NewPassword: newPassword,
		Code:        code,
	}

	resp, err := logic.ChangePassword(req)

	// 验证结果
	assert.NoError(t, err)
	_ = resp
	mockUsersModel.AssertExpectations(t)

	// 验证验证码被标记为已使用
	used := s.HGet(key, "used")
	assert.Equal(t, "1", used)
}

func TestChangePasswordLogic_ChangePassword_InvalidCode(t *testing.T) {
	s, _, mockUsersModel, svcCtx := setupChangePasswordTest(t)
	defer s.Close()

	ctx := createTestContextWithAccessToken(12345, "test@example.com")

	logic := NewChangePasswordLogic(ctx, svcCtx)

	email := "test@example.com"
	code := "123456"
	wrongCode := "wrongcode"
	oldPassword := "OldPassword123!"
	newPassword := "NewPassword123!"

	// �?redis 中设置验证码
	key := "user:change_password:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	// 执行测试 - 使用错误的验证码
	req := &types.ChangePasswordReq{
		OldPassword: oldPassword,
		NewPassword: newPassword,
		Code:        wrongCode,
	}

	resp, err := logic.ChangePassword(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidCode), "应该是验证码错误")
	mockUsersModel.AssertExpectations(t)
}

func TestChangePasswordLogic_ChangePassword_CodeAlreadyUsed(t *testing.T) {
	s, _, mockUsersModel, svcCtx := setupChangePasswordTest(t)
	defer s.Close()

	ctx := createTestContextWithAccessToken(12345, "test@example.com")

	logic := NewChangePasswordLogic(ctx, svcCtx)

	email := "test@example.com"
	code := "123456"
	oldPassword := "OldPassword123!"
	newPassword := "NewPassword123!"

	// �?redis 中设置验证码 - 已使�?
	key := "user:change_password:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "1")
	s.SetTTL(key, 5*time.Minute)

	// 执行测试
	req := &types.ChangePasswordReq{
		OldPassword: oldPassword,
		NewPassword: newPassword,
		Code:        code,
	}

	resp, err := logic.ChangePassword(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeCodeAlreadyUsed), "应该是验证码已使用错误")
	mockUsersModel.AssertExpectations(t)
}

func TestChangePasswordLogic_ChangePassword_CodeNotFound(t *testing.T) {
	s, _, mockUsersModel, svcCtx := setupChangePasswordTest(t)
	defer s.Close()

	ctx := createTestContextWithAccessToken(12345, "test@example.com")

	logic := NewChangePasswordLogic(ctx, svcCtx)

	code := "123456"
	oldPassword := "OldPassword123!"
	newPassword := "NewPassword123!"

	// 不设置验证码�?redis

	// 执行测试
	req := &types.ChangePasswordReq{
		OldPassword: oldPassword,
		NewPassword: newPassword,
		Code:        code,
	}

	resp, err := logic.ChangePassword(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidCode), "应该是验证码无效错误")
	mockUsersModel.AssertExpectations(t)
}

func TestChangePasswordLogic_ChangePassword_UserNotFound(t *testing.T) {
	s, _, mockUsersModel, svcCtx := setupChangePasswordTest(t)
	defer s.Close()

	ctx := createTestContextWithAccessToken(12345, "test@example.com")

	logic := NewChangePasswordLogic(ctx, svcCtx)

	email := "test@example.com"
	code := "123456"
	oldPassword := "OldPassword123!"
	newPassword := "NewPassword123!"

	// �?redis 中设置验证码
	key := "user:change_password:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	// 设置 mock 期望 - 用户不存�?
	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(nil, sqlx.ErrNotFound)

	// 执行测试
	req := &types.ChangePasswordReq{
		OldPassword: oldPassword,
		NewPassword: newPassword,
		Code:        code,
	}

	resp, err := logic.ChangePassword(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeUserNotFound), "应该是用户不存在错误")
	mockUsersModel.AssertExpectations(t)
}

func TestChangePasswordLogic_ChangePassword_OldPasswordIncorrect(t *testing.T) {
	s, _, mockUsersModel, svcCtx := setupChangePasswordTest(t)
	defer s.Close()

	ctx := createTestContextWithAccessToken(12345, "test@example.com")

	logic := NewChangePasswordLogic(ctx, svcCtx)

	email := "test@example.com"
	code := "123456"
	oldPassword := "OldPassword123!"
	wrongOldPassword := "WrongPassword123!"
	newPassword := "NewPassword123!"

	// 先生成旧密码的哈�?
	hashedOldPassword, _ := utils.HashPassword(oldPassword)

	// �?redis 中设置验证码
	key := "user:change_password:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	// 设置 mock 期望
	expectedUser := &model.Users{
		Id:           1,
		SnowflakeId:  12345,
		Email:        email,
		Nickname:     "testuser",
		PasswordHash: hashedOldPassword,
	}
	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(expectedUser, nil)

	// 执行测试 - 使用错误的旧密码
	req := &types.ChangePasswordReq{
		OldPassword: wrongOldPassword,
		NewPassword: newPassword,
		Code:        code,
	}

	resp, err := logic.ChangePassword(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeOldPasswordIncorrect), "应该是旧密码错误")
	mockUsersModel.AssertExpectations(t)
}

func TestChangePasswordLogic_ChangePassword_SameAsOldPassword(t *testing.T) {
	s, _, mockUsersModel, svcCtx := setupChangePasswordTest(t)
	defer s.Close()

	ctx := createTestContextWithAccessToken(12345, "test@example.com")

	logic := NewChangePasswordLogic(ctx, svcCtx)

	email := "test@example.com"
	code := "123456"
	oldPassword := "OldPassword123!"

	// 先生成旧密码的哈�?
	hashedOldPassword, _ := utils.HashPassword(oldPassword)

	// �?redis 中设置验证码
	key := "user:change_password:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	// 设置 mock 期望
	expectedUser := &model.Users{
		Id:           1,
		SnowflakeId:  12345,
		Email:        email,
		Nickname:     "testuser",
		PasswordHash: hashedOldPassword,
	}
	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(expectedUser, nil)

	// 执行测试 - 新密码与旧密码相�?
	req := &types.ChangePasswordReq{
		OldPassword: oldPassword,
		NewPassword: oldPassword,
		Code:        code,
	}

	resp, err := logic.ChangePassword(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodePasswordSameAsOld), "应该是新密码与旧密码相同错误")
	mockUsersModel.AssertExpectations(t)
}

// TestChangePasswordLogic_ChangePassword_WeakPassword 测试密码强度不足
func TestChangePasswordLogic_ChangePassword_WeakPassword(t *testing.T) {
	s, _, mockUsersModel, svcCtx := setupChangePasswordTest(t)
	defer s.Close()

	ctx := createTestContextWithAccessToken(12345, "test@example.com")

	logic := NewChangePasswordLogic(ctx, svcCtx)

	email := "test@example.com"
	code := "123456"
	oldPassword := "OldPassword123!"
	weakNewPassword := "weak"

	// 先生成旧密码的哈�?
	hashedOldPassword, _ := utils.HashPassword(oldPassword)

	// �?redis 中设置验证码
	key := "user:change_password:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	// 设置 mock 期望
	expectedUser := &model.Users{
		Id:           1,
		SnowflakeId:  12345,
		Email:        email,
		Nickname:     "testuser",
		PasswordHash: hashedOldPassword,
	}
	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(expectedUser, nil)

	// 执行测试 - 使用弱密�?
	req := &types.ChangePasswordReq{
		OldPassword: oldPassword,
		NewPassword: weakNewPassword,
		Code:        code,
	}

	resp, err := logic.ChangePassword(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeWeakPassword), "应该是密码强度不足错误")
	mockUsersModel.AssertExpectations(t)
}

func TestChangePasswordLogic_ChangePassword_UpdateFailed(t *testing.T) {
	s, _, mockUsersModel, svcCtx := setupChangePasswordTest(t)
	defer s.Close()

	ctx := createTestContextWithAccessToken(12345, "test@example.com")

	logic := NewChangePasswordLogic(ctx, svcCtx)

	email := "test@example.com"
	code := "123456"
	oldPassword := "OldPassword123!"
	newPassword := "NewPassword123!"

	// 先生成旧密码的哈�?
	hashedOldPassword, _ := utils.HashPassword(oldPassword)

	// �?redis 中设置验证码
	key := "user:change_password:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	// 设置 mock 期望
	expectedUser := &model.Users{
		Id:           1,
		SnowflakeId:  12345,
		Email:        email,
		Nickname:     "testuser",
		PasswordHash: hashedOldPassword,
	}
	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(expectedUser, nil)
	mockUsersModel.On("Update", ctx, mock2.AnythingOfType("*model.Users")).Return(assert.AnError)

	// 执行测试
	req := &types.ChangePasswordReq{
		OldPassword: oldPassword,
		NewPassword: newPassword,
		Code:        code,
	}

	resp, err := logic.ChangePassword(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

func TestChangePasswordLogic_ChangePassword_EmailNotInContext(t *testing.T) {
	s, _, _, svcCtx := setupChangePasswordTest(t)
	defer s.Close()

	// 不设�?claims 到上下文
	ctx := context.Background()

	logic := NewChangePasswordLogic(ctx, svcCtx)

	code := "123456"
	oldPassword := "OldPassword123!"
	newPassword := "NewPassword123!"

	// 执行测试
	req := &types.ChangePasswordReq{
		OldPassword: oldPassword,
		NewPassword: newPassword,
		Code:        code,
	}

	resp, err := logic.ChangePassword(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
}

package userutils

import (
	"context"
	"testing"

	"user/internal/config"
	"user/internal/errs"
	"user/internal/mock"
	"user/internal/model"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	mock2 "github.com/stretchr/testify/mock"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// setupVerifyCodeTest 设置验证码测试环境
func setupVerifyCodeTest(t *testing.T) (*miniredis.Miniredis, *redis.Redis, *mock.UsersModel, *mock.KqPusherClient, config.VerifyCodeConfig) {
	// 创建 miniredis
	s := miniredis.RunT(t)

	// 创建 redis 客户端
	rds := redis.New(s.Addr())

	// 创建 mock users model
	mockUsersModel := new(mock.UsersModel)

	// 创建 mock mq client
	mockMqClient := new(mock.KqPusherClient)

	// 创建配置
	cfg := config.VerifyCodeConfig{
		Type: config.VerifyCodeType{
			Register:         "register",
			ResetPassword:    "reset_password",
			ChangePassword:   "change_password",
			RemindRegistered: "remind_registered",
		},
		Time: config.VerifyCodeTime{
			RetryAfter: 60,
			ExpireIn:   300,
		},
	}

	return s, rds, mockUsersModel, mockMqClient, cfg
}

// ==================== ValidateVerifyCodeRequest 测试 ====================

func TestValidateVerifyCodeRequest_Success(t *testing.T) {
	cfg := config.VerifyCodeType{
		Register:       "register",
		ResetPassword:  "reset_password",
		ChangePassword: "change_password",
	}

	err := ValidateVerifyCodeRequest("test@example.com", "register", cfg)

	assert.NoError(t, err)
}

func TestValidateVerifyCodeRequest_InvalidType(t *testing.T) {
	cfg := config.VerifyCodeType{
		Register:       "register",
		ResetPassword:  "reset_password",
		ChangePassword: "change_password",
	}

	err := ValidateVerifyCodeRequest("test@example.com", "invalid_type", cfg)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidParam))
}

func TestValidateVerifyCodeRequest_InvalidEmail(t *testing.T) {
	cfg := config.VerifyCodeType{
		Register:       "register",
		ResetPassword:  "reset_password",
		ChangePassword: "change_password",
	}

	err := ValidateVerifyCodeRequest("invalid-email", "register", cfg)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidParam))
}

// ==================== IsValidCodeType 测试 ====================

func TestIsValidCodeType_Valid(t *testing.T) {
	cfg := config.VerifyCodeType{
		Register:       "register",
		ResetPassword:  "reset_password",
		ChangePassword: "change_password",
	}

	assert.True(t, IsValidCodeType("register", cfg))
	assert.True(t, IsValidCodeType("reset_password", cfg))
	assert.True(t, IsValidCodeType("change_password", cfg))
}

func TestIsValidCodeType_Invalid(t *testing.T) {
	cfg := config.VerifyCodeType{
		Register:       "register",
		ResetPassword:  "reset_password",
		ChangePassword: "change_password",
	}

	assert.False(t, IsValidCodeType("invalid", cfg))
	assert.False(t, IsValidCodeType("", cfg))
}

// ==================== CheckRateLimit 测试 ====================

func TestCheckRateLimit_Success(t *testing.T) {
	s, rds, _, _, _ := setupVerifyCodeTest(t)
	defer s.Close()

	ctx := context.Background()

	err := CheckRateLimit(ctx, rds, 60, "test@example.com", "register")

	assert.NoError(t, err)
}

func TestCheckRateLimit_RateLimited(t *testing.T) {
	s, rds, _, _, _ := setupVerifyCodeTest(t)
	defer s.Close()

	ctx := context.Background()

	// 第一次设置限流
	err := CheckRateLimit(ctx, rds, 60, "test@example.com", "register")
	assert.NoError(t, err)

	// 第二次应该被限流
	err = CheckRateLimit(ctx, rds, 60, "test@example.com", "register")
	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidParam))
}

// ==================== CheckRegisterLogic 测试 ====================

func TestCheckRegisterLogic_EmailNotRegistered(t *testing.T) {
	_, _, mockUsersModel, mockMqClient, cfg := setupVerifyCodeTest(t)

	ctx := context.Background()
	email := "test@example.com"

	// 邮箱未注册
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, sqlx.ErrNotFound)

	shouldContinue, err := CheckRegisterLogic(ctx, mockUsersModel, mockMqClient, cfg.Type.RemindRegistered, email)

	assert.NoError(t, err)
	assert.True(t, shouldContinue)
	mockUsersModel.AssertExpectations(t)
}

func TestCheckRegisterLogic_EmailAlreadyRegistered(t *testing.T) {
	_, _, mockUsersModel, mockMqClient, cfg := setupVerifyCodeTest(t)

	ctx := context.Background()
	email := "test@example.com"

	existingUser := &model.Users{
		Id:       1,
		Email:    email,
		Nickname: "testuser",
	}

	// 邮箱已注册
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(existingUser, nil)
	mockMqClient.On("Push", ctx, mock2.Anything).Return(nil)

	shouldContinue, err := CheckRegisterLogic(ctx, mockUsersModel, mockMqClient, cfg.Type.RemindRegistered, email)

	assert.NoError(t, err)
	assert.False(t, shouldContinue)
	mockUsersModel.AssertExpectations(t)
	mockMqClient.AssertExpectations(t)
}

func TestCheckRegisterLogic_DBError(t *testing.T) {
	_, _, mockUsersModel, mockMqClient, cfg := setupVerifyCodeTest(t)

	ctx := context.Background()
	email := "test@example.com"

	// 数据库错误
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, assert.AnError)

	shouldContinue, err := CheckRegisterLogic(ctx, mockUsersModel, mockMqClient, cfg.Type.RemindRegistered, email)

	assert.Error(t, err)
	assert.False(t, shouldContinue)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError))
	mockUsersModel.AssertExpectations(t)
}

// ==================== CheckResetPasswordLogic 测试 ====================

func TestCheckResetPasswordLogic_EmailRegistered(t *testing.T) {
	_, _, mockUsersModel, _, _ := setupVerifyCodeTest(t)

	ctx := context.Background()
	email := "test@example.com"

	existingUser := &model.Users{
		Id:       1,
		Email:    email,
		Nickname: "testuser",
	}

	mockUsersModel.On("FindOneByEmail", ctx, email).Return(existingUser, nil)

	shouldContinue, err := CheckResetPasswordLogic(ctx, mockUsersModel, email)

	assert.NoError(t, err)
	assert.True(t, shouldContinue)
	mockUsersModel.AssertExpectations(t)
}

func TestCheckResetPasswordLogic_EmailNotRegistered(t *testing.T) {
	_, _, mockUsersModel, _, _ := setupVerifyCodeTest(t)

	ctx := context.Background()
	email := "test@example.com"

	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, sqlx.ErrNotFound)

	shouldContinue, err := CheckResetPasswordLogic(ctx, mockUsersModel, email)

	assert.Error(t, err)
	assert.False(t, shouldContinue)
	assert.True(t, mock.IsCodeError(err, errs.CodeEmailNotRegistered))
	mockUsersModel.AssertExpectations(t)
}

func TestCheckResetPasswordLogic_DBError(t *testing.T) {
	_, _, mockUsersModel, _, _ := setupVerifyCodeTest(t)

	ctx := context.Background()
	email := "test@example.com"

	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, assert.AnError)

	shouldContinue, err := CheckResetPasswordLogic(ctx, mockUsersModel, email)

	assert.Error(t, err)
	assert.False(t, shouldContinue)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError))
	mockUsersModel.AssertExpectations(t)
}

// ==================== GenerateAndSaveVerifyCode 测试 ====================

func TestGenerateAndSaveVerifyCode_Success(t *testing.T) {
	s, rds, _, _, _ := setupVerifyCodeTest(t)
	defer s.Close()

	ctx := context.Background()

	code := GenerateAndSaveVerifyCode(ctx, rds, 300, "test@example.com", "register")

	assert.NotEmpty(t, code)
	assert.Equal(t, 6, len(code))
}

// ==================== SendVerifyCodeToMQ 测试 ====================

func TestSendVerifyCodeToMQ_Success(t *testing.T) {
	_, _, _, mockMqClient, _ := setupVerifyCodeTest(t)

	ctx := context.Background()

	mockMqClient.On("Push", ctx, mock2.Anything).Return(nil)

	err := SendVerifyCodeToMQ(ctx, mockMqClient, "test@example.com", "123456", "register")

	assert.NoError(t, err)
	mockMqClient.AssertExpectations(t)
}

func TestSendVerifyCodeToMQ_Failed(t *testing.T) {
	_, _, _, mockMqClient, _ := setupVerifyCodeTest(t)

	ctx := context.Background()

	mockMqClient.On("Push", ctx, mock2.Anything).Return(assert.AnError)

	err := SendVerifyCodeToMQ(ctx, mockMqClient, "test@example.com", "123456", "register")

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError))
	mockMqClient.AssertExpectations(t)
}

// ==================== CleanupRateLimit 测试 ====================

func TestCleanupRateLimit_Success(t *testing.T) {
	s, rds, _, _, _ := setupVerifyCodeTest(t)
	defer s.Close()

	ctx := context.Background()

	// 先设置限流
	rds.SetnxExCtx(ctx, "user:register:limit:test@example.com", "1", 60)

	// 清理限流
	CleanupRateLimit(ctx, rds, "test@example.com", "register")

	// 验证已清理
	exists, _ := rds.ExistsCtx(ctx, "user:register:limit:test@example.com")
	assert.False(t, exists)
}

// ==================== CleanupVerifyCode 测试 ====================

func TestCleanupVerifyCode_Success(t *testing.T) {
	s, rds, _, _, _ := setupVerifyCodeTest(t)
	defer s.Close()

	ctx := context.Background()

	// 先设置验证码
	rds.HsetCtx(ctx, "user:register:verify:test@example.com", "code", "123456")

	// 清理验证码
	CleanupVerifyCode(ctx, rds, "test@example.com", "register")

	// 验证已清理
	exists, _ := rds.ExistsCtx(ctx, "user:register:verify:test@example.com")
	assert.False(t, exists)
}

// ==================== CleanupVerifyCodeAll 测试 ====================

func TestCleanupVerifyCodeAll_Success(t *testing.T) {
	s, rds, _, _, _ := setupVerifyCodeTest(t)
	defer s.Close()

	ctx := context.Background()

	// 先设置数据
	rds.SetnxExCtx(ctx, "user:register:limit:test@example.com", "1", 60)
	rds.HsetCtx(ctx, "user:register:verify:test@example.com", "code", "123456")

	// 清理所有
	CleanupVerifyCodeAll(ctx, rds, "test@example.com", "register")

	// 验证已清理
	exists1, _ := rds.ExistsCtx(ctx, "user:register:limit:test@example.com")
	exists2, _ := rds.ExistsCtx(ctx, "user:register:verify:test@example.com")
	assert.False(t, exists1)
	assert.False(t, exists2)
}

// ==================== CheckBusinessLogic 测试 ====================

func TestCheckBusinessLogic_Register(t *testing.T) {
	_, _, mockUsersModel, mockMqClient, cfg := setupVerifyCodeTest(t)

	ctx := context.Background()
	email := "test@example.com"

	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, sqlx.ErrNotFound)

	shouldContinue, err := CheckBusinessLogic(ctx, mockUsersModel, mockMqClient, cfg, email, cfg.Type.Register)

	assert.NoError(t, err)
	assert.True(t, shouldContinue)
	mockUsersModel.AssertExpectations(t)
}

func TestCheckBusinessLogic_ResetPassword(t *testing.T) {
	_, _, mockUsersModel, mockMqClient, cfg := setupVerifyCodeTest(t)

	ctx := context.Background()
	email := "test@example.com"

	existingUser := &model.Users{
		Id:       1,
		Email:    email,
		Nickname: "testuser",
	}

	mockUsersModel.On("FindOneByEmail", ctx, email).Return(existingUser, nil)

	shouldContinue, err := CheckBusinessLogic(ctx, mockUsersModel, mockMqClient, cfg, email, cfg.Type.ResetPassword)

	assert.NoError(t, err)
	assert.True(t, shouldContinue)
	mockUsersModel.AssertExpectations(t)
}

func TestCheckBusinessLogic_ChangePassword(t *testing.T) {
	_, _, mockUsersModel, mockMqClient, cfg := setupVerifyCodeTest(t)

	ctx := context.Background()
	email := "test@example.com"

	existingUser := &model.Users{
		Id:       1,
		Email:    email,
		Nickname: "testuser",
	}

	mockUsersModel.On("FindOneByEmail", ctx, email).Return(existingUser, nil)

	shouldContinue, err := CheckBusinessLogic(ctx, mockUsersModel, mockMqClient, cfg, email, cfg.Type.ChangePassword)

	assert.NoError(t, err)
	assert.True(t, shouldContinue)
	mockUsersModel.AssertExpectations(t)
}

func TestCheckBusinessLogic_InvalidType(t *testing.T) {
	_, _, mockUsersModel, mockMqClient, cfg := setupVerifyCodeTest(t)

	ctx := context.Background()
	email := "test@example.com"

	shouldContinue, err := CheckBusinessLogic(ctx, mockUsersModel, mockMqClient, cfg, email, "invalid_type")

	assert.Error(t, err)
	assert.False(t, shouldContinue)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidParam))
}

// ==================== BuildVerifyCodeResponse 测试 ====================

func TestBuildVerifyCodeResponse(t *testing.T) {
	resp := BuildVerifyCodeResponse(60)

	assert.NotNil(t, resp)
	assert.Equal(t, 60, resp.RetryAfter)
}

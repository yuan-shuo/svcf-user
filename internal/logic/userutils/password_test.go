package userutils

import (
	"context"
	"encoding/json"
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

	// 创建 redis 客户�?
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
				// Redis: config.VerifyCodeRedisConfig{
				// 	KeyPrefix: "account",
				// },
			},
		},
		Redis:      rds,
		UsersModel: mockUsersModel,
	}

	// 初始化雪花算�?
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
	assert.NotEmpty(t, hashed) // 空密码也应该能生成哈�?
}

func TestResetUserPassword_Success(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	email := "test@example.com"
	newPassword := "NewPassword123!"

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

	// 直接使用 ResetUserPassword，传�?user 对象
	err := ResetUserPassword(ctx, svcCtx.UsersModel, existingUser, newPassword)

	assert.NoError(t, err)
	// 验证密码已被更新
	assert.NotEqual(t, "oldhashedpassword", existingUser.PasswordHash)
	mockUsersModel.AssertExpectations(t)
}

func TestResetUserPassword_SameAsOldPassword(t *testing.T) {
	_, _, _, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	email := "test@example.com"
	oldPassword := "OldPassword123!"

	// 创建一个已有密码的用户
	hashedOldPassword, _ := HashPassword(email, oldPassword)
	existingUser := &model.Users{
		Id:           1,
		SnowflakeId:  123456789,
		Email:        email,
		Nickname:     "testuser",
		PasswordHash: hashedOldPassword,
	}

	// 尝试使用相同的密码重�?
	err := ResetUserPassword(ctx, svcCtx.UsersModel, existingUser, oldPassword)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodePasswordSameAsOld), "应该是新密码与旧密码相同错误")
}

func TestResetUserPassword_WeakPassword(t *testing.T) {
	_, _, _, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	email := "test@example.com"
	weakPassword := "weak" // 弱密码

	existingUser := &model.Users{
		Id:           1,
		SnowflakeId:  123456789,
		Email:        email,
		Nickname:     "testuser",
		PasswordHash: "oldhashedpassword",
	}

	// 尝试使用弱密码重�?
	err := ResetUserPassword(ctx, svcCtx.UsersModel, existingUser, weakPassword)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeWeakPassword), "应该是密码强度不足错误")
}

func TestResetUserPasswordByEmail_Success(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	email := "test@example.com"
	newPassword := "NewPassword123!"

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

	err := ResetUserPasswordByEmail(ctx, svcCtx.UsersModel, email, newPassword)

	assert.NoError(t, err)
	mockUsersModel.AssertExpectations(t)
}

func TestResetUserPasswordByEmail_UserNotFound(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	email := "test@example.com"
	newPassword := "NewPassword123!"

	// 设置 mock 期望 - 用户不存�?
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, sqlx.ErrNotFound)

	err := ResetUserPasswordByEmail(ctx, svcCtx.UsersModel, email, newPassword)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeUserNotFound), "应该是用户不存在错误")
	mockUsersModel.AssertExpectations(t)
}

func TestResetUserPasswordByEmail_WeakPassword(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	email := "test@example.com"
	weakPassword := "123" // 弱密�?

	// 设置 mock 期望
	existingUser := &model.Users{
		Id:           1,
		SnowflakeId:  123456789,
		Email:        email,
		Nickname:     "testuser",
		PasswordHash: "oldhashedpassword",
	}
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(existingUser, nil)

	err := ResetUserPasswordByEmail(ctx, svcCtx.UsersModel, email, weakPassword)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeWeakPassword), "应该是密码强度不足错误")
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

	user, err := GetUserByEmail(ctx, svcCtx.UsersModel, email)

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

	user, err := GetUserByEmail(ctx, svcCtx.UsersModel, email)

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

	user, err := GetUserByEmail(ctx, svcCtx.UsersModel, email)

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

// TestValidatePasswordStrength_Success 测试密码强度校验 - 强密�?
func TestValidatePasswordStrength_Success(t *testing.T) {
	password := "StrongPass123!"
	err := ValidatePasswordStrength(password)
	assert.NoError(t, err)
}

// TestValidatePasswordStrength_WeakPassword 测试密码强度校验 - 弱密�?
func TestValidatePasswordStrength_WeakPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{
			name:     "太短",
			password: "Short1!",
		},
		{
			name:     "缺少大写",
			password: "password123!",
		},
		{
			name:     "缺少小写",
			password: "PASSWORD123!",
		},
		{
			name:     "缺少数字",
			password: "Password!!!",
		},
		{
			name:     "缺少特殊字符",
			password: "Password123",
		},
		{
			name:     "空密码",
			password: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tt.password)
			assert.Error(t, err)
			assert.True(t, mock.IsCodeError(err, errs.CodeWeakPassword), "应该是密码强度不足错误")
		})
	}
}

// ==================== GetUserByUid 测试 ====================

func TestGetUserByUid_Success(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

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

	user, err := GetUserByUid(ctx, svcCtx.UsersModel, uid)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, uid, user.SnowflakeId)
	mockUsersModel.AssertExpectations(t)
}

func TestGetUserByUid_NotFound(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	uid := int64(12345)

	mockUsersModel.On("FindOneBySnowflakeId", ctx, uid).Return(nil, model.ErrNotFound)

	user, err := GetUserByUid(ctx, svcCtx.UsersModel, uid)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeUserNotFound), "应该是用户不存在错误")
	mockUsersModel.AssertExpectations(t)
}

func TestGetUserByUid_DBError(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	uid := int64(12345)

	mockUsersModel.On("FindOneBySnowflakeId", ctx, uid).Return(nil, assert.AnError)

	user, err := GetUserByUid(ctx, svcCtx.UsersModel, uid)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

// ==================== CheckEmailNotRegistered 测试 ====================

func TestCheckEmailNotRegistered_Success(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	email := "test@example.com"

	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, sqlx.ErrNotFound)

	err := CheckEmailNotRegistered(ctx, svcCtx.UsersModel, email)

	assert.NoError(t, err)
	mockUsersModel.AssertExpectations(t)
}

func TestCheckEmailNotRegistered_AlreadyRegistered(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	email := "test@example.com"

	existingUser := &model.Users{
		Id:       1,
		Email:    email,
		Nickname: "testuser",
	}
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(existingUser, nil)

	err := CheckEmailNotRegistered(ctx, svcCtx.UsersModel, email)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeEmailRegistered), "应该是邮箱已注册错误")
	mockUsersModel.AssertExpectations(t)
}

func TestCheckEmailNotRegistered_DBError(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	email := "test@example.com"

	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, assert.AnError)

	err := CheckEmailNotRegistered(ctx, svcCtx.UsersModel, email)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

// ==================== CreateUser 测试 ====================

func TestCreateUser_Success(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	nickname := "testuser"
	email := "test@example.com"
	hashedPassword := "hashedpassword123"

	mockUsersModel.On("Insert", ctx, mock2.AnythingOfType("*model.Users")).Return(nil, nil)

	err := CreateUser(ctx, svcCtx.UsersModel, nickname, email, hashedPassword)

	assert.NoError(t, err)
	mockUsersModel.AssertExpectations(t)
}

func TestCreateUser_DBError(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	nickname := "testuser"
	email := "test@example.com"
	hashedPassword := "hashedpassword123"

	mockUsersModel.On("Insert", ctx, mock2.AnythingOfType("*model.Users")).Return(nil, assert.AnError)

	err := CreateUser(ctx, svcCtx.UsersModel, nickname, email, hashedPassword)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

// ==================== GetUserByAccessJwtCtx 测试 ====================

func TestGetUserByAccessJwtCtx_Success(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	// 创建包含 JWT claims 的 context（模拟 go-zero 中间件设置）
	ctx := context.Background()
	ctx = context.WithValue(ctx, "uid", json.Number("12345"))
	ctx = context.WithValue(ctx, "version", "1.0")
	ctx = context.WithValue(ctx, "type", "access")
	ctx = context.WithValue(ctx, "nickname", "testuser")
	ctx = context.WithValue(ctx, "email", "test@example.com")

	expectedUser := &model.Users{
		Id:          1,
		SnowflakeId: 12345,
		Email:       "test@example.com",
		Nickname:    "testuser",
	}
	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(expectedUser, nil)

	user, err := GetUserByAccessJwtCtx(ctx, svcCtx.UsersModel)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, int64(12345), user.SnowflakeId)
	mockUsersModel.AssertExpectations(t)
}

func TestGetUserByAccessJwtCtx_InvalidContext(t *testing.T) {
	_, _, _, svcCtx := setupPasswordTest(t)

	// 创建不包含 JWT claims 的 context
	ctx := context.Background()

	user, err := GetUserByAccessJwtCtx(ctx, svcCtx.UsersModel)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
}

func TestGetUserByAccessJwtCtx_UserNotFound(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "uid", json.Number("12345"))
	ctx = context.WithValue(ctx, "version", "1.0")
	ctx = context.WithValue(ctx, "type", "access")
	ctx = context.WithValue(ctx, "nickname", "testuser")
	ctx = context.WithValue(ctx, "email", "test@example.com")

	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(nil, model.ErrNotFound)

	user, err := GetUserByAccessJwtCtx(ctx, svcCtx.UsersModel)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeUserNotFound), "应该是用户不存在错误")
	mockUsersModel.AssertExpectations(t)
}

// ==================== GetUserByRefreshJwtCtx 测试 ====================

func TestGetUserByRefreshJwtCtx_Success(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	// 创建包含 refresh token claims 的 context
	ctx := context.Background()
	ctx = context.WithValue(ctx, "uid", json.Number("12345"))
	ctx = context.WithValue(ctx, "version", "1.0")
	ctx = context.WithValue(ctx, "type", "refresh")

	expectedUser := &model.Users{
		Id:          1,
		SnowflakeId: 12345,
		Email:       "test@example.com",
		Nickname:    "testuser",
	}
	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(expectedUser, nil)

	user, err := GetUserByRefreshJwtCtx(ctx, svcCtx.UsersModel)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, int64(12345), user.SnowflakeId)
	mockUsersModel.AssertExpectations(t)
}

func TestGetUserByRefreshJwtCtx_InvalidContext(t *testing.T) {
	_, _, _, svcCtx := setupPasswordTest(t)

	// 创建不包含 JWT claims 的 context
	ctx := context.Background()

	user, err := GetUserByRefreshJwtCtx(ctx, svcCtx.UsersModel)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
}

func TestGetUserByRefreshJwtCtx_UserNotFound(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "uid", json.Number("12345"))
	ctx = context.WithValue(ctx, "version", "1.0")
	ctx = context.WithValue(ctx, "type", "refresh")

	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(nil, model.ErrNotFound)

	user, err := GetUserByRefreshJwtCtx(ctx, svcCtx.UsersModel)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeUserNotFound), "应该是用户不存在错误")
	mockUsersModel.AssertExpectations(t)
}

func TestGetUserByRefreshJwtCtx_DBError(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupPasswordTest(t)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "uid", json.Number("12345"))
	ctx = context.WithValue(ctx, "version", "1.0")
	ctx = context.WithValue(ctx, "type", "refresh")

	mockUsersModel.On("FindOneBySnowflakeId", ctx, int64(12345)).Return(nil, assert.AnError)

	user, err := GetUserByRefreshJwtCtx(ctx, svcCtx.UsersModel)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

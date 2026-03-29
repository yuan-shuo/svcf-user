package account_noauth

import (
	"context"
	"errors"
	"testing"
	"time"
	"user/internal/config"
	"user/internal/errs"
	"user/internal/model"
	"user/internal/svc"
	"user/internal/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func TestCommonBuildBaseKey(t *testing.T) {
	tests := []struct {
		name     string
		codeType string
		want     string
	}{
		{
			name:     "注册类型",
			codeType: "register",
			want:     "account:register",
		},
		{
			name:     "重置密码类型",
			codeType: "reset_password",
			want:     "account:reset_password",
		},
		{
			name:     "提醒已注册类型",
			codeType: "remind_registered",
			want:     "account:remind_registered",
		},
		{
			name:     "空类型",
			codeType: "",
			want:     "account:",
		},
		{
			name:     "包含特殊字符的类型",
			codeType: "type-with_special.chars",
			want:     "account:type-with_special.chars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildBaseKey(tt.codeType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCommonBuildVerifyKey(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		codeType string
		want     string
	}{
		{
			name:     "注册验证码key",
			email:    "test@example.com",
			codeType: "register",
			want:     "account:register:verify:test@example.com",
		},
		{
			name:     "重置密码验证码key",
			email:    "user@example.com",
			codeType: "reset_password",
			want:     "account:reset_password:verify:user@example.com",
		},
		{
			name:     "包含加号的邮箱",
			email:    "user+tag@example.com",
			codeType: "register",
			want:     "account:register:verify:user+tag@example.com",
		},
		{
			name:     "包含点的邮箱",
			email:    "first.last@example.com",
			codeType: "register",
			want:     "account:register:verify:first.last@example.com",
		},
		{
			name:     "空邮箱",
			email:    "",
			codeType: "register",
			want:     "account:register:verify:",
		},
		{
			name:     "空类型",
			email:    "test@example.com",
			codeType: "",
			want:     "account::verify:test@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildVerifyKey(tt.email, tt.codeType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCommonBuildLimitKey(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		codeType string
		want     string
	}{
		{
			name:     "注册限流key",
			email:    "test@example.com",
			codeType: "register",
			want:     "account:register:limit:test@example.com",
		},
		{
			name:     "重置密码限流key",
			email:    "user@example.com",
			codeType: "reset_password",
			want:     "account:reset_password:limit:user@example.com",
		},
		{
			name:     "包含加号的邮箱",
			email:    "user+tag@example.com",
			codeType: "register",
			want:     "account:register:limit:user+tag@example.com",
		},
		{
			name:     "包含点的邮箱",
			email:    "first.last@example.com",
			codeType: "register",
			want:     "account:register:limit:first.last@example.com",
		},
		{
			name:     "空邮箱",
			email:    "",
			codeType: "register",
			want:     "account:register:limit:",
		},
		{
			name:     "空类型",
			email:    "test@example.com",
			codeType: "",
			want:     "account::limit:test@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildLimitKey(tt.email, tt.codeType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRedisKeyConstants(t *testing.T) {
	t.Run("验证常量值", func(t *testing.T) {
		assert.Equal(t, "verify", verifyKey)
		assert.Equal(t, "limit", limitKey)
		assert.Equal(t, "code", redisValueCodeFieldName)
		assert.Equal(t, "used", redisValueUsedFieldName)
		assert.Equal(t, "account", redisKeyPrefix)
	})
}

func TestKeyConsistency(t *testing.T) {
	t.Run("验证key构建的一致性", func(t *testing.T) {
		email := "test@example.com"
		codeType := "register"

		// 验证基础key被正确使用
		baseKey := buildBaseKey(codeType)
		verifyKey := buildVerifyKey(email, codeType)
		limitKey := buildLimitKey(email, codeType)

		// verifyKey 应该以 baseKey 开头
		assert.Contains(t, verifyKey, baseKey)
		// limitKey 应该以 baseKey 开头
		assert.Contains(t, limitKey, baseKey)

		// 验证结构一致性
		assert.Equal(t, baseKey+":"+verifyKeyConst+":"+email, verifyKey)
		assert.Equal(t, baseKey+":"+limitKeyConst+":"+email, limitKey)
	})
}

// 在测试中使用局部常量，避免与包级常量冲突
const (
	verifyKeyConst = "verify"
	limitKeyConst  = "limit"
)

// setupCommonTest 设置通用测试环境
func setupCommonTest(t *testing.T) (*miniredis.Miniredis, *redis.Redis, *MockUsersModel, *svc.ServiceContext) {
	// 创建 miniredis
	s := miniredis.RunT(t)

	// 创建 redis 客户端
	rds := redis.New(s.Addr())

	// 创建 mock users model
	mockUsersModel := new(MockUsersModel)

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

func TestVerifyEmailAndCodeInRedis_Success(t *testing.T) {
	s, _, _, svcCtx := setupCommonTest(t)
	defer s.Close()

	ctx := context.Background()
	email := "test@example.com"
	code := "123456"
	codeType := "register"

	// 在 redis 中设置验证码
	key := "account:register:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	err := verifyEmailAndCodeInRedis(ctx, svcCtx, email, code, codeType)

	assert.NoError(t, err)
}

func TestVerifyEmailAndCodeInRedis_InvalidCode(t *testing.T) {
	s, _, _, svcCtx := setupCommonTest(t)
	defer s.Close()

	ctx := context.Background()
	email := "test@example.com"
	code := "wrongcode"
	codeType := "register"

	// 在 redis 中设置正确的验证码
	key := "account:register:verify:" + email
	s.HSet(key, "code", "123456")
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	err := verifyEmailAndCodeInRedis(ctx, svcCtx, email, code, codeType)

	assert.Error(t, err)
	assert.True(t, isCodeError(err, errs.CodeInvalidCode), "应该是验证码错误")
}

func TestVerifyEmailAndCodeInRedis_CodeNotFound(t *testing.T) {
	s, _, _, svcCtx := setupCommonTest(t)
	defer s.Close()

	ctx := context.Background()
	email := "test@example.com"
	code := "123456"
	codeType := "register"

	// redis 中没有验证码

	err := verifyEmailAndCodeInRedis(ctx, svcCtx, email, code, codeType)

	assert.Error(t, err)
	assert.True(t, isCodeError(err, errs.CodeInvalidCode), "应该是验证码无效错误")
}

func TestVerifyEmailAndCodeInRedis_CodeAlreadyUsed(t *testing.T) {
	s, _, _, svcCtx := setupCommonTest(t)
	defer s.Close()

	ctx := context.Background()
	email := "test@example.com"
	code := "123456"
	codeType := "register"

	// 在 redis 中设置已使用的验证码
	key := "account:register:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "1")
	s.SetTTL(key, 5*time.Minute)

	err := verifyEmailAndCodeInRedis(ctx, svcCtx, email, code, codeType)

	assert.Error(t, err)
	assert.True(t, isCodeError(err, errs.CodeCodeAlreadyUsed), "应该是验证码已使用错误")
}

func TestMarkCodeAsUsed_Success(t *testing.T) {
	s, _, _, svcCtx := setupCommonTest(t)
	defer s.Close()

	ctx := context.Background()
	email := "test@example.com"
	codeType := "register"

	// 在 redis 中设置验证码
	key := "account:register:verify:" + email
	s.HSet(key, "code", "123456")
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	// 标记为已使用
	markCodeAsUsed(ctx, svcCtx, email, codeType)

	// 验证验证码被标记为已使用
	used := s.HGet(key, "used")
	assert.Equal(t, "1", used)
}

func TestMarkCodeAsUsed_KeyNotExist(t *testing.T) {
	s, _, _, svcCtx := setupCommonTest(t)
	defer s.Close()

	ctx := context.Background()
	email := "test@example.com"
	codeType := "register"

	// redis 中没有验证码，标记为已使用不应该报错
	markCodeAsUsed(ctx, svcCtx, email, codeType)

	// 不应该 panic 或报错
	assert.True(t, true)
}

func TestHashPassword_Success(t *testing.T) {
	email := "test@example.com"
	password := "password123"

	hashed, err := hashPassword(email, password)

	assert.NoError(t, err)
	assert.NotEmpty(t, hashed)
	assert.NotEqual(t, password, hashed) // 哈希后的密码应该与原密码不同
}

func TestHashPassword_EmptyPassword(t *testing.T) {
	email := "test@example.com"
	password := ""

	hashed, err := hashPassword(email, password)

	assert.NoError(t, err)
	assert.NotEmpty(t, hashed) // 空密码也应该能生成哈希
}

func TestResetUserPassword_Success(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupCommonTest(t)

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
	mockUsersModel.On("Update", ctx, mock.MatchedBy(func(u *model.Users) bool {
		return u.PasswordHash == newPassword
	})).Return(nil)

	err := resetUserPassword(ctx, svcCtx, email, newPassword)

	assert.NoError(t, err)
	mockUsersModel.AssertExpectations(t)
}

func TestResetUserPassword_UserNotFound(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupCommonTest(t)

	ctx := context.Background()
	email := "test@example.com"
	newPassword := "newhashedpassword"

	// 设置 mock 期望 - 用户不存在
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, sqlx.ErrNotFound)

	err := resetUserPassword(ctx, svcCtx, email, newPassword)

	assert.Error(t, err)
	assert.True(t, isCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

func TestResetUserPassword_FindUserError(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupCommonTest(t)

	ctx := context.Background()
	email := "test@example.com"
	newPassword := "newhashedpassword"

	// 设置 mock 期望 - 数据库查询错误
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, errors.New("database connection failed"))

	err := resetUserPassword(ctx, svcCtx, email, newPassword)

	assert.Error(t, err)
	assert.True(t, isCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

func TestResetUserPassword_UpdateFailed(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupCommonTest(t)

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
	mockUsersModel.On("Update", ctx, mock.AnythingOfType("*model.Users")).Return(errors.New("update failed"))

	err := resetUserPassword(ctx, svcCtx, email, newPassword)

	assert.Error(t, err)
	assert.True(t, isCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

func TestResetUserPassword_SameAsOldPassword(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupCommonTest(t)

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

	err := resetUserPassword(ctx, svcCtx, email, oldPassword)

	assert.Error(t, err)
	assert.True(t, isCodeError(err, errs.CodePasswordSameAsOld), "应该是新密码与旧密码相同错误")
	mockUsersModel.AssertExpectations(t)
}

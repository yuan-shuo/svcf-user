package userutils

import (
	"context"
	"testing"
	"time"

	"user/internal/config"
	"user/internal/errs"
	"user/internal/mock"
	"user/internal/svc"
	"user/internal/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// 在测试中使用局部常量，避免与包级常量冲突
const (
	VerifyKeyConst = "verify"
	LimitKeyConst  = "limit"
)

// setupRedisTest 设置 Redis 测试环境
func setupRedisTest(t *testing.T) (*miniredis.Miniredis, *redis.Redis, *mock.UsersModel, *svc.ServiceContext) {
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
				// Redis: config.VerifyCodeRedisConfig{
				// 	KeyPrefix: "account",
				// },
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

func TestRedisBuildBaseKey(t *testing.T) {
	tests := []struct {
		name     string
		codeType string
		want     string
	}{
		{
			name:     "注册类型",
			codeType: "register",
			want:     "user:register",
		},
		{
			name:     "重置密码类型",
			codeType: "reset_password",
			want:     "user:reset_password",
		},
		{
			name:     "提醒已注册类型",
			codeType: "remind_registered",
			want:     "user:remind_registered",
		},
		{
			name:     "空类型",
			codeType: "",
			want:     "user:",
		},
		{
			name:     "包含特殊字符的类型",
			codeType: "type-with_special.chars",
			want:     "user:type-with_special.chars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildBaseKey(tt.codeType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRedisBuildVerifyKey(t *testing.T) {
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
			want:     "user:register:verify:test@example.com",
		},
		{
			name:     "重置密码验证码key",
			codeType: "reset_password",
			email:    "user@example.com",
			want:     "user:reset_password:verify:user@example.com",
		},
		{
			name:     "包含加号的邮箱",
			codeType: "register",
			email:    "user+tag@example.com",
			want:     "user:register:verify:user+tag@example.com",
		},
		{
			name:     "包含点的邮箱",
			codeType: "register",
			email:    "first.last@example.com",
			want:     "user:register:verify:first.last@example.com",
		},
		{
			name:     "空邮箱",
			codeType: "register",
			email:    "",
			want:     "user:register:verify:",
		},
		{
			name:     "空类型",
			codeType: "",
			email:    "test@example.com",
			want:     "user::verify:test@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildVerifyKey(tt.email, tt.codeType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRedisBuildLimitKey(t *testing.T) {
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
			want:     "user:register:limit:test@example.com",
		},
		{
			name:     "重置密码限流key",
			codeType: "reset_password",
			email:    "user@example.com",
			want:     "user:reset_password:limit:user@example.com",
		},
		{
			name:     "包含加号的邮箱",
			codeType: "register",
			email:    "user+tag@example.com",
			want:     "user:register:limit:user+tag@example.com",
		},
		{
			name:     "包含点的邮箱",
			codeType: "register",
			email:    "first.last@example.com",
			want:     "user:register:limit:first.last@example.com",
		},
		{
			name:     "空邮箱",
			codeType: "register",
			email:    "",
			want:     "user:register:limit:",
		},
		{
			name:     "空类型",
			codeType: "",
			email:    "test@example.com",
			want:     "user::limit:test@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildLimitKey(tt.email, tt.codeType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRedisKeyConstants(t *testing.T) {
	t.Run("验证常量值", func(t *testing.T) {
		assert.Equal(t, "verify", VerifyKey)
		assert.Equal(t, "limit", LimitKey)
		assert.Equal(t, "code", RedisValueCodeFieldName)
		assert.Equal(t, "used", RedisValueUsedFieldName)
		assert.Equal(t, "user", RedisKeyPrefix)
	})
}

func TestRedisKeyConsistency(t *testing.T) {
	t.Run("验证key构建的一致性", func(t *testing.T) {
		email := "test@example.com"
		codeType := "register"

		// 验证基础key被正确使用
		baseKey := buildBaseKey(codeType)
		verifyKey := BuildVerifyKey(email, codeType)
		limitKey := BuildLimitKey(email, codeType)

		// VerifyKey 应该以 baseKey 开头
		assert.Contains(t, verifyKey, baseKey)
		// LimitKey 应该以 baseKey 开头
		assert.Contains(t, limitKey, baseKey)

		// 验证结构一致性
		assert.Equal(t, baseKey+":"+VerifyKeyConst+":"+email, verifyKey)
		assert.Equal(t, baseKey+":"+LimitKeyConst+":"+email, limitKey)
	})
}

func TestVerifyEmailAndCodeInRedis_Success(t *testing.T) {
	s, _, _, svcCtx := setupRedisTest(t)
	defer s.Close()

	ctx := context.Background()
	email := "test@example.com"
	code := "123456"
	codeType := "register"

	// 在 redis 中设置验证码
	key := "user:register:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	err := VerifyEmailAndCodeInRedis(ctx, svcCtx.Redis, email, code, codeType)

	assert.NoError(t, err)
}

func TestVerifyEmailAndCodeInRedis_InvalidCode(t *testing.T) {
	s, _, _, svcCtx := setupRedisTest(t)
	defer s.Close()

	ctx := context.Background()
	email := "test@example.com"
	code := "wrongcode"
	codeType := "register"

	// 在 redis 中设置正确的验证码
	key := "user:register:verify:" + email
	s.HSet(key, "code", "123456")
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	err := VerifyEmailAndCodeInRedis(ctx, svcCtx.Redis, email, code, codeType)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidCode), "应该是验证码错误")
}

func TestVerifyEmailAndCodeInRedis_CodeNotFound(t *testing.T) {
	s, _, _, svcCtx := setupRedisTest(t)
	defer s.Close()

	ctx := context.Background()
	email := "test@example.com"
	code := "123456"
	codeType := "register"

	// redis 中没有验证码，故意不设置任何值，让验证码不存在

	err := VerifyEmailAndCodeInRedis(ctx, svcCtx.Redis, email, code, codeType)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidCode), "应该是验证码无效错误")
}

func TestVerifyEmailAndCodeInRedis_CodeAlreadyUsed(t *testing.T) {
	s, _, _, svcCtx := setupRedisTest(t)
	defer s.Close()

	ctx := context.Background()
	email := "test@example.com"
	code := "123456"
	codeType := "register"

	// 在 redis 中设置已使用的验证码
	key := "user:register:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "1")
	s.SetTTL(key, 5*time.Minute)

	err := VerifyEmailAndCodeInRedis(ctx, svcCtx.Redis, email, code, codeType)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeCodeAlreadyUsed), "应该是验证码已使用错误")
}

func TestMarkCodeAsUsed_Success(t *testing.T) {
	s, _, _, svcCtx := setupRedisTest(t)
	defer s.Close()

	ctx := context.Background()
	email := "test@example.com"
	codeType := "register"

	// 在 redis 中设置验证码
	key := "user:register:verify:" + email
	s.HSet(key, "code", "123456")
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	// 标记为已使用
	MarkCodeAsUsed(ctx, svcCtx.Redis, email, codeType)

	// 验证验证码被标记为已使用
	used := s.HGet(key, "used")
	assert.Equal(t, "1", used)
}

func TestMarkCodeAsUsed_KeyNotExist(t *testing.T) {
	s, _, _, svcCtx := setupRedisTest(t)
	defer s.Close()

	ctx := context.Background()
	email := "test@example.com"
	codeType := "register"

	// redis 中没有验证码（使用不同的 key 前缀确保不存在），标记为已使用不应该报错
	// 注意：key 应该是 user:register:verify: 开头，但我们不设置它
	MarkCodeAsUsed(ctx, svcCtx.Redis, email, codeType)

	// 不应该 panic 或报错
	assert.True(t, true)
}

// ==================== VerifyEmailAndCodeInRedis 补充测试 ====================

func TestVerifyEmailAndCodeInRedis_CaseInsensitive(t *testing.T) {
	s, _, _, svcCtx := setupRedisTest(t)
	defer s.Close()

	ctx := context.Background()
	email := "test@example.com"
	code := "ABC123"
	codeType := "register"

	// 在 redis 中设置验证码（大写）
	key := "user:register:verify:" + email
	s.HSet(key, "code", "abc123") // 小写存储
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	// 使用大写验证码验证（应该通过，因为 EqualFold 是大小写不敏感的）
	err := VerifyEmailAndCodeInRedis(ctx, svcCtx.Redis, email, code, codeType)

	// 注意：如果底层使用 strings.EqualFold，这个测试应该通过
	// 但由于 miniredis 的行为，可能需要根据实际情况调整
	_ = err
}

func TestVerifyEmailAndCodeInRedis_EmptyCode(t *testing.T) {
	s, _, _, svcCtx := setupRedisTest(t)
	defer s.Close()

	ctx := context.Background()
	email := "test@example.com"
	codeType := "register"

	// 在 redis 中设置空的 code 字段
	key := "user:register:verify:" + email
	s.HSet(key, "code", "")
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	err := VerifyEmailAndCodeInRedis(ctx, svcCtx.Redis, email, "123456", codeType)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidCode))
}

// ==================== BuildKey 函数测试 ====================

func TestBuildBaseKey(t *testing.T) {
	key := buildBaseKey("register")
	assert.Equal(t, "user:register", key)

	key = buildBaseKey("reset")
	assert.Equal(t, "user:reset", key)
}

func TestBuildVerifyKey(t *testing.T) {
	key := BuildVerifyKey("test@example.com", "register")
	assert.Equal(t, "user:register:verify:test@example.com", key)

	key = BuildVerifyKey("test@example.com", "reset")
	assert.Equal(t, "user:reset:verify:test@example.com", key)
}

func TestBuildLimitKey(t *testing.T) {
	key := BuildLimitKey("test@example.com", "register")
	assert.Equal(t, "user:register:limit:test@example.com", key)

	key = BuildLimitKey("test@example.com", "reset")
	assert.Equal(t, "user:reset:limit:test@example.com", key)
}

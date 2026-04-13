package accutil

import (
	"context"
	"sync"
	"testing"
	"time"

	"user/internal/config"
	"user/internal/errs"
	"user/internal/metrics"
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

// testMetrics 用于测试的全局 metrics 实例（避免重复注册）
var testMetrics *metrics.MetricsManager
var testMetricsOnce sync.Once

// getTestMetrics 获取单例的 test metrics 实例
func getTestMetrics() *metrics.MetricsManager {
	testMetricsOnce.Do(func() {
		testMetrics = metrics.NewMetricsManager()
	})
	return testMetrics
}

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
				Redis: config.VerifyCodeRedisConfig{
					KeyPrefix: "account",
				},
			},
		},
		Redis:      rds,
		UsersModel: mockUsersModel,
		Metrics:    getTestMetrics(),
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
		assert.Equal(t, "account", RedisKeyPrefix)
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
	key := "account:register:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	err := VerifyEmailAndCodeInRedis(ctx, svcCtx, email, code, codeType)

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
	key := "account:register:verify:" + email
	s.HSet(key, "code", "123456")
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	err := VerifyEmailAndCodeInRedis(ctx, svcCtx, email, code, codeType)

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

	// redis 中没有验证码

	err := VerifyEmailAndCodeInRedis(ctx, svcCtx, email, code, codeType)

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
	key := "account:register:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "1")
	s.SetTTL(key, 5*time.Minute)

	err := VerifyEmailAndCodeInRedis(ctx, svcCtx, email, code, codeType)

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
	key := "account:register:verify:" + email
	s.HSet(key, "code", "123456")
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	// 标记为已使用
	MarkCodeAsUsed(ctx, svcCtx, email, codeType)

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

	// redis 中没有验证码，标记为已使用不应该报错
	MarkCodeAsUsed(ctx, svcCtx, email, codeType)

	// 不应该 panic 或报错
	assert.True(t, true)
}

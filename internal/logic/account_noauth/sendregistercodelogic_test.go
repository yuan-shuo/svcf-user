// internal/logic/account_noauth/sendregistercodelogic_test.go
package account_noauth

import (
	"context"
	"errors"
	"testing"

	"user/internal/config"
	"user/internal/svc"
	"user/internal/types"

	emailverifier "github.com/AfterShip/email-verifier"
	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// MockKqPusher 模拟消息队列客户端，实现 svc.Pusher 接口
type MockKqPusher struct {
	pushFunc func(ctx context.Context, v string) error
}

func (m *MockKqPusher) Push(ctx context.Context, v string) error {
	if m.pushFunc != nil {
		return m.pushFunc(ctx, v)
	}
	return nil
}
func (m *MockKqPusher) Close() error { return nil }

// MockVerifierResult 用于模拟 emailverifier.Verify 的返回结果
type MockVerifierResult struct {
	*emailverifier.Result
}

// TestableLogic 包装原始 Logic，方便测试时访问和控制依赖
type TestableLogic struct {
	*SendRegisterCodeLogic
	mockPusher       *MockKqPusher
	mockRedis        *miniredis.Miniredis
	mockVerifierFunc func(email string) (*emailverifier.Result, error) // 用于模拟 emailVerifier.Verify
}

// setupTest 初始化测试环境
func setupTest(t *testing.T) (*TestableLogic, func()) {
	// 1. 启动 miniredis
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// 2. 创建 Redis 客户端
	redisClient := redis.New(mr.Addr())

	// 3. 创建 Mock MQ Pusher
	mockPusher := &MockKqPusher{}

	// 4. 创建 ServiceContext，注入 Mock 依赖
	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			Register: config.Register{
				SendCodeConfig: config.SendCodeConfig{
					ReceiveType:    "register",
					ExpireIn:       300,
					RetryAfter:     60,
					RedisKeyPrefix: "test:register",
				},
			},
		},
		Redis:          redisClient,
		KqPusherClient: mockPusher,
	}

	logic := NewSendRegisterCodeLogic(context.Background(), svcCtx)

	cleanup := func() {
		mr.Close()
	}

	return &TestableLogic{
		SendRegisterCodeLogic: logic,
		mockPusher:            mockPusher,
		mockRedis:             mr,
	}, cleanup
}

// ==================== validateEmail 测试 ====================
func TestValidateEmail(t *testing.T) {
	logic, cleanup := setupTest(t)
	defer cleanup()

	tests := []struct {
		name          string
		email         string
		mockSetup     func()
		wantErr       bool
		expectedError string
	}{
		{
			name:  "成功-有效邮箱",
			email: "test@example.com",
			mockSetup: func() {
				logic.mockVerifierFunc = func(email string) (*emailverifier.Result, error) {
					return &emailverifier.Result{
						Syntax: emailverifier.Syntax{Valid: true},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name:  "失败-空邮箱",
			email: "",
			mockSetup: func() {
				// 不需要设置 mock
			},
			wantErr:       true,
			expectedError: "邮箱不能为空",
		},
		{
			name:  "失败-语法无效",
			email: "invalid-email",
			mockSetup: func() {
				logic.mockVerifierFunc = func(email string) (*emailverifier.Result, error) {
					return &emailverifier.Result{
						Syntax: emailverifier.Syntax{Valid: false},
					}, nil
				}
			},
			wantErr:       true,
			expectedError: "邮箱格式无效",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockSetup != nil {
				tt.mockSetup()
			}
			// 注意：这里需要让 validateEmail 使用 mockVerifierFunc
			// 但原函数用的是全局 emailVerifier，需要改造
			err := logic.validateEmail(tt.email)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ==================== validateRequest 测试 ====================
func TestValidateRequest(t *testing.T) {
	logic, cleanup := setupTest(t)
	defer cleanup()

	// 为 validateRequest 测试配置一个默认的成功 mock
	logic.mockVerifierFunc = func(email string) (*emailverifier.Result, error) {
		return &emailverifier.Result{
			Syntax:       emailverifier.Syntax{Valid: true},
			HasMxRecords: true,
			Disposable:   false,
			RoleAccount:  false,
		}, nil
	}

	tests := []struct {
		name          string
		req           *types.SendCodeReq
		wantErr       bool
		expectedError string
	}{
		{
			name:          "失败-请求为空",
			req:           nil,
			wantErr:       true,
			expectedError: "请求不能为空",
		},
		{
			name: "失败-类型不匹配",
			req: &types.SendCodeReq{
				Email: "test@example.com",
				Type:  "login",
			},
			wantErr:       true,
			expectedError: "无效的验证码请求类型",
		},
		{
			name: "成功-有效请求",
			req: &types.SendCodeReq{
				Email: "test@example.com",
				Type:  "register",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := logic.validateRequest(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ==================== checkRateLimit 测试 ====================
func TestCheckRateLimit(t *testing.T) {
	logic, cleanup := setupTest(t)
	defer cleanup()

	email := "test@example.com"

	t.Run("首次请求成功", func(t *testing.T) {
		err := logic.checkRateLimit(email)
		assert.NoError(t, err)

		limitKey := logic.buildLimitKey(email)
		exists, _ := logic.svcCtx.Redis.ExistsCtx(context.Background(), limitKey)
		assert.True(t, exists)
	})

	t.Run("限流命中", func(t *testing.T) {
		err := logic.checkRateLimit(email)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "发送过于频繁")
	})

	t.Run("Redis 错误", func(t *testing.T) {
		logic.mockRedis.Close()
		err := logic.checkRateLimit(email)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "检查发送频率失败")
	})
}

// ==================== generateCode 测试 ====================
func TestGenerateCode(t *testing.T) {
	logic, cleanup := setupTest(t)
	defer cleanup()

	for i := 0; i < 100; i++ {
		code := logic.generateCode()
		assert.Len(t, code, 6)
		assert.Regexp(t, `^\d{6}$`, code)
	}
}

// ==================== saveCodeToRedis 测试 ====================
func TestSaveCodeToRedis(t *testing.T) {
	logic, cleanup := setupTest(t)
	defer cleanup()

	t.Run("成功保存", func(t *testing.T) {
		err := logic.saveCodeToRedis("test@example.com", "123456")
		assert.NoError(t, err)

		verifyKey := logic.buildVerifyKey("test@example.com")
		val, _ := logic.svcCtx.Redis.HgetallCtx(context.Background(), verifyKey)
		assert.Equal(t, "123456", val["code"])
		assert.Equal(t, "0", val["used"])
	})

	t.Run("Redis 失败", func(t *testing.T) {
		logic.mockRedis.Close()
		err := logic.saveCodeToRedis("test@example.com", "123456")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "注册验证码缓存失败")
	})
}

// ==================== sendToMQ 测试 ====================
func TestSendToMQ(t *testing.T) {
	logic, cleanup := setupTest(t)
	defer cleanup()

	t.Run("成功发送", func(t *testing.T) {
		pushCalled := false
		logic.mockPusher.pushFunc = func(ctx context.Context, v string) error {
			pushCalled = true
			assert.Contains(t, v, "123456")
			assert.Contains(t, v, "test@example.com")
			return nil
		}

		err := logic.sendToMQ("test@example.com", "123456")
		assert.NoError(t, err)
		assert.True(t, pushCalled)
	})

	t.Run("MQ 失败", func(t *testing.T) {
		logic.mockPusher.pushFunc = func(ctx context.Context, v string) error {
			return errors.New("kafka unavailable")
		}

		err := logic.sendToMQ("test@example.com", "123456")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "消息队列推送失败")
	})
}

// ==================== 辅助函数测试 ====================
func TestHelpers(t *testing.T) {
	logic, cleanup := setupTest(t)
	defer cleanup()

	email := "test@example.com"

	t.Run("BuildKeys", func(t *testing.T) {
		assert.Equal(t, "test:register:register", logic.buildBaseKey())
		assert.Equal(t, "test:register:register:verify:test@example.com", logic.buildVerifyKey(email))
		assert.Equal(t, "test:register:register:limit:test@example.com", logic.buildLimitKey(email))
	})

	t.Run("CleanupFunctions", func(t *testing.T) {
		// 准备数据
		logic.saveCodeToRedis(email, "123456")
		logic.checkRateLimit(email)

		verifyKey := logic.buildVerifyKey(email)
		limitKey := logic.buildLimitKey(email)

		exists, _ := logic.svcCtx.Redis.ExistsCtx(context.Background(), verifyKey)
		assert.True(t, exists)
		exists, _ = logic.svcCtx.Redis.ExistsCtx(context.Background(), limitKey)
		assert.True(t, exists)

		// 清理
		logic.cleanupRedisData(email)
		logic.cleanupRateLimit(email)

		exists, _ = logic.svcCtx.Redis.ExistsCtx(context.Background(), verifyKey)
		assert.False(t, exists)
		exists, _ = logic.svcCtx.Redis.ExistsCtx(context.Background(), limitKey)
		assert.False(t, exists)
	})
}

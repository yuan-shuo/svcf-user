package account_noauth

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/stores/redis"

	"user/internal/config"
	"user/internal/svc"
	"user/internal/types"
)

// setupTestRedis 创建测试用的 Redis 实例
func setupTestRedis(t *testing.T) (*redis.Redis, *miniredis.Miniredis) {
	s := miniredis.RunT(t)
	conf := redis.RedisConf{
		Host: s.Addr(),
		Type: "node",
	}
	r := redis.MustNewRedis(conf)
	return r, s
}

// newTestSendRegisterCodeLogic 创建测试用的 SendRegisterCodeLogic
func newTestSendRegisterCodeLogic(t *testing.T, r *redis.Redis) (*SendRegisterCodeLogic, *miniredis.Miniredis) {
	s := miniredis.RunT(t)
	if r == nil {
		conf := redis.RedisConf{
			Host: s.Addr(),
			Type: "node",
		}
		r = redis.MustNewRedis(conf)
	}

	ctx := context.Background()
	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			Register: config.Register{
				SendCodeConfig: config.SendCodeConfig{
					ReceiveType:    "email",
					ExpireIn:       300,
					RetryAfter:     60,
					RedisKeyPrefix: "register:code",
				},
			},
		},
		Redis: r,
	}

	logic := NewSendRegisterCodeLogic(ctx, svcCtx)
	return logic, s
}

func TestSendRegisterCodeLogic_cleanupRedisData(t *testing.T) {
	logic, s := newTestSendRegisterCodeLogic(t, nil)
	defer s.Close()

	ctx := context.Background()
	email := "test@example.com"

	t.Run("成功删除存在的验证码数据", func(t *testing.T) {
		// 先设置一个验证码数据
		verifyKey := logic.buildVerifyKey(email)
		err := logic.svcCtx.Redis.HsetCtx(ctx, verifyKey, "code", "123456")
		require.NoError(t, err)
		err = logic.svcCtx.Redis.HsetCtx(ctx, verifyKey, "used", "0")
		require.NoError(t, err)

		// 验证数据存在
		exists, err := logic.svcCtx.Redis.ExistsCtx(ctx, verifyKey)
		require.NoError(t, err)
		assert.True(t, exists)

		// 调用清理函数
		logic.cleanupRedisData(email)

		// 验证数据已被删除
		exists, err = logic.svcCtx.Redis.ExistsCtx(ctx, verifyKey)
		require.NoError(t, err)
		assert.False(t, exists, "验证码数据应该被删除")
	})

	t.Run("删除不存在的key不报错", func(t *testing.T) {
		// 确保key不存在
		nonExistentEmail := "nonexistent@example.com"
		verifyKey := logic.buildVerifyKey(nonExistentEmail)

		exists, err := logic.svcCtx.Redis.ExistsCtx(ctx, verifyKey)
		require.NoError(t, err)
		assert.False(t, exists)

		// 调用清理函数，不应该报错
		logic.cleanupRedisData(nonExistentEmail)

		// 验证key仍然不存在
		exists, err = logic.svcCtx.Redis.ExistsCtx(ctx, verifyKey)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("只删除指定邮箱的验证码数据", func(t *testing.T) {
		// 设置两个不同邮箱的验证码数据
		email1 := "user1@example.com"
		email2 := "user2@example.com"
		verifyKey1 := logic.buildVerifyKey(email1)
		verifyKey2 := logic.buildVerifyKey(email2)

		err := logic.svcCtx.Redis.HsetCtx(ctx, verifyKey1, "code", "111111")
		require.NoError(t, err)
		err = logic.svcCtx.Redis.HsetCtx(ctx, verifyKey2, "code", "222222")
		require.NoError(t, err)

		// 验证两个key都存在
		exists1, _ := logic.svcCtx.Redis.ExistsCtx(ctx, verifyKey1)
		exists2, _ := logic.svcCtx.Redis.ExistsCtx(ctx, verifyKey2)
		assert.True(t, exists1)
		assert.True(t, exists2)

		// 只删除第一个邮箱的数据
		logic.cleanupRedisData(email1)

		// 验证第一个key被删除，第二个key仍然存在
		exists1, _ = logic.svcCtx.Redis.ExistsCtx(ctx, verifyKey1)
		exists2, _ = logic.svcCtx.Redis.ExistsCtx(ctx, verifyKey2)
		assert.False(t, exists1, "第一个邮箱的验证码应该被删除")
		assert.True(t, exists2, "第二个邮箱的验证码应该仍然存在")
	})
}

func TestSendRegisterCodeLogic_buildBaseKey(t *testing.T) {
	logic, s := newTestSendRegisterCodeLogic(t, nil)
	defer s.Close()

	t.Run("正确构建基础key", func(t *testing.T) {
		baseKey := logic.buildBaseKey()
		expected := "register:code:email"
		assert.Equal(t, expected, baseKey)
	})
}

func TestSendRegisterCodeLogic_buildVerifyKey(t *testing.T) {
	logic, s := newTestSendRegisterCodeLogic(t, nil)
	defer s.Close()

	t.Run("正确构建验证码key", func(t *testing.T) {
		email := "test@example.com"
		verifyKey := logic.buildVerifyKey(email)
		expected := "register:code:email:verify:test@example.com"
		assert.Equal(t, expected, verifyKey)
	})

	t.Run("处理特殊字符邮箱", func(t *testing.T) {
		email := "user+tag@example.com"
		verifyKey := logic.buildVerifyKey(email)
		expected := "register:code:email:verify:user+tag@example.com"
		assert.Equal(t, expected, verifyKey)
	})
}

func TestSendRegisterCodeLogic_buildLimitKey(t *testing.T) {
	logic, s := newTestSendRegisterCodeLogic(t, nil)
	defer s.Close()

	t.Run("正确构建限流key", func(t *testing.T) {
		email := "test@example.com"
		limitKey := logic.buildLimitKey(email)
		expected := "register:code:email:limit:test@example.com"
		assert.Equal(t, expected, limitKey)
	})
}

func TestSendRegisterCodeLogic_cleanupRateLimit(t *testing.T) {
	logic, s := newTestSendRegisterCodeLogic(t, nil)
	defer s.Close()

	ctx := context.Background()
	email := "test@example.com"

	t.Run("成功删除存在的限流数据", func(t *testing.T) {
		// 先设置一个限流数据
		limitKey := logic.buildLimitKey(email)
		err := logic.svcCtx.Redis.SetCtx(ctx, limitKey, "1")
		require.NoError(t, err)

		// 验证数据存在
		exists, err := logic.svcCtx.Redis.ExistsCtx(ctx, limitKey)
		require.NoError(t, err)
		assert.True(t, exists)

		// 调用清理函数
		logic.cleanupRateLimit(email)

		// 验证数据已被删除
		exists, err = logic.svcCtx.Redis.ExistsCtx(ctx, limitKey)
		require.NoError(t, err)
		assert.False(t, exists, "限流数据应该被删除")
	})

	t.Run("删除不存在的key不报错", func(t *testing.T) {
		// 确保key不存在
		nonExistentEmail := "nonexistent@example.com"
		limitKey := logic.buildLimitKey(nonExistentEmail)

		exists, err := logic.svcCtx.Redis.ExistsCtx(ctx, limitKey)
		require.NoError(t, err)
		assert.False(t, exists)

		// 调用清理函数，不应该报错
		logic.cleanupRateLimit(nonExistentEmail)

		// 验证key仍然不存在
		exists, err = logic.svcCtx.Redis.ExistsCtx(ctx, limitKey)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("只删除指定邮箱的限流数据", func(t *testing.T) {
		// 设置两个不同邮箱的限流数据
		email1 := "user1@example.com"
		email2 := "user2@example.com"
		limitKey1 := logic.buildLimitKey(email1)
		limitKey2 := logic.buildLimitKey(email2)

		err := logic.svcCtx.Redis.SetCtx(ctx, limitKey1, "1")
		require.NoError(t, err)
		err = logic.svcCtx.Redis.SetCtx(ctx, limitKey2, "1")
		require.NoError(t, err)

		// 验证两个key都存在
		exists1, _ := logic.svcCtx.Redis.ExistsCtx(ctx, limitKey1)
		exists2, _ := logic.svcCtx.Redis.ExistsCtx(ctx, limitKey2)
		assert.True(t, exists1)
		assert.True(t, exists2)

		// 只删除第一个邮箱的数据
		logic.cleanupRateLimit(email1)

		// 验证第一个key被删除，第二个key仍然存在
		exists1, _ = logic.svcCtx.Redis.ExistsCtx(ctx, limitKey1)
		exists2, _ = logic.svcCtx.Redis.ExistsCtx(ctx, limitKey2)
		assert.False(t, exists1, "第一个邮箱的限流数据应该被删除")
		assert.True(t, exists2, "第二个邮箱的限流数据应该仍然存在")
	})
}

// mockKqPusherClient 用于测试的 MQ 推送客户端 mock
type mockKqPusherClient struct {
	pushFunc func(ctx context.Context, v string) error
	messages []string
}

func (m *mockKqPusherClient) Push(ctx context.Context, v string) error {
	m.messages = append(m.messages, v)
	if m.pushFunc != nil {
		return m.pushFunc(ctx, v)
	}
	return nil
}

func (m *mockKqPusherClient) Close() error {
	return nil
}

// newTestSendRegisterCodeLogicWithMockMQ 创建带 mock MQ 的测试逻辑
func newTestSendRegisterCodeLogicWithMockMQ(t *testing.T, mockClient *mockKqPusherClient) (*SendRegisterCodeLogic, *miniredis.Miniredis) {
	s := miniredis.RunT(t)
	conf := redis.RedisConf{
		Host: s.Addr(),
		Type: "node",
	}
	r := redis.MustNewRedis(conf)

	ctx := context.Background()
	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			Register: config.Register{
				SendCodeConfig: config.SendCodeConfig{
					ReceiveType:    "email",
					ExpireIn:       300,
					RetryAfter:     60,
					RedisKeyPrefix: "register:code",
				},
			},
		},
		Redis:          r,
		KqPusherClient: mockClient,
	}

	logic := NewSendRegisterCodeLogic(ctx, svcCtx)
	return logic, s
}

func TestSendRegisterCodeLogic_sendToMQ(t *testing.T) {
	t.Run("成功发送消息到MQ", func(t *testing.T) {
		mockClient := &mockKqPusherClient{}
		logic, s := newTestSendRegisterCodeLogicWithMockMQ(t, mockClient)
		defer s.Close()

		email := "test@example.com"
		code := "123456"

		err := logic.sendToMQ(email, code)
		require.NoError(t, err)

		// 验证消息已发送
		require.Len(t, mockClient.messages, 1)

		// 验证消息内容
		var msg types.VerificationCodeMessage
		err = json.Unmarshal([]byte(mockClient.messages[0]), &msg)
		require.NoError(t, err)
		assert.Equal(t, code, msg.Code)
		assert.Equal(t, email, msg.Receiver)
		assert.Equal(t, "email", msg.Type)
		assert.Greater(t, msg.Timestamp, int64(0))
	})

	t.Run("MQ推送失败返回错误", func(t *testing.T) {
		mockClient := &mockKqPusherClient{
			pushFunc: func(ctx context.Context, v string) error {
				return errors.New("mq connection failed")
			},
		}
		logic, s := newTestSendRegisterCodeLogicWithMockMQ(t, mockClient)
		defer s.Close()

		email := "test@example.com"
		code := "123456"

		err := logic.sendToMQ(email, code)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "消息队列推送失败")
	})

	t.Run("验证消息格式正确", func(t *testing.T) {
		mockClient := &mockKqPusherClient{}
		logic, s := newTestSendRegisterCodeLogicWithMockMQ(t, mockClient)
		defer s.Close()

		email := "user@example.com"
		code := "654321"

		err := logic.sendToMQ(email, code)
		require.NoError(t, err)

		// 解析并验证 JSON 格式
		var msg types.VerificationCodeMessage
		err = json.Unmarshal([]byte(mockClient.messages[0]), &msg)
		require.NoError(t, err)

		// 验证所有字段
		assert.Equal(t, code, msg.Code)
		assert.Equal(t, email, msg.Receiver)
		assert.Equal(t, "email", msg.Type)
		assert.NotZero(t, msg.Timestamp)
	})
}

func TestSendRegisterCodeLogic_buildResponse(t *testing.T) {
	t.Run("正确构建响应", func(t *testing.T) {
		logic, s := newTestSendRegisterCodeLogic(t, nil)
		defer s.Close()

		resp := logic.buildResponse()

		require.NotNil(t, resp)
		assert.Equal(t, 60, resp.RetryAfter)
	})

	t.Run("响应包含正确的配置值", func(t *testing.T) {
		// 创建不同配置的 logic
		s := miniredis.RunT(t)
		defer s.Close()

		conf := redis.RedisConf{
			Host: s.Addr(),
			Type: "node",
		}
		r := redis.MustNewRedis(conf)

		ctx := context.Background()
		svcCtx := &svc.ServiceContext{
			Config: config.Config{
				Register: config.Register{
					SendCodeConfig: config.SendCodeConfig{
						ReceiveType:    "email",
						ExpireIn:       300,
						RetryAfter:     120, // 不同的重试时间
						RedisKeyPrefix: "register:code",
					},
				},
			},
			Redis: r,
		}

		logic := NewSendRegisterCodeLogic(ctx, svcCtx)
		resp := logic.buildResponse()

		require.NotNil(t, resp)
		assert.Equal(t, 120, resp.RetryAfter)
	})
}

func TestSendRegisterCodeLogic_generateCode(t *testing.T) {
	t.Run("生成的验证码长度为6位", func(t *testing.T) {
		logic, s := newTestSendRegisterCodeLogic(t, nil)
		defer s.Close()

		code := logic.generateCode()
		assert.Equal(t, 6, len(code), "验证码长度应该为6位")
	})

	t.Run("生成的验证码只包含数字和大写字母", func(t *testing.T) {
		logic, s := newTestSendRegisterCodeLogic(t, nil)
		defer s.Close()

		code := logic.generateCode()
		for _, ch := range code {
			assert.True(t,
				(ch >= '0' && ch <= '9') || (ch >= 'A' && ch <= 'Z'),
				"验证码字符必须是数字或大写字母，实际字符: %c", ch)
		}
	})

	t.Run("多次生成的验证码不相同", func(t *testing.T) {
		logic, s := newTestSendRegisterCodeLogic(t, nil)
		defer s.Close()

		codes := make(map[string]bool)
		for i := 0; i < 10; i++ {
			code := logic.generateCode()
			codes[code] = true
		}
		// 虽然理论上可能重复，但10次生成10个不同验证码的概率极高
		assert.GreaterOrEqual(t, len(codes), 8, "10次生成应该产生至少8个不同的验证码")
	})
}

func TestSendRegisterCodeLogic_saveCodeToRedis(t *testing.T) {
	t.Run("成功保存验证码到Redis", func(t *testing.T) {
		logic, s := newTestSendRegisterCodeLogic(t, nil)
		defer s.Close()

		ctx := context.Background()
		email := "test@example.com"
		code := "123456"

		err := logic.saveCodeToRedis(email, code)
		require.NoError(t, err)

		// 验证数据已保存
		redisKey := logic.buildVerifyKey(email)
		savedCode, err := logic.svcCtx.Redis.HgetCtx(ctx, redisKey, "code")
		require.NoError(t, err)
		assert.Equal(t, code, savedCode)

		savedUsed, err := logic.svcCtx.Redis.HgetCtx(ctx, redisKey, "used")
		require.NoError(t, err)
		assert.Equal(t, "0", savedUsed)
	})

	t.Run("保存的验证码有过期时间", func(t *testing.T) {
		logic, s := newTestSendRegisterCodeLogic(t, nil)
		defer s.Close()

		email := "test@example.com"
		code := "123456"

		err := logic.saveCodeToRedis(email, code)
		require.NoError(t, err)

		// 验证有过期时间（使用 miniredis 检查 TTL）
		redisKey := logic.buildVerifyKey(email)
		ttl := s.TTL(redisKey)
		assert.True(t, ttl > 0, "验证码应该设置过期时间")
		assert.True(t, ttl <= 300*time.Second, "过期时间应该不超过配置的300秒")
	})

	t.Run("覆盖已存在的验证码", func(t *testing.T) {
		logic, s := newTestSendRegisterCodeLogic(t, nil)
		defer s.Close()

		ctx := context.Background()
		email := "test@example.com"

		// 先保存旧验证码
		err := logic.saveCodeToRedis(email, "OLD123")
		require.NoError(t, err)

		// 保存新验证码
		err = logic.saveCodeToRedis(email, "NEW456")
		require.NoError(t, err)

		// 验证新验证码已覆盖旧验证码
		redisKey := logic.buildVerifyKey(email)
		savedCode, err := logic.svcCtx.Redis.HgetCtx(ctx, redisKey, "code")
		require.NoError(t, err)
		assert.Equal(t, "NEW456", savedCode)
	})

	t.Run("不同邮箱的验证码相互独立", func(t *testing.T) {
		logic, s := newTestSendRegisterCodeLogic(t, nil)
		defer s.Close()

		ctx := context.Background()
		email1 := "user1@example.com"
		email2 := "user2@example.com"

		// 为两个邮箱保存不同的验证码
		err := logic.saveCodeToRedis(email1, "CODE11")
		require.NoError(t, err)
		err = logic.saveCodeToRedis(email2, "CODE22")
		require.NoError(t, err)

		// 验证各自的验证码正确
		redisKey1 := logic.buildVerifyKey(email1)
		redisKey2 := logic.buildVerifyKey(email2)

		savedCode1, _ := logic.svcCtx.Redis.HgetCtx(ctx, redisKey1, "code")
		savedCode2, _ := logic.svcCtx.Redis.HgetCtx(ctx, redisKey2, "code")

		assert.Equal(t, "CODE11", savedCode1)
		assert.Equal(t, "CODE22", savedCode2)
	})
}

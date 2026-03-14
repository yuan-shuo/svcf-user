package account_noauth

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/stores/redis"

	"user/internal/config"
	"user/internal/svc"
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

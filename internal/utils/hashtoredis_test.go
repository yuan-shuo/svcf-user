package utils

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/stores/redis"
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

func TestSetHashWithExpire(t *testing.T) {
	r, s := setupTestRedis(t)
	defer s.Close()
	ctx := context.Background()

	t.Run("正常设置Hash并过期", func(t *testing.T) {
		key := "test:hash:1"
		fields := map[string]string{
			"code":  "123456",
			"phone": "13800138000",
		}

		err := SetHashWithExpire(r, ctx, key, fields, 60)
		require.NoError(t, err)

		// 验证字段存在
		code, err := r.HgetCtx(ctx, key, "code")
		require.NoError(t, err)
		assert.Equal(t, "123456", code)

		phone, err := r.HgetCtx(ctx, key, "phone")
		require.NoError(t, err)
		assert.Equal(t, "13800138000", phone)

		// 验证过期时间（miniredis 返回 time.Duration，直接比较）
		ttl := s.TTL(key)
		assert.True(t, ttl > 59*time.Second && ttl <= 60*time.Second,
			"TTL 应在 59-60 秒之间，实际: %v", ttl)
	})

	t.Run("覆盖已存在的Hash", func(t *testing.T) {
		key := "test:hash:2"

		// 先设置旧值
		err := r.HsetCtx(ctx, key, "old", "value")
		require.NoError(t, err)

		// 覆盖新值
		newFields := map[string]string{
			"new": "data",
		}
		err = SetHashWithExpire(r, ctx, key, newFields, 120)
		require.NoError(t, err)

		// 验证新值存在
		val, err := r.HgetCtx(ctx, key, "new")
		require.NoError(t, err)
		assert.Equal(t, "data", val)
	})

	t.Run("空fields应报错或跳过", func(t *testing.T) {
		key := "test:hash:empty"
		err := SetHashWithExpire(r, ctx, key, map[string]string{}, 60)
		// 空 map 导致 HMSet 失败，事务回滚
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Transaction discarded")
	})

	t.Run("过期时间准确", func(t *testing.T) {
		key := "test:hash:ttl"
		fields := map[string]string{"k": "v"}

		err := SetHashWithExpire(r, ctx, key, fields, 10)
		require.NoError(t, err)

		// 快进时间 9 秒
		s.FastForward(9 * time.Second)
		exists, _ := r.ExistsCtx(ctx, key)
		assert.True(t, exists, "9秒后key应仍存在")

		// 再快进 2 秒（总共11秒）
		s.FastForward(2 * time.Second)
		exists, _ = r.ExistsCtx(ctx, key)
		assert.False(t, exists, "11秒后key应已过期")
	})
}

func TestGetHashField(t *testing.T) {
	r, s := setupTestRedis(t)
	defer s.Close()
	ctx := context.Background()

	t.Run("获取存在的字段", func(t *testing.T) {
		key := "test:get:1"
		r.HsetCtx(ctx, key, "field1", "value1")

		val, err := GetHashField(r, ctx, key, "field1")
		require.NoError(t, err)
		assert.Equal(t, "value1", val)
	})

	t.Run("获取不存在的字段返回空和redis.Nil错误", func(t *testing.T) {
		key := "test:get:2"
		r.HsetCtx(ctx, key, "field1", "value1")

		val, err := GetHashField(r, ctx, key, "notexist")
		// go-zero 返回 redis.Nil 错误
		require.Error(t, err)
		assert.True(t, errors.Is(err, redis.Nil) || err.Error() == "redis: nil")
		assert.Empty(t, val)
	})

	t.Run("获取不存在的key返回空和redis.Nil错误", func(t *testing.T) {
		val, err := GetHashField(r, ctx, "test:get:notexist", "field")
		require.Error(t, err)
		assert.True(t, errors.Is(err, redis.Nil) || err.Error() == "redis: nil")
		assert.Empty(t, val)
	})
}

func TestSetHashField(t *testing.T) {
	r, s := setupTestRedis(t)
	defer s.Close()
	ctx := context.Background()

	t.Run("设置新字段", func(t *testing.T) {
		key := "test:set:1"
		err := SetHashField(r, ctx, key, "name", "张三")
		require.NoError(t, err)

		val, _ := r.HgetCtx(ctx, key, "name")
		assert.Equal(t, "张三", val)
	})

	t.Run("覆盖已有字段", func(t *testing.T) {
		key := "test:set:2"
		r.HsetCtx(ctx, key, "count", "1")

		err := SetHashField(r, ctx, key, "count", "100")
		require.NoError(t, err)

		val, _ := r.HgetCtx(ctx, key, "count")
		assert.Equal(t, "100", val)
	})
}

func TestHashExists(t *testing.T) {
	r, s := setupTestRedis(t)
	defer s.Close()
	ctx := context.Background()

	t.Run("key存在", func(t *testing.T) {
		key := "test:exists:1"
		r.HsetCtx(ctx, key, "f", "v")

		exists, err := HashExists(r, ctx, key)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("key不存在", func(t *testing.T) {
		exists, err := HashExists(r, ctx, "test:exists:not")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("key过期后", func(t *testing.T) {
		key := "test:exists:ttl"
		r.HsetCtx(ctx, key, "f", "v")
		r.ExpireCtx(ctx, key, 1)

		s.FastForward(2 * time.Second)

		exists, err := HashExists(r, ctx, key)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestGetAllHash(t *testing.T) {
	r, s := setupTestRedis(t)
	defer s.Close()
	ctx := context.Background()

	t.Run("获取完整Hash", func(t *testing.T) {
		key := "test:all:1"
		expected := map[string]string{
			"a": "1",
			"b": "2",
			"c": "3",
		}

		for k, v := range expected {
			r.HsetCtx(ctx, key, k, v)
		}

		got, err := GetAllHash(r, ctx, key)
		require.NoError(t, err)
		assert.Equal(t, expected, got)
	})

	t.Run("不存在的key返回空map", func(t *testing.T) {
		got, err := GetAllHash(r, ctx, "test:all:notexist")
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

// 集成测试：验证原子性
func TestSetHashWithExpire_Atomic(t *testing.T) {
	r, s := setupTestRedis(t)
	defer s.Close()
	ctx := context.Background()

	t.Run("原子性验证-HMSet和Expire同时成功", func(t *testing.T) {
		key := "test:atomic"
		fields := map[string]string{"k": "v"}

		err := SetHashWithExpire(r, ctx, key, fields, 100)
		require.NoError(t, err)

		// 验证数据存在
		val, _ := r.HgetCtx(ctx, key, "k")
		assert.Equal(t, "v", val)

		// 验证过期存在
		ttl := s.TTL(key)
		assert.True(t, ttl > 98*time.Second && ttl <= 100*time.Second)
	})
}

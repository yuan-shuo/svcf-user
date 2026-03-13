package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

// SetHashWithExpire 原子性地设置 Hash 并设置过期时间
// 使用 TxPipeline 保证原子性
func SetHashWithExpire(rc *redis.Redis, ctx context.Context, key string, fields map[string]string, expireSeconds int) error {
	// 开启事务管道
	pipe, err := rc.TxPipeline()
	if err != nil {
		return fmt.Errorf("创建 TxPipeline 失败: %w", err)
	}

	// 设置 Hash 字段
	pipe.HMSet(ctx, key, fields)

	// 设置过期时间 - 注意：需要将 int 转为 time.Duration
	pipe.Expire(ctx, key, time.Duration(expireSeconds)*time.Second) // 转换类型

	// 执行
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("执行 TxPipeline 失败: %w", err)
	}

	return nil
}

// GetHashField 获取 Hash 的单个字段
func GetHashField(rc *redis.Redis, ctx context.Context, key, field string) (string, error) {
	return rc.HgetCtx(ctx, key, field)
}

// SetHashField 设置 Hash 的单个字段（不修改过期时间）
func SetHashField(rc *redis.Redis, ctx context.Context, key, field, value string) error {
	return rc.HsetCtx(ctx, key, field, value)
}

// HashExists 检查 Hash 是否存在
func HashExists(rc *redis.Redis, ctx context.Context, key string) (bool, error) {
	val, err := rc.ExistsCtx(ctx, key)
	return val, err
}

// GetAllHash 获取整个 Hash（调试用）
func GetAllHash(rc *redis.Redis, ctx context.Context, key string) (map[string]string, error) {
	return rc.HgetallCtx(ctx, key)
}

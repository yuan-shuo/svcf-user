// 令牌桶限流器管理器
package limiter

import (
	"fmt"
	"time"
	"user/internal/config"

	"github.com/zeromicro/go-zero/core/limit"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// TokenLimiterManager 令牌桶限流器管理器
// 统一管理所有 TokenLimiter 实例，支持按 key 复用和自动清理
type TokenLimiterManager struct {
	Config config.Config
	redis  *redis.Redis
	cache  *CleanableCache
}

// NewTokenLimiterManager 创建令牌桶限流器管理器
func NewTokenLimiterManager(c config.Config, rds *redis.Redis) *TokenLimiterManager {
	return &TokenLimiterManager{
		Config: c,
		redis:  rds,
		cache:  NewCleanableCache(30 * time.Minute), // 30 分钟无访问则清理
	}
}

// GetLimiter 获取或创建指定 key 的 TokenLimiter
// 使用 ComputeIfAbsent 确保并发情况下只创建一个限流器实例
func (m *TokenLimiterManager) GetLimiter(prefix, key string, rate, burst int) *limit.TokenLimiter {
	cacheKey := fmt.Sprintf("%s:%s", prefix, key)

	val, err := m.cache.ComputeIfAbsent(cacheKey, func() interface{} {
		return limit.NewTokenLimiter(
			rate,
			burst,
			m.redis,
			fmt.Sprintf("%s:%s", m.Config.RateLimit.RedisKeyPrefix, cacheKey),
		)
	})
	if err != nil {
		logx.Errorf("Failed to create TokenLimiter for key %s: %v", cacheKey, err)
		// 如果创建失败，返回一个新的限流器（不缓存）
		return limit.NewTokenLimiter(
			rate,
			burst,
			m.redis,
			fmt.Sprintf("%s:%s", m.Config.RateLimit.RedisKeyPrefix, cacheKey),
		)
	}
	return val.(*limit.TokenLimiter)
}

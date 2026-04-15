package limiter

import (
	"fmt"
	"user/internal/config"

	"github.com/zeromicro/go-zero/core/limit"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// RefreshTokenLimiter 刷新token限流器（令牌桶）
// 支持按IP动态创建限流器
type RefreshTokenLimiter struct {
	config config.Config
	redis  *redis.Redis
}

// NewRefreshTokenLimiter 创建刷新token限流器管理器
func NewRefreshTokenLimiter(c config.Config, rds *redis.Redis) *RefreshTokenLimiter {
	return &RefreshTokenLimiter{
		config: c,
		redis:  rds,
	}
}

// GetLimiter 根据key获取对应的令牌桶限流器
func (l *RefreshTokenLimiter) GetLimiter(key string) *limit.TokenLimiter {
	return limit.NewTokenLimiter(
		l.config.RateLimit.RefreshToken.Rate,
		l.config.RateLimit.RefreshToken.Burst,
		l.redis,
		fmt.Sprintf("%s:refreshtoken:%s", l.config.RateLimit.RedisKeyPrefix, key),
	)
}

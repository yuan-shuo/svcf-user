package limiter

import (
	"fmt"
	"user/internal/config"

	"github.com/zeromicro/go-zero/core/limit"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	NoAuthKey         = "noauth"
	ChangePasswordKey = "changepassword"
)

func NewPeriodLimiter(periodLimit config.PeriodLimit, rds *redis.Redis, prefix, key string) *limit.PeriodLimit {
	return limit.NewPeriodLimit(periodLimit.Period,
		periodLimit.Quota, // 限流阈值，单位：次
		rds,
		fmt.Sprintf("%s:%s", prefix, key),
	)
}

// 无认证接口限流器
func NewNoAuthPeriodLimiter(c config.Config, rds *redis.Redis) *limit.PeriodLimit {
	return NewPeriodLimiter(c.RateLimit.NoAuth, rds, c.RateLimit.RedisKeyPrefix, NoAuthKey)
}

// 修改密码接口限流器（周期限流）
func NewChangePasswordPeriodLimiter(c config.Config, rds *redis.Redis) *limit.PeriodLimit {
	return NewPeriodLimiter(c.RateLimit.ChangePassword, rds, c.RateLimit.RedisKeyPrefix, ChangePasswordKey)
}

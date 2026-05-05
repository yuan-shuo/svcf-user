// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package middleware

import (
	"context"
	"user/internal/middleware/limiter"

	"github.com/zeromicro/go-zero/core/limit"
	"github.com/zeromicro/go-zero/rest"
)

// NewNoAuthLimitMiddleware 创建无认证接口限流中间件
func NewNoAuthLimitMiddleware(limiterMgr *limiter.PeriodLimiterManager) rest.Middleware {
	return limiter.CreateLimitMiddleware(func(ctx context.Context, key string) (bool, error) {
		periodLimiter := limiterMgr.GetLimiter(
			"noauth",
			key,
			limiterMgr.Config.RateLimit.NoAuth.Period,
			limiterMgr.Config.RateLimit.NoAuth.Quota,
		)
		// TakeCtx 的 key 参数传空字符串，因为 keyPrefix 已经包含了完整的标识
		code, err := periodLimiter.TakeCtx(ctx, "")
		return err == nil && code != limit.OverQuota, err
	})
}

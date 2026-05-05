// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package middleware

import (
	"context"
	"user/internal/middleware/limiter"

	"github.com/zeromicro/go-zero/core/limit"
	"github.com/zeromicro/go-zero/rest"
)

// NewChangePasswordLimitMiddleware 创建修改密码限流中间件
func NewChangePasswordLimitMiddleware(limiterMgr *limiter.PeriodLimiterManager) rest.Middleware {
	return limiter.CreateLimitMiddleware(func(ctx context.Context, key string) (bool, error) {
		periodLimiter := limiterMgr.GetLimiter(
			"changepassword",
			key,
			limiterMgr.Config.RateLimit.ChangePassword.Period,
			limiterMgr.Config.RateLimit.ChangePassword.Quota,
		)
		// TakeCtx 的 key 参数传空字符串，因为 keyPrefix 已经包含了完整的标识
		code, err := periodLimiter.TakeCtx(ctx, "")
		return err == nil && code != limit.OverQuota, err
	})
}

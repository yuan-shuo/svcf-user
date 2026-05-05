// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package middleware

import (
	"context"
	"user/internal/middleware/limiter"

	"github.com/zeromicro/go-zero/rest"
)

// NewRefreshTokenLimitMiddleware 创建刷新token限流中间件
func NewRefreshTokenLimitMiddleware(limiterMgr *limiter.TokenLimiterManager) rest.Middleware {
	return limiter.CreateLimitMiddleware(func(ctx context.Context, key string) (bool, error) {
		tokenLimiter := limiterMgr.GetLimiter(
			"refreshtoken",
			key,
			limiterMgr.Config.RateLimit.RefreshToken.Rate,
			limiterMgr.Config.RateLimit.RefreshToken.Burst,
		)
		return tokenLimiter.AllowCtx(ctx), nil
	})
}

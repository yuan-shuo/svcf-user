// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package middleware

import (
	"encoding/json"
	"net/http"
	"user/internal/errs"
	"user/internal/middleware/limiter"

	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// NewRefreshTokenLimitMiddleware 创建刷新token限流中间件
func NewRefreshTokenLimitMiddleware(l *limiter.RefreshTokenLimiter) rest.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// 使用客户端IP作为限流key，优先从X-Forwarded-For获取但会验证可信度
			key := httpx.GetRemoteAddr(r)

			// 获取对应IP的令牌桶限流器并检查
			if !l.GetLimiter(key).AllowCtx(r.Context()) {
				w.Header().Set("Content-Type", "application/json")

				// 使用统一的错误处理
				e := errs.New(errs.CodeTooManyRequests)
				w.WriteHeader(e.JudgeErrsStatus())

				resp := errs.CodeErrorResponse{
					Code: e.Code,
					Msg:  e.Msg,
				}
				json.NewEncoder(w).Encode(resp)
				return
			}
			next(w, r)
		}
	}
}

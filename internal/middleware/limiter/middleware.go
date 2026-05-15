package limiter

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"user/internal/errs"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// LimitChecker 限流检查函数类型，接收 context 和 key
type LimitChecker func(ctx context.Context, key string) (allowed bool, err error)

// CreateLimitMiddleware 创建通用的限流中间件
// checker: 限流检查函数，接收 context 和 key，返回是否允许通过
func CreateLimitMiddleware(checker LimitChecker) rest.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// 使用客户端IP作为限流key（去除端口号）
			key := getClientIP(r)

			// 执行限流检查，传入请求的 context
			allowed, err := checker(r.Context(), key)
			if err != nil || !allowed {
				// 记录限流日志
				logx.WithContext(r.Context()).Infow("Rate limit triggered",
					logx.Field("ip", key),
					logx.Field("path", r.URL.Path),
					logx.Field("method", r.Method),
					logx.Field("error", err),
				)
				writeLimitError(w)
				return
			}

			next(w, r)
		}
	}
}

// getClientIP 获取客户端 IP 地址（去除端口号）
func getClientIP(r *http.Request) string {
	addr := httpx.GetRemoteAddr(r)
	// 去除端口号
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// 如果无法解析，直接返回原值（可能是 IPv6 或其他格式）
		return addr
	}
	return host
}

// writeLimitError 写入限流错误响应
func writeLimitError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")

	e := errs.New(errs.CodeTooManyRequests)
	w.WriteHeader(e.JudgeErrsStatus())

	resp := errs.CodeErrorResponse{
		Code: e.Code,
		Msg:  e.Msg,
	}
	json.NewEncoder(w).Encode(resp)
}

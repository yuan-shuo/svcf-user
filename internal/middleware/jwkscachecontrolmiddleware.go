// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package middleware

import (
	"fmt"
	"net/http"

	"user/internal/config"
)

type JWKSCacheControlMiddleware struct {
	config config.JWKSConfig
}

func NewJWKSCacheControlMiddleware(cfg config.JWKSConfig) *JWKSCacheControlMiddleware {
	return &JWKSCacheControlMiddleware{
		config: cfg,
	}
}

func (m *JWKSCacheControlMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 设置 Cache-Control 头，告诉网关可以缓存 JWKS
		// public: 可以被任何缓存缓存（包括 CDN、网关等）
		// max-age: 缓存时间（从配置读取，默认 3600 秒）
		maxAge := m.config.CacheControlMaxAge
		if maxAge <= 0 {
			maxAge = 3600 // 默认 1 小时
		}
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))

		next(w, r)
	}
}

// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package middleware

import (
	"context"
	"net/http"

	"user/internal/config"
	"user/internal/utils"
)

// CookieSetter 用于在 logic 和中间件之间传递 Cookie 信息
type CookieSetter struct {
	AccessToken  string
	RefreshToken string
	ClearTokens  bool // 登出时清除
}

// setterKey context key
type setterKey struct{}

// RefreshTokenKey context key for storing refresh token from cookie
type RefreshTokenKey struct{}

// NewCookieSetterMiddleware 创建 Cookie 设置中间件，返回 rest.Middleware 类型
func NewCookieSetterMiddleware(cfg config.Config) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// 创建 Setter 放入 context
			setter := &CookieSetter{}
			ctx := context.WithValue(r.Context(), setterKey{}, setter)

			// 从 Cookie 读取 Refresh Token 并放入 context（供 refreshtoken 接口使用）
		if refreshToken, err := utils.GetTokenFromRequest(r, "refresh_token"); err == nil && refreshToken != "" {
			ctx = context.WithValue(ctx, RefreshTokenKey{}, refreshToken)
		}

			// 执行后续 handler
			next(w, r.WithContext(ctx))

			// 根据 setter 设置 Cookie
			cookieConfig := &utils.CookieConfig{
				Domain:   cfg.Cookie.Domain,
				Secure:   cfg.Cookie.Secure,
				HttpOnly: cfg.Cookie.HttpOnly,
				SameSite: parseSameSite(cfg.Cookie.SameSite),
				Path:     "/",
			}

			if setter.ClearTokens {
				// 登出：清除 Cookie
				utils.ClearTokenCookie(w, "access_token", cookieConfig)
				utils.ClearTokenCookie(w, "refresh_token", cookieConfig)
			} else {
				// 登录/刷新：设置 Cookie
				if setter.AccessToken != "" {
					// 使用 config.Auth.AccessExpire 作为过期时间
					utils.SetTokenCookie(w, "access_token", setter.AccessToken,
						int(cfg.Auth.AccessExpire), cookieConfig)
				}
				if setter.RefreshToken != "" {
					// 使用 config.RefreshExpire 作为过期时间
					utils.SetTokenCookie(w, "refresh_token", setter.RefreshToken,
						int(cfg.RefreshExpire), cookieConfig)
				}
			}
		}
	}
}

// GetCookieSetter 从 context 获取 CookieSetter
func GetCookieSetter(ctx context.Context) *CookieSetter {
	if setter, ok := ctx.Value(setterKey{}).(*CookieSetter); ok {
		return setter
	}
	return nil
}

// GetRefreshTokenFromContext 从 context 获取 Refresh Token（从 Cookie 读取的）
func GetRefreshTokenFromContext(ctx context.Context) (string, bool) {
	if token, ok := ctx.Value(RefreshTokenKey{}).(string); ok && token != "" {
		return token, true
	}
	return "", false
}

// parseSameSite 解析 SameSite 字符串
func parseSameSite(s string) http.SameSite {
	switch s {
	case "strict":
		return http.SameSiteStrictMode
	case "lax":
		return http.SameSiteLaxMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteStrictMode
	}
}

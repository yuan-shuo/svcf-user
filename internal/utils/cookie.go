package utils

import (
	"net/http"
)

// CookieConfig Cookie 配置
type CookieConfig struct {
	Domain   string
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
	Path     string
}

// SetTokenCookie 设置 Token Cookie
func SetTokenCookie(w http.ResponseWriter, name, value string, maxAge int, config *CookieConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   maxAge,
		Path:     config.Path,
		Domain:   config.Domain,
		Secure:   config.Secure,
		HttpOnly: config.HttpOnly,
		SameSite: config.SameSite,
	})
}

// ClearTokenCookie 清除 Token Cookie
func ClearTokenCookie(w http.ResponseWriter, name string, config *CookieConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		MaxAge:   -1,
		Path:     config.Path,
		Domain:   config.Domain,
		Secure:   config.Secure,
		HttpOnly: config.HttpOnly,
		SameSite: config.SameSite,
	})
}

// GetTokenFromRequest 从请求中读取指定名称的 Cookie 值
func GetTokenFromRequest(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

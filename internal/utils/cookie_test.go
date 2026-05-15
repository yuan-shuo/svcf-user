package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetTokenCookie(t *testing.T) {
	tests := []struct {
		name       string
		cookieName string
		value      string
		maxAge     int
		config     *CookieConfig
		expected   map[string]string
	}{
		{
			name:       "设置 access_token cookie",
			cookieName: "access_token",
			value:      "test-access-token",
			maxAge:     3600,
			config: &CookieConfig{
				Domain:   "",
				Secure:   false,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
				Path:     "/",
			},
			expected: map[string]string{
				"Name":     "access_token",
				"Value":    "test-access-token",
				"Path":     "/",
				"Domain":   "",
				"Secure":   "false",
				"HttpOnly": "true",
				"SameSite": "Strict",
			},
		},
		{
			name:       "设置 refresh_token cookie",
			cookieName: "refresh_token",
			value:      "test-refresh-token",
			maxAge:     7200,
			config: &CookieConfig{
				Domain:   "example.com",
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				Path:     "/api",
			},
			expected: map[string]string{
				"Name":     "refresh_token",
				"Value":    "test-refresh-token",
				"Path":     "/api",
				"Domain":   "example.com",
				"Secure":   "true",
				"HttpOnly": "true",
				"SameSite": "Lax",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			SetTokenCookie(recorder, tt.cookieName, tt.value, tt.maxAge, tt.config)

			cookies := recorder.Result().Cookies()
			assert.Len(t, cookies, 1)

			cookie := cookies[0]
			assert.Equal(t, tt.expected["Name"], cookie.Name)
			assert.Equal(t, tt.expected["Value"], cookie.Value)
			assert.Equal(t, tt.maxAge, cookie.MaxAge)
			assert.Equal(t, tt.expected["Path"], cookie.Path)
			assert.Equal(t, tt.expected["Domain"], cookie.Domain)
			assert.Equal(t, tt.config.Secure, cookie.Secure)
			assert.Equal(t, tt.config.HttpOnly, cookie.HttpOnly)
		})
	}
}

func TestClearTokenCookie(t *testing.T) {
	tests := []struct {
		name       string
		cookieName string
		config     *CookieConfig
	}{
		{
			name:       "清除 access_token cookie",
			cookieName: "access_token",
			config: &CookieConfig{
				Domain:   "",
				Secure:   false,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
				Path:     "/",
			},
		},
		{
			name:       "清除 refresh_token cookie",
			cookieName: "refresh_token",
			config: &CookieConfig{
				Domain:   "example.com",
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				Path:     "/api",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ClearTokenCookie(recorder, tt.cookieName, tt.config)

			cookies := recorder.Result().Cookies()
			assert.Len(t, cookies, 1)

			cookie := cookies[0]
			assert.Equal(t, tt.cookieName, cookie.Name)
			assert.Equal(t, "", cookie.Value)
			assert.Equal(t, -1, cookie.MaxAge) // MaxAge -1 表示删除 cookie
			assert.Equal(t, tt.config.Path, cookie.Path)
			assert.Equal(t, tt.config.Domain, cookie.Domain)
		})
	}
}

func TestCookieConfig_SecurityAttributes(t *testing.T) {
	tests := []struct {
		name     string
		config   *CookieConfig
		expected struct {
			secure   bool
			httpOnly bool
			sameSite http.SameSite
		}
	}{
		{
			name: "生产环境安全配置",
			config: &CookieConfig{
				Domain:   "",
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
				Path:     "/",
			},
			expected: struct {
				secure   bool
				httpOnly bool
				sameSite http.SameSite
			}{
				secure:   true,
				httpOnly: true,
				sameSite: http.SameSiteStrictMode,
			},
		},
		{
			name: "开发环境配置",
			config: &CookieConfig{
				Domain:   "",
				Secure:   false,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				Path:     "/",
			},
			expected: struct {
				secure   bool
				httpOnly bool
				sameSite http.SameSite
			}{
				secure:   false,
				httpOnly: true,
				sameSite: http.SameSiteLaxMode,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			SetTokenCookie(recorder, "test_cookie", "test_value", 3600, tt.config)

			cookies := recorder.Result().Cookies()
			assert.Len(t, cookies, 1)

			cookie := cookies[0]
			assert.Equal(t, tt.expected.secure, cookie.Secure)
			assert.Equal(t, tt.expected.httpOnly, cookie.HttpOnly)
			assert.Equal(t, tt.expected.sameSite, cookie.SameSite)
		})
	}
}

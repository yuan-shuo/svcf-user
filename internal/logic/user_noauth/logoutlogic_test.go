package user_noauth

import (
	"context"
	"testing"

	"user/internal/config"
	"user/internal/middleware"
	"user/internal/mock"
	"user/internal/svc"
	"user/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/logx"
)

func TestLogoutLogic_Logout_Success(t *testing.T) {
	ctx := context.Background()

	// 创建带有 CookieSetter 的 context
	setter := &middleware.CookieSetter{}
	ctx = context.WithValue(ctx, struct{}{}, setter)

	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			Cookie: config.CookieConfig{
				Domain:   "",
				Secure:   false,
				HttpOnly: true,
				SameSite: "strict",
			},
		},
		Metrics: mock.GetTestMetrics(),
	}

	logic := NewLogoutLogic(ctx, svcCtx)
	req := &types.LogoutReq{}

	resp, err := logic.Logout(req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestLogoutLogic_Logout_WithContextValue(t *testing.T) {
	logx.Disable()

	// 测试 context 中有 CookieSetter 的情况
	t.Run("with cookie setter in context", func(t *testing.T) {
		ctx := context.Background()
		setter := &middleware.CookieSetter{}
		ctx = context.WithValue(ctx, middleware.CookieSetter{}, setter)

		svcCtx := &svc.ServiceContext{
			Config:  config.Config{},
			Metrics: mock.GetTestMetrics(),
		}

		logic := NewLogoutLogic(ctx, svcCtx)
		req := &types.LogoutReq{}

		resp, err := logic.Logout(req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		// 注意：由于 GetCookieSetter 使用的是私有 setterKey，
		// 实际逻辑中无法通过 middleware.CookieSetter{} 获取
		// 这里只是验证代码不会 panic
	})

	// 测试 context 中没有 CookieSetter 的情况
	t.Run("without cookie setter in context", func(t *testing.T) {
		ctx := context.Background()

		svcCtx := &svc.ServiceContext{
			Config:  config.Config{},
			Metrics: mock.GetTestMetrics(),
		}

		logic := NewLogoutLogic(ctx, svcCtx)
		req := &types.LogoutReq{}

		resp, err := logic.Logout(req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

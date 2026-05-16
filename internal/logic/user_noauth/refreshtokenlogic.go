// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user_noauth

import (
	"context"

	"user/internal/errs"
	"user/internal/logic/userutils"
	"user/internal/metrics"
	"user/internal/middleware"
	"user/internal/svc"
	"user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RefreshTokenLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRefreshTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshTokenLogic {
	return &RefreshTokenLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RefreshTokenLogic) RefreshToken(req *types.RefreshTokenReq) (resp *types.RefreshTokenResp, err error) {
	// 从 context 获取 Refresh Token（由中间件从 Cookie 读取）
	refreshToken, ok := middleware.GetRefreshTokenFromContext(l.ctx)
	if !ok || refreshToken == "" {
		l.svcCtx.Metrics.TokenRefreshTotal.Inc(metrics.TokenRefreshTotalStatusFailedInvalid)
		return nil, errs.New(errs.CodeUnauthorized)
	}

	// 使用 KeyManager 获取用户实例（支持 RS256）
	user, err := userutils.GetUserByRefreshTokenWithKeyManager(l.ctx, l.svcCtx.UsersModel, refreshToken, l.svcCtx.KeyManager)
	if err != nil {
		// 根据错误类型判断具体失败原因
		if codeErr, ok := errs.IsCodeError(err); ok {
			switch codeErr.Code {
			case errs.CodeInvalidToken:
				l.svcCtx.Metrics.TokenRefreshTotal.Inc(metrics.TokenRefreshTotalStatusFailedInvalid)
			case errs.CodeUnauthorized:
				l.svcCtx.Metrics.TokenRefreshTotal.Inc(metrics.TokenRefreshTotalStatusFailedExpired)
			case errs.CodeUserNotFound:
				l.svcCtx.Metrics.TokenRefreshTotal.Inc(metrics.TokenRefreshTotalStatusFailedUserNotFound)
			default:
				l.svcCtx.Metrics.TokenRefreshTotal.Inc(metrics.TokenRefreshTotalStatusFailedInvalid)
			}
		} else {
			l.svcCtx.Metrics.TokenRefreshTotal.Inc(metrics.TokenRefreshTotalStatusFailedInvalid)
		}
		return nil, err
	}

	// 使用 RSA 重新签发新 token（RS256）
	newAccess, err := userutils.GenerateAccessTokenWithKeyManager(l.ctx, l.svcCtx.KeyManager, user)
	if err != nil {
		l.svcCtx.Metrics.TokenRefreshTotal.Inc(metrics.TokenRefreshTotalStatusFailedInvalid)
		return nil, err
	}
	newRefresh, err := userutils.GenerateRefreshTokenWithKeyManager(l.ctx, l.svcCtx.KeyManager, l.svcCtx.Config.RefreshExpire, user)
	if err != nil {
		l.svcCtx.Metrics.TokenRefreshTotal.Inc(metrics.TokenRefreshTotalStatusFailedInvalid)
		return nil, err
	}

	// 设置 Cookie（通过中间件）
	if setter := middleware.GetCookieSetter(l.ctx); setter != nil {
		setter.AccessToken = newAccess
		setter.RefreshToken = newRefresh
	}

	// 返回响应
	l.svcCtx.Metrics.TokenRefreshTotal.Inc(metrics.TokenRefreshTotalStatusSuccess)
	return &types.RefreshTokenResp{
		ExpiresIn: l.svcCtx.Config.Auth.AccessExpire,
	}, nil
}

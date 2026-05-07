// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user_noauth

import (
	"context"

	"user/internal/logic/userutils"
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
	// 获取用户实例
	user, err := userutils.GetUserByRefreshToken(l.ctx, l.svcCtx.UsersModel, req.RefreshToken, l.svcCtx.Config.RefreshSecret)
	if err != nil {
		l.svcCtx.Metrics.AccountNoauth.TokenRefreshesTotal.Inc("fail")
		return nil, err
	}

	// 重新签发新token
	newAccess, err := userutils.GenerateAccessToken(l.svcCtx.Config, user)
	if err != nil {
		l.svcCtx.Metrics.AccountNoauth.TokenRefreshesTotal.Inc("fail")
		return nil, err
	}
	newRefresh, err := userutils.GenerateRefreshToken(l.svcCtx.Config, user)
	if err != nil {
		l.svcCtx.Metrics.AccountNoauth.TokenRefreshesTotal.Inc("fail")
		return nil, err
	}

	// 返回响应
	l.svcCtx.Metrics.AccountNoauth.TokenRefreshesTotal.Inc("success")
	return &types.RefreshTokenResp{
		AccessToken:  newAccess,
		RefreshToken: newRefresh,
		ExpiresIn:    l.svcCtx.Config.Auth.AccessExpire,
	}, nil
}

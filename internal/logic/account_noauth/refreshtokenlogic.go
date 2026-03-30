// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package account_noauth

import (
	"context"

	"user/internal/logic/accutil"
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
	// // 校验rt签名
	// claims, err := accutil.GetClaimsByRefreshToken(l.ctx)
	// if err != nil {
	// 	return nil, err
	// }
	// // 确认type是rt
	// if err := accutil.IsTokenTypeEqualToRefreshToken(claims); err != nil {
	// 	return nil, err
	// }

	// 获取用户实例
	user, err := accutil.GetUserByRefreshToken(l.ctx, l.svcCtx, req.RefreshToken)
	if err != nil {
		return nil, err
	}

	// 重新签发新token
	newAccess, err := accutil.GenerateAccessToken(l.svcCtx.Config, user)
	if err != nil {
		return nil, err
	}
	newRefresh, err := accutil.GenerateRefreshToken(l.svcCtx.Config, user)
	if err != nil {
		return nil, err
	}

	// 返回响应
	return &types.RefreshTokenResp{
		AccessToken:  newAccess,
		RefreshToken: newRefresh,
		ExpiresIn:    l.svcCtx.Config.Auth.AccessExpire,
	}, nil
}

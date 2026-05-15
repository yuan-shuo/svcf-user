// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user_noauth

import (
	"context"

	"user/internal/middleware"
	"user/internal/svc"
	"user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LogoutLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LogoutLogic) Logout(req *types.LogoutReq) (resp *types.LogoutResp, err error) {
	// 设置清除 Cookie（通过中间件）
	if setter := middleware.GetCookieSetter(l.ctx); setter != nil {
		setter.ClearTokens = true
	}

	return &types.LogoutResp{}, nil
}

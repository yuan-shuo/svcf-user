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

type ResetPasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewResetPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetPasswordLogic {
	return &ResetPasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ResetPasswordLogic) ResetPassword(req *types.ResetPasswordReq) (resp *types.ResetPasswordResp, err error) {
	// 所需验证码类型 - 重置密码
	codeType := l.svcCtx.Config.VerifyCodeConfig.Type.ResetPassword

	// 检查验证码是否属于对应邮箱以及是否正确
	if err := accutil.VerifyEmailAndCodeInRedis(l.ctx, l.svcCtx, req.Email, req.Code, codeType); err != nil {
		l.svcCtx.Metrics.AccountNoauth.PasswordResetsTotal.Inc("fail")
		return nil, err
	}

	// 重置用户密码
	if err := accutil.ResetUserPasswordByEmail(l.ctx, l.svcCtx, req.Email, req.Password); err != nil {
		l.svcCtx.Metrics.AccountNoauth.PasswordResetsTotal.Inc("fail")
		return nil, err
	}

	// 标记已被使用
	accutil.MarkCodeAsUsed(l.ctx, l.svcCtx, req.Email, codeType)

	l.svcCtx.Metrics.AccountNoauth.PasswordResetsTotal.Inc("success")
	return
}

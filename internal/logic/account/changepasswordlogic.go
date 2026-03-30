// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package account

import (
	"context"

	"user/internal/logic/accutil"
	"user/internal/svc"
	"user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ChangePasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewChangePasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangePasswordLogic {
	return &ChangePasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ChangePasswordLogic) ChangePassword(req *types.ChangePasswordReq) (resp *types.ChangePasswordResp, err error) {
	// todo: add your logic here and delete this line
	// 所需验证码类型 - 修改密码
	codeType := l.svcCtx.Config.VerifyCodeConfig.Type.ChangePassword

	email, err := accutil.GetEmailByJwtCtx(l.ctx)
	if err != nil {
		return nil, err
	}
	// 校验验证码是否属于用户对应邮箱且正确
	if err := accutil.VerifyEmailAndCodeInRedis(l.ctx, l.svcCtx, email, req.Code, codeType); err != nil {
		return nil, err
	}

	// 先获取用户实例
	user, err := accutil.GetUserByAccessJwtCtx(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}

	// 验证旧密码是否正确
	if err := accutil.VerifyPasswordWithOldPasswordMismatchErrHint(user.PasswordHash, req.OldPassword, user.Email); err != nil {
		return nil, err
	}

	// 重置用户密码
	if err := accutil.ResetUserPassword(l.ctx, l.svcCtx, user, req.NewPassword); err != nil {
		return nil, err
	}

	// 标记已被使用
	accutil.MarkCodeAsUsed(l.ctx, l.svcCtx, email, codeType)

	return
}

// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"user/internal/errs"
	"user/internal/logic/userutils"
	"user/internal/metrics"
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
	// 所需验证码类型 - 修改密码
	codeType := l.svcCtx.Config.VerifyCodeConfig.Type.ChangePassword

	email, err := userutils.GetEmailByJwtCtx(l.ctx)
	if err != nil {
		l.svcCtx.Metrics.PasswordChangesTotal.Inc(metrics.PasswordChangesTotalActionChange, metrics.PasswordChangesTotalStatusFailedOldPassword)
		return nil, err
	}
	// 校验验证码是否属于用户对应邮箱且正确
	if err := userutils.VerifyEmailAndCodeInRedis(l.ctx, l.svcCtx.Redis, email, req.Code, codeType); err != nil {
		l.svcCtx.Metrics.PasswordChangesTotal.Inc(metrics.PasswordChangesTotalActionChange, metrics.PasswordChangesTotalStatusFailedOldPassword)
		return nil, err
	}

	// 先获取用户实例
	user, err := userutils.GetUserByAccessJwtCtx(l.ctx, l.svcCtx.UsersModel)
	if err != nil {
		l.svcCtx.Metrics.PasswordChangesTotal.Inc(metrics.PasswordChangesTotalActionChange, metrics.PasswordChangesTotalStatusFailedOldPassword)
		return nil, err
	}

	// 验证旧密码是否正确
	if err := userutils.VerifyPasswordWithOldPasswordMismatchErrHint(l.ctx, user.PasswordHash, req.OldPassword, user.Email); err != nil {
		l.svcCtx.Metrics.PasswordChangesTotal.Inc(metrics.PasswordChangesTotalActionChange, metrics.PasswordChangesTotalStatusFailedOldPassword)
		return nil, err
	}

	// 重置用户密码
	if err := userutils.ResetUserPassword(l.ctx, l.svcCtx.UsersModel, user, req.NewPassword, l.svcCtx.Config.BcryptCost); err != nil {
		// 根据错误类型判断具体失败原因
		if codeErr, ok := errs.IsCodeError(err); ok {
			switch codeErr.Code {
			case errs.CodePasswordSameAsOld:
				l.svcCtx.Metrics.PasswordChangesTotal.Inc(metrics.PasswordChangesTotalActionChange, metrics.PasswordChangesTotalStatusFailedSame)
			case errs.CodeWeakPassword:
				l.svcCtx.Metrics.PasswordChangesTotal.Inc(metrics.PasswordChangesTotalActionChange, metrics.PasswordChangesTotalStatusFailedWeak)
			default:
				l.svcCtx.Metrics.PasswordChangesTotal.Inc(metrics.PasswordChangesTotalActionChange, metrics.PasswordChangesTotalStatusFailedOldPassword)
			}
		} else {
			l.svcCtx.Metrics.PasswordChangesTotal.Inc(metrics.PasswordChangesTotalActionChange, metrics.PasswordChangesTotalStatusFailedOldPassword)
		}
		return nil, err
	}

	// 标记已被使用
	userutils.MarkCodeAsUsed(l.ctx, l.svcCtx.Redis, email, codeType)

	l.svcCtx.Metrics.PasswordChangesTotal.Inc(metrics.PasswordChangesTotalActionChange, metrics.PasswordChangesTotalStatusSuccess)
	return
}

// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user_noauth

import (
	"context"

	"user/internal/errs"
	"user/internal/logic/userutils"
	"user/internal/svc"
	"user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginResp, err error) {
	// 1. 根据邮箱获取用户
	user, err := userutils.GetUserByEmail(l.ctx, l.svcCtx.UsersModel, req.Email)
	if err != nil {
		l.svcCtx.Metrics.AccountNoauth.LoginsTotal.Inc("fail")
		// 登录时将"用户不存在"错误转换为"用户不存在或密码不正确"，避免暴露用户是否存在
		if codeErr, ok := errs.IsCodeError(err); ok && codeErr.Code == errs.CodeUserNotFound {
			return nil, errs.New(errs.CodeUserNotExistOrPasswordIncorrect)
		}
		return nil, err
	}

	// 2. 校验密码
	if err := userutils.VerifyPasswordWithVagueMismatchErrHint(user.PasswordHash, req.Password, req.Email); err != nil {
		l.svcCtx.Metrics.AccountNoauth.LoginsTotal.Inc("fail")
		return nil, err
	}

	// 3. 签发 accessToken
	accessToken, err := userutils.GenerateAccessToken(l.svcCtx.Config, user)
	if err != nil {
		l.svcCtx.Metrics.AccountNoauth.LoginsTotal.Inc("fail")
		return nil, err
	}

	// 4. 签发 refreshToken
	var refreshToken string
	if req.RememberMe {
		// 仅在用户主动选择 "记住我" 时提供RT
		refreshToken, err = userutils.GenerateRefreshToken(l.svcCtx.Config, user)
		if err != nil {
			l.svcCtx.Metrics.AccountNoauth.LoginsTotal.Inc("fail")
			return nil, err
		}
	}

	// 5. 构建响应
	l.svcCtx.Metrics.AccountNoauth.LoginsTotal.Inc("success")
	return &types.LoginResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    l.svcCtx.Config.Auth.AccessExpire,
	}, nil
}

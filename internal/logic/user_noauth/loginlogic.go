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
		// 登录时将"用户不存在"错误转换为"用户不存在或密码不正确"，避免暴露用户是否存在
		if codeErr, ok := errs.IsCodeError(err); ok && codeErr.Code == errs.CodeUserNotFound {
			l.svcCtx.Metrics.UserLoginsTotal.Inc(metrics.UserLoginsTotalSourceWeb, metrics.UserLoginsTotalStatusFailedNotFound)
			return nil, errs.New(errs.CodeUserNotExistOrPasswordIncorrect)
		}
		l.svcCtx.Metrics.UserLoginsTotal.Inc(metrics.UserLoginsTotalSourceWeb, metrics.UserLoginsTotalStatusFailedNotFound)
		return nil, err
	}

	// 2. 校验密码
	if err := userutils.VerifyPasswordWithVagueMismatchErrHint(l.ctx, user.PasswordHash, req.Password, req.Email); err != nil {
		l.svcCtx.Metrics.UserLoginsTotal.Inc(metrics.UserLoginsTotalSourceWeb, metrics.UserLoginsTotalStatusFailedPassword)
		return nil, err
	}

	// 3. 签发 accessToken
	accessToken, err := userutils.GenerateAccessToken(l.ctx, l.svcCtx.Config, user)
	if err != nil {
		l.svcCtx.Metrics.UserLoginsTotal.Inc(metrics.UserLoginsTotalSourceWeb, metrics.UserLoginsTotalStatusFailedPassword)
		return nil, err
	}

	// 4. 签发 refreshToken
	var refreshToken string
	if req.RememberMe {
		// 仅在用户主动选择 "记住我" 时提供RT
		refreshToken, err = userutils.GenerateRefreshToken(l.ctx, l.svcCtx.Config, user)
		if err != nil {
			l.svcCtx.Metrics.UserLoginsTotal.Inc(metrics.UserLoginsTotalSourceWeb, metrics.UserLoginsTotalStatusFailedPassword)
			return nil, err
		}
	}

	// 5. 设置 Cookie（通过中间件）
	if setter := middleware.GetCookieSetter(l.ctx); setter != nil {
		setter.AccessToken = accessToken
		if refreshToken != "" {
			setter.RefreshToken = refreshToken
		}
	}

	// 6. 构建响应
	l.svcCtx.Metrics.UserLoginsTotal.Inc(metrics.UserLoginsTotalSourceWeb, metrics.UserLoginsTotalStatusSuccess)
	return &types.LoginResp{
		ExpiresIn: l.svcCtx.Config.Auth.AccessExpire,
	}, nil
}

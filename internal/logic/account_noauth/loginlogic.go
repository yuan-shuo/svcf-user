// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package account_noauth

import (
	"context"
	"errors"

	"user/internal/errs"
	"user/internal/logic/accutil"
	"user/internal/model"
	"user/internal/svc"
	"user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
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
	user, err := l.getUserByEmail(req.Email)
	if err != nil {
		l.svcCtx.Metrics.AccountNoauth.LoginsTotal.Inc("fail")
		return nil, err
	}

	// 2. 校验密码
	if err := accutil.VerifyPasswordWithVagueMismatchErrHint(user.PasswordHash, req.Password, req.Email); err != nil {
		l.svcCtx.Metrics.AccountNoauth.LoginsTotal.Inc("fail")
		return nil, err
	}

	// 3. 签发 accessToken
	accessToken, err := accutil.GenerateAccessToken(l.svcCtx.Config, user)
	if err != nil {
		l.svcCtx.Metrics.AccountNoauth.LoginsTotal.Inc("fail")
		return nil, err
	}

	// 4. 签发 refreshToken
	var refreshToken string
	if req.RememberMe {
		// 仅在用户主动选择 "记住我" 时提供RT
		refreshToken, err = accutil.GenerateRefreshToken(l.svcCtx.Config, user)
		if err != nil {
			l.svcCtx.Metrics.AccountNoauth.LoginsTotal.Inc("fail")
			return nil, err
		}
	}

	// 5. 构建响应
	l.svcCtx.Metrics.AccountNoauth.LoginsTotal.Inc("success")
	return l.buildLoginResponse(accessToken, refreshToken), nil
}

// 1. 用户查询模块
func (l *LoginLogic) getUserByEmail(email string) (*model.Users, error) {
	user, err := l.svcCtx.UsersModel.FindOneByEmail(l.ctx, email)
	if err != nil {
		if errors.Is(err, sqlx.ErrNotFound) {
			return nil, errs.New(errs.CodeUserNotExistOrPasswordIncorrect)
		}
		logx.Errorf("查询用户信息失败, email=%s, err=%v", email, err)
		return nil, errs.New(errs.CodeInternalError)
	}
	return user, nil
}

// // 2. 密码校验模块
// func (l *LoginLogic) verifyPassword(hashedPassword, password, email string) error {
// 	if err := utils.ComparePassword(hashedPassword, password); err != nil {
// 		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
// 			return errs.New(errs.CodeUserNotExistOrPasswordIncorrect)
// 		}
// 		logx.Errorf("用户登录密码校验失败, email=%s, err=%v", email, err)
// 		return errs.New(errs.CodeInternalError)
// 	}
// 	return nil
// }

// // 3. 签发 accessToken
// func (l *LoginLogic) generateAccessToken(user *model.Users) (string, error) {
// 	accessToken, err := utils.GenerateAccessToken(
// 		l.svcCtx.Config.Auth.AccessSecret,
// 		l.svcCtx.Config.Auth.AccessExpire,
// 		user.SnowflakeId,
// 		user.Nickname,
// 		user.Email,
// 	)
// 	if err != nil {
// 		logx.Errorf("签发 accessToken 失败, email=%s, err=%v", user.Email, err)
// 		return "", errs.New(errs.CodeInternalError)
// 	}
// 	return accessToken, nil
// }

// // 4. 签发 refreshToken
// func (l *LoginLogic) generateRefreshToken(user *model.Users) (string, error) {
// 	refreshToken, err := utils.GenerateRefreshToken(
// 		l.svcCtx.Config.RefreshSecret,
// 		l.svcCtx.Config.RefreshExpire,
// 		user.SnowflakeId,
// 	)
// 	if err != nil {
// 		logx.Errorf("签发 refreshToken 失败, email=%s, err=%v", user.Email, err)
// 		return "", errs.New(errs.CodeInternalError)
// 	}
// 	return refreshToken, nil
// }

// 5. 响应构建模块
func (l *LoginLogic) buildLoginResponse(accessToken, refreshToken string) *types.LoginResp {
	resp := &types.LoginResp{
		AccessToken: accessToken,
		ExpiresIn:   l.svcCtx.Config.Auth.AccessExpire,
	}

	// 只有 refreshToken 非空时才返回
	if refreshToken != "" {
		resp.RefreshToken = refreshToken
	}

	return resp
}

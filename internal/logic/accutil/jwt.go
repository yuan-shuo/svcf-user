package accutil

import (
	"context"
	"user/internal/config"
	"user/internal/errs"
	"user/internal/model"
	"user/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

// GetEmailByJwtCtx 从上下文获取用户邮箱
func GetEmailByJwtCtx(ctx context.Context) (string, error) {
	email, err := utils.GetEmailByJwt(ctx)
	if err != nil {
		logx.Errorf("从JWT中提取用户邮箱失败, err=%v", err)
		return "", errs.New(errs.CodeInternalError)
	}
	return email, nil
}

// 签发 accessToken
func GenerateAccessToken(c config.Config, user *model.Users) (string, error) {
	accessToken, err := utils.GenerateAccessToken(
		c.Auth.AccessSecret,
		c.Auth.AccessExpire,
		user.SnowflakeId,
		user.Nickname,
		user.Email,
	)
	if err != nil {
		logx.Errorf("签发 accessToken 失败, email=%s, err=%v", user.Email, err)
		return "", errs.New(errs.CodeInternalError)
	}
	return accessToken, nil
}

// 签发 refreshToken
func GenerateRefreshToken(c config.Config, user *model.Users) (string, error) {
	refreshToken, err := utils.GenerateRefreshToken(
		c.RefreshSecret,
		c.RefreshExpire,
		user.SnowflakeId,
	)
	if err != nil {
		logx.Errorf("签发 refreshToken 失败, email=%s, err=%v", user.Email, err)
		return "", errs.New(errs.CodeInternalError)
	}
	return refreshToken, nil
}

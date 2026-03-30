package accutil

import (
	"context"
	"user/internal/config"
	"user/internal/errs"
	"user/internal/model"
	"user/internal/svc"
	"user/internal/utils"

	"github.com/golang-jwt/jwt/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

// GetUserByClaims 从 claims 中获取用户实例
func GetUserByClaims(ctx context.Context, svcCtx *svc.ServiceContext, claims jwt.MapClaims) (*model.Users, error) {
	uid, err := utils.GetUidFromClaims(claims)
	if err != nil {
		logx.Errorf("从 claims 中提取用户ID失败, err=%v", err)
		return nil, errs.New(errs.CodeInternalError)
	}
	user, err := svcCtx.UsersModel.FindOneBySnowflakeId(ctx, uid)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, errs.New(errs.CodeUserNotFound)
		}
		logx.Errorf("基于UID获取用户实例失败, uid=%d, err=%v", uid, err)
		return nil, errs.New(errs.CodeInternalError)
	}
	return user, nil
}

// GetClaimsByJWT 从 JWT 中解析 claims
// 如果 token 无效，返回 err=CodeInvalidToken
func GetClaimsByJWT(tokenString, secret string) (jwt.MapClaims, error) {
	claims, err := utils.ParseToken(tokenString, secret)
	if err != nil {
		logx.Errorf("从 JWT 中解析 claims 失败, err=%v", err)
		// token 解析错误（包括格式错误、过期、签名无效等）都返回 CodeInvalidToken
		return nil, errs.New(errs.CodeInvalidToken)
	}
	return claims, nil
}

// 校验rt是否正确
func IsTokenTypeEqualToRefreshToken(claims jwt.MapClaims) error {
	err := utils.IsRefreshToken(claims)
	if err != nil {
		logx.Errorf("校验 JWT.tokenType 是否为 refreshToken 失败, err=%v", err)
		return errs.New(errs.CodeInvalidToken)
	}
	return nil
}

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

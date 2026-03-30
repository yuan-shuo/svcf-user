package accutil

import (
	"context"
	"user/internal/config"
	"user/internal/errs"
	"user/internal/model"
	"user/internal/svc"
	"user/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

// GetUserByAccessTokenClaims 从 accessToken claims 中获取用户实例
func GetUserByAccessTokenClaims(ctx context.Context, svcCtx *svc.ServiceContext) (*model.Users, error) {
	uid, err := utils.UIDFromAccessToken(ctx)
	if err != nil {
		logx.Errorf("从 accessToken claims 中提取用户ID失败, err=%v", err)
		return nil, errs.New(errs.CodeInternalError)
	}

	return GetUserByUid(ctx, svcCtx, uid)
}

// GetUserByRefreshTokenClaims 从 refreshToken claims 中获取用户实例
func GetUserByRefreshTokenClaims(ctx context.Context, svcCtx *svc.ServiceContext) (*model.Users, error) {
	uid, err := utils.UIDFromRefreshToken(ctx)
	if err != nil {
		logx.Errorf("从 refreshToken claims 中提取用户ID失败, err=%v", err)
		return nil, errs.New(errs.CodeInternalError)
	}

	return GetUserByUid(ctx, svcCtx, uid)
}

func GetUserByRefreshToken(ctx context.Context, svcCtx *svc.ServiceContext, rtBase64 string) (*model.Users, error) {
	rt, err := utils.ParseRefreshToken(rtBase64, svcCtx.Config.RefreshSecret)
	if err != nil {
		logx.Errorf("从 refreshToken 中提取用户ID失败, err=%v", err)
		return nil, errs.New(errs.CodeInvalidToken)
	}
	uid, err := rt.GetUID()
	if err != nil {
		logx.Errorf("从 refreshToken 中提取用户ID失败, err=%v", err)
		return nil, errs.New(errs.CodeInvalidToken)
	}
	return GetUserByUid(ctx, svcCtx, uid)
}

// GetAccessTokenClaimsByJWT 从 JWT 中解析 accessToken claims
func GetAccessTokenClaimsByJWT(tokenString, secret string) (*utils.AccessToken, error) {
	accessToken, err := utils.ParseAccessToken(tokenString, secret)
	if err != nil {
		logx.Errorf("从 JWT 中解析 access claims 失败, err=%v", err)
		// token 解析错误（包括格式错误、过期、签名无效等）都返回 CodeInvalidToken
		return nil, errs.New(errs.CodeInvalidToken)
	}
	return accessToken, nil
}

// GetRefreshTokenClaimsByJWT 从 JWT 中解析 refreshToken claims
func GetRefreshTokenClaimsByJWT(tokenString, secret string) (*utils.RefreshToken, error) {
	refreshToken, err := utils.ParseRefreshToken(tokenString, secret)
	if err != nil {
		logx.Errorf("从 JWT 中解析 refresh claims 失败, err=%v", err)
		// token 解析错误（包括格式错误、过期、签名无效等）都返回 CodeInvalidToken
		return nil, errs.New(errs.CodeInvalidToken)
	}
	return refreshToken, nil
}

// // 校验rt是否正确
// func IsTokenTypeEqualToRefreshToken(claims jwt.MapClaims) error {
// 	err := utils.IsRefreshToken(claims)
// 	if err != nil {
// 		logx.Errorf("校验 JWT.tokenType 是否为 refreshToken 失败, err=%v", err)
// 		return errs.New(errs.CodeInvalidToken)
// 	}
// 	return nil
// }

// GetEmailByJwtCtx 从上下文获取用户邮箱
func GetEmailByJwtCtx(ctx context.Context) (string, error) {
	email, err := utils.GetEmailByAccessToken(ctx)
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

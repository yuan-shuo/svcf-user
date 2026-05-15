package userutils

import (
	"context"
	"user/internal/config"
	"user/internal/errs"
	"user/internal/logger"
	"user/internal/model"
	"user/internal/utils"
)

// GetUserByAccessTokenClaims 从 accessToken claims 中获取用户实例
func GetUserByAccessTokenClaims(ctx context.Context, usersModel model.UsersModel) (*model.Users, error) {
	uid, err := utils.UIDFromAccessToken(ctx)
	if err != nil {
		// logx.Errorf("从 accessToken claims 中提取用户ID失败, err=%v", err) // 旧写法是全局的，没 trace 和 span 所以弃用
		// logc.Errorw(ctx, "从 accessToken claims 中提取用户ID失败", logger.WErrorMsg(err.Error())) // 旧方法没有自动排序，工具更新后可以基于Methods自动排序字段了例如 Errors, Infos, Slows, Debugs
		logger.L(ctx, "从 accessToken claims 中提取用户ID失败").WErrorMsg(err.Error()).Errors()
		return nil, errs.New(errs.CodeInternalError)
	}

	return GetUserByUid(ctx, usersModel, uid)
}

// GetUserByRefreshTokenClaims 从 refreshToken claims 中获取用户实例
func GetUserByRefreshTokenClaims(ctx context.Context, usersModel model.UsersModel) (*model.Users, error) {
	uid, err := utils.UIDFromRefreshToken(ctx)
	if err != nil {
		// logx.Errorf("从 refreshToken claims 中提取用户ID失败, err=%v", err)
		// logc.Errorw(ctx, "从 refreshToken claims 中提取用户ID失败", logger.WErrorMsg(err.Error()))
		logger.L(ctx, "从 refreshToken claims 中提取用户ID失败").WErrorMsg(err.Error()).Errors()
		return nil, errs.New(errs.CodeInternalError)
	}

	return GetUserByUid(ctx, usersModel, uid)
}

func GetUserByRefreshToken(ctx context.Context, usersModel model.UsersModel, rtBase64 string, refreshSecret string) (*model.Users, error) {
	rt, err := utils.ParseRefreshToken(rtBase64, refreshSecret)
	if err != nil {
		// logx.Errorf("从 refreshToken 中提取用户ID失败, err=%v", err)
		// logc.Errorw(ctx, "从 refreshToken 中提取用户ID失败", logger.WErrorMsg(err.Error()))
		logger.L(ctx, "从 refreshToken 中提取用户ID失败").WErrorMsg(err.Error()).Errors()
		return nil, errs.New(errs.CodeInvalidToken)
	}
	uid, err := rt.GetUID()
	if err != nil {
		// logx.Errorf("从 refreshToken 中提取用户ID失败, err=%v", err)
		// logc.Errorw(ctx, "从 refreshToken 中提取用户ID失败", logger.WErrorMsg(err.Error()))
		logger.L(ctx, "从 refreshToken 中提取用户ID失败").WErrorMsg(err.Error()).Errors()
		return nil, errs.New(errs.CodeInvalidToken)
	}
	return GetUserByUid(ctx, usersModel, uid)
}

// GetAccessTokenClaimsByJWT 从 JWT 中解析 accessToken claims
func GetAccessTokenClaimsByJWT(ctx context.Context, tokenString, secret string) (*utils.AccessToken, error) {
	accessToken, err := utils.ParseAccessToken(tokenString, secret)
	if err != nil {
		// logx.Errorf("从 JWT 中解析 access claims 失败, err=%v", err)
		// logc.Errorw(ctx, "从 JWT 中解析 access claims 失败", logger.WErrorMsg(err.Error()))
		logger.L(ctx, "从 JWT 中解析 access claims 失败").WErrorMsg(err.Error()).Errors()

		// token 解析错误（包括格式错误、过期、签名无效等）都返回 CodeInvalidToken
		return nil, errs.New(errs.CodeInvalidToken)
	}
	return accessToken, nil
}

// GetRefreshTokenClaimsByJWT 从 JWT 中解析 refreshToken claims
func GetRefreshTokenClaimsByJWT(ctx context.Context, tokenString, secret string) (*utils.RefreshToken, error) {
	refreshToken, err := utils.ParseRefreshToken(tokenString, secret)
	if err != nil {
		// logx.Errorf("从 JWT 中解析 refresh claims 失败, err=%v", err)
		// logc.Errorw(ctx, "从 JWT 中解析 refresh claims 失败", logger.WErrorMsg(err.Error()))
		logger.L(ctx, "从 JWT 中解析 refresh claims 失败").WErrorMsg(err.Error()).Errors()

		// token 解析错误（包括格式错误、过期、签名无效等）都返回 CodeInvalidToken
		return nil, errs.New(errs.CodeInvalidToken)
	}
	return refreshToken, nil
}

// GetEmailByJwtCtx 从上下文获取用户邮箱
func GetEmailByJwtCtx(ctx context.Context) (string, error) {
	email, err := utils.GetEmailByAccessToken(ctx)
	if err != nil {
		// logx.Errorf("从JWT中提取用户邮箱失败, err=%v", err)
		// logc.Errorw(ctx, "从JWT中提取用户邮箱失败", logger.WErrorMsg(err.Error()))
		logger.L(ctx, "从 JWT 中提取用户邮箱失败").WErrorMsg(err.Error()).Errors()

		return "", errs.New(errs.CodeInternalError)
	}
	return email, nil
}

// 签发 accessToken
func GenerateAccessToken(ctx context.Context, c config.Config, user *model.Users) (string, error) {
	accessToken, err := utils.GenerateAccessToken(
		c.Auth.AccessSecret,
		c.Auth.AccessExpire,
		user.SnowflakeId,
		user.Nickname,
		user.Email,
	)
	if err != nil {
		// logx.Errorf("签发 accessToken 失败, email=%s, err=%v", user.Email, err)
		// logc.Errorw(
		// 	ctx, "签发 accessToken 失败",
		// 	logger.WErrorMsg(err.Error()),
		// 	logger.WEmail(user.Email),
		// )
		logger.L(ctx, "签发 accessToken 失败").WErrorMsg(err.Error()).WEmail(user.Email).Errors()
		return "", errs.New(errs.CodeInternalError)
	}
	return accessToken, nil
}

// 签发 refreshToken
func GenerateRefreshToken(ctx context.Context, c config.Config, user *model.Users) (string, error) {
	refreshToken, err := utils.GenerateRefreshToken(
		c.RefreshSecret,
		c.RefreshExpire,
		user.SnowflakeId,
	)
	if err != nil {
		// logx.Errorf("签发 refreshToken 失败, email=%s, err=%v", user.Email, err)
		// logc.Errorw(
		// 	ctx, "签发 refreshToken 失败",
		// 	logger.WErrorMsg(err.Error()),
		// 	logger.WEmail(user.Email),
		// )
		logger.L(ctx, "签发 refreshToken 失败").WErrorMsg(err.Error()).WEmail(user.Email).Errors()
		return "", errs.New(errs.CodeInternalError)
	}
	return refreshToken, nil
}

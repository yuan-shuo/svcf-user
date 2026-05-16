package userutils

import (
	"context"
	"user/internal/errs"
	"user/internal/logger"
	"user/internal/model"
	"user/internal/utils"

	"github.com/golang-jwt/jwt/v4"
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

// GetUserByRefreshTokenWithKeyManager 使用 KeyManager 从 refreshToken 中获取用户实例（支持 RS256）
func GetUserByRefreshTokenWithKeyManager(ctx context.Context, usersModel model.UsersModel, rtBase64 string, keyManager *utils.RSAKeyManager) (*model.Users, error) {
	// 解析 JWT header 获取 kid（不验证签名）
	token, _, err := new(jwt.Parser).ParseUnverified(rtBase64, &utils.RefreshToken{})
	if err != nil {
		logger.L(ctx, "解析 refreshToken header 失败").WErrorMsg(err.Error()).Errors()
		return nil, errs.New(errs.CodeInvalidToken)
	}

	// 从 header 获取 kid
	kid, ok := token.Header["kid"].(string)
	if !ok || kid == "" {
		logger.L(ctx, "refreshToken 中缺少 kid").Errors()
		return nil, errs.New(errs.CodeInvalidToken)
	}

	// 根据 kid 获取公钥
	publicKey, err := keyManager.GetPublicKeyByID(kid)
	if err != nil {
		logger.L(ctx, "获取公钥失败").WErrorMsg(err.Error()).Errors()
		return nil, errs.New(errs.CodeInvalidToken)
	}

	// 使用公钥验证 token
	rt, err := utils.ParseRefreshTokenWithRSA(rtBase64, publicKey)
	if err != nil {
		logger.L(ctx, "从 refreshToken 中提取用户ID失败").WErrorMsg(err.Error()).Errors()
		return nil, errs.New(errs.CodeInvalidToken)
	}
	uid, err := rt.GetUID()
	if err != nil {
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

// GenerateAccessTokenWithKeyManager 使用 RSAKeyManager 签发 accessToken（支持 RS256）
func GenerateAccessTokenWithKeyManager(ctx context.Context, keyManager *utils.RSAKeyManager, user *model.Users) (string, error) {
	privateKey := keyManager.GetCurrentPrivateKey()
	keyID := keyManager.GetCurrentKeyID()

	if privateKey == nil {
		logger.L(ctx, "签发 accessToken 失败：私钥为空").WEmail(user.Email).Errors()
		return "", errs.New(errs.CodeInternalError)
	}

	// 从配置获取过期时间
	expireSeconds := keyManager.GetConfig().TokenExpireSecs

	accessToken, err := utils.GenerateAccessTokenWithRSA(
		privateKey,
		keyID,
		expireSeconds,
		user.SnowflakeId,
		user.Nickname,
		user.Email,
	)
	if err != nil {
		logger.L(ctx, "使用 RSA 签发 accessToken 失败").WErrorMsg(err.Error()).WEmail(user.Email).Errors()
		return "", errs.New(errs.CodeInternalError)
	}
	return accessToken, nil
}

// GenerateRefreshTokenWithKeyManager 使用 RSAKeyManager 签发 refreshToken（支持 RS256）
func GenerateRefreshTokenWithKeyManager(ctx context.Context, keyManager *utils.RSAKeyManager, expireSeconds int64, user *model.Users) (string, error) {
	privateKey := keyManager.GetCurrentPrivateKey()
	keyID := keyManager.GetCurrentKeyID()

	if privateKey == nil {
		logger.L(ctx, "签发 refreshToken 失败：私钥为空").WEmail(user.Email).Errors()
		return "", errs.New(errs.CodeInternalError)
	}

	refreshToken, err := utils.GenerateRefreshTokenWithRSA(
		privateKey,
		keyID,
		expireSeconds,
		user.SnowflakeId,
	)
	if err != nil {
		logger.L(ctx, "使用 RSA 签发 refreshToken 失败").WErrorMsg(err.Error()).WEmail(user.Email).Errors()
		return "", errs.New(errs.CodeInternalError)
	}
	return refreshToken, nil
}

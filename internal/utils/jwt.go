package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	uidFieldName       string = "uid"
	emailFieldName     string = "email"
	tokenTypeFieldName string = "type"
	refreshTokenType   string = "refresh"
	accessTokenType    string = "access"
)

// 添加从 claims 中提取 uid 的方法
func GetUidFromClaims(claims jwt.MapClaims) (int64, error) {
	val, ok := claims[uidFieldName]
	if !ok {
		return 0, fmt.Errorf("uid not found in claims")
	}

	switch v := val.(type) {
	case float64:
		return int64(v), nil
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	case json.Number:
		return v.Int64()
	default:
		return 0, fmt.Errorf("unsupported uid type: %T", v)
	}
}

// 校验rt是否正确
func IsRefreshToken(claims jwt.MapClaims) error {
	// 校验 token 类型
	return ValidateClaimString(claims, tokenTypeFieldName, refreshTokenType)
}

// ValidateClaim 校验 claim 是否存在且值匹配
func ValidateClaimString(claims jwt.MapClaims, key string, expectedValue string) error {
	// 检查 key 是否存在
	val, ok := claims[key]
	if !ok {
		return fmt.Errorf("claim key: '%s' not found", key)
	}

	// 检查值是否匹配
	if val != expectedValue {
		return fmt.Errorf("claim key: '%s' mismatch: expected %v, got %v", key, expectedValue, val)
	}

	return nil
}

// GetUidByJwt 安全获取 uid，支持多种类型
func GetUidByJwt(ctx context.Context) (int64, error) {
	val := ctx.Value(uidFieldName)
	if val == nil {
		return 0, fmt.Errorf("uid not found in context")
	}

	switch v := val.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case json.Number: // ← 处理大数
		return v.Int64()
	case string:
		return strconv.ParseInt(v, 10, 64)
	case float64:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("uid type unsupported: %T, value: %v", v, v)
	}
}

// GetEmailByJwt 安全获取 email
func GetEmailByJwt(ctx context.Context) (string, error) {
	val := ctx.Value(emailFieldName)
	if val == nil {
		return "", fmt.Errorf("email not found in context")
	}

	if s, ok := val.(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("email type error: %T", val)
}

// GenerateAccessToken 生成 Access Token
func GenerateAccessToken(secret string, expireSeconds int64, uid int64, nickname, email string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		uidFieldName:       uid,                                                        // 用户ID
		"nickname":         nickname,                                                   // 昵称（常用）
		emailFieldName:     email,                                                      // 邮箱（常用）
		tokenTypeFieldName: accessTokenType,                                            // token类型
		"iat":              now.Unix(),                                                 // 签发时间
		"exp":              now.Add(time.Duration(expireSeconds) * time.Second).Unix(), // 过期时间
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken 生成 Refresh Token（JWT格式）
func GenerateRefreshToken(secret string, expireSeconds int64, uid int64) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"uid":              uid,
		tokenTypeFieldName: refreshTokenType,
		"iat":              now.Unix(),
		"exp":              now.Add(time.Duration(expireSeconds) * time.Second).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken 解析 JWT token
func ParseToken(tokenString, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}

// // 以下是 Redis 黑名单相关函数(也许后续会加入)

// // RevokeUserTokens 撤销用户的所有 token（修改密码、强制下线等场景）
// func RevokeUserTokens(redisClient *redis.Redis, uid int64, revokeTime int64) error {
//     // 记录撤销时间点
//     key := fmt.Sprintf("user:token:revoke:%d", uid)
//     return redisClient.Setex(key, fmt.Sprintf("%d", revokeTime), 7*24*3600) // 7天过期
// }

// // IsTokenRevoked 检查 token 是否已被撤销
// func IsTokenRevoked(redisClient *redis.Redis, claims jwt.MapClaims) (bool, error) {
//     uid := int64(claims["uid"].(float64))
//     iat := int64(claims["iat"].(float64))

//     // 查询撤销时间
//     revokeTimeStr, err := redisClient.Get(fmt.Sprintf("user:token:revoke:%d", uid))
//     if err != nil {
//         // key 不存在，说明没有被撤销
//         return false, nil
//     }

//     var revokeTime int64
//     fmt.Sscanf(revokeTimeStr, "%d", &revokeTime)

//     // 如果 token 签发时间早于撤销时间，说明已被撤销
//     return iat < revokeTime, nil
// }

// // ValidateToken 验证 token（同时检查黑名单）
// func ValidateToken(tokenString, secret string, redisClient *redis.Redis) (jwt.MapClaims, error) {
//     // 1. 解析 JWT
//     claims, err := ParseToken(tokenString, secret)
//     if err != nil {
//         return nil, err
//     }

//     // 2. 检查是否在黑名单中
//     revoked, err := IsTokenRevoked(redisClient, claims)
//     if err != nil {
//         return nil, err
//     }
//     if revoked {
//         return nil, errors.New("token has been revoked")
//     }

//     return claims, nil
// }

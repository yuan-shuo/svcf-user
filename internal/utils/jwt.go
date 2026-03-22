package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// GenerateAccessToken 生成 Access Token
func GenerateAccessToken(secret string, expireSeconds int64, uid int64, nickname, email string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"uid":      uid,                                                        // 用户ID
		"nickname": nickname,                                                   // 昵称（常用）
		"email":    email,                                                      // 邮箱（常用）
		"type":     "access",                                                   // token类型
		"iat":      now.Unix(),                                                 // 签发时间
		"exp":      now.Add(time.Duration(expireSeconds) * time.Second).Unix(), // 过期时间
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken 生成 Refresh Token（JWT格式）
func GenerateRefreshToken(secret string, expireSeconds int64, uid int64) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"uid":  uid,
		"type": "refresh",
		"iat":  now.Unix(),
		"exp":  now.Add(time.Duration(expireSeconds) * time.Second).Unix(),
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

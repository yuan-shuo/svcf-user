package utils

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	uidFieldName       string = "uid"      // 用户ID
	emailFieldName     string = "email"    // 用户邮箱
	tokenTypeFieldName string = "type"     // token类型: access/refresh
	nicknameFieldName  string = "nickname" // 昵称，用于显示

	refreshTokenType string = "refresh" // 刷新令牌类型
	accessTokenType  string = "access"  // 访问令牌类型
)

// ==================== 结构体定义（推荐方式）====================

// JwtClaims JWT 令牌声明
type JwtClaims struct {
	Uid       json.Number `json:"uid"`  // 用户ID（json.Number 避免精度丢失）
	TokenType string      `json:"type"` // token类型: access/refresh
	Iat       int64       `json:"iat"`  // 签发时间（可选，仅用于签发时）
	Exp       int64       `json:"exp"`  // 过期时间（可选，仅用于签发时）
}

// Valid 实现 jwt.Claims 接口
// 注意：当通过 ParseAccessToken/ParseRefreshToken 解析时，此验证会被调用
func (c JwtClaims) Valid() error {
	if c.Uid == "" {
		return errors.New("uid is required")
	}
	if _, err := c.Uid.Int64(); err != nil {
		return fmt.Errorf("invalid uid format: %w", err)
	}
	if c.TokenType == "" {
		return errors.New("token type is required")
	}

	// 验证过期时间（仅在 Exp 不为 0 时验证）
	if c.Exp != 0 {
		now := time.Now().Unix()
		if now > c.Exp {
			return errors.New("token is expired")
		}
	}

	return nil
}

// GetUID 安全获取 int64 类型的 UID
func (c JwtClaims) GetUID() (int64, error) {
	return c.Uid.Int64()
}

// AccessToken 访问令牌
type AccessToken struct {
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	JwtClaims
}

// Valid 实现接口
func (a AccessToken) Valid() error {
	if a.Email == "" {
		return errors.New("email is required in access token")
	}
	return a.JwtClaims.Valid()
}

// RefreshToken 刷新令牌
type RefreshToken struct {
	JwtClaims
}

// Valid 实现接口
func (r RefreshToken) Valid() error {
	if r.TokenType != refreshTokenType {
		return fmt.Errorf("invalid token type for refresh: %s", r.TokenType)
	}
	return r.JwtClaims.Valid()
}

// ==================== Token 生成（推荐）====================

// GenerateAccessToken 生成 Access Token
func GenerateAccessToken(secret string, expireSeconds int64, uid int64, nickname, email string) (string, error) {
	now := time.Now()
	claims := AccessToken{
		Nickname: nickname,
		Email:    email,
		JwtClaims: JwtClaims{
			Uid:       json.Number(strconv.FormatInt(uid, 10)),
			TokenType: accessTokenType,
			Iat:       now.Unix(),
			Exp:       now.Add(time.Duration(expireSeconds) * time.Second).Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken 生成 Refresh Token
func GenerateRefreshToken(secret string, expireSeconds int64, uid int64) (string, error) {
	now := time.Now()
	claims := RefreshToken{
		JwtClaims: JwtClaims{
			Uid:       json.Number(strconv.FormatInt(uid, 10)),
			TokenType: refreshTokenType,
			Iat:       now.Unix(),
			Exp:       now.Add(time.Duration(expireSeconds) * time.Second).Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ==================== Token 解析（推荐）====================

// ParseAccessToken 解析 Access Token
func ParseAccessToken(tokenString, secret string) (*AccessToken, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessToken{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse access token failed: %w", err)
	}
	claims, ok := token.Claims.(*AccessToken)
	if !ok || !token.Valid {
		return nil, errors.New("invalid access token claims")
	}
	return claims, nil
}

// ParseRefreshToken 解析 Refresh Token
func ParseRefreshToken(tokenString, secret string) (*RefreshToken, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshToken{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse refresh token failed: %w", err)
	}
	claims, ok := token.Claims.(*RefreshToken)
	if !ok || !token.Valid {
		return nil, errors.New("invalid refresh token claims")
	}
	return claims, nil
}

func GetEmailByAccessToken(ctx context.Context) (string, error) {
	at, err := AccessTokenFromContext(ctx)
	if err != nil {
		return "", errors.New("access token claims not found in context")
	}
	return at.Email, nil
}

// ==================== Context 操作（推荐）====================

// UIDFromAccessToken 从 context 获取 AccessToken 的 UID
func UIDFromAccessToken(ctx context.Context) (int64, error) {
	at, err := AccessTokenFromContext(ctx)
	if err != nil {
		return 0, errors.New("access token claims not found in context")
	}
	return at.GetUID()
}

// UIDFromRefreshToken 从 context 获取 RefreshToken 的 UID
func UIDFromRefreshToken(ctx context.Context) (int64, error) {
	rt, err := RefreshTokenFromContext(ctx)
	if err != nil {
		return 0, errors.New("refresh token claims not found in context")
	}
	return rt.GetUID()
}

// GetJWTClaimsByContext 从 context 获取 JWT claims
// 注意：iat 和 exp 是标准 JWT 字段，go-zero 中间件不会将其存入 context
func GetJWTClaimsByContext(ctx context.Context) (*JwtClaims, error) {
	uid, ok := ctx.Value(uidFieldName).(json.Number)
	if !ok {
		return nil, fmt.Errorf("uid not found in context or type mismatch")
	}
	tokenType, ok := ctx.Value(tokenTypeFieldName).(string)
	if !ok {
		return nil, fmt.Errorf("token type not found in context or type mismatch")
	}

	return &JwtClaims{
		Uid:       uid,
		TokenType: tokenType,
		// Iat 和 Exp 不从 context 获取，因为 go-zero 中间件会忽略标准 JWT 字段
	}, nil
}

// AccessTokenFromContext 获取完整的 AccessToken（需要其他字段时用）
func AccessTokenFromContext(ctx context.Context) (*AccessToken, error) {
	claims, err := GetJWTClaimsByContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("get jwt claims by context failed: %w", err)
	}
	nickname, ok := ctx.Value(nicknameFieldName).(string)
	if !ok {
		return nil, fmt.Errorf("nickname not found in context or type mismatch")
	}
	email, ok := ctx.Value(emailFieldName).(string)
	if !ok {
		return nil, fmt.Errorf("email not found in context or type mismatch")
	}
	return &AccessToken{
		Nickname:  nickname,
		Email:     email,
		JwtClaims: *claims,
	}, nil
}

// RefreshTokenFromContext 获取完整的 RefreshToken
func RefreshTokenFromContext(ctx context.Context) (*RefreshToken, error) {
	claims, err := GetJWTClaimsByContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("get jwt claims by context failed: %w", err)
	}
	return &RefreshToken{
		JwtClaims: *claims,
	}, nil
}

// ==================== RSA 非对称加密支持 ====================

// JWKS JSON Web Key Set 结构
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK JSON Web Key 结构
type JWK struct {
	Kty string `json:"kty"` // 密钥类型，RSA
	Kid string `json:"kid"` // 密钥标识符
	Use string `json:"use"` // 用途，sig 表示签名
	N   string `json:"n"`   // RSA 模数 (base64url 编码)
	E   string `json:"e"`   // RSA 指数 (base64url 编码)
	Alg string `json:"alg"` // 算法，RS256
}

// RSAKeyPair RSA 密钥对
type RSAKeyPair struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	KeyID      string // 密钥标识符，用于 JWKS 中的 kid
}

// GenerateRSAKeyPair 生成 RSA 密钥对
func GenerateRSAKeyPair(bits int) (*RSAKeyPair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, fmt.Errorf("generate rsa key pair failed: %w", err)
	}

	// 生成 key ID (使用时间戳和随机数)
	timestamp := time.Now().Unix()
	randNum, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	keyID := fmt.Sprintf("key-%d-%s", timestamp, randNum.String())

	return &RSAKeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
		KeyID:      keyID,
	}, nil
}

// PrivateKeyToPEM 将 RSA 私钥转换为 PEM 格式
func PrivateKeyToPEM(privateKey *rsa.PrivateKey) string {
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	return string(privateKeyPEM)
}

// PublicKeyToPEM 将 RSA 公钥转换为 PEM 格式
func PublicKeyToPEM(publicKey *rsa.PublicKey) (string, error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("marshal public key failed: %w", err)
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	return string(publicKeyPEM), nil
}

// ParseRSAPrivateKeyFromPEM 从 PEM 格式解析 RSA 私钥
func ParseRSAPrivateKeyFromPEM(pemString string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemString))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// 尝试解析 PKCS8 格式
		key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("parse private key failed: %w", err)
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA private key")
		}
	}

	return privateKey, nil
}

// ParseRSAPublicKeyFromPEM 从 PEM 格式解析 RSA 公钥
func ParseRSAPublicKeyFromPEM(pemString string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemString))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key failed: %w", err)
	}

	publicKey, ok := publicKeyInterface.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return publicKey, nil
}

// ToJWK 将 RSA 公钥转换为 JWK 格式
func (kp *RSAKeyPair) ToJWK() JWK {
	// 将模数 N 转换为 base64url 编码
	nBytes := kp.PublicKey.N.Bytes()
	nBase64 := base64.RawURLEncoding.EncodeToString(nBytes)

	// 将指数 E 转换为 base64url 编码
	eBytes := big.NewInt(int64(kp.PublicKey.E)).Bytes()
	eBase64 := base64.RawURLEncoding.EncodeToString(eBytes)

	return JWK{
		Kty: "RSA",
		Kid: kp.KeyID,
		Use: "sig",
		N:   nBase64,
		E:   eBase64,
		Alg: "RS256",
	}
}

// ToJWKS 返回 JWKS 格式（供网关获取）
func (kp *RSAKeyPair) ToJWKS() JWKS {
	return JWKS{
		Keys: []JWK{kp.ToJWK()},
	}
}

// ToJSON 将 JWKS 转换为 JSON 字符串
func (jwks JWKS) ToJSON() (string, error) {
	jsonBytes, err := json.MarshalIndent(jwks, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal JWKS failed: %w", err)
	}
	return string(jsonBytes), nil
}

// ToJWKSJSON 返回 JWKS 的 JSON 字符串
func (kp *RSAKeyPair) ToJWKSJSON() (string, error) {
	jwks := kp.ToJWKS()
	jsonBytes, err := json.MarshalIndent(jwks, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal JWKS failed: %w", err)
	}
	return string(jsonBytes), nil
}

// ==================== RS256 Token 生成（非对称加密）====================

// GenerateAccessTokenWithRSA 使用 RSA 私钥生成 Access Token
func GenerateAccessTokenWithRSA(privateKey *rsa.PrivateKey, keyID string, expireSeconds int64, uid int64, nickname, email string) (string, error) {
	now := time.Now()
	claims := AccessToken{
		Nickname: nickname,
		Email:    email,
		JwtClaims: JwtClaims{
			Uid:       json.Number(strconv.FormatInt(uid, 10)),
			TokenType: accessTokenType,
			Iat:       now.Unix(),
			Exp:       now.Add(time.Duration(expireSeconds) * time.Second).Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	if keyID != "" {
		token.Header["kid"] = keyID
	}
	return token.SignedString(privateKey)
}

// GenerateRefreshTokenWithRSA 使用 RSA 私钥生成 Refresh Token
func GenerateRefreshTokenWithRSA(privateKey *rsa.PrivateKey, keyID string, expireSeconds int64, uid int64) (string, error) {
	now := time.Now()
	claims := RefreshToken{
		JwtClaims: JwtClaims{
			Uid:       json.Number(strconv.FormatInt(uid, 10)),
			TokenType: refreshTokenType,
			Iat:       now.Unix(),
			Exp:       now.Add(time.Duration(expireSeconds) * time.Second).Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	if keyID != "" {
		token.Header["kid"] = keyID
	}
	return token.SignedString(privateKey)
}

// ==================== RS256 Token 解析（非对称加密，网关使用）====================

// ParseAccessTokenWithRSA 使用 RSA 公钥解析 Access Token
func ParseAccessTokenWithRSA(tokenString string, publicKey *rsa.PublicKey) (*AccessToken, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessToken{}, func(t *jwt.Token) (interface{}, error) {
		// 验证算法
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse access token failed: %w", err)
	}
	claims, ok := token.Claims.(*AccessToken)
	if !ok || !token.Valid {
		return nil, errors.New("invalid access token claims")
	}
	return claims, nil
}

// ParseRefreshTokenWithRSA 使用 RSA 公钥解析 Refresh Token
func ParseRefreshTokenWithRSA(tokenString string, publicKey *rsa.PublicKey) (*RefreshToken, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshToken{}, func(t *jwt.Token) (interface{}, error) {
		// 验证算法
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse refresh token failed: %w", err)
	}
	claims, ok := token.Claims.(*RefreshToken)
	if !ok || !token.Valid {
		return nil, errors.New("invalid refresh token claims")
	}
	return claims, nil
}

// ParseAccessTokenUnverified 解析 Access Token（不验证签名）
// 用于 user 服务解析网关已校验的 Token
func ParseAccessTokenUnverified(tokenString string) (*AccessToken, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &AccessToken{})
	if err != nil {
		return nil, fmt.Errorf("parse access token unverified failed: %w", err)
	}
	claims, ok := token.Claims.(*AccessToken)
	if !ok {
		return nil, errors.New("invalid access token claims")
	}
	return claims, nil
}

// ParseRefreshTokenUnverified 解析 Refresh Token（不验证签名）
// 用于 user 服务解析网关已校验的 Token
func ParseRefreshTokenUnverified(tokenString string) (*RefreshToken, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &RefreshToken{})
	if err != nil {
		return nil, fmt.Errorf("parse refresh token unverified failed: %w", err)
	}
	claims, ok := token.Claims.(*RefreshToken)
	if !ok {
		return nil, errors.New("invalid refresh token claims")
	}
	return claims, nil
}

// InitJWKSFromPEM 从公钥 PEM 字符串初始化 JWKS
// 用于 ServiceContext 初始化时加载公钥
func InitJWKSFromPEM(publicKeyPEM, keyID string) (JWKS, error) {
	// 如果配置了公钥 PEM，则解析并生成 JWKS
	if publicKeyPEM != "" {
		publicKey, err := ParseRSAPublicKeyFromPEM(publicKeyPEM)
		if err != nil {
			return JWKS{}, fmt.Errorf("parse RSA public key failed: %w", err)
		}

		// 创建临时 RSAKeyPair 来生成 JWK
		keyPair := &RSAKeyPair{
			PublicKey: publicKey,
			KeyID:     keyID,
		}
		return keyPair.ToJWKS(), nil
	}

	// 如果没有配置公钥，返回空的 JWKS
	return JWKS{Keys: []JWK{}}, nil
}

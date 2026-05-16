// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	KqConsumerConf kq.KqConf // 消息队列消费者配置

	SmtpConfig SmtpConfig // 邮件发送配置

	RedisConfig redis.RedisConf // redis配置

	PostgreSQL PostgreSQL      // pg数据库配置
	CacheRedis cache.CacheConf // 缓存数据库配置

	Auth          Auth   // jwt认证配置
	RefreshSecret string // Refresh Token 签名密钥
	RefreshExpire int64  // Refresh Token 有效期
	BcryptCost    int    // bcrypt 加密成本因子，取值范围 4-31

	VerifyCodeConfig VerifyCodeConfig // 验证码配置

	RateLimit RateLimit // 限流配置

	Cookie CookieConfig // Cookie 配置

	JWKS JWKSConfig // JWKS 配置

	KeyManager KeyManagerConfig // 密钥管理器配置

	rest.RestConf
}

// JWKS 配置
type JWKSConfig struct {
	CacheControlMaxAge int // JWKS 缓存时间（秒），默认 3600
}

// KeyManagerConfig 密钥管理器配置
type KeyManagerConfig struct {
	Bits             int   // 密钥位数，默认 2048
	RotationInterval int64 // 轮换周期（秒），默认 86400（24小时）
	GracePeriod      int64 // 旧密钥保留期（秒），默认 172800（48小时）
}

// 限流配置
type RateLimit struct {
	NoAuth         PeriodLimit     // 未认证接口限流配置
	RefreshToken   TokenLimit      // 刷新token接口限流配置
	ChangePassword PeriodLimit     // 修改密码接口限流配置
	RedisKeyPrefix string          // 限流redis数据库键的前缀
	RedisConfig    redis.RedisConf // 限流redis数据库配置
}

// 周期限流配置
type PeriodLimit struct {
	Period int // 限流周期，单位：秒
	Quota  int // 限流阈值，单位：次
}

// Token限流配置（令牌桶）
type TokenLimit struct {
	Rate  int // 每秒产生的令牌数
	Burst int // 桶容量（突发流量限制）
}

// 验证码配置
type VerifyCodeConfig struct {
	Type VerifyCodeType // 验证码类型
	Time VerifyCodeTime // 验证码有效期
}

type VerifyCodeTime struct {
	ExpireIn   int // 验证码有效期, 单位秒
	RetryAfter int // 验证码重试间隔, 单位秒
}

type VerifyCodeType struct {
	Register         string // 注册验证码类型
	ResetPassword    string // 重置密码验证码类型
	ChangePassword   string // 修改密码验证码类型
	RemindRegistered string // 邮箱已注册验证码类型(不会发送实际验证码而是仅提醒)
}

// jwt认证配置
type Auth struct {
	AccessExpire        int64  // Access Token 有效期（秒）
	AccessPrivateKeyPEM string // Access Token 签名私钥（RS256 使用，PEM 格式，留空则自动生成）
	KeyID               string // 密钥标识符，用于 JWKS 中的 kid（留空则自动生成）
}

// pg数据库配置
type PostgreSQL struct {
	Datasource string         // 数据源连接字符串
	Pool       PostgreSQLPool // 连接池配置
}

// PostgreSQLPool 连接池配置
type PostgreSQLPool struct {
	MaxOpenConns    int // 最大打开连接数，默认64
	MaxIdleConns    int // 最大空闲连接数，默认64
	ConnMaxLifetime int // 连接最大生命周期(秒)，默认3600秒(1小时)
	ConnMaxIdleTime int // 连接最大空闲时间(秒)，默认600秒(10分钟)
}

// 邮件发送配置
type SmtpConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// Cookie 配置
type CookieConfig struct {
	Domain   string // Cookie 域名
	Secure   bool   // 仅 HTTPS 传输
	HttpOnly bool   // 禁止 JS 读取
	SameSite string // SameSite 策略: strict, lax, none
}

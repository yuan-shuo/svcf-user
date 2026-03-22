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
	Register Register

	KqConsumerConf kq.KqConf // 消息队列消费者配置

	SmtpConfig SmtpConfig // 邮件发送配置

	RedisConfig redis.RedisConf // redis配置

	PostgreSQL PostgreSQL      // pg数据库配置
	CacheRedis cache.CacheConf // 缓存数据库配置

	Auth          Auth   // jwt认证配置
	RefreshSecret string // Refresh Token 签名密钥
	RefreshExpire int64  // Refresh Token 有效期

	rest.RestConf

	// KqPusherConf   KqPusherConf
}

// jwt认证配置
type Auth struct {
	AccessSecret string // Access Token 签名密钥
	AccessExpire int64  // Access Token 有效期
}

// pg数据库配置
type PostgreSQL struct {
	Datasource string
}

// 注册配置
type Register struct {
	SendCodeConfig SendCodeConfig
}

// 验证码发送配置
type SendCodeConfig struct {
	ReceiveType    string // 接收验证码类型
	ExpireIn       int
	RetryAfter     int
	RedisKeyPrefix string // 存放于redis时使用的键名前缀, 给入a则redis.key=a:receiver_email
	ReminderType   ReminderType
}

// 提醒类型消息
type ReminderType struct {
	Registered string // 邮箱已注册
}

// 邮件发送配置
type SmtpConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// // 消息队列生产者配置 暂时废弃
// type KqPusherConf struct {
// 	Brokers []string
// 	Topic   string
// }

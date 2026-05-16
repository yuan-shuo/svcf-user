// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"context"
	"fmt"
	"time"
	"user/internal/config"
	"user/internal/db"
	"user/internal/metrics"
	"user/internal/middleware"
	"user/internal/middleware/limiter"
	"user/internal/model"
	"user/internal/utils"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
)

type ServiceContext struct {
	Config              config.Config                 // 配置文件
	KqPusherClient      KqPusherClient                // 生产者实例
	Redis               *redis.Redis                  // redis 数据库
	UsersModel          model.UsersModel              // SQL 数据库
	Metrics             *metrics.Metrics              // 观测指标
	PeriodLimiterMgr    *limiter.PeriodLimiterManager // 周期限流器管理器
	TokenLimiterMgr     *limiter.TokenLimiterManager  // 令牌桶限流器管理器
	NoAuthLimit         rest.Middleware               // 无认证接口限流中间件
	RefreshTokenLimit   rest.Middleware               // 刷新token接口限流中间件
	ChangePasswordLimit rest.Middleware               // 修改密码接口限流中间件
	CookieSetter        rest.Middleware               // Cookie 设置中间件
	JWKSCacheControl    rest.Middleware               // JWKS 端点 Cache-Control 中间件
	KeyManager          *utils.RSAKeyManager          // RSA 密钥管理器（支持密钥轮换）
	KeyManagerStopFunc  func()                        // 密钥管理器自动轮换停止函数（用于优雅关闭）
}

// 定义为接口方便单元测试
type KqPusherClient interface {
	Push(ctx context.Context, v string) error
	Close() error
}

func NewServiceContext(c config.Config) *ServiceContext {

	// 初始化雪花id生成器
	if err := utils.InitSonyflake(1, "2024-01-01"); err != nil {
		panic(fmt.Sprintf("初始化雪花算法失败: %v", err))
	}

	// 初始化限流器redis数据库
	rateLimiterRedis := db.NewRedis(c.RateLimit.RedisConfig)
	// 初始化限流器管理器
	periodLimiterMgr := limiter.NewPeriodLimiterManager(c, rateLimiterRedis) // 周期限流器管理器
	tokenLimiterMgr := limiter.NewTokenLimiterManager(c, rateLimiterRedis)   // 令牌桶限流器管理器

	// 初始化 RSA 密钥管理器
	// 从配置文件读取配置
	keyManagerCfg := &utils.KeyManagerConfig{
		Bits:             c.KeyManager.Bits,
		RotationInterval: time.Duration(c.KeyManager.RotationInterval) * time.Second,
		GracePeriod:      time.Duration(c.KeyManager.GracePeriod) * time.Second,
		TokenExpireSecs:  c.Auth.AccessExpire,
	}

	var keyManager *utils.RSAKeyManager
	var err error
	if c.Auth.AccessPrivateKeyPEM != "" {
		// 从配置的私钥 PEM 加载
		keyManager, err = utils.NewRSAKeyManagerFromPEM(c.Auth.AccessPrivateKeyPEM, c.Auth.KeyID, keyManagerCfg)
	} else {
		// 使用配置创建新的密钥管理器
		keyManager, err = utils.NewRSAKeyManagerWithConfig(keyManagerCfg)
	}
	if err != nil {
		panic(fmt.Sprintf("初始化 RSA 密钥管理器失败: %v", err))
	}

	// 启动自动密钥轮换
	stopFunc, err := keyManager.AutoRotate(context.Background())
	if err != nil {
		panic(fmt.Sprintf("启动自动密钥轮换失败: %v", err))
	}

	// 返回上下文
	return &ServiceContext{
		Config: c,
		KqPusherClient: kq.NewPusher(
			// 生产者复用消费者brokers配置和topic，保持一致
			c.KqConsumerConf.Brokers,
			c.KqConsumerConf.Topic,
			// c.KqPusherConf.Brokers, // 已废弃KqPusherConf
			// c.KqPusherConf.Topic,
			kq.WithAllowAutoTopicCreation(),
		),
		Redis:               db.NewRedis(c.RedisConfig),
		UsersModel:          model.NewUsersModel(db.NewPostgreSQL(c.PostgreSQL), c.CacheRedis),
		Metrics:             metrics.NewMetrics(),
		PeriodLimiterMgr:    periodLimiterMgr,
		TokenLimiterMgr:     tokenLimiterMgr,
		NoAuthLimit:         middleware.NewNoAuthLimitMiddleware(periodLimiterMgr),
		RefreshTokenLimit:   middleware.NewRefreshTokenLimitMiddleware(tokenLimiterMgr),
		ChangePasswordLimit: middleware.NewChangePasswordLimitMiddleware(periodLimiterMgr),
		CookieSetter:        middleware.NewCookieSetterMiddleware(c),
		JWKSCacheControl:    middleware.NewJWKSCacheControlMiddleware(c.JWKS).Handle,
		KeyManager:          keyManager,
		KeyManagerStopFunc:  stopFunc,
	}
}

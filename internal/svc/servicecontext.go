// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"context"
	"fmt"
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
	Config              config.Config           // 配置文件
	KqPusherClient      KqPusherClient          // 生产者实例
	Redis               *redis.Redis            // redis 数据库
	UsersModel          model.UsersModel        // SQL 数据库
	Metrics             *metrics.MetricsManager // 观测指标
	NoAuthLimit         rest.Middleware         // 无认证接口限流中间件
	RefreshTokenLimit   rest.Middleware         // 刷新token接口限流中间件
	ChangePasswordLimit rest.Middleware         // 修改密码接口限流中间件
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
	// 初始化限流器
	noAuthLimiter := limiter.NewNoAuthPeriodLimiter(c, rateLimiterRedis)                 // 未认证接口限流器
	refreshTokenLimiter := limiter.NewRefreshTokenLimiter(c, rateLimiterRedis)           // 刷新token接口限流器
	changePasswordLimiter := limiter.NewChangePasswordPeriodLimiter(c, rateLimiterRedis) // 修改密码接口限流器

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
		Metrics:             metrics.NewMetricsManager(),
		NoAuthLimit:         middleware.NewNoAuthLimitMiddleware(noAuthLimiter),
		RefreshTokenLimit:   middleware.NewRefreshTokenLimitMiddleware(refreshTokenLimiter),
		ChangePasswordLimit: middleware.NewChangePasswordLimitMiddleware(changePasswordLimiter),
	}
}

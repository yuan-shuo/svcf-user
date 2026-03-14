// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"context"
	"user/internal/config"
	"user/internal/db"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config         config.Config
	KqPusherClient KqPusherClient
	Redis          *redis.Redis
}

// 定义为接口方便单元测试
type KqPusherClient interface {
	Push(ctx context.Context, v string) error
	Close() error
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		KqPusherClient: kq.NewPusher(
			c.KqPusherConf.Brokers,
			c.KqPusherConf.Topic,
			kq.WithAllowAutoTopicCreation(),
		),
		Redis: db.NewRedis(c.RedisConfig),
	}
}

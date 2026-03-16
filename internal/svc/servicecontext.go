// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"context"
	"fmt"
	"user/internal/config"
	"user/internal/db"
	"user/internal/model"
	"user/internal/utils"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config         config.Config
	KqPusherClient KqPusherClient
	Redis          *redis.Redis
	UsersModel     model.UsersModel
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

	// 返回上下文
	return &ServiceContext{
		Config: c,
		KqPusherClient: kq.NewPusher(
			c.KqPusherConf.Brokers,
			c.KqPusherConf.Topic,
			kq.WithAllowAutoTopicCreation(),
		),
		Redis:      db.NewRedis(c.RedisConfig),
		UsersModel: model.NewUsersModel(db.NewPostgreSQL(c.PostgreSQL), c.CacheRedis),
	}
}

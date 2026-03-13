// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"user/internal/config"

	"github.com/zeromicro/go-queue/kq"
)

type ServiceContext struct {
	Config         config.Config
	KqPusherClient *kq.Pusher
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		KqPusherClient: kq.NewPusher(
			c.KqPusherConf.Brokers,
			c.KqPusherConf.Topic,
			kq.WithAllowAutoTopicCreation(),
		),
	}
}

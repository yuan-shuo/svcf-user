//go:build !unit

package mqs

import (
	"context"

	"user/internal/config"
	"user/internal/mqs/sendemail"
	"user/internal/svc"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
)

func Consumers(c config.Config, ctx context.Context, svcContext *svc.ServiceContext) []service.Service {

	return []service.Service{
		// Listening for changes in consumption flow status
		// 消费者都在这里写, 每多一个消费者就写一行kq.MustNew...
		kq.MustNewQueue(c.KqConsumerConf, sendemail.NewSendEmail(ctx, svcContext)),
	}

}

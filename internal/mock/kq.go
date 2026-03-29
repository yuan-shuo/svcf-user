package mock

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// KqPusherClient 用于测试的 MQ 推送客户端 mock
type KqPusherClient struct {
	mock.Mock
	Messages []string
}

func (m *KqPusherClient) Push(ctx context.Context, v string) error {
	m.Messages = append(m.Messages, v)
	args := m.Called(ctx, v)
	return args.Error(0)
}

func (m *KqPusherClient) Close() error {
	return nil
}

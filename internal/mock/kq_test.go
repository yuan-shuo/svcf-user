package mock

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKqPusherClient_Push(t *testing.T) {
	t.Run("成功推送消息", func(t *testing.T) {
		m := new(KqPusherClient)
		ctx := context.Background()
		message := "test message"

		m.On("Push", ctx, message).Return(nil)

		err := m.Push(ctx, message)

		assert.NoError(t, err)
		assert.Len(t, m.Messages, 1)
		assert.Equal(t, message, m.Messages[0])
		m.AssertExpectations(t)
	})

	t.Run("推送多条消息", func(t *testing.T) {
		m := new(KqPusherClient)
		ctx := context.Background()
		messages := []string{"message1", "message2", "message3"}

		for _, msg := range messages {
			m.On("Push", ctx, msg).Return(nil)
		}

		for _, msg := range messages {
			err := m.Push(ctx, msg)
			assert.NoError(t, err)
		}

		assert.Len(t, m.Messages, 3)
		assert.Equal(t, messages, m.Messages)
		m.AssertExpectations(t)
	})

	t.Run("推送失败", func(t *testing.T) {
		m := new(KqPusherClient)
		ctx := context.Background()
		message := "test message"

		m.On("Push", ctx, message).Return(errors.New("push failed"))

		err := m.Push(ctx, message)

		assert.Error(t, err)
		assert.Equal(t, "push failed", err.Error())
		assert.Len(t, m.Messages, 1)
		m.AssertExpectations(t)
	})

	t.Run("不同上下文", func(t *testing.T) {
		m := new(KqPusherClient)
		ctx1 := context.Background()
		ctx2 := context.WithValue(context.Background(), "key", "value")
		message := "test message"

		m.On("Push", ctx1, message).Return(nil)
		m.On("Push", ctx2, message).Return(nil)

		err1 := m.Push(ctx1, message)
		err2 := m.Push(ctx2, message)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Len(t, m.Messages, 2)
		m.AssertExpectations(t)
	})
}

func TestKqPusherClient_Close(t *testing.T) {
	t.Run("关闭客户端", func(t *testing.T) {
		m := new(KqPusherClient)

		err := m.Close()

		assert.NoError(t, err)
	})
}

func TestKqPusherClient_Messages(t *testing.T) {
	t.Run("消息记录", func(t *testing.T) {
		m := new(KqPusherClient)
		ctx := context.Background()

		// 初始为空
		assert.Empty(t, m.Messages)

		// 推送消息
		m.On("Push", ctx, "msg1").Return(nil)
		m.On("Push", ctx, "msg2").Return(nil)

		m.Push(ctx, "msg1")
		m.Push(ctx, "msg2")

		// 验证消息被记录
		assert.Equal(t, []string{"msg1", "msg2"}, m.Messages)
		m.AssertExpectations(t)
	})
}

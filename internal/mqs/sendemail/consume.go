package sendemail

import (
	"context"
	"encoding/json"
	"fmt"
	"user/internal/svc"
	"user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendEmail struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSendEmail(ctx context.Context, svcCtx *svc.ServiceContext) *SendEmail {
	return &SendEmail{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendEmail) Consume(ctx context.Context, key, val string) error {
	var msg types.VerificationCodeMessage
	if err := json.Unmarshal([]byte(val), &msg); err != nil {
		return fmt.Errorf("消息解析失败: %w", err)
	}
	vt := l.svcCtx.Config.VerifyCodeConfig.Type
	// 根据消息类型分发到不同的处理器
	switch msg.Type {
	case vt.Register: // "register"
		return l.sendVerifyCodeEmail(&msg)

	case vt.RemindRegistered: // "remind_registered"
		return l.sendAlreadyRegisteredReminderEmail(&msg)

	default:
		logx.Errorf("未知的消息类型: %s", msg.Type)
		return nil // 不返回错误，避免阻塞队列
	}
}

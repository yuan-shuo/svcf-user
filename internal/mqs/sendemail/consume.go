package sendemail

import (
	"context"
	"encoding/json"
	"fmt"
	"user/internal/svc"
	"user/internal/types"
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
		return fmt.Errorf("邮箱验证码发送失败: %w", err)
	}
	return l.sendVerifyCodeEmail(&msg)
}

// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package account_noauth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"user/internal/svc"
	"user/internal/types"
	"user/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendRegisterCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSendRegisterCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendRegisterCodeLogic {
	return &SendRegisterCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendRegisterCodeLogic) SendRegisterCode(req *types.SendCodeReq) (resp *types.SendCodeResp, err error) {
	// todo: add your logic here and delete this line

	// 判断验证码类型是否为注册验证码
	if req.Type != l.svcCtx.Config.Register.SendCodeConfig.ReciveType {
		return nil, fmt.Errorf("无效的验证码类型: %s", req.Type)
	}

	code := utils.GenerateDigitCode(6)

	// 定义消息
	msg := types.VerificationCodeMessage{
		Code:      code,
		Receiver:  req.Email,
		Type:      l.svcCtx.Config.Register.SendCodeConfig.ReciveType,
		Timestamp: time.Now().Unix(),
	}

	// 序列化消息
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("消息序列化失败: %w", err)
	}

	// 将验证码推送到消息队列中
	if err := l.svcCtx.KqPusherClient.Push(context.Background(), string(msgBytes)); err != nil {
		return nil, fmt.Errorf("消息队列推送失败: %w", err)
	}

	return &types.SendCodeResp{
		RetryAfter: l.svcCtx.Config.Register.SendCodeConfig.RetryAfter,
	}, nil
}

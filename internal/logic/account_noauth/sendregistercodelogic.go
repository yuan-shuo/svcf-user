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
	if req.Type != l.svcCtx.Config.Register.SendCodeConfig.ReceiveType {
		return nil, fmt.Errorf("无效的验证码类型: %s", req.Type)
	}

	// 生成验证码
	code := utils.GenerateDigitCode(6)

	// 基础键前缀
	baseKey := fmt.Sprintf("%s:%s", l.svcCtx.Config.Register.SendCodeConfig.RedisKeyPrefix, l.svcCtx.Config.Register.SendCodeConfig.ReceiveType)

	// redis缓存验证码数据
	redisKey := fmt.Sprintf("%s:verify:%s", baseKey, req.Email)
	redisValue := map[string]string{
		"code": code,
		"used": "0", // 0: 未使用, 1: 已被使用
	}

	// 设置限流验证码键
	limitKey := fmt.Sprintf("%s:limit:%s", baseKey, req.Email)
	// 检查是否频繁发送（使用 Get 检查限制键是否存在）
	ttl, err := l.svcCtx.Redis.Ttl(limitKey)
	if err != nil {
		return nil, fmt.Errorf("检查发送频率失败: %w", err)
	}
	// ttl > 0 表示 key 存在且未过期
	if ttl > 0 {
		return nil, fmt.Errorf("发送过于频繁，请%d秒后重试", ttl)
	}

	// 写入注册验证码
	if err := utils.SetHashWithExpire(l.svcCtx.Redis, l.ctx, redisKey, redisValue, l.svcCtx.Config.Register.SendCodeConfig.ExpireIn); err != nil {
		return nil, fmt.Errorf("注册验证码缓存失败: %w", err)
	}
	// 写入限流验证码
	if err := l.svcCtx.Redis.SetexCtx(l.ctx, limitKey, "1", l.svcCtx.Config.Register.SendCodeConfig.RetryAfter); err != nil {
		return nil, fmt.Errorf("限流验证码缓存失败: %w", err)
	}

	// 定义消息
	msg := types.VerificationCodeMessage{
		Code:      code,
		Receiver:  req.Email,
		Type:      l.svcCtx.Config.Register.SendCodeConfig.ReceiveType,
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

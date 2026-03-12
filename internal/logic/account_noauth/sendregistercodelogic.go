// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package account_noauth

import (
	"context"
	"fmt"

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

	code, err := utils.GenerateDigitCode(6)
	if err != nil {
		return nil, fmt.Errorf("验证码生成失败: %w", err)
	}

	return &types.SendCodeResp{
		ExpireIn:   l.svcCtx.Config.Register.SendCodeConfig.ExpireIn,
		RetryAfter: l.svcCtx.Config.Register.SendCodeConfig.RetryAfter,
	}, nil
}

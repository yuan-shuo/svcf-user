// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user_noauth

import (
	"context"

	"user/internal/errs"
	"user/internal/logic/userutils"
	"user/internal/svc"
	"user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendVerifyCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSendVerifyCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendVerifyCodeLogic {
	return &SendVerifyCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// SendVerifyCode 发送验证码主流程
// 1. 验证请求参数
// 2. 检查限流
// 3. 根据验证码类型执行特定业务逻辑
// 4. 生成、存储验证码并发送到消息队列
func (l *SendVerifyCodeLogic) SendVerifyCode(req *types.SendVerifyCodeReq) (*types.SendVerifyCodeResp, error) {
	// 验证请求参数
	if err := userutils.ValidateVerifyCodeRequest(req.Email, req.Type, l.svcCtx.Config.VerifyCodeConfig.Type); err != nil {
		l.svcCtx.Metrics.Verifycode.CodesSentTotal.Inc(req.Type, "fail")
		return nil, err
	}

	// 检查限流
	if err := userutils.CheckRateLimit(l.ctx, l.svcCtx.Redis, l.svcCtx.Config.VerifyCodeConfig.Time.RetryAfter, req.Email, req.Type); err != nil {
		l.svcCtx.Metrics.Verifycode.CodesSentTotal.Inc(req.Type, "fail")
		return nil, err
	}

	// 3. 根据验证码类型执行对应的业务检查
	shouldContinue, err := userutils.CheckBusinessLogic(
		l.ctx,
		l.svcCtx.UsersModel,
		l.svcCtx.KqPusherClient,
		l.svcCtx.Config.VerifyCodeConfig,
		req.Email,
		req.Type,
	)
	if err != nil {
		userutils.CleanupRateLimit(l.ctx, l.svcCtx.Redis, req.Email, req.Type)
		l.svcCtx.Metrics.Verifycode.CodesSentTotal.Inc(req.Type, "fail")
		return nil, err
	}
	if !shouldContinue {
		// 邮箱已注册提醒，不发送验证码但不算失败
		l.svcCtx.Metrics.Verifycode.CodesSentTotal.Inc(req.Type, "skipped")
		return userutils.BuildVerifyCodeResponse(l.svcCtx.Config.VerifyCodeConfig.Time.RetryAfter), nil
	}

	// 清理旧验证码
	userutils.CleanupVerifyCode(l.ctx, l.svcCtx.Redis, req.Email, req.Type)

	// 生成并保存验证码
	code := userutils.GenerateAndSaveVerifyCode(l.ctx, l.svcCtx.Redis, l.svcCtx.Config.VerifyCodeConfig.Time.ExpireIn, req.Email, req.Type)
	if code == "" {
		userutils.CleanupRateLimit(l.ctx, l.svcCtx.Redis, req.Email, req.Type)
		l.svcCtx.Metrics.Verifycode.CodesSentTotal.Inc(req.Type, "fail")
		return nil, errs.New(errs.CodeInternalError)
	}

	// 发送到消息队列
	if err := userutils.SendVerifyCodeToMQ(l.ctx, l.svcCtx.KqPusherClient, req.Email, code, req.Type); err != nil {
		userutils.CleanupVerifyCodeAll(l.ctx, l.svcCtx.Redis, req.Email, req.Type)
		l.svcCtx.Metrics.Verifycode.CodesSentTotal.Inc(req.Type, "fail")
		return nil, err
	}

	l.svcCtx.Metrics.Verifycode.CodesSentTotal.Inc(req.Type, "success")
	return userutils.BuildVerifyCodeResponse(l.svcCtx.Config.VerifyCodeConfig.Time.RetryAfter), nil
}

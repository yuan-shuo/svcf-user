// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user_noauth

import (
	"context"

	"user/internal/errs"
	"user/internal/logic/userutils"
	"user/internal/metrics"
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
	// 获取验证码类型的枚举值
	codeTypeEnum := l.getCodeTypeEnum(req.Type)

	// 验证请求参数
	if err := userutils.ValidateVerifyCodeRequest(l.ctx, req.Email, req.Type, l.svcCtx.Config.VerifyCodeConfig.Type); err != nil {
		l.svcCtx.Metrics.VerifyCodeSendsTotal.Inc(codeTypeEnum, metrics.VerifyCodeSendsTotalStatusFailedRateLimit)
		return nil, err
	}

	// 检查限流
	if err := userutils.CheckRateLimit(l.ctx, l.svcCtx.Redis, l.svcCtx.Config.VerifyCodeConfig.Time.RetryAfter, req.Email, req.Type); err != nil {
		l.svcCtx.Metrics.VerifyCodeRateLimitHitsTotal.Inc(l.getRateLimitCodeTypeEnum(req.Type))
		l.svcCtx.Metrics.VerifyCodeSendsTotal.Inc(codeTypeEnum, metrics.VerifyCodeSendsTotalStatusFailedRateLimit)
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
		// 根据错误类型判断具体失败原因
		if codeErr, ok := errs.IsCodeError(err); ok {
			switch codeErr.Code {
			case errs.CodeEmailRegistered:
				l.svcCtx.Metrics.VerifyCodeSendsTotal.Inc(codeTypeEnum, metrics.VerifyCodeSendsTotalStatusFailedAlreadyRegistered)
			case errs.CodeEmailNotRegistered:
				l.svcCtx.Metrics.VerifyCodeSendsTotal.Inc(codeTypeEnum, metrics.VerifyCodeSendsTotalStatusFailedNotRegistered)
			default:
				l.svcCtx.Metrics.VerifyCodeSendsTotal.Inc(codeTypeEnum, metrics.VerifyCodeSendsTotalStatusFailedRateLimit)
			}
		} else {
			l.svcCtx.Metrics.VerifyCodeSendsTotal.Inc(codeTypeEnum, metrics.VerifyCodeSendsTotalStatusFailedRateLimit)
		}
		return nil, err
	}
	if !shouldContinue {
		// 邮箱已注册提醒，不发送验证码但不算失败
		return userutils.BuildVerifyCodeResponse(l.svcCtx.Config.VerifyCodeConfig.Time.RetryAfter), nil
	}

	// 清理旧验证码
	userutils.CleanupVerifyCode(l.ctx, l.svcCtx.Redis, req.Email, req.Type)

	// 生成并保存验证码
	code := userutils.GenerateAndSaveVerifyCode(l.ctx, l.svcCtx.Redis, l.svcCtx.Config.VerifyCodeConfig.Time.ExpireIn, req.Email, req.Type)
	if code == "" {
		userutils.CleanupRateLimit(l.ctx, l.svcCtx.Redis, req.Email, req.Type)
		l.svcCtx.Metrics.VerifyCodeSendsTotal.Inc(codeTypeEnum, metrics.VerifyCodeSendsTotalStatusFailedRateLimit)
		return nil, errs.New(errs.CodeInternalError)
	}

	// 发送到消息队列
	if err := userutils.SendVerifyCodeToMQ(l.ctx, l.svcCtx.KqPusherClient, req.Email, code, req.Type); err != nil {
		userutils.CleanupVerifyCodeAll(l.ctx, l.svcCtx.Redis, req.Email, req.Type)
		l.svcCtx.Metrics.VerifyCodeSendsTotal.Inc(codeTypeEnum, metrics.VerifyCodeSendsTotalStatusFailedRateLimit)
		return nil, err
	}

	l.svcCtx.Metrics.VerifyCodeSendsTotal.Inc(codeTypeEnum, metrics.VerifyCodeSendsTotalStatusSuccess)
	return userutils.BuildVerifyCodeResponse(l.svcCtx.Config.VerifyCodeConfig.Time.RetryAfter), nil
}

// getCodeTypeEnum 将字符串类型的验证码类型转换为指标枚举类型
func (l *SendVerifyCodeLogic) getCodeTypeEnum(codeType string) metrics.VerifyCodeSendsTotalCodeType {
	switch codeType {
	case l.svcCtx.Config.VerifyCodeConfig.Type.Register:
		return metrics.VerifyCodeSendsTotalCodeTypeRegister
	case l.svcCtx.Config.VerifyCodeConfig.Type.ResetPassword:
		return metrics.VerifyCodeSendsTotalCodeTypeResetPassword
	case l.svcCtx.Config.VerifyCodeConfig.Type.ChangePassword:
		return metrics.VerifyCodeSendsTotalCodeTypeChangePassword
	default:
		return metrics.VerifyCodeSendsTotalCodeTypeRegister
	}
}

// getRateLimitCodeTypeEnum 将字符串类型的验证码类型转换为限流指标枚举类型
func (l *SendVerifyCodeLogic) getRateLimitCodeTypeEnum(codeType string) metrics.VerifyCodeRateLimitHitsTotalCodeType {
	switch codeType {
	case l.svcCtx.Config.VerifyCodeConfig.Type.Register:
		return metrics.VerifyCodeRateLimitHitsTotalCodeTypeRegister
	case l.svcCtx.Config.VerifyCodeConfig.Type.ResetPassword:
		return metrics.VerifyCodeRateLimitHitsTotalCodeTypeResetPassword
	case l.svcCtx.Config.VerifyCodeConfig.Type.ChangePassword:
		return metrics.VerifyCodeRateLimitHitsTotalCodeTypeChangePassword
	default:
		return metrics.VerifyCodeRateLimitHitsTotalCodeTypeRegister
	}
}

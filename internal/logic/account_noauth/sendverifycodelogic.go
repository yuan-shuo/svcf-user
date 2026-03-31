// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package account_noauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/mail"
	"time"

	"user/internal/errs"
	"user/internal/logic/accutil"
	"user/internal/svc"
	"user/internal/types"
	"user/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
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
	if err := l.validateRequest(req); err != nil {
		l.svcCtx.Metrics.Verifycode.CodesSentTotal.Inc(req.Type, "fail")
		return nil, err
	}

	// 检查限流
	if err := l.checkRateLimit(req.Email, req.Type); err != nil {
		l.svcCtx.Metrics.Verifycode.CodesSentTotal.Inc(req.Type, "fail")
		return nil, err
	}

	// 3. 根据验证码类型执行对应的业务检查
	shouldContinue, err := l.checkBusinessLogic(req)
	if err != nil {
		l.cleanupRateLimit(req.Email, req.Type)
		l.svcCtx.Metrics.Verifycode.CodesSentTotal.Inc(req.Type, "fail")
		return nil, err
	}
	if !shouldContinue {
		// 邮箱已注册提醒，不发送验证码但不算失败
		l.svcCtx.Metrics.Verifycode.CodesSentTotal.Inc(req.Type, "skipped")
		return l.buildResponse(), nil
	}

	// 清理旧验证码
	l.cleanupVerifyCode(req.Email, req.Type)

	// 生成并保存验证码
	code := l.generateAndSaveCode(req)
	if code == "" {
		l.cleanupRateLimit(req.Email, req.Type)
		l.svcCtx.Metrics.Verifycode.CodesSentTotal.Inc(req.Type, "fail")
		return nil, errs.New(errs.CodeInternalError)
	}

	// 发送到消息队列
	if err := l.sendToMQ(req.Email, code, req.Type); err != nil {
		l.cleanupAll(req.Email, req.Type)
		l.svcCtx.Metrics.Verifycode.CodesSentTotal.Inc(req.Type, "fail")
		return nil, err
	}

	l.svcCtx.Metrics.Verifycode.CodesSentTotal.Inc(req.Type, "success")
	return l.buildResponse(), nil
}

// checkBusinessLogic 返回是否应该继续发送验证码
func (l *SendVerifyCodeLogic) checkBusinessLogic(req *types.SendVerifyCodeReq) (shouldContinue bool, err error) {
	switch req.Type {
	case l.svcCtx.Config.VerifyCodeConfig.Type.Register:
		return l.checkRegisterLogic(req.Email)
	case l.svcCtx.Config.VerifyCodeConfig.Type.ResetPassword, l.svcCtx.Config.VerifyCodeConfig.Type.ChangePassword:
		return l.checkResetPasswordLogic(req.Email)
	default:
		return false, errs.New(errs.CodeInvalidParam)
	}
}

// checkRegisterLogic 注册验证码业务检查
// 返回 shouldContinue: 是否继续发送验证码
func (l *SendVerifyCodeLogic) checkRegisterLogic(email string) (shouldContinue bool, err error) {
	_, err = l.svcCtx.UsersModel.FindOneByEmail(l.ctx, email)
	if err == nil {
		// 邮箱已存在，发送提醒邮件，不发送验证码
		if mqErr := l.sendToMQ(email, "", l.svcCtx.Config.VerifyCodeConfig.Type.RemindRegistered); mqErr != nil {
			logx.Errorf("发送已注册提醒邮件失败, email=%s, err=%v", email, mqErr)
			return false, errs.New(errs.CodeInternalError)
		}
		// 不继续发送验证码，但也不返回错误
		return false, nil
	}
	if err != sqlx.ErrNotFound {
		logx.Errorf("查询邮箱是否注册失败, email=%s, err=%v", email, err)
		return false, errs.New(errs.CodeInternalError)
	}
	// 邮箱未注册，继续发送验证码
	return true, nil
}

// checkResetPasswordLogic 重置密码验证码业务检查
// 返回 shouldContinue: 是否继续发送验证码
func (l *SendVerifyCodeLogic) checkResetPasswordLogic(email string) (shouldContinue bool, err error) {
	_, err = l.svcCtx.UsersModel.FindOneByEmail(l.ctx, email)
	if err == sqlx.ErrNotFound {
		// 邮箱不存在，返回错误，不发送验证码
		return false, errs.New(errs.CodeEmailNotRegistered)
	}
	if err != nil {
		logx.Errorf("查询邮箱是否注册失败, email=%s, err=%v", email, err)
		return false, errs.New(errs.CodeInternalError)
	}
	// 邮箱存在，继续发送验证码
	return true, nil
}

// validateRequest 验证请求参数
func (l *SendVerifyCodeLogic) validateRequest(req *types.SendVerifyCodeReq) error {
	if req == nil {
		return errs.New(errs.CodeInvalidParam)
	}

	if !l.isValidCodeType(req.Type) {
		logx.Errorf("无效的验证码请求类型, type=%s", req.Type)
		return errs.New(errs.CodeInvalidParam)
	}

	if _, err := mail.ParseAddress(req.Email); err != nil {
		logx.Errorf("邮箱格式不正确, email=%s, err=%v", req.Email, err)
		return errs.New(errs.CodeInvalidParam)
	}

	return nil
}

// isValidCodeType 检查验证码类型是否有效
func (l *SendVerifyCodeLogic) isValidCodeType(codeType string) bool {
	vt := l.svcCtx.Config.VerifyCodeConfig.Type
	return codeType == vt.Register || codeType == vt.ResetPassword || codeType == vt.ChangePassword
}

// checkRateLimit 检查发送频率限制
func (l *SendVerifyCodeLogic) checkRateLimit(email, codeType string) error {
	limitKey := accutil.BuildLimitKey(email, codeType)
	retryAfter := l.svcCtx.Config.VerifyCodeConfig.Time.RetryAfter

	// SET key value NX EX seconds：只有key不存在时才设置，并设置过期时间
	// 保证原子性
	ok, err := l.svcCtx.Redis.SetnxExCtx(l.ctx, limitKey, "1", retryAfter)
	if err != nil {
		logx.Errorf("限流检查失败, email=%s, err=%v", email, err)
		return errs.New(errs.CodeInternalError)
	}

	if !ok {
		// key已存在，获取剩余时间
		ttl, _ := l.svcCtx.Redis.Ttl(limitKey)
		logx.Errorf("发送过于频繁, email=%s, ttl=%d", email, ttl)
		l.svcCtx.Metrics.Verifycode.RateLimitHitsTotal.Inc(codeType)
		return errs.New(errs.CodeInvalidParam, fmt.Sprintf("发送过于频繁，请%d秒后重试", ttl))
	}

	return nil
}

// generateAndSaveCode 生成验证码并保存到Redis
func (l *SendVerifyCodeLogic) generateAndSaveCode(req *types.SendVerifyCodeReq) string {
	code := utils.GenerateMixedCode(6)
	redisKey := accutil.BuildVerifyKey(req.Email, req.Type)
	redisValue := map[string]string{
		accutil.RedisValueCodeFieldName: code,
		accutil.RedisValueUsedFieldName: "0",
	}

	if err := utils.SetHashWithExpire(
		l.svcCtx.Redis,
		l.ctx,
		redisKey,
		redisValue,
		l.svcCtx.Config.VerifyCodeConfig.Time.ExpireIn,
	); err != nil {
		logx.Errorf("验证码缓存失败, email=%s, err=%v", req.Email, err)
		return ""
	}

	return code
}

// sendToMQ 发送验证码消息到队列
func (l *SendVerifyCodeLogic) sendToMQ(email, code, codeType string) error {
	msg := types.VerificationCodeMessage{
		Code:      code,
		Receiver:  email,
		Type:      codeType,
		Timestamp: time.Now().Unix(),
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		logx.Errorf("消息序列化失败, email=%s, err=%v", email, err)
		return errs.New(errs.CodeInternalError)
	}

	if err := l.svcCtx.KqPusherClient.Push(l.ctx, string(msgBytes)); err != nil {
		logx.Errorf("消息队列推送失败, email=%s, err=%v", email, err)
		return errs.New(errs.CodeInternalError)
	}

	return nil
}

// buildResponse 构建响应
func (l *SendVerifyCodeLogic) buildResponse() *types.SendVerifyCodeResp {
	return &types.SendVerifyCodeResp{
		RetryAfter: l.svcCtx.Config.VerifyCodeConfig.Time.RetryAfter,
	}
}

// cleanupRateLimit 清理限流标记
func (l *SendVerifyCodeLogic) cleanupRateLimit(email, codeType string) {
	limitKey := accutil.BuildLimitKey(email, codeType)
	if _, err := l.svcCtx.Redis.DelCtx(l.ctx, limitKey); err != nil {
		logx.Errorf("清理限流标记失败, email=%s, err=%v", email, err)
	}
}

// cleanupVerifyCode 清理验证码数据
func (l *SendVerifyCodeLogic) cleanupVerifyCode(email, codeType string) {
	verifyKey := accutil.BuildVerifyKey(email, codeType)
	if _, err := l.svcCtx.Redis.DelCtx(l.ctx, verifyKey); err != nil {
		logx.Errorf("清理验证码数据失败, email=%s, err=%v", email, err)
	}
}

// cleanupAll 清理所有相关数据（用于失败回滚）
func (l *SendVerifyCodeLogic) cleanupAll(email, codeType string) {
	l.cleanupRateLimit(email, codeType)
	l.cleanupVerifyCode(email, codeType)
}

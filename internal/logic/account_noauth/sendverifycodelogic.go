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

func (l *SendVerifyCodeLogic) SendVerifyCode(req *types.SendVerifyCodeReq) (resp *types.SendVerifyCodeResp, err error) {
	// 验证请求（验证码类型及邮箱格式）
	if err := l.validateRequest(req); err != nil {
		return nil, err
	}

	// 检查限流
	if err := l.checkRateLimit(req.Email, req.Type); err != nil {
		return nil, err
	}

	// 注册验证码特殊逻辑
	if req.Type == l.svcCtx.Config.VerifyCodeConfig.Type.Register {
		// 检查邮箱是否被注册过
		_, err = l.svcCtx.UsersModel.FindOneByEmail(l.ctx, req.Email)
		if err == nil {
			// 邮箱已存在
			if err = l.sendToMQ(req.Email, "", l.svcCtx.Config.VerifyCodeConfig.Type.RemindRegistered); err != nil {
				return nil, err
			}
			return l.buildResponse(), nil
		}
		if err != sqlx.ErrNotFound {
			// 数据库查询出错
			logx.Errorf("查询邮箱是否注册失败, email=%s, err=%v", req.Email, err)
			return nil, errs.New(errs.CodeInternalError)
		}
	} else if req.Type == l.svcCtx.Config.VerifyCodeConfig.Type.ResetPassword {
		// 重置密码验证码特殊逻辑
		// 检查邮箱是否被注册过
		_, err = l.svcCtx.UsersModel.FindOneByEmail(l.ctx, req.Email)
		if err != nil {
			if err != sqlx.ErrNotFound {
				// 数据库查询出错
				logx.Errorf("查询邮箱是否注册失败, email=%s, err=%v", req.Email, err)
				return nil, errs.New(errs.CodeInternalError)
			}
			return nil, errs.New(errs.CodeEmailNotRegistered)
		}
	} else {
		return nil, errs.New(errs.CodeInvalidParam)
	}

	// 清理未使用的验证码
	l.cleanupRedisData(req.Email, req.Type)

	// 生成验证码
	code := l.generateCode()

	// 保存到 Redis
	if err := l.saveCodeToRedis(req.Email, code, req.Type); err != nil {
		// 如果保存失败，清理限流键
		l.cleanupRateLimit(req.Email, req.Type)
		return nil, err
	}

	// 发送到消息队列
	if err := l.sendToMQ(req.Email, code, req.Type); err != nil {
		// 如果发送失败，清理 Redis 数据
		l.cleanupRedisData(req.Email, req.Type)
		l.cleanupRateLimit(req.Email, req.Type)
		return nil, err
	}

	// 返回响应
	return l.buildResponse(), nil
}

// 这个函数后续最好有更好的实现方法
// 检查验证码类型是否正确
func (l *SendVerifyCodeLogic) isVerifyCodeValid(codeType string) (bool, string) {
	vt := l.svcCtx.Config.VerifyCodeConfig.Type
	needType := fmt.Sprintf("%s|%s", vt.Register, vt.ResetPassword)
	return codeType == vt.Register || codeType == vt.ResetPassword, needType
}

// 1. 请求验证模块
func (l *SendVerifyCodeLogic) validateRequest(req *types.SendVerifyCodeReq) error {
	if req == nil {
		return errs.New(errs.CodeInvalidParam)
	}
	// 检查验证码类型是否正确
	// if req.Type != l.svcCtx.Config.Register.SendCodeConfig.ReceiveType {
	ok, needType := l.isVerifyCodeValid(req.Type)
	if !ok {
		logx.Errorf("无效的验证码请求类型, type=%s, expected=%s", req.Type, needType)
		return errs.New(errs.CodeInvalidParam)
	}
	// 验证邮箱格式
	if _, err := mail.ParseAddress(req.Email); err != nil {
		logx.Errorf("邮箱格式不正确, email=%s, err=%v", req.Email, err)
		return errs.New(errs.CodeInvalidParam)
	}
	return nil
}

// 2. 限流检查模块
func (l *SendVerifyCodeLogic) checkRateLimit(email string, codeType string) error {
	limitKey := buildLimitKey(email, codeType)
	retryAfter := l.svcCtx.Config.VerifyCodeConfig.Time.RetryAfter

	// SET key value NX EX seconds：只有key不存在时才设置，并设置过期时间
	// 这是一个原子操作，Redis保证不会被打断
	ok, err := l.svcCtx.Redis.SetnxExCtx(l.ctx, limitKey, "1", retryAfter)
	if err != nil {
		logx.Errorf("限流检查失败, email=%s, err=%v", email, err)
		return errs.New(errs.CodeInternalError)
	}

	if !ok {
		// key已存在，获取剩余时间
		ttl, _ := l.svcCtx.Redis.Ttl(limitKey)
		logx.Errorf("发送过于频繁, email=%s, ttl=%d", email, ttl)
		return errs.New(errs.CodeInvalidParam, fmt.Sprintf("发送过于频繁，请%d秒后重试", ttl))
	}

	return nil // 设置成功，可以发送
}

// 3. 验证码生成模块
func (l *SendVerifyCodeLogic) generateCode() string {
	return utils.GenerateMixedCode(6)
}

// 4. Redis 存储模块
func (l *SendVerifyCodeLogic) saveCodeToRedis(email, code, codeType string) error {
	redisKey := buildVerifyKey(email, codeType)
	redisValue := map[string]string{
		redisValueCodeFieldName: code,
		redisValueUsedFieldName: "0",
	}

	if err := utils.SetHashWithExpire(
		l.svcCtx.Redis,
		l.ctx,
		redisKey,
		redisValue,
		l.svcCtx.Config.VerifyCodeConfig.Time.ExpireIn,
	); err != nil {
		logx.Errorf("注册验证码缓存失败, email=%s, err=%v", email, err)
		return errs.New(errs.CodeInternalError)
	}
	return nil
}

// 5. MQ 消息发送模块
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

	if err := l.svcCtx.KqPusherClient.Push(context.Background(), string(msgBytes)); err != nil {
		logx.Errorf("消息队列推送失败, email=%s, err=%v", email, err)
		return errs.New(errs.CodeInternalError)
	}
	return nil
}

// 6. 响应构建模块
func (l *SendVerifyCodeLogic) buildResponse() *types.SendVerifyCodeResp {
	return &types.SendVerifyCodeResp{
		RetryAfter: l.svcCtx.Config.VerifyCodeConfig.Time.RetryAfter,
	}
}

// 清理函数
func (l *SendVerifyCodeLogic) cleanupRateLimit(email string, codeType string) {
	limitKey := buildLimitKey(email, codeType)
	l.svcCtx.Redis.DelCtx(l.ctx, limitKey)
}

func (l *SendVerifyCodeLogic) cleanupRedisData(email string, codeType string) {
	verifyKey := buildVerifyKey(email, codeType)
	l.svcCtx.Redis.DelCtx(l.ctx, verifyKey)
}

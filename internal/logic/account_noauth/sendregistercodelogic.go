// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

// validateEmail 未实现

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

const (
	verifyKey               string = "verify" // 包级验证redis键词缀
	limitKey                string = "limit"  // 包级限流redis键词缀
	redisValueCodeFieldName string = "code"   // redis hash 验证码值的键名
	redisValueUsedFieldName string = "used"   // redis hash 是否使用过值的键名 "0": 未使用过
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
	// 验证请求（验证码类型及邮箱格式）
	if err := l.validateRequest(req); err != nil {
		return nil, err
	}

	// 检查邮箱是否被注册过
	_, err = l.svcCtx.UsersModel.FindOneByEmail(l.ctx, req.Email)
	if err == nil {
		// 邮箱已存在
		if err = l.sendReminderEmailRegisteredToMQ(req.Email); err != nil {
			return nil, err
		}
		return l.buildResponse(), nil
	}
	if err != sqlx.ErrNotFound {
		// 数据库查询出错
		logx.Errorf("查询邮箱是否注册失败, email=%s, err=%w", req.Email, err)
		return nil, errs.New(errs.CodeInternalError)
	}

	// 检查限流
	if err := l.checkRateLimit(req.Email); err != nil {
		return nil, err
	}

	// 清理未使用的验证码
	l.cleanupRedisData(req.Email)

	// 生成验证码
	code := l.generateCode()

	// 保存到 Redis
	if err := l.saveCodeToRedis(req.Email, code); err != nil {
		// 如果保存失败，清理限流键
		l.cleanupRateLimit(req.Email)
		return nil, err
	}

	// 发送到消息队列
	if err := l.sendToMQ(req.Email, code); err != nil {
		// 如果发送失败，清理 Redis 数据
		l.cleanupRedisData(req.Email)
		l.cleanupRateLimit(req.Email)
		return nil, err
	}

	// 返回响应
	return l.buildResponse(), nil
}

// 1. 请求验证模块
func (l *SendRegisterCodeLogic) validateRequest(req *types.SendCodeReq) error {
	if req == nil {
		return fmt.Errorf("请求不能为空")
	}
	// 检查验证码类型是否正确
	if req.Type != l.svcCtx.Config.Register.SendCodeConfig.ReceiveType {
		return fmt.Errorf("无效的验证码请求类型: %s", req.Type)
	}
	// 验证邮箱格式
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return fmt.Errorf("邮箱格式不正确: %w", err)
	}
	return nil
}

// 检查邮箱是否被注册
func (l *SendRegisterCodeLogic) checkIfEmailHasBeenRegistered(email string) error {
	_, err := l.svcCtx.UsersModel.FindOneByEmail(l.ctx, email)
	if err == nil {
		// 邮箱已存在
		return errs.New(errs.CodeEmailRegistered)
	}
	if err != sqlx.ErrNotFound {
		// 数据库查询出错
		logx.Errorf("查询邮箱是否注册失败, email=%s, err=%w", email, err)
		return errs.New(errs.CodeInternalError)
	}
	// 未找到，说明邮箱未注册
	return nil
}

// 2. 限流检查模块
func (l *SendRegisterCodeLogic) checkRateLimit(email string) error {
	limitKey := l.buildLimitKey(email)
	retryAfter := l.svcCtx.Config.Register.SendCodeConfig.RetryAfter

	// SET key value NX EX seconds：只有key不存在时才设置，并设置过期时间
	// 这是一个原子操作，Redis保证不会被打断
	ok, err := l.svcCtx.Redis.SetnxExCtx(l.ctx, limitKey, "1", retryAfter)
	if err != nil {
		return fmt.Errorf("限流检查失败: %w", err)
	}

	if !ok {
		// key已存在，获取剩余时间
		ttl, _ := l.svcCtx.Redis.Ttl(limitKey)
		return fmt.Errorf("发送过于频繁，请%d秒后重试", ttl)
	}

	return nil // 设置成功，可以发送
}

// 3. 验证码生成模块
func (l *SendRegisterCodeLogic) generateCode() string {
	return utils.GenerateMixedCode(6)
}

// 4. Redis 存储模块
func (l *SendRegisterCodeLogic) saveCodeToRedis(email, code string) error {
	redisKey := l.buildVerifyKey(email)
	redisValue := map[string]string{
		redisValueCodeFieldName: code,
		redisValueUsedFieldName: "0",
	}

	if err := utils.SetHashWithExpire(
		l.svcCtx.Redis,
		l.ctx,
		redisKey,
		redisValue,
		l.svcCtx.Config.Register.SendCodeConfig.ExpireIn,
	); err != nil {
		return fmt.Errorf("注册验证码缓存失败: %w", err)
	}
	return nil
}

// 5. MQ 消息发送模块
func (l *SendRegisterCodeLogic) sendToMQ(email, code string) error {
	msg := types.VerificationCodeMessage{
		Code:      code,
		Receiver:  email,
		Type:      l.svcCtx.Config.Register.SendCodeConfig.ReceiveType,
		Timestamp: time.Now().Unix(),
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("消息序列化失败: %w", err)
	}

	if err := l.svcCtx.KqPusherClient.Push(context.Background(), string(msgBytes)); err != nil {
		return fmt.Errorf("消息队列推送失败: %w", err)
	}
	return nil
}

// 5. MQ 消息发送模块（邮件提示已注册）
func (l *SendRegisterCodeLogic) sendReminderEmailRegisteredToMQ(email string) error {
	msg := types.VerificationCodeMessage{
		Receiver:  email,
		Type:      l.svcCtx.Config.Register.SendCodeConfig.ReminderType.Registered,
		Timestamp: time.Now().Unix(),
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("消息序列化失败: %w", err)
	}

	if err := l.svcCtx.KqPusherClient.Push(context.Background(), string(msgBytes)); err != nil {
		return fmt.Errorf("消息队列推送失败: %w", err)
	}
	return nil
}

// 6. 响应构建模块
func (l *SendRegisterCodeLogic) buildResponse() *types.SendCodeResp {
	return &types.SendCodeResp{
		RetryAfter: l.svcCtx.Config.Register.SendCodeConfig.RetryAfter,
	}
}

// ==================== 辅助函数 ====================

func (l *SendRegisterCodeLogic) buildBaseKey() string {
	return fmt.Sprintf("%s:%s",
		l.svcCtx.Config.Register.SendCodeConfig.RedisKeyPrefix,
		l.svcCtx.Config.Register.SendCodeConfig.ReceiveType)
}

func (l *SendRegisterCodeLogic) buildVerifyKey(email string) string {
	return fmt.Sprintf("%s:%s:%s", l.buildBaseKey(), verifyKey, email)
}

func (l *SendRegisterCodeLogic) buildLimitKey(email string) string {
	return fmt.Sprintf("%s:%s:%s", l.buildBaseKey(), limitKey, email)
}

// 清理函数
func (l *SendRegisterCodeLogic) cleanupRateLimit(email string) {
	limitKey := l.buildLimitKey(email)
	l.svcCtx.Redis.DelCtx(l.ctx, limitKey)
}

func (l *SendRegisterCodeLogic) cleanupRedisData(email string) {
	verifyKey := l.buildVerifyKey(email)
	l.svcCtx.Redis.DelCtx(l.ctx, verifyKey)
}

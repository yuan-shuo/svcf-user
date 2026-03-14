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

	emailverifier "github.com/AfterShip/email-verifier"
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

var emailVerifier = emailverifier.NewVerifier()

func (l *SendRegisterCodeLogic) SendRegisterCode(req *types.SendCodeReq) (resp *types.SendCodeResp, err error) {
	// 1. 验证请求
	if err := l.validateRequest(req); err != nil {
		return nil, err
	}

	// 2. 检查限流
	if err := l.checkRateLimit(req.Email); err != nil {
		return nil, err
	}

	// 3. 生成验证码
	code := l.generateCode()

	// 4. 保存到 Redis
	if err := l.saveCodeToRedis(req.Email, code); err != nil {
		// 如果保存失败，清理限流键
		l.cleanupRateLimit(req.Email)
		return nil, err
	}

	// 5. 发送到消息队列
	if err := l.sendToMQ(req.Email, code); err != nil {
		// 如果发送失败，清理 Redis 数据
		l.cleanupRedisData(req.Email)
		l.cleanupRateLimit(req.Email)
		return nil, err
	}

	// 6. 返回响应
	return l.buildResponse(), nil
}

// 1. 请求验证模块
func (l *SendRegisterCodeLogic) validateRequest(req *types.SendCodeReq) error {
	if req == nil {
		return fmt.Errorf("请求不能为空")
	}
	if req.Type != l.svcCtx.Config.Register.SendCodeConfig.ReceiveType {
		return fmt.Errorf("无效的验证码请求类型: %s", req.Type)
	}
	// 邮箱验证子模块
	if err := l.validateEmail(req.Email); err != nil {
		return err
	}
	return nil
}

// 邮箱验证子模块
func (l *SendRegisterCodeLogic) validateEmail(email string) error {
	// 1. 基础非空检查
	if email == "" {
		return fmt.Errorf("邮箱不能为空")
	}

	// 2. 使用 email-verifier 进行完整验证
	ret, err := emailVerifier.Verify(email)
	if err != nil {
		return fmt.Errorf("邮箱验证失败: %w", err)
	}

	// 3. 语法验证
	if !ret.Syntax.Valid {
		return fmt.Errorf("邮箱格式无效")
	}

	// 4. MX 记录检查（确保域名存在且能接收邮件）
	if !ret.HasMxRecords {
		return fmt.Errorf("域名不存在或无法接收邮件")
	}

	// 5. 可选：临时邮箱检查（防止恶意注册）
	if ret.Disposable {
		return fmt.Errorf("不支持使用临时邮箱")
	}

	// 6. 可选：角色账号检查（防止用 admin@, info@ 等）
	if ret.RoleAccount {
		return fmt.Errorf("请使用个人邮箱")
	}

	return nil
}

// 2. 限流检查模块
func (l *SendRegisterCodeLogic) checkRateLimit(email string) error {
	limitKey := l.buildLimitKey(email)

	ttl, err := l.svcCtx.Redis.Ttl(limitKey)
	if err != nil {
		return fmt.Errorf("检查发送频率失败: %w", err)
	}
	if ttl > 0 {
		return fmt.Errorf("发送过于频繁，请%d秒后重试", ttl)
	}

	// 设置限流
	if err := l.svcCtx.Redis.SetexCtx(l.ctx, limitKey, "1", l.svcCtx.Config.Register.SendCodeConfig.RetryAfter); err != nil {
		return fmt.Errorf("设置限流失败: %w", err)
	}
	return nil
}

// 3. 验证码生成模块
func (l *SendRegisterCodeLogic) generateCode() string {
	return utils.GenerateDigitCode(6)
}

// 4. Redis 存储模块
func (l *SendRegisterCodeLogic) saveCodeToRedis(email, code string) error {
	redisKey := l.buildVerifyKey(email)
	redisValue := map[string]string{
		"code": code,
		"used": "0",
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
	return fmt.Sprintf("%s:verify:%s", l.buildBaseKey(), email)
}

func (l *SendRegisterCodeLogic) buildLimitKey(email string) string {
	return fmt.Sprintf("%s:limit:%s", l.buildBaseKey(), email)
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

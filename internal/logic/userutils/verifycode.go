package userutils

import (
	"context"
	"encoding/json"
	"fmt"
	"net/mail"
	"time"

	"user/internal/config"
	"user/internal/errs"
	"user/internal/types"
	"user/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// ValidateVerifyCodeRequest 验证发送验证码请求参数
func ValidateVerifyCodeRequest(email, codeType string, validTypes config.VerifyCodeType) error {
	if !IsValidCodeType(codeType, validTypes) {
		logx.Errorf("无效的验证码请求类型, type=%s", codeType)
		return errs.New(errs.CodeInvalidParam)
	}

	if _, err := mail.ParseAddress(email); err != nil {
		logx.Errorf("邮箱格式不正确, email=%s, err=%v", email, err)
		return errs.New(errs.CodeInvalidParam)
	}

	return nil
}

// IsValidCodeType 检查验证码类型是否有效
func IsValidCodeType(codeType string, vt config.VerifyCodeType) bool {
	return codeType == vt.Register || codeType == vt.ResetPassword || codeType == vt.ChangePassword
}

// CheckRateLimit 检查发送频率限制
func CheckRateLimit(ctx context.Context, redisClient RedisInterface, retryAfter int, email, codeType string) error {
	limitKey := BuildLimitKey(email, codeType)

	// SET key value NX EX seconds：只有key不存在时才设置，并设置过期时间
	// 保证原子性
	ok, err := redisClient.SetnxExCtx(ctx, limitKey, "1", retryAfter)
	if err != nil {
		logx.Errorf("限流检查失败, email=%s, err=%v", email, err)
		return errs.New(errs.CodeInternalError)
	}

	if !ok {
		// key已存在，获取剩余时间
		ttl, _ := redisClient.Ttl(limitKey)
		logx.Errorf("发送过于频繁, email=%s, ttl=%d", email, ttl)
		return errs.New(errs.CodeInvalidParam, fmt.Sprintf("发送过于频繁，请%d秒后重试", ttl))
	}

	return nil
}

// CheckRegisterLogic 注册验证码业务检查
// 返回 shouldContinue: 是否继续发送验证码
func CheckRegisterLogic(ctx context.Context, usersModel UsersModelInterface, mqClient MqPusherClientInterface, remindRegisteredType string, email string) (shouldContinue bool, err error) {
	_, err = usersModel.FindOneByEmail(ctx, email)
	if err == nil {
		// 邮箱已存在，发送提醒邮件，不发送验证码
		if mqErr := SendVerifyCodeToMQ(ctx, mqClient, email, "", remindRegisteredType); mqErr != nil {
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

// CheckResetPasswordLogic 重置密码验证码业务检查
// 返回 shouldContinue: 是否继续发送验证码
func CheckResetPasswordLogic(ctx context.Context, usersModel UsersModelInterface, email string) (shouldContinue bool, err error) {
	_, err = usersModel.FindOneByEmail(ctx, email)
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

// GenerateAndSaveVerifyCode 生成验证码并保存到Redis
func GenerateAndSaveVerifyCode(ctx context.Context, redisClient RedisInterface, expireIn int, email, codeType string) string {
	code := utils.GenerateMixedCode(6)
	redisKey := BuildVerifyKey(email, codeType)

	// 尝试使用 Pipeline 批量执行
	if pipeliner, ok := redisClient.(RedisPipeliner); ok {
		pipe := pipeliner.Pipeline()
		pipe.HSet(ctx, redisKey, RedisValueCodeFieldName, code)
		pipe.HSet(ctx, redisKey, RedisValueUsedFieldName, "0")
		pipe.Expire(ctx, redisKey, time.Duration(expireIn)*time.Second)

		if err := pipe.Exec(ctx); err != nil {
			logx.Errorf("验证码缓存失败, email=%s, err=%v", email, err)
			return ""
		}
		return code
	}

	// 回退到普通方式
	fields := map[string]string{
		RedisValueCodeFieldName: code,
		RedisValueUsedFieldName: "0",
	}
	if err := redisClient.HmsetCtx(ctx, redisKey, fields); err != nil {
		logx.Errorf("验证码缓存失败, email=%s, err=%v", email, err)
		return ""
	}
	if err := redisClient.ExpireCtx(ctx, redisKey, expireIn); err != nil {
		logx.Errorf("设置验证码过期时间失败, email=%s, err=%v", email, err)
		return ""
	}

	return code
}

// SendVerifyCodeToMQ 发送验证码消息到队列
func SendVerifyCodeToMQ(ctx context.Context, mqClient MqPusherClientInterface, email, code, codeType string) error {
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

	if err := mqClient.Push(ctx, string(msgBytes)); err != nil {
		logx.Errorf("消息队列推送失败, email=%s, err=%v", email, err)
		return errs.New(errs.CodeInternalError)
	}

	return nil
}

// CleanupRateLimit 清理限流标记
func CleanupRateLimit(ctx context.Context, redisClient RedisInterface, email, codeType string) {
	limitKey := BuildLimitKey(email, codeType)
	if _, err := redisClient.DelCtx(ctx, limitKey); err != nil {
		logx.Errorf("清理限流标记失败, email=%s, err=%v", email, err)
	}
}

// CleanupVerifyCode 清理验证码数据
func CleanupVerifyCode(ctx context.Context, redisClient RedisInterface, email, codeType string) {
	verifyKey := BuildVerifyKey(email, codeType)
	if _, err := redisClient.DelCtx(ctx, verifyKey); err != nil {
		logx.Errorf("清理验证码数据失败, email=%s, err=%v", email, err)
	}
}

// CleanupVerifyCodeAll 清理所有相关数据（用于失败回滚）
func CleanupVerifyCodeAll(ctx context.Context, redisClient RedisInterface, email, codeType string) {
	CleanupRateLimit(ctx, redisClient, email, codeType)
	CleanupVerifyCode(ctx, redisClient, email, codeType)
}

// CheckBusinessLogic 根据验证码类型执行业务检查，返回是否应该继续发送验证码
func CheckBusinessLogic(ctx context.Context, usersModel UsersModelInterface, mqClient MqPusherClientInterface, cfg config.VerifyCodeConfig, email, codeType string) (shouldContinue bool, err error) {
	switch codeType {
	case cfg.Type.Register:
		return CheckRegisterLogic(ctx, usersModel, mqClient, cfg.Type.RemindRegistered, email)
	case cfg.Type.ResetPassword, cfg.Type.ChangePassword:
		return CheckResetPasswordLogic(ctx, usersModel, email)
	default:
		return false, errs.New(errs.CodeInvalidParam)
	}
}

// BuildVerifyCodeResponse 构建发送验证码响应
func BuildVerifyCodeResponse(retryAfter int) *types.SendVerifyCodeResp {
	return &types.SendVerifyCodeResp{
		RetryAfter: retryAfter,
	}
}

package accutil

import (
	"context"
	"fmt"
	"strings"
	"user/internal/errs"
	"user/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// buildBaseKey 构建基础 key
func buildBaseKey(codeType string) string {
	return fmt.Sprintf("%s:%s",
		RedisKeyPrefix,
		codeType,
	)
}

// BuildVerifyKey 构建验证码验证 key
func BuildVerifyKey(email string, codeType string) string {
	return fmt.Sprintf("%s:%s:%s", buildBaseKey(codeType), VerifyKey, email)
}

// BuildLimitKey 构建限流 key
func BuildLimitKey(email string, codeType string) string {
	return fmt.Sprintf("%s:%s:%s", buildBaseKey(codeType), LimitKey, email)
}

// VerifyEmailAndCodeInRedis 检查验证码是否属于对应邮箱以及是否正确
func VerifyEmailAndCodeInRedis(ctx context.Context, svcCtx *svc.ServiceContext, email string, code string, codeType string) error {
	key := BuildVerifyKey(email, codeType)

	// 一次获取所有字段（Hgetall）
	fields, err := svcCtx.Redis.HgetallCtx(ctx, key)
	if err != nil {
		logx.Errorf("获取验证码信息失败, email=%s, key=%s, err=%v", email, key, err)
		return errs.New(errs.CodeInternalError)
	}

	// 键不存在或没有 code 字段
	if len(fields) == 0 || fields[RedisValueCodeFieldName] == "" {
		return errs.New(errs.CodeInvalidCode)
	}

	// 检查是否已使用
	if fields[RedisValueUsedFieldName] != "0" {
		return errs.New(errs.CodeCodeAlreadyUsed)
	}

	// 比对验证码
	if !strings.EqualFold(fields[RedisValueCodeFieldName], code) {
		return errs.New(errs.CodeInvalidCode)
	}

	return nil
}

// MarkCodeAsUsed 标记验证码为已使用
func MarkCodeAsUsed(ctx context.Context, svcCtx *svc.ServiceContext, email string, codeType string) {
	key := BuildVerifyKey(email, codeType)
	if err := svcCtx.Redis.HsetCtx(ctx, key, RedisValueUsedFieldName, "1"); err != nil {
		logx.Errorf("标记验证码已使用失败, email=%s, key=%s, err=%v", email, key, err)
	}
}

package accutil

import (
	"context"
	"fmt"
	"strings"
	"user/internal/errs"
	"user/internal/svc"
	"user/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	VerifyKey               string = "verify"  // 包级验证redis键词缀
	LimitKey                string = "limit"   // 包级限流redis键词缀
	RedisValueCodeFieldName string = "code"    // redis hash 验证码值的键名
	RedisValueUsedFieldName string = "used"    // redis hash 是否使用过值的键名 "0": 未使用过
	RedisKeyPrefix          string = "account" // redis 键名前缀
)

func buildBaseKey(codeType string) string {
	return fmt.Sprintf("%s:%s",
		RedisKeyPrefix,
		codeType,
	)
}

func BuildVerifyKey(email string, codeType string) string {
	return fmt.Sprintf("%s:%s:%s", buildBaseKey(codeType), VerifyKey, email)
}

func BuildLimitKey(email string, codeType string) string {
	return fmt.Sprintf("%s:%s:%s", buildBaseKey(codeType), LimitKey, email)
}

// 检查验证码是否属于对应邮箱以及是否正确
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

// 标记验证码为已使用
func MarkCodeAsUsed(ctx context.Context, svcCtx *svc.ServiceContext, email string, codeType string) {
	key := BuildVerifyKey(email, codeType)
	if err := svcCtx.Redis.HsetCtx(ctx, key, RedisValueUsedFieldName, "1"); err != nil {
		logx.Errorf("标记验证码已使用失败, email=%s, key=%s, err=%v", email, key, err)
	}
}

// 密码加密
func HashPassword(email, password string) (string, error) {
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		// 记录详细错误日志
		logx.Errorf("密码加密失败, email=%s, err=%v", email, err)
		// 返回通用错误给客户端
		return "", errs.New(errs.CodeInternalError)
	}
	return hashedPassword, nil
}

// 重置用户密码
func ResetUserPassword(ctx context.Context, svcCtx *svc.ServiceContext, email, newPassword string) error {
	// 获取用户
	user, err := svcCtx.UsersModel.FindOneByEmail(ctx, email)
	if err != nil {
		logx.Errorf("获取用户实例失败, email=%s, err=%v", email, err)
		return errs.New(errs.CodeInternalError)
	}

	// 检查新密码是否与旧密码相同
	if user.PasswordHash == newPassword {
		return errs.New(errs.CodePasswordSameAsOld)
	}

	// 重设密码
	user.PasswordHash = newPassword
	// 更新数据库密码
	if err := svcCtx.UsersModel.Update(ctx, user); err != nil {
		logx.Errorf("重设用户密码实败, email=%s, err=%v", email, err)
		return errs.New(errs.CodeInternalError)
	}

	return nil
}

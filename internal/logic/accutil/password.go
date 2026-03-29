package accutil

import (
	"context"
	"errors"
	"user/internal/errs"
	"user/internal/svc"
	"user/internal/utils"

	"user/internal/model"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
)

// 密码校验函数: 模糊错误返回
func VerifyPasswordWithVagueMismatchErrHint(hashedPassword, password, email string) error {
	return verifyPassword(hashedPassword, password, email, errs.New(errs.CodeUserNotExistOrPasswordIncorrect))
}

// 密码校验函数: 旧密码错误返回
func VerifyPasswordWithOldPasswordMismatchErrHint(hashedPassword, password, email string) error {
	return verifyPassword(hashedPassword, password, email, errs.New(errs.CodeOldPasswordIncorrect))
}

// 密码校验函数
func verifyPassword(hashedPassword, password, email string, mismatchErrHint error) error {
	if err := utils.ComparePassword(hashedPassword, password); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return mismatchErrHint
		}
		logx.Errorf("用户登录密码校验失败, email=%s, err=%v", email, err)
		return errs.New(errs.CodeInternalError)
	}
	return nil
}

// HashPassword 密码加密
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

// ResetUserPassword 重置用户密码
func ResetUserPasswordByEmail(ctx context.Context, svcCtx *svc.ServiceContext, email, newPassword string) error {
	// 获取用户
	user, err := GetUserByEmail(ctx, svcCtx, email)
	if err != nil {
		return err
	}

	return ResetUserPassword(ctx, svcCtx, user, newPassword)
}

// GetUserByUid 获取用户实例
func GetUserByUid(ctx context.Context, svcCtx *svc.ServiceContext, uid int64) (*model.Users, error) {
	user, err := svcCtx.UsersModel.FindOneBySnowflakeId(ctx, uid)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, errs.New(errs.CodeUserNotFound)
		}
		logx.Errorf("基于UID获取用户实例失败, uid=%d, err=%v", uid, err)
		return nil, errs.New(errs.CodeInternalError)
	}
	return user, nil
}

// GetUserByJwtCtx 获取用户实例
func GetUserByJwtCtx(ctx context.Context, svcCtx *svc.ServiceContext) (*model.Users, error) {
	uid := utils.GetUidByJwt(ctx)
	return GetUserByUid(ctx, svcCtx, uid)
}

// getUserByEmail 获取用户实例
func GetUserByEmail(ctx context.Context, svcCtx *svc.ServiceContext, email string) (*model.Users, error) {
	user, err := svcCtx.UsersModel.FindOneByEmail(ctx, email)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, errs.New(errs.CodeUserNotFound)
		}
		logx.Errorf("基于邮箱获取用户实例失败, email=%s, err=%v", email, err)
		return nil, errs.New(errs.CodeInternalError)
	}
	return user, nil
}

// resetUserPassword 重置用户密码
func ResetUserPassword(ctx context.Context, svcCtx *svc.ServiceContext, user *model.Users, newPassword string) error {
	// 检查新密码是否与旧密码相同
	if err := utils.ComparePassword(user.PasswordHash, newPassword); err == nil {
		return errs.New(errs.CodePasswordSameAsOld)
	}

	newHashedPassword, err := HashPassword(user.Email, newPassword)
	if err != nil {
		return err
	}

	// 重设密码
	user.PasswordHash = newHashedPassword
	// 更新数据库密码
	if err := svcCtx.UsersModel.Update(ctx, user); err != nil {
		logx.Errorf("重设用户密码实败, email=%s, err=%v", user.Email, err)
		return errs.New(errs.CodeInternalError)
	}

	return nil
}

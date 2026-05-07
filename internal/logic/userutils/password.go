package userutils

import (
	"context"
	"database/sql"
	"errors"
	"user/internal/errs"
	"user/internal/model"
	"user/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
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

// ValidatePasswordStrength 校验密码强度
// 封装 utils.ValidatePassword，将普通错误转换为业务错误码
func ValidatePasswordStrength(password string) error {
	if err := utils.ValidatePassword(password); err != nil {
		return errs.New(errs.CodeWeakPassword)
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
func ResetUserPasswordByEmail(ctx context.Context, usersModel model.UsersModel, email, newPassword string) error {
	// 获取用户
	user, err := GetUserByEmail(ctx, usersModel, email)
	if err != nil {
		return err
	}

	return ResetUserPassword(ctx, usersModel, user, newPassword)
}

// GetUserByUid 获取用户实例
func GetUserByUid(ctx context.Context, usersModel model.UsersModel, uid int64) (*model.Users, error) {
	user, err := usersModel.FindOneBySnowflakeId(ctx, uid)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, errs.New(errs.CodeUserNotFound)
		}
		logx.Errorf("基于UID获取用户实例失败, uid=%d, err=%v", uid, err)
		return nil, errs.New(errs.CodeInternalError)
	}
	return user, nil
}

// GetUserByAccessTokenJwtCtx 获取用户实例
func GetUserByAccessJwtCtx(ctx context.Context, usersModel model.UsersModel) (*model.Users, error) {
	uid, err := utils.UIDFromAccessToken(ctx)
	if err != nil {
		logx.Errorf("从JWT中提取用户ID失败, err=%v", err)
		return nil, errs.New(errs.CodeInternalError)
	}
	return GetUserByUid(ctx, usersModel, uid)
}

// GetUserByRefreshTokenJwtCtx 获取用户实例
func GetUserByRefreshJwtCtx(ctx context.Context, usersModel model.UsersModel) (*model.Users, error) {
	uid, err := utils.UIDFromRefreshToken(ctx)
	if err != nil {
		logx.Errorf("从JWT中提取用户ID失败, err=%v", err)
		return nil, errs.New(errs.CodeInternalError)
	}
	return GetUserByUid(ctx, usersModel, uid)
}

// GetUserByEmail 获取用户实例
func GetUserByEmail(ctx context.Context, usersModel model.UsersModel, email string) (*model.Users, error) {
	user, err := usersModel.FindOneByEmail(ctx, email)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, errs.New(errs.CodeUserNotFound)
		}
		logx.Errorf("基于邮箱获取用户实例失败, email=%s, err=%v", email, err)
		return nil, errs.New(errs.CodeInternalError)
	}
	return user, nil
}

// CheckEmailNotRegistered 检查邮箱未被注册（用于注册）
func CheckEmailNotRegistered(ctx context.Context, usersModel model.UsersModel, email string) error {
	_, err := usersModel.FindOneByEmail(ctx, email)
	if err == nil {
		// 邮箱已存在
		return errs.New(errs.CodeEmailRegistered)
	}
	if err != sqlx.ErrNotFound {
		// 数据库查询出错
		logx.Errorf("查询邮箱是否注册失败, email=%s, err=%v", email, err)
		return errs.New(errs.CodeInternalError)
	}
	// 未找到，说明邮箱未注册
	return nil
}

// CreateUser 创建用户
func CreateUser(ctx context.Context, usersModel model.UsersModel, nickname, email, hashedPassword string) error {
	snowflakeId, err := utils.GenerateID()
	if err != nil {
		logx.Errorf("雪花id生成失败, email=%s, err=%v", email, err)
		return errs.New(errs.CodeInternalError)
	}
	_, err = usersModel.Insert(ctx, &model.Users{
		SnowflakeId:  snowflakeId,
		Nickname:     nickname,
		Email:        email,
		PasswordHash: hashedPassword,
		DeletedAt:    sql.NullTime{Valid: false},
	})
	if err != nil {
		logx.Errorf("数据库创建用户失败, email=%s, err=%v", email, err)
		return errs.New(errs.CodeInternalError)
	}
	return nil
}

// resetUserPassword 重置用户密码
func ResetUserPassword(ctx context.Context, usersModel model.UsersModel, user *model.Users, newPassword string) error {
	// 校验密码强度
	if err := ValidatePasswordStrength(newPassword); err != nil {
		return err
	}

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
	if err := usersModel.Update(ctx, user); err != nil {
		logx.Errorf("重设用户密码实败, email=%s, err=%v", user.Email, err)
		return errs.New(errs.CodeInternalError)
	}

	return nil
}

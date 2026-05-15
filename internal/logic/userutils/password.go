package userutils

import (
	"context"
	"database/sql"
	"errors"
	"user/internal/errs"
	"user/internal/logger"
	"user/internal/model"
	"user/internal/utils"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// 密码校验函数: 模糊错误返回
func VerifyPasswordWithVagueMismatchErrHint(ctx context.Context, hashedPassword, password, email string) error {
	return verifyPassword(ctx, hashedPassword, password, email, errs.New(errs.CodeUserNotExistOrPasswordIncorrect))
}

// 密码校验函数: 旧密码错误返回
func VerifyPasswordWithOldPasswordMismatchErrHint(ctx context.Context, hashedPassword, password, email string) error {
	return verifyPassword(ctx, hashedPassword, password, email, errs.New(errs.CodeOldPasswordIncorrect))
}

// 密码校验函数
func verifyPassword(ctx context.Context, hashedPassword, password, email string, mismatchErrHint error) error {
	if err := utils.ComparePassword(hashedPassword, password); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return mismatchErrHint
		}
		logger.L(ctx, "用户登录密码校验失败").WErrorMsg(err.Error()).WEmail(email).Errors()
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

func HashPassword(ctx context.Context, email, password string, cost int) (string, error) {
	hashedPassword, err := utils.HashPassword(password, cost)
	if err != nil {
		// 类型断言判断是否是 InvalidCostError
		if _, ok := err.(bcrypt.InvalidCostError); ok {
			// 这里后面应该搞个promtheus-warn指标之类的做提醒，毕竟不能panic连坐其他接口，但还得有提醒
			logger.L(ctx, "bcrypt cost 配置错误").WBcryptCost(cost).WErrorMsg(err.Error()).Errors()
			return "", errs.New(errs.CodeInternalError)
		}

		logger.L(ctx, "密码加密失败").WEmail(email).WErrorMsg(err.Error()).Errors()
		return "", errs.New(errs.CodeInternalError)
	}
	return hashedPassword, nil
}

// ResetUserPasswordByEmail 重置用户密码
func ResetUserPasswordByEmail(ctx context.Context, usersModel model.UsersModel, email, newPassword string, bcryptCost int) error {
	// 获取用户
	user, err := GetUserByEmail(ctx, usersModel, email)
	if err != nil {
		return err
	}

	return ResetUserPassword(ctx, usersModel, user, newPassword, bcryptCost)
}

// GetUserByUid 获取用户实例
func GetUserByUid(ctx context.Context, usersModel model.UsersModel, uid int64) (*model.Users, error) {
	user, err := usersModel.FindOneBySnowflakeId(ctx, uid)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, errs.New(errs.CodeUserNotFound)
		}
		logger.L(ctx, "基于UID获取用户实例失败").WUid(uid).WErrorMsg(err.Error()).Errors()
		return nil, errs.New(errs.CodeInternalError)
	}
	return user, nil
}

// GetUserByAccessTokenJwtCtx 获取用户实例
func GetUserByAccessJwtCtx(ctx context.Context, usersModel model.UsersModel) (*model.Users, error) {
	uid, err := utils.UIDFromAccessToken(ctx)
	if err != nil {
		logger.L(ctx, "从JWT中提取用户ID失败").WErrorMsg(err.Error()).Errors()
		return nil, errs.New(errs.CodeInternalError)
	}
	return GetUserByUid(ctx, usersModel, uid)
}

// GetUserByRefreshTokenJwtCtx 获取用户实例
func GetUserByRefreshJwtCtx(ctx context.Context, usersModel model.UsersModel) (*model.Users, error) {
	uid, err := utils.UIDFromRefreshToken(ctx)
	if err != nil {
		logger.L(ctx, "从JWT中提取用户ID失败").WErrorMsg(err.Error()).Errors()
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
		logger.L(ctx, "基于邮箱获取用户实例失败").WEmail(email).WErrorMsg(err.Error()).Errors()
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
		logger.L(ctx, "查询邮箱是否注册失败").WEmail(email).WErrorMsg(err.Error()).Errors()
		return errs.New(errs.CodeInternalError)
	}
	// 未找到，说明邮箱未注册
	return nil
}

// CreateUser 创建用户
func CreateUser(ctx context.Context, usersModel model.UsersModel, nickname, email, hashedPassword string) error {
	snowflakeId, err := utils.GenerateID()
	if err != nil {
		logger.L(ctx, "雪花id生成失败").WEmail(email).WErrorMsg(err.Error()).Errors()
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
		logger.L(ctx, "数据库创建用户失败").WEmail(email).WErrorMsg(err.Error()).Errors()
		return errs.New(errs.CodeInternalError)
	}
	return nil
}

// resetUserPassword 重置用户密码
func ResetUserPassword(ctx context.Context, usersModel model.UsersModel, user *model.Users, newPassword string, bcryptCost int) error {
	// 校验密码强度
	if err := ValidatePasswordStrength(newPassword); err != nil {
		return err
	}

	// 检查新密码是否与旧密码相同
	if err := utils.ComparePassword(user.PasswordHash, newPassword); err == nil {
		return errs.New(errs.CodePasswordSameAsOld)
	}

	newHashedPassword, err := HashPassword(ctx, user.Email, newPassword, bcryptCost)
	if err != nil {
		return err
	}

	// 重设密码
	user.PasswordHash = newHashedPassword
	// 更新数据库密码
	if err := usersModel.Update(ctx, user); err != nil {
		logger.L(ctx, "重设用户密码失败").WEmail(user.Email).WErrorMsg(err.Error()).Errors()
		return errs.New(errs.CodeInternalError)
	}

	return nil
}

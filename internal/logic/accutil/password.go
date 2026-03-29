package accutil

import (
	"context"
	"user/internal/errs"
	"user/internal/svc"
	"user/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

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

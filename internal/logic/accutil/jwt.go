package accutil

import (
	"context"
	"user/internal/errs"
	"user/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

// GetEmailByJwtCtx 从上下文获取用户邮箱
func GetEmailByJwtCtx(ctx context.Context) (string, error) {
	email, err := utils.GetEmailByJwt(ctx)
	if err != nil {
		logx.Errorf("从JWT中提取用户邮箱失败, err=%v", err)
		return "", errs.New(errs.CodeInternalError)
	}
	return email, nil
}

// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user_noauth

import (
	"context"

	"user/internal/logic/userutils"
	"user/internal/svc"
	"user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *types.RegisterReq) (resp *types.RegisterResp, err error) {
	// 所需验证码类型 - 注册
	codeType := l.svcCtx.Config.VerifyCodeConfig.Type.Register

	// 检查验证码是否属于对应邮箱以及是否正确
	if err := userutils.VerifyEmailAndCodeInRedis(l.ctx, l.svcCtx.Redis, req.Email, req.Code, codeType); err != nil {
		l.svcCtx.Metrics.AccountNoauth.RegistrationsTotal.Inc("fail")
		return nil, err
	}

	// 检查邮箱是否被注册过
	if err := userutils.CheckEmailNotRegistered(l.ctx, l.svcCtx.UsersModel, req.Email); err != nil {
		l.svcCtx.Metrics.AccountNoauth.RegistrationsTotal.Inc("fail")
		return nil, err
	}

	// 校验密码强度
	if err := userutils.ValidatePasswordStrength(req.Password); err != nil {
		l.svcCtx.Metrics.AccountNoauth.RegistrationsTotal.Inc("fail")
		return nil, err
	}

	// 密码加密
	hashedPassword, err := userutils.HashPassword(req.Email, req.Password, l.svcCtx.Config.BcryptCost)
	if err != nil {
		l.svcCtx.Metrics.AccountNoauth.RegistrationsTotal.Inc("fail")
		return nil, err
	}

	// 数据库创建用户
	if err := userutils.CreateUser(l.ctx, l.svcCtx.UsersModel, req.Nickname, req.Email, hashedPassword); err != nil {
		l.svcCtx.Metrics.AccountNoauth.RegistrationsTotal.Inc("fail")
		return nil, err
	}

	// 标记验证码已被使用
	userutils.MarkCodeAsUsed(l.ctx, l.svcCtx.Redis, req.Email, codeType)

	l.svcCtx.Metrics.AccountNoauth.RegistrationsTotal.Inc("success")
	return
}

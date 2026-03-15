// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package account_noauth

import (
	"context"
	"fmt"
	"strings"

	"user/internal/svc"
	"user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
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

	// 检查邮箱及验证码是否正确
	if err := l.verfiyEmailAndCodeInRedis(req.Email, req.Code); err != nil {
		return nil, fmt.Errorf("邮箱验证码验证失败: %w", err)
	}

	// 检查邮箱是否被注册
	// 还没实现，要和pg联动

	// 密码加密(等数据库接)
	// hashedPassword, err := l.hashPassword(req.Password)
	// if err != nil {
	// 	logx.Errorf("密码加密失败, email=%s, err=%v", req.Email, err)
	// 	return nil, errors.New("注册失败，请稍后重试")
	// }

	// 创建用户记录（写入数据库）
	// 还没实现，要和pg联动
	// if err := l.createUser(req, hashedPassword); err != nil {
	// 	logx.Errorf("创建用户失败, email=%s, err=%v", req.Email, err)
	// 	return nil, errors.New("注册失败，请稍后重试")
	// }

	// 标记验证码已被使用
	l.markCodeAsUsed(req.Email)

	return
}

func (l *RegisterLogic) verfiyEmailAndCodeInRedis(email string, code string) error {

	// 构建检查键
	key := l.buildVerifyKey(email)

	// 检查邮箱键是否存在
	exists, err := l.svcCtx.Redis.ExistsCtx(l.ctx, key)
	if err != nil {
		return fmt.Errorf("验证码验证键失败: %w", err)
	}
	if !exists {
		return fmt.Errorf("验证码不存在或已过期")
	}

	// 检查是否已被使用
	used, err := l.svcCtx.Redis.HgetCtx(l.ctx, key, redisValueUsedFieldName)
	if err != nil {
		return fmt.Errorf("redis获取%s字段失败: %w", redisValueUsedFieldName, err)
	}
	if used != "0" {
		return fmt.Errorf("验证码已被使用")
	}

	// 获取存储的验证码
	storedCode, err := l.svcCtx.Redis.HgetCtx(l.ctx, key, redisValueCodeFieldName)
	if err != nil {
		return fmt.Errorf("redis获取%s字段失败: %w", redisValueCodeFieldName, err)
	}

	// 比对验证码（忽略大小写）
	if !strings.EqualFold(storedCode, code) {
		return fmt.Errorf("验证码错误")
	}

	return nil
}

// 创建用户
// 还没写，等pg
// func (l *RegisterLogic) createUser(req *types.RegisterReq, hashedPassword string) (int64, error) {
//     user := &User{
//         Email:     req.Email,
//         Password:  hashedPassword,
//         Nickname:  req.Nickname,
//         CreatedAt: time.Now().Unix(),
//     }

//     err := l.svcCtx.DB.Create(user).Error
//     if err != nil {
//         return 0, err
//     }
//     return user.Id, nil
// }

// 密码加密
func (l *RegisterLogic) hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// 标记验证码为已使用
func (l *RegisterLogic) markCodeAsUsed(email string) {
	key := l.buildVerifyKey(email)
	if err := l.svcCtx.Redis.HsetCtx(l.ctx, key, redisValueUsedFieldName, "1"); err != nil {
		logx.Errorf("标记验证码已使用失败, email=%s, err=%v", email, err)
	}
}

func (l *RegisterLogic) buildBaseKey() string {
	return fmt.Sprintf("%s:%s",
		l.svcCtx.Config.Register.SendCodeConfig.RedisKeyPrefix,
		l.svcCtx.Config.Register.SendCodeConfig.ReceiveType)
}

func (l *RegisterLogic) buildVerifyKey(email string) string {
	return fmt.Sprintf("%s:%s:%s", l.buildBaseKey(), verifyKey, email)
}

// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package account_noauth

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"user/internal/model"
	"user/internal/svc"
	"user/internal/types"
	"user/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
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

	// 检查邮箱是否被注册过
	if err := l.checkIfEmailHasBeenRegistered(req.Email); err != nil {
		return nil, fmt.Errorf("邮箱注册信息验证失败: %w", err)
	}

	// 密码加密
	hashedPassword, err := l.hashPassword(req.Password)
	if err != nil {
		logx.Errorf("密码加密失败, email=%s, err=%v", req.Email, err)
		return nil, fmt.Errorf("注册失败，请稍后重试")
	}

	// 数据库创建用户
	if err := l.createUser(req.Nickname, req.Email, hashedPassword); err != nil {
		return nil, fmt.Errorf("用户创建失败: %w", err)
	}

	// 标记验证码已被使用
	l.markCodeAsUsed(req.Email)

	return
}

func (l *RegisterLogic) createUser(nickname, email, passwd string) error {
	// 创建用户（写入数据库）
	snowflakeId, err := utils.GenerateID()
	if err != nil {
		return fmt.Errorf("雪花id生成失败: %w", err)
	}
	_, err = l.svcCtx.UsersModel.Insert(l.ctx, &model.Users{
		SnowflakeId:  snowflakeId,
		Nickname:     nickname,
		Email:        email,
		PasswordHash: passwd,
		DeletedAt:    sql.NullTime{Valid: false},
	})
	if err != nil {
		return fmt.Errorf("数据库创建用户失败: %w", err)
	}

	return nil
}

// 检查邮箱是否被注册
func (l *RegisterLogic) checkIfEmailHasBeenRegistered(email string) error {
	_, err := l.svcCtx.UsersModel.FindOneByEmail(l.ctx, email)
	if err == nil {
		return fmt.Errorf("该邮箱已注册")
	}
	if err != sqlx.ErrNotFound {
		return fmt.Errorf("邮箱是否注册验证失败: %w", err)
	}
	return nil
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

package account_noauth

import (
	"context"
	"errors"
	"testing"
	"time"

	"user/internal/config"
	"user/internal/errs"
	"user/internal/mock"
	"user/internal/model"
	"user/internal/svc"
	"user/internal/types"
	"user/internal/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	mock2 "github.com/stretchr/testify/mock"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// setupRegisterTest 设置注册测试环境
func setupRegisterTest(t *testing.T) (*miniredis.Miniredis, *redis.Redis, *mock.UsersModel, *svc.ServiceContext) {
	// 创建 miniredis
	s := miniredis.RunT(t)

	// 创建 redis 客户端
	rds := redis.New(s.Addr())

	// 创建 mock users model
	mockUsersModel := new(mock.UsersModel)

	// 创建 service context
	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			VerifyCodeConfig: config.VerifyCodeConfig{
				Type: config.VerifyCodeType{
					Register: "register",
				},
				Redis: config.VerifyCodeRedisConfig{
					KeyPrefix: "account",
				},
			},
		},
		Redis:      rds,
		UsersModel: mockUsersModel,
		Metrics:    getTestMetrics(),
	}

	// 初始化雪花算法
	err := utils.InitSonyflake(1, "2024-01-01")
	assert.NoError(t, err)

	return s, rds, mockUsersModel, svcCtx
}

func TestRegisterLogic_Register_Success(t *testing.T) {
	s, _, mockUsersModel, svcCtx := setupRegisterTest(t)
	defer s.Close()

	ctx := context.Background()
	logic := NewRegisterLogic(ctx, svcCtx)

	// 准备测试数据
	email := "test@example.com"
	code := "123456"
	nickname := "testuser"
	password := "password123"

	// 在 redis 中设置验证码
	key := "account:register:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	// 设置 mock 期望
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, sqlx.ErrNotFound)
	mockUsersModel.On("Insert", ctx, mock2.AnythingOfType("*model.Users")).Return(&mock.SqlResult{LastID: 1, RA: 1}, nil)

	// 执行测试
	req := &types.RegisterReq{
		Email:    email,
		Password: password,
		Code:     code,
		Nickname: nickname,
	}

	resp, err := logic.Register(req)

	// 验证结果
	assert.NoError(t, err)
	// resp 在成功时可能为 nil（代码中 return 语句没有显式返回值）
	_ = resp
	mockUsersModel.AssertExpectations(t)

	// 验证验证码被标记为已使用
	used := s.HGet(key, "used")
	assert.Equal(t, "1", used)
}

func TestRegisterLogic_Register_InvalidCode(t *testing.T) {
	s, _, _, svcCtx := setupRegisterTest(t)
	defer s.Close()

	ctx := context.Background()
	logic := NewRegisterLogic(ctx, svcCtx)

	// 准备测试数据
	email := "test@example.com"
	code := "wrongcode"

	// 在 redis 中设置正确的验证码
	key := "account:register:verify:" + email
	s.HSet(key, "code", "123456")
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	// 执行测试
	req := &types.RegisterReq{
		Email:    email,
		Password: "password123",
		Code:     code,
		Nickname: "testuser",
	}

	resp, err := logic.Register(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidCode), "应该是验证码错误")
}

func TestRegisterLogic_Register_CodeExpired(t *testing.T) {
	s, _, _, svcCtx := setupRegisterTest(t)
	defer s.Close()

	ctx := context.Background()
	logic := NewRegisterLogic(ctx, svcCtx)

	// 准备测试数据
	email := "test@example.com"

	// 执行测试（redis 中没有验证码）
	req := &types.RegisterReq{
		Email:    email,
		Password: "password123",
		Code:     "123456",
		Nickname: "testuser",
	}

	resp, err := logic.Register(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidCode), "应该是验证码无效错误")
}

func TestRegisterLogic_Register_CodeAlreadyUsed(t *testing.T) {
	s, _, _, svcCtx := setupRegisterTest(t)
	defer s.Close()

	ctx := context.Background()
	logic := NewRegisterLogic(ctx, svcCtx)

	// 准备测试数据
	email := "test@example.com"
	code := "123456"

	// 在 redis 中设置已使用的验证码
	key := "account:register:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "1")
	s.SetTTL(key, 5*time.Minute)

	// 执行测试
	req := &types.RegisterReq{
		Email:    email,
		Password: "password123",
		Code:     code,
		Nickname: "testuser",
	}

	resp, err := logic.Register(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeCodeAlreadyUsed), "应该是验证码已使用错误")
}

func TestRegisterLogic_Register_EmailAlreadyRegistered(t *testing.T) {
	s, _, mockUsersModel, svcCtx := setupRegisterTest(t)
	defer s.Close()

	ctx := context.Background()
	logic := NewRegisterLogic(ctx, svcCtx)

	// 准备测试数据
	email := "test@example.com"
	code := "123456"

	// 在 redis 中设置验证码
	key := "account:register:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	// 设置 mock 期望 - 邮箱已注册
	existingUser := &model.Users{
		Id:           1,
		SnowflakeId:  123456789,
		Email:        email,
		Nickname:     "existing",
		PasswordHash: "hashedpassword",
	}
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(existingUser, nil)

	// 执行测试
	req := &types.RegisterReq{
		Email:    email,
		Password: "password123",
		Code:     code,
		Nickname: "testuser",
	}

	resp, err := logic.Register(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeEmailRegistered), "应该是邮箱已注册错误")
	mockUsersModel.AssertExpectations(t)
}

func TestRegisterLogic_Register_DatabaseError(t *testing.T) {
	s, _, mockUsersModel, svcCtx := setupRegisterTest(t)
	defer s.Close()

	ctx := context.Background()
	logic := NewRegisterLogic(ctx, svcCtx)

	// 准备测试数据
	email := "test@example.com"
	code := "123456"

	// 在 redis 中设置验证码
	key := "account:register:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	// 设置 mock 期望 - 数据库查询错误
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, errors.New("database connection failed"))

	// 执行测试
	req := &types.RegisterReq{
		Email:    email,
		Password: "password123",
		Code:     code,
		Nickname: "testuser",
	}

	resp, err := logic.Register(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

func TestRegisterLogic_Register_InsertFailed(t *testing.T) {
	s, _, mockUsersModel, svcCtx := setupRegisterTest(t)
	defer s.Close()

	ctx := context.Background()
	logic := NewRegisterLogic(ctx, svcCtx)

	// 准备测试数据
	email := "test@example.com"
	code := "123456"

	// 在 redis 中设置验证码
	key := "account:register:verify:" + email
	s.HSet(key, "code", code)
	s.HSet(key, "used", "0")
	s.SetTTL(key, 5*time.Minute)

	// 设置 mock 期望
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, sqlx.ErrNotFound)
	mockUsersModel.On("Insert", ctx, mock2.AnythingOfType("*model.Users")).Return(&mock.SqlResult{}, errors.New("insert failed"))

	// 执行测试
	req := &types.RegisterReq{
		Email:    email,
		Password: "password123",
		Code:     code,
		Nickname: "testuser",
	}

	resp, err := logic.Register(req)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

func TestRegisterLogic_createUser_Success(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupRegisterTest(t)

	ctx := context.Background()
	logic := NewRegisterLogic(ctx, svcCtx)

	nickname := "testuser"
	email := "test@example.com"
	passwd := "hashedpassword"

	// 设置 mock 期望
	mockUsersModel.On("Insert", ctx, mock2.MatchedBy(func(u *model.Users) bool {
		return u.Nickname == nickname &&
			u.Email == email &&
			u.PasswordHash == passwd &&
			u.SnowflakeId > 0 &&
			!u.DeletedAt.Valid
	})).Return(&mock.SqlResult{LastID: 1, RA: 1}, nil)

	err := logic.createUser(nickname, email, passwd)

	assert.NoError(t, err)
	mockUsersModel.AssertExpectations(t)
}

func TestRegisterLogic_createUser_InsertFailed(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupRegisterTest(t)

	ctx := context.Background()
	logic := NewRegisterLogic(ctx, svcCtx)

	nickname := "testuser"
	email := "test@example.com"
	passwd := "hashedpassword"

	// 设置 mock 期望 - 插入失败
	mockUsersModel.On("Insert", ctx, mock2.AnythingOfType("*model.Users")).Return(&mock.SqlResult{}, errors.New("insert failed"))

	err := logic.createUser(nickname, email, passwd)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

func TestRegisterLogic_checkIfEmailHasBeenRegistered_NotFound(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupRegisterTest(t)

	ctx := context.Background()
	logic := NewRegisterLogic(ctx, svcCtx)

	email := "new@example.com"

	// 设置 mock 期望 - 未找到
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, sqlx.ErrNotFound)

	err := logic.checkIfEmailHasBeenRegistered(email)

	assert.NoError(t, err)
	mockUsersModel.AssertExpectations(t)
}

func TestRegisterLogic_checkIfEmailHasBeenRegistered_Found(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupRegisterTest(t)

	ctx := context.Background()
	logic := NewRegisterLogic(ctx, svcCtx)

	email := "existing@example.com"

	// 设置 mock 期望 - 找到已存在的用户
	existingUser := &model.Users{
		Id:           1,
		Email:        email,
		Nickname:     "existing",
		PasswordHash: "hashed",
	}
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(existingUser, nil)

	err := logic.checkIfEmailHasBeenRegistered(email)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeEmailRegistered), "应该是邮箱已注册错误")
	mockUsersModel.AssertExpectations(t)
}

func TestRegisterLogic_checkIfEmailHasBeenRegistered_DatabaseError(t *testing.T) {
	_, _, mockUsersModel, svcCtx := setupRegisterTest(t)

	ctx := context.Background()
	logic := NewRegisterLogic(ctx, svcCtx)

	email := "test@example.com"

	// 设置 mock 期望 - 数据库查询出错
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, errors.New("database connection failed"))

	err := logic.checkIfEmailHasBeenRegistered(email)

	assert.Error(t, err)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

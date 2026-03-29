package account_noauth

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"user/internal/config"
	"user/internal/errs"
	"user/internal/model"
	"user/internal/svc"
	"user/internal/types"
	"user/internal/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// MockUsersModel 模拟 UsersModel
type MockUsersModel struct {
	mock.Mock
}

func (m *MockUsersModel) Insert(ctx context.Context, data *model.Users) (sql.Result, error) {
	args := m.Called(ctx, data)
	return args.Get(0).(sql.Result), args.Error(1)
}

func (m *MockUsersModel) FindOne(ctx context.Context, id int64) (*model.Users, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Users), args.Error(1)
}

func (m *MockUsersModel) FindOneByEmail(ctx context.Context, email string) (*model.Users, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Users), args.Error(1)
}

func (m *MockUsersModel) FindOneBySnowflakeId(ctx context.Context, snowflakeId int64) (*model.Users, error) {
	args := m.Called(ctx, snowflakeId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Users), args.Error(1)
}

func (m *MockUsersModel) Update(ctx context.Context, data *model.Users) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MockUsersModel) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockSqlResult 模拟 sql.Result
type MockSqlResult struct {
	lastID int64
	ra     int64
}

func (m *MockSqlResult) LastInsertId() (int64, error) {
	return m.lastID, nil
}

func (m *MockSqlResult) RowsAffected() (int64, error) {
	return m.ra, nil
}

// setupRegisterTest 设置注册测试环境
func setupRegisterTest(t *testing.T) (*miniredis.Miniredis, *redis.Redis, *MockUsersModel, *svc.ServiceContext) {
	// 创建 miniredis
	s := miniredis.RunT(t)

	// 创建 redis 客户端
	rds := redis.New(s.Addr())

	// 创建 mock users model
	mockUsersModel := new(MockUsersModel)

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
	}

	// 初始化雪花算法
	err := utils.InitSonyflake(1, "2024-01-01")
	assert.NoError(t, err)

	return s, rds, mockUsersModel, svcCtx
}

// isCodeError 检查错误是否为指定的错误码
func isCodeError(err error, code int) bool {
	if err == nil {
		return false
	}
	if e, ok := errs.IsCodeError(err); ok {
		return e.Code == code
	}
	return false
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
	mockUsersModel.On("Insert", ctx, mock.AnythingOfType("*model.Users")).Return(&MockSqlResult{lastID: 1, ra: 1}, nil)

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
	assert.True(t, isCodeError(err, errs.CodeInvalidCode), "应该是验证码错误")
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
	assert.True(t, isCodeError(err, errs.CodeInvalidCode), "应该是验证码无效错误")
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
	assert.True(t, isCodeError(err, errs.CodeCodeAlreadyUsed), "应该是验证码已使用错误")
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
	assert.True(t, isCodeError(err, errs.CodeEmailRegistered), "应该是邮箱已注册错误")
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
	assert.True(t, isCodeError(err, errs.CodeInternalError), "应该是内部错误")
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
	mockUsersModel.On("Insert", ctx, mock.AnythingOfType("*model.Users")).Return(&MockSqlResult{}, errors.New("insert failed"))

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
	assert.True(t, isCodeError(err, errs.CodeInternalError), "应该是内部错误")
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
	mockUsersModel.On("Insert", ctx, mock.MatchedBy(func(u *model.Users) bool {
		return u.Nickname == nickname &&
			u.Email == email &&
			u.PasswordHash == passwd &&
			u.SnowflakeId > 0 &&
			!u.DeletedAt.Valid
	})).Return(&MockSqlResult{lastID: 1, ra: 1}, nil)

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
	mockUsersModel.On("Insert", ctx, mock.AnythingOfType("*model.Users")).Return(&MockSqlResult{}, errors.New("insert failed"))

	err := logic.createUser(nickname, email, passwd)

	assert.Error(t, err)
	assert.True(t, isCodeError(err, errs.CodeInternalError), "应该是内部错误")
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
	assert.True(t, isCodeError(err, errs.CodeEmailRegistered), "应该是邮箱已注册错误")
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
	assert.True(t, isCodeError(err, errs.CodeInternalError), "应该是内部错误")
	mockUsersModel.AssertExpectations(t)
}

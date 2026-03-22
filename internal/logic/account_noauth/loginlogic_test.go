package account_noauth

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"user/internal/config"
	"user/internal/errs"
	"user/internal/model"
	"user/internal/svc"
	"user/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// MockUsersModelForLogin 模拟 UsersModel
type MockUsersModelForLogin struct {
	mock.Mock
}

func (m *MockUsersModelForLogin) Insert(ctx context.Context, data *model.Users) (sql.Result, error) {
	args := m.Called(ctx, data)
	return args.Get(0).(sql.Result), args.Error(1)
}

func (m *MockUsersModelForLogin) FindOne(ctx context.Context, id int64) (*model.Users, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Users), args.Error(1)
}

func (m *MockUsersModelForLogin) FindOneByEmail(ctx context.Context, email string) (*model.Users, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Users), args.Error(1)
}

func (m *MockUsersModelForLogin) FindOneBySnowflakeId(ctx context.Context, snowflakeId int64) (*model.Users, error) {
	args := m.Called(ctx, snowflakeId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Users), args.Error(1)
}

func (m *MockUsersModelForLogin) Update(ctx context.Context, data *model.Users) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MockUsersModelForLogin) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// 判断是否为 CodeError
func isCodeErrorForLogin(err error, code int) bool {
	if err == nil {
		return false
	}
	if ce, ok := err.(*errs.CodeError); ok {
		return ce.Code == code
	}
	return false
}

func TestLoginLogic_Login_Success(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(MockUsersModelForLogin)

	// 准备测试数据
	email := "test@example.com"
	password := "testpassword123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := &model.Users{
		Id:           1,
		SnowflakeId:  12345,
		Nickname:     "testuser",
		Email:        email,
		PasswordHash: string(hashedPassword),
	}

	// 设置 mock 期望
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(user, nil)

	svcCtx := &svc.ServiceContext{
		UsersModel: mockUsersModel,
		Config: config.Config{
			Auth: config.Auth{
				AccessSecret: "test-access-secret",
				AccessExpire: 3600,
			},
			RefreshSecret: "test-refresh-secret",
			RefreshExpire: 7200,
		},
	}

	logic := NewLoginLogic(ctx, svcCtx)
	req := &types.LoginReq{
		Email:    email,
		Password: password,
	}

	resp, err := logic.Login(req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, int64(3600), resp.ExpiresIn)
	mockUsersModel.AssertExpectations(t)
}

func TestLoginLogic_Login_UserNotFound(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(MockUsersModelForLogin)

	email := "notfound@example.com"
	password := "testpassword123"

	// 设置 mock 期望：用户不存在
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, sqlx.ErrNotFound)

	svcCtx := &svc.ServiceContext{
		UsersModel: mockUsersModel,
	}

	logic := NewLoginLogic(ctx, svcCtx)
	req := &types.LoginReq{
		Email:    email,
		Password: password,
	}

	resp, err := logic.Login(req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, isCodeErrorForLogin(err, errs.CodeUserNotExistOrPasswordIncorrect))
	mockUsersModel.AssertExpectations(t)
}

func TestLoginLogic_Login_DatabaseError(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(MockUsersModelForLogin)

	email := "test@example.com"
	password := "testpassword123"

	// 设置 mock 期望：数据库错误
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, errors.New("database connection failed"))

	svcCtx := &svc.ServiceContext{
		UsersModel: mockUsersModel,
	}

	// 禁用日志输出
	logx.Disable()

	logic := NewLoginLogic(ctx, svcCtx)
	req := &types.LoginReq{
		Email:    email,
		Password: password,
	}

	resp, err := logic.Login(req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, isCodeErrorForLogin(err, errs.CodeInternalError))
	mockUsersModel.AssertExpectations(t)
}

func TestLoginLogic_Login_InvalidPassword(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(MockUsersModelForLogin)

	email := "test@example.com"
	correctPassword := "correctpassword"
	wrongPassword := "wrongpassword"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(correctPassword), bcrypt.DefaultCost)
	user := &model.Users{
		Id:           1,
		SnowflakeId:  12345,
		Nickname:     "testuser",
		Email:        email,
		PasswordHash: string(hashedPassword),
	}

	// 设置 mock 期望
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(user, nil)

	svcCtx := &svc.ServiceContext{
		UsersModel: mockUsersModel,
	}

	logic := NewLoginLogic(ctx, svcCtx)
	req := &types.LoginReq{
		Email:    email,
		Password: wrongPassword,
	}

	resp, err := logic.Login(req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, isCodeErrorForLogin(err, errs.CodeInvalidPassword))
	mockUsersModel.AssertExpectations(t)
}

func TestLoginLogic_getUserByEmail_Success(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(MockUsersModelForLogin)

	email := "test@example.com"
	user := &model.Users{
		Id:          1,
		SnowflakeId: 12345,
		Nickname:    "testuser",
		Email:       email,
	}

	mockUsersModel.On("FindOneByEmail", ctx, email).Return(user, nil)

	svcCtx := &svc.ServiceContext{
		UsersModel: mockUsersModel,
	}

	logic := NewLoginLogic(ctx, svcCtx)
	result, err := logic.getUserByEmail(email)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, user, result)
	mockUsersModel.AssertExpectations(t)
}

func TestLoginLogic_getUserByEmail_NotFound(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(MockUsersModelForLogin)

	email := "notfound@example.com"

	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, sqlx.ErrNotFound)

	svcCtx := &svc.ServiceContext{
		UsersModel: mockUsersModel,
	}

	logic := NewLoginLogic(ctx, svcCtx)
	result, err := logic.getUserByEmail(email)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, isCodeErrorForLogin(err, errs.CodeUserNotExistOrPasswordIncorrect))
	mockUsersModel.AssertExpectations(t)
}

func TestLoginLogic_getUserByEmail_DatabaseError(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(MockUsersModelForLogin)

	email := "test@example.com"

	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, errors.New("database error"))

	svcCtx := &svc.ServiceContext{
		UsersModel: mockUsersModel,
	}

	logx.Disable()

	logic := NewLoginLogic(ctx, svcCtx)
	result, err := logic.getUserByEmail(email)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, isCodeErrorForLogin(err, errs.CodeInternalError))
	mockUsersModel.AssertExpectations(t)
}

func TestLoginLogic_verifyPassword_Success(t *testing.T) {
	ctx := context.Background()
	svcCtx := &svc.ServiceContext{}
	logic := NewLoginLogic(ctx, svcCtx)

	password := "testpassword123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	err := logic.verifyPassword(string(hashedPassword), password, "test@example.com")

	assert.NoError(t, err)
}

func TestLoginLogic_verifyPassword_InvalidPassword(t *testing.T) {
	ctx := context.Background()
	svcCtx := &svc.ServiceContext{}
	logic := NewLoginLogic(ctx, svcCtx)

	correctPassword := "correctpassword"
	wrongPassword := "wrongpassword"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(correctPassword), bcrypt.DefaultCost)

	err := logic.verifyPassword(string(hashedPassword), wrongPassword, "test@example.com")

	assert.Error(t, err)
	assert.True(t, isCodeErrorForLogin(err, errs.CodeInvalidPassword))
}

func TestLoginLogic_verifyPassword_HashError(t *testing.T) {
	ctx := context.Background()
	svcCtx := &svc.ServiceContext{}
	logic := NewLoginLogic(ctx, svcCtx)

	logx.Disable()

	// 使用无效的哈希
	err := logic.verifyPassword("invalidhash", "password", "test@example.com")

	assert.Error(t, err)
	assert.True(t, isCodeErrorForLogin(err, errs.CodeInternalError))
}

func TestLoginLogic_generateAccessToken_Success(t *testing.T) {
	ctx := context.Background()
	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			Auth: config.Auth{
				AccessSecret: "test-access-secret",
				AccessExpire: 3600,
			},
		},
	}

	logic := NewLoginLogic(ctx, svcCtx)
	user := &model.Users{
		SnowflakeId: 12345,
		Nickname:    "testuser",
		Email:       "test@example.com",
	}

	token, err := logic.generateAccessToken(user)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestLoginLogic_generateRefreshToken_Success(t *testing.T) {
	ctx := context.Background()
	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			RefreshSecret: "test-refresh-secret",
			RefreshExpire: 7200,
		},
	}

	logic := NewLoginLogic(ctx, svcCtx)
	user := &model.Users{
		SnowflakeId: 12345,
		Email:       "test@example.com",
	}

	token, err := logic.generateRefreshToken(user)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestLoginLogic_buildLoginResponse(t *testing.T) {
	ctx := context.Background()
	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			Auth: config.Auth{
				AccessExpire: 3600,
			},
		},
	}

	logic := NewLoginLogic(ctx, svcCtx)
	accessToken := "test-access-token"
	refreshToken := "test-refresh-token"

	resp := logic.buildLoginResponse(accessToken, refreshToken)

	assert.NotNil(t, resp)
	assert.Equal(t, accessToken, resp.AccessToken)
	assert.Equal(t, refreshToken, resp.RefreshToken)
	assert.Equal(t, int64(3600), resp.ExpiresIn)
}

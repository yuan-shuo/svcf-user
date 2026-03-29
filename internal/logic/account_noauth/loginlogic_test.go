package account_noauth

import (
	"context"
	"errors"
	"testing"

	"user/internal/config"
	"user/internal/errs"
	"user/internal/mock"
	"user/internal/model"
	"user/internal/svc"
	"user/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"golang.org/x/crypto/bcrypt"
)

func TestLoginLogic_Login_Success_WithRememberMe(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(mock.UsersModel)

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
		Email:      email,
		Password:   password,
		RememberMe: true, // 选择记住我
	}

	resp, err := logic.Login(req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken) // RememberMe=true 时应该有 refreshToken
	assert.Equal(t, int64(3600), resp.ExpiresIn)
	mockUsersModel.AssertExpectations(t)
}

func TestLoginLogic_Login_Success_WithoutRememberMe(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(mock.UsersModel)

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
			// 不需要 RefreshSecret 和 RefreshExpire，因为不签发 RT
		},
	}

	logic := NewLoginLogic(ctx, svcCtx)
	req := &types.LoginReq{
		Email:      email,
		Password:   password,
		RememberMe: false, // 不选择记住我
	}

	resp, err := logic.Login(req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.Empty(t, resp.RefreshToken) // RememberMe=false 时不应该有 refreshToken
	assert.Equal(t, int64(3600), resp.ExpiresIn)
	mockUsersModel.AssertExpectations(t)
}

func TestLoginLogic_Login_UserNotFound(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(mock.UsersModel)

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
	assert.True(t, mock.IsCodeError(err, errs.CodeUserNotExistOrPasswordIncorrect))
	mockUsersModel.AssertExpectations(t)
}

func TestLoginLogic_Login_DatabaseError(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(mock.UsersModel)

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
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError))
	mockUsersModel.AssertExpectations(t)
}

func TestLoginLogic_Login_InvalidPassword(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(mock.UsersModel)

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
	assert.True(t, mock.IsCodeError(err, errs.CodeUserNotExistOrPasswordIncorrect))
	mockUsersModel.AssertExpectations(t)
}

func TestLoginLogic_getUserByEmail_Success(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(mock.UsersModel)

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
	mockUsersModel := new(mock.UsersModel)

	email := "notfound@example.com"

	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, sqlx.ErrNotFound)

	svcCtx := &svc.ServiceContext{
		UsersModel: mockUsersModel,
	}

	logic := NewLoginLogic(ctx, svcCtx)
	result, err := logic.getUserByEmail(email)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, mock.IsCodeError(err, errs.CodeUserNotExistOrPasswordIncorrect))
	mockUsersModel.AssertExpectations(t)
}

func TestLoginLogic_getUserByEmail_DatabaseError(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(mock.UsersModel)

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
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError))
	mockUsersModel.AssertExpectations(t)
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

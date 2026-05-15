package user_noauth

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
		Metrics: mock.GetTestMetrics(),
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
	// Token 通过 Cookie 返回，不再在响应体中
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
		Metrics: mock.GetTestMetrics(),
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
	// Token 通过 Cookie 返回，不再在响应体中
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
		Metrics:    mock.GetTestMetrics(),
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
		Metrics:    mock.GetTestMetrics(),
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
		Metrics:    mock.GetTestMetrics(),
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

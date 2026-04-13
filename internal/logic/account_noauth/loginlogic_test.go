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

	// 鍑嗗娴嬭瘯鏁版嵁
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

	// 璁剧疆 mock 鏈熸湜
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
		RememberMe: true, // 閫夋嫨璁颁綇鎴?
	}

	resp, err := logic.Login(req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken) // RememberMe=true 鏃跺簲璇ユ湁 refreshToken
	assert.Equal(t, int64(3600), resp.ExpiresIn)
	mockUsersModel.AssertExpectations(t)
}

func TestLoginLogic_Login_Success_WithoutRememberMe(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(mock.UsersModel)

	// 鍑嗗娴嬭瘯鏁版嵁
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

	// 璁剧疆 mock 鏈熸湜
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(user, nil)

	svcCtx := &svc.ServiceContext{
		UsersModel: mockUsersModel,
		Config: config.Config{
			Auth: config.Auth{
				AccessSecret: "test-access-secret",
				AccessExpire: 3600,
			},
			// 涓嶉渶瑕?RefreshSecret 鍜?RefreshExpire锛屽洜涓轰笉绛惧彂 RT
		},
		Metrics: mock.GetTestMetrics(),
	}

	logic := NewLoginLogic(ctx, svcCtx)
	req := &types.LoginReq{
		Email:      email,
		Password:   password,
		RememberMe: false, // 涓嶉€夋嫨璁颁綇鎴?
	}

	resp, err := logic.Login(req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.Empty(t, resp.RefreshToken) // RememberMe=false 鏃朵笉搴旇鏈?refreshToken
	assert.Equal(t, int64(3600), resp.ExpiresIn)
	mockUsersModel.AssertExpectations(t)
}

func TestLoginLogic_Login_UserNotFound(t *testing.T) {
	ctx := context.Background()
	mockUsersModel := new(mock.UsersModel)

	email := "notfound@example.com"
	password := "testpassword123"

	// 璁剧疆 mock 鏈熸湜锛氱敤鎴蜂笉瀛樺湪
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

	// 璁剧疆 mock 鏈熸湜锛氭暟鎹簱閿欒
	mockUsersModel.On("FindOneByEmail", ctx, email).Return(nil, errors.New("database connection failed"))

	svcCtx := &svc.ServiceContext{
		UsersModel: mockUsersModel,
		Metrics:    mock.GetTestMetrics(),
	}

	// 绂佺敤鏃ュ織杈撳嚭
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

	// 璁剧疆 mock 鏈熸湜
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

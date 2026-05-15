package user_noauth

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	mock2 "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/stores/redis"

	"user/internal/config"
	"user/internal/errs"
	"user/internal/logic/userutils"
	"user/internal/mock"
	"user/internal/model"
	"user/internal/svc"
	"user/internal/types"
)

// setupSendVerifyCodeTest 创建测试用的环境和逻辑
func setupSendVerifyCodeTest(t *testing.T) (*SendVerifyCodeLogic, *miniredis.Miniredis, *mock.UsersModel, *mock.KqPusherClient) {
	s := miniredis.RunT(t)

	conf := redis.RedisConf{
		Host: s.Addr(),
		Type: "node",
	}
	r := redis.MustNewRedis(conf)

	mockUsers := new(mock.UsersModel)
	mockMQ := new(mock.KqPusherClient)

	ctx := context.Background()
	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			VerifyCodeConfig: config.VerifyCodeConfig{
				Type: config.VerifyCodeType{
					Register:         "register",
					ResetPassword:    "reset_password",
					ChangePassword:   "change_password",
					RemindRegistered: "remind_registered",
				},
				Time: config.VerifyCodeTime{
					ExpireIn:   300,
					RetryAfter: 60,
				},
			},
		},
		Redis:          r,
		KqPusherClient: mockMQ,
		UsersModel:     mockUsers,
		Metrics:        mock.GetTestMetrics(),
	}

	logic := NewSendVerifyCodeLogic(ctx, svcCtx)
	return logic, s, mockUsers, mockMQ
}

func TestSendVerifyCodeLogic_SendVerifyCode_RegisterSuccess(t *testing.T) {
	logic, s, mockUsers, mockMQ := setupSendVerifyCodeTest(t)
	defer s.Close()

	// 设置 mock 期望：邮箱未注册
	mockUsers.On("FindOneByEmail", mock2.Anything, "newuser@example.com").Return(nil, model.ErrNotFound)
	// 设置 mock 期望：MQ 推送成功
	mockMQ.On("Push", mock2.Anything, mock2.AnythingOfType("string")).Return(nil)

	req := &types.SendVerifyCodeReq{
		Email: "newuser@example.com",
		Type:  "register",
	}

	resp, err := logic.SendVerifyCode(req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 60, resp.RetryAfter)

	// 验证验证码已保存到 Redis
	verifyKey := userutils.BuildVerifyKey(req.Email, req.Type)
	exists, _ := logic.svcCtx.Redis.ExistsCtx(context.Background(), verifyKey)
	assert.True(t, exists, "验证码应该已保存到 Redis")

	mockUsers.AssertExpectations(t)
	mockMQ.AssertExpectations(t)
}

func TestSendVerifyCodeLogic_SendVerifyCode_ResetPasswordSuccess(t *testing.T) {
	logic, s, mockUsers, mockMQ := setupSendVerifyCodeTest(t)
	defer s.Close()

	// 设置 mock 期望：邮箱已存在
	mockUsers.On("FindOneByEmail", mock2.Anything, "existing@example.com").Return(&model.Users{
		Id:           1,
		Email:        "existing@example.com",
		PasswordHash: "hashedpassword",
	}, nil)
	// 设置 mock 期望：MQ 推送成功
	mockMQ.On("Push", mock2.Anything, mock2.AnythingOfType("string")).Return(nil)

	req := &types.SendVerifyCodeReq{
		Email: "existing@example.com",
		Type:  "reset_password",
	}

	resp, err := logic.SendVerifyCode(req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	// 验证验证码已保存到 Redis
	verifyKey := userutils.BuildVerifyKey(req.Email, req.Type)
	exists, _ := logic.svcCtx.Redis.ExistsCtx(context.Background(), verifyKey)
	assert.True(t, exists, "验证码应该已保存到 Redis")

	mockUsers.AssertExpectations(t)
	mockMQ.AssertExpectations(t)
}

func TestSendVerifyCodeLogic_SendVerifyCode_EmailAlreadyRegistered(t *testing.T) {
	logic, s, mockUsers, mockMQ := setupSendVerifyCodeTest(t)
	defer s.Close()

	// 设置 mock 期望：邮箱已存在
	mockUsers.On("FindOneByEmail", mock2.Anything, "existing@example.com").Return(&model.Users{
		Id:           1,
		Email:        "existing@example.com",
		PasswordHash: "hashedpassword",
	}, nil)
	// 设置 mock 期望：发送提醒邮件
	mockMQ.On("Push", mock2.Anything, mock2.AnythingOfType("string")).Return(nil)

	req := &types.SendVerifyCodeReq{
		Email: "existing@example.com",
		Type:  "register",
	}

	resp, err := logic.SendVerifyCode(req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	// 邮箱已注册时，不发送验证码但返回成功

	mockUsers.AssertExpectations(t)
	mockMQ.AssertExpectations(t)
}

func TestSendVerifyCodeLogic_SendVerifyCode_EmailNotRegisteredForReset(t *testing.T) {
	logic, s, mockUsers, _ := setupSendVerifyCodeTest(t)
	defer s.Close()

	// 设置 mock 期望：邮箱不存在
	mockUsers.On("FindOneByEmail", mock2.Anything, "nonexistent@example.com").Return(nil, model.ErrNotFound)

	req := &types.SendVerifyCodeReq{
		Email: "nonexistent@example.com",
		Type:  "reset_password",
	}

	resp, err := logic.SendVerifyCode(req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeEmailNotRegistered), "应该是邮箱未注册错误")

	mockUsers.AssertExpectations(t)
}

func TestSendVerifyCodeLogic_SendVerifyCode_InvalidEmail(t *testing.T) {
	logic, s, _, _ := setupSendVerifyCodeTest(t)
	defer s.Close()

	req := &types.SendVerifyCodeReq{
		Email: "invalid-email",
		Type:  "register",
	}

	resp, err := logic.SendVerifyCode(req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidParam), "应该是参数错误")
}

func TestSendVerifyCodeLogic_SendVerifyCode_InvalidType(t *testing.T) {
	logic, s, _, _ := setupSendVerifyCodeTest(t)
	defer s.Close()

	req := &types.SendVerifyCodeReq{
		Email: "test@example.com",
		Type:  "invalid_type",
	}

	resp, err := logic.SendVerifyCode(req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInvalidParam), "应该是参数错误")
}

func TestSendVerifyCodeLogic_SendVerifyCode_RateLimit(t *testing.T) {
	logic, s, mockUsers, mockMQ := setupSendVerifyCodeTest(t)
	defer s.Close()

	email := "test@example.com"

	// 设置 mock 期望：邮箱未注册
	mockUsers.On("FindOneByEmail", mock2.Anything, email).Return(nil, model.ErrNotFound)
	// 设置 mock 期望：MQ 推送成功
	mockMQ.On("Push", mock2.Anything, mock2.AnythingOfType("string")).Return(nil)

	// 第一次请求应该成功
	req := &types.SendVerifyCodeReq{
		Email: email,
		Type:  "register",
	}
	resp, err := logic.SendVerifyCode(req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// 第二次请求应该被限流
	resp2, err2 := logic.SendVerifyCode(req)
	require.Error(t, err2)
	assert.Nil(t, resp2)
	assert.True(t, mock.IsCodeError(err2, errs.CodeInvalidParam), "应该是参数错误（限流）")
	assert.Contains(t, err2.Error(), "发送过于频繁")

	mockUsers.AssertExpectations(t)
	mockMQ.AssertExpectations(t)
}

func TestSendVerifyCodeLogic_SendVerifyCode_MQFailed(t *testing.T) {
	logic, s, mockUsers, mockMQ := setupSendVerifyCodeTest(t)
	defer s.Close()

	email := "test@example.com"

	// 设置 mock 期望：邮箱未注册
	mockUsers.On("FindOneByEmail", mock2.Anything, email).Return(nil, model.ErrNotFound)
	// 设置 mock 期望：MQ 推送失败
	mockMQ.On("Push", mock2.Anything, mock2.AnythingOfType("string")).Return(errors.New("mq connection failed"))

	req := &types.SendVerifyCodeReq{
		Email: email,
		Type:  "register",
	}
	resp, err := logic.SendVerifyCode(req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")

	mockUsers.AssertExpectations(t)
	mockMQ.AssertExpectations(t)
}

func TestSendVerifyCodeLogic_SendVerifyCode_DBError(t *testing.T) {
	logic, s, mockUsers, _ := setupSendVerifyCodeTest(t)
	defer s.Close()

	email := "test@example.com"

	// 设置 mock 期望：数据库错误
	mockUsers.On("FindOneByEmail", mock2.Anything, email).Return(nil, errors.New("database connection failed"))

	req := &types.SendVerifyCodeReq{
		Email: email,
		Type:  "register",
	}
	resp, err := logic.SendVerifyCode(req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, mock.IsCodeError(err, errs.CodeInternalError), "应该是内部错误")

	mockUsers.AssertExpectations(t)
}

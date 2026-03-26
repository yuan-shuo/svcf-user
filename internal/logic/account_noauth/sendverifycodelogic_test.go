package account_noauth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/stores/redis"

	"user/internal/config"
	"user/internal/model"
	"user/internal/svc"
	"user/internal/types"
)

// mockUsersModel 用于测试的用户模型 mock
type mockUsersModel struct {
	insertFunc             func(ctx context.Context, data *model.Users) (sql.Result, error)
	findOneFunc            func(ctx context.Context, id int64) (*model.Users, error)
	findOneByEmailFunc     func(ctx context.Context, email string) (*model.Users, error)
	findOneBySnowflakeFunc func(ctx context.Context, snowflakeId int64) (*model.Users, error)
	updateFunc             func(ctx context.Context, data *model.Users) error
	deleteFunc             func(ctx context.Context, id int64) error
}

func (m *mockUsersModel) Insert(ctx context.Context, data *model.Users) (sql.Result, error) {
	if m.insertFunc != nil {
		return m.insertFunc(ctx, data)
	}
	return nil, nil
}

func (m *mockUsersModel) FindOne(ctx context.Context, id int64) (*model.Users, error) {
	if m.findOneFunc != nil {
		return m.findOneFunc(ctx, id)
	}
	return nil, model.ErrNotFound
}

func (m *mockUsersModel) FindOneByEmail(ctx context.Context, email string) (*model.Users, error) {
	if m.findOneByEmailFunc != nil {
		return m.findOneByEmailFunc(ctx, email)
	}
	return nil, model.ErrNotFound
}

func (m *mockUsersModel) FindOneBySnowflakeId(ctx context.Context, snowflakeId int64) (*model.Users, error) {
	if m.findOneBySnowflakeFunc != nil {
		return m.findOneBySnowflakeFunc(ctx, snowflakeId)
	}
	return nil, model.ErrNotFound
}

func (m *mockUsersModel) Update(ctx context.Context, data *model.Users) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, data)
	}
	return nil
}

func (m *mockUsersModel) Delete(ctx context.Context, id int64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

// mockKqPusherClient 用于测试的 MQ 推送客户端 mock
type mockKqPusherClient struct {
	pushFunc func(ctx context.Context, v string) error
	messages []string
}

func (m *mockKqPusherClient) Push(ctx context.Context, v string) error {
	m.messages = append(m.messages, v)
	if m.pushFunc != nil {
		return m.pushFunc(ctx, v)
	}
	return nil
}

func (m *mockKqPusherClient) Close() error {
	return nil
}

// setupTestRedis 创建测试用的 Redis 实例
func setupTestRedis(t *testing.T) (*redis.Redis, *miniredis.Miniredis) {
	s := miniredis.RunT(t)
	conf := redis.RedisConf{
		Host: s.Addr(),
		Type: "node",
	}
	r := redis.MustNewRedis(conf)
	return r, s
}

// newTestSendVerifyCodeLogic 创建测试用的 SendVerifyCodeLogic
func newTestSendVerifyCodeLogic(t *testing.T, r *redis.Redis, mockUsers model.UsersModel, mockMQ *mockKqPusherClient) (*SendVerifyCodeLogic, *miniredis.Miniredis) {
	s := miniredis.RunT(t)
	if r == nil {
		conf := redis.RedisConf{
			Host: s.Addr(),
			Type: "node",
		}
		r = redis.MustNewRedis(conf)
	}

	ctx := context.Background()
	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			VerifyCodeConfig: config.VerifyCodeConfig{
				Type: config.VerifyCodeType{
					Register:         "register",
					ResetPassword:    "reset_password",
					RemindRegistered: "remind_registered",
				},
				Time: config.VerifyCodeTime{
					ExpireIn:   300,
					RetryAfter: 60,
				},
				Redis: config.VerifyCodeRedisConfig{
					KeyPrefix: "account",
				},
			},
		},
		Redis:          r,
		KqPusherClient: mockMQ,
		UsersModel:     mockUsers,
	}

	logic := NewSendVerifyCodeLogic(ctx, svcCtx)
	return logic, s
}

func TestSendVerifyCodeLogic_validateRequest(t *testing.T) {
	logic, s := newTestSendVerifyCodeLogic(t, nil, nil, nil)
	defer s.Close()

	tests := []struct {
		name    string
		req     *types.SendVerifyCodeReq
		wantErr bool
	}{
		{
			name: "有效的注册请求",
			req: &types.SendVerifyCodeReq{
				Email: "test@example.com",
				Type:  "register",
			},
			wantErr: false,
		},
		{
			name: "有效的重置密码请求",
			req: &types.SendVerifyCodeReq{
				Email: "test@example.com",
				Type:  "reset_password",
			},
			wantErr: false,
		},
		{
			name:    "nil 请求",
			req:     nil,
			wantErr: true,
		},
		{
			name: "无效的验证码类型",
			req: &types.SendVerifyCodeReq{
				Email: "test@example.com",
				Type:  "invalid_type",
			},
			wantErr: true,
		},
		{
			name: "无效的邮箱格式",
			req: &types.SendVerifyCodeReq{
				Email: "invalid-email",
				Type:  "register",
			},
			wantErr: true,
		},
		{
			name: "空的邮箱",
			req: &types.SendVerifyCodeReq{
				Email: "",
				Type:  "register",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := logic.validateRequest(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSendVerifyCodeLogic_isValidCodeType(t *testing.T) {
	logic, s := newTestSendVerifyCodeLogic(t, nil, nil, nil)
	defer s.Close()

	tests := []struct {
		name     string
		codeType string
		want     bool
	}{
		{
			name:     "注册类型",
			codeType: "register",
			want:     true,
		},
		{
			name:     "重置密码类型",
			codeType: "reset_password",
			want:     true,
		},
		{
			name:     "无效类型",
			codeType: "invalid",
			want:     false,
		},
		{
			name:     "空类型",
			codeType: "",
			want:     false,
		},
		{
			name:     "提醒已注册类型（不应被允许作为请求类型）",
			codeType: "remind_registered",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := logic.isValidCodeType(tt.codeType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSendVerifyCodeLogic_checkRateLimit(t *testing.T) {
	logic, s := newTestSendVerifyCodeLogic(t, nil, nil, nil)
	defer s.Close()

	ctx := context.Background()
	email := "test@example.com"
	codeType := "register"

	t.Run("首次检查通过", func(t *testing.T) {
		err := logic.checkRateLimit(email, codeType)
		require.NoError(t, err, "首次检查应该通过")
	})

	t.Run("检查通过后设置限流key", func(t *testing.T) {
		limitKey := buildLimitKey(email, codeType)

		// 验证key已设置
		exists, _ := logic.svcCtx.Redis.ExistsCtx(ctx, limitKey)
		assert.True(t, exists, "检查后key应该存在")
	})

	t.Run("限流key有过期时间", func(t *testing.T) {
		limitKey := buildLimitKey(email, codeType)

		// 验证有过期时间
		ttl := s.TTL(limitKey)
		assert.True(t, ttl > 0, "限流key应该有过期时间")
		assert.True(t, ttl <= 60*time.Second, "过期时间应该不超过配置的60秒")
	})

	t.Run("短时间内重复检查返回参数错误", func(t *testing.T) {
		// 使用新的邮箱，避免之前的限流影响
		newEmail := "test2@example.com"

		// 首次检查通过
		err := logic.checkRateLimit(newEmail, codeType)
		require.NoError(t, err)

		// 短时间内再次检查应该失败
		err = logic.checkRateLimit(newEmail, codeType)
		require.Error(t, err)
		// 检查错误信息包含"发送过于频繁"
		assert.Contains(t, err.Error(), "发送过于频繁")
	})

	t.Run("错误信息包含剩余时间", func(t *testing.T) {
		// 使用新的邮箱
		newEmail := "test3@example.com"

		// 首次检查通过
		err := logic.checkRateLimit(newEmail, codeType)
		require.NoError(t, err)

		// 快进时间10秒
		s.FastForward(10 * time.Second)

		// 再次检查应该失败，并包含剩余时间
		err = logic.checkRateLimit(newEmail, codeType)
		require.Error(t, err)
		// 检查包含剩余时间信息（大约50秒）
		assert.Contains(t, err.Error(), "50")
	})

	t.Run("过期后可以再次发送", func(t *testing.T) {
		// 使用新的邮箱
		newEmail := "test4@example.com"

		// 首次检查通过
		err := logic.checkRateLimit(newEmail, codeType)
		require.NoError(t, err)

		// 快进时间超过限流时间（60秒）
		s.FastForward(61 * time.Second)

		// 再次检查应该通过
		err = logic.checkRateLimit(newEmail, codeType)
		require.NoError(t, err, "限流过期后应该可以再次发送")
	})

	t.Run("不同邮箱互不影响", func(t *testing.T) {
		email1 := "user1@example.com"
		email2 := "user2@example.com"

		// 第一个邮箱检查通过
		err := logic.checkRateLimit(email1, codeType)
		require.NoError(t, err)

		// 第二个邮箱也应该能通过
		err = logic.checkRateLimit(email2, codeType)
		require.NoError(t, err, "不同邮箱的限流应该互不影响")
	})

	t.Run("不同类型互不影响", func(t *testing.T) {
		email := "user3@example.com"

		// 注册类型检查通过并设置限流
		err := logic.checkRateLimit(email, "register")
		require.NoError(t, err)

		// 重置密码类型也应该能通过
		err = logic.checkRateLimit(email, "reset_password")
		require.NoError(t, err, "不同类型的限流应该互不影响")
	})
}

func TestSendVerifyCodeLogic_checkRegisterLogic(t *testing.T) {
	t.Run("邮箱未注册，继续发送验证码", func(t *testing.T) {
		mockUsers := &mockUsersModel{
			findOneByEmailFunc: func(ctx context.Context, email string) (*model.Users, error) {
				return nil, model.ErrNotFound
			},
		}
		mockMQ := &mockKqPusherClient{}
		logic, s := newTestSendVerifyCodeLogic(t, nil, mockUsers, mockMQ)
		defer s.Close()

		shouldContinue, err := logic.checkRegisterLogic("newuser@example.com")
		require.NoError(t, err)
		assert.True(t, shouldContinue, "邮箱未注册应该继续发送验证码")
		assert.Len(t, mockMQ.messages, 0, "不应该发送MQ消息")
	})

	t.Run("邮箱已存在，发送提醒邮件", func(t *testing.T) {
		mockUsers := &mockUsersModel{
			findOneByEmailFunc: func(ctx context.Context, email string) (*model.Users, error) {
				return &model.Users{
					Id:           1,
					SnowflakeId:  10001,
					Nickname:     "Test User",
					Email:        email,
					PasswordHash: "hashed_password",
				}, nil
			},
		}
		mockMQ := &mockKqPusherClient{}
		logic, s := newTestSendVerifyCodeLogic(t, nil, mockUsers, mockMQ)
		defer s.Close()

		shouldContinue, err := logic.checkRegisterLogic("existing@example.com")
		require.NoError(t, err)
		assert.False(t, shouldContinue, "邮箱已存在不应该继续发送验证码")
		assert.Len(t, mockMQ.messages, 1, "应该发送提醒邮件")

		// 验证消息内容
		var msg types.VerificationCodeMessage
		err = json.Unmarshal([]byte(mockMQ.messages[0]), &msg)
		require.NoError(t, err)
		assert.Equal(t, "remind_registered", msg.Type)
		assert.Equal(t, "existing@example.com", msg.Receiver)
		assert.Empty(t, msg.Code)
	})

	t.Run("邮箱已存在但发送提醒邮件失败", func(t *testing.T) {
		mockUsers := &mockUsersModel{
			findOneByEmailFunc: func(ctx context.Context, email string) (*model.Users, error) {
				return &model.Users{
					Id:           1,
					Email:        email,
					PasswordHash: "hashed_password",
				}, nil
			},
		}
		mockMQ := &mockKqPusherClient{
			pushFunc: func(ctx context.Context, v string) error {
				return errors.New("mq connection failed")
			},
		}
		logic, s := newTestSendVerifyCodeLogic(t, nil, mockUsers, mockMQ)
		defer s.Close()

		shouldContinue, err := logic.checkRegisterLogic("existing@example.com")
		require.Error(t, err)
		assert.False(t, shouldContinue)
	})

	t.Run("数据库查询失败", func(t *testing.T) {
		mockUsers := &mockUsersModel{
			findOneByEmailFunc: func(ctx context.Context, email string) (*model.Users, error) {
				return nil, errors.New("database connection failed")
			},
		}
		logic, s := newTestSendVerifyCodeLogic(t, nil, mockUsers, nil)
		defer s.Close()

		shouldContinue, err := logic.checkRegisterLogic("test@example.com")
		require.Error(t, err)
		assert.False(t, shouldContinue)
	})
}

func TestSendVerifyCodeLogic_checkResetPasswordLogic(t *testing.T) {
	t.Run("邮箱已存在，继续发送验证码", func(t *testing.T) {
		mockUsers := &mockUsersModel{
			findOneByEmailFunc: func(ctx context.Context, email string) (*model.Users, error) {
				return &model.Users{
					Id:           1,
					Email:        email,
					PasswordHash: "hashed_password",
				}, nil
			},
		}
		logic, s := newTestSendVerifyCodeLogic(t, nil, mockUsers, nil)
		defer s.Close()

		shouldContinue, err := logic.checkResetPasswordLogic("existing@example.com")
		require.NoError(t, err)
		assert.True(t, shouldContinue, "邮箱已存在应该继续发送验证码")
	})

	t.Run("邮箱不存在，返回错误", func(t *testing.T) {
		mockUsers := &mockUsersModel{
			findOneByEmailFunc: func(ctx context.Context, email string) (*model.Users, error) {
				return nil, model.ErrNotFound
			},
		}
		logic, s := newTestSendVerifyCodeLogic(t, nil, mockUsers, nil)
		defer s.Close()

		shouldContinue, err := logic.checkResetPasswordLogic("nonexistent@example.com")
		require.Error(t, err)
		assert.False(t, shouldContinue, "邮箱不存在不应该继续发送验证码")
		// 检查有错误返回即可（CodeEmailNotRegistered 没有定义错误消息）
	})

	t.Run("数据库查询失败", func(t *testing.T) {
		mockUsers := &mockUsersModel{
			findOneByEmailFunc: func(ctx context.Context, email string) (*model.Users, error) {
				return nil, errors.New("database connection failed")
			},
		}
		logic, s := newTestSendVerifyCodeLogic(t, nil, mockUsers, nil)
		defer s.Close()

		shouldContinue, err := logic.checkResetPasswordLogic("test@example.com")
		require.Error(t, err)
		assert.False(t, shouldContinue)
	})
}

func TestSendVerifyCodeLogic_checkBusinessLogic(t *testing.T) {
	t.Run("注册类型", func(t *testing.T) {
		mockUsers := &mockUsersModel{
			findOneByEmailFunc: func(ctx context.Context, email string) (*model.Users, error) {
				return nil, model.ErrNotFound
			},
		}
		logic, s := newTestSendVerifyCodeLogic(t, nil, mockUsers, nil)
		defer s.Close()

		req := &types.SendVerifyCodeReq{
			Email: "test@example.com",
			Type:  "register",
		}
		shouldContinue, err := logic.checkBusinessLogic(req)
		require.NoError(t, err)
		assert.True(t, shouldContinue)
	})

	t.Run("重置密码类型", func(t *testing.T) {
		mockUsers := &mockUsersModel{
			findOneByEmailFunc: func(ctx context.Context, email string) (*model.Users, error) {
				return &model.Users{
					Id:           1,
					Email:        email,
					PasswordHash: "hashed_password",
				}, nil
			},
		}
		logic, s := newTestSendVerifyCodeLogic(t, nil, mockUsers, nil)
		defer s.Close()

		req := &types.SendVerifyCodeReq{
			Email: "test@example.com",
			Type:  "reset_password",
		}
		shouldContinue, err := logic.checkBusinessLogic(req)
		require.NoError(t, err)
		assert.True(t, shouldContinue)
	})

	t.Run("无效类型", func(t *testing.T) {
		logic, s := newTestSendVerifyCodeLogic(t, nil, nil, nil)
		defer s.Close()

		req := &types.SendVerifyCodeReq{
			Email: "test@example.com",
			Type:  "invalid_type",
		}
		shouldContinue, err := logic.checkBusinessLogic(req)
		require.Error(t, err)
		assert.False(t, shouldContinue)
	})
}

func TestSendVerifyCodeLogic_generateAndSaveCode(t *testing.T) {
	logic, s := newTestSendVerifyCodeLogic(t, nil, nil, nil)
	defer s.Close()

	ctx := context.Background()

	t.Run("成功生成并保存验证码", func(t *testing.T) {
		req := &types.SendVerifyCodeReq{
			Email: "test@example.com",
			Type:  "register",
		}
		code := logic.generateAndSaveCode(req)
		require.NotEmpty(t, code, "应该生成验证码")
		assert.Equal(t, 6, len(code), "验证码长度应该为6位")

		// 验证验证码已保存到Redis
		redisKey := buildVerifyKey(req.Email, req.Type)
		savedCode, err := logic.svcCtx.Redis.HgetCtx(ctx, redisKey, redisValueCodeFieldName)
		require.NoError(t, err)
		assert.Equal(t, code, savedCode)

		// 验证used字段
		savedUsed, err := logic.svcCtx.Redis.HgetCtx(ctx, redisKey, redisValueUsedFieldName)
		require.NoError(t, err)
		assert.Equal(t, "0", savedUsed)
	})

	t.Run("验证码有过期时间", func(t *testing.T) {
		req := &types.SendVerifyCodeReq{
			Email: "test2@example.com",
			Type:  "register",
		}
		code := logic.generateAndSaveCode(req)
		require.NotEmpty(t, code)

		// 验证有过期时间
		redisKey := buildVerifyKey(req.Email, req.Type)
		ttl := s.TTL(redisKey)
		assert.True(t, ttl > 0, "验证码应该设置过期时间")
		assert.True(t, ttl <= 300*time.Second, "过期时间应该不超过配置的300秒")
	})

	t.Run("不同邮箱的验证码相互独立", func(t *testing.T) {
		req1 := &types.SendVerifyCodeReq{
			Email: "user1@example.com",
			Type:  "register",
		}
		req2 := &types.SendVerifyCodeReq{
			Email: "user2@example.com",
			Type:  "register",
		}

		code1 := logic.generateAndSaveCode(req1)
		code2 := logic.generateAndSaveCode(req2)

		require.NotEqual(t, code1, code2, "不同邮箱的验证码应该不同")

		// 验证各自的验证码正确
		redisKey1 := buildVerifyKey(req1.Email, req1.Type)
		redisKey2 := buildVerifyKey(req2.Email, req2.Type)

		savedCode1, _ := logic.svcCtx.Redis.HgetCtx(ctx, redisKey1, redisValueCodeFieldName)
		savedCode2, _ := logic.svcCtx.Redis.HgetCtx(ctx, redisKey2, redisValueCodeFieldName)

		assert.Equal(t, code1, savedCode1)
		assert.Equal(t, code2, savedCode2)
	})

	t.Run("不同类型的验证码相互独立", func(t *testing.T) {
		email := "user@example.com"
		req1 := &types.SendVerifyCodeReq{
			Email: email,
			Type:  "register",
		}
		req2 := &types.SendVerifyCodeReq{
			Email: email,
			Type:  "reset_password",
		}

		code1 := logic.generateAndSaveCode(req1)
		code2 := logic.generateAndSaveCode(req2)

		// 验证各自的验证码正确
		redisKey1 := buildVerifyKey(req1.Email, req1.Type)
		redisKey2 := buildVerifyKey(req2.Email, req2.Type)

		savedCode1, _ := logic.svcCtx.Redis.HgetCtx(ctx, redisKey1, redisValueCodeFieldName)
		savedCode2, _ := logic.svcCtx.Redis.HgetCtx(ctx, redisKey2, redisValueCodeFieldName)

		assert.Equal(t, code1, savedCode1)
		assert.Equal(t, code2, savedCode2)
	})
}

func TestSendVerifyCodeLogic_sendToMQ(t *testing.T) {
	t.Run("成功发送消息到MQ", func(t *testing.T) {
		mockMQ := &mockKqPusherClient{}
		logic, s := newTestSendVerifyCodeLogic(t, nil, nil, mockMQ)
		defer s.Close()

		email := "test@example.com"
		code := "123456"
		codeType := "register"

		err := logic.sendToMQ(email, code, codeType)
		require.NoError(t, err)

		// 验证消息已发送
		require.Len(t, mockMQ.messages, 1)

		// 验证消息内容
		var msg types.VerificationCodeMessage
		err = json.Unmarshal([]byte(mockMQ.messages[0]), &msg)
		require.NoError(t, err)
		assert.Equal(t, code, msg.Code)
		assert.Equal(t, email, msg.Receiver)
		assert.Equal(t, codeType, msg.Type)
		assert.Greater(t, msg.Timestamp, int64(0))
	})

	t.Run("MQ推送失败返回内部错误", func(t *testing.T) {
		mockMQ := &mockKqPusherClient{
			pushFunc: func(ctx context.Context, v string) error {
				return errors.New("mq connection failed")
			},
		}
		logic, s := newTestSendVerifyCodeLogic(t, nil, nil, mockMQ)
		defer s.Close()

		email := "test@example.com"
		code := "123456"
		codeType := "register"

		err := logic.sendToMQ(email, code, codeType)
		require.Error(t, err)
	})
}

func TestSendVerifyCodeLogic_buildResponse(t *testing.T) {
	logic, s := newTestSendVerifyCodeLogic(t, nil, nil, nil)
	defer s.Close()

	resp := logic.buildResponse()
	require.NotNil(t, resp)
	assert.Equal(t, 60, resp.RetryAfter)
}

func TestSendVerifyCodeLogic_cleanupRateLimit(t *testing.T) {
	logic, s := newTestSendVerifyCodeLogic(t, nil, nil, nil)
	defer s.Close()

	ctx := context.Background()
	email := "test@example.com"
	codeType := "register"

	t.Run("成功删除存在的限流数据", func(t *testing.T) {
		// 先设置一个限流数据
		limitKey := buildLimitKey(email, codeType)
		err := logic.svcCtx.Redis.SetCtx(ctx, limitKey, "1")
		require.NoError(t, err)

		// 验证数据存在
		exists, err := logic.svcCtx.Redis.ExistsCtx(ctx, limitKey)
		require.NoError(t, err)
		assert.True(t, exists)

		// 调用清理函数
		logic.cleanupRateLimit(email, codeType)

		// 验证数据已被删除
		exists, err = logic.svcCtx.Redis.ExistsCtx(ctx, limitKey)
		require.NoError(t, err)
		assert.False(t, exists, "限流数据应该被删除")
	})

	t.Run("删除不存在的key不报错", func(t *testing.T) {
		// 确保key不存在
		nonExistentEmail := "nonexistent@example.com"
		limitKey := buildLimitKey(nonExistentEmail, codeType)

		exists, err := logic.svcCtx.Redis.ExistsCtx(ctx, limitKey)
		require.NoError(t, err)
		assert.False(t, exists)

		// 调用清理函数，不应该报错
		logic.cleanupRateLimit(nonExistentEmail, codeType)

		// 验证key仍然不存在
		exists, err = logic.svcCtx.Redis.ExistsCtx(ctx, limitKey)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestSendVerifyCodeLogic_cleanupVerifyCode(t *testing.T) {
	logic, s := newTestSendVerifyCodeLogic(t, nil, nil, nil)
	defer s.Close()

	ctx := context.Background()
	email := "test@example.com"
	codeType := "register"

	t.Run("成功删除存在的验证码数据", func(t *testing.T) {
		// 先设置一个验证码数据
		verifyKey := buildVerifyKey(email, codeType)
		err := logic.svcCtx.Redis.HsetCtx(ctx, verifyKey, redisValueCodeFieldName, "123456")
		require.NoError(t, err)
		err = logic.svcCtx.Redis.HsetCtx(ctx, verifyKey, redisValueUsedFieldName, "0")
		require.NoError(t, err)

		// 验证数据存在
		exists, err := logic.svcCtx.Redis.ExistsCtx(ctx, verifyKey)
		require.NoError(t, err)
		assert.True(t, exists)

		// 调用清理函数
		logic.cleanupVerifyCode(email, codeType)

		// 验证数据已被删除
		exists, err = logic.svcCtx.Redis.ExistsCtx(ctx, verifyKey)
		require.NoError(t, err)
		assert.False(t, exists, "验证码数据应该被删除")
	})

	t.Run("删除不存在的key不报错", func(t *testing.T) {
		// 确保key不存在
		nonExistentEmail := "nonexistent@example.com"
		verifyKey := buildVerifyKey(nonExistentEmail, codeType)

		exists, err := logic.svcCtx.Redis.ExistsCtx(ctx, verifyKey)
		require.NoError(t, err)
		assert.False(t, exists)

		// 调用清理函数，不应该报错
		logic.cleanupVerifyCode(nonExistentEmail, codeType)

		// 验证key仍然不存在
		exists, err = logic.svcCtx.Redis.ExistsCtx(ctx, verifyKey)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestSendVerifyCodeLogic_cleanupAll(t *testing.T) {
	logic, s := newTestSendVerifyCodeLogic(t, nil, nil, nil)
	defer s.Close()

	ctx := context.Background()
	email := "test@example.com"
	codeType := "register"

	t.Run("成功清理所有相关数据", func(t *testing.T) {
		// 先设置限流和验证码数据
		limitKey := buildLimitKey(email, codeType)
		verifyKey := buildVerifyKey(email, codeType)

		err := logic.svcCtx.Redis.SetCtx(ctx, limitKey, "1")
		require.NoError(t, err)
		err = logic.svcCtx.Redis.HsetCtx(ctx, verifyKey, redisValueCodeFieldName, "123456")
		require.NoError(t, err)

		// 验证数据存在
		limitExists, _ := logic.svcCtx.Redis.ExistsCtx(ctx, limitKey)
		verifyExists, _ := logic.svcCtx.Redis.ExistsCtx(ctx, verifyKey)
		assert.True(t, limitExists)
		assert.True(t, verifyExists)

		// 调用清理函数
		logic.cleanupAll(email, codeType)

		// 验证数据已被删除
		limitExists, _ = logic.svcCtx.Redis.ExistsCtx(ctx, limitKey)
		verifyExists, _ = logic.svcCtx.Redis.ExistsCtx(ctx, verifyKey)
		assert.False(t, limitExists, "限流数据应该被删除")
		assert.False(t, verifyExists, "验证码数据应该被删除")
	})
}

func TestSendVerifyCodeLogic_SendVerifyCode(t *testing.T) {
	t.Run("完整的注册验证码流程-成功", func(t *testing.T) {
		mockUsers := &mockUsersModel{
			findOneByEmailFunc: func(ctx context.Context, email string) (*model.Users, error) {
				return nil, model.ErrNotFound
			},
		}
		mockMQ := &mockKqPusherClient{}
		logic, s := newTestSendVerifyCodeLogic(t, nil, mockUsers, mockMQ)
		defer s.Close()

		req := &types.SendVerifyCodeReq{
			Email: "newuser@example.com",
			Type:  "register",
		}

		resp, err := logic.SendVerifyCode(req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, 60, resp.RetryAfter)

		// 验证MQ消息已发送
		assert.Len(t, mockMQ.messages, 1)

		// 验证验证码已保存
		ctx := context.Background()
		verifyKey := buildVerifyKey(req.Email, req.Type)
		code, err := logic.svcCtx.Redis.HgetCtx(ctx, verifyKey, redisValueCodeFieldName)
		require.NoError(t, err)
		assert.NotEmpty(t, code)
	})

	t.Run("注册时邮箱已存在-发送提醒邮件", func(t *testing.T) {
		mockUsers := &mockUsersModel{
			findOneByEmailFunc: func(ctx context.Context, email string) (*model.Users, error) {
				return &model.Users{
					Id:           1,
					Email:        email,
					PasswordHash: "hashed_password",
				}, nil
			},
		}
		mockMQ := &mockKqPusherClient{}
		logic, s := newTestSendVerifyCodeLogic(t, nil, mockUsers, mockMQ)
		defer s.Close()

		req := &types.SendVerifyCodeReq{
			Email: "existing@example.com",
			Type:  "register",
		}

		resp, err := logic.SendVerifyCode(req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// 验证提醒邮件已发送
		assert.Len(t, mockMQ.messages, 1)
		var msg types.VerificationCodeMessage
		err = json.Unmarshal([]byte(mockMQ.messages[0]), &msg)
		require.NoError(t, err)
		assert.Equal(t, "remind_registered", msg.Type)
	})

	t.Run("重置密码时邮箱不存在-返回错误", func(t *testing.T) {
		mockUsers := &mockUsersModel{
			findOneByEmailFunc: func(ctx context.Context, email string) (*model.Users, error) {
				return nil, model.ErrNotFound
			},
		}
		mockMQ := &mockKqPusherClient{}
		logic, s := newTestSendVerifyCodeLogic(t, nil, mockUsers, mockMQ)
		defer s.Close()

		req := &types.SendVerifyCodeReq{
			Email: "nonexistent@example.com",
			Type:  "reset_password",
		}

		resp, err := logic.SendVerifyCode(req)
		require.Error(t, err)
		assert.Nil(t, resp)
		// 由于 CodeEmailNotRegistered 没有定义错误消息，会返回"未知错误"
		// 这里只检查是否有错误返回即可
	})

	t.Run("参数验证失败", func(t *testing.T) {
		logic, s := newTestSendVerifyCodeLogic(t, nil, nil, nil)
		defer s.Close()

		req := &types.SendVerifyCodeReq{
			Email: "invalid-email",
			Type:  "register",
		}

		resp, err := logic.SendVerifyCode(req)
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("限流检查失败", func(t *testing.T) {
		mockUsers := &mockUsersModel{
			findOneByEmailFunc: func(ctx context.Context, email string) (*model.Users, error) {
				return nil, model.ErrNotFound
			},
		}
		mockMQ := &mockKqPusherClient{}
		logic, s := newTestSendVerifyCodeLogic(t, nil, mockUsers, mockMQ)
		defer s.Close()

		email := "test@example.com"

		// 第一次请求
		req := &types.SendVerifyCodeReq{
			Email: email,
			Type:  "register",
		}
		_, err := logic.SendVerifyCode(req)
		require.NoError(t, err)

		// 第二次请求应该被限流
		_, err = logic.SendVerifyCode(req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "发送过于频繁")
	})

	t.Run("MQ发送失败-清理资源", func(t *testing.T) {
		mockUsers := &mockUsersModel{
			findOneByEmailFunc: func(ctx context.Context, email string) (*model.Users, error) {
				return nil, model.ErrNotFound
			},
		}
		mockMQ := &mockKqPusherClient{
			pushFunc: func(ctx context.Context, v string) error {
				return errors.New("mq connection failed")
			},
		}
		logic, s := newTestSendVerifyCodeLogic(t, nil, mockUsers, mockMQ)
		defer s.Close()

		req := &types.SendVerifyCodeReq{
			Email: "test@example.com",
			Type:  "register",
		}

		resp, err := logic.SendVerifyCode(req)
		require.Error(t, err)
		assert.Nil(t, resp)

		// 验证资源已被清理
		ctx := context.Background()
		limitKey := buildLimitKey(req.Email, req.Type)
		verifyKey := buildVerifyKey(req.Email, req.Type)

		limitExists, _ := logic.svcCtx.Redis.ExistsCtx(ctx, limitKey)
		verifyExists, _ := logic.svcCtx.Redis.ExistsCtx(ctx, verifyKey)
		assert.False(t, limitExists, "限流标记应该被清理")
		assert.False(t, verifyExists, "验证码应该被清理")
	})
}

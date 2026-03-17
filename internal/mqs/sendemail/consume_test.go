package sendemail

import (
	"context"
	"encoding/json"
	"testing"
	"user/internal/config"
	"user/internal/svc"
	"user/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSendEmail(t *testing.T) {
	ctx := context.Background()
	svcCtx := &svc.ServiceContext{}

	s := NewSendEmail(ctx, svcCtx)

	require.NotNil(t, s)
	assert.Equal(t, ctx, s.ctx)
	assert.Equal(t, svcCtx, s.svcCtx)
}

func TestSendEmail_Consume(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		val     string
		svcCtx  *svc.ServiceContext
		wantErr bool
		errMsg  string
	}{
		{
			name: "有效的验证码消息-缺少SMTP配置",
			key:  "test-key",
			val: func() string {
				msg := types.VerificationCodeMessage{
					Code:      "123456",
					Receiver:  "test@example.com",
					Type:      "register",
					Timestamp: 1234567890,
				}
				data, _ := json.Marshal(msg)
				return string(data)
			}(),
			svcCtx: &svc.ServiceContext{
				Config: config.Config{
					Register: config.Register{
						SendCodeConfig: config.SendCodeConfig{
							ReceiveType: "register",
						},
					},
					SmtpConfig: config.SmtpConfig{
						Host: "",
						Port: 0,
						From: "",
					},
				},
			},
			wantErr: true,
			errMsg:  "设置发件人失败",
		},
		{
			name:    "无效的JSON格式",
			key:     "test-key",
			val:     "invalid json",
			svcCtx:  &svc.ServiceContext{},
			wantErr: true,
			errMsg:  "消息解析失败",
		},
		{
			name:    "空的JSON值",
			key:     "test-key",
			val:     "",
			svcCtx:  &svc.ServiceContext{},
			wantErr: true,
			errMsg:  "消息解析失败",
		},
		{
			name: "缺少必填字段的JSON-类型为register",
			key:  "test-key",
			val:  `{"code": "123456", "type": "register"}`,
			svcCtx: &svc.ServiceContext{
				Config: config.Config{
					Register: config.Register{
						SendCodeConfig: config.SendCodeConfig{
							ReceiveType: "register",
						},
					},
					SmtpConfig: config.SmtpConfig{
						Host: "",
						Port: 0,
						From: "",
					},
				},
			},
			// type 匹配 register，会尝试发送验证码邮件，但缺少 SMTP 配置会失败
			wantErr: true,
			errMsg:  "设置发件人失败",
		},
		{
			name: "复杂JSON结构-未知类型",
			key:  "test-key",
			val: func() string {
				msg := types.VerificationCodeMessage{
					Code:      "ABC123",
					Receiver:  "user+test@example.com",
					Type:      "reset_password",
					Timestamp: 9999999999,
				}
				data, _ := json.Marshal(msg)
				return string(data)
			}(),
			svcCtx: &svc.ServiceContext{
				Config: config.Config{
					Register: config.Register{
						SendCodeConfig: config.SendCodeConfig{
							ReceiveType: "register",
							ReminderType: config.ReminderType{
								Registered: "reminder_registered",
							},
						},
					},
				},
			},
			// 未知类型返回 nil，不阻塞队列
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			s := NewSendEmail(ctx, tt.svcCtx)

			err := s.Consume(ctx, tt.key, tt.val)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSendEmail_Consume_JSONParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected types.VerificationCodeMessage
	}{
		{
			name:  "标准格式",
			input: `{"code":"123456","receiver":"test@example.com","type":"register","timestamp":1234567890}`,
			expected: types.VerificationCodeMessage{
				Code:      "123456",
				Receiver:  "test@example.com",
				Type:      "register",
				Timestamp: 1234567890,
			},
		},
		{
			name:  "字段顺序不同",
			input: `{"timestamp":1234567890,"type":"login","receiver":"user@test.com","code":"ABC789"}`,
			expected: types.VerificationCodeMessage{
				Code:      "ABC789",
				Receiver:  "user@test.com",
				Type:      "login",
				Timestamp: 1234567890,
			},
		},
		{
			name:  "包含额外字段",
			input: `{"code":"123456","receiver":"test@example.com","type":"register","timestamp":1234567890,"extra":"ignored"}`,
			expected: types.VerificationCodeMessage{
				Code:      "123456",
				Receiver:  "test@example.com",
				Type:      "register",
				Timestamp: 1234567890,
			},
		},
		{
			name:  "Unicode字符",
			input: `{"code":"测试码","receiver":"用户@例子.中国","type":"注册","timestamp":1234567890}`,
			expected: types.VerificationCodeMessage{
				Code:      "测试码",
				Receiver:  "用户@例子.中国",
				Type:      "注册",
				Timestamp: 1234567890,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg types.VerificationCodeMessage
			err := json.Unmarshal([]byte(tt.input), &msg)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, msg)
		})
	}
}

func TestSendEmail_StructFields(t *testing.T) {
	ctx := context.Background()
	svcCtx := &svc.ServiceContext{}
	s := NewSendEmail(ctx, svcCtx)

	// 验证结构体字段可访问
	assert.NotNil(t, s.ctx)
	assert.NotNil(t, s.svcCtx)
}

func TestSendEmail_Consume_ReminderRegistered(t *testing.T) {
	ctx := context.Background()
	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			SmtpConfig: config.SmtpConfig{
				Host: "",
				Port: 0,
				From: "",
			},
			Register: config.Register{
				SendCodeConfig: config.SendCodeConfig{
					ReceiveType: "register",
					ReminderType: config.ReminderType{
						Registered: "reminder_registered",
					},
				},
			},
		},
	}
	s := NewSendEmail(ctx, svcCtx)

	msg := types.VerificationCodeMessage{
		Code:      "",
		Receiver:  "test@example.com",
		Type:      "reminder_registered",
		Timestamp: 1234567890,
	}
	data, _ := json.Marshal(msg)

	err := s.Consume(ctx, "test-key", string(data))

	// 由于缺少SMTP配置，会返回错误
	require.Error(t, err)
	assert.Contains(t, err.Error(), "设置发件人失败")
}

func TestSendEmail_Consume_UnknownType(t *testing.T) {
	ctx := context.Background()
	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			Register: config.Register{
				SendCodeConfig: config.SendCodeConfig{
					ReceiveType: "register",
					ReminderType: config.ReminderType{
						Registered: "reminder_registered",
					},
				},
			},
		},
	}
	s := NewSendEmail(ctx, svcCtx)

	msg := types.VerificationCodeMessage{
		Code:      "123456",
		Receiver:  "test@example.com",
		Type:      "unknown_type",
		Timestamp: 1234567890,
	}
	data, _ := json.Marshal(msg)

	err := s.Consume(ctx, "test-key", string(data))

	// 未知类型应该返回 nil，不阻塞队列
	require.NoError(t, err)
}

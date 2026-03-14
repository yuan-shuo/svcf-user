package sendemail

import (
	"context"
	"encoding/json"
	"testing"
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
			wantErr: true,
			errMsg:  "设置发件人失败",
		},
		{
			name:    "无效的JSON格式",
			key:     "test-key",
			val:     "invalid json",
			wantErr: true,
			errMsg:  "邮箱验证码发送失败",
		},
		{
			name:    "空的JSON值",
			key:     "test-key",
			val:     "",
			wantErr: true,
			errMsg:  "邮箱验证码发送失败",
		},
		{
			name: "缺少必填字段的JSON",
			key:  "test-key",
			val:  `{"code": "123456"}`,
			// 解析不会报错，但缺少的字段会是零值
			wantErr: true,
			errMsg:  "设置发件人失败",
		},
		{
			name: "复杂JSON结构-缺少SMTP配置",
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
			wantErr: true,
			errMsg:  "设置发件人失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			svcCtx := &svc.ServiceContext{}
			s := NewSendEmail(ctx, svcCtx)

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

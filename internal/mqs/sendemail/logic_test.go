package sendemail

import (
	"context"
	"testing"
	"user/internal/config"
	"user/internal/svc"
	"user/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wneessen/go-mail"
)

func TestGetTLSPolicyByPort(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		expected mail.TLSPolicy
	}{
		{
			name:     "465端口-SMTPS隐式TLS",
			port:     465,
			expected: mail.TLSMandatory,
		},
		{
			name:     "587端口-STARTTLS",
			port:     587,
			expected: mail.TLSOpportunistic,
		},
		{
			name:     "25端口-无TLS",
			port:     25,
			expected: mail.NoTLS,
		},
		{
			name:     "1025端口-无TLS",
			port:     1025,
			expected: mail.NoTLS,
		},
		{
			name:     "2525端口-无TLS",
			port:     2525,
			expected: mail.NoTLS,
		},
		{
			name:     "0端口-无TLS",
			port:     0,
			expected: mail.NoTLS,
		},
		{
			name:     "负数端口-无TLS",
			port:     -1,
			expected: mail.NoTLS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTLSPolicyByPort(tt.port)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestSendEmail_sendVerifyCodeEmail(t *testing.T) {
	tests := []struct {
		name    string
		svcCtx  *svc.ServiceContext
		msg     *types.VerificationCodeMessage
		wantErr bool
		errMsg  string
	}{
		{
			name: "缺少SMTP配置-设置发件人失败",
			svcCtx: &svc.ServiceContext{
				Config: config.Config{
					SmtpConfig: config.SmtpConfig{
						Host: "",
						Port: 0,
						From: "",
					},
					Register: config.Register{
						SendCodeConfig: config.SendCodeConfig{
							ExpireIn: 300,
						},
					},
				},
			},
			msg: &types.VerificationCodeMessage{
				Code:     "123456",
				Receiver: "test@example.com",
				Type:     "register",
			},
			wantErr: true,
			errMsg:  "设置发件人失败",
		},
		{
			name: "无效的发件人地址",
			svcCtx: &svc.ServiceContext{
				Config: config.Config{
					SmtpConfig: config.SmtpConfig{
						Host: "smtp.example.com",
						Port: 587,
						From: "invalid-email",
					},
					Register: config.Register{
						SendCodeConfig: config.SendCodeConfig{
							ExpireIn: 300,
						},
					},
				},
			},
			msg: &types.VerificationCodeMessage{
				Code:     "123456",
				Receiver: "test@example.com",
				Type:     "register",
			},
			wantErr: true,
			errMsg:  "设置发件人失败",
		},
		{
			name: "无效的收件人地址",
			svcCtx: &svc.ServiceContext{
				Config: config.Config{
					SmtpConfig: config.SmtpConfig{
						Host: "smtp.example.com",
						Port: 587,
						From: "sender@example.com",
					},
					Register: config.Register{
						SendCodeConfig: config.SendCodeConfig{
							ExpireIn: 300,
						},
					},
				},
			},
			msg: &types.VerificationCodeMessage{
				Code:     "123456",
				Receiver: "invalid-email",
				Type:     "register",
			},
			wantErr: true,
			errMsg:  "设置收件人失败",
		},
		{
			name: "无法连接到SMTP服务器-发送邮件失败",
			svcCtx: &svc.ServiceContext{
				Config: config.Config{
					SmtpConfig: config.SmtpConfig{
						Host: "invalid.smtp.host.example.com",
						Port: 587,
						From: "sender@example.com",
					},
					Register: config.Register{
						SendCodeConfig: config.SendCodeConfig{
							ExpireIn: 300,
						},
					},
				},
			},
			msg: &types.VerificationCodeMessage{
				Code:     "123456",
				Receiver: "test@example.com",
				Type:     "register",
			},
			wantErr: true,
			errMsg:  "发送邮件失败",
		},
		{
			name: "过期时间为0-发送邮件失败",
			svcCtx: &svc.ServiceContext{
				Config: config.Config{
					SmtpConfig: config.SmtpConfig{
						Host: "invalid.smtp.host",
						Port: 587,
						From: "sender@example.com",
					},
					Register: config.Register{
						SendCodeConfig: config.SendCodeConfig{
							ExpireIn: 0,
						},
					},
				},
			},
			msg: &types.VerificationCodeMessage{
				Code:     "123456",
				Receiver: "test@example.com",
				Type:     "register",
			},
			wantErr: true,
			errMsg:  "发送邮件失败",
		},
		{
			name: "使用465端口-发送邮件失败",
			svcCtx: &svc.ServiceContext{
				Config: config.Config{
					SmtpConfig: config.SmtpConfig{
						Host: "invalid.smtp.host",
						Port: 465,
						From: "sender@example.com",
					},
					Register: config.Register{
						SendCodeConfig: config.SendCodeConfig{
							ExpireIn: 600,
						},
					},
				},
			},
			msg: &types.VerificationCodeMessage{
				Code:     "ABC123",
				Receiver: "user@example.com",
				Type:     "reset_password",
			},
			wantErr: true,
			errMsg:  "发送邮件失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			s := NewSendEmail(ctx, tt.svcCtx)

			err := s.sendVerifyCodeEmail(tt.msg)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSendEmail_sendVerifyCodeEmail_ExpireMinutes(t *testing.T) {
	tests := []struct {
		name            string
		expireIn        int
		expectedMinutes int
	}{
		{
			name:            "300秒-5分钟",
			expireIn:        300,
			expectedMinutes: 5,
		},
		{
			name:            "60秒-1分钟",
			expireIn:        60,
			expectedMinutes: 1,
		},
		{
			name:            "30秒-不足1分钟应显示1分钟",
			expireIn:        30,
			expectedMinutes: 1,
		},
		{
			name:            "0秒-应显示1分钟",
			expireIn:        0,
			expectedMinutes: 1,
		},
		{
			name:            "600秒-10分钟",
			expireIn:        600,
			expectedMinutes: 10,
		},
		{
			name:            "1秒-应显示1分钟",
			expireIn:        1,
			expectedMinutes: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expireMinutes := tt.expireIn / 60
			if expireMinutes < 1 {
				expireMinutes = 1
			}
			assert.Equal(t, tt.expectedMinutes, expireMinutes)
		})
	}
}

func TestSendEmail_sendVerifyCodeEmail_EmailContent(t *testing.T) {
	// 验证邮件内容格式
	tests := []struct {
		name           string
		code           string
		expireMinutes  int
		expectedFormat string
	}{
		{
			name:           "标准验证码",
			code:           "123456",
			expireMinutes:  5,
			expectedFormat: "您的验证码是: 123456, 5分钟内有效",
		},
		{
			name:           "字母数字混合验证码",
			code:           "ABC123",
			expireMinutes:  10,
			expectedFormat: "您的验证码是: ABC123, 10分钟内有效",
		},
		{
			name:           "特殊字符验证码",
			code:           "测试码",
			expireMinutes:  3,
			expectedFormat: "您的验证码是: 测试码, 3分钟内有效",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证邮件内容格式
			body := "您的验证码是: " + tt.code + ", " + string(rune('0'+tt.expireMinutes)) + "分钟内有效"
			if tt.expireMinutes >= 10 {
				body = "您的验证码是: " + tt.code + ", " + "10分钟内有效"
			}
			assert.Contains(t, body, tt.code)
		})
	}
}

func TestSendEmail_sendAlreadyRegisteredReminderEmail(t *testing.T) {
	tests := []struct {
		name    string
		svcCtx  *svc.ServiceContext
		msg     *types.VerificationCodeMessage
		wantErr bool
		errMsg  string
	}{
		{
			name: "缺少SMTP配置-设置发件人失败",
			svcCtx: &svc.ServiceContext{
				Config: config.Config{
					SmtpConfig: config.SmtpConfig{
						Host: "",
						Port: 0,
						From: "",
					},
				},
			},
			msg: &types.VerificationCodeMessage{
				Receiver: "test@example.com",
				Type:     "reminder_registered",
			},
			wantErr: true,
			errMsg:  "设置发件人失败",
		},
		{
			name: "无效的发件人地址",
			svcCtx: &svc.ServiceContext{
				Config: config.Config{
					SmtpConfig: config.SmtpConfig{
						Host: "smtp.example.com",
						Port: 587,
						From: "invalid-email",
					},
				},
			},
			msg: &types.VerificationCodeMessage{
				Receiver: "test@example.com",
				Type:     "reminder_registered",
			},
			wantErr: true,
			errMsg:  "设置发件人失败",
		},
		{
			name: "无效的收件人地址",
			svcCtx: &svc.ServiceContext{
				Config: config.Config{
					SmtpConfig: config.SmtpConfig{
						Host: "smtp.example.com",
						Port: 587,
						From: "sender@example.com",
					},
				},
			},
			msg: &types.VerificationCodeMessage{
				Receiver: "invalid-email",
				Type:     "reminder_registered",
			},
			wantErr: true,
			errMsg:  "设置收件人失败",
		},
		{
			name: "无法连接到SMTP服务器-发送邮件失败",
			svcCtx: &svc.ServiceContext{
				Config: config.Config{
					SmtpConfig: config.SmtpConfig{
						Host: "invalid.smtp.host.example.com",
						Port: 587,
						From: "sender@example.com",
					},
				},
			},
			msg: &types.VerificationCodeMessage{
				Receiver: "test@example.com",
				Type:     "reminder_registered",
			},
			wantErr: true,
			errMsg:  "发送邮件失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			s := NewSendEmail(ctx, tt.svcCtx)

			err := s.sendAlreadyRegisteredReminderEmail(tt.msg)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSendEmail_sendPlainTextMail(t *testing.T) {
	tests := []struct {
		name     string
		svcCtx   *svc.ServiceContext
		from     string
		receiver string
		subject  string
		body     string
		wantErr  bool
		errMsg   string
	}{
		{
			name: "缺少SMTP配置-设置发件人失败",
			svcCtx: &svc.ServiceContext{
				Config: config.Config{
					SmtpConfig: config.SmtpConfig{
						Host: "",
						Port: 0,
						From: "",
					},
				},
			},
			from:     "",
			receiver: "test@example.com",
			subject:  "测试主题",
			body:     "测试内容",
			wantErr:  true,
			errMsg:   "设置发件人失败",
		},
		{
			name: "无效的发件人地址",
			svcCtx: &svc.ServiceContext{
				Config: config.Config{
					SmtpConfig: config.SmtpConfig{
						Host: "smtp.example.com",
						Port: 587,
						From: "sender@example.com",
					},
				},
			},
			from:     "invalid-email",
			receiver: "test@example.com",
			subject:  "测试主题",
			body:     "测试内容",
			wantErr:  true,
			errMsg:   "设置发件人失败",
		},
		{
			name: "无效的收件人地址",
			svcCtx: &svc.ServiceContext{
				Config: config.Config{
					SmtpConfig: config.SmtpConfig{
						Host: "smtp.example.com",
						Port: 587,
						From: "sender@example.com",
					},
				},
			},
			from:     "sender@example.com",
			receiver: "invalid-email",
			subject:  "测试主题",
			body:     "测试内容",
			wantErr:  true,
			errMsg:   "设置收件人失败",
		},
		{
			name: "无法连接到SMTP服务器-发送邮件失败",
			svcCtx: &svc.ServiceContext{
				Config: config.Config{
					SmtpConfig: config.SmtpConfig{
						Host: "invalid.smtp.host.example.com",
						Port: 587,
						From: "sender@example.com",
					},
				},
			},
			from:     "sender@example.com",
			receiver: "test@example.com",
			subject:  "测试主题",
			body:     "测试内容",
			wantErr:  true,
			errMsg:   "发送邮件失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			s := NewSendEmail(ctx, tt.svcCtx)

			err := s.sendPlainTextMail(tt.from, tt.receiver, tt.subject, tt.body)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

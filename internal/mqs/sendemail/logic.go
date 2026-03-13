package sendemail

import (
	"fmt"
	"user/internal/types"

	"github.com/wneessen/go-mail"
)

func (l *SendEmail) sendVerifyCodeEmail(msg *types.VerificationCodeMessage) error {
	// 创建邮件消息
	m := mail.NewMsg()

	// 设置发件人
	if err := m.From(l.svcCtx.Config.SmtpConfig.From); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}

	// 设置收件人
	if err := m.To(msg.Receiver); err != nil {
		return fmt.Errorf("设置收件人失败: %w", err)
	}

	// 设置主题
	m.Subject("验证码")

	// 计算分钟，至少显示1分钟
	expireMinutes := l.svcCtx.Config.Register.SendCodeConfig.ExpireIn / 60
	if expireMinutes < 1 {
		expireMinutes = 1
	}
	// 设置邮件内容
	body := fmt.Sprintf("您的验证码是: %s, %d分钟内有效", msg.Code, expireMinutes)
	m.SetBodyString(mail.TypeTextPlain, body)

	// 创建客户端连接
	client, err := mail.NewClient(
		l.svcCtx.Config.SmtpConfig.Host,
		mail.WithPort(l.svcCtx.Config.SmtpConfig.Port),
		mail.WithTLSPolicy(getTLSPolicyByPort(l.svcCtx.Config.SmtpConfig.Port)),
	)
	if err != nil {
		return fmt.Errorf("创建邮件客户端失败: %w", err)
	}

	// 发送邮件
	if err := client.DialAndSend(m); err != nil {
		return fmt.Errorf("发送邮件失败: %w", err)
	}

	return nil
}

// getTLSPolicyByPort 根据端口返回对应的 TLS 策略
func getTLSPolicyByPort(port int) mail.TLSPolicy {
	switch port {
	case 465:
		// 465端口：SMTPS，隐式TLS
		return mail.TLSMandatory
	case 587:
		// 587端口：SMTP with STARTTLS，先明文后加密
		return mail.TLSOpportunistic // 或者用 WithSTARTTLS() 方式
	default:
		// 其他端口（25, 1025等）：无TLS或尝试STARTTLS
		// 25端口通常用STARTTLS，但为了安全，默认用NoTLS让上层决定
		return mail.NoTLS
	}
}

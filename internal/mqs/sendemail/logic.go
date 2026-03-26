package sendemail

import (
	"fmt"
	"user/internal/types"

	"github.com/wneessen/go-mail"
)

// 发送已注册提醒邮件
func (l *SendEmail) sendAlreadyRegisteredReminderEmail(msg *types.VerificationCodeMessage) error {
	return l.sendPlainTextMail(
		l.svcCtx.Config.SmtpConfig.From,
		msg.Receiver,
		"您尝试注册的邮箱已被使用",
		"收到此邮件的邮箱已经被注册过，您可以直接使用该邮箱登录",
	)
}

// 发送注册用验证码邮件
func (l *SendEmail) sendVerifyCodeEmail(msg *types.VerificationCodeMessage) error {
	// 计算分钟，至少显示1分钟
	expireMinutes := l.svcCtx.Config.VerifyCodeConfig.Time.ExpireIn / 60
	if expireMinutes < 1 {
		expireMinutes = 1
	}

	return l.sendPlainTextMail(
		l.svcCtx.Config.SmtpConfig.From,
		msg.Receiver,
		"您的注册验证码",
		fmt.Sprintf("您的验证码是: %s, %d分钟内有效", msg.Code, expireMinutes),
	)
}

// 统一的邮件发送方法
func (l *SendEmail) sendPlainTextMail(from, receiver, subject, body string) error {
	// 创建邮件消息
	m := mail.NewMsg()

	// 设置发件人
	if err := m.From(from); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}

	// 设置收件人
	if err := m.To(receiver); err != nil {
		return fmt.Errorf("设置收件人失败: %w", err)
	}

	// 设置主题
	m.Subject(subject)

	// 设置邮件内容
	m.SetBodyString(mail.TypeTextPlain, body)

	// 创建客户端连接
	client, err := mail.NewClient(
		l.svcCtx.Config.SmtpConfig.Host,
		mail.WithPort(l.svcCtx.Config.SmtpConfig.Port),
		mail.WithTLSPolicy(getTLSPolicyByPort(l.svcCtx.Config.SmtpConfig.Port)),
		mail.WithUsername(l.svcCtx.Config.SmtpConfig.Username),
		mail.WithPassword(l.svcCtx.Config.SmtpConfig.Password),
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

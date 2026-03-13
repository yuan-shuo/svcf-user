// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	Register Register

	KqConsumerConf kq.KqConf // 消息队列消费者配置
	KqPusherConf   KqPusherConf
	SmtpConfig     SmtpConfig

	rest.RestConf
}

// 消息队列生产者配置
type KqPusherConf struct {
	Brokers []string
	Topic   string
}

// 注册配置
type Register struct {
	SendCodeConfig SendCodeConfig
}

// 验证码发送配置
type SendCodeConfig struct {
	ReciveType string // 接收验证码类型
	ExpireIn   int
	RetryAfter int
}

// 邮件发送配置
type SmtpConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

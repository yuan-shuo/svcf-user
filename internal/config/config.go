// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	Register Register
	rest.RestConf
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

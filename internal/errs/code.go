// Package errs 提供统一的错误处理
// 逻辑层返回给 handler 的错误使用 errs.CodeError，包含错误码和通用错误信息
// 详细的错误信息使用 logx 在逻辑层内部记录
package errs

// 错误码定义

const (
	// 成功
	CodeSuccess = 0

	// 通用错误 (1000-1999)

	CodeInternalError   = 1000 // 内部错误
	CodeInvalidParam    = 1001 // 参数错误
	CodeUnauthorized    = 1002 // 未授权
	CodeForbidden       = 1003 // 禁止访问
	CodeNotFound        = 1004 // 资源不存在
	CodeTooManyRequests = 1009 // 请求过于频繁

	// 用户模块错误 (2000-2999)

	CodeUserNotFound       = 2000 // 用户不存在
	CodeUserAlreadyExists  = 2001 // 用户已存在
	CodeInvalidPassword    = 2002 // 密码错误
	CodeInvalidCode        = 2003 // 验证码错误
	CodeCodeNotFound       = 2004 // 验证码不存在
	CodeCodeAlreadyUsed    = 2005 // 验证码已使用
	CodeEmailRegistered    = 2006 // 邮箱已注册
	CodeEmailNotVerified   = 2007 // 邮箱未验证
	CodeEmailNotRegistered = 2008 // 邮箱未注册

	CodeUserNotExistOrPasswordIncorrect = 2100 // 登录时用户名不存在或密码错误
	CodePasswordSameAsOld               = 2101 // 新密码与旧密码相同
	CodeOldPasswordIncorrect            = 2102 // 旧密码错误
	CodeInvalidToken                    = 2103 // 无效的token

)

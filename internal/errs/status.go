package errs

import "net/http"

// errorHTTPStatus 错误码与 HTTP 状态码的映射表
// 通过映射表关联错误码和对应的 HTTP 响应状态码
var errorHTTPStatus = map[int]int{
	// 成功
	CodeSuccess: http.StatusOK,

	// 通用错误 (1000-1999)
	CodeInternalError: http.StatusInternalServerError, // 500 内部错误
	CodeInvalidParam:  http.StatusBadRequest,          // 400 参数错误
	CodeUnauthorized:  http.StatusUnauthorized,        // 401 未授权
	CodeForbidden:     http.StatusForbidden,           // 403 禁止访问
	CodeNotFound:      http.StatusNotFound,            // 404 资源不存在

	// 用户模块错误 (2000-2999)
	CodeUserNotFound:       http.StatusNotFound,     // 404 用户不存在
	CodeUserAlreadyExists:  http.StatusConflict,     // 409 用户已存在
	CodeInvalidPassword:    http.StatusUnauthorized, // 401 密码错误
	CodeInvalidCode:        http.StatusBadRequest,   // 400 验证码错误
	CodeCodeNotFound:       http.StatusBadRequest,   // 400 验证码不存在
	CodeCodeAlreadyUsed:    http.StatusBadRequest,   // 400 验证码已使用
	CodeEmailRegistered:    http.StatusConflict,     // 409 邮箱已注册
	CodeEmailNotVerified:   http.StatusForbidden,    // 403 邮箱未验证
	CodeEmailNotRegistered: http.StatusBadRequest,   // 400 邮箱未注册

	CodeUserNotExistOrPasswordIncorrect: http.StatusUnauthorized, // 401 登录时用户名不存在或密码错误
	CodePasswordSameAsOld:               http.StatusBadRequest,   // 400 新密码与旧密码相同
	CodeOldPasswordIncorrect:            http.StatusUnauthorized, // 401 旧密码错误
	CodeInvalidToken:                    http.StatusUnauthorized, // 401 无效的token
}

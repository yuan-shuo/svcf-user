package errs

// codeMsg 错误码对应的通用错误信息（返回给客户端）
var codeMsg = map[int]string{
	CodeSuccess:                         "success",
	CodeInternalError:                   "系统繁忙，请稍后重试",
	CodeInvalidParam:                    "请求参数错误",
	CodeUnauthorized:                    "请先登录",
	CodeForbidden:                       "没有权限执行此操作",
	CodeNotFound:                        "请求的资源不存在",
	CodeUserNotFound:                    "用户不存在",
	CodeUserAlreadyExists:               "用户已存在",
	CodeInvalidPassword:                 "密码错误",
	CodeInvalidCode:                     "验证码错误",
	CodeCodeNotFound:                    "验证码不存在或已过期，请重新获取",
	CodeCodeAlreadyUsed:                 "验证码已使用，请重新获取",
	CodeEmailRegistered:                 "该邮箱已注册",
	CodeEmailNotVerified:                "邮箱未验证",
	CodeEmailNotRegistered:              "该邮箱未注册",
	CodeUserNotExistOrPasswordIncorrect: "登录时用户名不存在或密码错误",
	CodePasswordSameAsOld:               "新密码与旧密码相同",
}

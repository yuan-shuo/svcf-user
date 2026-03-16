package errs

import "fmt"

// CodeError 带错误码的错误类型
type CodeError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *CodeError) Error() string {
	return fmt.Sprintf("code: %d, msg: %s", e.Code, e.Msg)
}

// GetMsg 获取错误码对应的错误信息
func GetMsg(code int) string {
	if msg, ok := codeMsg[code]; ok {
		return msg
	}
	return "未知错误"
}

// New 创建一个新的 CodeError
// 如果不传 msg，则使用默认的错误信息
func New(code int, msg ...string) *CodeError {
	e := &CodeError{Code: code}
	if len(msg) > 0 && msg[0] != "" {
		e.Msg = msg[0]
	} else {
		e.Msg = GetMsg(code)
	}
	return e
}

// IsCodeError 检查错误是否为 CodeError
func IsCodeError(err error) (*CodeError, bool) {
	if err == nil {
		return nil, false
	}
	if e, ok := err.(*CodeError); ok {
		return e, true
	}
	return nil, false
}

// Wrap 包装错误，将内部错误转换为 CodeError
// 如果已经是 CodeError，则直接返回
// 否则返回内部错误，并使用 logx 记录详细错误
func Wrap(err error, defaultCode int) *CodeError {
	if err == nil {
		return nil
	}
	if e, ok := IsCodeError(err); ok {
		return e
	}
	return New(defaultCode)
}

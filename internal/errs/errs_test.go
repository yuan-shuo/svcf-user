package errs

import (
	"errors"
	"testing"
)

func TestCodeError_Error(t *testing.T) {
	tests := []struct {
		name     string
		codeErr  *CodeError
		expected string
	}{
		{
			name:     "普通错误",
			codeErr:  &CodeError{Code: CodeInvalidParam, Msg: "参数错误"},
			expected: "code: 1001, msg: 参数错误",
		},
		{
			name:     "成功状态",
			codeErr:  &CodeError{Code: CodeSuccess, Msg: "success"},
			expected: "code: 0, msg: success",
		},
		{
			name:     "空消息",
			codeErr:  &CodeError{Code: CodeInternalError, Msg: ""},
			expected: "code: 1000, msg: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.codeErr.Error()
			if got != tt.expected {
				t.Errorf("Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetMsg(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected string
	}{
		{
			name:     "存在的错误码",
			code:     CodeInvalidParam,
			expected: "请求参数错误",
		},
		{
			name:     "不存在的错误码",
			code:     99999,
			expected: "未知错误",
		},
		{
			name:     "成功状态码",
			code:     CodeSuccess,
			expected: "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMsg(tt.code)
			if got != tt.expected {
				t.Errorf("GetMsg(%d) = %v, want %v", tt.code, got, tt.expected)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name         string
		code         int
		msg          []string
		expectedCode int
		expectedMsg  string
	}{
		{
			name:         "使用默认消息",
			code:         CodeInvalidParam,
			msg:          nil,
			expectedCode: CodeInvalidParam,
			expectedMsg:  "请求参数错误",
		},
		{
			name:         "自定义消息",
			code:         CodeInvalidParam,
			msg:          []string{"自定义参数错误"},
			expectedCode: CodeInvalidParam,
			expectedMsg:  "自定义参数错误",
		},
		{
			name:         "空字符串使用默认",
			code:         CodeInvalidParam,
			msg:          []string{""},
			expectedCode: CodeInvalidParam,
			expectedMsg:  "请求参数错误",
		},
		{
			name:         "多个消息参数取第一个",
			code:         CodeInvalidParam,
			msg:          []string{"第一个", "第二个"},
			expectedCode: CodeInvalidParam,
			expectedMsg:  "第一个",
		},
		{
			name:         "不存在的错误码无默认消息",
			code:         99999,
			msg:          nil,
			expectedCode: 99999,
			expectedMsg:  "未知错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.code, tt.msg...)
			if got.Code != tt.expectedCode {
				t.Errorf("New() Code = %v, want %v", got.Code, tt.expectedCode)
			}
			if got.Msg != tt.expectedMsg {
				t.Errorf("New() Msg = %v, want %v", got.Msg, tt.expectedMsg)
			}
		})
	}
}

func TestIsCodeError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantOk         bool
		wantCode       int
		wantMsg        string
	}{
		{
			name:     "nil错误",
			err:      nil,
			wantOk:   false,
			wantCode: 0,
			wantMsg:  "",
		},
		{
			name:     "CodeError类型",
			err:      New(CodeInvalidParam),
			wantOk:   true,
			wantCode: CodeInvalidParam,
			wantMsg:  "请求参数错误",
		},
		{
			name:     "普通error类型",
			err:      errors.New("普通错误"),
			wantOk:   false,
			wantCode: 0,
			wantMsg:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := IsCodeError(tt.err)
			if ok != tt.wantOk {
				t.Errorf("IsCodeError() ok = %v, want %v", ok, tt.wantOk)
				return
			}
			if !ok {
				return
			}
			if got.Code != tt.wantCode {
				t.Errorf("IsCodeError() Code = %v, want %v", got.Code, tt.wantCode)
			}
			if got.Msg != tt.wantMsg {
				t.Errorf("IsCodeError() Msg = %v, want %v", got.Msg, tt.wantMsg)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		defaultCode  int
		wantNil      bool
		wantCode     int
		wantMsg      string
	}{
		{
			name:        "nil错误",
			err:         nil,
			defaultCode: CodeInternalError,
			wantNil:     true,
		},
		{
			name:        "已经是CodeError",
			err:         New(CodeInvalidParam),
			defaultCode: CodeInternalError,
			wantNil:     false,
			wantCode:    CodeInvalidParam,
			wantMsg:     "请求参数错误",
		},
		{
			name:        "普通error转换为默认错误码",
			err:         errors.New("数据库连接失败"),
			defaultCode: CodeInternalError,
			wantNil:     false,
			wantCode:    CodeInternalError,
			wantMsg:     "系统繁忙，请稍后重试",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Wrap(tt.err, tt.defaultCode)
			if tt.wantNil {
				if got != nil {
					t.Errorf("Wrap() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Errorf("Wrap() = nil, want non-nil")
				return
			}
			if got.Code != tt.wantCode {
				t.Errorf("Wrap() Code = %v, want %v", got.Code, tt.wantCode)
			}
			if got.Msg != tt.wantMsg {
				t.Errorf("Wrap() Msg = %v, want %v", got.Msg, tt.wantMsg)
			}
		})
	}
}

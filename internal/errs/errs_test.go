package errs

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodeError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *CodeError
		want string
	}{
		{
			name: "有错误信息",
			err:  &CodeError{Code: 1000, Msg: "系统错误"},
			want: "系统错误",
		},
		{
			name: "空错误信息",
			err:  &CodeError{Code: 1000, Msg: ""},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetMsg(t *testing.T) {
	tests := []struct {
		name string
		code int
		want string
	}{
		{
			name: "存在的错误码",
			code: CodeSuccess,
			want: "success",
		},
		{
			name: "存在的错误码-内部错误",
			code: CodeInternalError,
			want: "系统繁忙，请稍后重试",
		},
		{
			name: "不存在的错误码",
			code: 99999,
			want: "未知错误",
		},
		{
			name: "负数错误码",
			code: -1,
			want: "未知错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMsg(tt.code)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCodeError_Data(t *testing.T) {
	tests := []struct {
		name string
		err  *CodeError
		want *CodeErrorResponse
	}{
		{
			name: "正常数据",
			err:  &CodeError{Code: 1000, Msg: "系统错误"},
			want: &CodeErrorResponse{Code: 1000, Msg: "系统错误"},
		},
		{
			name: "零值",
			err:  &CodeError{Code: 0, Msg: ""},
			want: &CodeErrorResponse{Code: 0, Msg: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Data()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestErrsHandler(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantData   *CodeErrorResponse
		isNilData  bool // 标记是否期望返回 nil
	}{
		{
			name:       "CodeError-内部错误",
			err:        New(CodeInternalError),
			wantStatus: http.StatusInternalServerError,
			wantData:   &CodeErrorResponse{Code: CodeInternalError, Msg: GetMsg(CodeInternalError)},
			isNilData:  false,
		},
		{
			name:       "CodeError-参数错误",
			err:        New(CodeInvalidParam),
			wantStatus: http.StatusBadRequest,
			wantData:   &CodeErrorResponse{Code: CodeInvalidParam, Msg: GetMsg(CodeInvalidParam)},
			isNilData:  false,
		},
		{
			name:       "CodeError-未授权",
			err:        New(CodeUnauthorized),
			wantStatus: http.StatusUnauthorized,
			wantData:   &CodeErrorResponse{Code: CodeUnauthorized, Msg: GetMsg(CodeUnauthorized)},
			isNilData:  false,
		},
		{
			name:       "CodeError-资源不存在",
			err:        New(CodeNotFound),
			wantStatus: http.StatusNotFound,
			wantData:   &CodeErrorResponse{Code: CodeNotFound, Msg: GetMsg(CodeNotFound)},
			isNilData:  false,
		},
		{
			name:       "CodeError-用户已存在",
			err:        New(CodeUserAlreadyExists),
			wantStatus: http.StatusConflict,
			wantData:   &CodeErrorResponse{Code: CodeUserAlreadyExists, Msg: GetMsg(CodeUserAlreadyExists)},
			isNilData:  false,
		},
		{
			name:       "普通错误",
			err:        errors.New("普通错误"),
			wantStatus: http.StatusInternalServerError,
			wantData:   nil,
			isNilData:  true,
		},
		{
			name:       "nil 错误",
			err:        nil,
			wantStatus: http.StatusInternalServerError,
			wantData:   nil,
			isNilData:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotData := ErrsHandler(tt.err)
			assert.Equal(t, tt.wantStatus, gotStatus)
			if tt.isNilData {
				assert.Nil(t, gotData)
			} else {
				assert.Equal(t, tt.wantData, gotData)
			}
		})
	}
}

func TestCodeError_judgeErrsStatus(t *testing.T) {
	tests := []struct {
		name string
		code int
		want int
	}{
		{
			name: "内部错误-500",
			code: CodeInternalError,
			want: http.StatusInternalServerError,
		},
		{
			name: "参数错误-400",
			code: CodeInvalidParam,
			want: http.StatusBadRequest,
		},
		{
			name: "未授权-401",
			code: CodeUnauthorized,
			want: http.StatusUnauthorized,
		},
		{
			name: "禁止访问-403",
			code: CodeForbidden,
			want: http.StatusForbidden,
		},
		{
			name: "资源不存在-404",
			code: CodeNotFound,
			want: http.StatusNotFound,
		},
		{
			name: "用户已存在-409",
			code: CodeUserAlreadyExists,
			want: http.StatusConflict,
		},
		{
			name: "不存在的错误码-返回0",
			code: 99999,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &CodeError{Code: tt.code}
			got := e.JudgeErrsStatus()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		msg      []string
		wantCode int
		wantMsg  string
	}{
		{
			name:     "使用默认消息",
			code:     CodeSuccess,
			msg:      nil,
			wantCode: CodeSuccess,
			wantMsg:  GetMsg(CodeSuccess),
		},
		{
			name:     "自定义消息",
			code:     CodeInternalError,
			msg:      []string{"自定义错误信息"},
			wantCode: CodeInternalError,
			wantMsg:  "自定义错误信息",
		},
		{
			name:     "空字符串使用默认消息",
			code:     CodeInvalidParam,
			msg:      []string{""},
			wantCode: CodeInvalidParam,
			wantMsg:  GetMsg(CodeInvalidParam),
		},
		{
			name:     "多个消息参数取第一个",
			code:     CodeUnauthorized,
			msg:      []string{"第一个", "第二个"},
			wantCode: CodeUnauthorized,
			wantMsg:  "第一个",
		},
		{
			name:     "未定义错误码使用未知错误",
			code:     99999,
			msg:      nil,
			wantCode: 99999,
			wantMsg:  "未知错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.code, tt.msg...)
			assert.Equal(t, tt.wantCode, got.Code)
			assert.Equal(t, tt.wantMsg, got.Msg)
		})
	}
}

func TestIsCodeError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantIsCode bool
		wantCode   int
	}{
		{
			name:       "是 CodeError",
			err:        New(CodeInternalError),
			wantIsCode: true,
			wantCode:   CodeInternalError,
		},
		{
			name:       "是普通错误",
			err:        errors.New("普通错误"),
			wantIsCode: false,
			wantCode:   0,
		},
		{
			name:       "nil 错误",
			err:        nil,
			wantIsCode: false,
			wantCode:   0,
		},
		{
			name:       "包装后的 CodeError",
			err:        fmt.Errorf("包装: %w", New(CodeInvalidParam)),
			wantIsCode: true,
			wantCode:   CodeInvalidParam,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codeErr, ok := IsCodeError(tt.err)
			assert.Equal(t, tt.wantIsCode, ok)
			if tt.wantIsCode {
				require.NotNil(t, codeErr)
				assert.Equal(t, tt.wantCode, codeErr.Code)
			} else {
				assert.Nil(t, codeErr)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		defaultCode int
		wantNil     bool
		wantCode    int
	}{
		{
			name:        "nil 错误",
			err:         nil,
			defaultCode: CodeInternalError,
			wantNil:     true,
			wantCode:    0,
		},
		{
			name:        "已经是 CodeError",
			err:         New(CodeInvalidParam),
			defaultCode: CodeInternalError,
			wantNil:     false,
			wantCode:    CodeInvalidParam,
		},
		{
			name:        "普通错误-使用默认错误码",
			err:         errors.New("数据库连接失败"),
			defaultCode: CodeInternalError,
			wantNil:     false,
			wantCode:    CodeInternalError,
		},
		{
			name:        "普通错误-使用自定义错误码",
			err:         errors.New("参数无效"),
			defaultCode: CodeInvalidParam,
			wantNil:     false,
			wantCode:    CodeInvalidParam,
		},
		{
			name:        "包装后的 CodeError",
			err:         fmt.Errorf("包装: %w", New(CodeUnauthorized)),
			defaultCode: CodeInternalError,
			wantNil:     false,
			wantCode:    CodeUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Wrap(tt.err, tt.defaultCode)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, tt.wantCode, got.Code)
			}
		})
	}
}

// TestErrorCodeRanges 测试错误码范围是否符合规范
func TestErrorCodeRanges(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		minRange int
		maxRange int
		codeName string
	}{
		{
			name:     "成功",
			code:     CodeSuccess,
			minRange: 0,
			maxRange: 0,
			codeName: "CodeSuccess",
		},
		{
			name:     "通用错误-内部错误",
			code:     CodeInternalError,
			minRange: 1000,
			maxRange: 1999,
			codeName: "CodeInternalError",
		},
		{
			name:     "通用错误-参数错误",
			code:     CodeInvalidParam,
			minRange: 1000,
			maxRange: 1999,
			codeName: "CodeInvalidParam",
		},
		{
			name:     "用户模块错误-用户不存在",
			code:     CodeUserNotFound,
			minRange: 2000,
			maxRange: 2999,
			codeName: "CodeUserNotFound",
		},
		{
			name:     "用户模块错误-邮箱未注册",
			code:     CodeEmailNotRegistered,
			minRange: 2000,
			maxRange: 2999,
			codeName: "CodeEmailNotRegistered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.GreaterOrEqual(t, tt.code, tt.minRange, "%s 应该在 [%d, %d] 范围内", tt.codeName, tt.minRange, tt.maxRange)
			assert.LessOrEqual(t, tt.code, tt.maxRange, "%s 应该在 [%d, %d] 范围内", tt.codeName, tt.minRange, tt.maxRange)
		})
	}
}

// TestHTTPStatusValidity 测试所有 HTTP 状态码是否有效
func TestHTTPStatusValidity(t *testing.T) {
	validStatuses := []int{
		http.StatusOK,                  // 200
		http.StatusBadRequest,          // 400
		http.StatusUnauthorized,        // 401
		http.StatusForbidden,           // 403
		http.StatusNotFound,            // 404
		http.StatusConflict,            // 409
		http.StatusTooManyRequests,     // 429
		http.StatusInternalServerError, // 500
	}

	// 检查所有定义的 HTTP 状态码都是有效的标准状态码
	for code, status := range errorHTTPStatus {
		found := false
		for _, valid := range validStatuses {
			if status == valid {
				found = true
				break
			}
		}
		assert.True(t, found, "错误码 %d 的 HTTP 状态码 %d 不是标准的常用状态码", code, status)
	}
}

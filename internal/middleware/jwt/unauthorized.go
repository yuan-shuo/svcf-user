package jwt

import (
	"net/http"

	"user/internal/errs"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// UnauthorizedCallback 返回 JWT 鉴权失败时的自定义回调函数
// 使用 errs 包的结构化错误响应格式
func UnauthorizedCallback() func(w http.ResponseWriter, r *http.Request, err error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		// 使用 errs 包的结构化错误响应
		codeErr := errs.New(errs.CodeUnauthorized)
		w.WriteHeader(codeErr.JudgeErrsStatus())
		httpx.WriteJson(w, codeErr.JudgeErrsStatus(), codeErr.Data())
	}
}

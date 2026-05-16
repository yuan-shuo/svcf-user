// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package jwks

import (
	"net/http"

	"user/internal/logic/jwks"
	"user/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func JwksHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := jwks.NewJwksLogic(r.Context(), svcCtx)
		resp, err := l.Jwks()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

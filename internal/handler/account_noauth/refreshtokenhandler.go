// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package account_noauth

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"user/internal/logic/account_noauth"
	"user/internal/svc"
	"user/internal/types"
)

func RefreshTokenHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RefreshTokenReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := account_noauth.NewRefreshTokenLogic(r.Context(), svcCtx)
		resp, err := l.RefreshToken(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

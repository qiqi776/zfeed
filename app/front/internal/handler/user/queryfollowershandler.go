package user

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zfeed/app/front/internal/logic/user"
	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
)

func QueryFollowersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.QueryFollowersReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := user.NewQueryFollowersLogic(r.Context(), svcCtx)
		resp, err := l.QueryFollowers(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

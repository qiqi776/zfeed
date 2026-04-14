package search

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zfeed/app/front/internal/logic/search"
	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
)

func SearchContentsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SearchContentsReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := search.NewSearchContentsLogic(r.Context(), svcCtx)
		resp, err := l.SearchContents(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

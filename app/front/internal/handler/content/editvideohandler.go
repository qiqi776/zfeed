package content

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zfeed/app/front/internal/logic/content"
	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
)

func EditVideoHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.EditVideoReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := content.NewEditVideoLogic(r.Context(), svcCtx)
		resp, err := l.EditVideo(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

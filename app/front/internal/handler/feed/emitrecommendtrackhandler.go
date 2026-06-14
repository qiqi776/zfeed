// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package feed

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"zfeed/app/front/internal/logic/feed"
	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
)

func EmitRecommendTrackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RecommendTrackReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := feed.NewEmitRecommendTrackLogic(r.Context(), svcCtx)
		resp, err := l.EmitRecommendTrack(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

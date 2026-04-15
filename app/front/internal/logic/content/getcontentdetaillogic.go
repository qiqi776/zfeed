// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package content

import (
	"context"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	contentservice "zfeed/app/rpc/content/contentservice"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetContentDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetContentDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetContentDetailLogic {
	return &GetContentDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetContentDetailLogic) GetContentDetail(req *types.GetContentDetailReq) (*types.GetContentDetailRes, error) {
	if req == nil || req.ContentId == nil || *req.ContentId <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	rpcReq := &contentservice.GetContentDetailReq{
		ContentId: *req.ContentId,
	}
	if viewerID := utils.GetContextUserIdWithDefault(l.ctx); viewerID > 0 {
		rpcReq.ViewerId = &viewerID
	}

	rpcResp, err := l.svcCtx.ContentRpc.GetContentDetail(l.ctx, rpcReq)
	if err != nil {
		return nil, err
	}
	if rpcResp == nil || rpcResp.GetDetail() == nil {
		return nil, errorx.NewNotFound("内容不存在")
	}

	detail := rpcResp.GetDetail()
	return &types.GetContentDetailRes{
		Detail: types.ContentDetail{
			ContentId:         detail.GetContentId(),
			ContentType:       int32(detail.GetContentType()),
			AuthorId:          detail.GetAuthorId(),
			AuthorName:        detail.GetAuthorName(),
			AuthorAvatar:      detail.GetAuthorAvatar(),
			Title:             detail.GetTitle(),
			Description:       detail.GetDescription(),
			CoverUrl:          detail.GetCoverUrl(),
			ArticleContent:    detail.GetArticleContent(),
			VideoUrl:          detail.GetVideoUrl(),
			VideoDuration:     detail.GetVideoDuration(),
			PublishedAt:       detail.GetPublishedAt(),
			LikeCount:         detail.GetLikeCount(),
			FavoriteCount:     detail.GetFavoriteCount(),
			CommentCount:      detail.GetCommentCount(),
			IsLiked:           detail.GetIsLiked(),
			IsFavorited:       detail.GetIsFavorited(),
			IsFollowingAuthor: detail.GetIsFollowingAuthor(),
		},
	}, nil
}

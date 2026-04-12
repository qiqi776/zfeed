// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package feed

import (
	"context"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	contentpb "zfeed/app/rpc/content/content"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UserFavoriteLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUserFavoriteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserFavoriteLogic {
	return &UserFavoriteLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UserFavoriteLogic) UserFavorite(req *types.UserFavoriteFeedReq) (resp *types.UserFavoriteFeedRes, err error) {
	if req == nil || req.UserId == nil || req.Cursor == nil || req.PageSize == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	var viewerID *int64
	if uid := utils.GetContextUserIdWithDefault(l.ctx); uid > 0 {
		viewerID = &uid
	}

	rpcResp, err := l.svcCtx.FeedRpc.UserFavoriteFeed(l.ctx, &contentpb.UserFavoriteFeedReq{
		ViewerId: viewerID,
		UserId:   *req.UserId,
		Cursor:   *req.Cursor,
		PageSize: *req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	items := make([]types.UserFavoriteFeedItem, 0, len(rpcResp.GetItems()))
	for _, item := range rpcResp.GetItems() {
		if item == nil {
			continue
		}
		items = append(items, types.UserFavoriteFeedItem{
			ContentId:    item.GetContentId(),
			ContentType:  int32(item.GetContentType()),
			AuthorId:     item.GetAuthorId(),
			AuthorName:   item.GetAuthorName(),
			AuthorAvatar: item.GetAuthorAvatar(),
			Title:        item.GetTitle(),
			CoverUrl:     item.GetCoverUrl(),
			PublishedAt:  item.GetPublishedAt(),
			IsLiked:      item.GetIsLiked(),
			LikeCount:    item.GetLikeCount(),
		})
	}

	return &types.UserFavoriteFeedRes{
		Items:      items,
		NextCursor: rpcResp.GetNextCursor(),
		HasMore:    rpcResp.GetHasMore(),
	}, nil
}

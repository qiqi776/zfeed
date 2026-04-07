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

type UserPublishLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUserPublishLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserPublishLogic {
	return &UserPublishLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UserPublishLogic) UserPublish(req *types.UserPublishFeedReq) (resp *types.UserPublishFeedRes, err error) {
	if req == nil || req.UserId == nil || req.Cursor == nil || req.PageSize == nil {
		return nil, errorx.NewMsg("参数错误")
	}

	viewerID := req.ViewerId
	if uid := utils.GetContextUserIdWithDefault(l.ctx); uid > 0 {
		viewerID = &uid
	}

	rpcResp, err := l.svcCtx.FeedRpc.UserPublishFeed(l.ctx, &contentpb.UserPublishFeedReq{
		AuthorId: *req.UserId,
		ViewerId: viewerID,
		Cursor:   *req.Cursor,
		PageSize: *req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	items := make([]types.UserPublishFeedItem, 0, len(rpcResp.GetItems()))
	for _, item := range rpcResp.GetItems() {
		if item == nil {
			continue
		}
		items = append(items, types.UserPublishFeedItem{
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

	return &types.UserPublishFeedRes{
		Items:      items,
		NextCursor: rpcResp.GetNextCursor(),
		HasMore:    rpcResp.GetHasMore(),
	}, nil
}

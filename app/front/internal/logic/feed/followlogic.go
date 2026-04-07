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

type FollowLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFollowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowLogic {
	return &FollowLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FollowLogic) Follow(req *types.FollowFeedReq) (resp *types.FollowFeedRes, err error) {
	if req == nil || req.Cursor == nil || req.PageSize == nil {
		return nil, errorx.NewMsg("参数错误")
	}
	userID, err := utils.GetContextUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	rpcResp, err := l.svcCtx.FeedRpc.FollowFeed(l.ctx, &contentpb.FollowFeedReq{
		UserId:   userID,
		Cursor:   *req.Cursor,
		PageSize: *req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	items := make([]types.FollowFeedItem, 0, len(rpcResp.GetItems()))
	for _, item := range rpcResp.GetItems() {
		if item == nil {
			continue
		}
		items = append(items, types.FollowFeedItem{
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

	return &types.FollowFeedRes{
		Items:      items,
		NextCursor: rpcResp.GetNextCursor(),
		HasMore:    rpcResp.GetHasMore(),
	}, nil
}

// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package feed

import (
	"context"
	"strconv"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	contentpb "zfeed/app/rpc/content/content"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type RecommendLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRecommendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecommendLogic {
	return &RecommendLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RecommendLogic) Recommend(req *types.RecommendFeedReq) (resp *types.RecommendFeedRes, err error) {
	if req == nil || req.Cursor == nil || req.PageSize == nil {
		return nil, errorx.NewMsg("参数错误")
	}

	var userID *int64
	if uid := utils.GetContextUserIdWithDefault(l.ctx); uid > 0 {
		userID = &uid
	}

	rpcResp, err := l.svcCtx.FeedRpc.RecommendFeed(l.ctx, &contentpb.RecommendFeedReq{
		UserId:     userID,
		Cursor:     *req.Cursor,
		PageSize:   *req.PageSize,
		SnapshotId: req.SnapshotId,
	})
	if err != nil {
		return nil, err
	}

	items := make([]types.RecommendFeedItem, 0, len(rpcResp.GetItems()))
	for _, item := range rpcResp.GetItems() {
		if item == nil {
			continue
		}
		items = append(items, types.RecommendFeedItem{
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

	nextCursor := ""
	if rpcResp.GetHasMore() {
		nextCursor = strconv.FormatInt(rpcResp.GetNextCursor(), 10)
	}

	return &types.RecommendFeedRes{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    rpcResp.GetHasMore(),
		SnapshotId: rpcResp.GetSnapshotId(),
	}, nil
}

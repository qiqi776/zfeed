package search

import (
	"context"
	"strings"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/search/searchservice"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchContentsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSearchContentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchContentsLogic {
	return &SearchContentsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchContentsLogic) SearchContents(req *types.SearchContentsReq) (*types.SearchContentsRes, error) {
	if req == nil || req.Query == nil || req.PageSize == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	query := strings.TrimSpace(*req.Query)
	if query == "" {
		return nil, errorx.NewBadRequest("搜索词不能为空")
	}

	pageSize := int(*req.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > maxSearchPageSize {
		pageSize = maxSearchPageSize
	}

	cursor := int64(0)
	if req.Cursor != nil && *req.Cursor > 0 {
		cursor = *req.Cursor
	}

	viewerID := utils.GetContextUserIdWithDefault(l.ctx)
	rpcReq := &searchservice.SearchContentsReq{
		Query:    query,
		Cursor:   cursor,
		PageSize: uint32(pageSize),
	}
	if viewerID > 0 {
		rpcReq.ViewerId = &viewerID
	}

	rpcResp, err := l.svcCtx.SearchRpc.SearchContents(l.ctx, rpcReq)
	if err != nil {
		return nil, err
	}

	items := make([]types.SearchContentItem, 0, len(rpcResp.GetItems()))
	for _, row := range rpcResp.GetItems() {
		if row == nil {
			continue
		}
		items = append(items, types.SearchContentItem{
			ContentId:    row.GetContentId(),
			ContentType:  row.GetContentType(),
			AuthorId:     row.GetAuthorId(),
			AuthorName:   row.GetAuthorName(),
			AuthorAvatar: row.GetAuthorAvatar(),
			Title:        row.GetTitle(),
			CoverUrl:     row.GetCoverUrl(),
			PublishedAt:  row.GetPublishedAt(),
		})
	}

	return &types.SearchContentsRes{
		Items:      items,
		NextCursor: rpcResp.GetNextCursor(),
		HasMore:    rpcResp.GetHasMore(),
	}, nil
}

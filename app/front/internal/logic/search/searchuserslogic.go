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

const maxSearchPageSize = 20

type SearchUsersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSearchUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchUsersLogic {
	return &SearchUsersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchUsersLogic) SearchUsers(req *types.SearchUsersReq) (*types.SearchUsersRes, error) {
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
	rpcReq := &searchservice.SearchUsersReq{
		Query:    query,
		Cursor:   cursor,
		PageSize: uint32(pageSize),
	}
	if viewerID > 0 {
		rpcReq.ViewerId = &viewerID
	}

	rpcResp, err := l.svcCtx.SearchRpc.SearchUsers(l.ctx, rpcReq)
	if err != nil {
		return nil, err
	}

	items := make([]types.SearchUserItem, 0, len(rpcResp.GetItems()))
	for _, row := range rpcResp.GetItems() {
		if row == nil {
			continue
		}
		items = append(items, types.SearchUserItem{
			UserId:      row.GetUserId(),
			Nickname:    row.GetNickname(),
			Avatar:      row.GetAvatar(),
			Bio:         row.GetBio(),
			IsFollowing: row.GetIsFollowing(),
		})
	}

	return &types.SearchUsersRes{
		Items:      items,
		NextCursor: rpcResp.GetNextCursor(),
		HasMore:    rpcResp.GetHasMore(),
	}, nil
}

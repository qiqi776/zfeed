package logic

import (
	"context"
	"strings"

	followservice "zfeed/app/rpc/interaction/client/followservice"
	"zfeed/app/rpc/search/internal/repositories"
	"zfeed/app/rpc/search/internal/svc"
	"zfeed/app/rpc/search/search"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

const maxSearchPageSize = 20

type SearchUsersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	searchRepo repositories.SearchRepository
}

func NewSearchUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchUsersLogic {
	return &SearchUsersLogic{
		ctx:        ctx,
		svcCtx:     svcCtx,
		Logger:     logx.WithContext(ctx),
		searchRepo: repositories.NewSearchRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *SearchUsersLogic) SearchUsers(in *search.SearchUsersReq) (*search.SearchUsersRes, error) {
	if in == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	query := strings.TrimSpace(in.GetQuery())
	if query == "" {
		return nil, errorx.NewBadRequest("搜索词不能为空")
	}

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > maxSearchPageSize {
		pageSize = maxSearchPageSize
	}

	rows, err := l.searchRepo.SearchUsers(query, in.GetCursor(), pageSize+1)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("搜索用户失败"))
	}

	hasMore := len(rows) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}

	userIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		userIDs = append(userIDs, row.UserID)
	}

	followingMap := make(map[int64]bool, len(userIDs))
	if viewerID := in.GetViewerId(); viewerID > 0 && l.svcCtx != nil && l.svcCtx.FollowRpc != nil {
		followResp, err := l.svcCtx.FollowRpc.BatchQueryFollowing(l.ctx, &followservice.BatchQueryFollowingReq{
			UserId:        viewerID,
			FollowUserIds: userIDs,
		})
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("搜索用户失败"))
		}
		for _, item := range followResp.GetItems() {
			if item == nil {
				continue
			}
			followingMap[item.GetUserId()] = item.GetIsFollowing()
		}
	}

	items := make([]*search.SearchUserItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &search.SearchUserItem{
			UserId:      row.UserID,
			Nickname:    row.Nickname,
			Avatar:      row.Avatar,
			Bio:         row.Bio,
			IsFollowing: followingMap[row.UserID],
		})
	}

	nextCursor := int64(0)
	if hasMore && len(rows) > 0 {
		nextCursor = rows[len(rows)-1].UserID
	}

	return &search.SearchUsersRes{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

package logic

import (
	"context"
	"time"

	followservice "zfeed/app/rpc/interaction/client/followservice"
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
}

func NewSearchUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchUsersLogic {
	return &SearchUsersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SearchUsersLogic) SearchUsers(in *search.SearchUsersReq) (*search.SearchUsersRes, error) {
	if in == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	normalized := l.svcCtx.NormalizeQuery(in.GetQuery())
	if normalized.Empty() {
		return nil, errorx.NewBadRequest("搜索词不能为空")
	}

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > maxSearchPageSize {
		pageSize = maxSearchPageSize
	}

	start := time.Now()
	backend := l.svcCtx.SearchBackend(l.ctx)
	result, err := backend.SearchUsers(l.ctx, normalized.SearchText, in.GetCursor(), pageSize+1)
	if err != nil {
		observeSearch(l.Logger, searchObservation{
			entity:            searchEntityUsers,
			query:             normalized,
			cursor:            in.GetCursor(),
			pageSize:          pageSize,
			err:               err,
			start:             start,
			meta:              result.Meta,
			cacheStatus:       cacheStatus(l.svcCtx),
			configuredBackend: l.svcCtx.ConfiguredSearchBackend(),
			effectiveBackend:  l.svcCtx.EffectiveSearchBackend(),
			svcCtx:            l.svcCtx,
		})
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("搜索用户失败"))
	}

	rows := result.Rows
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
			observeSearch(l.Logger, searchObservation{
				entity:            searchEntityUsers,
				query:             normalized,
				cursor:            in.GetCursor(),
				pageSize:          pageSize,
				resultCount:       len(rows),
				hasMore:           hasMore,
				err:               err,
				start:             start,
				meta:              result.Meta,
				cacheStatus:       cacheStatus(l.svcCtx),
				configuredBackend: l.svcCtx.ConfiguredSearchBackend(),
				effectiveBackend:  l.svcCtx.EffectiveSearchBackend(),
				svcCtx:            l.svcCtx,
			})
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

	observeSearch(l.Logger, searchObservation{
		entity:            searchEntityUsers,
		query:             normalized,
		cursor:            in.GetCursor(),
		pageSize:          pageSize,
		resultCount:       len(items),
		hasMore:           hasMore,
		start:             start,
		meta:              result.Meta,
		cacheStatus:       cacheStatus(l.svcCtx),
		configuredBackend: l.svcCtx.ConfiguredSearchBackend(),
		effectiveBackend:  l.svcCtx.EffectiveSearchBackend(),
		svcCtx:            l.svcCtx,
	})

	return &search.SearchUsersRes{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

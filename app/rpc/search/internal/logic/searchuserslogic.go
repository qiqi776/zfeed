package logic

import (
	"context"
	"time"

	followservice "zfeed/app/rpc/interaction/client/followservice"
	searchbackend "zfeed/app/rpc/search/internal/backend"
	"zfeed/app/rpc/search/internal/querynorm"
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
	mode, err := normalizeSearchMode(in.GetMode())
	if err != nil {
		return nil, err
	}
	if usesSnapshotPagination(mode) {
		if !snapshotPaginationEnabled(l.svcCtx) {
			return nil, errorx.NewBadRequest("搜索快照未开启")
		}
		return l.searchUsersWithSnapshot(in, normalized, mode, pageSize, start)
	}

	searchBackend := l.svcCtx.SearchBackend(l.ctx)
	var (
		result             searchbackend.SearchUsersResult
		requestCacheStatus = cacheStatus(l.svcCtx)
		usedCursorCache    bool
	)
	if in.GetCursor() > 0 {
		if cachedResult, ok := l.cachedUsersAfterCursor(normalized, mode, in.GetCursor(), pageSize); ok {
			result = cachedResult
			requestCacheStatus = searchCacheHit
			usedCursorCache = true
		}
	}
	if !usedCursorCache {
		limit := pageSize + 1
		cachePage := -1
		if in.GetCursor() == 0 {
			cachePage = 0
			if searchQueryCacheEnabled(l.svcCtx) {
				limit = latestQueryCacheLimit(l.svcCtx)
			}
		}
		result, requestCacheStatus, err = l.queryUsersWithCache(
			normalized,
			mode,
			cachePage,
			limit,
			func() (searchbackend.SearchUsersResult, error) {
				return searchBackend.SearchUsers(l.ctx, normalized.SearchText, in.GetCursor(), limit)
			},
		)
	}
	if err != nil {
		observeSearch(l.Logger, searchObservation{
			entity:            searchEntityUsers,
			query:             normalized,
			cursor:            in.GetCursor(),
			pageSize:          pageSize,
			mode:              mode,
			err:               err,
			start:             start,
			meta:              result.Meta,
			cacheStatus:       requestCacheStatus,
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

	items, err := l.buildSearchUserItems(rows, in.GetViewerId())
	if err != nil {
		observeSearch(l.Logger, searchObservation{
			entity:            searchEntityUsers,
			query:             normalized,
			cursor:            in.GetCursor(),
			pageSize:          pageSize,
			resultCount:       len(rows),
			hasMore:           hasMore,
			mode:              mode,
			err:               err,
			start:             start,
			meta:              result.Meta,
			cacheStatus:       requestCacheStatus,
			configuredBackend: l.svcCtx.ConfiguredSearchBackend(),
			effectiveBackend:  l.svcCtx.EffectiveSearchBackend(),
			svcCtx:            l.svcCtx,
		})
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("搜索用户失败"))
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
		mode:              mode,
		start:             start,
		meta:              result.Meta,
		cacheStatus:       requestCacheStatus,
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

func (l *SearchUsersLogic) searchUsersWithSnapshot(
	in *search.SearchUsersReq,
	normalized querynorm.Query,
	mode string,
	pageSize int,
	start time.Time,
) (*search.SearchUsersRes, error) {
	pageReq, hasSnapshot, err := snapshotPageRequest(in.GetPageToken(), in.GetSnapshotId())
	if err != nil {
		observeSearch(l.Logger, searchObservation{
			entity:            searchEntityUsers,
			query:             normalized,
			cursor:            in.GetCursor(),
			pageSize:          pageSize,
			mode:              mode,
			pageTokenProvided: in.GetPageToken() != "",
			err:               err,
			start:             start,
			meta:              repositories.SearchMeta{QueryPath: "snapshot"},
			cacheStatus:       cacheStatus(l.svcCtx),
			configuredBackend: l.svcCtx.ConfiguredSearchBackend(),
			effectiveBackend:  l.svcCtx.EffectiveSearchBackend(),
			svcCtx:            l.svcCtx,
		})
		return nil, err
	}

	var (
		rows           []repositories.SearchUserRow
		snapshotID     string
		offset         int
		meta           repositories.SearchMeta
		cacheState     = cacheStatus(l.svcCtx)
		snapshotStatus = "create"
	)

	if hasSnapshot {
		snapshot, err := loadSearchSnapshot(l.ctx, l.svcCtx, pageReq.SnapshotID, searchEntityUsers, mode, normalized)
		if err != nil {
			observeSearch(l.Logger, searchObservation{
				entity:            searchEntityUsers,
				query:             normalized,
				cursor:            in.GetCursor(),
				pageSize:          pageSize,
				mode:              mode,
				pageTokenProvided: in.GetPageToken() != "",
				snapshotID:        pageReq.SnapshotID,
				snapshotStatus:    "miss",
				err:               err,
				start:             start,
				meta:              repositories.SearchMeta{QueryPath: "snapshot"},
				cacheStatus:       cacheState,
				configuredBackend: l.svcCtx.ConfiguredSearchBackend(),
				effectiveBackend:  l.svcCtx.EffectiveSearchBackend(),
				svcCtx:            l.svcCtx,
			})
			return nil, err
		}
		rows = snapshot.UserRows
		snapshotID = pageReq.SnapshotID
		offset = pageReq.Offset
		meta = repositories.SearchMeta{QueryPath: "snapshot"}
		snapshotStatus = "hit"
	} else {
		searchBackend := l.svcCtx.SearchBackend(l.ctx)
		limit := snapshotMaxItems(l.svcCtx, pageSize)
		result, requestCacheStatus, err := l.queryUsersWithCache(
			normalized,
			mode,
			0,
			limit,
			func() (searchbackend.SearchUsersResult, error) {
				return searchBackend.SearchUsers(l.ctx, normalized.SearchText, 0, limit)
			},
		)
		cacheState = requestCacheStatus
		if err != nil {
			observeSearch(l.Logger, searchObservation{
				entity:            searchEntityUsers,
				query:             normalized,
				cursor:            in.GetCursor(),
				pageSize:          pageSize,
				mode:              mode,
				err:               err,
				start:             start,
				meta:              result.Meta,
				cacheStatus:       cacheState,
				configuredBackend: l.svcCtx.ConfiguredSearchBackend(),
				effectiveBackend:  l.svcCtx.EffectiveSearchBackend(),
				svcCtx:            l.svcCtx,
			})
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("搜索用户失败"))
		}

		rows = result.Rows
		meta = result.Meta
		if len(rows) > 0 {
			snapshotID, err = createUserSnapshot(l.ctx, l.svcCtx, mode, normalized, rows)
			if err != nil {
				observeSearch(l.Logger, searchObservation{
					entity:            searchEntityUsers,
					query:             normalized,
					cursor:            in.GetCursor(),
					pageSize:          pageSize,
					mode:              mode,
					snapshotStatus:    "error",
					err:               err,
					start:             start,
					meta:              result.Meta,
					cacheStatus:       cacheState,
					configuredBackend: l.svcCtx.ConfiguredSearchBackend(),
					effectiveBackend:  l.svcCtx.EffectiveSearchBackend(),
					svcCtx:            l.svcCtx,
				})
				return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("搜索用户失败"))
			}
		}
	}

	pagedRows, hasMore, nextOffset := pageRows(rows, offset, pageSize)
	items, err := l.buildSearchUserItems(pagedRows, in.GetViewerId())
	if err != nil {
		observeSearch(l.Logger, searchObservation{
			entity:            searchEntityUsers,
			query:             normalized,
			cursor:            in.GetCursor(),
			pageSize:          pageSize,
			resultCount:       len(pagedRows),
			hasMore:           hasMore,
			mode:              mode,
			pageTokenProvided: in.GetPageToken() != "",
			snapshotID:        snapshotID,
			snapshotStatus:    snapshotStatus,
			err:               err,
			start:             start,
			meta:              meta,
			cacheStatus:       cacheState,
			configuredBackend: l.svcCtx.ConfiguredSearchBackend(),
			effectiveBackend:  l.svcCtx.EffectiveSearchBackend(),
			svcCtx:            l.svcCtx,
		})
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("搜索用户失败"))
	}

	nextCursor := int64(0)
	if hasMore && len(pagedRows) > 0 {
		nextCursor = pagedRows[len(pagedRows)-1].UserID
	}

	nextPageToken := ""
	if hasMore && snapshotID != "" {
		nextPageToken, err = encodePageToken(snapshotID, nextOffset)
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("搜索用户失败"))
		}
	}

	observeSearch(l.Logger, searchObservation{
		entity:            searchEntityUsers,
		query:             normalized,
		cursor:            in.GetCursor(),
		pageSize:          pageSize,
		resultCount:       len(items),
		hasMore:           hasMore,
		mode:              mode,
		pageTokenProvided: in.GetPageToken() != "",
		snapshotID:        snapshotID,
		snapshotStatus:    snapshotStatus,
		start:             start,
		meta:              meta,
		cacheStatus:       cacheState,
		configuredBackend: l.svcCtx.ConfiguredSearchBackend(),
		effectiveBackend:  l.svcCtx.EffectiveSearchBackend(),
		svcCtx:            l.svcCtx,
	})

	return &search.SearchUsersRes{
		Items:         items,
		NextCursor:    nextCursor,
		HasMore:       hasMore,
		NextPageToken: nextPageToken,
		SnapshotId:    snapshotID,
	}, nil
}

func (l *SearchUsersLogic) buildSearchUserItems(rows []repositories.SearchUserRow, viewerID int64) ([]*search.SearchUserItem, error) {
	userIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		userIDs = append(userIDs, row.UserID)
	}

	followingMap := make(map[int64]bool, len(userIDs))
	if viewerID > 0 && l.svcCtx != nil && l.svcCtx.FollowRpc != nil {
		followResp, err := l.svcCtx.FollowRpc.BatchQueryFollowing(l.ctx, &followservice.BatchQueryFollowingReq{
			UserId:        viewerID,
			FollowUserIds: userIDs,
		})
		if err != nil {
			return nil, err
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

	return items, nil
}

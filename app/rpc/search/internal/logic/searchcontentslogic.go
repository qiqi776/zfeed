package logic

import (
	"context"
	"time"

	searchbackend "zfeed/app/rpc/search/internal/backend"
	"zfeed/app/rpc/search/internal/querynorm"
	"zfeed/app/rpc/search/internal/repositories"
	"zfeed/app/rpc/search/internal/svc"
	"zfeed/app/rpc/search/search"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchContentsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSearchContentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchContentsLogic {
	return &SearchContentsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SearchContentsLogic) SearchContents(in *search.SearchContentsReq) (*search.SearchContentsRes, error) {
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
		return l.searchContentsWithSnapshot(in, normalized, mode, pageSize, start)
	}

	searchBackend := l.svcCtx.SearchBackend(l.ctx)
	var (
		result             searchbackend.SearchContentsResult
		requestCacheStatus = cacheStatus(l.svcCtx)
		usedCursorCache    bool
	)
	if in.GetCursor() > 0 {
		if cachedResult, ok := l.cachedContentsAfterCursor(normalized, mode, in.GetCursor(), pageSize); ok {
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
		result, requestCacheStatus, err = l.queryContentsWithCache(
			normalized,
			mode,
			cachePage,
			limit,
			func() (searchbackend.SearchContentsResult, error) {
				return searchBackend.SearchContents(l.ctx, normalized.SearchText, mode, in.GetCursor(), limit)
			},
		)
	}
	if err != nil {
		observeSearch(l.Logger, searchObservation{
			entity:            searchEntityContents,
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
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("搜索内容失败"))
	}

	rows := result.Rows
	hasMore := len(rows) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}

	items := make([]*search.SearchContentItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &search.SearchContentItem{
			ContentId:    row.ContentID,
			ContentType:  row.ContentType,
			AuthorId:     row.AuthorID,
			AuthorName:   row.AuthorName,
			AuthorAvatar: row.AuthorAvatar,
			Title:        row.Title,
			CoverUrl:     row.CoverURL,
			PublishedAt:  unixOrZero(row.PublishedAt),
		})
	}

	nextCursor := int64(0)
	if hasMore && len(rows) > 0 {
		nextCursor = rows[len(rows)-1].ContentID
	}

	observeSearch(l.Logger, searchObservation{
		entity:            searchEntityContents,
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

	return &search.SearchContentsRes{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (l *SearchContentsLogic) searchContentsWithSnapshot(
	in *search.SearchContentsReq,
	normalized querynorm.Query,
	mode string,
	pageSize int,
	start time.Time,
) (*search.SearchContentsRes, error) {
	pageReq, hasSnapshot, err := snapshotPageRequest(in.GetPageToken(), in.GetSnapshotId())
	if err != nil {
		observeSearch(l.Logger, searchObservation{
			entity:            searchEntityContents,
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
		rows           []repositories.SearchContentRow
		snapshotID     string
		offset         int
		meta           repositories.SearchMeta
		cacheState     = cacheStatus(l.svcCtx)
		snapshotStatus = "create"
	)

	if hasSnapshot {
		snapshot, err := loadSearchSnapshot(l.ctx, l.svcCtx, pageReq.SnapshotID, searchEntityContents, mode, normalized)
		if err != nil {
			observeSearch(l.Logger, searchObservation{
				entity:            searchEntityContents,
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
		rows = snapshot.ContentRows
		snapshotID = pageReq.SnapshotID
		offset = pageReq.Offset
		meta = repositories.SearchMeta{QueryPath: "snapshot"}
		snapshotStatus = "hit"
	} else {
		searchBackend := l.svcCtx.SearchBackend(l.ctx)
		limit := snapshotMaxItems(l.svcCtx, pageSize)
		result, requestCacheStatus, err := l.queryContentsWithCache(
			normalized,
			mode,
			0,
			limit,
			func() (searchbackend.SearchContentsResult, error) {
				return searchBackend.SearchContents(l.ctx, normalized.SearchText, mode, 0, limit)
			},
		)
		cacheState = requestCacheStatus
		if err != nil {
			observeSearch(l.Logger, searchObservation{
				entity:            searchEntityContents,
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
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("搜索内容失败"))
		}

		rows = result.Rows
		meta = result.Meta
		if len(rows) > 0 {
			snapshotID, err = createContentSnapshot(l.ctx, l.svcCtx, mode, normalized, rows)
			if err != nil {
				observeSearch(l.Logger, searchObservation{
					entity:            searchEntityContents,
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
				return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("搜索内容失败"))
			}
		}
	}

	pagedRows, hasMore, nextOffset := pageRows(rows, offset, pageSize)
	items := buildSearchContentItems(pagedRows)

	nextCursor := int64(0)
	if hasMore && len(pagedRows) > 0 {
		nextCursor = pagedRows[len(pagedRows)-1].ContentID
	}

	nextPageToken := ""
	if hasMore && snapshotID != "" {
		nextPageToken, err = encodePageToken(snapshotID, nextOffset)
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("搜索内容失败"))
		}
	}

	observeSearch(l.Logger, searchObservation{
		entity:            searchEntityContents,
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

	return &search.SearchContentsRes{
		Items:         items,
		NextCursor:    nextCursor,
		HasMore:       hasMore,
		NextPageToken: nextPageToken,
		SnapshotId:    snapshotID,
	}, nil
}

func buildSearchContentItems(rows []repositories.SearchContentRow) []*search.SearchContentItem {
	items := make([]*search.SearchContentItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &search.SearchContentItem{
			ContentId:    row.ContentID,
			ContentType:  row.ContentType,
			AuthorId:     row.AuthorID,
			AuthorName:   row.AuthorName,
			AuthorAvatar: row.AuthorAvatar,
			Title:        row.Title,
			CoverUrl:     row.CoverURL,
			PublishedAt:  unixOrZero(row.PublishedAt),
		})
	}
	return items
}

func unixOrZero(value *time.Time) int64 {
	if value == nil {
		return 0
	}
	return value.Unix()
}

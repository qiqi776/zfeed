package logic

import (
	"context"
	"time"

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
	backend := l.svcCtx.SearchBackend(l.ctx)
	result, err := backend.SearchContents(l.ctx, normalized.SearchText, in.GetCursor(), pageSize+1)
	if err != nil {
		observeSearch(l.Logger, searchObservation{
			entity:            searchEntityContents,
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
		start:             start,
		meta:              result.Meta,
		cacheStatus:       cacheStatus(l.svcCtx),
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

func unixOrZero(value *time.Time) int64 {
	if value == nil {
		return 0
	}
	return value.Unix()
}
